# Agent Inbox Command Reference

Complete reference for the AILANG messaging system CLI commands.

## Command Overview

| Command | Alias | Purpose |
|---------|-------|---------|
| `ailang messages list` | `msg ls` | List messages |
| `ailang messages read` | `msg read` | Read full message |
| `ailang messages ack` | `msg ack` | Mark as read |
| `ailang messages unack` | `msg unack` | Mark as unread |
| `ailang messages send` | `msg send` | Send message |
| `ailang messages reply` | `msg reply` | Reply to GitHub issue |
| `ailang messages watch` | `msg watch` | Watch for new |
| `ailang messages cleanup` | `msg cleanup` | Remove old messages |
| `ailang messages import-github` | - | Import from GitHub |

## List Messages

```bash
ailang messages list [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--unread` | Only unread messages | false |
| `--inbox NAME` | Filter by inbox | all |
| `--from AGENT` | Filter by sender | all |
| `--limit N` | Max messages | 20 |
| `--json` | JSON output | false |

**Examples:**
```bash
ailang messages list --unread
ailang messages list --inbox user --limit 10
ailang messages list --from stapledon --json
```

## Read Message

```bash
ailang messages read MSG_ID [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--peek` | Don't mark as read | false |
| `--json` | JSON output | false |

**Examples:**
```bash
ailang messages read msg_20251210_123456_abc123
ailang messages read msg_20251210_123456_abc123 --peek
```

## Acknowledge Message

```bash
ailang messages ack [MSG_ID] [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--all` | Mark all as read | false |
| `--inbox NAME` | Filter for --all | all |

**Examples:**
```bash
ailang messages ack msg_20251210_123456_abc123
ailang messages ack --all
ailang messages ack --all --inbox user
```

## Un-acknowledge Message

```bash
ailang messages unack MSG_ID
```

Moves message back to unread status.

**Example:**
```bash
ailang messages unack msg_20251210_123456_abc123
```

## Send Message

```bash
ailang messages send INBOX "MESSAGE" [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--title TEXT` | Message title | truncated message |
| `--from AGENT` | Sender name | "cli" |
| `--correlation ID` | Correlation ID | none |
| `--github` | Create GitHub issue | false |
| `--type TYPE` | bug/feature/general | none |
| `--repo OWNER/REPO` | GitHub repo | config default |

**Examples:**
```bash
# Basic local message
ailang messages send user "Task complete" --title "Done" --from "agent"

# Bug report with GitHub sync
ailang messages send user "Parser crashes on nested records" \
  --title "Parser bug" --type bug --github

# Feature request
ailang messages send user "Need async support" \
  --title "Async" --type feature --github

# Override repo
ailang messages send user "Bug" --type bug --github --repo owner/other-repo
```

## Reply to GitHub Issue

```bash
ailang messages reply MSG_ID "REPLY_TEXT" [flags]
```

Adds a comment to an existing GitHub issue thread. Only works for messages that were created with `--github` flag.

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--from AGENT` | Sender name for attribution | "cli" |
| `--repo OWNER/REPO` | Override repo | message's repo or config default |

**Examples:**
```bash
# Reply to a bug report
ailang messages reply msg_20251210_123456_abc123 "Fixed in v0.5.10" --from "claude-code"

# Reply with explicit repo
ailang messages reply MSG_ID "Working on it" --repo owner/repo
```

**Note:** The message must have a linked GitHub issue. Messages without `--github` flag cannot be replied to.

## Import from GitHub

```bash
ailang messages import-github [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--repo OWNER/REPO` | GitHub repo | config default |
| `--labels LIST` | Comma-separated labels | config watch_labels |
| `--inbox NAME` | Target inbox | "user" |
| `--dry-run` | Preview only | false |

**Examples:**
```bash
ailang messages import-github
ailang messages import-github --labels bug,help-wanted
ailang messages import-github --dry-run
```

## Watch Messages

```bash
ailang messages watch [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--inbox NAME` | Watch specific inbox | all |

**Example:**
```bash
ailang messages watch --inbox user
```

## Cleanup Messages

```bash
ailang messages cleanup [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--older-than DURATION` | Remove older than (e.g., 7d) | required |
| `--expired` | Remove expired only | false |
| `--dry-run` | Preview only | false |

**Examples:**
```bash
ailang messages cleanup --older-than 7d
ailang messages cleanup --expired
ailang messages cleanup --older-than 30d --dry-run
```

## Database Schema

Messages stored in `~/.ailang/state/collaboration.db`:

```sql
CREATE TABLE inbox_messages (
    id TEXT PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    correlation_id TEXT,
    from_agent TEXT NOT NULL,
    to_inbox TEXT NOT NULL,
    message_type TEXT NOT NULL,
    title TEXT NOT NULL,
    payload TEXT,
    category TEXT,                -- bug, feature, general
    github_issue_number INTEGER,  -- Linked GitHub issue
    github_repo TEXT,             -- owner/repo
    status TEXT NOT NULL,         -- unread, read, archived, deleted
    created_at TEXT NOT NULL,
    read_at TEXT,
    expires_at TEXT
);
```

## GitHub Configuration

File: `~/.ailang/config.yaml`

```yaml
github:
  expected_user: YourGitHubUsername   # REQUIRED
  default_repo: sunholo-data/ailang   # Default repo
  create_labels:                      # Added to created issues
    - ailang-message
  watch_labels:                       # Filter for import
    - ailang-message
  auto_import: true                   # Auto-import on session start
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Database error |
| 4 | GitHub error |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `AILANG_STATE_DIR` | Override state directory |
| `AILANG_CONFIG_PATH` | Override config file path |
