#!/bin/bash
# Pilot-3 on-policy collection (docs/docs/finetuning/pilot-2.md §Diagnosis).
#
# Pilot-2's held-out failures were its OWN syntax mistakes (2/3 parse errors), which the
# staleness-pair repair data never showed it. This runs the pilot-2 fine-tune on the
# TRAINING-split benchmarks only (never tools/finetune/heldout.json), through
# tools/finetune/trace_proxy.py, so every file state it writes and every tool result it sees
# is recorded. onpolicy_pairs.py turns the trace into repair items.
#
#   tools/finetune/collect_onpolicy.sh [trials] [benchmarks-file]     (resumable: --skip-existing)
#   OP=<dir> LANE=<trace lane> tools/finetune/collect_onpolicy.sh     round 2+: another model, own dir
#     (OP must hold models.onpolicy.yml with LANE in it; pilot-4 round 2: OP=onpolicy-r2,
#      LANE=pi-gemma4-12b-ft3-trace)
#
# Takes the GPU by the cooperative handoff (the rotation lends it at a benchmark boundary)
# and holds it for the whole collection; --max-wall-clock bounds the run.
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
OP=${OP:-$FT/onpolicy}
TRIALS=${1:-1}
BFILE=${2:-$FT/train_benchmarks.txt}
BIN=$FT/bin/ailang-ft3
LANE=${LANE:-pi-gemma4-12b-ft2-trace}
PORT=11437
cd "$REPO" || exit 1
mkdir -p "$OP"
grep -q "^  $LANE:" "$OP/models.onpolicy.yml" 2>/dev/null || { echo "FATAL: $LANE not in $OP/models.onpolicy.yml"; exit 1; }
LOG=$OP/collect.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }

# Never collect on a held-out task.
BENCH=$(python3 -I - "$BFILE" <<'EOF'
import json,sys
h=json.load(open("tools/finetune/heldout.json")); H=set(h["core"]+h["stretch"]+h.get("retired_excluded",[]))
b=[l.strip() for l in open(sys.argv[1]) if l.strip()]
bad=[x for x in b if x in H]
if bad: sys.exit(f"held-out benchmarks in the collection list: {bad}")
print(",".join(b))
EOF
) || exit 1
log "collect: $(echo "$BENCH" | tr ',' '\n' | wc -l | tr -d ' ') benchmarks x $TRIALS trial(s), lane $LANE"

source tools/launchd/rig-lock.sh
REQ=finetune-onpolicy
rig_lock_request_yield "$REQ" 1800 || log "another handoff in force; waiting normally"
rig_lock_acquire wait "$REQ" || { log "FATAL: rig lock acquire failed"; exit 1; }
rig_lock_clear_yield "$REQ"
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder")"
PARENT=$$
( while kill -0 "$PARENT" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

python3 -I tools/finetune/trace_proxy.py --listen $PORT --upstream http://127.0.0.1:11434 \
  --out "$OP/trace.jsonl" >> "$OP/proxy.log" 2>&1 &
PROXY=$!
# ADD to rig_lock_acquire's EXIT trap, never replace it: a bare `trap ... EXIT` here dropped the
# lock release, and the 2026-11-09 collection left rig.lock.d behind for every other GPU user.
trap "kill $PROXY 2>/dev/null; rm -rf '$RIG_LOCK_DIR'" EXIT
sleep 1
kill -0 $PROXY 2>/dev/null || { log "FATAL: trace proxy did not start (port $PORT in use?)"; exit 1; }

AILANG_MODELS_PATH=$OP/models.onpolicy.yml AILANG_EVAL_TRAPS_CARD=off AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800 \
  "$BIN" eval-suite --agent --models "$LANE" --benchmarks "$BENCH" --langs ailang --trials "$TRIALS" \
  --parallel 1 --skip-existing --max-wall-clock "${MAX_WALL:-14h}" --tool-policy lean \
  --prompt-version ft-slim-v1 --output "$OP/run" > "$OP/eval.log" 2>&1
log "eval-suite rc=$? ; trace requests: $(wc -l < "$OP/trace.jsonl")"
