# Hazard Analysis and Risk Assessment (HARA)

**Standard**: ISO 26262:2018, Part 3, Clause 7

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Item Definition

The RHIVOS Edge Computing Unit processes sensor data, executes control algorithms, and manages actuator commands. It operates in mixed-criticality mode (ASIL and QM partitions).

---

## 3. Situation Analysis

| Mode | Description | Exposure |
|---|---|---|
| Highway driving | Vehicle at speed >80 km/h | E4 (High) |
| Urban driving | Vehicle at speed 30-80 km/h | E4 (High) |
| Parking/Low speed | Vehicle at speed <10 km/h | E3 (Medium) |
| Stationary/Parked | Vehicle stationary, engine off | E1 (Very low) |
| Maintenance mode | Service technician connected | E1 (Very low) |
| OTA update | Firmware update in progress | E2 (Low) |

---

## 4. Hazard Identification

| Hazard ID | Malfunction | Operational Situation | Hazardous Event |
|---|---|---|---|
| H-01 | Loss of sensor data input | Highway driving | Vehicle cannot perceive obstacles, potential collision |
| H-02 | Unintended actuator activation | Parking | Unexpected vehicle movement causing injury to bystanders |
| H-03 | Watchdog timeout / system reset | Highway driving | Temporary loss of control functions at speed |
| H-04 | Corrupted calibration data | Urban driving | Incorrect control algorithm behavior, degraded vehicle dynamics |
| H-05 | OTA update failure | OTA update | System left in inconsistent state, functions unavailable on next start |
| H-06 | Real-time task deadline miss | Highway driving | Safety-critical control loop delayed, incorrect actuator timing |
| H-07 | Memory corruption in inference engine | Urban driving | Incorrect perception/decision output leading to wrong control action |
| H-08 | CAN bus communication loss | Highway driving | Loss of coordination between ECUs, degraded vehicle behavior |
| H-09 | Power interruption during firmware write | OTA update | Bricked ECU, loss of all functions until physical recovery |
| H-10 | Stack overflow in interrupt handler | Highway driving | System crash, loss of safety-critical monitoring functions |

---

## 5. ASIL Classification

### 5.1 ASIL Lookup Table

| | C0 | C1 | C2 | C3 |
|---|---|---|---|---|
| **S1, E1** | QM | QM | QM | QM |
| **S1, E2** | QM | QM | QM | QM |
| **S1, E3** | QM | QM | QM | A |
| **S1, E4** | QM | QM | A | B |
| **S2, E1** | QM | QM | QM | QM |
| **S2, E2** | QM | QM | QM | A |
| **S2, E3** | QM | QM | A | B |
| **S2, E4** | QM | A | B | C |
| **S3, E1** | QM | QM | QM | A |
| **S3, E2** | QM | QM | A | B |
| **S3, E3** | QM | A | B | C |
| **S3, E4** | QM | B | C | D |

### 5.2 Hazard Classification

| Hazard | Severity | Exposure | Controllability | ASIL |
|---|---|---|---|---|
| H-01 | S3 (life-threatening) | E4 (highway) | C2 (normally controllable) | **C** |
| H-02 | S2 (severe injury) | E3 (parking) | C2 (normally controllable) | **A** |
| H-03 | S3 (life-threatening) | E4 (highway) | C2 (normally controllable) | **C** |
| H-04 | S2 (severe injury) | E4 (urban) | C1 (simply controllable) | **A** |
| H-05 | S1 (light injury) | E2 (OTA update) | C1 (simply controllable) | **QM** |
| H-06 | S3 (life-threatening) | E4 (highway) | C3 (difficult to control) | **D** |
| H-07 | S3 (life-threatening) | E4 (urban) | C2 (normally controllable) | **C** |
| H-08 | S3 (life-threatening) | E4 (highway) | C2 (normally controllable) | **C** |
| H-09 | S1 (light injury) | E2 (OTA update) | C0 (controllable) | **QM** |
| H-10 | S3 (life-threatening) | E4 (highway) | C3 (difficult to control) | **D** |

---

## 6. Safety Goals

| SG ID | Hazard(s) | Safety Goal | ASIL | Safe State | FTTI |
|---|---|---|---|---|---|
| SG-01 | H-01 | The system shall detect sensor data loss and transition to safe state within FTTI | C | Reduced functionality with driver warning | 100 ms |
| SG-02 | H-02 | The system shall prevent unintended actuator activation | A | Actuators de-energized | 50 ms |
| SG-03 | H-03 | The system shall recover from watchdog timeout within FTTI | C | System restart with safe defaults | 200 ms |
| SG-04 | H-04 | The system shall detect calibration data corruption before use | A | Use last-known-good calibration | 1 s |
| SG-05 | H-06 | The system shall guarantee RT task execution within deadline | D | Graceful degradation to safe state | 10 ms |
| SG-06 | H-07 | The system shall detect memory corruption in safety-critical regions | C | Stop affected function, alert driver | 50 ms |
| SG-07 | H-08 | The system shall detect CAN communication loss and degrade gracefully | C | Autonomous safe stop | 500 ms |
| SG-08 | H-10 | The system shall detect and recover from stack overflow | D | System reset with safe defaults | 10 ms |

---

## 7. Functional Safety Concept

| Mechanism | Safety Goals | Description |
|---|---|---|
| Hardware watchdog | SG-03, SG-05 | Independent hardware timer resets system if not serviced within FTTI |
| Memory Protection Unit | SG-06, SG-08 | MPU isolates safety-critical from non-critical partitions |
| CRC-32 integrity checks | SG-04 | All calibration data verified with CRC before use |
| Redundant computation | SG-05 | Dual-channel computation with cross-check for ASIL-D functions |
| E2E communication | SG-07 | End-to-end protection on inter-ECU CAN messages |
| Stack canaries | SG-08 | Compiler-inserted stack overflow detection (-fstack-protector-strong) |
| Graceful degradation | SG-01, SG-07 | Reduced functionality mode with driver notification |
| Safe state manager | All | Central component managing transitions to defined safe states |

---

## 8. Relationship to TARA

Cybersecurity threats that could cause safety-relevant hazards:

| Safety Goal | Related TARA Threat | Cybersecurity Goal |
|---|---|---|
| SG-04 (calibration integrity) | T-01 (OTA tampering) | CSG-01 |
| SG-07 (CAN communication) | T-02 (CAN injection), T-09 (CAN DoS) | CSG-02 |
| SG-01 (sensor data) | T-06 (sensor spoofing) | CSG-06 |
| SG-05 (RT deadlines) | T-09 (DoS) | CSG-02 |

---

## References

- [Safety Plan](safety-plan.md)
- [Safety Case](safety-case.md)
- [Safety Requirements](safety-requirements.md)
- [TARA](../iso21434/tara-template.md)
