#!/bin/bash
# Self-test for every observable refusal in check_autoclose.sh. Bash 3.2 only.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_autoclose.sh"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/test-check-autoclose.XXXXXX") || exit 1
trap 'rm -rf "$WORK"' EXIT HUP INT TERM

FAILED=0
FIXTURES_RUN=0
FIXTURES_EXPECTED=16
OUT=""
RC=0

pass() { echo "  ok   — $1"; FIXTURES_RUN=$((FIXTURES_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; FIXTURES_RUN=$((FIXTURES_RUN + 1)); }

run_text() {
	_name="$1"
	_text="$2"
	_file="$3"
	_dir="$WORK/$_name"
	mkdir -p "$_dir"
	printf '%s\n' "$_text" > "$_dir/text"
	printf '%s\n' "$_file" > "$_dir/files"
	OUT=$("$GATE" --text-file "$_dir/text" --files-from "$_dir/files" 2>&1)
	RC=$?
}

expect_rc() {
	_name="$1"
	_want="$2"
	_fragment="$3"
	if [ "$RC" -ne "$_want" ]; then
		fail "$_name — expected rc=$_want, got rc=$RC"
	elif ! printf '%s\n' "$OUT" | grep -qF "$_fragment"; then
		fail "$_name — rc=$RC came from the wrong path (missing: $_fragment)"
	else
		pass "$_name (rc=$RC)"
	fi
}

echo "autoclose gate:"

run_text bad1 'the arena fixes #676 completely' 'design_docs/record.md'
expect_rc 'BAD-1 docs-only arena phrase' 1 'issue #676'

run_text bad2 'a "Fixes #612" would wrongly auto-close the follow-up' 'docs/record.md'
expect_rc 'BAD-2 self-referential warning' 1 'issue #612'

run_text bad3 'Fix #2 (gpt5-6-sol): new bounded-execution section' 'changelogs/v0.18-current.md'
expect_rc 'BAD-3 ordinal-looking issue phrase' 1 'issue #2'

run_text good1 'Fixes #694' 'internal/foo/bar.go'
expect_rc 'GOOD-1 code-shipping record' 0 '1 records scanned, 0 violations'

run_text good2 '#598 closed as a duplicate of #602' 'design_docs/record.md'
expect_rc 'GOOD-2 inverted closure wording' 0 '1 records scanned, 0 violations'

run_text good3 'reported at #676 and filed as #612' 'README.md'
expect_rc 'GOOD-3 neutral references' 0 '1 records scanned, 0 violations'

run_text good4 'fixes #676

Autoclose-OK: #676' 'docs/record.md'
expect_rc 'GOOD-4 numbered escape trailer' 0 '1 records scanned, 0 violations'

# This directly pins refusal R2 (instrument/usage failure), including its rc=2 contract.
run_text malformed 'Autoclose-OK:' 'docs/record.md'
expect_rc 'MALFORMED bare escape trailer' 2 'malformed trailer'

# A valid git repository with a deliberately empty range pins refusal R3. This is
# not a missing-repository error: enumeration succeeds and produces zero records.
EMPTY_REPO="$WORK/empty-range"
mkdir -p "$EMPTY_REPO"
(
	cd "$EMPTY_REPO" || exit 1
	git init -q
	git config user.email test@example.invalid
	git config user.name 'Autoclose Gate Test'
	printf '%s\n' seed > seed.txt
	git add seed.txt
	git commit -q -m seed
)
OUT=$(cd "$EMPTY_REPO" && "$GATE" --commits HEAD..HEAD 2>&1)
RC=$?
expect_rc 'VACUITY empty commit range' 2 'INSTRUMENT FAILURE: no records enumerated'

# A nonexistent TMPDIR deterministically pins refusal R0 before enumeration starts.
OUT=$(TMPDIR="$WORK/no-such-parent/child" "$GATE" --commits HEAD..HEAD 2>&1)
RC=$?
expect_rc 'TEMPFAIL unavailable scratch directory' 2 'INSTRUMENT FAILURE: cannot create temporary workspace'

# --- Instrument-integrity arms (iteration 241 evaluator, BLOCKING findings) ---------------
# The guards below all EXISTED, but no fixture killed them: neutering the text-file-exists
# check left the whole self-test green while a missing text file passed rc=0 CLEAN. In CI the
# text file is written by jq from the event payload, so a silent pass there is exactly the
# vacuous-pass class this gate exists to prevent. A guard is not a gate until something reds.

OUT=$("$GATE" --text-file "$WORK/definitely-absent-text" --files-from "$WORK/definitely-absent-files" 2>&1)
RC=$?
expect_rc 'MISSINGTEXT absent text file' 2 'INSTRUMENT FAILURE: text file not found'

mkdir -p "$WORK/missingfiles"
printf '%s\n' 'a record with no issue references' > "$WORK/missingfiles/text"
OUT=$("$GATE" --text-file "$WORK/missingfiles/text" --files-from "$WORK/definitely-absent-files" 2>&1)
RC=$?
expect_rc 'MISSINGFILES absent changed-file list' 2 'INSTRUMENT FAILURE: changed-file list not found'

OUT=$(cd "$EMPTY_REPO" && "$GATE" --commits HEAD..HEAD --not-a-real-flag 2>&1)
RC=$?
expect_rc 'UNKNOWNARG unrecognised argument' 2 'unknown argument'

# An EXISTING but empty changed-file list is not an instrument failure: it is a record that
# ships no code, which is precisely the hazard condition. It must be SCANNED, not errored --
# otherwise an empty-diff PR whose body closes an issue escapes through an rc=2 that reads
# like a broken instrument.
mkdir -p "$WORK/emptyfiles"
printf '%s\n' 'this closes #4242 with no code at all' > "$WORK/emptyfiles/text"
: > "$WORK/emptyfiles/files"
OUT=$("$GATE" --text-file "$WORK/emptyfiles/text" --files-from "$WORK/emptyfiles/files" 2>&1)
RC=$?
expect_rc 'EMPTYFILES no-code record is scanned, not errored' 1 'issue #4242'

printf '%s\n' 'a rebase-only PR that references nothing' > "$WORK/emptyfiles/text2"
OUT=$("$GATE" --text-file "$WORK/emptyfiles/text2" --files-from "$WORK/emptyfiles/files" 2>&1)
RC=$?
expect_rc 'EMPTYFILES clean no-code record passes' 0 '1 records scanned, 0 violations'

# A real 2-parent merge commit. `git diff-tree --root` without `-m` reports ZERO files for a
# merge, so a merge whose own message carries a closing keyword would be misread as shipping no
# code and refused. Measured on origin/dev: 60 merge commits exist, and the as-written form saw
# 0 files where `-m` saw 5. This arm pins the `-m`.
MERGE_REPO="$WORK/merge-repo"
mkdir -p "$MERGE_REPO"
(
	cd "$MERGE_REPO" || exit 1
	git init -q
	git config user.email test@example.invalid
	git config user.name 'Autoclose Gate Test'
	git symbolic-ref HEAD refs/heads/main
	mkdir -p internal/foo
	printf '%s\n' 'package foo' > internal/foo/bar.go
	git add internal/foo/bar.go
	git commit -q -m base
	git checkout -q -b side
	printf '%s\n' 'package foo // side' > internal/foo/bar.go
	git commit -q -am side-change
	git checkout -q main
	printf '%s\n' 'main note' > MAINLINE.txt
	git add MAINLINE.txt
	git commit -q -m main-change
	git merge --no-ff -q -m 'merge side: fixes #4321' side 2>/dev/null || \
		git merge --no-ff -q --strategy-option=theirs -m 'merge side: fixes #4321' side
)
OUT=$(cd "$MERGE_REPO" && "$GATE" --commits 'HEAD^..HEAD' 2>&1)
RC=$?
expect_rc 'MERGECOMMIT ships code via -m, not misread as docs-only' 0 '0 violations'

if [ "$FIXTURES_RUN" -ne "$FIXTURES_EXPECTED" ]; then
	echo "  FAIL — $FIXTURES_RUN of $FIXTURES_EXPECTED fixtures ran; refusing vacuous green"
	FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
	echo "autoclose gate: FAILED ($FIXTURES_RUN fixtures)"
	exit 1
fi

echo "autoclose gate: OK ($FIXTURES_RUN fixtures)"
