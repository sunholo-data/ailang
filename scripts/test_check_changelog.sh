#!/usr/bin/env bash
# Self-test for scripts/check_changelog.sh -- the CI gate that keeps root
# CHANGELOG.md an index.
#
# WHY THIS EXISTS. The gate ran in CI (ci.yml "Check changelog index hygiene")
# from the day it was written and had NO test of its own. It was measured on
# 2026-08-17 (motoko iteration 9) against the repo's own HEAD at 0002c9b0b:
# root CHANGELOG.md held 168 lines of release notes in 5 blocks, and the gate
# flagged 1 of 5 -- the one whose header began with the Keep-a-Changelog keyword
# "Added". Its detector was a keyword allowlist, so 4 real offenders were
# invisible by construction and the gate printed a failure that understated the
# problem by 4x. A green from that gate meant "no block used a keyword I know",
# not "the index is clean".
#
# The lesson is about the ENUMERATOR, so every arm below feeds the gate an input
# shaped like the ones it used to miss, and requires a RED. An arm that passes
# here is a branch that would have shipped unguarded.
#
# bash 3.2 compatible (the rig's /bin/bash).
set -uo pipefail

PASS=0; FAIL=0
GATE="$(cd "$(dirname "$0")" && pwd)/check_changelog.sh"
RED='\033[0;31m'; GREEN='\033[0;32m'; RESET='\033[0m'

# Each case runs in a throwaway dir with its own CHANGELOG.md + changelogs/.
# NOT under /tmp-with-a-relative-cwd games: the gate reads fixed relative paths,
# so we cd into the fixture root, which is what CI does too.
run_case() {
	local name="$1" want_rc="$2" changelog="$3" make_changelogs="${4:-yes}"
	local dir; dir=$(mktemp -d)
	printf '%s' "$changelog" > "$dir/CHANGELOG.md"
	if [ "$make_changelogs" = "yes" ]; then
		mkdir -p "$dir/changelogs"
		printf '# active\n' > "$dir/changelogs/v0.18-current.md"
	fi
	local out rc
	out=$( cd "$dir" && bash "$GATE" 2>&1 ); rc=$?
	rm -rf "$dir"
	if [ "$rc" -eq "$want_rc" ]; then
		PASS=$((PASS+1)); printf "  ${GREEN}ok${RESET}   %s (rc=%s)\n" "$name" "$rc"
	else
		FAIL=$((FAIL+1)); printf "  ${RED}FAIL${RESET} %s: want rc=%s got rc=%s\n" "$name" "$want_rc" "$rc"
		printf '%s\n' "$out" | sed 's/^/         /'
	fi
}

INDEX_OK='# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## Changelog Archives

| File | Versions | Theme |
|------|----------|-------|
| [v0.18-current.md](changelogs/v0.18-current.md) | v0.18.0+ | Eval Harness |
'

echo "=== check_changelog.sh self-test ==="

# --- The known-positive control. If this reds, every RED below is meaningless,
# --- because a gate that rejects everything "catches" all offenders vacuously.
run_case "clean index passes (CONTROL: gate can say yes)" 0 "$INDEX_OK"

# --- The branch that already worked: a Keep-a-Changelog keyword header.
run_case "### Added ... is rejected (pre-existing branch)" 1 "${INDEX_OK}
### Added \`or-qwen3-8-27b\` — OpenRouter quality screen
- body
"

# --- THE FOUR SHAPES THE OLD DETECTOR MISSED. Each is verbatim-shaped from the
# --- blocks that were live in root CHANGELOG.md at 0002c9b0b.
run_case "### Docs — ... is rejected (was invisible)" 1 "${INDEX_OK}
### Docs — staleness/fluff audit (2026-08-17)
- body
"
run_case "### Eval cost accuracy — ... is rejected (was invisible)" 1 "${INDEX_OK}
### Eval cost accuracy — OpenRouter pricing drift
- body
"
run_case "### Mission infrastructure ... is rejected (was invisible)" 1 "${INDEX_OK}
### Mission infrastructure (no user-facing change)
- body
"
run_case "unbracketed ## v0.32.0 (Unreleased) is rejected (was invisible)" 1 "${INDEX_OK}
## v0.32.0 (Unreleased)

- body
"

# --- Bracketed forms the original detector did cover; kept so a future rewrite
# --- cannot narrow the gate back down without reddening.
run_case "## [Unreleased] is rejected" 1 "${INDEX_OK}
## [Unreleased]
- body
"
run_case "## [v0.32.0] is rejected" 1 "${INDEX_OK}
## [v0.32.0]
- body
"

# --- The archive table's own heading must NOT trip the gate. This is the arm
# --- that stops "reject every ##" from passing the four arms above vacuously.
run_case "## Changelog Archives does not trip the gate" 0 "$INDEX_OK"

# --- Anti-vacuity floor: absence of a destination is an instrument failure,
# --- not a clean bill of health.
run_case "missing changelogs/ dir fails loudly" 1 "$INDEX_OK" "no"

# --- The index must actually link the active changelog.
run_case "index not linking the active changelog fails" 1 '# AILANG Changelog

## Changelog Archives

no link here
'

echo ""
echo "check_changelog self-test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
