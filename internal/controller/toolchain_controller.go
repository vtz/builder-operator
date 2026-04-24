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
	"fmt"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ToolchainReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ToolchainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tc buildv1alpha1.Toolchain
	if err := r.Get(ctx, req.NamespacedName, &tc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if tc.Spec.Build == nil {
		if tc.Status.Phase != buildv1alpha1.ToolchainPhaseReady {
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseReady
			tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "ExternalImage", "Using externally provided image", tc.Generation))
			if err := r.Status().Update(ctx, &tc); err != nil {
				return ctrl.Result{}, fmt.Errorf("updating toolchain status: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// Build is requested — check if we already have a TaskRun for this generation.
	if tc.Status.CurrentBuildRun != "" {
		var tr unstructured.Unstructured
		tr.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "TaskRun"})
		if err := r.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Status.CurrentBuildRun}, &tr); err != nil {
			if apierrors.IsNotFound(err) {
				tc.Status.Phase = buildv1alpha1.ToolchainPhaseFailed
				tc.Status.CurrentBuildRun = ""
				tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
					buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "TaskRunMissing", "Build TaskRun no longer exists", tc.Generation))
				if updateErr := r.Status().Update(ctx, &tc); updateErr != nil {
					return ctrl.Result{RequeueAfter: 2 * time.Second}, fmt.Errorf("updating status for missing TaskRun: %w", updateErr)
				}
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("fetching TaskRun %q: %w", tc.Status.CurrentBuildRun, err)
		}

		phase := r.taskRunPhase(&tr)
		switch phase {
		case "Succeeded":
			digest := r.extractTaskRunResult(&tr, "IMAGE_DIGEST")
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseReady
			tc.Status.ResolvedDigest = digest
			tc.Status.LastBuildTime = time.Now().UTC().Format(time.RFC3339)
			tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "BuildSucceeded", "Toolchain image built and pushed", tc.Generation))
			if err := r.Status().Update(ctx, &tc); err != nil {
				return ctrl.Result{}, fmt.Errorf("updating status after build success: %w", err)
			}
			logger.Info("toolchain image built successfully", "image", tc.Spec.Image)
			return ctrl.Result{}, nil
		case "Failed":
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseFailed
			tc.Status.CurrentBuildRun = ""
			tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "BuildFailed", "Toolchain image build failed — update spec to retry", tc.Generation))
			if err := r.Status().Update(ctx, &tc); err != nil {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, fmt.Errorf("updating status after build failure: %w", err)
			}
			return ctrl.Result{}, nil
		default:
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Don't retry automatically after failure — require a spec change.
	if tc.Status.Phase == buildv1alpha1.ToolchainPhaseFailed {
		return ctrl.Result{}, nil
	}

	// No existing build — create a Buildah TaskRun.
	trName := fmt.Sprintf("tc-%s-%d", tc.Name, time.Now().Unix())
	taskRun := r.buildTaskRun(&tc, trName)
	if err := ctrl.SetControllerReference(&tc, taskRun, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting controller reference: %w", err)
	}
	if err := r.Create(ctx, taskRun); err != nil {
		if apierrors.IsAlreadyExists(err) {
			tc.Status.CurrentBuildRun = trName
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseBuilding
			if updateErr := r.Status().Update(ctx, &tc); updateErr != nil {
				logger.Error(updateErr, "status update failed after adopting existing TaskRun")
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("creating build TaskRun: %w", err)
	}

	tc.Status.CurrentBuildRun = trName
	tc.Status.Phase = buildv1alpha1.ToolchainPhaseBuilding
	tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
		buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "Building", "Building toolchain image", tc.Generation))
	if err := r.Status().Update(ctx, &tc); err != nil {
		logger.Error(err, "status update failed after creating TaskRun")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	logger.Info("created toolchain build TaskRun", "name", trName, "image", tc.Spec.Image)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

const BuildahImage = "quay.io/buildah/stable:v1.39.0"

func (r *ToolchainReconciler) buildTaskRun(tc *buildv1alpha1.Toolchain, name string) *unstructured.Unstructured {
	build := tc.Spec.Build

	const budFlags = "--storage-driver=vfs --isolation=chroot"
	const pushFlags = "--storage-driver=vfs"

	var script string
	if build.Dockerfile != "" {
		script = fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
CONTEXT_DIR=$(mktemp -d)
cat > "$CONTEXT_DIR/Dockerfile" << 'DOCKERFILE_EOF'
%s
DOCKERFILE_EOF
buildah bud %s -t %s -f "$CONTEXT_DIR/Dockerfile" "$CONTEXT_DIR"
buildah push %s %s
echo "Toolchain image pushed: %s"
`, build.Dockerfile, budFlags, tc.Spec.Image, pushFlags, tc.Spec.Image, tc.Spec.Image)
	} else if build.ContextGit != nil {
		dockerfilePath := build.DockerfilePath
		if dockerfilePath == "" {
			dockerfilePath = "Dockerfile"
		}
		rev := build.ContextGit.Revision
		if rev == "" {
			rev = "main"
		}
		script = fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
CONTEXT_DIR=$(mktemp -d)
git clone --branch '%s' --depth 1 '%s' "$CONTEXT_DIR"
buildah bud %s -t %s -f "$CONTEXT_DIR/%s" "$CONTEXT_DIR"
buildah push %s %s
echo "Toolchain image pushed: %s"
`, rev, build.ContextGit.URL, budFlags, tc.Spec.Image, dockerfilePath, pushFlags, tc.Spec.Image, tc.Spec.Image)
	}

	obj := map[string]interface{}{
		"apiVersion": "tekton.dev/v1",
		"kind":       "TaskRun",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": tc.Namespace,
			"labels": map[string]interface{}{
				"builder.sdv.cloud.redhat.com/toolchain": tc.Name,
			},
		},
		"spec": map[string]interface{}{
			"serviceAccountName": "pipeline",
			"taskSpec": map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{
						"name":  "build-push",
						"image": BuildahImage,
						"securityContext": map[string]interface{}{
							"privileged": true,
							"runAsUser":  int64(0),
						},
						"script": script,
					},
				},
			},
		},
	}

	return &unstructured.Unstructured{Object: obj}
}

// extractTaskRunResult iterates the TaskRun status.results array and returns
// the value for the named result, or empty string if not found.
func (r *ToolchainReconciler) extractTaskRunResult(tr *unstructured.Unstructured, name string) string {
	results, _, _ := unstructured.NestedSlice(tr.Object, "status", "results")
	for _, entry := range results {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		n, _, _ := unstructured.NestedString(m, "name")
		v, _, _ := unstructured.NestedString(m, "value")
		if n == name && v != "" {
			return v
		}
	}
	return ""
}

func (r *ToolchainReconciler) taskRunPhase(tr *unstructured.Unstructured) string {
	conditions, _, _ := unstructured.NestedSlice(tr.Object, "status", "conditions")
	for _, c := range conditions {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _, _ := unstructured.NestedString(m, "type")
		s, _, _ := unstructured.NestedString(m, "status")
		if t == "Succeeded" {
			switch s {
			case "True":
				return "Succeeded"
			case "False":
				return "Failed"
			}
		}
	}
	return "Running"
}

var taskRunGVK = schema.GroupVersionKind{
	Group:   "tekton.dev",
	Version: "v1",
	Kind:    "TaskRun",
}

func (r *ToolchainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	taskRun := &unstructured.Unstructured{}
	taskRun.SetGroupVersionKind(taskRunGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(&buildv1alpha1.Toolchain{}).
		Owns(taskRun).
		Complete(r)
}
