# M-TASK-HIERARCHY Sprint Plan

**Sprint ID**: M-TASK-HIERARCHY
**Design Doc**: [m-task-hierarchy-linking.md](m-task-hierarchy-linking.md)
**Target Version**: v0.6.5
**Duration**: 4 days (32 hours)
**Risk Level**: Medium
**Status**: IN PROGRESS (Partial Implementation)

## Sprint Summary

**Goal**: Connect coordinator tasks with Observatory traces to build the full entity hierarchy (WORKSPACE → TASK → AGENT → SPANS → EVENTS), transforming Observatory from a flat trace viewer into a unified observability platform.

**Key Deliverables**:
1. Entity sync from Coordinator → Observatory
2. Context propagation via OTEL resource attributes
3. Task-linked spans with real-time aggregates
4. Hierarchy API and UI view
5. Backfill tool for existing data

## Implementation Status (2026-01-05)

**CRITICAL: This sprint was previously marked complete but verification revealed partial implementation.**

| Milestone | Status | Verified | Notes |
|-----------|--------|----------|-------|
| M1: Entity Sync | ✅ DONE | ✅ 2026-01-05 | `observatory_sync.go` works, tested with real task |
| M2: Context Propagation | ❌ NOT DONE | - | OTEL_RESOURCE_ATTRIBUTES not passed to executors |
| M3: OTLP Extraction | ❌ NOT DONE | - | Receiver doesn't extract ailang.task_id |
| M4: Aggregations | ❌ NOT DONE | - | Task totals not updated from spans |
| M5: Hierarchy API | ✅ DONE | ✅ 2026-01-05 | Endpoint works, returns nested structure |
| M6: UI Component | ⚠️ PARTIAL | - | Component exists but no navigation to it |
| M7: Backfill Tool | ❌ NOT DONE | - | CLI command not implemented |

**Verified Working (2026-01-05):**
```bash
# Entity sync verified
curl -s http://localhost:1957/api/observatory/tasks | jq '.[0].id'
# Returns: "task-617522c0" ✅

# Hierarchy API verified
curl -s http://localhost:1957/api/observatory/tasks/task-617522c0/hierarchy | jq '.agents[0].agent.id'
# Returns: "aa_06211e2372b886c2" ✅
```

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

### M1: Entity Sync Layer (~150 LOC, Day 1 AM) ✅ COMPLETE

**Goal**: Sync coordinator entities to Observatory in real-time

**Tasks**:
- [x] Create `ObservatorySync` interface in coordinator package
- [x] Implement sync on task create/update/complete
- [x] Implement sync on agent_assignment create/update
- [x] Create workspace from git repo path on first task
- [x] Add unit tests for sync functions

**Files**:
- `internal/coordinator/observatory_sync.go` (~300 LOC) ✅
- `internal/coordinator/observatory_sync_test.go` (~300 LOC) ✅

**Verification (REQUIRED before marking complete)**:
```bash
# 1. Create test task via coordinator
ailang messages send coordinator "Test sync" --title "Sync Test" --from test

# 2. Wait for execution, then verify in Observatory DB
sqlite3 ~/.ailang/state/observatory.db "SELECT id, title FROM tasks"
# MUST show non-empty ID matching coordinator task ID

# 3. Verify via API
curl -s http://localhost:1957/api/observatory/tasks | jq '.[0].id'
# MUST return non-empty task ID
```

**Dependencies**: None (first milestone)

---

### M2: Context Propagation (~100 LOC, Day 1 PM) ❌ NOT STARTED

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

**Verification (REQUIRED before marking complete)**:
```bash
# 1. Start coordinator task
ailang messages send coordinator "Test context" --title "Context Test"

# 2. Check Claude Code environment (in worktree)
# The executor MUST pass OTEL_RESOURCE_ATTRIBUTES to subprocess

# 3. After execution, verify attributes appear in Observatory span
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT resource_attributes FROM spans ORDER BY created_at DESC LIMIT 1"
# MUST contain: ailang.task_id, ailang.assignment_id, ailang.workspace
```

**Dependencies**: M1 (needs sync to create entities first)

---

### M3: OTLP Receiver Enhancement (~100 LOC, Day 2 AM) ❌ NOT STARTED

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

**Verification (REQUIRED before marking complete)**:
```bash
# 1. After M2 context propagation, spans should have task_id set
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT id, task_id FROM spans WHERE task_id IS NOT NULL LIMIT 5"
# MUST show spans with non-NULL task_id

# 2. Query spans by task
curl -s "http://localhost:1957/api/observatory/spans?task_id=task-xxx" | jq 'length'
# MUST return > 0 if task executed
```

**Dependencies**: M1, M2 (needs entities synced and context propagated)

---

### M4: Aggregation Updates (~150 LOC, Day 2 PM) ❌ NOT STARTED

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

**Verification (REQUIRED before marking complete)**:
```bash
# 1. After task execution with linked spans
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT id, span_count, total_tokens_in, total_cost_usd FROM tasks WHERE id='task-xxx'"
# span_count MUST be > 0
# total_tokens_in MUST be > 0
# total_cost_usd MUST be > 0

# 2. Via API
curl -s http://localhost:1957/api/observatory/tasks/task-xxx | jq '.span_count'
# MUST be > 0
```

**Dependencies**: M3 (needs spans linked to tasks)

---

### M5: Hierarchy API (~200 LOC, Day 3 AM) ✅ COMPLETE

**Goal**: API endpoint returning full task hierarchy

**Tasks**:
- [x] Create `/api/observatory/tasks/:id/hierarchy` endpoint
- [x] Return nested structure: Task → Agents → Traces → Spans
- [x] Include aggregates at each level
- [x] Add depth parameter to limit nesting
- [x] Add integration tests

**Files**:
- `internal/observatory/api.go` (modify, ~100 LOC) ✅
- `internal/observatory/hierarchy.go` (new, ~200 LOC) ✅
- `internal/observatory/hierarchy_test.go` (new, ~100 LOC) ✅

**Verification (REQUIRED before marking complete)**:
```bash
# 1. Query hierarchy endpoint
curl -s http://localhost:1957/api/observatory/tasks/task-xxx/hierarchy | jq '.task.id'
# MUST return the task ID

# 2. Verify agents are included
curl -s http://localhost:1957/api/observatory/tasks/task-xxx/hierarchy | jq '.agents | length'
# MUST be >= 1 if task has agent assignment

# 3. Run integration tests
go test -v ./internal/observatory/... -run Hierarchy
# ALL MUST PASS
```

**Dependencies**: M4 (needs aggregates working)

---

### M6: UI Hierarchy View (~250 LOC, Day 3 PM - Day 4 AM) ⚠️ PARTIAL

**Goal**: React component for hierarchical task view

**Tasks**:
- [x] Create `useTaskHierarchy(taskId)` hook
- [x] Create `TaskHierarchyView` component with expandable tree
- [x] Show aggregates and status at each level
- [ ] **Add Tasks tab to Observatory navigation** ❌ MISSING
- [ ] Link to existing trace detail view
- [ ] Add WebSocket updates for real-time changes
- [ ] Add CSS styling

**Files**:
- `ui/src/hooks/useObservatory.ts` (added useTaskHierarchy) ✅
- `ui/src/features/observatory/TaskHierarchy.tsx` (~310 LOC) ✅
- `ui/src/features/observatory/TaskHierarchy.module.css` ✅
- `ui/src/features/observatory/Observatory.tsx` ❌ NEEDS TASKS TAB

**Verification (REQUIRED before marking complete)**:
```bash
# 1. Build UI
cd ui && npm run build

# 2. Start server
ailang serve

# 3. Open http://localhost:1957 in browser
# MUST see "Tasks" tab in Observatory navigation
# Clicking Tasks tab MUST show task list
# Clicking task MUST show TaskHierarchyView

# 4. Verify real-time updates
# Create new task while UI open
# Task MUST appear without page refresh
```

**Dependencies**: M5 (needs hierarchy API)

---

### M7: Backfill Tool (~100 LOC, Day 4 PM) ❌ NOT STARTED

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

**Verification (REQUIRED before marking complete)**:
```bash
# 1. Dry run
ailang observatory backfill --dry-run
# MUST show "Would link X spans to Y tasks"

# 2. Actual backfill
ailang observatory backfill
# MUST show "Linked X spans to Y tasks"

# 3. Verify spans now have task_id
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT COUNT(*) FROM spans WHERE task_id IS NOT NULL"
# Count MUST have increased
```

**Dependencies**: M3, M4 (needs linking infrastructure)

---

## Day-by-Day Schedule

| Day | Morning (4h) | Afternoon (4h) |
|-----|--------------|----------------|
| 1 | M1: Entity Sync Layer ✅ | M2: Context Propagation |
| 2 | M3: OTLP Receiver Enhancement | M4: Aggregation Updates |
| 3 | M5: Hierarchy API ✅ | M6: UI Hierarchy View (start) ⚠️ |
| 4 | M6: UI Hierarchy View (complete) | M7: Backfill Tool |

## Success Metrics

- [x] New coordinator tasks appear in Observatory ✅
- [ ] Executor spans have task_id populated ❌
- [ ] Task aggregates update in real-time ❌
- [x] `/api/observatory/tasks/:id/hierarchy` returns nested structure ✅
- [ ] UI shows expandable task → agent → span hierarchy (navigation missing) ⚠️
- [ ] Historical spans can be backfilled ❌
- [x] All tests passing (`make test`) ✅
- [x] Linting passes (`make lint`) ✅

## Mandatory Checkpoint Validation

**CRITICAL: Milestones MUST NOT be marked complete without running verification commands.**

### Checkpoint Script Requirements

The `milestone_checkpoint.sh` script MUST:

1. **Run verification commands** - Not just `make test && make lint`
2. **Check actual data** - Query databases, call APIs, verify non-empty results
3. **FAIL LOUDLY** - Exit with error if verification fails
4. **Block progression** - Do not allow marking complete if verification fails

### Enhanced Checkpoint Process

```bash
# After each milestone, run:
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M1

# The script MUST:
# 1. Run make test
# 2. Run make lint
# 3. Execute milestone-specific verification commands
# 4. Check that results are non-empty / expected values
# 5. Print PASS/FAIL for each check
# 6. Exit 0 only if ALL checks pass
```

### Example Verification for M1

```bash
#!/bin/bash
# M1 Verification - Entity Sync

echo "=== M1: Entity Sync Verification ==="

# Check 1: observatory_sync.go exists
if [ ! -f "internal/coordinator/observatory_sync.go" ]; then
  echo "FAIL: observatory_sync.go not found"
  exit 1
fi
echo "PASS: observatory_sync.go exists"

# Check 2: Tests pass
if ! go test -v ./internal/coordinator/... -run ObservatorySync; then
  echo "FAIL: ObservatorySync tests failed"
  exit 1
fi
echo "PASS: Tests pass"

# Check 3: Can create a task and see it in Observatory
# (This requires running coordinator - manual verification)
echo "MANUAL CHECK: Run coordinator, create task, verify in Observatory DB"
echo "  sqlite3 ~/.ailang/state/observatory.db 'SELECT id FROM tasks LIMIT 1'"
echo "  Expected: Non-empty task ID"

echo "=== M1 Verification Complete ==="
```

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Claude Code doesn't propagate OTEL attrs | Test with explicit env vars first, verify in span details |
| Cross-database transactions | Accept eventual consistency, log sync failures |
| Performance of hierarchy query | Add caching, limit default depth to 2 |
| Backfill accuracy | Use conservative time windows, require --force for uncertain matches |
| **Milestones marked complete without verification** | **ADDED: Mandatory verification commands in checkpoint script** |

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

4. **Verification commands** (MANDATORY):
   - Each milestone has specific verification commands
   - Commands MUST be run and MUST pass before marking complete
   - Results MUST show expected data (non-empty, correct values)

## Pause Points

**After M2**: Verify context propagation works before building on it
**After M4**: Verify aggregates are accurate before building UI
**After M6**: Get user feedback on UI before polishing

---

**Created**: 2026-01-05
**Last Updated**: 2026-01-05 (Status verification)
**Sprint Executor**: sprint-executor skill
