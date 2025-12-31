# M-COORD-UI-APPROVALS: Dashboard Coordinator Approval Workflow

**Status**: Planned
**Target**: v0.6.3
**Priority**: P0 - High (enables primary user interaction pattern)
**Estimated**: 3 days
**Dependencies**: M-COORD-STABLE (v0.6.2) - Complete

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | UI layer, doesn't affect language semantics |
| A2: Replayability | +1 | Approval decisions are logged and auditable |
| A3: Effect Legibility | +1 | Shows git diffs - makes effects visible |
| A4: Explicit Authority | +1 | Human approval gate for agent actions |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | Single-user UI, no concurrent concerns |
| A7: Machines First | +1 | Structured API for agent-UI communication |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | Future: could show token costs |
| A10: Composability | +1 | Integrates with existing approval system |
| A11: Structured Failure | +1 | Clear approve/reject with notes |
| A12: System Boundary | +1 | Explicit human-agent boundary |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): Enforces human authority over agent actions
- [x] A7 (Machines First): Structured JSON APIs

## Problem Statement

The coordinator daemon can execute tasks autonomously using AI agents (Claude Code, Gemini CLI), but **there's no way for users to approve or reject completed work through the dashboard UI**.

**Current State:**
- CLI commands exist (`ailang coordinator pending/approve/reject`) - functional but not discoverable
- Dashboard has `ApprovalQueue` component but it handles **capability approvals** (FS/Net/IO), not coordinator task approvals
- Users must use terminal to interact with the coordinator approval workflow
- No way to view git diffs of agent work before approving

**Impact:**
- Primary users (developers) need to context-switch to terminal for approvals
- Can't visualize what agents changed before merging
- No notification of pending approvals in the UI
- Approval workflow is hidden and undiscoverable

## Goals

**Primary Goal:** Make the dashboard the primary interface for approving/rejecting coordinator task completions, with full visibility into agent work.

**Success Metrics:**
- Users can approve/reject coordinator tasks entirely from the dashboard
- Git diffs are visible before approval decisions
- Pending approvals show notification badge in UI
- End-to-end workflow documented for users

## Solution Design

### Overview

Add a new `CoordinatorApprovalPanel` component to the dashboard that:
1. Fetches pending approvals from `/api/coordinator/pending`
2. Shows task details (title, type, agent, worktree path)
3. Displays git diff of changes made by the agent
4. Allows approve/reject with optional notes
5. Shows history of past approval decisions

### Architecture

**Components:**

1. **CoordinatorApprovalPanel** (new): Main approval interface
   - Fetches from `/api/coordinator/pending`
   - Displays pending tasks with expandable details
   - Integrates with DiffViewer for code review

2. **DiffViewer** (new): Git diff visualization
   - Fetches from `/api/coordinator/tasks/{id}/diff`
   - Syntax-highlighted diff view
   - File-by-file navigation

3. **ApprovalNotificationBadge** (new): Header notification
   - Shows count of pending approvals
   - Links to approval panel

**API Endpoints (already exist in coordinator.go):**
- `GET /api/coordinator/pending` - List pending approvals
- `POST /api/coordinator/approve/{id}` - Approve task
- `POST /api/coordinator/reject/{id}` - Reject task
- `GET /api/coordinator/tasks/{id}/diff` - Get git diff

### Implementation Plan

**Phase 1: Core Approval Panel** (~8 hours)
- [ ] Create `CoordinatorApprovalPanel.tsx` component
- [ ] Add approval panel to dashboard layout (new tab or section)
- [ ] Implement pending approvals list with basic details
- [ ] Add approve/reject buttons with confirmation
- [ ] Wire up to existing API endpoints

**Phase 2: Git Diff Viewer** (~6 hours)
- [ ] Create `DiffViewer.tsx` component
- [ ] Parse and render git diff output with syntax highlighting
- [ ] Add file-by-file navigation for multi-file diffs
- [ ] Show additions/deletions summary

**Phase 3: Notifications and Polish** (~4 hours)
- [ ] Add `ApprovalNotificationBadge` to header
- [ ] WebSocket subscription for real-time updates
- [ ] Approval history section
- [ ] Loading states and error handling

**Phase 4: Documentation** (~2 hours)
- [ ] Update collaboration-hub.md with approval workflow
- [ ] Add user guide section to coordinator.md
- [ ] Screenshots and workflow diagrams

### Files to Modify/Create

**New files:**
- `ui/src/features/coordinator/CoordinatorApprovalPanel/CoordinatorApprovalPanel.tsx` - Main component (~300 LOC)
- `ui/src/features/coordinator/CoordinatorApprovalPanel/CoordinatorApprovalPanel.module.css` - Styles (~150 LOC)
- `ui/src/features/coordinator/DiffViewer/DiffViewer.tsx` - Diff visualization (~200 LOC)
- `ui/src/features/coordinator/DiffViewer/DiffViewer.module.css` - Styles (~100 LOC)
- `ui/src/components/common/ApprovalNotificationBadge.tsx` - Header badge (~50 LOC)

**Modified files:**
- `ui/src/App.tsx` - Add coordinator approval tab/section (~10 LOC)
- `ui/src/types/index.ts` - Add CoordinatorApproval types (~30 LOC)
- `ui/src/components/layout/Header.tsx` - Add notification badge (~5 LOC)

## User Workflow Documentation

### The Dashboard as Your Command Center

The Collaboration Hub dashboard is designed to be your **primary interface** for managing autonomous AI agents. Here's how a typical day looks:

#### Morning Workflow

1. **Open the dashboard** at http://localhost:1957
2. **Check the notification badge** - shows pending approvals (if any)
3. **Review pending approvals** in the Coordinator Approvals section

#### Approval Process

When an agent completes work:

```
[Agent completes task]
        ↓
[Task appears in Pending Approvals]
        ↓
[Click to expand and review]
        ↓
[View Git Diff - see exactly what changed]
        ↓
[Approve (merges to dev) or Reject (discards worktree)]
```

**What you see for each pending approval:**
- **Task title**: What the agent was working on
- **Agent**: Which agent did the work (claude-code, gemini-cli)
- **Type**: bug-fix, feature, docs, etc.
- **Worktree**: Isolated git worktree where changes were made
- **Session ID**: For resuming conversation if needed
- **Git Diff**: Full diff of all changes

#### Decision Making

**When to Approve:**
- Code looks correct and follows project conventions
- Tests pass (agents should run tests before requesting approval)
- Changes match the original task description
- No obvious security issues or regressions

**When to Reject:**
- Code doesn't solve the problem
- Introduces bugs or fails tests
- Goes beyond scope of original task
- Needs human intervention or different approach

**Adding Notes:**
- When rejecting, explain what needs to change
- Notes are stored and can inform future agent behavior
- Use notes to document your reasoning for audit trail

### End-to-End Multi-Agent Pipeline

Here's the full autonomous development workflow:

```
┌─────────────────────────────────────────────────────────────┐
│  1. GitHub Issue Created                                     │
│     (Manual or from external project)                        │
└───────────────────────────┬─────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  2. Issue imported to inbox                                  │
│     ailang messages import-github (automatic on session)     │
└───────────────────────────┬─────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  3. design-doc-creator agent                                 │
│     - Picks up issue from inbox                              │
│     - Creates design doc in design_docs/planned/             │
│     - Sends handoff message to sprint-planner                │
│     - Requests approval                                      │
└───────────────────────────┬─────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  👤 APPROVAL GATE 1: Review design doc                       │
│     Dashboard: View diff, approve/reject                     │
└───────────────────────────┬─────────────────────────────────┘
                            ↓ (on approve)
┌─────────────────────────────────────────────────────────────┐
│  4. sprint-planner agent                                     │
│     - Receives handoff from design-doc-creator               │
│     - Creates sprint plan with milestones                    │
│     - Creates sprint JSON for tracking                       │
│     - Sends handoff to sprint-executor                       │
│     - Requests approval                                      │
└───────────────────────────┬─────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  👤 APPROVAL GATE 2: Review sprint plan                      │
│     Dashboard: View diff, approve/reject                     │
└───────────────────────────┬─────────────────────────────────┘
                            ↓ (on approve)
┌─────────────────────────────────────────────────────────────┐
│  5. sprint-executor agent                                    │
│     - Receives handoff from sprint-planner                   │
│     - Implements each milestone with TDD                     │
│     - Updates progress in sprint JSON                        │
│     - May request approval at milestone boundaries           │
│     - Final approval on completion                           │
└───────────────────────────┬─────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  👤 APPROVAL GATE 3: Review implementation                   │
│     Dashboard: View full diff, run tests, approve/reject     │
└───────────────────────────┬─────────────────────────────────┘
                            ↓ (on approve)
┌─────────────────────────────────────────────────────────────┐
│  6. Changes merged to dev branch                             │
│     Worktree cleaned up                                      │
│     GitHub issue auto-closed (if linked)                     │
└─────────────────────────────────────────────────────────────┘
```

### CLI Commands (Alternative)

If you prefer terminal over UI:

```bash
# List pending approvals
ailang coordinator pending

# Approve a task (merges changes)
ailang coordinator approve <task-id>

# Reject a task (discards worktree)
ailang coordinator reject <task-id>

# View task diff
ailang coordinator diff <task-id>
```

### Tips for Effective Agent Collaboration

1. **Write clear GitHub issues** - Agents work better with specific, well-defined tasks
2. **Review early, reject fast** - If a design doc is wrong, reject it before sprint planning starts
3. **Trust but verify** - Agents are capable but not perfect; always review diffs
4. **Use rejection notes** - Help agents learn from mistakes
5. **Break big tasks into smaller issues** - Agents handle focused tasks better than sprawling ones

## Examples

### Example 1: Approving a Bug Fix

**Task Details:**
```
Title: Fix null pointer in parser.go
Type: bug-fix
Agent: claude-code
Worktree: ~/.ailang/state/worktrees/coordinator/task-abc123/
Session: session_xyz789
```

**Git Diff Preview:**
```diff
--- a/internal/parser/parser.go
+++ b/internal/parser/parser.go
@@ -142,6 +142,9 @@ func (p *Parser) parseExpression() ast.Expression {
     if p.curToken == nil {
+        return nil
+    }
+    if p.curToken == nil {
         p.errors = append(p.errors, "unexpected nil token")
         return nil
     }
```

**Action:** Click "Approve" → Changes merged to dev, worktree deleted

### Example 2: Rejecting a Feature Implementation

**Task Details:**
```
Title: Add --verbose flag to CLI
Type: feature
Agent: gemini-cli
```

**Git Diff shows:** Flag added but no tests, breaks existing functionality

**Action:** Click "Reject" → Add note: "Missing tests and breaks --help output. Please add tests and fix the help text."

Worktree preserved for potential manual fixing, task marked as rejected.

## Success Criteria

- [ ] Users can list pending coordinator approvals in dashboard
- [ ] Users can view git diffs before approving/rejecting
- [ ] Approve merges worktree to dev branch
- [ ] Reject marks task as rejected and optionally deletes worktree
- [ ] Notification badge shows pending approval count
- [ ] WebSocket updates approval list in real-time
- [ ] All tests passing
- [ ] Documentation updated with user workflow guide
- [ ] Screenshots added to docs

## Testing Strategy

**Unit tests:**
- CoordinatorApprovalPanel renders pending approvals correctly
- DiffViewer parses and displays git diff format
- Approve/reject buttons call correct API endpoints

**Integration tests:**
- Full approval flow: pending → approve → merged
- Full reject flow: pending → reject → marked rejected
- WebSocket updates list when new approvals arrive

**Manual testing:**
- Create coordinator task, let agent complete, approve via UI
- Verify changes appear in dev branch
- Test reject flow and verify worktree handling

## Non-Goals

**Not in this feature:**
- Inline code comments on diffs - Too complex for v1
- Partial approval (approve some files, reject others) - Future enhancement
- Batch approval of multiple tasks - Future enhancement
- Mobile-responsive design - Desktop-first for now

## Timeline

**Day 1** (8 hours):
- Phase 1: Core Approval Panel

**Day 2** (6 hours):
- Phase 2: Git Diff Viewer
- Phase 3 start: Notifications

**Day 3** (6 hours):
- Phase 3 completion: Polish
- Phase 4: Documentation

**Total: ~20 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Git diff parsing edge cases | Medium | Use established diff parsing library |
| Large diffs slow down UI | Medium | Pagination, lazy loading |
| WebSocket connection issues | Low | Fallback to polling, reconnection logic |
| Merge conflicts on approve | Medium | Show error, suggest manual resolution |

## Related Documents

**Implemented (foundation):**
- [design_docs/implemented/v0_6_2/m-coord-stable.md](design_docs/implemented/v0_6_2/m-coord-stable.md) - Coordinator daemon
- [design_docs/planned/v0_6_2/global-collaboration-hub.md](design_docs/planned/v0_6_2/global-collaboration-hub.md) - Dashboard foundation

**Planned (related):**
- [design_docs/planned/v0_6_3/m-coord-human-loop.md](design_docs/planned/v0_6_3/m-coord-human-loop.md) - Human-in-the-loop workflow

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [docs/docs/guides/coordinator.md](docs/docs/guides/coordinator.md) - Coordinator guide
- [docs/docs/guides/collaboration-hub.md](docs/docs/guides/collaboration-hub.md) - Dashboard guide

## Future Work

- Inline diff comments for agent feedback
- Batch approval for related tasks
- Cost visibility per task (tokens, API calls)
- Approval delegation rules (auto-approve for low-risk changes)
- Mobile-responsive design

---

**Document created**: 2025-12-31
**Last updated**: 2025-12-31
