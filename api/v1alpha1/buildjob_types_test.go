package v1alpha1

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildJobSpec_JSONRoundTrip(t *testing.T) {
	spec := BuildJobSpec{
		Toolchain: ToolchainSpec{Image: "ghcr.io/zephyr:latest"},
		Source: SourceSpec{
			Type: SourceTypeGit,
			Git:  &GitSource{URL: "https://github.com/test/repo", Revision: "main"},
		},
		Target: TargetSpec{
			Board:        "nucleo_h755zi_q",
			Platform:     "zephyr",
			Architecture: "arm",
		},
		Stages: []NamedStage{
			{Name: "fetch", StageSpec: StageSpec{Command: "west init && west update"}},
			{Name: "build", StageSpec: StageSpec{Command: "west build -b $BOB_BOARD app"}},
		},
		Caches: []CacheMount{
			{Name: "ccache", MountPath: "/root/.ccache"},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded BuildJobSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Toolchain.Image != spec.Toolchain.Image {
		t.Errorf("toolchain image: got %q, want %q", decoded.Toolchain.Image, spec.Toolchain.Image)
	}
	if decoded.Source.Type != spec.Source.Type {
		t.Errorf("source type: got %q, want %q", decoded.Source.Type, spec.Source.Type)
	}
	if decoded.Source.Git.URL != spec.Source.Git.URL {
		t.Errorf("git URL: got %q, want %q", decoded.Source.Git.URL, spec.Source.Git.URL)
	}
	if decoded.Target.Board != spec.Target.Board {
		t.Errorf("board: got %q, want %q", decoded.Target.Board, spec.Target.Board)
	}
	if len(decoded.Stages) != len(spec.Stages) {
		t.Errorf("stages count: got %d, want %d", len(decoded.Stages), len(spec.Stages))
	}
	if len(decoded.Caches) != 1 || decoded.Caches[0].Name != "ccache" {
		t.Errorf("caches: got %+v", decoded.Caches)
	}
}

func TestBuildJobSpec_PVCSource(t *testing.T) {
	spec := BuildJobSpec{
		Source: SourceSpec{
			Type: SourceTypePVC,
			PVC:  &PVCSource{ClaimName: "my-pvc", Path: "/data"},
		},
		Stages: []NamedStage{
			{Name: "build", StageSpec: StageSpec{Command: "make"}},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded BuildJobSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Source.Type != SourceTypePVC {
		t.Errorf("expected pvc source type, got %q", decoded.Source.Type)
	}
	if decoded.Source.PVC.ClaimName != "my-pvc" {
		t.Errorf("expected claim my-pvc, got %q", decoded.Source.PVC.ClaimName)
	}
}

func TestBuildJobStatus_JSONRoundTrip(t *testing.T) {
	status := BuildJobStatus{
		Phase:              PhaseRunning,
		CurrentPipelineRun: "demo-run1",
		CommitSHA:          "abc123def456",
		ArtifactURI:        "/workspace/artifacts",
		RunCount:           3,
		LastRunAt:          "2026-04-14T10:00:00Z",
		Stages: []StageStatus{
			{Name: "build", State: "Running", Message: "compiling"},
		},
		Conditions: []metav1.Condition{
			NewCondition("Ready", metav1.ConditionFalse, "Running", "build in progress", 1),
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded BuildJobStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Phase != PhaseRunning {
		t.Errorf("phase: got %q, want %q", decoded.Phase, PhaseRunning)
	}
	if decoded.CurrentPipelineRun != "demo-run1" {
		t.Errorf("pipelineRun: got %q", decoded.CurrentPipelineRun)
	}
	if decoded.CommitSHA != "abc123def456" {
		t.Errorf("commitSHA: got %q, want abc123def456", decoded.CommitSHA)
	}
	if decoded.RunCount != 3 {
		t.Errorf("runCount: got %d, want 3", decoded.RunCount)
	}
	if decoded.LastRunAt != "2026-04-14T10:00:00Z" {
		t.Errorf("lastRunAt: got %q", decoded.LastRunAt)
	}
	if len(decoded.Stages) != 1 {
		t.Errorf("stages: got %d", len(decoded.Stages))
	}
}

func TestNewCondition(t *testing.T) {
	c := NewCondition("Ready", metav1.ConditionTrue, "Succeeded", "all done", 5)
	if c.Type != "Ready" {
		t.Errorf("type: got %q", c.Type)
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("status: got %q", c.Status)
	}
	if c.Reason != "Succeeded" {
		t.Errorf("reason: got %q", c.Reason)
	}
	if c.ObservedGeneration != 5 {
		t.Errorf("observedGeneration: got %d", c.ObservedGeneration)
	}
}

func TestPhaseConstants(t *testing.T) {
	phases := map[BuildJobPhase]string{
		PhasePending:   "Pending",
		PhaseRunning:   "Running",
		PhaseSucceeded: "Succeeded",
		PhaseFailed:    "Failed",
	}
	for phase, expected := range phases {
		if string(phase) != expected {
			t.Errorf("phase %q != %q", phase, expected)
		}
	}
}

func TestSourceTypeConstants(t *testing.T) {
	if string(SourceTypeGit) != "git" {
		t.Errorf("SourceTypeGit = %q", SourceTypeGit)
	}
	if string(SourceTypePVC) != "pvc" {
		t.Errorf("SourceTypePVC = %q", SourceTypePVC)
	}
}
