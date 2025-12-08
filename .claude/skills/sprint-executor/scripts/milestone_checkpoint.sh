#!/usr/bin/env bash
# Run checkpoint after completing a milestone

set -euo pipefail

MILESTONE_NAME="${1:-Unknown Milestone}"
SPRINT_DIR=".ailang/state/sprints"

echo "Running checkpoint for: $MILESTONE_NAME"
echo

FAILURES=0

# 0. Find current sprint JSON
CURRENT_SPRINT=""
SPRINT_ID=""
if [[ -d "$SPRINT_DIR" ]]; then
    # Find in_progress sprint
    for f in "$SPRINT_DIR"/sprint_*.json; do
        if [[ -f "$f" ]] && grep -q '"status": "in_progress"' "$f" 2>/dev/null; then
            CURRENT_SPRINT="$f"
            SPRINT_ID=$(basename "$f" .json | sed 's/sprint_//')
            break
        fi
    done
fi

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
    echo

    # 5. Sprint JSON reminder
    if [[ -n "$CURRENT_SPRINT" ]]; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "📋 SPRINT JSON UPDATE REQUIRED"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo
        echo "Sprint: $SPRINT_ID"
        echo "File:   $CURRENT_SPRINT"
        echo
        echo "Current milestone status:"
        # Show milestone statuses from JSON
        if command -v jq &> /dev/null; then
            jq -r '.features[] | "  • \(.id): passes=\(.passes // "null"), completed=\(.completed // "null")"' "$CURRENT_SPRINT" 2>/dev/null || echo "  (could not parse JSON)"
        else
            grep -E '"id"|"passes"|"completed"' "$CURRENT_SPRINT" | head -20 || echo "  (jq not available)"
        fi
        echo
        echo "⚠️  After completing $MILESTONE_NAME:"
        echo "   1. Update passes: true/false in the JSON"
        echo "   2. Set completed: \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\""
        echo "   3. Add notes about what was done"
        echo
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    fi
    echo
    echo "Ready to proceed to next milestone."
    exit 0
else
    echo "✗ $FAILURES check(s) failed"
    echo "Fix issues before marking milestone complete."
    exit 1
fi
