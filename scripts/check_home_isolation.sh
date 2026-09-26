#!/usr/bin/env bash
# Refuse any hand-rolled HOME override outside internal/testutil.
#
# os.UserHomeDir() reads a DIFFERENT variable per GOOS — USERPROFILE on windows,
# $home on plan9, HOME elsewhere — so a test that sets only HOME silently has no
# effect on windows: the runner's real profile resolves, the code under test
# never sees the fixture, and the arm fails for the PLATFORM rather than for the
# code. That defect reddened dev four times and had been fixed locally three
# times, each fix writing its own private helper, before this gate existed.
#
# The matcher is WHITESPACE-NORMALISED on purpose: `t.Setenv(\n\t"HOME",\n\tdir,\n)`
# is gofmt-canonical, so a line-oriented matcher would miss a form a contributor
# can reach by accident (found by the iteration-320 evaluator against the
# line-based first draft). It also covers `os.Setenv("HOME"`, which is how the
# instance that same evaluator found had evaded the sweep.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
SCAN_ROOT=${HOME_ISOLATION_ROOT:-$REPO_ROOT}
POSITIVE_FIXTURE=${HOME_ISOLATION_POSITIVE_FIXTURE:-$REPO_ROOT/scripts/testdata/home_isolation_gate_positive.go}
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT

broken() { printf 'INSTRUMENT BROKEN (%s): %s\n' "$1" "$2" >&2; exit 2; }

# The one implementation everything else must route through.
ALLOWLIST="internal/testutil/home.go"

# Files that set HOME for code which reads $HOME/$APPDATA DIRECTLY, per GOOS,
# rather than through os.UserHomeDir() — so the three-variable helper would be
# wrong there, not merely unnecessary. Each entry is asserted live below: an
# exemption for a file that no longer matches is a stale exemption, and stale
# exemptions are how an allowlist quietly becomes the rule.
EXEMPT="internal/loader/stdlib_resolver_test.go"

# Whitespace-normalised match: strip every space, tab and newline, then look for
# the call. `tr` cannot fail here in a way that matters, but its output is read
# through a count that is asserted numeric before use.
normalized_hits() {
	tr -d ' \t\r\n' <"$1" | grep -o -F 'Setenv("HOME"' | wc -l | tr -d ' '
}

[ -d "$SCAN_ROOT" ] || broken G0 "missing scan root: $SCAN_ROOT"
[ -f "$POSITIVE_FIXTURE" ] || broken G1 "positive fixture missing: $POSITIVE_FIXTURE"

# --- floor 1: the matcher must see a positive, in every shape it claims to cover.
FIXTURE_COUNT=$(normalized_hits "$POSITIVE_FIXTURE")
case "$FIXTURE_COUNT" in ''|*[!0-9]*) broken G1 "fixture count is not a number: '$FIXTURE_COUNT'" ;; esac
[ "$FIXTURE_COUNT" -ge 3 ] || broken G1 "positive fixture yielded $FIXTURE_COUNT matches; it must carry at least the bare, the os.Setenv and the multi-line shapes"

RAW_FILES="$TMP_DIR/raw-go-files"
FILES="$TMP_DIR/go-files"
find "$SCAN_ROOT" \
	\( -path "$SCAN_ROOT/.git" -o -path "$SCAN_ROOT/.snap" -o -path "$SCAN_ROOT/scripts/testdata" \) -prune -o \
	-type f -name '*.go' -print >"$RAW_FILES" || broken G0 "Go-file enumeration failed under $SCAN_ROOT"

# A linked worktree nested under the scan root has a .git FILE at its root.
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

# --- floor 2: an empty enumeration must FAIL LOUDLY, never print a checkmark.
TEST_COUNT=$(awk '/_test\.go$/ {n++} END {print n+0}' "$FILES")
case "$TEST_COUNT" in ''|*[!0-9]*) broken G0 "test-file count is not a number: '$TEST_COUNT'" ;; esac
[ "$TEST_COUNT" -gt 0 ] || broken G0 "enumerated zero _test.go files under $SCAN_ROOT"

VIOLATIONS="$TMP_DIR/violations"
: >"$VIOLATIONS"
EXEMPT_SEEN=0
while IFS= read -r file; do
	relative=${file#"$SCAN_ROOT"/}
	[ "$relative" = "$ALLOWLIST" ] && continue

	hits=$(normalized_hits "$file")
	case "$hits" in ''|*[!0-9]*) broken G2 "match count is not a number for $relative: '$hits'" ;; esac

	case " $EXEMPT " in
		*" $relative "*)
			# --- floor 3: a stale exemption is louder than a violation.
			if [ "$hits" -eq 0 ]; then
				broken G3 "exemption for $relative no longer matches; remove it"
			fi
			EXEMPT_SEEN=$((EXEMPT_SEEN + 1))
			continue
			;;
	esac

	[ "$hits" -eq 0 ] && continue
	# Report line numbers where the single-line form makes that possible, and
	# name the file otherwise — a whitespace-evading call has no one line.
	if grep -n -E '(t|os)\.Setenv\("HOME"' "$file" >"$TMP_DIR/lines" 2>/dev/null && [ -s "$TMP_DIR/lines" ]; then
		sed "s|^|$relative:|" "$TMP_DIR/lines" >>"$VIOLATIONS"
	else
		printf '%s: %s whitespace-spanning Setenv("HOME") call(s)\n' "$relative" "$hits" >>"$VIOLATIONS"
	fi
done <"$FILES"

EXPECTED_EXEMPT=$(printf '%s\n' $EXEMPT | awk 'NF {n++} END {print n+0}')
if [ "$SCAN_ROOT" = "$REPO_ROOT" ] && [ "$EXEMPT_SEEN" -ne "$EXPECTED_EXEMPT" ]; then
	broken G3 "saw $EXEMPT_SEEN of $EXPECTED_EXEMPT exempt files; an exemption names a path that no longer exists"
fi

printf 'home isolation scan: %s _test.go files; fixture matches: %s; live exemptions: %s\n' \
	"$TEST_COUNT" "$FIXTURE_COUNT" "$EXEMPT_SEEN"
printf 'RESIDUAL CLASSES (not covered): the variable name held in a variable or constant; a helper in another package that itself sets HOME; os/exec Env slices; build-tagged files excluded from this GOOS; non-Go surfaces (shell, CI yaml). Whitespace- and newline-spanning calls ARE covered, and so is os.Setenv.\n'
if [ -s "$VIOLATIONS" ]; then
	printf 'hand-rolled HOME override outside %s (use testutil.SetHomeDir):\n' "$ALLOWLIST" >&2
	sed 's/^/    /' "$VIOLATIONS" >&2
	exit 1
fi

printf 'home isolation gate: OK\n'
exit 0
