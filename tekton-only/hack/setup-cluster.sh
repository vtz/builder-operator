#!/usr/bin/env bash
# Create a kind cluster with Tekton Pipelines installed.
# No CRDs or operator binaries required.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="crd-demo"

echo "Creating required host directories..."
mkdir -p "${ROOT_DIR}/deployment"

echo "Generating kind config with absolute paths..."
sed \
  -e "s|__DEPLOYMENT_DIR__|${ROOT_DIR}/deployment|" \
  -e "s|__SAMPLES_DIR__|${ROOT_DIR}/samples|" \
  "${ROOT_DIR}/kind-config.yaml" > "${ROOT_DIR}/kind-config-resolved.yaml"

echo "Creating kind cluster..."
kind delete cluster --name "${CLUSTER_NAME}" 2>/dev/null || true
kind create cluster --config "${ROOT_DIR}/kind-config-resolved.yaml"

# On macOS with Colima, Docker port forwards reach the VM but not the host.
# Set up an SSH tunnel so kubectl on the host can reach the API server.
if [[ "$(uname)" == "Darwin" ]] && command -v colima &>/dev/null && colima status &>/dev/null; then
  API_PORT=$(docker inspect "${CLUSTER_NAME}-control-plane" \
    --format '{{(index (index .NetworkSettings.Ports "6443/tcp") 0).HostPort}}')
  COLIMA_SSH_CONFIG="${HOME}/.colima/ssh_config"
  if [[ -f "${COLIMA_SSH_CONFIG}" ]]; then
    echo "Colima detected -- setting up SSH tunnel for port ${API_PORT}..."
    ssh -F "${COLIMA_SSH_CONFIG}" -f -N -L "${API_PORT}:127.0.0.1:${API_PORT}" colima
    sleep 2
  fi
fi

echo "Verifying cluster connectivity..."
kubectl cluster-info

echo "Installing Tekton Pipelines..."
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
echo "Waiting for Tekton controller..."
sleep 10
kubectl wait --for=condition=ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=180s
echo "Waiting for Tekton webhook..."
kubectl wait --for=condition=ready pod -l app=tekton-pipelines-webhook -n tekton-pipelines --timeout=120s
sleep 5

echo "Setting up namespace and pipeline..."
kubectl create namespace hello-demo --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "${ROOT_DIR}/pipeline.yaml"

echo ""
echo "Cluster ready. Run a build with:"
echo "  ./hack/run-build.sh examples/hello-world.yaml"
