# M-TASK-HIERARCHY Sprint Plan

**Sprint ID**: M-TASK-HIERARCHY
**Design Doc**: [m-task-hierarchy-linking.md](m-task-hierarchy-linking.md)
**Target Version**: v0.6.5
**Duration**: 4 days (32 hours)
**Risk Level**: Medium
**Status**: Ready for Execution

## Sprint Summary

**Goal**: Connect coordinator tasks with Observatory traces to build the full entity hierarchy (WORKSPACE → TASK → AGENT → SPANS → EVENTS), transforming Observatory from a flat trace viewer into a unified observability platform.

**Key Deliverables**:
1. Entity sync from Coordinator → Observatory
2. Context propagation via OTEL resource attributes
3. Task-linked spans with real-time aggregates
4. Hierarchy API and UI view
5. Backfill tool for existing data

## Velocity Analysis

**Recent velocity (last 7 days):**
- M-OTEL-DASHBOARD: 7,456 LOC in ~2 days (active sprint)
- Average sprint velocity: ~250-400 LOC/day sustained
- Peak velocity: 1000+ LOC/day during focused sprints

**This sprint estimate:**
- ~1,050 LOC implementation
- ~500 LOC tests
- 4 days at ~250 LOC/day + buffer

## Milestones

### M1: Entity Sync Layer (~150 LOC, Day 1 AM)

**Goal**: Sync coordinator entities to Observatory in real-time

**Tasks**:
- [ ] Create `ObservatorySync` interface in coordinator package
- [ ] Implement sync on task create/update/complete
- [ ] Implement sync on agent_assignment create/update
- [ ] Create workspace from git repo path on first task
- [ ] Add unit tests for sync functions

**Files**:
- `internal/coordinator/observatory_sync.go` (~150 LOC)
- `internal/coordinator/observatory_sync_test.go` (~100 LOC)

**Acceptance Criteria**:
- Tasks appear in Observatory DB when created in Coordinator
- Agent assignments sync on creation
- Workspaces auto-created from git repo paths
- Unit tests pass

**Dependencies**: None (first milestone)

---

### M2: Context Propagation (~100 LOC, Day 1 PM)

**Goal**: Pass task context to executors via OTEL resource attributes

**Tasks**:
- [ ] Modify executor spawn to set OTEL_RESOURCE_ATTRIBUTES
- [ ] Include task_id, agent_id, workspace, assignment_id in attributes
- [ ] Handle existing OTEL_RESOURCE_ATTRIBUTES (merge, don't replace)
- [ ] Add unit tests for attribute building

**Files**:
- `internal/coordinator/daemon_tasks.go` (modify, ~50 LOC)
- `internal/executor/claude/executor.go` (modify, ~30 LOC)
- `internal/executor/gemini/executor.go` (modify, ~20 LOC)

**Acceptance Criteria**:
- Executors receive OTEL_RESOURCE_ATTRIBUTES with task context
- Claude Code/Gemini inherit attributes in their spans
- Existing OTEL_RESOURCE_ATTRIBUTES preserved (merged)

**Dependencies**: M1 (needs sync to create entities first)

---

### M3: OTLP Receiver Enhancement (~100 LOC, Day 2 AM)

**Goal**: Extract task context from span attributes and link to entities

**Tasks**:
- [ ] Extract `ailang.task_id` from resource attributes
- [ ] Extract `ailang.assignment_id` from resource attributes
- [ ] Set span.task_id and span.agent_assignment_id before storage
- [ ] Add validation (log warning if task/assignment not found)
- [ ] Add unit tests for extraction

**Files**:
- `internal/observatory/otlp_receiver.go` (modify, ~100 LOC)
- `internal/observatory/otlp_receiver_test.go` (add tests, ~80 LOC)

**Acceptance Criteria**:
- Spans ingested with task_id have it stored in DB
- Spans queryable by task_id
- Warning logged if task_id references unknown task

**Dependencies**: M1, M2 (needs entities synced and context propagated)

---

### M4: Aggregation Updates (~150 LOC, Day 2 PM)

**Goal**: Update task/agent aggregates when spans complete

**Tasks**:
- [ ] Trigger aggregation update on span insert (in transaction)
- [ ] Update task totals (tokens, cost, duration, span_count, error_count)
- [ ] Update agent_assignment totals
- [ ] Handle concurrent updates with transactions
- [ ] Add integration tests

**Files**:
- `internal/observatory/store.go` (modify, ~100 LOC)
- `internal/observatory/aggregations.go` (new, ~50 LOC)
- `internal/observatory/aggregations_test.go` (new, ~100 LOC)

**Acceptance Criteria**:
- Task totals update in real-time as spans complete
- Agent assignment totals accurate
- Concurrent span ingestion doesn't corrupt aggregates
- Integration tests pass

**Dependencies**: M3 (needs spans linked to tasks)

---

### M5: Hierarchy API (~200 LOC, Day 3 AM)

**Goal**: API endpoint returning full task hierarchy

**Tasks**:
- [ ] Create `/api/observatory/tasks/:id/hierarchy` endpoint
- [ ] Return nested structure: Task → Agents → Traces → Spans
- [ ] Include aggregates at each level
- [ ] Add depth parameter to limit nesting
- [ ] Add integration tests

**Files**:
- `internal/observatory/api.go` (modify, ~100 LOC)
- `internal/observatory/hierarchy.go` (new, ~100 LOC)
- `internal/observatory/hierarchy_test.go` (new, ~100 LOC)

**Acceptance Criteria**:
- Full hierarchy returned for any task
- Nested spans preserve parent-child relationships
- Aggregates accurate at each level
- Depth parameter works

**Dependencies**: M4 (needs aggregates working)

---

### M6: UI Hierarchy View (~250 LOC, Day 3 PM - Day 4 AM)

**Goal**: React component for hierarchical task view

**Tasks**:
- [ ] Create `useTaskHierarchy(taskId)` hook
- [ ] Create `TaskHierarchyView` component with expandable tree
- [ ] Show aggregates and status at each level
- [ ] Link to existing trace detail view
- [ ] Add WebSocket updates for real-time changes
- [ ] Add CSS styling

**Files**:
- `ui/src/hooks/useTaskHierarchy.ts` (~80 LOC)
- `ui/src/features/observatory/TaskHierarchy.tsx` (~150 LOC)
- `ui/src/features/observatory/TaskHierarchy.module.css` (~50 LOC)

**Acceptance Criteria**:
- Hierarchy expands/collapses smoothly
- Aggregates visible without expanding
- Click on span opens detail panel
- Real-time updates via WebSocket

**Dependencies**: M5 (needs hierarchy API)

---

### M7: Backfill Tool (~100 LOC, Day 4 PM)

**Goal**: Link existing spans to tasks using time correlation

**Tasks**:
- [ ] Create `ailang observatory backfill` command
- [ ] Match spans to tasks by time window + agent
- [ ] Add preview mode (--dry-run)
- [ ] Report linking statistics
- [ ] Add integration test

**Files**:
- `cmd/ailang/observatory_backfill.go` (~100 LOC)
- `cmd/ailang/observatory_backfill_test.go` (~50 LOC)

**Acceptance Criteria**:
- Historical spans linked to tasks
- Dry-run shows what would be linked
- Statistics reported (X spans linked to Y tasks)

**Dependencies**: M3, M4 (needs linking infrastructure)

---

## Day-by-Day Schedule

| Day | Morning (4h) | Afternoon (4h) |
|-----|--------------|----------------|
| 1 | M1: Entity Sync Layer | M2: Context Propagation |
| 2 | M3: OTLP Receiver Enhancement | M4: Aggregation Updates |
| 3 | M5: Hierarchy API | M6: UI Hierarchy View (start) |
| 4 | M6: UI Hierarchy View (complete) | M7: Backfill Tool |

## Success Metrics

- [ ] New coordinator tasks appear in Observatory
- [ ] Executor spans have task_id populated
- [ ] Task aggregates update in real-time
- [ ] `/api/observatory/tasks/:id/hierarchy` returns nested structure
- [ ] UI shows expandable task → agent → span hierarchy
- [ ] Historical spans can be backfilled
- [ ] All tests passing (`make test`)
- [ ] Linting passes (`make lint`)

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Claude Code doesn't propagate OTEL attrs | Test with explicit env vars first, verify in span details |
| Cross-database transactions | Accept eventual consistency, log sync failures |
| Performance of hierarchy query | Add caching, limit default depth to 2 |
| Backfill accuracy | Use conservative time windows, require --force for uncertain matches |

## Test Strategy

**TDD Approach**: Each milestone starts with tests

1. **Unit tests** (each milestone):
   - Write tests first for new functions
   - Mock external dependencies (coordinator store, observatory store)

2. **Integration tests** (M4, M5, M7):
   - Test full flow with real databases
   - Verify cross-database consistency

3. **End-to-end** (after M6):
   - Create real coordinator task
   - Execute with Claude Code
   - Verify spans linked in UI

## Pause Points

**After M2**: Verify context propagation works before building on it
**After M4**: Verify aggregates are accurate before building UI
**After M6**: Get user feedback on UI before polishing

---

**Created**: 2026-01-05
**Sprint Executor**: sprint-executor skill
