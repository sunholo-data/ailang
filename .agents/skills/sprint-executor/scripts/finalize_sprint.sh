#!/bin/bash
# finalize_sprint.sh - Finalize a completed sprint
# Moves design docs from planned/ to implemented/<version>/
# Updates sprint JSON status to "completed"

set -e

SPRINT_ID="${1:-}"
TARGET_VERSION="${2:-}"

if [ -z "$SPRINT_ID" ]; then
    echo "Usage: $0 <sprint-id> [target-version]"
    echo ""
    echo "Example: $0 M-BUG-RECORD-UPDATE-INFERENCE v0_4_9"
    echo ""
    echo "If target-version is not specified, will try to extract from design doc path"
    exit 1
fi

SPRINT_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"

if [ ! -f "$SPRINT_FILE" ]; then
    echo "Error: Sprint file not found: $SPRINT_FILE"
    exit 1
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Finalizing Sprint: $SPRINT_ID"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Check if all milestones pass
ALL_PASS=$(jq -r '.features | map(select(.passes != true)) | length' "$SPRINT_FILE")
if [ "$ALL_PASS" != "0" ]; then
    echo "Warning: Not all milestones have passes=true"
    echo "Milestones with passes != true:"
    jq -r '.features[] | select(.passes != true) | "  - \(.id): passes=\(.passes)"' "$SPRINT_FILE"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
fi

# Get design doc path from sprint JSON
DESIGN_DOC=$(jq -r '.design_doc // empty' "$SPRINT_FILE")
SPRINT_PLAN=$(jq -r '.sprint_plan // empty' "$SPRINT_FILE")

if [ -z "$DESIGN_DOC" ]; then
    echo "Warning: No design_doc field in sprint JSON"
    DESIGN_DOC=""
fi

# Determine target version
if [ -z "$TARGET_VERSION" ]; then
    # Try to extract from design doc path (e.g., design_docs/planned/v0_4_9/...)
    TARGET_VERSION=$(echo "$DESIGN_DOC" | grep -oE 'v[0-9]+_[0-9]+(_[0-9]+)?' | head -1)
    if [ -z "$TARGET_VERSION" ]; then
        echo "Error: Could not determine target version from design doc path"
        echo "Please specify target version as second argument"
        exit 1
    fi
fi

echo "Target version: $TARGET_VERSION"
echo ""

# Create implemented directory if needed
IMPL_DIR="design_docs/implemented/$TARGET_VERSION"
mkdir -p "$IMPL_DIR"

# Move design doc if it exists in planned/
MOVED_FILES=()

if [ -n "$DESIGN_DOC" ] && [ -f "$DESIGN_DOC" ]; then
    BASENAME=$(basename "$DESIGN_DOC")
    DEST="$IMPL_DIR/$BASENAME"

    if [ -f "$DEST" ]; then
        echo "Design doc already exists at: $DEST"
    else
        echo "Moving design doc:"
        echo "  From: $DESIGN_DOC"
        echo "  To:   $DEST"
        mv "$DESIGN_DOC" "$DEST"
        MOVED_FILES+=("$DESIGN_DOC -> $DEST")

        # Update design doc status to IMPLEMENTED
        if grep -q "^\\*\\*Status\\*\\*:" "$DEST"; then
            sed -i '' 's/^\*\*Status\*\*:.*$/\*\*Status\*\*: IMPLEMENTED/' "$DEST"
            echo "  Updated status to IMPLEMENTED"
        fi
    fi
fi

# Move sprint plan if it exists in planned/
if [ -n "$SPRINT_PLAN" ] && [ -f "$SPRINT_PLAN" ]; then
    BASENAME=$(basename "$SPRINT_PLAN")
    DEST="$IMPL_DIR/$BASENAME"

    if [ -f "$DEST" ]; then
        echo "Sprint plan already exists at: $DEST"
    else
        echo "Moving sprint plan:"
        echo "  From: $SPRINT_PLAN"
        echo "  To:   $DEST"
        mv "$SPRINT_PLAN" "$DEST"
        MOVED_FILES+=("$SPRINT_PLAN -> $DEST")
    fi
fi

# Update sprint JSON status to completed
echo ""
echo "Updating sprint JSON status to 'completed'..."
TEMP_FILE=$(mktemp)
jq '.status = "completed" | .completed = (now | strftime("%Y-%m-%dT%H:%M:%SZ"))' "$SPRINT_FILE" > "$TEMP_FILE"
mv "$TEMP_FILE" "$SPRINT_FILE"

# Update design_doc and sprint_plan paths in JSON to point to new locations
if [ ${#MOVED_FILES[@]} -gt 0 ]; then
    TEMP_FILE=$(mktemp)
    NEW_DESIGN_DOC="$IMPL_DIR/$(basename "$DESIGN_DOC" 2>/dev/null || echo "")"
    NEW_SPRINT_PLAN="$IMPL_DIR/$(basename "$SPRINT_PLAN" 2>/dev/null || echo "")"

    jq --arg dd "$NEW_DESIGN_DOC" --arg sp "$NEW_SPRINT_PLAN" '
        if .design_doc then .design_doc = $dd else . end |
        if .sprint_plan then .sprint_plan = $sp else . end
    ' "$SPRINT_FILE" > "$TEMP_FILE"
    mv "$TEMP_FILE" "$SPRINT_FILE"
    echo "Updated file paths in sprint JSON"
fi

# Get linked GitHub issues if any
GITHUB_ISSUES=$(jq -r '.github_issues // [] | map("#" + tostring) | join(", ")' "$SPRINT_FILE" 2>/dev/null || echo "")
ISSUE_REF=""
if [ -n "$GITHUB_ISSUES" ]; then
    ISSUE_REF=", refs $GITHUB_ISSUES"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Sprint Finalized Successfully!"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Summary:"
echo "  Sprint ID: $SPRINT_ID"
echo "  Status: completed"
echo "  Target version: $TARGET_VERSION"
if [ -n "$GITHUB_ISSUES" ]; then
    echo "  Linked issues: $GITHUB_ISSUES"
fi
if [ ${#MOVED_FILES[@]} -gt 0 ]; then
    echo ""
    echo "Files moved:"
    for f in "${MOVED_FILES[@]}"; do
        echo "  $f"
    done
fi
echo ""
echo "Next steps:"
if [ -n "$GITHUB_ISSUES" ]; then
    echo "  1. Commit the changes: git add -A && git commit -m 'Finalize sprint $SPRINT_ID${ISSUE_REF}'"
else
    echo "  1. Commit the changes: git add -A && git commit -m 'Finalize sprint $SPRINT_ID'"
fi
echo "  2. Update CHANGELOG.md if not already done"
echo "  3. Hand off to sprint-evaluator for independent quality assessment:"
echo "     ailang messages send sprint-evaluator '{"
echo "       \"type\": \"implementation_complete\","
echo "       \"sprint_id\": \"$SPRINT_ID\","
echo "       \"sprint_json_path\": \"$SPRINT_FILE\","
echo "       \"design_doc\": \"$(jq -r '.design_doc // ""' "$SPRINT_FILE")\","
echo "       \"evaluation_round\": 1"
echo "     }' --title \"Sprint $SPRINT_ID ready for evaluation\" --from \"sprint-executor\""
echo "  4. Consider creating a release if milestone reached"
