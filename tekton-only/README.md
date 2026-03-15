# Tekton-Only Build Pipeline

An alternative to the Builder Operator approach that uses Tekton Pipelines directly,
without any Custom Resources or operator binaries.

## Motivation

The `builder-operator/` approach requires:

1. A Go operator binary running in the cluster
2. A `SoftwareBuild` Custom Resource Definition (CRD)
3. The operator to reconcile CRs into Tekton PipelineRuns

This alternative removes all of that. Users write PipelineRun YAML files directly
and submit them to Tekton. The pipeline definition stays the same (5-stage configurable
pipeline), but there is no operator in the loop.

| Aspect             | Operator approach                 | Tekton-only approach            |
| ------------------ | --------------------------------- | ------------------------------- |
| User interface     | `SoftwareBuild` CR                | `PipelineRun` YAML              |
| Runtime dependency | Go operator process               | Tekton controller only          |
| Status tracking    | CR `.status` (synced by operator) | `PipelineRun` `.status` (native)|
| CRD required       | Yes                               | No                              |
| Custom Go code     | Yes                               | No                              |

## Quick start

### 1. Create the cluster

```bash
./hack/setup-cluster.sh
```

This creates a kind cluster, installs Tekton Pipelines, and applies the
`configurable-build-pipeline` Pipeline resource.

### 2. Run a build

```bash
./hack/run-build.sh examples/hello-world.yaml
```

The script creates the PipelineRun, waits for completion, and shows logs and
artifacts.

### 3. Check results

```bash
kubectl get pipelinerun -n hello-demo
kubectl describe pipelinerun/<name> -n hello-demo
ls -lR deployment/
```

## Available examples

| Example                                  | Image                                         | Description                                |
| ---------------------------------------- | --------------------------------------------- | ------------------------------------------ |
| `examples/hello-world.yaml`             | `gcc:13`                                      | C++ hello world, outputs to shared folder  |
| `examples/zephyr-hello-world.yaml`      | `ghcr.io/zephyrproject-rtos/ci-base:latest`   | Zephyr RTOS hello_world for native_sim     |
| `examples/openbsw-posix-freertos.yaml`  | `ubuntu:24.04`                                | Eclipse OpenBSW POSIX FreeRTOS build       |

## Writing your own PipelineRun

Create a YAML file following this structure:

```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: my-build-
  namespace: hello-demo
spec:
  pipelineRef:
    name: configurable-build-pipeline
  timeouts:
    pipeline: "15m"
  params:
    - name: containerImage
      value: "ubuntu:24.04"
    - name: fetchCommand
      value: "echo 'fetch stage'"
    - name: prebuildCommand
      value: "echo 'prebuild stage'"
    - name: buildCommand
      value: "echo 'build stage'"
    - name: postbuildCommand
      value: "echo 'postbuild stage'"
    - name: deployCommand
      value: "echo 'deploy stage'"
  workspaces:
    - name: shared-workspace
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 1Gi
```

### Parameters

| Parameter          | Description                                 |
| ------------------ | ------------------------------------------- |
| `containerImage`   | Container image for all stages (overridable per-stage in the Pipeline if needed) |
| `fetchCommand`     | Shell command for the fetch stage            |
| `prebuildCommand`  | Shell command for the prebuild stage         |
| `buildCommand`     | Shell command for the build stage            |
| `postbuildCommand` | Shell command for the postbuild stage        |
| `deployCommand`    | Shell command for the deploy stage           |

### Host mounts

The pipeline mounts two host paths (configured via kind):

- `/host-samples` (fetch task) -- read-only samples directory, mapped from `samples/`
- `/host-build` (deploy task) -- output directory, mapped from `deployment/`

## Directory structure

```
tekton-only/
  pipeline.yaml              # 5-stage configurable Pipeline (inline taskSpec)
  kind-config.yaml            # Kind cluster config with host mounts
  samples/                    # Source files mounted into the cluster
    hello-world-input.txt
    hello-world.cpp
  examples/                   # PipelineRun YAMLs (one per build scenario)
    hello-world.yaml
    zephyr-hello-world.yaml
    openbsw-posix-freertos.yaml
  hack/                       # Helper scripts
    setup-cluster.sh           # Create kind cluster + install Tekton
    run-build.sh               # Run a build from a PipelineRun YAML
  deployment/                  # Build artifacts appear here (host-mounted)
```
