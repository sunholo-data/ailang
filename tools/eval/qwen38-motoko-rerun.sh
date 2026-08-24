#!/bin/bash
# Motoko qwen3.8 A/B RERUN (2026-08-18), after two fixes:
#   1. OLLAMA_MAX_LOADED_MODELS 1 -> 2 (2eec9686c) — the embedder no longer
#      evicts the eval LLM mid-stream. Prior run lost 10/92 runs (11%) to that.
#   2. prelude println capability hole + prompt_injection caps (e820dcf0b).
#
# DESIGN CHANGE: arms run in SEPARATE PASSES, not interleaved per benchmark.
# The interleaved run forced 87 model swaps across 92 runs; one pass per arm
# needs ~1 each, and leaves the embedder's slot untouched throughout.
set -uo pipefail
cd "$(dirname "$0")/../.."

CORE=$(cat /tmp/ab_core_benchmarks.txt)
LOG=/tmp/ollama-serve-launchd.log
swaps() { grep -cE "stopping mlx runner|using llama-server for model" "$LOG"; }

BEFORE=$(swaps)
echo "[$(date +%H:%M:%S)] swap-event counter at start: $BEFORE"

for arm in motoko-local-qwen3-8-27b motoko-local-qwen3-6-35b-a3b-mxfp8; do
  S=$(swaps)
  echo "[$(date +%H:%M:%S)] PASS: $arm"
  ailang eval-suite --agent --models "$arm" \
    --benchmarks "$CORE" --langs ailang --trials 2 --parallel 1 \
    --output "/tmp/ab2_${arm}" > "/tmp/ab2_${arm}.log" 2>&1
  E=$(swaps)
  echo "[$(date +%H:%M:%S)] PASS DONE: $arm — model swaps during pass: $((E-S))"
done

AFTER=$(swaps)
echo "[$(date +%H:%M:%S)] TOTAL model swaps this rerun: $((AFTER-BEFORE))  (interleaved run: 87)"
ailang messages send --to controlplane \
  --title "qwen3.8 motoko RERUN complete" \
  --body "Two-pass rerun after the embedder-eviction fix. Swaps: $((AFTER-BEFORE)) vs 87 interleaved. Results: /tmp/ab2_motoko-local-qwen3-8-27b and /tmp/ab2_motoko-local-qwen3-6-35b-a3b-mxfp8" 2>/dev/null
