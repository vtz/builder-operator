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

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	CachePhaseReady   = "Ready"
	CachePhasePending = "Pending"
	CachePhaseFailed  = "Failed"
)

type CacheReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func cachePVCName(cache *buildv1alpha1.Cache) string {
	return fmt.Sprintf("bob-cache-%s-%s", cache.Name, cache.Spec.Type)
}

func (r *CacheReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cache buildv1alpha1.Cache
	if err := r.Get(ctx, req.NamespacedName, &cache); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cache.Status.ObservedGeneration = cache.Generation

	pvcName := cachePVCName(&cache)

	var existingPVC corev1.PersistentVolumeClaim
	err := r.Get(ctx, client.ObjectKey{Namespace: cache.Namespace, Name: pvcName}, &existingPVC)

	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("checking cache PVC: %w", err)
	}

	if apierrors.IsNotFound(err) {
		storageSize := cache.Spec.StorageSize
		if storageSize == "" {
			storageSize = "5Gi"
		}
		qty, parseErr := resource.ParseQuantity(storageSize)
		if parseErr != nil {
			cache.Status.Phase = CachePhaseFailed
			cache.Status.Conditions = mergeCondition(cache.Status.Conditions,
				buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "InvalidStorageSize",
					fmt.Sprintf("cannot parse storageSize %q: %v", storageSize, parseErr), cache.Generation))
			if statusErr := r.Status().Update(ctx, &cache); statusErr != nil {
				return ctrl.Result{}, fmt.Errorf("updating Cache status: %w", statusErr)
			}
			return ctrl.Result{}, nil
		}

		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: cache.Namespace,
				Labels: map[string]string{
					buildv1alpha1.LabelManagedBy: buildv1alpha1.ManagedByValue,
					"builder.sdv.cloud.redhat.com/cache": cache.Name,
					"builder.sdv.cloud.redhat.com/cache-type": string(cache.Spec.Type),
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: qty,
					},
				},
			},
		}

		if cache.Spec.StorageClassName != "" {
			pvc.Spec.StorageClassName = &cache.Spec.StorageClassName
		}

		if err := ctrl.SetControllerReference(&cache, pvc, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("setting controller reference on PVC: %w", err)
		}

		if err := r.Create(ctx, pvc); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Info("cache PVC already exists", "pvc", pvcName)
			} else {
				cache.Status.Phase = CachePhaseFailed
				cache.Status.Conditions = mergeCondition(cache.Status.Conditions,
					buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "PVCCreateFailed",
						fmt.Sprintf("failed to create PVC: %v", err), cache.Generation))
				if statusErr := r.Status().Update(ctx, &cache); statusErr != nil {
					return ctrl.Result{}, fmt.Errorf("updating Cache status: %w", statusErr)
				}
				return ctrl.Result{}, fmt.Errorf("creating cache PVC: %w", err)
			}
		} else {
			logger.Info("created cache PVC", "pvc", pvcName, "size", storageSize)
		}
	}

	cache.Status.PVCName = pvcName
	cache.Status.Phase = CachePhaseReady
	cache.Status.Conditions = mergeCondition(cache.Status.Conditions,
		buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "PVCProvisioned",
			fmt.Sprintf("Cache PVC %s is available", pvcName), cache.Generation))

	if err := r.Status().Update(ctx, &cache); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating Cache status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *CacheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildv1alpha1.Cache{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
