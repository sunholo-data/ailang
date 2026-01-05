#!/usr/bin/env bash
# Run checkpoint after completing a milestone
#
# CRITICAL: This script enforces VERIFICATION, not just compilation.
# Tests passing != Feature working. You MUST verify with real data.
#
# Usage:
#   milestone_checkpoint.sh <milestone_name> [sprint_id]
#
# Example:
#   milestone_checkpoint.sh M1 M-TASK-HIERARCHY
#
# The script will:
# 1. Run tests and linting (basic hygiene)
# 2. Run milestone-specific verification if defined
# 3. Remind you to manually verify with real data
# 4. FAIL LOUDLY if verification commands fail

set -euo pipefail

MILESTONE_NAME="${1:-Unknown Milestone}"
SPRINT_ID="${2:-}"
SPRINT_DIR=".ailang/state/sprints"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "MILESTONE CHECKPOINT: $MILESTONE_NAME"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "⚠️  REMINDER: Tests passing ≠ Feature working"
echo "   You MUST verify with real data before marking complete!"
echo

FAILURES=0
WARNINGS=0

# 0. Find current sprint JSON
CURRENT_SPRINT=""
if [[ -d "$SPRINT_DIR" ]]; then
    # If SPRINT_ID was passed as argument, use that directly
    if [[ -n "$SPRINT_ID" && -f "$SPRINT_DIR/sprint_$SPRINT_ID.json" ]]; then
        CURRENT_SPRINT="$SPRINT_DIR/sprint_$SPRINT_ID.json"
    else
        # Find in_progress or ready sprint
        for f in "$SPRINT_DIR"/sprint_*.json; do
            if [[ -f "$f" ]]; then
                STATUS=$(grep -o '"status": "[^"]*"' "$f" 2>/dev/null | head -1 | cut -d'"' -f4 || echo "")
                if [[ "$STATUS" == "in_progress" || "$STATUS" == "ready" ]]; then
                    CURRENT_SPRINT="$f"
                    SPRINT_ID=$(basename "$f" .json | sed 's/sprint_//')
                    break
                fi
            fi
        done
    fi
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

# 5. CRITICAL: Data Verification
echo "5/5 Running data verification..."
echo "    (Tests passing ≠ Feature working - verify real data!)"
echo

# M-TASK-HIERARCHY specific verification
if [[ "$SPRINT_ID" == "M-TASK-HIERARCHY" ]] || [[ -n "$CURRENT_SPRINT" && "$CURRENT_SPRINT" == *"M-TASK-HIERARCHY"* ]]; then
    echo "  📊 M-TASK-HIERARCHY Sprint Verification"
    echo

    case "$MILESTONE_NAME" in
        M1|M1:*|*Entity*Sync*)
            echo "  Verifying M1: Entity Sync..."

            # Check if observatory_sync.go exists
            if [[ -f "internal/coordinator/observatory_sync.go" ]]; then
                echo "    ✓ observatory_sync.go exists"
            else
                echo "    ✗ FAIL: observatory_sync.go not found!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check if there are tasks in observatory DB
            if command -v sqlite3 &> /dev/null; then
                TASK_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM tasks" 2>/dev/null || echo "0")
                if [[ "$TASK_COUNT" -gt 0 ]]; then
                    echo "    ✓ Observatory has $TASK_COUNT task(s)"
                    SAMPLE_TASK=$(sqlite3 ~/.ailang/state/observatory.db "SELECT id FROM tasks LIMIT 1" 2>/dev/null || echo "")
                    if [[ -n "$SAMPLE_TASK" && "$SAMPLE_TASK" != "" ]]; then
                        echo "    ✓ Sample task ID: $SAMPLE_TASK (non-empty)"
                    else
                        echo "    ✗ FAIL: Task IDs are empty! Sync not working."
                        FAILURES=$((FAILURES + 1))
                    fi
                else
                    echo "    ⚠️  WARNING: No tasks in Observatory DB"
                    echo "       Run a coordinator task first to verify sync"
                    WARNINGS=$((WARNINGS + 1))
                fi
            else
                echo "    ⚠️  sqlite3 not available - manual verification required"
                WARNINGS=$((WARNINGS + 1))
            fi
            ;;

        M2|M2:*|*Context*Propagation*)
            echo "  Verifying M2: Context Propagation..."

            # Check if OTEL_RESOURCE_ATTRIBUTES is set in executor code
            if grep -q "OTEL_RESOURCE_ATTRIBUTES" internal/executor/claude/claude.go 2>/dev/null; then
                echo "    ✓ OTEL_RESOURCE_ATTRIBUTES found in claude executor"
            else
                echo "    ✗ FAIL: OTEL_RESOURCE_ATTRIBUTES not in claude executor!"
                echo "       Context is NOT being propagated to executors."
                FAILURES=$((FAILURES + 1))
            fi

            # Check if ailang.task_id is being added to attributes
            if grep -q "ailang.task_id" internal/executor/claude/claude.go 2>/dev/null; then
                echo "    ✓ ailang.task_id attribute found in claude executor"
            else
                echo "    ✗ FAIL: ailang.task_id not added to OTEL attributes!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check for ailang.task_id in span attributes
            if command -v sqlite3 &> /dev/null; then
                SPANS_WITH_TASK=$(sqlite3 ~/.ailang/state/observatory.db \
                    "SELECT COUNT(*) FROM spans WHERE resource_attributes LIKE '%ailang.task_id%'" 2>/dev/null || echo "0")
                if [[ "$SPANS_WITH_TASK" -gt 0 ]]; then
                    echo "    ✓ Found $SPANS_WITH_TASK spans with ailang.task_id"
                else
                    echo "    ⚠️  WARNING: No spans have ailang.task_id attribute yet"
                    echo "       Run a task and check spans after execution"
                    WARNINGS=$((WARNINGS + 1))
                fi
            fi
            ;;

        M3|M3:*|*OTLP*|*Extraction*)
            echo "  Verifying M3: OTLP Extraction..."

            # Check if OTLP receiver code extracts ailang.task_id
            if grep -q 'extractString.*ailang.task_id' internal/observatory/otlp_receiver.go 2>/dev/null; then
                echo "    ✓ OTLP receiver extracts ailang.task_id from resource attrs"
            else
                echo "    ✗ FAIL: OTLP receiver doesn't extract ailang.task_id!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check if Span struct has TaskID field that gets set
            if grep -q 'TaskID.*taskID' internal/observatory/otlp_receiver.go 2>/dev/null; then
                echo "    ✓ TaskID is set on spans"
            else
                echo "    ✗ FAIL: TaskID not being set on spans"
                FAILURES=$((FAILURES + 1))
            fi

            # Check database for linked spans (informational)
            if command -v sqlite3 &> /dev/null; then
                LINKED_SPANS=$(sqlite3 ~/.ailang/state/observatory.db \
                    "SELECT COUNT(*) FROM spans WHERE task_id IS NOT NULL AND task_id != ''" 2>/dev/null || echo "0")
                if [[ "$LINKED_SPANS" -gt 0 ]]; then
                    echo "    ✓ Found $LINKED_SPANS spans already linked to tasks"
                else
                    echo "    ⚠️  WARNING: No spans linked yet (run a task to generate data)"
                    WARNINGS=$((WARNINGS + 1))
                fi
            fi
            ;;

        M4|M4:*|*Aggregation*)
            echo "  Verifying M4: Aggregations..."

            # Check if aggregation update code exists in store
            if grep -q 'updateTaskAggregatesFromSpan' internal/observatory/store.go 2>/dev/null; then
                echo "    ✓ updateTaskAggregatesFromSpan function exists"
            else
                echo "    ✗ FAIL: updateTaskAggregatesFromSpan not found!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check if CreateSpan calls aggregation update
            if grep -q 'updateTaskAggregatesFromSpan.*span' internal/observatory/store.go 2>/dev/null; then
                echo "    ✓ CreateSpan calls aggregation update"
            else
                echo "    ✗ FAIL: CreateSpan doesn't call aggregation update!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check database for tasks with stats (informational)
            if command -v sqlite3 &> /dev/null; then
                TASKS_WITH_STATS=$(sqlite3 ~/.ailang/state/observatory.db \
                    "SELECT COUNT(*) FROM tasks WHERE span_count > 0 OR total_tokens_in > 0" 2>/dev/null || echo "0")
                if [[ "$TASKS_WITH_STATS" -gt 0 ]]; then
                    echo "    ✓ Found $TASKS_WITH_STATS tasks with aggregated stats"
                else
                    echo "    ⚠️  WARNING: No tasks have stats yet (run a task with linked spans)"
                    WARNINGS=$((WARNINGS + 1))
                fi
            fi
            ;;

        M5|M5:*|*Hierarchy*API*)
            echo "  Verifying M5: Hierarchy API..."

            # Check if server is running and hierarchy endpoint works
            if curl -s http://localhost:1957/health > /dev/null 2>&1; then
                # Get a task ID
                TASK_ID=$(curl -s http://localhost:1957/api/observatory/tasks 2>/dev/null | jq -r '.[0].id // empty' 2>/dev/null || echo "")
                if [[ -n "$TASK_ID" ]]; then
                    HIERARCHY=$(curl -s "http://localhost:1957/api/observatory/tasks/$TASK_ID/hierarchy" 2>/dev/null)
                    if [[ -n "$HIERARCHY" && "$HIERARCHY" != "null" && "$HIERARCHY" != "{}" ]]; then
                        echo "    ✓ Hierarchy API returns data for task $TASK_ID"
                    else
                        echo "    ✗ FAIL: Hierarchy API returns empty for task $TASK_ID"
                        FAILURES=$((FAILURES + 1))
                    fi
                else
                    echo "    ⚠️  WARNING: No tasks found to test hierarchy API"
                    WARNINGS=$((WARNINGS + 1))
                fi
            else
                echo "    ⚠️  WARNING: Server not running - start with 'ailang serve'"
                WARNINGS=$((WARNINGS + 1))
            fi
            ;;

        M6|M6:*|*UI*)
            echo "  Verifying M6: UI Hierarchy View..."

            # Check if TaskHierarchy component exists
            if [[ -f "ui/src/features/observatory/TaskHierarchy.tsx" ]]; then
                echo "    ✓ TaskHierarchy.tsx exists"
            else
                echo "    ✗ FAIL: TaskHierarchy.tsx not found!"
                FAILURES=$((FAILURES + 1))
            fi

            # Check if TaskHierarchy is imported and used in Observatory
            if grep -q "TaskHierarchy" ui/src/features/observatory/Observatory.tsx 2>/dev/null; then
                echo "    ✓ TaskHierarchy imported in Observatory.tsx"
            else
                echo "    ✗ FAIL: TaskHierarchy not imported in Observatory.tsx!"
                echo "       Component exists but is not connected to main view"
                FAILURES=$((FAILURES + 1))
            fi

            # Check for actual tab navigation (button or tab that switches to tasks view)
            if grep -qE "setActiveView.*['\"]tasks['\"]|activeView.*===.*['\"]tasks['\"]|onClick.*tasks" ui/src/features/observatory/Observatory.tsx 2>/dev/null; then
                echo "    ✓ Tasks tab navigation found"
            else
                echo "    ✗ FAIL: No Tasks tab navigation in Observatory!"
                echo "       Users cannot switch to TaskHierarchyView"
                FAILURES=$((FAILURES + 1))
            fi
            ;;

        M7|M7:*|*Backfill*)
            echo "  Verifying M7: Backfill Tool..."

            # Check if backfill command exists
            if [[ -f "cmd/ailang/observatory_backfill.go" ]]; then
                echo "    ✓ observatory_backfill.go exists"
            else
                echo "    ✗ FAIL: observatory_backfill.go not found!"
                FAILURES=$((FAILURES + 1))
            fi

            # Test --help to verify command is registered
            if ./bin/ailang observatory backfill --help > /dev/null 2>&1; then
                echo "    ✓ 'ailang observatory backfill' command works"
            else
                echo "    ✗ FAIL: 'ailang observatory backfill' command not working"
                FAILURES=$((FAILURES + 1))
            fi
            ;;

        *)
            echo "    No specific verification defined for milestone: $MILESTONE_NAME"
            echo "    ⚠️  MANUAL VERIFICATION REQUIRED"
            echo "       Check the sprint plan for verification commands"
            WARNINGS=$((WARNINGS + 1))
            ;;
    esac
else
    echo "  No sprint-specific verification defined."
    echo "  ⚠️  MANUAL VERIFICATION REQUIRED"
    echo "     Ensure you test with REAL DATA before marking complete!"
    WARNINGS=$((WARNINGS + 1))
fi
echo

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [[ $FAILURES -eq 0 ]]; then
    if [[ $WARNINGS -gt 0 ]]; then
        echo "⚠️  Checkpoint passed with $WARNINGS warning(s)"
        echo "   Review warnings above - some manual verification may be needed"
    else
        echo "✓ Milestone checkpoint PASSED!"
    fi
    echo

    # Sprint JSON reminder
    if [[ -n "$CURRENT_SPRINT" ]]; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "📋 SPRINT JSON UPDATE REQUIRED"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo
        echo "Sprint: $SPRINT_ID"
        echo "File:   $CURRENT_SPRINT"
        echo
        # Check for linked GitHub issues
        if command -v jq &> /dev/null; then
            GITHUB_ISSUES=$(jq -r '.github_issues // [] | map("#" + tostring) | join(", ")' "$CURRENT_SPRINT" 2>/dev/null || echo "")
            if [[ -n "$GITHUB_ISSUES" ]]; then
                echo "🔗 Linked GitHub issues: $GITHUB_ISSUES"
                echo "   Include 'Refs $GITHUB_ISSUES' in commit messages"
                echo
            fi
        fi
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
        if command -v jq &> /dev/null; then
            GITHUB_ISSUES=$(jq -r '.github_issues // [] | map("#" + tostring) | join(", ")' "$CURRENT_SPRINT" 2>/dev/null || echo "")
            if [[ -n "$GITHUB_ISSUES" ]]; then
                echo "   4. Include 'Refs $GITHUB_ISSUES' in commit message"
            fi
        fi
        echo
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    fi
    echo
    echo "Ready to proceed to next milestone."
    exit 0
else
    echo "✗ CHECKPOINT FAILED: $FAILURES verification(s) failed"
    echo
    echo "❌ DO NOT mark this milestone as complete!"
    echo "   Fix the issues above first."
    echo
    echo "Common fixes:"
    echo "  - Run actual tasks to generate real data"
    echo "  - Check that code changes are actually deployed"
    echo "  - Verify with manual commands (curl, sqlite3)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 1
fi
