#!/usr/bin/env bash
# worktree_sweep.sh — remove worktrees (and local branches) whose work has landed (bash 3.2).
#
# WHY: nothing ever removed a worktree. On 2026-09-28 this clone carried 30 worktrees and 33
# branches; every non-infrastructure one removed that day was a leftover whose PR had already
# merged (agent-* worktrees two days after merge, dead sessions' /tmp scratchpads, coordinator
# evaluators of failed tasks). The pile hid the two commits that really were stranded.
#
# A worktree is REMOVED only when ALL hold — anything unknown means KEEP:
#   1. not the main checkout, and not under a protected prefix (mission driver pins, the A/B
#      and nightly checkouts, coordinator task worktrees — each has its own lifecycle);
#   2. no live process has its cwd inside it (lsof; if lsof fails, nothing is removed);
#   3. idle: its index and HEAD log are older than --min-age-hours (default 2), so a worktree
#      created a minute ago at origin/dev is not mistaken for a finished one;
#   4. clean: no modified, staged or untracked files (the committed hook log excepted);
#   5. landed: HEAD is an ancestor of origin/dev, OR its branch has a MERGED PR whose head is
#      exactly HEAD (squash merges defeat ancestry — `git cherry` reports them unmerged).
# A lock is respected, except Claude Code agent locks (.claude/worktrees/agent-*), which outlive
# the agent; rule 2 still protects a running one.
#
# Local branches with no worktree are deleted by the same rule 5 (never dev/main/prod or the
# checked-out branch). Every removal is logged with its SHA to ~/.ailang/state/worktree-sweep.log:
# recover with `git branch <name> <sha>`.
#
# Usage: scripts/worktree_sweep.sh [--apply] [--min-age-hours N] [--quiet]
#   default is a dry run that prints the verdict for every worktree.
#   AILANG_SWEEP_PROTECT=/a:/b adds protected prefixes.
set -u

APPLY=0; MIN_AGE_H=2; QUIET=0
while [ $# -gt 0 ]; do
	case "$1" in
		--apply) APPLY=1 ;;
		--min-age-hours) shift; MIN_AGE_H="${1:-2}" ;;
		--quiet) QUIET=1 ;;
		-h|--help) sed -n '2,29p' "$0"; exit 0 ;;
		*) echo "unknown argument: $1" >&2; exit 2 ;;
	esac
	shift
done

LOG="$HOME/.ailang/state/worktree-sweep.log"
mkdir -p "$(dirname "$LOG")" 2>/dev/null
say() { [ "$QUIET" = 1 ] || echo "$*"; }
log() { printf '[%s] [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$(basename "$MAIN")" "$*" >> "$LOG" 2>/dev/null; }
bounded() { local s="$1"; shift; perl -e 'alarm(shift @ARGV); exec @ARGV' "$s" "$@"; }
mtime() { perl -e '@s = stat shift or exit; print $s[9]' "$1" 2>/dev/null; }

COMMON=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || { echo "not in a git repository" >&2; exit 2; }
MAIN=$(dirname "$COMMON")
cd "$MAIN" || exit 2

PROTECT="$HOME/.ailang-driver-pin/:$HOME/.ailang-ab/:$HOME/.ailang-nightly/:$HOME/.ailang/state/worktrees/${AILANG_SWEEP_PROTECT:+:$AILANG_SWEEP_PROTECT}"
protected() {
	local IFS=':' p
	for p in $PROTECT; do
		[ -n "$p" ] || continue
		# git reports physical paths; compare against the physical prefix too.
		[ -d "$p" ] && p=$(cd "$p" 2>/dev/null && pwd -P)
		case "$1/" in "${p%/}/"*) return 0 ;; esac
	done
	return 1
}

WORK=$(mktemp -d "${TMPDIR:-/tmp}/wt-sweep.XXXXXX") || exit 2
trap 'rm -rf "$WORK"' EXIT

bounded 30 git fetch -q origin dev 2>/dev/null || say "note: fetch failed; judging against last-known origin/dev"
git rev-parse -q --verify origin/dev >/dev/null || { echo "no origin/dev ref — refusing to judge anything" >&2; exit 2; }

# Merged PRs: "<branch> <head sha>" per line. Without gh only rule 5's ancestry arm applies.
if command -v gh >/dev/null 2>&1 && bounded 30 gh pr list --state merged --limit 400 \
	--json headRefName,headRefOid --jq '.[] | "\(.headRefName) \(.headRefOid)"' > "$WORK/merged" 2>/dev/null; then :
else
	: > "$WORK/merged"; say "note: gh unavailable; squash-merged branches will be kept"
fi

# Every live process cwd. If lsof fails, sweep nothing: a missing answer is not "idle".
if ! bounded 30 lsof -n -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' > "$WORK/cwds" || [ ! -s "$WORK/cwds" ]; then
	echo "lsof gave no process list — refusing to remove anything" >&2; exit 2
fi
busy() { grep -qE "^$(printf '%s' "$1" | sed 's/[][\.*^$/]/\\&/g')(/|$)" "$WORK/cwds"; }

landed() { # landed SHA BRANCH -> 0 if on origin/dev or merged by PR at exactly SHA
	git merge-base --is-ancestor "$1" origin/dev 2>/dev/null && { REASON="on origin/dev"; return 0; }
	[ -n "$2" ] && grep -qxF "$2 $1" "$WORK/merged" && { REASON="PR merged at this commit"; return 0; }
	return 1
}

NOW=$(date +%s); MIN_AGE=$((MIN_AGE_H * 3600)); REMOVED=0; KEPT=0
git worktree list --porcelain > "$WORK/list"
git worktree list --porcelain | awk '/^branch /{sub("refs/heads/","",$2); print $2}' > "$WORK/checked_out"

# Records are blank-line separated: worktree / HEAD / branch|detached / locked [reason].
awk 'BEGIN{RS="";FS="\n"} {p=h=b=l=""; for(i=1;i<=NF;i++){ if($i~/^worktree /)p=substr($i,10); else if($i~/^HEAD /)h=substr($i,6); else if($i~/^branch /){b=substr($i,8); sub("refs/heads/","",b)} else if($i~/^locked/)l="locked"} print p "|" h "|" b "|" l}' "$WORK/list" > "$WORK/records"

# '|' not tab: tab is IFS whitespace, so `read` would collapse an empty branch field and
# shift "locked" into it (a detached, locked worktree then never got unlocked).
while IFS='|' read -r path sha branch lock; do
	[ "$path" = "$MAIN" ] && continue
	verdict=""
	if protected "$path"; then verdict="KEEP protected path"
	elif [ ! -d "$path" ]; then verdict="PRUNE directory gone"
	elif busy "$path"; then verdict="KEEP a process is running inside"
	elif [ -n "$lock" ] && case "$path" in */.claude/worktrees/agent-*) false ;; *) true ;; esac; then verdict="KEEP locked"
	else
		gd=$(git -C "$path" rev-parse --path-format=absolute --git-dir 2>/dev/null)
		newest=0
		for f in "$gd/index" "$gd/logs/HEAD" "$gd/HEAD"; do t=$(mtime "$f"); [ -n "$t" ] && [ "$t" -gt "$newest" ] && newest=$t; done
		if [ $((NOW - newest)) -lt "$MIN_AGE" ]; then verdict="KEEP active in the last ${MIN_AGE_H}h"
		elif [ -n "$(git -C "$path" --no-optional-locks status --porcelain 2>/dev/null | grep -v ' \.claude/fmt_hook_events\.jsonl$')" ]; then verdict="KEEP uncommitted changes"
		elif landed "$sha" "$branch"; then verdict="REMOVE $REASON"
		else verdict="KEEP work not on origin/dev"
		fi
	fi
	say "$(printf '%-6s %s  [%s] %s' "${verdict%% *}" "$path" "${branch:-detached ${sha:0:9}}" "${verdict#* }")"
	case "$verdict" in
		REMOVE*)
			if [ "$APPLY" = 1 ]; then
				[ -n "$lock" ] && git worktree unlock "$path" 2>/dev/null
				git -C "$path" checkout -q -- .claude/fmt_hook_events.jsonl 2>/dev/null
				if git worktree remove "$path" 2>/dev/null; then
					log "removed worktree $path branch=${branch:-none} sha=$sha (${verdict#* })"
					REMOVED=$((REMOVED + 1))
					if [ -n "$branch" ] && git branch -D "$branch" >/dev/null 2>&1; then
						log "deleted branch $branch sha=$sha"
					fi
				else
					log "remove FAILED for $path"; say "       ↳ remove failed; left in place"
				fi
			fi ;;
		*) KEPT=$((KEPT + 1)) ;;
	esac
done < "$WORK/records"

# Local branches with no worktree, landed by the same rule.
git worktree list --porcelain | awk '/^branch /{sub("refs/heads/","",$2); print $2}' > "$WORK/checked_out"
git for-each-ref --format='%(refname:short) %(objectname)' refs/heads | while read -r b s; do
	case "$b" in dev|main|prod) continue ;; esac
	grep -qxF "$b" "$WORK/checked_out" && continue
	landed "$s" "$b" || continue
	say "$(printf '%-6s branch %s (%s)' REMOVE "$b" "$REASON")"
	if [ "$APPLY" = 1 ] && git branch -D "$b" >/dev/null 2>&1; then log "deleted branch $b sha=$s ($REASON)"; fi
done

[ "$APPLY" = 1 ] && git worktree prune 2>/dev/null
if [ "$APPLY" = 1 ]; then say "swept: removed $REMOVED worktree(s), kept $KEPT — log: $LOG"
else say "dry run: re-run with --apply to remove the REMOVE rows (kept $KEPT)"; fi
exit 0
