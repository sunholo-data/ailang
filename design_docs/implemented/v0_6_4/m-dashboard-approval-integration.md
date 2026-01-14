# M-DASHBOARD-APPROVAL-INTEGRATION: Complete Dashboard Coordinator Approval Workflow

**Status**: IMPLEMENTED
**Target**: v0.6.4
**Priority**: P0 - High (enables primary user interaction pattern)
**Estimated**: 5 days (40 hours)
**Dependencies**: M-COORD-STABLE (v0.6.2), M-COORD-FEEDBACK (v0.6.2) - Complete

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | UI layer, doesn't affect language semantics |
| A2: Replayability | +1 | Approval decisions create OTEL spans for audit trail |
| A3: Effect Legibility | +1 | Shows git diffs - makes agent effects visible to humans |
| A4: Explicit Authority | +1 | Human approval gate required for agent actions |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | Single-user UI, no concurrent concerns |
| A7: Machines First | +1 | Structured JSON API, telemetry spans for machine analysis |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Shows iteration count, token usage visible |
| A10: Composability | +1 | Integrates with existing approval/feedback systems |
| A11: Structured Failure | +1 | Feedback loop provides structured rejection reasons |
| A12: System Boundary | +1 | Explicit human-agent boundary with approval checkpoints |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects (all made visible)
- [x] A4 (Authority): Enforces human authority over agent actions
- [x] A7 (Machines First): Designed for machine monitoring via OTEL spans

## Problem Statement

The coordinator daemon (v0.6.2) has a complete approval workflow with feedback loops, but **there's a gap between CLI functionality and Dashboard UI**. Users cannot fully interact with the approval system from the dashboard.

**Current State (Investigation Results):**

### What WORKS (CLI/API):
| Feature | Location | Status |
|---------|----------|--------|
| `ailang coordinator approve <id>` | cmd/ailang/coordinator_actions.go:20 | ✅ Full |
| `ailang coordinator reject <id> --feedback` | cmd/ailang/coordinator_actions.go:202 | ✅ Full |
| Feedback loop (3 iterations max) | internal/coordinator/human_interaction.go | ✅ Full |
| OTEL spans for approvals | coordinator_actions.go:52 | ✅ Full |
| Task iteration with `--resume` | PrepareTaskForRetrigger() | ✅ Full |
| GitHub label watching | approval_watcher.go | ✅ Full |
| Interactive `pending` CLI | coordinator_list.go | ✅ Full |

### What's BROKEN (Dashboard):
| Feature | Expected | Actual | Impact |
|---------|----------|--------|--------|
| Event queue filters | Filter by task_start, approval, etc. | Filters don't apply | Users see all events |
| Approval workflow UI | Approve/reject from dashboard | ApprovalPanel exists but incomplete | Must use CLI |
| Iteration status | Show "Iteration 2/3" | Not displayed | Users don't know retry count |
| GitHub feedback comments | Post rejection reason to issue | Only status updates posted | No feedback visibility |
| Approval spans in hierarchy | See approval decisions in span tree | No spans emitted on dashboard action | No audit trail from UI |

**Impact:**
- Primary users must context-switch between dashboard and CLI for approvals
- No visibility into feedback loop iterations
- GitHub issues don't show rejection feedback
- Event queue is noisy (can't filter to just approvals)

## Goals

**Primary Goal:** Enable humans to interact with the approval system through **any channel** (Dashboard, CLI, GitHub) with consistent behavior and full visibility.

**Success Metrics:**
1. All three interaction channels work equivalently:
   - Dashboard: approve/reject with feedback via UI
   - CLI: `ailang coordinator approve/reject --feedback`
   - GitHub: labels + comments harvested as feedback
2. Event queue filters work (filter to 'approval' type shows only approvals)
3. Dashboard shows activity from ALL channels in real-time
4. Feedback flows bidirectionally (dashboard→GitHub, GitHub→dashboard)
5. Approval spans appear in ExecHierarchy (regardless of source)
6. Iteration counter visible on pending approvals (e.g., "Iteration 2/3")
7. Future interaction sources can be added without architectural changes

## Solution Design

### Overview

**Key Insight: Multi-Channel Approval System**

Humans interact with the approval system through **multiple equal channels**. The system must handle all sources consistently and show activity from any channel in all others.

```
┌─────────────────────────────────────────────────────────────────────┐
│  MULTI-CHANNEL APPROVAL ARCHITECTURE                                 │
│                                                                      │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐                │
│  │  Dashboard  │   │  ailang CLI │   │   GitHub    │   [Future...]  │
│  │    UI       │   │  commands   │   │  labels +   │                │
│  │             │   │             │   │  comments   │                │
│  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘                │
│         │                 │                 │                        │
│         └─────────────────┼─────────────────┘                        │
│                           ▼                                          │
│         ┌─────────────────────────────────────┐                     │
│         │  Unified Approval Handler            │                     │
│         │  - Stores feedback event             │                     │
│         │  - Updates task iteration            │                     │
│         │  - Emits OTEL span                   │                     │
│         │  - Sends to agent inbox              │                     │
│         │  - Broadcasts to all channels        │                     │
│         └─────────────────────────────────────┘                     │
│                           │                                          │
│         ┌─────────────────┼─────────────────┐                        │
│         ▼                 ▼                 ▼                        │
│  Dashboard sees     CLI sees output    GitHub gets                   │
│  event in queue     confirmation       comment posted                │
└─────────────────────────────────────────────────────────────────────┘
```

**Channel Behaviors:**

| Channel | Approve | Reject | Feedback Source |
|---------|---------|--------|-----------------|
| Dashboard | Click button | Click + enter feedback | Text input field |
| CLI | `ailang coordinator approve` | `ailang coordinator reject -f "..."` | `--feedback` flag or prompt |
| GitHub | Add `*-approved` label | Add `needs-revision` label | Recent human comments |

**Five interconnected improvements:**

```
┌─────────────────────────────────────────────────────────────────────┐
│  PHASE 1: Fix Event Queue Filtering                                  │
│  - Debug why filters don't apply                                     │
│  - Ensure WebSocket events include correct 'type' field              │
└─────────────────────────────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PHASE 2: GitHub Comment Harvesting (NEW - Critical)                 │
│  - When label detected, fetch recent comments from issue             │
│  - Extract human comments (filter out bot comments)                  │
│  - Use as feedback reason for agent                                  │
│  - Display in dashboard event queue                                  │
└─────────────────────────────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PHASE 3: Complete Approval Workflow UI                              │
│  - Wire ApprovalPanel approve/reject to coordinator endpoints        │
│  - Add feedback text field for rejections (fallback if no GH comment)│
│  - Show iteration count badge                                        │
│  - Show harvested GitHub comments as "Reviewer feedback"             │
└─────────────────────────────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PHASE 4: Bidirectional GitHub Comments                              │
│  - Post agent status updates TO GitHub                               │
│  - Harvest human comments FROM GitHub                                │
│  - Sync both directions in dashboard                                 │
└─────────────────────────────────────────────────────────────────────┘
                                 ↓
┌─────────────────────────────────────────────────────────────────────┐
│  PHASE 5: Approval Telemetry Spans                                   │
│  - Emit spans on approval events (label + comment)                   │
│  - Include comment content as span attribute                         │
│  - Include in ExecHierarchy visualization                            │
└─────────────────────────────────────────────────────────────────────┘
```

### Architecture

**Component 1: Event Queue Fix**

Current issue: `MessageQueue.tsx` filters by `eventType` but WebSocket events may not include the correct type field, or the filtering logic has a bug.

Investigation needed in:
- `ui/src/features/controlplane/hooks/useEventQueue.ts` - How events are fetched
- `internal/websocket/events.go` - Event type definitions
- `internal/server/handlers_coordinator.go` - Event emission

**Component 2: GitHub Comment Harvesting (NEW - Critical)**

When `ApprovalWatcher.pollOnce()` detects a label change, it should:
1. Fetch recent comments from the GitHub issue
2. Filter to human comments (exclude bot/agent comments)
3. Extract the most recent human comment as "feedback"
4. Pass feedback to the approval handler

```go
// New method in GitHubPoster
func (g *GitHubPoster) GetRecentHumanComments(issueNum int, since time.Time) ([]IssueComment, error) {
    // Call GitHub API: GET /repos/{owner}/{repo}/issues/{issue}/comments?since={timestamp}
    // Filter out comments from bot accounts (check user.type == "Bot" or user.login contains "bot")
    // Return human comments ordered by created_at DESC
}

// IssueComment represents a GitHub issue comment
type IssueComment struct {
    ID        int64
    Body      string
    Author    string
    CreatedAt time.Time
    IsBot     bool
}

// Modified ApprovalWatcher.handleEvent
func (w *ApprovalWatcher) handleEvent(ctx context.Context, event *ApprovalEvent) error {
    // NEW: Harvest comments before calling handler
    comments, err := w.poster.GetRecentHumanComments(event.IssueNumber, w.lastPoll)
    if err == nil && len(comments) > 0 {
        event.Feedback = comments[0].Body  // Most recent human comment
        event.FeedbackAuthor = comments[0].Author
    }

    // Call existing handler with enriched event
    return handler(ctx, event)
}
```

**Comment Filtering Rules:**
- Exclude comments from users with `type: "Bot"`
- Exclude comments from users matching `*[bot]` or `*-bot` patterns
- Exclude comments starting with common bot prefixes (e.g., `<!-- Generated by`)
- Only consider comments made AFTER the last agent comment (prevents stale feedback)

**Dashboard Display:**
```
Event Queue:
┌─────────────────────────────────────────────────────────────────────┐
│ [github] needs-revision detected on #123                            │
│   Label added by: @MarkEdmondson1234                                │
│   Feedback: "Please add error handling for nil pointer cases"       │
│   Time: 2 minutes ago                                               │
└─────────────────────────────────────────────────────────────────────┘
```

**Component 3: Dashboard Approval Actions**

Current state:
- `ApprovalPanel.tsx` has approve/reject buttons
- `useApprovals()` hook has `approveApproval()` and `rejectApproval()` methods
- API endpoints exist: `POST /api/approvals/{id}/approve`, `POST /api/approvals/{id}/reject`

Gap:
- Rejection doesn't call coordinator feedback loop (just marks as rejected)
- No iteration count shown
- Server endpoint doesn't trigger OTEL spans or GitHub comments

Solution: Add new endpoint that mirrors CLI behavior:
```
POST /api/coordinator/reject-with-feedback/{id}
Body: { "feedback": "..." }
```

**Component 3: GitHub Feedback Comments**

Current state:
- `github_poster.go` has `PostComment()` method
- Templates exist for completion comments
- NO template for rejection/feedback comments

Solution: Add `RenderFeedbackComment()` template and call from rejection handler:
```go
func (g *GitHubPoster) PostFeedback(issueNum int, feedback HumanFeedback) error {
    body := g.templates.RenderFeedbackComment(feedback)
    return g.PostComment(issueNum, body)
}
```

**Component 4: Approval Telemetry**

Current state:
- CLI approve/reject creates spans (coordinator_actions.go:52)
- Server endpoints DON'T create spans

Solution: Add span creation to `handleCoordinatorApproval()`:
```go
ctx, span := otel.Tracer("coordinator").Start(ctx, "human.approval")
span.SetAttributes(
    attribute.String("task.id", taskID),
    attribute.String("action", action),
    attribute.String("resolved.by", "dashboard"),
)
defer span.End()
```

### Implementation Plan

**Phase 1: Event Queue Filtering Fix** (~4 hours)

- [ ] Debug `MessageQueue.tsx` filter logic
- [ ] Verify event `type` field is set correctly in WebSocket messages
- [ ] Test filter: select "approval" → only see approval events
- [ ] Test filter: select "task_start" → only see task start events
- [ ] Add unit test for filter behavior

**Phase 2: GitHub Comment Harvesting** (~8 hours) **[NEW - Critical]**

- [ ] Add `IssueComment` struct to `internal/coordinator/types.go`
- [ ] Add `GetRecentHumanComments(issueNum, since)` to `GitHubPoster`
- [ ] Implement bot comment filtering (user.type, username patterns)
- [ ] Extend `ApprovalEvent` with `Feedback` and `FeedbackAuthor` fields
- [ ] Modify `ApprovalWatcher.handleEvent()` to harvest comments before dispatch
- [ ] Add event type `github_feedback` to WebSocket events
- [ ] Broadcast harvested feedback to dashboard in real-time
- [ ] Test with real GitHub issue + human comment

**Phase 3: Complete Approval Workflow UI** (~10 hours)

- [ ] Add `POST /api/coordinator/reject-with-feedback/{id}` endpoint
- [ ] Modify `ApprovalPanel` to use new endpoint on reject
- [ ] Add feedback text field with validation (fallback if no GH comment)
- [ ] Display harvested GitHub feedback as "Reviewer says: ..."
- [ ] Fetch task iteration from `/api/coordinator/tasks/{id}` endpoint
- [ ] Display iteration badge: "⟳ Iteration 2/3"
- [ ] Show "Max iterations reached" warning on iteration 3
- [ ] Add confirmation dialog before final rejection
- [ ] Update `useApprovals()` hook to call new endpoint

**Phase 4: Bidirectional GitHub Comments** (~6 hours)

- [ ] Add `RenderFeedbackComment()` template to `templates.go`
- [ ] Add `PostFeedback()` method to `GitHubPoster`
- [ ] Call `PostFeedback()` from rejection handler when issue linked
- [ ] Display GitHub thread activity in dashboard (read-only mirror)
- [ ] Add "needs-revision" label on rejection (remove on re-approval)
- [ ] Test bidirectional: Human comment → Dashboard display → Agent response

**Phase 5: Approval Telemetry Spans** (~6 hours)

- [ ] Add OTEL span to `handleCoordinatorApproval()` server handler
- [ ] Include attributes: task.id, action, resolved.by, iteration, feedback, feedback_source (github/dashboard)
- [ ] Add span for GitHub comment harvest event
- [ ] Verify spans appear in ExecHierarchy after dashboard action
- [ ] Test span linkage: github_label → github_feedback → task_retrigger

**Phase 6: Test Data & Documentation** (~6 hours)

- [ ] Create test coordinator task with pending approval
- [ ] Test GitHub comment harvesting with real issue
- [ ] Document end-to-end workflow in collaboration-hub guide
- [ ] Add screenshots showing GitHub ↔ Dashboard flow
- [ ] Update CLAUDE.md with new workflow

### Files to Modify/Create

**Modified files:**

| File | Changes | LOC |
|------|---------|-----|
| `ui/src/features/controlplane/components/MessageQueue.tsx` | Fix filter logic | ~20 |
| `ui/src/features/controlplane/components/ApprovalPanel/ApprovalPanel.tsx` | Add feedback field, iteration badge, GH feedback display | ~100 |
| `ui/src/hooks/useObservatory.ts` | Update `rejectApproval()` to call new endpoint | ~30 |
| `internal/server/handlers_coordinator.go` | Add reject-with-feedback endpoint, add spans | ~100 |
| `internal/coordinator/github_poster.go` | Add `GetRecentHumanComments()` and `PostFeedback()` | ~120 |
| `internal/coordinator/templates.go` | Add `RenderFeedbackComment()` template | ~30 |
| `internal/coordinator/approval_watcher.go` | Harvest comments in `handleEvent()` | ~50 |
| `internal/coordinator/daemon_approval.go` | Call PostFeedback on GitHub-linked tasks | ~20 |
| `internal/websocket/events.go` | Add `github_feedback` event type | ~20 |

**New files:**

| File | Purpose | LOC |
|------|---------|-----|
| `internal/coordinator/github_comments.go` | `IssueComment` struct, bot filtering logic | ~80 |
| `ui/src/features/controlplane/components/ApprovalPanel/IterationBadge.tsx` | Iteration counter component | ~50 |
| `ui/src/features/controlplane/components/GitHubFeedback.tsx` | Display harvested GH comments | ~60 |
| `internal/server/handlers_coordinator_test.go` | Tests for new endpoint | ~100 |
| `internal/coordinator/github_comments_test.go` | Tests for comment harvesting | ~80 |

**Total estimated changes: ~860 LOC**

## Examples

### Example 1: Rejection via GitHub (Label + Comment)

**Scenario:** Human reviews design doc on GitHub, provides feedback via comment + label

**GitHub Thread:**
```markdown
Issue #123: Fix parser null pointer bug

---
🤖 **design-doc-creator** commented 2 hours ago:

## Design Document: Fix Parser Null Pointer

### Problem
The parser crashes when `curToken` is nil...

### Solution
Add nil check before dereferencing...

---
👤 **MarkEdmondson1234** commented 5 minutes ago:

Looks good overall, but please:
1. Add unit test for the nil case
2. Consider what happens when both curToken and peekToken are nil
3. Check if similar issues exist in other parser methods

---
👤 **MarkEdmondson1234** added label: `needs-revision`
```

**System Flow:**
```
1. ApprovalWatcher polls GitHub (every 60s)
2. Detects `needs-revision` label added to #123
3. NEW: Fetches recent comments since last poll
4. Filters out bot comments (🤖 design-doc-creator)
5. Extracts human comment as feedback
6. Creates ApprovalEvent:
   {
     TaskID: "task-abc123",
     IssueNumber: 123,
     Label: "needs-revision",
     EventType: "needs-revision",
     Feedback: "Looks good overall, but please:\n1. Add unit test...",
     FeedbackAuthor: "MarkEdmondson1234"
   }
7. Calls HandleRejection() with feedback
8. Broadcasts to dashboard via WebSocket
```

**Dashboard Event Queue:**
```
┌─────────────────────────────────────────────────────────────────────┐
│ 🏷️ [github] needs-revision on #123                     2 min ago   │
│   Task: Fix parser null pointer bug                                 │
│   Label added by: @MarkEdmondson1234                               │
│                                                                     │
│   💬 Reviewer feedback:                                            │
│   "Looks good overall, but please:                                 │
│    1. Add unit test for the nil case                               │
│    2. Consider what happens when both curToken and peekToken..."   │
│                                                                     │
│   Status: Task re-queued (Iteration 2/3)                           │
└─────────────────────────────────────────────────────────────────────┘
```

**Agent Receives:**
```
Message to design-doc-creator inbox:
  Title: "Revision requested (Iteration 2/3)"
  Content: "[HUMAN FEEDBACK - Iteration 2]
            Looks good overall, but please:
            1. Add unit test for the nil case
            2. Consider what happens when both curToken and peekToken are nil
            3. Check if similar issues exist in other parser methods"

  Session: claude-session-xyz (for --resume continuity)
```

### Example 2: Rejection via Dashboard UI

**Before (current):**
```
User: Clicks "Reject" button in ApprovalPanel
→ POST /api/approvals/{id}/reject (no feedback)
→ Task marked as rejected
→ NO GitHub comment posted
→ NO iteration support
→ NO span emitted
→ User must use CLI if they want feedback loop
```

**After:**
```
User: Clicks "Reject" button in ApprovalPanel
→ Feedback modal appears: "Why should the agent revise?"
→ User enters: "Missing error handling for edge cases"
→ POST /api/coordinator/reject-with-feedback/{id}
  Body: { "feedback": "Missing error handling for edge cases" }
→ Server:
  1. Creates OTEL span (human.feedback)
  2. Stores feedback event in task_events
  3. Checks iteration < 3
  4. Posts comment to GitHub issue #123
  5. Sends message to agent inbox
  6. Resets task status to queued
→ ApprovalPanel shows: "Task re-queued (Iteration 2/3)"
→ ExecHierarchy shows: approval span with feedback attribute
```

### Example 2: Event Queue Filtering

**Before (current):**
```
Event Queue (50 events, no filter applied):
- task_start: design-doc-creator started
- text: Reading file...
- tool_use: Glob patterns/*.md
- tool_result: Found 3 files
- approval: Pending approval for task-abc
- ... 45 more events ...

User clicks filter "approval"
→ Still shows all 50 events (filter broken)
```

**After:**
```
Event Queue (50 events total):
- [Filter: All] [Filter: approval ✓] [Filter: task_start] ...

User clicks filter "approval"
→ Shows 2 events:
  - approval: Pending approval for task-abc (design-doc-creator)
  - approval: Approved task-def (sprint-planner)
```

### Example 3: Iteration Badge

**Before:**
```
ApprovalPanel:
┌─────────────────────────────────────────┐
│ Pending Approvals (2)                   │
├─────────────────────────────────────────┤
│ [merge] task-abc123                     │
│   Title: Fix parser bug                 │
│   Agent: claude-code                    │
│   Created: 2h ago                       │
│   [Approve] [Reject]                    │
└─────────────────────────────────────────┘
```

**After:**
```
ApprovalPanel:
┌─────────────────────────────────────────┐
│ Pending Approvals (2)                   │
├─────────────────────────────────────────┤
│ [merge] task-abc123       ⟳ Iteration 2/3│
│   Title: Fix parser bug                 │
│   Agent: claude-code                    │
│   Created: 2h ago                       │
│   Previous feedback: "Add tests"        │
│   [Approve] [Reject with Feedback]      │
│                                         │
│ ⚠️ Final attempt - next rejection is    │
│   permanent                             │
└─────────────────────────────────────────┘
```

### Example 4: GitHub Feedback Comment

**Current (only completion comments):**
```markdown
<!-- Posted when task completes -->
## Task Completed: Fix parser bug

| Metric | Value |
|--------|-------|
| Duration | 12m 34s |
| Tokens | 45,231 |
| Cost | $0.23 |

Files changed: 3
```

**After (feedback comment on rejection):**
```markdown
<!-- Posted when task rejected with feedback -->
## 🔄 Revision Requested (Iteration 2/3)

**Reviewer:** dashboard (human)
**Feedback:**

> Missing error handling for edge cases. Please add:
> - Nil check before dereferencing
> - Test for empty input
> - Error message for invalid format

**Status:** Task re-queued for revision. Agent will continue with `--resume` for context continuity.

---
*Labels: `needs-revision`*
```

## Success Criteria

**Multi-Channel Consistency:**
- [ ] All three channels (Dashboard, CLI, GitHub) trigger same approval handler
- [ ] Feedback from any channel flows to agent inbox with session continuity
- [ ] Activity from any channel visible in dashboard event queue
- [ ] OTEL spans include `feedback_source` attribute identifying channel

**GitHub Channel:**
- [ ] Human adds comment + label → Comment extracted as feedback
- [ ] Bot comments filtered out (agent comments not treated as feedback)
- [ ] `github_feedback` event type appears in event queue

**Dashboard Channel:**
- [ ] Event queue filters work correctly (test each filter type)
- [ ] Reject action prompts for feedback text
- [ ] Rejection posts comment to linked GitHub issue (bidirectional)

**CLI Channel (already works - verify integration):**
- [ ] `ailang coordinator reject --feedback "..."` stores feedback event
- [ ] CLI actions appear in dashboard event queue

**UI Elements:**
- [ ] Iteration counter shown on pending approvals
- [ ] Final iteration shows warning before permanent rejection
- [ ] Harvested GitHub feedback displayed: "Reviewer says: ..."

**Telemetry:**
- [ ] OTEL spans created for dashboard approve/reject actions
- [ ] OTEL spans include `feedback_source` attribute (github/dashboard)
- [ ] Spans visible in ExecHierarchy with feedback attributes

**General:**
- [ ] All tests passing
- [ ] Documentation updated with screenshots showing GitHub ↔ Dashboard flow

## Testing Strategy

**Unit tests:**
- `MessageQueue.tsx`: Filter logic isolates events by type
- `ApprovalPanel.tsx`: Iteration badge renders correctly
- `github_comments.go`: Bot filtering excludes agent comments
- `github_comments.go`: Human comments extracted correctly
- `handlers_coordinator.go`: New endpoint stores feedback, updates task

**Integration tests (Multi-Channel):**

*GitHub Channel:*
- Human adds comment on GitHub issue
- Human adds `needs-revision` label
- ApprovalWatcher harvests comment as feedback
- Dashboard shows `github_feedback` event
- Agent receives message with feedback content

*Dashboard Channel:*
- Create task → complete → reject from dashboard UI → verify:
  - Feedback in task_events table
  - Message in agent inbox
  - Comment on GitHub issue (bidirectional)
  - Span in Cloud Trace with `feedback_source: dashboard`
  - Task re-queued with iteration+1

*CLI Channel (verify existing):*
- `ailang coordinator reject --feedback "test"` → verify:
  - Event appears in dashboard queue
  - Span has `feedback_source: cli`

*Cross-Channel Visibility:*
- Reject via CLI → Dashboard shows event
- Reject via GitHub → Dashboard shows event
- Reject via Dashboard → GitHub shows comment
- All channels show same iteration count

- Verify filter: Send mixed events → filter → correct subset shown

**Manual testing:**
1. Start coordinator with test task
2. Complete task to trigger approval
3. View in dashboard ApprovalPanel
4. Verify iteration badge shows "1/3"
5. Reject with feedback via dashboard
6. Verify GitHub comment posted
7. Verify task re-queued
8. View span in ExecHierarchy

**Test data creation:**
```bash
# Create test task
ailang messages send coordinator "Fix the test bug in parser.go" \
  --title "Test: Parser Bug" --from "test-user" --type bug

# Wait for task to complete and show in pending
ailang coordinator pending

# Or create synthetic approval request
sqlite3 ~/.ailang/state/coordinator.db <<EOF
INSERT INTO approval_requests (id, task_id, type, description, status, created_at)
VALUES ('apr-test-001', 'task-test-001', 'merge', 'Test approval', 'pending', datetime('now'));
EOF
```

## Non-Goals

**Not in this feature:**
- Inline diff comments (inline code review) - Too complex, future enhancement
- Multi-reviewer workflow (multiple humans must approve) - Single approver sufficient for v1
- Batch approval with individual feedback - All-or-nothing batch only
- Mobile-responsive approval UI - Desktop-first
- Automatic re-trigger on GitHub label removal - Only manual or CLI trigger

## Timeline

**Day 1** (8 hours):
- Phase 1: Fix event queue filtering
- Phase 2 start: GitHub comment harvesting (IssueComment struct, API)

**Day 2** (8 hours):
- Phase 2 complete: Comment harvesting, bot filtering, watcher integration
- Phase 3 start: Dashboard approval UI

**Day 3** (8 hours):
- Phase 3 complete: UI integration, iteration badge, GH feedback display
- Phase 4: Bidirectional GitHub comments

**Day 4** (8 hours):
- Phase 5: Approval telemetry spans
- Phase 6 start: Test data, documentation

**Day 5** (8 hours):
- Phase 6 complete: Testing with real GitHub issues
- End-to-end verification (GitHub → Dashboard → Agent)
- PR creation

**Total: ~40 hours across 5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Event queue filter may be WebSocket issue not UI | Medium | Debug both paths, add logging |
| GitHub API rate limits on comment fetching | Medium | Cache comments, respect rate limits, exponential backoff |
| Bot comment filtering may miss edge cases | Low | Start with conservative patterns, iterate based on false positives |
| Comment timing race (comment before label) | Medium | Fetch comments since last poll minus 60s buffer |
| Span linkage may be lost across dashboard/CLI | Medium | Use consistent trace context propagation |
| Iteration counter may drift if task modified externally | Low | Always fetch fresh task data before display |

## Related Documents

**Implemented (foundation):**
- [design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md](design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md) - Feedback loop implementation
- [design_docs/implemented/v0_6_2/m-coord-approvalwatcher-observability.md](design_docs/implemented/v0_6_2/m-coord-approvalwatcher-observability.md) - Approval watcher
- [design_docs/implemented/v0_6_2/m-coord-stable.md](design_docs/implemented/v0_6_2/m-coord-stable.md) - Coordinator daemon

**Planned (related):**
- [design_docs/planned/v0_6_3/m-coord-ui-approvals.md](design_docs/planned/v0_6_3/m-coord-ui-approvals.md) - Original UI approvals plan (superseded by this doc)
- [design_docs/planned/v0_6_4/m-control-plane-interactive-filtering.md](design_docs/planned/v0_6_4/m-control-plane-interactive-filtering.md) - Event filtering improvements

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [docs/docs/guides/coordinator.md](docs/docs/guides/coordinator.md) - Coordinator guide
- [docs/docs/guides/collaboration-hub.md](docs/docs/guides/collaboration-hub.md) - Dashboard guide
- `internal/coordinator/human_interaction.go` - Feedback loop implementation
- `cmd/ailang/coordinator_actions.go` - CLI approve/reject implementation
- `internal/server/handlers_coordinator.go` - Server API endpoints

## Current Implementation Analysis

### CLI/API Working Features (Verified)

| Feature | Implementation | Status |
|---------|----------------|--------|
| `coordinatorApprove()` | Auto-commits, merges worktree, emits span | ✅ Complete |
| `coordinatorRejectWithFeedback()` | Stores feedback, increments iteration, re-triggers | ✅ Complete |
| `HumanFeedback` struct | Captures task_id, iteration, feedback text | ✅ Complete |
| `StoreFeedbackEvent()` | Stores as task event for audit | ✅ Complete |
| `PrepareTaskForRetrigger()` | Appends feedback to task content | ✅ Complete |
| `CanRetrigger()` | Checks iteration < MaxIterations (3) | ✅ Complete |
| `StoreIterationStartEvent()` | Marks iteration boundary | ✅ Complete |
| Message to agent inbox | Sends on rejection | ✅ Complete |
| Interactive `pending` | [a]pprove, [r]eject, [d]iff, [c]hat | ✅ Complete |

### Dashboard Missing Features (To Be Implemented)

| Feature | Gap | Solution |
|---------|-----|----------|
| Event filters | Don't apply correctly | Debug filter logic in `MessageQueue.tsx` |
| Feedback on reject | No text field | Add modal with required feedback |
| Iteration display | Not shown | Add `IterationBadge` component |
| GitHub comments | Not posted on reject | Add `PostFeedback()` to GitHubPoster |
| Approval spans | Not emitted from UI | Add span to `handleCoordinatorApproval()` |

### GitHub Integration Status

| Feature | Current | Needed |
|---------|---------|--------|
| Status labels | ✅ Added on stage completion | - |
| Completion comments | ✅ Posted via templates | - |
| Feedback comments | ❌ Not implemented | Add template + PostFeedback() |
| Comment replies | ❌ Not implemented | Out of scope |
| Threaded conversations | ❌ Not supported | Out of scope |

## Future Work

- Inline diff comments for detailed code review
- Multi-reviewer approval workflow
- Collaborative feedback (multiple humans comment before decision)
- Approval SLA metrics (time-to-approval, approval rate)
- Cost limits with automatic rejection threshold
- Mobile-responsive approval interface
- Slack/Discord notifications for pending approvals

---

**Document created**: 2026-01-14
**Last updated**: 2026-01-14
