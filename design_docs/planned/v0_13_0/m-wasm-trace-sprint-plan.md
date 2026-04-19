# Sprint Plan: M-WASM-TRACE

## Summary
Add real-time trace event streaming from WASM to JavaScript, a `std/trace` AILANG module for custom spans, and OTEL-compatible span IDs — enabling the co-presenter demo's trace waterfall panel.

**Duration:** 3 days
**Dependencies:** trace.Collector (implemented), WASM runtime (implemented)
**Risk Level:** Low — builds on mature trace and WASM infrastructure

## Current Status Analysis

### Completed Recently
- M-PERF6B: 84 LOC in 1 day (gob serialization, stdout buffering, trace overhead fix)
- M-INCREMENTAL-TYPECHECK: 1,790 LOC in 2 days (JSON serialization, cache store, cache wiring)
- M-PERF-DOCPARSE: 96 LOC in 1 day (deferred substitution, GOGC tuning)

### Velocity
- Recent average: ~300-500 LOC/day (implementation + tests)
- Estimated capacity: ~450 LOC for this sprint (lighter than recent perf sprints)

### Remaining from Design Doc
- Phase 1: Streaming Collector + WASM Bridge (~185 LOC)
- Phase 2: std/trace module (~60 LOC)
- Phase 3: OTEL Span IDs (~80 LOC)

## Proposed Milestones

### M1: STREAMING_COLLECTOR_WASM_BRIDGE
**Goal:** Add OnEvent callback to trace.Collector and wire it to JS in WASM
**Estimated:** 120 implementation + 65 tests = ~185 LOC
**Duration:** 1 day

**Tasks:**
1. Add `OnEvent func(TraceEvent)` field to `trace.Collector`
2. Call `OnEvent` in each `Record*` method (nil-safe check)
3. Add `traceEventToJS(TraceEvent) js.Value` converter in `cmd/wasm/trace.go`
4. Register `ailangSetTraceHandler(callback)` global in `cmd/wasm/main.go`
5. Wire: handler sets `OnEvent` on the REPL's EffContext trace Collector
6. Create Collector on WASM REPL init, set on EffContext
7. Tests: verify OnEvent fires for each event type, verify nil handler is zero-cost

**Acceptance Criteria:**
- [ ] `ailangSetTraceHandler(fn)` callable from JS
- [ ] JS callback receives `function_enter`, `function_exit`, `effect`, `contract_check`, `budget_delta`, `error` events
- [ ] Events arrive during execution (not batched at end)
- [ ] No handler registered = zero overhead (nil check)
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- `syscall/js` Value conversion overhead — Mitigation: profile; use direct field setting not JSON marshal
- WASM build tag complexity — Mitigation: keep WASM-specific code in `cmd/wasm/trace.go`

### M2: STD_TRACE_MODULE
**Goal:** Create `std/trace` AILANG module with Trace effect for custom spans and events
**Estimated:** 40 implementation + 20 tests = ~60 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create `std/trace.ail` with `Trace` effect declaration, `span`, `event` exports
2. Register default Trace effect handler in Go (maps to `trace.Collector` calls)
3. Verify it loads in both CLI and WASM stdlib paths
4. Test: AILANG program using `span()` produces `function_enter`/`function_exit` with custom name
5. Test: `event()` produces trace event visible in `--emit-trace jsonl`

**Acceptance Criteria:**
- [ ] `import std/trace (span, event)` works in CLI and WASM
- [ ] `span("name", body)` wraps body with function_enter/exit trace events
- [ ] `event("name", "data")` emits a trace event
- [ ] Custom spans appear in `--emit-trace jsonl` output
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Effect registration timing in WASM stdlib loading — Mitigation: register Go handler before stdlib load

### M3: OTEL_SPAN_IDS
**Goal:** Add W3C-compatible span/parent IDs to trace events for distributed tracing
**Estimated:** 50 implementation + 30 tests = ~80 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `SpanID`, `ParentSpanID`, `TraceID` fields to `TraceEvent` in `schema.go`
2. Generate 16-hex-char span IDs in Collector using `crypto/rand`
3. Maintain span ID stack in Collector (push on enter, pop on exit)
4. Generate single `TraceID` per Collector instance
5. Test: span IDs form valid parent-child tree across nested calls
6. Test: TraceID consistent across all events in one execution

**Acceptance Criteria:**
- [ ] Every trace event has `span_id` and `trace_id`
- [ ] `function_enter`/`function_exit` pairs share same `span_id`
- [ ] Nested calls have correct `parent_span_id`
- [ ] IDs are valid 16-hex-char strings (W3C compatible)
- [ ] Existing `--emit-trace jsonl` output includes new fields
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- `crypto/rand` not available in WASM — Mitigation: use `math/rand` with seed from JS `crypto.getRandomValues`

### M4: DOCS_AND_CHANGELOG
**Goal:** Update documentation, changelog, and verify end-to-end
**Estimated:** ~50 LOC docs
**Duration:** 0.5 day

**Tasks:**
1. Update `docs/docs/guides/telemetry.md` with WASM trace section
2. Update CHANGELOG.md with M-WASM-TRACE entry
3. Update design doc status to implemented
4. Verify `make test` and `make lint` pass

**Acceptance Criteria:**
- [ ] Telemetry guide documents WASM trace handler API
- [ ] CHANGELOG updated
- [ ] Design doc moved to implemented
- [ ] `make ci` clean

## Success Metrics
- Test coverage: all new code has unit tests
- All 6 trace event types stream to JS in real-time
- std/trace module works in both CLI and WASM
- Span IDs form valid parent-child trees
- Zero overhead when no handler registered
