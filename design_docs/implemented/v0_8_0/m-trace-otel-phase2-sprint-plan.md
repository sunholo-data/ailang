# Sprint Plan: M-TRACE-EXPORT Phase 2 — OTEL Span Emission

## Summary

Convert program-level trace events (from `internal/trace/Collector`) into OTEL spans so they flow into the existing observatory/chains/dashboard infrastructure. After this sprint, `ailang run` will emit function, effect, and contract spans visible in Cloud Trace and `ailang chains view`.

**Duration:** 1 day (4-5 hours)
**Dependencies:** M-TRACE-EXPORT Phase 1 (complete), OTEL SDK (already in go.mod)
**Risk Level:** Low — batch emission after execution, follows established patterns exactly

## Current Status Analysis

### Completed Recently
- M-TRACE-EXPORT Phase 1: ~500 LOC in ~3 hours (trace package, collector, JSONL, CLI flag)
- M-STRUCTURED-AI-OUTPUT: ~370 LOC
- M-STDLIB-ZIP: ~315 LOC impl + ~388 LOC tests
- M-STDLIB-XML: ~530 LOC impl + ~530 LOC tests

### Velocity
- Recent average: ~400-600 LOC/day (implementation + tests)
- This sprint: ~350 LOC estimated (well within capacity)

### What Exists (Don't Duplicate)
- `internal/telemetry/` — Init, StartSpan, Tracer, resource attributes
- `internal/observatory/otlp_receiver.go` — HTTP endpoint, span storage
- `cmd/ailang/main.go` — Root span `"ailang run: <filename>"` with `ctx` propagation
- `internal/trace/collector.go` — Accumulates `[]TraceEvent` with enter/exit pairs + depth
- Tracer convention: `telemetry.Tracer("ailang.<component>")`
- Span naming: `<component>.<phase>` (e.g., `compile.parse`, `exec.turn`)

## Proposed Milestones

### Milestone 1: OTEL Emitter Core
**Goal:** Create `EmitOTELSpans()` that converts collected `TraceEvent` pairs into real OTEL spans with proper parent-child hierarchy.
**Estimated:** ~180 LOC implementation + ~150 LOC tests = ~330 LOC
**Duration:** ~3 hours

**Key Design Decision — Batch Post-Execution Emission:**

The collector accumulates a flat `[]TraceEvent` list during execution. After execution completes, `EmitOTELSpans` walks the list and reconstructs the span tree:

1. Walk events sequentially
2. `function_enter` → start a new span, push onto stack
3. `effect` / `contract_check` / `budget_delta` → add as child span or span event on current stack top
4. `function_exit` → end current span, pop stack
5. `module_start/end` → root span wrapping everything

This is simpler and safer than streaming spans during evaluation (which would require threading `context.Context` through the evaluator — a large refactor).

**Tasks:**

**Task 1: `internal/trace/otel_emitter.go` (~180 LOC)**

```go
package trace

import (
    "context"
    "fmt"
    "strings"
    "time"

    "go.opentelemetry.io/otel/attribute"
    oteltrace "go.opentelemetry.io/otel/trace"
)

// EmitOTELSpans converts collected trace events into OTEL spans.
// Called after program execution completes (batch emission).
// The parentCtx should carry the root "ailang run" span so program
// spans appear as children of the run command.
func EmitOTELSpans(parentCtx context.Context, tracer oteltrace.Tracer, events []TraceEvent, baseTime time.Time) error
```

Span mapping:
| Trace Events | OTEL Span | Attributes |
|---|---|---|
| `function_enter` + `function_exit` | `eval.function.<name>` | `ailang.function.name`, `.args`, `.result`, `.depth` |
| `effect` | `eval.effect.<effectName>.<opName>` | `ailang.effect.name`, `.op`, `.args`, `.result` |
| `contract_check` | SpanEvent on parent function | `ailang.contract.kind`, `.passed`, `.message` |
| `budget_delta` | Attributes on preceding effect span | `ailang.budget.effect`, `.used`, `.limit`, `.remaining` |
| `module_start` + `module_end` | `eval.module.<name>` | `ailang.module.name`, `.caps` |
| `error` | SpanEvent with Error status | `ailang.error.message`, `.location` |

Algorithm:
- Maintain a stack of `(context.Context, oteltrace.Span)` for nesting
- Each `function_enter` creates a child span from current stack top
- `function_exit` ends the span, pops the stack
- `effect` creates a short-lived child span of current function
- `contract_check` and `budget_delta` attach to current span as events/attributes
- Timestamps reconstructed from `baseTime + event.TimestampNS`

**Task 2: `internal/trace/otel_emitter_test.go` (~150 LOC)**

- Test with `sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))` and `tracetest.NewInMemoryExporter()`
- Verify span names: `eval.function.factorial`, `eval.effect.IO.println`, `eval.module.hello`
- Verify parent-child relationships (function spans nest correctly)
- Verify attributes round-trip (function args, effect names, budget values)
- Verify contract checks appear as span events
- Verify empty events list produces no spans
- Verify depth tracking matches span nesting

### Milestone 2: CLI Wiring + Multi-Mode Flag
**Goal:** Wire `EmitOTELSpans` into `cmd/ailang/main.go` and extend `--emit-trace` to support `otel`, `jsonl,otel`, and `auto`.
**Estimated:** ~30 LOC
**Duration:** ~30 minutes

**Tasks:**

**Task 1: Update `cmd/ailang/main.go`**

After the existing JSONL output block (around line 670), add OTEL emission:

```go
// M-TRACE-EXPORT Phase 2: Emit OTEL spans if requested
if emitTrace != "" && effCtx.Trace != nil {
    events := effCtx.Trace.Events()

    // JSONL output (existing)
    if strings.Contains(emitTrace, "jsonl") && len(events) > 0 {
        if err := ailtrace.WriteJSONL(os.Stdout, events); err != nil { ... }
    }

    // OTEL span emission (new)
    if strings.Contains(emitTrace, "otel") || emitTrace == "auto" {
        if len(events) > 0 {
            evalTracer := otel.Tracer("ailang.eval")
            if err := ailtrace.EmitOTELSpans(ctx, evalTracer, events, effCtx.Trace.BaseTime()); err != nil { ... }
        }
    }
}
```

Also need: expose `BaseTime()` on `Collector` (returns `c.startTime`) — 3 lines in collector.go.

**Flag behavior:**
| `--emit-trace` value | JSONL | OTEL |
|---|---|---|
| `jsonl` | yes | no |
| `otel` | no | yes |
| `jsonl,otel` | yes | yes |
| `auto` | no | yes (if OTEL exporter configured) |

**Acceptance Criteria:**
- [ ] `ailang run --emit-trace otel --caps IO --entry main hello.ail` emits OTEL spans (visible in Cloud Trace when `GOOGLE_CLOUD_PROJECT` set)
- [ ] `ailang run --emit-trace jsonl,otel` emits both JSONL to stdout and OTEL spans
- [ ] `ailang run --emit-trace jsonl` still works exactly as before (no regression)
- [ ] Program-level spans appear as children of the `"ailang run: <filename>"` root span
- [ ] Function spans have correct parent-child nesting (depth matches)
- [ ] Effect spans are children of their enclosing function span
- [ ] Contract checks appear as span events on the function span
- [ ] Budget deltas appear as attributes on effect spans
- [ ] No performance regression when OTEL is not configured (no spans created)
- [ ] `go test ./internal/trace/` passes (all existing + new tests)
- [ ] `go build ./...` clean
- [ ] `go test ./cmd/ailang/` passes

**Risks:**
- Span timing reconstruction: events have relative `timestamp_ns` from collector start. Need to convert to absolute `time.Time` for OTEL spans. **Mitigation:** `baseTime.Add(time.Duration(event.TimestampNS))` is straightforward.
- Deep recursion producing many spans: factorial(1000) → 1000 spans. **Mitigation:** Add optional depth limit (e.g., only emit spans for depth ≤ 50). Not blocking for M1 — can be Phase 2.1 if needed.

## Success Metrics
- All existing tests passing: `go test ./...`
- New tests: 6+ test cases in `otel_emitter_test.go`
- OTEL spans visible in Cloud Trace for `ailang run --emit-trace otel`
- `ailang chains view` shows eval spans when running under coordinator
- No performance regression (batch emission is post-execution)

## Open Questions
- **Depth limit for span emission?** Deep recursion could generate thousands of spans. Suggest: emit all for now, add `--trace-max-depth N` flag if it becomes a problem in practice.
- **Should `--emit-trace auto` be the default?** When OTEL is configured, program traces could always emit. Suggest: no, keep explicit opt-in for now.

## Notes
- The `internal/trace/` package currently has zero OTEL dependencies (stdlib only). Phase 2 adds the OTEL dependency, which is fine since the package is already internal.
- The existing `cmd/ailang/main.go` already has `ctx` carrying the root span and `otel.Tracer("ailang.cli")` — the eval tracer just needs `otel.Tracer("ailang.eval")`.
- The OTLP receiver in observatory already handles arbitrary spans — no changes needed to the ingestion path.
