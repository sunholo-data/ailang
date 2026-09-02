#!/usr/bin/env bash
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
SCAN_ROOT=${HOME_ISOLATION_ROOT:-$REPO_ROOT}
POSITIVE_FIXTURE=${HOME_ISOLATION_POSITIVE_FIXTURE:-$REPO_ROOT/scripts/testdata/home_isolation_gate_positive.go}
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT

broken() { printf 'INSTRUMENT BROKEN (%s): %s\n' "$1" "$2" >&2; exit 2; }

[ -d "$SCAN_ROOT" ] || broken G0 "missing scan root: $SCAN_ROOT"
[ -f "$POSITIVE_FIXTURE" ] || broken G1 "positive fixture missing: $POSITIVE_FIXTURE"

MATCH='t.Setenv("HOME"'
FIXTURE_COUNT=$(grep -F -c "$MATCH" "$POSITIVE_FIXTURE")
if [ "$FIXTURE_COUNT" -eq 0 ]; then
	broken G1 "positive fixture yielded zero matches"
fi

RAW_FILES="$TMP_DIR/raw-go-files"
FILES="$TMP_DIR/go-files"
find "$SCAN_ROOT" \
	\( -path "$SCAN_ROOT/.git" -o -path "$SCAN_ROOT/.snap" -o -path "$SCAN_ROOT/scripts/testdata" \) -prune -o \
	-type f -name '*.go' -print >"$RAW_FILES" || broken G0 "Go-file enumeration failed under $SCAN_ROOT"

# A linked worktree nested under the scan root has a .git file at its root.
# Filter those trees after enumeration without assuming their directory names.
: >"$FILES"
while IFS= read -r file; do
	parent=${file%/*}
	linked=0
	while [ "$parent" != "$SCAN_ROOT" ]; do
		if [ -f "$parent/.git" ]; then
			linked=1
			break
		fi
		parent=${parent%/*}
	done
	[ "$linked" -eq 1 ] || printf '%s\n' "$file" >>"$FILES"
done <"$RAW_FILES"

TEST_COUNT=$(awk '/_test\.go$/ {n++} END {print n+0}' "$FILES")
if [ "$TEST_COUNT" -eq 0 ]; then
	broken G0 "enumerated zero _test.go files under $SCAN_ROOT"
fi

VIOLATIONS="$TMP_DIR/violations"
: >"$VIOLATIONS"
while IFS= read -r file; do
	relative=${file#"$SCAN_ROOT"/}
	[ "$relative" = "internal/testutil/home.go" ] && continue
	grep -n -F "$MATCH" "$file" | sed "s|^|$relative:|" >>"$VIOLATIONS"
done <"$FILES"

printf 'home isolation scan: %s _test.go files; fixture matches: %s\n' "$TEST_COUNT" "$FIXTURE_COUNT"
printf 'RESIDUAL CLASSES (not covered): HOME held in a variable; os.Setenv; helpers in other packages; non-Go surfaces; dynamically assembled calls.\n'
if [ -s "$VIOLATIONS" ]; then
	printf 'bare t.Setenv("HOME" sites found outside internal/testutil/home.go:\n' >&2
	sed 's/^/    /' "$VIOLATIONS" >&2
	exit 1
fi

printf 'home isolation gate: OK\n'
exit 0
