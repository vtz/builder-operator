#!/usr/bin/env bash
# Deploy bob to an OpenShift cluster.
#
# Prerequisites:
#   - oc CLI installed and logged in
#   - OpenShift Pipelines (Tekton) installed via OLM
#
# Usage:
#   ./hack/deploy-openshift.sh                                  # build + push + deploy
#   ./hack/deploy-openshift.sh --skip-build                     # deploy only (image already pushed)
#   ./hack/deploy-openshift.sh --bootstrap=all                  # deploy + apply all example CRs
#   ./hack/deploy-openshift.sh --bootstrap=body-ecu-nucleo      # deploy + apply specific CR(s)
#   ./hack/deploy-openshift.sh --bootstrap=body-ecu-nucleo,zephyr-hello-world
#
# Bootstrap flags (one per example CR, all off by default):
#   --bootstrap=all                         Apply all example CRs
#   --bootstrap=name1,name2,...             Apply specific CRs by filename (without .yaml)
#   --list-examples                         List available example CRs and exit
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPERATOR_NS="bob-system"
BUILDS_NS="bob-builds"
SKIP_BUILD=false
BOOTSTRAP=""
LIST_EXAMPLES=false

for arg in "$@"; do
  case $arg in
    --skip-build) SKIP_BUILD=true ;;
    --bootstrap=*) BOOTSTRAP="${arg#*=}" ;;
    --list-examples) LIST_EXAMPLES=true ;;
  esac
done

EXAMPLES_DIR="${ROOT_DIR}/docs/examples"

if [ "$LIST_EXAMPLES" = true ]; then
  echo "Available example CRs:"
  for f in "${EXAMPLES_DIR}"/*.yaml; do
    name=$(basename "$f" .yaml)
    kind=$(grep -m1 'kind:' "$f" | awk '{print $2}')
    crname=$(grep -m1 'name:' "$f" | awk '{print $2}')
    echo "  ${name}  (${kind}: ${crname})"
  done
  exit 0
fi

echo "=== bob deploy to OpenShift ==="
echo ""

# ── Verify prerequisites ───────────────────────────────────────────────
echo "[1/10] Checking prerequisites..."
oc whoami > /dev/null 2>&1 || { echo "ERROR: not logged in. Run: oc login --server=<api>"; exit 1; }
echo "  Cluster:  $(oc whoami --show-server)"
echo "  User:     $(oc whoami)"
echo "  Arch:     $(oc get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')"

oc get csv -n openshift-operators 2>/dev/null | grep -q pipelines || {
  echo "ERROR: OpenShift Pipelines not installed."
  echo "Install it with:"
  cat <<'TEKTON'
  oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-pipelines-operator-rh
  namespace: openshift-operators
spec:
  channel: latest
  name: openshift-pipelines-operator-rh
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
TEKTON
  exit 1
}
echo "  Tekton:   installed"

# ── Create namespaces ──────────────────────────────────────────────────
echo ""
echo "[2/10] Creating namespaces..."
oc create namespace "${OPERATOR_NS}" --dry-run=client -o yaml | oc apply -f -
oc create namespace "${BUILDS_NS}" --dry-run=client -o yaml | oc apply -f -

# ── Ensure internal registry route is exposed ─────────────────────────
echo ""
echo "[3/10] Checking container registry..."
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}' 2>/dev/null || true)

if [ -z "${REGISTRY_HOST}" ]; then
  echo "  Internal registry route not exposed. Exposing it now..."
  oc patch configs.imageregistry.operator.openshift.io/cluster --type merge \
    -p '{"spec":{"defaultRoute":true}}' 2>/dev/null || true
  sleep 5
  REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}' 2>/dev/null || true)
fi

if [ -n "${REGISTRY_HOST}" ]; then
  echo "  Registry: ${REGISTRY_HOST}"
else
  echo "  WARNING: Could not expose internal registry route."
  echo "  You may need to push to an external registry (quay.io) manually."
fi

INTERNAL_IMG="image-registry.openshift-image-registry.svc:5000/${OPERATOR_NS}/bob:latest"

# ── Build and push operator image ──────────────────────────────────────
if [ "$SKIP_BUILD" = false ]; then
  echo ""
  echo "[4/10] Building operator image..."
  cd "${ROOT_DIR}"
  podman build -t bob:latest .

  if [ -n "${REGISTRY_HOST}" ]; then
    echo ""
    echo "[5/10] Pushing to internal registry (${REGISTRY_HOST})..."
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
  echo "[4/10] Skipping build (--skip-build)"
  echo "[5/10] Skipping push"
  IMG="${INTERNAL_IMG}"
fi

# ── Install CRDs ──────────────────────────────────────────────────────
echo ""
echo "[6/10] Installing CRDs..."
oc apply -f "${ROOT_DIR}/config/crd/bases/"
echo "  $(oc get crd | grep builder.sdv.cloud.redhat.com | wc -l | tr -d ' ') CRDs installed"

# ── Install RBAC ──────────────────────────────────────────────────────
echo ""
echo "[7/10] Setting up RBAC..."
oc apply -f "${ROOT_DIR}/config/rbac/role.yaml"

oc create serviceaccount bob-controller-manager -n "${OPERATOR_NS}" --dry-run=client -o yaml | oc apply -f -
oc create clusterrolebinding bob-manager-binding \
  --clusterrole=bob-manager-role \
  --serviceaccount="${OPERATOR_NS}:bob-controller-manager" \
  --dry-run=client -o yaml | oc apply -f -

oc adm policy add-role-to-user edit "system:serviceaccount:${OPERATOR_NS}:bob-controller-manager" -n "${BUILDS_NS}" 2>/dev/null || true

# ── Deploy the operator ───────────────────────────────────────────────
echo ""
echo "[8/10] Deploying operator (image: ${IMG})..."

# Use the canonical manager.yaml as the deployment spec, patching in the image
sed "s|image: controller:latest|image: ${IMG}|" "${ROOT_DIR}/config/manager/manager.yaml" | oc apply -f -

echo "  Waiting for operator pod to be ready..."
oc rollout status deployment/bob-controller-manager -n "${OPERATOR_NS}" --timeout=120s

# ── Expose Build API via Service + Route ──────────────────────────────
echo ""
echo "[9/10] Creating Build API Service and Route..."
oc apply -f "${ROOT_DIR}/config/manager/service.yaml"
oc apply -f "${ROOT_DIR}/config/manager/route.yaml"

BOB_ROUTE=$(oc get route bob-api -n "${OPERATOR_NS}" -o jsonpath='{.spec.host}' 2>/dev/null || echo "pending")
echo "  Service: bob-api.${OPERATOR_NS}.svc:8082"
echo "  Route:   https://${BOB_ROUTE}"

# ── Bootstrap example CRs ─────────────────────────────────────────────
echo ""
echo "[10/10] Bootstrapping example CRs..."

if [ -z "${BOOTSTRAP}" ]; then
  echo "  Skipped (use --bootstrap=all or --bootstrap=name1,name2 to apply example CRs)"
  echo "  Available: $(ls "${EXAMPLES_DIR}"/*.yaml 2>/dev/null | xargs -n1 basename | sed 's/.yaml//' | tr '\n' ', ' | sed 's/,$//')"
else
  if [ "${BOOTSTRAP}" = "all" ]; then
    echo "  Applying ALL example CRs..."
    for f in "${EXAMPLES_DIR}"/*.yaml; do
      name=$(basename "$f" .yaml)
      echo "    -> ${name}"
      oc apply -f "$f"
    done
  else
    IFS=',' read -ra SELECTED <<< "${BOOTSTRAP}"
    for name in "${SELECTED[@]}"; do
      name=$(echo "$name" | xargs) # trim whitespace
      f="${EXAMPLES_DIR}/${name}.yaml"
      if [ -f "$f" ]; then
        echo "    -> ${name}"
        oc apply -f "$f"
      else
        echo "    WARNING: ${name}.yaml not found in ${EXAMPLES_DIR}/"
        echo "    Available: $(ls "${EXAMPLES_DIR}"/*.yaml 2>/dev/null | xargs -n1 basename | sed 's/.yaml//' | tr '\n' ', ' | sed 's/,$//')"
      fi
    done
  fi
fi

# ── Done ──────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo "  bob deployed successfully!"
echo "============================================"
echo ""
echo "Configure the bob CLI:"
echo "  export BOB_SERVER=https://${BOB_ROUTE}"
echo "  export BOB_NAMESPACE=${BUILDS_NS}"
echo ""
echo "Verify:"
echo "  oc get pods -n ${OPERATOR_NS}"
echo "  oc get crd | grep builder.sdv.cloud.redhat.com"
echo "  curl -sk https://${BOB_ROUTE}/healthz"
echo ""
echo "Download bob CLI (share with colleagues — no oc login needed):"
echo "  curl -sLo bob https://${BOB_ROUTE}/v1/cli/darwin/arm64 && chmod +x bob   # Mac Apple Silicon"
echo "  curl -sLo bob https://${BOB_ROUTE}/v1/cli/darwin/amd64 && chmod +x bob   # Mac Intel"
echo "  curl -sLo bob https://${BOB_ROUTE}/v1/cli/linux/amd64  && chmod +x bob   # Linux x86_64"
echo "  curl -sLo bob https://${BOB_ROUTE}/v1/cli/linux/arm64  && chmod +x bob   # Linux arm64"
echo ""
echo "Quick start:"
echo "  bob list                                    # see available builds"
echo "  bob build body-ecu-nucleo                   # trigger a build"
echo "  bob logs body-ecu-nucleo                    # stream logs"
echo "  bob artifacts body-ecu-nucleo --download .  # download firmware"
echo ""
