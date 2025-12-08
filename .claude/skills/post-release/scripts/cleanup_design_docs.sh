#!/bin/bash
#
# Move design docs from planned/ to implemented/ after a release.
# Only moves docs with "Status: Implemented" in their frontmatter.
# Docs without this status are flagged for review.
#
# Usage: cleanup_design_docs.sh <version> [--dry-run] [--force]
# Example: cleanup_design_docs.sh 0.5.7
#          cleanup_design_docs.sh 0.5.7 --dry-run
#          cleanup_design_docs.sh 0.5.7 --force  # Move all, ignore status
#

set -e

VERSION="${1:-}"
DRY_RUN=false
FORCE=false

# Check for flags
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --force) FORCE=true ;;
    esac
done

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--dry-run] [--force]"
    echo "Example: $0 0.5.7"
    echo "         $0 0.5.7 --dry-run   # Preview without changes"
    echo "         $0 0.5.7 --force     # Move all docs, ignore status"
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

if [ "$FORCE" = true ]; then
    echo "Mode: FORCE (moving all docs regardless of status)"
    echo ""
fi

# Check if planned folder for this version exists
if [ ! -d "${PLANNED_DIR}/${FOLDER_VERSION}" ]; then
    echo "No ${PLANNED_DIR}/${FOLDER_VERSION}/ folder found."
    echo "Nothing to move."
    exit 0
fi

# Count docs
DOC_COUNT=$(find "${PLANNED_DIR}/${FOLDER_VERSION}" -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')

if [ "$DOC_COUNT" -eq 0 ]; then
    echo "No design docs found in ${PLANNED_DIR}/${FOLDER_VERSION}/"
    echo "Nothing to move."
    exit 0
fi

echo "Checking ${DOC_COUNT} design doc(s) in ${PLANNED_DIR}/${FOLDER_VERSION}/:"
echo ""

MOVED_COUNT=0
NEEDS_REVIEW_COUNT=0
NEEDS_REVIEW_DOCS=""

# Check each doc
for doc in "${PLANNED_DIR}/${FOLDER_VERSION}"/*.md; do
    if [ -f "$doc" ]; then
        basename="$(basename "$doc")"

        # Check for "Status: Implemented" in first 15 lines (handles various formats)
        is_implemented=false
        if head -15 "$doc" | grep -qiE "^\*?\*?Status\*?\*?:?\s*\*?\*?Implemented"; then
            is_implemented=true
        fi

        if [ "$FORCE" = true ] || [ "$is_implemented" = true ]; then
            if [ "$DRY_RUN" = true ]; then
                if [ "$is_implemented" = true ]; then
                    echo "  [WOULD MOVE] $basename (Status: Implemented)"
                else
                    echo "  [WOULD MOVE] $basename (forced)"
                fi
            else
                # Create implemented folder if needed
                mkdir -p "${IMPLEMENTED_DIR}/${FOLDER_VERSION}"

                # Move the doc
                mv "$doc" "${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
                if [ "$is_implemented" = true ]; then
                    echo "  [MOVED] $basename → ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
                else
                    echo "  [MOVED] $basename → ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/ (forced)"
                fi
            fi
            MOVED_COUNT=$((MOVED_COUNT + 1))
        else
            # Check what the actual status is
            status_line=$(head -15 "$doc" | grep -iE "^\*?\*?Status\*?\*?:" | head -1 || echo "")
            if [ -n "$status_line" ]; then
                echo "  [NEEDS REVIEW] $basename"
                echo "                 Found: $status_line"
            else
                echo "  [NEEDS REVIEW] $basename (no Status field found)"
            fi
            NEEDS_REVIEW_COUNT=$((NEEDS_REVIEW_COUNT + 1))
            NEEDS_REVIEW_DOCS="$NEEDS_REVIEW_DOCS$basename\n"
        fi
    fi
done

echo ""

# Check if planned folder is now empty and remove it
if [ "$DRY_RUN" = false ] && [ "$MOVED_COUNT" -gt 0 ]; then
    remaining=$(find "${PLANNED_DIR}/${FOLDER_VERSION}" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [ "$remaining" -eq 0 ]; then
        rmdir "${PLANNED_DIR}/${FOLDER_VERSION}" 2>/dev/null || true
        echo "Removed empty ${PLANNED_DIR}/${FOLDER_VERSION}/ folder"
        echo ""
    fi
fi

# Summary
echo "Summary:"
if [ "$DRY_RUN" = true ]; then
    echo "  Would move: $MOVED_COUNT doc(s)"
    echo "  Needs review: $NEEDS_REVIEW_COUNT doc(s)"
    echo ""
    echo "Run without --dry-run to move files."
else
    if [ "$MOVED_COUNT" -gt 0 ]; then
        echo "  ✓ Moved $MOVED_COUNT doc(s) to ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
    fi
    if [ "$NEEDS_REVIEW_COUNT" -gt 0 ]; then
        echo "  ⚠ $NEEDS_REVIEW_COUNT doc(s) need review (not marked as Implemented)"
        echo ""
        echo "To fix docs needing review:"
        echo "  1. Update the doc's Status field to 'Implemented'"
        echo "  2. Re-run this script"
        echo "  Or use --force to move regardless of status"
    fi
fi

if [ "$MOVED_COUNT" -gt 0 ] && [ "$DRY_RUN" = false ]; then
    echo ""
    echo "Next steps:"
    echo "  git add design_docs/"
    echo "  git commit -m 'docs: move design docs to implemented/${FOLDER_VERSION}'"
fi
