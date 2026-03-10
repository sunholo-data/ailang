# M-CLOUD-JOB-RELIABILITY: Silent Failure Detection & Recovery

**Status**: Planned
**Priority**: CRITICAL
**Version**: v0.9.1
**Triggered by**: First end-to-end website-builder test (2026-03-10)
**Message ID**: `e63a4ef5-3e11-42a2-995d-8b01bec15e8d`

## Problem Statement

During the first end-to-end test of the website-builder flow, the Cloud Run Job failed because the target repo (`sunholo-data/sunholo-websites`) did not exist. The failure was **completely silent** — the coordinator never learned the job failed, the portal polled indefinitely, and no error was surfaced anywhere.

This exposed **4 categories of silent failure** in the cloud execution path:

| # | Failure Mode | Root Cause | Impact |
|---|-------------|-----------|--------|
| 1 | Clone failure exits before completion | `executeCloudTask()` returns error before `PublishCompletion()` | Task stuck in `queued` forever |
| 2 | No job status polling | Coordinator fires-and-forgets Cloud Run Jobs | Crashed/OOM-killed jobs invisible |
| 3 | Portal polls indefinitely | Sidecar has no timeout on `pollCoordinatorMessages()` | User waits forever |
| 4 | Container failures invisible | Image pull / startup / resource failures produce no completion | Same as #1 |

### Error Path Audit: `executeCloudTask()`

The function in `cmd/ailang/coordinator_cloud.go` has **11 early-return error paths** that skip completion notification:

```
Line ~45-64:   Missing env vars (TASK_ID, AGENT_ID, WORKSPACE, PROVIDER, DIRECTIVE)
Line ~140-142: Missing AILANG_REPO_URL
Line ~147:     git clone failure         ← The actual failure in the website-builder test
Line ~155:     git checkout failure
Line ~176-179: Executor creation/execution failure
Line ~195-210: git add/commit failure
Line ~215:     git push failure
```

**Every one of these exits the process without publishing a TaskCompletion message.**

Additionally, if the Cloud Run container itself fails to start (image pull error, OOM, quota), no Go code runs at all — there is no completion message by design.

## Design Goals

1. **Every task gets a terminal status** — no task should remain `queued` indefinitely
2. **Fail loudly** — per CLAUDE.md principle, errors must be visible, not silently swallowed
3. **Bounded wait times** — both coordinator and portal must have finite timeouts
4. **Minimal infrastructure changes** — use existing Pub/Sub topics and Cloud Run APIs

## Non-Goals

- Automatic retry of failed Cloud Run Jobs (can be added later)
- Circuit breaker / provider fallback
- Alerting / PagerDuty integration (use Cloud Monitoring for that)

## Implementation Plan

### Phase 1: Defer-Based Completion Guard

**Goal**: Guarantee that `executeCloudTask()` always publishes a completion, even on panic or early error exit.

**File**: `cmd/ailang/coordinator_cloud.go`

Wrap the entire function body in a defer that catches any uncaught exit and publishes a `failed` completion:

```go
func executeCloudTask() error {
    taskID := os.Getenv("AILANG_TASK_ID")
    agentID := os.Getenv("AILANG_AGENT_ID")

    // Completion guard — runs on ANY exit (error, panic, or success)
    var completionSent atomic.Bool
    defer func() {
        if completionSent.Load() {
            return // Normal path already sent completion
        }
        // Something exited before we could send a proper completion
        r := recover()
        errMsg := "unknown failure"
        if r != nil {
            errMsg = fmt.Sprintf("panic: %v", r)
        }
        publishFailedCompletion(taskID, agentID, errMsg)
    }()

    // ... existing logic ...

    // On success path:
    publishCompletion(taskID, agentID, "completed", output)
    completionSent.Store(true)
    return nil
}
```

**Key design decisions:**
- Uses `sync/atomic.Bool` to avoid double-publish (normal success path sets it before defer runs)
- `recover()` in defer catches panics from any library code (git, executor, etc.)
- `publishFailedCompletion` is a new helper that creates a minimal `TaskCompletion` with status `failed`
- On error returns, we now call `publishFailedCompletion()` explicitly before returning

**Error wrapping pattern** — replace each early return with:

```go
// BEFORE (silent failure):
if err != nil {
    return fmt.Errorf("git clone failed: %w", err)
}

// AFTER (loud failure):
if err != nil {
    errMsg := fmt.Sprintf("git clone failed: %v", err)
    publishFailedCompletion(taskID, agentID, errMsg)
    completionSent.Store(true)
    return fmt.Errorf(errMsg)
}
```

The defer guard is a **safety net** — we explicitly publish on each known error path, and the defer catches anything we missed (panics, os.Exit from libraries, etc.).

**New helper function:**

```go
func publishFailedCompletion(taskID, agentID, errMsg string) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    completion := &TaskCompletion{
        TaskID:    taskID,
        AgentID:   agentID,
        Status:    "failed",
        Error:     errMsg,
        Timestamp: time.Now(),
    }

    // Best-effort publish — if Pub/Sub is also down, the stale task
    // detector (Phase 2) will catch it
    if err := publishToPubSub(ctx, completion); err != nil {
        log.Printf("ERROR: failed to publish failure completion for task %s: %v", taskID, err)
        // Log to stderr so Cloud Logging captures it even if Pub/Sub is dead
        fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|error=%s\n",
            taskID, agentID, errMsg)
    }
}
```

**Lines changed**: ~40 in `coordinator_cloud.go`

### Phase 2: Stale Task Detector

**Goal**: Catch tasks that remain `queued` beyond their timeout — the safety net for when Phase 1's Pub/Sub publish also fails, or when the container never starts.

**File**: New file `internal/coordinator/stale_task_detector.go`

```go
type StaleTaskDetector struct {
    store         Store
    agentRegistry *AgentRegistry
    msgStore      messaging.MessageStore
    logger        *log.Logger
    interval      time.Duration  // How often to check (default: 2 min)
}

func (d *StaleTaskDetector) Run(ctx context.Context) {
    ticker := time.NewTicker(d.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            d.detectAndMarkStale(ctx)
        }
    }
}

func (d *StaleTaskDetector) detectAndMarkStale(ctx context.Context) {
    staleTasks, err := d.store.GetStaleTasks(ctx)
    if err != nil {
        d.logger.Printf("stale detector: query error: %v", err)
        return
    }

    for _, task := range staleTasks {
        timeout := d.getTaskTimeout(task)
        if time.Since(task.QueuedAt) <= timeout {
            continue
        }

        errMsg := fmt.Sprintf("task timed out: no completion received within %v of being queued", timeout)
        d.logger.Printf("stale detector: marking task %s as failed: %s", task.ID, errMsg)

        if err := d.store.MarkTaskFailed(ctx, task.ID, errMsg); err != nil {
            d.logger.Printf("stale detector: failed to mark task %s: %v", task.ID, err)
            continue
        }

        // Post failure notification to agent inbox (for sidecar polling)
        d.postFailureNotification(ctx, task, errMsg)
    }
}
```

**Store query** — add to `internal/coordinator/store.go` interface:

```go
// GetStaleTasks returns tasks in "queued" or "running" status
// ordered by queued_at ascending (oldest first)
GetStaleTasks(ctx context.Context) ([]TaskRecord, error)
```

**SQLite implementation:**

```sql
SELECT * FROM tasks
WHERE status IN ('queued', 'running')
ORDER BY queued_at ASC
```

**Timeout resolution** (priority order):
1. Agent-specific timeout from config (`agent.Timeout`)
2. Default: 60 minutes (matches Cloud Run Job default timeout)
3. Multiply by 1.5x safety margin (allow for startup overhead)

**Integration** — start in daemon's cloud mode startup (`daemon.go`):

```go
if d.isCloudMode() {
    detector := NewStaleTaskDetector(d.taskStore, d.agentRegistry, d.msgStore, d.logger)
    go detector.Run(d.ctx)
}
```

**Lines changed**: ~120 new file, ~5 in `store.go`, ~10 in `store_sqlite.go`, ~5 in `daemon.go`

### Phase 3: Structured Error in Completion Notification

**Goal**: When the CompletionHandler receives a `failed` completion, post a message to the agent inbox with the error details so the sidecar (and portal) can display a specific error instead of timing out.

**File**: `internal/coordinator/pubsub_completion_handler.go`

The completion handler already posts inbox messages for successful completions. Extend the `failed` case:

```go
case "failed":
    if err := h.taskStore.MarkTaskFailed(ctx, completion.TaskID, completion.Error); err != nil {
        h.logger.Printf("failed to mark task %s as failed: %v", completion.TaskID, err)
    }

    // Post failure notification to agent inbox
    if h.msgStore != nil {
        failureMsg := &messaging.InboxMessage{
            FromAgent:     completion.AgentID,
            ToInbox:       completion.AgentID,
            MessageType:   "completion",
            Title:         fmt.Sprintf("Task %s: failed", completion.TaskID),
            Payload:       marshalFailurePayload(completion),
            CorrelationID: task.MessageID,
        }
        h.msgStore.InsertInboxMessage(ctx, failureMsg)
    }
```

The `marshalFailurePayload` includes:

```json
{
    "task_id": "task-abc12345",
    "status": "failed",
    "error": "git clone failed: repository not found",
    "timestamp": "2026-03-10T20:31:38Z"
}
```

**Sidecar impact**: The sidecar's `pollCoordinatorMessages()` already filters by `correlation_id`. A failed completion message arrives in the same inbox with the same correlation — the sidecar just needs to check `status` in the payload to show an error instead of "building...".

**Lines changed**: ~20 in `pubsub_completion_handler.go`

### Phase 4 (Deferred): Cloud Run Job Status Watcher

**Goal**: Actively monitor Cloud Run Job execution status after dispatch, detecting infrastructure-level failures (image pull, OOM, quota) that prevent any Go code from running.

This is **deferred** because:
- Phase 1 (defer guard) + Phase 2 (stale detector) cover 95% of failure modes
- Job status polling requires Cloud Run Admin API calls (cost + quota implications)
- The stale detector already catches the "job never started" case via timeout

**When to implement**: If we see recurring failures where:
- Jobs fail at the infrastructure level (not in Go code)
- The 60-minute stale detection timeout is too slow for user-facing flows
- We need sub-minute failure detection

**Sketch** (for future reference):

```go
type JobStatusWatcher struct {
    jobsClient *run.JobsClient   // Cloud Run Admin API
    store      Store
    interval   time.Duration      // 30 seconds
}

// After dispatch, register job execution for monitoring
func (w *JobStatusWatcher) Watch(taskID, executionName string) {
    // executionName from dispatch response: projects/P/locations/L/jobs/J/executions/E
    w.pending.Store(taskID, executionName)
}

// Periodic check
func (w *JobStatusWatcher) check(ctx context.Context) {
    w.pending.Range(func(taskID, execName any) bool {
        exec, err := w.jobsClient.GetExecution(ctx, execName.(string))
        if err != nil { return true }

        switch exec.GetTerminalCondition().GetType() {
        case "Completed":
            // Normal — completion handler will handle via Pub/Sub
            w.pending.Delete(taskID)
        case "Failed", "Cancelled":
            // Infrastructure failure — publish synthetic completion
            publishFailedCompletion(taskID.(string), "", exec.GetTerminalCondition().GetMessage())
            w.pending.Delete(taskID)
        }
        return true
    })
}
```

## File Change Summary

| Phase | File | Change | Lines |
|-------|------|--------|-------|
| 1 | `cmd/ailang/coordinator_cloud.go` | Defer guard + explicit error completions | ~40 |
| 2 | `internal/coordinator/stale_task_detector.go` | New file: periodic stale task detection | ~120 |
| 2 | `internal/coordinator/store.go` | Add `GetStaleTasks()` to interface | ~5 |
| 2 | `internal/coordinator/store_sqlite.go` | Implement `GetStaleTasks()` | ~10 |
| 2 | `internal/coordinator/daemon.go` | Start detector in cloud mode | ~5 |
| 3 | `internal/coordinator/pubsub_completion_handler.go` | Post failure notifications to inbox | ~20 |

**Total: ~200 lines across 6 files** (Phases 1-3)

## Verification Plan

### Unit Tests

1. **Defer guard test**: Mock `publishToPubSub`, trigger each error path in `executeCloudTask()`, verify completion published with correct error message
2. **Stale detector test**: Insert tasks with old `queued_at` timestamps, run detector, verify they're marked `failed`
3. **Completion handler test**: Send `failed` completion, verify inbox message posted with `correlation_id`

### Integration Tests (dev environment)

1. **Missing repo test**: Dispatch website-builder with non-existent repo
   - Verify: task marked `failed` within 30 seconds (not 60 minutes)
   - Verify: failure message in `website-builder` inbox with error details
   - Verify: sidecar receives failure response when polling

2. **Container crash test**: Deploy with intentionally bad image tag
   - Verify: stale detector marks task `failed` after timeout
   - Verify: failure notification posted

3. **Happy path regression**: Normal website-builder flow still works
   - Verify: successful completion still posted with `correlation_id`
   - Verify: stale detector does NOT interfere with normal tasks

### Acceptance Criteria

- [ ] No task remains in `queued` status for more than 1.5x its configured timeout
- [ ] Every failed task has an error message in the `error` field
- [ ] Every failed task has a corresponding inbox message for sidecar polling
- [ ] Panic in executor code still produces a completion (defer guard)
- [ ] Existing tests pass (`go test ./internal/coordinator/... ./cmd/ailang/...`)

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Double-publish (defer + explicit) | Low | `atomic.Bool` guard prevents it |
| Pub/Sub down during failure publish | Low | Stale detector is independent fallback |
| Stale detector false positive | Very Low | 1.5x timeout margin; only marks `queued`/`running` |
| Race between completion and stale detection | Low | `MarkTaskFailed` is idempotent if already completed |

## Relationship to Other Design Docs

- **M-CLOUD-DISPATCH** (implemented v0.9.0): Established the dispatch flow this doc improves
- **M-CLOUD-HEALTH** (implemented v0.9.0): Health endpoints complement but don't replace active monitoring
- **M-DASHBOARD-PUBSUB-EVENTS** (planned v0.9.1): Dashboard streaming will show failure events once wired
- **Website builder plan** (`expressive-strolling-turing.md`): This doc fixes the gap discovered during testing
