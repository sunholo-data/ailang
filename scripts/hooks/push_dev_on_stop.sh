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
# CONTRACT — FAST-FORWARD ONLY. Ahead AND behind means a real merge is owed, and merges here
# need judgement: the 2026-09-02 sync conflicted on the V1 charter, where a careless
# resolution silently eats decision rows (the skill records one eating decision-ledger:end
# and an entire Goal block). So this refuses that case loudly and leaves it for a human.
#
# NEVER touches the working tree: no pull, no reset, no stash, no checkout, no branch.
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

log() { printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >> "$LOG" 2>/dev/null; }

# perl alarm — the portable bound this repo already uses (macOS ships no timeout/gtimeout).
bounded() { local s="$1"; shift; perl -e 'alarm(shift @ARGV); exec @ARGV' "$s" "$@"; }

cd "$ROOT" 2>/dev/null || exit 0
git rev-parse --git-dir >/dev/null 2>&1 || exit 0

# Only ever act on dev itself. Mission worktrees sit on sprint/* or docs/* and are skipped
# here by construction — they land their work through PRs, which already works.
BRANCH=$(git branch --show-current 2>/dev/null)
[ "$BRANCH" = "dev" ] || exit 0

# An operation in flight owns the repo; do not race it.
GITDIR=$(git rev-parse --git-dir 2>/dev/null)
for f in MERGE_HEAD REBASE_HEAD CHERRY_PICK_HEAD BISECT_LOG; do
    if [ -e "$GITDIR/$f" ]; then
        log "skip: $f present (operation in flight)"
        exit 0
    fi
done

bounded 10 git fetch origin dev --quiet 2>/dev/null || { log "skip: fetch failed or timed out"; exit 0; }

COUNTS=$(git rev-list --left-right --count origin/dev...dev 2>/dev/null) || exit 0
BEHIND=$(printf '%s' "$COUNTS" | awk '{print $1}')
AHEAD=$(printf '%s' "$COUNTS" | awk '{print $2}')
[ -n "$AHEAD" ] || exit 0
[ "$AHEAD" -eq 0 ] 2>/dev/null && exit 0

if [ "$BEHIND" -gt 0 ] 2>/dev/null; then
    log "REFUSED: $AHEAD ahead but $BEHIND behind — a merge is owed, needs a human"
    echo "⚠️  local dev has $AHEAD unpushed commit(s) AND is $BEHIND behind origin/dev."
    echo "   Not auto-pushing: this needs a real merge, and conflicts here land in the"
    echo "   mission charter/changelog where a careless resolution drops decision rows."
    echo "   Resolve with a merge, verify, then push."
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
