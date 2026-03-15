# Approach Comparison: Operator vs Tekton-Only

## Operator + SoftwareBuild CR

### Benefits

- **Abstraction / simpler user surface** -- Users write a domain-specific YAML (`SoftwareBuild`) with structured fields like `source.git.url`, `destination.type: registry`, `stages.build.command`. They never need to know about Tekton's `PipelineRun`, workspaces, or `volumeClaimTemplate`. The operator translates intent into Tekton objects.

- **Schema validation at the API level** -- The CRD has Kubebuilder validation markers (e.g. `Enum=git;pvc;hostPath`, `Pattern` on git URLs). Invalid specs are rejected on `kubectl apply` before anything runs.

- **Platform guardrails** -- The architecture separates "customer-owned inputs" from "platform-owned behavior." The operator controls the PipelineRun shape, RBAC, security defaults, and allowed destinations. Users can't arbitrarily modify Tekton objects. This is important for a managed service.

- **Unified status** -- `kubectl get softwarebuild` gives you `phase`, `artifactURI`, `failureReason`, per-stage progress, and Kubernetes conditions -- all in one place. No need to know the PipelineRun name or query Tekton directly.

- **Extensibility hooks** -- The reconciler is a natural place to add policy enforcement (admission webhooks), retry/backoff logic, notifications, audit logging, or integration with external systems.

- **GitOps-friendly** -- A `SoftwareBuild` CR can live in a Git repo and be applied by ArgoCD/Flux. The operator handles the lifecycle, so the desired state is declarative and idempotent.

### Drawbacks

- **Operational overhead** -- You must build, deploy, and run the Go operator binary. It needs its own Deployment, ServiceAccount, RBAC (ClusterRole with permissions for PipelineRuns, events, secrets), and monitoring.

- **Development complexity** -- Requires Go expertise, Kubebuilder scaffolding, CRD generation, controller-runtime knowledge. The codebase is ~400 lines of Go plus CRD YAML, tests, and docs.

- **Extra failure surface** -- If the operator crashes or its RBAC is wrong, no PipelineRuns get created. The reconciliation loop (10s polling) adds latency vs direct submission. The status sync logic is another place for bugs.

- **CRD installation required** -- The `SoftwareBuild` CRD must be installed cluster-wide before anything works. This adds a deployment step and a cluster-admin dependency.

- **Indirection** -- Debugging requires tracing from CR to PipelineRun to TaskRun to pod logs. When something fails, you have to check `.status.failureReason` on the CR, then often fall back to `kubectl describe pipelinerun` anyway.

---

## Tekton-Only

### Benefits

- **Zero runtime dependencies beyond Tekton** -- No operator process, no CRD, no Go binary. Just `kubectl create -f pipelinerun.yaml` and Tekton does the rest. One less thing to deploy, monitor, and debug.

- **Faster to set up and iterate** -- The setup script is ~40 lines instead of requiring `go build` + operator startup + CRD installation. Changing the pipeline or adding a new build is just editing YAML.

- **Direct access to Tekton features** -- Users get the full Tekton API: `timeouts`, `retries`, `when` expressions, custom workspaces, `finally` tasks, result passing between tasks. The operator currently exposes only a subset (e.g. no retries, no `finally` block).

- **Transparent debugging** -- One layer: PipelineRun -> TaskRun -> Pod. `kubectl describe pipelinerun` and `kubectl logs` give you everything. No operator logs to correlate.

- **Lower barrier for contributors** -- YAML-only, no Go knowledge needed. Anyone who knows Tekton can add examples or modify the pipeline.

### Drawbacks

- **No abstraction** -- Users must understand Tekton's PipelineRun spec: `pipelineRef`, `params`, `workspaces`, `volumeClaimTemplate`. The YAML is more verbose and less domain-specific. Compare the SoftwareBuild CR (~25 lines) vs the PipelineRun YAML (~45 lines).

- **No input validation** -- There's no schema checking that the params make sense. A typo in `containerImage` or a missing `fetchCommand` won't be caught until the task pod fails at runtime.

- **No platform guardrails** -- Anyone with permission to create PipelineRuns can use any image, mount any hostPath, or write anywhere. There's no central control point to enforce policy. You'd need separate admission webhooks (OPA/Gatekeeper, Kyverno) to replicate this.

- **No unified status view** -- There's no single resource to `kubectl get` that shows build phase + artifact location + failure reason. You have to query PipelineRuns directly and interpret Tekton's native conditions.

- **Script-dependent UX** -- The `run-build.sh` script papers over the wait/log/artifact workflow, but it's not a Kubernetes-native experience. Without the script, users must chain `kubectl create`, `kubectl wait`, `kubectl logs` manually.

- **Harder to integrate with GitOps** -- PipelineRuns with `generateName` are inherently non-idempotent. ArgoCD/Flux can't reconcile them the same way they can a `SoftwareBuild` CR (which the operator makes idempotent).

---

## Practical consideration

Red Hat already ships the automotive-image-builder. Extending it with a `SoftwareBuild` CRD and controller to support additional toolchains would be incremental work -- a new types file, a new controller, and RBAC entries, all wired into the existing operator binary. The operational overhead listed above as a drawback of the operator approach is therefore largely already paid for: the operator is already running in the cluster, and the patterns for adding new build types are established.

---

## When to use which

| Scenario | Better approach |
|---|---|
| Quick spike / local dev / trying out builds | Tekton-only |
| Multi-tenant platform with policy enforcement | Operator |
| Team without Go expertise | Tekton-only |
| GitOps-managed build definitions | Operator |
| Need full Tekton feature set (retries, finally, when) | Tekton-only |
| Production service with SLA on build status | Operator |
| Minimal cluster footprint | Tekton-only |
