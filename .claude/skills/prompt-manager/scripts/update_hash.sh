#!/usr/bin/env bash
# Update hash for a prompt version in versions.json
#
# Usage:
#   update_hash.sh <version>
#
# Example:
#   update_hash.sh v0.3.17

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <version>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 v0.3.17" >&2
    exit 1
fi

VERSION="$1"

# Get file from versions.json
PROMPT_FILE=$(jq -r ".versions[\"$VERSION\"].file" prompts/versions.json)

if [ "$PROMPT_FILE" == "null" ]; then
    echo "Error: Version $VERSION not found in prompts/versions.json" >&2
    exit 1
fi

if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: Prompt file not found: $PROMPT_FILE" >&2
    exit 1
fi

# Compute new hash
NEW_HASH=$(shasum -a 256 "$PROMPT_FILE" | awk '{print $1}')
OLD_HASH=$(jq -r ".versions[\"$VERSION\"].hash" prompts/versions.json)

echo "Updating hash for $VERSION"
echo "  File: $PROMPT_FILE"
echo "  Old hash: $OLD_HASH"
echo "  New hash: $NEW_HASH"

if [ "$OLD_HASH" == "$NEW_HASH" ]; then
    echo "✓ Hash unchanged (file not modified)"
    exit 0
fi

# Update hash in versions.json
TMP_FILE=$(mktemp)
jq --arg version "$VERSION" \
   --arg hash "$NEW_HASH" \
   '.versions[$version].hash = $hash' prompts/versions.json > "$TMP_FILE"

mv "$TMP_FILE" prompts/versions.json
echo "✓ Updated hash in prompts/versions.json"
