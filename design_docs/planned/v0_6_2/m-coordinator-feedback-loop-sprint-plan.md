# M-COORD-FEEDBACK Sprint Plan

**Sprint ID**: M-COORD-FEEDBACK
**Duration**: 4 days (~30 hours)
**Target Version**: v0.6.2
**Design Doc**: [m-coordinator-feedback-loop.md](m-coordinator-feedback-loop.md)
**Risk Level**: Medium (significant new features, but building on existing infrastructure)

## Sprint Summary

**Goal**: Implement real-time feedback loop for coordinator executor with streaming logs, resource monitoring, and human approval checkpoints.

**Key Deliverables**:
1. Streaming event handler broadcasting executor events to dashboard/CLI
2. Resource monitoring (CPU, RAM, tokens, cost) during execution
3. Approval checkpoint system for merge decisions
4. Dashboard and CLI integration

## Current Status Analysis

**Existing Infrastructure** (can leverage):
- ✅ `executor.EventHandler` interface with OnTurnStart, OnText, OnToolUse, etc.
- ✅ WebSocket server in `internal/server/` with real-time broadcasting
- ✅ `ProcessStats` structure for CPU/RAM monitoring
- ✅ Approval system in `handlers_approvals.go` with approve/reject endpoints
- ✅ Coordinator daemon running tasks in isolated worktrees

**Gaps to Fill**:
- ❌ Coordinator uses `NoOpEventHandler` - no streaming
- ❌ No WebSocket channel for task events
- ❌ No pre-destroy hook in worktree manager
- ❌ No CLI watch mode or approve command

## Velocity Analysis

Recent development shows ~150-200 LOC/day for coordinator-related work.
Sprint estimate: ~1,290 LOC across 4 days = ~320 LOC/day (aggressive but achievable with existing infrastructure).

## Milestones

### M1: Streaming Event Handler (~8 hours, ~280 LOC)

**Goal**: Capture and broadcast executor events in real-time.

**Tasks**:
1. Create `internal/coordinator/event_handler.go` (~200 LOC)
   - Implement `executor.EventHandler` interface
   - Add WebSocket broadcast method
   - Implement rate limiting (10 events/sec)
   - Buffer events for replay
2. Wire handler in `daemon.go` (~30 LOC)
   - Replace `NoOpEventHandler` with new handler
   - Use `ExecuteStreaming` instead of `Execute`
3. Add WebSocket task events channel in `server.go` (~30 LOC)
4. Add `--watch` flag to CLI (`coordinator.go`) (~20 LOC)

**Acceptance Criteria**:
- [ ] Events appear in dashboard within 500ms of executor output
- [ ] Rate limiting prevents event flood
- [ ] CLI `--watch` shows live task progress
- [ ] Unit tests for EventHandler

**Files**:
- NEW: `internal/coordinator/event_handler.go` (~200 LOC)
- MOD: `internal/coordinator/daemon.go` (~30 LOC)
- MOD: `internal/server/server.go` (~30 LOC)
- MOD: `cmd/ailang/coordinator.go` (~20 LOC)

---

### M2: Resource Monitoring (~6 hours, ~250 LOC)

**Goal**: Track CPU, RAM, tokens, and cost during task execution.

**Tasks**:
1. Create `internal/coordinator/resource_tracker.go` (~150 LOC)
   - Poll process metrics every 5 seconds
   - Track accumulated tokens from events
   - Calculate running cost
   - Record peak usage
2. Extend `TaskRecord` with metrics fields (~20 LOC)
   - Add `peak_cpu`, `peak_memory`, `current_tokens`, `current_cost`
3. Update `/api/monitor` to include task metrics (~40 LOC)
4. Add metrics to CLI status output (~40 LOC)

**Acceptance Criteria**:
- [ ] CPU/RAM metrics update every 5 seconds
- [ ] Token count accumulates from streaming events
- [ ] Cost calculated in real-time
- [ ] Metrics visible in CLI and dashboard
- [ ] Unit tests for ResourceTracker

**Files**:
- NEW: `internal/coordinator/resource_tracker.go` (~150 LOC)
- MOD: `internal/coordinator/store.go` (~20 LOC)
- MOD: `internal/server/handlers_monitor.go` (~40 LOC)
- MOD: `cmd/ailang/coordinator.go` (~40 LOC)

---

### M3: Approval Checkpoints (~10 hours, ~410 LOC)

**Goal**: Pause execution for human decisions before destructive operations.

**Tasks**:
1. Create `internal/coordinator/approval_checkpoint.go` (~250 LOC)
   - Define `ApprovalType` enum
   - Implement `RequestApproval()` with blocking wait
   - Add timeout handling (default 1 hour)
   - Support auto-reject on timeout
2. Add pre-destroy hook in `worktree.go` (~40 LOC)
   - Check for changes before destruction
   - Create approval request if changes exist
   - Block until approval/rejection
3. Add CLI `approve` command (~60 LOC)
   - `ailang coordinator approve <task-id>`
   - `ailang coordinator reject <task-id>`
4. Add approval endpoints in `handlers_approvals.go` (~60 LOC)
   - GET `/api/coordinator/tasks/{id}/approval`
   - POST `/api/coordinator/tasks/{id}/approve`
   - POST `/api/coordinator/tasks/{id}/reject`

**Acceptance Criteria**:
- [ ] Task completion blocks on approval when worktree has changes
- [ ] CLI approve/reject commands work
- [ ] API endpoints functional
- [ ] Auto-reject after timeout
- [ ] Approval history logged
- [ ] Unit tests for ApprovalCheckpoint

**Files**:
- NEW: `internal/coordinator/approval_checkpoint.go` (~250 LOC)
- MOD: `internal/coordinator/worktree.go` (~40 LOC)
- MOD: `cmd/ailang/coordinator.go` (~60 LOC)
- MOD: `internal/server/handlers_approvals.go` (~60 LOC)

---

### M4: Dashboard Integration (~6 hours, ~350 LOC)

**Goal**: Create interactive dashboard UI for task monitoring and approvals.

**Tasks**:
1. Create `ui/src/components/TaskExecution/` (~250 LOC)
   - `TaskExecutionPanel.tsx` - Main component
   - `StreamingLog.tsx` - Live log display
   - `ResourceMetrics.tsx` - CPU/RAM/tokens charts
2. Add approval UI in dashboard (~100 LOC)
   - Pending approval banner
   - Diff preview modal
   - One-click approve/reject buttons

**Acceptance Criteria**:
- [ ] Live log streaming in dashboard
- [ ] Resource graphs update in real-time
- [ ] Approval modal shows diff preview
- [ ] One-click approve/reject works
- [ ] Sound/visual notification for pending approvals

**Files**:
- NEW: `ui/src/components/TaskExecution/TaskExecutionPanel.tsx` (~100 LOC)
- NEW: `ui/src/components/TaskExecution/StreamingLog.tsx` (~80 LOC)
- NEW: `ui/src/components/TaskExecution/ResourceMetrics.tsx` (~70 LOC)
- MOD: `ui/src/App.tsx` (~50 LOC)
- MOD: `ui/src/components/MessageCenter/` (~50 LOC)

---

## Day-by-Day Plan

### Day 1 (8 hours)
**Focus**: M1 - Streaming Event Handler

| Time | Task | LOC |
|------|------|-----|
| 2h | Create `event_handler.go` with EventHandler impl | 120 |
| 1h | Add rate limiting and buffering | 80 |
| 1h | Wire into daemon.go, use ExecuteStreaming | 30 |
| 1h | Add WebSocket task events channel | 30 |
| 1h | Add CLI `--watch` flag | 20 |
| 2h | Unit tests + integration test | 100 |

**Checkpoint**: Events streaming to console, basic test passes

### Day 2 (6 hours)
**Focus**: M2 - Resource Monitoring

| Time | Task | LOC |
|------|------|-----|
| 2h | Create `resource_tracker.go` | 100 |
| 1h | Add process metrics polling | 50 |
| 1h | Extend TaskRecord, update store | 40 |
| 1h | Update CLI status with metrics | 40 |
| 1h | Unit tests | 50 |

**Checkpoint**: `ailang coordinator status` shows CPU/RAM/tokens

### Day 3 (8 hours)
**Focus**: M3 - Approval Checkpoints

| Time | Task | LOC |
|------|------|-----|
| 3h | Create `approval_checkpoint.go` | 200 |
| 1h | Add blocking wait and timeout | 50 |
| 1h | Add pre-destroy hook in worktree.go | 40 |
| 1h | Add CLI approve/reject commands | 60 |
| 1h | Add API endpoints | 60 |
| 1h | Integration test: full approval flow | 80 |

**Checkpoint**: Can send task, see approval request, approve via CLI

### Day 4 (8 hours)
**Focus**: M4 - Dashboard + Polish

| Time | Task | LOC |
|------|------|-----|
| 3h | Create TaskExecution React components | 200 |
| 2h | Add approval UI with diff preview | 100 |
| 1h | Build and deploy UI to server | 50 |
| 1h | End-to-end testing | — |
| 1h | Documentation + cleanup | — |

**Checkpoint**: Full workflow works via dashboard

---

## Success Metrics

- [ ] All 4 milestones complete
- [ ] 10+ unit tests added
- [ ] Integration test: Send task → Stream logs → Approve → Merge
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Dashboard shows live task execution

## Total Estimates

| Category | LOC |
|----------|-----|
| New Go code | ~600 |
| Modified Go code | ~290 |
| New UI code | ~350 |
| Tests | ~200 |
| **Total** | ~1,440 |

## Open Questions

1. **Merge strategy**: Cherry-pick individual files or merge entire worktree?
2. **Approval timeout**: 1 hour default - configurable via CLI flag?
3. **Sound notifications**: Use browser notifications API or custom sounds?

## Dependencies

- Coordinator daemon must be running
- WebSocket server must be accessible
- Dashboard must be built and served

---

**Created**: 2025-12-29
**Sprint Start**: Upon approval
