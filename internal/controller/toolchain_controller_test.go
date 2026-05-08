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
	"strings"
	"testing"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTaskRunPhase_Succeeded(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Succeeded",
						"status": "True",
					},
				},
			},
		},
	}
	if phase := r.taskRunPhase(tr); phase != "Succeeded" {
		t.Fatalf("expected Succeeded, got %s", phase)
	}
}

func TestTaskRunPhase_Failed(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Succeeded",
						"status": "False",
					},
				},
			},
		},
	}
	if phase := r.taskRunPhase(tr); phase != "Failed" {
		t.Fatalf("expected Failed, got %s", phase)
	}
}

func TestTaskRunPhase_Running(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Succeeded",
						"status": "Unknown",
					},
				},
			},
		},
	}
	if phase := r.taskRunPhase(tr); phase != "Running" {
		t.Fatalf("expected Running, got %s", phase)
	}
}

func TestTaskRunPhase_NoConditions(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{},
		},
	}
	if phase := r.taskRunPhase(tr); phase != "Running" {
		t.Fatalf("expected Running when no conditions, got %s", phase)
	}
}

func TestBuildTaskRun_InlineDockerfile(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "openbsw-toolchain",
			Namespace:  "bob-builds",
			Generation: 1,
		},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/openbsw-toolchain:latest",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				Dockerfile: "FROM ubuntu:24.04\nRUN apt-get update && apt-get install -y cmake\n",
			},
		},
	}

	tr := r.buildTaskRun(tc, "tc-openbsw-toolchain-build-1")

	if tr.GetName() != "tc-openbsw-toolchain-build-1" {
		t.Fatalf("expected name tc-openbsw-toolchain-build-1, got %s", tr.GetName())
	}
	if tr.GetNamespace() != "bob-builds" {
		t.Fatalf("expected namespace bob-builds, got %s", tr.GetNamespace())
	}

	labels := tr.GetLabels()
	if labels["builder.sdv.cloud.redhat.com/toolchain"] != "openbsw-toolchain" {
		t.Fatalf("expected toolchain label, got %v", labels)
	}

	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	step := steps[0].(map[string]interface{})
	expectedImage := "quay.io/buildah/stable:v1.39.0"
	if step["image"] != expectedImage {
		t.Fatalf("expected pinned buildah image %s, got %v", expectedImage, step["image"])
	}

	script := step["script"].(string)
	if !strings.Contains(script, "base64 -d") {
		t.Fatal("script should decode base64-encoded Dockerfile")
	}
	if !strings.Contains(script, "buildah bud") {
		t.Fatal("script should use buildah bud")
	}
	if !strings.Contains(script, "buildah push") {
		t.Fatal("script should push the image")
	}
	if !strings.Contains(script, "registry.example.com/openbsw-toolchain:latest") {
		t.Fatal("script should reference the target image")
	}
}

func TestBuildTaskRun_GitContext(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "zephyr-custom",
			Namespace:  "bob-builds",
			Generation: 2,
		},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/zephyr-custom:v1",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				ContextGit: &buildv1alpha1.GitSource{
					URL:      "https://github.com/myorg/toolchains",
					Revision: "main",
				},
				DockerfilePath: "zephyr/Dockerfile",
			},
		},
	}

	tr := r.buildTaskRun(tc, "tc-zephyr-custom-build-2")
	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if !strings.Contains(script, "git clone") {
		t.Fatal("git context should clone the repo")
	}
	if !strings.Contains(script, "https://github.com/myorg/toolchains") {
		t.Fatal("script should reference the git URL")
	}
	if !strings.Contains(script, "zephyr/Dockerfile") {
		t.Fatal("script should use the specified Dockerfile path")
	}
}

func TestExtractTaskRunResult_Found(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"name":  "IMAGE_DIGEST",
						"value": "sha256:abc123def456",
					},
					map[string]interface{}{
						"name":  "IMAGE_URL",
						"value": "registry.example.com/image:latest",
					},
				},
			},
		},
	}
	digest := r.extractTaskRunResult(tr, "IMAGE_DIGEST")
	if digest != "sha256:abc123def456" {
		t.Fatalf("expected sha256:abc123def456, got %q", digest)
	}
}

func TestExtractTaskRunResult_NotFound(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"name":  "OTHER_RESULT",
						"value": "something",
					},
				},
			},
		},
	}
	digest := r.extractTaskRunResult(tr, "IMAGE_DIGEST")
	if digest != "" {
		t.Fatalf("expected empty string for missing result, got %q", digest)
	}
}

func TestExtractTaskRunResult_EmptyResults(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{},
		},
	}
	digest := r.extractTaskRunResult(tr, "IMAGE_DIGEST")
	if digest != "" {
		t.Fatalf("expected empty string with no results, got %q", digest)
	}
}

func TestExtractTaskRunResult_EmptyValue(t *testing.T) {
	r := &ToolchainReconciler{}
	tr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"name":  "IMAGE_DIGEST",
						"value": "",
					},
				},
			},
		},
	}
	digest := r.extractTaskRunResult(tr, "IMAGE_DIGEST")
	if digest != "" {
		t.Fatalf("expected empty string for empty value, got %q", digest)
	}
}

func TestBuildTaskRun_NoTLSVerifyFalse(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{Name: "test-tls", Namespace: "default", Generation: 1},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/test:v1",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				Dockerfile: "FROM scratch\n",
			},
		},
	}
	tr := r.buildTaskRun(tc, "tc-test-tls-1")
	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if strings.Contains(script, "--tls-verify=false") {
		t.Fatal("script should not contain --tls-verify=false (TLS verification must be enabled by default)")
	}
}

func TestBuildTaskRun_PinnedImage(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pin", Namespace: "default", Generation: 1},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/test:v1",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				Dockerfile: "FROM scratch\n",
			},
		},
	}
	tr := r.buildTaskRun(tc, "tc-test-pin-1")
	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	image := step["image"].(string)

	if strings.Contains(image, ":latest") {
		t.Fatalf("buildah image should be pinned, not :latest — got %q", image)
	}
	expectedBuildahImage := "quay.io/buildah/stable:v1.39.0"
	if image != expectedBuildahImage {
		t.Fatalf("expected buildah image constant %q, got %q", expectedBuildahImage, image)
	}
}

func TestBuildTaskRun_RootlessSecurityContext(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sec", Namespace: "default", Generation: 1},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/test:v1",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				Dockerfile: "FROM scratch\n",
			},
		},
	}
	tr := r.buildTaskRun(tc, "tc-test-sec-1")
	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})

	sc := step["securityContext"].(map[string]interface{})
	if priv, ok := sc["privileged"]; ok && priv == true {
		t.Fatal("container must not be privileged — use rootless buildah")
	}
	if runAsNonRoot, ok := sc["runAsNonRoot"]; !ok || runAsNonRoot != true {
		t.Fatal("container must set runAsNonRoot: true")
	}
	if uid, ok := sc["runAsUser"]; !ok || uid == int64(0) {
		t.Fatal("container must not run as root (uid 0)")
	}

	script := step["script"].(string)
	if !strings.Contains(script, "--isolation=rootless") {
		t.Fatal("buildah must use --isolation=rootless")
	}
	if strings.Contains(script, "--isolation=chroot") {
		t.Fatal("buildah must not use --isolation=chroot")
	}
}

func TestBuildTaskRun_DefaultDockerfilePath(t *testing.T) {
	r := &ToolchainReconciler{}
	tc := &buildv1alpha1.Toolchain{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: buildv1alpha1.ToolchainCRSpec{
			Image: "registry.example.com/test:latest",
			Build: &buildv1alpha1.ToolchainBuildSpec{
				ContextGit: &buildv1alpha1.GitSource{
					URL: "https://github.com/myorg/test",
				},
			},
		},
	}

	tr := r.buildTaskRun(tc, "tc-test-build-1")
	spec := tr.Object["spec"].(map[string]interface{})
	taskSpec := spec["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if !strings.Contains(script, "Dockerfile") {
		t.Fatal("should default to 'Dockerfile' path")
	}
}
