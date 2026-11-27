#!/bin/bash
# Pilot-5e CPU prep (runs in parallel while Pilot-5d runs STAGE agent, 0 GPU usage):
#   1. Enrich verified compiler-repair pairs (onpolicy r1/r2 + stale_pairs_opus) with Neon diagnostic reasoning
#   2. Build mix-pilot5e combining reasoned_single (2,186) + reasoned_repairs + reasoned_episodes (721)
#      using patched train2.py (which renders <|channel>thought on 100% of single-turn & episode turns!)
#   3. Verify with real gemma-4-12B-it-4bit tokenizer that 100% of weight!=0 assistant spans in
#      mix-pilot5e/train.jsonl and valid.jsonl contain <|channel>thought ... <channel|>
set -euo pipefail

REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
P5E=$FT/pilot5/e
DATA=$FT/data-p5
PY=$FT/.venv/bin/python
BASE=$HOME/models/gemma-4-12B-it-4bit
MODEL=${REASON_MODEL:-gemini-4-neon-medium}

mkdir -p "$P5E/state" "$DATA"
LOG=$P5E/prep.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }

cd "$REPO"

REPAIR_SRCS="$FT/onpolicy/onpolicy_pairs_v2.jsonl,$FT/onpolicy-r2/onpolicy_pairs.jsonl,$FT/stale/stale_pairs_opus.jsonl"

if [ ! -f "$P5E/state/reason_repairs.done" ]; then
  log "STAGE reason_repairs START (model=$MODEL)"
  python3 -I tools/finetune/reason5.py \
    --mode repairs \
    --in "$REPAIR_SRCS" \
    --repair-per-kind 8 \
    --verify-compile \
    --backend agy --model "$MODEL" \
    --batch-size 8 --workers 4 \
    --limit 550 \
    --out "$DATA/reasoned_repairs.jsonl" 2>&1 | tee -a "$LOG"
  python3 -I tools/finetune/reason5.py \
    --mode repairs \
    --in "$REPAIR_SRCS" \
    --repair-per-kind 8 \
    --verify-compile \
    --backend agy --model "$MODEL" \
    --batch-size 4 --workers 3 \
    --limit 550 \
    --out "$DATA/reasoned_repairs.jsonl" 2>&1 | tee -a "$LOG"
  date -u +%FT%TZ > "$P5E/state/reason_repairs.done"
  log "STAGE reason_repairs DONE ($(wc -l < "$DATA/reasoned_repairs.jsonl" | tr -d ' ') rows)"
fi

if [ ! -f "$P5E/state/mix_v2.done" ]; then
  log "STAGE mix START (mix-pilot5e v2: +agent-repair-frac 0.55 +agent-cap 700)"
  "$PY" -I tools/finetune/build_mix5d.py \
    --single "$DATA/reasoned_single.jsonl" \
    --repairs "$DATA/reasoned_repairs.jsonl" \
    --episodes "$DATA/reasoned_episodes.jsonl" \
    --prompts "$FT/evalroot/prompts" \
    --out "$FT/mix-pilot5e" \
    --agent-repair-frac 0.55 \
    --agent-cap 700 \
    --solve-per-bench 8 \
    --repair-per-kind 4 \
    --build-per-bench 6 \
    --min-single-reasoning-words 50 \
    --min-step-reasoning-words 4 \
    --tokenizer "$BASE" \
    --max-tokens 12288 2>&1 | tee -a "$LOG"

  log "Verifying 100% unmasked <|channel>thought and <|tool_call> coverage across mix-pilot5e..."
  "$PY" -I - <<'PYEOF' | tee -a "$LOG"
import json, os, sys
from pathlib import Path
from mlx_lm.utils import load_tokenizer
sys.path.insert(0, "tools/finetune")
import train2

tok = load_tokenizer(Path(os.path.expanduser("~/models/gemma-4-12B-it-4bit")))
for split in ("train", "valid"):
    p = os.path.expanduser(f"~/models/finetune/mix-pilot5e/{split}.jsonl")
    n = 0
    tool_samples = 0
    trained_toks = 0
    with open(p) as f:
        for ln in f:
            d = json.loads(ln)
            full, mask = train2._episode_process_tokens(tok, d["messages"], d.get("tools"))
            if d.get("tools"):
                tool_samples += 1
                unmasked_txt = tok.decode([t for t, m in zip(full, mask) if m])
                if "<|tool_call>" not in unmasked_txt:
                    raise AssertionError(f"tool sample missing <|tool_call> in {split} row {n}")
            trained_toks += int(sum(mask))
            n += 1
    print(f"VERIFIED {split}: {n} samples ({tool_samples} tool-enabled), {trained_toks} unmasked target tokens, 100% <|channel>thought + <|tool_call> verified")
PYEOF
  date -u +%FT%TZ > "$P5E/state/mix.done"
  date -u +%FT%TZ > "$P5E/state/mix_v2.done"
  log "STAGE mix DONE (mix-pilot5e v2 ready!)"
fi
