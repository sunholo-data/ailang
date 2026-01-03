# M-COORD-HUMAN-LOOP: Human-in-the-Loop Coordinator Workflow

| Field | Value |
|-------|-------|
| Status | Planned |
| Target | v0.6.2 |
| Priority | P0 (Critical) |
| Estimated | 1.5 days |
| Dependencies | M-COLLAB-PROVIDER-STATS (completed) |
| Created | 2025-12-30 |
| GitHub Issues | #79 (Heatmap calendar - work lost in worktree) |

## Problem Statement

The coordinator daemon executes tasks in isolated worktrees but **work is effectively lost** because:

1. **No streaming visibility**: Human operators cannot see what the agent is doing in real-time
2. **No human approval gate**: Completed work sits in worktrees forever, never merged
3. **No merge workflow**: Even if humans want to approve, there's no path to integrate the changes

### Evidence

- Task `task-593740ab` completed a full heatmap implementation (11.6KB React component, Go backend)
- Work existed in `~/.ailang/state/worktrees/coordinator/task-593740ab/`
- Changes never reached the main codebase
- $2+ spent on AI execution with no visible output
- Worktree was cleaned up, work is now permanently lost

## Goals

**Primary Goal:** Make coordinator work visible, reviewable, and mergeable to main worktree with human approval.

**Success Metrics:**
1. Real-time streaming of agent activity visible in dashboard
2. Completed tasks appear in "Pending Approval" queue
3. Human approval required before merge to main worktree
4. Work is never "lost" - always ends in merge or rejection

**No GitHub PRs** - just local git operations with approval gate.

## Solution Design

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Coordinator Daemon                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Task Execution        Streaming Events       Completion             │
│  ───────────────      ────────────────────   ────────────────────   │
│  1. Create worktree    → OnTurnStart          → Mark pending_approval│
│  2. Run executor       → OnText               → Show in dashboard    │
│  3. Track progress     → OnToolUse/Result     → Wait for human       │
│                        → OnError              → Merge or reject      │
│                                                                      │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼ HTTP POST /api/coordinator/events
┌─────────────────────────────────────────────────────────────────────┐
│                    Collaboration Hub Server                          │
├─────────────────────────────────────────────────────────────────────┤
│  1. Receive events     → WebSocket broadcast → UI Dashboard          │
│  2. Store task state   → SQLite database                             │
│  3. Handle approvals   → POST /api/coordinator/merge/{id}            │
│  4. Execute merge      → git merge to main worktree                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Fix Streaming

**Problem:** Events sent but not reaching UI

1. Add debug logging to HTTP broadcaster
2. Verify event handler is connected during execution
3. Test round-trip: executor → daemon → server → WebSocket → UI
4. Add connection health check and auto-reconnect

**Files:**
- `internal/coordinator/http_broadcaster.go` - Add logging, reconnect
- `internal/coordinator/event_handler.go` - Verify emission
- `internal/server/handlers_coordinator.go` - Add event logging

### Phase 2: Pending Approval Queue

**On task completion:**

1. Mark task as `pending_approval`
2. Keep worktree with committed changes
3. Show in dashboard "Pending Approval" section

**Dashboard UI additions:**

1. "Pending Approval" section showing completed tasks awaiting review
2. Diff preview from worktree
3. Approve/Reject buttons
4. Task details (cost, tokens, duration)

**Files:**
- `internal/coordinator/store_sqlite.go` - Add worktree_path, approval_status fields
- `internal/server/handlers_coordinator.go` - Pending merges endpoint
- `ui/src/components/PendingMerges/` - New component

### Phase 3: Merge and Cleanup

**After human approval:**

1. Merge worktree changes to main worktree via `git merge`
2. Clean up worktree
3. Update task status to `merged`
4. Send completion notification

**After rejection:**

1. Keep worktree for potential retry
2. Update task status to `rejected`
3. Create new message for retry if requested

**Files:**
- `internal/coordinator/daemon.go` - Approval handling
- `internal/coordinator/merge.go` - Merge logic

## Success Criteria

- [ ] Agent activity visible in real-time in dashboard
- [ ] Completed tasks appear in "Pending Approval" queue
- [ ] Human can view diff and approve/reject
- [ ] Approved changes merge to main worktree
- [ ] Rejected tasks can be retried
- [ ] No work is ever "lost" in worktrees

## Testing Plan

1. **Streaming test:** Send task, verify dashboard shows live updates
2. **Pending approval test:** Complete task, verify appears in queue
3. **Approval flow test:** Approve task, verify merge to main worktree
4. **Rejection flow test:** Reject task, verify proper handling
5. **Edge cases:** Network failures, merge conflicts, timeout handling

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Merge conflicts | Detect early, show to user for resolution |
| WebSocket disconnects | Reconnect logic, event buffering |
| Worktree accumulation | Scheduled cleanup job |

## Notes

This is a P0 priority because without it, all coordinator work is effectively discarded.
The current system spends money on AI execution with no lasting output.
