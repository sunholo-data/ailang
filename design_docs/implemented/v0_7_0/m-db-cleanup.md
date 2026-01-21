# M-DB-CLEANUP: Database Schema Cleanup and Consolidation

**Status:** Implemented (v0.6.3)
**Author:** Claude Code
**Date:** 2026-01-16
**Implemented:** 2026-01-16

## Summary

AILANG uses 3 SQLite databases with overlapping schemas that cause confusion for AI agents. This design doc captures the cleanup work done and proposes further consolidation.

## Current State (Post Phase 1)

### Database Architecture

| Database | Size | Purpose | Tables |
|----------|------|---------|--------|
| `collaboration.db` | ~500KB | Messaging, threads, approvals | 10 |
| `coordinator.db` | ~4MB | Task execution, state machine | 3 |
| `observatory.db` | ~320MB | Telemetry spans, sessions | 6 |

### Phase 1 Completed: Dead Table Removal

**Removed tables (0 rows, no API usage):**
- `collaboration.db:attachments` - designed for large payloads, never implemented
- `observatory.db:span_events` - designed for OTEL events, never implemented

**Migration versions:**
- `collaboration.db`: v1.5.0 → v1.6.0
- `observatory.db`: v3 → v4

## Remaining Issues

### Issue 1: Naming Overlap Causes AI Confusion

| Concept | collaboration.db | coordinator.db | observatory.db |
|---------|-----------------|----------------|----------------|
| Tasks | - | `tasks` (execution) | `tasks` (metrics) |
| Messages | `inbox_messages` | - | `messages` |
| Approvals | `approvals` | `approval_requests` | - |

**Problem:** An AI asked to "query the tasks table" doesn't know which database to use.

### Issue 2: Duplicate Approval Systems

Two separate approval tables exist:
1. `collaboration.db:approvals` - Used by server handlers for effect-gated approvals
2. `coordinator.db:approval_requests` - Used by coordinator daemon for task approvals

Both have:
- Similar schemas (id, status, reviewed_by, etc.)
- Different consumers (server API vs coordinator CLI)
- No cross-referencing

**Current workaround:** Server uses both stores via `s.store` and `s.approvalStore`.

### Issue 3: Duplicate Message Systems

Two message tables exist:
1. `collaboration.db:inbox_messages` - CLI messages (`ailang messages`)
2. `observatory.db:messages` - Observatory API messages

Both have:
- Similar schemas (inbox, from_agent, title, content, status)
- GitHub integration fields
- Embedding fields for search

**Key difference:** Observatory messages have `task_id` link; inbox_messages have `parent_task_id`.

## Proposed Phase 2: Schema Consolidation

### Option A: Rename for Clarity (Low Risk)

Keep 3 databases, rename tables to clarify purpose:

```sql
-- coordinator.db
ALTER TABLE tasks RENAME TO task_executions;

-- observatory.db
ALTER TABLE tasks RENAME TO task_metrics;
```

**Pros:** Minimal code changes, low risk
**Cons:** Doesn't eliminate duplication

### Option B: Consolidate Approvals (Medium Risk)

Merge approval systems into coordinator.db (single source of truth):

```sql
-- coordinator.db becomes the approval authority
-- collaboration.db:approvals → deprecated, redirect to coordinator

-- Add fields to coordinator.approval_requests:
ALTER TABLE approval_requests ADD COLUMN effect_delta_json TEXT;
ALTER TABLE approval_requests ADD COLUMN capability_token TEXT;
ALTER TABLE approval_requests ADD COLUMN token_expires_at INTEGER;
```

**Migration:**
1. Add new fields to coordinator.approval_requests
2. Update server handlers to use coordinator store only
3. Migrate any existing collaboration.approvals data
4. Drop collaboration.approvals in v1.7.0

**Pros:** Single approval source of truth
**Cons:** Requires server code changes, migration path needed

### Option C: Consolidate Messages (Higher Risk)

Merge message systems - use collaboration.db:inbox_messages as source of truth:

```sql
-- Add task_id to inbox_messages (already has parent_task_id)
ALTER TABLE inbox_messages ADD COLUMN task_id TEXT;

-- Observatory queries read from collaboration.db via adapter
```

**Migration:**
1. Add task_id to inbox_messages
2. Update observatory API to read via adapter
3. Drop observatory.messages

**Pros:** Single message store
**Cons:** Cross-database queries, adapter complexity

### Option D: Full Unification (Highest Risk)

Merge all three databases into single `ailang.db`:

```sql
-- ailang.db (unified)
CREATE TABLE entities (
    id TEXT PRIMARY KEY,
    entity_type TEXT,  -- 'task', 'message', 'span', 'approval'
    parent_id TEXT REFERENCES entities(id),
    ...
);
```

**Pros:** True single source of truth, real foreign keys
**Cons:** Major refactor, high migration risk, different data retention needs

## Recommendation

**Phase 2A (Q1 2026):** Implement Option A (rename for clarity)
- Low risk, immediate benefit for AI clarity
- No data migration needed

**Phase 2B (Q2 2026):** Implement Option B (consolidate approvals)
- Medium risk, eliminates duplicate approval logic
- Clear migration path

**Phase 2C (Future):** Evaluate Option C based on Phase 2B learnings

## Implementation Checklist

### Phase 1 (Completed)
- [x] Audit all databases and tables
- [x] Identify truly unused tables (0 rows, no API)
- [x] Add migrations to drop dead tables
- [x] Update tests
- [x] Create design doc

### Phase 2A (Planned)
- [ ] Rename coordinator.tasks → task_executions
- [ ] Rename observatory.tasks → task_metrics
- [ ] Update all code references
- [ ] Add migration with backward-compatible views

### Phase 2B (Planned)
- [ ] Design unified approval schema
- [ ] Implement migration from collaboration.approvals
- [ ] Update server handlers
- [ ] Remove collaboration.approvals

## Data Model Reference

### collaboration.db (v1.6.0)

```
schema_version
threads (id, title, status, target_agent, ...)
messages (id, thread_id, message_seq, kind, ...)
subscriptions (instance_id, thread_id, from_seq, ...)
approvals (id, thread_id, effect_delta_json, status, capability_token, ...)  # TODO: consolidate
approval_history (id, approval_id, action, actor, ...)
agents (id, label, status, config_json, ...)
metrics_aggregates (scope_type, scope_id, period, totals, ...)
instance_history (id, agent_id, instance_id, ...)
replay_snapshots (id, thread_id, model_id, seed, ...)
inbox_messages (id, message_id, to_inbox, from_agent, title, payload, ...)
```

### coordinator.db (v4)

```
tasks (id, message_id, thread_id, title, status, agent_id, worktree_path, ...)
approval_requests (id, task_id, type, description, status, ...)
task_events (id, task_id, stream_type, text, tool_name, ...)
```

### observatory.db (v4)

```
workspaces (id, name, path, git_remote, ...)
tasks (id, workspace_id, parent_task_id, title, source_type, status, totals, ...)
agent_assignments (id, task_id, agent_id, provider, status, metrics, ...)
spans (id, trace_id, parent_span_id, name, status, tokens, cost, ...)
messages (id, task_id, inbox, from_agent, title, content, ...)  # TODO: consolidate with inbox_messages
sessions (session_id, workspace, claude_version, turn_count, ...)
session_tools (tool_use_id, session_id, tool_name, tool_input, ...)
```

## Cross-Database Correlation

Current linking strategy (soft references):
- `coordinator.tasks.message_id` → `collaboration.inbox_messages.id`
- `coordinator.tasks.thread_id` → `collaboration.threads.id`
- `observatory.spans.task_id` → `observatory.tasks.id` (same DB)
- `inbox_messages.parent_task_id` → `coordinator.tasks.id`

No foreign key constraints (different databases). Sync maintained by:
- Message watcher creates tasks from inbox_messages
- `coordinatorSyncThreads` updates thread target_agent
- Timestamp correlation links spans to tasks

## Observatory Data Retention Strategy

### Data Analysis (2026-01-16)

**Current state:** 320MB, 217K spans, ~50K new spans/day

| Category | Count | % | Retention |
|----------|-------|---|-----------|
| operational_noise | 193,630 | 89% | 7 days (or disable) |
| tool_usage | 19,056 | 8.7% | 30 days |
| compilation | 1,968 | 0.9% | 90 days |
| execution | 1,304 | 0.6% | 90 days |
| llm_calls | 38 | 0.02% | Indefinite |

### Span Classification

```sql
-- operational_noise: Server polling, health checks
name IN ('ailang-server', 'api_request', 'messages.list', 'messages.send')

-- tool_usage: Agent tool calls
name LIKE 'claude_code.tool.%'

-- compilation: Compiler pipeline
name LIKE 'compile.%'

-- execution: Task execution
name LIKE 'eval.%' OR name LIKE 'executor.%'

-- llm_calls: AI provider calls (has cost/token data)
name LIKE 'anthropic.%' OR name LIKE 'openai.%' OR name LIKE 'gemini.%' OR name LIKE 'ollama.%'
```

### Recommended Cleanup SQL

```sql
-- Run periodically (e.g., daily via cron or coordinator)
-- Delete orphan noise spans older than 7 days
DELETE FROM spans
WHERE task_id IS NULL
AND created_at < datetime('now', '-7 days')
AND name NOT LIKE 'anthropic.%'
AND name NOT LIKE 'openai.%'
AND name NOT LIKE 'gemini.%'
AND name NOT LIKE 'ollama.%';

-- Delete old tool usage spans (30 days)
DELETE FROM spans
WHERE name LIKE 'claude_code.tool.%'
AND created_at < datetime('now', '-30 days');

-- Delete old compilation spans (90 days)
DELETE FROM spans
WHERE name LIKE 'compile.%'
AND created_at < datetime('now', '-90 days');
```

### Source Filtering (Implemented)

**Prevent noise spans from being created/stored in the first place:**

**server.go filter** (prevents span creation):
```go
// Endpoints now filtered at source:
- /api/controlplane/*     // 140K spans
- /api/coordinator/events // 15K spans
- /api/budget/*           // 2K spans
- /api/inbox              // 1K spans
```

**otlp_receiver.go filter** (prevents span storage):
```go
// Span names filtered by service:
- messages.list (ailang-coordinator, ailang-server)  // 10K spans
- messages.count, inbox.poll, agent.heartbeat        // coordinator only
```

**Kept for cost/debugging:**
- `api_request` (claude-code) - has cost/token data
- `messages.list` (ailang-messages) - deliberate CLI commands

### Implementation Options

1. **CLI command:** `ailang observatory cleanup --dry-run`
2. **Coordinator job:** Script agent that runs cleanup on schedule
3. **Auto-vacuum:** SQLite VACUUM after cleanup to reclaim space

### Actual Results (2026-01-16)

Cleanup executed with default retention (7-day noise, 30-day tools, 90-day compile):
- Removed 195,879 spans (90% reduction)
- Reduced DB size from 320MB to 101MB (68% reduction)
- Retained 22,331 spans (task-linked and cost-tracking data preserved)

```bash
# Command used
ailang observatory cleanup --vacuum
```

## Testing

Run migration tests:
```bash
go test ./internal/messaging/... ./internal/observatory/... -v
```

Verify databases after service restart:
```bash
sqlite3 ~/.ailang/state/collaboration.db ".tables"
sqlite3 ~/.ailang/state/coordinator.db ".tables"
sqlite3 ~/.ailang/state/observatory.db ".tables"
```
