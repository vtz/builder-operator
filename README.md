# bob — The Builder

`bob` is a Kubernetes operator for building embedded software on OpenShift (or any Tekton-capable cluster). It builds firmware and software for any target — Zephyr, OpenBSW, FreeRTOS, AUTOSAR Classic, bare-metal C, or any toolchain that runs in a container.

## What bob provides

- **BuildJob CRD** — declare what to build, which toolchain container to use, and what board/platform to target.
- **Target-aware builds** — first-class `board`, `platform`, and `architecture` fields with `${BOB_BOARD}`, `${BOB_PLATFORM}`, `${BOB_ARCH}` variable substitution.
- **Flexible stages** — user-defined stage names and ordering (not limited to fixed five).
- **Per-stage image overrides** — use different containers for different stages (e.g. flash tool for deploy).
- **Inline Tekton pipelines** — no pre-installed Pipeline resource needed; the operator generates the full PipelineRun.
- **Security** — `allowPrivilegeEscalation: false` on all build steps, shell-quoted commands.

## Quickstart

1. Ensure your cluster has Tekton Pipelines installed.
2. Apply the CRD:

```bash
kubectl apply -f config/crd/bases/builder.sdv.cloud.redhat.com_buildjobs.yaml
```

3. Run the operator locally:

```bash
go run ./main.go
```

4. Create a sample BuildJob:

```bash
kubectl apply -f docs/examples/zephyr-hello-world.yaml
```

5. Inspect status:

```bash
kubectl get buildjob -n bob-builds
kubectl get pipelineruns -n bob-builds
```

## Local development with Kind

```bash
./hack/setup-cluster.sh    # create kind cluster + install Tekton
./hack/run-zephyr.sh       # build Zephyr hello_world
./hack/run-openbsw.sh      # build OpenBSW posix-freertos
```

## Examples

| Example | Target | File |
|---------|--------|------|
| Zephyr hello_world (native_sim) | `zephyr` / `native` | [zephyr-hello-world.yaml](docs/examples/zephyr-hello-world.yaml) |
| Zephyr cross-compile (Nucleo) | `zephyr` / `arm` | [zephyr-nucleo-cross.yaml](docs/examples/zephyr-nucleo-cross.yaml) |
| OpenBSW posix-freertos | `openbsw` / `x86` | [openbsw-posix-freertos.yaml](docs/examples/openbsw-posix-freertos.yaml) |
| C++ hello world | `cmake` / `native` | [hello-world-shared-folder.yaml](docs/examples/hello-world-shared-folder.yaml) |

## Repository layout

```
api/v1alpha1/       CRD types (BuildJob)
internal/controller/ reconcile logic (BuildJobReconciler)
internal/tekton/     PipelineRun generation
config/              manifests (CRD, RBAC, manager)
docs/examples/       example BuildJob CRs
hack/                scripts for local validation
```
