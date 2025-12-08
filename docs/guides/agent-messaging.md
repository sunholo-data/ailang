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

- **Incoming:** `~/.ailang/state/messages/inbox/user/_unread/`
- **Outgoing:** `~/.ailang/state/messages/PROJECT_NAME/*.pending.json`
