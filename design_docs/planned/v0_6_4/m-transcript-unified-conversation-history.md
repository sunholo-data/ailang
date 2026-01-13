# M-TRANSCRIPT: Unified Conversation History & Feedback Loop

**Status**: Planned
**Target**: v0.6.4
**Priority**: P1 (Medium-High)
**Estimated**: 3 days
**Dependencies**: None (infrastructure already exists)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact - reads existing deterministic event log |
| A2: Replayability | +1 | Enables full conversation replay for debugging |
| A3: Effect Legibility | +1 | Makes agent execution AND human feedback visible |
| A4: Explicit Authority | 0 | No new capabilities required |
| A5: Bounded Verification | +1 | Enables local inspection of agent decisions |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured JSON, OTEL spans for machine consumption |
| A8: Minimal Syntax | 0 | No new syntax (CLI integration, not language) |
| A9: Cost Visibility | +1 | Exposes token/cost data per turn |
| A10: Composability | +1 | Composes with existing coordinator, evals, messages |
| A11: Structured Failure | +1 | Rejection feedback is typed and traceable |
| A12: System Boundary | +1 | Human-agent boundary explicitly captured |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): All effects visible (read-only + feedback writes)
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): JSON-first, OTEL spans for filtering

## Problem Statement

### Part 1: Conversation History Not Accessible

The coordinator captures conversation events to `task_events` table, but this data isn't exposed in the approval workflow. Reviewers see only the final output, not the reasoning.

**Motivating Example (task-39525386):**
- Agent completed with text output, no design doc written
- Only by querying SQLite directly did we discover the agent asked follow-up questions instead of writing a file
- The `[l] View execution logs` action shows raw logs, not the structured conversation

### Part 2: Rejection Has No Feedback Loop

When rejecting work, there's no way to tell the agent **why**. The task just dies. This wastes the context the agent built up.

**Current rejection flow:**
```
Human: [r] Reject
→ Task marked rejected
→ Worktree preserved (but agent context lost)
→ No iteration possible
```

**Desired rejection flow:**
```
Human: [r] Reject
Human: "You asked follow-up questions instead of writing the design doc. Please write it."
→ Feedback stored as event (with OTEL span)
→ Feedback sent via message system
→ Task re-triggered (same task ID, new iteration)
→ Agent resumes with human feedback in context
```

## Goals

**Primary Goal:** Integrate conversation history into approval workflow and enable feedback-driven iteration.

**Success Metrics:**
- `[c] View chat history` action in approval menu shows full conversation
- `[r] Reject` prompts for feedback reason
- Rejection feedback sent via message system to agent inbox
- Task re-triggers with same ID (iteration counter increments)
- All interactions have OTEL spans for filtering (`human.feedback`, `human.approval`)
- REST API exposes events for dashboard integration

## Solution Design

### Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Approval Workflow Integration                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ailang coordinator pending                                              │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  Task: task-39525386                                              │  │
│   │  Title: Investigate adding OpenAI codex...                        │  │
│   │  Status: approval                                                 │  │
│   │                                                                   │  │
│   │  Actions:                                                         │  │
│   │    [d]  View diff (full)                                          │  │
│   │    [s]  View diff summary (--stat)                                │  │
│   │    [f]  Browse files changed                                      │  │
│   │    [c]  View chat history  ← NEW                                  │  │
│   │    [l]  View execution logs                                       │  │
│   │    [a]  Approve and merge                                         │  │
│   │    [r]  Reject (with feedback)  ← ENHANCED                        │  │
│   │    [q]  Back to list                                              │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Event Storage (Existing)                        │
│  ┌─────────────────┐                    ┌─────────────────┐              │
│  │ task_events     │───GetTaskEvents()──▶│ CLI [c] action  │              │
│  │ (agent actions) │                    │ REST API        │              │
│  └─────────────────┘                    └─────────────────┘              │
├─────────────────────────────────────────────────────────────────────────┤
│                           NEW: Human Interaction Layer                    │
│                                                                          │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐      │
│  │ OTEL Spans      │    │ Message System  │    │ Task Re-trigger │      │
│  │ human.approval  │    │ feedback inbox  │    │ iteration++     │      │
│  │ human.feedback  │◀───│                 │───▶│ same task_id    │      │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘      │
│                                                                          │
│  stream_type: "human_feedback"  ← New event type stored alongside        │
│  stream_type: "human_approval"    agent events for complete audit trail  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Event Types

Extend `TaskEventRecord.StreamType` with human interaction events:

| StreamType | Source | Description |
|------------|--------|-------------|
| `turn_start` | Agent | Agent turn begins |
| `text` | Agent | Text output |
| `tool_use` | Agent | Tool invocation |
| `tool_result` | Agent | Tool response |
| `turn_end` | Agent | Agent turn ends |
| **`human_feedback`** | Human | **NEW**: Rejection reason or guidance |
| **`human_approval`** | Human | **NEW**: Approval with optional comment |
| **`iteration_start`** | System | **NEW**: Task re-triggered with feedback |

### Implementation Plan

**Phase 1: CLI Chat History Action** (~3 hours)
- [ ] Add `[c] View chat history` to `coordinator_list.go` action menu
- [ ] Reuse same code for both `pending` and task detail views
- [ ] Format conversation with turn separators and tool highlighting
- [ ] Support paging for long conversations

**Phase 2: REST API** (~2 hours)
- [ ] Add `GET /api/tasks/{id}/events` endpoint
- [ ] Return `TaskEventRecord` array as JSON
- [ ] Support `?limit=N`, `?turn=N`, `?type=text|tool_use|human_feedback`
- [ ] Add tests

**Phase 3: Rejection Feedback Loop** (~4 hours)
- [ ] Enhance `[r] Reject` to prompt for feedback reason
- [ ] Store feedback as `human_feedback` event in `task_events`
- [ ] Create OTEL span `human.feedback` with feedback content
- [ ] Send feedback via message system to agent's inbox:
  ```
  ailang messages send <agent_inbox> "<feedback>" \
    --title "Rejection feedback for task-XXX" \
    --from "human" \
    --type "feedback"
  ```
- [ ] Re-trigger task with same ID, increment iteration counter
- [ ] **Use `--resume <sessionId>` to preserve Claude Code context** (see Session Continuity below)
- [ ] Pass message ID to agent so it can read context

**Phase 4: Human Interaction OTEL Spans** (~2 hours)
- [ ] Add `human.approval` span when approving
- [ ] Add `human.feedback` span when rejecting with feedback
- [ ] Add `task.iteration` attribute to all spans
- [ ] Enable filtering: `stream_type IN ('human_feedback', 'human_approval')`

**Phase 5: Eval Harness Refactor** (~3 hours)
- [ ] Audit duplicate transcript code in `internal/executor/`
- [ ] Create shared event storage interface
- [ ] Refactor eval harness to use coordinator events
- [ ] Verify benchmarks still produce correct transcripts

### Files to Modify/Create

**New files:**
- `internal/server/handlers_task_events.go` - REST endpoint (~80 LOC)
- `internal/coordinator/event_formatter.go` - Shared formatting (~150 LOC)
- `internal/coordinator/human_interaction.go` - Feedback loop logic (~120 LOC)

**Modified files:**
- `cmd/ailang/coordinator_list.go` - Add `[c]` action, enhance `[r]` (~100 LOC)
- `internal/coordinator/store.go` - Add human event types (~20 LOC)
- `internal/coordinator/daemon_tasks.go` - Re-trigger with iteration (~50 LOC)
- `internal/executor/claude/claude.go` - Add `--resume` support (~30 LOC)
- `internal/server/routes.go` - Register endpoint (~5 LOC)

### Session Continuity (Critical)

**Claude Code has two different session modes:**

| Flag | Behavior |
|------|----------|
| `--session-id <uuid>` | Create NEW session with that UUID |
| `--resume <sessionId>` | **Resume EXISTING session** with full context |

**Current behavior** (iteration 1):
```bash
claude -p "..." --session-id abc-123  # Creates new session
# → Agent executes, produces output
# → SessionID "abc-123" stored in TaskRecord
```

**Feedback iteration** (iteration 2+):
```bash
claude -p "<feedback>" --resume abc-123  # Resumes with context!
# → Agent sees ALL previous turns
# → Agent sees rejection feedback as new user message
# → Agent can improve based on what it already did
```

**Implementation in `internal/executor/claude/claude.go`:**

```go
// Current code (always new session):
args := []string{
    "-p", task.Directive,
    "--session-id", sessionID,
}

// New code (resume if iteration > 1):
args := []string{"-p", task.Directive}
if task.Iteration > 1 && task.SessionID != "" {
    args = append(args, "--resume", task.SessionID)
} else {
    args = append(args, "--session-id", sessionID)
}
```

**Task struct changes:**

```go
type Task struct {
    // ... existing fields
    Iteration  int    `json:"iteration,omitempty"`   // NEW: 1-based iteration counter
    SessionID  string `json:"session_id,omitempty"`  // Existing: reused for resume
}
```

**Why this matters:**
- Without `--resume`, agent starts fresh with no memory
- With `--resume`, agent sees everything it did before + human feedback
- Dramatically improves feedback loop effectiveness
- Gemini CLI has equivalent `--conversation-id` for resumption

## Examples

### Example 1: View Chat History in Approval Menu

```
$ ailang coordinator pending

Pending Approvals (1):

  1. task-39525386 - Investigate adding OpenAI codex as a background executor
     Agent: design-doc-creator | Cost: $0.09 | Tokens: 895

Select task (1-1) or [q] quit: 1

══════════════════════════════════════════════════════════════════════
Task: task-39525386
Title: Investigate adding OpenAI codex as a background executor
Status: ⏳ approval
Agent: design-doc-creator

Actions:
  [d]  View diff (full)
  [s]  View diff summary (--stat)
  [f]  Browse files changed
  [c]  View chat history         ← NEW
  [l]  View execution logs
  [a]  Approve and merge
  [r]  Reject (with feedback)
  [q]  Back to list

Action: c

─── Chat History ─────────────────────────────────────────────────────

Turn 1 (07:11:18)
┌─────────────────────────────────────────────────────────────────────
│ I'll help you investigate the OpenAI Codex CLI documentation and
│ compare it with Claude and Gemini's JSONL formats.
│
│ [TOOL] WebFetch
│   url: https://developers.openai.com/codex/cli/
│   prompt: "Extract JSONL format specifications"
└─────────────────────────────────────────────────────────────────────

Turn 2 (07:11:23)
┌─────────────────────────────────────────────────────────────────────
│ The main CLI page doesn't specify JSONL format. Let me check the
│ reference documentation:
│
│ [TOOL] WebFetch
│   url: https://developers.openai.com/codex/cli/reference
└─────────────────────────────────────────────────────────────────────

Turn 3 (07:11:27)
┌─────────────────────────────────────────────────────────────────────
│ Let me check the Advanced Config documentation:
│
│ [TOOL] WebFetch
│   url: https://developers.openai.com/codex/cli/advanced
└─────────────────────────────────────────────────────────────────────

Turn 4 (07:11:31)
┌─────────────────────────────────────────────────────────────────────
│ Based on my research, here's what I found:
│
│ ## OpenAI Codex CLI JSONL Format
│ ...
│
│ Would you like me to:
│ - Examine AILANG's Claude and Gemini JSONL parsers?
│ - Create a design doc for multi-provider JSONL support?
│ - Check if there's existing code for OpenAI integration?
└─────────────────────────────────────────────────────────────────────

[Press Enter to return to actions, or 't N' to jump to turn N]
```

### Example 2: Rejection with Feedback

```
Action: r

─── Reject with Feedback ─────────────────────────────────────────────

Rejecting task: task-39525386
Agent: design-doc-creator

Why are you rejecting this work? Your feedback will be sent to the
agent and the task will be re-triggered for another iteration.

Feedback: You asked follow-up questions instead of writing the design
doc. Your role is design-doc-creator - you should ALWAYS write a
design doc file to design_docs/planned/, not just provide text output.
Please write the design doc for adding OpenAI Codex executor support.

Confirm rejection and re-trigger? [y/N]: y

✓ Feedback stored (event ID: 157)
✓ OTEL span created: human.feedback
✓ Message sent to inbox: design-doc-creator
✓ Task re-triggered (iteration: 2)

The agent will receive your feedback and attempt the task again.
```

### Example 3: Task Events API with Human Feedback

```bash
$ curl "http://localhost:1957/api/tasks/task-39525386/events?type=human_feedback"
```

```json
{
  "task_id": "task-39525386",
  "iteration": 2,
  "events": [
    {
      "id": 157,
      "stream_type": "human_feedback",
      "text": "You asked follow-up questions instead of writing...",
      "created_at": "2026-01-13T08:30:00.000Z",
      "otel_span_id": "abc123",
      "otel_trace_id": "def456"
    }
  ]
}
```

### Example 4: OTEL Span Hierarchy

```
task.execute (task-39525386, iteration=1, session_id=abc-123)
├── claude.execute
│   ├── exec.turn (turn=1)
│   │   └── tool.WebFetch
│   ├── exec.turn (turn=2)
│   │   └── tool.WebFetch
│   ├── exec.turn (turn=3)
│   │   └── tool.WebFetch
│   └── exec.turn (turn=4)
│       └── [text output - asks follow-up questions]
└── human.feedback  ← NEW: Filterable span
    └── attributes:
        feedback: "You asked follow-up questions..."
        iteration: 1
        next_iteration: 2
        session_id: abc-123  ← Preserved for resume

task.execute (task-39525386, iteration=2, session_id=abc-123)  ← Same session!
├── claude.execute
│   └── args: ["--resume", "abc-123"]  ← Resumes with full context
│       ├── [Agent sees turns 1-4 from iteration 1]
│       ├── [Agent sees human feedback as new user message]
│       └── exec.turn (turn=5)  ← Continues numbering
│           └── [writes design doc this time]
└── human.approval  ← NEW: Filterable span
    └── attributes:
        comment: "LGTM"
        iteration: 2
        session_id: abc-123
```

**Key insight:** With `--resume`, the agent continues the SAME conversation. Turn 5 follows turns 1-4. The agent has full memory of what it tried before.

## Success Criteria

- [ ] `[c] View chat history` action displays formatted conversation
- [ ] Chat history works in both `pending` list and task detail views
- [ ] `[r] Reject` prompts for feedback reason (not just silent rejection)
- [ ] Feedback stored as `human_feedback` event in `task_events`
- [ ] Feedback sent via message system to agent inbox
- [ ] Task re-triggers with same ID and incremented iteration
- [ ] Re-triggered task uses `--resume <sessionId>` (Claude) or `--conversation-id` (Gemini)
- [ ] Agent sees full previous context + feedback on resume
- [ ] OTEL spans created for `human.feedback` and `human.approval`
- [ ] REST API returns events with human interaction events included
- [ ] Filtering by `stream_type` works in API
- [ ] All tests passing
- [ ] CLAUDE.md updated with new workflow

## Testing Strategy

**Unit tests:**
- Event formatter renders turns correctly
- Human feedback events stored properly
- Iteration counter increments correctly

**Integration tests:**
- Full rejection → feedback → re-trigger flow
- Message delivery to agent inbox
- OTEL span creation and attributes

**Manual testing:**
- Complete approval workflow with chat history review
- Rejection with feedback, verify agent receives message
- Query OTEL traces for human interaction spans

## Non-Goals

**Not in this feature:**
- **Real-time streaming** - Events retrieved after completion (WebSocket handles live)
- **Multi-turn human conversation** - Single feedback per rejection (not a chat)
- **Automatic retry** - Human must explicitly reject with feedback
- **Full dashboard UI** - REST API enables dashboard, but UI is separate phase

## Timeline

**Day 1** (6 hours):
- Phase 1: CLI `[c]` action
- Phase 2: REST API endpoint
- Tests for both

**Day 2** (6 hours):
- Phase 3: Rejection feedback loop
- Phase 4: OTEL spans
- Message system integration

**Day 3** (4 hours):
- Phase 5: Eval harness refactor
- Documentation updates
- Final testing

**Total: ~16 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Large event counts slow API | Medium | Pagination, default limit=100 |
| Feedback loop creates infinite retries | Medium | Iteration limit (default: 3) |
| Agent doesn't read feedback message | Medium | Include feedback in task content directly |
| OTEL spans add latency | Low | Async span creation |

## Related Documents

**Implemented (informs design):**
- [M-DASH](../../implemented/v0_3_10/M-DASH.md) - Dashboard architecture
- [M-EXEC](../../implemented/v0_6_1/m-exec-multi-executor-support.md) - Executor streaming

**Planned (check for overlap):**
- [M-UI-REFACTOR](../v0_6_3/m-ui-refactor-ai-friendly.md) - Dashboard improvements
- [M-EVAL-UNIFIED](../v0_7_0/m-eval-unified-exec-codegen.md) - Eval infrastructure unification

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/coordinator/store.go:193` - `StoreTaskEvent()` interface
- `cmd/ailang/coordinator_list.go:148` - Action menu code
- `internal/executor/claude/claude.go:174` - Transcript code to refactor

## Future Work

- **Dashboard conversation viewer**: React component showing chat history in approval modal
- **Voice feedback**: Dictate rejection feedback (transcribe to text)
- **Feedback templates**: Common rejection reasons as quick-select options
- **Agent learning**: Aggregate feedback for prompt improvement
- **Cross-task threading**: Show parent/child task conversations together

---

**Document created**: 2026-01-13
**Last updated**: 2026-01-13
