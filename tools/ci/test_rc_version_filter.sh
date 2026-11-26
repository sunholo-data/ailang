#!/usr/bin/env bash
# test_rc_version_filter.sh — verify pre-release tags (e.g. v1.0.0-rc.1, v1.0.0-rc.2, v1.0.0-rc.10)
# never overwrite STABLE_RELEASE / AILANG_VERSION in docs/scripts/generate-version-constants.sh,
# docusaurus-deploy.yml, or release.yml's previous-final-tag changelog boundary.
# Compatible with bash 3.2 (macOS default) and Linux CI.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HERE/../.." && pwd)"
GEN_SCRIPT="$REPO_ROOT/docs/scripts/generate-version-constants.sh"

if [ ! -f "$GEN_SCRIPT" ]; then
	echo "FAIL — missing $GEN_SCRIPT" >&2
	exit 1
fi

T=$(mktemp -d "${TMPDIR:-/tmp}/rc-version-filter-test.XXXXXX") || exit 1
T=$(cd "$T" && pwd -P) || exit 1
trap 'rm -rf "$T"' EXIT

pass=0
fail=0
ck() {
	if [ "$2" = "$3" ]; then
		echo "  ok   — $1"
		pass=$((pass + 1))
	else
		echo "  FAIL — $1 (got '$2' want '$3')"
		fail=$((fail + 1))
	fi
}

# Set up an isolated temp git repo mirroring the directory layout generate-version-constants.sh expects.
REPO="$T/repo"
mkdir -p "$REPO/docs/scripts" "$REPO/docs/src/constants" "$REPO/prompts"
cp "$GEN_SCRIPT" "$REPO/docs/scripts/generate-version-constants.sh"
chmod +x "$REPO/docs/scripts/generate-version-constants.sh"
printf '{"active":"v0.16.20"}\n' > "$REPO/prompts/versions.json"
printf '# Changelog\n\n## [Unreleased - v1.0.0]\n' > "$REPO/CHANGELOG.md"

git -C "$REPO" init -q
git -C "$REPO" config user.email "test@ailang.local"
git -C "$REPO" config user.name "AILANG Test"

commit_step() {
	printf '%s\n' "$1" >> "$REPO/history.txt"
	git -C "$REPO" add -A
	git -C "$REPO" commit -qm "step: $1"
}

extract_const() {
	sed -n "s/^export const $1 = '\\([^']*\\)';$/\\1/p" "$REPO/docs/src/constants/version.js"
}

docusaurus_ailang_version() {
	git -C "$REPO" tag -l 'v*.*.*' --sort=-v:refname | grep -v '-' | head -1 || true
}

echo "rc_version_filter:"

# 1. Repo with ONLY pre-release tags must fail closed (never fall back to an RC tag).
commit_step "rc-only"
git -C "$REPO" tag v1.0.0-rc.1
OUT=$(bash "$REPO/docs/scripts/generate-version-constants.sh" 2>&1)
RC=$?
ck "pre-release-only repo fails closed (exit != 0)" "$([ "$RC" -ne 0 ] && echo nonzero || echo zero)" "nonzero"
git -C "$REPO" tag -d v1.0.0-rc.1 >/dev/null

# 2. Tag v0.63.0, v0.64.0, then v1.0.0-rc.1, v1.0.0-rc.2, v1.0.0-rc.10 (plus alpha).
# Note: git --sort=-v:refname sorts v1.0.0-rc.* ABOVE v0.64.0 and even above v1.0.0!
commit_step "release-0.63.0"
git -C "$REPO" tag v0.63.0

commit_step "release-0.64.0"
git -C "$REPO" tag v0.64.0

commit_step "rc-1"
git -C "$REPO" tag v1.0.0-rc.1

commit_step "rc-2"
git -C "$REPO" tag v1.0.0-rc.2

commit_step "rc-10"
git -C "$REPO" tag v1.0.0-rc.10
git -C "$REPO" tag v1.0.0-alpha.1

OUT=$(bash "$REPO/docs/scripts/generate-version-constants.sh" 2>&1)
RC=$?
ck "with v1.0.0-rc.{1,2,10} tagged, generate-version-constants.sh exits 0" "$RC" 0
ck "with v1.0.0-rc.{1,2,10} tagged, STABLE_RELEASE stays v0.64.0" "$(extract_const STABLE_RELEASE)" "v0.64.0"
ck "with v1.0.0-rc.{1,2,10} tagged, AILANG_VERSION stays v0.64.0" "$(extract_const AILANG_VERSION)" "v0.64.0"
ck "docusaurus AILANG_VERSION pipeline resolves v0.64.0 during RC soak" "$(docusaurus_ailang_version)" "v0.64.0"

# Verify release.yml changelog boundary skips v1.0.0-rc.* on HEAD^ and finds v0.64.0.
commit_step "release-1.0.0"
PREV_FINAL=$(git -C "$REPO" describe --tags --abbrev=0 --exclude='*-*' HEAD^ 2>/dev/null || echo "")
ck "release.yml PREV_FINAL skips v1.0.0-rc.* on HEAD^ and resolves v0.64.0" "$PREV_FINAL" "v0.64.0"

# 3. Tag final v1.0.0 (while v1.0.0-rc.1, v1.0.0-rc.2, v1.0.0-rc.10 still exist in git!).
# Crucially, `git tag -l 'v*.*.*' --sort=-v:refname | head -1` without `-` filtering
# would STILL return v1.0.0-rc.2 / v1.0.0-rc.10 here because git orders v1.0.0-rc.* > v1.0.0.
git -C "$REPO" tag v1.0.0
RAW_TOP=$(git -C "$REPO" tag -l 'v*.*.*' --sort=-v:refname | head -1)
ck "control: raw git --sort=-v:refname puts v1.0.0-rc.* above v1.0.0" "$(printf '%s' "$RAW_TOP" | grep -c -- '-')" 1

OUT=$(bash "$REPO/docs/scripts/generate-version-constants.sh" 2>&1)
RC=$?
ck "once final v1.0.0 is tagged, generate-version-constants.sh exits 0" "$RC" 0
ck "once final v1.0.0 is tagged, STABLE_RELEASE advances to v1.0.0" "$(extract_const STABLE_RELEASE)" "v1.0.0"
ck "once final v1.0.0 is tagged, AILANG_VERSION advances to v1.0.0" "$(extract_const AILANG_VERSION)" "v1.0.0"
ck "docusaurus AILANG_VERSION pipeline advances to v1.0.0" "$(docusaurus_ailang_version)" "v1.0.0"

# 4. Subsequent v1.0.1-rc.1 must not displace v1.0.0.
commit_step "rc-1.0.1"
git -C "$REPO" tag v1.0.1-rc.1
OUT=$(bash "$REPO/docs/scripts/generate-version-constants.sh" 2>&1)
RC=$?
ck "with v1.0.1-rc.1 tagged after v1.0.0, STABLE_RELEASE stays v1.0.0" "$RC:$(extract_const STABLE_RELEASE)" "0:v1.0.0"

# 5. Verify release.yml ci-gate tag regex accepts finals + RCs and rejects non-RC hyphenated tags.
TAG_REGEX='^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$'
match_tag() {
	if printf '%s\n' "$1" | grep -Eq "$TAG_REGEX"; then echo accept; else echo reject; fi
}
ck "ci-gate tag regex accepts v0.64.0" "$(match_tag v0.64.0)" "accept"
ck "ci-gate tag regex accepts v1.0.0" "$(match_tag v1.0.0)" "accept"
ck "ci-gate tag regex accepts v1.0.0-rc.1" "$(match_tag v1.0.0-rc.1)" "accept"
ck "ci-gate tag regex accepts v1.0.0-rc.10" "$(match_tag v1.0.0-rc.10)" "accept"
ck "ci-gate tag regex rejects v1.0.0-alpha.1" "$(match_tag v1.0.0-alpha.1)" "reject"
ck "ci-gate tag regex rejects v1.0.0-beta.2" "$(match_tag v1.0.0-beta.2)" "reject"
ck "ci-gate tag regex rejects v1.0.0-rc1 (non-dot)" "$(match_tag v1.0.0-rc1)" "reject"
ck "ci-gate tag regex rejects v1.0.0-rc." "$(match_tag v1.0.0-rc.)" "reject"
ck "ci-gate tag regex rejects v1.0.0-rc.1-extra" "$(match_tag v1.0.0-rc.1-extra)" "reject"
ck "ci-gate tag regex rejects 1.0.0 (missing v)" "$(match_tag 1.0.0)" "reject"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ] && [ "$pass" -eq 22 ]
