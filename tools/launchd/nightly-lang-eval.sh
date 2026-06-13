#!/usr/bin/env bash
# nightly-lang-eval.sh — weekly multi-LANGUAGE comparison on the local rig.
#
# Fills the language-comparison gap that cloud cost made prohibitive: runs the
# local Qwen (free) across ailang + python + javascript + go on every benchmark
# that supports all four. This is the "AILANG vs every language" data the
# published leaderboard is otherwise missing.
#
# Harness: opencode (validated). pi/motoko are cross-harness follow-ups once
# their local-ollama config is sorted (pi needs an ollama provider block; motoko
# hangs — see the agent-harness-instability task).
#
# Slow run (~39 benchmarks x 4 langs = ~156 agent loops on local Qwen, a few
# hours), so schedule WEEKLY, not nightly. Output feeds a dated baseline dir;
# the dashboard wiring is a manual `update_dashboard.sh` step for now.
set -uo pipefail   # NOT -e: a single benchmark failure must not abort the sweep

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO" || exit 1
# shellcheck source=tools/launchd/rig-lock.sh
source "$(dirname "$0")/rig-lock.sh"
LOG=/tmp/ailang-nightly-lang-eval.log
DATE=$(date +%Y%m%d)
RESULTS_DIR="/tmp/lang_eval_${DATE}"
MODEL="${LANG_EVAL_MODEL:-opencode-qwen3-5-35b-a3b-mxfp8}"
LANGS="${LANG_EVAL_LANGS:-ailang,python,javascript,go}"

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG"; }

# Ensure ollama is reachable.
if ! curl -s --max-time 3 http://localhost:11434/api/version >/dev/null 2>&1; then
    log "ollama not reachable — skipping language eval (retry next week)"
    ailang messages send controlplane "Weekly language eval skipped: ollama unreachable. Check rig-watchdog." \
        --title "Lang eval skipped" --from "lang-eval" 2>/dev/null || true
    exit 0
fi

# Derive the benchmark set: every benchmark whose `languages:` lists all 4 langs.
BENCH=$(for f in benchmarks/*.yml; do
    L=$(grep -E '^languages:' "$f" 2>/dev/null)
    echo "$L" | grep -q 'ailang'     || continue
    echo "$L" | grep -q 'python'     || continue
    echo "$L" | grep -q 'javascript' || continue
    echo "$L" | grep -qE '\bgo\b'    || continue
    basename "$f" .yml
done | paste -sd, -)
N=$(echo "$BENCH" | tr ',' '\n' | grep -c .)
if [[ "$N" -eq 0 ]]; then
    log "ERROR: no benchmarks support all 4 languages"
    exit 1
fi

log "acquiring rig lock (waits for any nightly-eval / os-rotation-filler to finish)…"
rig_lock_acquire wait
log "rig lock acquired"

log "=== Weekly LANGUAGE eval ==="
log "model: $MODEL   langs: $LANGS   benchmarks: $N   output: $RESULTS_DIR"

ailang eval-suite --agent \
    --models "$MODEL" \
    --benchmarks "$BENCH" \
    --langs "$LANGS" \
    --microrag on \
    --parallel 1 \
    --output "$RESULTS_DIR" 2>&1 | tee -a "$LOG"

# Per-language summary.
SUMMARY=""
for lang in ailang python javascript go; do
    p=0; t=0
    for jf in "$RESULTS_DIR"/agent/*.json; do
        [[ -e "$jf" ]] || continue
        l=$(jq -r '.lang // "?"' "$jf" 2>/dev/null)
        [[ "$l" == "$lang" ]] || continue
        t=$((t+1))
        ok=$(jq -r 'if .compile_ok and .runtime_ok and .stdout_ok then 1 else 0 end' "$jf" 2>/dev/null)
        p=$((p+ok))
    done
    [[ "$t" -gt 0 ]] && SUMMARY="${SUMMARY}${lang}=${p}/${t}  "
done
log "RESULT: $SUMMARY"

ailang messages send controlplane \
    "Weekly language eval ($DATE): $MODEL — $SUMMARY (set=$N benchmarks, $LANGS). Dir: $RESULTS_DIR." \
    --title "Weekly language eval: $SUMMARY" --from "lang-eval" 2>/dev/null || true

# Publish to the OS/Local dashboard (M-EVAL-OS-CONTINUOUS-ROTATION): emit the
# static JSON the OSLocalLeaderboard component reads, then commit+push just that
# file so the docusaurus deploy refreshes the page. Scoped to the single file so
# it's safe regardless of other working-tree state.
if ailang eval-publish "$DATE" --rotation "$RESULTS_DIR" \
        --os-json docs/static/benchmarks/os/latest.json >>"$LOG" 2>&1; then
    # Stage first, THEN check the staged diff — `git diff --quiet -- <file>` ignores
    # untracked files, so the first publish (file not yet in git) would silently skip
    # the commit. `git add` + --cached detects both new and modified files.
    git add docs/static/benchmarks/os/latest.json 2>>"$LOG" || true
    if ! git diff --cached --quiet -- docs/static/benchmarks/os/latest.json 2>/dev/null; then
        git commit -q -m "data(os): refresh OS/Local leaderboard from lang eval $DATE" \
            -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
        # Auto-push is opt-in (safe default): set OS_FILLER_PUSH=1 to publish.
        if [ "${OS_FILLER_PUSH:-0}" = "1" ]; then
            git pull --rebase --autostash origin dev >>"$LOG" 2>&1 || true
            git push origin dev >>"$LOG" 2>&1 && log "published OS/Local JSON + pushed" \
                || log "OS/Local JSON committed; push failed (retry next run)"
        else
            log "OS/Local JSON committed locally (auto-push OFF — set OS_FILLER_PUSH=1)"
        fi
    else
        log "OS/Local JSON unchanged — nothing to publish"
    fi
else
    log "eval-publish failed — see $LOG"
fi

log "done"
