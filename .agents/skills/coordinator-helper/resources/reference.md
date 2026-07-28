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
  [c]     View chat history (turn-by-turn with tool calls) (v0.6.4+)
  [d]     View full diff
  [s]     View diff summary (--stat)
  [f]     Browse changed files
  [b]     Browse worktree directory
  [l]     View execution logs
  [a]     Approve and merge (pending_approval only)
  [r]     Reject with feedback loop (pending_approval only) (v0.6.4+)
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

Reject a pending task with feedback loop support (v0.6.4+).

```bash
ailang coordinator reject <task-id> [options]

Options:
  --feedback, -f TEXT   Feedback explaining what needs revision
  --no-prompt           Skip interactive feedback prompt (use with --feedback)
  --no-retrigger        Permanent rejection without re-triggering
  --state-dir DIR       State directory

What it does (default - with feedback loop):
  1. Prompts for feedback reason (why the work needs revision)
  2. Stores feedback as human_feedback event in task_events
  3. Sends message to agent's inbox with feedback content
  4. Re-triggers task with iteration+1 (same task ID, preserves context)
  5. Agent resumes with --resume <sessionId> for full conversation history
  6. Max 3 iterations to prevent infinite loops

What it does (with --no-retrigger):
  1. Marks task as permanently rejected
  2. Preserves worktree for reference
  3. Records rejection reason
```

**Feedback Loop Example:**
```bash
# Interactive (prompts for feedback)
ailang coordinator reject task-abc123

# With inline feedback
ailang coordinator reject task-abc123 --feedback "Need to add error handling for edge cases"

# Permanent rejection (no re-trigger)
ailang coordinator reject task-abc123 --no-retrigger
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

### ailang coordinator worktree

Show or open the worktree directory for a task.

```bash
ailang coordinator worktree <task-id> [options]

Options:
  --open, -o         Open worktree in file manager (Finder on macOS)
  --cd               Output shell command to cd into worktree
  --state-dir DIR    State directory

Examples:
  ailang coordinator worktree task-abc123          # Print path
  ailang coordinator worktree task-abc123 --open   # Open in Finder
  cd $(ailang coordinator worktree task-abc123)    # cd into worktree
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
    session_id TEXT,           -- Claude/Gemini session for resumption (v0.6.4+)
    iteration INTEGER DEFAULT 0, -- Feedback loop iteration (v0.6.4+)
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

## Approval Types (v0.6.5+)

| Type | CLI Display | On Approve |
|------|-------------|------------|
| `merge` | `[merge]` | Merges code to dev branch only |
| `merge_handoff` | `[merge+handoff] → agent` | Merges code AND triggers next agent |
| `handoff` | `[handoff]` | (legacy) Sends to next agent only |

**Combined approvals** are created automatically when agent has `trigger_on_complete` with `auto_approve_handoffs: false`. This lets you review code AND approve handoff in one action.

**Context JSON for merge_handoff:**
```json
{
  "handoff_targets": ["sprint-planner"],
  "session_id": "sess-abc123",
  "source_agent": "design-doc-creator"
}
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
| `human_feedback` | text, turn_num, status | Human rejection feedback (v0.6.4+) |
| `human_approval` | text, status | Human approval event (v0.6.4+) |
| `iteration_start` | turn_num, text | New iteration started after feedback (v0.6.4+) |
| `handoff_triggered` | text | Handoff sent to next agent (v0.6.5+) |

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

## Chat History API (v0.6.4+)

View task conversation history via REST API:

```bash
# Get all events for a task
curl http://localhost:1957/api/tasks/{task-id}/events

# Filter by event type
curl http://localhost:1957/api/tasks/{task-id}/events?type=text,tool_use

# Limit results
curl http://localhost:1957/api/tasks/{task-id}/events?limit=50

# Get formatted text output (alternative endpoint)
curl http://localhost:1957/api/tasks/{task-id}/transcript
```

**Response format:**
```json
[
  {
    "id": 1,
    "task_id": "task-abc123",
    "stream_type": "turn_start",
    "turn_num": 1,
    "created_at": "2026-01-13T08:00:00Z"
  },
  {
    "id": 2,
    "task_id": "task-abc123",
    "stream_type": "text",
    "text": "Let me analyze the code...",
    "turn_num": 1,
    "created_at": "2026-01-13T08:00:01Z"
  },
  {
    "id": 3,
    "task_id": "task-abc123",
    "stream_type": "tool_use",
    "tool_name": "Read",
    "tool_input": "{\"file_path\": \"/path/to/file.go\"}",
    "turn_num": 1,
    "created_at": "2026-01-13T08:00:02Z"
  }
]
```

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
