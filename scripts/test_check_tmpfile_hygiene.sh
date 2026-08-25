#!/usr/bin/env bash
# Mutation-sensitive self-test for check_tmpfile_hygiene.sh (bash 3.2).
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_tmpfile_hygiene.sh"
TMP_DIR=$(mktemp -d)
FAILED=0
ARMS_RUN=0
ARMS_EXPECTED=9
OUT=""; RC=0

pass() { echo "  ok   — $1"; ARMS_RUN=$((ARMS_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; ARMS_RUN=$((ARMS_RUN + 1)); }
die() { echo "  FAIL — $1"; exit 1; }
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM

run_gate() {
	OUT=$(MK_ROOT="$1" MK_FILES_EXPECTED="$2" /bin/bash "$GATE" 2>&1)
	RC=$?
}

make_fixture() {
	_fixture=$1
	_line=$2
	mkdir -p "$_fixture"
	printf 'fixture:\n\t%s\n' "$_line" >"$_fixture/Makefile"
}

echo "tmpfile hygiene gate:"
[ -f "$GATE" ] || die "$GATE not found; refusing a vacuous green"

# (1) Real repository control.
run_gate "$REPO_ROOT" 12
RC_REAL=$RC
if [ "$RC" -eq 0 ]; then pass "real repository passes"; else fail "real repository expected rc=0, got rc=$RC: $OUT"; fi

# (2) Addition: a newly enumerated fixed path must be seen and named with its line.
FIXED="$TMP_DIR/fixed"
make_fixture "$FIXED" 'touch /tmp/newthing.txt'
run_gate "$FIXED" 1
RC_NEG=$RC
if [ "$RC" -eq 0 ]; then
	fail "fixed-path addition expected non-zero: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'Makefile:2'; then
	fail "fixed-path addition did not name Makefile:2: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF '/tmp/newthing.txt'; then
	fail "fixed-path addition did not name /tmp/newthing.txt: $OUT"
else
	pass "new fixed-path recipe was seen and named at Makefile:2"
fi

# (3) PID-suffixed paths are accepted.
PID_FIXTURE="$TMP_DIR/pid"
make_fixture "$PID_FIXTURE" 'touch /tmp/thing.$$$$'
run_gate "$PID_FIXTURE" 1
RC_POS=$RC
if [ "$RC" -eq 0 ]; then pass "PID-suffixed path is accepted"; else fail "PID-suffixed path expected rc=0, got rc=$RC: $OUT"; fi

# The negative and positive observations must be behaviorally distinct.
[ "$RC_NEG" -ne "$RC_POS" ] || die "fixed and uniquified arms did not differ (both rc=$RC_NEG)"
[ "$RC_NEG" -ne "$RC_REAL" ] || die "fixed addition and real control did not differ (both rc=$RC_NEG)"

# (4) mktemp XXXXXX paths, including TMPDIR fallback syntax, are accepted.
MKTEMP_FIXTURE="$TMP_DIR/mktemp"
make_fixture "$MKTEMP_FIXTURE" 'f=$$(mktemp "$${TMPDIR:-/tmp}/thing.XXXXXX")'
run_gate "$MKTEMP_FIXTURE" 1
if [ "$RC" -eq 0 ]; then pass "mktemp XXXXXX path is accepted"; else fail "mktemp path expected rc=0, got rc=$RC: $OUT"; fi

# (5) R1: empty make-file enumeration must fail as an instrument error.
EMPTY="$TMP_DIR/empty"
mkdir -p "$EMPTY"
run_gate "$EMPTY" 0
if [ "$RC" -ne 2 ]; then
	fail "R1 empty file set expected rc=2, got rc=$RC: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'instrument failure (R1)'; then
	fail "R1 empty file set did not name R1: $OUT"
else
	pass "R1 refuses an empty make-file set"
fi

# (6) R3: a readable file set with no /tmp/ known-positive must fail.
NO_TMP="$TMP_DIR/no-tmp"
make_fixture "$NO_TMP" 'echo clean'
run_gate "$NO_TMP" 1
if [ "$RC" -ne 2 ]; then
	fail "R3 no-/tmp control expected rc=2, got rc=$RC: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'instrument failure (R3)'; then
	fail "R3 no-/tmp control did not name R3: $OUT"
else
	pass "R3 refuses a file set with no /tmp/ occurrence"
fi

# (7) R4: an expected count above the actual enumeration must fail.
run_gate "$PID_FIXTURE" 2
if [ "$RC" -ne 2 ]; then
	fail "R4 count floor expected rc=2, got rc=$RC: $OUT"
elif ! printf '%s\n' "$OUT" | grep -qF 'instrument failure (R4)'; then
	fail "R4 count floor did not name R4: $OUT"
else
	pass "R4 refuses an enumerated count below MK_FILES_EXPECTED"
fi

# (8) A fixed path parked in a column-0 make VARIABLE and expanded in a recipe is the
# same defect, and a recipe-line-only scan cannot see it. Promoted into a committed arm
# from the iteration-273 controller drill (D5), where it escaped with rc=0.
VARFIX="$TMP_DIR/varfix"
mkdir -p "$VARFIX"
printf 'ITER_TMP := /tmp/via_variable.txt\nfixture:\n\t@echo hi > $(ITER_TMP)\n' >"$VARFIX/Makefile"
run_gate "$VARFIX" 1
if [ "$RC" -eq 0 ]; then
	fail "a fixed path defined in a make variable escaped the enumerator (rc=0)"
elif printf '%s' "$OUT" | grep -q 'via_variable.txt'; then
	pass "a fixed path defined in a make variable is caught and named"
else
	fail "refused but never named the variable-defined path: $OUT"
fi

# (9) Explicitly pin the self-test's own arm enumeration.
if [ "$ARMS_RUN" -ne 8 ]; then
	fail "arm-count precondition saw $ARMS_RUN of 8 substantive arms"
else
	pass "arm-count floor reached all substantive arms"
fi

if [ "$ARMS_RUN" -ne "$ARMS_EXPECTED" ]; then
	echo "  FAIL — $ARMS_RUN of $ARMS_EXPECTED arms ran; refusing a vacuous green"
	FAILED=1
fi
if [ "$FAILED" -eq 0 ]; then echo "tmpfile hygiene gate: OK ($ARMS_RUN arms)"; exit 0; fi
echo "tmpfile hygiene gate: FAILED"
exit 1
