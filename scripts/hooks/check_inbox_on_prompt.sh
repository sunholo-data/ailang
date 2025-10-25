#!/bin/bash
# check_inbox_on_prompt.sh - Testing hook to inject inbox messages on UserPromptSubmit
#
# This is a TEMPORARY hook for testing if hookSpecificOutput.additionalContext works.
# Once verified, this should be REMOVED and only SessionStart hook should be used.

set -euo pipefail

# Configuration
DEFAULT_STATE_DIR="${HOME}/.ailang/state"
STATE_DIR="${STATE_DIR:-$DEFAULT_STATE_DIR}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOG_FILE="${STATE_DIR}/hooks.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] [UserPromptSubmit] $*" >> "$LOG_FILE"
}

log "=== UserPromptSubmit Hook Started ==="

# Check inbox locations (same as session_start.sh)
INBOX_DIRS=(
    "$STATE_DIR/messages/inbox/user/_unread"
    "$PROJECT_ROOT/.ailang/state/messages/claude-code"
)

# Collect all unread messages
declare -a ALL_MESSAGES=()

for INBOX_DIR in "${INBOX_DIRS[@]}"; do
    if [ -d "$INBOX_DIR" ]; then
        while IFS= read -r -d '' MSG_FILE; do
            ALL_MESSAGES+=("$MSG_FILE")
        done < <(find "$INBOX_DIR" -maxdepth 1 -name "*.json" -type f -print0 2>/dev/null)
    fi
done

UNREAD_COUNT="${#ALL_MESSAGES[@]}"

if [ "$UNREAD_COUNT" -eq 0 ]; then
    log "No unread messages found"
    # No messages - output plain text
    echo "📭 [Inbox check: No unread agent messages]"
    log "Outputted 'no messages' context to Claude"
    exit 0
fi

log "Found $UNREAD_COUNT unread message(s)"

# Build messages JSON array (same as session_start.sh)
MESSAGES_JSON="["
FIRST=true

for MSG_FILE in "${ALL_MESSAGES[@]}"; do
    FROM_AGENT=$(jq -r '.from_agent // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    TIMESTAMP=$(jq -r '.timestamp // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    PAYLOAD=$(jq -c '.payload // {}' "$MSG_FILE" 2>/dev/null || echo '{}')

    if [ "$FIRST" = true ]; then
        FIRST=false
    else
        MESSAGES_JSON+=","
    fi

    MESSAGES_JSON+="{\"from\":\"$FROM_AGENT\",\"timestamp\":\"$TIMESTAMP\",\"payload\":$PAYLOAD}"
done

MESSAGES_JSON+="]"

# Build formatted context string
CONTEXT_MESSAGE=$(cat <<EOF

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📬 AGENT INBOX: $UNREAD_COUNT unread message(s) from autonomous agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

$(echo "$MESSAGES_JSON" | jq -r '.[] | "From: \(.from)\nTime: \(.timestamp)\nMessage: \(.payload | tojson)\n"')
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 Review the messages above and decide if any action is needed.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

EOF
)

# Try plain text output instead of JSON (per Claude Code docs, both should work)
echo "$CONTEXT_MESSAGE"

log "Context message prepared (length: ${#CONTEXT_MESSAGE} chars)"
log "=== UserPromptSubmit Hook Completed ==="
exit 0
