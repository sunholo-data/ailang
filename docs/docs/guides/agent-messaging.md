---
sidebar_position: 12
title: Agent Messaging
description: How to send and receive messages between AILANG core and external projects
---

# Agent Messaging Guide

How to send and receive messages between AILANG core and external projects.

## Quick Reference

```bash
# Check inbox for feedback from projects
ailang agent inbox --unread-only user

# Send response to a project
ailang agent send PROJECT_NAME '{"type":"response","title":"Title","description":"Details"}'

# Acknowledge processed messages
ailang agent ack MSG_ID
ailang agent ack --all
```

## Message Format

```json
{
  "type": "response",
  "title": "Brief title",
  "description": "Detailed message",
  "priority": "low|medium|high",
  "from_project": "ailang_core"
}
```

**Important:** Avoid hyphens in JSON string values - use underscores instead (e.g., `ailang_core` not `ailang-core`).

## Architecture

```
External Project                      AILANG Core
     │                                    │
     │  send --to-user --from "project"   │
     ├───────────────────────────────────►│
     │    → inbox/user/_unread/           │
     │                                    │
     │◄───────────────────────────────────┤
     │    send <project_name> '{...}'     │
     │    → <project_name>/*.pending.json │
```

## Workflows

### Responding to Bug Reports

1. Check inbox: `ailang agent inbox --unread-only user`
2. Create design doc if needed
3. Send acknowledgment:
   ```bash
   ailang agent send PROJECT_NAME '{"type":"response","title":"Bug acknowledged","description":"Design doc created, targeted for vX.Y.Z"}'
   ```
4. Acknowledge original: `ailang agent ack MSG_ID`

### Answering Questions

```bash
ailang agent send PROJECT_NAME '{"type":"response","title":"Answer: Topic","description":"Use foo() from std/bar. Example: let x = foo(42)"}'
```

## File Locations

| Type | Location |
|------|----------|
| **Incoming** | `~/.ailang/state/messages/inbox/user/_unread/` |
| **Outgoing** | `~/.ailang/state/messages/PROJECT_NAME/*.pending.json` |

## Message Types

| Type | Purpose |
|------|---------|
| `response` | Reply to bug report or feature request |
| `acknowledgment` | Confirm receipt of a message |
| `resolution` | Bug fixed or feature shipped |
| `bug_report` | Report a bug (from external projects) |
| `feature_request` | Request a new feature |
| `documentation_request` | Request documentation updates |

## Integration with Claude Code

The SessionStart hook automatically checks both inbox locations:

1. **User Inbox** (`~/.ailang/state/messages/inbox/user/`) - Home directory, persists across projects
2. **Claude-Code Inbox** (`.ailang/state/messages/claude-code/`) - Project-specific messages

Messages appear in system reminders at session start. After handling:

```bash
ailang agent ack <message-id>    # Acknowledge specific message
ailang agent ack --all           # Acknowledge all messages
```

## See Also

- [Agent Workflows](/docs/guides/agent-workflows) - Automated agent workflows
- [State System](/docs/guides/state-system-workflow) - Persistent state management
- [Cross-Project Messaging](/docs/guides/cross-project-messaging) - Advanced messaging patterns
