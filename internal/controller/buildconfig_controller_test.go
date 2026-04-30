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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeBuildConfigReconciler(bc *buildv1alpha1.BuildConfig) *BuildConfigReconciler {
	scheme := runtime.NewScheme()
	_ = buildv1alpha1.AddToScheme(scheme)

	builder := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(bc).
		WithStatusSubresource(&buildv1alpha1.BuildConfig{})
	return &BuildConfigReconciler{
		Client: builder.Build(),
		Scheme: scheme,
	}
}

func TestBuildConfigReconcile_ValidDefaults(t *testing.T) {
	bc := &buildv1alpha1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-config"},
		Spec: buildv1alpha1.BuildConfigSpec{
			Defaults: buildv1alpha1.BuildDefaultsSpec{
				Timeout:        "30m",
				ToolchainImage: "ubuntu:24.04",
			},
		},
	}
	r := newFakeBuildConfigReconciler(bc)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cluster-config"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Requeue {
		t.Fatal("should not requeue for valid config")
	}

	var updated buildv1alpha1.BuildConfig
	if err := r.Get(context.Background(), types.NamespacedName{Name: "cluster-config"}, &updated); err != nil {
		t.Fatalf("fetching updated BuildConfig: %v", err)
	}
	if len(updated.Status.Conditions) == 0 {
		t.Fatal("expected conditions to be set")
	}
	if updated.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %s", updated.Status.Conditions[0].Status)
	}
	if updated.Status.Conditions[0].Reason != "ConfigurationValid" {
		t.Fatalf("expected reason ConfigurationValid, got %s", updated.Status.Conditions[0].Reason)
	}
}

func TestBuildConfigReconcile_ComplianceMisconfigured(t *testing.T) {
	bc := &buildv1alpha1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-config"},
		Spec: buildv1alpha1.BuildConfigSpec{
			Compliance: buildv1alpha1.ComplianceSpec{
				Enabled: true,
			},
		},
	}
	r := newFakeBuildConfigReconciler(bc)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cluster-config"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated buildv1alpha1.BuildConfig
	if err := r.Get(context.Background(), types.NamespacedName{Name: "cluster-config"}, &updated); err != nil {
		t.Fatalf("fetching updated BuildConfig: %v", err)
	}
	if len(updated.Status.Conditions) == 0 {
		t.Fatal("expected conditions to be set")
	}
	if updated.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected Ready=False for misconfigured compliance, got %s", updated.Status.Conditions[0].Status)
	}
	if updated.Status.Conditions[0].Reason != "ComplianceMisconfigured" {
		t.Fatalf("expected reason ComplianceMisconfigured, got %s", updated.Status.Conditions[0].Reason)
	}
}

func TestBuildConfigReconcile_NotFound(t *testing.T) {
	bc := &buildv1alpha1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-config"},
	}
	r := newFakeBuildConfigReconciler(bc)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent"},
	})
	if err != nil {
		t.Fatalf("unexpected error for not-found: %v", err)
	}
	if result.Requeue {
		t.Fatal("should not requeue for not-found")
	}
}

func TestBuildConfigReconcile_SetsObservedGeneration(t *testing.T) {
	bc := &buildv1alpha1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-config", Generation: 3},
		Spec: buildv1alpha1.BuildConfigSpec{
			Defaults: buildv1alpha1.BuildDefaultsSpec{Timeout: "1h"},
		},
	}
	r := newFakeBuildConfigReconciler(bc)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cluster-config"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated buildv1alpha1.BuildConfig
	if err := r.Get(context.Background(), types.NamespacedName{Name: "cluster-config"}, &updated); err != nil {
		t.Fatalf("fetching updated BuildConfig: %v", err)
	}
	if updated.Status.ObservedGeneration != 3 {
		t.Fatalf("expected ObservedGeneration=3, got %d", updated.Status.ObservedGeneration)
	}
}
