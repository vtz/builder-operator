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
	"net/http"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Server struct {
	Client client.Client
	Addr   string
}

func NewServer(c client.Client, addr string) *Server {
	return &Server{Client: c, Addr: addr}
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("buildapi")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs", s.handleList)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}", s.handleGet)
	mux.HandleFunc("DELETE /v1/namespaces/{namespace}/buildjobs/{name}", s.handleDelete)
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
		writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		return
	}
	writeJSON(w, http.StatusOK, toSummary(&bj))
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	var bj buildv1alpha1.BuildJob
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &bj); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("BuildJob %q not found in namespace %q", name, ns))
		return
	}
	if err := s.Client.Delete(r.Context(), &bj); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting BuildJob: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
