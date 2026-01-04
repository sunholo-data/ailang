# M-OTEL-EXTENDED: Extended OpenTelemetry Instrumentation

**Status:** Planned
**Target:** v0.6.3
**Priority:** P1 (Medium)
**Estimated:** 3 days (24 hours)
**Dependencies:** Telemetry infrastructure (completed in v0.6.1)
**Created:** 2026-01-02
**Updated:** 2026-01-04

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
3. **Message System** - Partial coverage (send/search exist, but list/read/ack missing)
4. **REPL Command** - Interactive sessions completely dark, no visibility into user interactions
5. **Check Command** - Type checking invocations have no tracing

**Impact:**
- Cannot identify slow compilation phases
- No performance data for AI code generation benchmarks
- Message system operations partially invisible to observability tools
- REPL debugging sessions cannot be analyzed post-hoc
- Type check performance regressions go undetected

## Goals

**Primary Goal:** Add OpenTelemetry spans to Compiler Pipeline, Eval Harness, Message System, REPL, and Check commands with zero overhead when disabled.

**Success Metrics:**
1. Compiler phases visible as child spans (lexer, parser, elaborate, type-check, lower)
2. Eval benchmark runs traced with model/benchmark attributes
3. Message operations (send, list, read, ack, search) traced with operation metadata
4. REPL sessions traced with command/evaluation hierarchy
5. Check command traced with file and type-check results
6. Zero performance impact when telemetry disabled (no-op spans)
7. All new spans integrate with existing GCP/OTLP export

## Solution Design

### Architecture

All components use the **global TracerProvider pattern** already established:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         Global TracerProvider                               │
│  (initialized once by ailang serve/coordinator/run/repl/check/eval-suite)   │
├────────────────────────────────────────────────────────────────────────────┤
│  Server    │  Coordinator  │  AI Providers  │  Executors                   │
│  (HTTP)    │  (Tasks)      │  (API calls)   │  (Claude/Gemini)             │
├────────────────────────────────────────────────────────────────────────────┤
│  M1: Compiler   │  M2: Eval Harness  │  M3: Message System                 │
│  (phases)       │  (benchmarks)       │  (send/list/read/ack/search)       │
├────────────────────────────────────────────────────────────────────────────┤
│  M4: REPL Command                    │  M5: Check Command                  │
│  (session/input/eval hierarchy)      │  (file type-check with phases)     │
└────────────────────────────────────────────────────────────────────────────┘
```

### Milestone Overview

| Milestone | Component | Priority | Effort | Status |
|-----------|-----------|----------|--------|--------|
| M1 | Compiler Pipeline | High | 6h | ✅ COMPLETE |
| M2 | Eval Harness | High | 5h | 🔄 IN PROGRESS |
| M3 | Extended Messages | Medium | 4h | Planned |
| M4 | REPL Command | Medium | 5h | Planned |
| M5 | Check Command | Medium | 4h | Planned |

### Implementation Plan

#### M1: Compiler Pipeline Instrumentation ✅ COMPLETE

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

#### M2: Eval Harness Instrumentation (~5 hours)

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

#### M3: Extended Message System Instrumentation (~4 hours)

**Existing spans (already implemented):**
- `messages.send` - inbox.go
- `messages.search` - search.go

**NEW spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `messages.list` | `internal/messages/store.go` | `query.inbox`, `query.status`, `results.count` |
| `messages.read` | `internal/messages/store.go` | `message.id`, `message.inbox` |
| `messages.ack` | `internal/messages/store.go` | `message.id`, `message.status` |
| `messages.unack` | `internal/messages/store.go` | `message.id` |
| `messages.github_sync` | `internal/messages/github.go` | `github.repo`, `issues.imported`, `issues.created` |
| `messages.cleanup` | `internal/messages/store.go` | `deleted.count`, `older_than` |

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
- `internal/messages/store.go` (+40 LOC) - list/read/ack/unack/cleanup
- `internal/messages/github.go` (+20 LOC) - GitHub sync spans

#### M4: REPL Command Instrumentation (~5 hours)

**Design:** The REPL session becomes a parent span containing all user interactions.

**Span hierarchy:**

```
repl.session (session.id=abc123, started_at=...)
├── repl.input (input.text=":type foo", input.type=command)
├── repl.input (input.text="let x = 42", input.type=expression)
│   └── compile.pipeline (file.path=<repl>)
│       ├── compile.parse
│       ├── compile.elaborate
│       └── compile.typecheck
├── repl.input (input.text="x + 1", input.type=expression)
│   ├── compile.pipeline
│   └── eval.expression (result.type=int, result.value=43)
└── repl.input (input.text=":quit", input.type=command)
```

**Spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `repl.session` | `cmd/ailang/repl.go` | `session.id`, `session.started_at` |
| `repl.input` | `internal/repl/repl.go` | `input.text`, `input.type` (command/expression/empty) |
| `repl.eval` | `internal/repl/repl_eval.go` | `result.type`, `eval.duration_ms` |

**Implementation notes:**
- Initialize telemetry at REPL start with `telemetry.Init(ctx, "ailang-repl")`
- Create session span that lives for entire REPL duration
- Each input line becomes a child span
- Compilation spans (M1) automatically nest under input spans
- On `:quit` or EOF, end session span with final attributes

**Files to modify:**
- `cmd/ailang/repl.go` (+30 LOC) - Session span, telemetry init
- `internal/repl/repl.go` (+40 LOC) - Input spans
- `internal/repl/repl_eval.go` (+20 LOC) - Eval spans

#### M5: Check Command Instrumentation (~4 hours)

**Design:** The check command produces a root span with compilation phases as children.

**Span hierarchy:**

```
ailang.check (file.path=main.ail, file.count=1)
├── compile.pipeline (file.path=main.ail)
│   ├── compile.parse (tokens.count=156)
│   ├── compile.elaborate (core.nodes=42)
│   ├── compile.typecheck (types.inferred=12)
│   └── compile.validate (validation.passed=true)
└── check.result (passed=true, errors.count=0, warnings.count=2)
```

**Spans to add:**

| Span Name | Location | Attributes |
|-----------|----------|------------|
| `ailang.check` | `cmd/ailang/check.go` | `file.path`, `file.count`, `timeout_ms` |
| `check.result` | `cmd/ailang/check.go` | `passed`, `errors.count`, `warnings.count` |

**Implementation notes:**
- Initialize telemetry with `telemetry.Init(ctx, "ailang-check")`
- Root span wraps entire check operation
- Compilation spans (M1) automatically nest as children
- Result span captures final outcome
- Supports multi-file checking with file count attribute

**Files to modify:**
- `cmd/ailang/check.go` (+40 LOC) - Root span, telemetry init, result span

### Span Hierarchy Examples

**Coordinator task trace:**

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

**REPL session trace (M4):**

```
repl.session (session.id=repl_001, duration_ms=45000)
├── repl.input (input.type=expression, input.text="let double = \\x. x * 2")
│   └── compile.pipeline (file.path=<repl>)
│       ├── compile.parse
│       ├── compile.elaborate
│       └── compile.typecheck
├── repl.input (input.type=expression, input.text="double(21)")
│   ├── compile.pipeline
│   └── repl.eval (result.type=int, result.value=42)
├── repl.input (input.type=command, input.text=":type double")
└── repl.input (input.type=command, input.text=":quit")
```

**Check command trace (M5):**

```
ailang.check (file.path=main.ail)
├── compile.pipeline (file.path=main.ail)
│   ├── compile.parse (tokens.count=234)
│   ├── compile.elaborate (core.nodes=89)
│   ├── compile.typecheck (types.inferred=24, constraints.count=45)
│   └── compile.validate (validation.passed=true)
└── check.result (passed=true, errors.count=0, warnings.count=1)
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

**M1: Compiler Pipeline** ✅
- [x] Compiler pipeline phases visible as spans in traces
- [x] All spans integrate with existing GCP/OTLP export

**M2: Eval Harness**
- [ ] Eval suite runs produce parent span visible in GCP/Jaeger
- [ ] Each benchmark is a child span of suite
- [ ] Token/cost data visible in span attributes
- [ ] Existing eval tests pass

**M3: Extended Messages**
- [ ] `messages.list` traced with query parameters
- [ ] `messages.read` traced with message ID
- [ ] `messages.ack`/`unack` traced
- [ ] `messages.github_sync` traced with import counts

**M4: REPL Command**
- [ ] `repl.session` spans capture full interactive sessions
- [ ] `repl.input` spans show each user input
- [ ] Compilation phases nest correctly under input spans
- [ ] Session duration and input count captured

**M5: Check Command**
- [ ] `ailang.check` root span with file path
- [ ] Compilation phases nest as children
- [ ] `check.result` shows pass/fail with error counts
- [ ] Works with `--timeout` flag

**Cross-cutting**
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

| Milestone | Effort | Description | Status |
|-----------|--------|-------------|--------|
| M1 | 6h | Compiler pipeline instrumentation | ✅ Complete |
| M2 | 5h | Eval harness instrumentation | 🔄 In Progress |
| M3 | 4h | Extended message operations | Planned |
| M4 | 5h | REPL command instrumentation | Planned |
| M5 | 4h | Check command instrumentation | Planned |

**Total: 24 hours (~3 days)**
**Remaining: 18 hours (~2.5 days)**

## Files to Modify

### M1: Compiler Pipeline ✅
| File | Change | LOC |
|------|--------|-----|
| `internal/pipeline/pipeline.go` | Add phase spans | +80 |
| `internal/pipeline/module_pipeline.go` | Add module spans | +40 |

### M2: Eval Harness
| File | Change | LOC |
|------|--------|-----|
| `internal/eval_harness/runner.go` | Add suite/benchmark spans | +60 |
| `internal/eval_harness/model_runner.go` | Add model call spans | +40 |

### M3: Extended Messages
| File | Change | LOC |
|------|--------|-----|
| `internal/messages/store.go` | Add list/read/ack spans | +40 |
| `internal/messages/github.go` | Add sync spans | +20 |

### M4: REPL Command
| File | Change | LOC |
|------|--------|-----|
| `cmd/ailang/repl.go` | Session span, telemetry init | +30 |
| `internal/repl/repl.go` | Input spans | +40 |
| `internal/repl/repl_eval.go` | Eval spans | +20 |

### M5: Check Command
| File | Change | LOC |
|------|--------|-----|
| `cmd/ailang/check.go` | Root span, telemetry init, result span | +40 |

### Documentation
| File | Change | LOC |
|------|--------|-----|
| `docs/docs/guides/telemetry.md` | Update with new spans | +80 |

**Total: ~490 LOC**

## Related Documents

- [docs/docs/guides/telemetry.md](../../../docs/docs/guides/telemetry.md) - Current telemetry documentation
- [internal/telemetry/otel.go](../../../internal/telemetry/otel.go) - Telemetry infrastructure

## Open Questions

1. **Sampling for REPL spans?** - Long REPL sessions could generate many spans. Sample?
   - Recommendation: No sampling initially; measure impact first
   - Consider: Session-level sampling (trace some sessions, not others)

2. **Span attributes for type inference?** - Should we expose constraint solver metrics?
   - Recommendation: Start minimal, add detail based on debugging needs

3. **REPL input text truncation?** - Should long inputs be truncated in span attributes?
   - Recommendation: Truncate to 200 chars; full text rarely needed for debugging

4. **Check command multi-file handling?** - Create one span per file or aggregate?
   - Recommendation: One parent span, child spans per file for clarity

## Risk Assessment

**Risk Level: Low**

- All changes are additive (new spans, no behavioral changes)
- Global TracerProvider pattern proven in existing code
- No-op overhead measured at <5ns per span when disabled
- Existing tests continue to pass without telemetry configured
