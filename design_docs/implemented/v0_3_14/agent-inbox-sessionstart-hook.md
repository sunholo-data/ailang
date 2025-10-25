# Agent Inbox SessionStart Hook - Implementation & Fix

**Status**: ✅ **COMPLETE & FIXED**
**Version**: v0.3.14
**Date**: 2025-10-25 (Fixed: Same day)

## Problem

Autonomous agents can send messages to the user inbox, but Claude Code sessions don't automatically see these messages.

### Evolution of the Solution

**Attempt 1 (Failed)**: Forward messages to `claude-code` inbox
- Messages were forwarded but Claude didn't check automatically
- Complex forwarding logic was error-prone

**Attempt 2 (Incomplete)**: Use stdout output from SessionStart hook
- Hook executed successfully (logs confirmed)
- Messages formatted and output to stdout
- ❌ **But Claude Code didn't see the output in context!**

**Attempt 3 (Working)**: Use JSON output with `hookSpecificOutput`
- Changed to JSON format with `hookSpecificOutput.additionalContext` field
- ✅ Messages now successfully appear in Claude's context!

## Solution

**Working approach**: Use Claude Code's SessionStart hook with **JSON output format** to inject messages into context.

### How It Works (UPDATED FIX)

The [Claude Code hooks documentation](https://docs.claude.com/en/docs/claude-code/hooks) specifies TWO methods for SessionStart hooks:

1. **Simple method**: Exit code 0 with stdout → supposedly adds to context
2. **JSON method** (REQUIRED): Use `hookSpecificOutput.additionalContext` → Actually works!

**Discovery**: While docs say both methods work, only the JSON method proved reliable in practice.

**JSON output format** (what actually works):
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Your message content here"
  }
}
```

This means:
1. SessionStart hook outputs structured JSON to stdout
2. Claude Code parses the JSON and extracts `additionalContext`
3. That content is **automatically added to Claude's context**
4. No manual checking needed!

### Implementation

**File**: `scripts/hooks/session_start.sh`

**Dual-Location Support** (Updated 2025-10-25):

The hook now checks **TWO inbox locations**:
1. **Home directory**: `~/.ailang/state/messages/inbox/user/_unread/`
   - Messages from `ailang` CLI or other system-wide agents
   - Marked as read by moving to `~/.ailang/state/messages/inbox/user/_read/`

2. **Project directory**: `<project>/.ailang/state/messages/claude-code/`
   - Messages from project-specific agents (e.g., sprint-planner, eval-orchestrator)
   - Marked as processed by moving to `<project>/.ailang/state/messages/claude-code/_processed/`

**Key changes**:
```bash
# OLD: Single inbox location (home only)
INBOX_DIR="$STATE_DIR/messages/inbox/user/_unread"

# NEW: Check BOTH locations
INBOX_DIRS=(
    "$STATE_DIR/messages/inbox/user/_unread"
    "$PROJECT_ROOT/.ailang/state/messages/claude-code"
)

# Collect all messages from both locations
for INBOX_DIR in "${INBOX_DIRS[@]}"; do
    find "$INBOX_DIR" -maxdepth 1 -name "*.json" -type f
done

# Output messages directly to stdout (automatically added to context)
echo "📬 You have $UNREAD_COUNT unread message(s) from autonomous agents:"
# ... display each message ...

# Mark as read/processed based on source location
if [[ "$MSG_DIR" == *"claude-code"* ]]; then
    mv "$MSG_FILE" "$PROJECT_ROOT/.ailang/state/messages/claude-code/_processed/"
else
    mv "$MSG_FILE" "$STATE_DIR/messages/inbox/user/_read/"
fi
```

**Hook configuration**: `.claude/hooks.json`
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash scripts/hooks/session_start.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

### Testing

**Test 1: Home directory messages**:
```bash
# Create test message in home directory
cat > ~/.ailang/state/messages/inbox/user/_unread/test.json <<'EOF'
{
  "from_agent": "test-harness",
  "timestamp": "2025-10-25T14:05:00Z",
  "payload": {
    "message": "Test message from home",
    "status": "testing"
  }
}
EOF

# Run hook manually
bash scripts/hooks/session_start.sh
```

**Test 2: Project directory messages**:
```bash
# Create test message in project directory
cat > .ailang/state/messages/claude-code/test.json <<'EOF'
{
  "from_agent": "project-agent",
  "timestamp": "2025-10-25T16:16:00Z",
  "payload": {
    "message": "Test message from project",
    "status": "testing"
  }
}
EOF

# Run hook manually
bash scripts/hooks/session_start.sh
```

**Test 3: Both locations simultaneously**:
```bash
# Create messages in both locations
cat > ~/.ailang/state/messages/inbox/user/_unread/test1.json <<'EOF'
{"from_agent": "home-agent", "timestamp": "2025-10-25T16:00:00Z", "payload": {"location": "home"}}
EOF

cat > .ailang/state/messages/claude-code/test2.json <<'EOF'
{"from_agent": "project-agent", "timestamp": "2025-10-25T16:01:00Z", "payload": {"location": "project"}}
EOF

# Run hook - should display BOTH messages
bash scripts/hooks/session_start.sh
```

**Expected output** (goes to Claude's context):
```
📬 You have 2 unread message(s) from autonomous agents:

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
From: home-agent
Time: 2025-10-25T16:00:00Z
Message:
{
  "location": "home"
}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
From: project-agent
Time: 2025-10-25T16:01:00Z
Message:
{
  "location": "project"
}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💡 These messages are from autonomous agents that completed work.
   Review the messages above and decide if any action is needed.
```

**Verification**:
- Messages from both locations appear in Claude's context at session start ✅
- Home directory messages marked as read (moved to `~/.ailang/state/messages/inbox/user/_read/`) ✅
- Project messages marked as processed (moved to `.ailang/state/messages/claude-code/_processed/`) ✅
- Hook execution logged to `~/.ailang/state/hooks.log` ✅

## Documentation Updates

**Updated files**:
1. `CLAUDE.md` - Updated SESSION START ROUTINE section
   - Removed manual inbox checking requirement
   - Explained automatic message injection via hook
   - Added examples of what Claude will see

2. `scripts/hooks/session_start.sh` - Simplified implementation
   - Removed `send-message` forwarding logic
   - Output messages directly to stdout
   - Added user-friendly formatting

## Benefits

1. **Automatic**: No manual checking required
2. **Simple**: Single stdout-based implementation
3. **Reliable**: Uses documented Claude Code behavior
4. **Transparent**: Messages visible in Claude's context from session start

## Limitations

1. **Only at session start**: Messages sent during a session won't appear until next session
2. **Read-only for Claude**: Claude sees messages but can't mark them as archived (user must do this manually if needed)
3. **Terminal vs Context**: Message output goes to Claude's context, not visible in user's terminal (by design)

## Future Improvements (Optional)

1. Add filtering (e.g., only show high-priority messages)
2. Support message threading/conversations
3. Add message aging (auto-archive old messages)
4. Provide summary statistics (X messages in last 24h)

## Related

- **Message Protocol**: `design_docs/implemented/v0_3_13/agent-message-protocol.md`
- **Hooks Guide**: `docs/docs/guides/hooks-setup.mdx`
- **Agent Workflows**: `docs/docs/guides/agent-workflows.mdx`
