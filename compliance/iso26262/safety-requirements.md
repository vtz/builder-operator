# Software Safety Requirements

**Standard**: ISO 26262:2018, Part 6, Clause 6

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Derivation

These requirements are derived from the Safety Goals (SG-01 through SG-08) defined in the [HARA](hara-template.md) and the Functional Safety Concept.

---

## 3. Software Safety Requirements

| Req ID | Linked SG | Description | ASIL | Verification Method | Test ID | Status |
|---|---|---|---|---|---|---|
| SSR-001 | SG-01,03,05 | Watchdog shall reset system within FTTI (10-200ms per SG) if main task exceeds deadline | D | Unit test + MC/DC coverage | TC-WDG-001 | Draft |
| SSR-002 | SG-04 | All safety-critical data shall use CRC-32 for integrity verification before processing | C | Unit test + branch coverage | TC-CRC-001 | Draft |
| SSR-003 | SG-06 | Memory Protection Unit shall isolate safety-critical partitions from QM partitions | D | Unit test + integration test | TC-MPU-001 | Draft |
| SSR-004 | SG-08 | Stack overflow detection shall trigger safe state transition within 10ms | D | Unit test + MC/DC coverage | TC-STK-001 | Draft |
| SSR-005 | SG-02 | All external inputs shall be range-checked against defined min/max before processing | B | Unit test + branch coverage | TC-INP-001 | Draft |
| SSR-006 | SG-05 | ASIL-D functions shall use redundant computation with cross-check; mismatch triggers safe state | D | Unit test + MC/DC coverage | TC-RED-001 | Draft |
| SSR-007 | SG-04 | Firmware update shall verify cryptographic signature before application to flash | B | Unit test + integration test | TC-SIG-001 | Draft |
| SSR-008 | SG-01 | Diagnostic trouble codes shall be logged for all detected faults within 100ms | A | Unit test + statement coverage | TC-DTC-001 | Draft |
| SSR-009 | SG-06 | Boot integrity verification shall complete before any safety function activates | C | Integration test | TC-BOOT-001 | Draft |
| SSR-010 | SG-07 | Inter-ECU CAN communication shall use E2E protection (CRC + sequence counter + alive counter) | D | Unit test + MC/DC coverage | TC-E2E-001 | Draft |
| SSR-011 | SG-01 | Sensor data timeout shall be detected within 50ms and reported to safe state manager | C | Unit test + branch coverage | TC-SNS-001 | Draft |
| SSR-012 | SG-05 | Real-time task scheduler shall guarantee execution within configured period +/- 1ms jitter | D | Unit test + timing analysis | TC-RTS-001 | Draft |
| SSR-013 | SG-03 | Power supply voltage shall be monitored; brown-out below threshold triggers safe state | B | Integration test | TC-PWR-001 | Draft |
| SSR-014 | SG-06 | RAM test (march pattern) shall execute at startup and periodically during operation | C | Unit test + integration test | TC-RAM-001 | Draft |
| SSR-015 | SG-08 | Interrupt nesting depth shall be limited to prevent stack exhaustion | D | Static analysis + unit test | TC-IRQ-001 | Draft |

---

## 4. ASIL Decomposition

ASIL decomposition allows distributing safety requirements across redundant elements:

| Original ASIL | Decomposed To | Condition |
|---|---|---|
| D | D(B) + D(B) | Two independent elements, each ASIL-B |
| D | D(C) + D(A) | Two independent elements |
| C | C(B) + C(A) | Two independent elements |
| B | B(A) + B(A) | Two independent elements |

**Example**: SSR-006 (ASIL-D redundant computation) decomposes into:
- SSR-006a: Primary computation channel (ASIL-B)
- SSR-006b: Secondary computation channel (ASIL-B)
- SSR-006c: Cross-check comparator (ASIL-B)

Independence argument: Channels run on separate CPU cores with independent memory regions (MPU-enforced per SSR-003).

---

## 5. Verification Methods per ASIL

| Method | ASIL-A | ASIL-B | ASIL-C | ASIL-D | CI/CD Mapping |
|---|---|---|---|---|---|
| Statement coverage | HR | HR | HR | HR | gcov (ci.yml) |
| Branch coverage | R | HR | HR | HR | gcov --branch (ci.yml) |
| MC/DC coverage | - | R | HR | HR | Specialized tool |
| Code review | R | HR | HR | HR | PR approval |
| Static analysis | HR | HR | HR | HR | CodeQL + Semgrep (ci.yml) |
| Unit testing | HR | HR | HR | HR | make test (ci.yml) |
| Integration testing | R | HR | HR | HR | cd.yml integration-tests |
| Back-to-back testing | - | - | R | HR | Planned |
| Formal verification | - | - | R | HR | Planned |

**R** = Recommended, **HR** = Highly Recommended

---

## 6. Software Architecture Requirements

| Req ID | Description | ASIL | Rationale |
|---|---|---|---|
| SAR-001 | Safety-critical and QM software shall execute in separate MPU-protected partitions | D | Freedom from interference |
| SAR-002 | Safety-critical tasks shall have highest priority in the RT scheduler | D | Timing independence |
| SAR-003 | Shared memory between partitions shall use hardware-enforced access control | C | Freedom from interference |
| SAR-004 | Safety-relevant state machines shall be explicitly defined with all transitions documented | B | Deterministic behavior |

---

## 7. Traceability

Full bidirectional traceability is maintained in the [ASPICE Traceability Matrix](../aspice/traceability-matrix.md):

Safety Goals (HARA) -> Safety Requirements (this doc) -> Architecture -> Design -> Code -> Tests

---

## References

- [HARA](hara-template.md), [Safety Plan](safety-plan.md), [Safety Case](safety-case.md)
- [Tool Qualification](tool-qualification.md)
- [Traceability Matrix](../aspice/traceability-matrix.md)
