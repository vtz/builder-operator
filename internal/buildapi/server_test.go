package buildapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestServer(t *testing.T, objs ...runtime.Object) (*Server, *http.ServeMux) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := buildv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}
	runtimeObjs := make([]runtime.Object, len(objs))
	copy(runtimeObjs, objs)
	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(runtimeObjs...).WithStatusSubresource(&buildv1alpha1.BuildJob{}).Build()
	s := NewServer(cl, ":0", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs", s.handleList)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs", s.handleCreate)
	mux.HandleFunc("GET /v1/namespaces/{namespace}/buildjobs/{name}", s.handleGet)
	mux.HandleFunc("POST /v1/namespaces/{namespace}/buildjobs/{name}/run", s.handleRun)
	mux.HandleFunc("DELETE /v1/namespaces/{namespace}/buildjobs/{name}", s.handleDelete)
	return s, mux
}

func sampleBuildJob(name, ns string) *buildv1alpha1.BuildJob {
	return &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: buildv1alpha1.BuildJobSpec{
			Toolchain: buildv1alpha1.ToolchainSpec{Image: "test:latest"},
			Source:    buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit, Git: &buildv1alpha1.GitSource{URL: "https://github.com/test/repo", Revision: "main"}},
			Target:    buildv1alpha1.TargetSpec{Board: "nucleo", Platform: "zephyr", Architecture: "arm"},
			Stages:    []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
}

func TestHandleList_Empty(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/default/buildjobs", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp BuildJobListResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Items))
	}
}

func TestHandleList_WithItems(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	_, mux := newTestServer(t, bj)

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/builds/buildjobs", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp BuildJobListResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Name != "demo" {
		t.Fatalf("expected name demo, got %s", resp.Items[0].Name)
	}
}

func TestHandleGet_Found(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	_, mux := newTestServer(t, bj)

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/builds/buildjobs/demo", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp BuildJobSummary
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Name != "demo" {
		t.Fatalf("expected name demo, got %s", resp.Name)
	}
	if resp.Board != "nucleo" {
		t.Fatalf("expected board nucleo, got %s", resp.Board)
	}
}

func TestHandleGet_NotFound(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/builds/buildjobs/nope", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleCreate_New(t *testing.T) {
	_, mux := newTestServer(t)

	body := CreateBuildJobRequest{
		Name: "new-job",
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit, Git: &buildv1alpha1.GitSource{URL: "https://github.com/test/repo"}},
			Stages: []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp BuildJobSummary
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Name != "new-job" {
		t.Fatalf("expected name new-job, got %s", resp.Name)
	}
}

func TestHandleCreate_UpdateExisting(t *testing.T) {
	bj := sampleBuildJob("existing", "builds")
	_, mux := newTestServer(t, bj)

	body := CreateBuildJobRequest{
		Name: "existing",
		Spec: buildv1alpha1.BuildJobSpec{
			Toolchain: buildv1alpha1.ToolchainSpec{Image: "updated:latest"},
			Source:    buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit, Git: &buildv1alpha1.GitSource{URL: "https://github.com/test/repo"}},
			Stages:    []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for update, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp BuildJobSummary
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Image != "updated:latest" {
		t.Fatalf("expected updated image, got %s", resp.Image)
	}
}

func TestHandleCreate_MissingName(t *testing.T) {
	_, mux := newTestServer(t)
	body := CreateBuildJobRequest{Spec: buildv1alpha1.BuildJobSpec{}}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d", rr.Code)
	}
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d", rr.Code)
	}
}

func TestHandleRun_NotFound(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs/nope/run", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleRun_Success(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	_, mux := newTestServer(t, bj)

	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/builds/buildjobs/demo/run", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_Success(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	_, mux := newTestServer(t, bj)

	req := httptest.NewRequest(http.MethodDelete, "/v1/namespaces/builds/buildjobs/demo", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	_, mux := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/v1/namespaces/builds/buildjobs/nope", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	_, mux := newTestServer(t)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestToSummary_GitSource(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	bj.Status.Phase = buildv1alpha1.PhaseRunning
	bj.Status.CurrentPipelineRun = "demo-run1"
	summary := toSummary(bj)

	if summary.Source == nil {
		t.Fatal("expected non-nil source")
	}
	if summary.Source.Type != "git" {
		t.Fatalf("expected git source type, got %s", summary.Source.Type)
	}
	if summary.Source.URL != "https://github.com/test/repo" {
		t.Fatalf("expected repo URL, got %s", summary.Source.URL)
	}
	if summary.PipelineRun != "demo-run1" {
		t.Fatalf("expected pipelineRun demo-run1, got %s", summary.PipelineRun)
	}
}

func TestToSummary_PVCSource(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-job", Namespace: "ns"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypePVC},
			Stages: []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	summary := toSummary(bj)
	if summary.Source == nil || summary.Source.Type != "pvc" {
		t.Fatalf("expected pvc source type, got %+v", summary.Source)
	}
}

func TestToSummary_StageStatuses(t *testing.T) {
	bj := sampleBuildJob("demo", "builds")
	bj.Status.Stages = []buildv1alpha1.StageStatus{
		{Name: "build", State: "Succeeded", Message: "done"},
		{Name: "package", State: "Running", Message: "packaging"},
	}
	summary := toSummary(bj)
	if len(summary.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(summary.Stages))
	}
	if summary.Stages[0].Name != "build" || summary.Stages[0].State != "Succeeded" {
		t.Fatalf("unexpected first stage: %+v", summary.Stages[0])
	}
}

// Ensure unused import for context doesn't cause issues — used by fake client
var _ = context.Background
