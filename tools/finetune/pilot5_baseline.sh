#!/bin/bash
# Pilot-5 stage 0 (docs/docs/finetuning/pilot-5.md): the single-shot language test.
#
# Standard mode, no tools, no self-repair: task in, one program out, compile + run + compare.
# Arms differ only in the teaching prompt:  none (ft-none) | slim (ft-slim-v1) | full (v0.16.12).
# Same frozen inputs as the pilot-4 arms: pinned compiler, pinned stdlib, benchmark snapshot.
#
#   tools/finetune/pilot5_baseline.sh [LANE] [TRIALS] [BENCHMARKS] [ARMS]
#   LANE default ollama-gemma4-12b (the untrained model); a fine-tune passes its own lane.
# Resumable (--skip-existing). Results: eval_results/finetune-pilot5/singleshot/<lane>/<arm>.
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
FT=$HOME/models/finetune
BIN=$FT/bin/ailang-ft3
PINNED=$FT/bin/pinned
EVALROOT=$FT/evalroot
export AILANG_STDLIB_PATH=$FT/std-ft3/std
export AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800
[ -f "$AILANG_STDLIB_PATH/list.ail" ] || { echo "pinned stdlib missing" >&2; exit 1; }
[ -x "$PINNED/ailang" ] || { echo "pinned ailang missing" >&2; exit 1; }
[ -d "$EVALROOT/benchmarks" ] && [ ! -e "$EVALROOT/std" ] || { echo "bad eval root" >&2; exit 1; }
cd "$REPO" || exit 1
LANE=${1:-ollama-gemma4-12b}
TRIALS=${2:-5}
BENCHES=${3:-$(python3 -I -c "import json;h=json.load(open('tools/finetune/heldout.json'));print(','.join(h['core']+h['stretch']))")}
ARMS=${4:-none slim full}
OUT=$REPO/eval_results/finetune-pilot5/singleshot/$LANE
mkdir -p "$OUT"
LOG=$OUT/run.log
log(){ echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG"; }
pv(){ case $1 in none) echo ft-none;; slim) echo ft-slim-v1;; full) echo v0.16.12;; *) echo "bad arm $1" >&2; exit 1;; esac; }

source tools/launchd/rig-lock.sh
rig_lock_acquire wait finetune-pilot5 || { log "FATAL: rig lock acquire failed"; exit 1; }
rig_lock_clear_yield finetune-pilot5
log "rig lock held: $(cat "$RIG_LOCK_DIR/holder"); disk $(df -g "$HOME" | awk 'NR==2{print $4}') GiB free"
parent=$$
( while kill -0 "$parent" 2>/dev/null; do sleep 300; [ -d "$RIG_LOCK_DIR" ] && touch "$RIG_LOCK_DIR"; done ) &

for arm in $ARMS; do
  v=$(pv "$arm") || exit 1
  log "ARM $arm ($v) START lane=$LANE trials=$TRIALS"
  ( cd "$EVALROOT" && env PATH="$PINNED:$PATH" "$BIN" eval-suite --models "$LANE" --benchmarks "$BENCHES" \
      --langs ailang --trials "$TRIALS" --parallel 1 --no-self-repair --skip-existing \
      --prompt-version "$v" --output "$OUT/$arm" ) >> "$OUT/$arm.log" 2>&1
  python3 -I tools/finetune/singleshot_score.py "$OUT/$arm" --want "$v" --name "$arm" | tee -a "$LOG"
done
log "single-shot baseline finished: $LANE [$ARMS]"
