#!/usr/bin/env bash
# run.sh [model] — MID-SIZE compaction-probe task through the motoko harness: implement
# countList/avg/maxList/sumSquares in nums.ail so `main` prints "30 50 5500". ~10-15 turns —
# big enough to cross the compaction threshold (to exercise compaction_ai + the structural
# fallback), small enough to iterate in minutes. Grade = exact output match.
#   MOTOKO_REPO=/path/to/motoko bash eval_projects/stats_probe/run.sh [model]
set -uo pipefail
MODEL="${1:-ollama/qwen3.6:35b-a3b-mxfp8}"
HERE="$(cd "$(dirname "$0")" && pwd)"
STAMP=$(date +%Y%m%d-%H%M%S)
WS="/tmp/stats_probe-motoko-${STAMP}"
SESSION="session_statsprobe_motoko_${STAMP}"
MOTOKO_REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/mk-ast}"

AGENTMD="/tmp/ailang_agent_prompt_${STAMP}.md"
cat > "$AGENTMD" <<'AGENTPROMPT'
You are an autonomous coding agent working in an AILANG project. AILANG is a pure functional language with Hindley-Milner type inference and algebraic effects — it is NOT Python and NOT Haskell.

Work incrementally: implement one function, run `ailang check` to catch errors early, fix them, then the next. Persistence: call a tool every turn; keep going until the project type-checks and `main` prints the required output.

The complete, CANONICAL AILANG language reference (the single source of truth — never guess AILANG syntax) follows:

=== AILANG LANGUAGE REFERENCE (canonical) ===
AGENTPROMPT
ailang prompt --source=embedded >> "$AGENTMD" 2>/dev/null

cp -R "$HERE" "$WS"
rm -f "$WS"/run.sh "$WS"/expected.txt 2>/dev/null
ailang prompt --source=embedded > "$WS/syntax_reference.md" 2>/dev/null

read -r -d '' TASK <<'EOF'
nums.ail defines `sumList` (the reference pattern) but is MISSING four functions. Implement and EXPORT all four in nums.ail, each by recursion in the same shape as sumList:
  - countList(xs: [int]) -> int   : the number of elements
  - avg(xs: [int]) -> int         : integer mean = sumList(xs) / countList(xs)
  - maxList(xs: [int]) -> int     : the largest element (assume non-empty)
  - sumSquares(xs: [int]) -> int  : sum of x*x over the list

main.ail uses avg, maxList and sumSquares. When done, running this must print exactly:
  30 50 5500

Verify with: ailang run --entry main --caps IO main.ail . The full reference is in syntax_reference.md.
EOF

if [ -f "$HERE/../../tools/launchd/rig-lock.sh" ]; then
  # shellcheck source=/dev/null
  source "$HERE/../../tools/launchd/rig-lock.sh"
  rig_lock_acquire wait
fi
pkill -9 -f 'bun.*src/tui' 2>/dev/null; sleep 1

echo "[run.sh] model=$MODEL ws=$WS session=$SESSION repo=$MOTOKO_REPO"
env OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-dummy}" \
    WORKDIR="$WS" MODEL="$MODEL" MOTOKO_CONFIG=ollama MOTOKO_HEADLESS=1 SYSTEM_MD="$AGENTMD" \
    AILANG_RELAX_MODULES=1 AI_MAX_STEPS="${AI_MAX_STEPS:-30}" \
    ENV_PORT=0 AILANG_OLLAMA_MAX_TOKENS=65536 AILANG_OLLAMA_HTTP_TIMEOUT_SEC="${AILANG_OLLAMA_HTTP_TIMEOUT_SEC:-900}" MOTOKO_SESSION_ID="$SESSION" \
    "$MOTOKO_REPO/scripts/run-agent.sh" --headless "$TASK" > "/tmp/${SESSION}.out" 2>&1

echo "=== GRADE ==="
GOT=$(cd "$WS" && AILANG_RELAX_MODULES=1 ailang run --entry main --caps IO main.ail 2>/dev/null | grep -vE '^(→|✓|Warning|WARNING|Error| )' | tr -d '\n' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//')
WANT="$(tr -d '\n' < "$HERE/expected.txt")"
if [ "$GOT" = "$WANT" ]; then echo "Result: PASS — main printed '$GOT'"; else echo "Result: FAIL — got '${GOT}', want '${WANT}'"; fi
echo "[run.sh] ws=$WS"
echo "[run.sh] session_jsonl=$MOTOKO_REPO/.motoko/logfile/${SESSION}.jsonl"
echo "[run.sh] run_log=/tmp/${SESSION}.out"
