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
	"time"

	buildv1alpha1 "github.com/example/builder-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	TektonAPIVersion = "tekton.dev/v1"
	PipelineRunKind  = "PipelineRun"
	PipelineName     = "configurable-build-pipeline"
)

func BuildPipelineRun(sb *buildv1alpha1.SoftwareBuild) *unstructured.Unstructured {
	labels := map[string]interface{}{
		"build.mycompany.io/softwarebuild": sb.Name,
	}

	prName := fmt.Sprintf("%s-%d", sb.Name, time.Now().Unix())
	image := sb.Spec.Runtime.Image
	if image == "" {
		image = "ubuntu:24.04"
	}

	obj := map[string]interface{}{
		"apiVersion": TektonAPIVersion,
		"kind":       PipelineRunKind,
		"metadata": map[string]interface{}{
			"name":      prName,
			"namespace": sb.Namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"pipelineRef": map[string]interface{}{
				"name": PipelineName,
			},
			"params": []interface{}{
				map[string]interface{}{"name": "containerImage", "value": image},
				map[string]interface{}{"name": "fetchCommand", "value": sb.Spec.Stages.Fetch.Command},
				map[string]interface{}{"name": "prebuildCommand", "value": sb.Spec.Stages.Prebuild.Command},
				map[string]interface{}{"name": "buildCommand", "value": sb.Spec.Stages.Build.Command},
				map[string]interface{}{"name": "postbuildCommand", "value": sb.Spec.Stages.Postbuild.Command},
				map[string]interface{}{"name": "deployCommand", "value": sb.Spec.Stages.Deploy.Command},
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
		},
	}

	return &unstructured.Unstructured{Object: obj}
}
