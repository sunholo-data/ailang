#!/bin/bash
# Fine-tune pilot-4 runner (docs/docs/finetuning/pilot-4.md). Resumable: $STATE/<stage>.done.
#
#   (round-2 collection is separate: OP=~/models/finetune/onpolicy-r2 LANE=pi-gemma4-12b-ft3-trace
#    tools/finetune/collect_onpolicy.sh, GPU)
#   pairs    round-2 trace -> failing file states with their attempt history (CPU)
#   repair   Opus fixes each, verified (Claude subscription; no GPU)
#   data     pool round 1 + round 2 -> episodes (+ loop-break, + build-up) -> thoughts -> mix (CPU)
#   train    train2.py from the PILOT-2 adapter, 1 epoch (GPU, rig lock)            D-FT-13
#   serve    fuse -> strip audio config -> ollama create gemma4-12b-ailang-ft4
#   gate     30-min behaviour check on TRAINING benchmarks before the long evals    D-FT-16
#   eval_B   held-out 24, fine-tune + slim prompt, TRACED (dropped calls counted)
#   eval_A   held-out 24, base 12B + full teaching prompt, TRACED (once; reusable)
#   eval_A2  held-out 24, base 12B + slim prompt (diagnostic)
#   eval_D   held-out 24, pilot-3 + full teaching prompt (diagnostic)
#   summary
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
OP1=$FT/onpolicy
OP2=$FT/onpolicy-r2
P4=$FT/pilot4
STATE=$P4/state
BASE=$HOME/models/gemma-4-12B-it-4bit
EP=$FT/episodes-v4
MIX=$FT/mix-pilot4
ADAPTER=$P4/adapters
RESUME=$FT/pilot2/adapters/adapters.safetensors
FUSED=$P4/fused
OLLAMA_NAME=gemma4-12b-ailang-ft4
BIN=$FT/bin/ailang-ft3          # eval-suite binary, rebuilt 2026-11-10 with the #2151 fix
# The suite runs with cwd = the repo root, and stdlib resolution prefers <cwd>/std over the
# embedded copy, so a rebase of the checkout mid-arm changes the stdlib the scoring run loads
# (2026-11-12: #2230 added @deprecated, which the installed 03:31 build cannot parse). Pin it to
# the installed binary's commit: git archive 6d28e1ad5 std | tar -x -C $FT/std-ft3
export AILANG_STDLIB_PATH=$FT/std-ft3/std
[ -f "$AILANG_STDLIB_PATH/list.ail" ] || { echo "pinned stdlib missing: $AILANG_STDLIB_PATH" >&2; exit 1; }
# The env pin alone is not enough: scoring passes --stdlib-path <cwd>/std whenever the cwd holds
# a stdlib, and a flag beats the env. So the suite runs from a directory with no std/, holding a
# snapshot of benchmarks/ (rsync -a benchmarks/ $FT/evalroot/benchmarks/), which also freezes the tasks.
EVALROOT=$FT/evalroot
[ -d "$EVALROOT/benchmarks" ] && [ ! -e "$EVALROOT/std" ] || { echo "bad eval root: $EVALROOT" >&2; exit 1; }
in_evalroot(){ ( cd "$EVALROOT" && "$@" ); }
PY=$FT/.venv/bin/python
OUT=$REPO/eval_results/finetune-pilot4
PORT=11437
mkdir -p "$STATE" "$OUT"
LOG=$P4/run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
done_(){ [ -f "$STATE/$1.done" ]; }
mark(){ date -u +%FT%TZ > "$STATE/$1.done"; log "STAGE $1 DONE"; }
die(){ log "FATAL: $*"; exit 1; }
# Disk guard (2026-11-10: the rig was at 14 GiB free). Fail BEFORE a stage that writes big files,
# never halfway through one. GiB each stage needs, with headroom:
#   train ~6 (adapter checkpoints), serve ~40 (22 GB fused copy + 12 GB ollama import), evals ~5.
need_disk(){ # stage GiB
  local free; free=$(df -g "$HOME" | awk 'NR==2{print $4}')
  log "disk: ${free} GiB free (stage $1 needs ${2})"
  [ "$free" -ge "$2" ] || die "only ${free} GiB free, $1 needs ${2}. Regenerable candidates: \
fused/ dirs of served pilots (ollama holds its own copy), intermediate 0*_adapters.safetensors checkpoints"
}
cd "$REPO" || exit 1
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800
HELD=$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")

gpu_lock(){
  source tools/launchd/rig-lock.sh
  rig_lock_acquire wait finetune-pilot4 || die "rig lock acquire failed"
  rig_lock_clear_yield finetune-pilot4
  log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
  local parent=$$
  ( while kill -0 "$parent" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &
}

stage_pairs(){
  [ -s "$OP2/trace.jsonl" ] || die "no round-2 trace: run the collection first"
  "$PY" -I tools/finetune/onpolicy_pairs.py --trace "$OP2/trace.jsonl" --out "$OP2/items.jsonl" \
    --per-kind "${PER_KIND:-10}" | tee "$OP2/pairs_report.json" | tail -5 | tee -a "$LOG"
}

stage_repair(){
  env -u ANTHROPIC_API_KEY "$PY" -I tools/finetune/stale_fix.py fix --backend claude-cli --model claude-opus-5-5 \
    --prompt onpolicy --items "$OP2/items.jsonl" --out "$OP2" --pairs-out onpolicy_pairs.jsonl \
    --limit 100000 --workers "${WORKERS:-4}" --rounds 4 | tee -a "$LOG"
}

stage_data(){
  # Round 1's repairs carry `history` via onpolicy_pairs.py --enrich-pairs (no repair re-bought).
  [ -s "$OP1/onpolicy_pairs_v2.jsonl" ] || die "round-1 pairs not enriched with history"
  "$PY" -I tools/finetune/synth_episodes.py --out "$EP" --onpolicy "$OP1/onpolicy_pairs_v2.jsonl,$OP2/onpolicy_pairs.jsonl" \
    --build-min-chars 2500 | tail -40 | tee -a "$LOG"
  "$PY" -I tools/finetune/thoughts.py --episodes "$EP/episodes.jsonl" --out "$EP/episodes_thought.jsonl" | tee -a "$LOG"
  "$PY" -I tools/finetune/build_mix2.py --episodes "$EP/episodes_thought.jsonl" --out "$MIX" \
    --solve-per-bench 4 --repair-per-kind 2 --example-repeat 1 --onpolicy-per-kind 8 --build-per-bench 4 \
    --max-steps 24 --max-write-chars 6000 --tokenizer "$BASE" --max-tokens 12288 | tee -a "$LOG"
}

stage_train(){
  need_disk train 6
  local n upd
  n=$(wc -l < "$MIX/train.jsonl" | tr -d ' ')
  upd=$(( n / 4 ))
  sed -e "s/__ITERS__/$n/" -e "s/__UPDATES__/$upd/" tools/finetune/pilot4.yaml > "$P4/pilot4.resolved.yaml"
  echo "resume_adapter_file: $RESUME" >> "$P4/pilot4.resolved.yaml"
  log "train: $n samples, 1 epoch, $upd optimizer updates, from $RESUME"
  "$PY" -I tools/finetune/train2.py -c "$P4/pilot4.resolved.yaml" --model "$BASE" --data "$MIX" \
    --adapter-path "$ADAPTER" > "$P4/train.log" 2>&1
  rc=$?; sed 's/\x1b\[[0-9;]*m//g' "$P4/train.log" | tr '\r' '\n' | grep -E " val |save" | tail -8 | tee -a "$LOG"
  [ $rc = 0 ] && [ -f "$ADAPTER/adapters.safetensors" ] || die "training failed rc=$rc"
}

stage_serve(){
  need_disk serve 40
  "$PY" -I -m mlx_lm.fuse --model "$BASE" --adapter-path "$ADAPTER" --save-path "$FUSED" --dequantize > "$P4/fuse.log" 2>&1 || die "fuse failed"
  "$PY" -I - "$FUSED/config.json" <<'EOF'
import json,sys
p=sys.argv[1]; c=json.load(open(p))
for k in ["audio_config","audio_token_id","boa_token_id","eoa_token_index","eoa_token_id"]: c.pop(k,None)
json.dump(c,open(p,"w"),indent=2)
EOF
  printf 'FROM %s\nPARAMETER temperature 1\nPARAMETER top_k 64\nPARAMETER top_p 0.95\n' "$FUSED" > "$P4/Modelfile"
  ollama create "$OLLAMA_NAME" -q mxfp8 -f "$P4/Modelfile" > "$P4/create.log" 2>&1 || die "ollama create failed"
  ollama show "$OLLAMA_NAME" --parameters | tee -a "$LOG"
}

# Traced lanes exist only in pilot-4's own models file: they route through the trace proxy
# (pi provider ollama-trace -> :11437), so they never belong in the repo registry.
trace_models(){
  "$PY" -I - "$REPO/internal/modelreg/models.yml" "$P4/models.trace.yml" <<'EOF'
import re,sys
src,out=sys.argv[1],sys.argv[2]; s=open(src).read()
add=""
for lane in ("pi-gemma4-12b","pi-gemma4-12b-ft3","pi-gemma4-12b-ft4"):
    i=s.index(f"\n  {lane}:\n")+1; j=re.compile(r"^  [A-Za-z0-9#]",re.M).search(s,i+5).start()
    blk=s[i:j].replace(f"  {lane}:",f"  {lane}-trace:",1).replace('agent_model_name: "ollama-rig/','agent_model_name: "ollama-trace/')
    add+=blk
k=s.index("\nmodels:\n")+len("\nmodels:\n")
open(out,"w").write(s[:k]+add+s[k:])
EOF
  for m in gemma4:12b-mxfp8 gemma4-12b-ailang-ft3 "$OLLAMA_NAME"; do
    "$PY" -I - "$m" <<'EOF'
import json,os,sys
p=os.path.expanduser("~/.pi/agent/models.json"); d=json.load(open(p)); m=sys.argv[1]
prov=d["providers"]["ollama-trace"]
if m not in [x["id"] for x in prov["models"]]:
    tpl=dict(next(x for x in prov["models"] if x["id"].startswith("gemma4"))); tpl["id"]=m
    if "name" in tpl: tpl["name"]=m
    prov["models"].append(tpl); json.dump(d,open(p,"w"),indent=2)
EOF
  done
}

with_proxy(){ # trace-file cmd...
  local tr=$1; shift
  python3 -I tools/finetune/trace_proxy.py --listen $PORT --upstream http://127.0.0.1:11434 --out "$tr" >> "$P4/proxy.log" 2>&1 &
  local px=$!
  sleep 1; kill -0 $px 2>/dev/null || die "trace proxy did not start (port $PORT in use?)"
  "$@"; local rc=$?
  kill $px 2>/dev/null; wait $px 2>/dev/null
  return $rc
}

stage_gate(){
  need_disk gate 5
  # 1) after reading the template, does it act? 16 training benchmarks, from the round-2 trace.
  "$PY" -I tools/finetune/probe_after_read.py --trace "$OP2/trace.jsonl" --model "$OLLAMA_NAME" -n 16 \
    > "$P4/gate_probe.log" 2>&1 || die "probe failed"
  tail -1 "$P4/gate_probe.log" | tee -a "$LOG"
  # 2) dropped tool calls over 8 short TRAINING runs, traced
  trace_models
  local g=$P4/gate; mkdir -p "$g"
  local gb; gb=$(head -8 "$FT/train_benchmarks.txt" | paste -sd, -)
  with_proxy "$g/trace.jsonl" env AILANG_MODELS_PATH="$P4/models.trace.yml" AILANG_EVAL_TRAPS_CARD=off \
    "$BIN" eval-suite --agent --models pi-gemma4-12b-ft4-trace --benchmarks "$gb" --langs ailang --trials 1 \
    --parallel 1 --tool-policy lean --prompt-version ft-slim-v1 --output "$g/run" > "$g/eval.log" 2>&1
  python3 -I tools/finetune/trace_dropped.py "$g/trace.jsonl" | tee -a "$LOG"
  "$PY" -I - "$P4/gate_probe.log" "$g/trace.jsonl" <<'EOF' | tee -a "$LOG"
import json,subprocess,sys
probe=json.loads(open(sys.argv[1]).read().strip().splitlines()[-1])
d=json.loads(subprocess.run([sys.executable,"-I","tools/finetune/trace_dropped.py",sys.argv[2],"--json"],capture_output=True,text=True).stdout)
ok = probe["acted"] == probe["probed"] and (d["rate"] or 0) <= 0.01
print(f"GATE {'PASS' if ok else 'FAIL'}: acted after read {probe['acted']}/{probe['probed']}; dropped {d['dropped']}/{d['requests']} (limit 1%)")
sys.exit(0 if ok else 4)
EOF
  [ "${PIPESTATUS[0]}" = 0 ] || die "behaviour gate failed: see $LOG (override: touch $STATE/gate.done)"
}

run_arm(){ # name lane prompt_version traps(on/off)
  local name=$1 lane=$2 pv=$3 traps=$4
  trace_models
  ( [ "$traps" = off ] && export AILANG_EVAL_TRAPS_CARD=off
    with_proxy "$OUT/$name.trace.jsonl" in_evalroot env AILANG_MODELS_PATH="$P4/models.trace.yml" \
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
  echo "$name dropped: $(python3 -I tools/finetune/trace_dropped.py "$OUT/$name.trace.jsonl")" | tee -a "$LOG"
}

stage_eval_B(){ need_disk eval_B 5; run_arm B pi-gemma4-12b-ft4-trace ft-slim-v1 off; }
stage_eval_A(){ need_disk eval_A 5; run_arm A pi-gemma4-12b-trace v0.16.12 on; }   # base, once (D-FT-16); reusable later
# The two diagnostic arms the design doc asked for and no pilot ran (2026-11-12):
#   A2: base + slim prompt  -> how much of a fine-tune's slim-prompt score is the training?
#   D:  pilot-3 + full prompt -> did training damage the model, or just fail to replace the prompt?
stage_eval_A2(){ need_disk eval_A2 5; run_arm A2 pi-gemma4-12b-trace ft-slim-v1 off; }
stage_eval_D(){ need_disk eval_D 5; run_arm D pi-gemma4-12b-ft3-trace v0.16.12 on; }
stage_summary(){ grep -E "^(A|A2|B|D)[ :]|GATE" "$LOG" | tail -8; }

STAGES=${*:-pairs}
case " $STAGES " in *" train "*|*" serve "*|*" gate "*|*" eval_"*) gpu_lock ;; esac
for s in $STAGES; do
  if done_ "$s"; then log "skip $s (done)"; continue; fi
  log "STAGE $s START"; "stage_$s"; mark "$s"
done
log "pilot-4 runner finished: $STAGES"
