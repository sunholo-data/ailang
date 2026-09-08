#!/usr/bin/env bash
# Mutation-sensitive self-test for the scoped auto-push ShellCheck gate.
set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd) || exit 1
W=$(mktemp -d "$ROOT/.tmp-iter327.shellcheck-autopush.XXXXXX") || exit 1
trap 'rm -rf "$W"' EXIT
trap 'rm -rf "$W"; exit 1' HUP INT TERM

pass=0
fail=0
ck() {
    if [ "$2" = "$3" ]; then
        echo "  PASS $1"
        pass=$((pass + 1))
    else
        echo "  FAIL $1 (got '$2' want '$3')"
        fail=$((fail + 1))
    fi
}

fixture() {
    name="$1"
    mkdir -p "$W/$name/make" "$W/$name/scripts/hooks" || exit 1
    cp "$ROOT/make/code-health.mk" "$W/$name/make/code-health.mk" || exit 1
    cp "$ROOT/scripts/hooks/push_dev_on_stop.sh" "$W/$name/scripts/hooks/push_dev_on_stop.sh" || exit 1
    cp "$ROOT/scripts/hooks/test_push_dev_on_stop.sh" "$W/$name/scripts/hooks/test_push_dev_on_stop.sh" || exit 1
    printf 'include make/code-health.mk\n' > "$W/$name/Makefile" || exit 1
}

fixture production
printf '\necho %s\n' "\$known_bad_expansion" >> "$W/production/scripts/hooks/push_dev_on_stop.sh"
out=$(make -s -C "$W/production" shellcheck-autopush 2>&1); rc=$?
ck "shellcheck-autopush known-bad production mutation" "$([ "$rc" -ne 0 ] && echo red):$(printf '%s' "$out" | grep -q 'push_dev_on_stop.sh' && echo named)" "red:named"

fixture harness
sed 's/cd local || exit 1/cd local/' "$W/harness/scripts/hooks/test_push_dev_on_stop.sh" > "$W/harness/scripts/hooks/test_push_dev_on_stop.sh.mut" || exit 1
mv "$W/harness/scripts/hooks/test_push_dev_on_stop.sh.mut" "$W/harness/scripts/hooks/test_push_dev_on_stop.sh" || exit 1
out=$(make -s -C "$W/harness" shellcheck-autopush 2>&1); rc=$?
ck "shellcheck-autopush known-bad harness mutation" "$([ "$rc" -ne 0 ] && echo red):$(printf '%s' "$out" | grep -q 'test_push_dev_on_stop.sh' && echo named):$(printf '%s' "$out" | grep -q 'SC2164' && echo SC2164)" "red:named:SC2164"

workflow_hits=$(grep -c 'run: make shellcheck-autopush' "$ROOT/.github/workflows/ci.yml" 2>/dev/null || true)
ck "workflow wiring assertion" "$workflow_hits" "1"

# `make ci` is a LOCAL aggregate that CI never invokes (ci.yml says so in its own words),
# so a self-test reaches CI only via its own step. Without these two rows, the anti-vacuity
# controls for the two auto-push gates could be silently unwired and every gate above would
# still pass — which is exactly what the iteration-327 evaluator found. gate_wiring_test.go
# cannot cover them: its classifier is prefixed on `check-`/`test-check-`.
selftest_hits=$(grep -c 'run: make test-shellcheck-autopush' "$ROOT/.github/workflows/ci.yml" 2>/dev/null || true)
ck "shellcheck self-test reaches CI as its own step" "$selftest_hits" "1"
fmt_selftest_hits=$(grep -c 'run: make test-fmt-check' "$ROOT/.github/workflows/ci.yml" 2>/dev/null || true)
ck "fmt-check self-test reaches CI as its own step" "$fmt_selftest_hits" "1"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ]
