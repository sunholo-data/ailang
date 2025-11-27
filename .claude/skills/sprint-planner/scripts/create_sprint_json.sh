#!/bin/bash
#
# Create Sprint JSON Progress File
#
# Purpose: Generate structured JSON progress file from sprint plan
# Based on: https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
#
# Usage:
#   .claude/skills/sprint-planner/scripts/create_sprint_json.sh \
#     <sprint_id> \
#     <sprint_plan_md> \
#     <design_doc_md>
#
# Example:
#   .claude/skills/sprint-planner/scripts/create_sprint_json.sh \
#     "M-S1" \
#     "design_docs/planned/v0_4_0/m-s1-sprint-plan.md" \
#     "design_docs/planned/v0_4_0/m-s1-parser-improvements.md"
#
# This script implements the "Initializer" pattern from the Anthropic article:
# - Creates structured JSON with feature list
# - Only `passes` field should be modified during execution
# - Enables multi-session continuity

set -e  # Exit on error

# Check arguments
if [ $# -lt 2 ]; then
    echo "Usage: $0 <sprint_id> <sprint_plan_md> [design_doc_md]"
    echo "Example: $0 M-S1 design_docs/planned/v0_4_0/m-s1-sprint-plan.md design_docs/planned/v0_4_0/m-s1-parser-improvements.md"
    exit 1
fi

SPRINT_ID="$1"
SPRINT_PLAN="$2"
DESIGN_DOC="${3:-}"

# Output file
PROGRESS_DIR=".ailang/state/sprints"
PROGRESS_FILE="${PROGRESS_DIR}/sprint_${SPRINT_ID}.json"

# Create state directory if it doesn't exist
mkdir -p "$PROGRESS_DIR"

# Check if sprint plan exists
if [ ! -f "$SPRINT_PLAN" ]; then
    echo "Error: Sprint plan not found: $SPRINT_PLAN"
    exit 1
fi

# Check if design doc exists (if provided)
if [ -n "$DESIGN_DOC" ] && [ ! -f "$DESIGN_DOC" ]; then
    echo "Warning: Design doc not found: $DESIGN_DOC"
    DESIGN_DOC=""
fi

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "═══════════════════════════════════════════════════════════════"
echo " Creating Sprint JSON Progress File"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Sprint ID: $SPRINT_ID"
echo "Sprint Plan: $SPRINT_PLAN"
if [ -n "$DESIGN_DOC" ]; then
    echo "Design Doc: $DESIGN_DOC"
fi
echo "Output: $PROGRESS_FILE"
echo ""

# Check if file already exists
if [ -f "$PROGRESS_FILE" ]; then
    echo -e "⚠️  Progress file already exists: $PROGRESS_FILE"
    echo ""
    read -p "Overwrite? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
fi

# Extract milestones from sprint plan
# This is a simplified parser - you may need to customize based on your sprint plan format
echo "Parsing sprint plan..."

# Function to extract milestone info
# This assumes a specific format - adjust to match your sprint plans
extract_milestones() {
    local plan_file="$1"

    # This is a placeholder - needs customization based on actual sprint plan format
    # For now, create a template that can be manually filled in

    cat << 'EOF'
[
  {
    "id": "MILESTONE_ID",
    "description": "Milestone description",
    "estimated_loc": 200,
    "actual_loc": null,
    "dependencies": [],
    "acceptance_criteria": [
      "Criterion 1",
      "Criterion 2"
    ],
    "passes": null,
    "started": null,
    "completed": null,
    "notes": null
  }
]
EOF
}

# Get current timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Calculate estimated totals (placeholder - customize based on sprint plan)
ESTIMATED_TOTAL_LOC=1000
ESTIMATED_DAYS=7
TARGET_LOC_PER_DAY=150

# Create JSON structure
cat > "$PROGRESS_FILE" << EOF
{
  "sprint_id": "${SPRINT_ID}",
  "created": "${TIMESTAMP}",
  "estimated_duration_days": ${ESTIMATED_DAYS},
  "correlation_id": "sprint_${SPRINT_ID}",
  "design_doc": "${DESIGN_DOC}",
  "markdown_plan": "${SPRINT_PLAN}",
  "features": $(extract_milestones "$SPRINT_PLAN"),
  "velocity": {
    "target_loc_per_day": ${TARGET_LOC_PER_DAY},
    "actual_loc_per_day": 0,
    "target_milestones_per_week": 5,
    "actual_milestones_per_week": 0,
    "estimated_total_loc": ${ESTIMATED_TOTAL_LOC},
    "actual_total_loc": 0,
    "estimated_days": ${ESTIMATED_DAYS},
    "actual_days": null
  },
  "last_session": "${TIMESTAMP}",
  "last_checkpoint": null,
  "status": "not_started"
}
EOF

echo -e "${GREEN}✓ Created JSON progress file${NC}"
echo ""

# Validate JSON
if jq -e . "$PROGRESS_FILE" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ JSON validation passed${NC}"
else
    echo "✗ JSON validation failed!"
    echo "Please check $PROGRESS_FILE for syntax errors"
    exit 1
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Next Steps"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "1. Edit the JSON file to fill in actual milestone details:"
echo "   ${PROGRESS_FILE}"
echo ""
echo "2. Send handoff message to sprint-executor:"
echo "   ailang agent send sprint-executor '{"
echo "     \"type\": \"plan_ready\","
echo "     \"correlation_id\": \"sprint_${SPRINT_ID}\","
echo "     \"sprint_id\": \"${SPRINT_ID}\","
echo "     \"plan_path\": \"${SPRINT_PLAN}\","
echo "     \"progress_path\": \"${PROGRESS_FILE}\","
echo "     \"estimated_duration\": \"${ESTIMATED_DAYS} days\""
echo "   }'"
echo ""
echo "3. Start sprint execution:"
echo "   Use sprint-executor skill to begin implementing milestones"
echo ""
echo "═══════════════════════════════════════════════════════════════"

exit 0
