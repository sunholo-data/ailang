# Sprint Plan: M-MSG-ROUTING - Message Routing DX Improvements

## Summary

Improve message CLI developer experience with short IDs, interactive menu, forward command, and automatic label re-sync to fix the issue where GitHub labels added after import don't update message routing.

**Duration:** 1.5 days (~10 hours implementation)
**Dependencies:** None (builds on existing messaging system)
**Risk Level:** Low
**Design Doc:** [m-coord-auto-revision.md](m-coord-auto-revision.md) (Additional Features section)

## Current Status Analysis

### Completed Recently
- ✅ M-OTEL: OpenTelemetry integration (~630 LOC) in 2 days
- ✅ v0.6.2: Coordinator GitHub auto-routing
- ✅ Examples command with searchable metadata

### Velocity
- Recent average: ~300-400 LOC/day
- Estimated capacity: ~500 LOC for this sprint (conservative)

### Problem Being Solved
- GitHub issue #104 was imported BEFORE `coordinator:feature` label was added
- Message went to `user` inbox instead of `design-doc-creator`
- No way to forward messages between inboxes
- UUIDs like `87738035-3807-4b2f-935a-09ede9fb0c3a` are painful to work with

## Proposed Milestones

### Milestone 1: Short ID Prefix Matching
**Goal:** Allow short ID prefixes (like git) for all message commands
**Estimated:** 40 LOC implementation + 30 LOC tests = 70 LOC
**Duration:** 1 hour

**Tasks:**
- Add `resolveMessageID(prefix string) (string, error)` to messages_util.go
- Query DB for messages where ID starts with prefix
- Return error if 0 matches or >1 match (ambiguous)
- Update `read`, `ack`, `unack` commands to use resolver

**Acceptance Criteria:**
- [ ] `ailang messages read 877` works if unique prefix
- [ ] `ailang messages ack 87738` works
- [ ] Error on ambiguous prefix: "Multiple messages match '87', use longer prefix"
- [ ] Error on no match: "No message found with prefix 'xyz'"
- [ ] All existing tests passing
- [ ] Linting clean

**Risks:**
- None - simple string prefix matching

### Milestone 2: Interactive Menu Mode
**Goal:** Show numbered menu when running `ailang messages` without subcommand
**Estimated:** 100 LOC implementation + 40 LOC tests = 140 LOC
**Duration:** 2 hours

**Tasks:**
- Create `runMessagesInteractive()` in messages.go
- Display numbered message list with unread indicator (●)
- Accept single-key input: 1-9 for selection, r/f/a/d/q for actions
- Integrate with existing commands (read, ack, forward)

**Example output:**
```
┌─────────────────────────────────────────────────────────────────┐
│ AILANG Messages - user inbox (2 unread)                         │
├─────────────────────────────────────────────────────────────────┤
│ [1] ● Consider importing linting issues from Sonar Cloud  #104  │
│ [2]   Artifact Discovery Fix v2                            #106  │
│ [3]   Ultrathink                                           #107  │
└─────────────────────────────────────────────────────────────────┘
Actions: [r]ead [f]orward [a]ck [d]elete [q]uit

Select message (1-3) or action: _
```

**Acceptance Criteria:**
- [ ] `ailang messages` shows interactive menu
- [ ] Number keys select messages
- [ ] Action keys perform operations
- [ ] Menu refreshes after actions
- [ ] `q` exits cleanly
- [ ] Works in TTY (detects non-interactive and falls back to list)

**Risks:**
- Terminal raw mode handling - use simple line-based input first
- Non-TTY environments - detect and fall back to list mode

### Milestone 3: Message Forward CLI
**Goal:** Add `ailang messages forward` command to move messages between inboxes
**Estimated:** 80 LOC implementation + 40 LOC tests = 120 LOC
**Duration:** 1.5 hours

**Tasks:**
- Create `runMessagesForward(args)` in messages_crud.go
- Accept `MSG_ID --to INBOX` or interactive selection
- Update message's `to_id` in database
- Log forward action for audit trail

**Usage:**
```bash
ailang messages forward 877 --to design-doc-creator
ailang messages forward 877 --to design-doc-creator --reason "Label added after import"
```

**Acceptance Criteria:**
- [ ] Forward command updates message inbox
- [ ] Short ID prefix works with forward
- [ ] `--reason` flag logs reason
- [ ] Error if message not found
- [ ] Error if target inbox invalid
- [ ] All tests passing

**Risks:**
- None - simple database update

### Milestone 4: GitHub Label Re-sync
**Goal:** Periodically re-check GitHub labels and update message routing
**Estimated:** 100 LOC implementation + 60 LOC tests = 160 LOC
**Duration:** 2.5 hours

**Tasks:**
- Add `ResyncLabels(ctx)` to daemon_github.go
- Query messages with `from_id = 'github'` imported in last 7 days
- For each, fetch current labels via `gh api`
- If `coordinator:*` label exists, update `to_id` based on routing rules
- Add config options: `resync_labels`, `resync_interval_secs`

**Routing logic:**
- `coordinator:bug` → `design-doc-creator`
- `coordinator:feature` → `design-doc-creator`
- `coordinator:docs` → `coordinator`
- `coordinator:research` → `coordinator`

**Acceptance Criteria:**
- [ ] Daemon re-syncs labels on configured interval
- [ ] Messages routed correctly when labels added after import
- [ ] Rate limiting: only check messages from last 7 days
- [ ] Audit log shows label re-sync events
- [ ] Config options work correctly
- [ ] Tests for routing logic

**Risks:**
- GitHub API rate limits - batch requests, use `since` parameter
- Performance - limit to recent messages only

### Milestone 5: Integration Testing & Documentation
**Goal:** End-to-end testing and CLAUDE.md updates
**Estimated:** 40 LOC tests + 30 LOC docs = 70 LOC
**Duration:** 1 hour

**Tasks:**
- Write integration test: import → add label → resync → verify routing
- Update CLAUDE.md with new message commands
- Update teaching prompt if needed

**Acceptance Criteria:**
- [ ] Integration test passes
- [ ] CLAUDE.md documents forward command
- [ ] Help text updated for new commands
- [ ] All tests passing
- [ ] Linting clean

## Success Metrics
- Test coverage: >80% for new code
- All tests passing: ✅
- All linting passing: ✅
- Issue #104 can be forwarded with short command

## Dependencies
- None - builds on existing messaging infrastructure

## Implementation Order

1. **M1: Short ID Prefix** (foundational - other features use it)
2. **M3: Forward CLI** (needed to fix issue #104 immediately)
3. **M2: Interactive Menu** (DX improvement, uses short IDs)
4. **M4: Label Re-sync** (automated fix for future issues)
5. **M5: Testing & Docs** (finalization)

## Total Estimates

| Milestone | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| M1: Short ID | 40 | 30 | 70 |
| M2: Interactive Menu | 100 | 40 | 140 |
| M3: Forward CLI | 80 | 40 | 120 |
| M4: Label Re-sync | 100 | 60 | 160 |
| M5: Integration | 40 | 30 | 70 |
| **Total** | **360** | **200** | **560 LOC** |

## Notes
- Start with M1 (short IDs) as other features depend on it
- M3 (forward) is high priority - immediately fixes issue #104
- M2 (menu) can be simplified if time-constrained (use line input vs raw mode)
- M4 (re-sync) prevents future issues but not urgent

---

**Document created**: 2026-01-03
**Sprint ID**: M-MSG-ROUTING
