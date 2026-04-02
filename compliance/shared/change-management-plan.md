# Change Management Plan

**Standards**: ISO 26262 Part 8, ASPICE SUP.10, ISO 21434 Clause 7

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Change Request Workflow

```
[1. Submit] ── PR or GitHub Issue
      |
[2. Classify] ── Safety / Security / Process / General
      |
[3. Impact Analysis] ── Which ASIL/CAL/SL affected? Which work products?
      |
[4. Review & Approve] ── Required reviewers based on classification
      |
[5. Implement] ── Branch, develop, test
      |
[6. Verify] ── CI pipeline (quality gate must pass)
      |
[7. Close] ── Merge, update traceability, archive evidence
```

---

## 3. Impact Analysis Criteria

| Change Type | Required Assessment | Approver |
|---|---|---|
| **Safety-relevant** | ISO 26262 impact analysis: ASIL re-evaluation, affected safety goals, HARA update needed? | Safety Manager |
| **Security-relevant** | ISO 21434 impact analysis: TARA update, affected cybersecurity goals, CAL re-evaluation | Cybersecurity Engineer |
| **Process change** | ASPICE process re-assessment impact, affected base practices | Quality Manager |
| **Tool change** | ISO 26262 Part 8 re-qualification trigger assessment | Safety Engineer |
| **General** | Standard code review | Peer developer |

---

## 4. Approval Matrix

| Classification | Min. Reviewers | Required Approvers |
|---|---|---|
| Safety-critical (ASIL C/D) | 2 | Safety Manager + Safety Engineer |
| Safety-relevant (ASIL A/B) | 2 | Safety Engineer + Peer |
| Security-critical (CAL 3/4) | 2 | Cybersecurity Engineer + Peer |
| Security-relevant (CAL 1/2) | 1 | Cybersecurity Engineer |
| Process change | 1 | Quality Manager |
| Tool version update | 1 | Safety Engineer |
| General | 1 | Peer developer |

---

## 5. Configuration Items Under Control

| Category | Items | Location |
|---|---|---|
| Source code | All C/C++ source, headers, scripts | src/, include/ |
| Requirements | Safety/security requirements | compliance/ |
| Architecture | Architecture documents | docs/ |
| Test cases | Unit, integration, qualification tests | test/, it/, qt/ |
| Compliance docs | All TARA, HARA, plans, cases | compliance/ |
| Tool configs | Semgrep, CodeQL, Trivy, cosign | security/ |
| Pipeline definitions | CI/CD workflow YAML | .github/workflows/ |
| Build configs | Makefile, CMake toolchains, Containerfile | build/ |

---

## 6. Baseline Management

| Baseline Type | Trigger | Contents | Retention |
|---|---|---|---|
| Release baseline | Git tag (semver) | All config items at release point | Product lifetime + 5 years |
| Compliance archive | Each release | Generated compliance bundle (signed) | Product lifetime + 5 years |
| Safety baseline | Safety assessment milestone | Safety case + all evidence | Product lifetime + 10 years |

---

## 7. Audit Trail

All changes produce the following evidence automatically:
- **Git history**: Commit messages, diffs, author identity
- **PR reviews**: Review comments, approvals, change requests
- **CI/CD logs**: Build results, scan reports, quality gate decisions
- **Deployment records**: deployment-record.json in CD pipeline

---

## References

- [Safety Plan](../iso26262/safety-plan.md)
- [Cybersecurity Plan](../iso21434/cybersecurity-plan.md)
- [Process Assessment](../aspice/process-assessment.md)
