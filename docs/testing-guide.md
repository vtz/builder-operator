# Testing bob — Guide for Colleagues

How to test `bob` (the builder) against the shared OpenShift cluster.

---

## 0. What you need

One thing: the **Build API URL**. Ask the person who deployed bob for it.
It looks like: `https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com`

No cluster login required. No `oc` needed. Just the URL.

---

## 1. Get the `bob` binary

### Option A: Download from the cluster (recommended)

Set the URL you got in step 0, then download:

```bash
BOB_SERVER=https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com

# Mac (Apple Silicon — M1/M2/M3/M4)
curl -Lo bob $BOB_SERVER/v1/cli/darwin/arm64
chmod +x bob && sudo mv bob /usr/local/bin/

# Mac (Intel)
curl -Lo bob $BOB_SERVER/v1/cli/darwin/amd64
chmod +x bob && sudo mv bob /usr/local/bin/

# Linux (x86_64)
curl -Lo bob $BOB_SERVER/v1/cli/linux/amd64
chmod +x bob && sudo mv bob /usr/local/bin/

# Linux (arm64)
curl -Lo bob $BOB_SERVER/v1/cli/linux/arm64
chmod +x bob && sudo mv bob /usr/local/bin/
```

### Option B: Copy from the dist/ folder

If someone shared the repo with you, pre-built binaries are in `dist/`:

```bash
# Mac Apple Silicon
cp dist/bob-darwin-arm64 /usr/local/bin/bob && chmod +x /usr/local/bin/bob

# Mac Intel
cp dist/bob-darwin-amd64 /usr/local/bin/bob && chmod +x /usr/local/bin/bob

# Linux x86_64
sudo cp dist/bob-linux-amd64 /usr/local/bin/bob && sudo chmod +x /usr/local/bin/bob

# Linux arm64
sudo cp dist/bob-linux-arm64 /usr/local/bin/bob && sudo chmod +x /usr/local/bin/bob
```

### Option C: Build from source

Requires Go 1.22+:

```bash
cd builder-operator
make build-cli
# Binary at: bin/bob
```

### Verify

```bash
bob --help
```

---

## 2. Configure the CLI

Two environment variables. Add to your `~/.zshrc` or `~/.bashrc`:

```bash
export BOB_SERVER=https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com
export BOB_NAMESPACE=bob-builds
```

Replace the `BOB_SERVER` value with the URL from step 0.

### Verify the connection

```bash
bob list
```

You should see an empty list or existing BuildJobs. If you get a connection
error, check that the server is reachable:

```bash
curl -k $BOB_SERVER/healthz
# Should return: ok
```

---

## 3. Build the body-ecu repo — Step by step

### Step 1: Check what's already on the cluster

```bash
bob list
```

If you see `body-ecu-nucleo` in the list, skip to Step 3.

### Step 2: Create the BuildJob (one-time, needs `oc`)

If the BuildJob doesn't exist yet, someone with cluster access needs to
create it once. This is typically done by the person who deployed bob.

If the deploy script was run with `--bootstrap=all`, the BuildJobs are
already created and you can skip this step.

Otherwise:

```bash
oc apply -f docs/examples/zephyr-nucleo-cross.yaml
```

This tells bob:
- **What to build:** `https://github.com/vtz/body-ecu` (branch `feature/openbsw-integration`)
- **Toolchain:** Zephyr CI base image
- **Target board:** `nucleo_h755zi_q` (STM32H755 Cortex-M7)
- **Stages:** fetch (west init + update) --> build (west build) --> package (copy artifacts)

### Step 3: Trigger a build

```bash
bob build body-ecu-nucleo
```

This sends a request to the Build API, which triggers a Tekton PipelineRun
on the cluster. The build runs in a container with the Zephyr SDK -- nothing
is installed on your machine.

### Step 4: Watch the build

```bash
bob show body-ecu-nucleo
```

Shows the current phase (Pending --> Running --> Succeeded/Failed) and the
status of each stage.

### Step 5: Stream logs

```bash
bob logs body-ecu-nucleo
```

Streams the build output to your terminal in real time.

### Step 6: Get the artifacts

When the build succeeds, list available artifacts:

```bash
bob artifacts body-ecu-nucleo
```

Download them to your machine:

```bash
bob artifacts body-ecu-nucleo --download ./out
```

This downloads the firmware binaries to `./out/`:
- `zephyr.bin` -- raw binary for flashing
- `zephyr.hex` -- Intel HEX format
- `zephyr.elf` -- ELF with debug symbols

---

## 4. Try other examples

These need someone with `oc` access to create the BuildJobs first, or use
the deploy script with `--bootstrap=all` to pre-load everything. Once
created, anyone can trigger builds with `bob`.

### Zephyr Hello World (native simulation -- no hardware needed)

```bash
bob build zephyr-hello-world
bob logs zephyr-hello-world
```

### OpenBSW POSIX FreeRTOS

```bash
bob build openbsw-posix-freertos
bob logs openbsw-posix-freertos
```

---

## 5. Common commands

| Command | What it does |
|---------|-------------|
| `bob list` | List all BuildJobs and their status |
| `bob build <name>` | Trigger a build |
| `bob show <name>` | Detailed status + stages |
| `bob logs <name>` | Stream build logs |
| `bob artifacts <name>` | List build artifacts |
| `bob artifacts <name> -d ./out` | Download artifacts to local dir |
| `bob delete <name>` | Delete a BuildJob |

All commands accept `--server`, `-n` (namespace) as flags if you prefer not
to use environment variables.

---

## 6. Toolchain images — why they matter

Bob's first design principle: **the container IS the toolchain.** Each build
stage runs in a separate Tekton Task (= separate Pod), so anything you
`apt-get install` in one stage is gone by the next. The workspace PVC persists
files on disk, but system-level packages (`/usr/bin/cmake`, etc.) live in the
container filesystem and are wiped between stages.

**Don't install build tools at runtime.** Use a toolchain image that already
has everything.

### Good: toolchain image has all the tools

```yaml
spec:
  toolchain:
    image: ghcr.io/zephyrproject-rtos/ci-base:v0.29.1  # cmake, ninja, python, west, SDK — all pre-installed
  stages:
    - name: fetch
      command: "west init -l source && west update"
    - name: build
      command: "west build -b ${BOB_BOARD} source/app"   # cmake is available — it's in the image
```

### Bad: installing deps at runtime (fragile, slow, not reproducible)

```yaml
spec:
  toolchain:
    image: ubuntu:24.04      # bare image — has nothing
  stages:
    - name: deps
      command: "apt-get update && apt-get install -y cmake ninja-build"
    - name: build
      command: "cmake --build ."   # FAILS — cmake is gone (different container)
```

### Building a custom toolchain image

If no public image has what you need, create one. Example for OpenBSW:

```dockerfile
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake ninja-build g++ python3 make \
    && rm -rf /var/lib/apt/lists/*
```

Build, push, and reference it:

```bash
podman build -t quay.io/myorg/openbsw-toolchain:latest -f Dockerfile.toolchain .
podman push quay.io/myorg/openbsw-toolchain:latest
```

```yaml
spec:
  toolchain:
    image: quay.io/myorg/openbsw-toolchain:latest
  stages:
    - name: configure
      command: "cd source && cmake --preset posix-freertos"     # cmake is there
    - name: build
      command: "cd source && cmake --build --preset posix-freertos"  # still there
```

### Why not merge stages into one container?

You might wonder: why not run all stages in the same container so deps persist?

Because each stage being a separate Task means:

- **Tekton Chains** generates per-Task attestations for SLSA provenance
- **Enterprise Contract** policies can verify each stage independently
- **Security contexts** can differ per stage (e.g., network for fetch, no network for build)
- **Retries and timeouts** work per stage

These properties are essential for Red Hat's Trusted Software Supply Chain
Framework (TSSF) and SLSA compliance. The toolchain image is the right
abstraction: it makes deps a build-time concern (image layer), not a
runtime concern (shell commands).

---

## 7. Troubleshooting

### "BOB_SERVER is required"

You haven't set the `BOB_SERVER` environment variable:

```bash
export BOB_SERVER=https://...   # the URL from step 0
bob list
```

### "connection refused" or timeout

Check that the Build API is reachable:

```bash
curl -k $BOB_SERVER/healthz
```

### Build stuck in Pending

Tekton may be scheduling the PipelineRun. Ask someone with cluster access
to check:

```bash
oc get pipelineruns -n bob-builds
```

---

## 8. Running the test suite

### Unit tests (no cluster needed)

```bash
cd builder-operator
go test ./... -v
```

### Integration tests (require envtest / kubebuilder-assets)

```bash
# Install kubebuilder assets (one-time)
make envtest
export KUBEBUILDER_ASSETS=$(bin/setup-envtest use 1.31 -p path)
go test ./internal/controller/ -v
```

### What the tests cover

| Package | Coverage areas |
|---------|---------------|
| `api/v1alpha1` | JSON round-trip, conditions, phase constants |
| `internal/tekton` | PipelineRun generation, stages, env vars, caches, timeout, artifact upload env, configurable API host/port, no hardcoded bob-system |
| `internal/controller` | `syncStatusFromPipelineRun` (succeeded/failed/running/pending), `mergeCondition`, `nextRunNumber`, `ensureCachePVCs`, `extractTaskRunResult` (found/missing/empty), `taskRunPhase`, `buildTaskRun` (inline/git, no --tls-verify=false, pinned image) |
| `internal/buildapi` | List/get/create/run/delete, artifact upload (success, invalid gzip, oversized file, too many files, path traversal) |
| `internal/buildapi/client` | List, get, run, delete, logs, auth header |
| `cmd/bob` | Command wiring, flag validation |

### Artifact upload limits

The Build API enforces configurable upload safeguards:

| Limit | Default | Flag |
|-------|---------|------|
| Total upload size | 1 GiB | `--max-upload-bytes` |
| Files per upload | 1000 | `--max-upload-files` |
| Per-file size | 512 MiB | `--max-file-bytes` |
| Upload timeout | 5 min | (hardcoded in server) |

### Configurable API endpoint

Generated pipeline tasks no longer hardcode `bob-api.bob-system.svc`.
The API host defaults to `bob-api.<buildjob-namespace>.svc` and can be
overridden with `--pipeline-api-host` and `--pipeline-api-port` flags.

---

## 9. For deployers — Setting up bob on a new cluster

If you're the one deploying bob, use the deploy script:

```bash
# Full deploy with all example builds pre-loaded
./hack/deploy-openshift.sh --bootstrap=all

# Deploy with specific examples only
./hack/deploy-openshift.sh --bootstrap=zephyr-nucleo-cross,zephyr-hello-world

# Deploy without any example CRs
./hack/deploy-openshift.sh

# See available example CRs
./hack/deploy-openshift.sh --list-examples

# Re-deploy with existing image (skip build/push)
./hack/deploy-openshift.sh --skip-build --bootstrap=all
```

The script handles:
- Prerequisite checks (oc login, Tekton)
- Namespace creation (`bob-system`, `bob-builds`)
- Internal registry route exposure (auto)
- Image build and push
- CRD and RBAC installation
- Operator deployment with proper resource limits and artifact storage
- Service + Route for the Build API
- Optional bootstrapping of example BuildJob CRs

After deploying, share the `BOB_SERVER` URL with colleagues. They only
need the URL and the `bob` binary — no cluster access required.
