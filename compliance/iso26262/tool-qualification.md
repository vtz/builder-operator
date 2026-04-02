# Tool Qualification Report

**Standard**: ISO 26262:2018, Part 8, Clause 11

## 1. Document Control

| Version | Date | Author | Status |
|---|---|---|---|
| 0.1 | YYYY-MM-DD | [Name] | Draft |

---

## 2. Tool Classification Method

### TCL Determination Matrix

**Tool Impact (TI)**:
- **TI1**: Tool can introduce or fail to detect errors in a safety-related item
- **TI2**: Tool cannot introduce or fail to detect errors

**Tool Error Detection (TD)**:
- **TD1**: High degree of confidence that errors are detected/prevented
- **TD2**: Medium degree of confidence
- **TD3**: Low degree of confidence

| | TD1 | TD2 | TD3 |
|---|---|---|---|
| **TI1** | TCL1 | TCL2 | TCL3 |
| **TI2** | TCL1 | TCL1 | TCL1 |

---

## 3. Tool Inventory

| Tool | Version | Purpose in Pipeline | TI | TD | TCL | Qualification Method | Status |
|---|---|---|---|---|---|---|---|
| GCC | 13.x | C/C++ compiler (code generation) | TI1 | TD1 | **TCL1** | Validation (compiler test suite) | Qualified |
| Clang/LLVM | 17.x | Alternative compiler, fuzzing | TI1 | TD1 | **TCL1** | Validation (LLVM test suite) | Qualified |
| CMake | 3.28+ | Build system generator | TI1 | TD2 | **TCL2** | Evaluation of dev process | Qualified |
| GNU Make | 4.4+ | Build automation | TI1 | TD2 | **TCL2** | Evaluation of dev process | Qualified |
| CodeQL | Latest | SAST for C/C++ (safety verification) | TI1 | TD1 | **TCL1** | Validation (known-good/bad test suite) | Qualified |
| Semgrep | Latest | SAST for MISRA/CERT rules | TI1 | TD1 | **TCL1** | Validation (known-good/bad test suite) | Qualified |
| Trivy | 0.28.0+ | Vulnerability scanning | TI1 | TD2 | **TCL2** | Evaluation of tool development | Qualified |
| gcov | 13.x | Code coverage measurement | TI1 | TD1 | **TCL1** | Validation (coverage reference tests) | Qualified |
| llvm-cov | 17.x | MC/DC coverage measurement | TI1 | TD1 | **TCL1** | Validation (MC/DC reference tests) | Planned |
| Cosign | 3.x | Binary/image signing | TI1 | TD2 | **TCL2** | Evaluation of tool development | Qualified |
| Syft | Latest | SBOM generation | TI2 | TD1 | **TCL1** | No qualification needed (TI2) | N/A |
| GitHub Actions | Latest | CI/CD orchestration platform | TI1 | TD2 | **TCL2** | Evaluation of platform (SOC2) | Qualified |
| OSBuild | Latest | Image builder for RHIVOS | TI1 | TD2 | **TCL2** | Evaluation of dev process | Qualified |
| libFuzzer | LLVM 17.x | Fuzz testing engine | TI2 | TD1 | **TCL1** | No qualification needed (TI2) | N/A |
| ScanCode | Latest | License compliance scanning | TI2 | TD1 | **TCL1** | No qualification needed (TI2) | N/A |
| TruffleHog | Latest | Secrets detection | TI2 | TD1 | **TCL1** | No qualification needed (TI2) | N/A |

---

## 4. Qualification Methods per TCL

### TCL1: Increased Confidence from Use + Validation

Required for tools that directly impact safety evidence (compilers, coverage tools, SAST).

- **Validation suite**: Known-good and known-bad code samples that exercise the tool
- **Usage history**: Document production use across projects
- **Known bugs**: Track and document known tool limitations/bugs
- **Version pinning**: Exact version locked in CI pipeline
- **Regression testing**: Re-validate on tool version updates

### TCL2: Evaluation of Tool Development Process

Required for tools with medium confidence in error detection.

- **Development process review**: Verify vendor follows quality processes (ISO 9001, SOC2)
- **Release notes analysis**: Review for known issues affecting safety use
- **Operational constraints**: Document any usage restrictions for safety context
- **Version pinning**: Lock to validated version

### TCL3: Increased Confidence from Use

Minimal evidence for low-risk tools (automatically TCL1 for TI2 tools).

- **Usage documentation**: Record of successful use in project
- **No specific validation required**

---

## 5. Qualification Evidence Template

For each TCL1/TCL2 tool, the following evidence is maintained:

```
Tool: [Name]
Version: [X.Y.Z]
TCL: [1/2/3]

Validation Test Suite:
  - Total test cases: [N]
  - Passed: [N]
  - Failed: [N]
  - Known limitations: [List]

Usage Constraints:
  - [Any restrictions on how the tool may be used in safety context]

Known Bugs Affecting Safety:
  - [Bug ID]: [Description] - [Workaround/Mitigation]

Version Pinning:
  - Pinned in: [ci.yml line reference]
  - Last validated: [Date]

Qualification Status: [Qualified / Conditionally Qualified / Not Qualified]
Qualified By: [Name]
Date: [YYYY-MM-DD]
```

---

## 6. Tool Validation Plan

| Tool | Validation Approach | Test Suite Location | Cadence |
|---|---|---|---|
| GCC | Compile known-good/bad C code, verify output correctness | test/tool-validation/gcc/ | Each major version update |
| CodeQL | Run against code with known CWEs, verify detection | test/tool-validation/codeql/ | Each version update |
| Semgrep | Run custom rules against known violations and clean code | test/tool-validation/semgrep/ | Each rule change |
| gcov | Measure coverage on code with known coverage characteristics | test/tool-validation/gcov/ | Each compiler update |

---

## 7. Tool Change Management

**Re-qualification triggers**:
- Major version update of any TCL1/TCL2 tool
- Change in tool configuration (e.g., new Semgrep rules)
- Compiler flag changes affecting code generation
- Discovery of tool bug affecting safety-relevant output

**Process**:
1. Log change request per [Change Management Plan](../shared/change-management-plan.md)
2. Perform impact analysis on safety evidence
3. Re-run validation suite for affected tool
4. Update this document with new version and qualification status
5. Approval by Safety Manager

---

## References

- [Safety Plan](safety-plan.md)
- [Safety Case](safety-case.md)
- [Change Management Plan](../shared/change-management-plan.md)
