// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package buildapi

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultMaxUploadBytes   int64 = 1 << 30 // 1 GiB
	DefaultMaxUploadFiles         = 1000
	DefaultMaxFileBytes     int64 = 512 << 20 // 512 MiB per file
	DefaultUploadTimeoutSec       = 300
)

type Server struct {
	Client       client.Client
	Addr         string
	CLIDir       string
	ArtifactsDir string
	KubeClient   kubernetes.Interface

	MaxUploadBytes   int64
	MaxUploadFiles   int
	MaxFileBytes     int64
	UploadTimeoutSec int
}

func NewServer(c client.Client, addr, cliDir, artifactsDir string, restConfig *rest.Config) (*Server, error) {
	s := &Server{
		Client:           c,
		Addr:             addr,
		CLIDir:           cliDir,
		ArtifactsDir:     artifactsDir,
		MaxUploadBytes:   DefaultMaxUploadBytes,
		MaxUploadFiles:   DefaultMaxUploadFiles,
		MaxFileBytes:     DefaultMaxFileBytes,
		UploadTimeoutSec: DefaultUploadTimeoutSec,
	}
	if restConfig != nil {
		kc, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("creating kubernetes client: %w", err)
		}
		s.KubeClient = kc
	}
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("buildapi")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs", s.handleList)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs", s.handleCreate)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}", s.handleGet)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs/{name}/run", s.handleRun)
	mux.HandleFunc("DELETE /v1/namespaces/{namespace}/buildjobs/{name}", s.handleDelete)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}/logs", s.handleLogs)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs/{name}/artifacts/upload", s.handleArtifactUpload)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}/artifacts", s.handleArtifactList)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}/artifacts/{filename}", s.handleArtifactDownload)
	mux.HandleFunc("GET /v1/cli/{os}/{arch}", s.handleCLIDownload)
	mux.HandleFunc("GET /v1/cli", s.handleCLIList)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:              s.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      10 * time.Minute,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("Build API server starting", "addr", s.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("build API server: %w", err)
	}
	return nil
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")

	var list buildv1alpha1.BuildJobList
	if err := s.Client.List(r.Context(), &list, client.InNamespace(ns)); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing BuildJobs: %v", err))
		return
	}

	items := make([]BuildJobSummary, 0, len(list.Items))
	for _, bj := range list.Items {
		items = append(items, toSummary(&bj))
	}
	writeJSON(w, http.StatusOK, BuildJobListResponse{Items: items})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	var bj buildv1alpha1.BuildJob
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &bj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting BuildJob: %v", err))
		}
		return
	}
	writeJSON(w, http.StatusOK, toSummary(&bj))
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "reading request body")
		return
	}

	var req CreateBuildJobRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var existing buildv1alpha1.BuildJob
	key := client.ObjectKey{Namespace: ns, Name: req.Name}
	err = s.Client.Get(r.Context(), key, &existing)
	switch {
	case err == nil:
		existing.Spec = req.Spec
		if existing.Annotations == nil {
			existing.Annotations = map[string]string{}
		}
		existing.Annotations[buildv1alpha1.AnnotationRunAt] = time.Now().UTC().Format(time.RFC3339)
		if err := s.Client.Update(r.Context(), &existing); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating BuildJob: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, toSummary(&existing))
		return
	case !apierrors.IsNotFound(err):
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("looking up BuildJob: %v", err))
		return
	}

	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: ns,
		},
		Spec: req.Spec,
	}

	if err := s.Client.Create(r.Context(), bj); err != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("creating BuildJob: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, toSummary(bj))
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	var bj buildv1alpha1.BuildJob
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &bj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting BuildJob: %v", err))
		}
		return
	}

	// Bump an annotation to trigger a new reconciliation / PipelineRun generation.
	if bj.Annotations == nil {
		bj.Annotations = map[string]string{}
	}
	bj.Annotations[buildv1alpha1.AnnotationRunAt] = time.Now().UTC().Format(time.RFC3339)
	if err := s.Client.Update(r.Context(), &bj); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("triggering run: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, toSummary(&bj))
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	var bj buildv1alpha1.BuildJob
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &bj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting BuildJob: %v", err))
		}
		return
	}
	if err := s.Client.Delete(r.Context(), &bj); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting BuildJob: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	var bj buildv1alpha1.BuildJob
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &bj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting BuildJob: %v", err))
		}
		return
	}

	prName := bj.Status.CurrentPipelineRun
	if prName == "" {
		writeError(w, http.StatusNotFound, "no PipelineRun found for this BuildJob (build may not have started)")
		return
	}

	if s.KubeClient == nil {
		writeError(w, http.StatusInternalServerError, "kubernetes client not configured")
		return
	}

	pods, err := s.KubeClient.CoreV1().Pods(ns).List(r.Context(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", prName),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing pods: %v", err))
		return
	}

	if len(pods.Items) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("no pods found for PipelineRun %q (build may still be initializing)", prName))
		return
	}

	logCtx, logCancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer logCancel()

	var limitBytes int64 = 10 << 20 // 10 MiB per container

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			fmt.Fprintf(w, "=== %s/%s ===\n", pod.Name, container.Name)
			if flusher != nil {
				flusher.Flush()
			}

			stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container:  container.Name,
				LimitBytes: &limitBytes,
			}).Stream(logCtx)
			if err != nil {
				fmt.Fprintf(w, "[error reading logs: %v]\n\n", err)
				continue
			}
			_, _ = io.CopyN(w, stream, limitBytes)
			_ = stream.Close()
			_, _ = w.Write([]byte("\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

var validCLIPlatforms = map[string]map[string]bool{
	"darwin": {"amd64": true, "arm64": true},
	"linux":  {"amd64": true, "arm64": true},
}

func (s *Server) handleCLIDownload(w http.ResponseWriter, r *http.Request) {
	goos := r.PathValue("os")
	goarch := r.PathValue("arch")

	if arches, ok := validCLIPlatforms[goos]; !ok || !arches[goarch] {
		writeError(w, http.StatusNotFound, fmt.Sprintf("no binary for %s/%s — available: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64", goos, goarch))
		return
	}

	filename := fmt.Sprintf("bob-%s-%s", goos, goarch)
	path := filepath.Join(s.CLIDir, filename)

	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("binary not found: %s (cli-dir=%s)", filename, s.CLIDir))
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "stat failed")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"bob\""))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	http.ServeContent(w, r, filename, stat.ModTime(), f)
}

func (s *Server) handleCLIList(w http.ResponseWriter, r *http.Request) {
	type platformEntry struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
		URL  string `json:"url"`
	}

	var platforms []platformEntry
	for goos, arches := range validCLIPlatforms {
		for arch := range arches {
			filename := fmt.Sprintf("bob-%s-%s", goos, arch)
			path := filepath.Join(s.CLIDir, filename)
			if _, err := os.Stat(path); err == nil {
				platforms = append(platforms, platformEntry{
					OS:   goos,
					Arch: arch,
					URL:  fmt.Sprintf("/v1/cli/%s/%s", goos, arch),
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"platforms": platforms,
		"usage":     "curl -Lo bob https://<bob-server>/v1/cli/<os>/<arch> && chmod +x bob",
	})
}

func (s *Server) artifactDir(ns, name string) string {
	return filepath.Join(s.ArtifactsDir, ns, name)
}

func (s *Server) handleArtifactUpload(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	uploadTimeout := time.Duration(s.UploadTimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), uploadTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	limitedBody := http.MaxBytesReader(w, r.Body, s.MaxUploadBytes)
	defer func() { _ = limitedBody.Close() }()

	dir := s.artifactDir(ns, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating artifact dir: %v", err))
		return
	}

	gr, err := gzip.NewReader(limitedBody)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid gzip: %v", err))
		return
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	var count int
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if r.Context().Err() != nil {
				writeError(w, http.StatusRequestTimeout, "upload timed out")
				return
			}
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid tar entry: %v", err))
			return
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		clean := filepath.Base(hdr.Name)
		if clean == "." || clean == ".." || strings.Contains(clean, "/") {
			continue
		}
		if count >= s.MaxUploadFiles {
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("too many files: limit is %d", s.MaxUploadFiles))
			return
		}
		f, err := os.Create(filepath.Join(dir, clean))
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating file: %v", err))
			return
		}
		written, copyErr := io.Copy(f, io.LimitReader(tr, s.MaxFileBytes+1))
		_ = f.Close()
		if copyErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("writing file: %v", copyErr))
			return
		}
		if written > s.MaxFileBytes {
			_ = os.Remove(filepath.Join(dir, clean))
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file %q exceeds max size of %d bytes", clean, s.MaxFileBytes))
			return
		}
		count++
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"uploaded": count})
}

func (s *Server) handleArtifactList(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	dir := s.artifactDir(ns, name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, ArtifactListResponse{BuildJob: name, Namespace: ns, Files: []ArtifactFileInfo{}})
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("reading artifacts: %v", err))
		return
	}

	files := make([]ArtifactFileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, ArtifactFileInfo{Name: e.Name(), Size: info.Size()})
	}
	writeJSON(w, http.StatusOK, ArtifactListResponse{BuildJob: name, Namespace: ns, Files: files})
}

func (s *Server) handleArtifactDownload(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")
	filename := r.PathValue("filename")

	clean := filepath.Base(filename)
	path := filepath.Join(s.artifactDir(ns, name), clean)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("artifact %q not found", clean))
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening artifact: %v", err))
		}
		return
	}
	defer func() { _ = f.Close() }()

	stat, _ := f.Stat()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", clean))
	http.ServeContent(w, r, clean, stat.ModTime(), f)
}

func toSummary(bj *buildv1alpha1.BuildJob) BuildJobSummary {
	summary := BuildJobSummary{
		Name:              bj.Name,
		Namespace:         bj.Namespace,
		Phase:             string(bj.Status.Phase),
		Board:             bj.Spec.Target.Board,
		Platform:          bj.Spec.Target.Platform,
		Architecture:      bj.Spec.Target.Architecture,
		Image:             bj.Spec.Toolchain.Image,
		CommitSHA:         bj.Status.CommitSHA,
		ArtifactURI:       bj.Status.ArtifactURI,
		OCIArtifactRef:    bj.Status.OCIArtifactRef,
		OCIArtifactDigest: bj.Status.OCIArtifactDigest,
		PipelineRun:       bj.Status.CurrentPipelineRun,
	}

	if bj.Spec.Source.Type == buildv1alpha1.SourceTypeGit && bj.Spec.Source.Git != nil {
		summary.Source = &SourceInfo{
			Type:     "git",
			URL:      bj.Spec.Source.Git.URL,
			Revision: bj.Spec.Source.Git.Revision,
		}
	} else if bj.Spec.Source.Type == buildv1alpha1.SourceTypePVC {
		summary.Source = &SourceInfo{Type: "pvc"}
	}

	for _, s := range bj.Status.Stages {
		summary.Stages = append(summary.Stages, StageInfo{
			Name:    s.Name,
			State:   s.State,
			Message: s.Message,
		})
	}

	if !bj.CreationTimestamp.IsZero() {
		summary.Age = time.Since(bj.CreationTimestamp.Time).Truncate(time.Second).String()
	}

	return summary
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Log.Error(err, "failed to encode JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
