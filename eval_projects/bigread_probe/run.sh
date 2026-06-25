#!/usr/bin/env bash
# run.sh [model] — COMPACTION-TRIGGER probe: the model scans a large file (bigdata.txt, ~33K tokens)
# in chunks to find a magic number, then implements answer() returning it. The chunked reads build a
# large conversation that crosses the compaction threshold -> exercises compaction_ai + the structural
# fallback, AND tests reliability (does the summary preserve the magic number the model needs?).
# Grade = main prints the magic number (42).
#   MOTOKO_REPO=/path/to/motoko bash eval_projects/bigread_probe/run.sh [model]
set -uo pipefail
MODEL="${1:-ollama/qwen3.6:35b-a3b-mxfp8}"
HERE="$(cd "$(dirname "$0")" && pwd)"
STAMP=$(date +%Y%m%d-%H%M%S)
WS="/tmp/bigread_probe-motoko-${STAMP}"
SESSION="session_bigread_motoko_${STAMP}"
MOTOKO_REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/mk-ast}"

AGENTMD="/tmp/ailang_agent_prompt_${STAMP}.md"
cat > "$AGENTMD" <<'AGENTPROMPT'
You are an autonomous coding agent working in an AILANG project. AILANG is a pure functional language with Hindley-Milner type inference and algebraic effects — it is NOT Python and NOT Haskell.

Work step by step. Persistence: call a tool every turn; keep going until the project type-checks and `main` prints the required output.

The complete, CANONICAL AILANG language reference follows:

=== AILANG LANGUAGE REFERENCE (canonical) ===
AGENTPROMPT
ailang prompt --source=embedded >> "$AGENTMD" 2>/dev/null

cp -R "$HERE" "$WS"
rm -f "$WS"/run.sh "$WS"/expected.txt 2>/dev/null
ailang prompt --source=embedded > "$WS/syntax_reference.md" 2>/dev/null

read -r -d '' TASK <<'EOF'
bigdata.txt has ~3000 lines. ONE line states "the magic number is N" for some integer N. Find it, then implement the solution.

Steps:
1. Scan bigdata.txt to find the line with the magic number. Read it in CHUNKS — use ReadFile with start/end line ranges (about 300 lines at a time): lines 1-300, then 301-600, and so on, until you find the "magic number is N" line. (BashExec `grep` is also fine.)
2. Once you know N, implement and EXPORT `answer() -> int` in solution.ail returning that exact integer N.
3. main.ail prints answer(). When done, running `ailang run --entry main --caps IO main.ail` must print N.

Verify with: ailang run --entry main --caps IO main.ail
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
    AILANG_RELAX_MODULES=1 AI_MAX_STEPS="${AI_MAX_STEPS:-40}" \
    ENV_PORT=0 AILANG_OLLAMA_MAX_TOKENS=65536 AILANG_OLLAMA_HTTP_TIMEOUT_SEC="${AILANG_OLLAMA_HTTP_TIMEOUT_SEC:-1200}" MOTOKO_SESSION_ID="$SESSION" \
    "$MOTOKO_REPO/scripts/run-agent.sh" --headless "$TASK" > "/tmp/${SESSION}.out" 2>&1

echo "=== GRADE ==="
GOT=$(cd "$WS" && AILANG_RELAX_MODULES=1 ailang run --entry main --caps IO main.ail 2>/dev/null | grep -vE '^(→|✓|Warning|WARNING|Error| )' | tr -d '[:space:]')
WANT="$(tr -d '[:space:]' < "$HERE/expected.txt")"
if [ "$GOT" = "$WANT" ]; then echo "Result: PASS — main printed '$GOT' (found+preserved the magic number)"; else echo "Result: FAIL — got '${GOT}', want '${WANT}'"; fi
echo "[run.sh] ws=$WS"
echo "[run.sh] session_jsonl=$MOTOKO_REPO/.motoko/logfile/${SESSION}.jsonl"
echo "[run.sh] run_log=/tmp/${SESSION}.out"
