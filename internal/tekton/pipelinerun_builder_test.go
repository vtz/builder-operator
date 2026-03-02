package tekton

import (
	"testing"

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPipelineRun_UsesStageCommands(t *testing.T) {
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "ns1",
		},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Runtime: buildv1alpha1.RuntimeSpec{Image: "ubuntu:24.04"},
			Stages: buildv1alpha1.PipelineStages{
				Fetch:     buildv1alpha1.StageSpec{Command: "echo fetch"},
				Prebuild:  buildv1alpha1.StageSpec{Command: "echo pre"},
				Build:     buildv1alpha1.StageSpec{Command: "echo build"},
				Postbuild: buildv1alpha1.StageSpec{Command: "echo post"},
				Deploy:    buildv1alpha1.StageSpec{Command: "echo deploy"},
			},
		},
	}

	pr := BuildPipelineRun(sb)
	if pr.GetNamespace() != "ns1" {
		t.Fatalf("expected namespace ns1, got %s", pr.GetNamespace())
	}
	if pr.Object["kind"] != "PipelineRun" {
		t.Fatalf("expected kind PipelineRun")
	}

	spec, ok := pr.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected spec object")
	}
	params, ok := spec["params"].([]interface{})
	if !ok || len(params) < 6 {
		t.Fatalf("expected at least 6 params, got %d", len(params))
	}
}
