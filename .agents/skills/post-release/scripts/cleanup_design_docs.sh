#!/bin/bash
#
# Move design docs from planned/ to implemented/ after a release.
# Only moves docs with "Status: Implemented" in their frontmatter.
# Docs without this status are flagged for review.
#
# Features:
# - Detects duplicates (docs already in implemented/)
# - Detects misplaced docs (Target: field doesn't match folder)
# - Suggests removal of superseded docs
#
# Usage: cleanup_design_docs.sh <version> [--dry-run] [--force] [--check-only]
# Example: cleanup_design_docs.sh 0.5.7
#          cleanup_design_docs.sh 0.5.7 --dry-run     # Preview without changes
#          cleanup_design_docs.sh 0.5.7 --force       # Move all, ignore status
#          cleanup_design_docs.sh 0.5.7 --check-only  # Only report issues, don't move
#

set -e

VERSION="${1:-}"
DRY_RUN=false
FORCE=false
CHECK_ONLY=false

# Check for flags
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --force) FORCE=true ;;
        --check-only) CHECK_ONLY=true ;;
    esac
done

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--dry-run] [--force] [--check-only]"
    echo "Example: $0 0.5.7"
    echo "         $0 0.5.7 --dry-run     # Preview without changes"
    echo "         $0 0.5.7 --force       # Move all docs, ignore status"
    echo "         $0 0.5.7 --check-only  # Report issues only"
    exit 1
fi

# Convert version to folder format (0.5.7 -> v0_5_7)
FOLDER_VERSION="v${VERSION//./_}"

PLANNED_DIR="design_docs/planned"
IMPLEMENTED_DIR="design_docs/implemented"

echo "Design Doc Cleanup for v${VERSION}"
echo "=================================="
echo ""

if [ "$CHECK_ONLY" = true ]; then
    echo "Mode: CHECK ONLY (reporting issues, no changes)"
    echo ""
elif [ "$DRY_RUN" = true ]; then
    echo "Mode: DRY RUN (no changes will be made)"
    echo ""
fi

if [ "$FORCE" = true ] && [ "$CHECK_ONLY" = false ]; then
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

# --- Phase 1: Issue Detection ---
echo "Phase 1: Detecting issues..."
echo ""

DUPLICATE_COUNT=0
MISPLACED_COUNT=0
DUPLICATE_DOCS=""
MISPLACED_DOCS=""

for doc in "${PLANNED_DIR}/${FOLDER_VERSION}"/*.md; do
    if [ -f "$doc" ]; then
        basename="$(basename "$doc")"

        # Check for duplicates (already exists in implemented/)
        if [ -f "${IMPLEMENTED_DIR}/${FOLDER_VERSION}/${basename}" ]; then
            echo "  [DUPLICATE] $basename"
            echo "              Already exists in ${IMPLEMENTED_DIR}/${FOLDER_VERSION}/"
            DUPLICATE_COUNT=$((DUPLICATE_COUNT + 1))
            DUPLICATE_DOCS="$DUPLICATE_DOCS$basename\n"
            continue
        fi

        # Check for misplaced docs (Target: field doesn't match folder version)
        # Handles formats: **Target**: v0.5.10, Target: v0.5.10, **Target:** 0.5.10
        target_version=$(head -20 "$doc" | grep -iE "Target" | grep -oE "v?[0-9]+\.[0-9]+\.[0-9]+" | sed 's/^v//' | head -1 || echo "")
        if [ -n "$target_version" ] && [ "$target_version" != "$VERSION" ]; then
            target_folder="v${target_version//./_}"
            echo "  [MISPLACED] $basename"
            echo "              Target: v${target_version} (folder: ${FOLDER_VERSION})"
            echo "              Should be in: ${PLANNED_DIR}/${target_folder}/"
            MISPLACED_COUNT=$((MISPLACED_COUNT + 1))
            MISPLACED_DOCS="$MISPLACED_DOCS$basename:$target_folder\n"
        fi
    fi
done

if [ "$DUPLICATE_COUNT" -gt 0 ] || [ "$MISPLACED_COUNT" -gt 0 ]; then
    echo ""
    echo "Issues found:"
    [ "$DUPLICATE_COUNT" -gt 0 ] && echo "  - $DUPLICATE_COUNT duplicate(s) (can be deleted)"
    [ "$MISPLACED_COUNT" -gt 0 ] && echo "  - $MISPLACED_COUNT misplaced doc(s) (wrong version folder)"
    echo ""
fi

# In check-only mode, stop here
if [ "$CHECK_ONLY" = true ]; then
    echo "Check complete. Run without --check-only to process docs."
    exit 0
fi

# --- Phase 2: Move implemented docs ---
echo "Phase 2: Processing docs..."
echo ""

MOVED_COUNT=0
NEEDS_REVIEW_COUNT=0
NEEDS_REVIEW_DOCS=""
SKIPPED_DUPLICATE=0
SKIPPED_MISPLACED=0

# Check each doc
for doc in "${PLANNED_DIR}/${FOLDER_VERSION}"/*.md; do
    if [ -f "$doc" ]; then
        basename="$(basename "$doc")"

        # Skip duplicates (already flagged in Phase 1)
        if [ -f "${IMPLEMENTED_DIR}/${FOLDER_VERSION}/${basename}" ]; then
            if [ "$DRY_RUN" = true ]; then
                echo "  [WOULD DELETE] $basename (duplicate)"
            else
                rm "$doc"
                echo "  [DELETED] $basename (duplicate - already in implemented/)"
            fi
            SKIPPED_DUPLICATE=$((SKIPPED_DUPLICATE + 1))
            continue
        fi

        # Check for misplaced docs
        # Handles formats: **Target**: v0.5.10, Target: v0.5.10, **Target:** 0.5.10
        target_version=$(head -20 "$doc" | grep -iE "Target" | grep -oE "v?[0-9]+\.[0-9]+\.[0-9]+" | sed 's/^v//' | head -1 || echo "")
        if [ -n "$target_version" ] && [ "$target_version" != "$VERSION" ]; then
            target_folder="v${target_version//./_}"
            if [ "$DRY_RUN" = true ]; then
                echo "  [WOULD RELOCATE] $basename → ${PLANNED_DIR}/${target_folder}/"
            else
                mkdir -p "${PLANNED_DIR}/${target_folder}"
                mv "$doc" "${PLANNED_DIR}/${target_folder}/"
                echo "  [RELOCATED] $basename → ${PLANNED_DIR}/${target_folder}/"
            fi
            SKIPPED_MISPLACED=$((SKIPPED_MISPLACED + 1))
            continue
        fi

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
    [ "$SKIPPED_DUPLICATE" -gt 0 ] && echo "  Would delete: $SKIPPED_DUPLICATE duplicate(s)"
    [ "$SKIPPED_MISPLACED" -gt 0 ] && echo "  Would relocate: $SKIPPED_MISPLACED misplaced doc(s)"
    [ "$MOVED_COUNT" -gt 0 ] && echo "  Would move: $MOVED_COUNT doc(s) to implemented/"
    [ "$NEEDS_REVIEW_COUNT" -gt 0 ] && echo "  Needs review: $NEEDS_REVIEW_COUNT doc(s)"
    echo ""
    echo "Run without --dry-run to apply changes."
else
    if [ "$SKIPPED_DUPLICATE" -gt 0 ]; then
        echo "  ✓ Deleted $SKIPPED_DUPLICATE duplicate(s)"
    fi
    if [ "$SKIPPED_MISPLACED" -gt 0 ]; then
        echo "  ✓ Relocated $SKIPPED_MISPLACED doc(s) to correct version folder"
    fi
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

TOTAL_CHANGES=$((SKIPPED_DUPLICATE + SKIPPED_MISPLACED + MOVED_COUNT))
if [ "$TOTAL_CHANGES" -gt 0 ] && [ "$DRY_RUN" = false ]; then
    echo ""
    echo "Next steps:"
    echo "  git add design_docs/"
    echo "  git commit -m 'docs: cleanup design docs for ${FOLDER_VERSION}'"
fi
