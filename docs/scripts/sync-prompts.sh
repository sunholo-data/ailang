#!/bin/bash
# Sync the active teaching prompt from root /prompts/ to /docs/docs/prompts/
# Publishes only current.md (and python.md for benchmark control) so historical
# v*.md snapshots do not pollute Docusaurus site search.
# Historical versions remain in /prompts/ and prompts/versions.json via `ailang prompt --list`.

set -e

cd "$(dirname "$0")/../.."

PROMPTS_SRC="prompts"
PROMPTS_DEST="docs/docs/prompts"
VERSIONS_FILE="prompts/versions.json"

echo "🔄 Syncing active prompt from $PROMPTS_SRC to $PROMPTS_DEST..."

# Sync active prompt to docs/docs/prompts/current.md
bash docs/scripts/sync-active-prompt.sh

# Ensure no stale historical v*.md snapshots remain in docs/docs/prompts/
rm -f "$PROMPTS_DEST"/v*.md

# Keep python control prompt synced (unlisted benchmark reference)
if [ -f "$PROMPTS_SRC/python.md" ]; then
    cp "$PROMPTS_SRC/python.md" "$PROMPTS_DEST/python.md"
    echo "✅ Copied python.md"
fi

ACTIVE_PROMPT=$(jq -r '.active' "$VERSIONS_FILE" 2>/dev/null)
echo "✅ Prompt sync complete (active: ${ACTIVE_PROMPT}; historical versions via 'ailang prompt --list')"
