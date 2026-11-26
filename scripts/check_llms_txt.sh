#!/usr/bin/env bash
# check_llms_txt.sh — verify the concise /llms.txt index (3 copies, <35 KB) and
# docs/static/llms-full.txt (bundling the ACTIVE teaching prompt verbatim).
#
# Fix a failure with: make generate-llms-txt
set -euo pipefail

REPO_ROOT="${LLMS_TXT_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"  # LLMS_TXT_ROOT: self-test only
cd "$REPO_ROOT"

ACTIVE="$(jq -r '.active' prompts/versions.json)"
PROMPT="$(jq -r ".versions.\"$ACTIVE\".file" prompts/versions.json)"
if [ -z "$ACTIVE" ] || [ "$ACTIVE" = "null" ] || [ ! -f "$PROMPT" ]; then
    echo "check-llms-txt: cannot resolve the active prompt from prompts/versions.json" >&2
    exit 1
fi

MAX_INDEX_BYTES=35840
status=0

# 1. Verify all 3 concise /llms.txt routing index copies
for f in llms.txt docs/llms.txt docs/static/llms.txt; do
    if [ ! -f "$f" ]; then
        echo "✗ $f: missing (run: make generate-llms-txt)" >&2
        status=1
        continue
    fi
    size="$(wc -c < "$f" | tr -d ' ')"
    if [ "$size" -ge "$MAX_INDEX_BYTES" ]; then
        echo "✗ $f: index too large (${size} bytes >= ${MAX_INDEX_BYTES} bytes). Keep /llms.txt concise (<35 KB); put full corpus in docs/static/llms-full.txt." >&2
        status=1
    fi
    if ! grep -q "llms-full.txt" "$f"; then
        echo "✗ $f: missing pointer to llms-full.txt (run: make generate-llms-txt)" >&2
        status=1
    fi
    if ! grep -Fq "$ACTIVE" "$f"; then
        echo "✗ $f: does not reference active prompt version $ACTIVE (run: make generate-llms-txt)" >&2
        status=1
    fi
done

# 2. Verify docs/static/llms-full.txt exists and embeds the active prompt verbatim
FULL="docs/static/llms-full.txt"
if [ ! -f "$FULL" ]; then
    echo "✗ $FULL: missing (run: make generate-llms-txt)" >&2
    status=1
else
    if ! python3 -I -c 'import sys; sys.exit(0 if open(sys.argv[1], encoding="utf-8").read().replace("\r\n", "\n") in open(sys.argv[2], encoding="utf-8").read().replace("\r\n", "\n") else 1)' "$PROMPT" "$FULL"; then
        embedded="$(grep -m1 -o '^# AILANG v[0-9][0-9.]*' "$FULL" || echo "no prompt")"
        echo "✗ $FULL: does not embed the active prompt $ACTIVE ($PROMPT); it has: ${embedded}. Run: make generate-llms-txt" >&2
        status=1
    fi
fi

if [ "$status" -eq 0 ]; then
    echo "✓ llms.txt (3 index copies, <35 KB) + docs/static/llms-full.txt embed active prompt $ACTIVE"
fi
exit "$status"
