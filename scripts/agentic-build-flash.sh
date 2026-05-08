#!/usr/bin/env bash
#
# agentic-build-flash.sh -- End-to-end build and flash orchestration for body-ecu.
#
# Drives the MCU (Zephyr firmware) and/or HPC (Linux RPM + OS image) build-flash
# pipelines. Designed to be called by an AI coding agent (Cursor CLI, Claude Code)
# or by a developer from the terminal.
#
# Usage:
#   ./scripts/agentic-build-flash.sh [--target mcu|hpc|both] [OPTIONS]
#
# Options:
#   --target mcu|hpc|both    Which side to build and flash (default: both)
#   --board BOARD             Jumpstarter board selector (default: board=nucleo_h755zi_q)
#   --lease-duration DUR      Jumpstarter lease duration (default: 01:00:00)
#   --image-target TARGET     ADO image build target for HPC (default: qemu-autosd)
#   --skip-flash              Build only, do not flash
#   --skip-verify             Flash but skip post-flash verification
#   --dry-run                 Print commands without executing
#   -h, --help                Show this help

set -euo pipefail

TARGET="both"
BOARD_SELECTOR="board=nucleo_h755zi_q"
LEASE_DURATION="01:00:00"
IMAGE_TARGET="qemu-autosd"
SKIP_FLASH=false
SKIP_VERIFY=false
DRY_RUN=false
LEASE_NAME=""

usage() {
    sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

log() { echo "=== [$(date +%H:%M:%S)] $*" >&2; }
err() { echo "!!! [$(date +%H:%M:%S)] ERROR: $*" >&2; }

run() {
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [dry-run] $*" >&2
    else
        "$@"
    fi
}

cleanup() {
    if [[ -n "$LEASE_NAME" ]]; then
        log "Releasing Jumpstarter lease ${LEASE_NAME}..."
        jmp delete leases "$LEASE_NAME" 2>/dev/null || true
    fi
}
trap cleanup EXIT

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target)       TARGET="$2"; shift 2 ;;
        --board)        BOARD_SELECTOR="$2"; shift 2 ;;
        --lease-duration) LEASE_DURATION="$2"; shift 2 ;;
        --image-target) IMAGE_TARGET="$2"; shift 2 ;;
        --skip-flash)   SKIP_FLASH=true; shift ;;
        --skip-verify)  SKIP_VERIFY=true; shift ;;
        --dry-run)      DRY_RUN=true; shift ;;
        -h|--help)      usage ;;
        *)              err "Unknown option: $1"; usage ;;
    esac
done

if [[ "$TARGET" != "mcu" && "$TARGET" != "hpc" && "$TARGET" != "both" ]]; then
    err "Invalid --target: $TARGET (must be mcu, hpc, or both)"
    exit 1
fi

# ---------------------------------------------------------------------------
# MCU: Zephyr firmware build + flash
# ---------------------------------------------------------------------------
build_mcu() {
    log "MCU: Building Zephyr firmware (body-ecu-zephyr) ..."
    run bob build body-ecu-zephyr --local --wait

    log "MCU: Retrieving OCI artifact reference ..."
    local artifact_ref
    if [[ "$DRY_RUN" == "true" ]]; then
        artifact_ref="quay.io/myorg/body-ecu-firmware:body-ecu-zephyr-arm-v7m"
        log "MCU: [dry-run] Using placeholder artifact ref"
    else
        artifact_ref=$(bob artifacts body-ecu-zephyr 2>/dev/null | grep 'Reference:' | awk '{print $2}')
        if [[ -z "$artifact_ref" ]]; then
            err "MCU: No OCI artifact ref found (is the build using destination: oci?)"
            exit 1
        fi
    fi
    log "MCU: Artifact ref = ${artifact_ref}"
    echo "$artifact_ref"
}

flash_mcu() {
    local artifact_ref="$1"

    log "MCU: Creating Jumpstarter lease (selector=${BOARD_SELECTOR}) ..."
    if [[ "$DRY_RUN" == "true" ]]; then
        LEASE_NAME="dry-run-lease"
        run jmp create lease -l "$BOARD_SELECTOR" --duration "$LEASE_DURATION" -o name
    else
        LEASE_NAME=$(jmp create lease -l "$BOARD_SELECTOR" --duration "$LEASE_DURATION" -o name)
        if [[ -z "$LEASE_NAME" ]]; then
            err "MCU: Failed to create Jumpstarter lease"
            exit 1
        fi
    fi
    log "MCU: Lease acquired = ${LEASE_NAME}"

    log "MCU: Flashing firmware via Jumpstarter ..."
    run jmp shell --lease "$LEASE_NAME" -- j storage flash "oci://${artifact_ref}"
    log "MCU: Flash complete"
}

verify_mcu() {
    log "MCU: Verifying firmware via serial console ..."
    if [[ "$DRY_RUN" == "true" ]]; then
        run jmp shell --lease "$LEASE_NAME" -- j serial pipe --timeout 10
        log "MCU: [dry-run] Would check serial output for boot confirmation"
        return 0
    fi
    local output
    output=$(jmp shell --lease "$LEASE_NAME" -- j serial pipe --timeout 10 2>&1 || true)
    echo "$output" >&2
    if echo "$output" | grep -qi "booting\|ready\|zephyr"; then
        log "MCU: Firmware boot confirmed"
    else
        err "MCU: Could not confirm firmware boot from serial output"
        return 1
    fi
}

# ---------------------------------------------------------------------------
# HPC: RPM build + OS image compose + flash
# ---------------------------------------------------------------------------
build_hpc() {
    local rpm_dir="/tmp/body-ecu-rpms"

    log "HPC: Building RPM (body-ecu-mpu-rpm) ..."
    run bob build body-ecu-mpu-rpm --local --wait

    log "HPC: Downloading RPM artifacts ..."
    rm -rf "$rpm_dir"
    mkdir -p "$rpm_dir"
    run bob artifacts body-ecu-mpu-rpm --download "$rpm_dir"
    log "HPC: RPMs downloaded to ${rpm_dir}"
    ls -la "$rpm_dir"/*.rpm 2>/dev/null >&2 || true
    echo "$rpm_dir"
}

flash_hpc() {
    local rpm_dir="$1"

    log "HPC: Building OS image and flashing (target=${IMAGE_TARGET}) ..."
    if [[ "$DRY_RUN" == "true" ]]; then
        run caib build --target "$IMAGE_TARGET" --flash --extra-rpms "${rpm_dir}/*.rpm" --wait --follow
    else
        # shellcheck disable=SC2086
        caib build \
            --target "$IMAGE_TARGET" \
            --flash \
            --extra-rpms "${rpm_dir}"/*.rpm \
            --wait --follow
    fi
    log "HPC: Image built and flashed"
}

verify_hpc() {
    log "HPC: Verifying service via SSH ..."
    if [[ "$DRY_RUN" == "true" ]]; then
        run jmp shell --lease "$LEASE_NAME" -- j ssh -- systemctl status body-ecu-mpu-hostlike
        run jmp shell --lease "$LEASE_NAME" -- j ssh -- curl -sf http://localhost:8080/healthz
        log "HPC: [dry-run] Would check service status and health endpoint"
        return 0
    fi
    local status_output
    status_output=$(jmp shell --lease "$LEASE_NAME" -- \
        j ssh -- systemctl status body-ecu-mpu-hostlike 2>&1 || true)
    echo "$status_output" >&2
    if echo "$status_output" | grep -qi "active (running)"; then
        log "HPC: Service is running"
    else
        err "HPC: Service not running"
        return 1
    fi

    log "HPC: Checking health endpoint ..."
    local health_output
    health_output=$(jmp shell --lease "$LEASE_NAME" -- \
        j ssh -- curl -sf http://localhost:8080/healthz 2>&1 || true)
    echo "$health_output" >&2
    if [[ -n "$health_output" ]]; then
        log "HPC: Health endpoint responded"
    else
        log "HPC: Health endpoint not available (may not be implemented yet)"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
log "Starting agentic build-flash pipeline (target=${TARGET})"
echo "" >&2

MCU_ARTIFACT_REF=""
HPC_RPM_DIR=""

if [[ "$TARGET" == "mcu" || "$TARGET" == "both" ]]; then
    MCU_ARTIFACT_REF=$(build_mcu)
    if [[ "$SKIP_FLASH" != "true" ]]; then
        flash_mcu "$MCU_ARTIFACT_REF"
        if [[ "$SKIP_VERIFY" != "true" ]]; then
            verify_mcu
        fi
    fi
    echo "" >&2
fi

if [[ "$TARGET" == "hpc" || "$TARGET" == "both" ]]; then
    HPC_RPM_DIR=$(build_hpc)
    if [[ "$SKIP_FLASH" != "true" ]]; then
        flash_hpc "$HPC_RPM_DIR"
        if [[ "$SKIP_VERIFY" != "true" ]]; then
            verify_hpc
        fi
    fi
    echo "" >&2
fi

log "Pipeline complete (target=${TARGET})"
if [[ -n "$LEASE_NAME" ]]; then
    log "Jumpstarter lease ${LEASE_NAME} will be released on exit"
fi
