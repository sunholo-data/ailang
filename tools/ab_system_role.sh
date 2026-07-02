#!/usr/bin/env bash
# A/B: does delivering the AILANG system prompt in the PERSISTENT SYSTEM ROLE
# (vs folding it into the user message, where compaction strips it on long runs)
# improve convergence? This is the clean test that was never valid before — the
# prior 2026-06-18 "-2/18, not the lever" A/B ran while the prompt was chars=0
# (see memory project_motoko_system_prompt_delivery / _known).
#
#   Control   = AILANG_MOTOKO_SYSTEM_ROLE=0  -> legacy fold-into-user-message
#               (the OLD behaviour: teaching lost to compaction on long runs)
#   Treatment = AILANG_MOTOKO_SYSTEM_ROLE=1  -> persistent system-role delivery
#               (the new default; SYSTEM_MD survives compaction)
#
# The end-to-end delivery guard (M-RIG-RELIABILITY) records system_md per run,
# so we CONFIRM the arms actually differ: treatment must be 'set', control 'unset'.
#
# Usage:  tools/ab_system_role.sh [docx|BENCH,LIST] [TRIALS]
#   docx (default) -> docx_reimplement (the frontier; ~41min/run)
#
# Requires the rig FREE (single motoko env-server on :8080). Run from repo root.
set -euo pipefail
cd "$(dirname "$0")/.."

SET_ARG="${1:-docx}"
TRIALS="${2:-3}"
MODEL="motoko-local-qwen3-6-35b-a3b-mxfp8"
LOGDIR="$HOME/dev/mk-ast/.motoko/logfile"

case "$SET_ARG" in
  docx) BENCHES="docx_reimplement" ;;
  *)    BENCHES="$SET_ARG" ;;
esac

# Coordinate with the os-rotation filler via the shared rig-lock: acquire WAIT so we block until
# the current chunk releases, then HOLD it so the next filler tick defers. Any :8080 listener while
# we hold the lock is an orphan — clear it. Auto-released on EXIT.
# shellcheck source=/dev/null
source "$(dirname "$0")/launchd/rig-lock.sh"
echo "== waiting for the rig (rig-lock; any current os-rolling chunk must finish) =="
rig_lock_acquire wait
# HARD SAFETY: never proceed while ANOTHER eval-suite is alive (a wedged chunk fighting over :8080
# api_errors every run at 0ms). Our own eval-suite hasn't started here, so any match is someone else's.
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

echo "== building current binary (delivery default-on + guard) =="
make quick-install >/dev/null

OUT="eval_results/ab_sysrole_${SET_ARG}_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUT"
COMMON=(--agent --models "$MODEL" --benchmarks "$BENCHES" --langs ailang --trials "$TRIALS" --microrag on)

# Run one arm and record exactly which session logs it produced (single-writer under the rig-lock,
# so the before/after diff is clean — no cross-arm contamination).
run_arm() {
  local name="$1" before; shift
  before="$(mktemp)"
  ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort > "$before" || true
  "$@"
  comm -13 "$before" <(ls "$LOGDIR"/session_*.jsonl 2>/dev/null | sort) > "$OUT/$name.sessions.txt" || true
  rm -f "$before"
}

# Report the system_md delivery state across an arm's sessions — this is the
# INDEPENDENT-VARIABLE CHECK: the arms are only comparable if delivery differed.
delivery_state() {
  local list="$1"
  [ -s "$list" ] || { echo "    (no sessions)"; return; }
  # shellcheck disable=SC2046
  python3 - "$@" <<'PY'
import json, sys
from collections import Counter
c = Counter()
for path in [l.strip() for l in open(sys.argv[1]) if l.strip()]:
    state = "(no runtime_config_resolved)"
    try:
        for line in open(path):
            try: o = json.loads(line)
            except: continue
            if o.get("type") == "runtime_config_resolved":
                state = o.get("system_md", "(missing)"); break
    except FileNotFoundError:
        state = "(session file gone)"
    c[state] += 1
for k, v in c.most_common():
    print(f"    system_md={k!r}: {v} run(s)")
PY
}

echo "== CONTROL (AILANG_MOTOKO_SYSTEM_ROLE=0 — legacy user-message fold) =="
run_arm control env AILANG_MOTOKO_SYSTEM_ROLE=0 \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/control"

echo "== TREATMENT (AILANG_MOTOKO_SYSTEM_ROLE=1 — persistent system role) =="
run_arm treatment env AILANG_MOTOKO_SYSTEM_ROLE=1 \
  ailang eval-suite "${COMMON[@]}" --output "$OUT/treatment"

echo
echo "================= RESULTS ================="
for arm in control treatment; do
  echo "----- $arm: DELIVERY CHECK (independent variable) -----"
  delivery_state "$OUT/$arm.sessions.txt"
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
echo "Clean iff: CONTROL system_md='unset' AND TREATMENT system_md='set'."
echo "Then compare: pass-rate delta AND compile-stuck-rate delta (treatment should converge more)."
