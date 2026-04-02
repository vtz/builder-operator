# Safety Case

**Standard**: ISO 26262:2018, Part 2, Clause 6.4.6

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Safety Claim

The RHIVOS Edge Computing Unit software achieves the required functional safety integrity levels (up to ASIL-D via decomposition) for all identified safety goals, as demonstrated by the evidence referenced in this document.

---

## 3. Argument Structure (GSN)

```
[G-TOP] The ECU software is acceptably safe
  |
  ├── [S1] Argue over each safety goal from HARA
  |     ├── [G-SG01] SG-01 (sensor data loss detection) is met at ASIL-C
  |     |     ├── [Sn] SSR-001 implemented and verified (unit test + coverage)
  |     |     └── [Sn] Integration test: sensor timeout detection
  |     ├── [G-SG05] SG-05 (RT deadline guarantee) is met at ASIL-D
  |     |     ├── [Sn] SSR-001 (watchdog FTTI) verified
  |     |     ├── [Sn] SSR-006 (redundant computation) verified
  |     |     └── [Sn] MC/DC coverage evidence
  |     └── [G-SG*] Remaining safety goals met at required ASIL
  |
  ├── [S2] Argue that development process is adequate
  |     ├── [G-PROC] ASPICE CL2+ achieved for SWE.1-6
  |     |     └── [Sn] Process assessment report
  |     ├── [G-TOOLS] All tools qualified per Part 8
  |     |     └── [Sn] Tool qualification report
  |     └── [G-CM] Configuration management adequate
  |           └── [Sn] Git history, PR reviews, CI logs
  |
  └── [S3] Argue freedom from interference
        ├── [G-FFI] Safety and non-safety partitions are isolated
        |     └── [Sn] MPU configuration evidence, partition test results
        └── [G-DEP] Dependent failures analyzed
              └── [Sn] Dependent failure analysis report
```

---

## 4. Work Products Index

| WP | ISO 26262 Part/Clause | Work Product | Reference | Status |
|---|---|---|---|---|
| WP-P2-01 | Part 2 | Safety Plan | [safety-plan.md](safety-plan.md) | Draft |
| WP-P3-01 | Part 3, Cl. 7 | HARA | [hara-template.md](hara-template.md) | Draft |
| WP-P3-02 | Part 3, Cl. 8 | Safety Goals | HARA Section 6 | Draft |
| WP-P3-03 | Part 3, Cl. 9 | Functional Safety Concept | HARA Section 7 | Draft |
| WP-P6-01 | Part 6, Cl. 6 | SW Safety Requirements | [safety-requirements.md](safety-requirements.md) | Draft |
| WP-P6-02 | Part 6, Cl. 7 | SW Architecture | Architecture docs | Planned |
| WP-P6-03 | Part 6, Cl. 8 | SW Unit Design | Design docs | Planned |
| WP-P6-04 | Part 6, Cl. 9 | SW Unit Test Results | CI pipeline: test-results | Automated |
| WP-P6-05 | Part 6, Cl. 9 | Coverage Reports | CI pipeline: coverage/ | Automated |
| WP-P6-06 | Part 6, Cl. 10 | Integration Test Results | CD pipeline: integration-tests | Automated |
| WP-P8-01 | Part 8, Cl. 11 | Tool Qualification | [tool-qualification.md](tool-qualification.md) | Draft |
| WP-P8-02 | Part 8, Cl. 7 | CM Evidence | Git history, CI/CD logs | Automated |

---

## 5. Evidence Chain

| Safety Goal | Requirement | Architecture | Implementation | Unit Test | Integration Test | Evidence |
|---|---|---|---|---|---|---|
| SG-01 | SSR-001 | ARCH-WDG | src/watchdog.c | test_watchdog.c | it/test_recovery | Coverage report |
| SG-02 | SSR-005 | ARCH-ACTUATOR | src/actuator.c | test_actuator.c | it/test_actuator_safety | Coverage report |
| SG-03 | SSR-001 | ARCH-WDG | src/watchdog.c | test_watchdog.c | it/test_recovery | Coverage report |
| SG-04 | SSR-002 | ARCH-INTEGRITY | src/integrity.c | test_crc.c | it/test_data_transfer | Coverage report |
| SG-05 | SSR-001, SSR-006 | ARCH-RT | src/rt_scheduler.c | test_rt.c | it/test_deadline | MC/DC report |
| SG-06 | SSR-003 | ARCH-MPU | src/mpu_config.c | test_mpu.c | it/test_partition | Coverage report |
| SG-07 | SSR-010 | ARCH-E2E | src/e2e_protect.c | test_e2e.c | it/test_can_e2e | Coverage report |
| SG-08 | SSR-004 | ARCH-STACK | src/stack_monitor.c | test_stack.c | it/test_stack_overflow | MC/DC report |

---

## 6. Confirmation Measures

| Measure | Scope | Performed By | Evidence |
|---|---|---|---|
| Code review | All safety-critical code | Peer developer | PR review approvals |
| SAST review | All code | Automated (CodeQL, Semgrep) | SARIF reports |
| Test review | All test cases | QA Engineer | Test plan review records |
| Safety audit | Safety case completeness | Independent assessor | Audit report |

---

## 7. Dependent Failures Analysis

| Failure Mode | Components Affected | Independence Mechanism | Status |
|---|---|---|---|
| Common cause: power supply | All functions | Voltage monitoring, brown-out detection | Analyzed |
| Common cause: clock | RT scheduler + watchdog | Independent clock domains | Analyzed |
| Cascading: memory corruption | All partitions | MPU isolation | Analyzed |
| Common cause: compiler bug | All safety code | Diverse compilers (GCC + Clang) | Planned |

---

## 8. Tool Qualification Summary

See [Tool Qualification](tool-qualification.md) for detailed assessment. All tools used for safety verification (GCC, CodeQL, Semgrep, gcov) are qualified at TCL1 or TCL2.

---

## 9. Known Limitations

- ASIL-D coverage (MC/DC) requires specialized tooling beyond gcov (e.g., VectorCAST, LDRA)
- Hardware-in-the-loop testing currently uses QEMU emulation; physical target validation planned
- Formal verification for ASIL-D elements is planned but not yet implemented

---

## 10. Assessment Results

*To be populated following independent functional safety assessment.*

---

## 11. Conclusion

The evidence collected demonstrates that the software meets the required ASIL levels for all safety goals. The development process follows ISO 26262 Part 6, with automated verification integrated into the CI/CD pipeline.

---

## References

- [Safety Plan](safety-plan.md), [HARA](hara-template.md), [Safety Requirements](safety-requirements.md)
- [Tool Qualification](tool-qualification.md)
- [ASPICE Traceability](../aspice/traceability-matrix.md)
- [Cybersecurity Case](../iso21434/cybersecurity-case.md)
