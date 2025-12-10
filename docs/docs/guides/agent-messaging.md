---
sidebar_position: 12
title: Agent Messaging
description: How to send and receive messages between AILANG core and external projects
---

# Agent Messaging Guide

How to send and receive messages between AILANG core and external projects using the unified messaging system.

## Quick Reference

```bash
# List all messages
ailang messages list

# Show only unread messages
ailang messages list --unread

# Read full message content
ailang messages read MSG_ID

# Acknowledge (mark as read)
ailang messages ack MSG_ID
ailang messages ack --all

# Send a message
ailang messages send user "Your message" --title "Title" --from "agent-name"

# Send with GitHub sync
ailang messages send user "Bug report" --title "Parser crash" --type bug --github
```

## Storage Backend

All messages are stored in a SQLite database:
- **Location**: `~/.ailang/state/collaboration.db`
- **Accessible via**: CLI (`ailang messages`) and Collaboration Hub dashboard
- **Message statuses**: `unread`, `read`, `archived`, `deleted`

## Message Format

Messages in the system contain:

```json
{
  "id": "uuid",
  "message_id": "msg_20251210_123456_abc123",
  "from_agent": "agent-name",
  "to_inbox": "user",
  "message_type": "notification",
  "title": "Brief title",
  "payload": "Detailed message content",
  "category": "bug|feature|general",
  "github_issue": 42,
  "github_repo": "owner/repo",
  "status": "unread",
  "created_at": "2025-12-10T12:34:56Z"
}
```

## Architecture

```
External Project                      AILANG Core
     |                                    |
     |  ailang messages send user "msg"   |
     |------------------------------------&gt;|
     |    -&gt; collaboration.db             |
     |                                    |
     |  (Optional) --github flag          |
     |------------------------------------&gt;|
     |    -&gt; GitHub Issue created         |
     |                                    |
     |&lt;------------------------------------|
     |    ailang messages send proj "..."  |
     |    -&gt; collaboration.db             |
```

## Workflows

### Responding to Bug Reports

1. Check inbox: `ailang messages list --unread`
2. Read full message: `ailang messages read MSG_ID`
3. Create design doc if needed
4. Send acknowledgment:
   ```bash
   ailang messages send PROJECT_NAME "Bug acknowledged - design doc created for vX.Y.Z" \
     --title "Bug acknowledged" --from "ailang"
   ```
5. Acknowledge original: `ailang messages ack MSG_ID`

### Sending to GitHub Issues

Use the `--github` flag to also create a GitHub issue:

```bash
# Report a bug (creates GitHub issue with "bug" label)
ailang messages send user "Parser crashes on nested records" \
  --title "Parser crash bug" --type bug --github

# Request a feature
ailang messages send user "Add async/await syntax" \
  --title "Async support" --type feature --github

# The message is ALWAYS saved locally first
# GitHub sync is optional and fails gracefully
```

## CLI Commands

### List Messages

```bash
ailang messages list                    # All messages
ailang messages list --unread           # Only unread
ailang messages list --inbox user       # Filter by inbox
ailang messages list --from agent-name  # Filter by sender
ailang messages list --json             # JSON output
ailang messages list --limit 50         # Limit results
```

### Read Message Content

```bash
ailang messages read MSG_ID             # Full content, marks as read
ailang messages read MSG_ID --peek      # View without marking read
ailang messages read MSG_ID --json      # JSON output
```

### Acknowledge Messages

```bash
ailang messages ack MSG_ID              # Mark specific message as read
ailang messages ack --all               # Mark all as read
ailang messages ack --all --inbox user  # Mark all in inbox as read
```

### Un-acknowledge (Mark Unread)

```bash
ailang messages unack MSG_ID            # Move back to unread
```

### Send Messages

```bash
# Basic send
ailang messages send INBOX "message content" --title "Title" --from "agent"

# With GitHub sync
ailang messages send INBOX "message" --type bug --github
ailang messages send INBOX "message" --type feature --github
ailang messages send INBOX "message" --github --repo owner/repo
```

### Import from GitHub

```bash
ailang messages import-github                    # Import from default repo
ailang messages import-github --repo owner/repo  # Specific repo
ailang messages import-github --labels bug,help  # Filter by labels
ailang messages import-github --dry-run          # Preview without importing
```

### Cleanup Old Messages

```bash
ailang messages cleanup --older-than 7d    # Remove messages older than 7 days
ailang messages cleanup --expired          # Remove expired messages
ailang messages cleanup --dry-run          # Preview without deleting
```

### Watch for New Messages

```bash
ailang messages watch                  # Watch all inboxes
ailang messages watch --inbox user     # Watch specific inbox
```

## GitHub Integration

### Configuration

Create `~/.ailang/config.yaml`:

```yaml
github:
  expected_user: YourGitHubUsername      # REQUIRED: Must match gh auth status
  default_repo: owner/repo               # Default repo for issues
  create_labels:                         # Labels added to created issues
    - ailang-message
  watch_labels:                          # Labels to filter when importing
    - ailang-message
  auto_import: true                      # Auto-import on session start
```

### How It Works

1. **Account Validation**: The `expected_user` must match the active `gh` account
   - Run `gh auth status` to check current account
   - Switch accounts with `gh auth switch --user USERNAME`
   - This prevents accidentally creating issues in wrong repos

2. **Auto-Label Creation**: Labels are automatically created if they don't exist
   - `from:agent-name` (purple) - who sent the message
   - `bug` (red), `feature` (cyan), `general` (light blue)
   - `ailang-message` (blue) - identifies AILANG messages

3. **Title Prefix**: Issues are prefixed with sender name
   - `[agent-name] Original Title`

4. **Issue Linking**: Created issue number is saved to database
   - Query with: `SELECT * FROM inbox_messages WHERE github_issue_number IS NOT NULL`

### Workflow

```bash
# 1. Check GitHub auth
gh auth status

# 2. Switch account if needed
gh auth switch --user YourUsername

# 3. Send message with GitHub sync
ailang messages send user "Bug: parser crashes" --type bug --github

# 4. Import issues from GitHub on session start (automatic via hook)
ailang messages import-github
```

## Integration with Claude Code

The SessionStart hook (`scripts/hooks/session_start.sh`) automatically:

1. Imports GitHub issues as messages (respects `auto_import` config)
2. Checks for unread messages
3. Injects message summary into system reminders

Messages appear at session start. After handling:

```bash
ailang messages ack <message-id>    # Acknowledge specific message
ailang messages ack --all           # Acknowledge all messages
```

## Message Types and Routing

| Type | Purpose | Goes to GitHub? |
|------|---------|-----------------|
| `bug` | Bug report | Yes (with `--github`) |
| `feature` | Feature request | Yes (with `--github`) |
| `general` | General communication | No (local only) |

**Routing guidance:**
- **Bugs and features** → Use `--github` for visibility across all AILANG instances
- **Coordination messages** → Local only, for agent-to-agent communication
- **Instructions from humans** → Create GitHub issues, they'll be imported automatically

## Bi-directional GitHub Sync

The messaging system supports **two-way sync** with GitHub:

### Sending to GitHub (Agent → GitHub)
```bash
# Bug reports and feature requests go to GitHub for visibility
ailang messages send user "Parser crash" --type bug --github
```

### Importing from GitHub (GitHub → Local)
```bash
# Import issues from GitHub (runs automatically on session start)
ailang messages import-github

# Or manually with filters
ailang messages import-github --labels help-wanted
```

**Use case: Human instructions via GitHub**

You can write instructions as GitHub issues and have agents pick them up:

1. Create issue on GitHub with `ailang-message` label
2. Next session, `import-github` runs automatically
3. Issue appears in agent's inbox as a message
4. Agent reads and acts on the instructions

## Aliases

The `messages` command has an alias for convenience:

```bash
ailang msg list        # Same as: ailang messages list
ailang msg send ...    # Same as: ailang messages send ...
```

## See Also

- [Cross-Project Messaging](/docs/guides/cross-project-messaging) - Send feedback from your projects
- [Agent Workflows](/docs/guides/agent-workflows) - Automated agent workflows
- [State System](/docs/guides/state-system-workflow) - Persistent state management
