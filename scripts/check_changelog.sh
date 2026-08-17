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
# 2026-08-17 (motoko iteration 9) -- THE DETECTOR WAS NEARLY BLIND AND SAID SO
# ONLY ONCE IN FIVE CHANCES. It matched Keep-a-Changelog *keyword* headers
# (`### Added|Fixed|...`) and *bracketed* `## [Unreleased]`. This repo writes
# neither: its entries are `### Docs -- staleness/fluff audit (2026-08-17)`,
# `### Eval cost accuracy -- ...`, `### Mission infrastructure ...`, under an
# unbracketed `## v0.32.0 (Unreleased)`. Measured at 0002c9b0b: the index held
# 168 lines of release notes in 5 blocks and this gate flagged exactly ONE of
# them -- the single block whose author happened to start the header with the
# word "Added". The other 4 (and the section header above them) were invisible
# BY CONSTRUCTION, so the gate reported the repo's actual state as clean while
# release-manager would have dropped all of it.
#
# The enumerator, not the branches, was the defect: an index has no `###`
# sub-sections and no version headers at all, so that -- not a keyword list --
# is what this now tests. Guarded by scripts/test_check_changelog.sh.
set -uo pipefail

ROOT_CHANGELOG="CHANGELOG.md"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'

if [ ! -f "$ROOT_CHANGELOG" ]; then
	echo -e "${RED}✗ $ROOT_CHANGELOG not found (run from repo root)${RESET}"
	exit 1
fi

# Content that marks a file as a changelog rather than an index:
#   - ANY `###` sub-section (an index has none; keyword-independent by design)
#   - a version or Unreleased section header, bracketed or not
# -n gives us line numbers so the failure points at the exact offending lines.
OFFENDERS=$(grep -nE '^###[[:space:]]|^##[[:space:]]*\[?[[:space:]]*(Unreleased|v?[0-9]+\.[0-9]+)' \
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

# ANTI-VACUITY FLOOR. Every check below is an "absence" test, and absence is
# exactly what a missing/empty changelogs/ directory also produces -- so without
# this the gate prints a green checkmark for a repo whose release notes have
# vanished entirely. An enumerator that finds nothing must fail loudly, never pass.
ACTIVE=$(ls changelogs/ 2>/dev/null | grep current | head -1)
if [ -z "$ACTIVE" ]; then
	echo -e "${RED}✗ no changelogs/*current* file found -- the index points nowhere${RESET}"
	echo "  (instrument failure: this gate cannot certify an index with no destination)"
	exit 1
fi

# The index is only useful if it actually points at the active changelog.
if ! grep -qF "changelogs/$ACTIVE" "$ROOT_CHANGELOG"; then
	echo -e "${RED}✗ $ROOT_CHANGELOG does not link the active changelog (changelogs/$ACTIVE)${RESET}"
	exit 1
fi

echo -e "${GREEN}✓ $ROOT_CHANGELOG is index-only and links changelogs/${ACTIVE}${RESET}"
