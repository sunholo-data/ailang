#!/bin/bash
# Clean up stopped Pilot-5d artifacts (dead rig lock holder, orphaned trace_proxy, superseded p5d Ollama tag)
# and launch Pilot-5e (mix-pilot5e v2: 100% verified <|channel>thought + compiler repairs in single & lean-pi formats).
set -euo pipefail

REPO=/Users/voightkampff/dev/sunholo-data/ailang
cd "$REPO"

LOCK_DIR=/Users/Shared/ailang/rig.lock.d
if [ -d "$LOCK_DIR" ]; then
  if [ -f "$LOCK_DIR/holder" ]; then
    hpid=$(awk '{print $1}' "$LOCK_DIR/holder" 2>/dev/null || true)
    if [ -n "$hpid" ] && ! kill -0 "$hpid" 2>/dev/null; then
      echo "[pilot5e_launch] clearing stale rig lock from dead PID $hpid"
      rm -rf "$LOCK_DIR"
    fi
  else
    echo "[pilot5e_launch] clearing holder-less orphan rig lock directory"
    if [ -f "$LOCK_DIR/children" ]; then
      awk '{print $1}' "$LOCK_DIR/children" | while read -r cpid; do
        [ -n "$cpid" ] && kill -TERM "$cpid" 2>/dev/null || true
      done
    fi
    rm -rf "$LOCK_DIR"
  fi
fi

# Ensure orphaned trace_proxy on port 11437 (if any) from cancelled Pilot-5d agent stage is stopped
pkill -f "trace_proxy.py --listen 11437" 2>/dev/null || true

# Reclaim ~12 GB from superseded Pilot-5d Ollama tag (adapters + eval logs remain preserved)
if ollama list 2>/dev/null | awk '{print $1}' | grep -qx "gemma4-12b-ailang-p5d:latest"; then
  echo "[pilot5e_launch] removing superseded Ollama model gemma4-12b-ailang-p5d:latest"
  ollama rm gemma4-12b-ailang-p5d:latest || true
fi

exec env RUN=e RANK=32 EPOCHS=1.5 MIX="$HOME/models/finetune/mix-pilot5e" \
  bash "$REPO/tools/finetune/pilot5_run.sh" train serve indist test agent
