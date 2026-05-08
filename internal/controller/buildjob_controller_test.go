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
	"fmt"
	"testing"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSyncStatusFromPipelineRun_SetsSucceeded(t *testing.T) {
	r := &BuildJobReconciler{}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "demo",
			Generation: 3,
		},
		Spec: buildv1alpha1.BuildJobSpec{
			Artifacts: buildv1alpha1.ArtifactSpec{
				Path: "/workspace/artifacts",
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
				"results": []interface{}{
					map[string]interface{}{
						"name":  "commit-sha",
						"value": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
					},
				},
			},
		},
	}

	r.syncStatusFromPipelineRun(bj, pr)

	if bj.Status.Phase != buildv1alpha1.PhaseSucceeded {
		t.Fatalf("expected phase Succeeded, got %s", bj.Status.Phase)
	}
	expectedArtifactURI := fmt.Sprintf("/v1/namespaces/%s/buildjobs/%s/artifacts", bj.Namespace, bj.Name)
	if bj.Status.ArtifactURI != expectedArtifactURI {
		t.Fatalf("expected artifact URI %q, got %q", expectedArtifactURI, bj.Status.ArtifactURI)
	}
	if len(bj.Status.Stages) != 1 {
		t.Fatalf("expected one stage status, got %d", len(bj.Status.Stages))
	}
	if bj.Status.CommitSHA != "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" {
		t.Fatalf("expected commitSHA to be populated, got %q", bj.Status.CommitSHA)
	}
}

func TestSyncStatusFromPipelineRun_NoCommitSHAWithoutResults(t *testing.T) {
	r := &BuildJobReconciler{}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Generation: 1},
	}
	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Succeeded",
						"status": "True",
						"reason": "Succeeded",
					},
				},
			},
		},
	}
	r.syncStatusFromPipelineRun(bj, pr)
	if bj.Status.CommitSHA != "" {
		t.Fatalf("expected empty commitSHA without results, got %q", bj.Status.CommitSHA)
	}
}

func TestSyncStatusFromPipelineRun_SetsFailed(t *testing.T) {
	r := &BuildJobReconciler{}
	bj := &buildv1alpha1.BuildJob{
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

	r.syncStatusFromPipelineRun(bj, pr)

	if bj.Status.Phase != buildv1alpha1.PhaseFailed {
		t.Fatalf("expected phase Failed, got %s", bj.Status.Phase)
	}
	if bj.Status.FailureReason != "TaskRunFailed" {
		t.Fatalf("expected FailureReason TaskRunFailed, got %s", bj.Status.FailureReason)
	}
	if len(bj.Status.Conditions) == 0 {
		t.Fatalf("expected at least one condition")
	}
	if bj.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected Ready condition to be False")
	}
}

func TestSyncStatusFromPipelineRun_SetsRunning(t *testing.T) {
	r := &BuildJobReconciler{}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Generation: 1},
	}

	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"startTime":  "2026-04-14T10:00:00Z",
				"conditions": []interface{}{},
			},
		},
	}

	r.syncStatusFromPipelineRun(bj, pr)

	if bj.Status.Phase != buildv1alpha1.PhaseRunning {
		t.Fatalf("expected phase Running, got %s", bj.Status.Phase)
	}
}

func TestSyncStatusFromPipelineRun_DistinguishesPendingFromRunning(t *testing.T) {
	r := &BuildJobReconciler{}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Generation: 1},
	}

	pr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{},
			},
		},
	}

	r.syncStatusFromPipelineRun(bj, pr)

	if bj.Status.Phase != buildv1alpha1.PhasePending {
		t.Fatalf("expected phase Pending when no startTime, got %s", bj.Status.Phase)
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
		t.Fatalf("expected condition status True after update")
	}
	if result[0].Reason != "Succeeded" {
		t.Fatalf("expected reason Succeeded, got %s", result[0].Reason)
	}
}
