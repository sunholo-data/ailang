#!/bin/bash
# Fine-tune pilot-1 weekend runner (m-eval-finetuning-data-pipeline §Weekend pilot plan).
#
# Resumable: each stage writes $STATE/<stage>.done; re-running skips finished stages.
# Holds the rig lock for the whole run (the lock helper releases it on exit).
# PRECONDITION (done separately, visibly): the rotation filler and nightly are
# held with the launchd hold protocol (hold file + bootout) for the weekend.
#
# Usage: tools/finetune/weekend_pilot.sh [stage ...]   (default: all stages)
#   stages: preflight train serve eval_B_std eval_C_std eval_B_agent eval_C_agent eval_A_std summary
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
P1=$FT/pilot1
STATE=$P1/state
BASE=$HOME/models/gemma-4-12B-it-4bit
MIX=$FT/mix-pilot1
ADAPTER=$P1/adapters
FUSED=$P1/fused
OLLAMA_NAME=gemma4-12b-ailang-ft1
BIN=$FT/bin/ailang-ft
PY=$FT/.venv/bin/python
OUT=$REPO/eval_results/finetune-pilot1
mkdir -p "$STATE" "$OUT"
LOG=$P1/weekend.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
done_(){ [ -f "$STATE/$1.done" ]; }
mark(){ date -u +%FT%TZ > "$STATE/$1.done"; log "STAGE $1 DONE"; }
die(){ log "FATAL: $*"; exit 1; }

cd "$REPO" || exit 1
export AILANG_MODELS_PATH="$REPO/internal/modelreg/models.yml"
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800        # per-call deadline = lane budget (Phase 0 confound)
HELD=$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")

source tools/launchd/rig-lock.sh
REQ=finetune-pilot1
rig_lock_request_yield "$REQ" $((48*3600)) || log "yield request not placed (another handoff in force); waiting normally"
rig_lock_acquire wait "$REQ" || die "rig lock acquire failed"
rig_lock_clear_yield "$REQ"   # never leave our own handoff in force (self-yield deadlock, 2026-11-05)
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
# keep-alive: no EXIT trap of our own (it would replace the lock helper's release
# trap); the loop ends by itself when this script exits.
PARENT=$$
( while kill -0 "$PARENT" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

stage_preflight(){
  [ -x "$BIN" ] || die "private binary missing: $BIN"
  "$BIN" prompt --version ft-none >/dev/null 2>&1 || die "private binary lacks ft-none prompt"
  [ -d "$BASE" ] || die "base model missing: $BASE"
  free_gb=$(df -g "$HOME" | awk 'NR==2{print $4}'); [ "$free_gb" -ge 80 ] || die "need >=80 GB free, have ${free_gb}"
  for j in dev.ailang.os-rotation-filler dev.ailang.nightly-eval; do
    [ -f "$HOME/.ailang/state/launchd-hold/$j" ] || log "WARNING: no launchd hold for $j (rotation may contend)"
  done
  "$PY" -I tools/finetune/check_mix.py --mix "$MIX" --model "$BASE" 2>&1 | grep -v PyTorch | tee -a "$LOG"
  [ "${PIPESTATUS[0]}" = 0 ] || die "mix gate failed"
}

stage_train(){
  "$PY" -I tools/finetune/train.py -c tools/finetune/pilot1.yaml --model "$BASE" --data "$MIX" \
      --adapter-path "$ADAPTER" > "$P1/train.log" 2>&1
  rc=$?; sed 's/\x1b\[[0-9;]*m//g' "$P1/train.log" | grep -E "val |save" | tail -6 | tee -a "$LOG"
  [ $rc = 0 ] && [ -f "$ADAPTER/adapters.safetensors" ] || die "training failed rc=$rc (see $P1/train.log)"
}

strip_config(){ # $1 = level: 1 drop audio, 2 also drop vision tokens
  "$PY" -I - "$FUSED/config.json" "$1" <<'EOF'
import json,sys
p,lvl=sys.argv[1],int(sys.argv[2]); c=json.load(open(p))
drop=["audio_config","audio_token_id","boa_token_id","eoa_token_index","eoa_token_id"]
if lvl>=2: drop+=["vision_config","image_token_id","video_token_id","boi_token_id","eoi_token_id"]
for k in drop: c.pop(k,None)
json.dump(c,open(p,"w"),indent=2); print("config stripped, level",lvl)
EOF
}

smoke(){
  "$PY" -I -c "import json;print(json.dumps({'model':'$OLLAMA_NAME','prompt':'Write an AILANG program that prints 42.','stream':False,'options':{'num_predict':80,'temperature':0}}))" > "$P1/smoke.json"
  curl -s --max-time 600 http://127.0.0.1:11434/api/generate -H "X-Rig-Lease: $AILANG_RIG_LEASE" -d @"$P1/smoke.json" > "$P1/smoke.out"
  grep -q '"response"' "$P1/smoke.out"
}

stage_serve(){
  "$PY" -I -m mlx_lm.fuse --model "$BASE" --adapter-path "$ADAPTER" --save-path "$FUSED" --dequantize > "$P1/fuse.log" 2>&1 || die "fuse failed"
  for lvl in 1 2; do
    strip_config $lvl | tee -a "$LOG"
    printf 'FROM %s\n' "$FUSED" > "$P1/Modelfile"
    ollama create "$OLLAMA_NAME" -q q8_0 -f "$P1/Modelfile" > "$P1/create.log" 2>&1 \
      || ollama create "$OLLAMA_NAME" -f "$P1/Modelfile" >> "$P1/create.log" 2>&1 || die "ollama create failed"
    if smoke; then log "serve OK at strip level $lvl: $(python3 -I -c "import json;print(json.load(open('$P1/smoke.out'))['response'][:120])")"; return 0; fi
    log "smoke failed at strip level $lvl: $(head -c 300 "$P1/smoke.out")"
  done
  die "fine-tune does not load in ollama; fallback = mlx_lm.server lane (see design doc)"
}

run_eval(){ # name model trials prompt_version agent(0/1) wallclock
  local name=$1 model=$2 trials=$3 pv=$4 agent=$5 wc=$6 extra=()
  [ -n "$pv" ] && extra+=(--prompt-version "$pv")
  [ "$agent" = 1 ] && extra+=(--agent)
  ( [ -n "$pv" ] && export AILANG_EVAL_TRAPS_CARD=off
    "$BIN" eval-suite --models "$model" --benchmarks "$HELD" --langs ailang --trials "$trials" \
      --parallel 1 --skip-existing --max-wall-clock "$wc" ${extra[@]+"${extra[@]}"} --output "$OUT/$name" ) > "$OUT/$name.log" 2>&1
  # Guard against the loader's silent fallback to the full prompt: every banked
  # result must carry the requested prompt version.
  python3 -I - "$OUT/$name" "${pv:-ACTIVE}" <<'EOF' | tee -a "$LOG"
import json,glob,sys,collections
d,want=sys.argv[1],sys.argv[2]; c=collections.Counter(); n=0; ok=0
for p in glob.glob(d+"/**/*.json",recursive=True):
    try: r=json.load(open(p))
    except Exception: continue
    if not isinstance(r,dict) or "compile_ok" not in r: continue
    n+=1; ok+=bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"]); c[r.get("prompt_version") or "?"]+=1
print(f"{d.split('/')[-1]}: {ok}/{n} pass; prompt versions {dict(c)}")
if want!="ACTIVE" and any(k!=want for k in c): print("PROMPT-VERSION MISMATCH: arm is void"); sys.exit(3)
EOF
  [ "${PIPESTATUS[0]}" = 0 ] || die "$name: prompt-version guard failed"
}

stage_eval_B_std(){ run_eval B_std ollama-gemma4-12b-ft1 3 ft-slim-v1 0 150m; }
stage_eval_C_std(){ run_eval C_std ollama-gemma4-12b-ft1 3 ft-none 0 150m; }
stage_eval_B_agent(){ run_eval B_agent pi-gemma4-12b-ft1 1 ft-slim-v1 1 5h; }
stage_eval_C_agent(){ run_eval C_agent pi-gemma4-12b-ft1 1 ft-none 1 5h; }
stage_eval_A_std(){ run_eval A_std ollama-gemma4-12b 3 "" 0 150m; }
stage_summary(){ python3 -I tools/finetune/summarize_pilot.py "$OUT" | tee -a "$LOG"; }

STAGES=${*:-preflight train serve eval_B_std eval_C_std eval_B_agent eval_C_agent eval_A_std summary}
for s in $STAGES; do
  if done_ "$s"; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "weekend pilot finished"
