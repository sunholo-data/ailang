# Sprint Plan: M-TRANSCRIPT

**Sprint ID**: M-TRANSCRIPT
**Design Doc**: [m-transcript-unified-conversation-history.md](m-transcript-unified-conversation-history.md)
**Duration**: 3 days (~16 hours)
**Risk Level**: Low (infrastructure exists, mostly wiring)

## Sprint Summary

Expose existing conversation history through CLI and API, add feedback loop for rejection/re-trigger workflow with session continuity.

**Key Deliverables:**
1. `[c] View chat history` action in approval menu
2. REST API `/api/tasks/{id}/events`
3. Enhanced `[r] Reject` with feedback → message → re-trigger
4. OTEL spans for human interactions
5. (Stretch) Eval harness refactor

## Current Status

**Existing Infrastructure:**
- `task_events` table: ✅ Events stored via `StoreTaskEvent()`
- `GetTaskEvents()`: ✅ Retrieval API exists
- `TaskRecord.SessionID`: ✅ Already stored for resumption
- Action menu: ✅ `coordinator_list.go` has extensible actions

**What's Missing:**
- CLI action to display chat history
- REST endpoint to expose events
- Feedback prompt on rejection
- Re-trigger with `--resume` flag
- OTEL spans for human interactions

## Velocity Analysis

Based on recent CHANGELOG entries (telemetry, traces, coordinator work):
- Average: ~150-200 LOC/day of focused work
- This sprint: ~555 LOC total
- Conservative estimate: 3 days

## Milestones

### M1: Event Formatter (~150 LOC, ~2 hours)

Create shared event formatting logic used by both CLI and API.

**Tasks:**
- [ ] Create `internal/coordinator/event_formatter.go`
- [ ] `FormatEventsAsText()` - human-readable with turn separators
- [ ] `FormatEventsAsJSON()` - structured JSON output
- [ ] Handle all stream types: text, tool_use, turn_start, turn_end
- [ ] Add unit tests

**Files:**
- NEW: `internal/coordinator/event_formatter.go` (~120 LOC)
- NEW: `internal/coordinator/event_formatter_test.go` (~80 LOC)

**Acceptance Criteria:**
- [ ] Formatter produces readable turn-by-turn output
- [ ] Tool calls highlighted with input/output
- [ ] Tests cover all event types
- [ ] Works with empty event list (no panic)

---

### M2: CLI Chat History Action (~100 LOC, ~2 hours)

Add `[c] View chat history` to the approval action menu.

**Tasks:**
- [ ] Add `[c]` option to action menu in `coordinator_list.go`
- [ ] Fetch events via `store.GetTaskEvents()`
- [ ] Display using `FormatEventsAsText()`
- [ ] Support paging for long conversations (press Enter to continue)
- [ ] Reuse in both `pending` and task detail views

**Files:**
- MODIFY: `cmd/ailang/coordinator_list.go` (~60 LOC)
- MODIFY: `cmd/ailang/coordinator_browse.go` (~40 LOC)

**Acceptance Criteria:**
- [ ] `[c]` action displays formatted chat history
- [ ] Works in approval list view
- [ ] Works in task detail view
- [ ] Long conversations paginated

---

### M3: REST API Endpoint (~80 LOC, ~1.5 hours)

Add `/api/tasks/{id}/events` endpoint for dashboard integration.

**Tasks:**
- [ ] Create handler in `internal/server/handlers_task_events.go`
- [ ] Support query params: `?limit=N`, `?turn=N`, `?type=text|tool_use`
- [ ] Register route in `routes.go`
- [ ] Add handler tests

**Files:**
- NEW: `internal/server/handlers_task_events.go` (~60 LOC)
- NEW: `internal/server/handlers_task_events_test.go` (~60 LOC)
- MODIFY: `internal/server/routes.go` (~5 LOC)

**Acceptance Criteria:**
- [ ] `GET /api/tasks/{id}/events` returns JSON array
- [ ] Filtering by type works
- [ ] Pagination via limit works
- [ ] 404 for unknown task ID

---

### M4: Rejection Feedback Loop (~150 LOC, ~4 hours)

Enhance `[r] Reject` to prompt for feedback, send message, and re-trigger task.

**Tasks:**
- [ ] Prompt user for feedback reason when rejecting
- [ ] Store feedback as `human_feedback` event in `task_events`
- [ ] Send feedback via message system to agent inbox
- [ ] Add `Iteration` field to `executor.Task`
- [ ] Modify Claude executor to use `--resume` when `Iteration > 1`
- [ ] Re-trigger task with same ID, incremented iteration
- [ ] Create OTEL span `human.feedback`

**Files:**
- NEW: `internal/coordinator/human_interaction.go` (~80 LOC)
- MODIFY: `cmd/ailang/coordinator_list.go` (~40 LOC)
- MODIFY: `internal/executor/claude/claude.go` (~30 LOC)
- MODIFY: `internal/executor/task.go` (~10 LOC)
- MODIFY: `internal/coordinator/daemon_tasks.go` (~30 LOC)

**Acceptance Criteria:**
- [ ] `[r]` prompts for feedback (not silent rejection)
- [ ] Feedback stored in `task_events` table
- [ ] Message sent to agent inbox
- [ ] Task re-triggers with iteration=2
- [ ] Claude uses `--resume` for iteration 2+
- [ ] OTEL span created with feedback content

---

### M5: OTEL Spans & Documentation (~75 LOC, ~2 hours)

Add OTEL spans for approval and update documentation.

**Tasks:**
- [ ] Add `human.approval` span when approving
- [ ] Add `task.iteration` attribute to all task spans
- [ ] Update CLAUDE.md with new workflow
- [ ] Add example commands to coordinator section

**Files:**
- MODIFY: `cmd/ailang/coordinator_list.go` (~25 LOC)
- MODIFY: `internal/coordinator/daemon_tasks.go` (~20 LOC)
- MODIFY: `CLAUDE.md` (~30 LOC)

**Acceptance Criteria:**
- [ ] Approval creates `human.approval` span
- [ ] Spans include iteration attribute
- [ ] CLAUDE.md documents `[c]` action
- [ ] CLAUDE.md documents feedback workflow

---

### M6 (Stretch): Eval Harness Refactor (~100 LOC, ~3 hours)

Refactor eval harness to use shared event infrastructure.

**Tasks:**
- [ ] Audit duplicate transcript code in `internal/executor/claude/`
- [ ] Create shared transcript interface
- [ ] Refactor eval harness to use `StoreTaskEvent()` pattern
- [ ] Verify all benchmarks still work

**Files:**
- MODIFY: `internal/eval_harness/executor.go` (~50 LOC net change)
- MODIFY: `internal/executor/claude/claude.go` (~50 LOC refactor)

**Acceptance Criteria:**
- [ ] No duplicate transcript code
- [ ] Eval benchmarks produce same output
- [ ] All existing tests pass

## Day-by-Day Plan

### Day 1 (~6 hours)

| Time | Milestone | Tasks |
|------|-----------|-------|
| 2h | M1: Event Formatter | Create formatter, tests |
| 2h | M2: CLI Action | Add `[c]` to action menu |
| 1.5h | M3: REST API | Add endpoint, tests |
| 0.5h | Buffer | Testing, fixes |

**End of Day 1 Checkpoint:**
- [ ] `ailang coordinator pending` → `[c]` shows chat history
- [ ] `curl /api/tasks/{id}/events` returns JSON
- [ ] All tests passing

### Day 2 (~6 hours)

| Time | Milestone | Tasks |
|------|-----------|-------|
| 4h | M4: Feedback Loop | Full rejection workflow |
| 2h | M5: OTEL + Docs | Spans, CLAUDE.md |

**End of Day 2 Checkpoint:**
- [ ] `[r]` prompts for feedback
- [ ] Task re-triggers with `--resume`
- [ ] OTEL spans visible in traces
- [ ] Documentation updated

### Day 3 (~4 hours, optional)

| Time | Milestone | Tasks |
|------|-----------|-------|
| 3h | M6: Eval Refactor | Dedupe transcript code |
| 1h | Final Testing | End-to-end verification |

**End of Day 3 Checkpoint:**
- [ ] Eval harness uses shared events
- [ ] No regressions in benchmarks
- [ ] Ready for release

## Success Metrics

| Metric | Target |
|--------|--------|
| Total LOC | ~555 |
| Test Coverage | Maintain current (~30%) |
| New Tests | ~10 test cases |
| API Endpoints | 1 new |
| CLI Actions | 1 new |

## Dependencies

- None (all infrastructure exists)

## Risks

| Risk | Mitigation |
|------|------------|
| Event counts large | Pagination, default limit=100 |
| Session resume fails | Fall back to new session |
| Feedback loop infinite | Iteration limit (max 3) |

## Open Questions

1. Should iteration limit be configurable? (Default: 3)
2. Should feedback be required on rejection? (Proposal: Required)
3. Include tool results in chat history? (Proposal: Yes, collapsed by default)

---

**Created**: 2026-01-13
**Sprint Start**: Ready for execution
