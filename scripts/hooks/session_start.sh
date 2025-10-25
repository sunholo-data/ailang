#!/bin/bash
# session_start.sh - Claude Code SessionStart hook → Check User Inbox
#
# This script is called by Claude Code when a session starts.
# It checks the user inbox for new messages from autonomous agents
# and displays them to the user.
#
# Hook configuration (.claude/hooks.json):
# {
#   "hooks": {
#     "SessionStart": {
#       "command": "scripts/hooks/session_start.sh",
#       "timeout": 10
#     }
#   }
# }
#
# Environment variables:
#   CLAUDE_HOOK_JSON - JSON payload from Claude Code
#   STATE_DIR - Agent protocol state directory (default: .ailang/state)

set -euo pipefail

# Configuration
# Use home directory by default (where ailang CLI stores state)
DEFAULT_STATE_DIR="${HOME}/.ailang/state"
STATE_DIR="${STATE_DIR:-$DEFAULT_STATE_DIR}"
LOG_FILE="${STATE_DIR}/hooks.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"
}

log "=== Session Start Hook Started ==="

# Parse Claude hook JSON (if provided)
if [ -n "${CLAUDE_HOOK_JSON:-}" ]; then
    SESSION_ID=$(echo "$CLAUDE_HOOK_JSON" | jq -r '.sessionId // "unknown"')
    USER_ID=$(echo "$CLAUDE_HOOK_JSON" | jq -r '.userId // "unknown"')
    log "Session ID: $SESSION_ID"
    log "User ID: $USER_ID"
fi

# Check user inbox for unread messages
INBOX_DIR="$STATE_DIR/messages/inbox/user/_unread"

if [ ! -d "$INBOX_DIR" ]; then
    log "No user inbox found, skipping notification"
    exit 0
fi

# Count unread messages
UNREAD_COUNT=$(find "$INBOX_DIR" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

if [ "$UNREAD_COUNT" -eq 0 ]; then
    log "No unread messages in user inbox"
    exit 0
fi

log "Found $UNREAD_COUNT unread message(s) in user inbox"

# Display notification to user
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  📬 You have $UNREAD_COUNT unread message(s) from agents      ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "To view messages, run:"
echo "  ailang agent inbox user"
echo ""

# Optionally display a preview of the first message
FIRST_MSG=$(find "$INBOX_DIR" -name "*.json" -type f 2>/dev/null | head -1)

if [ -n "$FIRST_MSG" ]; then
    FROM_AGENT=$(jq -r '.from_agent // "unknown"' "$FIRST_MSG" 2>/dev/null || echo "unknown")
    MSG_ID=$(jq -r '.message_id // "unknown"' "$FIRST_MSG" 2>/dev/null || echo "unknown")

    echo "Preview (most recent):"
    echo "  From: $FROM_AGENT"
    echo "  Message ID: $MSG_ID"
    echo ""
fi

log "=== Session Start Hook Completed ==="
exit 0
