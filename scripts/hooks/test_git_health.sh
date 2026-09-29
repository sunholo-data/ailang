#!/usr/bin/env bash
# Self-test for git_health.sh (bash 3.2): throwaway repo + bare origin, fake HOME, and a stub
# worktree_sweep.sh that only records that it ran. Never touches a real repo.
set -u
HOOK="$(cd "$(dirname "$0")" && pwd)/git_health.sh"
T=$(mktemp -d "${TMPDIR:-/tmp}/git-health-test.XXXXXX") || exit 1
T=$(cd "$T" && pwd -P) || exit 1
trap 'rm -rf "$T"' EXIT
export HOME="$T/home"; mkdir -p "$HOME"
unset AILANG_MISSION_STAGE AILANG_TASK_ID AILANG_WORKTREE_SWEEP
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  ok   — $1"; pass=$((pass+1)); else echo "  FAIL — $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }

git init -q --bare "$T/origin.git"
git clone -q "$T/origin.git" "$T/main" 2>/dev/null; cd "$T/main" || exit 1
git config user.email t@t; git config user.name t
git checkout -q -b dev; echo a > a; git add a; git commit -qm init; git push -q origin dev
mkdir -p scripts; printf '#!/bin/sh\necho ran >> "%s/swept"\n' "$T" > scripts/worktree_sweep.sh; chmod +x scripts/worktree_sweep.sh
git worktree add -q --detach "$T/wt" origin/dev 2>/dev/null
run() { OUT=$(CLAUDE_PROJECT_DIR="$1" bash "$HOOK" 2>&1); RC=$?; sleep 1; }
swept() { [ -f "$T/swept" ] && wc -l < "$T/swept" | tr -d ' ' || echo 0; }

echo "git health:"
run "$T/main"
ck "healthy: silent, exit 0" "$RC:$OUT" "0:"
ck "first session: sweep launched in background" "$(swept)" "1"
run "$T/main"
ck "within 6h: sweep not relaunched" "$(swept)" "1"

mkdir "$T/main/.git/rebase-merge"
run "$T/wt"
ck "main checkout mid-rebase: reported from a worktree session" "$(echo "$OUT" | grep -c 'main checkout .* rebase-merge in progress')" "1"
rmdir "$T/main/.git/rebase-merge"

WT_GITDIR=$(git -C "$T/wt" rev-parse --path-format=absolute --git-dir); touch "$WT_GITDIR/MERGE_HEAD"
run "$T/wt"
ck "this worktree mid-merge: reported" "$(echo "$OUT" | grep -c 'this worktree has a MERGE_HEAD')" "1"
rm -f "$WT_GITDIR/MERGE_HEAD"

git clone -q "$T/origin.git" "$T/other" 2>/dev/null
(cd "$T/other" && git checkout -q dev && git config user.email t@t && git config user.name t && echo z > z && git add z && git commit -qm z && git push -q origin dev)
echo d > d; git add d; git commit -qm d; git fetch -q origin dev
run "$T/main"
ck "diverged dev: reported with counts" "$(echo "$OUT" | grep -c '1 unpushed commit(s) and is 1 behind')" "1"

OUT=$(AILANG_MISSION_STAGE=x CLAUDE_PROJECT_DIR="$T/main" bash "$HOOK" 2>&1)
ck "mission stage: silent" "$OUT" ""
OUT=$(AILANG_TASK_ID=x CLAUDE_PROJECT_DIR="$T/main" bash "$HOOK" 2>&1)
ck "coordinator task: silent" "$OUT" ""

rm -f "$HOME/.ailang/state/worktree-sweep.stamp"
AILANG_WORKTREE_SWEEP=0 CLAUDE_PROJECT_DIR="$T/main" bash "$HOOK" >/dev/null 2>&1; sleep 1
ck "AILANG_WORKTREE_SWEEP=0: no sweep" "$(swept)" "1"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ] && [ "$pass" -eq 9 ]
