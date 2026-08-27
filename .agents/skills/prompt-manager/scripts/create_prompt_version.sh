#!/usr/bin/env bash
# Create a new AILANG teaching prompt version
#
# Usage:
#   create_prompt_version.sh <new_version> <base_version> "<description>"
#
# Example:
#   create_prompt_version.sh v0.3.17 v0.3.16 "Fix httpRequest documentation"

set -euo pipefail

if [ $# -lt 3 ]; then
    echo "Usage: $0 <new_version> <base_version> <description>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 v0.3.17 v0.3.16 \"Fix httpRequest documentation\"" >&2
    exit 1
fi

NEW_VERSION="$1"
BASE_VERSION="$2"
DESCRIPTION="$3"

# Validate versions start with v
if [[ ! "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: New version must be in format vX.Y.Z (e.g., v0.3.17)" >&2
    exit 1
fi

if [[ ! "$BASE_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Base version must be in format vX.Y.Z (e.g., v0.3.16)" >&2
    exit 1
fi

# Check base version exists
BASE_FILE=$(jq -r ".versions[\"$BASE_VERSION\"].file" prompts/versions.json)
if [ "$BASE_FILE" == "null" ]; then
    echo "Error: Base version $BASE_VERSION not found in prompts/versions.json" >&2
    exit 1
fi

if [ ! -f "$BASE_FILE" ]; then
    echo "Error: Base prompt file not found: $BASE_FILE" >&2
    exit 1
fi

NEW_FILE="prompts/${NEW_VERSION}.md"

# Check new version doesn't already exist
if [ -f "$NEW_FILE" ]; then
    echo "Error: Prompt file already exists: $NEW_FILE" >&2
    exit 1
fi

if jq -e ".versions[\"$NEW_VERSION\"]" prompts/versions.json > /dev/null 2>&1; then
    echo "Error: Version $NEW_VERSION already exists in prompts/versions.json" >&2
    exit 1
fi

echo "Creating new prompt version: $NEW_VERSION"
echo "  Base: $BASE_VERSION ($BASE_FILE)"
echo "  New:  $NEW_FILE"
echo "  Description: $DESCRIPTION"
echo ""

# Copy base file to new file
cp "$BASE_FILE" "$NEW_FILE"
echo "✓ Copied $BASE_FILE → $NEW_FILE"

# Rewrite line 1 so the header names THIS version.
# The cp above copies the base version's header verbatim; leaving it is how 15 of 54
# prompt files came to misidentify their own version, including the active one that
# `ailang prompt` serves (fixed iteration 293). Runs BEFORE the hash is computed.
HEADER_TMP=$(mktemp)
awk -v ver="$NEW_VERSION" 'NR==1{ sub(/^# AILANG v[0-9]+\.[0-9]+\.[0-9]+/, "# AILANG " ver) } {print}' \
    "$NEW_FILE" > "$HEADER_TMP"
mv "$HEADER_TMP" "$NEW_FILE"

# Fail loudly rather than silently shipping a mislabelled prompt (no silent fallbacks).
if ! head -1 "$NEW_FILE" | grep -q "^# AILANG ${NEW_VERSION} "; then
    echo "Error: could not rewrite the version header of $NEW_FILE" >&2
    echo "       line 1 is: $(head -1 "$NEW_FILE")" >&2
    echo "       expected it to start: # AILANG ${NEW_VERSION} " >&2
    exit 1
fi
echo "✓ Header set to: $(head -1 "$NEW_FILE")"

# Compute hash
HASH=$(shasum -a 256 "$NEW_FILE" | awk '{print $1}')
echo "✓ Computed hash: $HASH"

# Add to versions.json
CREATED=$(date +%Y-%m-%d)
TMP_FILE=$(mktemp)

jq --arg version "$NEW_VERSION" \
   --arg file "$NEW_FILE" \
   --arg hash "$HASH" \
   --arg desc "$DESCRIPTION" \
   --arg created "$CREATED" \
   '.versions[$version] = {
     file: $file,
     hash: $hash,
     description: $desc,
     created: $created,
     tags: ["production", "latest"],
     notes: $desc
   } | .active = $version' prompts/versions.json > "$TMP_FILE"

mv "$TMP_FILE" prompts/versions.json
echo "✓ Added $NEW_VERSION to prompts/versions.json"
echo "✓ Set as active version"

# Mirror BOTH artefacts into cmd/ailang/prompts/, which is what gets embedded into
# the binary and what agent mode reads FIRST (internal/prompt/loader.go). The two
# registries are required to be byte-identical, and `make check-prompt-freeze`
# enforces it — so a source-only write leaves the tree inconsistent and the gate red.
cp "$NEW_FILE" "cmd/ailang/$NEW_FILE"
cp prompts/versions.json cmd/ailang/prompts/versions.json
echo "✓ Mirrored $NEW_FILE and the registry into cmd/ailang/prompts/"

echo ""
echo "Next steps:"
echo "  1. Edit $NEW_FILE to make your changes"
echo "  2. Run .claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh $NEW_VERSION"
echo "  3. Update hash: shasum -a 256 $NEW_FILE"
echo "  4. RE-MIRROR after any edit (steps 1 and 3 desync the embedded copy):"
echo "       cp $NEW_FILE cmd/ailang/$NEW_FILE"
echo "       cp prompts/versions.json cmd/ailang/prompts/versions.json"
echo "  5. Test with: ailang repl (check if changes work)"
echo "  6. Run make check-prompt-freeze before committing"
echo "  7. Commit: git add $NEW_FILE cmd/ailang/$NEW_FILE prompts/versions.json cmd/ailang/prompts/versions.json"
