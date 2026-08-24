#!/usr/bin/env bash
# Guard the BUILD closure of the public serveapi packages.
#
# This deliberately measures what a consumer LINKS. Test-only imports are out of
# scope by design: they are not part of a consumer's build closure.
# Both arms require a known-positive and at least one stdlib package. The
# serveapi module-root enumeration is floored separately from its dependency
# enumeration because those are two distinct go list calls.
set -uo pipefail

PROTOCOL_PACKAGE="${1:-./serveapi/protocol}"
SERVEAPI_PACKAGE="${2:-./serveapi}"
PROTOCOL_SELF="github.com/sunholo-data/ailang/serveapi/protocol"
SERVEAPI_SELF="github.com/sunholo-data/ailang/serveapi"
GO_BIN="${GO_BIN:-go}"
RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'

vacuous() {
	printf "%b✗ vacuous enumeration (%s): %s%b\n" "$RED" "$1" "$2" "$RESET"
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

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

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

printf 'protocol non-stdlib count: %s\n' "$PROTOCOL_NONSTDLIB"
# R4
if [ "$PROTOCOL_STDLIB" -eq 0 ]; then
	vacuous "protocol R4" "no stdlib package was enumerated"
	exit 2
fi
if [ -s "$PROTOCOL_VIOLATORS" ]; then
	printf "%b✗ protocol closure contains non-stdlib packages outside %s:%b\n" "$RED" "$PROTOCOL_SELF" "$RESET"
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

SERVEAPI_ROOTS="$TMP_DIR/serveapi.roots"
SERVEAPI_ROOTS_RAW="$TMP_DIR/serveapi.roots.raw"
SERVEAPI_ROOTS_ERR="$TMP_DIR/serveapi.roots.err"
SERVEAPI_VIOLATORS="$TMP_DIR/serveapi.violators"
"$GO_BIN" list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' "$SERVEAPI_PACKAGE" \
	>"$SERVEAPI_ROOTS_RAW" 2>"$SERVEAPI_ROOTS_ERR"
SERVEAPI_ROOTS_RC=$?
# R11
if [ "$SERVEAPI_ROOTS_RC" -ne 0 ]; then
	vacuous "serveapi R11" "go list module-root enumeration failed for $SERVEAPI_PACKAGE (rc=$SERVEAPI_ROOTS_RC)"
	sed 's/^/    /' "$SERVEAPI_ROOTS_ERR"
	exit 2
fi
sed '/^$/d' "$SERVEAPI_ROOTS_RAW" | sort -u >"$SERVEAPI_ROOTS"
if [ ! -s "$SERVEAPI_ROOTS" ]; then
	vacuous "serveapi R11" "module-root enumeration is empty"
	exit 2
fi
if ! grep -qxF 'github.com/sunholo-data/ailang' "$SERVEAPI_ROOTS"; then
	vacuous "serveapi R11" "known-positive github.com/sunholo-data/ailang is absent"
	exit 2
fi
: >"$SERVEAPI_VIOLATORS"
while IFS= read -r _root; do
	# R8
	case "$_root" in
		github.com/sunholo-data/ailang|\
		github.com/google/jsonschema-go|\
		github.com/modelcontextprotocol/go-sdk|\
		github.com/segmentio/asm|\
		github.com/segmentio/encoding|\
		github.com/yosida95/uritemplate/v3|\
		golang.org/x/oauth2|\
		golang.org/x/sync|\
		golang.org/x/sys|\
		golang.org/x/time) ;;
		*) printf '%s\n' "$_root" >>"$SERVEAPI_VIOLATORS" ;;
	esac
done <"$SERVEAPI_ROOTS"

if [ -s "$SERVEAPI_VIOLATORS" ]; then
	printf "%b✗ serveapi closure contains disallowed module roots:%b\n" "$RED" "$RESET"
	sed 's/^/    /' "$SERVEAPI_VIOLATORS"
	exit 1
fi

printf "%b✓ protocol build closure is stdlib-only (%s non-stdlib package); serveapi module roots are allowed%b\n" \
	"$GREEN" "$PROTOCOL_NONSTDLIB" "$RESET"
