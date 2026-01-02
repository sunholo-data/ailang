# M-OTEL-EXTENDED: Extended OpenTelemetry Instrumentation

**Status:** Planned
**Target:** v0.6.3
**Priority:** P1 (Medium)
**Estimated:** 2 days (16 hours)
**Dependencies:** Telemetry infrastructure (completed in v0.6.1)
**Created:** 2026-01-02

## Problem Statement

AILANG has comprehensive OpenTelemetry infrastructure (`internal/telemetry/`) with support for:
- Google Cloud Trace export
- Generic OTLP export (Jaeger, Grafana, Honeycomb)
- Dual export mode (both simultaneously)
- Zero overhead when disabled

However, this instrumentation is currently limited to:
- HTTP server (`ailang serve`) - via `otelhttp` middleware
- Coordinator daemon (`ailang coordinator start`) - task lifecycle spans
- AI Providers (`internal/ai/`) - API call spans
- Executors (`internal/executor/`) - agentic execution spans

**Missing instrumentation points:**
1. **Compiler Pipeline** - No visibility into compilation phases (lexer, parser, type inference, lowering)
2. **Eval Harness** - No tracing of benchmark execution, model performance
3. **Message System** - No observability of message send/receive, search operations

**Impact:**
- Cannot identify slow compilation phases
- No performance data for AI code generation benchmarks
- Message system operations invisible to observability tools

## Goals

**Primary Goal:** Add OpenTelemetry spans to Compiler Pipeline, Eval Harness, and Message System with zero overhead when disabled.

**Success Metrics:**
1. Compiler phases visible as child spans (lexer, parser, elaborate, type-check, lower)
2. Eval benchmark runs traced with model/benchmark attributes
3. Message operations (send, read, search) traced with operation metadata
4. Zero performance impact when telemetry disabled (no-op spans)
5. All new spans integrate with existing GCP/OTLP export

## Solution Design

### Architecture

All components use the **global TracerProvider pattern** already established:

```
┌──────────────────────────────────────────────────────────────────┐
│                    Global TracerProvider                          │
│  (initialized once by ailang serve/coordinator/run)               │
├──────────────────────────────────────────────────────────────────┤
│  Server    │  Coordinator  │  AI Providers  │  Executors         │
│  (HTTP)    │  (Tasks)      │  (API calls)   │  (Claude/Gemini)   │
├──────────────────────────────────────────────────────────────────┤
│  NEW: Compiler  │  NEW: Eval Harness  │  NEW: Message System     │
│  (phases)       │  (benchmarks)       │  (send/receive/search)   │
└──────────────────────────────────────────────────────────────────┘
```

### Implementation Plan

#### Phase 1: Compiler Pipeline Instrumentation (~6 hours)

**Spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `compile.pipeline` | `internal/pipeline/pipeline.go` | `file.path`, `file.size_bytes` |
| `compile.lex` | `internal/pipeline/pipeline.go` | `tokens.count` |
| `compile.parse` | `internal/pipeline/pipeline.go` | `ast.nodes` |
| `compile.elaborate` | `internal/pipeline/pipeline.go` | `core.nodes` |
| `compile.typecheck` | `internal/pipeline/pipeline.go` | `types.inferred`, `constraints.count` |
| `compile.validate` | `internal/pipeline/pipeline.go` | `validation.passed` |
| `compile.lower` | `internal/pipeline/pipeline.go` | `ir.size` |

**Implementation:**

```go
// internal/pipeline/pipeline.go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("ailang.compiler")

func (p *Pipeline) Compile(ctx context.Context, src string) (*Result, error) {
    ctx, span := tracer.Start(ctx, "compile.pipeline",
        trace.WithAttributes(
            attribute.String("file.path", p.filename),
            attribute.Int("file.size_bytes", len(src)),
        ))
    defer span.End()

    // Lexer phase
    ctx, lexSpan := tracer.Start(ctx, "compile.lex")
    tokens, err := p.lex(src)
    lexSpan.SetAttributes(attribute.Int("tokens.count", len(tokens)))
    lexSpan.End()
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Parser phase
    ctx, parseSpan := tracer.Start(ctx, "compile.parse")
    ast, err := p.parse(tokens)
    parseSpan.SetAttributes(attribute.Int("ast.nodes", countNodes(ast)))
    parseSpan.End()
    // ... continue for each phase
}
```

**Files to modify:**
- `internal/pipeline/pipeline.go` (+80 LOC) - Main pipeline instrumentation
- `internal/pipeline/module_pipeline.go` (+40 LOC) - Module compilation

**Zero-overhead guarantee:**
When no TracerProvider is configured, `otel.Tracer()` returns a no-op tracer. The `Start()` calls become no-ops with ~2ns overhead (measured).

#### Phase 2: Eval Harness Instrumentation (~5 hours)

**Spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `eval.suite` | `internal/eval_harness/runner.go` | `suite.benchmarks`, `suite.models` |
| `eval.benchmark` | `internal/eval_harness/runner.go` | `benchmark.id`, `benchmark.difficulty` |
| `eval.model_call` | `internal/eval_harness/runner.go` | `model.id`, `model.provider`, `tokens.*` |
| `eval.validation` | `internal/eval_harness/runner.go` | `validation.passed`, `validation.errors` |

**Implementation:**

```go
// internal/eval_harness/runner.go
var tracer = otel.Tracer("ailang.eval")

func (r *Runner) RunSuite(ctx context.Context, suite *Suite) (*Results, error) {
    ctx, span := tracer.Start(ctx, "eval.suite",
        trace.WithAttributes(
            attribute.Int("suite.benchmarks", len(suite.Benchmarks)),
            attribute.StringSlice("suite.models", suite.Models),
        ))
    defer span.End()

    for _, benchmark := range suite.Benchmarks {
        ctx, bmSpan := tracer.Start(ctx, "eval.benchmark",
            trace.WithAttributes(
                attribute.String("benchmark.id", benchmark.ID),
                attribute.String("benchmark.difficulty", benchmark.Difficulty),
            ))

        result, err := r.runBenchmark(ctx, benchmark)
        bmSpan.SetAttributes(
            attribute.Bool("benchmark.passed", result.Passed),
            attribute.Float64("benchmark.duration_ms", result.DurationMs),
        )
        bmSpan.End()
    }
    return results, nil
}
```

**Files to modify:**
- `internal/eval_harness/runner.go` (+60 LOC) - Suite and benchmark spans
- `internal/eval_harness/model_runner.go` (+40 LOC) - Model call spans

**Key benefit:** Each model call within a benchmark becomes a child span, enabling:
- Cost attribution per benchmark
- Token usage analysis
- Latency breakdown by model

#### Phase 3: Message System Instrumentation (~5 hours)

**Spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `messages.send` | `internal/messages/store.go` | `message.inbox`, `message.type`, `message.github` |
| `messages.read` | `internal/messages/store.go` | `message.id`, `message.inbox` |
| `messages.list` | `internal/messages/store.go` | `query.inbox`, `query.status`, `results.count` |
| `messages.search` | `internal/messages/search.go` | `search.query`, `search.neural`, `results.count` |
| `messages.ack` | `internal/messages/store.go` | `message.id` |
| `messages.github_sync` | `internal/messages/github.go` | `github.repo`, `issues.imported` |

**Implementation:**

```go
// internal/messages/store.go
var tracer = otel.Tracer("ailang.messages")

func (s *Store) Send(ctx context.Context, msg *Message) error {
    ctx, span := tracer.Start(ctx, "messages.send",
        trace.WithAttributes(
            attribute.String("message.inbox", msg.Inbox),
            attribute.String("message.type", msg.Type),
            attribute.Bool("message.github", msg.GitHubIssue != nil),
        ))
    defer span.End()

    // Existing send logic...
    if err != nil {
        span.RecordError(err)
        return err
    }
    span.SetAttributes(attribute.String("message.id", msg.ID))
    return nil
}
```

**Files to modify:**
- `internal/messages/store.go` (+50 LOC) - CRUD operations
- `internal/messages/search.go` (+30 LOC) - Search operations
- `internal/messages/github.go` (+30 LOC) - GitHub sync

### Span Hierarchy Example

With full instrumentation, a coordinator task produces this trace tree:

```
coordinator.execute_task (task.id=abc123)
├── messages.read (message.id=msg_001)
├── compile.pipeline (file.path=task.ail)
│   ├── compile.lex (tokens.count=156)
│   ├── compile.parse (ast.nodes=42)
│   ├── compile.elaborate (core.nodes=38)
│   ├── compile.typecheck (types.inferred=12)
│   └── compile.lower (ir.size=1024)
├── eval.benchmark (benchmark.id=BM001)
│   └── gemini.execute (model=gemini-2.5-pro)
│       └── gemini.generate (tokens_in=500, tokens_out=1200)
└── messages.send (message.inbox=sprint-executor)
```

## Testing Strategy

### Unit Tests

Each instrumented component gets span verification tests:

```go
// internal/pipeline/pipeline_test.go
func TestPipelineTracing(t *testing.T) {
    // Setup test tracer with span recorder
    sr := tracetest.NewSpanRecorder()
    tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
    otel.SetTracerProvider(tp)

    // Run compilation
    p := NewPipeline("test.ail")
    _, err := p.Compile(context.Background(), "let x = 42")
    require.NoError(t, err)

    // Verify spans
    spans := sr.Ended()
    require.Len(t, spans, 6) // pipeline + 5 phases

    // Verify parent-child relationships
    pipelineSpan := findSpan(spans, "compile.pipeline")
    lexSpan := findSpan(spans, "compile.lex")
    assert.Equal(t, pipelineSpan.SpanContext().SpanID(), lexSpan.Parent().SpanID())
}
```

### Integration Tests

```bash
# Run with GCP tracing to verify spans appear
GOOGLE_CLOUD_PROJECT=multivac-internal-dev go test -tags=integration ./internal/pipeline/...
```

## Success Criteria

- [ ] Compiler pipeline phases visible as spans in traces
- [ ] Eval harness benchmarks traced with model/token attributes
- [ ] Message system operations traced
- [ ] All spans integrate with existing GCP/OTLP export
- [ ] Zero performance impact when telemetry disabled
- [ ] Tests verify span creation and hierarchy
- [ ] Documentation updated (`docs/docs/guides/telemetry.md`)

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A2: Replayability | +1 | Traces enable replay and analysis of execution |
| A7: Machines First | +1 | Structured spans are machine-readable |
| A9: Cost Visibility | +1 | Token/cost attributes make resource usage explicit |
| A3: Effect Legibility | 0 | Neutral - observability doesn't hide effects |

**Net score: +3** ✅ Accept

## Timeline

| Phase | Effort | Description |
|-------|--------|-------------|
| 1 | 6h | Compiler pipeline instrumentation |
| 2 | 5h | Eval harness instrumentation |
| 3 | 5h | Message system instrumentation |

**Total: 16 hours (~2 days)**

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/pipeline/pipeline.go` | Add phase spans | +80 |
| `internal/pipeline/module_pipeline.go` | Add module spans | +40 |
| `internal/eval_harness/runner.go` | Add suite/benchmark spans | +60 |
| `internal/eval_harness/model_runner.go` | Add model call spans | +40 |
| `internal/messages/store.go` | Add CRUD spans | +50 |
| `internal/messages/search.go` | Add search spans | +30 |
| `internal/messages/github.go` | Add sync spans | +30 |
| `docs/docs/guides/telemetry.md` | Update documentation | +50 |

**Total: ~380 LOC**

## Related Documents

- [docs/docs/guides/telemetry.md](../../../docs/docs/guides/telemetry.md) - Current telemetry documentation
- [internal/telemetry/otel.go](../../../internal/telemetry/otel.go) - Telemetry infrastructure

## Open Questions

1. **Sampling for compiler spans?** - For high-volume compilation (REPL), should we sample spans to reduce overhead?
   - Recommendation: No sampling initially; measure impact first

2. **Span attributes for type inference?** - Should we expose constraint solver metrics?
   - Recommendation: Start minimal, add detail based on debugging needs

## Risk Assessment

**Risk Level: Low**

- All changes are additive (new spans, no behavioral changes)
- Global TracerProvider pattern proven in existing code
- No-op overhead measured at <5ns per span when disabled
- Existing tests continue to pass without telemetry configured
