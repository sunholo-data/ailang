# Claude Code Integration Setup

This guide explains how to configure Claude Code to work with AILANG's autonomous agent system, enabling seamless handoff between interactive sessions and background agents.

## Overview

The AILANG agent protocol provides two hooks for Claude Code integration:

1. **Stop Hook** (`agent_handoff.sh`) - Detects design docs created during your session and hands them off to autonomous agents for implementation
2. **SessionStart Hook** (`session_start.sh`) - Checks for messages from agents when you start a new session

## Prerequisites

- Claude Code installed and configured
- AILANG built and installed (`make install`)
- Go 1.21+ (for running agent examples)

## Setup Steps

### 1. Create Hooks Configuration

Create `.claude/hooks.json` in your AILANG project root:

```json
{
  "hooks": {
    "Stop": {
      "command": "scripts/hooks/agent_handoff.sh",
      "timeout": 30
    },
    "SessionStart": {
      "command": "scripts/hooks/session_start.sh",
      "timeout": 10
    }
  }
}
```

### 2. Make Hook Scripts Executable

```bash
chmod +x scripts/hooks/agent_handoff.sh
chmod +x scripts/hooks/session_start.sh
```

### 3. Configure Environment (Optional)

Set custom state directory if needed:

```bash
export STATE_DIR="/path/to/custom/state"
```

Default: `.ailang/state` (relative to project root)

### 4. Verify Setup

Test the hooks manually:

```bash
# Test Stop hook
CLAUDE_HOOK_JSON='{"sessionId":"test-123","userId":"test-user","event":"Stop"}' \
  scripts/hooks/agent_handoff.sh

# Test SessionStart hook
CLAUDE_HOOK_JSON='{"sessionId":"test-123","userId":"test-user","event":"SessionStart"}' \
  scripts/hooks/session_start.sh
```

Check logs:
```bash
tail -f .ailang/state/hooks.log
```

## How It Works

### Design Doc Handoff (Stop Hook)

When you stop a Claude Code session, the `agent_handoff.sh` hook:

1. Scans `design_docs/planned/` for files modified in the last 5 minutes
2. Computes SHA256 hashes of the documents
3. Stores them in the artifact store (`.ailang/state/artifacts/`)
4. Sends a message to the `sprint-planner` agent with artifact references
5. The sprint-planner can then autonomously implement the design

**Example workflow:**
```
You: "Design a fix for the import bug"
Claude Code: *creates design_docs/planned/M-IMPORT-FIX.md*
You: "Looks good" (session stops)
→ Hook fires automatically
→ Message sent to sprint-planner: {"task": "implement_design_doc", "artifacts": [...]}
→ Sprint-planner picks it up and starts implementation
```

### Notification on Session Start (SessionStart Hook)

When you start a new Claude Code session, the `session_start.sh` hook:

1. Checks `.ailang/state/messages/inbox/user/_unread/` for messages
2. Displays a notification if you have unread messages
3. Shows a preview of the most recent message

**Example output:**
```
╔═══════════════════════════════════════════════════════════╗
║  📬 You have 2 unread message(s) from agents              ║
╚═══════════════════════════════════════════════════════════╝

To view messages, run:
  go run examples/agents/check_inbox.go user

Preview (most recent):
  From: sprint-executor
  Message ID: msg_20251025_094523_abc123
```

## Usage Examples

### Viewing Your Inbox

```bash
# View all unread messages
go run examples/agents/check_inbox.go user

# View only unread messages (explicit)
go run examples/agents/check_inbox.go --unread-only user

# View read messages
go run examples/agents/check_inbox.go --read-only user

# View archived messages
go run examples/agents/check_inbox.go --archived user

# Archive all messages after viewing
go run examples/agents/check_inbox.go --archive user
```

### Sending Messages to Agents

```bash
# Send a task to an agent
go run examples/agents/send_message.go sprint-planner '{
  "task": "implement_design_doc",
  "design_doc": "design_docs/planned/M-TEST-123.md"
}'

# Send a message to the user inbox (for testing)
go run examples/agents/send_message.go --to-user '{
  "message": "Test notification from agent",
  "status": "completed"
}'

# Send and wait for response (timeout after 30 seconds)
go run examples/agents/send_message.go --wait 30s sprint-planner '{
  "action": "status"
}'
```

## Directory Structure

After setup, your `.ailang/state` directory will contain:

```
.ailang/state/
├── agents.db                          # SQLite control plane
├── signing_key.json                   # HMAC signing key
├── hooks.log                          # Hook execution log
├── artifacts/                         # Content-addressed artifact storage
│   └── sha256/
│       └── abc123.../
│           ├── content
│           └── metadata.json
├── messages/                          # Agent message inboxes
│   ├── sprint-planner/                # Agent inbox
│   │   └── msg_xyz.pending.json
│   └── inbox/                         # User inbox
│       └── user/
│           ├── _unread/               # New messages
│           ├── _read/                 # Read messages
│           └── _archive/              # Archived messages
```

## Troubleshooting

### Hook not firing

**Check hook configuration:**
```bash
cat .claude/hooks.json
```

**Verify hook scripts are executable:**
```bash
ls -la scripts/hooks/
```

**Check Claude Code logs:**
- Location varies by platform (see Claude Code documentation)

### Messages not appearing

**Check state directory exists:**
```bash
ls -la .ailang/state/
```

**Verify message was sent:**
```bash
ls -la .ailang/state/messages/inbox/user/_unread/
```

**Check hook logs:**
```bash
tail -50 .ailang/state/hooks.log
```

### Permission issues

**Ensure scripts are executable:**
```bash
chmod +x scripts/hooks/*.sh
```

**Check state directory permissions:**
```bash
chmod 755 .ailang/state
```

## Security Considerations

### Message Signing

All messages are signed with HMAC-SHA256 to prevent spoofing:
- Signing key stored in `.ailang/state/signing_key.json` (mode 0600)
- Each message includes `signature`, `signature_alg`, and `kid` fields
- Verification happens automatically when reading messages

### Path Sanitization

The artifact store rejects paths containing `..` to prevent directory traversal attacks.

### Hook Timeouts

Both hooks have timeouts (30s for Stop, 10s for SessionStart) to prevent hanging sessions.

## Advanced Configuration

### Custom Agent Target

Edit `scripts/hooks/agent_handoff.sh` to change the target agent:

```bash
TARGET_AGENT="your-custom-agent"
```

### Adjust Time Window for Design Docs

Change the `-mmin -5` parameter in `agent_handoff.sh`:

```bash
# Find files modified in last 10 minutes instead of 5
find "$DESIGN_DOCS_DIR" -type f -name "*.md" -mmin -10 -print0
```

### Rate Limiting (Future)

Rate limiting for user inbox notifications will be added in v0.3.21. For now, hooks fire unconditionally.

## Next Steps

- Read [AGENT_HANDOFF.md](AGENT_HANDOFF.md) for workflow examples
- See [M-AGENT-PROTOCOL design doc](../design_docs/planned/v0_3_19/M-AGENT-PROTOCOL.md) for protocol details
- Explore autonomous agent examples in `examples/agents/`

## Support

For issues or questions:
- Check logs: `.ailang/state/hooks.log`
- Run hooks manually with test data (see "Verify Setup" above)
- File an issue on GitHub with logs and config

---

**Last updated:** 2025-10-25 (v0.3.20)
