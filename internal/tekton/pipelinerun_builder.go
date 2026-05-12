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
	GitCloneImage    = "alpine/git:latest"
	RepoInitImage    = "python:3.12-slim"
	OrasImage        = "ghcr.io/oras-project/oras:v1.2.0"

	workspaceName     = "shared-workspace"
	taskWorkspaceName = "ws"
	taskCopySource    = "copy-source"
	taskClone         = "clone"
	taskRepoSync      = "repo-sync"
	mirrorVolumeName  = "repo-mirror"
	mirrorMountPath   = "/mnt/mirror"
	cacheVolumeName   = "bob-cache"

	DefaultFirmwareMediaType = "application/vnd.auto.firmware.layer.v1"
)

type PipelineConfig struct {
	APIHost string
	APIPort string
}

var DefaultPipelineConfig = PipelineConfig{}

func BuildPipelineRun(bj *buildv1alpha1.BuildJob) *unstructured.Unstructured {
	return BuildPipelineRunN(bj, bj.Generation)
}

func BuildPipelineRunN(bj *buildv1alpha1.BuildJob, runN int64) *unstructured.Unstructured {
	return BuildPipelineRunWithConfig(bj, runN, DefaultPipelineConfig)
}

func BuildPipelineRunWithConfig(bj *buildv1alpha1.BuildJob, runN int64, cfg PipelineConfig) *unstructured.Unstructured {
	labels := map[string]interface{}{
		buildv1alpha1.LabelBuildJob: bj.Name,
	}

	prName := fmt.Sprintf("%s-run%d", bj.Name, runN)

	image := bj.Spec.Toolchain.Image
	if image == "" {
		image = "ubuntu:24.04"
	}

	envVars := buildEnvVars(bj)

	tasks := make([]interface{}, 0, len(bj.Spec.Stages)+1)
	var prevStage string

	switch {
	case bj.Spec.Source.Type == buildv1alpha1.SourceTypeGit && bj.Spec.Source.Git != nil:
		rev := bj.Spec.Source.Git.Revision
		if rev == "" {
			rev = "main"
		}
		cloneScript := fmt.Sprintf(`#!/bin/sh
set -eu
cd $(workspaces.ws.path)
git clone %s source
cd source
git checkout %s
git rev-parse HEAD > $(results.commit-sha.path)
`, ShellQuote(bj.Spec.Source.Git.URL), ShellQuote(rev))
		cloneTask := buildCloneTaskSpec(taskClone, GitCloneImage, cloneScript, envVars)
		tasks = append(tasks, cloneTask)
		prevStage = taskClone

	case bj.Spec.Source.Type == buildv1alpha1.SourceTypePVC && bj.Spec.Source.PVC != nil:
		srcPath := bj.Spec.Source.PVC.Path
		if srcPath == "" {
			srcPath = "/"
		}
		if !strings.HasPrefix(srcPath, "/") {
			srcPath = "/" + srcPath
		}
		if !strings.HasSuffix(srcPath, "/") {
			srcPath += "/"
		}
		copyScript := fmt.Sprintf(`#!/bin/sh
set -eu
mkdir -p $(workspaces.ws.path)/source
cp -a /mnt/pvc-source%s. $(workspaces.ws.path)/source/
echo "Copied PVC source to workspace"
`, srcPath)
		copyTask := buildPVCCopyTaskSpec(taskCopySource, image, copyScript, envVars, bj.Spec.Source.PVC.ClaimName)
		tasks = append(tasks, copyTask)
		prevStage = taskCopySource

	case bj.Spec.Source.Type == buildv1alpha1.SourceTypeRepo && bj.Spec.Source.Repo != nil:
		repoTask := buildRepoTaskSpec(bj.Spec.Source.Repo, envVars)
		tasks = append(tasks, repoTask)
		prevStage = taskRepoSync
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
		switch bj.Spec.Artifacts.Destination {
		case buildv1alpha1.ArtifactDestinationOCI:
			if bj.Spec.Artifacts.OCI != nil {
				ociTask := buildOCIArtifactTask(bj, envVars, prevStage)
				tasks = append(tasks, ociTask)
			}
		default:
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

			apiHost := cfg.APIHost
			if apiHost == "" {
				apiHost = "bob-api.bob-system.svc"
			}
			apiPort := cfg.APIPort
			if apiPort == "" {
				apiPort = "8082"
			}
			collectEnv := make([]interface{}, len(envVars), len(envVars)+3)
			copy(collectEnv, envVars)
			collectEnv = append(collectEnv,
				map[string]interface{}{"name": "ARTIFACTS_DIR", "value": bj.Spec.Artifacts.Path},
				map[string]interface{}{"name": "BOB_API_HOST", "value": apiHost},
				map[string]interface{}{"name": "BOB_API_PORT", "value": apiPort},
			)
			collectTask := buildCollectTask(image, uploadScript, collectEnv, prevStage)
			tasks = append(tasks, collectTask)
		}
	}

	pipelineWorkspaces := []interface{}{
		map[string]interface{}{"name": workspaceName},
	}
	runWorkspaces := []interface{}{
		map[string]interface{}{
			"name": workspaceName,
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

	var pipelineResults []interface{}
	if bj.Spec.Source.Type == buildv1alpha1.SourceTypeGit && bj.Spec.Source.Git != nil {
		pipelineResults = append(pipelineResults, map[string]interface{}{
			"name":  "commit-sha",
			"value": "$(tasks.clone.results.commit-sha)",
		})
	}
	if bj.Spec.Artifacts.Destination == buildv1alpha1.ArtifactDestinationOCI && bj.Spec.Artifacts.OCI != nil {
		pipelineResults = append(pipelineResults,
			map[string]interface{}{
				"name":  "oci-ref",
				"value": "$(tasks.oci-push.results.oci-ref)",
			},
			map[string]interface{}{
				"name":  "oci-digest",
				"value": "$(tasks.oci-push.results.oci-digest)",
			},
		)
	}
	if len(pipelineResults) > 0 {
		pipelineSpec["results"] = pipelineResults
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
			map[string]interface{}{"name": taskWorkspaceName, "mountPath": "/workspace"},
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
			map[string]interface{}{"name": taskWorkspaceName, "workspace": workspaceName},
		},
	}
}

func buildPVCCopyTaskSpec(name, image, script string, envVars []interface{}, claimName string) map[string]interface{} {
	allowPrivEsc := false
	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "mountPath": "/workspace"},
		},
		"volumes": []interface{}{
			map[string]interface{}{
				"name": "pvc-source",
				"persistentVolumeClaim": map[string]interface{}{
					"claimName": claimName,
					"readOnly":  true,
				},
			},
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
				"volumeMounts": []interface{}{
					map[string]interface{}{
						"name":      "pvc-source",
						"mountPath": "/mnt/pvc-source",
						"readOnly":  true,
					},
				},
			},
		},
	}
	return map[string]interface{}{
		"name":     name,
		"taskSpec": taskSpec,
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "workspace": workspaceName},
		},
	}
}

func buildRepoTaskSpec(src *buildv1alpha1.RepoSource, envVars []interface{}) map[string]interface{} {
	syncJobs := src.SyncJobs
	if syncJobs <= 0 {
		syncJobs = 4
	}

	initArgs := fmt.Sprintf("-u %s", ShellQuote(src.ManifestURL))
	if src.Branch != "" {
		initArgs += fmt.Sprintf(" -b %s", ShellQuote(src.Branch))
	}
	if src.ManifestName != "" {
		initArgs += fmt.Sprintf(" -m %s", ShellQuote(src.ManifestName))
	}
	if src.MirrorRef != "" {
		initArgs += fmt.Sprintf(" --reference=%s", ShellQuote(mirrorMountPath))
	}

	script := fmt.Sprintf(`#!/bin/sh
set -eu
# Install repo if not present (repo is a Python script, no compiled binary needed)
if ! command -v repo > /dev/null 2>&1; then
  pip install --quiet repo
fi
cd $(workspaces.ws.path)
mkdir -p source
cd source
repo init %s
repo sync -j%d
`, initArgs, syncJobs)

	allowPrivEsc := false

	var volumes []interface{}
	var volumeMounts []interface{}
	if src.MirrorRef != "" {
		volumes = append(volumes, map[string]interface{}{
			"name": mirrorVolumeName,
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": src.MirrorRef,
				"readOnly":  true,
			},
		})
		volumeMounts = append(volumeMounts, map[string]interface{}{
			"name":      mirrorVolumeName,
			"mountPath": mirrorMountPath,
			"readOnly":  true,
		})
	}

	step := map[string]interface{}{
		"name":  "run",
		"image": RepoInitImage,
		"env":   envVars,
		"securityContext": map[string]interface{}{
			"allowPrivilegeEscalation": allowPrivEsc,
		},
		"script": script,
	}
	if len(volumeMounts) > 0 {
		step["volumeMounts"] = volumeMounts
	}

	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "mountPath": "/workspace"},
		},
		"steps": []interface{}{step},
	}
	if len(volumes) > 0 {
		taskSpec["volumes"] = volumes
	}

	return map[string]interface{}{
		"name":     taskRepoSync,
		"taskSpec": taskSpec,
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "workspace": workspaceName},
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
			"name": cacheVolumeName,
			"persistentVolumeClaim": map[string]interface{}{
				"claimName": cacheVolumeName,
			},
		})
		for _, cache := range caches {
			volumeMounts = append(volumeMounts, map[string]interface{}{
				"name":      cacheVolumeName,
				"mountPath": cache.MountPath,
				"subPath":   cache.Name,
			})
		}
		step["volumeMounts"] = volumeMounts
	}

	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{
				"name":      taskWorkspaceName,
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
				"name":      taskWorkspaceName,
				"workspace": workspaceName,
			},
		},
	}
	if runAfter != "" {
		task["runAfter"] = []interface{}{runAfter}
	}
	return task
}

func SharedCachePVCName() string {
	return cacheVolumeName
}

func buildCollectTask(image, script string, envVars []interface{}, runAfter string) map[string]interface{} {
	allowPrivEsc := false
	return map[string]interface{}{
		"name": "collect-artifacts",
		"taskSpec": map[string]interface{}{
			"workspaces": []interface{}{
				map[string]interface{}{"name": taskWorkspaceName, "mountPath": "/workspace"},
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
			map[string]interface{}{"name": taskWorkspaceName, "workspace": workspaceName},
		},
	}
}

func buildOCIArtifactTask(bj *buildv1alpha1.BuildJob, envVars []interface{}, runAfter string) map[string]interface{} {
	oci := bj.Spec.Artifacts.OCI
	if oci == nil {
		return nil
	}

	mediaType := oci.MediaType
	if mediaType == "" {
		mediaType = DefaultFirmwareMediaType
	}

	tag := oci.Tag
	if tag == "" {
		tag = fmt.Sprintf("%s-%d", bj.Name, bj.Generation)
	} else {
		tag = strings.ReplaceAll(tag, "${name}", bj.Name)
		tag = strings.ReplaceAll(tag, "${arch}", bj.Spec.Target.Architecture)
		tag = strings.ReplaceAll(tag, "${variant}", bj.Spec.Target.Variant)
	}

	ref := fmt.Sprintf("%s:%s", oci.Repository, tag)

	annotations := map[string]string{
		"org.opencontainers.image.title": bj.Name,
		"vnd.auto.target.board":          bj.Spec.Target.Board,
		"vnd.auto.target.platform":       bj.Spec.Target.Platform,
		"vnd.auto.target.architecture":   bj.Spec.Target.Architecture,
		"vnd.auto.build.generation":      fmt.Sprintf("%d", bj.Generation),
	}
	if bj.Spec.Target.Variant != "" {
		annotations["vnd.auto.target.variant"] = bj.Spec.Target.Variant
	}

	var annotationFlags string
	for k, v := range annotations {
		if v != "" {
			annotationFlags += fmt.Sprintf(" --annotation %s=%s", ShellQuote(k), ShellQuote(v))
		}
	}

	script := fmt.Sprintf(`#!/usr/bin/env sh
set -eu
export HOME=/tmp/oras-home
mkdir -p "$HOME"
cd $(workspaces.ws.path)
ARTIFACTS_DIR=%q
if [ ! -d "$ARTIFACTS_DIR" ] || [ -z "$(ls -A "$ARTIFACTS_DIR" 2>/dev/null)" ]; then
  echo "No artifacts to push"
  exit 0
fi

if [ -n "${REGISTRY_AUTH_FILE:-}" ]; then
  export DOCKER_CONFIG="$HOME/.docker"
  mkdir -p "$DOCKER_CONFIG"
  cp "$REGISTRY_AUTH_FILE" "$DOCKER_CONFIG/config.json"
fi

cd "$ARTIFACTS_DIR"
FILES=""
while IFS= read -r f; do
  FILES="$FILES $f:%s"
done < <(find . -type f | sed 's|^\./||')

echo "Pushing OCI artifact to %s"
ORAS_OUTPUT=$(oras push --insecure %s \
  --artifact-type %s \
  %s \
  $FILES 2>&1)
echo "$ORAS_OUTPUT"

DIGEST=$(echo "$ORAS_OUTPUT" | grep -o 'sha256:[a-f0-9]*' | tail -1)
echo "%s" > $(results.oci-ref.path)
echo "$DIGEST" > $(results.oci-digest.path)
`, bj.Spec.Artifacts.Path, mediaType, ref, ShellQuote(ref), ShellQuote(mediaType), annotationFlags, ref)

	ociEnv := make([]interface{}, len(envVars), len(envVars)+2)
	copy(ociEnv, envVars)
	ociEnv = append(ociEnv,
		map[string]interface{}{"name": "ARTIFACTS_DIR", "value": bj.Spec.Artifacts.Path},
	)

	volumes := []interface{}{}
	volumeMounts := []interface{}{}
	if oci.PushSecret != nil {
		ociEnv = append(ociEnv,
			map[string]interface{}{"name": "REGISTRY_AUTH_FILE", "value": "/etc/oci-push-secret/.dockerconfigjson"},
		)
		volumes = append(volumes, map[string]interface{}{
			"name": "push-secret",
			"secret": map[string]interface{}{
				"secretName": oci.PushSecret.Name,
			},
		})
		volumeMounts = append(volumeMounts, map[string]interface{}{
			"name":      "push-secret",
			"mountPath": "/etc/oci-push-secret",
			"readOnly":  true,
		})
	}

	allowPrivEsc := false
	var runAsUser int64 = 1000
	step := map[string]interface{}{
		"name":  "oras-push",
		"image": OrasImage,
		"env":   ociEnv,
		"securityContext": map[string]interface{}{
			"allowPrivilegeEscalation": allowPrivEsc,
			"runAsNonRoot":             true,
			"runAsUser":                runAsUser,
		},
		"script": script,
	}
	if len(volumeMounts) > 0 {
		step["volumeMounts"] = volumeMounts
	}

	taskSpec := map[string]interface{}{
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "mountPath": "/workspace"},
		},
		"results": []interface{}{
			map[string]interface{}{
				"name":        "oci-ref",
				"description": "Full OCI reference of the pushed artifact",
			},
			map[string]interface{}{
				"name":        "oci-digest",
				"description": "Immutable digest of the pushed manifest",
			},
		},
		"steps": []interface{}{step},
	}
	if len(volumes) > 0 {
		taskSpec["volumes"] = volumes
	}

	return map[string]interface{}{
		"name":     "oci-push",
		"taskSpec": taskSpec,
		"runAfter": []interface{}{runAfter},
		"workspaces": []interface{}{
			map[string]interface{}{"name": taskWorkspaceName, "workspace": workspaceName},
		},
	}
}

func buildEnvVars(bj *buildv1alpha1.BuildJob) []interface{} {
	sourceType := string(bj.Spec.Source.Type)
	if sourceType == "" {
		sourceType = "git"
	}
	vars := []interface{}{
		map[string]interface{}{"name": "BOB_NAME", "value": bj.Name},
		map[string]interface{}{"name": "BOB_NAMESPACE", "value": bj.Namespace},
		map[string]interface{}{"name": "BOB_SOURCE_TYPE", "value": sourceType},
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
	if bj.Spec.Source.Type == buildv1alpha1.SourceTypeRepo && bj.Spec.Source.Repo != nil {
		vars = append(vars, map[string]interface{}{"name": "BOB_MANIFEST_URL", "value": bj.Spec.Source.Repo.ManifestURL})
		vars = append(vars, map[string]interface{}{"name": "BOB_MANIFEST_BRANCH", "value": bj.Spec.Source.Repo.Branch})
	}
	return vars
}

func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
