# Incident Response Plan

**Standards**: ISO 21434 Clause 13, IEC 62443-4-1, UNECE R155

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Incident Classification

| Priority | Severity | Description | Response SLA | Example |
|---|---|---|---|---|
| **P1** | Critical | Safety-affecting or fleet-wide security breach | 1 hour | Remote code execution exploited in field |
| **P2** | High | Security vulnerability actively exploited (limited scope) | 4 hours | Single-vehicle compromise via diagnostic port |
| **P3** | Medium | Vulnerability identified, not yet exploited | 24 hours | New CVE in deployed component |
| **P4** | Low | Security concern, no immediate risk | 7 days | Configuration improvement opportunity |

---

## 3. Response Team

| Role | Responsibility | Contact |
|---|---|---|
| Incident Commander | Overall coordination, decisions | [TBD] |
| Cybersecurity Lead | Technical analysis, containment strategy | [TBD] |
| Safety Engineer | Assess safety impact (ISO 26262) | [TBD] |
| DevOps Lead | Pipeline execution, OTA deployment | [TBD] |
| Communications Lead | Internal/external communications | [TBD] |
| Legal/Compliance | Regulatory reporting (UNECE R155) | [TBD] |

### Escalation Chain

```
Developer/Automated Detection
    -> Cybersecurity Lead (P3/P4)
    -> Incident Commander (P2)
    -> Executive Leadership (P1)
    -> Regulatory Authorities (P1, if required by R155)
```

---

## 4. Response Phases

### Phase 1: Detection

- **Automated**: Trivy weekly scan, security-audit.yml findings
- **External**: CERT/CC advisory, Auto-ISAC notification, vendor disclosure
- **Internal**: Developer report, code review finding
- **Field**: Vehicle telemetry anomaly, customer report

### Phase 2: Containment

| Action | P1 | P2 | P3 | P4 |
|---|---|---|---|---|
| Isolate affected devices (disable OTA) | Immediate | 4 hours | N/A | N/A |
| Disable compromised interface | Immediate | 4 hours | N/A | N/A |
| Network segmentation | Immediate | 24 hours | N/A | N/A |
| Preserve forensic evidence | Immediate | Immediate | N/A | N/A |

### Phase 3: Eradication

1. Root cause analysis
2. Develop fix (branch, implement, test)
3. Run full CI pipeline (all security scans must pass)
4. Peer review by cybersecurity engineer

### Phase 4: Recovery

1. **Emergency OTA**: For P1/P2, use expedited CD pipeline (skip staging soak time)
2. **Staged rollout**: 5% -> 25% -> 100% with health monitoring
3. **Verification**: Confirm fix deployed, vulnerability no longer exploitable
4. **Fleet health check**: Monitor telemetry for 72 hours post-deployment

### Phase 5: Lessons Learned

Use the Post-Incident Review Template (Section 8) within 5 business days of resolution.

---

## 5. Emergency OTA Update Procedure

For P1/P2 incidents requiring immediate fleet-wide update:

1. Incident Commander authorizes emergency release
2. Fix developed and CI pipeline executed (quality gate must pass)
3. CD pipeline triggered with `workflow_dispatch` (environment: production)
4. **Bypass**: Staging soak time reduced to minimum (1 hour vs standard 24 hours)
5. **No bypass**: Signing, provenance, and quality gate requirements remain mandatory
6. Staged rollout accelerated: 10% (1hr) -> 50% (2hr) -> 100%
7. 72-hour monitoring period

---

## 6. Communication Plan

| Audience | Trigger | Channel | Timeline | Owner |
|---|---|---|---|---|
| Internal team | All incidents | Slack/Teams + email | Immediate | Incident Commander |
| Executive leadership | P1/P2 | Direct briefing | 1 hour | Incident Commander |
| Affected customers | P1 | Direct notification | 24 hours | Communications Lead |
| Regulatory (UNECE R155) | P1 (safety-relevant) | Official report | 72 hours | Legal/Compliance |
| Public advisory | P1 (if public exploitation) | Website + advisory | After containment | Communications Lead |

---

## 7. Evidence Preservation

- Capture system logs, audit trails, network captures before remediation
- Document chain of custody for all forensic artifacts
- Store evidence in tamper-evident archive (signed + timestamped)
- Retain for minimum 5 years (regulatory requirement)

---

## 8. Post-Incident Review Template

```
# Post-Incident Review: [Incident ID]

## Summary
- Incident date/time:
- Detection method:
- Priority:
- Duration (detection to resolution):
- Affected systems/vehicles:

## Timeline
| Time | Event |
|---|---|
| T+0 | [Detection] |
| T+X | [Containment] |
| T+X | [Fix developed] |
| T+X | [Fix deployed] |
| T+X | [Resolved] |

## Root Cause
[Description of root cause]

## Impact Assessment
- Safety impact (ISO 26262):
- Cybersecurity impact (ISO 21434):
- Number of affected devices:
- Data exposure:

## Remediation
- Fix description:
- CI pipeline run ID:
- OTA deployment record:

## Lessons Learned
- What went well:
- What could be improved:
- Action items:

## TARA Update Required?
[ ] Yes - update TARA with new threat scenario
[ ] No
```

---

## 9. Training and Exercises

| Activity | Frequency | Participants |
|---|---|---|
| Tabletop exercise (P1 scenario) | Quarterly | Full response team |
| OTA emergency drill | Semi-annually | DevOps + Cybersecurity |
| New team member onboarding | As needed | Individual |
| Plan review and update | Annually | Cybersecurity Lead |

---

## References

- [Vulnerability Management Plan](vulnerability-management-plan.md)
- [Cybersecurity Plan](../iso21434/cybersecurity-plan.md)
- [CD Pipeline](../../.github/workflows/cd.yml)
