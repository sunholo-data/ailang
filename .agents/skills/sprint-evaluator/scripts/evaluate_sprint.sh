#!/bin/bash
# evaluate_sprint.sh - Orchestrate automated quality checks for sprint evaluation
# Runs tests, lint, file size checks, and coverage collection
# Outputs structured JSON results for the evaluator skill to parse

set -e

SPRINT_ID="${1:-}"
BRANCH="${2:-}"

if [ -z "$SPRINT_ID" ]; then
    echo "Usage: $0 <sprint-id> [branch]"
    echo ""
    echo "Example: $0 M-CACHE coordinator/task-abc123"
    echo ""
    echo "Runs automated quality checks and outputs JSON results"
    exit 1
fi

SPRINT_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"

if [ ! -f "$SPRINT_FILE" ]; then
    echo "Error: Sprint file not found: $SPRINT_FILE"
    exit 1
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Sprint Evaluation: $SPRINT_ID"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Check sprint status
STATUS=$(jq -r '.status // "unknown"' "$SPRINT_FILE")
echo "Sprint status: $STATUS"

# If branch specified, check it exists
if [ -n "$BRANCH" ]; then
    if git rev-parse --verify "$BRANCH" >/dev/null 2>&1; then
        echo "Branch: $BRANCH (exists)"
    else
        echo "Warning: Branch $BRANCH not found locally"
    fi
fi

# --- Test Suite ---
echo ""
echo "── Running Tests ──────────────────────────────────────────────"
TESTS_PASS=false
TESTS_OUTPUT=""
if make test 2>&1; then
    TESTS_PASS=true
    echo "✅ Tests PASS"
else
    TESTS_OUTPUT=$(make test 2>&1 | tail -20)
    echo "❌ Tests FAIL"
fi

# --- Linting ---
echo ""
echo "── Running Lint ───────────────────────────────────────────────"
LINT_CLEAN=false
LINT_OUTPUT=""
if make lint 2>&1; then
    LINT_CLEAN=true
    echo "✅ Lint CLEAN"
else
    LINT_OUTPUT=$(make lint 2>&1 | tail -20)
    echo "❌ Lint FAIL"
fi

# --- File Sizes ---
echo ""
echo "── Checking File Sizes ──────────────────────────────────────"
FILE_SIZES_OK=false
FILE_SIZES_OUTPUT=""
if make check-file-sizes 2>&1; then
    FILE_SIZES_OK=true
    echo "✅ File sizes OK"
else
    FILE_SIZES_OUTPUT=$(make check-file-sizes 2>&1 | tail -20)
    echo "⚠️  File size warnings"
fi

# --- Coverage ---
echo ""
echo "── Collecting Coverage ──────────────────────────────────────"
COVERAGE_PCT="unknown"
if command -v go >/dev/null 2>&1; then
    COVERAGE_LINE=$(go test ./... -coverprofile=/tmp/coverage_eval.out 2>&1 | grep "^ok" | awk '{print $NF}' | grep -o '[0-9.]*%' | head -1 || echo "")
    if [ -n "$COVERAGE_LINE" ]; then
        COVERAGE_PCT="$COVERAGE_LINE"
    fi
fi
echo "Coverage: $COVERAGE_PCT"

# --- Performance Profiling (for perf sprints) ---
echo ""
echo "── Checking Performance Profiling ──────────────────────────────"
IS_PERF_SPRINT=false
HAS_PROFILE_DATA=false

# Detect performance sprint from design doc keywords
DESIGN_DOC=$(jq -r '.design_doc // ""' "$SPRINT_FILE" 2>/dev/null || echo "")
if [ -n "$DESIGN_DOC" ] && [ -f "$DESIGN_DOC" ]; then
    if grep -qi 'performance\|speedup\|bottleneck\|cpu.*profile\|benchmark.*result\|latency.*target' "$DESIGN_DOC"; then
        IS_PERF_SPRINT=true
    fi
fi
# Also check sprint ID for perf indicators
if echo "$SPRINT_ID" | grep -qi 'PERF'; then
    IS_PERF_SPRINT=true
fi

if [ "$IS_PERF_SPRINT" = true ]; then
    echo "⚡ Performance sprint detected"

    # Check for profile references in recent commits
    PROFILE_COMMITS=$(git log --oneline -20 --grep="profile\|benchmark.*result\|speedup\|cpu.*%" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$PROFILE_COMMITS" -gt 0 ]; then
        HAS_PROFILE_DATA=true
        echo "✅ Found $PROFILE_COMMITS commits with profiling/benchmark data"
    fi

    # Check for benchmark results in changelog
    CHANGELOG_FILE=$(ls changelogs/*current* 2>/dev/null | head -1)
    if [ -n "$CHANGELOG_FILE" ] && grep -qi 'before.*after\|speedup\|benchmark.*result' "$CHANGELOG_FILE" 2>/dev/null; then
        HAS_PROFILE_DATA=true
        echo "✅ Benchmark before/after results found in changelog"
    fi

    if [ "$HAS_PROFILE_DATA" = false ]; then
        echo "❌ HARD FAIL: Performance sprint with no profiling/benchmark data"
        echo "   Run: ailang run -cpuprofile /tmp/before.prof <benchmark_file>"
        echo "   Then: go tool pprof -top -cum /tmp/before.prof | head -20"
    fi
else
    echo "ℹ️  Not a performance sprint, skipping profiling check"
fi

# --- TODO/HACK/FIXME in new code ---
echo ""
echo "── Checking for TODO/HACK/FIXME ─────────────────────────────"
TODO_COUNT=0
if [ -n "$BRANCH" ] && git rev-parse --verify "$BRANCH" >/dev/null 2>&1; then
    TODO_COUNT=$(git diff dev..."$BRANCH" -- '*.go' | grep -c '^\+.*\(TODO\|HACK\|FIXME\)' 2>/dev/null || echo "0")
else
    TODO_COUNT=$(git diff --cached -- '*.go' | grep -c '^\+.*\(TODO\|HACK\|FIXME\)' 2>/dev/null || echo "0")
fi
echo "TODO/HACK/FIXME in new code: $TODO_COUNT"

# --- Summary ---
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Automated Check Summary"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Tests pass:     $([ "$TESTS_PASS" = true ] && echo "✅ YES" || echo "❌ NO (HARD FAIL)")"
echo "  Lint clean:     $([ "$LINT_CLEAN" = true ] && echo "✅ YES" || echo "⚠️  NO")"
echo "  File sizes OK:  $([ "$FILE_SIZES_OK" = true ] && echo "✅ YES" || echo "⚠️  NO")"
echo "  Coverage:       $COVERAGE_PCT"
echo "  TODO/HACK count: $TODO_COUNT"
if [ "$IS_PERF_SPRINT" = true ]; then
echo "  Perf profiling: $([ "$HAS_PROFILE_DATA" = true ] && echo "✅ YES" || echo "❌ NO (HARD FAIL)")"
fi
echo ""

# Output JSON for skill parsing
echo "--- EVALUATION_JSON_START ---"
cat <<EOF
{
  "sprint_id": "$SPRINT_ID",
  "branch": "$BRANCH",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "automated_checks": {
    "tests_pass": $TESTS_PASS,
    "lint_clean": $LINT_CLEAN,
    "file_sizes_ok": $FILE_SIZES_OK,
    "coverage_pct": "$COVERAGE_PCT",
    "todo_hack_count": $TODO_COUNT,
    "is_perf_sprint": $IS_PERF_SPRINT,
    "has_profile_data": $HAS_PROFILE_DATA
  },
  "hard_fails": {
    "tests_broken": $([ "$TESTS_PASS" = false ] && echo "true" || echo "false"),
    "perf_no_profile": $([ "$IS_PERF_SPRINT" = true ] && [ "$HAS_PROFILE_DATA" = false ] && echo "true" || echo "false")
  }
}
EOF
echo "--- EVALUATION_JSON_END ---"
