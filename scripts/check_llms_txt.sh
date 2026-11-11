#!/usr/bin/env bash
# check_llms_txt.sh — the committed llms.txt files embed the ACTIVE teaching prompt.
#
# llms.txt bundles the docs and the active prompt for LLM consumption, and the docs
# deploy copies the committed root llms.txt to https://ailang.sunholo.com/llms.txt.
# It is regenerated only by hand (tools/generate-llms-txt.sh), so it drifted: on
# 2026-11-11 all three copies still embedded prompt v0.16.12 (old stdlib names and a
# sample that did not type-check) while v0.16.13 was active. The generator stamps a
# timestamp, so a byte comparison of the whole file would always differ; this checks
# the part that matters: the active prompt file appears verbatim in each copy.
#
# Fix a failure with: make generate-llms-txt
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

ACTIVE="$(jq -r '.active' prompts/versions.json)"
PROMPT="$(jq -r ".versions.\"$ACTIVE\".file" prompts/versions.json)"
if [ -z "$ACTIVE" ] || [ "$ACTIVE" = "null" ] || [ ! -f "$PROMPT" ]; then
    echo "check-llms-txt: cannot resolve the active prompt from prompts/versions.json" >&2
    exit 1
fi

status=0
for f in llms.txt docs/llms.txt docs/static/llms.txt; do
    if [ ! -f "$f" ]; then
        echo "✗ $f: missing (run: make generate-llms-txt)" >&2
        status=1
        continue
    fi
    if ! python3 -I -c 'import sys; sys.exit(0 if open(sys.argv[1], encoding="utf-8").read().replace("\r\n", "\n") in open(sys.argv[2], encoding="utf-8").read().replace("\r\n", "\n") else 1)' "$PROMPT" "$f"; then
        embedded="$(grep -m1 -o '^# AILANG v[0-9][0-9.]*' "$f" || echo "no prompt")"
        echo "✗ $f: does not embed the active prompt $ACTIVE ($PROMPT); it has: ${embedded}. Run: make generate-llms-txt" >&2
        status=1
    fi
done
if [ "$status" -eq 0 ]; then
    echo "✓ llms.txt (3 copies) embeds the active prompt $ACTIVE"
fi
exit "$status"
