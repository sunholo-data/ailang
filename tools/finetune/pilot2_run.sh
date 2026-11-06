#!/bin/bash
# Fine-tune pilot-2 runner (m-eval-finetuning-data-pipeline §Pilot-2; D-FT-8/9).
#
# Resumable: $STATE/<stage>.done markers. Waits on the rig lock (pilot-1's re-run holds it).
#   dryrun    20 train2 steps on the real mix: tokens/s and peak memory, logged, then STOP
#             (the full run starts only when re-invoked with `train ...`, after the numbers
#             are reviewed and pilot2.yaml is adjusted).
#   train     train2.py with pilot2.yaml
#   serve     fuse -> strip audio config -> ollama create (mxfp8, base sampling defaults)
#   eval_*    agent arms on the held-out 24, ALL on the lean pi profile (--tool-policy lean)
#   summary
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
P2=$FT/pilot2
STATE=$P2/state
BASE=$HOME/models/gemma-4-12B-it-4bit
MIX=$FT/mix-pilot2
ADAPTER=$P2/adapters
FUSED=$P2/fused
OLLAMA_NAME=gemma4-12b-ailang-ft2
BIN=$FT/bin/ailang-ft3          # built with the lean profile (8faeacb73+)
PY=$FT/.venv/bin/python
OUT=$REPO/eval_results/finetune-pilot2
mkdir -p "$STATE" "$OUT"
LOG=$P2/run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
done_(){ [ -f "$STATE/$1.done" ]; }
mark(){ date -u +%FT%TZ > "$STATE/$1.done"; log "STAGE $1 DONE"; }
die(){ log "FATAL: $*"; exit 1; }

cd "$REPO" || exit 1
export AILANG_MODELS_PATH="$REPO/internal/modelreg/models.yml"
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800
HELD=$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")

source tools/launchd/rig-lock.sh
REQ=finetune-pilot2
rig_lock_acquire wait "$REQ" || die "rig lock acquire failed"
rig_lock_clear_yield "$REQ"
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
PARENT=$$
( while kill -0 "$PARENT" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

stage_dryrun(){
  sed -e 's/^iters: .*/iters: 20/' -e 's/^steps_per_eval: .*/steps_per_eval: 1000/' -e 's/^save_every: .*/save_every: 1000/' \
      tools/finetune/pilot2.yaml > "$P2/dryrun.yaml"
  /usr/bin/time -l "$PY" -I tools/finetune/train2.py -c "$P2/dryrun.yaml" --model "$BASE" --data "$MIX" \
      --adapter-path "$P2/adapters-dryrun" > "$P2/dryrun.log" 2>&1 || die "dry run failed (see $P2/dryrun.log)"
  sed 's/\x1b\[[0-9;]*m//g' "$P2/dryrun.log" | tr '\r' '\n' | grep -E "tok/s|Peak|peak|maximum resident" | tail -6 | tee -a "$LOG"
}

stage_train(){
  "$PY" -I tools/finetune/train2.py -c tools/finetune/pilot2.yaml --model "$BASE" --data "$MIX" \
      --adapter-path "$ADAPTER" > "$P2/train.log" 2>&1
  rc=$?; sed 's/\x1b\[[0-9;]*m//g' "$P2/train.log" | tr '\r' '\n' | grep -E "val |save" | tail -6 | tee -a "$LOG"
  [ $rc = 0 ] && [ -f "$ADAPTER/adapters.safetensors" ] || die "training failed rc=$rc"
}

stage_serve(){
  "$PY" -I -m mlx_lm.fuse --model "$BASE" --adapter-path "$ADAPTER" --save-path "$FUSED" --dequantize > "$P2/fuse.log" 2>&1 || die "fuse failed"
  "$PY" -I - "$FUSED/config.json" <<'EOF'
import json,sys
p=sys.argv[1]; c=json.load(open(p))
for k in ["audio_config","audio_token_id","boa_token_id","eoa_token_index","eoa_token_id"]: c.pop(k,None)
json.dump(c,open(p,"w"),indent=2)
EOF
  # the base model's sampling defaults (pilot-1 shipped without them)
  printf 'FROM %s\nPARAMETER temperature 1\nPARAMETER top_k 64\nPARAMETER top_p 0.95\n' "$FUSED" > "$P2/Modelfile"
  ollama create "$OLLAMA_NAME" -q mxfp8 -f "$P2/Modelfile" > "$P2/create.log" 2>&1 || die "ollama create failed"
  ollama show "$OLLAMA_NAME" --parameters | tee -a "$LOG"
}

run_arm(){ # name lane prompt_version traps(on/off)
  local name=$1 lane=$2 pv=$3 traps=$4
  ( [ "$traps" = off ] && export AILANG_EVAL_TRAPS_CARD=off
    "$BIN" eval-suite --agent --models "$lane" --benchmarks "$HELD" --langs ailang --trials 1 --parallel 1 \
      --skip-existing --max-wall-clock 5h --tool-policy lean --prompt-version "$pv" --output "$OUT/$name" ) > "$OUT/$name.log" 2>&1
  python3 -I - "$OUT/$name" "$pv" <<'EOF' | tee -a "$LOG"
import json,glob,sys,collections
d,want=sys.argv[1],sys.argv[2]; c=collections.Counter(); n=ok=0
for p in glob.glob(d+"/**/*.json",recursive=True):
    try: r=json.load(open(p))
    except Exception: continue
    if not isinstance(r,dict) or "compile_ok" not in r: continue
    n+=1; ok+=bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"]); c[r.get("prompt_version") or "?"]+=1
print(f"{d.split('/')[-1]}: {ok}/{n} pass; prompt versions {dict(c)}")
# Rows a run never reached the prompt for (non_agentic / pre-run failures) bank no
# prompt_version; they are not evidence of a fallback. Only a DIFFERENT version voids the arm.
unl=c.pop("?",0)
if unl: print(f"  ({unl} rows banked no prompt_version: early failures, not counted against the guard)")
if any(k!=want for k in c): print("PROMPT-VERSION MISMATCH: arm is void"); sys.exit(3)
EOF
  [ "${PIPESTATUS[0]}" = 0 ] || die "$name: prompt-version guard failed"
}

stage_eval_B(){ run_arm B_lean pi-gemma4-12b-ft2 ft-slim-v1 off; }
stage_eval_C(){ run_arm C_lean pi-gemma4-12b-ft2 ft-none off; }
stage_eval_A(){ run_arm A_lean pi-gemma4-12b v0.16.12 on; }
stage_eval_A2(){ run_arm A2_lean pi-gemma4-12b ft-slim-v1 off; }
stage_summary(){ grep -E "_lean: " "$LOG" | tail -8; }

STAGES=${*:-dryrun}
for s in $STAGES; do
  if done_ "$s"; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "pilot-2 runner finished: $STAGES"
