#!/bin/bash
# Phase 2 of the 2026-08-17 qwen3.8 on-device A/B: the motoko arms.
#
# Runs AFTER the opencode+pi A/B finishes. Gated by a smoke test because
# motoko routes tool-calling turns via ollama's /v1, and ollama v0.32.6 changed
# the /v1 streaming wire format (role only on first chunk, finish_reason on its
# own chunk, usage separate). The rig crossed that boundary in the 0.32.1 ->
# 0.32.14 upgrade, so motoko-on-ollama is UNVERIFIED post-upgrade. Without this
# gate a broken harness would burn ~21h banking failures that look like model
# losses but are really a wire-format break.
#
# Usage: tools/eval/qwen38-motoko-phase.sh
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 1

CORE=$(cat /tmp/ab_core_benchmarks.txt)
SMOKE_OUT=/tmp/motoko38_smoke
FULL_OUT=/tmp/ab_qwen38_motoko

echo "[$(date +%H:%M:%S)] waiting for the opencode+pi A/B to finish..."
while pgrep -f "eval-suite --agent --models opencode-qwen3-8-27b" >/dev/null; do sleep 60; done
echo "[$(date +%H:%M:%S)] phase 1 done."

# --- GATE: smoke both motoko arms before committing ~21h -------------------
echo "[$(date +%H:%M:%S)] motoko smoke (validates /v1 after the ollama upgrade)"
ailang eval-suite --agent \
  --models motoko-local-qwen3-8-27b,motoko-local-qwen3-6-35b-a3b-mxfp8 \
  --benchmarks fizzbuzz,cli_args --langs ailang --trials 1 --parallel 1 \
  --output "$SMOKE_OUT" > /tmp/motoko38_smoke.log 2>&1

PASSES=$(find "$SMOKE_OUT" -name '*.json' -exec python3 -c "
import json,sys
try:
    d=json.load(open(sys.argv[1]))
    print(1 if d.get('stdout_ok') else 0)
except Exception: print(0)
" {} \; 2>/dev/null | paste -sd+ - | bc 2>/dev/null || echo 0)
TOTAL=$(find "$SMOKE_OUT" -name '*.json' 2>/dev/null | wc -l | tr -d ' ')
echo "[$(date +%H:%M:%S)] motoko smoke: ${PASSES}/${TOTAL} passed"

# Require BOTH arms to produce output. Zero passes on 4 runs is the signature of
# a wire-format break (empty output / no tool calls), not of model weakness.
if [ "${PASSES:-0}" -lt 1 ] || [ "${TOTAL:-0}" -lt 4 ]; then
  echo "[$(date +%H:%M:%S)] GATE FAILED — motoko produced ${PASSES}/${TOTAL}."
  echo "  Do NOT read this as 'qwen3.8 is bad'. Check FIRST:"
  echo "   1. ollama /v1 wire format vs motoko's client (v0.32.6 change)"
  echo "   2. motoko-stderr log for empty-output / 0-tool-call turns"
  echo "   3. ailang chains chat <id> for what the agent was actually told"
  ailang messages send --to controlplane \
    --title "qwen3.8 motoko phase BLOCKED: smoke ${PASSES}/${TOTAL}" \
    --body "Motoko smoke failed after the ollama 0.32.1->0.32.14 upgrade. Suspect the v0.32.6 /v1 streaming wire-format change, NOT the model. Full motoko A/B was NOT started. Logs: /tmp/motoko38_smoke.log" 2>/dev/null
  exit 1
fi

# --- Full motoko A/B --------------------------------------------------------
echo "[$(date +%H:%M:%S)] gate passed — launching full motoko A/B (~21h)"
ailang eval-suite --agent \
  --models motoko-local-qwen3-8-27b,motoko-local-qwen3-6-35b-a3b-mxfp8 \
  --benchmarks "$CORE" --langs ailang --trials 2 --parallel 1 \
  --output "$FULL_OUT" > /tmp/ab_qwen38_motoko.log 2>&1
echo "[$(date +%H:%M:%S)] motoko A/B finished (exit $?)"
ailang messages send --to controlplane \
  --title "qwen3.8 motoko A/B complete" \
  --body "Results: $FULL_OUT — compare motoko-local-qwen3-8-27b vs motoko-local-qwen3-6-35b-a3b-mxfp8 (both hard_timeout_secs 3600, same ollama profile)." 2>/dev/null
