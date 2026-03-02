# Builder Operator

`builder-operator` is a Kubernetes operator (Go + Kubebuilder style) that provides a single customizable CRD (`SoftwareBuild`) and reconciles each CR to a Tekton `PipelineRun`.

## What this operator provides

- One customer-facing CRD: `SoftwareBuild`
- Five configurable stages:
  - `fetch`
  - `prebuild`
  - `build`
  - `postbuild`
  - `deploy`
- Per-stage command and image override support
- Secrets via references (`secretRef`) for git/artifactory/registry credentials
- CR status synchronization from Tekton `PipelineRun` state

## Quickstart

1. Ensure your cluster has Tekton Pipelines installed.
2. Apply the CRD:

```bash
kubectl apply -f config/crd/bases/build.mycompany.io_softwarebuilds.yaml
```

3. Apply a Tekton pipeline compatible with this operator:

```bash
kubectl apply -f ../tekton/pipeline.yaml
```

4. Run the operator locally:

```bash
go run ./main.go
```

5. Create a sample CR:

```bash
kubectl apply -f config/samples/build_v1alpha1_softwarebuild.yaml
```

6. Inspect status:

```bash
kubectl get softwarebuild hello-cpp -n hello-demo -o yaml
kubectl get pipelineruns -n hello-demo
```

## Documentation

- [Architecture](docs/architecture.md)
- [API Reference](docs/api.md)
- [Examples](docs/examples)
- [Troubleshooting](docs/troubleshooting.md)

## Repository layout

- `api/v1alpha1`: CRD types
- `internal/controller`: reconcile logic
- `internal/tekton`: PipelineRun rendering logic
- `config`: manifests (CRD/RBAC/manager/sample)
- `docs`: architecture and usage docs
- `hack`: scripts for local validation
