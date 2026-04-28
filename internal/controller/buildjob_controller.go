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
	"github.com/centos-automotive-suite/bob/internal/tekton"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

type BuildJobReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	PipelineConfig tekton.PipelineConfig
}

const runAtAnnotation = "builder.sdv.cloud.redhat.com/run-at"

func (r *BuildJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var bj buildv1alpha1.BuildJob
	if err := r.Get(ctx, req.NamespacedName, &bj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	needsNewRun := bj.Status.CurrentPipelineRun == ""
	if !needsNewRun {
		runAt := bj.Annotations[runAtAnnotation]
		if runAt != "" && runAt != bj.Status.LastRunAt {
			needsNewRun = true
		}
	}

	if needsNewRun {
		// Guard against runaway loop: if a previous reconcile created a
		// PipelineRun but the status update failed, we'd re-enter here
		// and create another. Check for an existing active PipelineRun first.
		if active := r.findActivePipelineRun(ctx, &bj); active != nil {
			logger.Info("adopting existing PipelineRun", "name", active.GetName())
			bj.Status.CurrentPipelineRun = active.GetName()
			bj.Status.Phase = buildv1alpha1.PhasePending
			bj.Status.LastRunAt = bj.Annotations[runAtAnnotation]
			r.syncStatusFromPipelineRun(&bj, active)
			if err := r.Status().Update(ctx, &bj); err != nil {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		if err := r.ensureCachePVCs(ctx, &bj); err != nil {
			return ctrl.Result{}, fmt.Errorf("ensuring cache PVCs: %w", err)
		}

		runN := r.nextRunNumber(ctx, &bj)
		pipelineRun := tekton.BuildPipelineRunWithConfig(&bj, runN, r.PipelineConfig)

		nodeSelector := map[string]interface{}{}

		if k8sArch := archToK8s(bj.Spec.Target.Architecture); k8sArch != "" {
			nodeSelector["kubernetes.io/arch"] = k8sArch
		}

		if len(bj.Spec.Caches) > 0 {
			if cacheSelector := r.cacheNodeSelector(ctx, &bj); cacheSelector != nil {
				for k, v := range cacheSelector {
					nodeSelector[k] = v
				}
			}
		}

		if len(nodeSelector) > 0 {
			if err := unstructured.SetNestedField(pipelineRun.Object, nodeSelector, "spec", "taskRunTemplate", "podTemplate", "nodeSelector"); err != nil {
				logger.Error(err, "failed to set nodeSelector")
			}
		}
		if err := ctrl.SetControllerReference(&bj, pipelineRun, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("setting controller reference: %w", err)
		}
		if err := r.Create(ctx, pipelineRun); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			return ctrl.Result{}, fmt.Errorf("creating PipelineRun: %w", err)
		}

		bj.Status.CurrentPipelineRun = pipelineRun.GetName()
		bj.Status.RunCount = runN
		bj.Status.Phase = buildv1alpha1.PhasePending
		bj.Status.LastRunAt = bj.Annotations[runAtAnnotation]
		bj.Status.FailureReason = ""
		bj.Status.Conditions = mergeCondition(
			bj.Status.Conditions,
			buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "PipelineRunCreated", "PipelineRun created for BuildJob", bj.Generation),
		)
		if err := r.Status().Update(ctx, &bj); err != nil {
			logger.Error(err, "status update failed after creating PipelineRun; will adopt on next reconcile", "pipelineRun", pipelineRun.GetName())
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		logger.Info("created PipelineRun", "name", pipelineRun.GetName(), "run", runN)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	var pr unstructured.Unstructured
	pr.SetGroupVersionKind(pipelineRunGVK)
	if err := r.Get(ctx, client.ObjectKey{Namespace: bj.Namespace, Name: bj.Status.CurrentPipelineRun}, &pr); err != nil {
		if apierrors.IsNotFound(err) {
			bj.Status.Phase = buildv1alpha1.PhaseFailed
			bj.Status.FailureReason = "PipelineRunNotFound"
			bj.Status.Conditions = mergeCondition(
				bj.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "PipelineRunMissing", "Referenced PipelineRun no longer exists", bj.Generation),
			)
			if err := r.Status().Update(ctx, &bj); err != nil {
				return ctrl.Result{}, fmt.Errorf("updating status for missing PipelineRun: %w", err)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching PipelineRun %q: %w", bj.Status.CurrentPipelineRun, err)
	}

	r.syncStatusFromPipelineRun(&bj, &pr)
	if err := r.Status().Update(ctx, &bj); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status from PipelineRun: %w", err)
	}

	if bj.Status.Phase == buildv1alpha1.PhaseRunning || bj.Status.Phase == buildv1alpha1.PhasePending {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *BuildJobReconciler) syncStatusFromPipelineRun(bj *buildv1alpha1.BuildJob, pr *unstructured.Unstructured) {
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
				bj.Status.FailureReason = rn
			default:
				phase = buildv1alpha1.PhaseRunning
			}
		}
	}

	// Distinguish Pending from Running by checking if startTime exists
	if phase == buildv1alpha1.PhaseRunning {
		_, found, _ := unstructured.NestedString(pr.Object, "status", "startTime")
		if !found {
			phase = buildv1alpha1.PhasePending
		}
	}

	bj.Status.Phase = phase
	bj.Status.Conditions = mergeCondition(bj.Status.Conditions, buildv1alpha1.NewCondition("Ready", readyStatus, reason, message, bj.Generation))

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
	bj.Status.Stages = stageStatuses

	if bj.Spec.Artifacts.Path != "" {
		bj.Status.ArtifactURI = bj.Spec.Artifacts.Path
	}

	results, _, _ := unstructured.NestedSlice(pr.Object, "status", "results")
	for _, r := range results {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(m, "name")
		value, _, _ := unstructured.NestedString(m, "value")
		if name == "commit-sha" && value != "" {
			bj.Status.CommitSHA = value
		}
	}
}

func mergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == newCondition.Type {
			conditions[i] = newCondition
			return conditions
		}
	}
	return append(conditions, newCondition)
}

func (r *BuildJobReconciler) findActivePipelineRun(ctx context.Context, bj *buildv1alpha1.BuildJob) *unstructured.Unstructured {
	var prList unstructured.UnstructuredList
	prList.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRunList"})
	if err := r.List(ctx, &prList, client.InNamespace(bj.Namespace), client.MatchingLabels{
		"builder.sdv.cloud.redhat.com/buildjob": bj.Name,
	}); err != nil || len(prList.Items) == 0 {
		return nil
	}

	for i := range prList.Items {
		pr := &prList.Items[i]
		conditions, _, _ := unstructured.NestedSlice(pr.Object, "status", "conditions")
		terminated := false
		for _, c := range conditions {
			m, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			t, _, _ := unstructured.NestedString(m, "type")
			s, _, _ := unstructured.NestedString(m, "status")
			if t == "Succeeded" && (s == "True" || s == "False") {
				terminated = true
				break
			}
		}
		if !terminated {
			return pr
		}
	}
	return nil
}

func (r *BuildJobReconciler) nextRunNumber(ctx context.Context, bj *buildv1alpha1.BuildJob) int64 {
	var prList unstructured.UnstructuredList
	prList.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRunList"})
	if err := r.List(ctx, &prList, client.InNamespace(bj.Namespace), client.MatchingLabels{
		"builder.sdv.cloud.redhat.com/buildjob": bj.Name,
	}); err != nil {
		return bj.Status.RunCount + 1
	}
	n := int64(len(prList.Items)) + 1
	if n <= bj.Status.RunCount {
		n = bj.Status.RunCount + 1
	}
	return n
}

// archToK8s maps BuildJob architecture values to Kubernetes node label values.
func archToK8s(arch string) string {
	switch arch {
	case "arm":
		return "arm64"
	case "x86":
		return "amd64"
	case "native":
		return ""
	default:
		return ""
	}
}

// cacheNodeSelector returns a nodeSelector map that pins pods to the same
// topology zone as the cache PV. This prevents scheduling failures when the
// cache PVC (RWO) is bound to a node in a different zone than the workspace.
func (r *BuildJobReconciler) cacheNodeSelector(ctx context.Context, bj *buildv1alpha1.BuildJob) map[string]interface{} {
	logger := log.FromContext(ctx)
	pvcName := tekton.SharedCachePVCName()

	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, client.ObjectKey{Namespace: bj.Namespace, Name: pvcName}, &pvc); err != nil {
		return nil
	}
	if pvc.Spec.VolumeName == "" {
		return nil
	}

	var pv corev1.PersistentVolume
	if err := r.Get(ctx, client.ObjectKey{Name: pvc.Spec.VolumeName}, &pv); err != nil {
		return nil
	}
	if pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		return nil
	}

	selector := map[string]interface{}{}
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Operator == corev1.NodeSelectorOpIn && len(expr.Values) == 1 {
				selector[expr.Key] = expr.Values[0]
			}
		}
	}
	if len(selector) > 0 {
		logger.Info("pinning pipeline to cache PV zone", "nodeSelector", selector)
		return selector
	}
	return nil
}

func (r *BuildJobReconciler) ensureCachePVCs(ctx context.Context, bj *buildv1alpha1.BuildJob) error {
	if len(bj.Spec.Caches) == 0 {
		return nil
	}
	logger := log.FromContext(ctx)
	pvcName := tekton.SharedCachePVCName()
	var existing corev1.PersistentVolumeClaim
	if err := r.Get(ctx, client.ObjectKey{Namespace: bj.Namespace, Name: pvcName}, &existing); err == nil {
		return nil
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: bj.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "bob",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
	if err := r.Create(ctx, pvc); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating shared cache PVC %q: %w", pvcName, err)
	}
	logger.Info("created shared cache PVC", "name", pvcName)
	return nil
}

func (r *BuildJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pipelineRun := &unstructured.Unstructured{}
	pipelineRun.SetGroupVersionKind(pipelineRunGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(&buildv1alpha1.BuildJob{}).
		Owns(pipelineRun).
		Complete(r)
}
