#!/usr/bin/env bash
# Run pre-release verification checks

set -euo pipefail

echo "Running pre-release checks..."
echo

# Track failures
FAILURES=0

# Test suite
echo "1/3 Running test suite..."
if make test > /tmp/pre_release_test.log 2>&1; then
    echo "  ✓ Tests passed"
else
    echo "  ✗ Tests failed"
    echo "  See: /tmp/pre_release_test.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# Linting
echo "2/3 Running linter..."
if make lint > /tmp/pre_release_lint.log 2>&1; then
    echo "  ✓ Linting passed"
else
    echo "  ✗ Linting failed"
    echo "  See: /tmp/pre_release_lint.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# File size check
echo "3/3 Checking file sizes..."
if make check-file-sizes > /tmp/pre_release_filesizes.log 2>&1; then
    echo "  ✓ File sizes OK (all files ≤800 lines)"
else
    echo "  ✗ Some files exceed 800 lines"
    echo "  See: /tmp/pre_release_filesizes.log"
    echo "  Consider using codebase-organizer agent to split large files"
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
