#!/bin/bash
# session_start.sh - Claude Code SessionStart hook → Check User Inbox
#
# This script is called by Claude Code when a session starts.
# It checks the user inbox for new messages from autonomous agents
# and exports them as environment variables for Claude to see.
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
# Input:
#   stdin - JSON payload from Claude Code (hook event data)
#
# Environment variables:
#   CLAUDE_ENV_FILE - File to write environment variables to (provided by Claude Code)
#   STATE_DIR - Agent protocol state directory (default: .ailang/state)

set -euo pipefail

# Read hook JSON from stdin (Claude Code sends hook data via stdin, not env var)
HOOK_JSON=$(cat || echo "{}")

# Configuration
# Use home directory by default (where ailang CLI stores state)
DEFAULT_STATE_DIR="${HOME}/.ailang/state"
STATE_DIR="${STATE_DIR:-$DEFAULT_STATE_DIR}"
LOG_FILE="${STATE_DIR}/hooks.log"
LOCK_FILE="${STATE_DIR}/session_start.lock"

# Detect project root (where .claude/ directory is)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"
}

log "=== Session Start Hook Started ==="

# Prevent duplicate execution within 3 seconds
# Claude Code seems to run SessionStart hook twice on session start
if [ -f "$LOCK_FILE" ]; then
    LOCK_AGE=$(($(date +%s) - $(stat -f %m "$LOCK_FILE" 2>/dev/null || stat -c %Y "$LOCK_FILE" 2>/dev/null || echo 0)))
    if [ "$LOCK_AGE" -lt 3 ]; then
        log "Skipping duplicate execution (lock file age: ${LOCK_AGE}s)"
        # Output empty context to avoid breaking Claude Code
        jq -n \
          --arg context "📭 Agent inbox: Already checked in this session." \
          '{
            "hookSpecificOutput": {
              "hookEventName": "SessionStart",
              "additionalContext": $context
            }
          }'
        exit 0
    fi
fi

# Create lock file
touch "$LOCK_FILE"
log "Created lock file to prevent duplicate execution"

# Parse Claude hook JSON from stdin
SESSION_ID=$(echo "$HOOK_JSON" | jq -r '.sessionId // "unknown"')
USER_ID=$(echo "$HOOK_JSON" | jq -r '.userId // "unknown"')
log "Session ID: $SESSION_ID"
log "User ID: $USER_ID"

# Check TWO inbox locations:
# 1. Home directory: ~/.ailang/state/messages/inbox/user/_unread/
# 2. Project directory: <project>/.ailang/state/messages/claude-code/

INBOX_DIRS=(
    "$STATE_DIR/messages/inbox/user/_unread"
    "$PROJECT_ROOT/.ailang/state/messages/claude-code"
)

# Collect all unread messages from both locations
declare -a ALL_MESSAGES=()

for INBOX_DIR in "${INBOX_DIRS[@]}"; do
    if [ -d "$INBOX_DIR" ]; then
        while IFS= read -r -d '' MSG_FILE; do
            ALL_MESSAGES+=("$MSG_FILE")
        done < <(find "$INBOX_DIR" -maxdepth 1 -name "*.json" -type f -print0 2>/dev/null)
    fi
done

# Count total unread messages
UNREAD_COUNT="${#ALL_MESSAGES[@]}"

# Function to check for active sprint and return context
get_sprint_context() {
    local SPRINT_DIR="$PROJECT_ROOT/.ailang/state/sprints"
    local SPRINT_CONTEXT=""

    if [ -d "$SPRINT_DIR" ]; then
        for SPRINT_FILE in "$SPRINT_DIR"/sprint_*.json; do
            if [ -f "$SPRINT_FILE" ] && grep -q '"status": "in_progress"' "$SPRINT_FILE" 2>/dev/null; then
                local SPRINT_ID=$(jq -r '.sprint_id // "unknown"' "$SPRINT_FILE" 2>/dev/null)
                local DESIGN_DOC=$(jq -r '.design_doc // ""' "$SPRINT_FILE" 2>/dev/null)
                local TOTAL_MILESTONES=$(jq '.features | length' "$SPRINT_FILE" 2>/dev/null || echo 0)
                local COMPLETED=$(jq '[.features[] | select(.passes == true)] | length' "$SPRINT_FILE" 2>/dev/null || echo 0)
                local NEXT_MILESTONE=$(jq -r '.features[] | select(.passes == null) | .id' "$SPRINT_FILE" 2>/dev/null | head -1)
                local NEXT_DESCRIPTION=$(jq -r --arg id "$NEXT_MILESTONE" '.features[] | select(.id == $id) | .description' "$SPRINT_FILE" 2>/dev/null)
                local NEXT_CRITERIA=$(jq -r --arg id "$NEXT_MILESTONE" '.features[] | select(.id == $id) | .acceptance_criteria // [] | .[]' "$SPRINT_FILE" 2>/dev/null | head -5)

                if [ -n "$NEXT_MILESTONE" ]; then
                    SPRINT_CONTEXT="
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🏃 ACTIVE SPRINT: $SPRINT_ID
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Progress: $COMPLETED/$TOTAL_MILESTONES milestones complete
Design doc: $DESIGN_DOC
JSON file: $SPRINT_FILE

▶ NEXT MILESTONE: $NEXT_MILESTONE
  $NEXT_DESCRIPTION

Acceptance criteria:
$(echo "$NEXT_CRITERIA" | sed 's/^/  • /')

💡 Use this milestone's criteria to guide your work.
   Run checkpoint when done: .claude/skills/sprint-executor/scripts/milestone_checkpoint.sh $NEXT_MILESTONE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                else
                    SPRINT_CONTEXT="
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🏃 ACTIVE SPRINT: $SPRINT_ID (ALL MILESTONES COMPLETE!)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Progress: $COMPLETED/$TOTAL_MILESTONES milestones complete ✅

💡 Sprint appears complete. Consider:
   1. Update JSON status to \"completed\"
   2. Final commit with sprint summary
   3. Move design doc to implemented/
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                fi
                log "Found active sprint: $SPRINT_ID ($COMPLETED/$TOTAL_MILESTONES complete)"
                break
            fi
        done
    fi
    echo "$SPRINT_CONTEXT"
}

if [ "$UNREAD_COUNT" -eq 0 ]; then
    log "No unread messages in any inbox location"

    # Export empty message indicator to Claude Code environment
    if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
        echo "export AGENT_INBOX_COUNT=0" >> "$CLAUDE_ENV_FILE"
        log "Exported AGENT_INBOX_COUNT=0 to CLAUDE_ENV_FILE"
    fi

    # Check for active sprint even when no inbox messages
    SPRINT_CONTEXT=$(get_sprint_context)

    # Output context (will appear in system reminders)
    if [ -n "$SPRINT_CONTEXT" ]; then
        echo "📭 Agent inbox: No unread messages from autonomous agents."
        echo "$SPRINT_CONTEXT"
    else
        echo "📭 Agent inbox: No unread messages from autonomous agents."
    fi

    log "Outputted context to Claude"
    exit 0
fi

log "Found $UNREAD_COUNT unread message(s) across all inbox locations"

# Export message count to Claude Code environment
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    echo "export AGENT_INBOX_COUNT=$UNREAD_COUNT" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_COUNT=$UNREAD_COUNT to CLAUDE_ENV_FILE"
fi

# Build a summary of all messages as a JSON array
MESSAGES_JSON="["
FIRST=true

# Process each unread message and collect into JSON array
for MSG_FILE in "${ALL_MESSAGES[@]}"; do
    FROM_AGENT=$(jq -r '.from_agent // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    TIMESTAMP=$(jq -r '.timestamp // "unknown"' "$MSG_FILE" 2>/dev/null || echo "unknown")
    PAYLOAD=$(jq -c '.payload // {}' "$MSG_FILE" 2>/dev/null || echo '{}')

    # Add message to JSON array
    if [ "$FIRST" = true ]; then
        FIRST=false
    else
        MESSAGES_JSON+=","
    fi

    # Escape quotes in payload for safe JSON embedding
    PAYLOAD_ESCAPED=$(echo "$PAYLOAD" | sed 's/"/\\"/g')

    MESSAGES_JSON+="{\"from\":\"$FROM_AGENT\",\"timestamp\":\"$TIMESTAMP\",\"payload\":$PAYLOAD,\"file\":\"$MSG_FILE\"}"

    log "Collected message from $FROM_AGENT (source: $MSG_FILE)"

    # NOTE: Messages are NOT marked as read automatically
    # Claude Code must explicitly acknowledge them using: ailang agent ack <message-id>
    # This prevents:
    # 1. Messages being consumed before Claude sees them
    # 2. Race conditions in multi-session scenarios
    # 3. Loss of messages if context injection fails
done

MESSAGES_JSON+="]"

# Export messages as environment variable (base64 encoded to handle special characters)
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    MESSAGES_B64=$(echo "$MESSAGES_JSON" | base64)
    echo "export AGENT_INBOX_MESSAGES='$MESSAGES_B64'" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_MESSAGES (base64 encoded JSON) to CLAUDE_ENV_FILE"
fi

# Get sprint context using the function defined earlier
SPRINT_CONTEXT=$(get_sprint_context)

# Build formatted context string for Claude
CONTEXT_MESSAGE=$(cat <<EOF

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📬 AGENT INBOX: $UNREAD_COUNT unread message(s) from autonomous agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

$(echo "$MESSAGES_JSON" | jq -r '.[] | "From: \(.from)\nTime: \(.timestamp)\nMessage: \(.payload | tojson)\n"')
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 Review the messages above and decide if any action is needed.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$SPRINT_CONTEXT
EOF
)

log "Context message prepared (length: ${#CONTEXT_MESSAGE} chars)"

# Output plain text to stdout
# Note: This appears in system reminders but may have length limits
# Messages are NOT marked as read - Claude must acknowledge them with: ailang agent ack <message-id>
echo "$CONTEXT_MESSAGE"

log "=== Session Start Hook Completed ==="
exit 0
