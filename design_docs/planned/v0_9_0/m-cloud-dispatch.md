# M-CLOUD-DISPATCH: Cloud Run Job Dispatch & Cloud Logging

**Status**: Planned
**Target**: v0.9.0
**Priority**: P0 (High) — Blocks end-to-end cloud task execution
**Estimated**: 3 hours
**Dependencies**: M-CLOUD-E2E (implemented), M-CLOUD-PUSH (implemented)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Tooling/infrastructure change — does not modify AILANG language semantics.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics change |
| A2: Replayability | +1 | Cloud logs now captured in Cloud Logging for replay |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | +1 | Dispatch interface requires explicit executor config |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Enables 24/7 autonomous task execution without laptop |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Cloud Run Job executions tracked with cost attribution |
| A10: Composability | +1 | Dispatcher interface composes with future executors |
| A11: Structured Failure | +1 | Job failures are typed and surfaced via completion handler |
| A12: System Boundary | +1 | Coordinator → Job boundary is explicit and auditable |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — IAM roles are explicit
- [x] A7 (Machines First): Enables machine-autonomous operation

## Problem Statement

Two blockers prevent end-to-end Cloud Run task execution:

### Blocker 1: Coordinator Logging Invisible in Cloud Logging

The coordinator's logger writes exclusively to `$HOME/.ailang/logs/coordinator.log` (file-based). Cloud Run only ingests stdout/stderr into Cloud Logging. All coordinator processing logs (push handler activity, task classification, dispatch decisions) are invisible in production.

**Current code** (`internal/coordinator/daemon.go:143-148`):
```go
logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
logger := log.New(logFile, "[coordinator] ", log.LstdFlags|log.Lshortfile)
```

**Impact:** No observability in Cloud Run — debugging requires `kubectl exec` or container shell access, which Cloud Run doesn't support.

### Blocker 2: No Cloud Run Job Trigger

`dispatchTasksCloud()` in `daemon_tasks_exec.go` publishes tasks to the Pub/Sub `${prefix}-tasks` topic, but **nothing triggers the Cloud Run Job** from that topic:

- `eventarc.tf` is intentionally empty (architecture states "coordinator calls Jobs API directly")
- The coordinator code ONLY publishes to Pub/Sub — no Jobs API call exists
- The Pub/Sub message sits in the `tasks-executor` subscription forever, unprocessed

**Current code** (`internal/coordinator/daemon_tasks_exec.go:100-107`):
```go
// Publish task dispatch to Pub/Sub (triggers Cloud Run Job via Eventarc)
if err := d.pubsubPublisher.PublishTask(ctx, task.ID, task.AgentID, task.Workspace, provider); err != nil {
    // ...
}
d.logger.Printf("Cloud dispatch: published task %s to Pub/Sub", task.ID, task.AgentID, provider)
// ← Nothing triggers the actual job execution!
```

**Impact:** Tasks are queued forever. The entire cloud execution pipeline is broken after the dispatch step.

## Goals

**Primary Goal:** Close the last mile — make `dispatchTasksCloud()` actually trigger Cloud Run Job execution, with a design that allows swapping executors in the future.

**Success Metrics:**
- Task published → Cloud Run Job starts within 5 seconds
- Coordinator logs visible in Cloud Logging (stdout/stderr)
- Executor backend is swappable via interface (Cloud Run Jobs today, K8s/Lambda/etc. tomorrow)
- All existing tests pass, 4+ new tests

## Solution Design

### Architecture Decision: Dispatcher Interface

**User preference: maximum decoupling.** The coordinator should not know or care whether tasks execute on Cloud Run Jobs, Kubernetes Jobs, AWS Lambda, or a local process. This rules out hard-coding the Cloud Run Jobs API in the coordinator.

#### Options Evaluated

| Option | Decoupling | Latency | Env Var Control | Complexity |
|--------|-----------|---------|-----------------|------------|
| **A: Eventarc** (Pub/Sub → Job) | Best (zero code) | ~5-10s | None (message data only) | Lowest |
| **B: Direct Jobs API** | Poor (imports `cloud.google.com/go/run`) | ~1s | Full (per-execution overrides) | Medium |
| **C: Hybrid** (Pub/Sub + Jobs API) | Poor (still imports Jobs API) | ~1s | Full | Highest |
| **D: Dispatcher Interface** | Best (interface + implementations) | ~1-2s | Full | Medium |

#### Recommended: Option D — Dispatcher Interface

Define a `CloudDispatcher` interface in the coordinator package. The coordinator calls `dispatcher.Dispatch(task)` without knowing the backend. Implementations are pluggable:

```go
// CloudDispatcher triggers remote task execution.
// Implementations are backend-specific (Cloud Run Jobs, K8s, etc.)
type CloudDispatcher interface {
    // Dispatch triggers execution of a task on the remote backend.
    // The task is already persisted in the task store — the dispatcher
    // only needs to trigger execution with the given parameters.
    Dispatch(ctx context.Context, params DispatchParams) error
}

type DispatchParams struct {
    TaskID    string
    AgentID   string
    Workspace string
    Provider  string  // "claude" or "gemini"
    Directive string  // Task prompt (optional — job can fetch from Firestore)
    RepoURL   string  // Git repo URL
    Branch    string  // Base branch (default: "dev")
}
```

**First implementation: `CloudRunJobDispatcher`** — calls Cloud Run Jobs API with per-execution env var overrides. Lives in a separate package (`internal/dispatch/cloudrun/`) so the coordinator never imports `cloud.google.com/go/run` directly.

**Pub/Sub publish is kept** for audit trail / event streaming, but is NOT the dispatch mechanism.

### Overview

```
dispatchTasksCloud()
    │
    ├── 1. Mark task as "queued" in store
    ├── 2. Publish to Pub/Sub tasks topic (audit/notification — NOT dispatch)
    └── 3. Call d.cloudDispatcher.Dispatch(ctx, params)
            │
            ▼
        CloudRunJobDispatcher.Dispatch()
            │
            ├── Build RunJobRequest with env var overrides:
            │     AILANG_TASK_ID, AILANG_AGENT_ID, AILANG_WORKSPACE,
            │     AILANG_PROVIDER, AILANG_DIRECTIVE, AILANG_REPO_URL, AILANG_BRANCH
            │
            └── client.RunJob(ctx, request)
                    │
                    ▼
                Cloud Run Job starts → execute-job → completion → Pub/Sub
```

### Components

1. **`CloudDispatcher` interface** (`internal/coordinator/cloud_dispatcher.go`)
   - Interface definition + `DispatchParams` struct
   - ~30 LOC

2. **`CloudRunJobDispatcher`** (`internal/dispatch/cloudrun/dispatcher.go`)
   - Implements `CloudDispatcher` using Cloud Run Jobs Admin API
   - Constructs job name from config: `projects/{project}/locations/{region}/jobs/{prefix}-agent-executor`
   - Sets per-execution env var overrides via `RunJobRequest.Overrides.ContainerOverrides`
   - ~80 LOC

3. **Cloud logging MultiWriter** (`internal/coordinator/daemon.go`)
   - In cloud mode, logger writes to both file AND `os.Stderr`
   - ~5 LOC change

4. **Wire dispatcher into daemon** (`internal/coordinator/daemon_tasks_init.go`, `daemon.go`)
   - Create `CloudRunJobDispatcher` when `COORDINATOR_MODE=cloud`
   - Store on `Daemon` struct as `cloudDispatcher CloudDispatcher`
   - ~15 LOC

5. **Update `dispatchTasksCloud()`** (`internal/coordinator/daemon_tasks_exec.go`)
   - After Pub/Sub publish, call `d.cloudDispatcher.Dispatch()`
   - If dispatch fails, reset task to pending
   - ~10 LOC

### Implementation Plan

**Phase 1: Interface + Cloud Logging** (~1 hour)
- [ ] Define `CloudDispatcher` interface in `internal/coordinator/cloud_dispatcher.go`
- [ ] Add `io.MultiWriter(logFile, os.Stderr)` in cloud mode in `daemon.go`
- [ ] Add `cloudDispatcher CloudDispatcher` field to `Daemon` struct

**Phase 2: Cloud Run Jobs Implementation** (~1.5 hours)
- [ ] Create `internal/dispatch/cloudrun/dispatcher.go` with `CloudRunJobDispatcher`
- [ ] Add `cloud.google.com/go/run/apiv2` dependency
- [ ] Create `internal/dispatch/cloudrun/dispatcher_test.go` with unit tests
- [ ] Wire into `daemon_tasks_init.go` (create dispatcher when cloud mode)
- [ ] Update `dispatchTasksCloud()` to call dispatcher after Pub/Sub publish

**Phase 3: Documentation & Testing** (~30 min)
- [ ] Update CHANGELOG
- [ ] Update CLAUDE.md env var table (add `AILANG_CLOUD_REGION`)
- [ ] Run full test suite

### Files to Modify/Create

**New files:**
- `internal/coordinator/cloud_dispatcher.go` — Interface + DispatchParams (~30 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — Cloud Run Jobs implementation (~80 LOC)
- `internal/dispatch/cloudrun/dispatcher_test.go` — Unit tests (~60 LOC)

**Modified files:**
- `internal/coordinator/daemon.go` — MultiWriter for cloud logging, dispatcher field (~10 LOC)
- `internal/coordinator/daemon_tasks_init.go` — Create dispatcher in cloud mode (~15 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — Call dispatcher after Pub/Sub publish (~10 LOC)

**Total: ~205 LOC**

## Examples

### Example 1: Cloud Logging (Before/After)

**Before** (invisible in Cloud Logging):
```go
logger := log.New(logFile, "[coordinator] ", log.LstdFlags|log.Lshortfile)
// All logs go to file only — Cloud Run can't see them
```

**After** (visible in Cloud Logging):
```go
var writer io.Writer = logFile
if os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud {
    writer = io.MultiWriter(logFile, os.Stderr)
}
logger := log.New(writer, "[coordinator] ", log.LstdFlags|log.Lshortfile)
```

### Example 2: Dispatcher Interface Usage

**Coordinator dispatches a task (doesn't know about Cloud Run):**
```go
// In dispatchTasksCloud():
if d.cloudDispatcher != nil {
    params := DispatchParams{
        TaskID:    task.ID,
        AgentID:   task.AgentID,
        Workspace: task.Workspace,
        Provider:  provider,
        Directive: task.Directive,
        RepoURL:   repoURL,
        Branch:    branch,
    }
    if err := d.cloudDispatcher.Dispatch(d.ctx, params); err != nil {
        d.logger.Printf("Failed to dispatch task %s: %v", task.ID, err)
        _ = d.taskStore.ResetTaskToPending(d.ctx, task.ID)
        continue
    }
}
```

**Cloud Run Jobs implementation (separate package):**
```go
func (d *Dispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
    jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s-agent-executor",
        d.projectID, d.region, d.prefix)

    envOverrides := []*runpb.EnvVar{
        {Name: "AILANG_TASK_ID", Value: params.TaskID},
        {Name: "AILANG_AGENT_ID", Value: params.AgentID},
        {Name: "AILANG_WORKSPACE", Value: params.Workspace},
        {Name: "AILANG_PROVIDER", Value: params.Provider},
        {Name: "AILANG_DIRECTIVE", Value: params.Directive},
        {Name: "AILANG_REPO_URL", Value: params.RepoURL},
        {Name: "AILANG_BRANCH", Value: params.Branch},
    }

    _, err := d.client.RunJob(ctx, &runpb.RunJobRequest{
        Name: jobName,
        Overrides: &runpb.RunJobRequest_Overrides{
            ContainerOverrides: []*runpb.RunJobRequest_Overrides_ContainerOverride{{
                Env: envOverrides,
            }},
        },
    })
    return err
}
```

### Example 3: Swapping Executors (Future)

```go
// Future: Kubernetes Jobs dispatcher
type K8sJobDispatcher struct {
    clientset *kubernetes.Clientset
    namespace string
}

func (d *K8sJobDispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
    // Create K8s Job with env vars...
}

// Future: Local process dispatcher (for testing)
type LocalDispatcher struct {
    binaryPath string
}

func (d *LocalDispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
    cmd := exec.CommandContext(ctx, d.binaryPath, "coordinator", "execute-job")
    cmd.Env = append(os.Environ(),
        "AILANG_TASK_ID="+params.TaskID,
        // ...
    )
    return cmd.Start()
}
```

## Success Criteria

- [ ] `dispatchTasksCloud()` triggers Cloud Run Job execution via `CloudDispatcher`
- [ ] Cloud Run Job starts within 5s of task being dispatched
- [ ] Coordinator logs visible in Cloud Logging (stderr) when `COORDINATOR_MODE=cloud`
- [ ] `CloudDispatcher` interface allows swapping backends without changing coordinator
- [ ] All existing tests pass
- [ ] 4+ new tests for dispatcher
- [ ] Documentation updated (CHANGELOG, CLAUDE.md)

## Testing Strategy

**Unit tests:**
- `CloudRunJobDispatcher` with mocked Cloud Run client (verify request construction, env var overrides)
- `dispatchTasksCloud()` with mock dispatcher (verify dispatch called after Pub/Sub publish)
- MultiWriter logger outputs to both file and stderr

**Integration tests (manual on Cloud Run):**
- Deploy coordinator → send message → verify Cloud Run Job starts
- Verify coordinator logs appear in Cloud Logging
- Verify task goes through full lifecycle: pending → queued → running → completed

## Non-Goals

**Not in this feature:**
- Eventarc triggers — Using direct API call via interface, not Pub/Sub-triggered jobs
- K8s/Lambda dispatchers — Future work, interface designed to support them
- Rate limiting / concurrency control — Existing `Limit: 5` in `dispatchTasksCloud()` is sufficient for now
- Job monitoring / health checks — Out of scope, Cloud Run Job handles its own lifecycle
- Retry on job failure — Pub/Sub completion handler already handles failed completions

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `COORDINATOR_MODE` | `cloud` enables dispatcher + MultiWriter logging | `local` |
| `AILANG_CLOUD_PROJECT` | GCP project for Jobs API | (required in cloud mode) |
| `AILANG_CLOUD_REGION` | GCP region for Cloud Run Jobs | `europe-west1` |
| `AILANG_TOPIC_PREFIX` | Constructs job name: `{prefix}-agent-executor` | `ailang` |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `cloud.google.com/go/run` adds dependency weight | Low | Isolated in `internal/dispatch/cloudrun/` — only imported in cloud mode |
| Jobs API rate limits | Low | Batch dispatch already limited to 5 tasks per cycle |
| Job name mismatch with Terraform | High | Job name follows same `{prefix}-agent-executor` pattern as `cloud_run_jobs.tf` |
| Circular import (coordinator ↔ dispatch) | Medium | Interface in coordinator, implementation in separate package |

## Related Documents

<!-- Auto-populated by Ollama neural search on "cloud dispatch" -->

**Implemented (informs design):**
- [design_docs/implemented/v0_9_0/m-cloud-e2e.md](../../implemented/v0_9_0/m-cloud-e2e.md) — Cloud E2E wiring (this doc closes the last gap)
- [design_docs/implemented/v0_7_0/m-otel-cross-process-linking.md](../../implemented/v0_7_0/m-otel-cross-process-linking.md) — Cross-process trace linking

**Planned (context):**
- [design_docs/planned/v0_8_2/m-cloud-infra.md](../v0_8_2/m-cloud-infra.md) — Original cloud architecture design
- [design_docs/planned/v0_9_0/m-cloud-e2e.md](m-cloud-e2e.md) — E2E message flow design
- [design_docs/planned/v0_9_0/m-cloud-health.md](m-cloud-health.md) — Health endpoints design

## References

- [Design Axioms](/docs/references/axioms)
- [Cloud Run Jobs API](https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.jobs/run)
- `ailang-multivac/terraform/cloud_run_jobs.tf` — Job definition
- `ailang-multivac/terraform/iam.tf:68-73` — `roles/run.developer` for coordinator SA
- `ailang-multivac/terraform/eventarc.tf` — Empty, documents "coordinator calls Jobs API directly"

## Future Work

- **K8s Job dispatcher** — For self-hosted deployments without GCP
- **Local process dispatcher** — For development/testing without cloud
- **Lambda dispatcher** — For AWS deployments
- **Rate limiting** — Token bucket or leaky bucket for dispatch rate
- **Job monitoring** — Poll Cloud Run Job status for stuck/long-running jobs
- **Cost attribution** — Track Cloud Run Job cost per task

---

**Document created**: 2026-03-06
**Last updated**: 2026-03-06
