#!/usr/bin/env bash
# Validate prerequisites before starting sprint execution

set -euo pipefail

echo "Validating sprint prerequisites..."
echo

FAILURES=0

# 0. Sync GitHub issues via ailang messages
echo "0/5 Syncing GitHub issues..."
if command -v ailang &> /dev/null; then
    if ailang messages import-github --labels bug,feature,ailang-message 2>/dev/null; then
        echo "  ✓ Synced issues from GitHub"
    else
        echo "  ⚠ GitHub sync failed (continuing anyway)"
    fi
else
    echo "  ⚠ ailang not found, skipping GitHub sync"
fi
echo

# 1. Check working directory is clean
echo "1/5 Checking working directory..."
if [[ -z $(git status --short) ]]; then
    echo "  ✓ Working directory clean"
else
    echo "  ⚠ Working directory has uncommitted changes:"
    git status --short | head -10
    echo "  Consider committing or stashing before starting sprint"
fi
echo

# 2. Check current branch
echo "2/5 Checking current branch..."
BRANCH=$(git branch --show-current)
if [[ "$BRANCH" == "dev" ]] || [[ "$BRANCH" == "main" ]]; then
    echo "  ✓ On branch: $BRANCH"
else
    echo "  ⚠ On branch: $BRANCH (expected 'dev' or 'main')"
fi
echo

# 3. Run tests
echo "3/5 Running tests..."
if make test > /tmp/sprint_prereq_test.log 2>&1; then
    echo "  ✓ All tests pass"
else
    echo "  ✗ Tests failing"
    echo "  See: /tmp/sprint_prereq_test.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# 4. Run linting
echo "4/5 Running linter..."
if make lint > /tmp/sprint_prereq_lint.log 2>&1; then
    echo "  ✓ Linting passes"
else
    echo "  ✗ Linting fails"
    echo "  See: /tmp/sprint_prereq_lint.log"
    FAILURES=$((FAILURES + 1))
fi
echo

# 5. Show unread messages (potential issues/feedback)
echo "5/5 Checking unread messages..."
if command -v ailang &> /dev/null; then
    UNREAD_COUNT=$(ailang messages list --unread --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
    if [[ "$UNREAD_COUNT" -gt 0 ]]; then
        echo "  ⚠ $UNREAD_COUNT unread message(s):"
        ailang messages list --unread --limit 5 2>/dev/null | head -10 | sed 's/^/    /'
        echo "  Consider reviewing before starting sprint"
    else
        echo "  ✓ No unread messages"
    fi
else
    echo "  ⚠ ailang not found, skipping message check"
fi
echo

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ All prerequisites validated!"
    echo "Ready to start sprint execution."
    exit 0
else
    echo "✗ $FAILURES prerequisite(s) failed"
    echo "Fix issues before starting sprint."
    exit 1
fi
