#!/usr/bin/env bash
# Run every time you want to execute a build pipeline.
# Requires the cluster to be already running (setup-cluster.sh).
set -euo pipefail

PIPELINE_START=$(date +%s)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CR_NAME="hello-world-shared-folder"
NAMESPACE="hello-demo"

echo "Reapplying CRD and pipeline in case they changed..."
kubectl apply -f "${ROOT_DIR}/config/crd/bases/build.mycompany.io_softwarebuilds.yaml"
kubectl apply -f "${ROOT_DIR}/config/tekton/pipeline.yaml"

echo "Cleaning up previous run..."
kubectl delete softwarebuild "${CR_NAME}" -n "${NAMESPACE}" --ignore-not-found

echo "Building operator..."
go build -o "${ROOT_DIR}/bin/operator" "${ROOT_DIR}/main.go"

echo "Starting operator in background..."
"${ROOT_DIR}/bin/operator" &
OPERATOR_PID=$!
trap "echo 'Stopping operator...'; kill ${OPERATOR_PID} 2>/dev/null || true" EXIT
sleep 3

echo "Applying SoftwareBuild CR..."
kubectl apply -f "${ROOT_DIR}/docs/examples/hello-world-shared-folder.yaml"

echo "Waiting for PipelineRun to be created..."
sleep 5
RUN=$(kubectl get softwarebuild "${CR_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.currentPipelineRun}')
echo "PipelineRun: ${RUN}"

echo "Waiting for pipeline to complete..."
kubectl wait --for=condition=Succeeded "pipelinerun/${RUN}" -n "${NAMESPACE}" --timeout=300s || true

echo "Status:"
kubectl get softwarebuild "${CR_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.phase}{"\n"}'
kubectl get taskruns -n "${NAMESPACE}" -l "tekton.dev/pipelineRun=${RUN}"

echo "Artifacts:"
ls -lR "${ROOT_DIR}/deployment/" 2>/dev/null || echo "No artifacts found"

PIPELINE_END=$(date +%s)
echo "Total time: $((PIPELINE_END - PIPELINE_START))s"
