# M-COORD-AUTO-REVISION: Automatic Agent Revision on GitHub Feedback

| Field | Value |
|-------|-------|
| Status | Planned |
| Target | v0.6.3 |
| Priority | P1 (High) |
| Estimated | 3.5 days |
| Dependencies | M-COORD-GITHUB-AUTO-ROUTING (implemented), ApprovalWatcher (implemented) |
| Created | 2026-01-02 |

## Problem Statement

When a human reviewer adds the `needs-revision` label to a GitHub issue along with feedback comments, the pipeline **pauses but does nothing**. The agent that created the artifact (design doc, sprint plan, or implementation) is not notified and cannot automatically revise its work.

**Current Behavior:**
1. Agent creates design doc, posts to GitHub
2. Coordinator adds `needs-design-approval` label
3. Human reviews, adds comment: "Missing error handling section"
4. Human adds `needs-revision` label
5. **Pipeline pauses indefinitely**
6. Human must manually fix OR manually send new message to agent

**Desired Behavior:**
1-4. Same as above
5. Coordinator detects `needs-revision` + extracts feedback comment
6. **Coordinator automatically resumes agent with feedback context**
7. Agent revises artifact based on feedback
8. Agent resubmits for approval

### Impact

- Revision workflow is broken - requires manual intervention
- Humans expect agents to respond to feedback like a human collaborator would
- Current system posts "I'll resume work once the revised artifact is approved" but doesn't actually do anything
- Breaks the autonomous development loop promise

## Goals

**Primary Goal:** When `needs-revision` is detected, automatically resume the agent with GitHub feedback to revise the artifact.

**Success Metrics:**
1. Agent automatically restarts within 60s of `needs-revision` label being added
2. Agent receives full context: original task + previous work + revision feedback
3. Revised artifact posted to GitHub for re-approval
4. Zero manual intervention required for revision cycle
5. Session continuity preserved (agent remembers context if configured)

## Current State Analysis

**Session ID Tracking (already implemented):**
- `TaskRecord.SessionID` - Stored in SQLite after execution
- `ExecuteResult.SessionID` - Returned by Claude/Gemini executors
- `AgentConfig.SessionContinuity` - Config option exists

**Gap - Session Resume NOT Implemented:**
- `executor.Task` has NO `ResumeSessionID` field
- Claude executor generates NEW session ID every time (`uuid.New()`)
- `--resume` flag is never passed to Claude Code CLI
- Config option `session_continuity: true` is **ignored**

This feature must implement the actual resume mechanism.

## Solution Design

### Overview

```
GitHub Issue                          Coordinator                        Agent
     │                                      │                              │
     │  adds: needs-revision               │                              │
     │  adds: comment with feedback        │                              │
     ├─────────────────────────────────────▶│                              │
     │                                      │ ApprovalWatcher detects      │
     │                                      │ label + fetches comments     │
     │                                      │                              │
     │                                      │ Creates revision task        │
     │                                      │ with feedback context        │
     │                                      ├─────────────────────────────▶│
     │                                      │                              │ Revises artifact
     │                                      │                              │ based on feedback
     │                                      │◀─────────────────────────────┤
     │                                      │ Posts revised artifact       │
     │  comment: Revised design doc        │                              │
     │  label: needs-design-approval       │                              │
     │◀─────────────────────────────────────┤                              │
```

### Architecture

**Component 1: Feedback Extractor**

When `needs-revision` is detected, extract the most recent feedback:

```go
type RevisionFeedback struct {
    IssueNumber    int
    TaskID         string
    Stage          TaskStage           // design, sprint, implementation
    Comments       []GitHubComment     // Recent comments since last agent post
    OriginalPrompt string              // Original task content
    ArtifactPath   string              // Path to artifact being revised
    SessionID      string              // For session continuity
}

// FetchRevisionFeedback retrieves feedback context from GitHub
func (p *GitHubPoster) FetchRevisionFeedback(issueNumber int, taskID string) (*RevisionFeedback, error)
```

**Component 2: Revision Task Creator**

Create a new task (or resume session) with revision context:

```go
// CreateRevisionTask creates a task to revise an artifact
func (d *Daemon) CreateRevisionTask(ctx context.Context, feedback *RevisionFeedback) (*TaskRecord, error) {
    // Build revision prompt
    prompt := buildRevisionPrompt(feedback)

    // Determine if we should resume or start fresh
    if feedback.SessionID != "" && agent.SessionContinuity {
        return d.createResumeTask(feedback, prompt)
    }
    return d.createNewTask(feedback, prompt)
}
```

**Component 3: Revision Prompt Builder**

Build a context-rich prompt for the agent:

```go
func buildRevisionPrompt(f *RevisionFeedback) string {
    return fmt.Sprintf(`## Revision Requested

**Original Task:** %s

**Current Artifact:** %s

**Reviewer Feedback:**
%s

---

Please revise the artifact based on the feedback above. When done:
1. Update the artifact file
2. Post a comment explaining the changes
3. The pipeline will automatically request re-approval

IMPORTANT: Focus only on addressing the feedback. Do not expand scope.
`, f.OriginalPrompt, f.ArtifactPath, formatComments(f.Comments))
}
```

### Implementation Plan

**Phase 0: Session Resume Support** (~3 hours) - PREREQUISITE

The session resume mechanism must be implemented first:

- [ ] Add `ResumeSessionID` field to `executor.Task` struct
- [ ] Update Claude executor to use `--resume` when `ResumeSessionID` is set:
  ```go
  if task.ResumeSessionID != "" {
      args = append(args, "--resume", task.ResumeSessionID)
  } else {
      sessionID = uuid.New().String()
      args = append(args, "--session-id", sessionID)
  }
  ```
- [ ] Update Gemini executor to use `--conversation-id` when resuming
- [ ] Add test for session resume capability
- [ ] Verify session resume works with Claude Code CLI

**Phase 1: Feedback Extraction** (~4 hours)

- [ ] Add `GetRecentComments(issueNumber int, since time.Time) ([]GitHubComment, error)` to GitHubPoster
- [ ] Create `RevisionFeedback` struct with all context needed
- [ ] Add `FetchRevisionFeedback()` method that collects:
  - Recent comments (since last coordinator comment)
  - Original task content
  - Current artifact path
  - Session ID if exists
- [ ] Unit tests for comment extraction

**Phase 2: Revision Handler** (~4 hours)

- [ ] Modify `OnNeedsRevision()` in TaskChain to:
  1. Fetch revision feedback
  2. Create revision task
  3. Queue for execution
- [ ] Add `revision_of` field to TaskRecord (links to original task)
- [ ] Track revision count to prevent infinite loops (max 3 revisions)
- [ ] Update task stage to indicate revision in progress

**Phase 3: Agent Invocation** (~4 hours)

- [ ] Build revision prompt with full context
- [ ] Support session resume (`--resume` flag for Claude Code)
- [ ] Support fresh start with context injection
- [ ] Ensure worktree is reused (same branch) for revision
- [ ] Post "Starting revision..." comment to GitHub

**Phase 4: Re-approval Flow** (~4 hours)

- [ ] After revision completes, reset approval state
- [ ] Remove `needs-revision` label
- [ ] Add `needs-X-approval` label (appropriate for stage)
- [ ] Post revised artifact content to GitHub
- [ ] Update templates to show revision number

### Files to Modify/Create

**New files:**
- `internal/coordinator/revision.go` - RevisionFeedback, buildRevisionPrompt (~150 LOC)

**Modified files (coordinator):**
- `internal/coordinator/github_poster.go` - Add GetRecentComments (~40 LOC)
- `internal/coordinator/task_chain.go` - Enhance OnNeedsRevision (~60 LOC)
- `internal/coordinator/daemon.go` - Add CreateRevisionTask (~50 LOC)
- `internal/coordinator/store_sqlite.go` - Add revision_of, revision_count fields (~20 LOC)
- `internal/coordinator/templates.go` - Revision templates (~30 LOC)

**Modified files (executor - Phase 0):**
- `internal/executor/executor.go` - Add ResumeSessionID to Task struct (~5 LOC)
- `internal/executor/claude/claude.go` - Use --resume when ResumeSessionID set (~15 LOC)
- `internal/executor/gemini/gemini.go` - Use --conversation-id when resuming (~15 LOC)

**Total: ~385 LOC**

## Examples

### Example 1: Design Doc Revision Flow

**Step 1: Human requests revision on GitHub**

```markdown
Comment by @MarkEdmondson1234:
> The design doc is missing:
> 1. Error handling strategy
> 2. Rollback plan if migration fails
> 3. Performance impact analysis
>
> Please add these sections before approval.
```

*Human adds `needs-revision` label*

**Step 2: Coordinator detects and responds**

```markdown
Comment by @sunholo-voight-kampff:
## Revision In Progress

I'm revising the design document based on your feedback:
- Adding error handling strategy
- Adding rollback plan
- Adding performance impact analysis

I'll post the updated document when ready.

Revision: 1 of 3 max
```

**Step 3: Agent revises and resubmits**

```markdown
Comment by @sunholo-voight-kampff:
## Design Document Revised

I've updated the design document to address your feedback:

**Changes made:**
- Added "Error Handling" section with retry logic and circuit breakers
- Added "Rollback Strategy" with step-by-step recovery plan
- Added "Performance Impact" with benchmark estimates

<details>
<summary>View Updated Design Document</summary>

[Full markdown content here]

</details>

---

**Next Steps:**
1. Review the updated design document
2. Add `design-approved` to proceed to sprint planning
3. Add `needs-revision` if more changes needed (2 of 3 max)
```

*Coordinator removes `needs-revision`, adds `needs-design-approval`*

### Example 2: Session Continuity

If `session_continuity: true` is configured:

```bash
# Agent is resumed with previous context
claude --resume session-abc123 \
  -p "Revision requested. Feedback: ..."
```

The agent retains full conversation history, including:
- Original task understanding
- Files it read
- Design decisions made
- Previous attempts

**Note:** Claude Code's `--resume` flag continues an existing session by ID.
The session must still exist on the local machine (sessions are stored in
`~/.claude/projects/`). If the session was cleaned up, the revision falls
back to a fresh start with context injection.

### Example 3: Revision Limit

After 3 revision cycles:

```markdown
Comment by @sunholo-voight-kampff:
## Revision Limit Reached

This artifact has been revised 3 times. To prevent infinite loops,
automatic revision is now disabled.

**Options:**
1. Manually edit the artifact and add `design-approved`
2. Close this issue and create a new one with clearer requirements
3. Run `/coordinator retry` to reset the revision count

Previous revisions:
- Revision 1: Added error handling
- Revision 2: Fixed rollback strategy
- Revision 3: Updated performance numbers
```

## Success Criteria

- [ ] `needs-revision` triggers automatic agent invocation within 60s
- [ ] Agent receives full context (original task + artifact + feedback)
- [ ] Session continuity works when configured
- [ ] Revised artifact posted to GitHub
- [ ] Re-approval flow works correctly
- [ ] Revision count tracked and limited to 3
- [ ] Templates show revision status
- [ ] All existing tests still pass
- [ ] New unit tests for revision flow
- [ ] Integration test for full revision cycle

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics |
| A2: Replayability | +1 | Revision history creates clear audit trail |
| A3: Effect Legibility | +1 | Revision actions visible in GitHub comments |
| A4: Explicit Authority | +1 | Human must explicitly request revision via label |
| A7: Machines First | +1 | Automated revision loop, no manual intervention |
| A11: Structured Failure | +1 | Revision limit prevents runaway loops |

**Net Score: +5** - Move forward

## Testing Strategy

**Unit tests:**
- `TestFetchRevisionFeedback` - Extracts correct comments
- `TestBuildRevisionPrompt` - Generates proper prompt
- `TestRevisionLimit` - Stops after 3 revisions
- `TestSessionContinuity` - Resume flag passed correctly

**Integration tests:**
- Full cycle: create → reject → revise → approve
- Multiple revisions up to limit
- Session continuity preservation

**Manual testing:**
- Create GitHub issue, trigger full revision flow
- Verify agent responds appropriately to feedback
- Test revision limit behavior

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Infinite revision loop | High | Hard limit of 3 revisions per artifact |
| Agent ignores feedback | Medium | Clear prompt structure, explicit instructions |
| Session context too large | Medium | Summarize previous turns if needed |
| Comment extraction fails | Low | Fallback to generic "revision requested" |
| Rate limiting by GitHub | Low | Respect API limits, batch comment fetches |

## Non-Goals

- **Automatic approval after revision** - Human must always approve
- **Multiple concurrent revisions** - One revision at a time per issue
- **Cross-stage revision** - Can't revise design doc during sprint stage
- **Undo revision** - No rollback to pre-revision state

## Related Documents

**Implemented (build upon):**
- [m-coord-github-auto-routing.md](../../../implemented/v0_6_2/m-coord-github-auto-routing.md) - GitHub integration
- [m-coordinator-feedback-loop.md](../../../implemented/v0_6_2/m-coordinator-feedback-loop.md) - Streaming & approval

**Planned (check for overlap):**
- [m-coord-generic-workflows.md](../v0_6_3/m-coord-generic-workflows.md) - Generic stage configuration

## Additional Features: Message Routing Improvements

### Background

While investigating why GitHub issue #104 wasn't picked up by the coordinator, we discovered a timing issue:

1. Issue #104 was created WITHOUT the `coordinator:feature` label
2. Session start hook imported it to `user` inbox (default)
3. Label was added ~13 hours LATER
4. Message was already imported - label change had no effect

Two features address this gap:

### Feature A: Interactive Message Menu (DX Improvement)

The current UX requires copying long UUIDs like `87738035-3807-4b2f-935a-09ede9fb0c3a`. Add an interactive menu system:

```bash
# Interactive mode - shows numbered menu
ailang messages

# Output:
# ┌─────────────────────────────────────────────────────────────────┐
# │ AILANG Messages - user inbox (2 unread)                         │
# ├─────────────────────────────────────────────────────────────────┤
# │ [1] ● Consider importing linting issues from Sonar Cloud  #104  │
# │ [2]   Artifact Discovery Fix v2                            #106  │
# │ [3]   Ultrathink                                           #107  │
# └─────────────────────────────────────────────────────────────────┘
# │ Actions: [r]ead [f]orward [a]ck [d]elete [q]uit                 │
#
# Select message (1-3) or action: _
```

**Keyboard navigation:**
- `1-9` - Select message by number
- `r` - Read selected message
- `f` - Forward to another inbox (prompts for target)
- `a` - Acknowledge (mark as read)
- `A` - Acknowledge all
- `d` - Delete message
- `q` - Quit
- `j/k` or `↑/↓` - Navigate (if using TUI library)
- `/` - Filter/search messages

**Implementation options:**
1. **Simple numbered menu** - No dependencies, basic stdin reading (~100 LOC)
2. **promptui** - `github.com/manifoldco/promptui` - Nice prompts (~150 LOC)
3. **bubbletea** - `github.com/charmbracelet/bubbletea` - Full TUI (~300 LOC)

**Recommendation:** Start with simple numbered menu, upgrade to bubbletea if needed.

**Short IDs:**
Also support short ID prefixes (like git):
```bash
ailang messages read 87738     # Matches 87738035-3807-4b2f-935a-09ede9fb0c3a
ailang messages ack 877        # Works if unique prefix
```

**Estimated: ~4 hours** (simple menu + short IDs)

### Feature B: Message Forward CLI (uses interactive menu)

Allow manual forwarding of messages between inboxes:

```bash
# Forward a message to a different inbox
ailang messages forward MSG_ID --to design-doc-creator

# Forward with reason (logged)
ailang messages forward MSG_ID --to design-doc-creator --reason "Label added after import"

# Bulk forward by query
ailang messages forward --from user --to design-doc-creator --filter "github_issue IS NOT NULL"
```

**Implementation:**

```go
// cmd/ailang/messages.go
func runMessagesForward(args []string) {
    fs := flag.NewFlagSet("messages forward", flag.ExitOnError)
    toInbox := fs.String("to", "", "Target inbox (required)")
    reason := fs.String("reason", "", "Reason for forwarding (optional)")
    // ...
}
```

**Database update:**
```sql
UPDATE messages SET to_id = ? WHERE id = ?;
-- Optionally log the forward in an audit table
```

**Estimated: ~2 hours**

### Feature C: GitHub Label Re-sync

Periodically check if GitHub labels changed and update message routing:

```go
// internal/coordinator/daemon_github.go

// ResyncLabels checks imported messages for label changes
func (d *Daemon) ResyncLabels(ctx context.Context) error {
    // 1. Get all messages imported from GitHub (to_id = 'user' or misrouted)
    // 2. For each, fetch current labels from GitHub API
    // 3. If coordinator:* label exists, update to_id to appropriate inbox
    // 4. Log changes for audit
}
```

**Configuration:**
```yaml
coordinator:
  github_sync:
    enabled: true
    interval_secs: 300
    resync_labels: true           # NEW: Re-check labels on imported messages
    resync_interval_secs: 3600    # NEW: How often to re-sync (1 hour default)
```

**Routing logic:**
- `coordinator:bug` → `design-doc-creator` (bugs need design docs too)
- `coordinator:feature` → `design-doc-creator`
- `coordinator:docs` → `coordinator` (direct to general)
- `coordinator:research` → `coordinator`

**Rate limiting considerations:**
- Only check messages imported in last 7 days
- Batch GitHub API calls (list issues with `since` parameter)
- Cache label state to minimize API calls

**Estimated: ~4 hours**

### Implementation Plan Update

Add to Phase 1:

- [ ] Add short ID prefix matching for all message commands (~40 LOC)
- [ ] Add interactive menu mode for `ailang messages` (~100 LOC)
- [ ] Add `ailang messages forward` CLI command (~60 LOC)
- [ ] Add forward logging/audit support (~20 LOC)
- [ ] Add `ResyncLabels()` to daemon (~80 LOC)
- [ ] Add `resync_labels` config option (~10 LOC)
- [ ] Unit tests for all new features (~120 LOC)

**Additional LOC: ~430**

## Future Work

- **Partial revision** - Revise specific sections only
- **Reviewer assignment** - Route revisions to specific reviewers
- **Revision templates** - Structured feedback forms
- **Auto-summarize changes** - Generate diff summary for long revisions
- **Revision analytics** - Track common revision patterns

---

**Document created**: 2026-01-02
**Last updated**: 2026-01-02 (added interactive menu, forward CLI, label re-sync)
