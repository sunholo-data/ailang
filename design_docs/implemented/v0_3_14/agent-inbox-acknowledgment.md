# Agent Inbox Message Acknowledgment System

**Status**: ✅ COMPLETE (October 2025)
**Version**: v0.3.14+
**Implementation Time**: ~3 hours

## Problem Statement

The original SessionStart hook automatically marked messages as read before Claude Code could see them, causing race conditions in multi-session scenarios:

1. Agent sends message to `_unread` folder
2. SessionStart hook runs on new session start
3. Hook outputs message content to stdout
4. Hook **immediately** moves message to `_read` folder
5. If Claude Code doesn't inject hook output into context (timing/reliability issue), message is lost
6. User opens another session → message already marked as read, no longer visible

**Root Cause**: Auto-mark-as-read happened BEFORE context injection was confirmed successful.

## Solution: Manual Acknowledgment

Messages remain in `_unread` (or `.pending.json` for claude-code inbox) until explicitly acknowledged:

1. SessionStart hook outputs message content
2. Message **stays in _unread folder**
3. Claude Code receives message in context (or not - doesn't matter for message persistence)
4. Claude or user explicitly acknowledges with `ailang agent ack <message-id>`
5. Message moves to `_processed` folder
6. If session crashes before ack, message still in _unread on next session

## Implementation

### 1. Hook Changes (`scripts/hooks/session_start.sh`)

**Removed auto-mark-as-read logic** (lines 150-165):
```bash
# ❌ OLD CODE (removed):
if [ "$MARK_READ" = "true" ]; then
    mark_messages_as_read "${UNREAD_MSGS[@]}"
fi

# ✅ NEW CODE (messages stay in _unread):
# NOTE: Messages are NOT marked as read automatically
# Claude Code must explicitly acknowledge them using: ailang agent ack <message-id>
```

**Added lock file mechanism** (prevents duplicate execution):
```bash
LOCK_FILE="${STATE_DIR}/session_start.lock"
if [ -f "$LOCK_FILE" ]; then
    LOCK_AGE=$(($(date +%s) - $(stat -f %m "$LOCK_FILE" 2>/dev/null || echo 0)))
    if [ "$LOCK_AGE" -lt 3 ]; then
        echo "📭 Agent inbox: Already checked in this session."
        exit 0
    fi
fi
touch "$LOCK_FILE"
```

### 2. CLI Command (`cmd/ailang/agent.go`)

**Added `ailang agent ack` command**:
```go
case "ack", "acknowledge":
    agentAckCommand()

func agentAckCommand() {
    flags := flag.NewFlagSet("ack", flag.ExitOnError)
    all := flags.Bool("all", false, "Acknowledge all unread messages")
    stateDir := flags.String("state-dir", "", "State directory")
    flags.Parse(os.Args[3:])

    if *all {
        ackAllMessages(*stateDir)
        return
    }

    if flags.NArg() == 0 {
        fmt.Fprintf(os.Stderr, "Usage: ailang agent ack <message-id> or --all\n")
        os.Exit(1)
    }

    messageID := flags.Arg(0)
    if err := ackMessage(*stateDir, messageID); err != nil {
        fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
        os.Exit(1)
    }
    fmt.Printf("%s Message %s acknowledged\n", green("✓"), messageID)
}
```

**Handles multiple filename patterns** (for `.pending.json` suffix):
```go
patterns := []string{
    messageID + ".json",
    messageID + ".pending.json",
    messageID, // In case user provides full filename
}
```

**Moves to appropriate processed folder**:
```go
// For user inbox: _unread → _read
dstDir := filepath.Join(filepath.Dir(filepath.Dir(srcPath)), "_read")

// For claude-code inbox: pending.json → _processed/message.json
dstDir := filepath.Join(filepath.Dir(srcPath), "_processed")
baseName := strings.TrimSuffix(filepath.Base(srcPath), ".pending.json") + ".json"
```

### 3. Un-acknowledgment Command (`cmd/ailang/agent.go`)

**Added `ailang agent unack` command** for moving messages back to unread:
```go
case "unack", "unacknowledge":
    agentUnackCommand()

func agentUnackCommand() {
    // Parse message ID, find in _processed or _read
    // Move back to _unread (or .pending.json for claude-code)
}

func unackMessage(stateDir, messageID string) error {
    // Search in _processed and _read folders
    // Move back to _unread with proper filename (.pending.json for claude-code)
}
```

**Key features**:
- Searches `_read` (user inbox) and `_processed` (claude-code inbox)
- Restores `.pending.json` suffix for claude-code messages
- Clear error if message not found in processed folders

### 4. Permissions (`.claude/settings.local.json`)

**Added auto-approve permissions**:
```json
"allow": [
  "Bash(ailang agent ack:*)",
  "Bash(ailang agent ack --all)",
  "Bash(ailang agent unack:*)"
]
```

### 5. Documentation (`CLAUDE.md`)

**Updated SessionStart routine**:
- Changed: "Marks messages as read after displaying them"
- To: "Does NOT mark messages as read - you must explicitly acknowledge them"

**Added acknowledgment responsibilities**:
```markdown
4. **Mark messages as acknowledged** - Use `ailang agent ack <message-id>` or `ailang agent ack --all` to move messages to processed folder
5. **Un-acknowledge if you fail** - If you can't complete the task (errors, blockers, need user help), use `ailang agent unack <message-id>` to move it back to unread for the next session
```

## Usage

### For Claude Code (automated)

When you see agent messages in context:
```bash
# Acknowledge a specific message (task completed successfully)
ailang agent ack msg_20251025_155729_a5f3e77ee975

# Acknowledge all unread messages
ailang agent ack --all

# Un-acknowledge if task failed (move back to unread for retry)
ailang agent unack msg_20251025_155729_a5f3e77ee975
```

### For Users (manual)

```bash
# Check inbox
ailang agent inbox user --unread-only

# Read message details
ailang agent inbox user

# Acknowledge message after reading
ailang agent ack msg_20251025_155729_a5f3e77ee975

# Un-acknowledge if you couldn't solve the issue
ailang agent unack msg_20251025_155729_a5f3e77ee975
```

## Benefits

1. **No message loss**: Messages persist until explicitly acknowledged
2. **Session-safe**: Crashed/interrupted sessions don't lose messages
3. **Multi-session safe**: Multiple Claude Code sessions see same messages
4. **Explicit acknowledgment**: Clear when messages are processed
5. **Batch acknowledgment**: `--all` flag for bulk processing
6. **Retry capability**: Failed tasks can be un-acknowledged for retry in next session

## Testing

**Verified behaviors**:
- ✅ Messages stay in _unread after SessionStart hook runs
- ✅ `ailang agent ack` moves message to _processed
- ✅ `ailang agent ack --all` processes all unread messages
- ✅ `ailang agent unack` moves message back to _unread
- ✅ Handles `.pending.json` suffix in claude-code inbox (both directions)
- ✅ Auto-approve permissions work in Claude Code
- ✅ Lock file prevents duplicate hook execution
- ✅ Logged activity visible in `~/.ailang/state/hooks.log`

**Test command used**:
```bash
# Send test message
./bin/send-message claude-code '{"test": "acknowledgment system", "timestamp": "now"}'

# Start new session, check if message appears
# (should see message in context via SessionStart hook)

# Acknowledge message
ailang agent ack msg_20251025_155729_a5f3e77ee975

# Verify message moved to _processed
ls -la .ailang/state/messages/claude-code/_processed/
```

## Known Limitations

1. **Hook output reliability**: SessionStart hook output may not always appear in Claude Code context (Claude Code limitation, not our bug)
2. **Manual fallback required**: If hook output doesn't appear, user must manually run `ailang agent inbox` to see messages
3. **No automatic retry**: If ack command fails, user must retry manually

## Future Enhancements

1. **Batch acknowledgment by sender**: `ailang agent ack --from sprint-planner`
2. **Acknowledgment with reply**: `ailang agent ack <id> --reply "message"`
3. **Expiration-based auto-ack**: Auto-acknowledge messages older than N days
4. **Acknowledgment history**: Track which sessions acknowledged which messages

## Metrics

- **Lines Changed**: ~150 lines (removed auto-mark-as-read, added ack command)
- **Files Modified**: 4 (session_start.sh, agent.go, settings.local.json, CLAUDE.md)
- **Test Coverage**: Manual testing (no automated tests yet)
- **Development Time**: ~3 hours (including debugging and documentation)

## Related Documents

- [Agent Protocol Documentation](../../docs/AGENT_PROTOCOL.md)
- [Hooks Configuration Guide](../../docs/guides/hooks-setup.mdx)
- [Agent Inbox Skill](.claude/skills/agent-inbox/README.md)
