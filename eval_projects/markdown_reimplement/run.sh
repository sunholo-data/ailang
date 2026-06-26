#!/usr/bin/env bash
# run.sh <motoko|pi> [model] — run a harness on the stubbed markdown_reimplement task (P0 large-context),
# then grade vs golden. Copies ailang-parse to an isolated workspace, applies the stub, runs the
# harness headless (edits markdown_parser.ail in place), and scores with verify.sh.
set -uo pipefail
HARNESS="${1:?usage: run.sh <motoko|pi> [model]}"
MODEL="${2:-ollama/qwen3.6:35b-a3b-mxfp8}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SRC=/Users/voightkampff/dev/sunholo-data/ailang-parse
STAMP=$(date +%Y%m%d-%H%M%S)
# AILANG guidance — REPLICATE THE AGENT-EVAL SETUP (motoko runs via agent eval, NOT standard eval): the agent
# system prompt is a LEAN coding prompt, and the full AILANG teaching prompt is written to syntax_reference.md
# IN THE WORKSPACE for the model to read on demand (see internal/eval_harness/agent_prompt.go). run.sh had
# bypassed this, leaving the model with no AILANG syntax -> \x. drift.
AGENTMD="/tmp/ailang_agent_prompt_${STAMP}.md"
cat > "$AGENTMD" <<'AGENTPROMPT'
You are an autonomous coding agent working in an AILANG project. AILANG is a pure functional language with Hindley-Milner type inference and algebraic effects — it is NOT Python and NOT Haskell.

Work incrementally: implement a few functions, run `ailang check` to catch errors early, fix them, then add more — do not write the whole file in one shot. Persistence: call a tool every turn to make progress; keep going until the project type-checks and the task is fully complete.

The complete, CANONICAL AILANG language reference (the single source of truth — never guess AILANG syntax from other languages) follows:

=== AILANG LANGUAGE REFERENCE (canonical) ===
AGENTPROMPT
# Append the CANONICAL teaching prompt (active versioned, v0.16.2) from the binary EMBEDDED copy — NOT
# `ailang prompt` (auto), which fetches a STALE v0.16.0 from MCP. Served ONCE in the system prompt (motoko
# never elides the system message; the provider prompt-caches it) — the eval-proven best delivery.
ailang prompt --source=embedded >> "$AGENTMD" 2>/dev/null
WS="/tmp/markdown-${HARNESS}-${STAMP}"
SESSION="session_docx_${HARNESS}_${STAMP}"
MOTOKO_REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/arniwesth/motoko_agent}"

echo "[run.sh] harness=$HARNESS model=$MODEL ws=$WS session=$SESSION"
cp -R "$SRC" "$WS"
cp "$HERE/stub_markdown_parser.ail" "$WS/docparse/services/markdown_parser.ail"
ailang prompt --source=embedded > "$WS/syntax_reference.md" 2>/dev/null

read -r -d '' TASK <<'EOF'
The exported function `parseMarkdown` in docparse/services/markdown_parser.ail has been STUBBED (it currently returns an empty list). Reimplement `parseMarkdown(content: string) -> [Block]` so it correctly converts Markdown text into the Block ADT.

The helper functions in the SAME file are already implemented and available — read and use them:
- emptyParseState() : the initial MdParseState
- mdProcessLine(state, line) : fold one line into the state (handles headings, lists, tables, code, paragraphs)
- flushState(state) : flush accumulated state into the final [Block] list (note: it returns them in reverse)

Study these dependencies (read them before writing):
- docparse/types/document : the Block ADT + constructors (TextBlock, HeadingBlock, TableBlock, ListBlock, CodeBlock, etc.).
- the rest of markdown_parser.ail : the MdParseState type and the flush/utility helpers.

The reference shape: split the content into lines, fold mdProcessLine over them starting from emptyParseState(), then flush. Match the real parser's output exactly.

Test on the fixtures: for each file in data/test_files/*.md run
  ailang run --entry main --caps IO,FS,Env docparse/main.ail data/test_files/<name>.md
and make the output reproduce the document's real content. The whole package must still typecheck:
  ailang check --package .
EOF

# Coordinate with the shared single-GPU rig (nightly / os-rotation-filler / eval-suite). Without
# this, a concurrent job's process-hygiene `pkill -f 'bun.*src/tui'` kills this run mid-task
# (observed: docx run 3 died at step 7 when an eval-suite started). rig_lock_acquire waits for
# the rig; the lock auto-releases on EXIT.
if [ -f "$HERE/../../tools/launchd/rig-lock.sh" ]; then
  # shellcheck source=/dev/null
  source "$HERE/../../tools/launchd/rig-lock.sh"
  echo "[run.sh] acquiring rig lock (waits for any nightly/filler/eval-suite to finish)…"
  rig_lock_acquire wait
  echo "[run.sh] rig lock acquired"
fi

# port / process hygiene (fixed ENV_PORT=8080 → serial only)
pkill -9 -f 'bun.*src/tui' 2>/dev/null; sleep 1

case "$HARNESS" in
  motoko)
    env OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-dummy}" \
        WORKDIR="$WS" MODEL="$MODEL" MOTOKO_CONFIG=ollama MOTOKO_HEADLESS=1 SYSTEM_MD="$AGENTMD" \
        MOTOKO_AST_AUTOREAD=1 MOTOKO_AST_READ_FULL="markdown_parser.ail:main.ail" \
        ENV_PORT=8080 AILANG_OLLAMA_MAX_TOKENS=65536 AILANG_OLLAMA_HTTP_TIMEOUT_SEC="${AILANG_OLLAMA_HTTP_TIMEOUT_SEC:-1800}" MOTOKO_SESSION_ID="$SESSION" \
        "$MOTOKO_REPO/scripts/run-agent.sh" --headless "$TASK" > "/tmp/${SESSION}.out" 2>&1
    ;;
  pi)
    echo "[run.sh] pi harness invocation not wired yet — TODO" ; exit 2 ;;
  *) echo "unknown harness: $HARNESS"; exit 2 ;;
esac

echo "=== GRADE ($HARNESS) ==="
bash "$HERE/verify.sh" "$WS" 2>&1 | tail -3
echo "[run.sh] ws=$WS"
echo "[run.sh] session_jsonl=$MOTOKO_REPO/.motoko/logfile/${SESSION}.jsonl"
echo "[run.sh] run_log=/tmp/${SESSION}.out"
