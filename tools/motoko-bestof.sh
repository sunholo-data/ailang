#!/usr/bin/env bash
# motoko-bestof — best-of-N deployment wrapper for motoko (M-AILANG-NATIVE-HARNESS P1).
#
# Runs motoko N independent times on the same task (each in an isolated workspace copy),
# then uses `ailang select-best` (typecheck + run; runs > typechecks > neither) to pick the
# candidate solution that actually COMPILES + RUNS — the reference-free exact selector that a
# general harness lacks. The winning solution is copied back into WORKDIR.
#
# Why a wrapper and not a motoko extension: the on_solver_candidate ExtCtx carries no caps/
# entrypoint, so an extension can only `ailang check` (== DP7, A/B'd as noise), not `ailang run`.
# The validated +6.8pp lever is RUN-based, so it must live where caps+entry are known — here.
#
# Usage:
#   WORKDIR=<project> MODEL=<m> N=3 CAPS=IO ENTRY=main SOLUTION=solution.ail \
#     tools/motoko-bestof.sh "<task prompt>"
#
# Env:
#   WORKDIR   base project dir motoko edits (required)
#   MODEL     model string (default: ollama/qwen3.6:35b-a3b-mxfp8)
#   N         number of independent candidates (default: 3)
#   CAPS      capabilities for running candidates (default: IO)
#   ENTRY     entrypoint function (default: main)
#   SOLUTION  solution file path relative to WORKDIR (default: solution.ail)
#   MOTOKO    motoko launcher (default: arniwesth run-agent.sh)
#   MOTOKO_CONFIG / AILANG_OLLAMA_MAX_TOKENS  forwarded to motoko
#   MOTOKO_BESTOF_STUB=1  skip real motoko runs; expect candidate workspaces pre-seeded at
#                         $STUB_DIR/cand_<i>/$SOLUTION (for testing the select+deploy path)
set -uo pipefail

TASK="${1:-}"
[[ -z "$TASK" && "${MOTOKO_BESTOF_STUB:-0}" != "1" ]] && { echo "Usage: $0 <task>" >&2; exit 2; }
WORKDIR="${WORKDIR:?WORKDIR required}"
MODEL="${MODEL:-ollama/qwen3.6:35b-a3b-mxfp8}"
N="${N:-3}"
CAPS="${CAPS:-IO}"
ENTRY="${ENTRY:-main}"
SOLUTION="${SOLUTION:-solution.ail}"
MOTOKO="${MOTOKO:-/Users/voightkampff/dev/arniwesth/motoko_agent/scripts/run-agent.sh}"

log() { echo "[motoko-bestof] $*" >&2; }

# 1. Generate N candidates, each in an isolated workspace copy.
CANDS=()
ROOT="$(mktemp -d)"
for i in $(seq 1 "$N"); do
  WS="$ROOT/cand_$i"
  if [[ "${MOTOKO_BESTOF_STUB:-0}" == "1" ]]; then
    # Test mode: candidates pre-seeded by the caller under $STUB_DIR.
    mkdir -p "$WS"
    [[ -f "${STUB_DIR:?}/cand_$i/$SOLUTION" ]] && cp "${STUB_DIR}/cand_$i/$SOLUTION" "$WS/$SOLUTION"
  else
    cp -R "$WORKDIR" "$WS"
    log "candidate $i/$N → $WS"
    env WORKDIR="$WS" MODEL="$MODEL" \
        ${MOTOKO_CONFIG:+MOTOKO_CONFIG="$MOTOKO_CONFIG"} \
        ${AILANG_OLLAMA_MAX_TOKENS:+AILANG_OLLAMA_MAX_TOKENS="$AILANG_OLLAMA_MAX_TOKENS"} \
        "$MOTOKO" "$TASK" >/dev/null 2>&1 || log "candidate $i exited non-zero (kept for selection)"
  fi
  [[ -f "$WS/$SOLUTION" ]] && CANDS+=("$WS/$SOLUTION") || log "candidate $i produced no $SOLUTION"
done

if [[ "${#CANDS[@]}" -eq 0 ]]; then
  log "no candidates produced a solution file; nothing to select"
  rm -rf "$ROOT"; exit 1
fi

# 2. Select the candidate that typechecks + runs (exact, reference-free).
log "selecting best of ${#CANDS[@]} via: ailang select-best --caps $CAPS --entry $ENTRY"
WINNER="$(ailang select-best --caps "$CAPS" --entry "$ENTRY" "${CANDS[@]}" 2>/dev/null)"
if [[ -z "$WINNER" || ! -f "$WINNER" ]]; then
  log "select-best returned no winner; falling back to candidate 1"
  WINNER="${CANDS[0]}"
fi

# 3. Deploy the winning solution back into WORKDIR.
cp "$WINNER" "$WORKDIR/$SOLUTION"
log "winner: $WINNER → $WORKDIR/$SOLUTION"
echo "$WORKDIR/$SOLUTION"
rm -rf "$ROOT"
