#!/bin/bash
# claude_telemetry.sh - Claude Code hook for AILANG observatory telemetry
#
# This script is called by Claude Code hooks to report session and tool metadata
# to the AILANG observatory. The data is used to enrich OTEL spans with workspace
# information for better dashboard filtering and hierarchy visualization.
#
# Global hook configuration (~/.claude/settings.json):
# {
#   "hooks": {
#     "SessionStart": [{"hooks": [{"type": "command", "command": "~/.claude/hooks/claude_telemetry.sh", "timeout": 5}]}],
#     "PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": "~/.claude/hooks/claude_telemetry.sh", "timeout": 3}]}],
#     "PostToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": "~/.claude/hooks/claude_telemetry.sh", "timeout": 3}]}],
#     "Stop": [{"hooks": [{"type": "command", "command": "~/.claude/hooks/claude_telemetry.sh", "timeout": 5}]}]
#   }
# }
#
# Input:
#   stdin - JSON payload from Claude Code (hook event data)
#
# Environment:
#   AILANG_OBSERVATORY_URL - Observatory endpoint (default: http://localhost:1957)

set -euo pipefail

# Configuration
OBSERVATORY_URL="${AILANG_OBSERVATORY_URL:-http://localhost:1957}"
HOOKS_ENDPOINT="${OBSERVATORY_URL}/api/observatory/hooks"
LOG_FILE="${HOME}/.ailang/state/telemetry_hooks.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function (silent by default, set DEBUG_TELEMETRY_HOOK=1 to enable)
log() {
    if [ "${DEBUG_TELEMETRY_HOOK:-}" = "1" ]; then
        echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"
    fi
}

# Read hook JSON from stdin
HOOK_JSON=$(cat || echo "{}")

# Extract event name from hook payload
# Claude Code sends hook_event_name in the JSON
EVENT_NAME=$(echo "$HOOK_JSON" | jq -r '.hook_event_name // "unknown"' 2>/dev/null)
if [ "$EVENT_NAME" = "unknown" ] || [ "$EVENT_NAME" = "null" ]; then
    log "No hook_event_name found, skipping"
    exit 0
fi

# Extract common fields
SESSION_ID=$(echo "$HOOK_JSON" | jq -r '.session_id // "unknown"' 2>/dev/null)
WORKSPACE=$(echo "$HOOK_JSON" | jq -r '.cwd // ""' 2>/dev/null)
[ -z "$WORKSPACE" ] && WORKSPACE=$(pwd)

log "Processing $EVENT_NAME event for session $SESSION_ID"

# Build payload based on event type
case "$EVENT_NAME" in
    "SessionStart")
        CLAUDE_VERSION=$(echo "$HOOK_JSON" | jq -r '.version // ""' 2>/dev/null)
        PAYLOAD=$(jq -n \
            --arg e "$EVENT_NAME" \
            --arg s "$SESSION_ID" \
            --arg w "$WORKSPACE" \
            --arg v "$CLAUDE_VERSION" \
            '{
                event: $e,
                session_id: $s,
                workspace: $w,
                claude_version: $v,
                timestamp: (now | todate)
            }')
        ;;

    "PreToolUse")
        TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // "unknown"' 2>/dev/null)
        TOOL_USE_ID=$(echo "$HOOK_JSON" | jq -r '.tool_use_id // ""' 2>/dev/null)
        # Extract tool_input safely with fallback to empty object
        TOOL_INPUT=$(echo "$HOOK_JSON" | jq -c '.tool_input // {}' 2>/dev/null || echo "{}")

        PAYLOAD=$(jq -n \
            --arg e "$EVENT_NAME" \
            --arg s "$SESSION_ID" \
            --arg tn "$TOOL_NAME" \
            --arg ti "$TOOL_USE_ID" \
            --argjson inp "$TOOL_INPUT" \
            '{
                event: $e,
                session_id: $s,
                tool_name: $tn,
                tool_use_id: $ti,
                tool_input: $inp,
                timestamp: (now | todate)
            }')
        ;;

    "PostToolUse")
        TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // "unknown"' 2>/dev/null)
        TOOL_USE_ID=$(echo "$HOOK_JSON" | jq -r '.tool_use_id // ""' 2>/dev/null)
        # Extract tool_response safely - use string if not valid JSON object
        # Truncate at jq level to preserve valid JSON structure
        TOOL_RESPONSE=$(echo "$HOOK_JSON" | jq -c '.tool_response // null | if type == "string" then .[0:10000] else . end' 2>/dev/null || echo "null")

        PAYLOAD=$(jq -n \
            --arg e "$EVENT_NAME" \
            --arg s "$SESSION_ID" \
            --arg tn "$TOOL_NAME" \
            --arg ti "$TOOL_USE_ID" \
            --argjson resp "$TOOL_RESPONSE" \
            '{
                event: $e,
                session_id: $s,
                tool_name: $tn,
                tool_use_id: $ti,
                tool_response: $resp,
                timestamp: (now | todate)
            }')
        ;;

    "Stop")
        PAYLOAD=$(jq -n \
            --arg e "$EVENT_NAME" \
            --arg s "$SESSION_ID" \
            '{
                event: $e,
                session_id: $s,
                timestamp: (now | todate)
            }')
        ;;

    *)
        log "Unknown event type: $EVENT_NAME, skipping"
        exit 0
        ;;
esac

# POST to observatory hooks endpoint
# Use --fail-early and timeout to avoid blocking Claude Code
# Silent failures - we don't want hook errors to affect Claude Code operation
if curl -s -f --max-time 3 -X POST "$HOOKS_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" >/dev/null 2>&1; then
    log "Successfully reported $EVENT_NAME event"
else
    log "Failed to report $EVENT_NAME event (observatory may not be running)"
fi

exit 0
