#!/usr/bin/env bash
# Verify AILANG cloud environment is correctly configured

set -euo pipefail

echo "=========================================="
echo "AILANG Environment Verification"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

success() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }

FAILURES=0
WARNINGS=0

# Ensure PATH includes Go
export PATH=/usr/local/go/bin:$PATH

echo "Checking tools..."
echo ""

# Check Go
if command -v go &> /dev/null; then
    GO_VER=$(go version 2>/dev/null | awk '{print $3}')
    if [[ "$GO_VER" == "go1.24"* ]]; then
        success "Go: $GO_VER"
    else
        warn "Go: $GO_VER (expected 1.24+)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    fail "Go: not found"
    FAILURES=$((FAILURES + 1))
fi

# Check make
if command -v make &> /dev/null; then
    success "make: $(make --version 2>/dev/null | head -1 | awk '{print $3}')"
else
    fail "make: not found"
    FAILURES=$((FAILURES + 1))
fi

# Check gh
if command -v gh &> /dev/null; then
    success "gh: $(gh --version 2>/dev/null | head -1 | awk '{print $3}')"
else
    warn "gh: not found (optional for GitHub operations)"
    WARNINGS=$((WARNINGS + 1))
fi

# Check git
if command -v git &> /dev/null; then
    success "git: $(git --version 2>/dev/null | awk '{print $3}')"
else
    fail "git: not found"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "Checking AILANG binary..."
echo ""

# Check AILANG binary exists
if [[ -f "bin/ailang" ]]; then
    success "bin/ailang exists"
else
    fail "bin/ailang not found (run: make build)"
    FAILURES=$((FAILURES + 1))
fi

# Test AILANG runs
if [[ -f "bin/ailang" ]]; then
    OUTPUT=$(./bin/ailang run examples/runnable/hello.ail 2>&1 || true)
    if echo "$OUTPUT" | grep -q "Hello, AILANG!"; then
        success "AILANG executes correctly"
    else
        fail "AILANG execution failed"
        echo "    Output: $OUTPUT"
        FAILURES=$((FAILURES + 1))
    fi
fi

echo ""
echo "Checking environment variables..."
echo ""

# Check GOTOOLCHAIN
if [[ "${GOTOOLCHAIN:-}" == "local" ]]; then
    success "GOTOOLCHAIN=local"
else
    warn "GOTOOLCHAIN not set to 'local' (may cause auto-download issues)"
    WARNINGS=$((WARNINGS + 1))
fi

# Check GOPROXY
if [[ "${GOPROXY:-}" == "direct" ]]; then
    success "GOPROXY=direct"
else
    warn "GOPROXY not set to 'direct' (may cause proxy issues)"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""
echo "Checking network..."
echo ""

# Check network connectivity
if curl -s --connect-timeout 5 -o /dev/null https://api.github.com 2>/dev/null; then
    success "Network: GitHub API accessible"
else
    warn "Network: GitHub API not accessible"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""
echo "Running quick tests..."
echo ""

# Run lexer tests
if go test ./internal/lexer/... -count=1 > /tmp/verify_test.log 2>&1; then
    success "Lexer tests pass"
else
    fail "Lexer tests fail - see /tmp/verify_test.log"
    FAILURES=$((FAILURES + 1))
fi

# Run parser tests
if go test ./internal/parser/... -count=1 >> /tmp/verify_test.log 2>&1; then
    success "Parser tests pass"
else
    fail "Parser tests fail - see /tmp/verify_test.log"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "=========================================="

if [[ $FAILURES -eq 0 ]] && [[ $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo ""
    echo "Environment is fully configured and ready."
elif [[ $FAILURES -eq 0 ]]; then
    echo -e "${YELLOW}⚠ Passed with $WARNINGS warning(s)${NC}"
    echo ""
    echo "Environment is usable but may have minor issues."
    echo "Consider setting environment variables:"
    echo "  export PATH=/usr/local/go/bin:\$PATH"
    echo "  export GOTOOLCHAIN=local"
    echo "  export GOPROXY=direct"
else
    echo -e "${RED}✗ $FAILURES check(s) failed${NC}"
    echo ""
    echo "Run setup script to fix:"
    echo "  .claude/skills/cloud-setup/scripts/setup.sh"
    exit 1
fi
echo "=========================================="
