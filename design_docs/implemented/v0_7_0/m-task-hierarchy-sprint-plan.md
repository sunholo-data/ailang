# M-TASK-HIERARCHY Sprint Plan

**Sprint ID**: M-TASK-HIERARCHY
**Design Doc**: [m-task-hierarchy-linking.md](m-task-hierarchy-linking.md)
**Target Version**: v0.6.5
**Duration**: 4 days (32 hours)
**Risk Level**: Medium
**Status**: ✅ COMPLETE (Verified 2026-01-08)

## Sprint Summary

**Goal**: Connect coordinator tasks with Observatory traces to build the full entity hierarchy (WORKSPACE → TASK → AGENT → SPANS → EVENTS), transforming Observatory from a flat trace viewer into a unified observability platform.

**Key Deliverables**:
1. Entity sync from Coordinator → Observatory
2. Context propagation via OTEL resource attributes
3. Task-linked spans with real-time aggregates
4. Hierarchy API and UI view
5. Backfill tool for existing data

## Implementation Status (Verified 2026-01-08)

| Milestone | Status | Implementation | LOC |
|-----------|--------|----------------|-----|
| M1: Entity Sync | ✅ DONE | `internal/coordinator/observatory_sync.go` | 303 |
| M2: Context Propagation | ✅ DONE | `internal/executor/environment.go` | 193 |
| M3: OTLP Extraction | ✅ DONE | `internal/observatory/otlp_receiver.go` | 759 (includes 6-level fallback) |
| M4: Aggregations | ✅ DONE | `internal/observatory/store.go` + aggregations | ~150 |
| M5: Hierarchy API | ✅ DONE | `internal/observatory/hierarchy.go` | 413 |
| M6: UI Component | ✅ DONE | `ui/src/features/observatory/TaskHierarchy.tsx` | 315 |
| M7: Backfill Tool | ✅ DONE | `cmd/ailang/observatory.go` | 438 |

**Total Implementation**: ~2,571 LOC (exceeded 1,050 estimate by 145%)

### Verification Commands (All Passing)

```bash
# Entity sync
curl -s http://localhost:1957/api/observatory/tasks | jq 'length'
# Returns: 1

# Hierarchy API
curl -s "http://localhost:1957/api/observatory/tasks/task-617522c0/hierarchy" | jq '.task.id'
# Returns: "task-617522c0"

# Control Plane stats (aggregates working)
curl -s http://localhost:1957/api/controlplane/stats | jq '.observatory'
# Returns: {"total_spans": 1468, "total_cost_usd": 2.38, ...}

# Backfill tool
ailang observatory backfill --dry-run
# Works ✅

# Tests
go test ./internal/observatory/... -run Hierarchy
# 36+ tests passing ✅
```

## Implementation Highlights

### M2: Context Propagation (Exceeded Design)

The implementation in `internal/executor/environment.go` goes beyond the design:

- Sets `AILANG_PARENT_TASK_ID` for hierarchy tracking
- Calls `BuildResourceAttributes()` to create OTEL attrs
- Merges existing `OTEL_RESOURCE_ATTRIBUTES` (priority: Env → Task Metadata → Defaults)
- Sets `ailang.task_id`, `ailang.session_id`, `ailang.source`

### M3: OTLP Extraction (6-Level Fallback Chain)

The receiver implements a sophisticated extraction priority:

1. `ailang.task_id` from resource attributes
2. `task.id` span attribute
3. `exec.parent_task_id` span attribute
4. `task.workspace` span attribute
5. `process.cwd` worktree path extraction
6. Session-based correlation via claude.execute lookup

### M5: Bonus Feature - Timestamp Correlation

The hierarchy API includes virtual re-parenting of Claude Code subprocess spans using timestamp correlation. This handles the known limitation that Claude Code doesn't propagate TRACEPARENT to subprocess environments.

## Lessons Learned

### What Went Well
- Implementation exceeded design specifications
- 6-level fallback chain provides robust task linking
- Timestamp correlation solved cross-process tracing without runtime changes

### What Could Be Improved
- **Documentation debt**: Sprint plan showed "NOT DONE" for features that were implemented
- **Verification gap**: No automated check that docs match implementation
- **Server restart requirement**: After `make install`, server must be restarted (easy to forget)

## Success Metrics (All Met)

- [x] New coordinator tasks appear in Observatory
- [x] Executor spans have task_id populated
- [x] Task aggregates update in real-time
- [x] `/api/observatory/tasks/:id/hierarchy` returns nested structure
- [x] UI shows expandable task → agent → span hierarchy
- [x] Historical spans can be backfilled
- [x] All tests passing (`make test`)
- [x] Linting passes (`make lint`)

---

**Created**: 2026-01-05
**Completed**: 2026-01-08
**Verified by**: Code audit + API testing
