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

# Sync GitHub issues via ailang messages integration
echo "Syncing GitHub issues via ailang messages..."
if command -v ailang &> /dev/null; then
    ailang messages import-github --labels bug,feature,ailang-message 2>/dev/null || true
    echo -e "${GREEN}  ✓ Synced issues from GitHub${NC}"
else
    echo "  ⚠️  ailang not found, skipping GitHub sync"
fi
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
    "estimated_loc": 0,
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
  "github_issues": [],
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

# Try to extract GitHub issue numbers from multiple sources
echo ""
echo "Discovering linked GitHub issues..."

GITHUB_ISSUES=""
DISCOVERED_ISSUES=""

# Source 1: Extract message IDs from design doc Bug Report field
if [ -n "$DESIGN_DOC" ] && [ -f "$DESIGN_DOC" ]; then
    MSG_IDS=$(grep -oE 'msg_[0-9]{8}_[0-9]{6}_[a-f0-9]+' "$DESIGN_DOC" 2>/dev/null || echo "")

    if [ -n "$MSG_IDS" ]; then
        for msg_id in $MSG_IDS; do
            ISSUE_NUM=$(ailang messages read "$msg_id" --json 2>/dev/null | jq -r '.github_issue // empty' 2>/dev/null || echo "")
            if [ -n "$ISSUE_NUM" ] && [ "$ISSUE_NUM" != "null" ] && [ "$ISSUE_NUM" != "0" ]; then
                DISCOVERED_ISSUES="${DISCOVERED_ISSUES} ${ISSUE_NUM}"
                echo -e "${GREEN}  ✓ Found #${ISSUE_NUM} from Bug Report message${NC}"
            fi
        done
    fi
fi

# Source 2: Query ailang messages for issues matching design doc keywords
if command -v ailang &> /dev/null && [ -n "$DESIGN_DOC" ] && [ -f "$DESIGN_DOC" ]; then
    echo "  Matching messages against design doc keywords..."

    # Extract keywords from design doc (significant words)
    DOC_TEXT=$(cat "$DESIGN_DOC" 2>/dev/null || echo "")
    MESSAGES_JSON=$(ailang messages list --json --limit 100 2>/dev/null || echo "[]")

    # Get all messages with github_issue field
    echo "$MESSAGES_JSON" | jq -c '.[] | select(.github_issue != null and .github_issue > 0)' 2>/dev/null | while IFS= read -r msg_json; do
        issue_num=$(echo "$msg_json" | jq -r '.github_issue' 2>/dev/null || echo "")
        title=$(echo "$msg_json" | jq -r '.title // ""' 2>/dev/null || echo "")
        payload=$(echo "$msg_json" | jq -r '.payload // ""' 2>/dev/null || echo "")

        [ -z "$issue_num" ] && continue
        # Skip if already found
        echo "$DISCOVERED_ISSUES" | grep -qw "$issue_num" && continue

        # Check if any significant word from issue appears in design doc
        for word in $(echo "$title $payload" | tr '[:upper:]' '[:lower:]' | grep -oE '\b[a-z]{5,}\b' | grep -vE '^(the|and|for|with|this|that|from|have|been|will|error|should|expected|context|function|return|value)$' | head -5); do
            if echo "$DOC_TEXT" | grep -qi "$word" 2>/dev/null; then
                echo "$issue_num" >> /tmp/sprint_issues_$$
                echo -e "${GREEN}  ✓ Found #${issue_num} matching keyword '${word}'${NC}"
                break
            fi
        done
    done

    # Collect keyword-matched issues
    if [ -f /tmp/sprint_issues_$$ ]; then
        KEYWORD_ISSUES=$(cat /tmp/sprint_issues_$$ | sort -u | tr '\n' ' ')
        DISCOVERED_ISSUES="${DISCOVERED_ISSUES} ${KEYWORD_ISSUES}"
        rm -f /tmp/sprint_issues_$$
    fi
fi

# Source 3: Extract explicit #123 references from design doc
if [ -n "$DESIGN_DOC" ] && [ -f "$DESIGN_DOC" ]; then
    EXPLICIT_REFS=$(grep -oE '#[0-9]+' "$DESIGN_DOC" 2>/dev/null | tr -d '#' | sort -u || echo "")
    for issue_num in $EXPLICIT_REFS; do
        if ! echo "$DISCOVERED_ISSUES" | grep -qw "$issue_num"; then
            DISCOVERED_ISSUES="${DISCOVERED_ISSUES} ${issue_num}"
            echo -e "${GREEN}  ✓ Found #${issue_num} from explicit reference${NC}"
        fi
    done
fi

# Deduplicate and format
GITHUB_ISSUES=$(echo "$DISCOVERED_ISSUES" | tr ' ' '\n' | grep -v '^$' | sort -un | tr '\n' ',' | sed 's/,$//')

if [ -n "$GITHUB_ISSUES" ]; then
    # Update sprint JSON with GitHub issues
    TEMP_FILE=$(mktemp)
    jq --argjson issues "[$GITHUB_ISSUES]" '.github_issues = $issues' "$PROGRESS_FILE" > "$TEMP_FILE"
    mv "$TEMP_FILE" "$PROGRESS_FILE"
    echo -e "${GREEN}  ✓ Added github_issues: [${GITHUB_ISSUES}] to sprint JSON${NC}"
else
    echo -e "  ℹ️  No related GitHub issues found"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo " Next Steps"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "1. Edit the JSON file to fill in actual milestone details:"
echo "   ${PROGRESS_FILE}"
echo ""
echo "2. If this sprint addresses a GitHub issue, ensure github_issues is set:"
echo "   jq '.github_issues = [123, 456]' ${PROGRESS_FILE} > tmp && mv tmp ${PROGRESS_FILE}"
echo ""
echo "3. Send handoff message to sprint-executor:"
echo "   ailang agent send sprint-executor '{"
echo "     \"type\": \"plan_ready\","
echo "     \"correlation_id\": \"sprint_${SPRINT_ID}\","
echo "     \"sprint_id\": \"${SPRINT_ID}\","
echo "     \"plan_path\": \"${SPRINT_PLAN}\","
echo "     \"progress_path\": \"${PROGRESS_FILE}\","
echo "     \"estimated_duration\": \"${ESTIMATED_DAYS} days\""
echo "   }'"
echo ""
echo "4. Start sprint execution:"
echo "   Use sprint-executor skill to begin implementing milestones"
echo ""
echo "═══════════════════════════════════════════════════════════════"

exit 0
