# Troubleshooting

## `SoftwareBuild` stuck in `Pending`

- Verify Tekton CRDs are installed:
  - `kubectl get crd pipelineruns.tekton.dev`
- Verify compatible Tekton pipeline exists:
  - `kubectl get pipeline configurable-build-pipeline -n <namespace>`

## No `PipelineRun` created

- Check operator logs:
  - `kubectl logs deploy/builder-operator-controller-manager -n builder-operator-system`
- Ensure CRD and RBAC were applied from `config/`.

## `PipelineRun` fails

- Inspect run:
  - `kubectl describe pipelinerun <name> -n <namespace>`
- Inspect task logs:
  - `kubectl logs -l tekton.dev/pipelineRun=<name> -n <namespace> --all-containers=true`

## Secret-related failures

- Confirm secret exists in same namespace as `SoftwareBuild`.
- Confirm key names expected by your stage commands.
- Avoid printing secret values in scripts/logs.

## Artifact path not populated

- Verify `spec.destination.path`.
- Verify deploy command writes to that path.
- For local kind usage, confirm host mount is present.
