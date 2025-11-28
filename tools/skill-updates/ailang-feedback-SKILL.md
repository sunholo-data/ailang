# AILANG Feedback

Bidirectional messaging between AILANG core and external projects.

## Quick Start

```bash
# Check for incoming feedback from projects
ailang agent inbox --unread-only user

# Send response to a project
ailang agent send PROJECT_NAME '{"type":"response","title":"Title","description":"Details"}'
```

## When to Use This Skill

**Receiving feedback (in AILANG core repo):**
- Check `ailang agent inbox user` for bug reports, feature requests
- Messages arrive in `~/.ailang/state/messages/inbox/user/_unread/`

**Sending responses (from AILANG core):**
- Send to project inbox: `ailang agent send PROJECT_NAME '{...}'`
- Messages go to `~/.ailang/state/messages/PROJECT_NAME/`

## Command Reference

### Receiving Feedback

```bash
# Check user inbox (where external projects send feedback)
ailang agent inbox user
ailang agent inbox --unread-only user

# Acknowledge after processing
ailang agent ack MSG_ID
ailang agent ack --all
```

### Sending Responses

```bash
# Send response to a project
ailang agent send PROJECT_NAME '{"type":"response","title":"Title","description":"Message body","priority":"medium"}'

# Common response types
# - response: Reply to a bug report or feature request
# - acknowledgment: Confirm receipt
# - update: Provide status update
```

### Message Format

```json
{
  "type": "response",
  "title": "Brief title",
  "description": "Detailed message",
  "priority": "low|medium|high",
  "from_project": "ailang_core"
}
```

**Note:** Avoid hyphens in JSON values as they can cause parsing issues. Use underscores instead (e.g., `ailang_core` not `ailang-core`).

## Messaging Architecture

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

## Workflow Examples

### Processing Bug Reports

```bash
# 1. Check inbox for new messages
ailang agent inbox --unread-only user

# 2. Review the message content (in SessionStart hook output)

# 3. Create design doc if needed
# design_docs/planned/vX_Y_Z/m-bug-description.md

# 4. Send acknowledgment
ailang agent send stapledons_voyage '{"type":"response","title":"Bug acknowledged","description":"Design doc created: m-bug-foo.md, targeted for v0.4.9","priority":"medium"}'

# 5. Acknowledge the original message
ailang agent ack MSG_ID
```

### Answering Documentation Questions

```bash
# Send answer directly
ailang agent send stapledons_voyage '{"type":"response","title":"Answer: How to do X","description":"Use the foo() function from std/bar. Example: let x = foo(42)","priority":"medium"}'
```

## Notes

- **Outgoing to projects:** `~/.ailang/state/messages/PROJECT_NAME/`
- **Incoming from projects:** `~/.ailang/state/messages/inbox/user/`
- Messages persist globally across all projects
- Requires AILANG CLI installed
- Avoid hyphens in JSON string values (use underscores)
