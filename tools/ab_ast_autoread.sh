#!/usr/bin/env bash
# SETTLE THE QUESTION: does AST auto-read (ReadFile of a dependency .ail returns its
# compact iface instead of full source) make a good positive difference?
#
# Controlled A/B via MOTOKO_AST_AUTOREAD (no code change). Only MULTI-FILE benchmarks
# have dependencies to read, so single-file benches are pointless here.
#   OFF (control)   = MOTOKO_AST_AUTOREAD=0  -> ReadFile returns FULL dependency source
#   ON  (treatment) = MOTOKO_AST_AUTOREAD=1  -> ReadFile returns compact iface (the moat)
#
# Usage:  tools/ab_ast_autoread.sh [markdown|docx|BENCH,LIST] [TRIALS]
#   markdown (default) -> markdown_reimplement (fast; ~1 dep — tests "is iface ENOUGH?")
#   docx               -> docx_reimplement (slow; ~5 dep-reads — tests the token saving at scale)
#
# PRE-REGISTERED decision (decide BEFORE looking, so this gets SETTLED, not re-litigated):
#   • AUTOREAD HELPS  -> pass-rate ON >= OFF AND compile-stuck ON <= OFF AND tokens ON < OFF
#   • AUTOREAD HURTS  -> pass-rate ON < OFF (iface elides info the model needs -> full source wins)
#   • NEUTRAL         -> pass/compile-stuck within noise; keep ON only for the token saving
# Run N>=5 so a 1-trial swing isn't mistaken for signal.
#
# Requires the rig FREE (single motoko env-server on :8080). Run from repo root.
set -euo pipefail
cd "$(dirname "$0")/.."

SET_ARG="${1:-markdown}"
TRIALS="${2:-5}"
MODEL="motoko-local-qwen3-6-35b-a3b-mxfp8"
LOGDIR="$HOME/dev/mk-ast/.motoko/logfile"

case "$SET_ARG" in
  markdown) BENCHES="markdown_reimplement" ;;
  docx)     BENCHES="docx_reimplement" ;;
  *)        BENCHES="$SET_ARG" ;;
esac

# Coordinate with the os-rotation filler via the shared rig-lock instead of fighting it:
# acquire WAIT -> block until the current os-rolling chunk releases, then HOLD the lock so the
# next 45-min filler tick defers to us (it acquires nowait and skips). Auto-released on EXIT.
# shellcheck source=/dev/null
source "$(dirname "$0")/launchd/rig-lock.sh"
echo "== waiting for the rig (rig-lock; os-rolling chunk must finish — may be a while) =="
rig_lock_acquire wait
# Zombie guard: lock free but :8080 still held = a hung motoko (the port-8080-zombie failure mode).
if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "ERROR: rig-lock acquired but :8080 still held (zombie motoko). Clear it, then re-run." >&2
  exit 1
fi

echo "== building current binary =="
make quick-install >/dev/null

OUT="eval_results/ab_autoread_${SET_ARG}_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUT"
COMMON=(--agent --models "$MODEL" --benchmarks "$BENCHES" --langs ailang --trials "$TRIALS" --microrag on)

run_arm() {
  local name="$1" before; shift
  before="$(mktemp)"
  ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort > "$before" || true
  "$@"
  comm -13 "$before" <(ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort) > "$OUT/$name.sessions.txt" || true
  rm -f "$before"
}

echo "== OFF arm (control): MOTOKO_AST_AUTOREAD=0 — full dependency source =="
run_arm off env MOTOKO_AST_AUTOREAD=0 \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/off"

echo "== ON arm (treatment): MOTOKO_AST_AUTOREAD=1 — compact iface on dep reads =="
run_arm on env MOTOKO_AST_AUTOREAD=1 \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/on"

echo
echo "================= RESULTS (pre-registered criteria in header) ================="
for arm in off on; do
  echo "----- AUTOREAD=$arm : pass rate -----"
  ailang eval-summary "$OUT/$arm" 2>/dev/null || true
  echo "----- AUTOREAD=$arm : compile-stuck + median tokens -----"
  if [ -s "$OUT/$arm.sessions.txt" ]; then
    # shellcheck disable=SC2046
    python3 tools/analyze_stuck.py $(cat "$OUT/$arm.sessions.txt") 2>/dev/null \
      | grep -E "SUMMARY|^ +[0-9]+ +(COMPILE|BEHAV|MIXED|NOT)" || true
    python3 - "$OUT/$arm.sessions.txt" <<'PY'
import json,sys,statistics
toks=[]
for p in open(sys.argv[1]):
    p=p.strip()
    if not p: continue
    try:
        for l in open(p):
            o=json.loads(l)
            if o.get('type')=='run_summary':
                u=o.get('usage',{}) or {}
                t=u.get('total_tokens') or (u.get('input_tokens',0)+u.get('output_tokens',0))
                if t: toks.append(t)
    except Exception: pass
if toks:
    print(f"  median total tokens: {int(statistics.median(toks)):,}  (n={len(toks)})")
PY
  fi
  echo
done
echo "Decide by the header rule. Artifacts: $OUT"
