#!/bin/bash
# Self-test for ci_gate.sh against a fake gh. bash 3.2-safe.
# Run: /bin/bash .claude/skills/sprint-evaluator/scripts/test_ci_gate.sh
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
GATE="$HERE/ci_gate.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/ci-gate-test.XXXXXX")
trap 'rm -rf "$TMP"' EXIT

# The fake answers `pr list` with $FAKE_PR_LINE ("<number> <age-seconds>", or
# empty for no PR) and `pr checks` with checks.N for the Nth call, falling back
# to the highest N written, so a test can script pending -> green.
cat >"$TMP/gh" <<'EOF'
#!/bin/bash
case "$1 $2" in
    "pr list") printf '%s\n' "${FAKE_PR_LINE:-}" | grep . ;;
    "pr checks")
        n=$(( $(cat "$FAKE_DIR/calls" 2>/dev/null || echo 0) + 1 ))
        echo "$n" >"$FAKE_DIR/calls"
        while [ "$n" -gt 0 ] && [ ! -f "$FAKE_DIR/checks.$n" ]; do n=$((n - 1)); done
        [ "$n" -gt 0 ] && cat "$FAKE_DIR/checks.$n"
        ;;
esac
exit 0
EOF
chmod +x "$TMP/gh"

pass=0; fail=0
run_case() { # name expected-prefix [env...] -- writes checks first
    local name="$1" want="$2"; shift 2
    local got
    got=$(env EVAL_GH="$TMP/gh" FAKE_DIR="$TMP" EVAL_CI_POLL_SEC=0 "$@" /bin/bash "$GATE" coordinator/task-abc | grep '^CI_GATE=' | tail -1)
    case "$got" in
        "$want"*) echo "  PASS: $name"; pass=$((pass + 1)) ;;
        *) echo "  FAIL: $name: want '${want}...', got '${got}'"; fail=$((fail + 1)) ;;
    esac
    rm -f "$TMP"/checks.* "$TMP/calls"
}
checks() { # N line...
    local n="$1"; shift
    printf '%s\n' "$@" >"$TMP/checks.$n"
}

checks 1 "pass test" "pass lint"
run_case "no pull request falls back to local gates" "CI_GATE=none" FAKE_PR_LINE=

checks 1 "pass test" "pass lint" "skipping deploy"
run_case "all finished checks green passes" "CI_GATE=pass" FAKE_PR_LINE="1701 900"

checks 1 "pass lint" "fail test" "cancel windows"
run_case "a failed or cancelled check fails, naming it" "CI_GATE=fail PR #1701: failed checks: test;windows" FAKE_PR_LINE="1701 900"

checks 1 "skipping test" "skipping lint"
run_case "every check skipped fails (conflicting PR)" "CI_GATE=fail PR #1701: every check was skipped" FAKE_PR_LINE="1701 900"

checks 1 "pass lint" "pending test"
run_case "still running at the per-call deadline reports pending, not pass" "CI_GATE=pending" FAKE_PR_LINE="1701 900" EVAL_CI_WAIT_MIN=0

checks 1 "pending lint" "pending test"
checks 3 "pass lint" "pass test"
run_case "pending then green within one call passes" "CI_GATE=pass" FAKE_PR_LINE="1701 900" EVAL_CI_WAIT_MIN=1

: >"$TMP/checks.1"
run_case "no checks on a fresh PR is pending" "CI_GATE=pending" FAKE_PR_LINE="1701 30" EVAL_CI_WAIT_MIN=0

: >"$TMP/checks.1"
run_case "no checks on a PR older than the no-checks window falls back" "CI_GATE=none" FAKE_PR_LINE="1701 3600"

checks 1 "pass test"
run_case "an unparseable PR age is treated as zero, not a crash" "CI_GATE=pass" FAKE_PR_LINE="1701 x"

echo ""
echo "==== $pass passed, $fail failed ===="
[ "$fail" -eq 0 ]
