#!/bin/bash
#
# Move design docs from planned/ to implemented/ after a release.
# Automatically moves ALL docs in planned/vX_Y_Z/ for the released version.
#
# Usage: cleanup_design_docs.sh <version> [--dry-run]
# Example: cleanup_design_docs.sh 0.5.7
#          cleanup_design_docs.sh 0.5.7 --dry-run
#

set -e

VERSION="${1:-}"
DRY_RUN=false

# Check for --dry-run flag
for arg in "$@"; do
    if [ "$arg" = "--dry-run" ]; then
        DRY_RUN=true
    fi
done

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--dry-run]"
    echo "Example: $0 0.5.7"
    echo "         $0 0.5.7 --dry-run"
    exit 1
fi

# Convert version to folder format (0.5.7 -> v0_5_7)
FOLDER_VERSION="v${VERSION//./_}"

PLANNED_DIR="design_docs/planned"
IMPLEMENTED_DIR="design_docs/implemented"

echo "Design Doc Cleanup for v${VERSION}"
echo "=================================="
echo ""

if [ "$DRY_RUN" = true ]; then
    echo "Mode: DRY RUN (no changes will be made)"
    echo ""
fi

# Check if planned folder for this version exists
if [ ! -d "${PLANNED_DIR}/${FOLDER_VERSION}" ]; then
    echo "No ${PLANNED_DIR}/${FOLDER_VERSION}/ folder found."
    echo "Nothing to move."
    exit 0
fi

# Count docs to move
DOC_COUNT=$(find "${PLANNED_DIR}/${FOLDER_VERSION}" -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')

if [ "$DOC_COUNT" -eq 0 ]; then
    echo "No design docs found in ${PLANNED_DIR}/${FOLDER_VERSION}/"
    echo "Nothing to move."
    exit 0
fi

echo "Found ${DOC_COUNT} design doc(s) in ${PLANNED_DIR}/${FOLDER_VERSION}/:"
echo ""

# List and optionally move docs
for doc in "${PLANNED_DIR}/${FOLDER_VERSION}"/*.md; do
    if [ -f "$doc" ]; then
        basename="$(basename "$doc")"

        if [ "$DRY_RUN" = true ]; then
            echo "  [WOULD MOVE] $basename"
        else
            # Create implemented folder if needed
            mkdir -p "${IMPLEMENTED_DIR}/${FOLDER_VERSION}"

            # Move the doc
            mv "$doc" "${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
            echo "  [MOVED] $basename → ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
        fi
    fi
done

echo ""

# Check if planned folder is now empty and remove it
if [ "$DRY_RUN" = false ]; then
    remaining=$(find "${PLANNED_DIR}/${FOLDER_VERSION}" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [ "$remaining" -eq 0 ]; then
        rmdir "${PLANNED_DIR}/${FOLDER_VERSION}" 2>/dev/null || true
        echo "Removed empty ${PLANNED_DIR}/${FOLDER_VERSION}/ folder"
        echo ""
    fi
fi

# Summary
if [ "$DRY_RUN" = true ]; then
    echo "Dry run complete. Run without --dry-run to move files."
else
    echo "✓ Moved ${DOC_COUNT} design doc(s) to ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
    echo ""
    echo "Next steps:"
    echo "  git add design_docs/"
    echo "  git commit -m 'docs: move design docs to implemented/${FOLDER_VERSION}'"
fi
