# M-TASK-HIERARCHY: Task-to-Trace Linking

**Status**: Planned
**Priority**: P0 (Key Feature)
**Target**: v0.6.5
**Estimated**: 4 days (32 hours)
**Dependencies**: M-OTEL-DASHBOARD (v0.6.4)
**Created**: 2026-01-05

## Related Documents

- [M-OTEL-DASHBOARD Implementation](../../implemented/v0_6_4/m-otel-dashboard.md) - Foundation this builds on
- [M-GEMINI-TRACE Investigation](../v0_6_4/m-gemini-trace-investigation.md) - Related Gemini telemetry work

## Executive Summary

Connect coordinator tasks with Observatory traces to build the full entity hierarchy:

```
WORKSPACE → TASK → AGENT_ASSIGNMENT → SPANS → EVENTS
```

Currently, we have 2,193 spans in Observatory and 39 tasks in Coordinator, but they're not linked. This feature bridges that gap by:

1. Syncing coordinator entities to Observatory
2. Passing `ailang.task_id` through the execution chain
3. Extracting task context from telemetry attributes

**This is the key feature** that transforms Observatory from a flat trace viewer into a true unified observability platform.

## Current State Analysis

### What We Have

| Component | Location | Count | Status |
|-----------|----------|-------|--------|
| Spans | `observatory.db` | 2,193 | Working, but task_id = NULL |
| Tasks | `coordinator.db` | 39 | Working, isolated |
| Agent Assignments | `coordinator.db` | ~50 | Working, isolated |
| Workspaces | None | 0 | Not created |

### The Gap

```
┌─────────────────────────────────────────────────────────────────┐
│  COORDINATOR (coordinator.db)     │  OBSERVATORY (observatory.db) │
├───────────────────────────────────┼───────────────────────────────┤
│  tasks                            │  spans (task_id = NULL)       │
│  agent_assignments                │  agent_assignments (empty)    │
│  worktrees                        │  workspaces (empty)           │
└───────────────────────────────────┴───────────────────────────────┘
                    ↑                              ↑
                    └── NO CONNECTION ─────────────┘
```

### Root Causes

1. **Coordinator doesn't emit task_id to executors** - Claude Code/Gemini don't know which task they're working on
2. **No cross-database sync** - Coordinator and Observatory use separate SQLite files
3. **OTLP receiver doesn't extract task context** - Even if present, task_id wouldn't be stored

## Solution Design

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           COORDINATOR                                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐              │
│  │ Task Created │───▶│ Sync to Obs │───▶│ Observatory │              │
│  └─────────────┘    └─────────────┘    │   Backend   │              │
│         │                               └──────┬──────┘              │
│         ▼                                      │                     │
│  ┌─────────────────────────────────────┐       │                     │
│  │        Execute with Context          │       │                     │
│  │  OTEL_RESOURCE_ATTRIBUTES:           │       │                     │
│  │    ailang.task_id=task-abc123        │       │                     │
│  │    ailang.workspace=/path/to/repo    │       │                     │
│  │    ailang.agent_id=design-doc-creator│       │                     │
│  └───────────────┬─────────────────────┘       │                     │
│                  │                              │                     │
└──────────────────┼──────────────────────────────┼─────────────────────┘
                   │                              │
                   ▼                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        CLAUDE CODE / GEMINI                          │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Spans emitted with inherited resource attributes:           │    │
│  │    service.name: claude-code                                 │    │
│  │    ailang.task_id: task-abc123      ◀── FROM COORDINATOR    │    │
│  │    ailang.workspace: /path/to/repo  ◀── FROM COORDINATOR    │    │
│  │    ailang.agent_id: design-doc-creator                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        OBSERVATORY                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  OTLP Receiver extracts:                                     │    │
│  │    task_id = attrs["ailang.task_id"]                         │    │
│  │    agent_assignment_id = lookup(task_id, agent_id)           │    │
│  │                                                               │    │
│  │  Stores span WITH task_id populated                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Coordinator creates task** → Syncs to Observatory
2. **Coordinator creates agent_assignment** → Syncs to Observatory
3. **Coordinator spawns executor** with env vars:
   ```bash
   OTEL_RESOURCE_ATTRIBUTES="ailang.task_id=task-abc,ailang.agent_id=design-doc-creator,ailang.workspace=/path"
   ```
4. **Executor (Claude/Gemini) emits spans** with inherited attributes
5. **OTLP Receiver extracts** `ailang.task_id` and sets `span.task_id`
6. **UI queries** full hierarchy: Task → Agent → Spans

## Implementation Plan

### M1: Entity Sync Layer (~150 LOC, Day 1)

**Goal:** Sync coordinator entities to Observatory in real-time

**Tasks:**
- [ ] Create `ObservatorySync` interface in coordinator
- [ ] Implement sync on task create/update/complete
- [ ] Implement sync on agent_assignment create/update
- [ ] Create workspace from git repo on first task

**Files:**
- `internal/coordinator/observatory_sync.go` (~150 LOC)

**Code:**
```go
// ObservatorySync syncs coordinator entities to Observatory
type ObservatorySync struct {
    obsBackend observatory.Backend
}

func (s *ObservatorySync) SyncTask(task *Task) error {
    obsTask := &observatory.Task{
        ID:          task.ID,
        WorkspaceID: s.getOrCreateWorkspace(task.Workspace),
        Title:       task.Title,
        Description: task.Description,
        SourceType:  task.SourceType,
        SourceRef:   task.SourceRef,
        Status:      task.Status,
        CreatedAt:   task.CreatedAt,
        // ... aggregates updated via span ingestion
    }
    return s.obsBackend.CreateTask(ctx, obsTask)
}

func (s *ObservatorySync) SyncAgentAssignment(aa *AgentAssignment) error {
    obsAA := &observatory.AgentAssignment{
        ID:       aa.ID,
        TaskID:   aa.TaskID,
        AgentID:  aa.AgentID,
        Provider: aa.Provider,
        Status:   aa.Status,
        // ...
    }
    return s.obsBackend.CreateAgentAssignment(ctx, obsAA)
}
```

**Acceptance Criteria:**
- Tasks appear in Observatory DB when created in Coordinator
- Agent assignments sync on creation
- Workspaces auto-created from git repo paths

---

### M2: Context Propagation (~100 LOC, Day 1)

**Goal:** Pass task context to executors via OTEL resource attributes

**Tasks:**
- [ ] Modify executor spawn to set OTEL_RESOURCE_ATTRIBUTES
- [ ] Include task_id, agent_id, workspace in attributes
- [ ] Handle existing OTEL_RESOURCE_ATTRIBUTES (merge, don't replace)

**Files:**
- `internal/coordinator/executor.go` (modify, ~50 LOC)
- `internal/executor/claude/executor.go` (modify, ~30 LOC)
- `internal/executor/gemini/executor.go` (modify, ~20 LOC)

**Code:**
```go
func (c *Coordinator) executeTask(task *Task, aa *AgentAssignment) error {
    // Build OTEL resource attributes for context propagation
    otelAttrs := fmt.Sprintf(
        "ailang.task_id=%s,ailang.agent_id=%s,ailang.workspace=%s,ailang.assignment_id=%s",
        task.ID,
        aa.AgentID,
        task.Workspace,
        aa.ID,
    )

    // Merge with existing OTEL_RESOURCE_ATTRIBUTES if present
    existing := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
    if existing != "" {
        otelAttrs = existing + "," + otelAttrs
    }

    env := append(os.Environ(), "OTEL_RESOURCE_ATTRIBUTES="+otelAttrs)

    // Execute with enriched environment
    return c.executor.Execute(ctx, task, env)
}
```

**Acceptance Criteria:**
- Executors receive OTEL_RESOURCE_ATTRIBUTES with task context
- Claude Code/Gemini inherit attributes in their spans
- Attributes visible in Observatory span details

---

### M3: OTLP Receiver Enhancement (~100 LOC, Day 2)

**Goal:** Extract task context from span attributes and link to entities

**Tasks:**
- [ ] Extract `ailang.task_id` from resource attributes
- [ ] Extract `ailang.assignment_id` from resource attributes
- [ ] Set span.task_id and span.agent_assignment_id before storage
- [ ] Validate task/assignment exists (log warning if not)

**Files:**
- `internal/observatory/otlp_receiver.go` (modify, ~100 LOC)

**Code:**
```go
func (r *OTLPReceiver) extractTaskContext(resourceAttrs map[string]any) (taskID, assignmentID string) {
    if tid, ok := resourceAttrs["ailang.task_id"].(string); ok {
        taskID = tid
    }
    if aid, ok := resourceAttrs["ailang.assignment_id"].(string); ok {
        assignmentID = aid
    }
    return
}

func (r *OTLPReceiver) processResourceSpans(ctx context.Context, rs *tracepb.ResourceSpans) error {
    resourceAttrs := extractResourceAttrs(rs)
    taskID, assignmentID := r.extractTaskContext(resourceAttrs)

    for _, scopeSpans := range rs.ScopeSpans {
        for _, span := range scopeSpans.Spans {
            normalized := r.convertSpan(span, resourceAttrs)

            // Link to task hierarchy
            normalized.TaskID = taskID
            normalized.AgentAssignmentID = assignmentID

            if err := r.backend.CreateSpan(ctx, normalized); err != nil {
                return err
            }
        }
    }
    return nil
}
```

**Acceptance Criteria:**
- Spans ingested with task_id have it stored in DB
- Spans can be queried by task_id
- Warning logged if task_id references unknown task

---

### M4: Aggregation Updates (~150 LOC, Day 2)

**Goal:** Update task/agent aggregates when spans complete

**Tasks:**
- [ ] Trigger aggregation update on span insert
- [ ] Update task totals (tokens, cost, duration, span_count)
- [ ] Update agent_assignment totals
- [ ] Handle concurrent updates (use transactions)

**Files:**
- `internal/observatory/store.go` (modify, ~100 LOC)
- `internal/observatory/aggregations.go` (new, ~50 LOC)

**Code:**
```go
func (s *SQLiteStore) CreateSpan(ctx context.Context, span *Span) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Insert span
    if err := s.insertSpan(tx, span); err != nil {
        return err
    }

    // Update aggregates if task linked
    if span.TaskID != "" {
        if err := s.updateTaskAggregates(tx, span.TaskID, span); err != nil {
            log.Printf("Warning: failed to update task aggregates: %v", err)
        }
    }

    if span.AgentAssignmentID != "" {
        if err := s.updateAgentAggregates(tx, span.AgentAssignmentID, span); err != nil {
            log.Printf("Warning: failed to update agent aggregates: %v", err)
        }
    }

    return tx.Commit()
}

func (s *SQLiteStore) updateTaskAggregates(tx *sql.Tx, taskID string, span *Span) error {
    _, err := tx.Exec(`
        UPDATE tasks SET
            total_duration_ms = total_duration_ms + ?,
            total_tokens_in = total_tokens_in + ?,
            total_tokens_out = total_tokens_out + ?,
            total_cost_usd = total_cost_usd + ?,
            span_count = span_count + 1,
            error_count = error_count + CASE WHEN ? = 'error' THEN 1 ELSE 0 END
        WHERE id = ?
    `, span.DurationMs, span.TokensIn, span.TokensOut, span.CostUSD, span.Status, taskID)
    return err
}
```

**Acceptance Criteria:**
- Task totals update in real-time as spans complete
- Agent assignment totals accurate
- Concurrent span ingestion doesn't corrupt aggregates

---

### M5: Hierarchy API (~200 LOC, Day 3)

**Goal:** API endpoint returning full task hierarchy

**Tasks:**
- [ ] Create `/api/observatory/tasks/:id/hierarchy` endpoint
- [ ] Return nested structure: Task → Agents → Traces → Spans
- [ ] Include aggregates at each level
- [ ] Add depth parameter to limit nesting

**Files:**
- `internal/observatory/api.go` (modify, ~100 LOC)
- `internal/observatory/hierarchy.go` (new, ~100 LOC)

**Response:**
```json
{
  "task": {
    "id": "task-abc123",
    "title": "Implement semantic caching",
    "status": "running",
    "totals": {
      "tokens_in": 45000,
      "tokens_out": 12000,
      "cost_usd": 0.45,
      "duration_ms": 180000,
      "span_count": 23
    }
  },
  "agents": [
    {
      "id": "assign-001",
      "agent_id": "design-doc-creator",
      "provider": "claude",
      "status": "completed",
      "totals": { "tokens_in": 15000, "cost_usd": 0.15 },
      "traces": [
        {
          "trace_id": "abc123def456",
          "root_span": "executor.claude.execute",
          "span_count": 8,
          "duration_ms": 60000,
          "spans": [
            { "id": "span-1", "name": "api_request", "tokens_in": 5000 },
            { "id": "span-2", "name": "tool.Edit", "parent_id": "span-1" }
          ]
        }
      ]
    },
    {
      "id": "assign-002",
      "agent_id": "sprint-planner",
      "provider": "claude",
      "status": "running",
      "traces": []
    }
  ]
}
```

**Acceptance Criteria:**
- Full hierarchy returned for any task
- Nested spans preserve parent-child relationships
- Aggregates accurate at each level

---

### M6: UI Hierarchy View (~250 LOC, Day 3-4)

**Goal:** React component for hierarchical task view

**Tasks:**
- [ ] Create `useTaskHierarchy(taskId)` hook
- [ ] Create `TaskHierarchyView` component
- [ ] Expandable tree: Task → Agents → Traces → Spans
- [ ] Show aggregates and status at each level
- [ ] Link to existing trace detail view

**Files:**
- `ui/src/hooks/useTaskHierarchy.ts` (~80 LOC)
- `ui/src/features/observatory/TaskHierarchy.tsx` (~150 LOC)
- `ui/src/features/observatory/TaskHierarchy.module.css` (~50 LOC)

**Acceptance Criteria:**
- Hierarchy expands/collapses smoothly
- Aggregates visible without expanding
- Click on span opens detail panel
- Real-time updates via WebSocket

---

### M7: Backfill Tool (~100 LOC, Day 4)

**Goal:** Link existing spans to tasks using time correlation

**Tasks:**
- [ ] Create `ailang observatory backfill` command
- [ ] Match spans to tasks by time window + agent
- [ ] Preview mode (--dry-run) before applying
- [ ] Report linking statistics

**Files:**
- `cmd/ailang/observatory_backfill.go` (~100 LOC)

**Algorithm:**
```
For each task in coordinator:
  Find spans where:
    - start_time >= task.started_at
    - end_time <= task.completed_at (or now if running)
    - service.name matches expected provider
  Set span.task_id = task.id
```

**Acceptance Criteria:**
- Historical spans linked to tasks
- Dry-run shows what would be linked
- Statistics reported (X spans linked to Y tasks)

---

## Timeline

| Day | Milestones | Focus |
|-----|------------|-------|
| 1 | M1, M2 | Entity sync, context propagation |
| 2 | M3, M4 | OTLP extraction, aggregations |
| 3 | M5, M6 (start) | Hierarchy API, UI component |
| 4 | M6 (complete), M7 | UI polish, backfill tool |

## Success Criteria

- [ ] New coordinator tasks appear in Observatory
- [ ] Executor spans have task_id populated
- [ ] Task aggregates update in real-time
- [ ] `/api/observatory/tasks/:id/hierarchy` returns nested structure
- [ ] UI shows expandable task → agent → span hierarchy
- [ ] Historical spans can be backfilled
- [ ] All tests passing

## Risk Factors

| Risk | Impact | Mitigation |
|------|--------|------------|
| Claude Code doesn't propagate OTEL attrs | High | Test with explicit env vars first |
| Cross-database transactions | Medium | Accept eventual consistency |
| Performance of hierarchy query | Medium | Add caching, limit depth |
| Backfill accuracy | Low | Use conservative time windows |

## Testing Strategy

**Unit tests:**
- Entity sync functions (task sync, agent sync, workspace creation)
- Context extraction from OTEL attributes
- Aggregation calculations (tokens, cost, duration)
- Hierarchy query builder

**Integration tests:**
- Full flow: coordinator task → executor spawn → span ingestion → linked data
- Cross-database sync verification
- WebSocket updates on hierarchy changes
- Backfill tool with sample data

**End-to-end tests:**
- Create task via coordinator → verify appears in Observatory
- Execute real Claude Code task → verify spans linked
- UI hierarchy view loads and expands correctly

**Performance tests:**
- Hierarchy query with 1000+ spans
- Aggregation update latency
- WebSocket broadcast timing

## Non-Goals

**Not in this feature:**
- **Gemini CLI trace integration** - Blocked by [M-GEMINI-TRACE](../v0_6_4/m-gemini-trace-investigation.md) investigation
- **Cross-workspace queries** - Single workspace focus for now
- **Historical analytics** - No time-series aggregates yet
- **Cost budgeting/alerts** - Future feature
- **Task chains visualization** - Deferred to v0.6.6

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Task IDs provide deterministic trace correlation |
| A2: Replayability | +1 | Full hierarchy enables trace replay per task |
| A3: Effect Legibility | +1 | Makes AI operations visible through hierarchy |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Local task validation without global analysis |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured JSON API for programmatic access |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Aggregates tokens/cost at task level |
| A10: Composability | +1 | Task hierarchy composes with existing traces |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | +1 | Explicit boundary between coordinator and executors |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Task IDs are deterministic UUIDs, no randomness in linking
- [x] A3 (Effects): All operations explicitly tracked in traces
- [x] A4 (Authority): No new capabilities granted, uses existing OTEL attrs
- [x] A7 (Machines First): API-first design, UI is presentation layer

## Future Work

- **Multi-workspace view** - Dashboard showing all workspaces
- **Task chains** - Visualize design-doc → sprint-plan → implementation flow
- **Cost attribution** - Roll up costs to workspace/project level
- **Time-series aggregates** - Hourly/daily cost/token summaries

## References

- **Foundation**: [M-OTEL-DASHBOARD Implementation Report](../../implemented/v0_6_4/m-otel-dashboard.md)
- **Coordinator Guide**: [docs/docs/guides/coordinator.md](../../../docs/docs/guides/coordinator.md)
- **OTEL Spec**: [OpenTelemetry Resource Attributes](https://opentelemetry.io/docs/specs/semconv/resource/)
- **Axiom Reference**: [Design Axioms](/docs/references/axioms)

---

**Document created**: 2026-01-05
**Author**: Claude Code + Human collaboration
