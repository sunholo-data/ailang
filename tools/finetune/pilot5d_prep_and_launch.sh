#!/bin/bash
# Pilot-5d end-to-end prep & launch:
#   1. Write frozen compact reference card (from prompts/v0.16.12.md to match the pinned 6d28e1ad5 scorer)
#   2. Generate batched Gemini 4 Neon reasoning traces for single-turn items (repo + eval + runnable examples)
#      with a second small-batch backfill pass for any transient misses
#   3. Generate batched Gemini 4 Neon reasoning traces for 550 lean-pi agent episodes
#   4. Build joint leakage-free mix-pilot5d (verified with real Gemma 4 tokenizer + non-empty reasoning gate)
#   5. Run pilot5_run.sh (d, rank 32, 1.5 epochs): train -> serve -> indist -> test -> agent
set -euo pipefail

REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
P5D=$FT/pilot5/d
DATA=$FT/data-p5
PY=$FT/.venv/bin/python
BASE=$HOME/models/gemma-4-12B-it-4bit
MODEL=${REASON_MODEL:-gemini-4-neon-medium}

mkdir -p "$P5D/state" "$DATA"
LOG=$P5D/prep_and_run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }

cd "$REPO"

# 1. Compact prompt card aligned with frozen evalroot (v0.16.12)
if [ ! -s "$FT/evalroot/prompts/ft-compact-v1.md" ]; then
  python3 -I - <<'PYEOF'
import os, sys
sys.path.insert(0, "tools/finetune")
from build_mix5d import build_compact_prompt
out = os.path.expanduser("~/models/finetune/evalroot/prompts/ft-compact-v1.md")
txt = build_compact_prompt("prompts/v0.16.12.md", max_chars=6400)
open(out, "w", encoding="utf-8").write(txt + "\n")
print("wrote ft-compact-v1.md:", len(txt), "chars")
PYEOF
  log "generated ft-compact-v1.md"
fi

# 2. Single-turn reasoning traces (repo + training-bench solutions + verified runnable examples)
if [ ! -f "$P5D/state/reason_single.done" ]; then
  log "STAGE reason_single START (model=$MODEL)"
  python3 -I tools/finetune/reason5.py \
    --mode single \
    --in "$DATA/desc.jsonl" \
    --eval "$FT/data-v1/eval.jsonl" --eval-cap 8 \
    --examples-dir "$REPO/examples/runnable" \
    --verify-compile \
    --backend agy --model "$MODEL" \
    --batch-size 8 --workers 4 \
    --out "$DATA/reasoned_single.jsonl" 2>&1 | tee -a "$LOG"
  # Second pass at batch-size 4 to backfill any items rejected or dropped in pass 1
  python3 -I tools/finetune/reason5.py \
    --mode single \
    --in "$DATA/desc.jsonl" \
    --eval "$FT/data-v1/eval.jsonl" --eval-cap 8 \
    --examples-dir "$REPO/examples/runnable" \
    --verify-compile \
    --backend agy --model "$MODEL" \
    --batch-size 4 --workers 3 \
    --out "$DATA/reasoned_single.jsonl" 2>&1 | tee -a "$LOG"
  date -u +%FT%TZ > "$P5D/state/reason_single.done"
  log "STAGE reason_single DONE ($(wc -l < "$DATA/reasoned_single.jsonl" | tr -d ' ') rows)"
fi

# 3. Multi-turn lean-pi episode planning reasoning traces (550 episodes -> 450 capped in mix)
if [ ! -f "$P5D/state/reason_episodes.done" ]; then
  log "STAGE reason_episodes START (model=$MODEL)"
  python3 -I tools/finetune/reason5.py \
    --mode episodes \
    --in "$FT/episodes-v4/episodes_thought.jsonl" \
    --backend agy --model "$MODEL" \
    --batch-size 8 --workers 4 \
    --limit 550 \
    --out "$DATA/reasoned_episodes.jsonl" 2>&1 | tee -a "$LOG"
  python3 -I tools/finetune/reason5.py \
    --mode episodes \
    --in "$FT/episodes-v4/episodes_thought.jsonl" \
    --backend agy --model "$MODEL" \
    --batch-size 4 --workers 3 \
    --limit 550 \
    --out "$DATA/reasoned_episodes.jsonl" 2>&1 | tee -a "$LOG"
  date -u +%FT%TZ > "$P5D/state/reason_episodes.done"
  log "STAGE reason_episodes DONE ($(wc -l < "$DATA/reasoned_episodes.jsonl" | tr -d ' ') rows)"
fi

# 4. Build leakage-free joint mix-pilot5d with exact tokenizer check + non-empty reasoning gate
if [ ! -f "$P5D/state/mix.done" ]; then
  log "STAGE mix START"
  "$PY" -I tools/finetune/build_mix5d.py \
    --single "$DATA/reasoned_single.jsonl" \
    --episodes "$DATA/reasoned_episodes.jsonl" \
    --prompts "$FT/evalroot/prompts" \
    --out "$FT/mix-pilot5d" \
    --agent-cap 450 \
    --min-single-reasoning-words 50 \
    --min-step-reasoning-words 4 \
    --tokenizer "$BASE" \
    --max-tokens 12288 2>&1 | tee -a "$LOG"
  date -u +%FT%TZ > "$P5D/state/mix.done"
  log "STAGE mix DONE"
fi

# 5. Train (1.5 epochs, rank 32, auto-prune intermediate checkpoints) -> serve -> indist -> single-shot -> agent
log "HANDOFF to pilot5_run.sh (RUN=d, RANK=32, EPOCHS=1.5)"
exec env RUN=d RANK=32 EPOCHS=1.5 MIX="$FT/mix-pilot5d" \
  bash "$REPO/tools/finetune/pilot5_run.sh" train serve indist test agent
