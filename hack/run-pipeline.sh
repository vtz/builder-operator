#!/usr/bin/env bash
# Run every time you want to execute a build pipeline.
# Requires the cluster to be already running (setup-cluster.sh).
set -euo pipefail

PIPELINE_START=$(date +%s)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CR_FILE="${1:-${ROOT_DIR}/docs/examples/hello-world-shared-folder.yaml}"
NAMESPACE="bob-builds"

echo "Reapplying CRD..."
kubectl apply -f "${ROOT_DIR}/config/crd/bases/builder.sdv.cloud.redhat.com_buildjobs.yaml"

CR_NAME=$(grep 'name:' "${CR_FILE}" | head -1 | awk '{print $2}')
echo "Using BuildJob: ${CR_NAME} from ${CR_FILE}"

echo "Cleaning up previous run..."
kubectl delete buildjob "${CR_NAME}" -n "${NAMESPACE}" --ignore-not-found

echo "Building operator..."
go build -o "${ROOT_DIR}/bin/bob-operator" "${ROOT_DIR}/main.go"

echo "Starting operator in background..."
"${ROOT_DIR}/bin/bob-operator" &
OPERATOR_PID=$!
trap "echo 'Stopping operator...'; kill ${OPERATOR_PID} 2>/dev/null || true" EXIT
sleep 3

echo "Applying BuildJob CR..."
kubectl apply -f "${CR_FILE}"

echo "Waiting for PipelineRun to be created..."
sleep 5
RUN=$(kubectl get buildjob "${CR_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.currentPipelineRun}')
echo "PipelineRun: ${RUN}"

echo "Waiting for pipeline to complete..."
kubectl wait --for=condition=Succeeded "pipelinerun/${RUN}" -n "${NAMESPACE}" --timeout=300s || true

echo ""
echo "Status:"
kubectl get buildjob "${CR_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.phase}{"\n"}'
kubectl get taskruns -n "${NAMESPACE}" -l "tekton.dev/pipelineRun=${RUN}"

PIPELINE_END=$(date +%s)
echo ""
echo "Total time: $((PIPELINE_END - PIPELINE_START))s"
