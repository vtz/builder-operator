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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestArchToK8s(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"arm", "arm64", false},
		{"x86", "amd64", false},
		{"riscv", "riscv64", false},
		{"native", "", false},
		{"", "", false},
		{"xtensa", "", true},
		{"mips", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := archToK8s(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("archToK8s(%q): error=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("archToK8s(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestArchToK8s_XtensaErrorMessage(t *testing.T) {
	_, err := archToK8s("xtensa")
	if err == nil {
		t.Fatal("expected error for xtensa")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message for xtensa")
	}
}

func TestValidateSourcePVC_GitSource(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit},
		},
	}
	r := newFakeReconciler(bj)
	err := r.validateSourcePVC(context.Background(), bj)
	if err != nil {
		t.Fatalf("git source should not validate PVC: %v", err)
	}
}

func TestValidateSourcePVC_PVCExists(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-source", Namespace: "builds"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "my-source"},
			},
		},
	}
	r := newFakeReconciler(bj, pvc)
	err := r.validateSourcePVC(context.Background(), bj)
	if err != nil {
		t.Fatalf("expected no error when PVC exists: %v", err)
	}
}

func TestValidateSourcePVC_PVCMissing(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "missing-pvc"},
			},
		},
	}
	r := newFakeReconciler(bj)
	err := r.validateSourcePVC(context.Background(), bj)
	if err == nil {
		t.Fatal("expected error when PVC does not exist")
	}
}

func TestValidateSourcePVC_NilPVCSpec(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypePVC},
		},
	}
	r := newFakeReconciler(bj)
	err := r.validateSourcePVC(context.Background(), bj)
	if err != nil {
		t.Fatalf("nil PVC spec should not error: %v", err)
	}
}

func TestPVNodeSelector_NoPVC(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj)
	result := r.pvNodeSelector(context.Background(), "builds", "nonexistent")
	if result != nil {
		t.Fatalf("expected nil selector when PVC does not exist, got %v", result)
	}
}

func TestPVNodeSelector_PVCWithoutVolumeName(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "builds"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj, pvc)
	result := r.pvNodeSelector(context.Background(), "builds", "my-pvc")
	if result != nil {
		t.Fatalf("expected nil selector when PVC has no volume name, got %v", result)
	}
}

func TestPVNodeSelector_PVWithNodeAffinity(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "builds"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			VolumeName:  "my-pv",
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pv"},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "topology.kubernetes.io/zone",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"us-east-1a"},
								},
							},
						},
					},
				},
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/data"},
			},
		},
	}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj, pvc, pv)
	result := r.pvNodeSelector(context.Background(), "builds", "my-pvc")
	if result == nil {
		t.Fatal("expected non-nil selector when PV has node affinity")
	}
	zone, ok := result["topology.kubernetes.io/zone"]
	if !ok || zone != "us-east-1a" {
		t.Fatalf("expected zone=us-east-1a, got %v", result)
	}
}

func TestPVNodeSelector_PVWithoutNodeAffinity(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "builds"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			VolumeName:  "my-pv",
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pv"},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/data"},
			},
		},
	}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj, pvc, pv)
	result := r.pvNodeSelector(context.Background(), "builds", "my-pvc")
	if result != nil {
		t.Fatalf("expected nil selector when PV has no node affinity, got %v", result)
	}
}

func TestCacheNodeSelector_DelegatesToPVNodeSelector(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj)
	result := r.cacheNodeSelector(context.Background(), bj)
	if result != nil {
		t.Fatalf("expected nil when no cache PVC exists, got %v", result)
	}
}

func TestPvcNodeSelector_DelegatesToPVNodeSelector(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj)
	result := r.pvcNodeSelector(context.Background(), "builds", "nonexistent")
	if result != nil {
		t.Fatalf("expected nil when PVC does not exist, got %v", result)
	}
}

func TestMergeCondition_AddsNew(t *testing.T) {
	c := buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "Test", "test msg", 1)
	result := mergeCondition(nil, c)
	if len(result) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(result))
	}
	if result[0].Type != "Ready" {
		t.Fatalf("expected Ready condition, got %s", result[0].Type)
	}
}

func TestMergeCondition_ReplacesExisting(t *testing.T) {
	existing := []metav1.Condition{
		buildv1alpha1.NewCondition("Ready", metav1.ConditionFalse, "Old", "old msg", 1),
	}
	c := buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "New", "new msg", 2)
	result := mergeCondition(existing, c)
	if len(result) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(result))
	}
	if result[0].Reason != "New" {
		t.Fatalf("expected reason New, got %s", result[0].Reason)
	}
}

func TestMergeCondition_PreservesOtherTypes(t *testing.T) {
	existing := []metav1.Condition{
		buildv1alpha1.NewCondition("Progressing", metav1.ConditionTrue, "Prog", "progressing", 1),
	}
	c := buildv1alpha1.NewCondition("Ready", metav1.ConditionTrue, "Ready", "ready", 1)
	result := mergeCondition(existing, c)
	if len(result) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(result))
	}
}

func TestComputeOCIRef_DefaultTag(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "body-ecu", Generation: 5},
		Spec: buildv1alpha1.BuildJobSpec{
			Artifacts: buildv1alpha1.ArtifactSpec{
				OCI: &buildv1alpha1.OCIArtifactConfig{
					Repository: "quay.io/myorg/firmware",
				},
			},
		},
	}
	ref := computeOCIRef(bj)
	expected := "quay.io/myorg/firmware:body-ecu-5"
	if ref != expected {
		t.Fatalf("expected %q, got %q", expected, ref)
	}
}

func TestComputeOCIRef_CustomTag(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "body-ecu", Generation: 3},
		Spec: buildv1alpha1.BuildJobSpec{
			Target: buildv1alpha1.TargetSpec{Architecture: "arm", Variant: "v7m"},
			Artifacts: buildv1alpha1.ArtifactSpec{
				OCI: &buildv1alpha1.OCIArtifactConfig{
					Repository: "quay.io/myorg/firmware",
					Tag:        "${name}-${arch}-${variant}",
				},
			},
		},
	}
	ref := computeOCIRef(bj)
	expected := "quay.io/myorg/firmware:body-ecu-arm-v7m"
	if ref != expected {
		t.Fatalf("expected %q, got %q", expected, ref)
	}
}

func TestComputeOCIRef_NilOCI(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "body-ecu"},
		Spec: buildv1alpha1.BuildJobSpec{
			Artifacts: buildv1alpha1.ArtifactSpec{},
		},
	}
	ref := computeOCIRef(bj)
	if ref != "" {
		t.Fatalf("expected empty ref for nil OCI, got %q", ref)
	}
}
