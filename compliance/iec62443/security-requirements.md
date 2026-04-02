# IEC 62443-4-2 Component Security Requirements

**Standard**: IEC 62443-4-2:2019

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Target Security Level Justification

**Target**: SL2 - Protection against intentional violations using simple means with low resources, generic skills, and low motivation.

**Rationale**: The edge ECU operates in a vehicle network with physical access controls. Direct internet exposure is limited to the OTA channel (via telematics unit). SL2 is appropriate for the threat landscape; SL3 may be required for fleet management interfaces.

---

## 3. Foundational Requirements Mapping

**Legend**: R = Required at this SL, - = Not required, Impl = Implementation, Verif = Verification

### FR1: Identification and Authentication Control (IAC)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 1.1 | Human user identification and authentication | R | R | R | R | PAM + local user database, no default users (aib-manifest.yml) | TC-IAC-01 | Implemented |
| CR 1.2 | Software process identification and authentication | - | R | R | R | systemd service identity, SELinux contexts | TC-IAC-02 | Implemented |
| CR 1.3 | Hardware device identification | - | - | R | R | TPM-based device identity | TC-IAC-03 | Planned |
| CR 1.5 | Authenticator management | R | R | R | R | Credential rotation via fleet management | TC-IAC-05 | Planned |
| CR 1.7 | Strength of password-based auth | R | R | R | R | PAM password policy (min 12 chars, complexity) | TC-IAC-07 | Implemented |
| CR 1.8 | PKI certificates | - | R | R | R | mTLS for fleet management, OSTree GPG | TC-IAC-08 | Implemented |
| CR 1.9 | Strength of public key auth | - | R | R | R | RSA-4096 or Ed25519 keys | TC-IAC-09 | Implemented |
| CR 1.11 | Unsuccessful login attempts | R | R | R | R | pam_faillock: 5 attempts, 15 min lockout | TC-IAC-11 | Implemented |
| CR 1.14 | Strength of symmetric key auth | - | - | R | R | AES-256 for symmetric operations | TC-IAC-14 | Planned |

### FR2: Use Control (UC)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 2.1 | Authorization enforcement | R | R | R | R | SELinux mandatory access control, systemd sandboxing | TC-UC-01 | Implemented |
| CR 2.5 | Session lock | - | R | R | R | Auto-logout after 15 min inactivity | TC-UC-05 | Planned |
| CR 2.6 | Remote session termination | - | R | R | R | Fleet management can terminate sessions | TC-UC-06 | Planned |
| CR 2.8 | Auditable events | R | R | R | R | auditd logging all security events | TC-UC-08 | Implemented |
| CR 2.9 | Audit storage capacity | R | R | R | R | systemd-journal with size limits + forwarding | TC-UC-09 | Implemented |
| CR 2.11 | Timestamps | R | R | R | R | chronyd NTP synchronization | TC-UC-11 | Implemented |
| CR 2.12 | Non-repudiation | - | - | R | R | Signed audit logs | TC-UC-12 | Planned |

### FR3: System Integrity (SI)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 3.1 | Communication integrity | R | R | R | R | TLS 1.3 for all external communication | TC-SI-01 | Implemented |
| CR 3.4 | Software and information integrity | - | R | R | R | Secure boot + cosign + OSTree GPG signing | TC-SI-04 | Implemented |
| CR 3.5 | Input validation | R | R | R | R | All external inputs range-checked (Semgrep enforced) | TC-SI-05 | Implemented |
| CR 3.7 | Error handling | R | R | R | R | Safe state transitions on error, no info disclosure | TC-SI-07 | Implemented |
| CR 3.9 | Protection of audit information | - | R | R | R | Append-only logs, forwarding to SIEM | TC-SI-09 | Implemented |

### FR4: Data Confidentiality (DC)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 4.1 | Information confidentiality | - | R | R | R | TLS for data in transit, dm-crypt for data at rest | TC-DC-01 | Partial |
| CR 4.3 | Use of cryptography | - | R | R | R | AES-256, SHA-256+, no weak algorithms (Semgrep enforced) | TC-DC-03 | Implemented |

### FR5: Restricted Data Flow (RDF)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 5.1 | Network segmentation | R | R | R | R | Firewall: drop all, no open ports (aib-manifest.yml) | TC-RDF-01 | Implemented |
| CR 5.2 | Zone boundary protection | - | R | R | R | iptables/nftables rules, VPN for management | TC-RDF-02 | Planned |
| CR 5.4 | Application partitioning | - | R | R | R | Podman containers, systemd-nspawn, MPU (RTOS) | TC-RDF-04 | Implemented |

### FR6: Timely Response to Events (TRE)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 6.1 | Audit log accessibility | R | R | R | R | systemd-journal-remote for central access | TC-TRE-01 | Implemented |
| CR 6.2 | Continuous monitoring | - | R | R | R | Health endpoint + metrics + security-audit.yml | TC-TRE-02 | Implemented |

### FR7: Resource Availability (RA)

| CR ID | Requirement | SL1 | SL2 | SL3 | SL4 | Implementation | Verif Test | Status |
|---|---|---|---|---|---|---|---|---|
| CR 7.1 | DoS protection | R | R | R | R | Rate limiting, resource quotas (MemoryMax, CPUQuota) | TC-RA-01 | Implemented |
| CR 7.2 | Resource management | R | R | R | R | systemd resource controls, cgroups | TC-RA-02 | Implemented |
| CR 7.3 | Control system backup | - | R | R | R | OSTree A/B partitions for rollback | TC-RA-03 | Implemented |
| CR 7.4 | Control system recovery | R | R | R | R | Automatic rollback on health check failure | TC-RA-04 | Implemented |
| CR 7.7 | Least functionality | R | R | R | R | Minimal image, sshd disabled, no default users | TC-RA-07 | Implemented |
| CR 7.8 | Component inventory | - | R | R | R | SBOM (CycloneDX + SPDX) generated per build | TC-RA-08 | Implemented |

---

## 4. Gap Analysis

| CR | Gap Description | Remediation | Priority |
|---|---|---|---|
| CR 1.3 | Hardware device ID not yet implemented | Integrate TPM2 device attestation | Medium |
| CR 4.1 | Data-at-rest encryption partial | Implement dm-crypt for /var/lib/edge-app | High |
| CR 5.2 | Zone boundary protection planned | Configure nftables rules | Medium |
| CR 2.5/2.6 | Session management not yet implemented | Implement session timeout and remote kill | Low |

---

## 5. IEC 62443-4-1 Process Mapping

| IEC 62443-4-1 Practice | Pipeline Stage | Evidence |
|---|---|---|
| SM-1: Security management | All | Cybersecurity Plan |
| SR-2: Secure coding standards | ci.yml: sast-* | Semgrep + CodeQL reports |
| SR-5: Security requirements | Pre-CI | This document |
| SVV-1: Security verification testing | ci.yml: quality-gate | Quality gate results |
| SVV-3: Vulnerability testing | ci.yml: vulnerability-scan | Trivy + OSV reports |
| DM-1: Defect management | GitHub Issues | Issue tracker |
| SUM-4: Security update delivery | cd.yml | OTA update pipeline |

---

## References

- [TARA](../iso21434/tara-template.md)
- [Vulnerability Management Plan](../shared/vulnerability-management-plan.md)
- [Pipeline Architecture](../../docs/pipeline-architecture.md)
