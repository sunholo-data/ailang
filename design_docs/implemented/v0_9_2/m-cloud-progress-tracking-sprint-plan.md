# Sprint Plan: M-CLOUD-PROGRESS-TRACKING

## Summary

Wire mid-execution progress events, cost tracking, cost budgets, and Cloud Trace into the cloud executor path. The building blocks (PubSubBroadcaster, CoordinatorEventHandler, dual OTEL exporter, trace context propagation) all exist — this sprint is about connecting them in the Cloud Run Job code path.

**Duration:** 2 days (~12h implementation + testing)
**Dependencies:** M-CLOUD-E2E (done), M-CLOUD-DISPATCH (done), M-CLOUD-JOB-RELIABILITY (done)
**Risk Level:** Low — all building blocks exist, changes are wiring + small extensions
**Design Doc:** [m-cloud-progress-tracking.md](m-cloud-progress-tracking.md)

## Current Status Analysis

### What Already Exists (No Changes Needed)

- `PubSubBroadcaster` — publishes `TaskStreamEvent` to ailang-events topic
- `TaskStreamEvent` — already has `Cost`, `TokensIn`, `TokensOut`, `DurationSec` fields
- `CoordinatorEventHandler` — full event handler with broadcasting + metrics (used in local mode)
- `InitDual()` in `internal/telemetry/otel.go` — dual OTEL exporter to Cloud Trace + OTLP
- `InjectTraceContext()` / `ExtractTraceContext()` — W3C traceparent propagation via env vars
- `TaskMaxCost` config field in `AgentConfig` — already parsed from YAML
- Pre-execution budget checking in `daemon_tasks_budget.go`

### Velocity

Recent 14 days: ~3,150 insertions across 46 files (20 commits). ~225 LOC/day.
Sprint target: ~200 LOC across 8 files. Well within capacity.

## Proposed Milestones

### Milestone 1: Progress Streaming (GAP 1)

**Goal:** Cloud Run Job publishes turn-by-turn events to Pub/Sub during execution, not just at completion.

**Estimated:** ~50 LOC implementation + ~30 LOC tests = ~80 LOC
**Duration:** ~3h

**Approach:** Add a `broadcaster` field to `cloudEventHandler` in `coordinator_cloud.go`. Create a `PubSubBroadcaster` in `executeCloudTask()` using the existing Pub/Sub publisher. Each handler method calls `broadcaster.Broadcast()` after logging.

**Tasks:**
1. In `executeCloudTask()`, create `PubSubBroadcaster` from the existing publisher
2. Add `broadcaster *coordinator.PubSubBroadcaster` field to `cloudEventHandler`
3. In each event method (OnTurnStart, OnText, OnToolUse, OnToolResult, OnTurnEnd, OnError), broadcast a `TaskStreamEvent` after the stderr log
4. Rate-limit OnText broadcasts (one per 500ms, like CoordinatorEventHandler)
5. Add unit test: mock broadcaster, fire events, verify broadcasts

**Key files:**
- `cmd/ailang/coordinator_cloud.go` — Wire broadcaster into handler (~40 LOC)
- `cmd/ailang/coordinator_cloud_test.go` — Test event broadcasting (~30 LOC)

**Acceptance Criteria:**
- [x] `OnTurnStart` broadcasts `TaskStreamTurnStart` event
- [x] `OnText` broadcasts `TaskStreamText` (rate-limited)
- [x] `OnToolUse` broadcasts `TaskStreamToolUse`
- [x] `OnTurnEnd` broadcasts `TaskStreamTurnEnd`
- [x] `OnError` broadcasts `TaskStreamError`
- [x] Events include `TaskID`, `AgentID`, `Workspace`
- [x] `make test` passes
- [x] `make lint` passes

**Risks:**
- Pub/Sub publish latency per event — Mitigation: fire-and-forget, don't block executor

---

### Milestone 2: Turn-by-Turn Cost Tracking (GAP 2)

**Goal:** Include running cost/token counts in progress events so the dashboard shows accumulating cost.

**Estimated:** ~40 LOC implementation + ~20 LOC tests = ~60 LOC
**Duration:** ~3h

**Approach:** The Claude executor tracks token counts per API call internally but doesn't expose them to the event handler during execution. We need to:
1. Add `OnTurnMetrics(stats TurnStats)` to the `executor.EventHandler` interface
2. Call it from `claude.go` after each `message_stop` when we have the `usage` data
3. `cloudEventHandler` accumulates running cost and includes it in TurnEnd broadcasts

This is cleaner than modifying `OnTurnEnd(int)` signature (which would break all existing implementations).

**Tasks:**
1. Add `TurnStats` struct and `OnTurnMetrics(TurnStats)` to `executor.EventHandler` interface
2. Add no-op implementations in existing handlers (CoordinatorEventHandler, NoOpEventHandler)
3. In `claude.go`, after parsing `usage` from the stream-json `result` event, call `handler.OnTurnMetrics(stats)`
4. In `cloudEventHandler`, accumulate `runningCost` and `runningTokens`
5. Include accumulated cost in TurnEnd broadcast (`Cost` and `TokensIn`/`TokensOut` fields)
6. Test: verify metrics accumulate correctly across turns

**Key files:**
- `internal/executor/handler.go` (or wherever EventHandler interface lives) — Add TurnStats + method (~15 LOC)
- `internal/executor/claude/claude.go` — Call OnTurnMetrics after usage parse (~10 LOC)
- `cmd/ailang/coordinator_cloud.go` — Accumulate and broadcast cost (~15 LOC)
- `internal/coordinator/event_handler.go` — No-op OnTurnMetrics (~3 LOC)

**Acceptance Criteria:**
- [x] `ExecutionMetrics` includes InputTokens, OutputTokens, CostUSD, NumTurns, DurationMS, SessionID, Success
- [x] `cloudEventHandler` receives final metrics via `OnMetrics` (optional `MetricsHandler` interface)
- [x] Status broadcasts include `cost` and `tokens_in`/`tokens_out` fields
- [x] Cost from executor's `result` event (already calculated in claude.go)
- [x] `make test` passes
- [x] `make lint` passes

**Risks:**
- Token counts may not be available in all stream-json formats — Mitigation: only emit when usage data present, gracefully skip otherwise

---

### Milestone 3: Per-Task Cost Budget & Early Abort (GAP 3)

**Goal:** Abort task execution when running cost exceeds the configured `TaskMaxCost` limit.

**Estimated:** ~40 LOC implementation + ~20 LOC tests = ~60 LOC
**Duration:** ~3h

**Approach:** Pass `TaskMaxCost` from agent config → `DispatchParams` → `AILANG_MAX_COST_USD` env var → Cloud Run Job. The `cloudEventHandler` checks budget after each turn and cancels the context if exceeded.

**Tasks:**
1. Add `MaxCostUSD float64` to `DispatchParams` struct
2. In `daemon_tasks_exec.go`, populate `MaxCostUSD` from agent config's `TaskMaxCost`
3. In `dispatcher.go`, pass `AILANG_MAX_COST_USD` env var override if > 0
4. In `coordinator_cloud.go`, add `cancel context.CancelFunc` and `maxCostUSD float64` to `cloudEventHandler`
5. In `OnTurnMetrics`, check `runningCost > maxCostUSD` and call `cancel()` if exceeded
6. After `ExecuteStreaming` returns, check if cancellation was due to budget and report appropriately
7. Test: verify budget enforcement triggers cancellation

**Key files:**
- `internal/coordinator/cloud_dispatcher.go` — Add MaxCostUSD field (~2 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — Pass env var (~5 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — Populate from config (~8 LOC)
- `cmd/ailang/coordinator_cloud.go` — Budget check + cancel (~20 LOC)
- `cmd/ailang/coordinator_cloud_test.go` — Test budget abort (~20 LOC)

**Acceptance Criteria:**
- [x] `AILANG_MAX_COST_USD` passed as env var to Cloud Run Job
- [x] Task aborts when running cost exceeds budget
- [x] Error event broadcast with `cost budget exceeded ($X.XX > $Y.YY limit)` message
- [x] Budget of 0 (or unset) means unlimited (no enforcement)
- [x] `make test` passes
- [x] `make lint` passes

**Risks:**
- Context cancellation race with normal completion — Mitigation: check cancellation reason before reporting

---

### Milestone 4: Cloud Trace Integration (GAP 4)

**Goal:** Link coordinator spans → Cloud Run Job spans in GCP Cloud Trace.

**Estimated:** ~20 LOC implementation + ~10 LOC tests = ~30 LOC
**Duration:** ~2h

**Approach:** The dual exporter and trace context propagation already exist. We just need to:
1. Ensure `GOOGLE_CLOUD_PROJECT` is set in Cloud Run Job env (Terraform may need check)
2. Pass `TRACEPARENT` via dispatcher env overrides
3. Call `ExtractTraceContext()` at Cloud Run Job startup

**Tasks:**
1. In `dispatcher.go`, call `telemetry.InjectTraceContext(ctx, envOverrides)` to propagate traceparent
2. In `coordinator_cloud.go` `executeCloudTask()`, call `telemetry.ExtractTraceContext(ctx)` at startup
3. Wrap `executeCloudTask` body in a span: `tracer.Start(ctx, "cloud_job.execute")`
4. Verify `GOOGLE_CLOUD_PROJECT` env var is set on Cloud Run Job (check Terraform)
5. Test: verify trace context is injected into env overrides

**Key files:**
- `internal/dispatch/cloudrun/dispatcher.go` — Inject trace context (~5 LOC)
- `cmd/ailang/coordinator_cloud.go` — Extract trace context + create span (~10 LOC)
- `internal/dispatch/cloudrun/dispatcher_test.go` — Test trace injection (~10 LOC)

**Acceptance Criteria:**
- [x] Coordinator span linked to Cloud Run Job span via W3C traceparent
- [x] `TRACEPARENT` env var passed to Cloud Run Job
- [x] Cloud Run Job creates child span `cloud_job.execute`
- [x] Spans visible in GCP Cloud Trace (if GOOGLE_CLOUD_PROJECT set)
- [x] `make test` passes
- [x] `make lint` passes

**Risks:**
- `GOOGLE_CLOUD_PROJECT` may not be set on Cloud Run Job — Mitigation: trace init is no-op when not set, graceful degradation

---

## Day-by-Day Breakdown

### Day 1: GAPs 1 + 2 (Progress Streaming + Cost Tracking)

| Time | Task |
|------|------|
| Morning | M1: Wire PubSubBroadcaster into cloudEventHandler |
| Morning | M1: Add rate-limiting for OnText broadcasts |
| Morning | M1: Tests |
| Afternoon | M2: Add OnTurnMetrics to EventHandler interface |
| Afternoon | M2: Call from claude.go, accumulate in handler |
| Afternoon | M2: Tests |
| EOD | `make test && make lint` — commit M1+M2 |

### Day 2: GAPs 3 + 4 (Cost Budgets + Cloud Trace)

| Time | Task |
|------|------|
| Morning | M3: Add MaxCostUSD to DispatchParams + env var |
| Morning | M3: Budget check + context cancellation in handler |
| Morning | M3: Tests |
| Afternoon | M4: Inject trace context in dispatcher |
| Afternoon | M4: Extract trace context in Cloud Run Job |
| Afternoon | M4: Tests |
| EOD | `make test && make lint` — commit M3+M4 |
| EOD | Update CHANGELOG |

## Success Metrics

- All existing tests passing
- All linting clean (`make lint`)
- 4 new test functions (one per milestone)
- No regression in local mode
- CHANGELOG updated

## Open Questions

1. **Pub/Sub publisher in Cloud Run Job**: Does `executeCloudTask()` already have a Pub/Sub publisher for completion? If so, reuse it. If not, need to create one (adds ~5 LOC).
2. **Token usage in stream-json**: Verify the `usage` field location in Claude's stream-json `result` event — need exact JSON path for InputTokens/OutputTokens.
3. **GOOGLE_CLOUD_PROJECT on Cloud Run Job**: Confirm this env var is set in Terraform for the agent-executor job. If not, add it.

## Notes

- All building blocks exist — this sprint is primarily wiring work
- The `CoordinatorEventHandler` (local mode) is the reference implementation for how events should flow
- No Terraform changes expected (env vars already provisioned per ailang-multivac message)
- ~230 LOC total across 8 files — conservative 2-day estimate given wiring complexity
