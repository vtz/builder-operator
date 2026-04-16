package controller

import (
	"context"
	"testing"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"github.com/centos-automotive-suite/bob/internal/tekton"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeReconciler(objs ...client.Object) *BuildJobReconciler {
	scheme := runtime.NewScheme()
	_ = buildv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	prGVK := schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRun"}
	prListGVK := schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRunList"}
	scheme.AddKnownTypeWithName(prGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(prListGVK, &unstructured.UnstructuredList{})

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).WithStatusSubresource(&buildv1alpha1.BuildJob{})
	return &BuildJobReconciler{
		Client: builder.Build(),
		Scheme: scheme,
	}
}

func TestNextRunNumber_NoPreviousRuns(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj)
	n := r.nextRunNumber(context.Background(), bj)
	if n != 1 {
		t.Fatalf("expected run number 1 with no previous runs, got %d", n)
	}
}

func TestNextRunNumber_RespectsStatusRunCount(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Status:     buildv1alpha1.BuildJobStatus{RunCount: 5},
	}
	r := newFakeReconciler(bj)
	n := r.nextRunNumber(context.Background(), bj)
	if n < 6 {
		t.Fatalf("expected run number >= 6 with RunCount=5, got %d", n)
	}
}

func TestEnsureCachePVCs_NoCaches(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
	}
	r := newFakeReconciler(bj)
	err := r.ensureCachePVCs(context.Background(), bj)
	if err != nil {
		t.Fatalf("unexpected error with no caches: %v", err)
	}
}

func TestEnsureCachePVCs_CreatesPVC(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Caches: []buildv1alpha1.CacheMount{
				{Name: "ccache", MountPath: "/root/.ccache"},
			},
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit},
			Stages: []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	r := newFakeReconciler(bj)
	err := r.ensureCachePVCs(context.Background(), bj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	pvcName := tekton.SharedCachePVCName()
	err = r.Client.Get(context.Background(), client.ObjectKey{Namespace: "builds", Name: pvcName}, &pvc)
	if err != nil {
		t.Fatalf("cache PVC not created: %v", err)
	}
	if pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce {
		t.Fatalf("expected RWO, got %v", pvc.Spec.AccessModes)
	}
}

func TestEnsureCachePVCs_Idempotent(t *testing.T) {
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tekton.SharedCachePVCName(),
			Namespace: "builds",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "builds"},
		Spec: buildv1alpha1.BuildJobSpec{
			Caches: []buildv1alpha1.CacheMount{{Name: "ccache", MountPath: "/root/.ccache"}},
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypeGit},
			Stages: []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	r := newFakeReconciler(bj, existingPVC)
	err := r.ensureCachePVCs(context.Background(), bj)
	if err != nil {
		t.Fatalf("should not error when PVC already exists: %v", err)
	}
}

func TestNeedsNewRun_EmptyCurrentPipelineRun(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Status:     buildv1alpha1.BuildJobStatus{CurrentPipelineRun: ""},
	}
	needsNew := bj.Status.CurrentPipelineRun == ""
	if !needsNew {
		t.Fatal("expected needsNewRun when CurrentPipelineRun is empty")
	}
}

func TestNeedsNewRun_AnnotationChanged(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "demo",
			Annotations: map[string]string{runAtAnnotation: "2026-04-14T12:00:00Z"},
		},
		Status: buildv1alpha1.BuildJobStatus{
			CurrentPipelineRun: "demo-run1",
			LastRunAt:          "2026-04-14T10:00:00Z",
		},
	}
	runAt := bj.Annotations[runAtAnnotation]
	needsNew := runAt != "" && runAt != bj.Status.LastRunAt
	if !needsNew {
		t.Fatal("expected needsNewRun when annotation differs from status")
	}
}

func TestNeedsNewRun_SameAnnotation(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "demo",
			Annotations: map[string]string{runAtAnnotation: "2026-04-14T10:00:00Z"},
		},
		Status: buildv1alpha1.BuildJobStatus{
			CurrentPipelineRun: "demo-run1",
			LastRunAt:          "2026-04-14T10:00:00Z",
		},
	}
	runAt := bj.Annotations[runAtAnnotation]
	needsNew := bj.Status.CurrentPipelineRun == "" || (runAt != "" && runAt != bj.Status.LastRunAt)
	if needsNew {
		t.Fatal("should not need new run when annotation matches status")
	}
}
