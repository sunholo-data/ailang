# Agent Inbox Command Reference

Complete reference for the AILANG messaging system CLI commands.

## Command Overview

| Command | Alias | Purpose |
|---------|-------|---------|
| `ailang messages list` | `msg ls` | List messages |
| `ailang messages read` | `msg read` | Read full message |
| `ailang messages ack` | `msg ack` | Mark as read |
| `ailang messages unack` | `msg unack` | Mark as unread |
| `ailang messages send` | `msg send` | Send message (with optional envelope) |
| `ailang messages search` | `msg search` | Semantic search (with optional envelope space) |
| `ailang messages triage` | `msg triage` | Cluster unread messages by envelope similarity |
| `ailang messages dedupe` | `msg dedupe` | Find and mark duplicate messages |
| `ailang messages reply` | `msg reply` | Reply to GitHub issue |
| `ailang messages watch` | `msg watch` | Watch for new |
| `ailang messages cleanup` | `msg cleanup` | Remove old messages |
| `ailang messages import-github` | - | Import from GitHub |
| `ailang pkg notify-upgrade` | - | Emit upgrade-available message |
| `ailang pkg affected-by` | - | List workspaces depending on package |

### Package-Scoped Inboxes

Messages support typed inbox addressing for package coordination:
- `pkg:vendor/name` — e.g., `pkg:sunholo/auth`
- `workspace:name` — e.g., `workspace:docparse`
- `team:name` — e.g., `team:registry-admin`

Use `--inbox pkg:sunholo/auth` with list/search to filter package messages.

11 package message kinds: `upgrade-available`, `interface-change-notice`, `effect-widening-warning`, `compatibility-request`, `compatibility-report`, `contract-regression`, `migration-request`, `deprecation-notice`, `upgrade-complete`, `blocked`, `superseded`.

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
| `--envelope-code FILES` | Comma-separated file paths for code envelope slot (v0.8.1+) | none |
| `--envelope-context DESC` | Context description for context envelope slot (v0.8.1+) | none |
| `--github` | Create GitHub issue | false |
| `--type TYPE` | Category (any string; bug/feature imply --github) | none |
| `--repo OWNER/REPO` | GitHub repo | config default |

**Examples:**
```bash
# Basic local message
ailang messages send user "Task complete" --title "Done" --from "agent"

# With semantic envelope (v0.8.1+)
ailang messages send executor "Fix parser bug" --title "Bug" \
  --envelope-code internal/parser/parser.go
ailang messages send executor "Fix type system" \
  --envelope-code internal/types/unify.go,internal/types/subst.go
ailang messages send executor "Fix crash" --title "Bug" \
  --envelope-context "reviewing ast.Type switches, found missing TypeVar case"
ailang messages send executor "Fix bug" \
  --envelope-code internal/iface/builder.go \
  --envelope-context "constructor type variables are TypeVar not SimpleType"

# Bug report (--type bug implies --github)
ailang messages send user "Parser crashes on nested records" \
  --title "Parser bug" --type bug

# Feature request (--type feature implies --github)
ailang messages send user "Need async support" \
  --title "Async" --type feature

# Custom type (local only, no GitHub sync)
ailang messages send user "Research findings" --type research --from "agent"

# Custom type WITH GitHub sync (explicit --github)
ailang messages send user "Docs update" --type docs --github
```

## Search Messages

```bash
ailang messages search "QUERY" [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--inbox NAME` | Filter by inbox | all |
| `--threshold N` | Minimum similarity (0.0-1.0) | 0.70 |
| `--limit N` | Maximum results | 20 |
| `--max-scan N` | Maximum messages to scan | 1000 |
| `--neural` | Use neural embeddings (requires Ollama) | false |
| `--simhash` | Force SimHash mode | true |
| `--space SLOT` | Search a specific envelope space (v0.8.1+) | none |
| `--json` | JSON output | false |

**Envelope spaces:** `intent`, `code`, `context`, `skill`, `resolution`

**Examples:**
```bash
# Basic semantic search
ailang messages search "parser error"
ailang messages search "type inference" --neural

# Search by envelope space (v0.8.1+)
ailang messages search --space code "internal/types"
ailang messages search --space intent "fix crash"
ailang messages search --space resolution "parser"
```

## Triage Messages (v0.8.1+)

```bash
ailang messages triage [flags]
```

Clusters unread messages by envelope similarity.

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--inbox NAME` | Filter by inbox | all |
| `--cluster-by SLOT` | Envelope slot to cluster on | intent |
| `--top N` | Show top-N clusters | 10 |
| `--threshold N` | Minimum similarity for clustering | 0.75 |
| `--json` | JSON output | false |

**Examples:**
```bash
ailang messages triage                        # Cluster by intent
ailang messages triage --cluster-by code      # Cluster by code region
ailang messages triage --inbox user --top 5   # Top 5 clusters in user inbox
ailang messages triage --json                 # JSON output
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
    category TEXT,                -- any string (bug/feature have special behavior)
    github_issue_number INTEGER,  -- Linked GitHub issue
    github_repo TEXT,             -- owner/repo
    status TEXT NOT NULL,         -- unread, read, archived, deleted
    envelope TEXT DEFAULT '{}',   -- JSON: named embedding vectors (v0.8.1+)
    created_at TEXT NOT NULL,
    read_at TEXT,
    expires_at TEXT
);
```

### Envelope Schema (v0.8.1+)

The `envelope` column stores a JSON object with named embedding slots:

```json
{
  "slots": {
    "intent": {
      "vector": [0.123, 0.456, ...],
      "model": "ollama:nomic-embed-text",
      "dimension": 768
    },
    "code": {
      "vector": [0.789, 0.012, ...],
      "model": "ollama:nomic-embed-text",
      "dimension": 768
    }
  }
}
```

**Slots:** `intent`, `code`, `context`, `skill`, `resolution`

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
