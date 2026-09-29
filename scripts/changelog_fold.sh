#!/usr/bin/env bash
# Fold changelog fragments into the active changelog (bash 3.2).
#
# Why fragments: every PR used to add its entry at the same line, directly under
# `## [Unreleased]` in changelogs/v*-current.md. Any merge therefore conflicted
# with every other open PR, and each rebase restarted a 25-45 min CI run. On
# 2026-09-28 one PR (#1369) needed four such rebases. A fragment is its own file,
# so two PRs can never conflict on it.
#
# A fragment is changelogs/unreleased/<YYYY-MM-DD>-<slug>.md holding one or more
# complete `### ...` sections, exactly as they would appear in the active file.
# README.md in that directory is documentation, never folded.
#
# Usage:
#   scripts/changelog_fold.sh           fold: insert every fragment, newest first,
#                                       directly under `## [Unreleased]`, then
#                                       delete the fragments (git rm when tracked)
#   scripts/changelog_fold.sh --check   validate only; used by check-changelog
#
# release-manager runs the fold BEFORE it renames `## [Unreleased]`, so every
# downstream reader (release notes, broadcast, docs search) sees one file, as before.
set -uo pipefail

FRAG_DIR="changelogs/unreleased"
MODE="${1:-fold}"
RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'

fail() { echo -e "${RED}✗ $1${RESET}"; exit 1; }

ACTIVE=$(ls changelogs/ 2>/dev/null | grep current | head -1)
[ -n "$ACTIVE" ] || fail "no changelogs/*current* file found"
ACTIVE="changelogs/$ACTIVE"

# Newest first, matching the active file's order. Filenames start with a date,
# so a reverse lexical sort is a reverse chronological one.
FRAGMENTS=$(ls "$FRAG_DIR"/*.md 2>/dev/null | grep -v '/README\.md$' | sort -r)

BAD=""
for f in $FRAGMENTS; do
	base=$(basename "$f")
	echo "$base" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9][a-z0-9._-]*\.md$' \
		|| BAD="$BAD\n    $f: name must be YYYY-MM-DD-<slug>.md (lowercase slug)"
	first=$(grep -m1 -vE '^[[:space:]]*$' "$f")
	echo "$first" | grep -qE '^### [^[:space:]]' \
		|| BAD="$BAD\n    $f: must start with a '### ' section heading (got: ${first:-<empty file>})"
	if grep -qE '^#{1,2}[[:space:]]' "$f"; then
		BAD="$BAD\n    $f: '#'/'##' headings belong to the active file, not a fragment"
	fi
done
[ -z "$BAD" ] || fail "invalid changelog fragment(s):$BAD"

UNRELEASED_LINE=$(grep -n -m1 -E '^## \[Unreleased\][[:space:]]*$' "$ACTIVE" | cut -d: -f1)
[ -n "$UNRELEASED_LINE" ] || fail "$ACTIVE has no '## [Unreleased]' heading to fold under"

COUNT=$(echo "$FRAGMENTS" | grep -c . || true)
if [ "$MODE" = "--check" ]; then
	# Between releases `## [Unreleased]` stays EMPTY: entries live only in fragments, and
	# the release commit folds them and renames the heading in one step. Fragments alone
	# did not change the habit: the day they landed (#1382, 2026-09-28) five entries were
	# still written straight under the heading, and each was a guaranteed conflict with
	# every other open branch. Structural rule, not a lexical one: ANY `### ` section
	# between `## [Unreleased]` and the next `## ` heading is an entry in the wrong place.
	DIRECT=$(awk -v start="$UNRELEASED_LINE" 'NR > start && /^## / { exit } NR > start && /^### / { print "    " NR ": " $0 }' "$ACTIVE")
	if [ -n "$DIRECT" ]; then
		fail "$ACTIVE has entries directly under '## [Unreleased]':
$DIRECT
  Move each section into its own fragment, changelogs/unreleased/YYYY-MM-DD-<slug>.md
  (see changelogs/unreleased/README.md). Editing the shared block conflicts with every
  other open branch; separate files cannot."
	fi
	echo -e "${GREEN}✓ $COUNT changelog fragment(s) valid; $ACTIVE has '## [Unreleased]'${RESET}"
	exit 0
fi
[ "$MODE" = "fold" ] || fail "unknown mode '$MODE' (use no argument or --check)"
if [ "$COUNT" -eq 0 ]; then
	echo "no changelog fragments to fold"
	exit 0
fi

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
{
	head -n "$UNRELEASED_LINE" "$ACTIVE"
	for f in $FRAGMENTS; do
		echo ""
		# Strip leading/trailing blank lines so spacing is uniform.
		sed -e '/./,$!d' "$f" | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}'
	done
	tail -n +"$((UNRELEASED_LINE + 1))" "$ACTIVE"
} >"$TMP"
cat "$TMP" >"$ACTIVE"

for f in $FRAGMENTS; do
	if git ls-files --error-unmatch "$f" >/dev/null 2>&1; then git rm -q "$f"; else rm -f "$f"; fi
done
echo -e "${GREEN}✓ folded $COUNT fragment(s) into $ACTIVE${RESET}"
