#!/usr/bin/env bash
# check_messages.sh - Check claude-code inbox for messages from agents
#
# Usage:
#   bash .claude/skills/agent-inbox/scripts/check_messages.sh [inbox_dir]
#
# This script checks for pending messages from autonomous AILANG agents
# and displays them with formatted output.

set -euo pipefail

# Configuration
INBOX_DIR="${1:-.ailang/state/messages/claude-code}"
PROCESSED_DIR="$INBOX_DIR/_processed"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Count messages
MSG_COUNT=$(find "$INBOX_DIR" -maxdepth 1 -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

if [ "$MSG_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✅ No pending messages from agents${NC}"
    exit 0
fi

echo ""
echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════╗${NC}"
printf "${YELLOW}║  📬 You have %-2s message(s) from autonomous agents      ║${NC}\n" "$MSG_COUNT"
echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Display each message
MSG_NUM=0
for MSG_FILE in "$INBOX_DIR"/*.json; do
    [ -e "$MSG_FILE" ] || continue
    MSG_NUM=$((MSG_NUM + 1))

    # Extract metadata
    FROM_AGENT=$(jq -r '.from_agent // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    MSG_ID=$(jq -r '.message_id // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    TIMESTAMP=$(jq -r '.timestamp // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    PAYLOAD=$(jq -c '.payload // {}' "$MSG_FILE" 2>/dev/null || echo "{}")

    echo -e "${BLUE}▶ Message $MSG_NUM/$MSG_COUNT${NC}"
    echo -e "  ${BLUE}From:${NC} $FROM_AGENT"
    echo -e "  ${BLUE}ID:${NC} $MSG_ID"
    echo -e "  ${BLUE}Time:${NC} $TIMESTAMP"
    echo -e "  ${BLUE}Payload:${NC}"
    echo "$PAYLOAD" | jq '.' 2>/dev/null | sed 's/^/    /' || echo "    $PAYLOAD"
    echo ""
done

echo -e "${GREEN}Next steps:${NC}"
echo "  1. Review the messages above"
echo "  2. Take any necessary actions"
echo "  3. Mark as processed:"
echo "     mkdir -p $PROCESSED_DIR"
echo "     mv $INBOX_DIR/*.json $PROCESSED_DIR/"
echo ""
