#!/usr/bin/env bash
# A/B the convergence-workflow card vs the default dialect-traps card, on a
# benchmark set, via AILANG_EVAL_TRAPS_CARD (no code change — the eval harness
# already swaps the front-loaded card by path; see agent_runner_multi.go).
#
# Control   = default prompts/agent/dialect-traps.md
# Treatment = prompts/agent/convergence-workflow.md (traps + the convergence workflow)
#
# Usage:  tools/ab_convergence_card.sh [fast|docx|BENCH,LIST] [TRIALS]
#   fast (default) -> expression_evaluator,json_parse,lambda_calc,markdown_reimplement
#   docx           -> docx_reimplement (the real frontier; slow — needs the long budget)
#
# Requires the rig FREE (single motoko env-server on :8080). Run from repo root.
set -euo pipefail
cd "$(dirname "$0")/.."

SET_ARG="${1:-fast}"
TRIALS="${2:-5}"
MODEL="motoko-local-qwen3-6-35b-a3b-mxfp8"
TREATMENT_CARD="prompts/agent/convergence-workflow.md"
LOGDIR="$HOME/dev/mk-ast/.motoko/logfile"

case "$SET_ARG" in
  fast) BENCHES="expression_evaluator,json_parse,lambda_calc,markdown_reimplement" ;;
  docx) BENCHES="docx_reimplement" ;;
  *)    BENCHES="$SET_ARG" ;;
esac

# Coordinate with the os-rotation filler via the shared rig-lock: acquire WAIT so we block until
# the current chunk releases, then HOLD it so the next filler tick defers (it acquires nowait +
# skips). Any :8080 listener while we hold the lock is an orphan — clear it. Auto-released on EXIT.
# shellcheck source=/dev/null
source "$(dirname "$0")/launchd/rig-lock.sh"
echo "== waiting for the rig (rig-lock; current os-rolling chunk must finish — may be a while) =="
rig_lock_acquire wait
# HARD SAFETY (learned the hard way): the rig-lock's 6h stale-steal can JUMP a WEDGED os-rolling
# chunk that's still alive -> both fight over :8080 -> every run api_errors at 0ms. So never
# proceed while ANOTHER eval-suite is alive; wait it out. (Our own eval-suite hasn't started yet
# here, so any match is someone else's — a wedge that needs a manual kill.)
while pgrep -f "ailang eval-suite" >/dev/null 2>&1; do
  echo "== another eval-suite is alive (likely a WEDGED chunk) — waiting to avoid a collision; kill it to proceed =="
  sleep 60
done
for _ in 1 2 3 4 5 6; do lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1 || break; sleep 10; done
if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "== :8080 still held after 60s — clearing orphaned listener(s) (we hold the rig-lock) =="
  for pid in $(lsof -ti :8080 2>/dev/null); do kill "$pid" 2>/dev/null; done
  sleep 5
fi
if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "ERROR: :8080 still held after clear attempt — needs a manual kill, then re-run." >&2
  exit 1
fi

[ -f "$TREATMENT_CARD" ] || { echo "ERROR: missing $TREATMENT_CARD" >&2; exit 1; }

echo "== building current binary (60-step cap, etc.) =="
make quick-install >/dev/null

OUT="eval_results/ab_conv_${SET_ARG}_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUT"
COMMON=(--agent --models "$MODEL" --benchmarks "$BENCHES" --langs ailang --trials "$TRIALS" --microrag on)

# Run one arm and record exactly which session logs it produced (for analyze_stuck).
run_arm() {
  local name="$1" before; shift
  before="$(mktemp)"
  ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort > "$before" || true
  "$@"
  comm -13 "$before" <(ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort) > "$OUT/$name.sessions.txt" || true
  rm -f "$before"
}

echo "== CONTROL (default dialect-traps card) =="
run_arm control env -u AILANG_EVAL_TRAPS_CARD \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/control"

echo "== TREATMENT (convergence-workflow card) =="
run_arm treatment env AILANG_EVAL_TRAPS_CARD="$TREATMENT_CARD" \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/treatment"

echo
echo "================= RESULTS ================="
for arm in control treatment; do
  echo "----- $arm: pass rate -----"
  ailang eval-summary "$OUT/$arm" 2>/dev/null || true
  echo "----- $arm: failure-mode histogram (compile-stuck = the dialect wall) -----"
  if [ -s "$OUT/$arm.sessions.txt" ]; then
    # shellcheck disable=SC2046
    python3 tools/analyze_stuck.py $(cat "$OUT/$arm.sessions.txt") 2>/dev/null \
      | grep -E "MODE:|SUMMARY|^ +[0-9]+ +(COMPILE|BEHAV|MIXED|NOT)" || true
  fi
  echo
done
echo "Artifacts: $OUT  (per-arm session lists in *.sessions.txt)"
echo "Compare: pass-rate delta AND compile-stuck-rate delta (treatment should lower compile-stuck)."
