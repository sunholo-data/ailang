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

# Use a fresh checkout for formatting arms; C intentionally left the original diverged.
git clone -q "$W/origin.git" "$W/fmtlocal"; cd "$W/fmtlocal"
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/fmtlocal"

# H. an unformatted committed Go blob refuses the push.
before=$(ORIGIN_SHA)
printf 'package bad\nfunc x( ){ }\n' > bad.go
git add bad.go; git commit -qm 'unformatted go'
out=$(bash "$HOOK" 2>&1)
after=$(ORIGIN_SHA)
ck "H unformatted commit: refused with guidance" "$([ "$before" = "$after" ] && echo stayed):$(echo "$out" | grep -c 'run make fmt')" "stayed:1"

# I. formatting that same ahead-only commit allows it to move origin.
gofmt -w bad.go; git add bad.go; git commit --amend -qm 'formatted go'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "I formatted commit: pushed" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"

# J. an uncommitted unformatted Go file is outside the committed-content gate.
echo clean > clean.txt; git add clean.txt; git commit -qm 'clean non-go commit'
printf 'package dirty\nfunc y( ){ }\n' > dirty.go
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "J uncommitted unformatted Go: pushed" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"
rm -f dirty.go

# K. deleting an unformatted Go file is allowed: seed it directly, then test the hook.
git clone -q "$W/origin.git" "$W/seed"; cd "$W/seed"
git config user.email t@t; git config user.name t; git checkout -q dev
printf 'package old\nfunc z( ){ }\n' > old.go
git add old.go; git commit -qm 'seed unformatted go'; git push -q origin dev
cd "$W/fmtlocal"; git pull -q --ff-only origin dev
git rm -q old.go; git commit -qm 'delete unformatted go'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "K deleted unformatted Go: pushed" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"

# L. a committed Go blob with no trailing newline is not gofmt-clean.
git clone -q "$W/origin.git" "$W/no-newline"; cd "$W/no-newline"
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/no-newline"
printf 'package nonewline' > no_newline.go
git add no_newline.go; git commit -qm 'go file without trailing newline'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "L no trailing newline: refused" "$([ "$before" = "$after" ] && echo stayed || echo moved)" "stayed"

# M. a committed Go blob with trailing blank lines is not gofmt-clean.
git clone -q "$W/origin.git" "$W/trailing-blank"; cd "$W/trailing-blank"
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/trailing-blank"
printf 'package trailing\n\n\n' > trailing_blank.go
git add trailing_blank.go; git commit -qm 'go file with trailing blank lines'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "M trailing blank lines: refused" "$([ "$before" = "$after" ] && echo stayed || echo moved)" "stayed"

echo "  ---- $pass passed, $fail failed"
rm -rf "$W"
[ "$fail" -eq 0 ]
