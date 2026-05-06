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
	"strings"
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

	if pr.GetName() != "body-ecu-run3" {
		t.Fatalf("expected deterministic name body-ecu-run3, got %s", pr.GetName())
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

func TestBuildPipelineRunN_DifferentRunNumbers(t *testing.T) {
	bj := newTestBuildJob()
	pr1 := BuildPipelineRunN(bj, 1)
	pr5 := BuildPipelineRunN(bj, 5)

	if pr1.GetName() != "body-ecu-run1" {
		t.Fatalf("expected body-ecu-run1, got %s", pr1.GetName())
	}
	if pr5.GetName() != "body-ecu-run5" {
		t.Fatalf("expected body-ecu-run5, got %s", pr5.GetName())
	}
}

func TestBuildPipelineRun_GitCloneTask(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Source = buildv1alpha1.SourceSpec{
		Type: buildv1alpha1.SourceTypeGit,
		Git:  &buildv1alpha1.GitSource{URL: "https://github.com/test/repo", Revision: "feature/test"},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	first := tasks[0].(map[string]interface{})
	if first["name"] != "clone" {
		t.Fatalf("expected first task to be clone, got %s", first["name"])
	}

	taskSpec := first["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)
	if script == "" {
		t.Fatal("clone script should not be empty")
	}

	results, ok := taskSpec["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("clone task should declare a commit-sha result")
	}
	result := results[0].(map[string]interface{})
	if result["name"] != "commit-sha" {
		t.Fatalf("expected result name commit-sha, got %v", result["name"])
	}

	img := step["image"].(string)
	if img != GitCloneImage {
		t.Fatalf("clone step should use GitCloneImage (%s), got %s", GitCloneImage, img)
	}
}

func TestBuildPipelineRun_PipelineLevelCommitSHAResult(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Source = buildv1alpha1.SourceSpec{
		Type: buildv1alpha1.SourceTypeGit,
		Git:  &buildv1alpha1.GitSource{URL: "https://github.com/test/repo", Revision: "main"},
	}
	pr := BuildPipelineRun(bj)

	spec := pr.Object["spec"].(map[string]interface{})
	ps := spec["pipelineSpec"].(map[string]interface{})
	results, ok := ps["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("pipelineSpec should declare a commit-sha result")
	}
	r := results[0].(map[string]interface{})
	if r["name"] != "commit-sha" {
		t.Fatalf("expected pipeline result commit-sha, got %v", r["name"])
	}
	if r["value"] != "$(tasks.clone.results.commit-sha)" {
		t.Fatalf("expected result wired to clone task, got %v", r["value"])
	}
}

func TestBuildPipelineRun_NoPipelineResultForPVC(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-build-noresult", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "my-source"},
			},
			Stages: []buildv1alpha1.NamedStage{{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}}},
		},
	}
	pr := BuildPipelineRun(bj)
	spec := pr.Object["spec"].(map[string]interface{})
	ps := spec["pipelineSpec"].(map[string]interface{})
	if _, has := ps["results"]; has {
		t.Fatal("PVC source should not have pipeline-level results")
	}
}

func TestBuildPipelineRun_NoCacheVolumes(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Caches = nil
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	for _, task := range tasks {
		m := task.(map[string]interface{})
		taskSpec := m["taskSpec"].(map[string]interface{})
		if _, has := taskSpec["volumes"]; has {
			t.Fatalf("task %s should not have volumes when no caches", m["name"])
		}
	}
}

func TestBuildPipelineRun_WithCacheVolumes(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Caches = []buildv1alpha1.CacheMount{
		{Name: "ccache", MountPath: "/root/.ccache"},
		{Name: "west-modules", MountPath: "/root/.west-modules"},
	}
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	for _, task := range tasks {
		m := task.(map[string]interface{})
		if m["name"] == "clone" {
			continue
		}
		taskSpec := m["taskSpec"].(map[string]interface{})
		volumes, ok := taskSpec["volumes"].([]interface{})
		if !ok || len(volumes) == 0 {
			t.Fatalf("task %s should have volumes for caches", m["name"])
		}
		vol := volumes[0].(map[string]interface{})
		if vol["name"] != "bob-cache" {
			t.Fatalf("expected volume name bob-cache, got %v", vol["name"])
		}

		steps := taskSpec["steps"].([]interface{})
		step := steps[0].(map[string]interface{})
		mounts, ok := step["volumeMounts"].([]interface{})
		if !ok || len(mounts) != 2 {
			t.Fatalf("expected 2 volume mounts for caches, got %d", len(mounts))
		}
		firstMount := mounts[0].(map[string]interface{})
		if firstMount["mountPath"] != "/root/.ccache" {
			t.Fatalf("expected ccache mount at /root/.ccache, got %v", firstMount["mountPath"])
		}
		if firstMount["subPath"] != "ccache" {
			t.Fatalf("expected subPath ccache, got %v", firstMount["subPath"])
		}
	}
}

func TestBuildPipelineRun_PVCSourceNoPVCSpec_SkipsClone(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-build-nopvc", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{Type: buildv1alpha1.SourceTypePVC},
			Stages: []buildv1alpha1.NamedStage{
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task (no copy-source when PVC spec is nil), got %d", len(tasks))
	}
	first := tasks[0].(map[string]interface{})
	if first["name"] != "build" {
		t.Fatalf("expected first task to be build, got %s", first["name"])
	}
}

func TestBuildPipelineRun_PVCSourceCopyTask(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-build", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "source-code", Path: "/src"},
			},
			Stages: []buildv1alpha1.NamedStage{
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks (copy-source + build), got %d", len(tasks))
	}

	copyTask := tasks[0].(map[string]interface{})
	if copyTask["name"] != "copy-source" {
		t.Fatalf("expected first task to be copy-source, got %s", copyTask["name"])
	}

	taskSpec := copyTask["taskSpec"].(map[string]interface{})

	volumes := taskSpec["volumes"].([]interface{})
	if len(volumes) != 1 {
		t.Fatalf("expected 1 volume for PVC source, got %d", len(volumes))
	}
	vol := volumes[0].(map[string]interface{})
	if vol["name"] != "pvc-source" {
		t.Fatalf("expected volume name pvc-source, got %v", vol["name"])
	}
	pvcSpec := vol["persistentVolumeClaim"].(map[string]interface{})
	if pvcSpec["claimName"] != "source-code" {
		t.Fatalf("expected claimName source-code, got %v", pvcSpec["claimName"])
	}
	if pvcSpec["readOnly"] != true {
		t.Fatal("expected PVC source to be mounted read-only")
	}

	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	mounts := step["volumeMounts"].([]interface{})
	if len(mounts) != 1 {
		t.Fatalf("expected 1 volume mount, got %d", len(mounts))
	}
	mount := mounts[0].(map[string]interface{})
	if mount["mountPath"] != "/mnt/pvc-source" {
		t.Fatalf("expected mountPath /mnt/pvc-source, got %v", mount["mountPath"])
	}

	script := step["script"].(string)
	if !strings.Contains(script, "/mnt/pvc-source/src/") {
		t.Fatalf("expected script to reference PVC path /src, got:\n%s", script)
	}
}

func TestBuildPipelineRun_PVCSourceRunAfterChaining(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-chain", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "my-source"},
			},
			Stages: []buildv1alpha1.NamedStage{
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}},
				{Name: "test", StageSpec: buildv1alpha1.StageSpec{Command: "make test"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	copyTask := tasks[0].(map[string]interface{})
	if _, has := copyTask["runAfter"]; has {
		t.Fatal("copy-source should not have runAfter")
	}

	buildTask := tasks[1].(map[string]interface{})
	runAfter := buildTask["runAfter"].([]interface{})
	if runAfter[0] != "copy-source" {
		t.Fatalf("build should runAfter copy-source, got %v", runAfter)
	}

	testTask := tasks[2].(map[string]interface{})
	testRunAfter := testTask["runAfter"].([]interface{})
	if testRunAfter[0] != "build" {
		t.Fatalf("test should runAfter build, got %v", testRunAfter)
	}
}

func TestBuildPipelineRun_PVCSourceDefaultPath(t *testing.T) {
	bj := &buildv1alpha1.BuildJob{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-default-path", Namespace: "default"},
		Spec: buildv1alpha1.BuildJobSpec{
			Source: buildv1alpha1.SourceSpec{
				Type: buildv1alpha1.SourceTypePVC,
				PVC:  &buildv1alpha1.PVCSource{ClaimName: "my-source"},
			},
			Stages: []buildv1alpha1.NamedStage{
				{Name: "build", StageSpec: buildv1alpha1.StageSpec{Command: "make"}},
			},
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)
	copyTask := tasks[0].(map[string]interface{})
	taskSpec := copyTask["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if !strings.Contains(script, "/mnt/pvc-source/") {
		t.Fatalf("expected script to copy from PVC root, got:\n%s", script)
	}
}

func TestBuildPipelineRun_ArtifactUpload_DefaultAPIHost(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{Path: "/workspace/artifacts"}
	pr := BuildPipelineRun(bj)

	tasks := getPipelineTasks(t, pr)
	last := tasks[len(tasks)-1].(map[string]interface{})
	if last["name"] != "collect-artifacts" {
		t.Fatalf("expected collect-artifacts task, got %s", last["name"])
	}

	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	envs := step["env"].([]interface{})

	envMap := make(map[string]string)
	for _, e := range envs {
		m := e.(map[string]interface{})
		envMap[m["name"].(string)] = m["value"].(string)
	}

	expectedHost := "bob-api.bob-system.svc"
	if envMap["BOB_API_HOST"] != expectedHost {
		t.Fatalf("expected BOB_API_HOST=%q (operator namespace), got %q", expectedHost, envMap["BOB_API_HOST"])
	}
	if envMap["BOB_API_PORT"] != "8082" {
		t.Fatalf("expected BOB_API_PORT=8082 default, got %q", envMap["BOB_API_PORT"])
	}
}

func TestBuildPipelineRun_ArtifactUpload_CustomAPIHost(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{Path: "/workspace/artifacts"}

	cfg := PipelineConfig{APIHost: "custom-api.custom-ns.svc", APIPort: "9090"}
	pr := BuildPipelineRunWithConfig(bj, 1, cfg)

	tasks := getPipelineTasks(t, pr)
	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	envs := step["env"].([]interface{})

	envMap := make(map[string]string)
	for _, e := range envs {
		m := e.(map[string]interface{})
		envMap[m["name"].(string)] = m["value"].(string)
	}

	if envMap["BOB_API_HOST"] != "custom-api.custom-ns.svc" {
		t.Fatalf("expected custom host, got %q", envMap["BOB_API_HOST"])
	}
	if envMap["BOB_API_PORT"] != "9090" {
		t.Fatalf("expected custom port, got %q", envMap["BOB_API_PORT"])
	}
}

func TestBuildPipelineRun_CustomConfigOverridesDefault(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{Path: "/workspace/artifacts"}

	cfg := PipelineConfig{APIHost: "my-api.custom-ns.svc"}
	pr := BuildPipelineRunWithConfig(bj, 1, cfg)

	tasks := getPipelineTasks(t, pr)
	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	envs := step["env"].([]interface{})

	for _, e := range envs {
		m := e.(map[string]interface{})
		if m["name"] == "BOB_API_HOST" && m["value"] != "my-api.custom-ns.svc" {
			t.Fatalf("custom config should override default, got %v", m["value"])
		}
	}
}

func TestSharedCachePVCName(t *testing.T) {
	name := SharedCachePVCName()
	if name != "bob-cache" {
		t.Fatalf("expected bob-cache, got %s", name)
	}
}

func TestBuildPipelineRun_OCIArtifactTask(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationOCI,
		Path:        "/workspace/artifacts",
		OCI: &buildv1alpha1.OCIArtifactConfig{
			Repository: "quay.io/myorg/firmware",
			PushSecret: &buildv1alpha1.SecretReference{Name: "quay-push-creds"},
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	if last["name"] != "oci-push" {
		t.Fatalf("expected last task to be oci-push, got %s", last["name"])
	}

	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})

	if step["image"] != OrasImage {
		t.Fatalf("expected oras image %s, got %v", OrasImage, step["image"])
	}

	script := step["script"].(string)
	if !strings.Contains(script, "oras push") {
		t.Fatal("expected oras push command in script")
	}
	if !strings.Contains(script, "quay.io/myorg/firmware:body-ecu-3") {
		t.Fatalf("expected default tag body-ecu-3, script:\n%s", script)
	}
	if !strings.Contains(script, "application/vnd.auto.firmware.layer.v1") {
		t.Fatal("expected default firmware media type in script")
	}

	envs := step["env"].([]interface{})
	envMap := make(map[string]string)
	for _, e := range envs {
		m := e.(map[string]interface{})
		envMap[m["name"].(string)] = m["value"].(string)
	}
	if envMap["REGISTRY_AUTH_FILE"] != "/etc/oci-push-secret/.dockerconfigjson" {
		t.Fatalf("expected REGISTRY_AUTH_FILE env, got %q", envMap["REGISTRY_AUTH_FILE"])
	}

	// Verify volume mount for secret
	mounts := step["volumeMounts"].([]interface{})
	if len(mounts) != 1 {
		t.Fatalf("expected 1 volume mount for push secret, got %d", len(mounts))
	}
	mount := mounts[0].(map[string]interface{})
	if mount["mountPath"] != "/etc/oci-push-secret" {
		t.Fatalf("expected secret mount at /etc/oci-push-secret, got %v", mount["mountPath"])
	}

	// Verify volume for secret
	volumes := taskSpec["volumes"].([]interface{})
	if len(volumes) != 1 {
		t.Fatalf("expected 1 volume for push secret, got %d", len(volumes))
	}
	vol := volumes[0].(map[string]interface{})
	secret := vol["secret"].(map[string]interface{})
	if secret["secretName"] != "quay-push-creds" {
		t.Fatalf("expected secret name quay-push-creds, got %v", secret["secretName"])
	}

	// Verify task results
	results := taskSpec["results"].([]interface{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results (oci-ref, oci-digest), got %d", len(results))
	}
	result := results[0].(map[string]interface{})
	if result["name"] != "oci-ref" {
		t.Fatalf("expected result name oci-ref, got %v", result["name"])
	}
	digestResult := results[1].(map[string]interface{})
	if digestResult["name"] != "oci-digest" {
		t.Fatalf("expected result name oci-digest, got %v", digestResult["name"])
	}
}

func TestBuildPipelineRun_OCIArtifactTask_CustomTag(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationOCI,
		Path:        "/workspace/artifacts",
		OCI: &buildv1alpha1.OCIArtifactConfig{
			Repository: "quay.io/myorg/firmware",
			Tag:        "${name}-${arch}-latest",
			MediaType:  "application/vnd.custom.binary",
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if !strings.Contains(script, "quay.io/myorg/firmware:body-ecu-arm-latest") {
		t.Fatalf("expected custom tag substitution, script:\n%s", script)
	}
	if !strings.Contains(script, "application/vnd.custom.binary") {
		t.Fatal("expected custom media type in script")
	}
}

func TestBuildPipelineRun_OCIArtifactTask_NoSecret(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationOCI,
		Path:        "/workspace/artifacts",
		OCI: &buildv1alpha1.OCIArtifactConfig{
			Repository: "quay.io/myorg/firmware",
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})

	if _, has := step["volumeMounts"]; has {
		t.Fatal("expected no volumeMounts when no push secret")
	}

	if _, has := taskSpec["volumes"]; has {
		t.Fatal("expected no volumes when no push secret")
	}
}

func TestBuildPipelineRun_OCIArtifactTask_Annotations(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Target.Variant = "v7m"
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationOCI,
		Path:        "/workspace/artifacts",
		OCI: &buildv1alpha1.OCIArtifactConfig{
			Repository: "quay.io/myorg/firmware",
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	taskSpec := last["taskSpec"].(map[string]interface{})
	steps := taskSpec["steps"].([]interface{})
	step := steps[0].(map[string]interface{})
	script := step["script"].(string)

	if !strings.Contains(script, "vnd.auto.target.board") {
		t.Fatal("expected board annotation in oras push")
	}
	if !strings.Contains(script, "vnd.auto.target.variant") {
		t.Fatal("expected variant annotation when variant is set")
	}
}

func TestBuildPipelineRun_OCIArtifactTask_RunAfterLastStage(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationOCI,
		Path:        "/workspace/artifacts",
		OCI: &buildv1alpha1.OCIArtifactConfig{
			Repository: "quay.io/myorg/firmware",
		},
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	runAfter := last["runAfter"].([]interface{})
	if runAfter[0] != "package" {
		t.Fatalf("oci-push should runAfter last stage 'package', got %v", runAfter[0])
	}
}

func TestBuildPipelineRun_PVCArtifactStillWorks(t *testing.T) {
	bj := newTestBuildJob()
	bj.Spec.Artifacts = buildv1alpha1.ArtifactSpec{
		Destination: buildv1alpha1.ArtifactDestinationPVC,
		Path:        "/workspace/artifacts",
	}
	pr := BuildPipelineRun(bj)
	tasks := getPipelineTasks(t, pr)

	last := tasks[len(tasks)-1].(map[string]interface{})
	if last["name"] != "collect-artifacts" {
		t.Fatalf("expected PVC destination to use collect-artifacts task, got %s", last["name"])
	}
}

func getPipelineTasks(t *testing.T, pr *unstructured.Unstructured) []interface{} {
	t.Helper()
	spec := pr.Object["spec"].(map[string]interface{})
	ps := spec["pipelineSpec"].(map[string]interface{})
	tasks := ps["tasks"].([]interface{})
	return tasks
}
