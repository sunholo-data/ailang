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

# Each gate runs ONCE: output is captured to a file and the failure tail is
# read from it. (This script used to re-run `make test`/`make lint` on failure
# just to capture the tail, and then ran `go test ./...` a third time for
# coverage: up to three whole-repo test runs per evaluation, 2026-10-02.)
EVAL_TMP=$(mktemp -d "${TMPDIR:-/tmp}/sprint-eval.XXXXXX")
KEEP_LOGS=false
trap '$KEEP_LOGS || rm -rf "$EVAL_TMP"' EXIT

# run_gate NAME CMD... : runs CMD once, echoes PASS/FAIL, leaves output in $EVAL_TMP/NAME.log
run_gate() {
    local name="$1"; shift
    if "$@" >"$EVAL_TMP/$name.log" 2>&1; then
        return 0
    fi
    return 1
}

# --- CI verdict (preferred) ---
# When the work under evaluation has a pull request, its CI has already run the
# full suite, lint and verify-examples on that head. ci_gate.sh waits for it,
# printing a line a minute so an executor's idle timeout never fires (the cloud
# evaluator was killed mid-`make test` on every run 09-29..10-02). Set
# EVAL_LOCAL_GATES=1 to force the local runs below.
GATE_BRANCH="${BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null)}"
GATE_SOURCE="local"
CI_RESULT=""
if [ "${EVAL_LOCAL_GATES:-0}" != "1" ] && [ -n "$GATE_BRANCH" ] && [ "$GATE_BRANCH" != "dev" ] && [ "$GATE_BRANCH" != "HEAD" ]; then
    echo ""
    echo "── CI Verdict ($GATE_BRANCH) ────────────────────────────────"
    CI_RESULT=$("$(dirname "$0")/ci_gate.sh" "$GATE_BRANCH" | tee /dev/stderr | grep '^CI_GATE=' | tail -1)
fi

# --- Test Suite ---
# EVAL_PACKAGES="./internal/foo/... ./cmd/bar" scopes the run to the sprint's
# packages (a fast pre-check). Unset = `make test`, the CI-equivalent run.
echo ""
echo "── Running Tests ──────────────────────────────────────────────"
TESTS_PASS=false
TESTS_OUTPUT=""
case "$CI_RESULT" in
    CI_GATE=pass*|CI_GATE=fail*|CI_GATE=pending*|CI_GATE=error*) GATE_SOURCE="ci" ;;
esac
if [ "$GATE_SOURCE" = "ci" ]; then
    echo "From CI: ${CI_RESULT#CI_GATE=}"
    case "$CI_RESULT" in
        CI_GATE=pass*) TESTS_PASS=true; echo "✅ Tests PASS (CI)" ;;
        CI_GATE=pending*)
            TESTS_OUTPUT="CI still running"
            echo "⏳ CI still running. Re-run: $(dirname "$0")/ci_gate.sh $GATE_BRANCH"
            echo "   until it prints CI_GATE=pass or CI_GATE=fail (each call waits up to 4 minutes)."
            echo "   Do NOT run make test yourself: CI is running the full suite on this head." ;;
        CI_GATE=error*)
            # Unreadable is not "no PR": falling back to local gates here is how
            # an auth failure once turned into a silent local run.
            TESTS_OUTPUT="CI verdict unreadable: ${CI_RESULT#CI_GATE=error }"
            echo "❌ Tests NOT CONFIRMED: $TESTS_OUTPUT" ;;
        *) TESTS_OUTPUT="${CI_RESULT#CI_GATE=fail }"; echo "❌ Tests FAIL (CI)"; echo "$TESTS_OUTPUT" ;;
    esac
elif [ -n "${EVAL_PACKAGES:-}" ]; then
    echo "Scope: EVAL_PACKAGES=$EVAL_PACKAGES"
    read -r -a EVAL_PKGS <<<"$EVAL_PACKAGES"   # bash 3.2-safe (no mapfile)
    TEST_CMD=(go test "${EVAL_PKGS[@]}" -count=1)
else
    TEST_CMD=(make test)
fi
if [ "$GATE_SOURCE" = "ci" ]; then
    :
elif run_gate tests "${TEST_CMD[@]}"; then
    TESTS_PASS=true
    echo "✅ Tests PASS"
else
    # The failing packages and tests, not the last 20 lines: `make test` is
    # verbose and alphabetical, so its tail is usually passing packages.
    TESTS_OUTPUT=$(grep -E '^(--- FAIL|FAIL[[:space:]]|panic:)' "$EVAL_TMP/tests.log" | head -40)
    KEEP_LOGS=true
    echo "❌ Tests FAIL"
    echo "$TESTS_OUTPUT"
    echo "Full log kept: $EVAL_TMP/tests.log"
fi

# --- Linting ---
echo ""
echo "── Running Lint ───────────────────────────────────────────────"
LINT_CLEAN=false
LINT_OUTPUT=""
if [ "$GATE_SOURCE" = "ci" ]; then
    # CI's lint job is one of the checks the verdict above covers.
    LINT_CLEAN=$TESTS_PASS
    echo "$([ "$LINT_CLEAN" = true ] && echo "✅ Lint CLEAN (CI)" || echo "❌ Lint not confirmed (CI red)")"
elif run_gate lint make lint; then
    LINT_CLEAN=true
    echo "✅ Lint CLEAN"
else
    LINT_OUTPUT=$(tail -20 "$EVAL_TMP/lint.log")
    echo "❌ Lint FAIL"
    echo "$LINT_OUTPUT"
fi

# --- File Sizes ---
echo ""
echo "── Checking File Sizes ──────────────────────────────────────"
FILE_SIZES_OK=false
FILE_SIZES_OUTPUT=""
if run_gate sizes make check-file-sizes; then
    FILE_SIZES_OK=true
    echo "✅ File sizes OK"
else
    FILE_SIZES_OUTPUT=$(tail -20 "$EVAL_TMP/sizes.log")
    echo "⚠️  File size warnings"
    echo "$FILE_SIZES_OUTPUT"
fi

# --- Coverage (opt-in) ---
# A whole-repo coverage run is a second full `go test ./...`; opt in with
# EVAL_COVERAGE=1 (it honours EVAL_PACKAGES when set).
echo ""
echo "── Collecting Coverage ──────────────────────────────────────"
COVERAGE_PCT="skipped (set EVAL_COVERAGE=1)"
if [ "${EVAL_COVERAGE:-0}" = "1" ] && command -v go >/dev/null 2>&1; then
    # shellcheck disable=SC2086
    COVERAGE_LINE=$(go test ${EVAL_PACKAGES:-./...} -coverprofile="$EVAL_TMP/coverage.out" 2>&1 | grep "^ok" | awk '{print $NF}' | grep -o '[0-9.]*%' | head -1 || echo "")
    COVERAGE_PCT="${COVERAGE_LINE:-unknown}"
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
    # Unfolded fragments (changelogs/unreleased/) count too: that is where new entries live now.
    if [ -n "$CHANGELOG_FILE" ] && grep -qi 'before.*after\|speedup\|benchmark.*result' "$CHANGELOG_FILE" changelogs/unreleased/*.md 2>/dev/null; then
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
  "gate_source": "$GATE_SOURCE",
  "ci_result": "$(printf '%s' "${CI_RESULT:-}" | sed 's/"/\\"/g')",
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
