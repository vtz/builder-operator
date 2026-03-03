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

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	"github.com/example/builder-operator/internal/tekton"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var pipelineRunGVK = schema.GroupVersionKind{
	Group:   "tekton.dev",
	Version: "v1",
	Kind:    "PipelineRun",
}

// SoftwareBuildReconciler reconciles a SoftwareBuild object.
type SoftwareBuildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SoftwareBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sb buildv1alpha1.SoftwareBuild
	if err := r.Get(ctx, req.NamespacedName, &sb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if sb.Status.CurrentPipelineRun == "" {
		pipelineRun := tekton.BuildPipelineRun(&sb)
		if err := ctrl.SetControllerReference(&sb, pipelineRun, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, pipelineRun); err != nil {
			return ctrl.Result{}, err
		}

		sb.Status.CurrentPipelineRun = pipelineRun.GetName()
		sb.Status.Phase = buildv1alpha1.PhasePending
		sb.Status.Conditions = mergeCondition(
			sb.Status.Conditions,
			buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "PipelineRunCreated", "PipelineRun created for SoftwareBuild", sb.Generation),
		)
		if err := r.Status().Update(ctx, &sb); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("created PipelineRun", "name", pipelineRun.GetName())
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	var pr unstructured.Unstructured
	pr.SetGroupVersionKind(pipelineRunGVK)
	if err := r.Get(ctx, client.ObjectKey{Namespace: sb.Namespace, Name: sb.Status.CurrentPipelineRun}, &pr); err != nil {
		if apierrors.IsNotFound(err) {
			sb.Status.Phase = buildv1alpha1.PhaseFailed
			sb.Status.FailureReason = "PipelineRunNotFound"
			sb.Status.Conditions = mergeCondition(
				sb.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "PipelineRunMissing", "Referenced PipelineRun no longer exists", sb.Generation),
			)
			_ = r.Status().Update(ctx, &sb)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.syncStatusFromPipelineRun(&sb, &pr)
	if err := r.Status().Update(ctx, &sb); err != nil {
		return ctrl.Result{}, err
	}

	if sb.Status.Phase == buildv1alpha1.PhaseRunning || sb.Status.Phase == buildv1alpha1.PhasePending {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SoftwareBuildReconciler) syncStatusFromPipelineRun(sb *buildv1alpha1.SoftwareBuild, pr *unstructured.Unstructured) {
	conditions, _, _ := unstructured.NestedSlice(pr.Object, "status", "conditions")
	phase := buildv1alpha1.PhaseRunning
	readyStatus := metav1.ConditionFalse
	reason := "Running"
	message := "PipelineRun is in progress"

	for _, c := range conditions {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _, _ := unstructured.NestedString(m, "type")
		s, _, _ := unstructured.NestedString(m, "status")
		rn, _, _ := unstructured.NestedString(m, "reason")
		msg, _, _ := unstructured.NestedString(m, "message")
		if t == "Succeeded" {
			switch s {
			case "True":
				phase = buildv1alpha1.PhaseSucceeded
				readyStatus = metav1.ConditionTrue
				reason = rn
				message = msg
			case "False":
				phase = buildv1alpha1.PhaseFailed
				readyStatus = metav1.ConditionFalse
				reason = rn
				message = msg
				sb.Status.FailureReason = rn
			default:
				phase = buildv1alpha1.PhaseRunning
			}
		}
	}

	sb.Status.Phase = phase
	sb.Status.Conditions = mergeCondition(sb.Status.Conditions, buildv1alpha1.NewCondition("Ready", readyStatus, reason, message, sb.Generation))

	childRefs, _, _ := unstructured.NestedSlice(pr.Object, "status", "childReferences")
	stageStatuses := make([]buildv1alpha1.StageStatus, 0, len(childRefs))
	for _, cr := range childRefs {
		m, ok := cr.(map[string]interface{})
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(m, "name")
		pipelineTaskName, _, _ := unstructured.NestedString(m, "pipelineTaskName")
		stageStatuses = append(stageStatuses, buildv1alpha1.StageStatus{
			Name:    pipelineTaskName,
			State:   "Created",
			Message: fmt.Sprintf("TaskRun: %s", name),
		})
	}
	sb.Status.Stages = stageStatuses

	if sb.Spec.Destination.Path != "" {
		sb.Status.ArtifactURI = sb.Spec.Destination.Path
	}
}

func mergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	updated := false
	for i := range conditions {
		if conditions[i].Type == newCondition.Type {
			conditions[i] = newCondition
			updated = true
			break
		}
	}
	if !updated {
		conditions = append(conditions, newCondition)
	}
	return conditions
}

// SetupWithManager sets up the controller with the Manager.
func (r *SoftwareBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildv1alpha1.SoftwareBuild{}).
		Complete(r)
}
