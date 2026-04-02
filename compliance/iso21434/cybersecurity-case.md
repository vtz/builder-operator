# Cybersecurity Case

**Standard**: ISO/SAE 21434:2021, Clause 9.5
**Document Type**: Work Product WP-09-05

## 1. Document Control

| Field | Value |
|---|---|
| Version | 0.1 |
| Date | YYYY-MM-DD |
| Author | [Name] |
| Reviewer | [Name] |
| Approver | [Name] |
| Status | Draft |

---

## 2. Cybersecurity Claim

**Claim**: The RHIVOS Edge Computing Unit, as specified in the item definition, achieves an acceptable level of cybersecurity throughout its lifecycle. All identified cybersecurity risks have been reduced to acceptable levels through the implementation of appropriate cybersecurity controls, as demonstrated by the evidence referenced in this document.

---

## 3. Work Products Index

| WP ID | ISO 21434 Clause | Work Product | Document/Artifact Reference | Version | Status |
|---|---|---|---|---|---|
| WP-06-01 | Clause 6 | Cybersecurity Plan | [cybersecurity-plan.md](cybersecurity-plan.md) | 0.1 | Draft |
| WP-09-01 | Clause 9 | Item Definition | [tara-template.md](tara-template.md) Section 2 | 0.1 | Draft |
| WP-09-02 | Clause 9 | Asset Identification | [tara-template.md](tara-template.md) Section 3 | 0.1 | Draft |
| WP-09-03 | Clause 9 | Threat Scenarios | [tara-template.md](tara-template.md) Section 4 | 0.1 | Draft |
| WP-09-04 | Clause 9 | Risk Assessment | [tara-template.md](tara-template.md) Sections 5-7 | 0.1 | Draft |
| WP-09-05 | Clause 9 | Cybersecurity Case | This document | 0.1 | Draft |
| WP-10-01 | Clause 10 | Cybersecurity Requirements | Derived from TARA CSG-01 to CSG-09 | 0.1 | Draft |
| WP-10-02 | Clause 10 | Secure Coding Guidelines | [secure-coding-guidelines.md](secure-coding-guidelines.md) | 0.1 | Draft |
| WP-10-03 | Clause 10 | SAST Results | CI Pipeline: CodeQL SARIF, Semgrep SARIF | Per build | Auto |
| WP-10-04 | Clause 10 | SCA Results | CI Pipeline: Trivy JSON, OSV JSON | Per build | Auto |
| WP-10-05 | Clause 10 | Vulnerability Analysis | CI Pipeline: trivy-fs-report.json | Per build | Auto |
| WP-10-06 | Clause 10 | SBOM | CI Pipeline: sbom.cyclonedx.json, sbom.spdx.json | Per build | Auto |
| WP-10-07 | Clause 10 | Unit Test Results | CI Pipeline: test-results-{platform} | Per build | Auto |
| WP-10-08 | Clause 10 | Code Coverage Report | CI Pipeline: coverage/ | Per build | Auto |
| WP-11-01 | Clause 11 | Integration Test Results | CD Pipeline: integration-test-results | Per deploy | Auto |
| WP-12-01 | Clause 12 | Signing Evidence | CD Pipeline: cosign signatures, SLSA provenance | Per release | Auto |
| WP-13-01 | Clause 13 | Vulnerability Management Plan | [../shared/vulnerability-management-plan.md](../shared/vulnerability-management-plan.md) | 0.1 | Draft |
| WP-13-02 | Clause 13 | Incident Response Plan | [../shared/incident-response-plan.md](../shared/incident-response-plan.md) | 0.1 | Draft |
| WP-08-01 | Clause 8 | Security Audit Reports | Security Audit Workflow: security-audit-report | Weekly | Auto |

---

## 4. Evidence Chain

For each cybersecurity goal, the following chain demonstrates that the goal is met through requirements, controls, verification, and evidence.

### CSG-01: Firmware Update Integrity and Authenticity

| Layer | Item | Reference |
|---|---|---|
| **Goal** | CSG-01: Firmware updates shall be verified for integrity and authenticity | TARA Section 8 |
| **Requirement** | CSR-01: All firmware images must be cryptographically signed using Sigstore keyless signing | Derived from CSG-01 |
| **Requirement** | CSR-02: SLSA Level 3 provenance must be generated for all builds | Derived from CSG-01 |
| **Requirement** | CSR-03: Anti-rollback mechanism must prevent installation of older firmware | Derived from CSG-01 |
| **Control** | Cosign keyless signing via Sigstore/Fulcio in CD pipeline | cd.yml: image-sign job |
| **Control** | SLSA provenance generation via GitHub attestations | cd.yml: provenance job |
| **Control** | OSTree commit GPG signing | cd.yml: deploy-staging job |
| **Verification** | Signature verification before production deployment | cd.yml: deploy-production job |
| **Verification** | Device-side OSTree GPG verification on update | deployment-guide.md Section 6 |
| **Evidence** | Cosign signature in OCI registry | Automated per release |
| **Evidence** | SLSA provenance attestation | Automated per release |
| **Evidence** | Integration test: secure boot verification | CD pipeline artifact |

### CSG-02: CAN Bus Communication Protection

| Layer | Item | Reference |
|---|---|---|
| **Goal** | CSG-02: CAN bus communication shall be protected against injection and DoS | TARA Section 8 |
| **Requirement** | CSR-04: All CAN message handlers must validate CAN ID before processing | Derived from CSG-02 |
| **Requirement** | CSR-05: CAN DLC must be checked before accessing data bytes | Derived from CSG-02 |
| **Control** | Semgrep rules: automotive-can-no-validation, automotive-can-dlc-unchecked | security/semgrep.yml |
| **Control** | CAN rate limiting in application layer | Application code |
| **Verification** | SAST analysis (Semgrep) in CI pipeline | ci.yml: sast-semgrep job |
| **Verification** | Unit tests for CAN message handlers | ci.yml: build job (test target) |
| **Evidence** | Semgrep SARIF report (zero CAN rule violations) | CI pipeline artifact |
| **Evidence** | Unit test results for CAN handling | CI pipeline artifact |

### CSG-07: Supply Chain Protection

| Layer | Item | Reference |
|---|---|---|
| **Goal** | CSG-07: Software supply chain shall be protected against compromise | TARA Section 8 |
| **Requirement** | CSR-12: SBOM shall be generated in CycloneDX and SPDX formats | Derived from CSG-07 |
| **Requirement** | CSR-13: All dependencies shall be scanned for known vulnerabilities | Derived from CSG-07 |
| **Requirement** | CSR-14: Build provenance shall be attested at SLSA Level 3 | Derived from CSG-07 |
| **Control** | Syft SBOM generation | ci.yml: sbom-generate job |
| **Control** | Trivy vulnerability scanning | ci.yml: vulnerability-scan job |
| **Control** | OSV-Scanner dependency audit | ci.yml: osv-scan job |
| **Control** | SLSA provenance generation | cd.yml: provenance job |
| **Verification** | Quality gate: SBOM generation required | ci.yml: quality-gate job |
| **Verification** | Quality gate: zero CRITICAL/HIGH vulnerabilities | ci.yml: quality-gate job |
| **Evidence** | sbom.cyclonedx.json, sbom.spdx.json | CI pipeline artifact |
| **Evidence** | trivy-fs-report.json | CI pipeline artifact |
| **Evidence** | SLSA provenance attestation | CD pipeline artifact |

*(Evidence chains for CSG-03 through CSG-09 follow the same pattern and should be completed during the cybersecurity assessment.)*

---

## 5. Argument Structure

The following Goal Structuring Notation (GSN) outlines the cybersecurity argument:

```
[G1] The RHIVOS Edge ECU is acceptably secure
  |
  ├── [S1] Strategy: Argue over each cybersecurity goal from TARA
  |     |
  |     ├── [G1.1] CSG-01 (OTA integrity) is met
  |     |     ├── [Sn1] Evidence: Cosign signature verification logs
  |     |     ├── [Sn2] Evidence: SLSA provenance attestation
  |     |     └── [Sn3] Evidence: Integration test - secure boot verification
  |     |
  |     ├── [G1.2] CSG-02 (CAN protection) is met
  |     |     ├── [Sn4] Evidence: Semgrep SARIF (zero CAN violations)
  |     |     └── [Sn5] Evidence: Unit test results for CAN handlers
  |     |
  |     ├── [G1.3] CSG-03 through CSG-09 are met
  |     |     └── [Sn*] Evidence: Corresponding CI/CD pipeline artifacts
  |     |
  |     └── [G1.4] All residual risks are acceptable (R2 or below)
  |           └── [Sn*] Evidence: TARA Section 10 - residual risk assessment
  |
  ├── [S2] Strategy: Argue that development process is secure
  |     ├── [G2.1] Secure coding guidelines are followed
  |     |     ├── [Sn*] Evidence: Semgrep + CodeQL SAST results
  |     |     └── [Sn*] Evidence: Code review records (PR approvals)
  |     |
  |     └── [G2.2] Supply chain is monitored
  |           ├── [Sn*] Evidence: SBOM (CycloneDX + SPDX)
  |           └── [Sn*] Evidence: Weekly security audit reports
  |
  └── [S3] Strategy: Argue that post-deployment monitoring is active
        ├── [G3.1] Vulnerability management is operational
        |     └── [Sn*] Evidence: Vulnerability management plan + audit trail
        └── [G3.2] Incident response is prepared
              └── [Sn*] Evidence: Incident response plan + exercise records
```

---

## 6. Tool Qualification Summary

| Tool | Purpose | Qualification Basis | Status |
|---|---|---|---|
| CodeQL | SAST for C/C++ | Industry-standard tool, extensive validation suite | Qualified |
| Semgrep | SAST for MISRA/CERT rules | Custom rule validation against known-good/bad code | Qualified |
| Trivy | Vulnerability scanning | Cross-validated against NVD, OSV databases | Qualified |
| Cosign | Binary/image signing | Sigstore ecosystem, Fulcio CA, Rekor transparency | Qualified |
| Syft | SBOM generation | Cross-validated CycloneDX/SPDX output | Qualified |
| gcov | Code coverage | GCC-integrated, cross-validated with manual analysis | Qualified |

See also: [ISO 26262 Tool Qualification](../iso26262/tool-qualification.md)

---

## 7. Deviations and Rationale

| Deviation ID | ISO 21434 Requirement | Deviation Description | Rationale | Risk Assessment |
|---|---|---|---|---|
| DEV-001 | [Example] | [Description of deviation] | [Why it is acceptable] | [Residual risk] |

*No deviations identified at this time. Update as needed during development.*

---

## 8. Independent Assessment

*This section will be populated following independent cybersecurity assessment by a qualified third party.*

| Assessment | Assessor | Date | Findings | Status |
|---|---|---|---|---|
| [Planned] | [TBD] | [TBD] | [TBD] | Pending |

---

## 9. Conclusion

Based on the evidence collected and referenced in this cybersecurity case:

1. All identified threats have been assessed through the TARA process
2. Cybersecurity goals have been defined for all risks requiring treatment
3. Controls have been implemented and integrated into the CI/CD pipeline
4. Verification evidence is automatically generated and archived
5. Post-deployment monitoring is planned through vulnerability management and incident response

The cybersecurity case provides sufficient evidence that the RHIVOS Edge Computing Unit achieves an acceptable level of cybersecurity for its intended operational context.

---

## References

- [TARA](tara-template.md)
- [Cybersecurity Plan](cybersecurity-plan.md)
- [Secure Coding Guidelines](secure-coding-guidelines.md)
- [Vulnerability Management Plan](../shared/vulnerability-management-plan.md)
- [Incident Response Plan](../shared/incident-response-plan.md)
- [IEC 62443 Security Requirements](../iec62443/security-requirements.md)
- [ISO 26262 Safety Case](../iso26262/safety-case.md)
