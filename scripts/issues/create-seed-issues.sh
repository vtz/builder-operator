#!/usr/bin/env bash
set -euo pipefail

REPO="${1:-vtz/body-ecu}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Creating seed issues on ${REPO} ==="

echo ""
echo "--- Bug: CAN frame drop (MCU) ---"
gh issue create \
  --repo "$REPO" \
  --title "CAN message handler drops frames under high bus load" \
  --body-file "${SCRIPT_DIR}/bug-mcu-can-frame-drop.md" \
  --label "bug" --label "mcu"

echo ""
echo "--- Feature: health-check endpoint (HPC) ---"
gh issue create \
  --repo "$REPO" \
  --title "Add health-check endpoint to SOME/IP service bridge" \
  --body-file "${SCRIPT_DIR}/feature-hpc-health-check.md" \
  --label "enhancement" --label "hpc"

echo ""
echo "=== Done. Issues created on ${REPO} ==="
