# Secure Coding Guidelines

**Standard**: ISO/SAE 21434:2021, Clause 10 [RQ-10-05]
**Document Type**: Work Product WP-10-02

## 1. Document Control

| Field | Value |
|---|---|
| Version | 0.1 |
| Date | YYYY-MM-DD |
| Status | Draft |

---

## 2. Scope

These guidelines apply to all C and C++ code developed for the RHIVOS Edge Computing Unit and RTOS firmware targets. They cover safety-critical and security-critical code paths.

---

## 3. Applicable Standards

| Standard | Applicability |
|---|---|
| MISRA C:2012 (Amendment 2) | All C code |
| CERT C (SEI CERT C Coding Standard) | All C code |
| AUTOSAR C++14 Guidelines | All C++ code |
| ISO 26262:2018 Part 6, Table 1 | Safety-critical code |
| IEC 62443-4-1, SR-2 | Security-critical code |

---

## 4. Mandatory Rules

Violations of mandatory rules **block the CI pipeline** (quality gate failure).

| # | Rule ID | Standard | Description | Enforcement | CWE |
|---|---|---|---|---|---|
| 1 | Rule 21.17 / STR31-C | MISRA/CERT | Do not use unsafe string functions (strcpy, strcat, sprintf, gets) | Semgrep: automotive-unsafe-string-functions | CWE-120 |
| 2 | Rule 1.3 / EXP34-C | MISRA/CERT | Do not dereference null pointers | Semgrep: automotive-null-pointer-deref, CodeQL: cpp/null-dereference | CWE-476 |
| 3 | FIO30-C | CERT | Do not use non-literal format strings | Semgrep: automotive-format-string | CWE-134 |
| 4 | ERR33-C / Rule 17.7 | CERT/MISRA | Check return values of critical functions (malloc, fopen, read, write) | Semgrep: automotive-unchecked-return | CWE-252 |
| 5 | MEM35-C | CERT | Validate memory copy sizes against destination buffer | Semgrep: automotive-unsafe-memory-functions | CWE-120 |
| 6 | INT32-C / Rule 12.1 | CERT/MISRA | Prevent integer overflow in arithmetic | Semgrep: automotive-integer-overflow, CodeQL: cpp/integer-overflow | CWE-190 |
| 7 | Rule 10.3 | MISRA | Avoid implicit sign conversion | Semgrep: automotive-implicit-sign-conversion | CWE-195 |
| 8 | STR38-C | CERT | Do not use hardcoded cryptographic keys or secrets | Semgrep: automotive-hardcoded-key | CWE-798 |
| 9 | MSC30-C | CERT | Do not use weak cryptographic algorithms (MD5, SHA1, DES) | Semgrep: automotive-weak-crypto | CWE-327 |
| 10 | Rule 16.4 | MISRA | All switch statements must have a default case | Semgrep: automotive-missing-default-switch | N/A |
| 11 | SIG30-C | CERT | Do not call async-signal-unsafe functions in signal handlers | Semgrep: automotive-signal-in-handler | CWE-479 |
| 12 | MEM30-C | CERT | Do not access freed memory (use-after-free) | CodeQL: cpp/use-after-free | CWE-416 |
| 13 | MEM34-C | CERT | Do not free memory not allocated by malloc-family | CodeQL: cpp/double-free | CWE-415 |
| 14 | ARR38-C | CERT | Validate array bounds before access | CodeQL: cpp/out-of-bounds-access | CWE-125 |
| 15 | CON32-C | CERT | Prevent data races in multi-threaded code | CodeQL: cpp/data-race | CWE-362 |
| 16 | ENV33-C | CERT | Do not call system() with untrusted input | CodeQL: cpp/command-injection | CWE-78 |
| 17 | FIO42-C | CERT | Close files when no longer needed | CodeQL: cpp/resource-leak | CWE-775 |
| 18 | Rule 21.6 | MISRA | Do not use standard I/O in safety-critical code | Semgrep custom rule | N/A |
| 19 | Custom | ISO 21434 | Validate all CAN message IDs before processing data | Semgrep: automotive-can-no-validation | CWE-20 |
| 20 | Custom | ISO 21434 | Check CAN DLC before accessing data bytes | Semgrep: automotive-can-dlc-unchecked | CWE-125 |

---

## 5. Advisory Rules

Violations generate warnings but **do not block** the pipeline.

| # | Rule ID | Standard | Description | Enforcement | CWE |
|---|---|---|---|---|---|
| 1 | Rule 15.5 | MISRA | A function should have a single point of exit | Semgrep custom rule | N/A |
| 2 | Rule 11.5 | MISRA | Avoid casts from void pointer | CodeQL query | N/A |
| 3 | Rule 2.7 | MISRA | No unused parameters | Compiler warning -Wunused-parameter | N/A |
| 4 | Rule 8.13 | MISRA | Pointer parameters should be const if not modified | Compiler warning -Wsuggest-attribute=const | N/A |
| 5 | DCL06-C | CERT | Use meaningful symbolic constants | Manual review | N/A |

---

## 6. Deviations Process

1. **Request**: Developer submits a deviation request via GitHub Issue with label `misra-deviation`
2. **Justification**: Must include: rule ID, code location, why compliance is impractical, risk assessment
3. **Review**: Cybersecurity Engineer reviews and assesses safety/security impact
4. **Approval**: Safety Manager approves if risk is acceptable
5. **Documentation**: Deviation recorded in code as structured comment:
   ```c
   /* DEVIATION: MISRA Rule 21.17 - STR31-C
    * Justification: Fixed-size buffer with compile-time size validation
    * Approved by: [Name], Date: [YYYY-MM-DD]
    * Deviation ID: DEV-XXX
    */
   ```
6. **Tracking**: All deviations tracked in cybersecurity case work products index

---

## 7. Tooling Configuration

| Tool | Config File | Purpose |
|---|---|---|
| Semgrep | [security/semgrep.yml](../../security/semgrep.yml) | Custom MISRA/CERT/automotive rules |
| CodeQL | [security/codeql-config.yml](../../security/codeql-config.yml) | Security-extended + quality queries for C/C++ |
| GCC/Clang | [build/Makefile](../../build/Makefile) | Compiler warnings as errors (-Werror) |

**Local development**:
```bash
# Run Semgrep locally
semgrep --config security/semgrep.yml src/

# Run cppcheck
cppcheck --enable=all --std=c11 src/

# Build with all warnings
make PLATFORM=rhivos-x86_64 all
```

---

## 8. Code Review Checklist

Before approving a PR, reviewers must verify:

- [ ] No MISRA mandatory rule violations (Semgrep clean)
- [ ] No CodeQL error-severity findings
- [ ] All external inputs validated (range, type, length)
- [ ] No hardcoded secrets, keys, or credentials
- [ ] Memory allocations checked for failure
- [ ] No buffer overflows (bounded copy operations)
- [ ] Error paths handle resources correctly (no leaks)
- [ ] Thread-safety verified for shared data
- [ ] CAN message handlers validate ID and DLC
- [ ] Cryptographic operations use approved algorithms (AES-256, SHA-256+)
- [ ] Unit tests cover new code paths
- [ ] ASIL decomposition documented if applicable

---

## References

- [Semgrep Rules](../../security/semgrep.yml)
- [CodeQL Config](../../security/codeql-config.yml)
- [TARA](tara-template.md)
- [Cybersecurity Case](cybersecurity-case.md)
- [Safety Requirements](../iso26262/safety-requirements.md)
