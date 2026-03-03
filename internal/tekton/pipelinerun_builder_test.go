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

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPipelineRun_UsesStageCommands(t *testing.T) {
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "ns1",
		},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Runtime: buildv1alpha1.RuntimeSpec{Image: "ubuntu:24.04"},
			Stages: buildv1alpha1.PipelineStages{
				Fetch:     buildv1alpha1.StageSpec{Command: "echo fetch"},
				Prebuild:  buildv1alpha1.StageSpec{Command: "echo pre"},
				Build:     buildv1alpha1.StageSpec{Command: "echo build"},
				Postbuild: buildv1alpha1.StageSpec{Command: "echo post"},
				Deploy:    buildv1alpha1.StageSpec{Command: "echo deploy"},
			},
		},
	}

	pr := BuildPipelineRun(sb)
	if pr.GetNamespace() != "ns1" {
		t.Fatalf("expected namespace ns1, got %s", pr.GetNamespace())
	}
	if pr.Object["kind"] != "PipelineRun" {
		t.Fatalf("expected kind PipelineRun")
	}

	spec, ok := pr.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected spec object")
	}
	params, ok := spec["params"].([]interface{})
	if !ok || len(params) < 6 {
		t.Fatalf("expected at least 6 params, got %d", len(params))
	}
}

func TestBuildPipelineRun_DefaultImage(t *testing.T) {
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "no-image", Namespace: "default"},
		Spec: buildv1alpha1.SoftwareBuildSpec{
			Runtime: buildv1alpha1.RuntimeSpec{Image: ""},
		},
	}

	pr := BuildPipelineRun(sb)

	spec, ok := pr.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected spec object")
	}
	params, ok := spec["params"].([]interface{})
	if !ok {
		t.Fatalf("expected params slice")
	}

	var containerImage string
	for _, p := range params {
		m, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if m["name"] == "containerImage" {
			containerImage, _ = m["value"].(string)
		}
	}
	if containerImage != "ubuntu:24.04" {
		t.Fatalf("expected default image ubuntu:24.04, got %q", containerImage)
	}
}

func TestBuildPipelineRun_LabelsContainSoftwareBuildName(t *testing.T) {
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "default"},
	}

	pr := BuildPipelineRun(sb)

	labels := pr.GetLabels()
	if labels["build.mycompany.io/softwarebuild"] != "my-build" {
		t.Fatalf("expected label build.mycompany.io/softwarebuild=my-build, got %v", labels)
	}
}

func TestBuildPipelineRun_WorkspaceIsVolumeClaimTemplate(t *testing.T) {
	sb := &buildv1alpha1.SoftwareBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "ws-build", Namespace: "default"},
	}

	pr := BuildPipelineRun(sb)

	spec, ok := pr.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected spec object")
	}
	workspaces, ok := spec["workspaces"].([]interface{})
	if !ok || len(workspaces) == 0 {
		t.Fatalf("expected at least one workspace")
	}
	ws, ok := workspaces[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected workspace to be a map")
	}
	if ws["name"] != "shared-workspace" {
		t.Fatalf("expected workspace name shared-workspace, got %v", ws["name"])
	}
	if _, hasvct := ws["volumeClaimTemplate"]; !hasvct {
		t.Fatalf("expected volumeClaimTemplate in workspace")
	}
}
