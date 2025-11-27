#!/bin/bash
#
# Session Start Script for Sprint Executor
#
# Purpose: Resume a sprint across multiple Claude Code sessions
# Based on: https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
#
# Usage:
#   .claude/skills/sprint-executor/scripts/session_start.sh <sprint_id>
#
# Example:
#   .claude/skills/sprint-executor/scripts/session_start.sh M-S1
#
# This script implements the "Session Startup Routine" pattern from the Anthropic article:
# 1. Check pwd (working directory)
# 2. Read progress JSON file
# 3. Review git log (recent commits)
# 4. Run tests to verify clean state
# 5. Print "Here's where we left off" summary

set -e  # Exit on error

# Check arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <sprint_id>"
    echo "Example: $0 M-S1"
    exit 1
fi

SPRINT_ID="$1"
PROGRESS_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "═══════════════════════════════════════════════════════════════"
echo " Sprint Continuation Check - ${SPRINT_ID}"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# 1. Check working directory
echo -e "${BLUE}1. Working Directory${NC}"
echo "   $(pwd)"
echo ""

# 2. Check if progress file exists
if [ ! -f "$PROGRESS_FILE" ]; then
    echo -e "${RED}✗ Progress file not found: $PROGRESS_FILE${NC}"
    echo ""
    echo "Options:"
    echo "  1. If this is a new sprint, run sprint-planner first"
    echo "  2. If continuing an old sprint, check the sprint ID is correct"
    echo "  3. If migrating from markdown-only sprint, see migration notes"
    exit 1
fi

echo -e "${GREEN}✓ Found progress file: $PROGRESS_FILE${NC}"
echo ""

# 3. Load sprint metadata
echo -e "${BLUE}2. Sprint Metadata${NC}"
SPRINT_STATUS=$(jq -r '.status' "$PROGRESS_FILE")
CREATED=$(jq -r '.created' "$PROGRESS_FILE")
LAST_SESSION=$(jq -r '.last_session' "$PROGRESS_FILE")
LAST_CHECKPOINT=$(jq -r '.last_checkpoint // "Not set"' "$PROGRESS_FILE")

echo "   Status: $SPRINT_STATUS"
echo "   Created: $CREATED"
echo "   Last session: $LAST_SESSION"
echo "   Last checkpoint: $LAST_CHECKPOINT"
echo ""

# 4. Feature progress summary
echo -e "${BLUE}3. Feature Progress${NC}"
TOTAL_FEATURES=$(jq '.features | length' "$PROGRESS_FILE")
COMPLETE_FEATURES=$(jq '[.features[] | select(.passes == true)] | length' "$PROGRESS_FILE")
FAILED_FEATURES=$(jq '[.features[] | select(.passes == false)] | length' "$PROGRESS_FILE")
IN_PROGRESS=$(jq '[.features[] | select(.passes == null and .started != null)] | length' "$PROGRESS_FILE")
NOT_STARTED=$(jq '[.features[] | select(.started == null)] | length' "$PROGRESS_FILE")

echo "   Total: $TOTAL_FEATURES features"
echo -e "   ${GREEN}✓ Complete: $COMPLETE_FEATURES${NC}"
if [ "$FAILED_FEATURES" -gt 0 ]; then
    echo -e "   ${RED}✗ Failed: $FAILED_FEATURES${NC}"
fi
if [ "$IN_PROGRESS" -gt 0 ]; then
    echo -e "   ${YELLOW}⟳ In progress: $IN_PROGRESS${NC}"
fi
if [ "$NOT_STARTED" -gt 0 ]; then
    echo "   ○ Not started: $NOT_STARTED"
fi
echo ""

# 5. Show completed features
if [ "$COMPLETE_FEATURES" -gt 0 ]; then
    echo -e "${GREEN}Completed Features:${NC}"
    jq -r '.features[] | select(.passes == true) | "  ✓ \(.id): \(.description) (\(.actual_loc) LOC)"' "$PROGRESS_FILE"
    echo ""
fi

# 6. Show failed features
if [ "$FAILED_FEATURES" -gt 0 ]; then
    echo -e "${RED}Failed Features:${NC}"
    jq -r '.features[] | select(.passes == false) | "  ✗ \(.id): \(.description) - \(.notes // "No notes")"' "$PROGRESS_FILE"
    echo ""
fi

# 7. Show in-progress features
if [ "$IN_PROGRESS" -gt 0 ]; then
    echo -e "${YELLOW}In Progress:${NC}"
    jq -r '.features[] | select(.passes == null and .started != null) | "  ⟳ \(.id): \(.description)\n    Status: \(.notes // "No notes")"' "$PROGRESS_FILE"
    echo ""
fi

# 8. Show next feature to work on
if [ "$NOT_STARTED" -gt 0 ]; then
    echo -e "${BLUE}Next Feature:${NC}"
    # Find first feature with no dependencies or all dependencies complete
    NEXT_FEATURE=$(jq -r '
        .features[] |
        select(.started == null) |
        select(
            (.dependencies | length) == 0 or
            all(.dependencies[]; . as $dep | any(.features[]; .id == $dep and .passes == true))
        ) |
        "  → \(.id): \(.description) (estimated: \(.estimated_loc) LOC)" |
        @text
    ' "$PROGRESS_FILE" | head -1)

    if [ -n "$NEXT_FEATURE" ]; then
        echo "$NEXT_FEATURE"
    else
        echo "  → Check dependencies - some features may be blocked"
    fi
    echo ""
fi

# 9. Velocity metrics
echo -e "${BLUE}4. Velocity Metrics${NC}"
TARGET_LOC=$(jq -r '.velocity.target_loc_per_day' "$PROGRESS_FILE")
ACTUAL_LOC=$(jq -r '.velocity.actual_loc_per_day' "$PROGRESS_FILE")
ESTIMATED_TOTAL=$(jq -r '.velocity.estimated_total_loc' "$PROGRESS_FILE")
ACTUAL_TOTAL=$(jq -r '.velocity.actual_total_loc' "$PROGRESS_FILE")
ESTIMATED_DAYS=$(jq -r '.velocity.estimated_days' "$PROGRESS_FILE")
ACTUAL_DAYS=$(jq -r '.velocity.actual_days // 0' "$PROGRESS_FILE")

echo "   Target: ${TARGET_LOC} LOC/day"
echo "   Actual: ${ACTUAL_LOC} LOC/day"
echo "   Progress: ${ACTUAL_TOTAL}/${ESTIMATED_TOTAL} LOC ($(echo "scale=1; $ACTUAL_TOTAL * 100 / $ESTIMATED_TOTAL" | bc)%)"
echo "   Days: ${ACTUAL_DAYS}/${ESTIMATED_DAYS}"
echo ""

# 10. Recent git commits
echo -e "${BLUE}5. Recent Work (Last 3 Commits)${NC}"
git log --oneline -3 --color=always | sed 's/^/   /'
echo ""

# 11. Git status
echo -e "${BLUE}6. Working Directory Status${NC}"
if ! git diff-index --quiet HEAD --; then
    echo -e "   ${YELLOW}⚠  Uncommitted changes detected${NC}"
    git status --short | sed 's/^/   /'
else
    echo -e "   ${GREEN}✓ Working directory clean${NC}"
fi
echo ""

# 12. Run tests to verify clean state
echo -e "${BLUE}7. Pre-Session Validation${NC}"
echo "   Running tests..."

if make test 2>&1 | grep -q "FAIL"; then
    echo -e "   ${RED}✗ Tests failing!${NC}"
    echo ""
    echo "⚠️  You should fix tests before continuing the sprint."
    echo "   Run 'make test' to see details."
    exit 1
else
    echo -e "   ${GREEN}✓ All tests pass${NC}"
fi
echo ""

# 13. Summary and recommendations
echo "═══════════════════════════════════════════════════════════════"
echo " Summary"
echo "═══════════════════════════════════════════════════════════════"
echo ""

if [ "$SPRINT_STATUS" = "completed" ]; then
    echo -e "${GREEN}🎉 This sprint is complete!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review the completed work"
    echo "  2. Create git tag if this is a release"
    echo "  3. Move design docs to implemented/"
    echo "  4. Start planning next sprint"
elif [ "$SPRINT_STATUS" = "paused" ]; then
    echo -e "${YELLOW}⏸  Sprint is paused${NC}"
    echo ""
    echo "Last checkpoint: $LAST_CHECKPOINT"
    echo ""
    echo "To resume:"
    echo "  1. Review the in-progress feature above"
    echo "  2. Continue implementation from where you left off"
    echo "  3. Update JSON progress file as you complete milestones"
elif [ "$SPRINT_STATUS" = "in_progress" ]; then
    echo -e "${BLUE}▶  Sprint in progress${NC}"
    echo ""
    echo "Current progress: $COMPLETE_FEATURES/$TOTAL_FEATURES features complete"
    echo ""
    if [ "$IN_PROGRESS" -gt 0 ]; then
        echo "Continue working on in-progress features first."
    elif [ "$NOT_STARTED" -gt 0 ]; then
        echo "Start working on the next feature listed above."
    fi
else
    echo -e "${BLUE}Ready to start sprint${NC}"
    echo ""
    echo "Begin with the first feature listed above."
fi

echo ""
echo "Progress file: $PROGRESS_FILE"
echo "═══════════════════════════════════════════════════════════════"

exit 0
