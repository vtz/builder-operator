package controller

import (
	"testing"

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSyncStatusFromPipelineRun_SetsSucceeded(t *testing.T) {
	r := &SoftwareBuildReconciler{}
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "demo",
			Generation: 3,
		},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Destination: buildv1alpha1.DestinationSpec{
				Path: "/host-build/deployment",
			},
		},
	}

	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":    "Succeeded",
						"status":  "True",
						"reason":  "Succeeded",
						"message": "All tasks completed",
					},
				},
				"childReferences": []interface{}{
					map[string]interface{}{
						"name":             "taskrun-1",
						"pipelineTaskName": "build",
					},
				},
			},
		},
	}

	r.syncStatusFromPipelineRun(sb, pr)

	if sb.Status.Phase != buildv1alpha1.PhaseSucceeded {
		t.Fatalf("expected phase Succeeded, got %s", sb.Status.Phase)
	}
	if sb.Status.ArtifactURI != "/host-build/deployment" {
		t.Fatalf("expected artifact uri to be populated")
	}
	if len(sb.Status.Stages) != 1 {
		t.Fatalf("expected one stage status")
	}
}
