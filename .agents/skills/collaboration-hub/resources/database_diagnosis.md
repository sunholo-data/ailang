# Observatory Database Diagnosis

SQLite queries and tools for diagnosing issues with the Observatory database.

## Database Location

```
~/.ailang/state/observatory.db
```

## Schema Overview

### Core Entities

```sql
-- Workspaces
workspaces (
  id TEXT PRIMARY KEY,
  name TEXT,
  path TEXT,
  git_remote TEXT,
  created_at DATETIME,
  updated_at DATETIME
)

-- Tasks
tasks (
  id TEXT PRIMARY KEY,
  workspace_id TEXT,
  parent_task_id TEXT,
  title TEXT,
  description TEXT,
  source_type TEXT,
  source_ref TEXT,
  status TEXT,
  priority INTEGER,
  created_at DATETIME,
  started_at DATETIME,
  completed_at DATETIME,
  -- Aggregated metrics:
  total_duration_ms INTEGER,
  total_tokens_in INTEGER,
  total_tokens_out INTEGER,
  total_cost_usd REAL,
  agent_count INTEGER,
  span_count INTEGER,
  error_count INTEGER
)

-- Agent Assignments
agent_assignments (
  id TEXT PRIMARY KEY,
  task_id TEXT,
  agent_id TEXT,
  provider TEXT,
  status TEXT,
  assigned_at DATETIME,
  started_at DATETIME,
  completed_at DATETIME,
  parent_assignment_id TEXT,
  duration_ms INTEGER,
  tokens_in INTEGER,
  tokens_out INTEGER,
  cost_usd REAL,
  tool_calls INTEGER,
  turns INTEGER
)
```

### Telemetry Data

```sql
-- Spans
spans (
  id TEXT PRIMARY KEY,
  trace_id TEXT,
  parent_span_id TEXT,
  task_id TEXT,
  agent_assignment_id TEXT,
  name TEXT,
  kind TEXT,
  status TEXT,
  status_message TEXT,
  start_time DATETIME,
  end_time DATETIME,
  duration_ms INTEGER,
  tokens_in INTEGER,
  tokens_out INTEGER,
  cost_usd REAL,
  model TEXT,
  provider TEXT,
  attributes TEXT,      -- JSON
  resource_attributes TEXT,  -- JSON
  created_at DATETIME
)

-- Span Events
span_events (
  id TEXT PRIMARY KEY,
  span_id TEXT,
  name TEXT,
  timestamp DATETIME,
  attributes TEXT,      -- JSON
  event_type TEXT,
  approval_status TEXT,
  tool_name TEXT,
  error_message TEXT
)
```

### Messaging

```sql
messages (
  id TEXT PRIMARY KEY,
  task_id TEXT,
  inbox TEXT,
  from_agent TEXT,
  -- ... additional fields
)
```

## Quick Diagnostic Queries

### Database Health

```bash
# Total counts
sqlite3 ~/.ailang/state/observatory.db "SELECT
  (SELECT COUNT(*) FROM workspaces) as workspaces,
  (SELECT COUNT(*) FROM tasks) as tasks,
  (SELECT COUNT(*) FROM spans) as spans,
  (SELECT COUNT(*) FROM agent_assignments) as agents"

# Recent spans by provider
sqlite3 ~/.ailang/state/observatory.db "SELECT provider, COUNT(*), SUM(cost_usd)
  FROM spans WHERE provider IS NOT NULL GROUP BY provider"

# Spans without task_id (orphans)
sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM spans WHERE task_id IS NULL"
```

### Find Spans for a Task

```bash
# All spans linked to a task
sqlite3 ~/.ailang/state/observatory.db "SELECT name, trace_id, parent_span_id
  FROM spans WHERE task_id = 'eval-1768075123600532000' LIMIT 10"

# Trace IDs for a task (to find child spans)
sqlite3 ~/.ailang/state/observatory.db "SELECT DISTINCT trace_id
  FROM spans WHERE task_id = 'eval-1768075123600532000'"

# All spans in those traces (including children without task_id)
sqlite3 ~/.ailang/state/observatory.db "SELECT name, task_id, parent_span_id
  FROM spans WHERE trace_id IN (
    SELECT DISTINCT trace_id FROM spans WHERE task_id = 'eval-1768075123600532000'
  ) ORDER BY start_time"
```

### Check Span Hierarchy

```bash
# Root spans (no parent)
sqlite3 ~/.ailang/state/observatory.db "SELECT name, trace_id
  FROM spans WHERE parent_span_id IS NULL OR parent_span_id = '' LIMIT 10"

# Children of a specific span
sqlite3 ~/.ailang/state/observatory.db "SELECT name, id
  FROM spans WHERE parent_span_id = 'SPAN_ID_HERE'"
```

### Cost Analysis

```bash
# Cost by model
sqlite3 ~/.ailang/state/observatory.db "SELECT model,
  COUNT(*), SUM(tokens_in), SUM(tokens_out), SUM(cost_usd)
  FROM spans WHERE model IS NOT NULL GROUP BY model ORDER BY SUM(cost_usd) DESC"

# Cost by day
sqlite3 ~/.ailang/state/observatory.db "SELECT DATE(start_time), SUM(cost_usd)
  FROM spans GROUP BY DATE(start_time) ORDER BY 1 DESC LIMIT 7"
```

### Task ID Format

Tasks are created with format `task-{first 8 chars of message UUID}`:

```
Message ID: 50b9518d-53f0-4dd7-893e-779ac90672b6
Task ID:    task-50b9518d
```

## Troubleshooting

### Task not found

```bash
sqlite3 ~/.ailang/state/observatory.db "SELECT * FROM tasks WHERE id = 'task-XXXXXXXX'"
```

### Check spans exist for task

```bash
sqlite3 ~/.ailang/state/observatory.db "SELECT name, trace_id FROM spans WHERE task_id = 'task-XXXXXXXX'"
```

### Verify span timestamps overlap

```bash
sqlite3 ~/.ailang/state/observatory.db "SELECT name, start_time,
  datetime(end_time) as end_time FROM spans
  WHERE trace_id = 'TRACE_ID' ORDER BY start_time"
```

### Clear old data

```bash
sqlite3 ~/.ailang/state/observatory.db "DELETE FROM spans; DELETE FROM span_events; DELETE FROM agent_assignments; DELETE FROM tasks; DELETE FROM workspaces;"
```

## Related Files

| File | Purpose |
|------|---------|
| `internal/observatory/store.go` | SQLite implementation |
| `internal/observatory/backend.go` | Backend interface |
| `internal/observatory/api.go` | REST API handlers |
| `internal/observatory/hierarchy.go` | Hierarchy building |
