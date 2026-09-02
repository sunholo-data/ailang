#!/bin/bash
# Covers tools/ and scripts/ prefixes only. docs/scripts/... and generated paths
# such as .stdlib-golden/option.sh are deliberately out of scope.
set -u

SCAN_ROOT="${REFERENCED_PATHS_SCAN_ROOT:-$(pwd)}"
REPO_ROOT="${REFERENCED_PATHS_REPO_ROOT:-$SCAN_ROOT}"
MINIMUM=20

usage() {
	echo "usage: $0 [--scan-root DIR] [--repo-root DIR]" >&2
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--scan-root)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			SCAN_ROOT="$2"
			shift 2
			;;
		--repo-root)
			[ "$#" -ge 2 ] || { usage; exit 2; }
			REPO_ROOT="$2"
			shift 2
			;;
		*)
			usage
			exit 2
			;;
	esac
done

WORK=$(mktemp -d "${TMPDIR:-/tmp}/check-referenced-paths.XXXXXX") || {
	echo "instrument failure: cannot create temporary workspace" >&2
	exit 2
}
trap 'rm -rf "$WORK"' EXIT HUP INT TERM
INPUTS="$WORK/inputs"
PATHS="$WORK/paths"
: > "$INPUTS"

[ -f "$SCAN_ROOT/Makefile" ] && printf '%s\n' "$SCAN_ROOT/Makefile" >> "$INPUTS"
for file in "$SCAN_ROOT"/make/*.mk "$SCAN_ROOT"/.github/workflows/*.yml; do
	[ -f "$file" ] && printf '%s\n' "$file" >> "$INPUTS"
done

if [ -s "$INPUTS" ]; then
	# Both boundaries are intentional: a path-like prefix (for example docs/scripts/)
	# must not be truncated into an apparently in-scope token.
	# shellcheck disable=SC2016 # The dollar expressions below belong to awk.
	xargs awk '
	{
		line = $0
		while (match(line, /(^|[^A-Za-z0-9_.\/-])((tools|scripts)\/[A-Za-z0-9_.\/-]+\.(sh|py))([^A-Za-z0-9_.\/-]|$)/)) {
			hit = substr(line, RSTART, RLENGTH)
			if (hit !~ /^(tools|scripts)\//) hit = substr(hit, 2)
			sub(/[^A-Za-z0-9_.\/-]$/, "", hit)
			print hit
			line = substr(line, RSTART + RLENGTH)
		}
	}' < "$INPUTS" | LC_ALL=C sort -u > "$PATHS"
else
	: > "$PATHS"
fi

COUNT=$(wc -l < "$PATHS" | tr -d ' ')
echo "referenced-paths: enumerated $COUNT paths"
if [ "$COUNT" -lt "$MINIMUM" ]; then
	echo "instrument failure: enumeration returned $COUNT paths" >&2
	exit 2
fi

FAILED=0
while IFS= read -r path; do
	if [ ! -e "$REPO_ROOT/$path" ]; then
		echo "missing referenced path: $path" >&2
		FAILED=1
	elif ! git -C "$REPO_ROOT" ls-files --error-unmatch -- "$path" >/dev/null 2>&1; then
		echo "untracked referenced path: $path" >&2
		FAILED=1
	fi
done < "$PATHS"

if [ "$FAILED" -ne 0 ]; then
	exit 1
fi

echo "referenced-paths: checked $COUNT paths"
