# Cloud Setup Troubleshooting Guide

This guide covers common issues when setting up AILANG in cloud/mobile Claude Code environments.

## DNS Issues

### Symptom
```
dial tcp: lookup storage.googleapis.com on [::1]:53: read udp ... connection refused
```

### Cause
Cloud environment has no DNS configured or broken DNS resolution.

### Solution
```bash
# Add Google DNS servers
echo "nameserver 8.8.8.8" > /etc/resolv.conf
echo "nameserver 8.8.4.4" >> /etc/resolv.conf
```

---

## Go Toolchain Auto-Download

### Symptom
```
go: downloading go1.24.11 (linux/amd64)
go: download go1.24.11: ... connection refused
```

### Cause
The `go.mod` file specifies a `toolchain` directive, and Go tries to download that exact version.

### Solution
```bash
# Tell Go to use the installed version
export GOTOOLCHAIN=local
```

This prevents Go from trying to download a newer toolchain.

---

## Go Module Proxy Blocked

### Symptom
```
Get "https://proxy.golang.org/...": i/o timeout
```
or
```
Get "https://storage.googleapis.com/...": context deadline exceeded
```

### Cause
The Go module proxy (proxy.golang.org) or its backend (storage.googleapis.com) is blocked or slow.

### Solution
```bash
# Bypass proxy, download directly from sources
export GOPROXY=direct
go mod download
```

---

## SSL/TLS Handshake Failures

### Symptom
```
curl: (35) OpenSSL/3.0.13: error:0A000410:SSL routines::sslv3 alert handshake failure
```

### Cause
Some cloud environments have SSL/TLS issues with certain endpoints.

### Solution
Use `wget` with certificate checking disabled:
```bash
# Instead of curl:
wget --no-check-certificate https://go.dev/dl/go1.24.4.linux-amd64.tar.gz -O /tmp/go.tar.gz
```

---

## Missing Basic Commands

### Symptom
```
/bin/bash: line 1: head: command not found
/bin/bash: line 1: tail: command not found
/bin/bash: line 1: grep: command not found
```

### Cause
Minimal container image without coreutils.

### Solutions

**Option 1**: Install coreutils
```bash
apt-get update && apt-get install -y coreutils grep
```

**Option 2**: Avoid piping, run commands directly
```bash
# Instead of: go test ./... 2>&1 | tail -20
# Just run: go test ./...
# And manually scan output
```

---

## apt-get Issues

### Symptom
```
E: Malformed entry in list file
E: The list of sources could not be read
```

### Cause
Corrupted apt sources list.

### Solution
```bash
# Remove problematic source
rm -f /etc/apt/sources.list.d/problematic-file.list
apt-get update
```

---

## Go Version Mismatch

### Symptom
```
go: go.mod requires go >= 1.24
```

### Cause
apt-get installed older Go version (e.g., 1.22).

### Solution
Install Go directly from golang.org:
```bash
wget --no-check-certificate https://go.dev/dl/go1.24.4.linux-amd64.tar.gz -O /tmp/go.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH=/usr/local/go/bin:$PATH
```

---

## Tests Hanging

### Symptom
Tests start running but hang indefinitely on certain packages.

### Cause
Some test packages (eval harness, AI clients) may require network access that's blocked or slow.

### Solution
Run targeted tests instead of full suite:
```bash
# Core packages (no network needed)
go test ./internal/lexer/... ./internal/parser/... ./internal/types/... ./internal/eval/... -count=1

# Skip problematic packages
go test $(go list ./... | grep -v eval_harness | grep -v /ai/) -count=1
```

---

## GitHub CLI Authentication

### Symptom
```
gh: To use GitHub CLI, run 'gh auth login' first.
```

### Cause
gh CLI is installed but not authenticated.

### Solution
```bash
# Interactive login
gh auth login

# Or use token
gh auth login --with-token < token.txt

# Check status
gh auth status
```

---

## Build Failures

### Symptom
```
make: *** [Makefile:40: build] Error 1
```

### Cause
Various - check the actual error message above this line.

### Common Fixes

**Missing dependencies:**
```bash
go mod download
```

**Wrong Go version:**
```bash
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
```

**Network issues:**
```bash
export GOPROXY=direct
```

---

## Quick Diagnostic Commands

```bash
# Check all tools
which go make gh git

# Check Go setup
go version
go env GOPATH GOROOT GOTOOLCHAIN GOPROXY

# Check network
curl -s --connect-timeout 5 https://api.github.com | head -1

# Check DNS
cat /etc/resolv.conf
host google.com

# Check disk space
df -h .

# Check memory
free -h
```

---

## Environment Variables Checklist

Set these at the start of every session:

```bash
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
export GOPROXY=direct
```

Or add to `~/.bashrc` for persistence:

```bash
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
echo 'export GOTOOLCHAIN=local' >> ~/.bashrc
echo 'export GOPROXY=direct' >> ~/.bashrc
```
