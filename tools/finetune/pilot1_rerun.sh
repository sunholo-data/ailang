#!/bin/bash
# Pilot-1 agent re-run with the session-protocol gate disarmed for evals (2026-11-06).
#
# Pilot-1's agent arms (B_agent 0/24, C_agent 0/24) are void: the pi session-protocol
# gate refused every write until the model called session_protocol_ack, which the
# fine-tune never learned. Fixed in 5dc3ba42b (Task.EvalRun -> AILANG_EVAL_RUN=1).
# This re-runs ALL THREE agent arms with the fixed binary so they share one
# condition, including base 12B (arm A), whose Phase 0 passes had to ack first.
#
# Waits for the rig lock (pilot-1's runner holds it until it exits), then:
#   A_agent2  base 12B, full teaching prompt pinned to v0.16.11 (= Phase 0 / A_std)
#   B_agent2  ft1, ft-slim-v1, traps card off
#   C_agent2  ft1, ft-none,    traps card off
#   summary2
# Resumable like weekend_pilot.sh: $STATE/<stage>.done markers.
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
P1=$FT/pilot1
STATE=$P1/state
BIN=$FT/bin/ailang-ft2          # built from 5dc3ba42b (EvalRun); never overwrite while running
OUT=$REPO/eval_results/finetune-pilot1
LOG=$P1/rerun.log
mkdir -p "$STATE" "$OUT"
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
done_(){ [ -f "$STATE/$1.done" ]; }
mark(){ date -u +%FT%TZ > "$STATE/$1.done"; log "STAGE $1 DONE"; }
die(){ log "FATAL: $*"; exit 1; }

cd "$REPO" || exit 1
export AILANG_MODELS_PATH="$REPO/internal/modelreg/models.yml"
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800
HELD=$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")
[ -x "$BIN" ] || die "binary missing: $BIN"

source tools/launchd/rig-lock.sh
REQ=finetune-pilot1-rerun
rig_lock_acquire wait "$REQ" || die "rig lock acquire failed"
rig_lock_clear_yield "$REQ"
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
PARENT=$$
( while kill -0 "$PARENT" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

run_eval(){ # name model prompt_version traps(on/off) wallclock
  local name=$1 model=$2 pv=$3 traps=$4 wc=$5
  ( [ "$traps" = off ] && export AILANG_EVAL_TRAPS_CARD=off
    "$BIN" eval-suite --agent --models "$model" --benchmarks "$HELD" --langs ailang --trials 1 \
      --parallel 1 --skip-existing --max-wall-clock "$wc" --prompt-version "$pv" --output "$OUT/$name" ) > "$OUT/$name.log" 2>&1
  python3 -I - "$OUT/$name" "$pv" <<'EOF' | tee -a "$LOG"
import json,glob,sys,collections
d,want=sys.argv[1],sys.argv[2]; c=collections.Counter(); n=ok=ack=0
for p in glob.glob(d+"/**/*.json",recursive=True):
    try: r=json.load(open(p))
    except Exception: continue
    if not isinstance(r,dict) or "compile_ok" not in r: continue
    n+=1; ok+=bool(r["compile_ok"] and r["runtime_ok"] and r["stdout_ok"]); c[r.get("prompt_version") or "?"]+=1
    ack+="session_protocol_ack" in (r.get("agent_tool_histogram") or {})
print(f"{d.split('/')[-1]}: {ok}/{n} pass; prompt versions {dict(c)}; runs that called session_protocol_ack: {ack}")
if any(k!=want for k in c): print("PROMPT-VERSION MISMATCH: arm is void"); sys.exit(3)
EOF
  [ "${PIPESTATUS[0]}" = 0 ] || die "$name: prompt-version guard failed"
}

stage_eval_A_agent2(){ run_eval A_agent2 pi-gemma4-12b v0.16.11 on 5h; }
stage_eval_B_agent2(){ run_eval B_agent2 pi-gemma4-12b-ft1 ft-slim-v1 off 5h; }
stage_eval_C_agent2(){ run_eval C_agent2 pi-gemma4-12b-ft1 ft-none off 5h; }
stage_summary2(){ python3 -I tools/finetune/summarize_pilot.py "$OUT" | tee -a "$LOG"; }

STAGES=${*:-eval_B_agent2 eval_A_agent2 eval_C_agent2 summary2}
for s in $STAGES; do
  if done_ "$s"; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "pilot-1 rerun finished"
