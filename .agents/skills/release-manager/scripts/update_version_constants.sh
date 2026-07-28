#!/bin/bash
#
# Update docs/src/constants/version.js with the new release version.
# Called automatically during release process.
#
# Usage: update_version_constants.sh <version>
# Example: update_version_constants.sh 0.5.7
#

set -e

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.5.7"
    exit 1
fi

# Normalize version (add v prefix if missing)
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

VERSION_FILE="docs/src/constants/version.js"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: $VERSION_FILE not found"
    exit 1
fi

# Get current values
CURRENT_STABLE=$(grep "STABLE_RELEASE" "$VERSION_FILE" | grep -oE "v[0-9]+\.[0-9]+\.[0-9]+" || echo "unknown")
CURRENT_PROMPT=$(grep "ACTIVE_PROMPT" "$VERSION_FILE" | grep -oE "v[0-9]+\.[0-9]+\.[0-9]+" || echo "unknown")

# Find latest prompt version
LATEST_PROMPT=$(ls prompts/*.md 2>/dev/null | grep -oE "v[0-9]+\.[0-9]+\.[0-9]+" | sort -V | tail -1 || echo "$CURRENT_PROMPT")

echo "Updating $VERSION_FILE..."
echo "  STABLE_RELEASE: $CURRENT_STABLE → $VERSION"
echo "  ACTIVE_PROMPT: $CURRENT_PROMPT → $LATEST_PROMPT"

# Update the file
cat > "$VERSION_FILE" << EOF
export const STABLE_RELEASE = '$VERSION';
export const ACTIVE_PROMPT = '$LATEST_PROMPT';
EOF

echo "✓ Updated $VERSION_FILE"
