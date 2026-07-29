#!/usr/bin/env bash
# Guard: root CHANGELOG.md is an INDEX, not a changelog.
#
# Release notes belong in changelogs/v*-current.md. Between v0.30.0 and v0.31.0,
# 8 commits appended entries to root CHANGELOG.md while 33 correctly used the
# archive -- stranding 12 entries that would have silently vanished from the
# release notes, because release-manager reads the archive file.
#
# This gate makes that mistake fail fast instead of at release time.
set -uo pipefail

ROOT_CHANGELOG="CHANGELOG.md"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'

if [ ! -f "$ROOT_CHANGELOG" ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG not found (run from repo root)${RESET}"
	exit 1
fi

# Content that marks a file as a changelog rather than an index.
# -n gives us line numbers so the failure points at the exact offending lines.
OFFENDERS=$(grep -nE '^##[[:space:]]*\[(Unreleased|v?[0-9]+\.[0-9]+)|^###[[:space:]]+(Added|Fixed|Changed|Removed|Deprecated|Security)' \
	"$ROOT_CHANGELOG" || true)

if [ -n "$OFFENDERS" ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG contains release-note content; it must stay an index.${RESET}"
	echo ""
	echo "$OFFENDERS" | sed 's/^/    /'
	echo ""
	ACTIVE=$(ls changelogs/ 2>/dev/null | grep current | head -1)
	echo -e "${YELLOW}Move these entries into changelogs/${ACTIVE:-v*-current.md} and leave"
	echo -e "$ROOT_CHANGELOG as the archive table only.${RESET}"
	echo ""
	echo "Why this matters: release-manager builds release notes from the active"
	echo "changelogs/ file. Anything left here is silently dropped from the release."
	exit 1
fi

# The index is only useful if it actually points at the active changelog.
ACTIVE=$(ls changelogs/ 2>/dev/null | grep current | head -1)
if [ -n "$ACTIVE" ] && ! grep -qF "changelogs/$ACTIVE" "$ROOT_CHANGELOG"; then
	echo -e "${RED}✗ $ROOT_CHANGELOG does not link the active changelog (changelogs/$ACTIVE)${RESET}"
	exit 1
fi

echo -e "${GREEN}✓ $ROOT_CHANGELOG is index-only and links changelogs/${ACTIVE}${RESET}"
