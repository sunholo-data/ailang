#!/usr/bin/env bash
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
SCAN_ROOT=${GIT_EXEC_ROOT:-$REPO_ROOT}
BASELINE=${GIT_EXEC_BASELINE:-$REPO_ROOT/scripts/git_exec_baseline.txt}
POSITIVE_FIXTURE=${GIT_EXEC_POSITIVE_FIXTURE:-$REPO_ROOT/scripts/testdata/git_exec_gate_positive.txt}
TOOL=${GIT_EXEC_TOOL:-$REPO_ROOT/tools/check-git-exec}
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT

broken() { printf 'INSTRUMENT BROKEN (%s): %s\n' "$1" "$2" >&2; exit 2; }

if [ ! -f "$POSITIVE_FIXTURE" ]; then
	broken G1 "positive fixture missing: $POSITIVE_FIXTURE"
fi
mkdir -p "$TMP_DIR/fixture"
cp "$POSITIVE_FIXTURE" "$TMP_DIR/fixture/positive.go" || broken G2 "cannot copy positive fixture"
go run "$TOOL" --root "$TMP_DIR" fixture >"$TMP_DIR/fixture.out" 2>"$TMP_DIR/fixture.err" || broken G2 "fixture enumeration failed: $(cat "$TMP_DIR/fixture.err")"
FIXTURE_COUNT=$(awk '$1=="TOTAL" {print $2}' "$TMP_DIR/fixture.out")
if [ "${FIXTURE_COUNT:-0}" -eq 0 ]; then
	broken G2 "positive fixture yielded zero matches"
fi

[ -d "$SCAN_ROOT/cmd" ] || broken G0 "missing cmd/ under $SCAN_ROOT"
[ -d "$SCAN_ROOT/internal" ] || broken G0 "missing internal/ under $SCAN_ROOT"
[ -f "$BASELINE" ] || broken G0 "missing baseline: $BASELINE"
go run "$TOOL" --root "$SCAN_ROOT" cmd internal >"$TMP_DIR/ast.out" 2>"$TMP_DIR/ast.err" || broken G0 "tree enumeration failed: $(cat "$TMP_DIR/ast.err")"
AST_TOTAL=$(awk '$1=="TOTAL" {print $2}' "$TMP_DIR/ast.out")

FAILED=0
while read -r tag file n; do
	[ "$tag" = COUNT ] || continue
	base=$(awk -v f="$file" '$1==f {print $2}' "$BASELINE")
	if [ -z "$base" ]; then
		printf 'git exec site absent from baseline: %s (%s)\n' "$file" "$n" >&2; FAILED=1; continue
	fi
	if [ "$n" -gt "$base" ]; then
		printf 'git exec count increased: %s actual=%s baseline=%s\n' "$file" "$n" "$base" >&2; grep "^SITE $file:" "$TMP_DIR/ast.out" >&2; FAILED=1
	fi
	if [ "$n" -lt "$base" ]; then
		printf 'git exec count decreased: tighten the baseline for %s (actual=%s baseline=%s)\n' "$file" "$n" "$base" >&2; FAILED=1
	fi
done <"$TMP_DIR/ast.out"
while read -r file base; do
	[ -n "$file" ] || continue
	n=$(awk -v f="$file" '$1=="COUNT" && $2==f {print $3}' "$TMP_DIR/ast.out")
	if [ -z "$n" ] && [ "$base" -ne 0 ]; then
		printf 'git exec count decreased: tighten the baseline for %s (actual=0 baseline=%s)\n' "$file" "$base" >&2; FAILED=1
	fi
done <"$BASELINE"

# AST-based, for the same reason the exec.Command enumeration is (HID-6): a
# line-oriented matcher cannot see a gofmt-canonical multi-line call, nor an
# aliased os/exec import. Both evasions were demonstrated against the previous
# grep form by the iteration-298 evaluator and reproduced by the controller.
LOOKPATH_ALL=$(sed -n 's/^LOOKPATH //p' "$TMP_DIR/ast.out")
LOOKPATH_TOTAL=$(sed -n 's/^LOOKPATH_TOTAL //p' "$TMP_DIR/ast.out" | head -1)
case "$LOOKPATH_TOTAL" in ''|*[!0-9]*)
	printf 'INSTRUMENT BROKEN: enumerator emitted no LOOKPATH_TOTAL\n' >&2; exit 2 ;;
esac
LOOKPATH_OUTSIDE=$(printf '%s\n' "$LOOKPATH_ALL" | grep -v '^internal/gitexec/' | awk 'NF {n++} END {print n+0}')
if [ "$LOOKPATH_OUTSIDE" -gt 0 ]; then
	printf 'LookPath("git") outside internal/gitexec:\n%s\n' "$LOOKPATH_ALL" >&2; FAILED=1
fi
if [ "$LOOKPATH_TOTAL" -ne 1 ]; then
	printf 'expected exactly one LookPath("git"), found %s\n' "$LOOKPATH_TOTAL" >&2; FAILED=1
fi

RX_TOTAL=$(grep -R -h -E 'exec\.Command(Context)?\([^)"]*"git"' "$SCAN_ROOT/cmd" "$SCAN_ROOT/internal" --include='*.go' --exclude='*_test.go' | wc -l | tr -d ' ')
if [ "$AST_TOTAL" -ne "$RX_TOTAL" ]; then
	printf 'AST/regex disagreement on real tree: AST=%s regex=%s\n' "$AST_TOTAL" "$RX_TOTAL" >&2; FAILED=1
fi

printf 'git exec AST total: %s; regex total: %s; fixture total: %s\n' "$AST_TOTAL" "$RX_TOTAL" "$FIXTURE_COUNT"
printf 'Residual classes (not covered): command name via a variable or constant (dataflow); dot-imports of os/exec; shell strings; syscall.Exec; non-Go surfaces. Test files are excluded. Aliased os/exec imports and multi-line calls ARE covered: both arms resolve through the AST and the import declarations.\n'
[ "$FAILED" -eq 0 ] || exit 1
printf 'git exec gate: OK\n'
