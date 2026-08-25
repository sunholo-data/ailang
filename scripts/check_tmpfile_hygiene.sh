#!/usr/bin/env bash
# Refuse fixed /tmp paths in Makefile recipes.
#
# SCOPE, stated rather than implied: this scans tab-led recipe lines and
# column-0 variable assignments in `Makefile` + `make/*.mk`. It does NOT see a
# fixed path reached via an `export FOO := ...` line, an included file outside
# `make/`, or a path built at runtime from string concatenation. Those are
# declared limitations, not oversights.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
MK_ROOT="${MK_ROOT:-$REPO_ROOT}"
# R4 floor: this repository currently enumerates Makefile plus 11 make/*.mk files.
MK_FILES_EXPECTED="${MK_FILES_EXPECTED:-12}"
RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'

instrument_failure() {
	printf "%b✗ tmpfile hygiene instrument failure (%s): %s%b\n" "$RED" "$1" "$2" "$RESET"
}

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
FILES="$TMP_DIR/files"
VIOLATIONS="$TMP_DIR/violations"
: >"$FILES"
: >"$VIOLATIONS"

if [ -e "$MK_ROOT/Makefile" ]; then
	printf '%s\n' "$MK_ROOT/Makefile" >>"$FILES"
fi
for _file in "$MK_ROOT"/make/*.mk; do
	[ -e "$_file" ] || continue
	printf '%s\n' "$_file" >>"$FILES"
done

FILE_COUNT=$(wc -l <"$FILES" | tr -d ' ')
# R1: the make-file enumeration must not be empty.
if [ "$FILE_COUNT" -eq 0 ]; then
	instrument_failure R1 "enumerated make-file set is empty under $MK_ROOT"
	exit 2
fi

# R4: pin the enumeration floor so a silently narrowed glob cannot pass.
case "$MK_FILES_EXPECTED" in
	''|*[!0-9]*)
		instrument_failure R4 "MK_FILES_EXPECTED must be a non-negative integer (got $MK_FILES_EXPECTED)"
		exit 2
		;;
esac
if [ "$FILE_COUNT" -lt "$MK_FILES_EXPECTED" ]; then
	instrument_failure R4 "enumerated $FILE_COUNT files; expected at least $MK_FILES_EXPECTED"
	exit 2
fi

TMP_OCCURRENCES=0
while IFS= read -r _file; do
	# R2: every member admitted by the enumerator must still exist and be readable.
	if [ ! -f "$_file" ] || [ ! -r "$_file" ]; then
		instrument_failure R2 "enumerated file is missing or unreadable: $_file"
		exit 2
	fi
	_relative=${_file#"$MK_ROOT"/}
	_file_violations="$TMP_DIR/file.violations"
	_file_count="$TMP_DIR/file.count"
	awk -v file="$_relative" -v count_file="$_file_count" '
		BEGIN { count = 0 }
		# Recipe lines (tab-led) AND column-0 make variable assignments: a fixed
		# path parked in a variable and expanded in a recipe is the same defect,
		# and a recipe-only scan cannot see it (measured, iteration 273 drill D5).
		/^\t/ || /^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*[:?+]?=/ {
			field_count = split($0, fields, /[[:space:]]+/)
			for (i = 1; i <= field_count; i++) {
				rest = fields[i]
				while (match(rest, /\/tmp(\/|\}\/)[^[:space:]";|&<>()]+/)) {
					token = substr(rest, RSTART, RLENGTH)
					count++
					if (index(fields[i], "$") == 0 && index(fields[i], "XXXXXX") == 0) {
						printf "%s:%d: %s\n", file, NR, token
					}
					rest = substr(rest, RSTART + RLENGTH)
				}
			}
		}
		END { print count > count_file }
	' "$_file" >"$_file_violations"
	AWK_RC=$?
	if [ "$AWK_RC" -ne 0 ] || [ ! -s "$_file_count" ]; then
		instrument_failure R2 "could not scan enumerated file $_file (rc=$AWK_RC)"
		exit 2
	fi
	_file_tmp_count=$(cat "$_file_count")
	TMP_OCCURRENCES=$((TMP_OCCURRENCES + _file_tmp_count))
	cat "$_file_violations" >>"$VIOLATIONS"
done <"$FILES"

# R3: the matcher must see a known-positive in the same files used for the verdict.
if [ "$TMP_OCCURRENCES" -eq 0 ]; then
	instrument_failure R3 "no /tmp/ occurrence was observed in the enumerated make recipes"
	exit 2
fi

if [ -s "$VIOLATIONS" ]; then
	printf "%b✗ fixed /tmp path(s) found in make recipes:%b\n" "$RED" "$RESET"
	sed 's/^/    /' "$VIOLATIONS"
	exit 1
fi

printf "%b✓ tmpfile hygiene holds across %s make files%b\n" "$GREEN" "$FILE_COUNT" "$RESET"
