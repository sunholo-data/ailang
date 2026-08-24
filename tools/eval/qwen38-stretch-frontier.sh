#!/bin/bash
# qwen3.8 on motoko, stretch + frontier (33 benchmarks) — 2026-08-18.
# WHY: the core-tier A/B could not discriminate. motoko-qwen3.6 sits at 100% on
# core (published leaderboard AND our 46/46 rerun), so the control was at ceiling
# and the tier ranked nothing. Headroom lives here: qwen3.6 scores 48.8% stretch
# / 21.4% frontier.
# Single arm by request (Mark) — see the caveat recorded alongside this run about
# the banked qwen3.6 control being pre-fix and crash-contaminated.
set -uo pipefail
cd "$(dirname "$0")/../.."
BENCH=$(cat /tmp/ab_sf_benchmarks.txt)
LOG=/tmp/ollama-serve-launchd.log
swaps() { grep -cE "stopping mlx runner|using llama-server for model" "$LOG"; }
S=$(swaps); echo "[$(date +%H:%M:%S)] start; swap counter $S"
ailang eval-suite --agent --models motoko-local-qwen3-8-27b \
  --benchmarks "$BENCH" --langs ailang --trials 2 --parallel 1 \
  --output /tmp/sf_qwen38 > /tmp/sf_qwen38.log 2>&1
E=$(swaps)
echo "[$(date +%H:%M:%S)] done; model swaps during run: $((E-S))"
ailang messages send --to controlplane --title "qwen3.8 stretch+frontier complete" \
  --body "33 benchmarks x2 trials on motoko. Swaps: $((E-S)). Results /tmp/sf_qwen38" 2>/dev/null
