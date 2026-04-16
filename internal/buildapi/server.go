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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Server struct {
	Client client.Client
	Addr   string
	CLIDir string
}

func NewServer(c client.Client, addr, cliDir string) *Server {
	return &Server{Client: c, Addr: addr, CLIDir: cliDir}
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("buildapi")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs", s.handleList)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs", s.handleCreate)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}", s.handleGet)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs/{name}/run", s.handleRun)
	mux.HandleFunc("DELETE /v1/namespaces/{namespace}/buildjobs/{name}", s.handleDelete)
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
		existing.Annotations["builder.sdv.cloud.redhat.com/run-at"] = time.Now().UTC().Format(time.RFC3339)
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
	bj.Annotations["builder.sdv.cloud.redhat.com/run-at"] = time.Now().UTC().Format(time.RFC3339)
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
	defer f.Close()

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

func toSummary(bj *buildv1alpha1.BuildJob) BuildJobSummary {
	summary := BuildJobSummary{
		Name:         bj.Name,
		Namespace:    bj.Namespace,
		Phase:        string(bj.Status.Phase),
		Board:        bj.Spec.Target.Board,
		Platform:     bj.Spec.Target.Platform,
		Architecture: bj.Spec.Target.Architecture,
		Image:        bj.Spec.Toolchain.Image,
		ArtifactURI:  bj.Status.ArtifactURI,
		PipelineRun:  bj.Status.CurrentPipelineRun,
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
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
