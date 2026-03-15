#!/usr/bin/env bash
# Run a build pipeline from a PipelineRun YAML file.
# Usage: ./hack/run-build.sh examples/hello-world.yaml [TIMEOUT]
#
# Requires the cluster to be running (setup-cluster.sh).
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <pipelinerun-yaml> [timeout]"
  echo "  Example: $0 examples/hello-world.yaml 10m"
  exit 1
fi

PIPELINE_START=$(date +%s)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIPELINERUN_FILE="${1}"
TIMEOUT="${2:-15m}"
NAMESPACE="hello-demo"

if [[ ! "${PIPELINERUN_FILE}" = /* ]]; then
  PIPELINERUN_FILE="${ROOT_DIR}/${PIPELINERUN_FILE}"
fi

if [[ ! -f "${PIPELINERUN_FILE}" ]]; then
  echo "Error: PipelineRun file not found: ${PIPELINERUN_FILE}"
  exit 1
fi

echo "Reapplying pipeline in case it changed..."
kubectl apply -f "${ROOT_DIR}/pipeline.yaml"

echo "Creating PipelineRun from ${PIPELINERUN_FILE}..."
RUN_RESOURCE="$(kubectl create -f "${PIPELINERUN_FILE}" -o name)"
RUN_NAME="${RUN_RESOURCE##*/}"
echo "PipelineRun created: ${RUN_NAME}"

echo "Waiting for PipelineRun completion (timeout: ${TIMEOUT})..."
if ! kubectl wait --for=condition=Succeeded "${RUN_RESOURCE}" -n "${NAMESPACE}" --timeout="${TIMEOUT}"; then
  echo ""
  echo "PipelineRun failed or timed out: ${RUN_NAME}"
  echo ""
  echo "Task logs:"
  kubectl logs -n "${NAMESPACE}" -l "tekton.dev/pipelineRun=${RUN_NAME}" --all-containers=true --tail=50 || true
  echo ""
  echo "Describe PipelineRun:"
  kubectl describe "${RUN_RESOURCE}" -n "${NAMESPACE}" || true
  exit 1
fi

echo ""
echo "PipelineRun succeeded: ${RUN_NAME}"
echo ""
echo "Task logs:"
kubectl logs -n "${NAMESPACE}" -l "tekton.dev/pipelineRun=${RUN_NAME}" --all-containers=true || true

echo ""
echo "TaskRuns:"
kubectl get taskruns -n "${NAMESPACE}" -l "tekton.dev/pipelineRun=${RUN_NAME}"

echo ""
echo "Artifacts:"
ls -lR "${ROOT_DIR}/deployment/" 2>/dev/null || echo "No artifacts found on host yet."

PIPELINE_END=$(date +%s)
echo ""
echo "Total time: $((PIPELINE_END - PIPELINE_START))s"
