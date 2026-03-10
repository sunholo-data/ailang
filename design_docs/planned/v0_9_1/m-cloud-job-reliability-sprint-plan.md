# Sprint Plan: M-CLOUD-JOB-RELIABILITY

**Design Doc**: `design_docs/planned/v0_9_1/m-cloud-job-reliability.md`
**Sprint ID**: M-CLOUD-JOB-RELIABILITY
**Duration**: ~2 hours (single session)
**Risk Level**: Low
**Estimated LOC**: ~180

## Analysis Summary

### Code Audit Findings

After reading the actual code, the design doc slightly overstated the issue:

1. **`coordinatorExecuteJob()` already catches `executeCloudTask()` errors** (lines 98-127 in `coordinator_cloud.go`). All 11 error paths inside `executeCloudTask()` return errors that are caught by the outer function, which publishes a "failed" completion. **This is NOT broken.**

2. **Phase 3 (failure notifications) is already implemented** — `postCompletionNotification()` in `pubsub_completion_handler.go` already handles the `"failed"` case (line 127) and includes `error_msg` in the payload (line 150).

### Actual Gaps

| Gap | Description | Fix |
|-----|-------------|-----|
| **Early exits in `coordinatorExecuteJob`** | Lines 44-64 (env vars) and 88-91 (Pub/Sub client) return before any Pub/Sub client exists. No completion can be published. | Phase 1: Defer guard with stderr fallback |
| **Panics** | Any panic in the execution flow crashes without publishing. | Phase 1: Defer with `recover()` |
| **Container never starts** | Image pull, OOM, quota failures — no Go code runs at all. | Phase 2: Stale task detector |
| **`RecoverStaleTasks` is startup-only** | Existing method only runs at daemon init, not continuously. | Phase 2: Periodic detector |

### Velocity

Recent 14-day velocity: ~20 commits, mix of features and fixes. Cloud infrastructure work (Pub/Sub, dispatch, completion handler) averages ~60 LOC/hour including tests.

## Milestones

### M1: Defer-Based Completion Guard (~60 LOC, ~45 min)

**File**: `cmd/ailang/coordinator_cloud.go`

**What**: Restructure `coordinatorExecuteJob()` so that Pub/Sub client is initialized first (even before env var validation), and a `defer` guard ensures completion is always published.

**Approach**:
- Move Pub/Sub client initialization to the top (needs only `projectID` which is the first check)
- Add `defer` with `recover()` that publishes failed completion if not already sent
- Use `sync/atomic.Bool` to prevent double-publish
- For the absolute-earliest failures (no project ID at all), log to stderr with structured format so Cloud Logging captures it

**Acceptance Criteria**:
- [ ] `coordinatorExecuteJob` always publishes a completion for any error after Pub/Sub is initialized
- [ ] Panics produce a "failed" completion with the panic message
- [ ] Early exits before Pub/Sub log to stderr in structured format
- [ ] Existing happy path unchanged
- [ ] Unit test: mock Pub/Sub, trigger each error path, verify completion published

### M2: Stale Task Detector (~100 LOC, ~45 min)

**Files**:
- `internal/coordinator/stale_task_detector.go` (new, ~80 LOC)
- `internal/coordinator/store_sqlite.go` (~10 LOC — add `GetQueuedTasks`)
- `internal/coordinator/store.go` (~3 LOC — interface method)
- `internal/coordinator/daemon.go` (~5 LOC — start detector)

**What**: Periodic goroutine that marks `queued`/`running` tasks as `failed` when they exceed their timeout.

**Approach**:
- Reuse existing `AgentRegistry` for per-agent timeout config
- Default timeout: 90 minutes (1.5x Cloud Run's 60-minute default)
- Check interval: 2 minutes
- Post failure notification to agent inbox via `msgStore`
- **Only run in cloud mode** (`COORDINATOR_MODE=cloud`) — local mode uses existing `RecoverStaleTasks` at startup

**Acceptance Criteria**:
- [ ] Tasks stuck in `queued` for >1.5x timeout are marked `failed` with descriptive error
- [ ] Failure notification posted to agent inbox with `correlation_id`
- [ ] Does NOT interfere with tasks that have recent completions
- [ ] Idempotent: re-running on already-failed task is a no-op
- [ ] Unit test: insert stale task, run detector, verify marked failed + notification posted

### M3: Tests and Verification (~20 LOC, ~15 min)

- Verify existing coordinator tests still pass
- Run `go test ./cmd/ailang/... ./internal/coordinator/...`
- Verify the mock store in `task_chain_test.go` has the new method

## Success Metrics

- [ ] `go test ./internal/coordinator/...` passes
- [ ] `go test ./cmd/ailang/...` passes
- [ ] No task can remain `queued` indefinitely in cloud mode
- [ ] Every failure path in `coordinatorExecuteJob` produces a completion or stderr log

## Dependencies

- None — all changes are additive, no breaking interface changes

## Notes

- Phase 3 from the design doc (structured error notifications) is **already implemented** in commit `1880e03e`. No work needed.
- Phase 4 (Cloud Run Job Status Watcher) remains deferred per the design doc.
- The `RecoverStaleTasks` at startup handles local mode. The new detector handles cloud mode's ongoing operation.
