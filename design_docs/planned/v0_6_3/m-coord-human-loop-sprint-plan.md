# M-COORD-HUMAN-LOOP Sprint Plan

| Field | Value |
|-------|-------|
| Sprint ID | M-COORD-HUMAN-LOOP |
| Design Doc | [m-coord-human-loop.md](m-coord-human-loop.md) |
| Target Version | v0.6.2 |
| Duration | 1.5 days (6 hours) |
| Priority | P0 (Critical) |
| Risk Level | Low |
| Created | 2025-12-30 |

## Sprint Goal

**Make coordinator work visible and mergeable to main worktree with human approval.**

Currently, AI-executed tasks complete in isolated worktrees but the work is never visible or integrated. This sprint fixes that by:
1. Streaming real-time updates to dashboard
2. Requiring human approval before merging to main worktree
3. Providing merge/reject controls in dashboard

**No GitHub PRs** - just local git operations with approval gate.

## Success Metrics

- [ ] Real-time streaming visible in dashboard during task execution
- [ ] Completed tasks show in "Pending Approval" section
- [ ] Human can view diff and approve/reject
- [ ] Approved work merges to main worktree
- [ ] Rejected work can be retried or discarded
- [ ] No work is ever "lost" in worktrees

## Milestones

### M1: Fix Streaming Pipeline (1.5 hours)

**Goal**: Verify events flow from executor → daemon → server → UI

**Tasks**:
1. Add debug logging to HTTP broadcaster
2. Test event endpoint manually with curl
3. Verify event handler is attached during execution
4. Add reconnect logic if server restarts
5. Test with simple task, verify UI shows updates

**Estimated LOC**: ~60 LOC
- `http_broadcaster.go`: +25 LOC (logging, reconnect)
- `event_handler.go`: +15 LOC (debug logging)
- `handlers_coordinator.go`: +20 LOC (event logging)

**Acceptance Criteria**:
- [ ] Send test event via curl, see in browser console
- [ ] Start simple task, see streaming text in dashboard
- [ ] Restart server mid-task, events resume

**APPROVAL GATE**: User verifies streaming works in dashboard before continuing.

---

### M2: Pending Approval Queue (1.5 hours)

**Goal**: Show completed tasks awaiting human approval in dashboard

**Tasks**:
1. Add `worktree_path` and `approval_status` to task record
2. Create `/api/coordinator/pending-merges` endpoint
3. Create `PendingMerges` React component
4. Show task summary, diff preview, approve/reject buttons
5. Wire up to existing dashboard layout

**Estimated LOC**: ~180 LOC
- `internal/coordinator/store_sqlite.go`: +20 LOC (new fields)
- `internal/server/handlers_coordinator.go`: +40 LOC (endpoint)
- `ui/src/components/PendingMerges/PendingMerges.tsx`: +90 LOC
- `ui/src/components/PendingMerges/PendingMerges.module.css`: +30 LOC

**Acceptance Criteria**:
- [ ] Dashboard shows completed tasks in "Pending Approval" section
- [ ] Each task shows: title, cost, tokens, worktree path
- [ ] "View Diff" shows git diff from worktree
- [ ] Approve and Reject buttons visible

**APPROVAL GATE**: User sees real completed task in pending approval queue.

---

### M3: Merge and Cleanup Workflow (1.5 hours)

**Goal**: Merge approved work to main worktree, clean up

**Tasks**:
1. On approval: `git merge` worktree changes to main
2. Clean up worktree after successful merge
3. On rejection: mark task as rejected, optionally keep worktree
4. Update task status and notify

**Estimated LOC**: ~120 LOC
- `internal/coordinator/daemon.go`: +60 LOC (approval handling)
- `internal/coordinator/merge.go`: +60 LOC (merge logic)

**Acceptance Criteria**:
- [ ] Approve → changes merged to main worktree
- [ ] Worktree cleaned up after merge
- [ ] Reject → task marked rejected
- [ ] Can see merged changes with `git log` in main repo

**APPROVAL GATE**: User approves task via dashboard, sees changes in main worktree.

---

### M4: End-to-End Verification (1.5 hours)

**Goal**: Verify complete flow works reliably

**Tasks**:
1. Send new task to coordinator
2. Watch streaming in dashboard
3. Task completes → appears in pending approval
4. Review diff → approve
5. Verify changes in main worktree
6. Test rejection flow
7. Test multiple concurrent tasks

**Acceptance Criteria**:
- [ ] Full happy path works
- [ ] Rejection flow works
- [ ] Streaming stays connected during long tasks
- [ ] Multiple tasks don't interfere

**APPROVAL GATE**: User declares victory after successful end-to-end test.

---

## Total Estimates

| Milestone | LOC | Duration |
|-----------|-----|----------|
| M1: Fix Streaming | 60 | 1.5 hours |
| M2: Pending Approval | 180 | 1.5 hours |
| M3: Merge Workflow | 120 | 1.5 hours |
| M4: E2E Verification | 0 | 1.5 hours |
| **Total** | **360** | **6 hours** |

## Workflow After Sprint

```
1. Human creates GitHub issue or sends message
2. Coordinator picks up task
3. Dashboard shows streaming output (REAL-TIME)
4. Task completes → appears in "Pending Approval"
5. Human reviews diff → clicks Approve/Reject
6. Approved: changes merge to main worktree
7. Rejected: worktree kept for retry
```

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Merge conflicts | Medium | Medium | Detect early, show to user |
| WebSocket disconnects | Medium | Low | Reconnect logic |
| Long-running tasks | Low | Low | Timeout handling exists |

## Dependencies

- Coordinator daemon running
- Server running on port 1957
- Worktree already created by coordinator

## Approval Gates Summary

| Gate | What User Verifies |
|------|-------------------|
| After M1 | Streaming events visible in dashboard |
| After M2 | Completed task shows in pending approval |
| After M3 | Approved task merges to main worktree |
| After M4 | Full end-to-end flow works |

**No milestone proceeds without user approval.**
