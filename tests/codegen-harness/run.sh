#!/usr/bin/env bash
# Codegen harness (make test-codegen): AILANG -> Go through `ailang compile --emit-go`.
#
#   1. build   — every tests/codegen-harness/*.ail, compiled together, must emit
#                Go that builds.
#   2. parity  — every parity/*.ail exports summary() -> string; the generated
#                Go's summary() must print exactly what `ailang run --entry
#                summary` prints.
#   3. known failures — every case in known_failures/known_failures.tsv must
#                still fail, with its recorded message. A case that passes or
#                fails differently fails the harness, so the list cannot rot.
#
# Usage: tests/codegen-harness/run.sh [ailang-binary]   (default ./bin/ailang)
# Bash 3.2 compatible. Writes only under a temp dir (compile cache included).
set -u

AILANG=${1:-./bin/ailang}
case "$AILANG" in /*) ;; *) AILANG="$PWD/$AILANG" ;; esac
HARNESS="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HARNESS/../.." && pwd)"
cd "$ROOT" || exit 1
WORK="$(mktemp -d "${TMPDIR:-/tmp}/codegen-harness.XXXXXX")" || exit 1
trap 'rm -rf "$WORK"' EXIT
export AILANG_CACHE_DIR="$WORK/cache" AILANG_RELAX_MODULES=1
FAILED=0

say() { printf '%s\n' "$*"; }
fail() { say "  ✗ $*"; FAILED=1; }

# gomod <dir>: the generated tree builds as its own module.
gomod() { printf 'module harnessgen\n\ngo 1.21\n' > "$1/go.mod"; }

# go_summary <out-dir> <pkg> <go-prefix>: print what the generated summary()
# returns (or the panic), via a throwaway test in the generated package.
go_summary() {
	local out=$1 pkg=$2 prefix=$3
	gomod "$out"
	printf 'package %s\n\nimport (\n\t"fmt"\n\t"testing"\n)\n\nfunc TestHarnessSummary(t *testing.T) {\n\tfmt.Print("<<<" + %s__Summary() + ">>>")\n}\n' \
		"$pkg" "$prefix" > "$out/$pkg/harness_summary_test.go"
	(cd "$out" && go test -count=1 -v "./$pkg/" 2>&1)
}

# module_prefix <file>: the Go function prefix the emitter uses (last module
# path segment).
module_prefix() { sed -n 's/^module[[:space:]][[:space:]]*\([^[:space:]]*\).*/\1/p' "$1" | head -1 | sed 's#.*/##'; }

# parity <entry.ail> <out-dir> <pkg> [all .ail files]: compare evaluator and Go.
# Sets PARITY_EVAL, PARITY_GO, PARITY_LOG; returns 0 only when they match.
parity() {
	local entry=$1 out=$2 pkg=$3; shift 3
	PARITY_EVAL="$("$AILANG" run --entry summary "$entry" 2>/dev/null)" || PARITY_EVAL="<evaluator failed>"
	if ! "$AILANG" compile --emit-go --package-name "$pkg" --out "$out" "$@" > "$out.log" 2>&1; then
		PARITY_LOG="$(cat "$out.log")"; PARITY_GO="<emit failed>"; return 1
	fi
	PARITY_LOG="$(go_summary "$out" "$pkg" "$(module_prefix "$entry")")"
	PARITY_GO="$(printf '%s\n' "$PARITY_LOG" | awk '/<<</{p=1; sub(/.*<<</,"")} p{ if (sub(/>>>.*/,"")) {print; exit} print }')"
	[ "$PARITY_EVAL" = "$PARITY_GO" ]
}

START=$(date +%s)

say "1. build: tests/codegen-harness/*.ail"
if "$AILANG" compile -emit-go -out "$WORK/build" "$HARNESS"/*.ail > "$WORK/build.log" 2>&1 \
	&& gomod "$WORK/build" && (cd "$WORK/build" && go build ./... > "$WORK/gobuild.log" 2>&1); then
	say "  ✓ compile + go build"
else
	fail "build: $(grep -E 'codegen:|Error' "$WORK/build.log" "$WORK/gobuild.log" 2>/dev/null | head -5)"
fi

say "2. parity: evaluator vs generated Go"
for f in "$HARNESS"/parity/*.ail; do
	name=$(basename "$f" .ail)
	if parity "$f" "$WORK/parity_$name" "p_$name" "$f"; then
		say "  ✓ $name ($(printf '%s\n' "$PARITY_EVAL" | wc -l | tr -d ' ') lines identical)"
	else
		fail "$name: generated Go differs from the evaluator"
		diff <(printf '%s\n' "$PARITY_EVAL") <(printf '%s\n' "$PARITY_GO") | sed 's/^/      /' | head -20
		[ "$PARITY_GO" = "<emit failed>" ] && printf '%s\n' "$PARITY_LOG" | grep -E 'codegen:|Error' | head -5 | sed 's/^/      /'
	fi
done

say "3. known failures (known_failures/known_failures.tsv)"
TAB="$(printf '\t')"
while IFS="$TAB" read -r kcase mode issue expect; do
	cd "$ROOT" || exit 1
	case "$kcase" in ''|'#'*) continue ;; esac
	dir="$HARNESS/known_failures/$kcase"
	# A case compiles from its own directory, so a multi-module case imports
	# its siblings by bare module name (a module path cannot contain "-").
	cd "$dir" 2>/dev/null || { fail "$kcase ($issue): no directory $dir"; continue; }
	files=$(ls ./*.ail 2>/dev/null)
	if [ -z "$files" ]; then fail "$kcase ($issue): no .ail files in $dir"; continue; fi
	out="$WORK/kf_$kcase"
	# shellcheck disable=SC2086
	case "$mode" in
	build)
		if "$AILANG" compile --emit-go --package-name "k_$kcase" --out "$out" $files > "$out.log" 2>&1; then
			fail "$kcase ($issue) now BUILDS — the bug looks fixed: remove it from known_failures.tsv (and its directory), and add the program to the harness"
		elif grep -qF -- "$expect" "$out.log"; then
			say "  ✓ $kcase still fails as recorded ($issue)"
		else
			fail "$kcase ($issue) fails DIFFERENTLY — expected \"$expect\":"; grep -E 'codegen:|Error' "$out.log" | head -5 | sed 's/^/      /'
		fi ;;
	parity)
		entry=$(grep -l 'func summary()' $files | head -1)
		if parity "$entry" "$out" "k_$kcase" $files; then
			fail "$kcase ($issue) now MATCHES the evaluator — the bug looks fixed: move it to parity/ and remove it from known_failures.tsv"
		elif printf '%s\n%s\n' "$PARITY_GO" "$PARITY_LOG" | grep -qF -- "$expect"; then
			say "  ✓ $kcase still differs as recorded ($issue)"
		else
			fail "$kcase ($issue) fails DIFFERENTLY — expected \"$expect\"; evaluator: $PARITY_EVAL; Go: $PARITY_GO"
		fi ;;
	emit)
		# The generated Go builds and summary() may even match, but the emitted
		# source still contains the recorded defect (e.g. a wrapper that panics
		# only when a Go caller invokes it, which summary() never does).
		if ! "$AILANG" compile --emit-go --package-name "k_$kcase" --out "$out" $files > "$out.log" 2>&1; then
			fail "$kcase ($issue) no longer EMITS — expected emitted Go containing \"$expect\":"; grep -E 'codegen:|Error' "$out.log" | head -5 | sed 's/^/      /'
		elif grep -rqF -- "$expect" "$out"; then
			say "  ✓ $kcase still emits as recorded ($issue)"
		else
			fail "$kcase ($issue) no longer emits \"$expect\" — the bug looks fixed: remove it from known_failures.tsv (and its directory), and add a parity case calling the exported function"
		fi ;;
	*) fail "$kcase: unknown mode '$mode'" ;;
	esac
done < "$HARNESS/known_failures/known_failures.tsv"
cd "$ROOT" || exit 1

# Every case directory must be listed (an unlisted case is never run).
for d in "$HARNESS"/known_failures/*/; do
	c=$(basename "$d")
	grep -q "^$c$TAB" "$HARNESS/known_failures/known_failures.tsv" || fail "known_failures/$c is not listed in known_failures.tsv"
done

say "codegen harness: $(( $(date +%s) - START ))s"
if [ "$FAILED" -ne 0 ]; then say "✗ codegen harness FAILED"; exit 1; fi
say "✓ codegen harness passed"
