#!/usr/bin/env bash
# Cloud Environment Setup for AILANG Development
# Installs Go, make, gh, and builds AILANG binary

set -euo pipefail

echo "=========================================="
echo "AILANG Cloud Environment Setup"
echo "=========================================="
echo ""

# Colors (if supported)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
info() { echo "→ $1"; }

FAILURES=0

# Step 1: Check current environment
echo "Step 1/7: Assessing current environment..."
echo ""

info "System info:"
echo "  RAM: $(free -h 2>/dev/null | awk '/Mem:/ {print $2}' || echo 'unknown')"
echo "  CPUs: $(nproc 2>/dev/null || echo 'unknown')"
echo "  Disk: $(df -h . 2>/dev/null | awk 'NR==2 {print $4}' || echo 'unknown') available"
echo ""

# Step 2: Fix DNS if needed
echo "Step 2/7: Checking DNS..."

if ! host google.com > /dev/null 2>&1; then
    warn "DNS not working, configuring Google DNS..."
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    echo "nameserver 8.8.4.4" >> /etc/resolv.conf
    success "DNS configured"
else
    success "DNS working"
fi
echo ""

# Step 3: Install make
echo "Step 3/7: Installing make..."

if command -v make &> /dev/null; then
    success "make already installed: $(make --version 2>/dev/null | head -1)"
else
    info "Installing make via apt..."
    apt-get update -qq
    apt-get install -y -qq make
    success "make installed"
fi
echo ""

# Step 4: Install Go
echo "Step 4/7: Installing Go 1.24..."

GO_VERSION="1.24.4"
GO_INSTALLED_VERSION=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//' || echo "none")

if [[ "$GO_INSTALLED_VERSION" == "1.24"* ]]; then
    success "Go $GO_INSTALLED_VERSION already installed"
else
    info "Downloading Go $GO_VERSION..."

    # Try wget first (more reliable in some environments)
    if command -v wget &> /dev/null; then
        wget --no-check-certificate -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    else
        curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
    fi

    info "Installing Go..."
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz

    success "Go $GO_VERSION installed to /usr/local/go"
fi

# Ensure Go is in PATH
export PATH=/usr/local/go/bin:$PATH
echo ""

# Step 5: Install GitHub CLI
echo "Step 5/7: Installing GitHub CLI..."

GH_VERSION="2.63.2"

if command -v gh &> /dev/null; then
    success "gh already installed: $(gh --version 2>/dev/null | head -1)"
else
    info "Downloading gh $GH_VERSION..."

    if command -v wget &> /dev/null; then
        wget --no-check-certificate -q "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" -O /tmp/gh.tar.gz
    else
        curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" -o /tmp/gh.tar.gz
    fi

    info "Installing gh..."
    tar -xzf /tmp/gh.tar.gz -C /tmp
    mv /tmp/gh_${GH_VERSION}_linux_amd64/bin/gh /usr/local/bin/
    rm -rf /tmp/gh*

    success "gh $GH_VERSION installed"
fi
echo ""

# Step 6: Download Go modules and build AILANG
echo "Step 6/7: Building AILANG..."

# Set environment for reliable builds
export GOTOOLCHAIN=local
export GOPROXY=direct

info "Downloading Go modules..."
go mod download

info "Building AILANG..."
make build

if [[ -f "bin/ailang" ]]; then
    success "AILANG built: bin/ailang"
else
    fail "AILANG build failed"
    FAILURES=$((FAILURES + 1))
fi
echo ""

# Step 7: Verify
echo "Step 7/7: Verification..."

# Test AILANG runs
if ./bin/ailang run examples/runnable/hello.ail 2>&1 | grep -q "Hello, AILANG!"; then
    success "AILANG runs correctly"
else
    fail "AILANG test failed"
    FAILURES=$((FAILURES + 1))
fi

# Quick test of core packages
info "Running quick test..."
if go test ./internal/lexer/... -count=1 > /tmp/setup_test.log 2>&1; then
    success "Core tests pass"
else
    warn "Some tests failed - see /tmp/setup_test.log"
fi
echo ""

# Summary
echo "=========================================="
if [[ $FAILURES -eq 0 ]]; then
    echo -e "${GREEN}✓ Setup Complete!${NC}"
    echo ""
    echo "Environment ready. You can now:"
    echo "  • Build: make build"
    echo "  • Run: ./bin/ailang run --caps IO file.ail"
    echo "  • Test: go test ./internal/..."
    echo ""
    echo "Set these in each session:"
    echo "  export PATH=/usr/local/go/bin:\$PATH"
    echo "  export GOTOOLCHAIN=local"
    echo "  export GOPROXY=direct"
else
    echo -e "${RED}✗ Setup had $FAILURES failure(s)${NC}"
    echo "Check the errors above and try again."
    exit 1
fi
echo "=========================================="
