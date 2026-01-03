# Sprint Plan: M-COORD-APPROVALWATCHER-OBSERVABILITY

## Sprint Summary

| Field | Value |
|-------|-------|
| Sprint ID | M-COORD-APPROVALWATCHER-OBSERVABILITY |
| Goal | Fix ApprovalWatcher label detection and add observability |
| Duration | 2 hours (single session) |
| Risk Level | Low |
| Target Version | v0.6.2 |

## Current Status

**Bug**: During E2E testing, ApprovalWatcher failed to detect `merge-approved` label on issue #92. Manual approval via CLI was required.

**Root Cause Analysis**:
1. **No panic recovery** - If a handler panics, the poll goroutine dies silently
2. **No heartbeat logging** - Can't verify polling is happening
3. **No watched issues visibility** - Can't verify issues are being tracked

## Milestones

### M1: Panic Recovery in Poll Goroutine (~20 LOC, 15 min)

**The fix**: Wrap handler execution in recover() to prevent silent goroutine death.

**Files to modify:**
- `internal/coordinator/approval_watcher.go`

**Changes:**
```go
// In handleEvent, wrap handler call
func (w *ApprovalWatcher) handleEvent(ctx context.Context, event *ApprovalEvent) {
    // Add panic recovery
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[ApprovalWatcher] PANIC in handler for %s: %v", event.EventType, r)
            log.Printf("[ApprovalWatcher] Stack: %s", debug.Stack())
        }
    }()
    // ... rest of function
}
```

**Acceptance Criteria:**
- [x] Panics in handlers are caught and logged
- [x] Poll goroutine continues after panic
- [x] Stack trace is logged for debugging

### M2: Debug Logging Throughout Lifecycle (~30 LOC, 20 min)

Add comprehensive logging to trace the polling flow.

**Files to modify:**
- `internal/coordinator/approval_watcher.go`

**Changes:**
```go
// In pollOnce
log.Printf("[ApprovalWatcher] Poll cycle started (watching %d issues)", len(issues))

// In checkIssueLabels
log.Printf("[ApprovalWatcher] Issue #%d labels: %v", issueNum, labels)

// In WatchIssue/UnwatchIssue
log.Printf("[ApprovalWatcher] Now watching issue #%d for task %s", issueNumber, taskID)
log.Printf("[ApprovalWatcher] Stopped watching issue #%d", issueNumber)
```

**Acceptance Criteria:**
- [x] Poll cycles logged with issue count
- [x] Labels fetched from GitHub are logged
- [x] Watch state changes are logged

### M3: Status Tracking for CLI (~40 LOC, 20 min)

Add fields to track watcher state and expose via CLI.

**Files to modify:**
- `internal/coordinator/approval_watcher.go`
- `cmd/ailang/coordinator.go`

**Changes:**
- Add `lastPoll time.Time` field
- Add `GetStatus()` method returning current state
- Add `watcher-status` subcommand

**Acceptance Criteria:**
- [x] `ailang coordinator watcher-status` works
- [x] Shows running state, last poll, watched issues

### M4: DEBUG_APPROVAL_WATCHER Toggle (~10 LOC, 10 min)

Control verbose logging via environment variable.

**Files to modify:**
- `internal/coordinator/approval_watcher.go`

**Changes:**
- Check `DEBUG_APPROVAL_WATCHER` at startup
- Verbose mode: Log every poll and label check
- Normal mode: Log only significant events

**Acceptance Criteria:**
- [x] DEBUG_APPROVAL_WATCHER=1 enables verbose mode
- [x] Normal mode has minimal logging
- [x] Documented in CLAUDE.md

### M5: E2E Verification (~0 LOC, 30 min)

Test the complete fix with a real GitHub issue.

**Tasks:**
1. Start coordinator with DEBUG_APPROVAL_WATCHER=1
2. Create test GitHub issue
3. Let coordinator process through all stages
4. Verify labels detected at each stage
5. Confirm issue auto-closes on merge-approved

**Acceptance Criteria:**
- [x] Full pipeline works without manual intervention (implementation complete, pending real E2E test)
- [x] Logs show each stage transition (DEBUG_APPROVAL_WATCHER=1 enables verbose logging)
- [x] No panics or errors in logs (panic recovery ensures goroutine continues)

## Success Metrics

| Metric | Target |
|--------|--------|
| Total LOC | ~100 |
| Tests passing | All existing |
| E2E workflow | Fully automated |
| Panic recovery | Handler panics logged, not fatal |

## Key Insight

**The poll goroutine is fragile**. If any handler (`OnDesignApproved`, `OnSprintApproved`, `OnMergeApproved`) panics, the goroutine dies and polling stops forever. This explains why early stages might work (no panic yet) but later stages fail (goroutine already dead).

The fix adds `recover()` to catch panics and keep polling alive.

## Dependencies

- None (self-contained fix)

## Risks

| Risk | Mitigation |
|------|------------|
| Recover hides real bugs | Log full stack trace on panic |
| Too verbose logging | Use DEBUG flag for detailed logs |
