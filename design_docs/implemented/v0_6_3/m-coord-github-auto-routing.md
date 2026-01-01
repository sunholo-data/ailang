# M-COORD-GITHUB-AUTO-ROUTING: GitHub-Driven Autonomous Workflow

| Field | Value |
|-------|-------|
| Status | **PARTIALLY IMPLEMENTED** |
| Target | v0.6.3 |
| Priority | P0 (High) |
| Estimated | 3-4 days |
| Dependencies | M-COORD-UI-APPROVALS (in progress), M-COORD-HUMAN-LOOP (complete) |
| Created | 2025-12-31 |
| Last Updated | 2026-01-01 |

## Implementation Status (2026-01-01)

### ✅ COMPLETED (Infrastructure Layer)

| Component | Status | Files |
|-----------|--------|-------|
| GitHub issue import with routing | ✅ Done | `internal/messages/github_import.go` |
| Message → Task with github_issue field | ✅ Done | `internal/coordinator/daemon_tasks.go:218` |
| GitHubPoster (comments, labels, close) | ✅ Done | `internal/coordinator/github_poster.go` |
| Comment templates (working, design, sprint, merge) | ✅ Done | `internal/coordinator/templates.go` |
| ApprovalWatcher (polls for labels) | ✅ Done | `internal/coordinator/approval_watcher.go` |
| TaskChain (stage transitions) | ✅ Done | `internal/coordinator/task_chain.go` |
| Database schema (github_issue, stage columns) | ✅ Done | `internal/coordinator/store_sqlite.go` |
| "Working" comment posted on task start | ✅ Done | `TaskChain.StartTask()` |
| GitHub labels created on repo | ✅ Done | All 10 labels exist |

### ❌ NOT IMPLEMENTED (Critical Gaps)

| Gap | Description | Impact |
|-----|-------------|--------|
| **Skill routing by stage** | Daemon doesn't route to `design-doc-creator`, `sprint-planner`, `sprint-executor` based on task stage | Tasks run generic prompts, not skill-specific workflows |
| **Stage completion detection** | Daemon doesn't detect when Claude Code output contains a design doc / sprint plan / implementation | TaskChain callbacks (`OnDesignDocComplete`, etc.) are never called |
| **Design doc summary extraction** | No code to parse Claude Code output and extract design doc path/summary | GitHub comment has no design doc content |
| **Sprint plan summary extraction** | No code to parse Claude Code output and extract sprint plan | GitHub comment has no sprint plan content |
| **Diff summary generation** | No code to generate git diff summary from worktree | GitHub comment has no diff content |
| **Actual merge execution** | `OnMergeApproved` doesn't actually merge worktree to dev | Worktree sits orphaned |
| **PR creation** | No code to create GitHub PR from worktree | Design says "no PRs" but users expect them |

### 🔄 PARTIALLY WORKING

| Component | What Works | What's Missing |
|-----------|------------|----------------|
| ApprovalWatcher | Detects `design-approved` label | Doesn't trigger `sprint-planner` skill |
| ApprovalWatcher | Detects `sprint-approved` label | Doesn't trigger `sprint-executor` skill |
| ApprovalWatcher | Detects `merge-approved` label | Doesn't execute actual merge |
| TaskChain.OnDesignApproved | Posts "Working on Sprint Planning" comment | Doesn't invoke sprint-planner |
| TaskChain.OnSprintApproved | Posts "Working on Implementation" comment | Doesn't invoke sprint-executor |

### Root Cause Analysis

**The fundamental gap:** The daemon's task execution flow doesn't integrate with the skill system.

```
CURRENT FLOW (broken):
1. Issue → Message → Task created
2. TaskChain.StartTask() posts "Working" comment ✅
3. Daemon calls executeTask() with generic prompt
4. Claude Code runs, produces some output
5. Task marked complete/pending_approval
6. ❌ NO CALLBACK to TaskChain
7. ❌ NO skill routing based on stage
8. ❌ NO summary posted to GitHub

INTENDED FLOW (not implemented):
1. Issue → Message → Task created
2. TaskChain.StartTask() posts "Working" comment ✅
3. Daemon detects stage="design", routes to design-doc-creator skill
4. Claude Code creates design doc
5. Daemon parses output, extracts design doc path
6. TaskChain.OnDesignDocComplete() posts summary to GitHub
7. Task enters pending_approval, waits for design-approved label
8. ApprovalWatcher detects design-approved
9. TaskChain.OnDesignApproved() triggers sprint-planner
10. ... and so on through the pipeline
```

### Next Steps (v0.6.3)

**Phase 1: Skill Routing (~4 hours)**
- [ ] Add stage-to-skill mapping in daemon
- [ ] Route `design` stage to `design-doc-creator` skill
- [ ] Route `sprint` stage to `sprint-planner` skill
- [ ] Route `implementation` stage to `sprint-executor` skill

**Phase 2: Output Parsing (~4 hours)**
- [ ] Parse Claude Code output for design doc paths
- [ ] Parse Claude Code output for sprint plan paths
- [ ] Extract summary from skill output (first 500 chars or structured JSON)

**Phase 3: Callback Integration (~4 hours)**
- [ ] Call `TaskChain.OnDesignDocComplete()` after design-doc-creator
- [ ] Call `TaskChain.OnSprintPlanComplete()` after sprint-planner
- [ ] Call `TaskChain.OnImplementationComplete()` after sprint-executor

**Phase 4: Merge Execution (~2 hours)**
- [ ] Implement actual git merge in `OnMergeApproved`
- [ ] Handle merge conflicts (post to GitHub, pause)
- [ ] Clean up worktree after successful merge

**Phase 5: E2E Test (~2 hours)**
- [ ] Create test issue with `coordinator:feature` label
- [ ] Verify full pipeline: design → sprint → implementation → merge
- [ ] Verify all GitHub comments and labels appear correctly

## Problem Statement

**The current coordinator workflow requires manual intervention at multiple points:**

1. GitHub issues must be manually triaged and delegated to the coordinator
2. Design docs are created but humans must manually review via dashboard or CLI
3. Approval happens via dashboard/CLI, not where the issue originated (GitHub)
4. No feedback loop to the original GitHub issue thread
5. Issue closure is manual after work is complete

**What users want:**

A fully autonomous pipeline where:
- GitHub issues with specific labels are automatically picked up
- Agents post updates back to the GitHub issue thread
- Approval/rejection happens via GitHub labels
- Issues auto-close when work is merged

## Goals

**Primary Goal:** Enable a GitHub-centric autonomous development workflow where GitHub Issues are the primary interface for task management, with agents posting progress updates and humans approving via GitHub.

**Success Metrics:**
1. Labeled GitHub issues auto-route to coordinator without manual intervention
2. Design doc summaries posted as GitHub issue comments
3. Human approval via `approved` label triggers sprint execution
4. Merged work auto-closes the originating GitHub issue
5. Full audit trail visible in GitHub issue thread

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Infrastructure, no language semantics impact |
| A2: Replayability | +1 | Full audit trail in GitHub issue history |
| A3: Effect Legibility | +1 | All agent actions visible in issue comments |
| A4: Explicit Authority | +1 | Human approval required via explicit label |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | Sequential pipeline, no concurrency concerns |
| A7: Machines First | +1 | Structured JSON payloads, GitHub API |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Can post cost summary to issue |
| A10: Composability | +1 | Integrates with existing skills pipeline |
| A11: Structured Failure | +1 | Failures reported as issue comments |
| A12: System Boundary | +1 | Explicit GitHub↔Agent boundary via comments |

**Net Score: +8** - Move forward

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GITHUB-DRIVEN WORKFLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

 ┌──────────────────────────────────────────────────────────────────────────┐
 │  1. ISSUE CREATION                                                        │
 │     User creates issue with label: `coordinator:bug` or `coordinator:feature`│
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  2. AUTO-IMPORT                                                           │
 │     `ailang messages import-github` (runs on session start)               │
 │     - Detects `coordinator:*` labels                                      │
 │     - Routes to `coordinator` inbox instead of `user` inbox               │
 │     - Stores GitHub issue number for bidirectional sync                   │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  3. COORDINATOR PICKUP                                                    │
 │     Daemon polls coordinator inbox, creates task                          │
 │     - Spawns worktree                                                     │
 │     - Routes to design-doc-creator skill                                  │
 │     - Posts "🤖 Working on design doc..." comment to GitHub               │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  4. DESIGN DOC CREATION                                                   │
 │     Claude Code runs design-doc-creator skill                             │
 │     - Creates design_docs/planned/<issue>.md                              │
 │     - Posts design doc SUMMARY to GitHub issue (not full doc)             │
 │     - Requests approval by mentioning "Ready for review"                  │
 │     - Adds label: `needs-design-approval`                                 │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  👤 APPROVAL GATE 1: Human reviews on GitHub                             │
 │     - Reads design doc summary in issue comment                           │
 │     - Can view full doc via link                                          │
 │     - Approves: Adds label `design-approved`                              │
 │     - Rejects: Adds comment with feedback, adds `needs-revision`          │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │ (on design-approved label)
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  5. SPRINT PLANNING                                                       │
 │     Coordinator detects approval, routes to sprint-planner                │
 │     - Creates sprint plan with milestones                                 │
 │     - Creates sprint JSON for progress tracking                           │
 │     - Posts sprint summary to GitHub issue                                │
 │     - Adds label: `needs-sprint-approval`                                 │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  👤 APPROVAL GATE 2: Human reviews sprint plan on GitHub                 │
 │     - Reads sprint summary in issue comment                               │
 │     - Approves: Adds label `sprint-approved`                              │
 │     - Rejects: Feedback in comment                                        │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │ (on sprint-approved label)
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  6. SPRINT EXECUTION                                                      │
 │     Coordinator routes to sprint-executor                                 │
 │     - Implements each milestone with TDD                                  │
 │     - Posts progress updates to GitHub issue                              │
 │     - On completion: Posts diff summary, adds `needs-merge-approval`      │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  👤 APPROVAL GATE 3: Human reviews implementation                        │
 │     - Views diff summary in issue comment                                 │
 │     - Can checkout worktree locally for detailed review                   │
 │     - Approves: Adds label `merge-approved`                               │
 │     - Rejects: Feedback in comment                                        │
 └────────────────────────────────────┬─────────────────────────────────────┘
                                      │ (on merge-approved label)
                                      ▼
 ┌──────────────────────────────────────────────────────────────────────────┐
 │  7. MERGE & CLOSE                                                         │
 │     Coordinator merges worktree to dev                                    │
 │     - Posts merge confirmation to GitHub issue                            │
 │     - Auto-closes issue with summary comment                              │
 │     - Cleans up worktree                                                  │
 └──────────────────────────────────────────────────────────────────────────┘
```

### GitHub Labels

**Routing labels** (added by humans to trigger automation):
- `coordinator:bug` - Route to coordinator for bug fix
- `coordinator:feature` - Route to coordinator for feature
- `coordinator:docs` - Route to coordinator for documentation
- `coordinator:research` - Route to coordinator for research

**Status labels** (added by agents):
- `needs-design-approval` - Waiting for human to approve design doc
- `needs-sprint-approval` - Waiting for human to approve sprint plan
- `needs-merge-approval` - Waiting for human to approve merge

**Approval labels** (added by humans):
- `design-approved` - Human approved the design doc
- `sprint-approved` - Human approved the sprint plan
- `merge-approved` - Human approved the implementation
- `needs-revision` - Human rejected, needs agent to revise

### Implementation Plan

**Phase 1: Label-Based Routing** (~4 hours)

Extend `ailang messages import-github` to route based on labels:

```go
// internal/messages/github_import.go

func (i *GitHubImporter) routeByLabel(issue *github.Issue) string {
    for _, label := range issue.Labels {
        switch label.GetName() {
        case "coordinator:bug", "coordinator:feature",
             "coordinator:docs", "coordinator:research":
            return "coordinator"
        }
    }
    return "user" // Default inbox
}
```

**Files:**
- `internal/messages/github_import.go` - Add label routing (~50 LOC)
- `internal/messages/messages.go` - Store github_issue_number field (~10 LOC)

**Phase 2: GitHub Comment Posting** (~6 hours)

Add ability for coordinator to post comments to GitHub issues:

```go
// internal/coordinator/github_poster.go

type GitHubPoster struct {
    client *github.Client
    repo   string
}

func (p *GitHubPoster) PostComment(issueNum int, body string) error {
    _, _, err := p.client.Issues.CreateComment(ctx, owner, repo, issueNum, &github.IssueComment{
        Body: &body,
    })
    return err
}

func (p *GitHubPoster) AddLabel(issueNum int, label string) error {
    _, _, err := p.client.Issues.AddLabelsToIssue(ctx, owner, repo, issueNum, []string{label})
    return err
}

func (p *GitHubPoster) RemoveLabel(issueNum int, label string) error {
    _, err := p.client.Issues.RemoveLabelForIssue(ctx, owner, repo, issueNum, label)
    return err
}

func (p *GitHubPoster) CloseIssue(issueNum int) error {
    closed := "closed"
    _, _, err := p.client.Issues.Edit(ctx, owner, repo, issueNum, &github.IssueRequest{
        State: &closed,
    })
    return err
}
```

**Files:**
- `internal/coordinator/github_poster.go` - GitHub API wrapper (~150 LOC)
- `internal/coordinator/daemon.go` - Integrate poster into task lifecycle (~50 LOC)

**Phase 3: Approval Detection** (~4 hours)

Poll for approval labels and trigger next stage:

```go
// internal/coordinator/approval_watcher.go

type ApprovalWatcher struct {
    client     *github.Client
    store      Store
    poster     *GitHubPoster
    pollInterval time.Duration
}

func (w *ApprovalWatcher) WatchForApprovals(ctx context.Context) {
    ticker := time.NewTicker(w.pollInterval)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.checkPendingTasks()
        }
    }
}

func (w *ApprovalWatcher) checkPendingTasks() {
    tasks, _ := w.store.ListByStatus("pending_approval")
    for _, task := range tasks {
        if task.GithubIssue == 0 {
            continue // No linked GitHub issue
        }

        labels, _, _ := w.client.Issues.ListLabels(ctx, owner, repo, task.GithubIssue, nil)

        switch task.Stage {
        case "design":
            if hasLabel(labels, "design-approved") {
                w.triggerSprintPlanning(task)
            }
        case "sprint":
            if hasLabel(labels, "sprint-approved") {
                w.triggerSprintExecution(task)
            }
        case "implementation":
            if hasLabel(labels, "merge-approved") {
                w.triggerMerge(task)
            }
        }
    }
}
```

**Files:**
- `internal/coordinator/approval_watcher.go` - Label polling (~200 LOC)
- `internal/coordinator/store_sqlite.go` - Add github_issue, stage fields (~30 LOC)

**Phase 4: Task Chaining** (~4 hours)

Chain tasks: design-doc → sprint-planner → sprint-executor

```go
// internal/coordinator/task_chain.go

type TaskChain struct {
    store  Store
    poster *GitHubPoster
}

func (c *TaskChain) OnDesignDocComplete(task *Task) {
    // Post summary to GitHub
    summary := c.generateDesignSummary(task)
    c.poster.PostComment(task.GithubIssue, summary)
    c.poster.AddLabel(task.GithubIssue, "needs-design-approval")

    // Update task stage
    task.Stage = "design"
    task.Status = "pending_approval"
    c.store.Update(task)
}

func (c *TaskChain) OnSprintPlanComplete(task *Task) {
    summary := c.generateSprintSummary(task)
    c.poster.PostComment(task.GithubIssue, summary)
    c.poster.RemoveLabel(task.GithubIssue, "design-approved")
    c.poster.AddLabel(task.GithubIssue, "needs-sprint-approval")

    task.Stage = "sprint"
    task.Status = "pending_approval"
    c.store.Update(task)
}

func (c *TaskChain) OnImplementationComplete(task *Task) {
    summary := c.generateDiffSummary(task)
    c.poster.PostComment(task.GithubIssue, summary)
    c.poster.RemoveLabel(task.GithubIssue, "sprint-approved")
    c.poster.AddLabel(task.GithubIssue, "needs-merge-approval")

    task.Stage = "implementation"
    task.Status = "pending_approval"
    c.store.Update(task)
}

func (c *TaskChain) OnMergeComplete(task *Task) {
    comment := fmt.Sprintf("Changes merged to dev branch.\n\nThis issue was resolved by autonomous agents.\n\n**Summary:**\n%s", task.Summary)
    c.poster.PostComment(task.GithubIssue, comment)
    c.poster.CloseIssue(task.GithubIssue)

    task.Status = "completed"
    c.store.Update(task)
}
```

**Files:**
- `internal/coordinator/task_chain.go` - Task chaining logic (~250 LOC)
- `internal/coordinator/daemon.go` - Wire chain into task lifecycle (~30 LOC)

**Phase 5: GitHub Comment Templates** (~2 hours)

Create templates for GitHub comments:

**Files:**
- `internal/coordinator/templates/design_summary.md` - Design doc summary template
- `internal/coordinator/templates/sprint_summary.md` - Sprint plan summary template
- `internal/coordinator/templates/diff_summary.md` - Implementation diff summary
- `internal/coordinator/templates/completion.md` - Issue closure message

### Database Schema Changes

```sql
-- Add to tasks table
ALTER TABLE tasks ADD COLUMN github_issue INTEGER;
ALTER TABLE tasks ADD COLUMN stage TEXT; -- 'design', 'sprint', 'implementation'
```

### Configuration

Add to `~/.ailang/config.yaml`:

```yaml
github:
  # Existing
  expected_user: sunholo-voight-kampff
  default_repo: sunholo-data/ailang

  # New for auto-routing
  auto_routing:
    enabled: true
    poll_interval: 60s  # How often to check for approval labels

  # Labels
  labels:
    routing:
      - coordinator:bug
      - coordinator:feature
      - coordinator:docs
      - coordinator:research
    approval:
      design: design-approved
      sprint: sprint-approved
      merge: merge-approved
    pending:
      design: needs-design-approval
      sprint: needs-sprint-approval
      merge: needs-merge-approval
    revision: needs-revision
```

## Example Workflow

### 1. User Creates GitHub Issue

```markdown
**Title:** Fix null pointer in parser.go

**Labels:** `coordinator:bug`

**Body:**
Line 142 of parser.go throws a null pointer exception when parsing empty input.

Reproduction:
1. Run `ailang run empty.ail` with empty file
2. Observe panic
```

### 2. Auto-Import Routes to Coordinator

```
$ ailang messages import-github
Importing issues from sunholo-data/ailang...
  Issue #87: Fix null pointer in parser.go
    Labels: coordinator:bug
    Routed to: coordinator (auto-routed by label)
```

### 3. Coordinator Posts Progress

GitHub Issue #87 gets comment:

> **Agent Working**
>
> I've picked up this issue and am creating a design document.
>
> **Task ID:** `task-a1b2c3d4`
> **Agent:** Claude Code
> **Started:** 2025-12-31T15:30:00Z

### 4. Design Doc Summary Posted

GitHub Issue #87 gets comment:

> **Design Document Ready for Review**
>
> ## Summary
>
> **Problem:** Null pointer in `parseExpression()` when `curToken` is nil
>
> **Solution:** Add nil check before accessing token fields
>
> **Files to modify:**
> - `internal/parser/parser.go` (~10 LOC)
>
> **Tests to add:**
> - `TestParseEmptyInput` in parser_test.go
>
> [View Full Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_6_3/fix-parser-null-pointer.md)
>
> ---
> To approve: Add the `design-approved` label
> To request changes: Comment with feedback and add `needs-revision`

### 5. Human Approves (adds label)

Human adds `design-approved` label to Issue #87.

### 6. Sprint Plan Posted

GitHub Issue #87 gets comment:

> **Sprint Plan Ready for Review**
>
> ## Milestones
>
> | ID | Task | Est. LOC | Est. Time |
> |----|------|----------|-----------|
> | M1 | Add nil check to parseExpression | 5 | 15min |
> | M2 | Add TestParseEmptyInput test | 20 | 30min |
> | M3 | Verify all tests pass | - | 5min |
>
> **Total:** ~25 LOC, ~50 minutes
>
> ---
> To approve: Add the `sprint-approved` label

### 7. Implementation Complete

GitHub Issue #87 gets comment:

> **Implementation Complete - Ready for Merge**
>
> ## Changes
>
> ```diff
> --- a/internal/parser/parser.go
> +++ b/internal/parser/parser.go
> @@ -140,6 +140,9 @@ func (p *Parser) parseExpression() ast.Expression {
> +    if p.curToken == nil {
> +        return nil
> +    }
>      // existing code...
> ```
>
> ## Test Results
> - All 847 tests passing
> - New test: `TestParseEmptyInput`
>
> **Cost:** $0.12 (Claude Sonnet 4.5)
>
> ---
> To merge: Add the `merge-approved` label
> To request changes: Comment with feedback

### 8. Auto-Close on Merge

GitHub Issue #87 gets comment and is closed:

> **Issue Resolved**
>
> Changes have been merged to the `dev` branch.
>
> **Summary:**
> - Added nil check to `parseExpression()` (5 LOC)
> - Added `TestParseEmptyInput` test (20 LOC)
> - All tests passing
>
> **Agent:** Claude Code
> **Total Cost:** $0.12
> **Duration:** 45 minutes
>
> This issue was automatically resolved by AILANG autonomous agents.

Issue #87 is now **Closed**.

## Success Criteria

- [x] Issues with `coordinator:*` labels auto-route to coordinator inbox ✅ **DONE**
- [x] Coordinator posts "working" comment when picking up task ✅ **DONE**
- [ ] Design doc summary posted as GitHub comment ❌ **NOT DONE** - callback never called
- [ ] `design-approved` label triggers sprint planning ❌ **PARTIAL** - label detected, skill not invoked
- [ ] Sprint summary posted as GitHub comment ❌ **NOT DONE** - callback never called
- [ ] `sprint-approved` label triggers sprint execution ❌ **PARTIAL** - label detected, skill not invoked
- [ ] Diff summary posted on implementation completion ❌ **NOT DONE** - callback never called
- [ ] `merge-approved` label triggers merge to dev ❌ **NOT DONE** - no actual merge
- [ ] Issue auto-closes on successful merge ❌ **NOT DONE** - depends on merge
- [x] Rejection via `needs-revision` label pauses workflow ✅ **DONE**
- [ ] Full audit trail visible in GitHub issue thread ❌ **PARTIAL** - only "working" comment visible
- [x] All tests passing ✅ **DONE**
- [ ] Documentation updated ❌ **NOT DONE**

**Summary: 4/13 complete, 9 remaining**

## Testing Strategy

**Unit tests:**
- Label routing logic in import-github
- Comment formatting templates
- Approval detection logic
- Task chaining state machine

**Integration tests:**
- Full workflow with mock GitHub API
- Approval label → next stage transition
- Error handling (API failures, merge conflicts)

**Manual testing:**
- Create real GitHub issue with label
- Verify full workflow end-to-end
- Test rejection and revision flow

## Non-Goals

**Not in this feature:**
- GitHub Pull Requests (PRs) - We use local git merge, not PRs
- Multiple repository support - Single repo for now
- Webhook-based triggers - Polling is simpler and sufficient
- Branch protection integration - Direct merge to dev

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| GitHub API rate limits | High | Cache labels, batch API calls, respect X-RateLimit headers |
| Stale label state | Medium | Always re-fetch labels before acting |
| Merge conflicts | Medium | Detect early, post to GitHub for human resolution |
| Missing `gh` auth | High | Validate auth on daemon start, fail fast |
| Comment spam | Low | Consolidate updates, edit existing comments |

## Related Documents

**Implemented:**
- [m-coord-stable.md](../../implemented/v0_6_2/m-coord-stable.md) - Coordinator daemon
- [m-coordinator-feedback-loop.md](../../implemented/v0_6_2/m-coordinator-feedback-loop.md) - Event streaming

**Planned:**
- [m-coord-ui-approvals.md](m-coord-ui-approvals.md) - Dashboard approval UI (alternative)
- [m-coord-human-loop.md](../v0_6_2/m-coord-human-loop.md) - Human-in-the-loop foundation

## Future Work

- **Webhook triggers** - Replace polling with webhooks for instant response
- **PR-based flow** - Option to create GitHub PRs instead of direct merge
- **Multi-repo support** - Route issues from multiple repos
- **Slack/Discord integration** - Post notifications to chat
- **Cost budgets** - Auto-reject tasks exceeding cost threshold

---

**Document created:** 2025-12-31
**Last updated:** 2025-12-31
