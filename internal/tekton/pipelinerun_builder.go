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
	"fmt"
	"strings"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	TektonAPIVersion = "tekton.dev/v1"
	PipelineRunKind  = "PipelineRun"
)

func BuildPipelineRun(bj *buildv1alpha1.BuildJob) *unstructured.Unstructured {
	return BuildPipelineRunN(bj, bj.Generation)
}

func BuildPipelineRunN(bj *buildv1alpha1.BuildJob, runN int64) *unstructured.Unstructured {
	labels := map[string]interface{}{
		"builder.sdv.cloud.redhat.com/buildjob": bj.Name,
	}

	prName := fmt.Sprintf("%s-run%d", bj.Name, runN)

	image := bj.Spec.Toolchain.Image
	if image == "" {
		image = "ubuntu:24.04"
	}

	envVars := buildEnvVars(bj)

	tasks := make([]interface{}, 0, len(bj.Spec.Stages)+1)
	var prevStage string

	if bj.Spec.Source.Type == buildv1alpha1.SourceTypeGit && bj.Spec.Source.Git != nil {
		rev := bj.Spec.Source.Git.Revision
		if rev == "" {
			rev = "main"
		}
		cloneCmd := fmt.Sprintf("git clone --branch %s --depth 1 %s source", shellQuote(rev), shellQuote(bj.Spec.Source.Git.URL))
		cloneTask := buildTaskSpec("clone", image, cloneCmd, envVars, "", nil)
		tasks = append(tasks, cloneTask)
		prevStage = "clone"
	}

	for _, stage := range bj.Spec.Stages {
		stageImage := image
		if stage.Image != "" {
			stageImage = stage.Image
		}
		task := buildTaskSpec(stage.Name, stageImage, stage.Command, envVars, prevStage, bj.Spec.Caches)
		tasks = append(tasks, task)
		prevStage = stage.Name
	}

	pipelineWorkspaces := []interface{}{
		map[string]interface{}{"name": "shared-workspace"},
	}
	runWorkspaces := []interface{}{
		map[string]interface{}{
			"name": "shared-workspace",
			"volumeClaimTemplate": map[string]interface{}{
				"spec": map[string]interface{}{
					"accessModes": []interface{}{"ReadWriteOnce"},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"storage": "10Gi",
						},
					},
				},
			},
		},
	}

	// Cache PVCs are mounted directly as pod volumes in buildTaskSpec,
	// not as Tekton workspaces (Tekton disallows multiple PVCs per TaskRun).

	spec := map[string]interface{}{
		"pipelineSpec": map[string]interface{}{
			"workspaces": pipelineWorkspaces,
			"tasks":      tasks,
		},
		"workspaces": runWorkspaces,
	}

	if bj.Spec.Timeout != nil {
		spec["timeouts"] = map[string]interface{}{
			"pipeline": bj.Spec.Timeout.Duration.String(),
		}
	}

	if bj.Spec.Toolchain.ServiceAccountName != "" {
		spec["taskRunTemplate"] = map[string]interface{}{
			"serviceAccountName": bj.Spec.Toolchain.ServiceAccountName,
		}
	}

	obj := map[string]interface{}{
		"apiVersion": TektonAPIVersion,
		"kind":       PipelineRunKind,
		"metadata": map[string]interface{}{
			"name":      prName,
			"namespace": bj.Namespace,
			"labels":    labels,
		},
		"spec": spec,
	}

	return &unstructured.Unstructured{Object: obj}
}

func buildTaskSpec(name, image, command string, envVars []interface{}, runAfter string, caches []buildv1alpha1.CacheMount) map[string]interface{} {
	allowPrivEsc := false
	step := map[string]interface{}{
		"name":  "run",
		"image": image,
		"env":   envVars,
		"securityContext": map[string]interface{}{
			"allowPrivilegeEscalation": allowPrivEsc,
		},
		"script": fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\ncd $(workspaces.ws.path)\n%s\n", command),
	}

	var volumes []interface{}
	var volumeMounts []interface{}
	if len(caches) > 0 {
		volumes = append(volumes, map[string]interface{}{
			"name": "bob-cache",
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": "bob-cache",
			},
		})
		for _, cache := range caches {
			volumeMounts = append(volumeMounts, map[string]interface{}{
				"name":      "bob-cache",
				"mountPath": cache.MountPath,
				"subPath":   cache.Name,
			})
		}
		step["volumeMounts"] = volumeMounts
	}

	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{
				"name":      "ws",
				"mountPath": "/workspace",
			},
		},
		"steps": []interface{}{step},
	}
	if len(volumes) > 0 {
		taskSpec["volumes"] = volumes
	}

	task := map[string]interface{}{
		"name":     name,
		"taskSpec": taskSpec,
		"workspaces": []interface{}{
			map[string]interface{}{
				"name":      "ws",
				"workspace": "shared-workspace",
			},
		},
	}
	if runAfter != "" {
		task["runAfter"] = []interface{}{runAfter}
	}
	return task
}

func SharedCachePVCName() string {
	return "bob-cache"
}

func buildEnvVars(bj *buildv1alpha1.BuildJob) []interface{} {
	vars := []interface{}{
		map[string]interface{}{"name": "BOB_NAME", "value": bj.Name},
	}
	if bj.Spec.Target.Board != "" {
		vars = append(vars, map[string]interface{}{"name": "BOB_BOARD", "value": bj.Spec.Target.Board})
	}
	if bj.Spec.Target.Platform != "" {
		vars = append(vars, map[string]interface{}{"name": "BOB_PLATFORM", "value": bj.Spec.Target.Platform})
	}
	if bj.Spec.Target.Architecture != "" {
		vars = append(vars, map[string]interface{}{"name": "BOB_ARCH", "value": bj.Spec.Target.Architecture})
	}
	if bj.Spec.Target.Variant != "" {
		vars = append(vars, map[string]interface{}{"name": "BOB_VARIANT", "value": bj.Spec.Target.Variant})
	}
	return vars
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
