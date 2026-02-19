# M-WASM-STREAM-BRIDGE: WASM Stream Effect Handler Bridge

**Status**: Planned
**Target**: v0.8.2
**Priority**: P1 (Medium-High) - Blocks Gemini Live browser demo
**Estimated**: 2 days (8h implementation + 4h testing + 2h docs + buffer)
**Dependencies**: M-STREAM-BIDI (v0.8.1, already implemented)
**GitHub Issues**: #137, #138

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics; same AILANG code runs identically on CLI and WASM |
| A2: Replayability | 0 | No impact on traces; JS-bridged effects produce same trace events as native |
| A3: Effect Legibility | +1 | Stream effects become visible and functional in WASM — currently silently fail with E_STREAM_NO_CONTEXT |
| A4: Explicit Authority | 0 | Same capability model; `setEffectHandler` already auto-grants capabilities |
| A5: Bounded Verification | 0 | No change to verification model |
| A6: Safe Concurrency | 0 | Single-threaded WASM; no concurrency changes |
| A7: Machines First | +1 | Same `std/stream` module works on CLI and browser — no platform-specific code needed |
| A8: Minimal Syntax | +1 | No new syntax; fixes existing patterns to work correctly |
| A9: Cost Visibility | 0 | Budget tracking preserved through `effects.Call()` dispatch |
| A10: Composability | +1 | Stream effects compose with JS effect handlers the same way IO/FS/Net already do |
| A11: Structured Failure | +1 | Replaces panic (`syscall/js: call of Value.Get on string`) with proper error handling |
| A12: System Boundary | +1 | AILANG↔JS boundary for closures becomes explicit and bidirectional |

**Net Score: +6** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Effects become MORE visible (currently broken/hidden in WASM)
- [x] A4 (Authority): No ambient access granted; same capability model
- [x] A7 (Machines First): Improves cross-platform portability

## Problem Statement

Three bugs prevent `std/stream` from working in browser WASM, blocking the Gemini Live demo from using the same AILANG module that works on CLI.

**Bug 1: Stream builtins bypass effect handler registry**

Stream builtins (`_stream_connect`, `_stream_send`, `_stream_onEvent`, etc.) are registered with direct function references:

```go
// internal/builtins/stream.go
RegisterEffectBuiltin(BuiltinSpec{
    Impl: effects.StreamConnect,  // Direct function reference — bypasses Registry
})
```

The runtime calls `builtinSpec.Impl(ctx, args)` ([internal/runtime/builtins.go:84](internal/runtime/builtins.go#L84)), which invokes `effects.StreamConnect()` directly. This function checks `ctx.Stream == nil` and returns `E_STREAM_NO_CONTEXT` in WASM (gorilla/websocket doesn't work in browsers).

Meanwhile, `registerJSEffectHandler()` calls `effects.RegisterOp("Stream", "connect", jsHandler)` which correctly overwrites the Registry entry — but stream builtins never consult the Registry.

**Contrast with FS builtins** which work correctly in WASM:

```go
// internal/builtins/fs.go — CORRECT pattern
RegisterEffectBuiltin(BuiltinSpec{
    Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        return effects.Call(ctx, "FS", "readFile", args)  // Goes through Registry!
    },
})
```

**Bug 2: ailang-repl.js 3-arg/2-arg API mismatch**

The JS wrapper passes 3 arguments:
```javascript
// web/ailang-repl.js:232
return window.ailangSetEffectHandler(capability, operation, handler);
```

But the Go function expects 2 arguments (effectName, handlersObject):
```go
// cmd/wasm/effects.go:174
effectName := args[0].String()
handlers := args[1]  // Expects JS object, gets operation string
```

When called with 3 args, `args[1]` is the operation name string. The Go code calls `Object.keys(handlers)` on it → panic: `syscall/js: call of Value.Get on string`.

**Bug 3: ailangValueToJS cannot convert closures**

`ailangValueToJS` ([cmd/wasm/effects.go:116](cmd/wasm/effects.go#L116)) falls through to `formatValue(v)` for closures, converting them to their string representation. JS cannot invoke string values.

This blocks `onEvent(conn, handler)` where AILANG passes a closure callback to be invoked by JS on each WebSocket message.

**Current State:**
- FS/IO/Net effects work in WASM via `effects.Call()` → Registry → JS handlers
- Stream effects bypass Registry → always fail with `E_STREAM_NO_CONTEXT`
- JS wrapper panics when calling `setEffectHandler` with per-operation syntax
- AILANG closures arrive in JS as strings, not callable functions

**Impact:**
- Gemini Live browser demo blocked — falls back to pure helper functions only
- Any browser app using `std/stream` is broken
- Same AILANG module cannot run on both CLI and browser

## Goals

**Primary Goal:** Make `std/stream` work identically in browser WASM and CLI by routing stream builtins through the effect handler registry.

**Success Metrics:**
1. `std/stream` module runs in browser WASM with JS-provided WebSocket transport
2. `setEffectHandler('Stream', { connect: fn, send: fn, ... })` works without panic
3. AILANG closures passed to JS effect handlers are callable from JS
4. All existing CLI stream tests continue to pass (no regression)
5. Gemini Live browser demo uses same AILANG module as CLI

## Systemic Analysis

**Is this part of a larger pattern?** Yes — stream builtins are the ONLY effect builtins that bypass the Registry. All others (FS, IO, Net, Clock, Debug, Process) either go through `effects.Call()` or don't need WASM bridging.

| Effect | Builtin Pattern | WASM Compatible? |
|--------|----------------|-----------------|
| FS | `effects.Call(ctx, "FS", ...)` | ✅ Yes |
| IO | Direct `Impl` but no native context check | ✅ Yes (print → console.log) |
| Net | `effects.Call(ctx, "Net", ...)` | ✅ Yes |
| Stream | Direct `Impl` → `ctx.Stream == nil` | ❌ **No** |

The fix is systemic: align Stream with the pattern already used by FS/Net.

## Solution Design

### Overview

Three changes, in dependency order:

1. **Route stream builtins through `effects.Call()`** — same pattern as FS builtins
2. **Fix ailang-repl.js to use 2-arg API** — accumulate per-operation calls into object
3. **Add closure support to `ailangValueToJS`** — wrap as `js.FuncOf`

### Architecture

**Change 1: Stream builtin dispatch (internal/builtins/stream.go)**

Replace direct function references with `effects.Call()` wrappers:

```go
// BEFORE (bypasses Registry):
RegisterEffectBuiltin(BuiltinSpec{
    Impl: effects.StreamConnect,
})

// AFTER (goes through Registry):
RegisterEffectBuiltin(BuiltinSpec{
    Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        return effects.Call(ctx, "Stream", "connect", args)
    },
})
```

This means:
- CLI: `effects.Call()` → Registry → `effects.StreamConnect()` (native Go, unchanged)
- WASM: `effects.Call()` → Registry → JS handler (overwritten by `registerJSEffectHandler`)

**Note:** `effects.StreamConnect` does its own `RequireCapWithBudget` check. Since `effects.Call()` also checks capabilities, the stream implementations should skip the redundant check when called through `Call()`. However, to keep the change minimal and avoid breaking direct callers, we'll accept the double-check for now — `RequireCapWithBudget` is idempotent.

**Change 2: Fix ailang-repl.js API (web/ailang-repl.js)**

Support both 2-arg and 3-arg forms by accumulating per-operation handlers:

```javascript
// Internal accumulator for per-operation registration
#pendingHandlers = {};

setEffectHandler(capability, operationOrHandlers, handler) {
    if (typeof operationOrHandlers === 'object') {
        // 2-arg form: setEffectHandler('Stream', { connect: fn, send: fn })
        return window.ailangSetEffectHandler(capability, operationOrHandlers);
    }
    // 3-arg form: setEffectHandler('Stream', 'connect', fn)
    if (!this.#pendingHandlers[capability]) {
        this.#pendingHandlers[capability] = {};
    }
    this.#pendingHandlers[capability][operationOrHandlers] = handler;
    // Register accumulated handlers
    return window.ailangSetEffectHandler(capability, this.#pendingHandlers[capability]);
}
```

**Change 3: Closure-to-JS bridge (cmd/wasm/effects.go)**

Add `*eval.ClosureValue` case to `ailangValueToJS`:

```go
case *eval.ClosureValue:
    // Wrap AILANG closure as callable JS function
    closure := val
    return js.FuncOf(func(this js.Value, jsArgs []js.Value) interface{} {
        // Convert JS args to AILANG values
        ailangArgs := make([]eval.Value, len(jsArgs))
        for i, jsArg := range jsArgs {
            ailangArgs[i] = jsToAILANGValue(jsArg)
        }
        // Apply closure (requires evaluator access)
        result, err := replInstance.repl.ApplyClosure(closure, ailangArgs)
        if err != nil {
            js.Global().Get("console").Call("error", "AILANG closure error: "+err.Error())
            return nil
        }
        return ailangValueToJS(result)
    })
```

**Note:** This requires a new `ApplyClosure(closure, args)` method on the REPL to safely invoke an AILANG closure from outside the evaluator. The REPL already has evaluator access; this just exposes a focused entry point.

### Implementation Plan

**Phase 1: Stream Registry Dispatch** (~3 hours)
- [ ] Update all 9 stream builtin registrations in `internal/builtins/stream.go` to use `effects.Call()`
- [ ] Verify `effects.Call()` handles capability checking (avoid double-check issues)
- [ ] Run existing stream tests: `go test ./internal/effects/... -run Stream`
- [ ] Run existing builtin tests: `go test ./internal/builtins/... -run Stream`

**Phase 2: JS API Fix** (~2 hours)
- [ ] Update `setEffectHandler()` in `web/ailang-repl.js` to handle both 2-arg and 3-arg forms
- [ ] Add accumulation logic for per-operation registration
- [ ] Update JSDoc comments
- [ ] Add unit test for both calling conventions

**Phase 3: Closure Bridge** (~4 hours)
- [ ] Add `*eval.ClosureValue` case to `ailangValueToJS` in `cmd/wasm/effects.go`
- [ ] Add `ApplyClosure(closure *eval.ClosureValue, args []eval.Value)` to REPL
- [ ] Handle ADT constructor values (StreamEvent variants) in `jsToAILANGValue`
- [ ] Add `js.FuncOf` leak prevention (release tracking or weak refs)
- [ ] Test closure round-trip: AILANG → JS → invoke → AILANG result

**Phase 4: Integration & Docs** (~3 hours)
- [ ] Test with Gemini Live browser demo end-to-end
- [ ] Document `setEffectHandler` auto-grants capability
- [ ] Document Stream operation signatures for browser developers
- [ ] Update CHANGELOG.md
- [ ] Add example: `examples/wasm_stream_bridge.ail`

### Files to Modify/Create

**Modified files:**
- `internal/builtins/stream.go` - Change 9 `Impl` fields to use `effects.Call()` (~30 LOC changed)
- `web/ailang-repl.js` - Fix `setEffectHandler` to support 2-arg and 3-arg forms (~20 LOC)
- `cmd/wasm/effects.go` - Add closure case to `ailangValueToJS` (~25 LOC)

**New files:**
- `internal/repl/apply_closure.go` - `ApplyClosure` method (~40 LOC)
- `internal/repl/apply_closure_test.go` - Tests (~60 LOC)
- `cmd/wasm/effects_closure_test.go` - WASM closure bridge tests (~80 LOC)

**Total: ~255 LOC new/changed**

## Examples

### Example 1: Stream Effect Handler Registration (JS)

**Before (panics):**
```javascript
// 3-arg form — panics with "syscall/js: call of Value.Get on string"
repl.setEffectHandler('Stream', 'connect', async (url, config) => {
    const ws = new WebSocket(url);
    return ws;
});
```

**After (both forms work):**
```javascript
// 3-arg form — works, accumulates into object internally
repl.setEffectHandler('Stream', 'connect', async (url, config) => { ... });
repl.setEffectHandler('Stream', 'send', async (conn, msg) => { ... });

// 2-arg form — also works (direct object)
repl.setEffectHandler('Stream', {
    connect: async (url, config) => { ... },
    send: async (conn, msg) => { ... },
    onEvent: async (conn, handler) => { handler(event); },
});
```

### Example 2: AILANG Closure Callback from JS

**Before (closure arrives as string):**
```javascript
repl.setEffectHandler('Stream', 'onEvent', (conn, handler) => {
    console.log(typeof handler);  // "string" — e.g. "<closure>"
    handler(event);               // TypeError: handler is not a function
});
```

**After (closure is callable):**
```javascript
repl.setEffectHandler('Stream', 'onEvent', (conn, handler) => {
    console.log(typeof handler);  // "function"
    const result = handler(eventData);  // Invokes AILANG closure, returns AILANG value
});
```

### Example 3: Same AILANG Module on CLI and Browser

```ailang
module demos/streaming/gemini_live

import std/stream (connect, send, onEvent, runEventLoop, close)
import std/json

let main = \config.
  let conn = connect(config.url, {protocol: "WebSocket", headers: config.headers})
  in match conn {
    Ok(c) =>
      let _ = send(c, json.encode(config.setup))
      in let _ = onEvent(c, \event. match event {
        Binary(data) => handleAudio(data)
        Text(msg) => handleText(msg)
        Closed(code, reason) => false
        _ => true
      })
      in runEventLoop(c)
    Err(e) => printError(e)
  }
```

**CLI:** `ailang run --caps IO,Stream --entry main demos/gemini_live.ail`
**Browser:** JS provides WebSocket transport via `setEffectHandler('Stream', {...})`

## Success Criteria

- [ ] `effects.Call(ctx, "Stream", "connect", args)` dispatches to JS handler when registered
- [ ] `setEffectHandler('Stream', 'connect', fn)` (3-arg) does not panic
- [ ] `setEffectHandler('Stream', {connect: fn})` (2-arg) continues to work
- [ ] AILANG closures arrive in JS as callable functions
- [ ] JS can invoke AILANG closures and receive return values
- [ ] All existing `go test ./internal/effects/... -run Stream` pass
- [ ] All existing `go test ./internal/builtins/... -run Stream` pass
- [ ] Gemini Live browser demo works with `std/stream` module
- [ ] Documentation updated
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests:**
- Stream builtins dispatch through `effects.Call()` (mock Registry swap)
- `ailang-repl.js` 2-arg and 3-arg forms produce correct native calls
- `ailangValueToJS` correctly wraps closures as `js.FuncOf`
- `ApplyClosure` correctly invokes closures with converted args

**Integration tests:**
- Register JS handler → call stream builtin → verify JS handler invoked
- Closure round-trip: AILANG closure → JS function → invoke → AILANG result

**Manual testing:**
- Gemini Live browser demo with `std/stream` module
- Verify no `js.FuncOf` memory leaks (check browser DevTools)

## Non-Goals

**Not in this feature:**
- SSE protocol support in WASM — deferred to M-STREAM-BIDI Phase 2
- ADT constructor values in `jsToAILANGValue` (e.g., `StreamEvent` variants) — can use records/strings for now
- `js.FuncOf` automatic cleanup/release — document as known limitation, fix in follow-up
- Bidirectional ADT marshalling (JS objects ↔ AILANG ADTs) — separate design doc

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Double capability check (Call + StreamConnect) | Low | Idempotent; optimize later if profiling shows overhead |
| `js.FuncOf` memory leaks from closure wrapping | Medium | Document; add release tracking in follow-up |
| `ApplyClosure` re-entrancy (JS calls closure during eval) | Medium | WASM is single-threaded; REPL eval is synchronous; test carefully |
| Breaking existing CLI stream behavior | High | All existing tests must pass; `effects.Call()` routes to same native functions |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_7_1/m-wasm-repl-sprint-plan.md](../../../design_docs/implemented/v0_7_1/m-wasm-repl-sprint-plan.md) — WASM REPL module loading (0.43)
- [design_docs/implemented/v0_7_3/ailang-demo-invoice-processor.md](../../../design_docs/implemented/v0_7_3/ailang-demo-invoice-processor.md) — WASM demo patterns (0.42)

**Planned (check for overlap):**
- [design_docs/planned/v0_8_1/m-stream-bidi-primitives.md](../v0_8_1/m-stream-bidi-primitives.md) — Stream primitives design (0.49)
- [design_docs/planned/v0_8_1/m-stream-bidi-sprint-plan.md](../v0_8_1/m-stream-bidi-sprint-plan.md) — Stream sprint plan (0.44)
- [design_docs/planned/v0_8_1/m-stream-phase2-dx.md](../v0_8_1/m-stream-phase2-dx.md) — Stream DX improvements (0.42)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/builtins/fs.go` — Correct `effects.Call()` pattern to follow
- `internal/effects/ops.go` — Registry and `Call()` dispatch
- `cmd/wasm/effects.go` — WASM effects bridge
- `web/ailang-repl.js` — JS wrapper API

## Future Work

- Bidirectional ADT marshalling (JS ↔ AILANG ADTs like `StreamEvent`)
- `js.FuncOf` lifecycle management (release tracking, weak references)
- SSE streaming in WASM browsers
- Typed JS interop layer (generate TS declarations from AILANG types)

---

**Document created**: 2026-02-19
**Last updated**: 2026-02-19
