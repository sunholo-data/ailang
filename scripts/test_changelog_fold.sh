#!/usr/bin/env bash
# Self-test for changelog_fold.sh (bash 3.2). Each arm runs in a throwaway git
# repo so the real changelogs/ is never touched.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
FOLD="$REPO_ROOT/scripts/changelog_fold.sh"
FAILED=0; ARMS_RUN=0; ARMS_EXPECTED=8
pass() { echo "  ok   — $1"; ARMS_RUN=$((ARMS_RUN + 1)); }
fail() { echo "  FAIL — $1"; FAILED=1; ARMS_RUN=$((ARMS_RUN + 1)); }

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# fresh_repo DIR: a git repo with an active changelog: an empty [Unreleased] (the invariant
# between releases) above a released section holding one existing entry.
fresh_repo() {
	mkdir -p "$1/changelogs/unreleased"
	printf '%s\n' '# Changelog' '' '## [Unreleased]' '' '## [v0.1.0] - 2026-01-01' '' '### Fixed — older entry' '' 'old body' >"$1/changelogs/v0.1-current.md"
	printf '%s\n' '# How to write a fragment' >"$1/changelogs/unreleased/README.md"
	(cd "$1" && git init -q && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
}
frag() { printf '%s\n' "$3" >"$1/changelogs/unreleased/$2"; (cd "$1" && git add -A && git -c user.name=t -c user.email=t@t commit -qm "frag $2"); }
run() { OUT=$(cd "$1" && /bin/bash "$FOLD" ${2:-} 2>&1); RC=$?; }

echo "changelog fold:"

# (1) Fold: newest first, above the existing entry, fragments removed, README kept.
R="$WORK/r1"; fresh_repo "$R"
frag "$R" 2026-09-01-alpha.md '### Added — alpha'
frag "$R" 2026-09-02-beta.md "$(printf '%s\n' '### Fixed — beta' '' 'beta body')"
run "$R"
F="$R/changelogs/v0.1-current.md"
ORDER=$(grep -nE '^### ' "$F" | cut -d: -f2- | tr '\n' '|')
if [ "$RC" -ne 0 ]; then
	fail "fold rc=$RC: $OUT"
elif [ "$ORDER" != "### Fixed — beta|### Added — alpha|### Fixed — older entry|" ]; then
	fail "fold order wrong: $ORDER"
elif [ "$(grep -n -m1 '^## \[Unreleased\]' "$F" | cut -d: -f1)" -gt "$(grep -n -m1 '^### Fixed — beta' "$F" | cut -d: -f1)" ]; then
	fail "fragments landed above ## [Unreleased]"
elif ls "$R"/changelogs/unreleased/2026-* >/dev/null 2>&1; then
	fail "fragments were not removed after folding"
elif [ ! -f "$R/changelogs/unreleased/README.md" ]; then
	fail "README.md was folded or deleted"
elif ! (cd "$R" && git status --porcelain | grep -q '^D  changelogs/unreleased/2026-09-01-alpha.md'); then
	fail "tracked fragment was not git rm'd (status: $(cd "$R" && git status --porcelain | tr '\n' ' '))"
else
	pass "folds newest-first under [Unreleased], above existing entries; removes fragments, keeps README"
fi

# (2) No fragments: no-op, file unchanged.
R="$WORK/r2"; fresh_repo "$R"; BEFORE=$(cat "$R/changelogs/v0.1-current.md")
run "$R"
if [ "$RC" -eq 0 ] && [ "$(cat "$R/changelogs/v0.1-current.md")" = "$BEFORE" ]; then
	pass "no fragments is a no-op"
else fail "no-fragment run rc=$RC or changed the file: $OUT"; fi

# (3) --check never modifies anything.
R="$WORK/r3"; fresh_repo "$R"; frag "$R" 2026-09-03-gamma.md '### Changed — gamma'
BEFORE=$(cat "$R/changelogs/v0.1-current.md")
run "$R" --check
if [ "$RC" -eq 0 ] && [ -f "$R/changelogs/unreleased/2026-09-03-gamma.md" ] && [ "$(cat "$R/changelogs/v0.1-current.md")" = "$BEFORE" ]; then
	pass "--check validates without folding"
else fail "--check rc=$RC or modified state: $OUT"; fi

# (4)-(6) --check refuses malformed fragments, naming the file.
refuse_arm() {
	R="$WORK/$1"; fresh_repo "$R"; frag "$R" "$2" "$3"
	run "$R" --check
	if [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -qF "$2" && printf '%s' "$OUT" | grep -qF "$4"; then
		pass "$5"
	else fail "$5 — rc=$RC: $OUT"; fi
}
refuse_arm r4 'my-entry.md' '### Fixed — x' 'name must be YYYY-MM-DD' "refuses a fragment without a date-prefixed name"
refuse_arm r5 2026-09-04-noheading.md 'just prose, no heading' "must start with a '### '" "refuses a fragment that does not open with a ### heading"
refuse_arm r6 2026-09-05-toplevel.md "$(printf '%s\n' '### Fixed — ok' '' '## [v9.9.9]')" "belong to the active file" "refuses a fragment carrying a ## heading"

# (7) No [Unreleased] heading to fold under: refuse, never append somewhere arbitrary.
R="$WORK/r7"; fresh_repo "$R"
sed -i.bak 's/^## \[Unreleased\]$/## [v0.2.0] - 2026-02-02/' "$R/changelogs/v0.1-current.md" && rm -f "$R/changelogs/v0.1-current.md.bak"
frag "$R" 2026-09-06-delta.md '### Added — delta'
run "$R"
if [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -qF "no '## [Unreleased]'" && [ -f "$R/changelogs/unreleased/2026-09-06-delta.md" ]; then
	pass "refuses to fold when there is no [Unreleased] heading, and keeps the fragment"
else fail "missing-[Unreleased] arm rc=$RC: $OUT"; fi

# (8) An entry written straight under [Unreleased] is refused, naming its line: that shared
# block is what every branch conflicted on before fragments.
R="$WORK/r8"; fresh_repo "$R"
awk '{print} /^## \[Unreleased\]$/ {print ""; print "### Fixed — written in place"}' "$R/changelogs/v0.1-current.md" >"$R/x" && mv "$R/x" "$R/changelogs/v0.1-current.md"
run "$R" --check
if [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -qF "directly under '## [Unreleased]'" && printf '%s' "$OUT" | grep -qF "5: ### Fixed — written in place"; then
	pass "refuses an entry written directly under [Unreleased], naming its line"
else fail "direct-entry arm rc=$RC: $OUT"; fi

if [ "$ARMS_RUN" -ne "$ARMS_EXPECTED" ]; then
	echo "  FAIL — $ARMS_RUN of $ARMS_EXPECTED arms ran; refusing a vacuous green"; FAILED=1
fi
if [ "$FAILED" -eq 0 ]; then echo "changelog fold: OK ($ARMS_RUN arms)"; exit 0; fi
echo "changelog fold: FAILED"; exit 1
