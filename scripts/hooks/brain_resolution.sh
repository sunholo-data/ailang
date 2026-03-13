#!/bin/bash
# brain_resolution.sh - Capture git commit resolutions into the AILANG brain
#
# PostToolUse hook for Bash — detects git commit commands and stores
# resolution frames automatically. Runs async, <200ms budget.
#
# Claude Code PostToolUse hook sends JSON on stdin with:
#   { "tool_name": "Bash", "tool_input": { "command": "..." }, "tool_output": "..." }

set -euo pipefail

# Check if brain hooks are disabled
if [ "${AILANG_BRAIN_HOOKS:-1}" = "0" ]; then
    exit 0
fi

# Read hook JSON from stdin
HOOK_JSON=$(cat || echo "{}")

# Only process Bash tool calls
TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)
if [ "$TOOL_NAME" != "Bash" ]; then
    exit 0
fi

# Check if this is a git commit command
COMMAND=$(echo "$HOOK_JSON" | jq -r '.tool_input.command // ""' 2>/dev/null)
if ! echo "$COMMAND" | grep -qE 'git\s+commit'; then
    exit 0
fi

# Check if ailang is available
if ! command -v ailang &> /dev/null; then
    exit 0
fi

# Extract commit info from the tool output
TOOL_OUTPUT=$(echo "$HOOK_JSON" | jq -r '.tool_output // ""' 2>/dev/null)

# Try to get the latest commit message
COMMIT_MSG=$(git log -1 --format="%s" 2>/dev/null || echo "")
if [ -z "$COMMIT_MSG" ]; then
    exit 0
fi

# Get diff stats for the commit
DIFF_SUMMARY=$(git diff --stat HEAD~1 HEAD 2>/dev/null | tail -1 || echo "")
CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD 2>/dev/null | tr '\n' ',' | sed 's/,$//' || echo "")

# Store as resolution frame (fire-and-forget, don't block Claude)
ailang cache put-resolution \
    --commit-msg "$COMMIT_MSG" \
    --diff-summary "$DIFF_SUMMARY" \
    --files "$CHANGED_FILES" \
    --source "hook:commit" \
    >/dev/null 2>&1 &

exit 0
