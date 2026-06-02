#!/usr/bin/env bash
# nightly-eval.sh — nightly regression-guard smoke run on the local rig.
#
# Runs the canonical smoke tier (tier: smoke, 17 benchmarks) against the
# accuracy-first local Qwen model (opencode-qwen3-5-35b-a3b-mxfp8).
# Feeds persistent failures to the design-doc-creator inbox via the
# triage-router (m-msg-triage-router) so regressions become design docs
# automatically.
#
# Called by dev.ailang.nightly-eval.plist at 03:00 daily.
#
# Installation:
#   cp tools/launchd/dev.ailang.nightly-eval.plist ~/Library/LaunchAgents/
#   launchctl load ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist
#
# Manual one-shot:
#   AILANG_NIGHTLY_EVAL_DRY_RUN=1 tools/launchd/nightly-eval.sh

set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO"

LOG=/tmp/ailang-nightly-eval.log
RESULTS_DIR="/tmp/nightly_eval_$(date +%Y%m%d)"
DATE=$(date +%Y-%m-%d)

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG"; }

log "=== nightly eval started (${DATE}) ==="

# Ensure ollama is running (rig-watchdog usually handles this, but be safe)
if ! curl -s --max-time 3 http://localhost:11434/api/version >/dev/null 2>&1; then
    log "ollama not reachable — skipping eval (will retry tomorrow)"
    ailang messages send controlplane "Nightly eval skipped: ollama not reachable at 03:00. Check rig-watchdog." \
        --title "Nightly eval skipped (${DATE})" --from "nightly-eval" 2>/dev/null || true
    exit 0
fi

# Derive smoke benchmarks from tier tags (stays in sync automatically)
SMOKE=$(grep -l 'tier: smoke' benchmarks/*.yml 2>/dev/null \
    | xargs -n1 basename | sed 's/\.yml$//' | sort | paste -sd, -)

if [[ -z "$SMOKE" ]]; then
    log "ERROR: could not derive smoke set — no tier:smoke benchmarks found"
    exit 1
fi

log "smoke tier: $(echo $SMOKE | tr ',' '\n' | wc -l | tr -d ' ') benchmarks"
log "output: $RESULTS_DIR"

# Dry-run mode for testing the plist without spending GPU time
if [[ "${AILANG_NIGHTLY_EVAL_DRY_RUN:-0}" == "1" ]]; then
    log "DRY RUN — exiting without running eval"
    exit 0
fi

# Run the eval (--trials 2 for regression confidence; serial to avoid GPU contention)
ailang eval-suite --agent \
    --models opencode-qwen3-5-35b-a3b-mxfp8 \
    --benchmarks "$SMOKE" \
    --langs ailang \
    --output "$RESULTS_DIR" \
    --parallel 1 \
    --trials 2 >> "$LOG" 2>&1

PASS=$(grep -oE '[0-9]+/[0-9]+ passed' "$LOG" 2>/dev/null | tail -1 | cut -d/ -f1 || echo "?")
TOTAL=$(grep -oE '[0-9]+/[0-9]+ passed' "$LOG" 2>/dev/null | tail -1 | cut -d/ -f2 | cut -d' ' -f1 || echo "?")
RATE=$(grep -oE '\([0-9.]+%\)' "$LOG" 2>/dev/null | tail -1 | tr -d '()' || echo "?%")

log "result: ${PASS}/${TOTAL} (${RATE})"

# Feed failures to design-doc-creator inbox if ≥2 consecutive trials failed
# (i.e., passes=0 in the deduplicated latest results)
FAILURES=$(python3 - <<'PY'
import json, glob, os
from collections import defaultdict

# Load newest result per benchmark (trials mode may have multiple files)
latest = {}
for f in glob.glob("${RESULTS_DIR}/agent/*.json".replace("${RESULTS_DIR}", os.environ.get("RESULTS_DIR",""))):
    key = os.path.basename(f).rsplit("_", 1)[0]
    ts  = int(os.path.basename(f).rsplit("_", 1)[1][:-5])
    if key not in latest or ts > latest[key][0]:
        latest[key] = (ts, f)

# Collect benchmarks that failed in all trials
trials = defaultdict(list)
for key,(ts,f) in latest.items():
    try: d = json.load(open(f))
    except: continue
    bench = key.split("_ailang_")[0]
    ok = d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")
    trials[bench].append((ok, d.get("error_category") or "—"))

persistent = []
for bench, results in trials.items():
    if all(not ok for ok,_ in results):
        cats = list({c for _,c in results})
        persistent.append(f"{bench} [{','.join(cats)}]")
print("\n".join(sorted(persistent)))
PY
RESULTS_DIR="$RESULTS_DIR"
)

if [[ -n "$FAILURES" ]]; then
    COUNT=$(echo "$FAILURES" | wc -l | tr -d ' ')
    log "persistent failures (${COUNT}): $FAILURES"

    # Send one message per failure to design-doc-creator intake
    # (triage-router will route categorized bug reports through the approval chain)
    while IFS= read -r failure; do
        BENCH=$(echo "$failure" | cut -d' ' -f1)
        CATS=$(echo "$failure" | grep -oE '\[.*\]' || echo "[unknown]")
        ailang messages send design-doc-creator \
            "Nightly eval regression: benchmark '${BENCH}' failed both trials on ${DATE}.
Error category: ${CATS}
Model: opencode-qwen3-5-35b-a3b-mxfp8 (local, tier:smoke)
Results: ${RESULTS_DIR}
This may indicate a prompt gap, stdlib regression, or model-capability boundary.
If it passed in a previous nightly run, this is a regression. If it has never passed,
it is a known language gap worth documenting." \
            --title "Nightly regression: ${BENCH} (${DATE})" \
            --from "nightly-eval" \
            --type "bug" 2>/dev/null || true
        log "  filed: ${BENCH}"
    done <<< "$FAILURES"
else
    log "no persistent failures — all smoke benchmarks passing"
fi

# Broadcast summary to controlplane inbox
ailang messages send controlplane \
    "Nightly eval complete: ${PASS}/${TOTAL} (${RATE}) on ${DATE}.
Model: opencode-qwen3-5-35b-a3b-mxfp8 | Tier: smoke | Trials: 2
$([ -n "$FAILURES" ] && echo "Persistent failures filed to design-doc-creator: $(echo $FAILURES | tr '\n' ', ')" || echo "All benchmarks passing.")" \
    --title "Nightly eval: ${PASS}/${TOTAL} (${DATE})" \
    --from "nightly-eval" 2>/dev/null || true

log "=== nightly eval done ==="
