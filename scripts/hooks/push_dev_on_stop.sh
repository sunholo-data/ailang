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

# The log is shared across every clone that installs this hook (main checkout, ailang-docs,
# ailang-motoko, the separate ailang-world repo), so every line names its repo — otherwise a
# "pushed 2 commits" line is unattributable and the log stops being evidence.
REPO_NAME="$(basename "$ROOT")"
log() { printf '[%s] [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$REPO_NAME" "$*" >> "$LOG" 2>/dev/null; }

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
    log "REFUSED: $AHEAD ahead but $BEHIND behind — a merge is owed, needs a human"
    echo "⚠️  local dev has $AHEAD unpushed commit(s) AND is $BEHIND behind origin/dev."
    echo "   Not auto-pushing: this needs a real merge, and conflicts here land in the"
    echo "   mission charter/changelog where a careless resolution drops decision rows."
    echo "   Resolve with a merge, verify, then push."
    exit 0
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
trap 'rm -rf "$FMT_TMP"' EXIT
trap 'rm -rf "$FMT_TMP"; exit 0' HUP INT TERM

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
