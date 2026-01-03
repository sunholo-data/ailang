# M-OTEL Sprint Plan: OpenTelemetry OTLP Integration

**Sprint ID**: M-OTEL
**Design Doc**: [m-otel-integration.md](m-otel-integration.md)
**Duration**: 3 days (~24 hours)
**Risk Level**: Low (additive feature, opt-in)

## Sprint Summary

**Goal**: Add native OpenTelemetry OTLP export to AILANG's coordinator, server, and executors for unified observability with ai-observer and other OTLP backends.

**Key Deliverables**:
1. `internal/telemetry/` package with OTEL setup
2. HTTP server instrumentation via otelhttp middleware
3. Coordinator task lifecycle tracing
4. Executor (Claude/Gemini) turn-level spans
5. AI provider API call instrumentation

## Current Status

**Velocity Analysis** (last 7 days):
- ~1,550 LOC added to coordinator (ApprovalWatcher, TaskChain, StageExecution)
- ~15 commits merged
- Average: ~220 LOC/day

**Estimated LOC for M-OTEL**: ~750 LOC
**Estimated Duration**: 3 days at current velocity

## Milestone Breakdown

### M1: Foundation (~200 LOC, 4 hours)

**Description**: Create `internal/telemetry/` package with OTEL providers

**Tasks**:
- [ ] Add OTEL dependencies to go.mod
- [ ] Create `internal/telemetry/otel.go` - InitOTLP function
- [ ] Create `internal/telemetry/resource.go` - Service resource
- [ ] Create `internal/telemetry/doc.go` - Package docs
- [ ] Unit tests for initialization and shutdown

**Files to Create**:
- `internal/telemetry/otel.go` (~120 LOC)
- `internal/telemetry/resource.go` (~50 LOC)
- `internal/telemetry/doc.go` (~10 LOC)
- `internal/telemetry/otel_test.go` (~80 LOC)

**Acceptance Criteria**:
- [ ] `telemetry.InitOTLP(ctx, "service-name")` returns working providers
- [ ] Graceful shutdown on context cancellation
- [ ] No-op when OTEL env vars not set
- [ ] Unit tests pass

**Dependencies**: None

---

### M2: Server Instrumentation (~100 LOC, 4 hours)

**Description**: Wrap HTTP server with otelhttp middleware

**Tasks**:
- [ ] Add otelhttp middleware to server.go
- [ ] Filter /health endpoint from tracing
- [ ] Add basic HTTP metrics
- [ ] Manual test with ai-observer

**Files to Modify**:
- `internal/server/server.go` (~30 LOC)
- `cmd/ailang/main.go` - Initialize telemetry (~15 LOC)

**Files to Create**:
- `internal/telemetry/metrics.go` (~100 LOC) - Metric definitions

**Acceptance Criteria**:
- [ ] HTTP requests create spans in OTLP collector
- [ ] /health endpoint excluded from traces
- [ ] Request duration histogram exported
- [ ] Manual test with ai-observer passes

**Dependencies**: M1

---

### M3: Coordinator Instrumentation (~150 LOC, 6 hours)

**Description**: Add task lifecycle spans to coordinator daemon

**Tasks**:
- [ ] Root span for task execution in daemon.go
- [ ] Child spans for agent execution
- [ ] Approval workflow spans
- [ ] Task metrics (running count, cost, duration)

**Files to Modify**:
- `internal/coordinator/daemon.go` (~50 LOC)
- `internal/coordinator/daemon_tasks.go` (~40 LOC)
- `internal/coordinator/task_executor.go` (~30 LOC)
- `internal/coordinator/approval_checkpoint.go` (~30 LOC)

**Acceptance Criteria**:
- [ ] Task execution creates hierarchical spans
- [ ] Span attributes include task_id, type, agent_id
- [ ] Cost and token counts recorded on span
- [ ] Approval wait time tracked

**Dependencies**: M1, M2

---

### M4: Executor Instrumentation (~160 LOC, 6 hours)

**Description**: Add turn-level spans to Claude/Gemini executors

**Tasks**:
- [ ] Wrap Execute() with parent span
- [ ] Child span per turn
- [ ] Token/cost attributes on each turn
- [ ] Error recording with span status

**Files to Modify**:
- `internal/executor/claude/claude.go` (~80 LOC)
- `internal/executor/gemini/gemini.go` (~80 LOC)

**Acceptance Criteria**:
- [ ] Each executor call creates a span
- [ ] Turns appear as child spans
- [ ] Token counts accurate per turn
- [ ] Errors properly recorded

**Dependencies**: M1, M3

---

### M5: AI Provider Instrumentation (~140 LOC, 4 hours)

**Description**: Add spans for AI API calls

**Tasks**:
- [ ] Wrap Generate() calls in all providers
- [ ] Record model, tokens, latency as attributes
- [ ] Track rate limits and retries

**Files to Modify**:
- `internal/ai/anthropic/client.go` (~35 LOC)
- `internal/ai/openai/client.go` (~35 LOC)
- `internal/ai/gemini/client.go` (~35 LOC)
- `internal/ai/ollama/client.go` (~35 LOC)

**Acceptance Criteria**:
- [ ] Each API call creates a span
- [ ] Model and token counts visible
- [ ] Rate limit events recorded
- [ ] Provider errors properly attributed

**Dependencies**: M1

---

## Day-by-Day Plan

### Day 1 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-2h | Add OTEL deps, create telemetry package structure | M1 |
| 2-4h | Implement InitOTLP, resource, shutdown | M1 |
| 4-6h | Add otelhttp middleware to server | M2 |
| 6-8h | Test with ai-observer, fix issues | M2 |

**End of Day 1**: Server requests visible in ai-observer

### Day 2 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-3h | Add task lifecycle spans to daemon | M3 |
| 3-5h | Add approval workflow spans | M3 |
| 5-8h | Begin executor instrumentation | M4 |

**End of Day 2**: Coordinator tasks create full trace hierarchy

### Day 3 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-3h | Complete executor instrumentation | M4 |
| 3-6h | AI provider instrumentation | M5 |
| 6-7h | Integration testing | All |
| 7-8h | Documentation, cleanup | All |

**End of Day 3**: Full OTLP integration complete

## Success Metrics

- [ ] All 5 milestones complete
- [ ] ~750 LOC added (±100)
- [ ] All existing tests pass
- [ ] Manual test with ai-observer successful
- [ ] Documentation updated with configuration guide
- [ ] No performance regression (benchmark before/after)

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| OTEL SDK complexity | Start with minimal instrumentation, add incrementally |
| Context propagation issues | Test each milestone independently |
| Performance overhead | Use async batching (default), measure before/after |

## Open Questions

None - design doc provides clear guidance.

## Dependencies

**Go packages to add**:
```
go.opentelemetry.io/otel v1.28.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.28.0
go.opentelemetry.io/otel/sdk v1.28.0
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
```

---

**Document created**: 2026-01-02
**Sprint starts**: Immediately upon approval
