# Safety Plan

**Standard**: ISO 26262:2018, Part 2
**Document Type**: Work Product

## 1. Document Control

| Field | Value |
|---|---|
| Version | 0.1 |
| Date | YYYY-MM-DD |
| Author | [Name] |
| Safety Manager | [Name] |
| Status | Draft |

---

## 2. Scope and Item Definition

The RHIVOS Edge Computing Unit is an automotive ECU performing real-time sensor processing, actuator control, and edge AI inference. Maximum target ASIL: **ASIL-B** (with ASIL-D elements via decomposition).

---

## 3. Applicable Standards

| Standard | Relevance |
|---|---|
| ISO 26262:2018 Parts 1-12 | Primary functional safety standard |
| ISO/SAE 21434:2021 | Cybersecurity engineering - [Cybersecurity Plan](../iso21434/cybersecurity-plan.md) |
| ASPICE v3.1 | Process maturity - [Process Assessment](../aspice/process-assessment.md) |

---

## 4. Safety Lifecycle Mapping to CI/CD

| ISO 26262 Phase | Part | CI/CD Pipeline Stage | Artifacts |
|---|---|---|---|
| Concept Phase | Part 3 | Requirements analysis | HARA, Safety Goals |
| System Development | Part 4 | Architecture design | System safety requirements |
| Software Development | Part 6 | ci.yml: build, test, SAST | Source code, test results, SAST reports |
| SW Unit Verification | Part 6 | ci.yml: build (test + coverage) | Unit test results, coverage reports |
| SW Integration | Part 6 | cd.yml: integration-tests | Integration test results |
| Production | Part 7 | cd.yml: image-build, sign, deploy | Signed images, deployment records |
| Supporting Processes | Part 8 | All workflows | Tool qualification, CM evidence |

---

## 5. Safety Activities

| Activity | ISO 26262 Ref | Role | Pipeline Stage | Work Product |
|---|---|---|---|---|
| HARA execution | Part 3 | Safety Engineer | Pre-CI | [HARA](hara-template.md) |
| Safety goal definition | Part 3, Cl. 8 | Safety Engineer | Pre-CI | HARA Section 6 |
| SW safety requirements | Part 6, Cl. 6 | Safety Engineer | Pre-CI | [Safety Requirements](safety-requirements.md) |
| Secure coding (MISRA) | Part 6, Table 1 | Developer | ci.yml: build | Source code |
| Static analysis | Part 6, Table 7 | Automated | ci.yml: sast-* | SARIF reports |
| Unit test execution | Part 6, Table 10 | Automated | ci.yml: build | Test results |
| Coverage measurement | Part 6, Table 12 | Automated | ci.yml: build | Coverage report |
| Integration testing | Part 6, Cl. 10 | Automated | cd.yml: integration-tests | Test results |
| Tool qualification | Part 8, Cl. 11 | Safety Engineer | Pre-CI | [Tool Qualification](tool-qualification.md) |
| Safety case compilation | Part 2, Cl. 6 | Safety Manager | compliance-report.yml | [Safety Case](safety-case.md) |

---

## 6. Roles and Responsibilities (RACI)

| Activity | Safety Manager | Safety Engineer | Developer | QA | Assessor |
|---|---|---|---|---|---|
| Safety Plan | A | R | I | I | I |
| HARA | A | R | C | I | I |
| Safety Requirements | C | R | C | C | I |
| Implementation | I | C | R | I | - |
| Unit Testing | I | C | R | A | - |
| Integration Testing | I | C | C | R | - |
| Safety Case | A | R | C | C | R |
| Tool Qualification | A | R | C | C | - |
| Confirmation Review | I | C | - | - | R |

---

## 7. Tool Environment

All tools are qualified per ISO 26262 Part 8. See [Tool Qualification](tool-qualification.md).

---

## 8. Configuration Management

- All safety work products under git version control
- Release baselines created via git tags
- Compliance archive branch retains generated evidence
- Change impact analysis required for safety-relevant changes: [Change Management Plan](../shared/change-management-plan.md)

---

## 9. Verification Methods per ASIL

| Method | QM | ASIL-A | ASIL-B | ASIL-C | ASIL-D |
|---|---|---|---|---|---|
| Code review | R | R | HR | HR | HR |
| Code walkthrough | - | R | R | HR | HR |
| Formal inspection | - | - | R | R | HR |
| Static analysis (SAST) | R | HR | HR | HR | HR |
| Statement coverage | R | HR | HR | HR | HR |
| Branch coverage | - | R | HR | HR | HR |
| MC/DC coverage | - | - | R | HR | HR |
| Formal verification | - | - | - | R | HR |

**R** = Recommended, **HR** = Highly Recommended

---

## 10. Schedule and Milestones

| Milestone | Criteria | Pipeline Gate |
|---|---|---|
| HARA approved | All hazards identified, ASIL assigned | Pre-development |
| Tools qualified | All CI/CD tools assessed per Part 8 | Pre-development |
| Development entry | Safety plan + requirements approved | CI quality gate operational |
| Alpha release | Unit test coverage meets ASIL targets | Quality gate passing |
| Beta release | Integration tests passing in staging | CD pipeline operational |
| Safety assessment | Independent assessor review complete | Assessment report |
| Production release | Safety case approved | Production deploy gate |

---

## 11. Training

| Role | Training | Frequency |
|---|---|---|
| All developers | ISO 26262 awareness, MISRA C:2012 | Annual |
| Safety Engineers | ISO 26262 Parts 2-8 deep-dive | Initial + updates |
| QA Engineers | Safety testing, coverage measurement | Annual |

---

## References

- [HARA](hara-template.md)
- [Safety Case](safety-case.md)
- [Safety Requirements](safety-requirements.md)
- [Tool Qualification](tool-qualification.md)
- [Cybersecurity Plan](../iso21434/cybersecurity-plan.md)
- [Process Assessment](../aspice/process-assessment.md)
