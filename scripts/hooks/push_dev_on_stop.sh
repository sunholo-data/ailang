#!/usr/bin/env bash
# push_dev_on_stop.sh — Stop hook: push local dev when it is a clean fast-forward ahead.
#
# WHY THIS EXISTS (measured 2026-09-02, Mark attended). CLAUDE.md tells sessions to commit
# straight to dev in the main checkout and never to branch there, but nothing in that
# workflow pushes — and mission-control Gate 1 forbids the loop from pulling or pushing the
# shared tree (it may hold a sibling's uncommitted work), so the loop cannot clean up after
# an attended session either. Commits therefore strand on local dev indefinitely: the sync
# that prompted this moved 25 of them, one session's worth of standalone work plus several
# days of attended commits, and the sibling clone ailang-docs was 58 behind with 1 stranded.
# The loop's own worktree -> PR pushes were never the problem (0 push failures in any
# mission log); the attended path was the whole gap.
#
# CONTRACT — FAST-FORWARD, OR A CLEAN AUTO-REBASE. Ahead AND behind used to be refused
# outright, and the refusal is what stranded work (2026-09-28: two commits sat unpushed, a
# hand-run `pull --rebase` then stopped on the changelog and left the shared checkout
# mid-rebase overnight). Now it rebases — but only when no uncommitted edit touches a file
# the rebase rewrites, never resolving a conflict (the 2026-09-02 V1-charter case, where a
# careless resolution eats decision rows, still goes to a human), and aborting on any
# failure so the checkout is never left half-done. AILANG_AUTOREBASE=0 restores refuse-only.
#
# Touches the working tree ONLY through that guarded rebase: no pull, no reset, no checkout,
# no branch, and no stash beyond --autostash of files the rebase does not touch.
# Always exits 0 — a sync problem must never stop a session from ending.
# Opt out with AILANG_AUTOPUSH=0. Portable to macOS bash 3.2; no GNU timeout on this rig.

set -u

[ "${AILANG_AUTOPUSH:-1}" = "0" ] && exit 0

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="${CLAUDE_PROJECT_DIR:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
LOG="$HOME/.ailang/state/autopush.log"
mkdir -p "$(dirname "$LOG")" 2>/dev/null

# Never let git block on a credential prompt in a headless session.
export GIT_TERMINAL_PROMPT=0

# The log is shared across every clone that installs this hook (main checkout, ailang-docs,
# ailang-motoko, the separate ailang-world repo), so every line names its repo — otherwise a
# "pushed 2 commits" line is unattributable and the log stops being evidence.
REPO_NAME="$(basename "$ROOT")"
log() { printf '[%s] [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$REPO_NAME" "$*" >> "$LOG" 2>/dev/null; }

# ONE exit trap for everything this hook creates: a second `trap ... EXIT` would silently
# replace the first and strand the auto-rebase lock.
LOCK=""; FMT_TMP=""
# cleanup runs only through the EXIT/signal traps; shellcheck calls that unreachable
# (SC2329 on 0.11+, SC2317 on the older CI shellcheck).
# shellcheck disable=SC2317,SC2329
cleanup() { [ -n "$FMT_TMP" ] && rm -rf "$FMT_TMP"; [ -n "$LOCK" ] && rmdir "$LOCK" 2>/dev/null; return 0; }

# perl alarm — the portable bound this repo already uses (macOS ships no timeout/gtimeout).
bounded() { local s="$1"; shift; perl -e 'alarm(shift @ARGV); exec @ARGV' "$s" "$@"; }

if ! cd "$ROOT" 2>/dev/null; then
    log "SKIP_ROOT: could not cd to resolved root: $ROOT"
    exit 0
fi
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    log "SKIP_NOT_GIT: resolved root is not a Git worktree: $ROOT"
    exit 0
fi

# An operation in flight owns the repo; do not race it — but SAY so. This check runs BEFORE
# the branch check on purpose: mid-rebase HEAD is detached, `branch --show-current` prints
# nothing, and the old order exited as "not on dev" without ever reaching here. On
# 2026-09-28 a `pull --rebase` stopped on a changelog conflict in the main checkout and sat
# half-done from 19:12 to the next morning while every session's Stop hook stayed silent.
GITDIR=$(git rev-parse --git-dir 2>/dev/null)
for f in rebase-merge rebase-apply MERGE_HEAD REBASE_HEAD CHERRY_PICK_HEAD BISECT_LOG; do
    if [ -e "$GITDIR/$f" ]; then
        SINCE=$(perl -e '@s = stat shift or exit; @t = localtime $s[9]; printf "%04d-%02d-%02d %02d:%02d", $t[5]+1900, $t[4]+1, @t[3,2,1]' "$GITDIR/$f" 2>/dev/null)
        log "IN_FLIGHT: $f present since ${SINCE:-unknown} — nothing pushed"
        echo "⚠️  $ROOT has a git operation in progress ($f, since ${SINCE:-unknown})."
        echo "   Nothing was pushed. If no session is resolving it, finish or abort it:"
        case "$f" in
            rebase-*|REBASE_HEAD) echo "   git -C \"$ROOT\" rebase --continue   # or: rebase --abort" ;;
            MERGE_HEAD) echo "   git -C \"$ROOT\" merge --continue    # or: merge --abort" ;;
            CHERRY_PICK_HEAD) echo "   git -C \"$ROOT\" cherry-pick --continue   # or: cherry-pick --abort" ;;
            BISECT_LOG) echo "   git -C \"$ROOT\" bisect reset" ;;
        esac
        exit 0
    fi
done

# Only ever act on dev itself. Mission worktrees sit on sprint/* or docs/* and are skipped
# here by construction — they land their work through PRs, which already works.
BRANCH=$(git branch --show-current 2>/dev/null)
[ "$BRANCH" = "dev" ] || exit 0

# Fetch is bounded, retried once, and its failure is LOUD. A silent skip here would
# re-open the exact hole this hook closes (Principle 2: no silent fallback on anything
# affecting integrity). 10s proved too tight in the wild — this repo carries ~55 worktrees
# and a cold fetch timed out at 10s on 2026-09-02, logging "skip" where nobody would see it.
if ! bounded 20 git fetch origin dev --quiet 2>/dev/null; then
    sleep 1
    if ! bounded 20 git fetch origin dev --quiet 2>/dev/null; then
        AHEAD_LOCAL=$(git rev-list --count origin/dev..dev 2>/dev/null || echo "?")
        log "fetch FAILED twice — cannot verify; $AHEAD_LOCAL commit(s) ahead of last-known origin"
        echo "⚠️  could not reach origin to check for unpushed work (fetch failed twice)."
        echo "   local dev is $AHEAD_LOCAL commit(s) ahead of the last-known origin/dev."
        echo "   Push manually so it does not strand: git push origin dev"
        exit 0
    fi
fi

COUNTS=$(git rev-list --left-right --count origin/dev...dev 2>/dev/null) || exit 0
BEHIND=$(printf '%s' "$COUNTS" | awk '{print $1}')
AHEAD=$(printf '%s' "$COUNTS" | awk '{print $2}')
[ -n "$AHEAD" ] || exit 0
[ "$AHEAD" -eq 0 ] 2>/dev/null && exit 0

if [ "$BEHIND" -gt 0 ] 2>/dev/null; then
    # Ahead AND behind. Refusing outright is what stranded work: the refusal went to a log,
    # the commits sat, and the next person to `pull --rebase` by hand hit a conflict and left
    # the shared checkout mid-rebase (2026-09-28). With changelog fragments the usual conflict
    # is gone, so try the rebase — under three rules that keep the shared tree safe:
    #   1. Never touch a sibling's uncommitted work: refuse if any dirty tracked file is one
    #      the rebase would rewrite (incoming commits or ours). Otherwise --autostash cannot
    #      conflict on re-apply, because the stashed files are untouched by the rebase.
    #   2. Never leave it half-done: ANY failure is `rebase --abort`, then verified.
    #   3. Never resolve a conflict: a textual conflict (the charter case) still needs a human.
    # AILANG_AUTOREBASE=0 restores refuse-only.
    refuse_diverged() {
        log "REFUSED: $AHEAD ahead, $BEHIND behind — $1"
        echo "⚠️  local dev has $AHEAD unpushed commit(s) AND is $BEHIND behind origin/dev."
        echo "   Not auto-pushing: $1."
        echo "   Rebase onto origin/dev (git -C \"$ROOT\" pull --rebase origin dev), resolve, then push."
        exit 0
    }
    [ "${AILANG_AUTOREBASE:-1}" = "0" ] && refuse_diverged "AILANG_AUTOREBASE=0"

    # One rebase at a time per checkout: two sessions ending together must not interleave.
    # A lock older than 10 min is from a hook that was killed mid-run; take it over.
    LOCK="$GITDIR/ailang-autorebase.lock"
    if [ -d "$LOCK" ] && perl -e 'exit((time - (stat shift)[9]) > 600 ? 0 : 1)' "$LOCK" 2>/dev/null; then
        log "stale auto-rebase lock (>10 min) removed"
        rmdir "$LOCK" 2>/dev/null
    fi
    mkdir "$LOCK" 2>/dev/null || { LOCK=""; refuse_diverged "another session's auto-rebase is running"; }
    trap cleanup EXIT

    BASE=$(git merge-base origin/dev dev 2>/dev/null) || refuse_diverged "no merge base with origin/dev"
    TOUCHED=$( { git diff --name-only "$BASE" origin/dev; git diff --name-only "$BASE" dev; } 2>/dev/null | sort -u)
    DIRTY=$( { git diff --name-only; git diff --cached --name-only; } 2>/dev/null | sort -u)
    OVERLAP=$(printf '%s\n' "$DIRTY" | grep -Fxf <(printf '%s\n' "$TOUCHED") 2>/dev/null | grep -v '^$' | tr '\n' ' ')
    [ -z "$OVERLAP" ] || refuse_diverged "uncommitted edits touch files the rebase would change: $OVERLAP"

    if bounded 60 git -c rebase.autoSquash=false rebase --autostash origin/dev >/dev/null 2>&1; then
        log "REBASED: $AHEAD commit(s) onto origin/dev ($BEHIND behind)"
        echo "↻ rebased $AHEAD local commit(s) onto origin/dev ($BEHIND new upstream)."
        COUNTS=$(git rev-list --left-right --count origin/dev...dev 2>/dev/null) || exit 0
        BEHIND=$(printf '%s' "$COUNTS" | awk '{print $1}')
        AHEAD=$(printf '%s' "$COUNTS" | awk '{print $2}')
        [ "$AHEAD" -eq 0 ] 2>/dev/null && { echo "   (all of them were already upstream — nothing to push)"; exit 0; }
    else
        CONFLICTS=$(git diff --name-only --diff-filter=U 2>/dev/null | tr '\n' ' ')
        bounded 30 git rebase --abort >/dev/null 2>&1
        if [ -e "$GITDIR/rebase-merge" ] || [ -e "$GITDIR/rebase-apply" ]; then
            log "ABORT_FAILED: rebase could not be aborted — checkout is mid-rebase"
            echo "🛑 auto-rebase failed AND could not be aborted: $ROOT is mid-rebase."
            echo "   Fix now: git -C \"$ROOT\" rebase --abort"
            exit 0
        fi
        refuse_diverged "auto-rebase conflicted${CONFLICTS:+ in $CONFLICTS}and was aborted cleanly"
    fi
fi

# Judge only the committed Go blobs that this push would add to origin/dev. The shared
# checkout may contain unrelated edits, so neither the working tree nor deleted/renamed-away
# paths are part of this integrity gate.
if ! command -v gofmt >/dev/null 2>&1; then
    log "REFUSED_FMT_TOOL: gofmt not found; refusing $AHEAD commit(s)"
    echo "⚠️  local dev has $AHEAD unpushed commit(s), but gofmt is not on PATH."
    echo "   Not auto-pushing: cannot verify committed Go formatting without gofmt."
    echo "   Restore gofmt, run make fmt, commit, and let the next Stop push it."
    exit 0
fi

FMT_TMP=$(mktemp -d "${TMPDIR:-/tmp}/ailang-autopush-fmt.XXXXXX" 2>/dev/null) || {
    log "REFUSED_FMT_CHECK: could not create formatting workspace; refusing $AHEAD commit(s)"
    echo "⚠️  local dev has $AHEAD unpushed commit(s), but its committed Go files could not be checked."
    echo "   Not auto-pushing: run make fmt, commit, and let the next Stop push it."
    exit 0
}
trap cleanup EXIT
trap 'cleanup; exit 0' HUP INT TERM

if ! bounded 10 git diff --name-only -z --diff-filter=ACM origin/dev..dev -- '*.go' > "$FMT_TMP/go-files" 2>/dev/null; then
    log "REFUSED_FMT_CHECK: could not list committed Go files; refusing $AHEAD commit(s)"
    echo "⚠️  local dev has $AHEAD unpushed commit(s), but its committed Go files could not be checked."
    echo "   Not auto-pushing: run make fmt, commit, and let the next Stop push it."
    exit 0
fi

BAD_GO_FILES=""
while IFS= read -r -d '' go_file; do
    if ! bounded 10 git show "dev:$go_file" > "$FMT_TMP/committed" 2>/dev/null; then
        BAD_GO_FILES="${BAD_GO_FILES}${go_file}
"
        continue
    fi
    if [ ! -s "$FMT_TMP/committed" ]; then
        BAD_GO_FILES="${BAD_GO_FILES}${go_file}
"
        continue
    fi
    if ! bounded 10 gofmt < "$FMT_TMP/committed" > "$FMT_TMP/formatted" 2>/dev/null; then
        BAD_GO_FILES="${BAD_GO_FILES}${go_file}
"
        continue
    fi
    cmp -s "$FMT_TMP/committed" "$FMT_TMP/formatted" || BAD_GO_FILES="${BAD_GO_FILES}${go_file}
"
done < "$FMT_TMP/go-files"

if [ -n "$BAD_GO_FILES" ]; then
    BAD_GO_LOG=$(printf '%s' "$BAD_GO_FILES" | tr '\n' ' ')
    log "REFUSED_FMT: committed Go files are not gofmt-clean: $BAD_GO_LOG"
    echo "⚠️  local dev has $AHEAD unpushed commit(s) with Go files that are not gofmt-clean:"
    printf '   %s\n' "$BAD_GO_FILES"
    echo "   Not auto-pushing: run make fmt, commit the result, and let the next Stop push it."
    exit 0
fi

if bounded 25 git push origin dev >/dev/null 2>&1; then
    log "pushed $AHEAD commit(s) to origin/dev"
    echo "✅ pushed $AHEAD unpushed commit(s) from local dev to origin/dev."
else
    log "push FAILED for $AHEAD commit(s) (auth, network, or a race)"
    echo "⚠️  local dev has $AHEAD unpushed commit(s); the auto-push failed."
    echo "   Push manually so the work does not strand: git push origin dev"
fi
exit 0
