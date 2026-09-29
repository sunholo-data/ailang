#!/usr/bin/env bash
# git_health.sh — SessionStart: say so when the shared checkout is in a bad git state, and
# sweep finished worktrees in the background. Silent when healthy. Local only (no fetch), so
# it costs well under a second; bash 3.2.
#
# WHY: on 2026-09-28 the main checkout sat mid-rebase from 19:12 to the next morning, with
# local dev both ahead of and behind origin. Every session that started in that time was told
# nothing, and the pile of 30 worktrees made the real problem hard to see. This hook makes
# both visible at the moment a session starts, where the next decision is made.
#
#   - a rebase/merge/cherry-pick/bisect in progress in the main checkout OR this worktree
#   - main-checkout dev diverged from last-known origin/dev (the Stop hook will try to rebase)
#   - at most every 6h, scripts/worktree_sweep.sh --apply runs detached in the background
#     (it removes only landed, clean, idle worktrees; log: ~/.ailang/state/worktree-sweep.log)
#
# Unattended runs (mission stages, coordinator tasks) get nothing: frozen input, no side jobs.
# AILANG_WORKTREE_SWEEP=0 disables the background sweep. Always exits 0.
[ -n "${AILANG_MISSION_STAGE:-}" ] && exit 0
[ -n "${AILANG_TASK_ID:-}" ] && exit 0

ROOT="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$ROOT" 2>/dev/null || exit 0
COMMON=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || exit 0
HERE_GITDIR=$(git rev-parse --path-format=absolute --git-dir 2>/dev/null)
MAIN=$(dirname "$COMMON")

in_flight() { # in_flight GITDIR LABEL
	local f
	for f in rebase-merge rebase-apply MERGE_HEAD CHERRY_PICK_HEAD BISECT_LOG; do
		if [ -e "$1/$f" ]; then
			echo "⚠️  GIT: $2 has a $f in progress (since $(perl -e '@s = stat shift or exit; @t = localtime $s[9]; printf "%04d-%02d-%02d %02d:%02d", $t[5]+1900, $t[4]+1, @t[3,2,1]' "$1/$f" 2>/dev/null))."
			echo "   If no session is resolving it, finish or abort it before anything else:"
			echo "   git -C \"$3\" status   # then --continue or --abort"
			return
		fi
	done
}
in_flight "$COMMON" "the main checkout ($MAIN)" "$MAIN"
[ "$HERE_GITDIR" != "$COMMON" ] && in_flight "$HERE_GITDIR" "this worktree" "$ROOT"

if git rev-parse -q --verify refs/remotes/origin/dev >/dev/null && git rev-parse -q --verify refs/heads/dev >/dev/null; then
	COUNTS=$(git rev-list --left-right --count origin/dev...dev 2>/dev/null)
	BEHIND=$(printf '%s' "$COUNTS" | awk '{print $1}'); AHEAD=$(printf '%s' "$COUNTS" | awk '{print $2}')
	if [ "${AHEAD:-0}" -gt 0 ] 2>/dev/null && [ "${BEHIND:-0}" -gt 0 ] 2>/dev/null; then
		echo "⚠️  GIT: local dev has $AHEAD unpushed commit(s) and is $BEHIND behind origin/dev (last fetch)."
		echo "   The Stop hook will try a guarded rebase + push; if it reports a conflict, resolve it then."
	fi
fi

if [ "${AILANG_WORKTREE_SWEEP:-1}" != "0" ] && [ -x "$ROOT/scripts/worktree_sweep.sh" ]; then
	STAMP="$HOME/.ailang/state/worktree-sweep.stamp"
	mkdir -p "$(dirname "$STAMP")" 2>/dev/null
	if [ ! -e "$STAMP" ] || perl -e 'exit((time - (stat shift)[9]) > 6*3600 ? 0 : 1)' "$STAMP" 2>/dev/null; then
		touch "$STAMP"
		( cd "$ROOT" && nohup /bin/bash scripts/worktree_sweep.sh --apply --quiet \
			>> "$HOME/.ailang/state/worktree-sweep.log" 2>&1 & ) >/dev/null 2>&1
	fi
fi
exit 0
