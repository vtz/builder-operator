# API Reference

> **Note:** This document previously described a legacy `SoftwareBuild` CRD.
> The canonical CRD is now **`BuildJob`** (see [Architecture — API naming history](architecture.md)).

## Group and Version

- Group: `builder.sdv.cloud.redhat.com`
- Version: `v1alpha1`
- Kind: `BuildJob`

## `BuildJob.spec`

### `toolchain`

- `image` (string, default: `ubuntu:24.04`)
- `serviceAccountName` (optional)

### `source`

- `type`: `git` | `pvc`
- `git` (when `type=git`):
  - `url`
  - `revision` (default: `main`)
- `pvc` (when `type=pvc`):
  - `claimName`

### `target`

- `board` (optional)
- `platform` (optional)
- `architecture` (optional)
- `variant` (optional)

### `stages`

User-defined ordered list of named stages. Each stage contains:

- `name` (required)
- `command` (required)
- `image` (optional override — uses toolchain image by default)

### `artifacts`

- `path` (optional) — workspace-relative path to collect artifacts from
- `destination` (optional): `pvc` | `oci`

### `caches`

List of cache mounts. Each entry:

- `name` — subpath on the shared cache PVC
- `mountPath` — where to mount in the container

### `timeout`

- Optional `metav1.Duration` (e.g. `30m`).

## `BuildJob.status`

- `phase`: `Pending` | `Running` | `Succeeded` | `Failed`
- `currentPipelineRun`: active Tekton run name
- `commitSHA`: resolved git commit SHA
- `artifactURI`: output location
- `failureReason`: failure reason when failed
- `stages[]`: stage-level progress snapshots
- `conditions[]`: Kubernetes conditions
- `runCount`: number of runs created
- `lastRunAt`: timestamp of last run trigger
