# Sprint Plan: M-COORD-STABLE

| Field | Value |
|-------|-------|
| Sprint ID | M-COORD-STABLE |
| Design Doc | [m-coord-stable.md](m-coord-stable.md) |
| Target Version | v0.6.2 |
| Duration | 3 days (20 hours) |
| Priority | P0 (Critical) |
| Risk Level | Medium |
| Created | 2025-12-31 |

## Sprint Goal

**Make the coordinator daemon a rock-solid foundation for autonomous agent collaboration.**

Currently:
- Work is lost in worktrees (deleted immediately after completion)
- ApprovalCheckpoint exists but isn't wired
- Dashboard streaming returns 404
- Only hardcoded "coordinator" inbox

After this sprint:
- Zero work lost - preserved until explicit approval/rejection
- Real-time streaming to dashboard
- Configurable agents with workspaces
- Agent-to-agent messaging with approval gates
- GitHub issues auto-trigger tasks

## Current Status Analysis

### Completed Recently
- ✅ Coordinator daemon basic structure
- ✅ SQLite task store (23 tasks, 387 events stored)
- ✅ ApprovalCheckpoint mechanism (but not wired!)
- ✅ WorktreeManager (but deletes immediately!)
- ✅ HTTPBroadcaster (but getting 404s!)
- ✅ Server handlers for /api/coordinator/events

### Velocity
- Recent work: ~200-300 LOC/day (coordinator refactoring)
- Estimated capacity: ~350 LOC/day for focused work
- Sprint estimate: 1030 LOC over 3 days (~343 LOC/day)

### Critical Bugs to Fix
| Bug | Location | Impact |
|-----|----------|--------|
| Worktrees deleted | daemon.go:509-514 | Work lost |
| ApprovalCheckpoint unwired | daemon.go | Approvals broken |
| HTTP 404 | http_broadcaster.go | No streaming |
| Metrics race | event_handler.go:204 | All metrics 0 |

## Milestones

### M1: Fix Critical Bugs (Day 1 Morning)
**Goal:** Stop losing work, fix streaming
**Estimated:** 150 LOC (implementation) + 50 LOC (tests) = 200 LOC
**Duration:** 4 hours

**Tasks:**
1. Remove premature worktree deletion (daemon.go:509-514)
2. Add `pending_approval` status and `worktree_path` field
3. Wire ApprovalCheckpoint in Daemon struct
4. Debug and fix HTTP 404 (check route registration)
5. Fix metrics race condition (store with final status)

**Acceptance Criteria:**
- [ ] Task completes → worktree NOT deleted
- [ ] Task marked `pending_approval` with worktree path
- [ ] HTTP POST to /api/coordinator/events returns 200
- [ ] Final status event includes token/cost metrics
- [ ] All tests passing, linting clean

**APPROVAL GATE**: User verifies worktrees preserved after task completion.

---

### M2: Agent Configuration System (Day 1 Afternoon)
**Goal:** Load agent configuration from YAML
**Estimated:** 350 LOC (implementation) + 100 LOC (tests) = 450 LOC
**Duration:** 6 hours

**Tasks:**
1. Create `internal/coordinator/agent_registry.go` (~150 LOC)
2. Create `internal/coordinator/config.go` - YAML loading (~100 LOC)
3. Add agent config schema to ~/.ailang/config.yaml
4. Create sample config with coordinator, sprint-planner, sprint-executor
5. Write unit tests for registry and config loading

**Files to create/modify:**
- NEW: `internal/coordinator/agent_registry.go`
- NEW: `internal/coordinator/config.go`
- NEW: `internal/coordinator/agent_registry_test.go`
- MOD: `~/.ailang/config.yaml` (sample config)

**Acceptance Criteria:**
- [ ] AgentConfig struct with all fields
- [ ] Load agents from YAML config
- [ ] GetAgentForInbox(inbox) returns correct agent
- [ ] GetAgentByID(id) returns correct agent
- [ ] Missing config handled gracefully
- [ ] All tests passing

**APPROVAL GATE**: User verifies agents load from config.

---

### M3: Multi-Inbox Message Routing (Day 2 Morning)
**Goal:** Watch multiple agent inboxes, route to correct agent
**Estimated:** 150 LOC
**Duration:** 3 hours

**Tasks:**
1. Replace hardcoded "coordinator" inbox with registry lookup
2. Create inbox watchers for each configured agent
3. Route messages to agent-specific worktree managers
4. Use agent's workspace for worktree creation

**Files to modify:**
- MOD: `internal/coordinator/daemon.go` - initTaskProcessing, pollAndProcessTasks

**Acceptance Criteria:**
- [ ] Messages to `sprint-planner` inbox reach sprint-planner agent
- [ ] Each agent gets worktrees in its configured workspace
- [ ] Unconfigured inboxes logged but don't crash
- [ ] All tests passing

**APPROVAL GATE**: User sends message to `sprint-planner`, verifies it's picked up.

---

### M4: Agent-to-Agent Messaging (Day 2 Afternoon)
**Goal:** Agents can trigger other agents with approval gates
**Estimated:** 200 LOC
**Duration:** 4 hours

**Tasks:**
1. Add `trigger_on_complete` handling in onTaskCompleted
2. Implement approval gate for handoffs (when auto_approve_handoffs=false)
3. Add `ApprovalTypeHandoff` constant
4. Store and pass session IDs for conversation continuity
5. Parse @agent directives in executor output

**Files to modify:**
- MOD: `internal/coordinator/daemon.go` - onTaskCompleted
- MOD: `internal/coordinator/approval_checkpoint.go` - ApprovalTypeHandoff
- MOD: `internal/coordinator/store.go` - SessionID field

**Acceptance Criteria:**
- [ ] Agent A completion → approval request for Agent B handoff
- [ ] Approved handoff → message sent to Agent B inbox
- [ ] Session ID passed in handoff message
- [ ] auto_approve_handoffs=true skips approval
- [ ] All tests passing

**APPROVAL GATE**: User approves handoff, verifies downstream agent triggered.

---

### M5: GitHub Sync Auto-Trigger (Day 3 Morning)
**Goal:** GitHub issues auto-import and trigger tasks
**Estimated:** 80 LOC
**Duration:** 2 hours

**Tasks:**
1. Add GitHub sync ticker to main loop (5min interval)
2. Call `ailang messages import-github` periodically
3. Add config for sync interval and watch labels
4. Log sync results

**Files to modify:**
- MOD: `internal/coordinator/daemon.go` - Run loop

**Acceptance Criteria:**
- [ ] GitHub sync runs every 5 minutes (configurable)
- [ ] New issues imported as messages
- [ ] Messages trigger task creation
- [ ] Sync errors logged, don't crash daemon
- [ ] All tests passing

**APPROVAL GATE**: User creates GitHub issue, verifies auto-imported and task created.

---

### M6: Approval Workflow & Merge (Day 3 Afternoon)
**Goal:** Human approval → merge to main branch
**Estimated:** 200 LOC
**Duration:** 4 hours

**Tasks:**
1. Implement requestApproval() - create approval request after task completion
2. Implement handleApproval() - merge worktree to main branch
3. Implement handleRejection() - keep worktree, mark rejected
4. Add auto-commit of uncommitted changes before merge
5. Handle merge conflicts gracefully

**Files to modify:**
- MOD: `internal/coordinator/daemon.go` - requestApproval, handleApproval, handleRejection
- NEW: `internal/coordinator/merge.go` (~100 LOC)

**Acceptance Criteria:**
- [ ] Task completion → approval request created
- [ ] Dashboard shows pending approvals
- [ ] Approve → changes merged to main branch
- [ ] Reject → worktree preserved, task marked rejected
- [ ] Merge conflicts detected and reported
- [ ] All tests passing

**APPROVAL GATE**: User approves task via dashboard, verifies changes in git log.

---

### M7: Dashboard Integration (Day 3 Evening)
**Goal:** Dashboard shows pending approvals with diff viewer
**Estimated:** 100 LOC
**Duration:** 2 hours

**Tasks:**
1. Enhance /api/coordinator/pending with worktree info
2. Add /api/coordinator/tasks/{id}/diff endpoint
3. Verify WebSocket broadcast working
4. End-to-end test

**Files to modify:**
- MOD: `internal/server/handlers_coordinator.go`
- MOD: `internal/server/server.go` - route registration

**Acceptance Criteria:**
- [ ] Pending approvals show worktree path, files changed
- [ ] Diff endpoint returns git diff output
- [ ] Streaming events reach dashboard
- [ ] Approve/reject buttons work
- [ ] All tests passing

**APPROVAL GATE**: User runs full end-to-end test successfully.

---

## Day-by-Day Schedule

| Day | Morning (4h) | Afternoon (4h) | Evening (2h) |
|-----|--------------|----------------|--------------|
| Day 1 | M1: Fix Critical Bugs | M2: Agent Configuration | Buffer |
| Day 2 | M3: Multi-Inbox Routing | M4: Agent-to-Agent Messaging | Buffer |
| Day 3 | M5: GitHub Sync | M6: Approval Workflow | M7: Dashboard |

## Success Metrics

- [ ] Zero work lost in worktrees
- [ ] Dashboard shows real-time streaming
- [ ] 3+ agents configurable via YAML
- [ ] Agent-to-agent handoffs work with approval
- [ ] GitHub issues auto-trigger tasks
- [ ] Approve/reject workflow functional
- [ ] All tests passing
- [ ] Linting clean

## Total Estimates

| Milestone | LOC | Duration |
|-----------|-----|----------|
| M1: Critical Bugs | 200 | 4h |
| M2: Agent Config | 450 | 6h |
| M3: Multi-Inbox | 150 | 3h |
| M4: Agent Messaging | 200 | 4h |
| M5: GitHub Sync | 80 | 2h |
| M6: Approval Workflow | 200 | 4h |
| M7: Dashboard | 100 | 2h |
| **Total** | **1380** | **25h** |

*Note: 25h estimated but scheduled over 3 days with buffer time.*

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Merge conflicts | Medium | Medium | Detect early, keep worktree for manual fix |
| Config loading issues | Low | Medium | Validate config at startup, clear errors |
| WebSocket disconnects | Medium | Low | Reconnect logic already exists |
| Agent infinite loops | Low | High | Max-depth for handoff chains |

## Dependencies

- Coordinator daemon running
- Server running on port 1957
- SQLite databases accessible
- GitHub CLI authenticated

## Notes

- Each milestone has an approval gate requiring user verification
- Buffer time on Days 1-2 evenings for overflow
- M2 is the largest milestone - may need to split if running long
- Session ID integration leverages Claude Code/Gemini CLI features (don't reimplement)
