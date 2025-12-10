#!/bin/bash
# session_start.sh - Claude Code SessionStart hook → Check User Inbox
#
# This script is called by Claude Code when a session starts.
# It checks the user inbox for new messages from autonomous agents
# using the ailang messages CLI (backed by unified collaboration.db).
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

set -euo pipefail

# Read hook JSON from stdin (Claude Code sends hook data via stdin, not env var)
HOOK_JSON=$(cat || echo "{}")

# Configuration
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

# Get current version from CHANGELOG.md (most reliable source)
get_current_version() {
    local CHANGELOG="$PROJECT_ROOT/CHANGELOG.md"
    if [ -f "$CHANGELOG" ]; then
        grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' "$CHANGELOG" | head -1
    else
        echo "unknown"
    fi
}

CURRENT_VERSION=$(get_current_version)
log "Current AILANG version: $CURRENT_VERSION"

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

# Import GitHub issues as messages (respects auto_import config setting)
# This is silent if disabled or if there are no new issues
if ailang messages import-github 2>/dev/null; then
    log "GitHub import check completed"
else
    log "GitHub import check skipped or failed (this is normal if not configured)"
fi

# Use ailang messages CLI to get unread messages (SQLite-backed)
# The CLI handles all inbox types via the unified collaboration.db
MESSAGES_JSON=$(ailang messages list --unread --json 2>/dev/null || echo "[]")
UNREAD_COUNT=$(echo "$MESSAGES_JSON" | jq 'length' 2>/dev/null || echo "0")

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
    log "No unread messages"

    # Export empty message indicator to Claude Code environment
    if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
        echo "export AGENT_INBOX_COUNT=0" >> "$CLAUDE_ENV_FILE"
        log "Exported AGENT_INBOX_COUNT=0 to CLAUDE_ENV_FILE"
    fi

    # Check for active sprint even when no inbox messages
    SPRINT_CONTEXT=$(get_sprint_context)

    # Output context (will appear in system reminders)
    echo "📦 AILANG $CURRENT_VERSION"
    if [ -n "$SPRINT_CONTEXT" ]; then
        echo "📭 Agent inbox: No unread messages from autonomous agents."
        echo "$SPRINT_CONTEXT"
    else
        echo "📭 Agent inbox: No unread messages from autonomous agents."
    fi

    log "Outputted context to Claude"
    exit 0
fi

log "Found $UNREAD_COUNT unread message(s)"

# Export message count to Claude Code environment
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    echo "export AGENT_INBOX_COUNT=$UNREAD_COUNT" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_COUNT=$UNREAD_COUNT to CLAUDE_ENV_FILE"
fi

# Export messages as environment variable (base64 encoded to handle special characters)
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    MESSAGES_B64=$(echo "$MESSAGES_JSON" | base64)
    echo "export AGENT_INBOX_MESSAGES='$MESSAGES_B64'" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_MESSAGES (base64 encoded JSON) to CLAUDE_ENV_FILE"
fi

# Get sprint context using the function defined earlier
SPRINT_CONTEXT=$(get_sprint_context)

# Build formatted context string for Claude
# The MESSAGES_JSON comes from the CLI and has format:
# [{"id":"msg_xxx","from_agent":"test","to_inbox":"user","title":"Title","payload":"...","status":"unread","created_at":"..."}]
CONTEXT_MESSAGE=$(cat <<EOF
📦 AILANG $CURRENT_VERSION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📬 AGENT INBOX: $UNREAD_COUNT unread message(s) from autonomous agents
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

$(echo "$MESSAGES_JSON" | jq -r '.[] | "ID: \(.id)\nFrom: \(.from_agent)\nTitle: \(.title)\nTime: \(.created_at)\n"')
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 Use 'ailang messages read <id>' to view full content
   Use 'ailang messages ack <id>' to mark as read
   Use 'ailang messages ack --all' to mark all as read
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$SPRINT_CONTEXT
EOF
)

log "Context message prepared (length: ${#CONTEXT_MESSAGE} chars)"

# Output plain text to stdout
# Note: This appears in system reminders but may have length limits
# Messages are NOT marked as read - Claude must acknowledge them with: ailang messages ack <id>
echo "$CONTEXT_MESSAGE"

log "=== Session Start Hook Completed ==="
exit 0
