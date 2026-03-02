# API Reference

## Group and Version

- Group: `build.mycompany.io`
- Version: `v1alpha1`
- Kind: `SoftwareBuild`

## `SoftwareBuild.spec`

### `runtime`

- `image` (string, default: `ubuntu:24.04`)
- `serviceAccountName` (optional)

### `source`

- `type`: `git` | `pvc` | `hostPath`
- `git` (when `type=git`):
  - `url`
  - `revision` (default: `main`)
  - `credentialsSecretRef` (`name`, optional `key`)
- `pvc` (when `type=pvc`):
  - `claimName`
  - `path` (default: `/`)
- `hostPath` (when `type=hostPath`):
  - `path`

### `stages`

Each stage contains:

- `command` (required)
- `image` (optional override)

Stages:
- `fetch`
- `prebuild`
- `build`
- `postbuild`
- `deploy`

### `destination`

- `type`: `sharedFolder` | `registry` | `artifactory` | `quay`
- `path` (optional)
- `repository` (optional)
- `credentialsSecretRef` (`name`, optional `key`)

### `timeoutSeconds`

- Optional positive integer.

## `SoftwareBuild.status`

- `phase`: `Pending` | `Running` | `Succeeded` | `Failed`
- `currentPipelineRun`: active Tekton run name
- `artifactURI`: output location
- `failureReason`: failure reason when failed
- `stages[]`: stage-level progress snapshots
- `conditions[]`: Kubernetes conditions

## Secret model

Secrets are referenced by name from CR fields. Credentials are not embedded directly in the CR.

Examples:

- Git:
  - `spec.source.git.credentialsSecretRef.name`
- Deploy destination:
  - `spec.destination.credentialsSecretRef.name`
