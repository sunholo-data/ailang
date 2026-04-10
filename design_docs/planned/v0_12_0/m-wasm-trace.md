# M-WASM-TRACE: WASM Trace Effect Handler

**Status**: Planned
**Target**: v0.12.0
**Priority**: P1 (Medium — enables real-time observability in browser-hosted AILANG)
**Estimated**: 3-4 days
**Dependencies**: M-TRACE-EXPORT (implemented v0.8.0), WASM runtime (implemented)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Trace output is a side-channel; execution semantics unchanged |
| A2: Replayability | +1 | Extends trace capture to WASM — browser runs become replayable |
| A3: Effect Legibility | +1 | Trace is modeled as an explicit effect, not hidden instrumentation |
| A4: Explicit Authority | 0 | No new ambient access; JS callbacks are opt-in |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | Single-threaded WASM, no concurrency impact |
| A7: Machines First | +1 | Structured JSON events consumable by AI/tooling |
| A8: Minimal Syntax | 0 | No new syntax (effect handler + stdlib module) |
| A9: Cost Visibility | +1 | Budget events visible in real-time via trace stream |
| A10: Composability | +1 | std/trace spans compose with existing effect system |
| A11: Structured Failure | 0 | Error events already captured in trace schema |
| A12: System Boundary | +1 | Explicit boundary crossing: AILANG → JS via effect handler |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Trace emission is observation-only, no execution changes
- [x] A3 (Effects): Trace is an explicit effect, registered via `setEffectHandler`
- [x] A4 (Authority): No ambient access — JS must opt in to receive events
- [x] A7 (Machines First): JSON event schema, not human-pretty-printed

## Problem Statement

AILANG's `--emit-trace jsonl,otel` works on CLI — it captures `function_enter`, `function_exit`, `effect`, `contract_check`, and `budget_delta` events as structured JSONL, with optional OTEL export. **None of this is available in WASM mode.**

**Current State:**
- CLI trace infrastructure is mature (`internal/trace/` schema + collector, `--emit-trace` flag)
- WASM runtime (`cmd/wasm/`) has JS effect handler bridge (`registerJSEffectHandler`)
- WASM has no access to the `Collector` or any trace event stream
- Co-presenter demo emits traces manually via `println` — only captures high-level events

**Impact:**
- Browser-hosted AILANG programs have zero observability into internal execution
- Co-presenter demo (live presentation AI assistant) cannot show real-time trace waterfall of AI decisions
- WASM-based tooling (playground, embedded editors) cannot offer trace debugging
- Axiom A2 (Replayability) partially unfulfilled in browser context

## Goals

**Primary Goal:** Expose AILANG execution trace events to JavaScript in WASM mode via the existing effect handler bridge.

**Success Metrics:**
- JS receives all 6 trace event types (`function_enter`, `function_exit`, `effect`, `contract_check`, `budget_delta`, `error`) in real-time during WASM execution
- `std/trace` module allows AILANG code to emit custom `span` and `event` annotations
- Co-presenter demo renders trace waterfall from native AILANG events (not manual println)
- Trace events include span IDs enabling downstream OTEL forwarding from JS

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Trace delivery: JS callback vs SharedArrayBuffer | Determines latency and complexity of JS integration | human | design | high |
| Span ID format: W3C trace-context compatible? | Affects whether JS can forward to OTEL collectors | agent | compile | med |
| std/trace as stdlib (.ail) vs builtin effect | Determines whether user code can emit custom spans | human | design | med |
| Event batching: per-event vs buffered flush | Affects real-time waterfall latency vs overhead | agent | runtime | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] JS callback delivery (not SharedArrayBuffer) — simpler, matches existing `registerJSEffectHandler` pattern
- [x] std/trace: stdlib module using Trace effect (decided: effect-based approach, AILANG-idiomatic)
- [ ] Span ID format decision (W3C traceparent or AILANG-native IDs)

## Solution Design

### Overview

Three-tier approach, each independently useful:

1. **Tier 1 — Trace callback bridge** (core): Wire the existing `trace.Collector` into WASM, expose events to JS via a `Trace` effect handler callback
2. **Tier 2 — `std/trace` module** (language-level): Let AILANG code emit custom spans and events that merge into the trace stream
3. **Tier 3 — OTEL-compatible span IDs** (stretch): Add span/parent IDs to trace events so JS can forward to Cloud Trace

### Architecture

**Tier 1: WASM Trace Callback Bridge**

The existing `trace.Collector` accumulates events during evaluation. In CLI mode, events are flushed to JSONL/OTEL at program end. For WASM, we add a **streaming callback mode**: each event is also dispatched to a registered JS function as it occurs.

```
AILANG Evaluator
    │
    ├─ trace.Collector.Record*()  ← existing
    │       │
    │       ├─ append to events[]  ← existing (for post-hoc access)
    │       └─ onEvent callback()  ← NEW: real-time dispatch
    │               │
    │               └─ JS: window.ailangSetTraceHandler(fn)
    │                       fn({event: "function_enter", ...})
```

**Tier 2: `std/trace` Module**

A stdlib AILANG module that uses a `Trace` effect to emit custom spans:

```ailang
module std/trace

import std/list (map)

effect Trace {
  spanStart(name: String, attrs: Map[String, String]) -> String  // returns span ID
  spanEnd(spanId: String) -> Unit
  event(name: String, data: String) -> Unit
}

export func span(name: String, body: () -> a) -> a with Trace {
  let id = perform Trace.spanStart(name, {})
  let result = body()
  perform Trace.spanEnd(id)
  result
}

export func event(name: String, data: String) -> Unit with Trace {
  perform Trace.event(name, data)
}
```

**Tier 3: OTEL Span IDs**

Extend `TraceEvent` with optional `span_id` and `parent_span_id` fields (W3C-compatible hex strings). The Collector generates these; JS can use them to construct `traceparent` headers for forwarding.

### Implementation Plan

**Phase 1: Streaming Collector + WASM Bridge** (~6 hours)
- [ ] Add `OnEvent func(TraceEvent)` callback field to `trace.Collector`
- [ ] Call `OnEvent` in each `Record*` method (nil-safe)
- [ ] Add `ailangSetTraceHandler(callback)` to `cmd/wasm/main.go` — registers a JS function that receives trace events as JSON objects
- [ ] Create Collector in WASM eval path, wire OnEvent to JS dispatch
- [ ] Convert `TraceEvent` to `js.Value` for zero-copy delivery to JS
- [ ] Test: WASM test that registers handler and verifies events received

**Phase 2: `std/trace` Module** (~4 hours)
- [ ] Create `std/trace.ail` with `Trace` effect, `span`, `event` exports
- [ ] Register default `Trace` effect handler in WASM that emits to Collector
- [ ] Register default `Trace` effect handler in CLI that emits to Collector
- [ ] Test: AILANG program using `std/trace` produces custom span events
- [ ] Test: Custom spans appear in `--emit-trace jsonl` output

**Phase 3: OTEL Span IDs** (~3 hours)
- [ ] Add `SpanID` and `ParentSpanID` fields to `TraceEvent` schema
- [ ] Generate 16-hex-char span IDs in Collector (maintain stack for parent tracking)
- [ ] Include trace ID (single per execution) for W3C `traceparent` construction
- [ ] Test: Span IDs form valid parent-child tree
- [ ] Document JS-side OTEL forwarding pattern

### Files to Modify/Create

**New files:**
- `std/trace.ail` — std/trace module with Trace effect (~30 LOC)
- `cmd/wasm/trace.go` — WASM trace bridge (JS handler registration, event dispatch) (~120 LOC)
- `internal/trace/collector_test.go` — Additional tests for OnEvent callback (~50 LOC)

**Modified files:**
- `internal/trace/collector.go` — Add `OnEvent` callback, span ID generation (~+40 LOC)
- `internal/trace/schema.go` — Add `SpanID`, `ParentSpanID`, `TraceID` fields (~+10 LOC)
- `cmd/wasm/main.go` — Register `ailangSetTraceHandler` global function (~+15 LOC)
- `cmd/wasm/effects.go` — Register default Trace effect handler (~+30 LOC)

## Examples

### Example 1: Real-Time Trace Waterfall (JS)

```javascript
// Browser: register trace handler before running AILANG
window.ailangSetTraceHandler((event) => {
  switch (event.event) {
    case "function_enter":
      waterfall.pushSpan(event.function.name, event.span_id, event.depth);
      break;
    case "function_exit":
      waterfall.closeSpan(event.span_id, event.function.duration_ns);
      break;
    case "effect":
      waterfall.addEffect(event.effect.effect_name, event.effect.op_name);
      break;
    case "budget_delta":
      waterfall.updateBudget(event.budget.used, event.budget.limit);
      break;
  }
});

// Run AILANG program — events stream to handler in real-time
wasmRepl.eval('import co_presenter (run)\nrun()');
```

### Example 2: Custom Spans from AILANG Code

```ailang
module co_presenter

import std/trace (span, event)
import std/json (encode)

func analyzeSentiment(text: String) -> Sentiment with AI, Trace {
  span("analyzeSentiment", \(). {
    event("input_length", toString(length(text)))
    let result = perform AI.complete("Classify sentiment: " ++ text)
    event("model_response", result)
    parseSentiment(result)
  })
}
```

### Example 3: OTEL Forwarding from JS (Tier 3)

```javascript
// Forward AILANG trace events to Cloud Trace via OTEL HTTP exporter
window.ailangSetTraceHandler((event) => {
  if (event.span_id) {
    otelBatcher.addSpan({
      traceId: event.trace_id,
      spanId: event.span_id,
      parentSpanId: event.parent_span_id,
      name: event.function?.name || event.effect?.op_name || event.event,
      startTimeUnixNano: event.timestamp_ns,
      endTimeUnixNano: event.timestamp_ns + (event.function?.duration_ns || 0),
    });
  }
});
```

## Success Criteria

- [ ] JS handler receives all 6 event types during WASM AILANG execution
- [ ] Events arrive in real-time (not batched at program end)
- [ ] `std/trace` `span()` and `event()` produce trace events in both CLI and WASM
- [ ] Span IDs form valid parent-child tree (Tier 3)
- [ ] Co-presenter demo renders trace waterfall from native events
- [ ] No performance regression: WASM execution with no handler registered adds <1% overhead
- [ ] All tests passing
- [ ] Documentation updated (telemetry guide + WASM guide)
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- `trace.Collector` OnEvent callback fires for each Record* call
- Span ID generation produces unique, correctly-nested IDs
- Nil handler (no JS callback registered) has zero overhead

**Integration tests:**
- WASM binary with trace handler receives events from a simple AILANG program
- `std/trace` span/event functions produce correct TraceEvent output
- `--emit-trace jsonl` includes custom spans from `std/trace`

**Manual testing:**
- Co-presenter demo trace waterfall renders correctly
- Browser console shows structured events during WASM execution
- OTEL forwarding to Cloud Trace produces valid spans (Tier 3)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Event batching strategy (per-event vs micro-batch) — agent may choose based on profiling
- JS API naming (`ailangSetTraceHandler` vs `ailang.onTrace`) — agent may choose
- `TraceEvent` → `js.Value` conversion strategy (JSON marshal vs field-by-field) — agent may choose based on performance

## Non-Goals

**Not attempted in this feature:**
- Full OTEL SDK in WASM — too heavy; JS-side forwarding is sufficient
- Trace replay in browser — CLI `ailang replay` is sufficient for now
- Trace persistence/storage in browser — IndexedDB integration is a separate feature
- Modifying the trace schema for non-WASM use cases — schema extensions are additive only

## Timeline

**Day 1** (~6 hours):
- Phase 1: Streaming Collector + WASM bridge
- End-to-end: JS receives events from WASM AILANG execution

**Day 2** (~4 hours):
- Phase 2: `std/trace` module
- Custom spans working in both CLI and WASM

**Day 3** (~3 hours):
- Phase 3: OTEL span IDs
- Documentation and examples

**Day 4** (~2 hours):
- Co-presenter demo integration testing
- Polish and edge cases

**Total: ~15 hours across 3-4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Per-event JS callback overhead in hot loops | Med | Nil-check fast path; optional batching; benchmark with/without handler |
| `syscall/js` Value conversion GC pressure | Med | Reuse js.Value objects where possible; profile allocation |
| Trace effect in std/trace conflicts with existing effect names | Low | Check effect registry; "Trace" is unlikely to conflict |
| Span ID generation overhead in Collector | Low | Use fast RNG (math/rand); only generate when span IDs enabled |

## Related Documents

<!-- Auto-populated by Ollama neural search on "wasm trace" -->

**Implemented (may inform design):**
- [m-trace-replay-phase3-sprint-plan.md](../../implemented/v0_8_0/m-trace-replay-phase3-sprint-plan.md) — Trace replay architecture
- [trace-test.md](../../implemented/v0_7_1/trace-test.md) — Trace testing framework
- [m-trace-instrumentation.md](../../implemented/v0_7_1/m-trace-instrumentation.md) — OTEL bridge to TraceRegistry
- [m-trace-export.md](../../implemented/v0_8_0/m-trace-export.md) — --emit-trace JSONL/OTEL implementation

**Planned (check for overlap):**
- [m-provenance-tracing.md](../v0_11_0/m-provenance-tracing.md) — Data provenance tracking (complementary, not overlapping)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/trace/schema.go` — Current trace event schema (6 event types)
- `internal/trace/collector.go` — Current Collector implementation
- `cmd/wasm/main.go` — WASM entry point
- `cmd/wasm/effects.go` — JS effect handler bridge
- Agent message `676e28e2` — Original feature request from co-presenter-demo

## Future Work

- **Browser trace viewer component** — Reusable React/Web Component for trace waterfall rendering
- **Trace-driven debugging in playground** — Step through execution using trace events
- **IndexedDB trace persistence** — Store traces in browser for offline analysis
- **Bidirectional trace streaming** — JS can inject mock trace events for testing

---

**Document created**: 2026-04-10
**Last updated**: 2026-04-10
