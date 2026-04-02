# Bidirectional Traceability Matrix

**Standard**: Automotive SPICE v3.1 (SWE.1 BP6, SWE.2 BP6, SWE.3 BP5, SWE.5 BP5)

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Traceability Concept

ASPICE requires **bidirectional traceability** across all development layers:

```
Requirements <-> Architecture <-> Detailed Design <-> Source Code <-> Unit Tests <-> Integration Tests <-> Qualification Tests
```

Each link must be traceable in both directions: forward (requirement -> test) and backward (test -> requirement).

---

## 3. Traceability Matrix

| Req ID | Safety Goal | Architecture | Design | Source File(s) | Unit Test(s) | Integration Test(s) | Qualification Test(s) | Status |
|---|---|---|---|---|---|---|---|---|
| SSR-001 | SG-01,03,05 | ARCH-WDG | DES-WDG-TIMER | src/watchdog.c, src/watchdog.h | test/test_watchdog.c:test_wdg_reset, test_wdg_ftti | it/test_system_recovery | qt/QT-WDG-001 | Complete |
| SSR-002 | SG-04 | ARCH-INTEGRITY | DES-CRC32 | src/integrity.c, src/integrity.h | test/test_crc.c:test_crc32_valid, test_crc32_corrupt | it/test_data_transfer | qt/QT-INT-001 | Complete |
| SSR-003 | SG-06 | ARCH-MPU | DES-MPU-CONFIG | src/mpu_config.c, src/mpu_config.h | test/test_mpu.c:test_partition_isolation | it/test_partition_boundary | qt/QT-MPU-001 | Complete |
| SSR-004 | SG-08 | ARCH-STACK | DES-STACK-MON | src/stack_monitor.c | test/test_stack.c:test_stack_canary, test_overflow_detect | it/test_stack_overflow_recovery | qt/QT-STK-001 | Complete |
| SSR-005 | SG-02 | ARCH-INPUT | DES-RANGE-CHK | src/input_validation.c | test/test_input.c:test_range_check, test_invalid_input | it/test_actuator_safety | qt/QT-INP-001 | Complete |
| SSR-006 | SG-05 | ARCH-REDUNDANT | DES-DUAL-COMPUTE | src/redundant_compute.c | test/test_redundant.c:test_cross_check, test_mismatch | it/test_dual_channel | qt/QT-RED-001 | Complete |
| SSR-007 | SG-04 | ARCH-SIGN | DES-SIG-VERIFY | src/sig_verify.c | test/test_sig.c:test_valid_sig, test_invalid_sig | it/test_ota_security | qt/QT-SIG-001 | Complete |
| SSR-008 | SG-01 | ARCH-DIAG | DES-DTC-LOG | src/dtc_logger.c | test/test_dtc.c:test_dtc_store, test_dtc_overflow | it/test_diagnostics | qt/QT-DTC-001 | Complete |
| SSR-009 | SG-06 | ARCH-BOOT | DES-BOOT-VERIFY | src/boot_verify.c | test/test_boot.c:test_boot_chain | it/test_secure_boot | qt/QT-BOOT-001 | Complete |
| SSR-010 | SG-07 | ARCH-E2E | DES-E2E-PROTECT | src/e2e_protect.c, src/e2e_protect.h | test/test_e2e.c:test_crc_seq_alive | it/test_can_e2e | qt/QT-E2E-001 | Complete |
| SSR-011 | SG-01 | ARCH-SENSOR | DES-SENSOR-MON | src/sensor_monitor.c | test/test_sensor.c:test_timeout_detect | it/test_sensor_loss | qt/QT-SNS-001 | Complete |
| SSR-012 | SG-05 | ARCH-RT | DES-RT-SCHED | src/rt_scheduler.c | test/test_rt.c:test_deadline_met, test_jitter | it/test_timing | qt/QT-RTS-001 | Complete |
| SSR-013 | SG-03 | ARCH-POWER | DES-BROWNOUT | src/power_monitor.c | test/test_power.c:test_brownout_detect | it/test_power_fail | qt/QT-PWR-001 | Incomplete |
| SSR-014 | SG-06 | ARCH-RAM | DES-RAM-TEST | src/ram_test.c | test/test_ram.c:test_march_pattern | it/test_ram_integrity | qt/QT-RAM-001 | Incomplete |
| SSR-015 | SG-08 | ARCH-IRQ | DES-IRQ-LIMIT | src/irq_manager.c | test/test_irq.c:test_nesting_limit | it/test_irq_safety | qt/QT-IRQ-001 | Incomplete |

---

## 4. Coverage Analysis

| Metric | Count | Percentage |
|---|---|---|
| Total requirements | 15 | - |
| Requirements with architecture link | 15 | 100% |
| Requirements with design link | 15 | 100% |
| Requirements with source code link | 15 | 100% |
| Requirements with unit tests | 15 | 100% |
| Requirements with integration tests | 15 | 100% |
| Requirements with qualification tests | 15 | 100% |
| Fully traced (all layers complete) | 12 | 80% |
| Incomplete traceability | 3 | 20% |

**Gaps identified**: SSR-013, SSR-014, SSR-015 have incomplete implementation and testing.

---

## 5. Automated Traceability

The CI pipeline validates traceability via the `compliance-report.yml` workflow:

1. Parse structured traceability YAML data (see Section 6)
2. Verify every requirement has links at all layers
3. Verify every test has a backward link to a requirement
4. Generate coverage report
5. Flag gaps as quality gate warnings

---

## 6. Traceability Data Format (YAML)

Machine-readable traceability data for CI pipeline processing:

```yaml
traceability:
  - req_id: SSR-001
    safety_goal: [SG-01, SG-03, SG-05]
    architecture: ARCH-WDG
    design: DES-WDG-TIMER
    source_files:
      - src/watchdog.c
      - src/watchdog.h
    unit_tests:
      - test/test_watchdog.c:test_wdg_reset
      - test/test_watchdog.c:test_wdg_ftti
    integration_tests:
      - it/test_system_recovery
    qualification_tests:
      - qt/QT-WDG-001
    status: complete

  - req_id: SSR-002
    safety_goal: [SG-04]
    architecture: ARCH-INTEGRITY
    design: DES-CRC32
    source_files:
      - src/integrity.c
      - src/integrity.h
    unit_tests:
      - test/test_crc.c:test_crc32_valid
      - test/test_crc.c:test_crc32_corrupt
    integration_tests:
      - it/test_data_transfer
    qualification_tests:
      - qt/QT-INT-001
    status: complete

  - req_id: SSR-010
    safety_goal: [SG-07]
    architecture: ARCH-E2E
    design: DES-E2E-PROTECT
    source_files:
      - src/e2e_protect.c
      - src/e2e_protect.h
    unit_tests:
      - test/test_e2e.c:test_crc_seq_alive
    integration_tests:
      - it/test_can_e2e
    qualification_tests:
      - qt/QT-E2E-001
    status: complete
```

---

## References

- [Safety Requirements](../iso26262/safety-requirements.md)
- [HARA](../iso26262/hara-template.md)
- [Process Assessment](process-assessment.md)
- [Pipeline Architecture](../../docs/pipeline-architecture.md)
