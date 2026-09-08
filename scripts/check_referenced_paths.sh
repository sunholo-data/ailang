#!/bin/bash
# SCOPE, STATED IN FULL — an under-disclosed scope is how a gate gets quoted for a
# sentence it does not support. This enumerator finds a reference only when ALL of:
#   (a) the path is written LITERALLY, starting with `tools/` or `scripts/`; and
#   (b) its extension is one of .sh .bash .py .pl (matched case-insensitively).
# KNOWN BLIND SPOTS, deliberate and unfixable at this layer:
#   - Make-variable composition. `$(TOOLS_DIR)/x.sh` does not start literally with
#     `tools/`, so it is invisible here. Resolving it would mean evaluating make
#     variables, which this gate does not do. Measured 2026-09-03 (iteration 323):
#     a fixture with such a reference dangling passes rc=0.
#   - Any other extension (.rb, .ts, .mjs, a bare interpreterless file).
#   - Prefixes other than tools/ and scripts/ — docs/scripts/... and generated paths
#     such as .stdlib-golden/option.sh are out of scope on purpose.
# So a green here means "no LITERAL tools//scripts/ script reference dangles", never
# "no reference dangles".
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
	BEGIN { split("sh bash py pl", e, " "); for (i in e) want[e[i]] = 1 }
	{
		line = $0
		while (match(line, /(^|[^A-Za-z0-9_.\/-])((tools|scripts)\/[A-Za-z0-9_.\/-]+\.[A-Za-z][A-Za-z0-9]*)([^A-Za-z0-9_.\/-]|$)/)) {
			hit = substr(line, RSTART, RLENGTH)
			if (hit !~ /^(tools|scripts)\//) hit = substr(hit, 2)
			sub(/[^A-Za-z0-9_.\/-]$/, "", hit)
			ext = hit
			sub(/^.*\./, "", ext)
			if (tolower(ext) in want) print hit
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
