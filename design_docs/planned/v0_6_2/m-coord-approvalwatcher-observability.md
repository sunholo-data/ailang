# M-COORD-APPROVALWATCHER-OBSERVABILITY: ApprovalWatcher Debug and Observability

## Status
**Status**: Planned
**Target Version**: v0.6.2
**Priority**: P1 (High - blocks autonomous workflow debugging)
**Created**: 2026-01-01
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
- [ ] Can verify watcher is polling via logs
- [ ] Can see which issues are being watched
- [ ] Can see labels found during each poll cycle
- [ ] Can diagnose label detection failures in production

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

- [ ] Poll cycles are logged with issue count
- [ ] Each issue check is logged with labels found
- [ ] Watch/unwatch operations are logged
- [ ] `ailang coordinator watcher-status` shows current state
- [ ] DEBUG_APPROVAL_WATCHER=1 enables verbose mode
- [ ] Existing tests pass
- [ ] E2E test with GitHub labels works without manual intervention

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
