# Troubleshooting

> **Note:** This document previously used the legacy CRD name `SoftwareBuild`.
> The canonical CRD is now **`BuildJob`** (see [Architecture — API naming history](architecture.md)).

## `BuildJob` stuck in `Pending`

- Verify Tekton CRDs are installed:
  - `kubectl get crd pipelineruns.tekton.dev`
- Check that the operator is running:
  - `kubectl logs deploy/bob-operator -n bob-system`

## No `PipelineRun` created

- Check operator logs:
  - `kubectl logs deploy/bob-operator -n bob-system`
- Ensure CRD and RBAC were applied from `config/`.

## `PipelineRun` fails

- Inspect run:
  - `kubectl describe pipelinerun <name> -n <namespace>`
- Inspect task logs:
  - `kubectl logs -l tekton.dev/pipelineRun=<name> -n <namespace> --all-containers=true`
- Use the CLI: `bob logs <buildjob-name>`

## Secret-related failures

- Confirm secret exists in same namespace as the `BuildJob`.
- Confirm key names expected by your stage commands.
- Avoid printing secret values in scripts/logs.

## Artifact path not populated

- Verify `spec.artifacts.path` is set in the BuildJob.
- Verify stage commands write to that path.
- Check Build API logs for upload errors.
