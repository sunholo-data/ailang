#!/usr/bin/env bash
# Run pre-release verification checks

set -euo pipefail

echo "Running pre-release checks..."
echo

# Track failures
FAILURES=0

# Test suite
echo "1/5 Running test suite..."
if make test > /tmp/pre_release_test.log 2>&1; then
    echo "  ✓ Tests passed"
else
    echo "  ✗ Tests failed"
    echo "  See: /tmp/pre_release_test.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# Linting
echo "2/5 Running linter..."
if make lint > /tmp/pre_release_lint.log 2>&1; then
    echo "  ✓ Linting passed"
else
    echo "  ✗ Linting failed"
    echo "  See: /tmp/pre_release_lint.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# File size check
echo "3/5 Checking file sizes..."
if make check-file-sizes > /tmp/pre_release_filesizes.log 2>&1; then
    echo "  ✓ File sizes OK (all files ≤800 lines)"
else
    echo "  ✗ Some files exceed 800 lines"
    echo "  See: /tmp/pre_release_filesizes.log"
    echo "  Consider using codebase-organizer agent to split large files"
    FAILURES=$((FAILURES + 1))
fi
echo

# Golden file validation
echo "4/5 Validating golden files..."
if make test-import-errors > /tmp/pre_release_goldens.log 2>&1; then
    echo "  ✓ Golden files match current behavior"
else
    echo "  ✗ Golden file mismatch detected"
    echo "  See: /tmp/pre_release_goldens.log"
    echo "  Run: make regen-import-error-goldens"
    FAILURES=$((FAILURES + 1))
fi
echo

# Agent eval configuration validation
echo "5/5 Validating agent eval configuration..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
if "$PROJECT_ROOT/.claude/skills/post-release/scripts/run_eval_baseline.sh" --validate > /tmp/pre_release_agent.log 2>&1; then
    echo "  ✓ Agent eval configuration valid"
else
    echo "  ✗ Agent eval configuration invalid"
    echo "  See: /tmp/pre_release_agent.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ All pre-release checks passed!"
    echo "Ready to proceed with release."
    exit 0
else
    echo "✗ $FAILURES check(s) failed"
    echo "Fix issues before proceeding with release."
    exit 1
fi
