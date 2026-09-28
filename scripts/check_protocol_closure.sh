#!/usr/bin/env bash
# Guard the BUILD closure of the public serveapi packages.
#
# This deliberately measures what a consumer LINKS. Test-only imports are out of
# scope by design: they are not part of a consumer's build closure.
# The closure is checked across a GOOS matrix because a platform-suffixed Go file
# is invisible to dependency enumeration performed for only one GOOS.
# DECLARED SCOPE: GOARCH is deliberately NOT varied. No GOARCH-suffixed file exists
# anywhere in the tree today, and CI runs only ubuntu and windows runners, so the
# GOOS axis is the one that can actually differ in practice. A future _arm64.go or
# _amd64.go under serveapi would NOT be seen by this gate.
# Every arm requires a known-positive and at least one stdlib package. Since
# ailang#885 the facade and the two executable subpackages under protocol/
# (hostcall, mcphttp) are checked by exact PACKAGE path: their non-stdlib closure
# may contain only named ailang packages, never another package of the module.
set -uo pipefail

PROTOCOL_PACKAGE="${1:-./serveapi/protocol}"
SERVEAPI_PACKAGE="${2:-./serveapi}"
PROTOCOL_SELF="github.com/sunholo-data/ailang/serveapi/protocol"
SERVEAPI_SELF="github.com/sunholo-data/ailang/serveapi"
MCPHTTP_PACKAGE="${3:-./serveapi/protocol/mcphttp}"
HOSTCALL_PACKAGE="${4:-./serveapi/protocol/hostcall}"
MCPHTTP_SELF="github.com/sunholo-data/ailang/serveapi/protocol/mcphttp"
HOSTCALL_SELF="github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
GO_BIN="${GO_BIN:-go}"
GOOS_MATRIX="${GOOS_MATRIX-linux darwin windows}"
RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'

vacuous() {
	printf "%b[GOOS=%s] ✗ vacuous enumeration (%s): %s%b\n" "$RED" "${CURRENT_GOOS:-none}" "$1" "$2" "$RESET"
}

# R9: This function is intentionally separate. The dot-rule says a package is
# non-stdlib exactly when its first path segment contains a dot.
is_nonstdlib() {
	_first=${1%%/*}
	case "$_first" in
		*.*) return 0 ;;
		*) return 1 ;;
	esac
}

list_deps() {
	_ld_package="$1"
	_ld_output="$2"
	_ld_error="$3"
	"$GO_BIN" list -deps "$_ld_package" >"$_ld_output" 2>"$_ld_error"
}

# check_exact_arm LABEL PACKAGE SELF "ALLOWED..." — the non-stdlib part of
# PACKAGE's build closure must be a subset of ALLOWED, must contain SELF (known
# positive) and must enumerate at least one stdlib package (anti-vacuity).
check_exact_arm() {
	_label="$1"; _pkg="$2"; _self="$3"; _allowed="$4"
	_deps="$TMP_DIR/$_label.deps"; _err="$TMP_DIR/$_label.err"; _bad="$TMP_DIR/$_label.violators"
	if ! list_deps "$_pkg" "$_deps" "$_err" || [ ! -s "$_deps" ]; then
		vacuous "$_label R14" "go list failed or dependency enumeration is empty for $_pkg"
		sed 's/^/    /' "$_err"
		return 2
	fi
	if ! grep -qxF "$_self" "$_deps"; then
		vacuous "$_label R14" "known-positive $_self is absent"
		return 2
	fi
	_std=0; : >"$_bad"
	while IFS= read -r _package; do
		if ! is_nonstdlib "$_package"; then _std=$((_std + 1)); continue; fi
		_ok=1
		for _a in $_allowed; do [ "$_package" = "$_a" ] && _ok=0; done
		[ "$_ok" -eq 0 ] || printf '%s\n' "$_package" >>"$_bad"
	done <"$_deps"
	if [ "$_std" -eq 0 ]; then
		vacuous "$_label R14" "no stdlib package was enumerated"
		return 2
	fi
	if [ -s "$_bad" ]; then
		printf "%b✗ [GOOS=%s] %s closure contains disallowed packages:%b\n" "$RED" "$CURRENT_GOOS" "$_label" "$RESET"
		sed 's/^/    /' "$_bad"
		return 1
	fi
	return 0
}

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# R12: The platform matrix is itself an enumeration and must not be vacuous.
# Both matrix loops rely on word splitting, so pathname expansion is disabled around
# them: an entry containing a glob character must stay a (bogus) GOOS token rather
# than expanding to filenames.
set -f
MATRIX_EXPECTED=0
for _matrix_goos in $GOOS_MATRIX; do
	MATRIX_EXPECTED=$((MATRIX_EXPECTED + 1))
done
if [ "$MATRIX_EXPECTED" -eq 0 ]; then
	CURRENT_GOOS=none
	vacuous "matrix R12" "GOOS_MATRIX is empty or whitespace-only"
	exit 2
fi

MATRIX_COMPLETED=0
for CURRENT_GOOS in $GOOS_MATRIX; do
	export GOOS="$CURRENT_GOOS"

PROTOCOL_DEPS="$TMP_DIR/protocol.deps"
PROTOCOL_ERR="$TMP_DIR/protocol.err"
list_deps "$PROTOCOL_PACKAGE" "$PROTOCOL_DEPS" "$PROTOCOL_ERR"
PROTOCOL_RC=$?
# R1
if [ "$PROTOCOL_RC" -ne 0 ]; then
	vacuous "protocol R1" "go list failed for $PROTOCOL_PACKAGE (rc=$PROTOCOL_RC)"
	sed 's/^/    /' "$PROTOCOL_ERR"
	exit 2
fi
# R2
if [ ! -s "$PROTOCOL_DEPS" ]; then
	vacuous "protocol R2" "dependency enumeration is empty"
	exit 2
fi
# R3
if ! grep -qxF "$PROTOCOL_SELF" "$PROTOCOL_DEPS"; then
	vacuous "protocol R3" "known-positive $PROTOCOL_SELF is absent"
	exit 2
fi

PROTOCOL_STDLIB=0
PROTOCOL_NONSTDLIB=0
PROTOCOL_VIOLATORS="$TMP_DIR/protocol.violators"
: >"$PROTOCOL_VIOLATORS"
while IFS= read -r _package; do
	if is_nonstdlib "$_package"; then
		PROTOCOL_NONSTDLIB=$((PROTOCOL_NONSTDLIB + 1))
		# R5
		case "$_package" in
			"$PROTOCOL_SELF"*) ;;
			*) printf '%s\n' "$_package" >>"$PROTOCOL_VIOLATORS" ;;
		esac
	else
		PROTOCOL_STDLIB=$((PROTOCOL_STDLIB + 1))
	fi
done <"$PROTOCOL_DEPS"

printf '[GOOS=%s] protocol non-stdlib count: %s\n' "$CURRENT_GOOS" "$PROTOCOL_NONSTDLIB"
# R4
if [ "$PROTOCOL_STDLIB" -eq 0 ]; then
	vacuous "protocol R4" "no stdlib package was enumerated"
	exit 2
fi
if [ -s "$PROTOCOL_VIOLATORS" ]; then
	printf "%b✗ [GOOS=%s] protocol closure contains non-stdlib packages outside %s:%b\n" "$RED" "$CURRENT_GOOS" "$PROTOCOL_SELF" "$RESET"
	sed 's/^/    /' "$PROTOCOL_VIOLATORS"
	exit 1
fi

SERVEAPI_DEPS="$TMP_DIR/serveapi.deps"
SERVEAPI_ERR="$TMP_DIR/serveapi.err"
list_deps "$SERVEAPI_PACKAGE" "$SERVEAPI_DEPS" "$SERVEAPI_ERR"
SERVEAPI_RC=$?
# R6
if [ "$SERVEAPI_RC" -ne 0 ] || [ ! -s "$SERVEAPI_DEPS" ]; then
	vacuous "serveapi R6" "go list failed or dependency enumeration is empty for $SERVEAPI_PACKAGE (rc=$SERVEAPI_RC)"
	sed 's/^/    /' "$SERVEAPI_ERR"
	exit 2
fi
# R7
if ! grep -qxF "$SERVEAPI_SELF" "$SERVEAPI_DEPS"; then
	vacuous "serveapi R7" "known-positive $SERVEAPI_SELF is absent"
	exit 2
fi

SERVEAPI_STDLIB=0
while IFS= read -r _package; do
	if ! is_nonstdlib "$_package"; then
		SERVEAPI_STDLIB=$((SERVEAPI_STDLIB + 1))
	fi
done <"$SERVEAPI_DEPS"
# R10
if [ "$SERVEAPI_STDLIB" -eq 0 ]; then
	vacuous "serveapi R10" "no stdlib package was enumerated"
	exit 2
fi

# R8 (#885): exact PACKAGE paths, not module roots. A module-root allowlist would
# admit any other ailang package, including stdlib-only ones, and the facade's
# closure is now meant to be exactly these ailang packages plus the stdlib.
SERVEAPI_VIOLATORS="$TMP_DIR/serveapi.violators"
: >"$SERVEAPI_VIOLATORS"
while IFS= read -r _package; do
	is_nonstdlib "$_package" || continue
	case "$_package" in
		"$SERVEAPI_SELF"|"$PROTOCOL_SELF"|"$HOSTCALL_SELF"|"$MCPHTTP_SELF") ;;
		*) printf '%s\n' "$_package" >>"$SERVEAPI_VIOLATORS" ;;
	esac
done <"$SERVEAPI_DEPS"
if [ -s "$SERVEAPI_VIOLATORS" ]; then
	printf "%b✗ [GOOS=%s] serveapi closure contains disallowed packages:%b\n" "$RED" "$CURRENT_GOOS" "$RESET"
	sed 's/^/    /' "$SERVEAPI_VIOLATORS"
	exit 1
fi

# R14 (#885): the two executable subpackages under protocol/ are exactly
# closed too, so a consumer admitting the serveapi/protocol prefix links nothing else.
check_exact_arm "mcphttp" "$MCPHTTP_PACKAGE" "$MCPHTTP_SELF" "$MCPHTTP_SELF $HOSTCALL_SELF $PROTOCOL_SELF" || exit $?
check_exact_arm "hostcall" "$HOSTCALL_PACKAGE" "$HOSTCALL_SELF" "$HOSTCALL_SELF $PROTOCOL_SELF" || exit $?

	MATRIX_COMPLETED=$((MATRIX_COMPLETED + 1))
done
set +f

# R13: Refuse success unless every enumerated matrix entry completed both arms.
# DECLARED SCOPE: no value of GOOS_MATRIX can reach this branch today — every loop
# iteration either exits non-zero or increments, so completed always equals expected
# (measured across empty, whitespace, single and multi-entry matrices). It is retained
# as a regression floor: inserting a `continue` into the loop makes it fire with
# rc=2 "completed 0 of 3". It is therefore NOT covered by a self-test arm, because no
# input can drive it; only an edit to this loop can.
if [ "$MATRIX_COMPLETED" -eq 0 ] || [ "$MATRIX_COMPLETED" -ne "$MATRIX_EXPECTED" ]; then
	CURRENT_GOOS=matrix
	vacuous "matrix R13" "completed $MATRIX_COMPLETED of $MATRIX_EXPECTED GOOS iterations"
	exit 2
fi

printf "%b✓ protocol, hostcall, mcphttp and serveapi build closures hold across GOOS matrix: %s%b\n" \
	"$GREEN" "$GOOS_MATRIX" "$RESET"
