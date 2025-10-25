---
name: Agent Inbox
description: Check and process messages from autonomous AILANG agents. Use when starting a session, after agent handoffs, or when checking for completion notifications.
---

# Agent Inbox

**Check for messages from autonomous agents at session start and process completion notifications.**

## Quick Start

**Most common usage:**
```bash
# At session start, check for messages using AILANG CLI:
ailang agent inbox user

# Show only unread messages:
ailang agent inbox user --unread-only

# Archive messages after processing:
ailang agent inbox user --archive
```

**Expected output:**
```
📬 User Inbox (2 messages)
================================================================================

▶ Message 1/2
  ID: msg_20251025_143021_abc123
  From: sprint-executor
  Type: request
  Timestamp: 2025-10-25T14:30:21Z
  Correlation: cycle_20251025_450
  Payload: {"status":"completed","milestone":"5/5","result":"All tests passing, docs updated"}

▶ Message 2/2
  ID: msg_20251025_143055_def456
  From: sprint-planner
  Type: request
  Timestamp: 2025-10-25T14:30:55Z
  Payload: {"status":"plan_ready","design_doc":"design_docs/planned/M-FIX-123.md"}
```

## When to Use This Skill

**Invoke this skill when:**
- **Session starts** - First action in every Claude Code session (required by CLAUDE.md)
- **After handoffs** - When you've sent work to autonomous agents
- **Periodic checks** - User asks "any updates from agents?"
- **Debugging** - To see agent communication history

## Available Commands

### `ailang agent inbox user`

Check user inbox and display all messages. Automatically marks messages as read.

**Usage:**
```bash
# View all messages (unread + read)
ailang agent inbox user

# Show only unread messages
ailang agent inbox user --unread-only

# Show only read messages
ailang agent inbox user --read-only

# Show archived messages
ailang agent inbox user --archived

# Archive messages after viewing
ailang agent inbox user --archive

# Limit number of messages shown
ailang agent inbox user --limit 5
```

### `ailang agent send`

Send a message to an agent or user inbox.

**Usage:**
```bash
# Send to user inbox (for agent → user communication)
ailang agent send --to-user --from "agent-name" '{"message": "Task complete", "status": "done"}'

# Send to specific agent
ailang agent send agent-name '{"task": "do_something"}'
```

## Workflow

### 1. Session Start Check (REQUIRED)

**At the start of EVERY session, run:**
```bash
# Check for messages from autonomous agents
ailang agent inbox user
```

**If messages exist:**
- Read and summarize each message to user
- Identify message type (completion, error, handoff)
- Ask user if they want action taken
- Archive after handling: `ailang agent inbox user --archive`

### 2. Process Completion Notifications

**When agent reports completion:**
```bash
# 1. Check messages
ailang agent inbox user --unread-only

# 2. Review message payload for artifact paths
# (Payload shown in inbox output)

# 3. Review results
ls -la eval_results/baselines/v0.4.2/

# 4. Report to user
echo "✅ Sprint complete! Results at: eval_results/baselines/v0.4.2/"

# 5. Archive after processing
ailang agent inbox user --archive
```

### 3. Handle Error Reports

**When agent reports errors:**
```bash
# 1. Check messages to see error details
ailang agent inbox user --unread-only

# 2. Review payload for error and log path
# Example payload:
# {"status": "error", "milestone": "3/5", "error": "Tests failing", "details": "logs/sprint.log"}

# 3. Read logs
cat .ailang/state/logs/sprint-executor.log

# 4. Diagnose and report to user
echo "⚠️  Agent encountered error: Tests failing at milestone 3/5"

# 5. Either fix manually or send corrective instructions
```

### 4. Respond to Agent

**Send response back to agent:**
```bash
# Send approval/instructions to agent
ailang agent send sprint-planner '{
  "status": "approved",
  "action": "proceed_with_execution"
}'

# Or send to user for notification
ailang agent send --to-user --from "claude-code" '{
  "message": "Issue resolved, agent can continue"
}'
```

## Message Types

### Completion Notification
```json
{
  "from_agent": "sprint-executor",
  "payload": {
    "status": "completed",
    "milestone": "5/5",
    "result": "All tests passing, docs updated",
    "artifacts": ["eval_results/baselines/v0.3.15/"]
  }
}
```

### Handoff Instruction
```json
{
  "from_agent": "sprint-planner",
  "payload": {
    "status": "plan_ready",
    "design_doc": "design_docs/planned/M-FIX-123.md",
    "next_steps": "Review plan, approve for execution"
  }
}
```

### Error Report
```json
{
  "from_agent": "sprint-executor",
  "payload": {
    "status": "error",
    "milestone": "3/5",
    "error": "Tests failing: 5 benchmarks broken",
    "details": ".ailang/state/logs/sprint-executor.log"
  }
}
```

## Resources

### Message Format Reference
See [`resources/message_format.md`](resources/message_format.md) for complete message format specification.

### Troubleshooting Guide
See [`resources/troubleshooting.md`](resources/troubleshooting.md) for common issues and solutions.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + workflow)
2. **Execute as needed**: Scripts in `scripts/` (check_messages.sh, mark_processed.sh)
3. **Load on demand**: `resources/` (detailed references, troubleshooting)

## Notes

- **Required by CLAUDE.md**: Session start check is mandatory for all Claude Code sessions
- **Inbox location**: User inbox at `~/.ailang/state/messages/inbox/user/` (global, home directory)
- **Hook integration**: SessionStart hook automatically forwards messages from user inbox
- **CLI commands**: Use `ailang agent inbox` and `ailang agent send` - avoid bash scripts
- **Auto-marking**: Messages automatically marked as read when viewed
- **Message lifecycle**: Unread → Read → Archived (optional)

## CLI Command Reference

| Command | Purpose |
|---------|---------|
| `ailang agent inbox user` | View all messages (auto-marks as read) |
| `ailang agent inbox user --unread-only` | View only unread messages |
| `ailang agent inbox user --archive` | Archive messages after viewing |
| `ailang agent send --to-user --from "agent" '{...}'` | Send message to user inbox |
| `ailang agent send agent-name '{...}'` | Send message to specific agent |
