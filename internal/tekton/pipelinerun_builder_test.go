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

package tekton

import (
	"testing"
	"time"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newTestBuildJob() *buildv1alpha1.BuildJob {
	return &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "body-ecu",
			Namespace:  "builds",
			Generation: 3,
		},
		Spec: buildv1alpha1.BuildJobSpec{
			Toolchain: buildv1alpha1.ToolchainSpec{Image: "ghcr.io/zephyrproject-rtos/ci-base:v0.27.4"},
			Target: buildv1alpha1.TargetSpec{
				Board:        "nucleo_h755zi_q",
				Platform:     "zephyr",
				Architecture: "arm",
			},
			Stages: []buildv1alpha1.NamedStage{
				{Name: "fetch", StageSpec: buildv1alpha1.StageSpec{Command: "west init -l . && west update"}},
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "west build -b $BOB_BOARD src/app"}},
				{Name: "package", StageSpec: buildv1alpha1.StageSpec{Command: "cp build/zephyr/zephyr.bin /workspace/artifacts/"}},
			},
		},
	}
}

func TestBuildPipelineRun_DeterministicName(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	if pr.GetName() != "body-ecu-gen3" {
		t.Fatalf("expected deterministic name body-ecu-gen3, got %s", pr.GetName())
	}
}

func TestBuildPipelineRun_Namespace(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	if pr.GetNamespace() != "builds" {
		t.Fatalf("expected namespace builds, got %s", pr.GetNamespace())
	}
}

func TestBuildPipelineRun_Labels(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	labels := pr.GetLabels()
	if labels["builder.sdv.cloud.redhat.com/buildjob"] != "body-ecu" {
		t.Fatalf("expected buildjob label, got %v", labels)
	}
}

func TestBuildPipelineRun_FlexibleStages(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	names := make([]string, len(tasks))
	for i, task := range tasks {
		m := task.(map[string]interface{})
		names[i] = m["name"].(string)
	}
	expected := []string{"fetch", "build", "package"}
	for i, name := range names {
		if name != expected[i] {
			t.Fatalf("task %d: expected name %q, got %q", i, expected[i], name)
		}
	}
}

func TestBuildPipelineRun_RunAfterChaining(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)

	// First task should have no runAfter
	first := tasks[0].(map[string]interface{})
	if _, has := first["runAfter"]; has {
		t.Fatal("first task should not have runAfter")
	}

	// Second task should run after first
	second := tasks[1].(map[string]interface{})
	runAfter := second["runAfter"].([]interface{})
	if runAfter[0] != "fetch" {
		t.Fatalf("second task should runAfter fetch, got %v", runAfter)
	}
}

func TestBuildPipelineRun_PerStageImage(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Stages = append(bj.Spec.Stages, buildv1alpha1.NamedStage{
		Name:      "deploy",
		StageSpec: buildv1alpha1.StageSpec{Command: "flash", Image: "ghcr.io/jumpstarter/flash:latest"},
	})
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	if step["image"] != "ghcr.io/jumpstarter/flash:latest" {
		t.Fatalf("expected per-stage image override, got %v", step["image"])
	}
}

func TestBuildPipelineRun_DefaultImage(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "no-image", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Stages: []buildv1alpha1.NamedStage{
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	task := tasks[0].(map[string]interface{})
	taskSpec := task["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	if step["image"] != "ubuntu:24.04" {
		t.Fatalf("expected default image ubuntu:24.04, got %v", step["image"])
	}
}

func TestBuildPipelineRun_EnvVars(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	task := tasks[0].(map[string]interface{})
	taskSpec := task["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	envs := step["env"].([]interface{})

	envMap := make(map[string]string)
	for _, e := range envs {
		m := e.(map[string]interface{})
		envMap[m["name"].(string)] = m["value"].(string)
	}

	if envMap["BOB_BOARD"] != "nucleo_h755zi_q" {
		t.Fatalf("expected BOB_BOARD=nucleo_h755zi_q, got %q", envMap["BOB_BOARD"])
	}
	if envMap["BOB_PLATFORM"] != "zephyr" {
		t.Fatalf("expected BOB_PLATFORM=zephyr, got %q", envMap["BOB_PLATFORM"])
	}
	if envMap["BOB_ARCH"] != "arm" {
		t.Fatalf("expected BOB_ARCH=arm, got %q", envMap["BOB_ARCH"])
	}
}

func TestBuildPipelineRun_SecurityContext(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	task := tasks[0].(map[string]interface{})
	taskSpec := task["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	sc := step["securityContext"].(map[string]interface{})
	if sc["allowPrivilegeEscalation"] != false {
		t.Fatal("expected allowPrivilegeEscalation=false")
	}
}

func TestBuildPipelineRun_Timeout(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Timeout = &metav1.Duration{Duration: 30 * time.Minute}
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	timeouts := spec["timeouts"].(map[string]interface{})
	if timeouts["pipeline"] != "30m0s" {
		t.Fatalf("expected timeout 30m0s, got %v", timeouts["pipeline"])
	}
}

func TestBuildPipelineRun_NoTimeoutWhenNil(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Timeout = nil
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	if _, has := spec["timeouts"]; has {
		t.Fatal("expected no timeouts when Timeout is nil")
	}
}

func TestBuildPipelineRun_ServiceAccountName(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Toolchain.ServiceAccountName = "build-sa"
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	trt := spec["taskRunTemplate"].(map[string]interface{})
	if trt["serviceAccountName"] != "build-sa" {
		t.Fatalf("expected serviceAccountName build-sa, got %v", trt["serviceAccountName"])
	}
}

func TestBuildPipelineRun_InlinePipelineSpec(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	if _, has := spec["pipelineSpec"]; !has {
		t.Fatal("expected inline pipelineSpec instead of pipelineRef")
	}
	if _, has := spec["pipelineRef"]; has {
		t.Fatal("should not have pipelineRef with inline spec")
	}
}

func TestBuildPipelineRun_CommandQuoting(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "quote-test", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Stages: []buildv1alpha1.NamedStage{
				{Name: "test", StageSpec: buildv1alpha1.StageSpec{Command: "echo 'hello world'"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	task := tasks[0].(map[string]interface{})
	taskSpec := task["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if script == "" {
		t.Fatal("expected non-empty script")
	}
}

func TestBuildPipelineRun_WorkspaceIsVolumeClaimTemplate(t *testing.T) {
	bj := newTestBuildJob()
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	workspaces := spec["workspaces"].([]interface{})
	if len(workspaces) == 0 {
		t.Fatal("expected at least one workspace")
	}
	ws := workspaces[0].(map[string]interface{})
	if ws["name"] != "shared-workspace" {
		t.Fatalf("expected workspace name shared-workspace, got %v", ws["name"])
	}
	if _, has := ws["volumeClaimTemplate"]; !has {
		t.Fatal("expected volumeClaimTemplate in workspace")
	}
}

func getPipelineTasks(t *testing.T, pr *unstructured.Unstructured) []interface{} {
	t.Helper()
	spec := pr.Object["spec"].(map[string]interface{})
	ps := spec["pipelineSpec"].(map[string]interface{})
	tasks := ps["tasks"].([]interface{})
	return tasks
}
