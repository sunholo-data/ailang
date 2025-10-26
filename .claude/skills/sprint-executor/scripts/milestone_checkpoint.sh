#!/usr/bin/env bash
# Run checkpoint after completing a milestone

set -euo pipefail

MILESTONE_NAME="${1:-Unknown Milestone}"

echo "Running checkpoint for: $MILESTONE_NAME"
echo

FAILURES=0

# 1. Run tests
echo "1/4 Running tests..."
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
echo "2/4 Running linter..."
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
echo "3/4 Files changed in this milestone..."
git diff --stat HEAD | tail -10 || echo "No changes yet"
echo

# 4. File size warnings (AI-friendly codebase - keep files <800 lines)
echo "4/4 Checking file sizes..."
LARGE_FILES=0
for file in $(git diff --name-only --diff-filter=AM HEAD 2>/dev/null || echo ""); do
    if [[ -f "$file" && "$file" == *.go ]]; then
        lines=$(wc -l < "$file" 2>/dev/null || echo "0")
        if [[ $lines -gt 800 ]]; then
            echo "  ❌ $file: $lines lines (CRITICAL: >800, must split!)"
            LARGE_FILES=$((LARGE_FILES + 1))
        elif [[ $lines -gt 500 ]]; then
            echo "  ⚠️  $file: $lines lines (warning: consider splitting if >800)"
        elif [[ $lines -gt 300 ]]; then
            echo "  📝 $file: $lines lines (healthy)"
        fi
    fi
done

if [[ $LARGE_FILES -gt 0 ]]; then
    echo "  ⚠️  $LARGE_FILES file(s) exceed 800 lines (see codebase-health guidelines)"
    echo "  Consider splitting before adding more features"
else
    echo "  ✓ All files within size guidelines"
fi
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
