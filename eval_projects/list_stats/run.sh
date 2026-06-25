#!/usr/bin/env bash
# run.sh [model] — run the motoko harness on the SMALL list_stats task (implement `avg`),
# then grade: running `main` must print 30. A FAST-ITERATION baseline (seconds, tiny context)
# for testing the harness + extensions — the docx task is too big to iterate on.
#   MOTOKO_REPO=/path/to/motoko bash eval_projects/list_stats/run.sh [model]
set -uo pipefail
MODEL="${1:-ollama/qwen3.6:35b-a3b-mxfp8}"
HERE="$(cd "$(dirname "$0")" && pwd)"
STAMP=$(date +%Y%m%d-%H%M%S)
WS="/tmp/list_stats-motoko-${STAMP}"
SESSION="session_liststats_motoko_${STAMP}"
MOTOKO_REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/mk-ast}"

# Agent system prompt + the CANONICAL AILANG syntax reference (same delivery as docx run.sh).
AGENTMD="/tmp/ailang_agent_prompt_${STAMP}.md"
cat > "$AGENTMD" <<'AGENTPROMPT'
You are an autonomous coding agent working in an AILANG project. AILANG is a pure functional language with Hindley-Milner type inference and algebraic effects — it is NOT Python and NOT Haskell.

Work incrementally: implement the function, run `ailang check` to catch errors early, fix them. Persistence: call a tool every turn to make progress; keep going until the project type-checks and `main` prints the required output.

The complete, CANONICAL AILANG language reference (the single source of truth — never guess AILANG syntax from other languages) follows:

=== AILANG LANGUAGE REFERENCE (canonical) ===
AGENTPROMPT
ailang prompt --source=embedded >> "$AGENTMD" 2>/dev/null

# Workspace = copy of the project; strip the eval scaffolding so the agent only sees code.
cp -R "$HERE" "$WS"
rm -f "$WS"/run.sh "$WS"/TASK.md "$WS"/expected.txt 2>/dev/null
ailang prompt --source=embedded > "$WS/syntax_reference.md" 2>/dev/null

read -r -d '' TASK <<'EOF'
nums.ail defines `sumList` but is MISSING `avg`. Implement `avg(xs: [int]) -> int` in nums.ail — the integer mean (sum divided by the number of elements) — and export it so main.ail can use it. AILANG has no for/while loops: count the elements with recursion (the same shape as sumList).

When done, running this must print exactly 30:
  ailang run --entry main --caps IO main.ail

Use ReadFile/EditFile/WriteFile and run `ailang check` to verify as you go. The reference is in syntax_reference.md.
EOF

# Coordinate with the shared single-GPU rig (nightly / filler / eval-suite).
if [ -f "$HERE/../../tools/launchd/rig-lock.sh" ]; then
  # shellcheck source=/dev/null
  source "$HERE/../../tools/launchd/rig-lock.sh"
  rig_lock_acquire wait
fi
pkill -9 -f 'bun.*src/tui' 2>/dev/null; sleep 1

echo "[run.sh] model=$MODEL ws=$WS session=$SESSION repo=$MOTOKO_REPO"
env OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-dummy}" \
    WORKDIR="$WS" MODEL="$MODEL" MOTOKO_CONFIG=ollama MOTOKO_HEADLESS=1 SYSTEM_MD="$AGENTMD" \
    AILANG_RELAX_MODULES=1 AI_MAX_STEPS="${AI_MAX_STEPS:-20}" \
    ENV_PORT=0 AILANG_OLLAMA_MAX_TOKENS=65536 AILANG_OLLAMA_HTTP_TIMEOUT_SEC="${AILANG_OLLAMA_HTTP_TIMEOUT_SEC:-600}" MOTOKO_SESSION_ID="$SESSION" \
    "$MOTOKO_REPO/scripts/run-agent.sh" --headless "$TASK" > "/tmp/${SESSION}.out" 2>&1

echo "=== GRADE ==="
# filter ailang's banner lines (→ Type checking / ✓ Running / Warning) so we grade only program output
GOT=$(cd "$WS" && AILANG_RELAX_MODULES=1 ailang run --entry main --caps IO main.ail 2>/dev/null | grep -vE '^(→|✓|Warning|WARNING|Error| )' | tr -d '[:space:]')
if [ "$GOT" = "30" ]; then echo "Result: PASS — main printed 30"; else echo "Result: FAIL — got '${GOT}', want 30"; fi
echo "[run.sh] ws=$WS"
echo "[run.sh] session_jsonl=$MOTOKO_REPO/.motoko/logfile/${SESSION}.jsonl"
echo "[run.sh] run_log=/tmp/${SESSION}.out"
