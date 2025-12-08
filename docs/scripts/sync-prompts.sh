#!/bin/bash
# Sync teaching prompts from root /prompts/ to /docs/docs/prompts/
# Ensures website always shows latest prompt versions
# Reads from versions.json for single source of truth (no hardcoded versions!)

set -e

# Change to repo root
cd "$(dirname "$0")/../.."

PROMPTS_SRC="prompts"
PROMPTS_DEST="docs/docs/prompts"
VERSIONS_FILE="prompts/versions.json"

echo "🔄 Syncing prompts from $PROMPTS_SRC to $PROMPTS_DEST..."

# Get active prompt version from versions.json
ACTIVE_PROMPT=$(jq -r '.active' "$VERSIONS_FILE" 2>/dev/null)
if [ -z "$ACTIVE_PROMPT" ] || [ "$ACTIVE_PROMPT" = "null" ]; then
    echo "❌ Error: Could not read active version from $VERSIONS_FILE"
    exit 1
fi

# Get production/latest tagged versions (for website display)
# Sort by version number, take top 5, deduplicate
PRODUCTION_VERSIONS=$(jq -r '.versions | to_entries[] | select(.value.tags[]? == "production" or .value.tags[]? == "latest") | .key' "$VERSIONS_FILE" | sort -rV | uniq | head -5)

# Copy active prompt
if [ -f "$PROMPTS_SRC/${ACTIVE_PROMPT}.md" ]; then
    cp "$PROMPTS_SRC/${ACTIVE_PROMPT}.md" "$PROMPTS_DEST/${ACTIVE_PROMPT}.md"
    echo "✅ Copied active prompt: ${ACTIVE_PROMPT}.md"
else
    echo "⚠️  Warning: Active prompt ${ACTIVE_PROMPT}.md not found"
fi

# Copy python control prompt (used for benchmarks)
if [ -f "$PROMPTS_SRC/python.md" ]; then
    cp "$PROMPTS_SRC/python.md" "$PROMPTS_DEST/python.md"
    echo "✅ Copied python.md"
fi

# Copy production/latest versions from versions.json (NO HARDCODED VERSIONS!)
count=0
for version in $PRODUCTION_VERSIONS; do
    if [ -f "$PROMPTS_SRC/${version}.md" ]; then
        cp "$PROMPTS_SRC/${version}.md" "$PROMPTS_DEST/${version}.md"
        echo "✅ Copied ${version}.md (production/latest tag)"
        count=$((count + 1))
    fi
done

echo ""
echo "📝 Summary:"
echo "   Active prompt: ${ACTIVE_PROMPT}"
echo "   Production versions synced: ${count}"
echo "   Destination: $PROMPTS_DEST"
echo "   Source of truth: $VERSIONS_FILE"
echo ""
echo "✅ Prompt sync complete!"
echo ""
echo "💡 All versions available via: ailang prompt --list"
