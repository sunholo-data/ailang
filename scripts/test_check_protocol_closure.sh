#!/usr/bin/env bash
# Mutation-sensitive self-test for check_protocol_closure.sh (bash 3.2).
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_protocol_closure.sh"
PROTOCOL_INTRUDER="$REPO_ROOT/serveapi/protocol/zz_intruder.go"
TEST_INTRUDER="$REPO_ROOT/serveapi/protocol/zz_intruder_test.go"
SERVEAPI_INTRUDER="$REPO_ROOT/serveapi/zz_intruder.go"
FAILED=0
ARMS_RUN=0
ARMS_EXPECTED=5
OUT=""; RC=0

pass() { echo "  ok   — $1"; ARMS_RUN=$((ARMS_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; ARMS_RUN=$((ARMS_RUN + 1)); }
die() { echo "  FAIL — $1"; cleanup; exit 1; }

BEFORE_STATUS=$(cd "$REPO_ROOT" && git status --porcelain)
cleanup() {
	rm -f "$PROTOCOL_INTRUDER" "$TEST_INTRUDER" "$SERVEAPI_INTRUDER"
}
trap cleanup EXIT HUP INT TERM

assert_restored() {
	_now=$(cd "$REPO_ROOT" && git status --porcelain)
	[ "$_now" = "$BEFORE_STATUS" ] || die "$1 cleanup did not restore byte-identical git status"
}

run_gate() {
	OUT=$(cd "$REPO_ROOT" && /bin/bash "$GATE" "$@" 2>&1)
	RC=$?
}

extract_count() {
	printf '%s\n' "$1" | sed -n 's/^protocol non-stdlib count: \([0-9][0-9]*\)$/\1/p' | tail -1
}

echo "protocol closure gate:"
[ -f "$GATE" ] || die "$GATE not found; refusing a vacuous green"
[ ! -e "$PROTOCOL_INTRUDER" ] || die "$PROTOCOL_INTRUDER already exists"
[ ! -e "$TEST_INTRUDER" ] || die "$TEST_INTRUDER already exists"
[ ! -e "$SERVEAPI_INTRUDER" ] || die "$SERVEAPI_INTRUDER already exists"

# Clean control and baseline count used by arm (i).
run_gate
CLEAN_OUT="$OUT"; CLEAN_RC=$RC; CLEAN_COUNT=$(extract_count "$OUT")
if [ "$CLEAN_RC" -ne 0 ] || [ -z "$CLEAN_COUNT" ]; then
	die "clean control failed (rc=$CLEAN_RC, count=${CLEAN_COUNT:-missing}): $CLEAN_OUT"
fi

# (i) Addition: a real non-test import must be seen, named, and move 1 -> 2.
printf '%s\n' 'package protocol' '' 'import _ "github.com/google/uuid"' >"$PROTOCOL_INTRUDER"
[ -s "$PROTOCOL_INTRUDER" ] || die "intruder arm setup did not land"
grep -qF 'github.com/google/uuid' "$PROTOCOL_INTRUDER" || die "intruder arm setup lacks uuid"
run_gate
INTRUDER_COUNT=$(extract_count "$OUT")
if [ "$RC" -ne 1 ]; then
	fail "intruder arm expected rc=1, got rc=$RC: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'github.com/google/uuid'; then
	fail "intruder arm did not name github.com/google/uuid: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'protocol closure contains non-stdlib packages'; then
	fail "intruder arm was refused by the wrong branch: $OUT"
elif [ "$CLEAN_COUNT" -ne 1 ] || [ "$INTRUDER_COUNT" -ne 2 ]; then
	fail "intruder arm count did not move 1 -> 2 (observed $CLEAN_COUNT -> ${INTRUDER_COUNT:-missing})"
else
	pass "intruder named github.com/google/uuid; non-stdlib count moved $CLEAN_COUNT -> $INTRUDER_COUNT"
fi
rm -f "$PROTOCOL_INTRUDER"; assert_restored "intruder arm"

# (ii) Vacuity: probe each independently-neuterable floor branch.
VACUITY_FAILURE=""
run_gate ./definitely/not/a/package
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (protocol R1)' || VACUITY_FAILURE="R1 probe: rc=$RC output=$OUT"
printf '%s' "$OUT" | grep -q '✓' && VACUITY_FAILURE="R1 probe printed a checkmark: $OUT"

FAKE_DIR=$(mktemp -d)
printf '%s\n' '#!/bin/sh' 'exit 0' >"$FAKE_DIR/go"; chmod +x "$FAKE_DIR/go"
OUT=$(cd "$REPO_ROOT" && GO_BIN="$FAKE_DIR/go" /bin/bash "$GATE" 2>&1); RC=$?
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (protocol R2)' || VACUITY_FAILURE="R2 probe: rc=$RC output=$OUT"
rm -rf "$FAKE_DIR"

run_gate ./cmd/astdump
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (protocol R3)' || VACUITY_FAILURE="R3 probe: rc=$RC output=$OUT"

FAKE_DIR=$(mktemp -d)
printf '%s\n' '#!/bin/sh' 'printf "%s\n" github.com/sunholo-data/ailang/serveapi/protocol' >"$FAKE_DIR/go"; chmod +x "$FAKE_DIR/go"
OUT=$(cd "$REPO_ROOT" && GO_BIN="$FAKE_DIR/go" /bin/bash "$GATE" 2>&1); RC=$?
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (protocol R4)' || VACUITY_FAILURE="R4 probe: rc=$RC output=$OUT"
rm -rf "$FAKE_DIR"

run_gate ./serveapi/protocol ./definitely/not/a/package
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (serveapi R6)' || VACUITY_FAILURE="R6 probe: rc=$RC output=$OUT"

run_gate ./serveapi/protocol ./cmd/astdump
[ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -qF 'vacuous enumeration (serveapi R7)' || VACUITY_FAILURE="R7 probe: rc=$RC output=$OUT"

if [ -n "$VACUITY_FAILURE" ]; then fail "vacuity arm — $VACUITY_FAILURE"; else pass "vacuity probes R1/R2/R3/R4/R6/R7 refuse with rc=2 and no checkmark"; fi

# (iii) Restoration.
run_gate
if [ "$RC" -eq 0 ]; then pass "restoration arm clean gate passes"; else fail "restoration arm rc=$RC: $OUT"; fi

# (iv) Scope: test-only imports deliberately do not enter a consumer's link closure.
printf '%s\n' 'package protocol' '' 'import _ "github.com/google/uuid"' >"$TEST_INTRUDER"
[ -s "$TEST_INTRUDER" ] || die "scope arm setup did not land"
run_gate
if [ "$RC" -eq 0 ]; then
	pass "scope arm stays green: build closure models consumer links; test-only imports are never linked by a consumer"
else
	fail "scope arm expected rc=0, got rc=$RC: $OUT"
fi
rm -f "$TEST_INTRUDER"; assert_restored "scope arm"

# (v) Serveapi addition: uuid is outside the ten-root facade allowlist.
printf '%s\n' 'package serveapi' '' 'import _ "github.com/google/uuid"' >"$SERVEAPI_INTRUDER"
[ -s "$SERVEAPI_INTRUDER" ] || die "serveapi arm setup did not land"
run_gate
if [ "$RC" -ne 1 ]; then
	fail "serveapi arm expected rc=1, got rc=$RC: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'github.com/google/uuid'; then
	fail "serveapi arm did not name github.com/google/uuid: $OUT"
else
	pass "serveapi arm named disallowed root github.com/google/uuid"
fi
rm -f "$SERVEAPI_INTRUDER"; assert_restored "serveapi arm"

if [ "$ARMS_RUN" -ne "$ARMS_EXPECTED" ]; then
	echo "  FAIL — $ARMS_RUN of $ARMS_EXPECTED arms ran; refusing a vacuous green"
	FAILED=1
fi
if [ "$FAILED" -eq 0 ]; then echo "protocol closure gate: OK ($ARMS_RUN arms)"; exit 0; fi
echo "protocol closure gate: FAILED"
exit 1
