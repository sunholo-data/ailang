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

# Extract milestones from sprint plan markdown
echo "Parsing sprint plan..."

# Function to extract milestone info from sprint plan markdown
# Parses ### M1: ... sections, extracting:
#   - ID and description from heading
#   - estimated_loc from (~NNN LOC) in heading
#   - acceptance_criteria from - [ ] checklist items
#   - dependencies from Dependencies: line
extract_milestones() {
    local plan_file="$1"

    python3 - "$plan_file" << 'PYEOF'
import re, sys, json

plan_file = sys.argv[1]
with open(plan_file) as f:
    content = f.read()

# Split into milestone sections: ### M1: Title (~NNN LOC)
# Also handles ### M1: Title ... (~N hours, ~NNN LOC)
sections = re.split(r'(?=^### M\d)', content, flags=re.MULTILINE)

milestones = []
for section in sections:
    # Match heading: ### M1: Description (~120 LOC) or ### M1: Description (~3 hours, ~180 LOC)
    heading_match = re.match(r'^### (M\d+)[:\s]+(.+?)(?:\s*\(.*?~(\d+)\s*LOC.*?\))?\s*$', section, re.MULTILINE)
    if not heading_match:
        continue

    milestone_id = heading_match.group(1).strip()
    description = heading_match.group(2).strip()
    # Remove trailing parenthetical from description if it leaked
    description = re.sub(r'\s*\(~\d+\s*(hours?|LOC).*\)\s*$', '', description)
    estimated_loc = int(heading_match.group(3)) if heading_match.group(3) else 0

    # Extract acceptance criteria from - [ ] lines
    criteria = []
    for line in section.split('\n'):
        m = re.match(r'^\s*-\s*\[\s*\]\s*(.+)$', line)
        if m:
            criteria.append(m.group(1).strip())

    # Extract dependencies from "Dependencies:" line
    deps = []
    dep_match = re.search(r'\*\*Dependencies[:\*]*\s*(.+)', section)
    if dep_match:
        dep_text = dep_match.group(1).strip()
        if dep_text.lower() not in ('none', 'n/a', '—', '-'):
            # Find M1, M2, etc. references
            deps = re.findall(r'M\d+', dep_text)

    milestones.append({
        "id": milestone_id + "_" + re.sub(r'[^A-Z0-9]+', '_', description.upper()).strip('_')[:30],
        "description": description,
        "estimated_loc": estimated_loc,
        "dependencies": deps,
        "acceptance_criteria": criteria if criteria else ["Acceptance criteria not parsed - fill manually"],
        "passes": None,
        "started": None,
        "completed": None,
        "notes": None
    })

if not milestones:
    # Fallback: return a single placeholder if parsing found nothing
    milestones = [{
        "id": "MILESTONE_ID",
        "description": "Milestone description (auto-parse failed - fill manually)",
        "estimated_loc": 0,
        "dependencies": [],
        "acceptance_criteria": ["Criterion 1", "Criterion 2"],
        "passes": None,
        "started": None,
        "completed": None,
        "notes": None
    }]

print(json.dumps(milestones, indent=2))
PYEOF
}

# Get current timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Extract velocity estimates from sprint plan
ESTIMATED_TOTAL_LOC=$(grep -oE 'Total.*\*\*.*\|.*\|.*(\d+)' "$SPRINT_PLAN" 2>/dev/null | grep -oE '[0-9]+$' | tail -1 || echo "")
if [ -z "$ESTIMATED_TOTAL_LOC" ]; then
    # Try alternative: look for "~NNN LOC total" or "Estimated LOC" table
    ESTIMATED_TOTAL_LOC=$(grep -iE '(total|estimated).*\b[0-9]+\b' "$SPRINT_PLAN" 2>/dev/null | grep -oE '\b[0-9]{2,}\b' | tail -1 || echo "")
fi
ESTIMATED_TOTAL_LOC=${ESTIMATED_TOTAL_LOC:-1000}

ESTIMATED_DAYS=$(grep -iE '(duration|estimated).*\b[0-9]+\b.*day' "$SPRINT_PLAN" 2>/dev/null | grep -oE '\b[0-9]+\b' | head -1 || echo "")
ESTIMATED_DAYS=${ESTIMATED_DAYS:-7}

TARGET_LOC_PER_DAY=$((ESTIMATED_TOTAL_LOC / ESTIMATED_DAYS))

# Extract milestones into temp file (avoid heredoc interpolation issues with null)
MILESTONES_TMPFILE=$(mktemp)
extract_milestones "$SPRINT_PLAN" > "$MILESTONES_TMPFILE"
MILESTONE_COUNT=$(python3 -c "import json; print(len(json.load(open('$MILESTONES_TMPFILE'))))" 2>/dev/null || echo "0")

if [ "$MILESTONE_COUNT" -gt 1 ]; then
    echo -e "${GREEN}✓ Parsed ${MILESTONE_COUNT} milestones from sprint plan${NC}"
else
    echo -e "⚠️  Could not auto-parse milestones (found ${MILESTONE_COUNT}). Template created — fill manually."
fi

# Create JSON structure using python3 — reads milestones from temp file
python3 - "$SPRINT_ID" "$TIMESTAMP" "$ESTIMATED_DAYS" "$DESIGN_DOC" "$SPRINT_PLAN" "$ESTIMATED_TOTAL_LOC" "$TARGET_LOC_PER_DAY" "$PROGRESS_FILE" "$MILESTONES_TMPFILE" << 'PYEOF2'
import json, sys

sprint_id = sys.argv[1]
timestamp = sys.argv[2]
estimated_days = int(sys.argv[3])
design_doc = sys.argv[4]
sprint_plan = sys.argv[5]
estimated_total_loc = int(sys.argv[6])
target_loc_per_day = int(sys.argv[7])
output_file = sys.argv[8]
milestones_file = sys.argv[9]

with open(milestones_file) as f:
    milestones = json.load(f)

data = {
    "sprint_id": sprint_id,
    "status": "not_started",
    "created": timestamp,
    "design_doc": design_doc,
    "sprint_plan": sprint_plan,
    "github_issues": [],
    "velocity": {
        "target_loc_per_day": target_loc_per_day,
        "estimated_total_loc": estimated_total_loc,
        "estimated_days": estimated_days
    },
    "features": milestones
}

with open(output_file, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PYEOF2

# Clean up temp file
rm -f "$MILESTONES_TMPFILE"

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
