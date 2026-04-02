# Cybersecurity Plan

**Standard**: ISO/SAE 21434:2021, Clause 6
**Document Type**: Work Product WP-06-01

## 1. Document Control

| Field | Value |
|---|---|
| Version | 0.1 |
| Date | YYYY-MM-DD |
| Author | [Name] |
| Approver | [Name] |
| Status | Draft |

---

## 2. Project Overview

This plan defines cybersecurity activities for the **RHIVOS Edge Computing Unit**, an automotive edge device running Red Hat In-Vehicle OS and RTOS firmware for safety-relevant vehicle functions.

---

## 3. Applicable Standards

| Standard | Scope | Relationship |
|---|---|---|
| ISO/SAE 21434:2021 | Cybersecurity engineering (primary) | This plan |
| UNECE R155 | Cybersecurity Management System | Organizational-level; ISO 21434 provides technical framework |
| UNECE R156 | Software Update Management System | OTA update procedures in CD pipeline |
| ISO 26262:2018 | Functional safety | Cross-reference: [Safety Plan](../iso26262/safety-plan.md) |
| IEC 62443 | Industrial cybersecurity | Cross-reference: [Security Requirements](../iec62443/security-requirements.md) |
| ASPICE | Process maturity | Cross-reference: [Process Assessment](../aspice/process-assessment.md) |

---

## 4. Cybersecurity Activities

| ISO 21434 Clause | Activity | Pipeline Stage | Work Product | Status |
|---|---|---|---|---|
| **Clause 5**: Organizational CS Management | Establish organizational CS policies | Pre-project | CS policy document | Planned |
| **Clause 6**: Project-Dependent CS Management | Create this cybersecurity plan | Planning | This document | In Progress |
| **Clause 7**: Distributed CS Activities | Define supplier CS requirements | Planning | CS Interface Agreement | Planned |
| **Clause 8**: Continual CS Activities | Vulnerability monitoring | security-audit.yml (weekly) | Audit reports | Active |
| | Incident response readiness | Ongoing | [Incident Response Plan](../shared/incident-response-plan.md) | Draft |
| **Clause 9**: Concept Phase | TARA execution | Requirements analysis | [TARA](tara-template.md) | Draft |
| | Cybersecurity goal definition | Requirements analysis | TARA Section 8 | Draft |
| | Cybersecurity concept | Architecture design | TARA Section 9 | Draft |
| **Clause 10**: Product Development | Secure coding (MISRA/CERT) | ci.yml: build + sast | [Secure Coding Guidelines](secure-coding-guidelines.md) | Draft |
| | Static analysis (SAST) | ci.yml: sast-codeql, sast-semgrep | SARIF reports | Automated |
| | Software composition analysis | ci.yml: sbom-generate, vulnerability-scan | SBOM + Trivy reports | Automated |
| | Unit testing | ci.yml: build (test target) | Test results + coverage | Automated |
| | Integration testing | cd.yml: integration-tests | Integration test results | Automated |
| **Clause 11**: CS Validation | Cybersecurity validation | cd.yml: quality-gate | Quality gate results | Automated |
| **Clause 12**: Production | Secure build and signing | cd.yml: image-build, image-sign | Signed images + provenance | Automated |
| | Secure deployment | cd.yml: deploy-staging/production | Deployment records | Automated |
| **Clause 13**: Operations & Maintenance | Field vulnerability monitoring | security-audit.yml | Weekly audit reports | Active |
| | Patch management | CI/CD pipeline (hotfix flow) | [Vulnerability Mgmt Plan](../shared/vulnerability-management-plan.md) | Draft |
| **Clause 14**: End of CS Support | Decommissioning plan | End of life | Decommissioning document | Planned |
| **Clause 15**: TARA Methods | Risk assessment methodology | Analysis | [TARA](tara-template.md) | Draft |

---

## 5. Tailoring Rationale

| Clause | Tailoring | Rationale |
|---|---|---|
| Clause 5.4.4 | Partial | Organizational audit not applicable for greenfield reference pipeline |
| Clause 14 | Deferred | End-of-support procedures will be defined closer to product maturity |
| All others | Full compliance | No tailoring applied |

---

## 6. Roles and Responsibilities

| Activity | Safety Manager | CS Engineer | Developer | QA Engineer | Independent Assessor |
|---|---|---|---|---|---|
| Cybersecurity Plan | C | R | I | I | I |
| TARA | C | R | C | I | I |
| Secure Coding | I | A | R | C | - |
| SAST Configuration | - | R | C | A | - |
| Vulnerability Mgmt | I | R | C | A | - |
| Incident Response | C | R | C | C | - |
| Cybersecurity Case | C | R | C | C | A |
| Independent Assessment | I | C | - | - | R |

**R** = Responsible, **A** = Accountable, **C** = Consulted, **I** = Informed

---

## 7. Tool Environment

| Tool | Version | Purpose | Qualification | Pipeline Stage |
|---|---|---|---|---|
| GitHub Actions | Latest | CI/CD platform | [Tool Qualification](../iso26262/tool-qualification.md) | All |
| CodeQL | Latest | SAST for C/C++ | TCL1 - Validated | ci.yml: sast-codeql |
| Semgrep | Latest | SAST for MISRA/CERT | TCL1 - Validated | ci.yml: sast-semgrep |
| Trivy | 0.28.0+ | Vulnerability scanning | TCL2 - Evaluated | ci.yml: vulnerability-scan |
| Syft | Latest | SBOM generation | TCL3 - Confidence from use | ci.yml: sbom-generate |
| Cosign | 3.x | Binary/image signing | TCL2 - Evaluated | cd.yml: image-sign |
| GCC | 13.x | Compiler | TCL1 - Validated | ci.yml: build |
| gcov | 13.x | Coverage analysis | TCL1 - Validated | ci.yml: build (coverage) |
| TruffleHog | Latest | Secrets detection | TCL3 - Confidence from use | ci.yml: secrets-scan |
| ScanCode | Latest | License compliance | TCL3 - Confidence from use | ci.yml: license-scan |

---

## 8. Configuration Management

- **Source control**: Git (GitHub)
- **Branching strategy**: trunk-based development with release branches
- **Artifact versioning**: Git tags (semver), compliance archive branch
- **SBOM versioning**: Generated per build, archived with compliance bundle
- **Change management**: See [Change Management Plan](../shared/change-management-plan.md)
- **Retention**: Compliance bundles retained for product lifetime + 5 years

---

## 9. Cybersecurity Lifecycle Mapping

```
Concept Phase (Clause 9)     ──►  Requirements Analysis / TARA
        │
Product Development (Clause 10) ──►  CI Pipeline (ci.yml)
        │                              ├── Build + Unit Test
        │                              ├── SAST (CodeQL, Semgrep)
        │                              ├── SCA (Trivy, OSV)
        │                              ├── SBOM Generation
        │                              ├── Secrets + License Scan
        │                              └── Quality Gate
        │
CS Validation (Clause 11)    ──►  Quality Gate Decision
        │
Production (Clause 12)      ──►  CD Pipeline (cd.yml)
        │                              ├── Image Build
        │                              ├── Signing (Cosign)
        │                              ├── SLSA Provenance
        │                              ├── Staging Deploy
        │                              ├── Integration Tests
        │                              └── Production Deploy
        │
Operations (Clause 13)      ──►  Security Audit (security-audit.yml)
                                       ├── Weekly Vulnerability Re-scan
                                       ├── Dependency Review
                                       └── Incident Response
```

---

## 10. Schedule and Milestones

| Milestone | Target Date | Gate Criteria |
|---|---|---|
| TARA Complete | [Date] | All threats identified, goals defined, CAL assigned |
| Development Phase Entry | [Date] | Cybersecurity plan approved, tools qualified |
| Alpha Release | [Date] | CI pipeline operational, quality gate passing |
| Beta Release | [Date] | CD pipeline operational, staging deployment verified |
| Security Assessment | [Date] | Independent assessment complete |
| Production Release | [Date] | Cybersecurity case approved, all evidence collected |

---

## 11. Training Requirements

| Role | Training | Frequency |
|---|---|---|
| All developers | ISO 21434 awareness | Annual |
| All developers | MISRA C:2012 / CERT C secure coding | Annual |
| CS Engineers | ISO 21434 deep-dive (TARA, cybersecurity case) | Initial + updates |
| QA Engineers | Security testing techniques | Annual |
| Safety Managers | ISO 21434 + ISO 26262 interaction | Initial |

---

## References

- [TARA](tara-template.md)
- [Cybersecurity Case](cybersecurity-case.md)
- [Secure Coding Guidelines](secure-coding-guidelines.md)
- [Vulnerability Management Plan](../shared/vulnerability-management-plan.md)
- [Incident Response Plan](../shared/incident-response-plan.md)
- [Safety Plan](../iso26262/safety-plan.md)
- [Pipeline Architecture](../../docs/pipeline-architecture.md)
