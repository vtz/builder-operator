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

func TestSyncStatusFromPipelineRun_SetsFailed(t *testing.T) {
	r := &SoftwareBuildReconciler{}
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Generation: 1},
	}

	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":    "Succeeded",
						"status":  "False",
						"reason":  "TaskRunFailed",
						"message": "build task failed",
					},
				},
			},
		},
	}

	r.syncStatusFromPipelineRun(sb, pr)

	if sb.Status.Phase != buildv1alpha1.PhaseFailed {
		t.Fatalf("expected phase Failed, got %s", sb.Status.Phase)
	}
	if sb.Status.FailureReason != "TaskRunFailed" {
		t.Fatalf("expected FailureReason TaskRunFailed, got %s", sb.Status.FailureReason)
	}
	if len(sb.Status.Conditions) == 0 {
		t.Fatalf("expected at least one condition")
	}
	if sb.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected Ready condition to be False")
	}
}

func TestSyncStatusFromPipelineRun_SetsRunning(t *testing.T) {
	r := &SoftwareBuildReconciler{}
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Generation: 1},
	}

	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{},
			},
		},
	}

	r.syncStatusFromPipelineRun(sb, pr)

	if sb.Status.Phase != buildv1alpha1.PhaseRunning {
		t.Fatalf("expected phase Running, got %s", sb.Status.Phase)
	}
}

func TestMergeCondition_AddsNewCondition(t *testing.T) {
	newCond := buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "Pending", "not ready", 1)
	result := mergeCondition(nil, newCond)

	if len(result) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(result))
	}
	if result[0].Type != "Ready" {
		t.Fatalf("expected condition type Ready, got %s", result[0].Type)
	}
}

func TestMergeCondition_UpdatesExistingCondition(t *testing.T) {
	existing := []metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Old"},
	}
	updated := buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "Succeeded", "done", 2)
	result := mergeCondition(existing, updated)

	if len(result) != 1 {
		t.Fatalf("expected 1 condition after merge, got %d", len(result))
	}
	if result[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected condition status True after update, got %s", result[0].Status)
	}
	if result[0].Reason != "Succeeded" {
		t.Fatalf("expected reason Succeeded, got %s", result[0].Reason)
	}
}
