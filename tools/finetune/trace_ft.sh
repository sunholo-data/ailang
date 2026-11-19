#!/bin/bash
# Capture a real transcript of the fine-tune in a pi agent eval (diagnostics, 2026-11-06).
#
# pi banks only tool-call counts; the fine-tune stops after ~2 calls and nothing records
# what it said. This takes the GPU by the COOPERATIVE HANDOFF (the running pilot eval-suite
# lends it at its next benchmark boundary, exactly as for Daneel's intake), runs the given
# benchmarks through tools/finetune/trace_proxy.py, and hands the GPU straight back.
#
#   tools/finetune/trace_ft.sh <lane> <prompt-version> <out.jsonl> <bench,bench>
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
LANE=${1:?lane}; PV=${2:?prompt version}; OUTJ=${3:?trace out}; BENCH=${4:?benchmarks}
cd "$REPO" || exit 1
source tools/launchd/rig-lock.sh
REQ=ft-trace
rig_lock_request_yield "$REQ" 1200 || echo "another handoff in force; waiting normally"
rig_lock_acquire wait "$REQ" || { echo "acquire failed"; exit 1; }
rig_lock_clear_yield "$REQ"
echo "lock held: $(cat "$RIG_LOCK_DIR/holder")"
python3 -I tools/finetune/trace_proxy.py --listen 11437 --upstream http://127.0.0.1:11434 --out "$OUTJ" &
PROXY=$!
sleep 1
AILANG_MODELS_PATH=$FT/traces/models.trace.yml AILANG_EVAL_TRAPS_CARD=off AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800 \
  "$FT/bin/ailang-ft2" eval-suite --agent --models "$LANE" --benchmarks "$BENCH" --langs ailang --trials 1 \
  --parallel 1 --prompt-version "$PV" --output "$FT/traces/run_$(date +%H%M%S)" > "$OUTJ.log" 2>&1
echo "eval rc=$?"
kill "$PROXY" 2>/dev/null
echo "trace: $(wc -l < "$OUTJ") requests"
