#!/bin/bash
# Sync version constants from prompts/versions.json to docs
# Run: make sync-versions

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSIONS_JSON="$REPO_ROOT/prompts/versions.json"
VERSION_JS="$REPO_ROOT/docs/src/constants/version.js"

# Extract active version from versions.json
ACTIVE_PROMPT=$(jq -r '.active' "$VERSIONS_JSON")

if [ -z "$ACTIVE_PROMPT" ] || [ "$ACTIVE_PROMPT" = "null" ]; then
    echo "Error: Could not read 'active' from $VERSIONS_JSON"
    exit 1
fi

# For now, STABLE_RELEASE matches ACTIVE_PROMPT
# In future, these could be different (e.g., prompt ahead of release)
STABLE_RELEASE="$ACTIVE_PROMPT"

# Generate version.js
cat > "$VERSION_JS" << EOF
// Auto-generated from prompts/versions.json - do not edit manually
// Run: make sync-versions
export const STABLE_RELEASE = '$STABLE_RELEASE';
export const ACTIVE_PROMPT = '$ACTIVE_PROMPT';
EOF

echo "Updated $VERSION_JS:"
echo "  STABLE_RELEASE = $STABLE_RELEASE"
echo "  ACTIVE_PROMPT = $ACTIVE_PROMPT"
