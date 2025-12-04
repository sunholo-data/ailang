#!/bin/bash
#
# Cleanup design docs after a release.
# Helps identify docs in planned/ that should be moved to implemented/.
#
# Usage: cleanup_design_docs.sh <version>
# Example: cleanup_design_docs.sh 0.5.6
#

set -e

VERSION="${1:-}"

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.5.6"
    exit 1
fi

# Convert version to folder format (0.5.6 -> v0_5_6)
FOLDER_VERSION="v${VERSION//./_}"

PLANNED_DIR="design_docs/planned"
IMPLEMENTED_DIR="design_docs/implemented"

echo "Design Doc Cleanup for v${VERSION}"
echo "=================================="
echo ""

# Check if planned folder for this version exists
if [ -d "${PLANNED_DIR}/${FOLDER_VERSION}" ]; then
    echo "Planned docs for ${FOLDER_VERSION}:"
    echo ""

    for doc in "${PLANNED_DIR}/${FOLDER_VERSION}"/*.md; do
        if [ -f "$doc" ]; then
            basename="$(basename "$doc")"

            # Check if doc says IMPLEMENTED in the first 10 lines
            if head -10 "$doc" | grep -qi "IMPLEMENTED\|Status.*IMPLEMENTED"; then
                echo "  [MOVE] $basename (marked as implemented)"
            elif head -10 "$doc" | grep -qi "SUPERSEDED"; then
                echo "  [SUPERSEDED] $basename (marked as superseded)"
            else
                echo "  [PENDING] $basename"
            fi
        fi
    done
    echo ""
else
    echo "No planned/${FOLDER_VERSION} folder found."
    echo ""
fi

# Check CHANGELOG for this version to find implemented milestones
echo "Checking CHANGELOG for v${VERSION} milestones..."
echo ""

if grep -q "## \[v${VERSION}\]" CHANGELOG.md; then
    echo "Milestones in CHANGELOG for v${VERSION}:"
    # Extract section between this version and previous version
    awk "/## \[v${VERSION}\]/,/## \[v[0-9]/" CHANGELOG.md | \
        grep -E "M-[A-Z0-9-]+|DX-[0-9]+" | \
        head -20 | \
        sed 's/^/  /'
    echo ""
else
    echo "  No CHANGELOG entry found for v${VERSION} yet."
    echo ""
fi

# List any docs in planned/ that mention IMPLEMENTED
echo "Other docs in planned/ marked as implemented:"
echo ""
found=0
for doc in "${PLANNED_DIR}"/*/*.md "${PLANNED_DIR}"/*.md; do
    if [ -f "$doc" ]; then
        if head -10 "$doc" | grep -qi "Status.*IMPLEMENTED"; then
            echo "  [MOVE] $doc"
            found=1
        fi
    fi
done 2>/dev/null

if [ $found -eq 0 ]; then
    echo "  (none found)"
fi
echo ""

echo "Next steps:"
echo "  1. Review docs marked [MOVE] and move to implemented/${FOLDER_VERSION}/"
echo "  2. Update docs marked [SUPERSEDED] and move to implemented/"
echo "  3. Create implemented/${FOLDER_VERSION}/ if it doesn't exist:"
echo "     mkdir -p ${IMPLEMENTED_DIR}/${FOLDER_VERSION}"
echo "  4. Move docs:"
echo "     mv ${PLANNED_DIR}/${FOLDER_VERSION}/<doc>.md ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
echo ""
