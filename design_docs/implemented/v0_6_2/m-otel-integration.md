# M-OTEL: OpenTelemetry OTLP Integration

**Status**: Planned
**Target**: v0.6.2
**Priority**: P1 - Medium (enables ecosystem integration)
**Estimated**: 3 days (24 hours implementation + testing)
**Dependencies**: None (additive feature)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on language semantics; observability is side-channel |
| A2: Replayability | +1 | Distributed traces enable cross-service replay/debugging |
| A3: Effect Legibility | +1 | Makes coordinator/executor effects visible to external tools |
| A4: Explicit Authority | 0 | No new capabilities; telemetry is opt-in via env vars |
| A5: Bounded Verification | 0 | No impact on type checking |
| A6: Safe Concurrency | 0 | No concurrency changes; async export is handled by OTEL SDK |
| A7: Machines First | +1 | Structured telemetry is machine-readable (OTLP protocol) |
| A8: Minimal Syntax | +1 | No new syntax; uses standard OTEL environment variables |
| A9: Cost Visibility | +1 | Exposes token costs, latency, resource usage as metrics |
| A10: Composability | +1 | OTEL composes with any OTLP backend (Grafana, Honeycomb, etc.) |
| A11: Structured Failure | +1 | Errors become structured spans with attributes |
| A12: System Boundary | +1 | Makes coordinator→executor→provider boundaries explicit |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects (telemetry is explicit opt-in)
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): OTLP is a machine-readable protocol

## Problem Statement

AILANG's coordinator, server, and executors generate valuable operational data (task lifecycle, costs, latencies) but this data is siloed in custom formats that don't integrate with the broader observability ecosystem.

**Current State:**
- Custom JSON events over HTTP/WebSocket
- Standard Go `log` package (unstructured text)
- SQLite storage only (no real-time export)
- No distributed tracing across coordinator→executor→provider
- Cannot correlate AILANG events with Claude Code/Gemini CLI telemetry

**Impact:**
- Operators cannot use existing observability tools (Grafana, Datadog, Honeycomb)
- No unified view of AI tool usage across Claude Code + Gemini CLI + AILANG
- Debugging distributed workflows requires manual log correlation
- Cost tracking is fragmented across systems

## Goals

**Primary Goal:** Enable AILANG to export telemetry via OpenTelemetry Protocol (OTLP), integrating with ai-observer and other OTLP backends for unified observability.

**Success Metrics:**
- AILANG coordinator/server/executor emit OTLP traces, metrics, logs
- Traces visible in ai-observer dashboard alongside Claude Code/Gemini telemetry
- Task lifecycle spans (create→execute→complete) have <10ms overhead
- Cost metrics aggregated correctly across all providers
- Zero breaking changes to existing functionality

## Solution Design

### Overview

Add OpenTelemetry instrumentation to AILANG's core systems:
1. **HTTP Server** - Auto-instrument all API handlers
2. **Coordinator** - Manual spans for task lifecycle
3. **Executors** - Spans for Claude/Gemini execution with turn-level granularity
4. **AI Providers** - Spans for API calls with token/cost attributes

All telemetry is opt-in via standard OTEL environment variables.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     OTLP Collector (4318)                       │
│              (ai-observer / Grafana / Honeycomb)                │
└─────────────────────────────────────────────────────────────────┘
         ▲                    ▲                    ▲
         │ OTLP/HTTP          │ OTLP/HTTP          │ OTLP/HTTP
    ┌────┴────┐         ┌────┴────┐         ┌────┴────┐
    │ Claude  │         │ AILANG  │         │ Gemini  │
    │  Code   │         │  Stack  │         │  CLI    │
    │ (native)│         │  (new)  │         │(native) │
    └─────────┘         └────┬────┘         └─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
        ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐
        │  Server   │  │Coordinator│  │ Executors │
        │  (HTTP)   │  │  (Daemon) │  │(Claude/   │
        │           │  │           │  │ Gemini)   │
        └───────────┘  └───────────┘  └───────────┘
```

**Components:**

1. **internal/telemetry/** - New package for OTEL setup and configuration
2. **otelhttp middleware** - Wraps server mux for automatic HTTP tracing
3. **Manual spans** - Task lifecycle, executor calls, provider requests
4. **Metrics** - Counters, histograms, gauges for key operations
5. **Structured logs** - slog integration with trace context

### Implementation Plan

**Phase 1: Foundation** (~4 hours)
- [ ] Add OTEL dependencies to go.mod
- [ ] Create `internal/telemetry/` package
- [ ] Implement `InitOTLP()` with tracer/meter/logger providers
- [ ] Add resource configuration (service name, version, env)
- [ ] Environment variable configuration (standard OTEL vars)
- [ ] Graceful shutdown handling

**Phase 2: Server Instrumentation** (~4 hours)
- [ ] Wrap HTTP mux with `otelhttp.NewHandler()`
- [ ] Add filter for /health endpoint
- [ ] Instrument WebSocket connections (manual spans)
- [ ] Add request/response metrics
- [ ] Test with ai-observer

**Phase 3: Coordinator Instrumentation** (~6 hours)
- [ ] Root span for `processMessages()` loop
- [ ] Task lifecycle spans (created→running→completed/failed)
- [ ] Approval workflow spans
- [ ] GitHub sync spans
- [ ] Metrics: tasks.running, cost.total, approval.latency

**Phase 4: Executor Instrumentation** (~6 hours)
- [ ] Span for `Execute()` method
- [ ] Turn-level child spans
- [ ] Token/cost attributes on spans
- [ ] Metrics: tokens.in/out, cost.per_task, duration
- [ ] Error recording with span status

**Phase 5: AI Provider Instrumentation** (~4 hours)
- [ ] Wrap provider `Generate()` calls
- [ ] HTTP client transport instrumentation
- [ ] Rate limit/retry tracking
- [ ] Per-provider metrics

### Files to Modify/Create

**New files:**
- `internal/telemetry/otel.go` - OTEL setup and configuration (~200 LOC)
- `internal/telemetry/resource.go` - Service resource attributes (~50 LOC)
- `internal/telemetry/metrics.go` - Metric definitions (~150 LOC)
- `internal/telemetry/doc.go` - Package documentation (~20 LOC)

**Modified files:**
- `go.mod` - Add OTEL dependencies (~10 lines)
- `internal/server/server.go` - Add otelhttp middleware (~20 LOC)
- `internal/coordinator/daemon.go` - Add task lifecycle spans (~50 LOC)
- `internal/coordinator/task_executor.go` - Add execution spans (~30 LOC)
- `internal/executor/claude/claude.go` - Add executor spans (~40 LOC)
- `internal/executor/gemini/gemini.go` - Add executor spans (~40 LOC)
- `internal/ai/anthropic/client.go` - Add provider spans (~30 LOC)
- `internal/ai/openai/client.go` - Add provider spans (~30 LOC)
- `internal/ai/gemini/client.go` - Add provider spans (~30 LOC)
- `internal/ai/ollama/client.go` - Add provider spans (~30 LOC)
- `cmd/ailang/main.go` - Initialize telemetry on startup (~15 LOC)

**Total new code:** ~750 LOC

## Examples

### Example 1: Environment Configuration

**Enable OTLP export to ai-observer:**
```bash
export OTEL_SERVICE_NAME=ailang-coordinator
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp

# Start coordinator with telemetry
ailang coordinator start
```

### Example 2: Task Lifecycle Trace

**Trace structure for a task execution:**
```
coordinator.task.execute (root span)
├── attributes: task_id, task_type, agent_id, provider
├── duration: 45.2s
├── status: OK
│
├── coordinator.agent.execute
│   ├── attributes: agent_id=sprint-executor, workspace=/path
│   │
│   └── executor.claude.execute
│       ├── attributes: model=claude-sonnet-4-5
│       ├── duration: 44.8s
│       │
│       ├── executor.turn (turn 1)
│       │   ├── attributes: turn_num=1, tokens_in=1200, tokens_out=450
│       │   └── tool_use: Read file.go
│       │
│       ├── executor.turn (turn 2)
│       │   ├── attributes: turn_num=2, tokens_in=800, tokens_out=1200
│       │   └── tool_use: Edit file.go
│       │
│       └── executor.turn (turn 3)
│           └── attributes: turn_num=3, tokens_in=200, tokens_out=50
│
└── coordinator.approval.wait
    ├── attributes: approval_type=merge, timeout=3600s
    └── duration: 120.5s
```

### Example 3: Metrics Exported

**Counters:**
```
ailang.coordinator.tasks.total{type="bug-fix", status="completed"} 42
ailang.coordinator.tasks.total{type="feature", status="failed"} 3
ailang.executor.calls.total{provider="claude"} 156
ailang.executor.calls.total{provider="gemini"} 23
```

**Histograms:**
```
ailang.coordinator.task.duration_ms{type="bug-fix"} p50=12000 p95=45000 p99=120000
ailang.executor.tokens.output{provider="claude"} p50=800 p95=2500 p99=5000
ailang.executor.cost_usd{provider="claude"} sum=12.45
```

**Gauges:**
```
ailang.coordinator.tasks.running 3
ailang.coordinator.approvals.pending 2
```

### Example 4: Code Integration

**Server instrumentation:**
```go
// internal/server/server.go
func (s *Server) Start(ctx context.Context) error {
    // Initialize OTEL (no-op if env vars not set)
    shutdown, err := telemetry.InitOTLP(ctx, "ailang-server")
    if err != nil {
        log.Printf("OTEL init failed (continuing without telemetry): %v", err)
    } else {
        defer shutdown(ctx)
    }

    // Wrap mux with otelhttp
    handler := otelhttp.NewHandler(s.mux, "ailang-server",
        otelhttp.WithFilter(func(r *http.Request) bool {
            return r.URL.Path != "/health"
        }),
    )

    return http.ListenAndServe(s.addr, handler)
}
```

**Coordinator task span:**
```go
// internal/coordinator/daemon.go
func (d *Daemon) executeTask(ctx context.Context, task *Task) error {
    ctx, span := otel.Tracer("coordinator").Start(ctx, "task.execute",
        trace.WithAttributes(
            attribute.String("task.id", task.ID),
            attribute.String("task.type", string(task.Type)),
            attribute.String("agent.id", task.AgentID),
        ),
    )
    defer span.End()

    result, err := d.executor.Execute(ctx, task)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    // Record metrics
    span.SetAttributes(
        attribute.Float64("task.cost_usd", result.Cost),
        attribute.Int64("task.tokens_out", result.TokensOut),
        attribute.Int64("task.duration_ms", result.DurationMs),
    )

    return nil
}
```

## Success Criteria

- [ ] `ailang coordinator start` with OTEL env vars exports traces to OTLP endpoint
- [ ] `ailang serve` with OTEL env vars shows HTTP request spans
- [ ] Task execution creates hierarchical spans (coordinator→executor→turns)
- [ ] Metrics visible in ai-observer/Grafana/Prometheus
- [ ] <10ms overhead per request when telemetry enabled
- [ ] Zero overhead when telemetry disabled (env vars not set)
- [ ] All existing tests passing
- [ ] Documentation updated with configuration guide
- [ ] Example docker-compose with ai-observer integration

## Testing Strategy

**Unit tests:**
- `internal/telemetry/otel_test.go` - Provider initialization, shutdown
- `internal/telemetry/metrics_test.go` - Metric recording
- Mock OTEL exporter to verify span/metric creation

**Integration tests:**
- Start coordinator with OTLP exporter pointing to test server
- Execute sample task, verify spans received
- Verify trace hierarchy is correct
- Verify metrics have correct labels

**Manual testing:**
1. Run ai-observer locally (`docker run -p 4318:4318 -p 8080:8080 ai-observer`)
2. Start AILANG coordinator with OTEL env vars
3. Execute a task via `ailang messages send`
4. Verify traces/metrics in ai-observer dashboard
5. Correlate with Claude Code telemetry

## Non-Goals

**Not in this feature:**
- Custom AILANG dashboard for OTEL data - Use ai-observer/Grafana instead
- Trace sampling configuration - Use 100% initially, add later if needed
- Custom exporters (Jaeger, Zipkin) - OTLP covers all via collectors
- Automatic retry/circuit breaker telemetry - Add in future iteration
- Database query tracing - Add in future iteration (requires sql wrapper)

## Timeline

**Day 1** (8 hours):
- Phase 1: Foundation (4h)
- Phase 2: Server instrumentation (4h)
- Manual test with ai-observer

**Day 2** (8 hours):
- Phase 3: Coordinator instrumentation (6h)
- Begin Phase 4: Executor instrumentation (2h)

**Day 3** (8 hours):
- Complete Phase 4: Executor instrumentation (4h)
- Phase 5: AI provider instrumentation (4h)
- Integration tests
- Documentation

**Total: ~24 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OTEL SDK adds latency | Medium | Async batching (default), benchmark before/after |
| Context propagation breaks | High | Careful ctx passing, integration tests |
| Dependency bloat | Low | OTEL packages are well-maintained, minimal deps |
| Breaking existing behavior | High | All telemetry is opt-in via env vars |
| ai-observer API changes | Low | Using standard OTLP, not ai-observer specific APIs |

## Related Documents

**External references:**
- [ai-observer](https://github.com/sunholo-data/ai-observer) - Target OTLP backend
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/) - Official docs
- [otelhttp package](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp) - HTTP middleware

**AILANG docs:**
- [design_docs/planned/v0_6_2/global-collaboration-hub.md](global-collaboration-hub.md) - Dashboard may display OTEL data
- [docs/docs/guides/coordinator.md](../../../docs/docs/guides/coordinator.md) - Coordinator architecture

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/) - Protocol spec
- [Go OTEL Best Practices](https://opentelemetry.io/docs/languages/go/instrumentation/) - Implementation guide

## Future Work

**v0.6.3+:**
- Database query tracing (wrap sql.DB)
- Eval harness telemetry (benchmark runs as traces)
- Custom sampling rules (reduce volume in production)
- Baggage propagation for cross-service correlation
- Prometheus scrape endpoint alternative to push

**v0.7.0+:**
- AILANG runtime instrumentation (effect execution traces)
- User-defined spans in AILANG code (`trace "name" { ... }`)

---

**Document created**: 2026-01-02
**Last updated**: 2026-01-02
