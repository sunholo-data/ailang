#!/bin/bash
# Pilot-5 runner (docs/docs/finetuning/pilot-5.md). Resumable: $P5/state/<stage>.done.
#
#   RUN=a RANK=32 EPOCHS=2 MIX=~/models/finetune/mix-pilot5a tools/finetune/pilot5_run.sh train serve test
#
#   train   train2.py, fresh adapter on the 4-bit base (GPU, rig lock)
#   serve   fuse -> strip audio config -> ollama create gemma4-12b-ailang-p5$RUN
#   indist  held-back repository tasks: does the answer compile? (indist_test.py; MODELS="name[:limit] ...")
#   test    single-shot language test, three prompts (pilot5_baseline.sh; ARMS="none slim" to pick)
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
RUN=${RUN:?set RUN}; RANK=${RANK:-32}; EPOCHS=${EPOCHS:-2}
SCALE=${SCALE:-$(python3 -c "print(320/$RANK)")}
MIX=${MIX:-$FT/mix-pilot5a}
P5=$FT/pilot5/$RUN
BASE=$HOME/models/gemma-4-12B-it-4bit
PY=$FT/.venv/bin/python
NAME=gemma4-12b-ailang-p5$RUN
LANE=ollama-gemma4-12b-p5$RUN
mkdir -p "$P5/state"
LOG=$P5/run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
die(){ log "FATAL: $*"; exit 1; }
mark(){ date -u +%FT%TZ > "$P5/state/$1.done"; log "STAGE $1 DONE"; }
need_disk(){ local free; free=$(df -g "$HOME" | awk 'NR==2{print $4}'); log "disk: ${free} GiB free (stage $1 needs $2)"; [ "$free" -ge "$2" ] || die "only ${free} GiB free"; }
cd "$REPO" || exit 1

stage_train(){
  need_disk train 10
  local n iters upd rc
  n=$(wc -l < "$MIX/train.jsonl" | tr -d ' '); iters=$(( n * EPOCHS )); upd=$(( iters / 4 ))
  sed -e "s/__ITERS__/$iters/" -e "s/__UPDATES__/$upd/" -e "s/__RANK__/$RANK/" -e "s/__SCALE__/$SCALE/" tools/finetune/pilot5.yaml > "$P5/resolved.yaml"
  log "train: $n samples x $EPOCHS epochs = $iters iterations, rank $RANK, scale $SCALE, mix $MIX"
  "$PY" -I tools/finetune/train2.py -c "$P5/resolved.yaml" --model "$BASE" --data "$MIX" --adapter-path "$P5/adapters" > "$P5/train.log" 2>&1 &
  local tp=$!
  # Divergence guard: run b sat at a loss of 7 for six hours before anyone looked (2026-11-18).
  ( while kill -0 $tp 2>/dev/null; do sleep 300
      l=$(sed 's/\x1b\[[0-9;]*m//g' "$P5/train.log" | tr '\r' '\n' | awk '$1+0>=100 && $2+0>0 {v=$2} END{print v+0}')
      if awk -v l="$l" 'BEGIN{exit !(l>3)}'; then log "DIVERGED: training loss $l, stopping"; kill $tp; fi
    done ) &
  wait $tp
  rc=$?
  sed 's/\x1b\[[0-9;]*m//g' "$P5/train.log" | tr '\r' '\n' | grep -E " Val | val |Saved" | tail -8 | tee -a "$LOG"
  [ $rc = 0 ] && [ -f "$P5/adapters/adapters.safetensors" ] || die "training failed rc=$rc"
}

stage_serve(){
  need_disk serve 40
  "$PY" -I -m mlx_lm.fuse --model "$BASE" --adapter-path "$P5/adapters" --save-path "$P5/fused" --dequantize > "$P5/fuse.log" 2>&1 || die "fuse failed"
  "$PY" -I - "$P5/fused/config.json" <<'EOP'
import json,sys
p=sys.argv[1]; c=json.load(open(p))
for k in ["audio_config","audio_token_id","boa_token_id","eoa_token_index","eoa_token_id"]: c.pop(k,None)
json.dump(c,open(p,"w"),indent=2)
EOP
  printf 'FROM %s\nPARAMETER temperature 1\nPARAMETER top_k 64\nPARAMETER top_p 0.95\n' "$P5/fused" > "$P5/Modelfile"
  ollama create "$NAME" -q mxfp8 -f "$P5/Modelfile" > "$P5/create.log" 2>&1 || die "ollama create failed"
  rm -rf "$P5/fused"   # ollama holds its own copy; the fused directory is 22 GB
  log "served: $NAME"
}

stage_indist(){
  local spec m lim
  for spec in ${MODELS:-$NAME}; do
    m=${spec%%@*}; lim=0; case $spec in *@*) lim=${spec##*@};; esac
    for pr in none slim; do
      "$PY" -I tools/finetune/indist_test.py --desc "$FT/data-p5/desc.jsonl" --model "$m" --prompt "$pr" \
        --prompts "$FT/evalroot/prompts" --out "$FT/pilot5/indist.$(echo "$m" | tr ':/' '__').$pr.jsonl" --limit "$lim" 2>&1 | tail -1 | tee -a "$LOG"
    done
  done
}

stage_test(){
  # The lane lives only in this run's models file: a copy of the untrained lane with the model swapped.
  "$PY" -I - "$REPO/internal/modelreg/models.yml" "$P5/models.yml" "$LANE" "$NAME" <<'EOP'
import re,sys
src,out,lane,name=sys.argv[1:5]; s=open(src).read()
i=s.index("\n  ollama-gemma4-12b:\n")+1; j=re.compile(r"^  [A-Za-z0-9#]",re.M).search(s,i+5).start()
blk=s[i:j].replace("  ollama-gemma4-12b:",f"  {lane}:",1).replace('api_name: "gemma4:12b-mxfp8"',f'api_name: "{name}"')
k=s.index("\nmodels:\n")+len("\nmodels:\n")
open(out,"w").write(s[:k]+blk+s[k:])
EOP
  AILANG_MODELS_PATH="$P5/models.yml" tools/finetune/pilot5_baseline.sh "$LANE" "${TRIALS:-3}" "" "${ARMS:-none slim full}" 2>&1 | tee -a "$LOG"
}

# One lock for the whole run; the rotation filler is asked to step aside at its next checkpoint.
source tools/launchd/rig-lock.sh
rig_lock_request_yield finetune-pilot5 1800 || log "another handoff in force; waiting normally"
rig_lock_acquire wait finetune-pilot5 || die "rig lock acquire failed"
rig_lock_clear_yield finetune-pilot5
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
parent=$$
( while kill -0 "$parent" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

for s in "$@"; do
  if [ -f "$P5/state/$s.done" ]; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "pilot-5 run $RUN finished: $*"
