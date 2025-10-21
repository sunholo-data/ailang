#!/usr/bin/env bash
# Run checkpoint after completing a milestone

set -euo pipefail

MILESTONE_NAME="${1:-Unknown Milestone}"

echo "Running checkpoint for: $MILESTONE_NAME"
echo

FAILURES=0

# 1. Run tests
echo "1/3 Running tests..."
if make test > /tmp/milestone_test.log 2>&1; then
    echo "  ✓ Tests pass"
else
    echo "  ✗ Tests fail"
    echo "  See: /tmp/milestone_test.log"
    echo "  FIX BEFORE PROCEEDING!"
    FAILURES=$((FAILURES + 1))
fi
echo

# 2. Run linting
echo "2/3 Running linter..."
if make lint > /tmp/milestone_lint.log 2>&1; then
    echo "  ✓ Linting passes"
else
    echo "  ✗ Linting fails"
    echo "  See: /tmp/milestone_lint.log"
    echo "  FIX BEFORE PROCEEDING!"
    FAILURES=$((FAILURES + 1))
fi
echo

# 3. Show files changed
echo "3/3 Files changed in this milestone..."
git diff --stat HEAD | tail -10 || echo "No changes yet"
echo

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ Milestone checkpoint passed!"
    echo "Ready to proceed to next milestone."
    exit 0
else
    echo "✗ $FAILURES check(s) failed"
    echo "Fix issues before marking milestone complete."
    exit 1
fi
