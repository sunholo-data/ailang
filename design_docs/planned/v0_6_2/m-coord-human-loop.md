# M-COORD-HUMAN-LOOP: Human-in-the-Loop Coordinator Workflow

| Field | Value |
|-------|-------|
| Status | Planned |
| Target | v0.6.2 |
| Priority | P0 (Critical) |
| Estimated | 2 days |
| Dependencies | M-COLLAB-PROVIDER-STATS (completed) |
| Created | 2025-12-30 |
| GitHub Issues | #79 (Heatmap calendar - work lost in worktree) |

## Problem Statement

The coordinator daemon executes tasks in isolated worktrees but **work is effectively lost** because:

1. **No streaming visibility**: Human operators cannot see what the agent is doing in real-time
2. **No human approval gate**: Completed work sits in worktrees forever, never merged
3. **No PR creation**: Work is not converted to reviewable pull requests
4. **No merge workflow**: Even if humans approve, there's no path to integrate the changes

### Evidence

- Task `task-593740ab` completed a full heatmap implementation (11.6KB React component, Go backend)
- Work exists in `~/.ailang/state/worktrees/coordinator/task-593740ab/`
- Changes never reached the main codebase or GitHub
- $2+ spent on AI execution with no visible output

## Goals

**Primary Goal:** Make coordinator work visible, reviewable, and mergeable by humans.

**Success Metrics:**
1. Real-time streaming of agent activity visible in dashboard
2. Automatic PR creation when task completes
3. Human approval required before merge
4. Work is never "lost" - always ends in PR or rejection

## Solution Design

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Coordinator Daemon                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Task Execution        Streaming Events       Completion             │
│  ───────────────      ────────────────────   ────────────────────   │
│  1. Create worktree    → OnTurnStart          → Create PR            │
│  2. Run executor       → OnText               → Request approval     │
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
│  3. Handle approvals   → POST /api/coordinator/approve/{id}          │
└─────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Fix Streaming (Day 1 Morning)

**Problem:** Events sent but not reaching UI

1. Add debug logging to HTTP broadcaster
2. Verify event handler is connected during execution
3. Test round-trip: executor → daemon → server → WebSocket → UI
4. Add connection health check and auto-reconnect

**Files:**
- `internal/coordinator/http_broadcaster.go` - Add logging, reconnect
- `internal/coordinator/event_handler.go` - Verify emission
- `internal/server/handlers_coordinator.go` - Add event logging

### Phase 2: Automatic PR Creation (Day 1 Afternoon)

**On task completion:**

1. Commit all changes in worktree with descriptive message
2. Push branch to origin (branch name: `coordinator/<task-id>`)
3. Create GitHub PR using `gh pr create`
4. Link PR to original GitHub issue if present
5. Update task record with PR URL

**Files:**
- `internal/coordinator/daemon.go` - Add PR creation after task completes
- `internal/coordinator/github_pr.go` - New file: PR creation logic

**PR Template:**
```markdown
## Task: {task.Title}

{task.Content}

---

### Changes Made

{git diff --stat}

### Executor Output

{task.Output (truncated)}

---

**Created by:** AILANG Coordinator
**Task ID:** {task.ID}
**Thread:** {task.ThreadID}
**Cost:** ${task.Cost}
**Tokens:** {task.TokensUsed}

🤖 This PR was automatically created by the AILANG Coordinator daemon.
Human review required before merge.
```

### Phase 3: Human Approval Gate (Day 2 Morning)

**Dashboard UI additions:**

1. "Pending PRs" section showing completed tasks awaiting review
2. PR preview with diff viewer
3. Approve/Reject buttons
4. Comments field for feedback

**Backend:**

1. New approval status: `pr_pending_review`
2. Approval resolves to `approved` or `rejected`
3. On approval: Merge PR and clean up worktree
4. On rejection: Close PR with reason, keep worktree for retry

**Files:**
- `internal/coordinator/store_sqlite.go` - Add PR tracking fields
- `internal/server/handlers_coordinator.go` - Approval with merge
- `ui/src/components/PendingPRs/` - New component

### Phase 4: Merge and Cleanup (Day 2 Afternoon)

**After human approval:**

1. Merge PR using `gh pr merge --merge`
2. Delete remote branch
3. Clean up worktree
4. Update task status to `merged`
5. Send completion notification

**After rejection:**

1. Close PR with rejection reason
2. Keep worktree for potential retry
3. Update task status to `rejected`
4. Create new message for retry if requested

**Files:**
- `internal/coordinator/daemon.go` - Merge workflow
- `internal/coordinator/cleanup.go` - Worktree cleanup

## Implementation Plan

### Day 1: Streaming + PR Creation

**Morning (4h):**
- [ ] Debug current streaming pipeline
- [ ] Add reconnect logic to HTTP broadcaster
- [ ] Verify events reach UI

**Afternoon (4h):**
- [ ] Implement PR creation on task completion
- [ ] Test with real task
- [ ] Handle edge cases (no changes, conflicts)

### Day 2: Approval + Merge

**Morning (4h):**
- [ ] Add PR tracking to database
- [ ] Implement approval API endpoints
- [ ] Build Pending PRs UI component

**Afternoon (4h):**
- [ ] Implement merge workflow
- [ ] Implement rejection workflow
- [ ] Test full cycle: task → PR → approval → merge

## Success Criteria

- [ ] Agent activity visible in real-time in dashboard
- [ ] Completed tasks automatically create GitHub PRs
- [ ] PRs cannot be merged without human approval
- [ ] Approved PRs merge cleanly
- [ ] Rejected tasks can be retried
- [ ] No work is ever "lost" in worktrees

## Testing Plan

1. **Streaming test:** Send task, verify dashboard shows live updates
2. **PR creation test:** Complete task, verify PR appears on GitHub
3. **Approval flow test:** Approve PR, verify merge
4. **Rejection flow test:** Reject PR, verify proper cleanup
5. **Edge cases:** Network failures, conflicts, timeout handling

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Merge conflicts | Detect early, request human resolution |
| GitHub rate limits | Batch operations, exponential backoff |
| Worktree accumulation | Scheduled cleanup job |
| Large diffs | Truncate in PR, link to full diff |

## Notes

This is a P0 priority because without it, all coordinator work is effectively discarded.
The current system spends money on AI execution with no lasting output.
