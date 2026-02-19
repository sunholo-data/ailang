# Sprint Plan: M-WASM-STREAM-BRIDGE

**Sprint ID**: M-WASM-STREAM-BRIDGE
**Duration**: 1.5 days (12 hours)
**Risk Level**: Low (well-scoped changes following existing patterns)
**Design Doc**: [m-wasm-stream-bridge.md](m-wasm-stream-bridge.md)
**GitHub Issues**: #137, #138

## Sprint Summary

**Goal**: Fix three bugs preventing `std/stream` from working in browser WASM by aligning stream builtins with the existing FS/Net dispatch pattern, fixing the JS API mismatch, and adding closure-to-JS bridging.

**Key Deliverables**:
1. Stream builtins dispatch through `effects.Call()` (same pattern as FS)
2. `ailang-repl.js` supports both 2-arg and 3-arg `setEffectHandler`
3. AILANG closures are callable from JS via `js.FuncOf`

**Total LOC**: ~255 (implementation + tests)

## Current Status Analysis

### Velocity
- Recent average: ~200-400 LOC/day (from M-PROCESS: 650 LOC, M-STREAM-PHASE2-DX: 280 LOC)
- 255 LOC is well within 1.5-day capacity
- Changes follow established patterns (FS builtins as template)

### Key Files (already analyzed)
- `internal/builtins/stream.go` — 9 stream builtins with direct `Impl` refs (the bug)
- `internal/builtins/fs.go` — correct `effects.Call()` pattern to follow
- `internal/effects/ops.go` — Registry + `Call()` dispatch
- `cmd/wasm/effects.go` — WASM bridge (`ailangValueToJS`, `registerJSEffectHandler`)
- `web/ailang-repl.js` — JS wrapper (3-arg bug at line 232)

## Milestones

### M1: Stream Builtin Registry Dispatch (4 hours, ~50 LOC)

**Goal**: Route all 9 stream builtins through `effects.Call()` so WASM JS handlers are used when registered.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `internal/builtins/stream.go` | ~30 | Change 9 `Impl` fields to `effects.Call()` wrappers |
| `internal/builtins/stream_test.go` | ~20 | Test that stream builtins go through Registry |

**Tasks**:
- [ ] Change `registerStreamConnect` Impl to `effects.Call(ctx, "Stream", "connect", args)`
- [ ] Change `registerStreamSSEConnect` Impl to `effects.Call(ctx, "Stream", "sseConnect", args)`
- [ ] Change `registerStreamSSEPost` Impl to `effects.Call(ctx, "Stream", "ssePost", args)`
- [ ] Change `registerStreamSend` Impl to `effects.Call(ctx, "Stream", "send", args)`
- [ ] Change `registerStreamTransmitBinary` Impl to `effects.Call(ctx, "Stream", "transmitBinary", args)`
- [ ] Change `registerStreamOnEvent` Impl to `effects.Call(ctx, "Stream", "onEvent", args)`
- [ ] Change `registerStreamRunEventLoop` Impl to `effects.Call(ctx, "Stream", "runEventLoop", args)`
- [ ] Change `registerStreamClose` Impl to `effects.Call(ctx, "Stream", "close", args)`
- [ ] Change `registerStreamGetStatus` Impl to `effects.Call(ctx, "Stream", "status", args)`
- [ ] Verify operation names match `effects/stream.go` init() registration
- [ ] Run `go test ./internal/builtins/... -run Stream`
- [ ] Run `go test ./internal/effects/... -run Stream`
- [ ] Run `make test` for full regression

**Acceptance Criteria**:
- [ ] All 9 stream builtins dispatch through `effects.Call()`
- [ ] Registry override (as done by WASM bridge) takes effect for stream ops
- [ ] All existing stream tests pass unchanged
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks**:
- Double capability check (`effects.Call` + `StreamConnect`) — Low impact, idempotent

---

### M2: Fix ailang-repl.js API Mismatch (2 hours, ~25 LOC)

**Goal**: Support both 2-arg `(effectName, handlersObj)` and 3-arg `(effectName, operation, handler)` calling conventions.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `web/ailang-repl.js` | ~25 | Update `setEffectHandler` with accumulator pattern |

**Tasks**:
- [ ] Add `#pendingHandlers = {}` accumulator field to class
- [ ] Update `setEffectHandler` to detect 2-arg vs 3-arg form
- [ ] For 3-arg form: accumulate into object, then call native 2-arg API
- [ ] For 2-arg form: pass through directly (existing behavior)
- [ ] Update JSDoc to document both calling conventions
- [ ] Test both forms manually in browser console

**Acceptance Criteria**:
- [ ] `setEffectHandler('Stream', { connect: fn })` works (2-arg, existing)
- [ ] `setEffectHandler('Stream', 'connect', fn)` works (3-arg, new)
- [ ] No panic on 3-arg call
- [ ] JSDoc documents both forms

**Risks**:
- None — pure JS change, no Go compilation needed

---

### M3: Closure-to-JS Bridge (5 hours, ~150 LOC)

**Goal**: AILANG closures passed to JS effect handlers arrive as callable JS functions.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `cmd/wasm/effects.go` | ~30 | Add `*eval.ClosureValue` case to `ailangValueToJS` |
| `internal/repl/apply_closure.go` | ~40 | New `ApplyClosure` method on REPL |
| `internal/repl/apply_closure_test.go` | ~80 | Tests for closure application |

**Tasks**:
- [ ] Add `ApplyClosure(closure eval.Value, args []eval.Value) (eval.Value, error)` to REPL
- [ ] Implement closure invocation via evaluator's apply mechanism
- [ ] Add `*eval.ClosureValue` case in `ailangValueToJS` wrapping as `js.FuncOf`
- [ ] Handle return value conversion (ailangValueToJS on result)
- [ ] Handle errors (console.error, return null)
- [ ] Write unit tests for `ApplyClosure` with simple closures
- [ ] Write unit tests for `ApplyClosure` with multi-arg closures
- [ ] Document `js.FuncOf` leak caveat (no auto-release)

**Acceptance Criteria**:
- [ ] `ailangValueToJS` wraps closures as callable `js.FuncOf`
- [ ] JS can invoke AILANG closures and receive return values
- [ ] Error in closure execution → console.error, returns null (no panic)
- [ ] `go test ./internal/repl/... -run ApplyClosure` passes
- [ ] WASM build succeeds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`

**Risks**:
- `js.FuncOf` memory leaks — Document as known limitation, not blocking
- Re-entrancy in evaluator — WASM is single-threaded, should be safe
- Need access to evaluator from REPL — REPL already has evaluator reference

---

### M4: Integration & Documentation (1 hour, ~30 LOC)

**Goal**: Verify end-to-end and update docs.

**Tasks**:
- [ ] Build WASM binary: `GOOS=js GOARCH=wasm go build -o web/ailang.wasm ./cmd/wasm/`
- [ ] Update CHANGELOG.md with M-WASM-STREAM-BRIDGE entry
- [ ] Add JSDoc for `setEffectHandler` auto-grants capability behavior
- [ ] Run `make lint` and `make test`

**Acceptance Criteria**:
- [ ] WASM binary builds successfully
- [ ] CHANGELOG.md updated
- [ ] All tests pass
- [ ] All linting passes

---

## Success Metrics

- All existing tests passing: `make test` ✅
- Linting clean: `make lint` ✅
- WASM binary builds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/` ✅
- Stream builtins dispatch through Registry ✅
- JS wrapper handles both API forms ✅
- Closures are callable from JS ✅

## Dependencies

- None — all changes are to existing files with established patterns

## Open Questions

- None — design is fully specified from bug reports and source analysis
