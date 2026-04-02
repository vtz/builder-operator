# ASPICE Process Capability Assessment

**Standard**: Automotive SPICE v3.1 (based on ISO/IEC 33000)

## 1. Document Control

| Version | Date | Assessor | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Assessment Scope

| Parameter | Value |
|---|---|
| Target Capability Level | CL2 (Managed) |
| Processes in Scope | SWE.1-6, SUP.1, SUP.8-10, MAN.3 |
| Project | RHIVOS Edge Computing Unit |

---

## 3. Process Assessments

**Rating Scale**: N = Not achieved (0-15%), P = Partially (15-50%), L = Largely (50-85%), F = Fully (85-100%)

### SWE.1: Software Requirements Analysis

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Specify software requirements | Requirements YAML files in repo | L |
| BP2 | Structure software requirements | Categorized by safety goal (SSR-001 to SSR-015) | L |
| BP3 | Analyze software requirements | Cross-ref to HARA safety goals and TARA cybersecurity goals | L |
| BP4 | Analyze impact on operating environment | RHIVOS/RTOS platform analysis in aib-manifest.yml | P |
| BP5 | Develop verification criteria | Test IDs linked to each requirement (TC-*) | L |
| BP6 | Establish bidirectional traceability | [Traceability Matrix](traceability-matrix.md) - req <-> architecture | L |
| BP7 | Ensure consistency | CI traceability validation in compliance-report.yml | P |

**Overall SWE.1**: **L** (Largely achieved)

### SWE.2: Software Architectural Design

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop SW architecture | Architecture documented (ARCH-* elements) | P |
| BP2 | Identify interfaces | CAN, Ethernet, OTA, diagnostic interfaces defined | L |
| BP3 | Describe dynamic behavior | RT scheduling, state machines documented | P |
| BP4 | Evaluate resource consumption | Memory limits in systemd unit, CPU quota in aib-manifest.yml | L |
| BP5 | Evaluate alternative architectures | Mixed-criticality (ASIL/QM partition) design rationale | P |
| BP6 | Establish bidirectional traceability | Architecture -> requirements traceability in matrix | L |

**Overall SWE.2**: **P** (Partially achieved)

### SWE.3: Software Detailed Design and Unit Construction

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop detailed design | Design elements (DES-*) mapped in traceability matrix | P |
| BP2 | Define interfaces | Header files with function prototypes | L |
| BP3 | Describe dynamic behavior | State machines, sequence descriptions | P |
| BP4 | Evaluate detailed design | SAST analysis: ci.yml sast-codeql + sast-semgrep | F |
| BP5 | Establish bidirectional traceability | Design -> code traceability in matrix | L |
| BP6 | Apply coding standards | MISRA C:2012 + CERT C via [Secure Coding Guidelines](../iso21434/secure-coding-guidelines.md) | F |
| BP7 | Implement code | Source code in src/ directory | L |

**Overall SWE.3**: **L** (Largely achieved)

### SWE.4: Software Unit Verification

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop unit test strategy | Test plan with coverage targets per ASIL | L |
| BP2 | Develop unit test cases | test/ directory, TC-* test IDs | L |
| BP3 | Execute unit tests | ci.yml: build job (make test) - automated per commit | F |
| BP4 | Evaluate unit test results | ci.yml: quality-gate job - coverage thresholds | F |

**Overall SWE.4**: **F** (Fully achieved)

### SWE.5: Software Integration and Integration Test

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop integration strategy | Staging -> production promotion in cd.yml | L |
| BP2 | Develop integration test cases | Integration tests in cd.yml: integration-tests job | P |
| BP3 | Execute integration tests | cd.yml: integration-tests (automated after staging deploy) | L |
| BP4 | Evaluate integration test results | Manual review + automated reporting | L |
| BP5 | Establish bidirectional traceability | Integration tests traced in traceability matrix | L |

**Overall SWE.5**: **L** (Largely achieved)

### SWE.6: Software Qualification Test

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop qualification strategy | Production deployment gate with required approvals | L |
| BP2 | Develop qualification test cases | Qualification tests (qt/QT-*) in traceability matrix | P |
| BP3 | Execute qualification tests | cd.yml: deploy-production (pre-deployment verification) | P |
| BP4 | Evaluate qualification results | compliance-report.yml: generate compliance bundle | P |

**Overall SWE.6**: **P** (Partially achieved)

### SUP.1: Quality Assurance

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop QA strategy | Quality gate with blocking/advisory criteria | F |
| BP2 | Assure quality of processes | ASPICE assessment (this document) | L |
| BP3 | Assure quality of work products | Automated scan results + PR reviews | F |
| BP4 | Report QA findings | Quality gate PR comments, GitHub Issues | F |

**Overall SUP.1**: **F** (Fully achieved)

### SUP.8: Configuration Management

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop CM strategy | Git-based CM, branching strategy, artifact versioning | F |
| BP2 | Identify configuration items | Source code, requirements, configs, compliance docs | F |
| BP3 | Manage change history | Git history, PR reviews, CI/CD run logs | F |
| BP4 | Manage baselines | Git tags for releases, compliance archive branch | F |
| BP5 | Report CM status | Git log, GitHub releases | F |

**Overall SUP.8**: **F** (Fully achieved)

### SUP.9: Problem Resolution Management

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop problem resolution strategy | [Vulnerability Management Plan](../shared/vulnerability-management-plan.md) | L |
| BP2 | Identify and record problems | GitHub Issues, Trivy findings, security audit reports | F |
| BP3 | Analyze problems | Triage process with severity classification | L |
| BP4 | Resolve problems | Remediation workflow: Issue -> PR -> fix -> re-scan | F |
| BP5 | Track problems to closure | GitHub Issue lifecycle, SLA tracking | L |

**Overall SUP.9**: **L** (Largely achieved)

### SUP.10: Change Request Management

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Develop change management strategy | [Change Management Plan](../shared/change-management-plan.md) | L |
| BP2 | Receive and record change requests | GitHub PRs and Issues | F |
| BP3 | Analyze change requests | Impact analysis (safety, security, process) | L |
| BP4 | Implement change requests | Branch -> develop -> CI -> review -> merge | F |
| BP5 | Track change requests | PR lifecycle, CI/CD logs | F |

**Overall SUP.10**: **L** (Largely achieved)

### MAN.3: Project Management

| BP | Practice | Evidence / Pipeline Mapping | Rating |
|---|---|---|---|
| BP1 | Define project scope | Item definition in HARA and TARA | L |
| BP2 | Define project lifecycle | Safety lifecycle mapped to CI/CD stages | L |
| BP3 | Evaluate feasibility | Technical analysis documented | P |
| BP4 | Define and maintain project plan | Safety Plan + Cybersecurity Plan | L |
| BP5 | Monitor project progress | GitHub milestones, CI/CD dashboards | L |

**Overall MAN.3**: **L** (Largely achieved)

---

## 4. Gap Analysis Summary

| Process | Current Level | Target Level | Gap | Priority |
|---|---|---|---|---|
| SWE.1 | CL1 (L) | CL2 | Formalize consistency checks | Medium |
| SWE.2 | CL1 (P) | CL2 | Complete architecture documentation | High |
| SWE.3 | CL1 (L) | CL2 | Formalize detailed design docs | Medium |
| SWE.4 | CL2 (F) | CL2 | Target met | - |
| SWE.5 | CL1 (L) | CL2 | Expand integration test coverage | Medium |
| SWE.6 | CL1 (P) | CL2 | Develop formal qualification tests | High |
| SUP.1 | CL2 (F) | CL2 | Target met | - |
| SUP.8 | CL2 (F) | CL2 | Target met | - |
| SUP.9 | CL1 (L) | CL2 | Formalize SLA tracking metrics | Low |
| SUP.10 | CL1 (L) | CL2 | Formalize impact analysis templates | Medium |
| MAN.3 | CL1 (L) | CL2 | Formalize feasibility evaluation | Low |

---

## 5. Improvement Roadmap

| Phase | Timeline | Actions |
|---|---|---|
| Phase 1 | Month 1-2 | Complete architecture docs (SWE.2), formalize qualification tests (SWE.6) |
| Phase 2 | Month 3-4 | Expand integration tests (SWE.5), formalize detailed design (SWE.3) |
| Phase 3 | Month 5-6 | Achieve CL2 across all processes, prepare for CL3 assessment |

---

## References

- [Traceability Matrix](traceability-matrix.md)
- [Safety Plan](../iso26262/safety-plan.md)
- [Cybersecurity Plan](../iso21434/cybersecurity-plan.md)
- [Pipeline Architecture](../../docs/pipeline-architecture.md)
