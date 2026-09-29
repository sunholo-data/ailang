#!/usr/bin/env bash
# Self-test for worktree_sweep.sh (bash 3.2): a throwaway repo + bare origin, a fake HOME, and
# shims for gh (the merged-PR list) and lsof (the live-process list). Never touches a real repo.
set -u
SWEEP="$(cd "$(dirname "$0")" && pwd)/worktree_sweep.sh"
T=$(mktemp -d "${TMPDIR:-/tmp}/wt-sweep-test.XXXXXX") || exit 1
# git and lsof report physical paths; resolve the /tmp and /var symlinks up front.
T=$(cd "$T" && pwd -P) || exit 1
trap 'rm -rf "$T"' EXIT
export HOME="$T/home"; mkdir -p "$HOME" "$T/bin"
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  ok   — $1"; pass=$((pass+1)); else echo "  FAIL — $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }

# Shims: gh prints $T/merged; lsof prints "n<cwd>" for each line of $T/cwds, or fails.
printf '#!/bin/sh\ncat "%s/merged"\n' "$T" > "$T/bin/gh"
printf '#!/bin/sh\n[ -f "%s/lsof-fail" ] && exit 1\necho p1\nsed "s/^/n/" "%s/cwds"\n' "$T" "$T" > "$T/bin/lsof"
chmod +x "$T/bin/gh" "$T/bin/lsof"; export PATH="$T/bin:$PATH"
: > "$T/merged"; echo "/nowhere" > "$T/cwds"

git init -q --bare "$T/origin.git"
git clone -q "$T/origin.git" "$T/main" 2>/dev/null; cd "$T/main" || exit 1
git config user.email t@t; git config user.name t
git checkout -q -b dev; echo a > a; git add a; git commit -qm init; git push -q origin dev

g() { git -C "$T/main" "$@"; }
age_out() { # make a worktree look idle for a day
	local gd; gd=$(git -C "$1" rev-parse --path-format=absolute --git-dir)
	touch -t "$(date -v-1d +%Y%m%d%H%M 2>/dev/null || date -d '1 day ago' +%Y%m%d%H%M)" "$gd/index" "$gd/HEAD" "$gd/logs/HEAD" 2>/dev/null
}
mkwt() { g worktree add -q "$@" >/dev/null 2>&1; }
exists() { [ -d "$1" ] && echo yes || echo no; }
run() { OUT=$(cd "$T/main" && /bin/bash "$SWEEP" "$@" 2>&1); RC=$?; }

echo "worktree sweep:"
mkwt --detach "$T/landed" origin/dev;                              age_out "$T/landed"
mkwt -b wip "$T/wip" origin/dev; (cd "$T/wip" && echo w > w && git add w && git commit -qm wip); age_out "$T/wip"
mkwt -b squashed "$T/squashed" origin/dev; (cd "$T/squashed" && echo s > s && git add s && git commit -qm s); age_out "$T/squashed"
echo "squashed $(git -C "$T/squashed" rev-parse HEAD)" > "$T/merged"
mkwt --detach "$T/dirty" origin/dev; echo junk > "$T/dirty/untracked"; age_out "$T/dirty"
mkwt --detach "$T/busy" origin/dev; age_out "$T/busy"; echo "$T/busy/sub" > "$T/cwds"
mkdir -p "$HOME/.ailang-driver-pin"; mkwt --detach "$HOME/.ailang-driver-pin/v1" origin/dev; age_out "$HOME/.ailang-driver-pin/v1"
mkwt --detach "$T/fresh" origin/dev
mkwt --detach "$T/locked" origin/dev; g worktree lock "$T/locked"; age_out "$T/locked"
mkdir -p "$T/main/.claude/worktrees"; mkwt --detach "$T/main/.claude/worktrees/agent-x" origin/dev
g worktree lock "$T/main/.claude/worktrees/agent-x"; age_out "$T/main/.claude/worktrees/agent-x"
g branch old-landed origin/dev; g branch unlanded wip

# (1) Dry run: verdicts, nothing removed.
run
ck "dry run removes nothing" "$(exists "$T/landed"):$(exists "$T/squashed")" "yes:yes"
ck "dry run: landed detached -> REMOVE" "$(echo "$OUT" | grep -c "^REMOVE $T/landed .*on origin/dev")" "1"

# (2) lsof failing: refuse everything, loudly.
touch "$T/lsof-fail"; run --apply; rm -f "$T/lsof-fail"
ck "lsof failure: exit 2, nothing removed" "$RC:$(exists "$T/landed")" "2:yes"

# (3) Apply.
run --apply
ck "landed + idle + clean -> removed" "$(exists "$T/landed")" "no"
ck "unpushed commit -> kept" "$(exists "$T/wip"):$(echo "$OUT" | grep -c "$T/wip .*work not on origin/dev")" "yes:1"
ck "squash-merged PR at HEAD -> removed, branch deleted" "$(exists "$T/squashed"):$(g branch --list squashed | wc -l | tr -d ' ')" "no:0"
# The verdict, not just survival: `git worktree remove` would refuse a dirty tree anyway, so
# checking existence alone could not see this guard break.
ck "untracked file -> kept" "$(exists "$T/dirty"):$(echo "$OUT" | grep -c "$T/dirty .*uncommitted changes")" "yes:1"
ck "process inside -> kept" "$(exists "$T/busy")" "yes"
ck "protected mission pin -> kept" "$(exists "$HOME/.ailang-driver-pin/v1")" "yes"
ck "created just now -> kept" "$(exists "$T/fresh"):$(echo "$OUT" | grep -c "$T/fresh .*active in the last")" "yes:1"
ck "user lock -> kept" "$(exists "$T/locked")" "yes"
ck "stale agent lock -> removed" "$(exists "$T/main/.claude/worktrees/agent-x")" "no"
ck "landed local branch without worktree -> deleted" "$(g branch --list old-landed | wc -l | tr -d ' ')" "0"
ck "unlanded branch and dev never deleted" "$(g branch --list unlanded dev | wc -l | tr -d ' ')" "2"
ck "every removal logged with its sha" "$(grep -c 'removed worktree .* sha=[0-9a-f]\{40\}' "$HOME/.ailang/state/worktree-sweep.log")" "3"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ] && [ "$pass" -eq 15 ]
