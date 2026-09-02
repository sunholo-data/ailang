#!/usr/bin/env bash
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_home_isolation.sh"
FIXTURE="$REPO_ROOT/scripts/testdata/home_isolation_gate_positive.go"
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT
FAILED=0

pass() { printf '  PASS — %s\n' "$1"; }
fail() { FAILED=1; printf '  FAIL — %s\n' "$1"; }
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

printf 'package main\nfunc f(t interface{ Setenv(string, string) }, d string) { t.Setenv("HOME", d) }\n' >"$ROOT/cmd/bare_test.go"
run_gate "$ROOT" "$FIXTURE"
[ "$RC" -eq 1 ] && pass "bare HOME site exits 1" || fail "bare site rc=$RC: $OUT"

run_gate "$ROOT" "$TMP_DIR/missing-positive.go"
[ "$RC" -eq 2 ] && pass "missing fixture exits 2" || fail "missing fixture rc=$RC: $OUT"

if [ "$FAILED" -eq 0 ]; then
	printf 'home isolation gate self-test: OK (3 arms)\n'
	exit 0
fi
printf 'home isolation gate self-test: FAILED\n'
exit 1
