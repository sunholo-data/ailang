#!/usr/bin/env bash
# Generate a complete GitHub issue triage report
# Usage: triage_report.sh [--output FILE] [--json]
#
# Report includes:
# - Summary statistics
# - Closable issues (implemented)
# - Covered issues (have design docs)
# - Orphaned issues (need attention)
# - Stale issues (no activity 30+ days)
# - Recommendations

set -eo pipefail  # Note: removed -u to handle unset vars in loops

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# Defaults
OUTPUT_FILE=""
JSON_OUTPUT=false
REPO="sunholo-data/ailang"
STALE_DAYS=30

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --stale-days)
            STALE_DAYS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Check auth
"$SCRIPT_DIR/check_auth.sh" || exit 1

echo ""
echo "Generating Triage Report..."
echo ""

# Get all open issues with full details
ISSUES=$(gh issue list --repo "$REPO" --state open --limit 100 --json number,title,labels,createdAt,updatedAt,assignees,url,body 2>/dev/null || echo "[]")

TOTAL=$(echo "$ISSUES" | jq 'length')

if [[ "$TOTAL" == "0" ]]; then
    echo "No open issues found. Issue tracker is clean!"
    exit 0
fi

# Calculate date threshold for stale
if command -v gdate &> /dev/null; then
    STALE_DATE=$(gdate -d "$STALE_DAYS days ago" +%Y-%m-%dT%H:%M:%SZ)
elif date --version 2>/dev/null | grep -q GNU; then
    STALE_DATE=$(date -d "$STALE_DAYS days ago" +%Y-%m-%dT%H:%M:%SZ)
else
    # macOS date
    STALE_DATE=$(date -v-${STALE_DAYS}d +%Y-%m-%dT%H:%M:%SZ)
fi

# Categorize issues
CLOSABLE_ISSUES=""
COVERED_ISSUES=""
ORPHANED_ISSUES=""
STALE_ISSUES=""

CLOSABLE_COUNT=0
COVERED_COUNT=0
ORPHANED_COUNT=0
STALE_COUNT=0
BUG_COUNT=0
ENHANCEMENT_COUNT=0

echo "$ISSUES" | jq -c '.[]' | while read -r issue; do
    NUMBER=$(echo "$issue" | jq -r '.number')
    TITLE=$(echo "$issue" | jq -r '.title')
    UPDATED=$(echo "$issue" | jq -r '.updatedAt')
    LABELS=$(echo "$issue" | jq -r '.labels | map(.name) | join(",")')
    URL=$(echo "$issue" | jq -r '.url')

    # Track label counts
    if echo "$LABELS" | grep -q "bug"; then
        BUG_COUNT=$((BUG_COUNT + 1))
    fi
    if echo "$LABELS" | grep -q "enhancement"; then
        ENHANCEMENT_COUNT=$((ENHANCEMENT_COUNT + 1))
    fi

    # Check if closable (in implemented docs or changelog)
    IN_IMPLEMENTED=""
    IN_IMPLEMENTED=$(grep -rl "#$NUMBER" design_docs/implemented/ 2>/dev/null | head -1) || true
    IN_CHANGELOG=""
    IN_CHANGELOG=$(grep -l "#$NUMBER" CHANGELOG.md 2>/dev/null) || true

    if [[ -n "$IN_IMPLEMENTED" ]] || [[ -n "$IN_CHANGELOG" ]]; then
        echo "CLOSABLE|$NUMBER|$TITLE|$LABELS|$URL"
        continue
    fi

    # Check if covered (in planned docs)
    IN_PLANNED=""
    IN_PLANNED=$(grep -rl "#$NUMBER" design_docs/planned/ 2>/dev/null | head -1) || true

    # Also check by M-ID
    M_ID=""
    M_ID=$(echo "$TITLE" | grep -oE "M-[A-Z0-9-]+" | head -1) || true
    if [[ -n "$M_ID" ]] && [[ -z "$IN_PLANNED" ]]; then
        M_ID_LOWER=$(echo "$M_ID" | tr '[:upper:]' '[:lower:]' | tr '-' '_')
        IN_PLANNED=$(find design_docs/planned/ -name "*${M_ID_LOWER}*" 2>/dev/null | head -1) || true
    fi

    if [[ -n "$IN_PLANNED" ]]; then
        echo "COVERED|$NUMBER|$TITLE|$LABELS|$IN_PLANNED"
        continue
    fi

    # Check if stale
    if [[ "$UPDATED" < "$STALE_DATE" ]]; then
        echo "STALE|$NUMBER|$TITLE|$LABELS|$UPDATED"
    else
        echo "ORPHANED|$NUMBER|$TITLE|$LABELS|$URL"
    fi
done > /tmp/triage_categories_$$

# Count categories
count_category() {
    local pattern="$1"
    local count
    count=$(grep -c "$pattern" /tmp/triage_categories_$$ 2>/dev/null) || count=0
    echo "$count"
}

CLOSABLE_COUNT=$(count_category "^CLOSABLE|")
COVERED_COUNT=$(count_category "^COVERED|")
STALE_COUNT=$(count_category "^STALE|")
ORPHANED_COUNT=$(count_category "^ORPHANED|")

# Generate report
REPORT=""
REPORT+="# GitHub Issue Triage Report\n"
REPORT+="\n"
REPORT+="**Generated:** $(date '+%Y-%m-%d %H:%M:%S')\n"
REPORT+="**Repository:** $REPO\n"
REPORT+="\n"
REPORT+="## Summary\n"
REPORT+="\n"
REPORT+="| Category | Count |\n"
REPORT+="|----------|-------|\n"
REPORT+="| Total Open | $TOTAL |\n"
REPORT+="| Closable (implemented) | $CLOSABLE_COUNT |\n"
REPORT+="| Covered (has design doc) | $COVERED_COUNT |\n"
REPORT+="| Orphaned (needs attention) | $ORPHANED_COUNT |\n"
REPORT+="| Stale (${STALE_DAYS}+ days inactive) | $STALE_COUNT |\n"
REPORT+="\n"

# Closable section
if [[ "$CLOSABLE_COUNT" -gt 0 ]]; then
    REPORT+="## Closable Issues\n"
    REPORT+="\n"
    REPORT+="These issues are implemented and can be closed:\n"
    REPORT+="\n"
    while IFS='|' read -r cat num title labels extra; do
        [[ "$cat" != "CLOSABLE" ]] && continue
        REPORT+="- **#$num**: $title\n"
        REPORT+="  - Labels: ${labels:-none}\n"
    done < /tmp/triage_categories_$$
    REPORT+="\n"
    REPORT+="**Action:** Run \`.claude/skills/github-issue-triage/scripts/find_closable.sh --close\`\n"
    REPORT+="\n"
fi

# Covered section
if [[ "$COVERED_COUNT" -gt 0 ]]; then
    REPORT+="## Covered Issues (In Progress)\n"
    REPORT+="\n"
    REPORT+="These issues have design docs and are being worked on:\n"
    REPORT+="\n"
    while IFS='|' read -r cat num title labels doc; do
        [[ "$cat" != "COVERED" ]] && continue
        REPORT+="- **#$num**: $title\n"
        REPORT+="  - Design doc: \`$doc\`\n"
    done < /tmp/triage_categories_$$
    REPORT+="\n"
fi

# Orphaned section
if [[ "$ORPHANED_COUNT" -gt 0 ]]; then
    REPORT+="## Orphaned Issues (Need Attention)\n"
    REPORT+="\n"
    REPORT+="These issues have no design doc coverage:\n"
    REPORT+="\n"
    while IFS='|' read -r cat num title labels url; do
        [[ "$cat" != "ORPHANED" ]] && continue
        REPORT+="- **#$num**: $title\n"
        REPORT+="  - Labels: ${labels:-none}\n"
    done < /tmp/triage_categories_$$
    REPORT+="\n"
    REPORT+="**Action:** Create design docs or close with 'won't fix'\n"
    REPORT+="\n"
fi

# Stale section
if [[ "$STALE_COUNT" -gt 0 ]]; then
    REPORT+="## Stale Issues\n"
    REPORT+="\n"
    REPORT+="No activity in ${STALE_DAYS}+ days:\n"
    REPORT+="\n"
    while IFS='|' read -r cat num title labels updated; do
        [[ "$cat" != "STALE" ]] && continue
        REPORT+="- **#$num**: $title\n"
        REPORT+="  - Last updated: ${updated%%T*}\n"
    done < /tmp/triage_categories_$$
    REPORT+="\n"
    REPORT+="**Action:** Request update from reporter or close if inactive\n"
    REPORT+="\n"
fi

# Recommendations
REPORT+="## Recommendations\n"
REPORT+="\n"

if [[ "$CLOSABLE_COUNT" -gt 0 ]]; then
    REPORT+="1. **Close $CLOSABLE_COUNT implemented issues** - These are done and just need closing\n"
fi

if [[ "$ORPHANED_COUNT" -gt 0 ]]; then
    REPORT+="2. **Triage $ORPHANED_COUNT orphaned issues** - Create design docs or close\n"
fi

if [[ "$STALE_COUNT" -gt 0 ]]; then
    REPORT+="3. **Review $STALE_COUNT stale issues** - Request updates or close\n"
fi

REPORT+="\n"
REPORT+="---\n"
REPORT+="*Generated by github-issue-triage skill*\n"

# Clean up temp file
rm -f /tmp/triage_categories_$$

# Output
if [[ -n "$OUTPUT_FILE" ]]; then
    echo -e "$REPORT" > "$OUTPUT_FILE"
    echo "Report written to: $OUTPUT_FILE"
else
    echo -e "$REPORT"
fi
