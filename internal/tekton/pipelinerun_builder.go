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
		cloneScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cd $(workspaces.ws.path)
git clone %s source
cd source
git checkout %s
git rev-parse HEAD > $(results.commit-sha.path)
`, shellQuote(bj.Spec.Source.Git.URL), shellQuote(rev))
		cloneTask := buildCloneTaskSpec("clone", image, cloneScript, envVars)
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

	if bj.Spec.Artifacts.Path != "" {
		uploadScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cd $(workspaces.ws.path)
ARTIFACTS_DIR=%q
if [ ! -d "$ARTIFACTS_DIR" ] || [ -z "$(ls -A "$ARTIFACTS_DIR" 2>/dev/null)" ]; then
  echo "No artifacts to upload"
  exit 0
fi
python3 -c "
import http.client, tarfile, io, os, sys
d = os.environ['ARTIFACTS_DIR']
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w:gz') as t:
    for f in os.listdir(d):
        t.add(os.path.join(d, f), arcname=f)
data = buf.getvalue()
conn = http.client.HTTPConnection(os.environ['BOB_API_HOST'], int(os.environ['BOB_API_PORT']))
conn.request('POST',
    '/v1/namespaces/' + os.environ['BOB_NAMESPACE'] + '/buildjobs/' + os.environ['BOB_NAME'] + '/artifacts/upload',
    body=data, headers={'Content-Type': 'application/gzip'})
r = conn.getresponse()
print('Artifact upload: ' + str(r.status) + ' (' + str(len(data)) + ' bytes)')
if r.status >= 400:
    print(r.read().decode())
    sys.exit(1)
"`, bj.Spec.Artifacts.Path)

		collectEnv := append(envVars,
			map[string]interface{}{"name": "ARTIFACTS_DIR", "value": bj.Spec.Artifacts.Path},
			map[string]interface{}{"name": "BOB_API_HOST", "value": "bob-api.bob-system.svc"},
			map[string]interface{}{"name": "BOB_API_PORT", "value": "8082"},
		)
		collectTask := buildCollectTask(image, uploadScript, collectEnv, prevStage)
		tasks = append(tasks, collectTask)
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

	pipelineSpec := map[string]interface{}{
		"workspaces": pipelineWorkspaces,
		"tasks":      tasks,
	}

	if bj.Spec.Source.Type == buildv1alpha1.SourceTypeGit && bj.Spec.Source.Git != nil {
		pipelineSpec["results"] = []interface{}{
			map[string]interface{}{
				"name":  "commit-sha",
				"value": "$(tasks.clone.results.commit-sha)",
			},
		}
	}

	spec := map[string]interface{}{
		"pipelineSpec": pipelineSpec,
		"workspaces":   runWorkspaces,
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

func buildCloneTaskSpec(name, image, script string, envVars []interface{}) map[string]interface{} {
	allowPrivEsc := false
	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{"name": "ws", "mountPath": "/workspace"},
		},
		"results": []interface{}{
			map[string]interface{}{"name": "commit-sha", "description": "The resolved git commit SHA"},
		},
		"steps": []interface{}{
			map[string]interface{}{
				"name":  "run",
				"image": image,
				"env":   envVars,
				"securityContext": map[string]interface{}{
					"allowPrivilegeEscalation": allowPrivEsc,
				},
				"script": script,
			},
		},
	}
	return map[string]interface{}{
		"name":     name,
		"taskSpec": taskSpec,
		"workspaces": []interface{}{
			map[string]interface{}{"name": "ws", "workspace": "shared-workspace"},
		},
	}
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

func buildCollectTask(image, script string, envVars []interface{}, runAfter string) map[string]interface{} {
	allowPrivEsc := false
	return map[string]interface{}{
		"name": "collect-artifacts",
		"taskSpec": map[string]interface{}{
			"workspaces": []interface{}{
				map[string]interface{}{"name": "ws", "mountPath": "/workspace"},
			},
			"steps": []interface{}{
				map[string]interface{}{
					"name":  "upload",
					"image": image,
					"env":   envVars,
					"securityContext": map[string]interface{}{
						"allowPrivilegeEscalation": allowPrivEsc,
					},
					"script": script,
				},
			},
		},
		"runAfter": []interface{}{runAfter},
		"workspaces": []interface{}{
			map[string]interface{}{"name": "ws", "workspace": "shared-workspace"},
		},
	}
}

func buildEnvVars(bj *buildv1alpha1.BuildJob) []interface{} {
	vars := []interface{}{
		map[string]interface{}{"name": "BOB_NAME", "value": bj.Name},
		map[string]interface{}{"name": "BOB_NAMESPACE", "value": bj.Namespace},
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
