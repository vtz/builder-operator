#!/usr/bin/env bash
# Convenience wrapper to build the Zephyr hello-world example.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "${ROOT_DIR}/hack/run-pipeline.sh" "${ROOT_DIR}/docs/examples/zephyr-hello-world.yaml"
