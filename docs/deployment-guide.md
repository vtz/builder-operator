# Deployment Guide

## 1. Prerequisites

| Requirement | Details |
|---|---|
| Target hardware | AArch64 or x86_64 board with UEFI firmware |
| Network | Internet access for OTA (cellular or Ethernet) |
| Secure boot keys | PK, KEK, db keys enrolled in UEFI firmware |
| GPG keyring | OSTree signing key provisioned on device |
| Fleet management | mTLS client certificate provisioned |

---

## 2. Initial Device Provisioning

### 2.1 Secure Boot Key Enrollment

```bash
# Generate secure boot keys (do this ONCE, store securely)
openssl req -new -x509 -newkey rsa:4096 -keyout PK.key -out PK.crt -days 3650 -nodes -subj "/CN=Edge PK"
openssl req -new -x509 -newkey rsa:4096 -keyout KEK.key -out KEK.crt -days 3650 -nodes -subj "/CN=Edge KEK"
openssl req -new -x509 -newkey rsa:4096 -keyout db.key -out db.crt -days 3650 -nodes -subj "/CN=Edge db"

# Enroll keys in UEFI (device-specific, typically via firmware setup or KeyTool.efi)
# 1. Enter UEFI Setup
# 2. Navigate to Secure Boot key management
# 3. Enroll PK, KEK, and db certificates
# 4. Enable Secure Boot
```

### 2.2 Initial OS Flash

```bash
# Download the RHIVOS image built by the CD pipeline
# Flash to device storage via USB or network boot

# Option A: USB flash
dd if=rhivos-edge-aarch64.raw of=/dev/sdX bs=4M status=progress

# Option B: Network boot (PXE)
# Configure DHCP/TFTP server with RHIVOS installer
```

---

## 3. RHIVOS Image Deployment

### 3.1 Initial OSTree Configuration

```bash
# On the device, configure the OSTree remote
sudo ostree remote add --no-gpg-verify production \
  https://ostree.example.com/repo \
  --set=tls-client-cert-path=/etc/pki/edge/client.crt \
  --set=tls-client-key-path=/etc/pki/edge/client.key

# Import GPG key for commit verification
sudo ostree remote gpg-import production /etc/pki/rpm-gpg/RPM-GPG-KEY-edge-app

# Enable GPG verification
sudo ostree remote set-gpg-verify production true
```

### 3.2 Deploy from OSTree

```bash
# Rebase to production ref
sudo rpm-ostree rebase production:rhivos/edge/aarch64/production

# Reboot to apply
sudo systemctl reboot

# Verify deployment
rpm-ostree status
```

---

## 4. OTA Update Configuration

### 4.1 Automatic Updates

```ini
# /etc/ostree/remotes.d/production.conf
[remote "production"]
url=https://ostree.example.com/repo
gpg-verify=true
tls-client-cert-path=/etc/pki/edge/client.crt
tls-client-key-path=/etc/pki/edge/client.key
```

### 4.2 Update Policy

| Parameter | Value | Rationale |
|---|---|---|
| Automatic check | Every 6 hours | Balance freshness vs bandwidth |
| Automatic apply | Disabled | Fleet management controls deployment |
| Delta updates | Enabled | Reduce bandwidth by ~90% |
| Rollback on failure | Enabled | Health check auto-rollback |
| Health check command | `/usr/bin/edge-app --health-check` | Verify application starts correctly |
| Health check timeout | 60 seconds | Allow for initialization |

### 4.3 Manual Update

```bash
# Check for updates
sudo rpm-ostree upgrade --check

# Download and stage update
sudo rpm-ostree upgrade --download-only

# Apply update (requires reboot)
sudo rpm-ostree upgrade
sudo systemctl reboot
```

---

## 5. RTOS Firmware Deployment

### 5.1 Initial Flash (JTAG/SWD)

```bash
# Using OpenOCD for STM32F4
openocd -f interface/stlink.cfg -f target/stm32f4x.cfg \
  -c "program build/out/edge-app.bin verify reset exit 0x08000000"
```

### 5.2 Field OTA Update

```bash
# Verify firmware signature before flashing
cosign verify-blob \
  --certificate firmware.cert \
  --signature firmware.sig \
  firmware.bin

# Upload via bootloader OTA mechanism
# (Device-specific: MQTT, HTTP, USB mass storage)
```

---

## 6. Signature Verification on Device

### 6.1 Container Image Verification

```bash
# Install cosign on device
# Verify container image from registry
cosign verify \
  --certificate-identity-regexp "https://github.com/ORG/REPO/.github/workflows/cd.yml@refs/heads/main" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/ORG/REPO@sha256:DIGEST
```

### 6.2 SLSA Provenance Verification

```bash
# Verify SLSA provenance
slsa-verifier verify-image ghcr.io/ORG/REPO@sha256:DIGEST \
  --source-uri github.com/ORG/REPO
```

### 6.3 OSTree GPG Verification

```bash
# Verify OSTree commit signature
ostree --repo=/ostree/repo gpg-verify COMMIT_HASH \
  --keyring=/etc/pki/rpm-gpg/RPM-GPG-KEY-edge-app
```

---

## 7. Fleet Management

### 7.1 BlueChi Orchestration

```bash
# List managed nodes
bluechictl list-units

# Monitor node health
bluechictl monitor

# Trigger service restart across fleet
bluechictl restart edge-app.service --node=edge-001
```

### 7.2 Staged Rollout

| Stage | Fleet % | Duration | Criteria to Proceed |
|---|---|---|---|
| Canary | 5% | 24 hours | Zero errors, health checks passing |
| Early adopters | 25% | 48 hours | Error rate < 0.1% |
| General availability | 100% | - | No regressions detected |

---

## 8. Rollback Procedures

### 8.1 Automatic Rollback

If the health check fails after an update, OSTree automatically rolls back:

```
Update applied -> Reboot -> Health check runs
                                |
                          Pass? -> Continue with new version
                          Fail? -> Automatic rollback to previous deployment
```

### 8.2 Manual Rollback

```bash
# List available deployments
rpm-ostree status

# Rollback to previous deployment
sudo rpm-ostree rollback
sudo systemctl reboot

# Verify rollback
rpm-ostree status
```

---

## 9. Monitoring and Logging

### 9.1 System Journal

```bash
# View edge-app logs
journalctl -u edge-app.service -f

# Forward logs to central SIEM
# Configured via systemd-journal-remote in aib-manifest.yml
```

### 9.2 Health Endpoint

```bash
# Check application health
curl -s http://localhost:8080/health | jq .

# Expected response:
# { "status": "healthy", "version": "0.1.0", "uptime": 3600 }
```

---

## 10. Troubleshooting

| Issue | Diagnosis | Solution |
|---|---|---|
| Update fails to apply | `rpm-ostree status`, check journal | Verify network, retry, check GPG key |
| Boot loop after update | Automatic rollback should trigger | If not: hold button for recovery mode |
| Signature verification fails | `ostree gpg-verify` output | Re-import GPG key, check key expiry |
| Health check fails | `journalctl -u edge-app` | Check application config, dependencies |
| OTA download slow | Network diagnostics | Enable delta updates, check bandwidth |
| Secure boot violation | UEFI log output | Re-sign image with enrolled db key |

---

## 11. Security Hardening Checklist

Post-deployment verification:

- [ ] Secure boot enabled and keys enrolled
- [ ] SELinux in enforcing mode (`getenforce`)
- [ ] No default users or passwords
- [ ] SSH disabled (`systemctl status sshd`)
- [ ] Firewall active, no open ports (`firewall-cmd --list-all`)
- [ ] OSTree GPG verification enabled
- [ ] Audit logging active (`systemctl status auditd`)
- [ ] Time synchronization active (`chronyc tracking`)
- [ ] Application running as non-root user
- [ ] Resource limits configured (MemoryMax, CPUQuota)
- [ ] Health check endpoint responding
- [ ] Log forwarding configured and active

---

## References

- [Pipeline Architecture](pipeline-architecture.md)
- [AIB Manifest](../build/aib-manifest.yml)
- [Cosign Policy](../security/cosign-policy.yml)
- [Incident Response Plan](../compliance/shared/incident-response-plan.md)
