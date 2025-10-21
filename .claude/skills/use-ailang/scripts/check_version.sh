#!/usr/bin/env bash
# Check current active AILANG prompt version

set -euo pipefail

PROMPTS_DIR="${PROMPTS_DIR:-prompts}"
VERSIONS_FILE="$PROMPTS_DIR/versions.json"

if [[ ! -f "$VERSIONS_FILE" ]]; then
    echo "Error: versions.json not found at $VERSIONS_FILE" >&2
    exit 1
fi

# Get active version
ACTIVE_VERSION=$(jq -r '.active' "$VERSIONS_FILE")

if [[ -z "$ACTIVE_VERSION" || "$ACTIVE_VERSION" == "null" ]]; then
    echo "Error: No active version found in versions.json" >&2
    exit 1
fi

# Get prompt file path
PROMPT_FILE=$(jq -r ".versions[\"$ACTIVE_VERSION\"].file" "$VERSIONS_FILE")

if [[ -z "$PROMPT_FILE" || "$PROMPT_FILE" == "null" ]]; then
    echo "Error: No file path found for version $ACTIVE_VERSION" >&2
    exit 1
fi

# Output results
echo "Active AILANG version: $ACTIVE_VERSION"
echo "Prompt file: $PROMPT_FILE"

# Verify file exists
if [[ -f "$PROMPT_FILE" ]]; then
    echo "Status: File exists ✓"
    exit 0
else
    echo "Status: File not found ✗" >&2
    exit 1
fi
