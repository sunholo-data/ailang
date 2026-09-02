#!/usr/bin/env bash
# Arms for push_dev_on_stop.sh, against a real bare origin. No network, no real repo.
set -u
HOOK="$1"
W=$(mktemp -d); cd "$W" || exit 1
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }

git init -q --bare origin.git
git clone -q origin.git local; cd local
git config user.email t@t; git config user.name t
git checkout -q -b dev; echo a > a; git add a; git commit -qm init; git push -q origin dev
export CLAUDE_PROJECT_DIR="$W/local"
ORIGIN_SHA() { git -C "$W/origin.git" rev-parse dev; }

# A. clean — nothing ahead
out=$(bash "$HOOK" 2>&1); ck "A clean: no output" "$out" ""
ck "A clean: exit 0" "$?" "0"

# B. 2 ahead, 0 behind -> pushes
before=$(ORIGIN_SHA)
echo b > b; git add b; git commit -qm b; echo c > c; git add c; git commit -qm c
out=$(bash "$HOOK" 2>&1)
after=$(ORIGIN_SHA)
ck "B ahead-only: pushed (origin moved)" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"
ck "B ahead-only: reports 2" "$(echo "$out" | grep -c '2 unpushed')" "1"

# C. 1 ahead AND 1 behind -> refuses
git clone -q "$W/origin.git" "$W/other"; cd "$W/other"; git config user.email t@t; git config user.name t
git checkout -q dev; echo z > z; git add z; git commit -qm z; git push -q origin dev; cd "$W/local"
echo d > d; git add d; git commit -qm d
git fetch -q origin dev
ck "C setup: really diverged (behind ahead)" "$(git rev-list --left-right --count origin/dev...dev | tr -d '\t' )" "11"
before=$(ORIGIN_SHA)
out=$(bash "$HOOK" 2>&1)
after=$(ORIGIN_SHA)
ck "C diverged: did NOT push" "$before" "$after"
ck "C diverged: says not auto-pushing" "$(echo "$out" | grep -c 'Not auto-pushing')" "1"

# D. non-dev branch with commits ahead -> no-op
git checkout -q -b sprint/x; echo e > e; git add e; git commit -qm e
out=$(bash "$HOOK" 2>&1); ck "D non-dev branch: silent" "$out" ""
git checkout -q dev

# E. merge in flight -> no-op
touch "$(git rev-parse --git-dir)/MERGE_HEAD"
out=$(bash "$HOOK" 2>&1); ck "E merge in flight: silent" "$out" ""
rm -f "$(git rev-parse --git-dir)/MERGE_HEAD"

# G. origin unreachable -> LOUD, never silent (regression pin: a silent skip here
# re-opens the stranding hole; found live 2026-09-02 when a 10s fetch bound timed out).
git remote set-url origin "$W/does-not-exist.git"
out=$(bash "$HOOK" 2>&1)
ck "G unreachable origin: warns loudly" "$(echo "$out" | grep -c 'fetch failed twice')" "1"
ck "G unreachable origin: not silent" "$([ -n "$out" ] && echo nonempty || echo empty)" "nonempty"
git remote set-url origin "$W/origin.git"

# F. opt-out
out=$(AILANG_AUTOPUSH=0 bash "$HOOK" 2>&1); ck "F opt-out: silent" "$out" ""

echo "  ---- $pass passed, $fail failed"
rm -rf "$W"
[ "$fail" -eq 0 ]
