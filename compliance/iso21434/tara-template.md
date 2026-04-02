# Threat Analysis and Risk Assessment (TARA)

**Standard**: ISO/SAE 21434:2021, Clause 15
**Document Type**: Work Product WP-15-01

## 1. Document Control

| Field | Value |
|---|---|
| Version | 0.1 |
| Date | YYYY-MM-DD |
| Author | [Name] |
| Reviewer | [Name] |
| Approver | [Name] |
| Status | Draft |
| Classification | Confidential |

| Version | Date | Author | Changes |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Initial TARA creation |

---

## 2. Scope and Item Definition

### 2.1 Item Description

The item under analysis is the **RHIVOS Edge Computing Unit (ECU)**, an embedded computing platform running Red Hat In-Vehicle OS. It performs real-time sensor data processing, actuator control, and edge AI inference for automotive applications.

### 2.2 System Boundaries

The ECU interfaces with:

| Interface | Protocol | Direction | Trust Boundary |
|---|---|---|---|
| Vehicle CAN Bus | CAN 2.0B / CAN FD | Bidirectional | Internal vehicle network |
| Ethernet | TCP/IP, SOME/IP | Bidirectional | Internal vehicle network |
| OTA Update Server | HTTPS + OSTree | Inbound | External (internet) |
| Bluetooth (BLE) | BLE 5.0 | Inbound | External (wireless) |
| JTAG/SWD Debug Port | JTAG | Inbound | Physical access |
| USB | USB 2.0 | Inbound | Physical access |
| Telematics Control Unit | Ethernet | Bidirectional | Internal vehicle |

### 2.3 Assumptions

- The vehicle network is not directly exposed to the internet
- Physical access to the ECU requires bypassing vehicle access controls
- The OTA update channel traverses the public internet via cellular connection
- The ECU operates in safety-relevant contexts up to ASIL-B

---

## 3. Asset Identification

| Asset ID | Asset Name | Description | Confidentiality | Integrity | Availability |
|---|---|---|---|---|---|
| A-01 | Firmware Image | OS and application firmware stored in flash | M | H | H |
| A-02 | Cryptographic Keys | Secure boot keys, TLS certs, signing keys | H | H | M |
| A-03 | Calibration Data | Sensor calibration, control algorithm parameters | M | H | H |
| A-04 | CAN Messages | Safety-critical control messages on CAN bus | L | H | H |
| A-05 | Sensor Data | Real-time sensor readings (LiDAR, camera, IMU) | L | H | H |
| A-06 | OTA Update Channel | Communication path for firmware updates | M | H | H |
| A-07 | Diagnostic Interface | UDS/OBD-II diagnostic access | M | H | M |
| A-08 | Boot Chain | Bootloader, UEFI, kernel boot sequence | L | H | H |
| A-09 | User Credentials | Fleet management and service credentials | H | H | M |
| A-10 | Log Data | Security audit logs, diagnostic trouble codes | M | H | M |

**Rating**: H = High, M = Medium, L = Low

---

## 4. Threat Scenario Identification

| Threat ID | Target Asset | Threat Source | Attack Vector | Threat Description |
|---|---|---|---|---|
| T-01 | A-06 (OTA Channel) | Remote attacker | Network | Man-in-the-middle attack on OTA update channel to inject malicious firmware |
| T-02 | A-04 (CAN Messages) | Local attacker | CAN bus | Injection of forged CAN messages to manipulate vehicle behavior |
| T-03 | A-01 (Firmware) | Remote attacker | Physical/Network | Extraction of firmware image to reverse-engineer proprietary algorithms |
| T-04 | A-02 (Crypto Keys) | Skilled attacker | Side-channel | Side-channel attack (power/EM analysis) to extract cryptographic keys |
| T-05 | A-07 (Diagnostic) | Local attacker | Physical | Exploitation of diagnostic port (UDS) to gain unauthorized access |
| T-06 | A-05 (Sensor Data) | Remote attacker | Wireless | Spoofing of sensor inputs to corrupt perception/control algorithms |
| T-07 | A-01 (Firmware) | Supply chain | Development | Compromise of build pipeline to inject backdoor in firmware |
| T-08 | A-08 (Boot Chain) | Skilled attacker | Physical | Bypass of secure boot to load unsigned/modified firmware |
| T-09 | A-04 (CAN Messages) | Remote attacker | Network | Denial of service on CAN bus to prevent safety-critical communication |
| T-10 | A-10 (Log Data) | Insider | Local | Exfiltration of security logs and diagnostic data |
| T-11 | A-09 (Credentials) | Remote attacker | Network | Privilege escalation via compromised fleet management credentials |
| T-12 | A-01 (Firmware) | Skilled attacker | Network | Rollback attack: forcing installation of older firmware with known vulnerabilities |

---

## 5. Impact Rating

Per ISO 21434 Annex G, each threat is rated across four impact categories.

| Threat ID | Safety (S0-S3) | Financial (F0-F3) | Operational (O0-O3) | Privacy (P0-P3) | Max Impact |
|---|---|---|---|---|---|
| T-01 | S3 (life-threatening) | F2 (significant) | O3 (fleet-wide) | P1 (low) | Severe |
| T-02 | S3 (life-threatening) | F2 (significant) | O2 (multiple vehicles) | P0 (none) | Severe |
| T-03 | S0 (none) | F3 (major IP loss) | O1 (single vehicle) | P1 (low) | Major |
| T-04 | S2 (severe injury) | F2 (significant) | O2 (multiple vehicles) | P0 (none) | Major |
| T-05 | S2 (severe injury) | F1 (moderate) | O1 (single vehicle) | P2 (moderate) | Major |
| T-06 | S3 (life-threatening) | F1 (moderate) | O1 (single vehicle) | P0 (none) | Severe |
| T-07 | S3 (life-threatening) | F3 (major) | O3 (fleet-wide) | P2 (moderate) | Severe |
| T-08 | S2 (severe injury) | F2 (significant) | O1 (single vehicle) | P0 (none) | Major |
| T-09 | S3 (life-threatening) | F1 (moderate) | O1 (single vehicle) | P0 (none) | Severe |
| T-10 | S0 (none) | F1 (moderate) | O0 (none) | P3 (high) | Moderate |
| T-11 | S1 (light injury) | F2 (significant) | O2 (multiple vehicles) | P2 (moderate) | Major |
| T-12 | S2 (severe injury) | F1 (moderate) | O2 (multiple vehicles) | P0 (none) | Major |

---

## 6. Attack Feasibility Assessment

| Threat ID | Elapsed Time | Specialist Expertise | Knowledge of Item | Window of Opportunity | Equipment | Feasibility |
|---|---|---|---|---|---|---|
| T-01 | Weeks (2) | Expert (4) | Restricted (2) | Easy (0) | Specialized (3) | Medium (11) |
| T-02 | Days (1) | Proficient (2) | Public (0) | Easy (0) | Standard (1) | High (4) |
| T-03 | Days (1) | Proficient (2) | Restricted (2) | Easy (0) | Standard (1) | High (6) |
| T-04 | Months (3) | Expert (4) | Sensitive (3) | Moderate (2) | Specialized (3) | Low (15) |
| T-05 | Hours (0) | Proficient (2) | Public (0) | Easy (0) | Standard (1) | High (3) |
| T-06 | Days (1) | Expert (4) | Restricted (2) | Moderate (2) | Specialized (3) | Medium (12) |
| T-07 | Months (3) | Expert (4) | Sensitive (3) | Difficult (3) | Specialized (3) | Low (16) |
| T-08 | Weeks (2) | Expert (4) | Sensitive (3) | Moderate (2) | Specialized (3) | Low (14) |
| T-09 | Hours (0) | Layperson (0) | Public (0) | Easy (0) | Standard (1) | High (1) |
| T-10 | Days (1) | Proficient (2) | Restricted (2) | Easy (0) | Standard (1) | High (6) |
| T-11 | Days (1) | Proficient (2) | Restricted (2) | Easy (0) | Standard (1) | High (6) |
| T-12 | Days (1) | Proficient (2) | Public (0) | Easy (0) | Standard (1) | High (4) |

**Feasibility Scoring**: Sum of all factors. High: 0-9, Medium: 10-13, Low: 14-19, Very Low: 20+

---

## 7. Risk Determination

### 7.1 Risk Matrix

|  | **High Feasibility** | **Medium Feasibility** | **Low Feasibility** | **Very Low Feasibility** |
|---|---|---|---|---|
| **Severe Impact** | R5 | R4 | R3 | R2 |
| **Major Impact** | R4 | R3 | R2 | R1 |
| **Moderate Impact** | R3 | R2 | R1 | R1 |
| **Negligible Impact** | R2 | R1 | R1 | R1 |

### 7.2 Risk Assignment

| Threat ID | Impact | Feasibility | Risk Value | Risk Treatment Required |
|---|---|---|---|---|
| T-01 | Severe | Medium | **R4** | Yes - Mitigate |
| T-02 | Severe | High | **R5** | Yes - Mitigate |
| T-03 | Major | High | **R4** | Yes - Mitigate |
| T-04 | Major | Low | **R2** | Monitor |
| T-05 | Major | High | **R4** | Yes - Mitigate |
| T-06 | Severe | Medium | **R4** | Yes - Mitigate |
| T-07 | Severe | Low | **R3** | Yes - Mitigate |
| T-08 | Major | Low | **R2** | Monitor |
| T-09 | Severe | High | **R5** | Yes - Mitigate |
| T-10 | Moderate | High | **R3** | Yes - Mitigate |
| T-11 | Major | High | **R4** | Yes - Mitigate |
| T-12 | Major | High | **R4** | Yes - Mitigate |

---

## 8. Cybersecurity Goals

| Goal ID | Linked Threats | Cybersecurity Goal Description | CAL |
|---|---|---|---|
| CSG-01 | T-01, T-12 | The integrity and authenticity of firmware updates shall be verified before installation | CAL-4 |
| CSG-02 | T-02, T-09 | CAN bus communication shall be protected against injection and denial of service | CAL-4 |
| CSG-03 | T-03, T-08 | Firmware shall be protected against unauthorized extraction and modification | CAL-3 |
| CSG-04 | T-04 | Cryptographic keys shall be protected against side-channel extraction | CAL-3 |
| CSG-05 | T-05 | Diagnostic interfaces shall enforce authentication and access control | CAL-3 |
| CSG-06 | T-06 | Sensor inputs shall be validated for plausibility before processing | CAL-4 |
| CSG-07 | T-07 | The software supply chain shall be protected against compromise | CAL-3 |
| CSG-08 | T-10 | Security logs shall be protected against unauthorized access and tampering | CAL-2 |
| CSG-09 | T-11 | Fleet management access shall enforce strong authentication and least privilege | CAL-3 |

---

## 9. Risk Treatment

| Goal ID | Treatment | Control Description | Implementation Reference |
|---|---|---|---|
| CSG-01 | Mitigate | Cosign keyless signing + SLSA provenance + OSTree signature verification + anti-rollback counter | cd.yml: image-sign, security/cosign-policy.yml |
| CSG-02 | Mitigate | CAN message authentication (SecOC), DLC validation, rate limiting, Semgrep CAN rules | security/semgrep.yml: automotive-can-* rules |
| CSG-03 | Mitigate | Secure boot chain (UEFI + shim), encrypted firmware storage, read-only rootfs (OSTree) | build/aib-manifest.yml: signing config |
| CSG-04 | Mitigate | TPM-based key storage, constant-time crypto implementations, side-channel resistant algorithms | Hardware + SW controls |
| CSG-05 | Mitigate | UDS authentication (ISO 14229 Security Access), diagnostic session timeout, audit logging | Application-level controls |
| CSG-06 | Mitigate | Input range validation, sensor fusion cross-checks, plausibility monitoring | security/semgrep.yml: input validation rules |
| CSG-07 | Mitigate | SLSA Level 3 provenance, SBOM generation, Trivy scanning, cosign signing, GitHub Actions OIDC | ci.yml + cd.yml |
| CSG-08 | Mitigate | Append-only audit logs, log forwarding to central SIEM, log integrity verification | build/aib-manifest.yml: auditd config |
| CSG-09 | Mitigate | mTLS authentication, RBAC, session management, credential rotation | IEC 62443 FR1 controls |

---

## 10. Residual Risk Assessment

| Goal ID | Pre-Treatment Risk | Controls Applied | Post-Treatment Risk | Acceptable |
|---|---|---|---|---|
| CSG-01 | R4 | Cosign + SLSA + anti-rollback | R1 | Yes |
| CSG-02 | R5 | SecOC + DLC validation + rate limiting | R2 | Yes |
| CSG-03 | R4 | Secure boot + encrypted storage | R1 | Yes |
| CSG-04 | R2 | TPM + constant-time crypto | R1 | Yes |
| CSG-05 | R4 | UDS auth + timeout + audit | R2 | Yes |
| CSG-06 | R4 | Range validation + cross-checks | R2 | Yes |
| CSG-07 | R3 | SLSA + SBOM + scanning + signing | R1 | Yes |
| CSG-08 | R3 | Append-only + forwarding | R1 | Yes |
| CSG-09 | R4 | mTLS + RBAC + rotation | R2 | Yes |

All residual risks are at R2 or below, which is within the acceptable risk threshold for this item.

---

## References

- ISO/SAE 21434:2021, Road vehicles - Cybersecurity engineering
- UNECE WP.29 R155 - Cybersecurity Management System
- UNECE WP.29 R156 - Software Update Management System
- [Cybersecurity Plan](cybersecurity-plan.md)
- [Cybersecurity Case](cybersecurity-case.md)
- [Secure Coding Guidelines](secure-coding-guidelines.md)
