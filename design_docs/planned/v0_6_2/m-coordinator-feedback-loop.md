# M-COORD-FEEDBACK: Coordinator Executor Feedback Loop

**Status**: Planned
**Target: v0.6.2
**Priority**: P0 - High
**Estimated**: 3-4 days (24-32 hours)
**Dependencies**: Coordinator daemon (v0.6.1), WebSocket server, Approval system

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics |
| A2: Replayability | +1 | Streaming logs create audit trail for executor sessions |
| A3: Effect Legibility | +1 | Makes executor effects visible in real-time |
| A4: Explicit Authority | +1 | Human approval required before merging to main branch |
| A5: Bounded Verification | 0 | No change to verification |
| A6: Safe Concurrency | 0 | Coordinator already handles isolation via worktrees |
| A7: Machines First | +1 | Enables programmatic monitoring of agent behavior |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Real-time cost/token tracking visible in dashboard |
| A10: Composability | 0 | Integrates with existing systems |
| A11: Structured Failure | +1 | Executor failures captured with structured error data |
| A12: System Boundary | +1 | Explicit boundary between agent execution and human approval |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects (all made visible)
- [x] A4 (Authority): Human approval required for destructive actions
- [x] A7 (Machines First): Designed for machine monitoring and automation

## Problem Statement

The coordinator daemon executes tasks autonomously but provides no visibility into execution progress, no resource monitoring, and no human checkpoint for destructive operations.

**Current State:**
- Coordinator runs tasks in isolated worktrees but logs only show "Starting execution" and "Completed"
- No streaming output from Claude Code CLI visible to dashboard or CLI
- No CPU/RAM/token/cost metrics during execution
- Worktrees are automatically destroyed after execution - no option to review and merge changes
- No mechanism for executor to pause and request human decisions

**Impact:**
- Users have no visibility into long-running tasks (5+ minutes of black box)
- Cannot cancel runaway tasks or detect resource issues
- Changes made in worktree are lost unless executor commits/pushes (which it shouldn't auto-do)
- No human-in-the-loop for critical decisions like "merge to main?"

## Goals

**Primary Goal:** Create a real-time feedback loop between coordinator executors and humans via dashboard/CLI.

**Success Metrics:**
1. Streaming logs visible in dashboard within 500ms of executor output
2. CPU/RAM/tokens/cost metrics updated every 5 seconds during execution
3. Human can approve/reject worktree merge before destruction
4. Executor can pause for arbitrary decisions (file confirmations, merge conflicts, etc.)
5. All metrics visible in both dashboard UI and `ailang coordinator status --watch`

## Solution Design

### Overview

Three interconnected systems working together:

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  Executor Process   │────▶│   Event Broker      │────▶│  Dashboard/CLI      │
│  (Claude Code CLI)  │     │   (WebSocket Hub)   │     │  (Real-time View)   │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
         │                           │                           │
         │                           │                           │
         ▼                           ▼                           ▼
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  Resource Monitor   │     │   Approval Queue    │◀───▶│  Human Decision     │
│  (CPU/RAM/tokens)   │     │   (SQLite)          │     │  (Approve/Reject)   │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
```

### Architecture

**Component 1: StreamingEventHandler**
Implements `executor.EventHandler` to capture and broadcast executor events:
- OnTurnStart → Broadcast turn number
- OnText → Broadcast text chunks (with rate limiting for high-frequency updates)
- OnToolUse → Broadcast tool name and input
- OnToolResult → Broadcast tool output (truncated for large outputs)
- OnError → Broadcast error with context

**Component 2: ResourceMonitor**
Tracks executor process metrics:
- CPU percentage (via `ps` or `/proc`)
- Memory MB (via `ps` or `/proc`)
- Accumulated tokens (from executor events)
- Running cost (calculated from tokens)
- Updates stored in `ProcessStats` and broadcast via WebSocket

**Component 3: ApprovalCheckpoint**
Pauses execution at decision points:
- Creates approval request in SQLite
- Posts message to thread: "Waiting for approval: Merge worktree to main?"
- Blocks execution until approval/rejection received
- Returns decision to executor to continue or abort

### Implementation Plan

**Phase 1: Streaming Event Handler** (~8 hours)

- [ ] Create `CoordinatorEventHandler` implementing `executor.EventHandler`
- [ ] Add `OnEvent(type, payload)` to broadcast structured events via WebSocket
- [ ] Implement rate limiting (max 10 events/second per task)
- [ ] Store events in task log table for replay
- [ ] Add `--streaming` flag to `ailang coordinator status`

**Phase 2: Resource Monitoring** (~6 hours)

- [ ] Extend `ProcessStats` with live token/cost tracking
- [ ] Create `ResourceTracker` goroutine that polls metrics every 5s
- [ ] Integrate with existing `/api/monitor` endpoint
- [ ] Add task-specific metrics to `TaskRecord` (peak_cpu, peak_memory)
- [ ] Create dashboard widget for resource graphs

**Phase 3: Approval Checkpoints** (~10 hours)

- [ ] Define `ApprovalType` enum: `MergeWorktree`, `ConfirmDelete`, `ResolveConflict`, `Custom`
- [ ] Create `RequestApproval(taskID, approvalType, message) (decision, error)` method
- [ ] Implement blocking wait with timeout (default: 1 hour)
- [ ] Add pre-destroy hook in `WorktreeManager.RemoveWorktree()`
- [ ] Create approval UI in dashboard with diff preview
- [ ] Add CLI command: `ailang coordinator approve <task-id>`
- [ ] Implement auto-reject on timeout with configurable behavior

**Phase 4: Dashboard Integration** (~8 hours)

- [ ] Add "Task Execution" panel with live log streaming
- [ ] Create resource usage graphs (CPU, RAM, tokens, cost)
- [ ] Add approval queue with one-click approve/reject
- [ ] Show worktree diff preview before merge approval
- [ ] Add sound/notification for pending approvals

### Files to Modify/Create

**New files:**
- `internal/coordinator/event_handler.go` - StreamingEventHandler (~200 LOC)
- `internal/coordinator/resource_tracker.go` - ResourceTracker (~150 LOC)
- `internal/coordinator/approval_checkpoint.go` - ApprovalCheckpoint (~250 LOC)
- `ui/src/components/TaskExecution/` - Dashboard components (~400 LOC)

**Modified files:**
- `internal/coordinator/daemon.go` - Wire up event handler and checkpoints (~50 LOC)
- `internal/coordinator/task_executor.go` - Use streaming execution (~30 LOC)
- `internal/coordinator/worktree.go` - Add pre-destroy hook (~40 LOC)
- `internal/server/server.go` - Add WebSocket task events channel (~30 LOC)
- `internal/server/handlers_approvals.go` - Add task approval endpoints (~60 LOC)
- `cmd/ailang/coordinator.go` - Add `--watch`, `approve` commands (~80 LOC)

## Examples

### Example 1: Streaming Logs in Dashboard

**Before (current):**
```
Task Statistics
  Completed:  1
  Running:    1     <-- No visibility into what's happening
  Total Cost: $0.08
```

**After:**
```
Task: Review v0.6.2 design docs (task-c62486b3)
Status: Running (2m 34s)

[Live Log]
[TURN 1]
Reading design_docs/planned/v0_6_2/eval-dashboard-reliability.md...
[TOOL] Read design_docs/planned/v0_6_2/eval-dashboard-reliability.md
[TOOL] Grep "atomic" internal/eval_analysis/
Found 3 matches in report.go

[TURN 2]
Checking implementation status...
[TOOL] Read internal/eval_analysis/report.go

Resources: CPU 45% | RAM 280MB | Tokens 4,521 | Cost $0.23
```

### Example 2: Merge Approval

**Workflow:**
```
1. Task completes in worktree
2. Coordinator detects changes: 3 files modified, 1 created
3. Creates approval request:

   ┌─────────────────────────────────────────────────────┐
   │  APPROVAL REQUIRED                                   │
   │                                                      │
   │  Task: Review v0.6.2 design docs                    │
   │  Request: Merge worktree changes to dev branch?     │
   │                                                      │
   │  Changes:                                           │
   │  + design_docs/implemented/v0_6_2/eval-dashboard... │
   │  + design_docs/implemented/v0_6_2/m-ai-ollama...    │
   │  M DESIGN_DOCS_IMPLEMENTATION_REPORT.md             │
   │                                                      │
   │  [View Diff]  [Approve]  [Reject]                   │
   └─────────────────────────────────────────────────────┘

4. Human clicks "Approve" or runs: ailang coordinator approve task-c62486b3
5. Coordinator merges changes to dev branch
6. Worktree is destroyed
```

### Example 3: CLI Watch Mode

```bash
$ ailang coordinator status --watch

Coordinator Status (live, press Ctrl+C to exit)

  State:      ▶ running
  PID:        63270

Task Statistics
  Completed:  2
  Running:    1
  Pending:    0
  Total Cost: $0.97

Active Tasks:
  task-c62486b3  Review v0.6.2 design docs
    Status:   Running (3m 12s)
    Turn:     5 / ~8 estimated
    Tokens:   6,234 in / 1,456 out
    Cost:     $0.31
    CPU:      52%
    Memory:   312 MB

  Last event: [TOOL] Read internal/coordinator/daemon.go
```

## Success Criteria

- [ ] Streaming events appear in dashboard within 500ms
- [ ] Resource metrics update every 5 seconds
- [ ] Approval request blocks task completion until decision
- [ ] `ailang coordinator status --watch` shows live updates
- [ ] `ailang coordinator approve <id>` works from CLI
- [ ] Diff preview available before merge approval
- [ ] Auto-reject after configurable timeout
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- EventHandler broadcasts correct event types
- ResourceTracker calculates metrics correctly
- ApprovalCheckpoint blocks and resumes properly
- Timeout behavior works as expected

**Integration tests:**
- End-to-end: Send task → See streaming logs → Approve merge → Changes applied
- Dashboard WebSocket receives events
- CLI watch mode displays updates

**Manual testing:**
- Run 5-minute task and verify dashboard shows progress
- Reject merge approval and verify worktree preserved
- Test timeout behavior (1 hour in prod, 10s in test)

## Non-Goals

**Not in this feature:**
- Automatic merge without human approval (safety first)
- Video/audio streaming from executor
- Multi-reviewer approval workflow (single approver sufficient)
- Undo/rollback after merge approved

## Timeline

**Day 1** (8 hours):
- Phase 1: Streaming Event Handler
- Basic dashboard log view

**Day 2** (6 hours):
- Phase 2: Resource Monitoring
- Dashboard resource widgets

**Day 3** (8 hours):
- Phase 3: Approval Checkpoints (core)
- CLI approve command

**Day 4** (8 hours):
- Phase 3: Approval UI (dashboard)
- Phase 4: Polish and integration
- Documentation

**Total: ~30 hours across 4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| High-frequency events overwhelm WebSocket | Medium | Rate limiting (10 events/sec), batching |
| Long approval timeout blocks worktree resources | Medium | Auto-reject with warning, configurable timeout |
| PID-based resource tracking unreliable | Low | Fallback to task-level metrics from executor |
| Dashboard not open when approval needed | High | CLI notification, optional webhook/email |

## Related Documents

**Implemented (inform design):**
- [design_docs/implemented/v0_5_6/m-msg-agent-messaging-improvements.md](design_docs/implemented/v0_5_6/m-msg-agent-messaging-improvements.md) - Messaging patterns
- [design_docs/implemented/v0_3_0/m_eval_ai_benchmarking.md](design_docs/implemented/v0_3_0/m_eval_ai_benchmarking.md) - Eval harness patterns

**Planned (check for overlap):**
- [design_docs/planned/v0_6_2/execution-profiles.md](design_docs/planned/v0_6_2/execution-profiles.md) - Resource profiles
- [design_docs/planned/v0_6_2/global-collaboration-hub.md](design_docs/planned/v0_6_2/global-collaboration-hub.md) - Cloud scaling

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/executor/executor.go` - EventHandler interface
- `internal/server/monitor.go` - ProcessStats structure
- `internal/server/handlers_approvals.go` - Existing approval system
- `internal/coordinator/daemon.go` - Current coordinator implementation

## Future Work

- Cloud-hosted coordinator with global approval queue
- Slack/Discord notifications for pending approvals
- Multi-reviewer approval for high-risk changes
- Automated merge policies (e.g., auto-approve test-only changes)
- Resource quotas and cost limits per task

---

**Document created**: 2025-12-29
**Last updated**: 2025-12-29
