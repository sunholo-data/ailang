# M-CLOUD-PROGRESS-TRACKING: Cloud Execution Visibility & Cost Controls

**Status**: Planned
**Target**: v0.9.2
**Priority**: P0 (High — blocks laptop→cloud migration)
**Estimated**: 2–3 days (~20h)
**Dependencies**: M-CLOUD-E2E (v0.9.0, implemented), M-CLOUD-DISPATCH (v0.9.0, implemented), M-CLOUD-JOB-RELIABILITY (v0.9.1, implemented)
**Source**: ailang-multivac message `msg_20260313_132028_56089900` — "Cloud Tracking Gaps"
**Author**: Claude + Mark
**Created**: 2026-03-13

---

## Executive Summary

The cloud coordinator can dispatch tasks, detect stale jobs, and report completions — but execution is **blind** between dispatch and completion. A 30-minute opus task shows "running" with zero feedback. Cost accumulates invisibly. There is no way to abort a runaway task before it finishes.

This doc covers 4 gaps that close the visibility loop:

| # | Gap | Priority | Status |
|---|-----|----------|--------|
| 1 | Mid-execution progress streaming (producer) | HIGH | **New work** |
| 2 | Turn-by-turn cost tracking | HIGH | **New work** |
| 3 | Per-task cost budgets & early abort | MEDIUM | **New work** (pre-execution budgets exist) |
| 4 | Google Cloud Trace integration | MEDIUM | **New work** |

**Note**: A 5th gap (dashboard cloud-mode API endpoints) is already covered by two existing planned docs:
- [M-DASHBOARD-PUBSUB-EVENTS](../v0_9_1/m-dashboard-pubsub-events.md) — SSE consumer of progress events
- [M-CLOUD-OBSERVATORY](../v0_10_0/m-cloud-observatory.md) — Firestore backend for exec/span hierarchy endpoints

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics change |
| A2: Replayability | +1 | Progress events enable execution replay |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | +1 | Cost budgets are explicit, no silent fallbacks |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | Single-threaded per Cloud Run Job |
| A7: Machines First | +2 | Enables autonomous operation with cost safety nets |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +3 | Real-time cost tracking, budgets, early abort |
| A10: Composability | 0 | Uses existing interfaces |
| A11: Structured Failure | +1 | Budget exceeded = typed failure, not silent overrun |
| A12: System Boundary | +1 | Cloud Trace links coordinator ↔ job spans |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: No implicit nondeterminism
- [x] A3: No hidden side effects
- [x] A4: Cost limits are explicit, never silently ignored
- [x] A11: Budget exceeded → explicit "failed" status with error message

---

## What Already Works

Before describing gaps, it's worth documenting what's already in place (confirmed working as of 2026-03-13):

| Capability | Status | Component |
|-----------|--------|-----------|
| Task lifecycle state machine | Working | `daemon_tasks_exec.go` |
| Pub/Sub completion handling with full metrics | Working | `pubsub_completion_handler.go` |
| Stale task detector (2-min polling) | Working | `stale_task_detector.go` |
| Defer-based completion guard (fail-loud) | Working | `coordinator_cloud.go` |
| Multi-agent chain tracking (ChainID/StageID) | Working | `daemon_tasks_chain.go` |
| Message correlation (inbox → task → completion) | Working | Coordinator E2E flow |
| KMS encryption for API keys | Working | `coordinator/kms.go`, `claude/kms.go` |
| PubSubBroadcaster (publishes events to topic) | Working | `pubsub_broadcaster.go` |
| Pre-execution budget checking | Working | `daemon_tasks_budget.go` |
| cloudEventHandler (logs to stderr in Cloud Run) | Working | `coordinator_cloud.go` |
| OTEL tracing with context propagation | Working | `internal/telemetry/` |

---

## GAP 1: Mid-Execution Progress Streaming (Producer)

### Problem

The Cloud Run Job executor captures events internally via `cloudEventHandler` (TurnStart, Text, ToolUse, ToolResult, TurnEnd) but only logs them to stderr for Cloud Logging. Events are never published to the `ailang-events` Pub/Sub topic. The dashboard shows "running" for 30+ minutes with zero feedback.

The `PubSubBroadcaster` already exists and works — it publishes `TaskStreamEvent` JSON to the events topic. But it's only wired up in **local mode** (via `daemon_tasks_init.go`). The Cloud Run Job executor in `coordinator_cloud.go` uses `cloudEventHandler` which writes to stderr only.

### Current Flow (Cloud Run Job)

```
executor.ExecuteStreaming(ctx, task, &cloudEventHandler{})
    │
    ├── OnTurnStart → log.Printf to stderr
    ├── OnText      → log.Printf to stderr (truncated)
    ├── OnToolUse   → log.Printf to stderr
    ├── OnToolResult → log.Printf to stderr (truncated)
    └── OnTurnEnd   → log.Printf to stderr

    → Events visible in Cloud Logging only
    → Dashboard shows "running" with no progress
```

### Target Flow

```
executor.ExecuteStreaming(ctx, task, &cloudEventHandler{broadcaster: pubsubBroadcaster})
    │
    ├── OnTurnStart → log.Printf to stderr + broadcaster.Broadcast()
    ├── OnText      → log.Printf to stderr + broadcaster.Broadcast()
    ├── OnToolUse   → log.Printf to stderr + broadcaster.Broadcast()
    ├── OnToolResult → log.Printf to stderr + broadcaster.Broadcast()
    └── OnTurnEnd   → log.Printf to stderr + broadcaster.Broadcast()

    → Events in Cloud Logging AND Pub/Sub ailang-events topic
    → Dashboard receives via subscription (M-DASHBOARD-PUBSUB-EVENTS)
```

### Implementation

**File**: `cmd/ailang/coordinator_cloud.go`

**Step 1**: Create a `PubSubBroadcaster` in `executeCloudTask()`:

```go
// After initializing Pub/Sub client (already exists for completion publishing):
broadcaster := coordinator.NewPubSubBroadcaster(publisher, workspace, logger)
```

**Step 2**: Add broadcaster to `cloudEventHandler`:

```go
type cloudEventHandler struct {
    taskID      string
    broadcaster *coordinator.PubSubBroadcaster // NEW
}
```

**Step 3**: In each event method, broadcast after logging:

```go
func (h *cloudEventHandler) OnTurnStart(turnNum int) {
    log.Printf("[task=%s] Turn %d started", h.taskID, turnNum)
    if h.broadcaster != nil {
        h.broadcaster.Broadcast(&websocket.TaskStreamEvent{
            TaskID:     h.taskID,
            StreamType: websocket.TaskStreamStatus,
            Text:       fmt.Sprintf("Turn %d started", turnNum),
        })
    }
}
```

**Step 4**: Same pattern for OnText, OnToolUse, OnToolResult, OnTurnEnd.

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `cmd/ailang/coordinator_cloud.go` | Add broadcaster to handler, broadcast in each event | ~40 |

### Prerequisite

The Cloud Run Job needs Pub/Sub publisher access. The Terraform service account already has `roles/pubsub.publisher` on the events topic (confirmed in `ailang-multivac/terraform/iam.tf`). No infra changes needed.

---

## GAP 2: Turn-by-Turn Cost Tracking

### Problem

Cost is only calculated at completion. For expensive tasks (opus model, 30+ turns), there's no way to see cost accumulating. This matters for:
- Dashboard operators watching task progress
- Cost budget enforcement (GAP 3)
- Early abort decisions

### Current State

The executor already tracks per-turn token usage internally (input/output tokens per API call). The completion message includes total cost. But per-turn cost is never surfaced during execution.

### Implementation

**Step 1**: Add running cost fields to progress events.

**File**: `internal/websocket/types.go` (or wherever `TaskStreamEvent` is defined)

```go
type TaskStreamEvent struct {
    // ... existing fields ...
    RunningCostUSD  float64 `json:"running_cost_usd,omitempty"`
    RunningTokens   int     `json:"running_tokens,omitempty"`
    TurnNumber      int     `json:"turn_number,omitempty"`
}
```

**Step 2**: Track running cost in `cloudEventHandler`.

**File**: `cmd/ailang/coordinator_cloud.go`

```go
type cloudEventHandler struct {
    taskID        string
    broadcaster   *coordinator.PubSubBroadcaster
    runningCost   float64    // Accumulated cost
    runningTokens int        // Accumulated tokens
    turnNumber    int        // Current turn
    provider      string     // For cost calculation
}
```

**Step 3**: Update cost after each turn end.

The executor's `claude.go` returns token counts per turn. After each API call completes, the event handler gets `OnTurnEnd` — at which point we calculate incremental cost from token usage and add to running total.

**Cost calculation**: Use the existing `internal/ai/pricing.go` model pricing table.

```go
func (h *cloudEventHandler) OnTurnEnd(turnNum int) {
    h.turnNumber = turnNum
    // Cost is updated by the executor after each API call
    // (see step 4 — executor passes cost data to event handler)

    if h.broadcaster != nil {
        h.broadcaster.Broadcast(&websocket.TaskStreamEvent{
            TaskID:         h.taskID,
            StreamType:     websocket.TaskStreamStatus,
            Text:           fmt.Sprintf("Turn %d completed", turnNum),
            RunningCostUSD: h.runningCost,
            RunningTokens:  h.runningTokens,
            TurnNumber:     turnNum,
        })
    }
}
```

**Step 4**: Extend executor event callback to include cost data.

The executor interface may need a new method or the existing `OnTurnEnd` may need additional parameters. Two options:

- **Option A**: New callback `OnCostUpdate(inputTokens, outputTokens, costUSD)` — cleanest
- **Option B**: Add fields to existing `OnTurnEnd(turnNum int, stats TurnStats)` — simplest

Recommend **Option B** to minimize interface changes. Add an optional `TurnStats` struct:

```go
type TurnStats struct {
    InputTokens  int
    OutputTokens int
    CostUSD      float64
    Model        string
}

// Updated interface (backwards compatible — stats can be nil)
OnTurnEnd(turnNum int, stats *TurnStats)
```

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `internal/websocket/types.go` | Add RunningCostUSD, RunningTokens, TurnNumber fields | ~5 |
| `cmd/ailang/coordinator_cloud.go` | Track running cost, broadcast with cost data | ~25 |
| `internal/executor/claude/claude.go` | Pass TurnStats to OnTurnEnd | ~15 |
| `internal/coordinator/event_handler.go` | Update EventHandler interface | ~10 |

---

## GAP 3: Per-Task Cost Budgets & Early Abort

### Problem

Pre-execution budget checking exists (`daemon_tasks_budget.go`) — it blocks tasks from starting if daily budget is exceeded. But once a task starts, there's no enforcement. A runaway task with opus model could burn through the daily budget within a single execution.

### What Exists

| Feature | Status | Location |
|---------|--------|----------|
| Pre-execution daily budget check | Working | `daemon_tasks_budget.go` |
| `TaskMaxCost` config field | Exists | `agent_config.go` (never enforced mid-execution) |
| Per-provider budget limits | Working | `agent_config.go` |
| Hard limit → approval gate | Working | `daemon_tasks_budget.go` |

### What's Missing

- **Mid-execution cost check**: After each turn, compare running cost against `TaskMaxCost`
- **Abort mechanism**: Signal the executor to stop if budget exceeded
- **Failure reporting**: Publish completion with `status=failed, error='cost budget exceeded'`

### Implementation

**Step 1**: Pass budget limit to Cloud Run Job via env var.

**File**: `internal/dispatch/cloudrun/dispatcher.go`

```go
// In Dispatch(), after existing env overrides:
if params.MaxCostUSD > 0 {
    envOverrides = append(envOverrides, &runpb.EnvVar{
        Name:  "AILANG_MAX_COST_USD",
        Value: fmt.Sprintf("%.2f", params.MaxCostUSD),
    })
}
```

**File**: `internal/coordinator/cloud_dispatcher.go`

```go
type DispatchParams struct {
    // ... existing fields ...
    MaxCostUSD float64 // Per-task cost limit (0 = unlimited)
}
```

**File**: `internal/coordinator/daemon_tasks_exec.go`

```go
// When building DispatchParams, populate MaxCostUSD from agent config:
maxCost := 0.0
if agentConfig != nil {
    maxCost = agentConfig.TaskMaxCost
}
if maxCost == 0 && budgetsCfg != nil && budgetsCfg.Global != nil {
    maxCost = budgetsCfg.Global.TaskMaxCost
}
params.MaxCostUSD = maxCost
```

**Step 2**: Check budget in Cloud Run Job after each turn.

**File**: `cmd/ailang/coordinator_cloud.go`

```go
func (h *cloudEventHandler) OnTurnEnd(turnNum int, stats *TurnStats) {
    if stats != nil {
        h.runningCost += stats.CostUSD
        h.runningTokens += stats.InputTokens + stats.OutputTokens
    }

    // Budget enforcement
    maxCost := parseFloat(os.Getenv("AILANG_MAX_COST_USD"))
    if maxCost > 0 && h.runningCost > maxCost {
        log.Printf("[task=%s] COST BUDGET EXCEEDED: $%.2f > $%.2f limit",
            h.taskID, h.runningCost, maxCost)
        h.cancel() // Cancel the execution context
    }

    // ... broadcast event (from GAP 1/2) ...
}
```

The `cancel()` function is the context cancellation for the executor. Pass the cancel func when creating the handler:

```go
ctx, cancel := context.WithCancel(ctx)
handler := &cloudEventHandler{
    taskID: taskID,
    cancel: cancel,
    // ...
}
result, err := exec.ExecuteStreaming(ctx, task, handler)
```

When the context is cancelled, the executor's HTTP client will abort the current API call, and the executor loop will exit.

**Step 3**: Report budget failure in completion.

After `ExecuteStreaming` returns with a context cancellation, the existing completion guard detects the error and publishes a `failed` completion:

```go
if errors.Is(err, context.Canceled) && handler.runningCost > maxCost {
    errMsg := fmt.Sprintf("cost budget exceeded ($%.2f > $%.2f limit)", handler.runningCost, maxCost)
    publishFailedCompletion(taskID, agentID, errMsg)
    completionSent.Store(true)
    return fmt.Errorf(errMsg)
}
```

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `internal/coordinator/cloud_dispatcher.go` | Add MaxCostUSD to DispatchParams | ~3 |
| `internal/dispatch/cloudrun/dispatcher.go` | Pass AILANG_MAX_COST_USD env var | ~5 |
| `internal/coordinator/daemon_tasks_exec.go` | Populate MaxCostUSD from config | ~10 |
| `cmd/ailang/coordinator_cloud.go` | Check budget after each turn, cancel if exceeded | ~20 |

---

## GAP 4: Google Cloud Trace Integration

### Problem

OTEL traces go only to the dashboard's internal OTLP receiver. No traces appear in the GCP Cloud Trace console. When debugging Cloud Run issues, operators can't correlate coordinator spans with Cloud Run Job spans in a single trace view.

### What Exists

| Component | Status | Location |
|-----------|--------|----------|
| OTEL SDK initialization | Working | `internal/telemetry/otel.go` |
| Trace context propagation (W3C) | Working | `internal/telemetry/context_propagation.go` |
| OTLP receiver (dashboard-side) | Working | `internal/observatory/otlp_receiver.go` |
| Cloud Trace API enabled | Provisioned | `ailang-multivac/terraform/apis.tf` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` env var | Set | Cloud Run service env vars |

### What's Missing

1. **Dual OTEL exporter**: Send traces to both dashboard OTLP receiver AND Google Cloud Trace
2. **Trace context propagation**: Coordinator → Cloud Run Job (via env var override `AILANG_TRACE_PARENT`)
3. **Child span creation**: Cloud Run Job picks up trace context and creates child spans

### Implementation

**Step 1**: Add Google Cloud Trace exporter alongside existing OTLP exporter.

**File**: `internal/telemetry/otel.go`

```go
import (
    texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
)

func initTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
    // Existing OTLP exporter (to dashboard)
    otlpExporter, err := otlptracegrpc.New(ctx, ...)

    // NEW: Cloud Trace exporter (if running on GCP)
    var exporters []sdktrace.SpanExporter
    exporters = append(exporters, otlpExporter)

    if project := os.Getenv("AILANG_CLOUD_PROJECT"); project != "" {
        cloudExporter, err := texporter.New(texporter.WithProjectID(project))
        if err == nil {
            exporters = append(exporters, cloudExporter)
        } else {
            log.Printf("Warning: Cloud Trace exporter init failed: %v", err)
        }
    }

    // Multi-exporter via composite SpanProcessor
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(otlpExporter),
    )
    if len(exporters) > 1 {
        tp = sdktrace.NewTracerProvider(
            sdktrace.WithBatcher(exporters[0]),
            sdktrace.WithBatcher(exporters[1]),
        )
    }
    return tp, nil
}
```

**Step 2**: Propagate trace context from coordinator to Cloud Run Job.

**File**: `internal/dispatch/cloudrun/dispatcher.go`

```go
func (d *Dispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
    // Extract W3C traceparent from current span context
    traceparent := telemetry.ExtractTraceparent(ctx)

    envOverrides := []*runpb.EnvVar{
        // ... existing overrides ...
    }

    if traceparent != "" {
        envOverrides = append(envOverrides, &runpb.EnvVar{
            Name:  "AILANG_TRACE_PARENT",
            Value: traceparent,
        })
    }
    // ...
}
```

**File**: `internal/telemetry/context_propagation.go` (add helper)

```go
// ExtractTraceparent returns the W3C traceparent header value from the current span.
func ExtractTraceparent(ctx context.Context) string {
    sc := trace.SpanFromContext(ctx).SpanContext()
    if !sc.IsValid() {
        return ""
    }
    return fmt.Sprintf("00-%s-%s-%s",
        sc.TraceID().String(), sc.SpanID().String(), sc.TraceFlags().String())
}

// InjectTraceparent creates a context with the parent trace from AILANG_TRACE_PARENT env var.
func InjectTraceparent(ctx context.Context) context.Context {
    traceparent := os.Getenv("AILANG_TRACE_PARENT")
    if traceparent == "" {
        return ctx
    }
    // Parse W3C traceparent and create remote span context
    sc, err := parseTraceparent(traceparent)
    if err != nil {
        return ctx
    }
    return trace.ContextWithRemoteSpanContext(ctx, sc)
}
```

**Step 3**: Cloud Run Job picks up trace context.

**File**: `cmd/ailang/coordinator_cloud.go`

```go
func executeCloudTask() error {
    ctx := context.Background()

    // Pick up parent trace from coordinator
    ctx = telemetry.InjectTraceparent(ctx)

    // Create child span for this Cloud Run Job execution
    ctx, span := tracer.Start(ctx, "cloud_job.execute",
        trace.WithAttributes(
            attribute.String("task_id", taskID),
            attribute.String("agent_id", agentID),
        ))
    defer span.End()

    // ... rest of execution ...
}
```

### New Dependency

```
go get github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace
```

### Files Changed

| File | Change | LOC |
|------|--------|-----|
| `internal/telemetry/otel.go` | Add Cloud Trace exporter (dual export) | ~25 |
| `internal/telemetry/context_propagation.go` | Add ExtractTraceparent, InjectTraceparent | ~30 |
| `internal/dispatch/cloudrun/dispatcher.go` | Pass AILANG_TRACE_PARENT env var | ~5 |
| `cmd/ailang/coordinator_cloud.go` | Pick up trace context, create child span | ~10 |
| `go.mod` | Add opentelemetry-operations-go dependency | ~1 |

---

## Implementation Priority

```
GAP 1 (progress streaming) ─── unlocks everything else
    │
    ├── GAP 2 (cost tracking) ─── piggybacks on GAP 1 transport
    │       │
    │       └── GAP 3 (cost budgets) ─── requires GAP 2 cost data
    │
    └── M-DASHBOARD-PUBSUB-EVENTS ─── consumer side (existing planned doc)

GAP 4 (Cloud Trace) ─── independent, ops improvement
```

### Sprint Plan

**Day 1: GAPs 1 + 2** (~8h)
- Wire PubSubBroadcaster into cloudEventHandler (~2h)
- Add running cost tracking to event handler (~2h)
- Extend executor interface with TurnStats (~2h)
- Test: verify events appear in Pub/Sub topic (~2h)

**Day 2: GAPs 3 + 4** (~8h)
- Add AILANG_MAX_COST_USD to dispatch params (~1h)
- Implement mid-execution budget check with context cancellation (~3h)
- Add Cloud Trace dual exporter (~2h)
- Add trace context propagation to dispatcher (~2h)

**Day 3: Integration testing + docs** (~4h)
- End-to-end test: task dispatch → progress events → dashboard (~2h)
- End-to-end test: cost budget exceeded → task aborted (~1h)
- Verify traces in Cloud Trace console (~30min)
- Update CHANGELOG, cloud-messaging-integration guide (~30min)

---

## Total File Changes

| Phase | File | Change | LOC |
|-------|------|--------|-----|
| GAP 1 | `cmd/ailang/coordinator_cloud.go` | Wire PubSubBroadcaster into handler | ~40 |
| GAP 2 | `internal/websocket/types.go` | Add cost/token fields to TaskStreamEvent | ~5 |
| GAP 2 | `cmd/ailang/coordinator_cloud.go` | Track running cost | ~25 |
| GAP 2 | `internal/executor/claude/claude.go` | Pass TurnStats to OnTurnEnd | ~15 |
| GAP 2 | `internal/coordinator/event_handler.go` | Update EventHandler interface | ~10 |
| GAP 3 | `internal/coordinator/cloud_dispatcher.go` | Add MaxCostUSD | ~3 |
| GAP 3 | `internal/dispatch/cloudrun/dispatcher.go` | Pass AILANG_MAX_COST_USD env var | ~5 |
| GAP 3 | `internal/coordinator/daemon_tasks_exec.go` | Populate MaxCostUSD from config | ~10 |
| GAP 3 | `cmd/ailang/coordinator_cloud.go` | Budget check + cancel | ~20 |
| GAP 4 | `internal/telemetry/otel.go` | Dual Cloud Trace exporter | ~25 |
| GAP 4 | `internal/telemetry/context_propagation.go` | Extract/Inject traceparent | ~30 |
| GAP 4 | `internal/dispatch/cloudrun/dispatcher.go` | Pass AILANG_TRACE_PARENT | ~5 |
| GAP 4 | `cmd/ailang/coordinator_cloud.go` | Pick up trace context | ~10 |

**Total: ~203 LOC across 8 files + 1 new dependency**

---

## Success Criteria

- [ ] Cloud Run Job progress events appear in `ailang-events` Pub/Sub topic
- [ ] Dashboard shows turn-by-turn progress for cloud tasks (requires M-DASHBOARD-PUBSUB-EVENTS)
- [ ] Running cost visible in progress events (`running_cost_usd` field)
- [ ] Task with `TaskMaxCost=5.0` aborts when cost exceeds $5.00
- [ ] Aborted task has `status=failed` with error `"cost budget exceeded ($X.XX > $Y.YY limit)"`
- [ ] Traces appear in Google Cloud Trace console
- [ ] Coordinator span → Cloud Run Job span linked as parent→child in trace view
- [ ] All existing tests pass (`make test`)
- [ ] No regression in local mode

---

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Pub/Sub publish latency per event | Low | Events are fire-and-forget, don't block execution |
| Cost calculation inaccuracy (cached pricing) | Medium | Use same pricing table as completion handler; acceptable if ~90% accurate for budget enforcement |
| Context cancellation doesn't abort HTTP call | Low | Go's `net/http` respects context cancellation on in-flight requests |
| Cloud Trace exporter increases cold start | Low | Exporter init is <100ms; Cloud Run min-instances=1 |
| Double-publish on TurnEnd (progress + completion) | Very Low | TurnEnd is progress, completion is separate; different event types |

---

## Related Documents

### Implemented (foundation)

- [M-CLOUD-E2E](../../implemented/v0_9_0/m-cloud-e2e.md) — End-to-end cloud message flow
- [M-CLOUD-DISPATCH](../../implemented/v0_9_0/m-cloud-dispatch.md) — Cloud Run Job dispatch interface
- [M-CLOUD-JOB-RELIABILITY](../v0_9_1/m-cloud-job-reliability.md) — Defer guard + stale task detector (**now implemented**)
- [M-CLOUD-DUAL-AUTH](m-cloud-dual-auth.md) — KMS encryption for API keys (**now implemented**)
- [M-HTTP-HOOKS-CLOUD-TELEMETRY](../../implemented/v0_9_0/m-http-hooks-cloud-telemetry.md) — HTTP hooks for telemetry

### Planned (consumer side)

- [M-DASHBOARD-PUBSUB-EVENTS](../v0_9_1/m-dashboard-pubsub-events.md) — Dashboard subscribes to events topic (consumer of GAP 1)
- [M-CLOUD-OBSERVATORY](../v0_10_0/m-cloud-observatory.md) — Firestore backend for Observatory endpoints

### Supersedes

None — this is new work that fills the gap between dispatch and completion.

---

**Document created**: 2026-03-13
**Last updated**: 2026-03-13
