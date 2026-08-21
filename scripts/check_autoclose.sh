#!/usr/bin/env bash
# Refuse GitHub auto-close phrases in records that ship documentation only.
# GitHub interprets commit messages and squash-merge PR title/body text, but not
# repository file contents. Keep this script compatible with bash 3.2.
set -eu

KEYWORD_PATTERN='(close[sd]?|fix(es|ed)?|resolve[sd]?)[[:space:]]*:?[[:space:]]*#[0-9]+'

COMMITS_RANGE=""
TEXT_FILE=""
FILES_FROM=""
INSTRUMENT_ERROR=0
ERROR_MESSAGE=""
RECORDS_SCANNED=0
VIOLATIONS=0

# Refusal branch R0: without private scratch space the enumerator cannot run.
if ! TMPDIR_AUTOCLOSE=$(mktemp -d "${TMPDIR:-/tmp}/check-autoclose.XXXXXX"); then
	echo "INSTRUMENT FAILURE: cannot create temporary workspace" >&2
	exit 2
fi
trap 'rm -rf "$TMPDIR_AUTOCLOSE"' EXIT HUP INT TERM

record_error() {
	if [ "$INSTRUMENT_ERROR" -eq 0 ]; then
		ERROR_MESSAGE="$1"
	fi
	INSTRUMENT_ERROR=1
}

usage_error() {
	record_error "$1
usage: scripts/check_autoclose.sh [--commits <git-range>] [--text-file <f> --files-from <g>]"
}

is_docs_path() {
	case "$1" in
		docs/*|design_docs/*|changelogs/*|CHANGELOG|CHANGELOG.*|README|README.*|.claude/*|.agents/*)
			return 0
			;;
		*)
			case "$1" in
				*/*) return 1 ;;
				*.md) return 0 ;;
				*) return 1 ;;
			esac
			;;
	esac
}

ships_code() {
	_files="$1"
	while IFS= read -r _path || [ -n "$_path" ]; do
		[ -z "$_path" ] && continue
		if ! is_docs_path "$_path"; then
			return 0
		fi
	done < "$_files"
	return 1
}

trailer_allows() {
	_text="$1"
	_issue="$2"
	grep -Eiq "^[[:space:]]*Autoclose-OK:[[:space:]]*#${_issue}([[:space:]]*)$" "$_text"
}

scan_record() {
	_label="$1"
	_text="$2"
	_files="$3"
	RECORDS_SCANNED=$((RECORDS_SCANNED + 1))

	if grep -Eq '^[[:space:]]*Autoclose-OK:[[:space:]]*$' "$_text"; then
		record_error "ERROR: $_label has malformed trailer 'Autoclose-OK:'; name an issue as Autoclose-OK: #N"
		return
	fi

	_hits="$TMPDIR_AUTOCLOSE/hits.$RECORDS_SCANNED"
	if ! grep -Ein "$KEYWORD_PATTERN" "$_text" > "$_hits"; then
		return
	fi
	if ships_code "$_files"; then
		return
	fi

	while IFS= read -r _hit || [ -n "$_hit" ]; do
		_phrases="$TMPDIR_AUTOCLOSE/phrases.$RECORDS_SCANNED"
		printf '%s\n' "$_hit" | grep -Eio "$KEYWORD_PATTERN" > "$_phrases"
		while IFS= read -r _phrase || [ -n "$_phrase" ]; do
			_issue=$(printf '%s\n' "$_phrase" | grep -Eo '#[0-9]+' | head -1 | tr -d '#')
			if trailer_allows "$_text" "$_issue"; then
				continue
			fi
			VIOLATIONS=$((VIOLATIONS + 1))
			printf 'VIOLATION: %s: issue #%s\n' "$_label" "$_issue"
			printf '  context: %s\n' "$_hit"
			printf '  matched: %s\n' "$_phrase"
		done < "$_phrases"
	done < "$_hits"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--commits)
			if [ "$#" -lt 2 ]; then usage_error "ERROR: --commits requires a git range"; break; fi
			if [ -n "$COMMITS_RANGE" ]; then usage_error "ERROR: --commits may be specified only once"; break; fi
			COMMITS_RANGE="$2"
			shift 2
			;;
		--text-file)
			if [ "$#" -lt 2 ]; then usage_error "ERROR: --text-file requires a path"; break; fi
			TEXT_FILE="$2"
			shift 2
			;;
		--files-from)
			if [ "$#" -lt 2 ]; then usage_error "ERROR: --files-from requires a path"; break; fi
			FILES_FROM="$2"
			shift 2
			;;
		*)
			usage_error "ERROR: unknown argument: $1"
			break
			;;
	esac
done

if [ -z "$COMMITS_RANGE" ] && [ -z "$TEXT_FILE" ] && [ -z "$FILES_FROM" ]; then
	usage_error "ERROR: select --commits and/or --text-file with --files-from"
fi
if { [ -n "$TEXT_FILE" ] && [ -z "$FILES_FROM" ]; } || { [ -z "$TEXT_FILE" ] && [ -n "$FILES_FROM" ]; }; then
	usage_error "ERROR: --text-file and --files-from must be provided together"
fi

if [ "$INSTRUMENT_ERROR" -eq 0 ] && [ -n "$COMMITS_RANGE" ]; then
	_commits="$TMPDIR_AUTOCLOSE/commits"
	if ! git rev-list "$COMMITS_RANGE" > "$_commits" 2> "$TMPDIR_AUTOCLOSE/git-error"; then
		record_error "INSTRUMENT FAILURE: cannot enumerate commit range '$COMMITS_RANGE': $(sed -n '1p' "$TMPDIR_AUTOCLOSE/git-error")"
	else
		while IFS= read -r _sha || [ -n "$_sha" ]; do
			[ -z "$_sha" ] && continue
			_message="$TMPDIR_AUTOCLOSE/message.$_sha"
			_files="$TMPDIR_AUTOCLOSE/files.$_sha"
			if ! git show -s --format=%B "$_sha" > "$_message" || ! git diff-tree --no-commit-id --name-only -r --root "$_sha" > "$_files"; then
				record_error "INSTRUMENT FAILURE: cannot read commit $_sha"
				break
			fi
			scan_record "$_sha" "$_message" "$_files"
		done < "$_commits"
	fi
fi

if [ "$INSTRUMENT_ERROR" -eq 0 ] && [ -n "$TEXT_FILE" ]; then
	if [ ! -f "$TEXT_FILE" ]; then
		record_error "INSTRUMENT FAILURE: text file not found: $TEXT_FILE"
	elif [ ! -f "$FILES_FROM" ]; then
		record_error "INSTRUMENT FAILURE: changed-file list not found: $FILES_FROM"
	elif [ ! -s "$FILES_FROM" ]; then
		record_error "INSTRUMENT FAILURE: no changed files enumerated from $FILES_FROM"
	else
		scan_record "PR title/body" "$TEXT_FILE" "$FILES_FROM"
	fi
fi

# Refusal branch R2: malformed input, enumeration/read failure, or usage error.
if [ "$INSTRUMENT_ERROR" -ne 0 ]; then
	printf '%s\n' "$ERROR_MESSAGE" >&2
	exit 2
fi

# Refusal branch R3: an empty commit range must never certify itself green.
if [ "$RECORDS_SCANNED" -eq 0 ]; then
	echo "INSTRUMENT FAILURE: no records enumerated" >&2
	exit 2
fi

# Refusal branch R1: docs-only text that GitHub would interpret as issue closure.
if [ "$VIOLATIONS" -ne 0 ]; then
	cat <<'EOF'

Remedy:
  - rephrase so the keyword does not precede the number:
      "fixes #676"  ->  "#676 is fixed by ..."  /  "the defect in #676"  /  "reported at #676"
  - or, if this docs commit really does close the issue, add a trailer:  Autoclose-OK: #676
EOF
	printf '✗ check-autoclose: %s records scanned, %s violations\n' "$RECORDS_SCANNED" "$VIOLATIONS"
	exit 1
fi

printf '✓ check-autoclose: %s records scanned, 0 violations\n' "$RECORDS_SCANNED"
