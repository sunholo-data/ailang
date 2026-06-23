#!/usr/bin/env bash
# run.sh <motoko|pi> [model] — run a harness on the stubbed docx_reimplement task (P0 large-context),
# then grade vs golden. Copies ailang-parse to an isolated workspace, applies the stub, runs the
# harness headless (edits docx_parser.ail in place), and scores with verify.sh.
set -uo pipefail
HARNESS="${1:?usage: run.sh <motoko|pi> [model]}"
MODEL="${2:-ollama/qwen3.6:35b-a3b-mxfp8}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SRC=/Users/voightkampff/dev/sunholo-data/ailang-parse
STAMP=$(date +%Y%m%d-%H%M%S)
# AILANG guidance: the docx run.sh bypasses the eval harness's teaching-prompt injection, so feed the model
# the canonical AILANG teaching prompt (the same source the standard eval uses) as its system prompt. Append
# a generic persistence directive (don't narrate-and-stop). Without this the model writes Haskell-ish AILANG.
TEACH="/tmp/ailang_teaching_${STAMP}.md"
ailang prompt > "$TEACH" 2>/dev/null
printf '\n## Persistence (agent behavior)\nNever end your turn by only describing what you will do next. If the task is not finished, CALL A TOOL this turn (write/edit the file, then run ailang check). Keep working until the code compiles and runs; give a final answer only when the task is complete.\n' >> "$TEACH"
WS="/tmp/docx-${HARNESS}-${STAMP}"
SESSION="session_docx_${HARNESS}_${STAMP}"
MOTOKO_REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/arniwesth/motoko_agent}"

echo "[run.sh] harness=$HARNESS model=$MODEL ws=$WS session=$SESSION"
cp -R "$SRC" "$WS"
cp "$HERE/stub_docx_parser.ail" "$WS/docparse/services/docx_parser.ail"

read -r -d '' TASK <<'EOF'
The file docparse/services/docx_parser.ail has been stubbed: all 13 exported functions currently return empty values. Reimplement it FULLY so the document parser correctly converts DOCX XML into the Block ADT.

Study these dependencies (large context — read them before writing):
- docparse/types/document  : the Block ADT + constructors (TextBlock, HeadingBlock, TableBlock, ListBlock, ImageBlock, SectionBlock, ChangeBlock, TableCell, DocMetadata, simpleCell, mergedCell, spanCell, mkImage, mkTable).
- docparse/services/zip_extract : readDocxContent, readCoreProperties, findMediaEntries, readEmbeddedImage, findHeaderEntries, findFooterEntries, findFootnoteEntries, findEndnoteEntries, readComments.
- std/xml : parse, findAll, findFirst, getText, getAttr, getChildren, getTag.

Handle every case the real parser does: paragraphs (style/text/runs), headings, tables with merged cells (gridSpan / vMerge), lists, images, track changes (insert/delete/move), headers, footers, footnotes, endnotes, comments, nested SDTs.

Reimplement ALL exports: parseDocx, parseDocxMetadata, parseDocxImages, parseDocxHeaders, parseDocxFooters, parseDocxFootnotes, parseDocxEndnotes, parseDocxComments, parseSectionXml, hasContent, extractBodyBlocks, extractMetadata, getMetaField.

Test your work on the fixtures: for each file in data/test_files/*.docx run
  ailang run --entry main --caps IO,FS,Env docparse/main.ail data/test_files/<name>.docx
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
        WORKDIR="$WS" MODEL="$MODEL" MOTOKO_CONFIG=ollama MOTOKO_HEADLESS=1 SYSTEM_MD="$TEACH" \
        ENV_PORT=8080 AILANG_OLLAMA_MAX_TOKENS=32768 AILANG_OLLAMA_HTTP_TIMEOUT_SEC="${AILANG_OLLAMA_HTTP_TIMEOUT_SEC:-1800}" MOTOKO_SESSION_ID="$SESSION" \
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
