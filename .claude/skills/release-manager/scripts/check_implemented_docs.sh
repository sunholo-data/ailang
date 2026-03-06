#!/bin/bash
# Check that implemented design docs are documented in CHANGELOG
# Usage: ./check_implemented_docs.sh <version>
# Example: ./check_implemented_docs.sh 0.5.10

set -e

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.5.10"
    exit 1
fi

# Normalize version (remove v prefix if present)
VERSION="${VERSION#v}"
VERSION_DIR="v${VERSION//./_}"

IMPLEMENTED_DIR="design_docs/implemented/$VERSION_DIR"
CHANGELOG_DIR="changelogs"

echo "Checking implemented design docs for v$VERSION..."
echo ""

# Check if implemented directory exists
if [ ! -d "$IMPLEMENTED_DIR" ]; then
    echo "  ⚠ No implemented docs directory: $IMPLEMENTED_DIR"
    echo "  This may be expected for minor releases with no new features."
    exit 0
fi

# Count docs
DOC_COUNT=$(ls -1 "$IMPLEMENTED_DIR"/*.md 2>/dev/null | wc -l | tr -d ' ')
echo "  Found $DOC_COUNT design doc(s) in $IMPLEMENTED_DIR"
echo ""

# Check each doc is referenced in CHANGELOG
MISSING=()
FOUND=()

for doc in "$IMPLEMENTED_DIR"/*.md; do
    [ -e "$doc" ] || continue

    # Extract the doc filename
    filename=$(basename "$doc" .md)

    # Skip sprint plans and analysis docs (they're implementation artifacts, not features)
    if [[ "$filename" == *"-sprint"* ]] || [[ "$filename" == *"-sprint-plan"* ]] || [[ "$filename" == *"-analysis"* ]]; then
        continue
    fi

    # Check if it's referenced in any changelog file
    if grep -rqi "$filename" "$CHANGELOG_DIR" 2>/dev/null; then
        FOUND+=("$filename")
    else
        MISSING+=("$filename")
    fi
done

echo "  Referenced in changelogs: ${#FOUND[@]}"
for f in "${FOUND[@]}"; do
    echo "    ✓ $f"
done

echo ""

if [ ${#MISSING[@]} -gt 0 ]; then
    echo "  ⚠ NOT in changelogs: ${#MISSING[@]}"
    for f in "${MISSING[@]}"; do
        echo "    ✗ $f"
    done
    echo ""
    echo "  These design docs should be added to the active changelog in changelogs/:"
    for f in "${MISSING[@]}"; do
        echo "    - design_docs/implemented/$VERSION_DIR/$f.md"
    done
    echo ""
    echo "  Action: Add entries for these features before releasing."
    echo "  Active changelog: ls changelogs/ | grep current"
    exit 1
else
    echo "  ✓ All feature design docs are referenced in changelogs"
fi
