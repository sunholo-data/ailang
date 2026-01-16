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

# Semantic search
ailang messages search "parser error"
ailang messages search "bugs" --neural    # Uses Ollama embeddings

# Find duplicates
ailang messages dedupe
ailang messages dedupe --apply            # Mark duplicates
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
  "simhash": 1234567890123456789,
  "dup_of": "original_message_id",
  "embedding": "[0.123, 0.456, ...]",
  "embedding_model": "ollama:nomic-embed-text",
  "status": "unread",
  "created_at": "2025-12-10T12:34:56Z"
}
```

### Semantic Search Fields (v0.5.11+)

| Field | Description |
|-------|-------------|
| `simhash` | 64-bit locality-sensitive hash for fast similarity |
| `dup_of` | Message ID this is a duplicate of (set by dedupe) |
| `embedding` | JSON-encoded float32 vector (neural search) |
| `embedding_model` | Model used to generate embedding (e.g., "ollama:nomic-embed-text") |

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

### Reply to GitHub Issues

Add comments to existing GitHub issue threads:

```bash
# Reply to a message that has a linked GitHub issue
ailang messages reply MSG_ID "Fixed in v0.5.10" --from "claude-code"

# Reply with explicit repo override
ailang messages reply MSG_ID "Working on it" --repo owner/repo
```

The reply command only works for messages created with `--github` flag. It adds a comment to the same issue thread, keeping the conversation together.

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

## Semantic Search

Find messages by meaning, not just exact text matches. AILANG uses **SimHash** (locality-sensitive hashing) for fast, zero-cost semantic search.

### Search Commands

```bash
# Search for messages by semantic content
ailang messages search "parser error handling"
ailang messages search "type inference bugs" --threshold 0.5

# Find messages similar to a specific message
ailang messages list --similar-to MSG_ID --threshold 0.70

# Hide duplicate messages (collapsed view)
ailang messages list --collapsed

# Show only duplicates of a specific message
ailang messages list --duplicates-of MSG_ID
```

### Search Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--threshold` | 0.70 | Minimum similarity (0.0-1.0) |
| `--limit` | 20 | Maximum results |
| `--max-scan` | 1000 | Maximum messages to scan |
| `--inbox` | (all) | Filter by inbox |
| `--neural` | false | Use neural embeddings (requires Ollama) |
| `--simhash` | true | Force SimHash mode (default, fast) |
| `--json` | false | Output as JSON |

### How SimHash Works

SimHash generates a 64-bit fingerprint for each message based on word frequencies. Similar messages have similar fingerprints, allowing fast similarity comparison using Hamming distance:

- **Score 1.0**: Identical or near-identical messages
- **Score 0.9+**: Very similar (likely duplicates)
- **Score 0.7-0.9**: Related topics
- **Score below 0.7**: Different content

**Benefits:**
- ✅ Zero API costs (runs locally)
- ✅ Fast (O(1) comparison)
- ✅ Deterministic (same input → same output)
- ✅ Works offline

## Deduplication

Find and mark duplicate messages to reduce inbox noise.

### Dedupe Commands

```bash
# Report duplicates (dry run)
ailang messages dedupe

# Report with custom threshold
ailang messages dedupe --threshold 0.90

# Actually mark duplicates
ailang messages dedupe --apply

# Filter by inbox
ailang messages dedupe --inbox user --apply
```

### How Deduplication Works

1. **Find groups**: Messages with similarity ≥ threshold are grouped
2. **Select representative**: Oldest message in each group is kept
3. **Mark duplicates**: Newer messages get `dup_of` set to representative's ID
4. **View behavior**: `--collapsed` hides messages with `dup_of` set

### Dedupe Report

```
Duplicate Report

Found 3 duplicate groups (7 messages)

Group 1 (95% similar, 2 duplicates):
  * Keep: msg_20251210_123456_abc123
    Parser crashes on nested records
  - Archive: msg_20251210_134500_def456
    Parser crash with nested records
  - Archive: msg_20251210_145600_ghi789
    Nested record parser crash
```

## Neural Embeddings (Ollama)

For more sophisticated semantic search, use neural embeddings via local Ollama.

### Prerequisites

1. Install Ollama: https://ollama.ai
2. Start Ollama server: `ollama serve`
3. Pull an embedding model:
   ```bash
   ollama pull nomic-embed-text      # Fast, good quality
   ollama pull embeddinggemma        # Google's embedding model
   ollama pull mxbai-embed-large     # High quality, slower
   ```

### Configuration

Create or update `~/.ailang/config.yaml`:

```yaml
embeddings:
  # Provider: "ollama" (local) or "none" (SimHash only)
  provider: ollama

  ollama:
    # Model name - see 'ollama list' for available models
    model: nomic-embed-text

    # Ollama API endpoint
    endpoint: http://localhost:11434

    # Request timeout
    timeout: 30s

  search:
    # Default search mode: "simhash" (fast) or "neural" (semantic)
    default_mode: simhash

    # Similarity thresholds (0.0-1.0)
    simhash_threshold: 0.70
    neural_threshold: 0.75
```

### Environment Variables

Override config with environment variables:

```bash
# Provider
export AILANG_EMBED_PROVIDER=ollama

# Model
export AILANG_OLLAMA_MODEL=nomic-embed-text

# Endpoint
export AILANG_OLLAMA_ENDPOINT=http://localhost:11434
```

### Using Neural Search

```bash
# Require Ollama to be running
ailang messages search "parser bugs" --neural

# Compare: SimHash (default, fast)
ailang messages search "parser bugs" --simhash
```

### How Neural Search Works

1. **Query embedding**: Your search query is converted to a vector
2. **Lazy embedding**: Messages without embeddings are embedded on-demand (up to 50 per search)
3. **Cosine similarity**: Vectors are compared using cosine similarity
4. **Cached**: Embeddings are stored in the database for reuse

**Benefits:**
- ✅ Understands semantic meaning ("error" matches "bug", "crash", "failure")
- ✅ Cross-lingual potential (with multilingual models)
- ✅ Better for conceptual search

**Trade-offs:**
- ⚠️ Requires Ollama running locally
- ⚠️ First search embeds messages (slower startup)
- ⚠️ Uses more storage (768+ floats per message)

### Model Recommendations

| Model | Dimension | Speed | Quality | Use Case |
|-------|-----------|-------|---------|----------|
| `nomic-embed-text` | 768 | Fast | Good | General purpose |
| `mxbai-embed-large` | 1024 | Medium | Better | High accuracy needs |
| `embeddinggemma` | 768 | Fast | Good | Google model |
| `all-minilm` | 384 | Very Fast | OK | Quick searches |

### Checking Ollama Status

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# List available models
ollama list

# Pull a model
ollama pull nomic-embed-text
```

## Aliases

The `messages` command has an alias for convenience:

```bash
ailang msg list        # Same as: ailang messages list
ailang msg send ...    # Same as: ailang messages send ...
```

## See Also

- [Collaboration Hub](/docs/guides/collaboration-hub) - Web UI shares the same SQLite database
- [Coordinator Guide](/docs/guides/coordinator) - Message routing and task execution
- [Semantic Search](/docs/guides/semantic-search) - Deep dive on SimHash and neural embeddings
- [Semantic Caching vs Vector DBs](/docs/guides/semantic-caching-vs-vectordb) - When to use semantic caching
- [Cross-Project Messaging](/docs/guides/cross-project-messaging) - Send feedback from your projects
- [Agent Workflows](/docs/guides/agent-workflows) - Automated agent workflows
- [State System](/docs/guides/state-system-workflow) - Persistent state management
