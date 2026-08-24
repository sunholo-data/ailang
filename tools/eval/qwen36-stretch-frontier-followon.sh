#!/bin/bash
# qwen3.6 CONTROL arm on stretch+frontier, chained after the qwen3.8 pass.
#
# WHY a fresh control instead of the banked rows: the existing motoko-qwen3.6
# stretch/frontier bank is unusable as a comparator — 23% of stretch and 59% of
# frontier rows are api_error/step_exhausted (the embedder-eviction bug fixed in
# 2eec9686c), nearly all of it predates the ollama 0.32.1->0.32.14 boundary, and
# it was gated on wall-clock rather than the token WORK gate. That contamination
# UNDERSTATES qwen3.6 — on core it went 41/46 -> 46/46 once crashes stopped — so
# comparing fresh qwen3.8 against it would flatter qwen3.8.
#
# Runs as a SEPARATE PASS (never interleaved): two on-device LLMs evict each
# other, which cost 87 swaps and 11% of runs on 2026-08-17.
set -uo pipefail
cd "$(dirname "$0")/../.."
BENCH=$(cat /tmp/ab_sf_benchmarks.txt)
LOG=/tmp/ollama-serve-launchd.log
swaps() { grep -cE "stopping mlx runner|using llama-server for model" "$LOG"; }

echo "[$(date +%H:%M:%S)] waiting for the qwen3.8 stretch+frontier pass..."
while pgrep -f "eval-suite --agent --models motoko-local-qwen3-8-27b" >/dev/null; do sleep 120; done
BANKED=$(find /tmp/sf_qwen38 -name '*.json' 2>/dev/null | wc -l | tr -d ' ')
echo "[$(date +%H:%M:%S)] qwen3.8 pass done; $BANKED rows banked"
if [ "${BANKED:-0}" -lt 5 ]; then
  echo "[$(date +%H:%M:%S)] ABORT: qwen3.8 pass banked almost nothing — investigate before spending rig time on the control."
  ailang messages send --to controlplane --title "qwen3.6 control NOT started" \
    --body "qwen3.8 stretch+frontier banked only $BANKED rows; control arm withheld pending investigation." 2>/dev/null
  exit 1
fi

S=$(swaps); echo "[$(date +%H:%M:%S)] starting qwen3.6 control; swap counter $S"
ailang eval-suite --agent --models motoko-local-qwen3-6-35b-a3b-mxfp8 \
  --benchmarks "$BENCH" --langs ailang --trials 2 --parallel 1 \
  --output /tmp/sf_qwen36 > /tmp/sf_qwen36.log 2>&1
E=$(swaps)
echo "[$(date +%H:%M:%S)] control done; model swaps during pass: $((E-S))"
ailang messages send --to controlplane --title "qwen3.8 vs 3.6 stretch+frontier COMPLETE" \
  --body "Both passes done on the discriminating tiers. Swaps this pass: $((E-S)). Results: /tmp/sf_qwen38 and /tmp/sf_qwen36" 2>/dev/null
