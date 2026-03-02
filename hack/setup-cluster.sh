#!/usr/bin/env bash
# Run once per session to create the cluster and install Tekton.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Creating required host directories..."
mkdir -p "${ROOT_DIR}/deployment"

echo "Creating kind cluster..."
kind delete cluster --name crd-demo 2>/dev/null || true
kind create cluster --config "${ROOT_DIR}/kind-config.yaml"

echo "Installing Tekton Pipelines..."
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
echo "Waiting for Tekton controller pod to appear..."
sleep 10
kubectl wait --for=condition=ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=180s

echo "Setting up namespace, CRD and pipeline..."
kubectl create namespace hello-demo --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "${ROOT_DIR}/config/crd/bases/build.mycompany.io_softwarebuilds.yaml"
kubectl apply -f "${ROOT_DIR}/config/tekton/pipeline.yaml"

echo "Cluster ready. Run ./hack/run-pipeline.sh to execute a build."
