#!/usr/bin/env bash
# check_model_fit.sh — does a model fit the rig's memory/shape envelope?
#
# The rig is a single Mac Studio (M4 Max, 128 GB unified, ~546 GB/s bandwidth).
# Token generation is MEMORY-BANDWIDTH-bound, not compute-bound, so the eval
# rotation runs --parallel 1 (concurrent requests thrash bandwidth and trigger
# TTFT timeouts — see the skill). This script reports whether a candidate model
# fits the single-stream memory budget and whether its SHAPE (MoE small-active)
# suits the hardware.
#
# Usage:
#   check_model_fit.sh <ollama-model-tag>     # measure a pulled model's real VRAM
#   check_model_fit.sh --disk <GB> [--active <B>]   # estimate before pulling
#
# Portable to macOS bash 3.2 (no mapfile/associative arrays).
set -uo pipefail

# --- Hardware budget (single-stream, p=1) -----------------------------------
TOTAL_GB=128
USABLE_GB=96      # Metal wired default ~75% of unified
RESERVE_GB=22     # OS + ailang server + opencode/pi subprocesses + KV headroom
BUDGET_GB=$((USABLE_GB - RESERVE_GB))   # ~74 GB safe ceiling for model resident
GREEN_GB=50       # comfortable: leaves slack even at long context
OLLAMA=http://localhost:11434

say() { printf '%s\n' "$*"; }
hr()  { printf -- '----------------------------------------\n'; }

verdict() { # $1 = resident GB (float ok)
  local r="$1"
  # integer compare on rounded value (bash has no float compare)
  local ri; ri=$(printf '%.0f' "$r")
  if   [ "$ri" -le "$GREEN_GB" ]; then say "VERDICT: ✅ GREEN — fits comfortably at p=1 (resident ~${r} GB ≤ ${GREEN_GB} GB)";
  elif [ "$ri" -le "$BUDGET_GB" ]; then say "VERDICT: 🟡 YELLOW — fits at p=1 but tight (resident ~${r} GB, ceiling ${BUDGET_GB} GB). Watch KV at long context.";
  else say "VERDICT: 🔴 RED — over budget (resident ~${r} GB > ${BUDGET_GB} GB). Would need a raised iogpu.wired_limit_mb and risks OS pressure. Prefer a smaller quant or a lower-active MoE.";
  fi
}

say "Rig envelope: ${TOTAL_GB} GB unified · usable ~${USABLE_GB} GB · reserve ${RESERVE_GB} GB → model ceiling ~${BUDGET_GB} GB (p=1)"
hr

if [ "${1:-}" = "--disk" ]; then
  DISK_GB="${2:-0}"
  ACTIVE_B=""
  [ "${3:-}" = "--active" ] && ACTIVE_B="${4:-}"
  # Resident ≈ weights (≈disk) + ~15% runtime/KV at a working context.
  RESIDENT=$(awk -v d="$DISK_GB" 'BEGIN{printf "%.1f", d*1.15}')
  say "Estimate from disk size: ${DISK_GB} GB on disk → ~${RESIDENT} GB resident (weights + ~15% KV/runtime)"
  [ -n "$ACTIVE_B" ] && say "Active params: ${ACTIVE_B}B"
  verdict "$RESIDENT"
  hr
  say "SHAPE check (manual): ideal = MoE with active ≤ ~8B (bandwidth-bound box)."
  say "  • A dense model of the same disk size will generate tokens MUCH slower."
  if [ -n "$ACTIVE_B" ]; then
    ai=$(printf '%.0f' "$ACTIVE_B")
    if [ "$ai" -le 8 ]; then say "  • active ${ACTIVE_B}B ≤ 8B → ✅ good shape (snappy agent loops)";
    else say "  • active ${ACTIVE_B}B > 8B → 🟡 slower per-token; verify TTFT/throughput before rotation"; fi
  fi
  exit 0
fi

MODEL="${1:-}"
if [ -z "$MODEL" ]; then
  say "usage: check_model_fit.sh <ollama-model-tag>   |   check_model_fit.sh --disk <GB> [--active <B>]"
  exit 2
fi

# Pulled? Get on-disk size + quant + param size from /api/tags.
TAGS_JSON=$(curl -s --max-time 5 "$OLLAMA/api/tags" 2>/dev/null)
if [ -z "$TAGS_JSON" ]; then say "ollama not reachable at $OLLAMA — start it or use --disk to estimate"; exit 1; fi

python3 - "$MODEL" <<'PY' || { say "model '$MODEL' not found locally — 'ollama pull $MODEL' first, or use --disk to estimate"; exit 1; }
import sys,json,urllib.request
model=sys.argv[1]
tags=json.load(urllib.request.urlopen("http://localhost:11434/api/tags",timeout=5))
m=next((x for x in tags.get("models",[]) if x["name"]==model or x["name"].split(":")[0]==model.split(":")[0]),None)
if not m: sys.exit(1)
gb=m["size"]/1e9
det=m.get("details",{})
print(f"On disk:    {gb:.1f} GB")
print(f"Params:     {det.get('parameter_size','?')}")
print(f"Quant:      {det.get('quantization_level','?')}")
PY

# Warm it briefly and read REAL resident VRAM from /api/ps (most accurate).
say ""
say "Warming model to measure real resident VRAM (single short request)…"
curl -s --max-time 120 "$OLLAMA/api/generate" -d "{\"model\":\"$MODEL\",\"prompt\":\"ok\",\"stream\":false,\"options\":{\"num_predict\":1}}" >/dev/null 2>&1 || true
RES=$(curl -s --max-time 5 "$OLLAMA/api/ps" 2>/dev/null | python3 -c "
import sys,json
try: d=json.load(sys.stdin)
except: print(''); sys.exit()
import os
m=os.environ.get('M','')
for x in d.get('models',[]):
    if x['name']==m or x['name'].split(':')[0]==m.split(':')[0]:
        print('%.1f'%(x.get('size_vram',0)/1e9)); break
" M="$MODEL" 2>/dev/null)

hr
if [ -n "$RES" ] && [ "$RES" != "0.0" ]; then
  say "Measured resident VRAM: ${RES} GB (from /api/ps)"
  verdict "$RES"
else
  say "Could not read resident VRAM from /api/ps (model may have unloaded). Re-run after warmup, or use --disk to estimate."
fi
hr
say "Reminder: the rotation runs --parallel 1 (bandwidth). Do NOT raise it. Register"
say "the model as BOTH opencode-<id> and pi-<id> (cross-harness pair) with"
say "budgets.hard_timeout_secs: 600 and pricing 0 — see the skill."
