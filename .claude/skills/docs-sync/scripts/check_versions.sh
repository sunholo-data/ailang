#!/bin/bash
# check_versions.sh - Verify version constants match actual releases
# Usage: ./check_versions.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "=== Version Consistency Check ==="
echo ""

# Get actual version from git
ACTUAL_VERSION=$(cd "$PROJECT_ROOT" && git describe --tags --abbrev=0 2>/dev/null || echo "unknown")
echo "Git tag (actual version): $ACTUAL_VERSION"

# Get version from website constants
VERSION_FILE="$PROJECT_ROOT/docs/src/constants/version.js"
if [[ -f "$VERSION_FILE" ]]; then
    WEBSITE_VERSION=$(grep "STABLE_RELEASE" "$VERSION_FILE" | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" || echo "unknown")
    PROMPT_VERSION=$(grep "ACTIVE_PROMPT" "$VERSION_FILE" | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" || echo "unknown")
    echo "Website STABLE_RELEASE: $WEBSITE_VERSION"
    echo "Website ACTIVE_PROMPT: $PROMPT_VERSION"
else
    echo "ERROR: Version file not found: $VERSION_FILE"
    exit 1
fi

# Get latest prompt version
PROMPTS_DIR="$PROJECT_ROOT/prompts"
if [[ -d "$PROMPTS_DIR" ]]; then
    LATEST_PROMPT=$(ls -1 "$PROMPTS_DIR"/v*.md 2>/dev/null | sort -V | tail -1 | xargs basename 2>/dev/null | sed 's/.md//' || echo "unknown")
    echo "Latest prompt file: $LATEST_PROMPT"
fi

echo ""
echo "=== Version Checks ==="

# Check 1: Website version vs git tag
if [[ "$WEBSITE_VERSION" == "$ACTUAL_VERSION" ]]; then
    echo "  [OK] STABLE_RELEASE matches git tag"
else
    echo "  [MISMATCH] STABLE_RELEASE ($WEBSITE_VERSION) != git tag ($ACTUAL_VERSION)"
    echo "  ACTION: Update docs/src/constants/version.js"
fi

# Check 2: Active prompt vs latest prompt file
if [[ "$PROMPT_VERSION" == "$LATEST_PROMPT" ]]; then
    echo "  [OK] ACTIVE_PROMPT matches latest prompt file"
else
    echo "  [MISMATCH] ACTIVE_PROMPT ($PROMPT_VERSION) != latest prompt ($LATEST_PROMPT)"
    echo "  ACTION: Update docs/src/constants/version.js"
fi

# Check 3: References in intro.mdx
INTRO_FILE="$PROJECT_ROOT/docs/docs/intro.mdx"
if [[ -f "$INTRO_FILE" ]]; then
    INTRO_PROMPT_REF=$(grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" "$INTRO_FILE" | head -1 || echo "none")
    if [[ "$INTRO_PROMPT_REF" != "$LATEST_PROMPT" && "$INTRO_PROMPT_REF" != "none" ]]; then
        echo "  [STALE] intro.mdx references $INTRO_PROMPT_REF, latest is $LATEST_PROMPT"
    fi
fi

echo ""
echo "=== Quick Fix ==="
echo "To update version.js to current:"
echo ""
echo "cat > docs/src/constants/version.js << 'EOF'"
echo "export const STABLE_RELEASE = '$ACTUAL_VERSION';"
echo "export const ACTIVE_PROMPT = '$LATEST_PROMPT';"
echo "EOF"
echo ""
