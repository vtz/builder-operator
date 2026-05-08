# Add health-check endpoint to SOME/IP service bridge

## Labels
`enhancement`, `hpc`

## Description

The body-ecu HPC (posix-mpu) application runs a SOME/IP client bridge that
connects the MCU's CAN-based signals to cloud services. There is currently no
way for the container orchestrator or monitoring system to determine whether the
bridge is healthy and actively forwarding signals.

## Desired Behavior

Add a lightweight HTTP health-check endpoint on a configurable port (default
`8080`) that reports:

- **`/healthz`** -- returns `200 OK` if the process is alive
- **`/readyz`** -- returns `200 OK` only when:
  - The SOME/IP service discovery has completed
  - At least one signal subscription is active
  - The last signal was received within the configurable staleness window (default 30s)

## Affected Files

- `source/platforms/posix-mpu/src/main.cpp` -- application entry point, needs HTTP server init
- `source/platforms/posix-mpu/src/someip_bridge.cpp` -- bridge status accessors
- `source/platforms/posix-mpu/CMakeLists.txt` -- add HTTP server dependency

## Acceptance Criteria

- [ ] `/healthz` returns `200` when the process is running
- [ ] `/readyz` returns `200` only when SOME/IP discovery is complete and signals are flowing
- [ ] `/readyz` returns `503` if no signal received in the staleness window
- [ ] Health port is configurable via `--health-port` CLI flag or `BODY_ECU_HEALTH_PORT` env var
- [ ] RPM builds cleanly via `rpmbuild` in the bob pipeline
- [ ] Verified on target: `curl http://localhost:8080/healthz` returns `200` after image flash
