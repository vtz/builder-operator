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
	"encoding/base64"
	"fmt"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"github.com/centos-automotive-suite/bob/internal/tekton"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ToolchainReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *ToolchainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tc buildv1alpha1.Toolchain
	if err := r.Get(ctx, req.NamespacedName, &tc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	tc.Status.ObservedGeneration = tc.Generation

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
		case conditionTypeSucceeded:
			digest := r.extractImageDigest(&tr)
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseReady
			tc.Status.ResolvedDigest = digest
			tc.Status.LastBuildTime = time.Now().UTC().Format(time.RFC3339)
			tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "BuildSucceeded", "Toolchain image built and pushed", tc.Generation))
			if err := r.Status().Update(ctx, &tc); err != nil {
				return ctrl.Result{}, fmt.Errorf("updating status after build success: %w", err)
			}
			logger.Info("toolchain image built successfully", "image", tc.Spec.Image)
			if r.Recorder != nil {
				r.Recorder.Eventf(&tc, corev1.EventTypeNormal, "BuildSucceeded", "Toolchain image %s built and pushed", tc.Spec.Image)
			}
			ToolchainBuildsTotal.WithLabelValues(tc.Namespace, "succeeded").Inc()
			return ctrl.Result{}, nil
		case CachePhaseFailed:
			tc.Status.Phase = buildv1alpha1.ToolchainPhaseFailed
			tc.Status.CurrentBuildRun = ""
			tc.Status.Conditions = mergeCondition(tc.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "BuildFailed", "Toolchain image build failed — update spec to retry", tc.Generation))
			if err := r.Status().Update(ctx, &tc); err != nil {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, fmt.Errorf("updating status after build failure: %w", err)
			}
			if r.Recorder != nil {
				r.Recorder.Event(&tc, corev1.EventTypeWarning, "BuildFailed", "Toolchain image build failed")
			}
			ToolchainBuildsTotal.WithLabelValues(tc.Namespace, "failed").Inc()
			return ctrl.Result{}, nil
		default:
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	if tc.Status.Phase == buildv1alpha1.ToolchainPhaseFailed {
		return ctrl.Result{}, nil
	}

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
	if r.Recorder != nil {
		r.Recorder.Eventf(&tc, corev1.EventTypeNormal, "BuildStarted", "Started toolchain build TaskRun %s for image %s", trName, tc.Spec.Image)
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

const buildahImage = "quay.io/buildah/stable:v1.39.0"

func (r *ToolchainReconciler) buildTaskRun(tc *buildv1alpha1.Toolchain, name string) *unstructured.Unstructured {
	build := tc.Spec.Build

	const budFlags = "--storage-driver=vfs --isolation=rootless"
	const pushFlags = "--storage-driver=vfs"

	var script string
	if build.Dockerfile != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(build.Dockerfile))
		script = fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
CONTEXT_DIR=$(mktemp -d)
echo '%s' | base64 -d > "$CONTEXT_DIR/Dockerfile"
buildah bud %s -t %s -f "$CONTEXT_DIR/Dockerfile" "$CONTEXT_DIR"
buildah push %s %s
echo "Toolchain image pushed: %s"
`, encoded, budFlags, tc.Spec.Image, pushFlags, tc.Spec.Image, tc.Spec.Image)
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
git clone --branch %s --depth 1 %s "$CONTEXT_DIR"
buildah bud %s -t %s -f "$CONTEXT_DIR/%s" "$CONTEXT_DIR"
buildah push %s %s
echo "Toolchain image pushed: %s"
`, tekton.ShellQuote(rev), tekton.ShellQuote(build.ContextGit.URL), budFlags, tc.Spec.Image, dockerfilePath, pushFlags, tc.Spec.Image, tc.Spec.Image)
	}

	obj := map[string]interface{}{
		"apiVersion": "tekton.dev/v1",
		"kind":       "TaskRun",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": tc.Namespace,
			"labels": map[string]interface{}{
				buildv1alpha1.LabelToolchain: tc.Name,
			},
		},
		"spec": map[string]interface{}{
			"serviceAccountName": buildv1alpha1.DefaultServiceAccount,
			"taskSpec": map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{
						"name":  "build-push",
						"image": buildahImage,
						"securityContext": map[string]interface{}{
							"runAsNonRoot": true,
							"runAsUser":    int64(1000),
						},
						"script": script,
					},
				},
			},
		},
	}

	return &unstructured.Unstructured{Object: obj}
}
func (r *ToolchainReconciler) extractImageDigest(tr *unstructured.Unstructured) string {
	results, _, _ := unstructured.NestedSlice(tr.Object, "status", "results")
	for _, entry := range results {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		n, _, _ := unstructured.NestedString(m, "name")
		v, _, _ := unstructured.NestedString(m, "value")
		if n == "IMAGE_DIGEST" && v != "" {
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
		if t == conditionTypeSucceeded {
			switch s {
			case conditionStatusTrue:
				return conditionTypeSucceeded
			case conditionStatusFalse:
				return CachePhaseFailed
			}
		}
	}
	return conditionReasonRunning
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
