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

# Guard: never collide with a run in progress (the :8080 zombie/contention failure mode).
if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "ERROR: :8080 is busy — the rig is in use. Wait for it to free, then re-run." >&2
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
