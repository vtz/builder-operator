# Claude AI Instructions

This file provides instructions for Claude AI when working with the
builder-operator (Bob) project.

## What This Project Is

Bob (builder-operator) is a Kubernetes operator that manages firmware and
software builds for automotive workloads. It watches `BuildJob` Custom
Resources and generates Tekton `PipelineRun`s to execute multi-stage build
pipelines. The companion `bob` CLI lets developers trigger builds, inspect
artifacts, and stream logs from their terminal.

## Project Structure

```
api/v1alpha1/        CRD types (BuildJob, BuildJobList)
cmd/bob/             CLI entry points (build, list, show, logs, artifacts, …)
internal/
  buildapi/          BuildJob helpers and client wrappers
  controller/        Kubernetes controller (Reconcile loop)
  tekton/            Tekton PipelineRun generation from BuildJob specs
config/              Kustomize manifests (CRD, RBAC, manager deployment)
deployment/          Helm chart / OLM bundle
docs/examples/       Example BuildJob YAMLs (body-ecu, firmware, RPM)
hack/                Dev scripts (install CRDs, local run, etc.)
scripts/             Agentic workflow tooling
  agentic-build-flash.sh   Orchestration script for build + flash
  issues/                  Seed GitHub issues for agent-driven fixes
.cursor/
  mcp.json                 Jumpstarter MCP server config
  rules/                   Cursor rules for agentic workflows
main.go              Operator entry point
Makefile             Build targets
```

## Key Commands

```bash
# Build everything
go build ./...

# Run tests
go test ./... -count=1 -race -v

# Lint
go vet ./...
golangci-lint run

# Run the operator locally (requires kubeconfig)
go run ./main.go

# Build the container image
make docker-build IMG=bob:latest

# Install CRDs into the cluster
make install

# All-in-one (fmt, vet, lint, test, build)
make all
```

## Coding Conventions

- Go 1.22+, standard library preferred over external dependencies.
- All exported functions and types must have doc comments.
- Error handling: always check and propagate errors; never silently discard
  a `Close()` return value on writable resources.
- Tests: table-driven tests preferred; use `t.Run` sub-tests.
- Linting: code must pass `golangci-lint` (v1.61) with the project config.

## CRD and API

The primary CRD is `BuildJob` (`api/v1alpha1/`). Key spec fields:

- `toolchain.image` — container image for the build environment.
- `source.git.url` / `source.git.revision` — where to clone source.
- `target.board` / `target.platform` — build target metadata.
- `stages[]` — ordered list of build steps (`name`, `command`, optional `image`).
- `artifacts.destination` — `oci` or `pvc`; controls where build outputs go.
- `caches[]` — PVC-backed caches mounted into the build pod.

## CLI (`bob`)

The CLI lives in `cmd/bob/` and provides:

| Command | Purpose |
|---------|---------|
| `bob build <name>` | Trigger a BuildJob (with `--local` for dev iteration) |
| `bob list` | List BuildJobs in the namespace |
| `bob show <name>` | Show BuildJob details and status |
| `bob logs <name>` | Stream build logs |
| `bob artifacts <name>` | Show or download OCI/PVC artifacts |
| `bob delete <name>` | Delete a BuildJob |
| `bob sync` | Sync local BuildJob YAMLs to the cluster |

## Agentic Workflow

This repo includes tooling for AI-agent-driven development on the body-ecu
codebase. See `.cursor/rules/agentic-build-flash.mdc` for the full workflow
(issue → code change → build → flash → verify → PR). The orchestration
script at `scripts/agentic-build-flash.sh` automates the MCU and HPC
build-flash pipelines.
