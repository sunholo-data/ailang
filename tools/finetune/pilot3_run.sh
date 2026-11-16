#!/bin/bash
# Fine-tune pilot-3 runner (docs/docs/finetuning/pilot-3.md). Resumable: $STATE/<stage>.done.
#
#   (collection is separate: tools/finetune/collect_onpolicy.sh, GPU)
#   pairs    onpolicy_pairs.py: trace -> failing file states (CPU)
#   repair   stale_fix.py --prompt onpolicy, Claude on the subscription (no GPU; quota)
#   data     synth_episodes.py --onpolicy -> thoughts.py -> build_mix2.py (CPU)
#   train    train2.py, warm start from the pilot-2 adapter (GPU, rig lock)
#   serve    fuse -> strip audio config -> ollama create gemma4-12b-ailang-ft3
#   eval_*   held-out 24, lean pi profile, agent mode, NO wall-clock cap (pilot-2's caps cut arms short).
#            Before evals: install the #2151 fix (make quick-install) so every arm sees clean tool output.
#   summary
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
OP=$FT/onpolicy
P3=$FT/pilot3
STATE=$P3/state
BASE=$HOME/models/gemma-4-12B-it-4bit
MIX=$FT/mix-pilot3
ADAPTER=$P3/adapters
RESUME=$FT/pilot2/adapters/adapters.safetensors
FUSED=$P3/fused
OLLAMA_NAME=gemma4-12b-ailang-ft3
BIN=$FT/bin/ailang-ft3
PY=$FT/.venv/bin/python
OUT=$REPO/eval_results/finetune-pilot3
mkdir -p "$STATE" "$OUT"
LOG=$P3/run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
done_(){ [ -f "$STATE/$1.done" ]; }
mark(){ date -u +%FT%TZ > "$STATE/$1.done"; log "STAGE $1 DONE"; }
die(){ log "FATAL: $*"; exit 1; }
cd "$REPO" || exit 1
export AILANG_MODELS_PATH="$REPO/internal/modelreg/models.yml"
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800
HELD=$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")

gpu_lock(){
  source tools/launchd/rig-lock.sh
  rig_lock_acquire wait finetune-pilot3 || die "rig lock acquire failed"
  rig_lock_clear_yield finetune-pilot3
  log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
  local parent=$$
  ( while kill -0 "$parent" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &
}

stage_pairs(){
  "$PY" -I tools/finetune/onpolicy_pairs.py --trace "$OP/trace.jsonl" --out "$OP/items.jsonl" \
    --per-kind "${PER_KIND:-10}" | tee "$OP/pairs_report.json" | tail -5 | tee -a "$LOG"
}

stage_repair(){
  env -u ANTHROPIC_API_KEY "$PY" -I tools/finetune/stale_fix.py fix --backend claude-cli --model claude-opus-5-5 \
    --prompt onpolicy --items "$OP/items.jsonl" --out "$OP" --pairs-out onpolicy_pairs.jsonl \
    --limit 100000 --workers "${WORKERS:-4}" --rounds 4 | tee -a "$LOG"
}

stage_data(){
  local EP=$FT/episodes-v3
  "$PY" -I tools/finetune/synth_episodes.py --out "$EP" --onpolicy "$OP/onpolicy_pairs.jsonl" | tail -30 | tee -a "$LOG"
  "$PY" -I tools/finetune/thoughts.py --episodes "$EP/episodes.jsonl" --out "$EP/episodes_thought.jsonl" | tee -a "$LOG"
  # Replay: a smaller sample of pilot-2's kinds next to every on-policy episode (cap per kind).
  "$PY" -I tools/finetune/build_mix2.py --episodes "$EP/episodes_thought.jsonl" --out "$MIX" \
    --solve-per-bench 4 --repair-per-kind 2 --example-repeat 1 --onpolicy-per-kind 8 --max-steps 24 \
    --tokenizer "$BASE" --max-tokens 12288 | tee -a "$LOG"
}

stage_train(){
  local n upd
  n=$(wc -l < "$MIX/train.jsonl" | tr -d ' ')
  upd=$(( n / 4 ))
  sed -e "s/__ITERS__/$n/" -e "s/__UPDATES__/$upd/" tools/finetune/pilot3.yaml > "$P3/pilot3.resolved.yaml"
  echo "resume_adapter_file: $RESUME" >> "$P3/pilot3.resolved.yaml"
  log "train: $n samples, 1 epoch, $upd optimizer updates, warm start $RESUME"
  "$PY" -I tools/finetune/train2.py -c "$P3/pilot3.resolved.yaml" --model "$BASE" --data "$MIX" \
    --adapter-path "$ADAPTER" > "$P3/train.log" 2>&1
  rc=$?; sed 's/\x1b\[[0-9;]*m//g' "$P3/train.log" | tr '\r' '\n' | grep -E "Val loss|val |save" | tail -8 | tee -a "$LOG"
  [ $rc = 0 ] && [ -f "$ADAPTER/adapters.safetensors" ] || die "training failed rc=$rc"
}

stage_serve(){
  "$PY" -I -m mlx_lm.fuse --model "$BASE" --adapter-path "$ADAPTER" --save-path "$FUSED" --dequantize > "$P3/fuse.log" 2>&1 || die "fuse failed"
  "$PY" -I - "$FUSED/config.json" <<'EOF'
import json,sys
p=sys.argv[1]; c=json.load(open(p))
for k in ["audio_config","audio_token_id","boa_token_id","eoa_token_index","eoa_token_id"]: c.pop(k,None)
json.dump(c,open(p,"w"),indent=2)
EOF
  printf 'FROM %s\nPARAMETER temperature 1\nPARAMETER top_k 64\nPARAMETER top_p 0.95\n' "$FUSED" > "$P3/Modelfile"
  ollama create "$OLLAMA_NAME" -q mxfp8 -f "$P3/Modelfile" > "$P3/create.log" 2>&1 || die "ollama create failed"
  ollama show "$OLLAMA_NAME" --parameters | tee -a "$LOG"
}

run_arm(){ # name lane prompt_version traps(on/off)
  local name=$1 lane=$2 pv=$3 traps=$4
  ( [ "$traps" = off ] && export AILANG_EVAL_TRAPS_CARD=off
    "$BIN" eval-suite --agent --models "$lane" --benchmarks "$HELD" --langs ailang --trials 1 --parallel 1 \
      --skip-existing --tool-policy lean --prompt-version "$pv" --output "$OUT/$name" ) > "$OUT/$name.log" 2>&1
  python3 -I - "$OUT/$name" "$pv" "$name" <<'EOF2' | tee -a "$LOG"
import json,glob,sys,collections
d,want,name=sys.argv[1:4]; c=collections.Counter(); n=ok=0
for p in glob.glob(d+"/agent/*.json"):
    r=json.load(open(p)); n+=1; ok+=bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"]); c[r.get("prompt_version") or "?"]+=1
print(f"{name}: {ok}/{n} pass; prompt versions {dict(c)}")
c.pop("?",0)
if any(k!=want for k in c): print("PROMPT-VERSION MISMATCH: arm is void"); sys.exit(3)
EOF2
  [ "${PIPESTATUS[0]}" = 0 ] || die "$name: prompt-version guard failed"
}

stage_eval_B(){ run_arm B pi-gemma4-12b-ft3 ft-slim-v1 off; }
stage_eval_D(){ run_arm D pi-gemma4-12b-ft3 v0.16.12 on; }     # fine-tune + the full teaching prompt
# Base, run FRESH: pilot-2's 13 base rows saw the Cloud Trace notice atop every tool result
# (#2151, fixed d76958190); pilot-3's arms do not. Mixing them would confound the comparison.
stage_eval_A(){ run_arm A pi-gemma4-12b v0.16.12 on; }
stage_eval_C(){ run_arm C pi-gemma4-12b-ft3 ft-none off; }
stage_summary(){ grep -E "^(A|B|C|D)[_:]" "$LOG" | tail -8; }

STAGES=${*:-pairs}
case " $STAGES " in *" train "*|*" serve "*|*" eval_"*) gpu_lock ;; esac
for s in $STAGES; do
  if done_ "$s"; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "pilot-3 runner finished: $STAGES"
