#!/usr/bin/env bash
# check_messages.sh - Check inbox for messages from agents
#
# Usage:
#   bash .claude/skills/agent-inbox/scripts/check_messages.sh
#
# This script uses the ailang messages CLI to check for unread messages
# from autonomous AILANG agents and display them with formatted output.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if ailang is available
if ! command -v ailang &> /dev/null; then
    echo -e "${RED}Error: ailang CLI not found${NC}"
    echo "Install with: go install github.com/sunholo-data/ailang/cmd/ailang@latest"
    exit 1
fi

# Get unread message count using new CLI
MSG_JSON=$(ailang messages list --unread --json 2>/dev/null || echo "[]")
MSG_COUNT=$(echo "$MSG_JSON" | jq 'length' 2>/dev/null || echo "0")

if [ "$MSG_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✅ No unread messages from agents${NC}"
    exit 0
fi

echo ""
echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════╗${NC}"
printf "${YELLOW}║  📬 You have %-2s unread message(s) from agents           ║${NC}\n" "$MSG_COUNT"
echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Display each message
MSG_NUM=0
echo "$MSG_JSON" | jq -c '.[]' | while read -r MSG; do
    MSG_NUM=$((MSG_NUM + 1))

    # Extract metadata
    MSG_ID=$(echo "$MSG" | jq -r '.message_id // "unknown"')
    FROM_AGENT=$(echo "$MSG" | jq -r '.from_agent // "unknown"')
    TITLE=$(echo "$MSG" | jq -r '.title // "No title"')
    CREATED_AT=$(echo "$MSG" | jq -r '.created_at // "unknown"')
    CATEGORY=$(echo "$MSG" | jq -r '.category // ""')
    GITHUB_ISSUE=$(echo "$MSG" | jq -r '.github_issue // ""')

    echo -e "${BLUE}▶ Message $MSG_NUM${NC}"
    echo -e "  ${BLUE}ID:${NC} $MSG_ID"
    echo -e "  ${BLUE}From:${NC} $FROM_AGENT"
    echo -e "  ${BLUE}Title:${NC} $TITLE"
    echo -e "  ${BLUE}Time:${NC} $CREATED_AT"

    if [ -n "$CATEGORY" ] && [ "$CATEGORY" != "null" ]; then
        echo -e "  ${BLUE}Category:${NC} $CATEGORY"
    fi

    if [ -n "$GITHUB_ISSUE" ] && [ "$GITHUB_ISSUE" != "null" ]; then
        echo -e "  ${BLUE}GitHub Issue:${NC} #$GITHUB_ISSUE"
    fi

    echo ""
done

echo -e "${GREEN}Next steps:${NC}"
echo "  1. Read full message:"
echo "     ailang messages read MSG_ID"
echo "  2. Acknowledge after processing:"
echo "     ailang messages ack MSG_ID"
echo "     ailang messages ack --all"
echo ""
