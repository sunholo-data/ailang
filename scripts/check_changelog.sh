#!/usr/bin/env bash
# Guard: root CHANGELOG.md is an INDEX, not a changelog.
#
# Release notes belong in changelogs/v*-current.md. Between v0.30.0 and v0.31.0,
# 8 commits appended entries to root CHANGELOG.md while 33 correctly used the
# archive -- stranding 12 entries that would have silently vanished from the
# release notes, because release-manager reads the archive file.
#
# This gate makes that mistake fail fast instead of at release time.
#
# The heading rule is a DENY-LIST INVERSION, deliberately. Until 2026-08-17 this
# gate matched named section verbs (`### Added|Fixed|Changed|...`) plus bracketed
# version headings (`## [Unreleased]`, `## [v1.2]`). A gate's coverage is a
# property of its enumerator, so that pattern decided what counted as a release
# note -- and release notes written any other way were invisible to it. Measured
# on dev at 0002c9b0b: root CHANGELOG.md carried FIVE stranded sections spanning
# 169 lines, and the gate flagged exactly ONE of them. `## v0.32.0 (Unreleased)`
# has no brackets; `### Docs -- staleness/fluff audit`, `### Eval cost accuracy
# -- ...` (x2) and `### Mission infrastructure ...` open with no Keep-a-Changelog
# verb. All four sailed through while the gate printed a failure about the fifth.
#
# So the rule is now structural rather than lexical: an index has exactly one
# heading, the archive table's. ANY other heading is release-note content, no
# matter how it is worded.
set -uo pipefail

ROOT_CHANGELOG="CHANGELOG.md"
ARCHIVE_HEADING="## Changelog Archives"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'

if [ ! -f "$ROOT_CHANGELOG" ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG not found (run from repo root)${RESET}"
	exit 1
fi

ACTIVE=$(ls changelogs/ 2>/dev/null | grep current | head -1)

# Anti-vacuity floor: the archive heading is the one heading an index is allowed
# to have, and every check below is stated relative to it. Without it we cannot
# tell an index from an empty file, and "no offenders found" would be vacuous.
ARCHIVE_COUNT=$(grep -cF "$ARCHIVE_HEADING" "$ROOT_CHANGELOG")
if [ "$ARCHIVE_COUNT" -ne 1 ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG has $ARCHIVE_COUNT '$ARCHIVE_HEADING' headings (expected exactly 1)${RESET}"
	echo ""
	echo "The archive table is what makes this file an index. Without exactly one"
	echo "such heading this gate cannot tell an index from a changelog, so it"
	echo "refuses rather than passing on an unrecognised shape."
	exit 1
fi

# Any heading other than the archive table is release-note content.
# -n gives us line numbers so the failure points at the exact offending lines.
OFFENDERS=$(grep -nE '^#{2,6}[[:space:]]' "$ROOT_CHANGELOG" \
	| grep -vE "^[0-9]+:${ARCHIVE_HEADING}\$" || true)

if [ -n "$OFFENDERS" ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG contains release-note content; it must stay an index.${RESET}"
	echo ""
	echo "$OFFENDERS" | sed 's/^/    /'
	echo ""
	echo -e "${YELLOW}Move these entries into changelogs/${ACTIVE:-v*-current.md} and leave"
	echo -e "$ROOT_CHANGELOG as the archive table only.${RESET}"
	echo ""
	echo "Why this matters: release-manager builds release notes from the active"
	echo "changelogs/ file. Anything left here is silently dropped from the release."
	echo ""
	echo "The only heading an index may carry is '$ARCHIVE_HEADING'."
	exit 1
fi

# The index is only useful if it actually points at the active changelog.
if [ -n "$ACTIVE" ] && ! grep -qF "changelogs/$ACTIVE" "$ROOT_CHANGELOG"; then
	echo -e "${RED}✗ $ROOT_CHANGELOG does not link the active changelog (changelogs/$ACTIVE)${RESET}"
	exit 1
fi

echo -e "${GREEN}✓ $ROOT_CHANGELOG is index-only and links changelogs/${ACTIVE}${RESET}"
