#!/usr/bin/env bash
# Arms for push_dev_on_stop.sh, against a real bare origin. No network, no real repo.
set -u
HOOK="$1"
W=$(mktemp -d "$PWD/.tmp-iter327.harness.XXXXXX") || exit 1
# The caller's log is READ ONLY here. An earlier draft of this arm SEEDED a sentinel into it
# (`printf ... > "$CALLER_LOG"`) and then asserted the sentinel was unchanged. That destroyed the
# real fleet log the first time it ran outside a sandbox: 92 lines of cross-clone push evidence
# replaced by one line. A test written to prove this harness never touches the caller's log must
# not be the thing that touches it. Observe, never write; ABSENT is a legitimate reading.
CALLER_HOME="$HOME"
CALLER_LOG="$CALLER_HOME/.ailang/state/autopush.log"
if [ -f "$CALLER_LOG" ]; then
    CALLER_LOG_SHA_BEFORE=$(shasum "$CALLER_LOG" | awk '{print $1}')
    CALLER_LOG_LINES_BEFORE=$(wc -l < "$CALLER_LOG" | tr -d ' ')
else
    CALLER_LOG_SHA_BEFORE=absent
    CALLER_LOG_LINES_BEFORE=absent
fi
mkdir -p "$W/home" || exit 1
export HOME="$W/home"
cd "$W" || exit 1
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }

git init -q --bare origin.git
git clone -q origin.git local; cd local || exit 1
git config user.email t@t; git config user.name t
git checkout -q -b dev; echo a > a; git add a; git commit -qm init; git push -q origin dev
export CLAUDE_PROJECT_DIR="$W/local"
ORIGIN_SHA() { git -C "$W/origin.git" rev-parse dev; }

# Q/R. The earliest precondition exits are quiet but leave distinct evidence.
PRIVATE_LOG="$W/home/.ailang/state/autopush.log"
q_before=$(grep -c 'SKIP_ROOT' "$PRIVATE_LOG" 2>/dev/null || true)
out=$(CLAUDE_PROJECT_DIR="$W/absent-root" bash "$HOOK" 2>&1); q_rc=$?
q_after=$(grep -c 'SKIP_ROOT' "$PRIVATE_LOG" 2>/dev/null || true)
ck "Q missing root: logged, stdout quiet" "$((q_after - q_before)):${q_rc}:$([ -z "$out" ] && echo quiet || echo noisy)" "1:0:quiet"

mkdir -p "$W/not-git" || exit 1
r_before=$(grep -c 'SKIP_NOT_GIT' "$PRIVATE_LOG" 2>/dev/null || true)
out=$(GIT_CEILING_DIRECTORIES="$W" CLAUDE_PROJECT_DIR="$W/not-git" bash "$HOOK" 2>&1); r_rc=$?
r_after=$(grep -c 'SKIP_NOT_GIT' "$PRIVATE_LOG" 2>/dev/null || true)
ck "R non-Git root: logged, stdout quiet" "$((r_after - r_before)):${r_rc}:$([ -z "$out" ] && echo quiet || echo noisy)" "1:0:quiet"
qr_rows=$(tail -2 "$PRIVATE_LOG" | sed -n 's/.*\(SKIP_[A-Z_]*\):.*/\1/p' | tr '\n' ' ')
ck "Q/R distinction assertion" "$qr_rows" "SKIP_ROOT SKIP_NOT_GIT "

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
git clone -q "$W/origin.git" "$W/other"; cd "$W/other" || exit 1; git config user.email t@t; git config user.name t
git checkout -q dev; echo z > z; git add z; git commit -qm z; git push -q origin dev; cd "$W/local" || exit 1
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
git clone -q "$W/origin.git" "$W/fmtlocal"; cd "$W/fmtlocal" || exit 1
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
git clone -q "$W/origin.git" "$W/seed"; cd "$W/seed" || exit 1
git config user.email t@t; git config user.name t; git checkout -q dev
printf 'package old\nfunc z( ){ }\n' > old.go
git add old.go; git commit -qm 'seed unformatted go'; git push -q origin dev
cd "$W/fmtlocal" || exit 1; git pull -q --ff-only origin dev
git rm -q old.go; git commit -qm 'delete unformatted go'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "K deleted unformatted Go: pushed" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"

# L. a committed Go blob with no trailing newline is not gofmt-clean.
git clone -q "$W/origin.git" "$W/no-newline"; cd "$W/no-newline" || exit 1
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/no-newline"
printf 'package nonewline' > no_newline.go
git add no_newline.go; git commit -qm 'go file without trailing newline'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "L no trailing newline: refused" "$([ "$before" = "$after" ] && echo stayed || echo moved)" "stayed"

# M. a committed Go blob with trailing blank lines is not gofmt-clean.
git clone -q "$W/origin.git" "$W/trailing-blank"; cd "$W/trailing-blank" || exit 1
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/trailing-blank"
printf 'package trailing\n\n\n' > trailing_blank.go
git add trailing_blank.go; git commit -qm 'go file with trailing blank lines'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "M trailing blank lines: refused" "$([ "$before" = "$after" ] && echo stayed || echo moved)" "stayed"

# O. a committed empty Go blob is a formatter error, not a clean file.
git clone -q "$W/origin.git" "$W/empty-go"; cd "$W/empty-go" || exit 1
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/empty-go"
: > empty.go
git add empty.go; git commit -qm 'empty go file'
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "O empty committed Go: refused" "$([ "$before" = "$after" ] && echo stayed):$(echo "$out" | grep -c 'run make fmt')" "stayed:1"

# P. Git's quoted presentation of a non-ASCII pathname must not become a literal path.
git clone -q "$W/origin.git" "$W/non-ascii"; cd "$W/non-ascii" || exit 1
git config user.email t@t; git config user.name t; git checkout -q dev
export CLAUDE_PROJECT_DIR="$W/non-ascii"
non_ascii_file="nonascii_$(printf '\303\245').go"
printf 'package nonascii\n' > "$non_ascii_file"
git add "$non_ascii_file"; git commit -qm 'non-ASCII Go pathname'
quoted_path=$(git -c core.quotepath=true diff --name-only HEAD^..HEAD)
ck "P setup: human Git output is quoted" "$([ "${quoted_path#\"}" != "$quoted_path" ] && echo quoted || echo literal)" "quoted"
before=$(ORIGIN_SHA); out=$(bash "$HOOK" 2>&1); after=$(ORIGIN_SHA)
ck "P non-ASCII Go pathname: pushed" "$([ "$before" != "$after" ] && echo yes || echo no)" "yes"

ck "N harness HOME: private log populated" "$([ -s "$PRIVATE_LOG" ] && grep -c '\[fmtlocal\]' "$PRIVATE_LOG" || echo 0)" "4"
if [ -f "$CALLER_LOG" ]; then
    CALLER_LOG_SHA_AFTER=$(shasum "$CALLER_LOG" | awk '{print $1}')
    CALLER_LOG_LINES_AFTER=$(wc -l < "$CALLER_LOG" | tr -d ' ')
else
    CALLER_LOG_SHA_AFTER=absent
    CALLER_LOG_LINES_AFTER=absent
fi
ck "N harness HOME: caller log unchanged" "${CALLER_LOG_SHA_AFTER}:${CALLER_LOG_LINES_AFTER}" "${CALLER_LOG_SHA_BEFORE}:${CALLER_LOG_LINES_BEFORE}"

echo "  ---- $pass passed, $fail failed"
rm -rf "$W"
[ "$fail" -eq 0 ]
