# Coordinator Helper Reference

Complete reference for the coordinator daemon and CLI commands.

## CLI Command Reference

### ailang coordinator start

Start the coordinator daemon.

```bash
ailang coordinator start [options]

Options:
  --poll-interval DURATION   How often to check for new tasks (default: 5s)
  --max-worktrees N          Maximum concurrent worktrees (default: 3)
  --state-dir DIR            State directory (default: ~/.ailang/state)
```

### ailang coordinator stop

Stop the coordinator daemon.

```bash
ailang coordinator stop [options]

Options:
  --state-dir DIR   State directory
```

### ailang coordinator status

Check if the daemon is running.

```bash
ailang coordinator status [options]

Options:
  --state-dir DIR   State directory
```

### ailang coordinator list

List tasks with interactive explorer.

```bash
ailang coordinator list [options]

Options:
  --limit N              Number of tasks to show (default: 10)
  --status STATUS,...    Filter by status (comma-separated)
  --pending              Shorthand for --status pending,queued,pending_approval
  --running              Shorthand for --status running
  --completed            Shorthand for --status completed
  --failed               Shorthand for --status failed,rejected,cancelled
  --json                 Output as JSON (non-interactive)
  --state-dir DIR        State directory

Interactive Actions:
  [1-N]   Select task by number
  [q]     Quit

Task Detail Actions:
  [d]     View full diff
  [s]     View diff summary (--stat)
  [f]     Browse changed files
  [b]     Browse worktree directory
  [l]     View execution logs
  [a]     Approve and merge (pending_approval only)
  [r]     Reject (pending_approval only)
  [q]     Back to list
```

### ailang coordinator approve

Approve a pending task and merge changes.

```bash
ailang coordinator approve <task-id> [options]

Options:
  --skip-merge       Approve without merging
  --keep-worktree    Don't remove worktree after merge
  --state-dir DIR    State directory

What it does:
  1. Verifies task is pending_approval
  2. Auto-commits any uncommitted changes in worktree
  3. Merges worktree branch to dev
  4. Marks task as completed
  5. Cleans up worktree and branch (unless --keep-worktree)
```

### ailang coordinator reject

Reject a pending task.

```bash
ailang coordinator reject <task-id> [options]

Options:
  --reason TEXT      Reason for rejection
  --state-dir DIR    State directory

What it does:
  1. Marks task as rejected
  2. Preserves worktree for reference
  3. Records rejection reason
```

### ailang coordinator pending

Show all pending approval tasks (shorthand for list --pending).

```bash
ailang coordinator pending [options]

Options:
  --json             Output as JSON
  --state-dir DIR    State directory
```

### ailang coordinator cleanup

Clean up stale worktrees and reset stuck tasks.

```bash
ailang coordinator cleanup [options]

Options:
  --dry-run          Show what would be cleaned without doing it
  --force            Force cleanup even if daemon is running
  --state-dir DIR    State directory
```

## Task Status Values

| Status | Description |
|--------|-------------|
| `pending` | Task received, waiting to be picked up by daemon |
| `queued` | In execution queue, waiting for available slot |
| `running` | Currently being executed by an agent |
| `pending_approval` | Execution complete, awaiting human review |
| `completed` | Approved and merged to dev branch |
| `failed` | Execution failed (check error field) |
| `rejected` | Human rejected the work |
| `cancelled` | Task was cancelled |

## Task Types

| Type | Executor | Description |
|------|----------|-------------|
| `bug-fix` | Claude Code CLI | Bug fixes requiring code changes |
| `feature` | Claude Code CLI | New feature implementation |
| `refactor` | Claude Code CLI | Code restructuring |
| `test` | Claude Code CLI | Writing or fixing tests |
| `docs` | Gemini API | Documentation writing |
| `research` | Gemini API | Investigation and exploration |

## Database Schema

Tasks are stored in `~/.ailang/state/coordinator.db`:

```sql
-- Task records
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    thread_id TEXT,
    title TEXT NOT NULL,
    content TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    provider TEXT,
    worktree_path TEXT,
    error TEXT,
    cost REAL DEFAULT 0,
    tokens_used INTEGER DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME,
    started_at DATETIME,
    completed_at DATETIME
);

-- Task events (execution logs)
CREATE TABLE task_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    thread_id TEXT,
    stream_type TEXT NOT NULL,
    turn_num INTEGER,
    text TEXT,
    tool_name TEXT,
    tool_input TEXT,
    tool_output TEXT,
    error_msg TEXT,
    status TEXT,
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost REAL,
    duration_sec INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Event Stream Types

Events stored in `task_events` table:

| Type | Fields | Description |
|------|--------|-------------|
| `status` | status | Task status change |
| `turn_start` | turn_num | Agent turn started |
| `turn_end` | turn_num | Agent turn ended |
| `text` | text, turn_num | Text output from agent |
| `tool_use` | tool_name, tool_input, turn_num | Tool invocation |
| `tool_result` | tool_output, turn_num | Tool result |
| `error` | error_msg | Error occurred |

## Configuration

The coordinator uses settings from `~/.ailang/config.yaml`:

```yaml
coordinator:
  poll_interval: 5s
  max_worktrees: 3
  task_timeout: 30m
  auto_cleanup: true

  # Provider preferences
  providers:
    code: claude      # For code tasks
    docs: gemini      # For docs tasks
    research: gemini  # For research tasks
```

## Storage Locations

| Path | Purpose |
|------|---------|
| `~/.ailang/state/coordinator.db` | SQLite database (tasks, events) |
| `~/.ailang/state/worktrees/coordinator/` | Git worktrees for tasks |
| `~/.ailang/logs/coordinator.log` | Daemon logs |

## Integration with Dashboard

The coordinator streams events to the Collaboration Hub server:

1. Start server: `ailang serve` or `make services-start`
2. Events POST to: `http://127.0.0.1:1957/api/coordinator/events`
3. Server broadcasts via WebSocket to browsers
4. View at: http://localhost:1957

## Examples

### Delegate a bug fix

```bash
ailang messages send coordinator \
    "Fix the type error in internal/parser/parser.go line 42. The issue is that we're comparing int64 to int." \
    --title "Bug: Type mismatch in parser" \
    --from "claude-code" \
    --type bug
```

### Check and approve

```bash
# See pending approvals
ailang coordinator list --pending

# Interactive review
ailang coordinator list
# Select task number, review diff [d], approve [a]
```

### Bulk operations via JSON

```bash
# Get all pending tasks
ailang coordinator list --status pending_approval --json | jq '.[].id'

# Approve all (use with caution!)
for id in $(ailang coordinator list --status pending_approval --json | jq -r '.[].id'); do
    ailang coordinator approve "$id"
done
```
