# Edge CI/CD Pipeline

End-to-end CI/CD pipeline for building, securing, and deploying software on edge devices (RHIVOS & RTOS) with full safety and cybersecurity compliance artifacts.

## Standards Coverage

| Standard | Scope | Key Artifacts |
|----------|-------|---------------|
| **ISO 21434** | Automotive Cybersecurity | TARA, Cybersecurity Case, Secure Coding Guidelines |
| **ISO 26262** | Functional Safety | HARA, Safety Case, Safety Requirements, Tool Qualification |
| **IEC 62443** | Industrial Cybersecurity | Security Requirements (SL1–SL4), 7 Foundational Requirements |
| **ASPICE** | Process Maturity | Process Assessment (SWE/SUP/MAN), Bidirectional Traceability |

## Target Platforms

| Platform | Architecture | Build Target |
|----------|-------------|--------------|
| RHIVOS (Red Hat In-Vehicle OS) | AArch64 | `rhivos-aarch64` |
| RHIVOS | x86_64 | `rhivos-x86_64` |
| FreeRTOS (Cortex-M4) | ARM Thumb | `rtos-cortex-m4` |

## Repository Structure

```
├── .github/workflows/
│   ├── ci.yml                  # 10-job CI pipeline (build, SAST, SBOM, vuln scan, quality gate)
│   ├── cd.yml                  # CD pipeline (image build, sign, provenance, staged deploy)
│   ├── security-audit.yml      # Weekly security audit (rescan, fuzz, auto-issue creation)
│   └── compliance-report.yml   # On-demand compliance bundle generator
├── build/
│   ├── Makefile                # Multi-platform build with hardening flags
│   ├── Containerfile           # Multi-stage container build (UBI9)
│   ├── aib-manifest.yml        # Automotive Image Builder manifest for RHIVOS
│   └── toolchain/
│       ├── aarch64-rhivos.cmake    # CMake toolchain for AArch64
│       └── rtos-freertos.cmake     # CMake toolchain for Cortex-M4
├── security/
│   ├── trivy.yaml              # Vulnerability scanner config
│   ├── semgrep.yml             # 14 custom SAST rules (MISRA C, CERT C, CWE)
│   ├── codeql-config.yml       # CodeQL security queries
│   ├── cosign-policy.yml       # Sigstore keyless signing policy
│   └── trufflehog.yml          # Secrets detection config
├── compliance/
│   ├── iso21434/               # Cybersecurity work products
│   ├── iso26262/               # Safety work products
│   ├── iec62443/               # Industrial security requirements
│   ├── aspice/                 # Process assessment & traceability
│   └── shared/                 # Vulnerability mgmt, incident response, change mgmt
└── docs/
    ├── pipeline-architecture.md    # Architecture overview with diagrams
    └── deployment-guide.md         # Secure boot, OTA, fleet management
```

## Getting Started

### Prerequisites

- GitHub repository with Actions enabled
- Runner with Docker support (for container builds)
- Cross-compilation toolchains:
  - `aarch64-linux-gnu-gcc` (for RHIVOS AArch64)
  - `arm-none-eabi-gcc` (for RTOS Cortex-M4)

### 1. Clone and Configure

```bash
git clone https://github.com/<your-org>/edge-pipeline.git
cd edge-pipeline
```

### 2. Set Up GitHub Secrets

The pipeline requires the following secrets configured in your GitHub repository:

| Secret | Purpose | Required By |
|--------|---------|-------------|
| `STAGING_OSTREE_URL` | OSTree repo URL for staging | cd.yml |
| `STAGING_SSH_KEY` | SSH key for staging deployment | cd.yml |
| `PRODUCTION_OSTREE_URL` | OSTree repo URL for production | cd.yml |
| `PRODUCTION_SSH_KEY` | SSH key for production deployment | cd.yml |

> **Note:** Container and artifact signing uses Cosign keyless mode (Sigstore/Fulcio OIDC), so no signing keys need to be configured — GitHub's OIDC identity is used automatically.

### 3. Build Locally

```bash
# Build for RHIVOS AArch64
make PLATFORM=rhivos-aarch64 all

# Build for RTOS Cortex-M4
make PLATFORM=rtos-cortex-m4 CROSS_COMPILE=arm-none-eabi- all

# Run tests
make test

# Generate SBOM
make sbom

# Run SAST
make lint

# Generate coverage report (ISO 26262)
make coverage

# Run fuzz testing (IEC 62443)
make fuzz FUZZ_DURATION=300
```

### 4. CI Pipeline (Automatic)

The CI pipeline runs on every push and pull request. It executes 10 jobs:

```
build (matrix: 3 platforms)
  → sast-codeql
  → sast-semgrep
  → sbom-generate (CycloneDX + SPDX)
  → vulnerability-scan (Trivy)
  → osv-scan
  → secrets-scan (TruffleHog)
  → license-scan (ScanCode)
  → quality-gate (aggregates all results, blocks on critical findings)
  → compliance-snapshot (archives evidence bundle)
```

**Quality Gates (Blocking):**
- CRITICAL or HIGH vulnerabilities
- MISRA C mandatory rule violations
- Secrets detected in source
- Test failures
- SBOM generation failure
- Coverage below ASIL threshold

**Quality Gates (Advisory):**
- MEDIUM/LOW vulnerabilities
- Compiler warnings
- License compatibility issues

### 5. CD Pipeline (on release tags)

Triggered on tags matching `v*`. Deploys through staging → integration tests → production:

```bash
# Create a release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The CD pipeline:
1. Builds container image (RHIVOS) or firmware binary (RTOS)
2. Signs with Cosign keyless (Sigstore)
3. Generates SLSA Level 3 provenance attestation
4. Deploys to staging
5. Runs integration tests
6. Deploys to production (requires manual approval)

### 6. Generate Compliance Report

Run on-demand to produce a compliance evidence bundle for audits:

```bash
gh workflow run compliance-report.yml \
  -f release_version=v1.0.0 \
  -f security_level=SL2 \
  -f target_asil=ASIL-B
```

This collects all artifacts, generates traceability matrices, and packages a signed ZIP bundle.

### 7. Weekly Security Audit

Runs automatically every Monday at 06:00 UTC:
- Re-scans all dependencies against latest vulnerability databases
- Runs fuzz regression tests
- Auto-creates GitHub Issues for any new critical findings

## Signing & Verification

### Verify a Container Image

```bash
# Verify signature
cosign verify \
  --certificate-identity-regexp="github.com/workflows/.+" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/<org>/edge-app:v1.0.0

# Verify SLSA provenance
gh attestation verify oci://ghcr.io/<org>/edge-app:v1.0.0
```

### Verify on Device (OSTree)

```bash
# Add signed OSTree remote
ostree remote add edge-prod \
  https://ostree.example.com/repo \
  --gpg-import=/etc/pki/rpm-gpg/RPM-GPG-KEY-edge

# Pull and deploy (atomic)
rpm-ostree rebase edge-prod:rhivos/edge/aarch64/production
systemctl reboot
```

## Compliance Documents

### ISO 21434 (Cybersecurity)
- [TARA Template](compliance/iso21434/tara-template.md) — Threat Analysis & Risk Assessment with 12 threat scenarios
- [Cybersecurity Case](compliance/iso21434/cybersecurity-case.md) — Evidence binder with 18 work products
- [Cybersecurity Plan](compliance/iso21434/cybersecurity-plan.md) — Lifecycle mapping to CI/CD stages
- [Secure Coding Guidelines](compliance/iso21434/secure-coding-guidelines.md) — 20 mandatory rules (MISRA/CERT)

### ISO 26262 (Functional Safety)
- [HARA Template](compliance/iso26262/hara-template.md) — 10 hazards with ASIL classification
- [Safety Plan](compliance/iso26262/safety-plan.md) — Safety lifecycle mapped to CI/CD
- [Safety Case](compliance/iso26262/safety-case.md) — GSN argument structure
- [Safety Requirements](compliance/iso26262/safety-requirements.md) — 15 SSRs with verification methods
- [Tool Qualification](compliance/iso26262/tool-qualification.md) — TCL assessment for 16 pipeline tools

### IEC 62443 (Industrial Security)
- [Security Requirements](compliance/iec62443/security-requirements.md) — 30+ CRs across 7 Foundational Requirements

### ASPICE (Process Maturity)
- [Process Assessment](compliance/aspice/process-assessment.md) — 11 processes (SWE.1-6, SUP, MAN)
- [Traceability Matrix](compliance/aspice/traceability-matrix.md) — Bidirectional req-to-test tracing

### Shared
- [Vulnerability Management Plan](compliance/shared/vulnerability-management-plan.md)
- [Incident Response Plan](compliance/shared/incident-response-plan.md)
- [Change Management Plan](compliance/shared/change-management-plan.md)

## Customization

### Adding Your Application Code

1. Place source files under `src/`
2. Update `build/Makefile` with your source files and targets
3. Update `build/Containerfile` to copy your application
4. Add test files under `tests/`

### Adjusting ASIL Level

Edit the coverage thresholds in `.github/workflows/ci.yml` under the `quality-gate` job:

| ASIL | Statement | Branch | MC/DC |
|------|-----------|--------|-------|
| QM | — | — | — |
| A | 80% | — | — |
| B | 90% | 80% | — |
| C | 95% | 90% | 80% |
| D | 100% | 100% | 100% |

### Adjusting Security Level (IEC 62443)

The compliance report workflow accepts `security_level` as input (SL1–SL4). Higher levels enable additional Component Requirements. See `compliance/iec62443/security-requirements.md` for the full mapping.

## License

This is a reference pipeline template. Adapt to your organization's requirements and licensing policies.
