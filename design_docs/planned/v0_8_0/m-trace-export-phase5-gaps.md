# M-TRACE-EXPORT Phase 5: Instrumentation Gaps & Polish

**Status**: Planned
**Priority**: P2 (Low-Medium)
**Estimated**: 1-2 days
**Dependencies**: M-TRACE-EXPORT Phases 1-4 (all implemented)

---

## Context

Phases 1-4 of M-TRACE-EXPORT are complete, delivering:
- JSONL trace collection (`--emit-trace jsonl`)
- OTEL span emission (`--emit-trace otel/auto`)
- Trace replay (`ailang replay`)
- Training data export (`ailang export-training`)

End-to-end verification revealed several instrumentation gaps and polish opportunities.

---

## Gap 1: Effect Events Not Captured for Most IO Operations

**Severity**: Medium
**Impact**: Effect diversity scores are 0.0 for most programs; training data lacks effect-level detail.

**Root cause**: The `effects.Call()` path records effect events, but many IO operations (like `println`, `print`) are handled by direct Go function calls in the builtin implementations. They don't go through `effects.Call()`.

**Current behavior**:
```
module_start
function_enter  std/io.println  (captured via eval TraceRecorder)
function_exit   std/io.println
module_end
```

**Expected behavior**:
```
module_start
function_enter  main
effect          IO.println      ← Missing
budget_delta    IO used=1       ← Missing (if budgets enabled)
function_exit   main
module_end
```

**Fix options**:
1. **Instrument builtin dispatch**: Record effect events when builtins invoke IO/FS/Net operations
2. **Instrument `effects.Call()`**: Verify the call path fires for all effect operations
3. **Hybrid**: Some builtins are "stdlib functions" (no effect), others are "effect operations" — only trace the latter

**Estimated**: ~100 LOC, affects `internal/effects/ops.go` and/or `internal/builtins/`

---

## Gap 2: Replay Step-Through Mode Not Implemented

**Severity**: Low
**Impact**: Design doc mentioned `--step` flag for interactive stepping. Not implemented.

**Decision**: Defer. Step-through requires terminal interactivity which conflicts with the headless/AI-first design. If needed, implement as JSON event-by-event output that a UI could consume:

```bash
ailang replay --stream trace.jsonl  # Output events one at a time with pause
```

---

## Gap 3: Non-Deterministic Effect Flagging

**Severity**: Low
**Impact**: Replay doesn't distinguish "expected mismatch" (Clock.now, IO.readLine) from "unexpected mismatch" (bug).

**Fix**: Add `Deterministic bool` field to `EffectEvent` schema. Known non-deterministic operations:
- `Clock.now` → always non-deterministic
- `IO.readLine` → depends on stdin
- `Net.httpGet` → depends on network
- `FS.readFile` → deterministic if file unchanged

**Estimated**: ~50 LOC in schema + comparator

---

## Gap 4: Cloud Trace Dashboard Integration

**Severity**: Medium
**Impact**: OTEL spans emit correctly but haven't been verified visible in GCP Cloud Trace console.

**Verification needed**:
1. Run `ailang run --emit-trace otel` with `OTLP_GOOGLE_CLOUD_PROJECT=ailang-dev`
2. Open Cloud Trace console for `ailang-dev` project
3. Verify `eval.function.*` and `eval.module.*` spans appear
4. Verify parent-child hierarchy is correct
5. Verify `ailang chains view` can see program-level spans

**Not a code change** — just verification and potential span attribute adjustments.

---

## Gap 5: Duration Missing on module_end Events

**Severity**: Low
**Impact**: Module execution duration isn't recorded (always 0).

**Fix**: Capture `time.Now()` before execution, compute duration after, pass to `RecordModuleEnd`.

**Estimated**: ~10 LOC in `cmd/ailang/main.go`

---

## Gap 6: Trace Collection for Non-Module Files

**Severity**: Low
**Impact**: Single-expression files (REPL-style, non-module) don't get `module_start`/`module_end` events.

Phase 1 design doc noted this as out of scope. Module files are the primary use case. Non-module files get function-level traces but no module envelope.

**Decision**: Keep as-is. Non-module files are for quick testing; module files are for production traces.

---

## Gap 7: Record/Dict Field Ordering Non-Determinism in Traces

**Severity**: Medium
**Impact**: Trace replay comparison must ignore function arguments because Go map iteration order makes record/dict field string representations non-deterministic between runs. This reduces the fidelity of trace-based regression detection.

**Root cause**: AILANG records and dicts are backed by Go maps internally. When `Value.String()` is called during trace capture (for function arguments and results), fields are serialized in whatever order Go's map iteration provides. This order varies between runs.

**Observed behavior**:
```
Run 1: findKey([{key: name, value: JString(Alice)}, {key: age, value: JNumber(30.0)}], name)
Run 2: findKey([{key: age, value: JNumber(30.0)}, {key: name, value: JString(Alice)}], name)
```

Both runs are semantically identical, but string comparison fails.

**Current workaround**: The CI trace comparison (`scripts/verify_examples.go:parseTraceEvents`) only compares event type + function name, ignoring args entirely. This means argument-level regressions go undetected.

**Fix options**:
1. **Sort record fields by key in `Value.String()`**: Canonical ordering ensures deterministic output. Affects `internal/eval/value.go` (the `String()` method on record/dict values). Minimal performance cost (sort at display time only).
2. **Use ordered map for records**: Replace Go `map[string]Value` with a slice-based ordered structure that preserves insertion order. Higher effort, broader impact.
3. **Normalize at trace capture**: Sort fields when building trace event argument strings, not in the core Value type. Keeps core fast, fixes only trace output.

**Recommended**: Option 1 — sort fields by key in `Value.String()`. This is the simplest fix with the broadest benefit (affects all string representations, not just traces). Records with sorted fields are also easier for humans and AI to read.

**Estimated**: ~20 LOC in `internal/eval/value.go`

---

## Implementation Priority

| Gap | Priority | Effort | Impact |
|-----|----------|--------|--------|
| 1. Effect events | High | ~100 LOC | Rich training data |
| 7. Field ordering | High | ~20 LOC | Deterministic traces, full arg comparison in CI |
| 4. Cloud Trace verify | Medium | ~0 LOC | Confidence |
| 5. Module duration | Low | ~10 LOC | Completeness |
| 3. Non-deterministic flags | Low | ~50 LOC | Better replay |
| 2. Step-through | Deferred | ~200 LOC | Interactive debugging |
| 6. Non-module traces | Deferred | ~50 LOC | Edge case |

**Recommended next sprint**: Gap 7 (field ordering — quick win) + Gap 1 (effect events) + Gap 4 (verification) + Gap 5 (module duration).

---

**Document created**: 2026-02-11
