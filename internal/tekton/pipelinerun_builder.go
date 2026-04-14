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
	labels := map[string]interface{}{
		"builder.sdv.cloud.redhat.com/buildjob": bj.Name,
	}

	prName := fmt.Sprintf("%s-gen%d", bj.Name, bj.Generation)

	image := bj.Spec.Toolchain.Image
	if image == "" {
		image = "ubuntu:24.04"
	}

	envVars := buildEnvVars(bj)

	tasks := make([]interface{}, 0, len(bj.Spec.Stages))
	var prevStage string
	for _, stage := range bj.Spec.Stages {
		stageImage := image
		if stage.Image != "" {
			stageImage = stage.Image
		}
		task := buildTaskSpec(stage.Name, stageImage, stage.Command, envVars, prevStage)
		tasks = append(tasks, task)
		prevStage = stage.Name
	}

	spec := map[string]interface{}{
		"pipelineSpec": map[string]interface{}{
			"workspaces": []interface{}{
				map[string]interface{}{"name": "shared-workspace"},
			},
			"tasks": tasks,
		},
		"workspaces": []interface{}{
			map[string]interface{}{
				"name": "shared-workspace",
				"volumeClaimTemplate": map[string]interface{}{
					"spec": map[string]interface{}{
						"accessModes": []interface{}{"ReadWriteOnce"},
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "1Gi",
							},
						},
					},
				},
			},
		},
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

func buildTaskSpec(name, image, command string, envVars []interface{}, runAfter string) map[string]interface{} {
	allowPrivEsc := false
	task := map[string]interface{}{
		"name": name,
		"taskSpec": map[string]interface{}{
			"workspaces": []interface{}{
				map[string]interface{}{"name": "ws"},
			},
			"steps": []interface{}{
				map[string]interface{}{
					"name":  "run",
					"image": image,
					"env":   envVars,
					"securityContext": map[string]interface{}{
						"allowPrivilegeEscalation": allowPrivEsc,
					},
					"script": fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\ncd $(workspaces.ws.path)\nbash -lc %s\n", shellQuote(command)),
				},
			},
		},
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
