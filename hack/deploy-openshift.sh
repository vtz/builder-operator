#!/usr/bin/env bash
# Deploy bob to an OpenShift cluster.
#
# Prerequisites:
#   - oc CLI installed and logged in
#   - OpenShift Pipelines (Tekton) installed via OLM
#   - Internal image registry route exposed
#
# Usage:
#   ./hack/deploy-openshift.sh              # build + push + deploy
#   ./hack/deploy-openshift.sh --skip-build  # deploy only (image already pushed)
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPERATOR_NS="bob-system"
BUILDS_NS="bob-builds"
SKIP_BUILD=false

for arg in "$@"; do
  case $arg in
    --skip-build) SKIP_BUILD=true ;;
  esac
done

echo "=== bob deploy to OpenShift ==="
echo ""

# ── Verify prerequisites ───────────────────────────────────────────────
echo "[1/9] Checking prerequisites..."
oc whoami > /dev/null 2>&1 || { echo "ERROR: not logged in. Run: oc login --server=<api>"; exit 1; }
echo "  Cluster:  $(oc whoami --show-server)"
echo "  User:     $(oc whoami)"
echo "  Arch:     $(oc get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')"

oc get csv -n openshift-operators 2>/dev/null | grep -q pipelines || {
  echo "ERROR: OpenShift Pipelines not installed."
  echo "Install it with:"
  echo "  oc apply -f - <<EOF"
  echo "apiVersion: operators.coreos.com/v1alpha1"
  echo "kind: Subscription"
  echo "metadata:"
  echo "  name: openshift-pipelines-operator-rh"
  echo "  namespace: openshift-operators"
  echo "spec:"
  echo "  channel: latest"
  echo "  name: openshift-pipelines-operator-rh"
  echo "  source: redhat-operators"
  echo "  sourceNamespace: openshift-marketplace"
  echo "EOF"
  exit 1
}
echo "  Tekton:   installed"

# ── Create namespaces ──────────────────────────────────────────────────
echo ""
echo "[2/9] Creating namespaces..."
oc create namespace "${OPERATOR_NS}" --dry-run=client -o yaml | oc apply -f -
oc create namespace "${BUILDS_NS}" --dry-run=client -o yaml | oc apply -f -

# ── Build and push operator image ──────────────────────────────────────
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}' 2>/dev/null || true)
INTERNAL_IMG="image-registry.openshift-image-registry.svc:5000/${OPERATOR_NS}/bob:latest"

if [ "$SKIP_BUILD" = false ]; then
  echo ""
  echo "[3/9] Building operator image..."
  cd "${ROOT_DIR}"
  podman build -t bob:latest .

  if [ -n "${REGISTRY_HOST}" ]; then
    echo ""
    echo "[4/9] Pushing to internal registry (${REGISTRY_HOST})..."
    podman login -u "$(oc whoami)" -p "$(oc whoami -t)" "${REGISTRY_HOST}" --tls-verify=false 2>/dev/null
    EXTERNAL_IMG="${REGISTRY_HOST}/${OPERATOR_NS}/bob:latest"
    podman tag bob:latest "${EXTERNAL_IMG}"
    podman push "${EXTERNAL_IMG}" --tls-verify=false
    IMG="${INTERNAL_IMG}"
  else
    echo ""
    echo "WARNING: No registry route found. Using podman image directly."
    echo "Consider pushing to quay.io instead:"
    echo "  podman push bob:latest quay.io/<user>/bob:latest"
    IMG="bob:latest"
  fi
else
  echo ""
  echo "[3/9] Skipping build (--skip-build)"
  echo "[4/9] Skipping push"
  IMG="${INTERNAL_IMG}"
fi

# ── Install CRDs ──────────────────────────────────────────────────────
echo ""
echo "[5/9] Installing CRDs..."
oc apply -f "${ROOT_DIR}/config/crd/bases/"
echo "  $(oc get crd | grep builder.sdv.cloud.redhat.com | wc -l | tr -d ' ') CRDs installed"

# ── Install RBAC ──────────────────────────────────────────────────────
echo ""
echo "[6/9] Setting up RBAC..."
oc apply -f "${ROOT_DIR}/config/rbac/role.yaml"

oc create serviceaccount bob-controller-manager -n "${OPERATOR_NS}" --dry-run=client -o yaml | oc apply -f -
oc create clusterrolebinding bob-manager-binding \
  --clusterrole=bob-manager-role \
  --serviceaccount="${OPERATOR_NS}:bob-controller-manager" \
  --dry-run=client -o yaml | oc apply -f -

oc adm policy add-role-to-user edit "system:serviceaccount:${OPERATOR_NS}:bob-controller-manager" -n "${BUILDS_NS}" 2>/dev/null || true

# ── Deploy the operator ───────────────────────────────────────────────
echo ""
echo "[7/9] Deploying operator (image: ${IMG})..."

oc apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bob-controller-manager
  namespace: ${OPERATOR_NS}
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
        app: bob-api
    spec:
      serviceAccountName: bob-controller-manager
      containers:
        - name: manager
          image: ${IMG}
          command: ["/manager"]
          args: ["--leader-elect", "--api-bind-address=:8082", "--cli-dir=/cli"]
          ports:
            - name: api
              containerPort: 8082
              protocol: TCP
          resources:
            limits:
              cpu: 500m
              memory: 256Mi
            requests:
              cpu: 100m
              memory: 128Mi
EOF

echo "  Waiting for operator pod to be ready..."
oc rollout status deployment/bob-controller-manager -n "${OPERATOR_NS}" --timeout=120s

# ── Expose Build API via Service + Route ──────────────────────────────
echo ""
echo "[8/9] Creating Build API Service and Route..."
oc apply -f "${ROOT_DIR}/config/manager/service.yaml"
oc apply -f "${ROOT_DIR}/config/manager/route.yaml"

BOB_ROUTE=$(oc get route bob-api -n "${OPERATOR_NS}" -o jsonpath='{.spec.host}' 2>/dev/null || echo "pending")
echo "  Service: bob-api.${OPERATOR_NS}.svc:8082"
echo "  Route:   https://${BOB_ROUTE}"

# ── Build the CLI ─────────────────────────────────────────────────────
echo ""
echo "[9/9] Building bob CLI..."
cd "${ROOT_DIR}"
go build -o bin/bob ./cmd/bob
echo "  Binary: ${ROOT_DIR}/bin/bob"
echo "  Install: sudo cp ${ROOT_DIR}/bin/bob /usr/local/bin/bob"

# ── Done ──────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo "  bob deployed successfully!"
echo "============================================"
echo ""
echo "Configure the bob CLI:"
echo "  export BOB_SERVER=https://${BOB_ROUTE}"
echo "  export BOB_TOKEN=\$(oc whoami -t)"
echo "  export BOB_NAMESPACE=${BUILDS_NS}"
echo ""
echo "Verify:"
echo "  oc get pods -n ${OPERATOR_NS}"
echo "  oc get crd | grep builder.sdv.cloud.redhat.com"
echo "  curl -k https://${BOB_ROUTE}/healthz"
echo ""
echo "Download bob CLI (share this with colleagues):"
echo "  curl -Lo bob https://${BOB_ROUTE}/v1/cli/darwin/arm64 && chmod +x bob   # Mac Apple Silicon"
echo "  curl -Lo bob https://${BOB_ROUTE}/v1/cli/darwin/amd64 && chmod +x bob   # Mac Intel"
echo "  curl -Lo bob https://${BOB_ROUTE}/v1/cli/linux/amd64  && chmod +x bob   # Linux x86_64"
echo "  curl -Lo bob https://${BOB_ROUTE}/v1/cli/linux/arm64  && chmod +x bob   # Linux arm64"
echo ""
echo "Deploy a sample BuildJob:"
echo "  oc apply -f ${ROOT_DIR}/docs/examples/zephyr-hello-world.yaml"
echo "  bob list"
echo ""
