package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/centos-automotive-suite/bob/internal/buildapi"
)

func TestNew_TrimsTrailingSlash(t *testing.T) {
	c := New("https://example.com/api/", "tok", "ns")
	if c.BaseURL != "https://example.com/api" {
		t.Fatalf("expected trailing slash trimmed, got %q", c.BaseURL)
	}
}

func TestNew_SetsFields(t *testing.T) {
	c := New("https://api.test", "my-token", "my-ns")
	if c.Token != "my-token" {
		t.Errorf("token: got %q", c.Token)
	}
	if c.Namespace != "my-ns" {
		t.Errorf("namespace: got %q", c.Namespace)
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestClient_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/namespaces/builds/buildjobs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := buildapi.BuildJobListResponse{Items: []buildapi.BuildJobSummary{
			{Name: "job-1", Phase: "Running"},
			{Name: "job-2", Phase: "Succeeded"},
		}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "builds")
	items, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "job-1" {
		t.Errorf("first item name: %s", items[0].Name)
	}
}

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/namespaces/builds/buildjobs/demo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(buildapi.BuildJobSummary{Name: "demo", Phase: "Succeeded"})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "builds")
	got, err := c.Get(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "demo" {
		t.Errorf("name: got %q", got.Name)
	}
}

func TestClient_Get_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "", "builds")
	_, err := c.Get(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_Run(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/namespaces/ns/buildjobs/job/run" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(buildapi.BuildJobSummary{Name: "job", Phase: "Pending"})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "ns")
	got, err := c.Run(context.Background(), "job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Phase != "Pending" {
		t.Errorf("phase: got %q", got.Phase)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "ns")
	err := c.Delete(context.Background(), "job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Delete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New(srv.URL, "", "ns")
	err := c.Delete(context.Background(), "job")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestClient_Logs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/namespaces/ns/buildjobs/job/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte("line1\nline2\n"))
	}))
	defer srv.Close()

	c := New(srv.URL, "", "ns")
	reader, err := c.Logs(context.Background(), "job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	if string(data) != "line1\nline2\n" {
		t.Errorf("unexpected logs: %q", string(data))
	}
}

func TestClient_AuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token" {
			t.Errorf("expected bearer token, got %q", auth)
		}
		json.NewEncoder(w).Encode(buildapi.BuildJobListResponse{})
	}))
	defer srv.Close()

	c := New(srv.URL, "my-token", "ns")
	c.List(context.Background())
}

func TestClient_NoAuthWhenTokenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no auth header, got %q", auth)
		}
		json.NewEncoder(w).Encode(buildapi.BuildJobListResponse{})
	}))
	defer srv.Close()

	c := New(srv.URL, "", "ns")
	c.List(context.Background())
}
