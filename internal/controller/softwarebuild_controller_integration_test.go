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
	"context"
	"testing"
	"time"

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	integrationPollInterval = 250 * time.Millisecond
	integrationTimeout      = 15 * time.Second
)

// TestReconcile_CreatesPipelineRunOnFirstReconcile verifies that the controller
// creates a PipelineRun and sets Status.CurrentPipelineRun when a new
// SoftwareBuild is reconciled for the first time.
func TestReconcile_CreatesPipelineRunOnFirstReconcile(t *testing.T) {
	if testClient == nil {
		t.Skip("integration test requires KUBEBUILDER_ASSETS to be set")
	}

	ctx := context.Background()
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-first-reconcile",
			Namespace: "default",
		},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Runtime: buildv1alpha1.RuntimeSpec{Image: "ubuntu:24.04"},
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypeHostPath,
				HostPath: &buildv1alpha1.HostPathSource{Path: "/src"},
			},
			Stages: buildv1alpha1.PipelineStages{
				Fetch:     buildv1alpha1.StageSpec{Command: "echo fetch"},
				Prebuild:  buildv1alpha1.StageSpec{Command: "echo pre"},
				Build:     buildv1alpha1.StageSpec{Command: "echo build"},
				Postbuild: buildv1alpha1.StageSpec{Command: "echo post"},
				Deploy:    buildv1alpha1.StageSpec{Command: "echo deploy"},
			},
			Destination: buildv1alpha1.DestinationSpec{
				Type: buildv1alpha1.DestinationTypeSharedFolder,
				Path: "/out",
			},
		},
	}

	if err := testClient.Create(ctx, sb); err != nil {
		t.Fatalf("failed to create SoftwareBuild: %v", err)
	}
	t.Cleanup(func() { _ = testClient.Delete(ctx, sb) })

	key := types.NamespacedName{Name: sb.Name, Namespace: sb.Namespace}
	if err := waitUntil(ctx, t, func() bool {
		var current buildv1alpha1.SoftwareBuild
		if err := testClient.Get(ctx, key, &current); err != nil {
			return false
		}
		return current.Status.CurrentPipelineRun != ""
	}); err != nil {
		t.Fatalf("timed out waiting for CurrentPipelineRun to be set: %v", err)
	}

	var current buildv1alpha1.SoftwareBuild
	if err := testClient.Get(ctx, key, &current); err != nil {
		t.Fatalf("failed to get SoftwareBuild: %v", err)
	}

	prName := current.Status.CurrentPipelineRun
	if prName == "" {
		t.Fatal("expected Status.CurrentPipelineRun to be set")
	}

	var pr unstructured.Unstructured
	pr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tekton.dev",
		Version: "v1",
		Kind:    "PipelineRun",
	})
	if err := testClient.Get(ctx, client.ObjectKey{Namespace: sb.Namespace, Name: prName}, &pr); err != nil {
		t.Fatalf("expected PipelineRun %q to exist: %v", prName, err)
	}

	if current.Status.Phase != buildv1alpha1.PhasePending {
		t.Fatalf("expected phase Pending after first reconcile, got %s", current.Status.Phase)
	}
}

// TestReconcile_PhaseFailedWhenPipelineRunMissing verifies that the controller
// transitions the SoftwareBuild phase to Failed when the referenced PipelineRun
// no longer exists.
func TestReconcile_PhaseFailedWhenPipelineRunMissing(t *testing.T) {
	if testClient == nil {
		t.Skip("integration test requires KUBEBUILDER_ASSETS to be set")
	}

	ctx := context.Background()
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-missing-pr",
			Namespace: "default",
		},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Runtime: buildv1alpha1.RuntimeSpec{Image: "ubuntu:24.04"},
			Source: buildv1alpha1.SourceSpec{
				Type:     buildv1alpha1.SourceTypeHostPath,
				HostPath: &buildv1alpha1.HostPathSource{Path: "/src"},
			},
			Stages: buildv1alpha1.PipelineStages{
				Fetch:     buildv1alpha1.StageSpec{Command: "echo fetch"},
				Prebuild:  buildv1alpha1.StageSpec{Command: "echo pre"},
				Build:     buildv1alpha1.StageSpec{Command: "echo build"},
				Postbuild: buildv1alpha1.StageSpec{Command: "echo post"},
				Deploy:    buildv1alpha1.StageSpec{Command: "echo deploy"},
			},
			Destination: buildv1alpha1.DestinationSpec{
				Type: buildv1alpha1.DestinationTypeSharedFolder,
				Path: "/out",
			},
		},
	}

	if err := testClient.Create(ctx, sb); err != nil {
		t.Fatalf("failed to create SoftwareBuild: %v", err)
	}
	t.Cleanup(func() { _ = testClient.Delete(ctx, sb) })

	key := types.NamespacedName{Name: sb.Name, Namespace: sb.Namespace}

	// Wait for the controller to create the PipelineRun and set the reference.
	if err := waitUntil(ctx, t, func() bool {
		var current buildv1alpha1.SoftwareBuild
		if err := testClient.Get(ctx, key, &current); err != nil {
			return false
		}
		return current.Status.CurrentPipelineRun != ""
	}); err != nil {
		t.Fatalf("timed out waiting for CurrentPipelineRun to be set: %v", err)
	}

	var current buildv1alpha1.SoftwareBuild
	if err := testClient.Get(ctx, key, &current); err != nil {
		t.Fatalf("failed to get SoftwareBuild: %v", err)
	}

	// Delete the PipelineRun to simulate it going missing.
	var pr unstructured.Unstructured
	pr.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRun"})
	pr.SetName(current.Status.CurrentPipelineRun)
	pr.SetNamespace(sb.Namespace)
	if err := testClient.Delete(ctx, &pr); err != nil {
		t.Fatalf("failed to delete PipelineRun: %v", err)
	}

	// The controller should detect the missing PipelineRun and set phase to Failed.
	if err := waitUntil(ctx, t, func() bool {
		var updated buildv1alpha1.SoftwareBuild
		if err := testClient.Get(ctx, key, &updated); err != nil {
			return false
		}
		return updated.Status.Phase == buildv1alpha1.PhaseFailed
	}); err != nil {
		t.Fatalf("timed out waiting for phase Failed: %v", err)
	}

	var updated buildv1alpha1.SoftwareBuild
	if err := testClient.Get(ctx, key, &updated); err != nil {
		t.Fatalf("failed to get updated SoftwareBuild: %v", err)
	}
	if updated.Status.FailureReason != "PipelineRunNotFound" {
		t.Fatalf("expected FailureReason PipelineRunNotFound, got %q", updated.Status.FailureReason)
	}
}

func waitUntil(ctx context.Context, t *testing.T, condition func() bool) error {
	t.Helper()
	deadline := time.Now().Add(integrationTimeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(integrationPollInterval):
		}
	}
	return context.DeadlineExceeded
}
