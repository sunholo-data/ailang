#!/usr/bin/env bash
# The home-isolation gate's own self-test. A guard is not a gate until something
# reds when you remove it, so every arm here asserts an exit code captured
# WITHOUT a pipe, and the three shapes the gate claims to cover each get their
# own arm — the multi-line one because the gate's first draft was line-oriented
# and missed a form gofmt actively produces.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_home_isolation.sh"
FIXTURE="$REPO_ROOT/scripts/testdata/home_isolation_gate_positive.go"
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT
FAILED=0
ARMS=0

pass() { ARMS=$((ARMS + 1)); printf '  PASS — %s\n' "$1"; }
fail() { FAILED=1; ARMS=$((ARMS + 1)); printf '  FAIL — %s\n' "$1"; }
make_root() {
	mkdir -p "$1/internal/testutil" "$1/cmd"
	printf 'package testutil\nfunc f(t interface{ Setenv(string, string) }, d string) { t.Setenv("HOME", d) }\n' >"$1/internal/testutil/home.go"
	printf 'package main\n' >"$1/cmd/clean_test.go"
}
run_gate() {
	HOME_ISOLATION_ROOT="$1" HOME_ISOLATION_POSITIVE_FIXTURE="$2" bash "$GATE" >"$TMP_DIR/out" 2>&1
	RC=$?
	OUT=$(cat "$TMP_DIR/out")
}

ROOT="$TMP_DIR/root"
make_root "$ROOT"
run_gate "$ROOT" "$FIXTURE"
[ "$RC" -eq 0 ] && pass "clean tree exits 0" || fail "clean tree rc=$RC: $OUT"

# The allowlisted implementation is the one file allowed to hold the call; the
# clean arm above already proves it is not flagged.

printf 'package main\nfunc f(t interface{ Setenv(string, string) }, d string) { t.Setenv("HOME", d) }\n' >"$ROOT/cmd/bare_test.go"
run_gate "$ROOT" "$FIXTURE"
[ "$RC" -eq 1 ] && pass "bare t.Setenv HOME site exits 1" || fail "bare site rc=$RC: $OUT"
rm -f "$ROOT/cmd/bare_test.go"

printf 'package main\nimport "os"\nfunc f(d string) { _ = os.Setenv("HOME", d) }\n' >"$ROOT/cmd/os_test.go"
run_gate "$ROOT" "$FIXTURE"
[ "$RC" -eq 1 ] && pass "os.Setenv HOME site exits 1" || fail "os.Setenv site rc=$RC: $OUT"
rm -f "$ROOT/cmd/os_test.go"

# gofmt-canonical and newline-spanning: the shape a line-oriented matcher misses.
printf 'package main\nfunc f(t interface{ Setenv(string, string) }, d string) {\n\tt.Setenv(\n\t\t"HOME",\n\t\td,\n\t)\n}\n' >"$ROOT/cmd/multiline_test.go"
run_gate "$ROOT" "$FIXTURE"
[ "$RC" -eq 1 ] && pass "newline-spanning call exits 1" || fail "multi-line site rc=$RC: $OUT"
rm -f "$ROOT/cmd/multiline_test.go"

run_gate "$ROOT" "$TMP_DIR/missing-positive.go"
[ "$RC" -eq 2 ] && pass "missing fixture exits 2" || fail "missing fixture rc=$RC: $OUT"

# A fixture that no longer carries all three shapes is an INSTRUMENT failure, not
# a pass: it would silently narrow what the gate can see.
printf 'package fixture\nfunc f(t interface{ Setenv(string, string) }, d string) { t.Setenv("HOME", d) }\n' >"$TMP_DIR/thin-positive.go"
run_gate "$ROOT" "$TMP_DIR/thin-positive.go"
[ "$RC" -eq 2 ] && pass "under-specified fixture exits 2" || fail "thin fixture rc=$RC: $OUT"

mkdir -p "$TMP_DIR/empty-root"
run_gate "$TMP_DIR/empty-root" "$FIXTURE"
[ "$RC" -eq 2 ] && pass "zero enumerated test files exits 2" || fail "empty root rc=$RC: $OUT"

run_gate "$TMP_DIR/no-such-root" "$FIXTURE"
[ "$RC" -eq 2 ] && pass "missing scan root exits 2" || fail "missing root rc=$RC: $OUT"

if [ "$FAILED" -eq 0 ]; then
	printf 'home isolation gate self-test: OK (%s arms)\n' "$ARMS"
	exit 0
fi
printf 'home isolation gate self-test: FAILED (%s arms)\n' "$ARMS"
exit 1
