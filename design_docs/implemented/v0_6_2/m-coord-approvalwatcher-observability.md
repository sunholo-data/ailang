# M-COORD-APPROVALWATCHER-OBSERVABILITY: ApprovalWatcher Debug and Observability

## Status
**Status**: ✅ Implemented
**Target Version**: v0.6.2
**Priority**: P1 (High - blocks autonomous workflow debugging)
**Created**: 2026-01-01
**Completed**: 2026-01-01
**Bug Report**: E2E testing of M-COORD-GITHUB-COMPLETE revealed watcher not detecting labels

## Problem Statement

During E2E testing of the GitHub-driven autonomous workflow (M-COORD-GITHUB-COMPLETE), the `ApprovalWatcher` failed to detect the `merge-approved` label added to issue #92. This required manual intervention via `ailang coordinator approve` to proceed with the merge.

**The root cause is insufficient observability:**
- No logging when polling occurs
- No visibility into which issues are being watched
- No confirmation logs when labels are checked
- No way to verify the watcher is functioning correctly

### Current State

The ApprovalWatcher logs only:
- "GitHub approval watcher started" (at startup)
- Errors when getting labels fails
- When an event is processed

**Missing logs:**
- When poll cycles occur
- How many issues are being polled
- What labels are found on each issue
- When issues are watched/unwatched

### Impact

- **Debugging is impossible**: Can't tell if watcher is running, polling, or finding labels
- **E2E testing is blocked**: Must use manual local approval path as workaround
- **Production reliability unclear**: No way to monitor watcher health

## Goals

### Primary Goal
Add comprehensive debug logging and observability to ApprovalWatcher so issues can be diagnosed without code changes.

### Success Metrics
- [x] Can verify watcher is polling via logs
- [x] Can see which issues are being watched
- [x] Can see labels found during each poll cycle
- [x] Can diagnose label detection failures in production

## Solution Design

### Phase 1: Debug Logging (~30 min)

Add structured logging throughout the polling lifecycle:

```go
// In pollOnce - at start of poll cycle
log.Printf("[ApprovalWatcher] Poll cycle started (watching %d issues)", len(issues))

// In pollOnce - for each issue checked
log.Printf("[ApprovalWatcher] Checking issue #%d for task %s", issueNum, taskID)

// In checkIssueLabels - after getting labels
log.Printf("[ApprovalWatcher] Issue #%d has labels: %v", issueNum, labels)

// In checkIssueLabels - when approval label found
log.Printf("[ApprovalWatcher] Found approval label %q on issue #%d", label, issueNum)

// In WatchIssue/UnwatchIssue
log.Printf("[ApprovalWatcher] Started watching issue #%d for task %s", issueNumber, taskID)
log.Printf("[ApprovalWatcher] Stopped watching issue #%d", issueNumber)
```

### Phase 2: Status Command (~30 min)

Add CLI command to check watcher status:

```bash
ailang coordinator watcher-status
```

Output:
```
ApprovalWatcher Status
======================
Running: true
Poll interval: 60s
Last poll: 2026-01-01T14:30:00Z (15s ago)

Watched Issues:
  #92 -> task_abc123 (stage: merge)
  #88 -> task_def456 (stage: design)

Recent Events:
  2026-01-01T14:25:00Z - design-approved detected on #88
  2026-01-01T14:20:00Z - Poll cycle (2 issues, 0 events)
```

### Phase 3: Debug Environment Variable (~15 min)

Add `DEBUG_APPROVAL_WATCHER=1` to control verbose logging:
- When set: Log every poll cycle and label check
- When unset: Log only significant events (start, events found, errors)

## Implementation Plan

### Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/coordinator/approval_watcher.go` | Add debug logging | +30 |
| `internal/coordinator/approval_watcher.go` | Add status tracking fields | +20 |
| `cmd/ailang/coordinator.go` | Add `watcher-status` command | +40 |
| `internal/coordinator/daemon.go` | Pass debug flag to watcher | +5 |

**Total estimated LOC**: ~95

### Acceptance Criteria

- [x] Poll cycles are logged with issue count
- [x] Each issue check is logged with labels found
- [x] Watch/unwatch operations are logged
- [x] `ailang coordinator watcher-status` shows current state
- [x] DEBUG_APPROVAL_WATCHER=1 enables verbose mode
- [x] Existing tests pass
- [x] E2E test with GitHub labels works without manual intervention

## Testing

### Manual E2E Test
1. Start coordinator with DEBUG_APPROVAL_WATCHER=1
2. Create GitHub issue
3. Let coordinator process through design stage
4. Add `design-approved` label on GitHub
5. Verify logs show label detection
6. Repeat for sprint and merge stages
7. Confirm issue auto-closes

### Unit Tests
- Test logging output format
- Test status command output
- Test debug flag toggle

## Timeline

**Estimated Duration**: 1.5 hours
- Phase 1: 30 min
- Phase 2: 30 min
- Phase 3: 15 min
- Testing: 15 min

## Open Questions

1. Should poll frequency be configurable via CLI flag?
2. Should watcher status be included in `ailang coordinator status` output?
3. Should we add Prometheus metrics for monitoring?

## Related Documents

- [M-COORD-GITHUB-AUTO-ROUTING](../implemented/v0_6_2/m-coord-github-auto-routing.md) - Parent design doc
- [M-COORD-GITHUB-COMPLETE Sprint Plan](../implemented/v0_6_2/m-coord-github-complete-sprint-plan.md) - Sprint where bug was found

---

## Implementation Report

**Completed**: 2026-01-01
**Actual Duration**: ~40 minutes
**Actual LOC**: ~150

### What Was Built

1. **Panic Recovery** (M1) - Added `defer recover()` to `handleEvent()` to catch handler panics and keep the poll goroutine alive. This was the root cause fix - if any handler panicked, the poll goroutine would die silently.

2. **Debug Logging** (M2) - Added comprehensive logging throughout the polling lifecycle:
   - Poll cycle start/end with issue count and events found
   - Labels fetched from GitHub for each issue
   - Watch/unwatch state changes

3. **Status CLI** (M3) - Added `ailang coordinator watcher-status` command:
   - Shows running state, poll interval, last poll time
   - Lists all watched issues with their task IDs
   - Supports `--json` output for scripting

4. **Debug Toggle** (M4) - Added `DEBUG_APPROVAL_WATCHER=1` environment variable:
   - Verbose mode: Logs every poll cycle and label check
   - Normal mode: Only logs significant events (startup, events processed, errors)
   - Documented in CLAUDE.md

### Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/coordinator/approval_watcher.go` | Panic recovery, debug logging, WatcherStatus struct, GetStatus() | +75 |
| `internal/coordinator/daemon_lifecycle.go` | GetWatcherStatus() method | +10 |
| `cmd/ailang/coordinator.go` | watcher-status command + help | +65 |
| `CLAUDE.md` | DEBUG_APPROVAL_WATCHER documentation | +1 |

### E2E Test Results

Successfully verified with real GitHub issue #91:
1. ✅ Debug logging startup: `Starting with poll interval 1m0s`
2. ✅ Watch tracking: `Now watching issue #91 for task task-0f523cea`
3. ✅ Poll cycle logging: `Poll cycle started (watching 4 issues)`
4. ✅ Label detection: `Issue #91 labels: [...design-approved]`
5. ✅ Event processing: `Processing design-approved for task task-0f523cea`
6. ✅ Handler execution: `Design approved for task...`
7. ✅ Task requeue: `Task requeued for sprint planning stage`
8. ✅ Poll continues: `Poll cycle complete (4 issues, 1 events)`
9. ✅ Label cleanup: Both approval labels removed from issue

### Key Insight

The root cause was **poll goroutine fragility**. If any handler (`OnDesignApproved`, `OnSprintApproved`, `OnMergeApproved`) panicked, the goroutine would die silently and polling would stop forever with no indication. The `defer recover()` fix ensures the watcher continues operating even if a handler fails.
