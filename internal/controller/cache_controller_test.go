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

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeCacheReconciler(objs ...client.Object) *CacheReconciler {
	scheme := runtime.NewScheme()
	_ = buildv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	builder := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&buildv1alpha1.Cache{})
	return &CacheReconciler{
		Client: builder.Build(),
		Scheme: scheme,
	}
}

func TestCacheReconcile_CreatesPVC(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "build-cache", Namespace: "builds"},
		Spec: buildv1alpha1.CacheCRSpec{
			Type:        buildv1alpha1.CacheTypeCCache,
			StorageSize: "10Gi",
		},
	}
	r := newFakeCacheReconciler(cache)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "build-cache", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	pvcName := cachePVCName(cache)
	if err := r.Get(context.Background(), types.NamespacedName{Name: pvcName, Namespace: "builds"}, &pvc); err != nil {
		t.Fatalf("PVC not created: %v", err)
	}

	if pvc.Labels[buildv1alpha1.LabelManagedBy] != buildv1alpha1.ManagedByValue {
		t.Fatalf("expected managed-by label %q, got %q", buildv1alpha1.ManagedByValue, pvc.Labels[buildv1alpha1.LabelManagedBy])
	}

	var updated buildv1alpha1.Cache
	if err := r.Get(context.Background(), types.NamespacedName{Name: "build-cache", Namespace: "builds"}, &updated); err != nil {
		t.Fatalf("fetching updated Cache: %v", err)
	}
	if updated.Status.Phase != CachePhaseReady {
		t.Fatalf("expected phase Ready, got %s", updated.Status.Phase)
	}
	if updated.Status.PVCName != pvcName {
		t.Fatalf("expected pvcName %q, got %q", pvcName, updated.Status.PVCName)
	}
}

func TestCacheReconcile_ExistingPVC(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "build-cache", Namespace: "builds"},
		Spec: buildv1alpha1.CacheCRSpec{
			Type:        buildv1alpha1.CacheTypeCCache,
			StorageSize: "5Gi",
		},
	}
	pvcName := cachePVCName(cache)
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: pvcName, Namespace: "builds"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	r := newFakeCacheReconciler(cache, existingPVC)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "build-cache", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error when PVC already exists: %v", err)
	}

	var updated buildv1alpha1.Cache
	if err := r.Get(context.Background(), types.NamespacedName{Name: "build-cache", Namespace: "builds"}, &updated); err != nil {
		t.Fatalf("fetching updated Cache: %v", err)
	}
	if updated.Status.Phase != CachePhaseReady {
		t.Fatalf("expected phase Ready, got %s", updated.Status.Phase)
	}
}

func TestCacheReconcile_InvalidStorageSize(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-cache", Namespace: "builds"},
		Spec: buildv1alpha1.CacheCRSpec{
			Type:        buildv1alpha1.CacheTypeGeneric,
			StorageSize: "not-a-size",
		},
	}
	r := newFakeCacheReconciler(cache)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "bad-cache", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error (should be reflected in status): %v", err)
	}

	var updated buildv1alpha1.Cache
	if err := r.Get(context.Background(), types.NamespacedName{Name: "bad-cache", Namespace: "builds"}, &updated); err != nil {
		t.Fatalf("fetching updated Cache: %v", err)
	}
	if updated.Status.Phase != CachePhaseFailed {
		t.Fatalf("expected phase Failed for invalid storage size, got %s", updated.Status.Phase)
	}
}

func TestCacheReconcile_NotFound(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "exists", Namespace: "builds"},
		Spec:       buildv1alpha1.CacheCRSpec{Type: buildv1alpha1.CacheTypeCCache},
	}
	r := newFakeCacheReconciler(cache)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error for not-found: %v", err)
	}
	if result.Requeue {
		t.Fatal("should not requeue for not-found")
	}
}

func TestCacheReconcile_CustomStorageClass(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-cache", Namespace: "builds"},
		Spec: buildv1alpha1.CacheCRSpec{
			Type:             buildv1alpha1.CacheTypeCCache,
			StorageSize:      "20Gi",
			StorageClassName: "fast-ssd",
		},
	}
	r := newFakeCacheReconciler(cache)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "custom-cache", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	pvcName := cachePVCName(cache)
	if err := r.Get(context.Background(), types.NamespacedName{Name: pvcName, Namespace: "builds"}, &pvc); err != nil {
		t.Fatalf("PVC not created: %v", err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "fast-ssd" {
		t.Fatalf("expected storageClassName fast-ssd, got %v", pvc.Spec.StorageClassName)
	}
}

func TestCacheReconcile_SetsObservedGeneration(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "gen-cache", Namespace: "builds", Generation: 5},
		Spec:       buildv1alpha1.CacheCRSpec{Type: buildv1alpha1.CacheTypeCCache, StorageSize: "5Gi"},
	}
	r := newFakeCacheReconciler(cache)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "gen-cache", Namespace: "builds"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated buildv1alpha1.Cache
	if err := r.Get(context.Background(), types.NamespacedName{Name: "gen-cache", Namespace: "builds"}, &updated); err != nil {
		t.Fatalf("fetching updated Cache: %v", err)
	}
	if updated.Status.ObservedGeneration != 5 {
		t.Fatalf("expected ObservedGeneration=5, got %d", updated.Status.ObservedGeneration)
	}
}

func TestCachePVCName(t *testing.T) {
	cache := &buildv1alpha1.Cache{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cache"},
		Spec:       buildv1alpha1.CacheCRSpec{Type: buildv1alpha1.CacheTypeCCache},
	}
	name := cachePVCName(cache)
	if name != "bob-cache-my-cache-ccache" {
		t.Fatalf("expected bob-cache-my-cache-ccache, got %s", name)
	}
}
