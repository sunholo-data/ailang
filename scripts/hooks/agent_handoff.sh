#!/bin/bash
# agent_handoff.sh - Claude Code Stop hook → Agent Protocol Bridge
#
# This script is called by Claude Code when a session stops.
# It detects design docs created during the session and hands them off
# to the sprint-planner agent for implementation.
#
# Hook configuration (.claude/hooks.json):
# {
#   "hooks": {
#     "Stop": {
#       "command": "scripts/hooks/agent_handoff.sh",
#       "timeout": 30
#     }
#   }
# }
#
# Input:
#   stdin - JSON payload from Claude Code (hook event data)
#
# Environment variables:
#   STATE_DIR - Agent protocol state directory (default: .ailang/state)

set -euo pipefail

# Read hook JSON from stdin (Claude Code sends hook data via stdin, not env var)
HOOK_JSON=$(cat || echo "{}")

# Configuration
# Use home directory by default (where ailang CLI stores state)
DEFAULT_STATE_DIR="${HOME}/.ailang/state"
STATE_DIR="${STATE_DIR:-$DEFAULT_STATE_DIR}"
DESIGN_DOCS_DIR="design_docs/planned"
TARGET_AGENT="sprint-planner"
LOG_FILE="${STATE_DIR}/hooks.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] [Stop] $*" >> "$LOG_FILE"
}

log "=== Stop Hook (Agent Handoff) Started ==="

# Parse Claude hook JSON from stdin
log "Received hook event: $HOOK_JSON"

# Extract session info
SESSION_ID=$(echo "$HOOK_JSON" | jq -r '.sessionId // "unknown"')
USER_ID=$(echo "$HOOK_JSON" | jq -r '.userId // "unknown"')
EVENT_TYPE=$(echo "$HOOK_JSON" | jq -r '.event // "Stop"')

log "Session ID: $SESSION_ID"
log "User ID: $USER_ID"
log "Event: $EVENT_TYPE"

# Check if ailang binary is available
if ! command -v ailang &> /dev/null; then
    log "ERROR: ailang binary not found in PATH"
    echo "ERROR: ailang binary not found. Run 'make install' first." >&2
    exit 1
fi

# Detect design docs created/modified in this session
# Look for files in design_docs/planned/ modified in the last 5 minutes
log "Scanning for recent design docs in $DESIGN_DOCS_DIR..."

if [ ! -d "$DESIGN_DOCS_DIR" ]; then
    log "No design docs directory found, skipping handoff"
    exit 0
fi

# Find design docs modified in the last 5 minutes
RECENT_DOCS=()
while IFS= read -r -d '' file; do
    RECENT_DOCS+=("$file")
    log "Found recent design doc: $file"
done < <(find "$DESIGN_DOCS_DIR" -type f -name "*.md" -mmin -5 -print0)

if [ ${#RECENT_DOCS[@]} -eq 0 ]; then
    log "No recent design docs found, skipping handoff"
    exit 0
fi

log "Found ${#RECENT_DOCS[@]} recent design doc(s), preparing handoff..."

# For each design doc, create an artifact and send a message
for doc in "${RECENT_DOCS[@]}"; do
    log "Processing design doc: $doc"

    # Extract document title from first # heading
    DOC_TITLE=$(grep -m 1 '^#' "$doc" | sed 's/^#* *//' || echo "$(basename "$doc")")
    log "Document title: $DOC_TITLE"

    # Compute artifact hash (we'll let the Go code store it)
    DOC_HASH=$(ailang debug hash "$doc" 2>/dev/null || echo "unknown")
    log "Document hash: $DOC_HASH"

    # Build message payload
    PAYLOAD=$(cat <<EOF
{
  "task": "implement_design_doc",
  "event": {
    "session_id": "$SESSION_ID",
    "user_id": "$USER_ID",
    "event": "$EVENT_TYPE",
    "provider": "claude-code",
    "notes": "User stopped session after creating design doc"
  },
  "artifacts": [
    {
      "path": "$doc",
      "title": "$DOC_TITLE"
    }
  ]
}
EOF
)

    log "Sending message to $TARGET_AGENT..."

    # Send message using ailang CLI
    if ailang agent send "$TARGET_AGENT" "$PAYLOAD" >> "$LOG_FILE" 2>&1; then
        log "✓ Successfully sent message to $TARGET_AGENT for $doc"
    else
        log "ERROR: Failed to send message to $TARGET_AGENT for $doc"
    fi
done

log "=== Stop Hook (Agent Handoff) Completed ==="
exit 0
