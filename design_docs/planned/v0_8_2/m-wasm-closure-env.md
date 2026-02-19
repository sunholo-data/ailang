# M-WASM-CLOSURE-ENV: Fix Closure Environment Resolution in WASM Bridge

**Status**: Planned
**Target**: v0.8.2
**Priority**: P0 - High (blocks real-time streaming in browser)
**Estimated**: 3-4 hours
**Dependencies**: M-WASM-STREAM-BRIDGE (Phase 1 + Phase 2, completed)

## Bug Report

**Source**: `demos` inbox, msg_20260219 — "Bug: ApplyClosure fails for closures called from js.FuncOf"

When an AILANG closure is passed to a JS effect handler (e.g., `onEvent(conn, handler)`), the closure captures its environment at creation time. When JS later invokes the closure via `ApplyClosure`, functions referenced inside the closure body (e.g., module-scoped helpers like `handleEvent`) resolve to `nil` — causing "undefined variable" errors at runtime.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes nondeterministic failure (closure works in some contexts, not others) |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Effect handler mechanism unchanged |
| A4: Explicit Authority | 0 | Capability model unchanged |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Eliminates confusing runtime failure that blocks AI-generated code |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Closures now compose correctly across WASM boundary |
| A11: Structured Failure | +1 | Replaces opaque "undefined variable" with correct behavior |
| A12: System Boundary | +1 | WASM↔JS boundary crossing preserves closure semantics |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis (closures work reliably)

## Problem Statement

After M-WASM-STREAM-BRIDGE Phase 1-2, AILANG closures are correctly wrapped as callable `js.FuncOf` values and JS handlers can return ADTs. However, when a closure captures references to other module-scoped functions, those references fail at invocation time.

**Reproduction:**

```ailang
module demos/streaming/gemini_live

import std/stream (onEvent, send, close)

-- This helper is defined at module scope
let handleEvent = \conn. \event.
  send(conn, "processed: " ++ event)

-- This closure captures handleEvent
let startStream = \conn.
  onEvent(conn, \event. handleEvent(conn, event))
```

When JS invokes the `\event. handleEvent(conn, event)` closure via `ApplyClosure`:
1. The closure's body has a `core.Var` reference to `handleEvent`
2. `evalCoreVar` calls `e.env.Get("handleEvent")`
3. The environment chain doesn't contain `handleEvent` → **"undefined variable: handleEvent"**

**Current State:**
- Closures that only use their own parameters work fine
- Closures that reference module-scoped bindings fail when invoked from JS
- Same closures work when invoked purely within AILANG (module evaluator env is live)

**Impact:**
- Blocks the `onEvent(conn, handler)` pattern for real-time streaming
- Any effect handler that receives an AILANG closure callback is affected
- Browser demos can only use pure functions, not closures referencing module scope

## Root Cause Analysis

### Two Variable Resolution Paths

AILANG Core IR has two variable reference types:

1. **`core.Var`** — resolved via `e.env.Get(name)` — walks the `Environment` parent chain
2. **`core.VarGlobal`** — resolved via `e.resolver.ResolveValue(ref)` — uses `GlobalResolver` interface

Same-module references are `core.Var`. Cross-module imports are `core.VarGlobal`.

### The Environment Chain Break

When a module is loaded via `ModuleRegistry.LoadModule()`:

```
LoadModule() creates:
  evaluator := eval.NewCoreEvaluator()    ← fresh evaluator, fresh root env

For each declaration:
  evaluator.Eval(decl)                     ← evaluates in evaluator.env
  evaluator.Env().Set(name, val)           ← stores binding in root env

Closure created during eval captures:
  FunctionValue{Env: evaluator.env}        ← pointer to evaluator's env
```

The closure's `Env` points to the evaluator's environment **at the time the closure was created**. For declarations evaluated in order, the closure may capture an env that has `handleEvent` set on it (if `handleEvent` was evaluated before the closure).

**But there's a subtlety:** When the closure is later invoked via `ApplyClosure`:

```go
// ApplyClosure calls r.evaluator.CallValue(fn, arg)
// CallValue calls e.CallFunction(fn, args)
// CallFunction does:
//   newEnv := fn.Env.Clone()       ← clones the CLOSURE's env
//   e.env = newEnv                  ← swaps to closure env
//   result = e.evalCore(body)       ← evaluates body
//   e.env = oldEnv                  ← restores
```

The problem: `r.evaluator` in `ApplyClosure` is the **REPL's evaluator**, not the **module's evaluator**. The REPL evaluator has a different root environment. When `CallFunction` clones `fn.Env` and evaluates the body, any `core.VarGlobal` references would use `r.evaluator.resolver` — which may resolve correctly via `RegistryResolver`. But `core.Var` references walk `fn.Env`'s parent chain, which terminates at the **module evaluator's** root env (with builtins) — not the REPL evaluator's env.

**Key insight:** The closure's captured `fn.Env` should contain all module-scoped bindings (they were `.Set()` on the same root env). But the REPL evaluator that runs `CallFunction` might interfere with variable resolution if:
1. The module evaluator's env was garbage-collected or its state changed
2. LetRec Phase 2.5 propagation didn't run for all bindings
3. The closure was created before all module bindings were evaluated

### Related Past Bug: M-BUG-MODULE-LET-SCOPE

`design_docs/archive/v0_4_9_m-bug-module-let-scope.md` documents the same family of scoping bugs:

> Module-level let bindings weren't accessible in function bodies due to elaboration ordering — functions elaborated before lets were in scope.

That bug was in the **elaboration** phase (surface AST → Core IR). This new bug is in the **runtime** phase (closure invocation across evaluator boundaries). Same symptom (undefined variable), different layer.

## Goals

**Primary Goal:** AILANG closures invoked from JS via `ApplyClosure` can reference all bindings that were in scope when the closure was created.

**Success Metrics:**
- `onEvent(conn, \event. handleEvent(conn, event))` works in browser WASM
- Closures referencing module-scoped let bindings work
- Closures referencing imported functions work
- No regression in existing REPL or module tests
- WASM build succeeds

## Solution Design

### Overview

Ensure the REPL evaluator used by `ApplyClosure` can resolve all variables the closure's body needs, by either:

**Option A (Preferred):** Ensure closure's captured env is self-sufficient — verify that module bindings are in the env chain the closure captured, and that `CallFunction`'s env cloning preserves access.

**Option B (Fallback):** Set a `GlobalResolver` on the REPL evaluator that can resolve module-scoped bindings via the `ModuleRegistry`.

### Investigation Steps

Before implementing, we need to instrument and determine the exact failure point:

1. **Trace the env chain:** Add debug logging to `CallFunction` to print what bindings exist in `fn.Env` when invoked from `ApplyClosure`
2. **Check binding timing:** Verify whether the closure is created before or after `handleEvent` is `.Set()` on the module evaluator's env
3. **Check Core IR:** Verify whether `handleEvent` in the closure body is `core.Var` or `core.VarGlobal`
4. **Check LetRec Phase 2.5:** If the closure and `handleEvent` are in the same `LetRec`, Phase 2.5 propagation should handle it. If they're in separate `Let`/`LetRec` blocks, the ordering matters.

### Implementation Plan

**Phase 1: Reproduce and Diagnose** (~1 hour)
- [ ] Write a minimal test: load a module with closure referencing sibling binding, invoke via `ApplyClosure`
- [ ] Add debug prints to `CallFunction` showing `fn.Env` contents
- [ ] Determine if issue is env capture timing, env chain break, or resolver absence

**Phase 2: Fix** (~2 hours, depends on diagnosis)

**If env capture timing issue:**
- [ ] Ensure module evaluation order puts all bindings in env before closures that reference them
- [ ] Potentially defer closure creation until all module bindings are evaluated

**If env chain break (Clone doesn't preserve needed bindings):**
- [ ] Fix `Clone()` to deep-copy or properly chain to module root env
- [ ] Ensure `CallFunction` preserves the full parent chain

**If resolver absence (REPL evaluator lacks resolver for module vars):**
- [ ] Propagate the `GlobalResolver` from the module evaluator to the REPL evaluator
- [ ] Or: set a `RegistryResolver` on the REPL evaluator that can resolve via `ModuleRegistry`

**Phase 3: Tests and Verification** (~1 hour)
- [ ] Add test for closure-with-sibling-ref invoked via ApplyClosure
- [ ] Add test for closure-with-import-ref invoked via ApplyClosure
- [ ] Verify WASM build: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`
- [ ] Run `make test` and `make lint`

### Files to Modify/Create

**Likely modified files:**
- `internal/repl/apply_closure.go` - May need to set up resolver/env before calling (~10-30 LOC)
- `cmd/wasm/main.go` - May need to propagate resolver to REPL evaluator (~5-10 LOC)
- `internal/repl/module_registry.go` - May need to expose module env for closure resolution (~10-20 LOC)

**New files:**
- `internal/repl/apply_closure_test.go` - Tests for closure env resolution (~80-100 LOC)

**Estimated total:** ~100-160 LOC

## Examples

### Example 1: Module-scoped function reference in closure

**Current (broken):**
```ailang
module demos/test

let helper = \x. x + 1

-- This closure captures 'helper' from module scope
let makeCallback = \n. \event. helper(n)

-- When JS calls the result of makeCallback(5), helper is undefined
```

**Expected (after fix):**
```ailang
-- Same code works: helper resolves correctly when JS invokes the closure
```

### Example 2: Imported function reference in closure

```ailang
module demos/test

import std/string (toUpper)

let makeProcessor = \prefix. \msg. toUpper(prefix ++ msg)

-- When JS calls the result of makeProcessor("DEBUG: "), toUpper resolves correctly
```

## Success Criteria

- [ ] Closure referencing sibling module binding works via ApplyClosure
- [ ] Closure referencing imported function works via ApplyClosure
- [ ] Existing `go test ./internal/repl/... -run ApplyClosure` tests still pass
- [ ] WASM build succeeds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`
- [ ] `make test` passes
- [ ] `make lint` passes

## Testing Strategy

**Unit tests:**
- Load module with `let helper = ...` and `let callback = \x. helper(x)`, invoke `callback` via `ApplyClosure`
- Load module with import + closure referencing import, invoke via `ApplyClosure`
- Verify closure captured env contains expected bindings

**Integration tests:**
- WASM build succeeds (compilation test)

**Manual testing:**
- Browser: register JS stream handler, have AILANG pass closure with module-scoped refs, verify it works

## Non-Goals

**Not in this feature:**
- Changing the Core IR variable resolution model — too large, not needed
- Supporting closures that modify module state — AILANG is pure functional
- Fixing hypothetical issues with deeply nested module imports — address if reported

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Root cause is different than hypothesized | Med | Phase 1 diagnoses before implementing |
| Fix introduces env sharing that breaks isolation | Med | Test with concurrent module loads |
| Performance regression from env chain changes | Low | Env chains are typically <10 deep |

## Related Documents

**Same bug family:**
- [design_docs/archive/v0_4_9_m-bug-module-let-scope.md](../../archive/v0_4_9_m-bug-module-let-scope.md) — Elaboration-phase variant of same scoping issue

**Parent feature:**
- [design_docs/planned/v0_8_2/m-wasm-stream-bridge.md](m-wasm-stream-bridge.md) — Phase 1-2 completed, this is Phase 3

**Auto-populated by neural search:**
- [design_docs/implemented/v0_7_2/m-wasm-stdlib.md](../../implemented/v0_7_2/m-wasm-stdlib.md) (0.45) — WASM stdlib loading
- [design_docs/planned/v0_8_2/m-wasm-stream-bridge-sprint-plan.md](m-wasm-stream-bridge-sprint-plan.md) (0.46)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/eval/env.go` — Environment struct, Clone(), Get() parent chain
- `internal/eval/eval_evaluator.go:163` — CallFunction clones fn.Env
- `internal/eval/eval_expressions.go:86` — evalCoreVar uses e.env.Get()
- `internal/eval/eval_expressions.go:102` — evalCoreVarGlobal uses e.resolver
- `internal/repl/apply_closure.go` — ApplyClosure entry point
- `internal/repl/module_registry.go:534` — Fresh evaluator per module load

## Future Work

- If this pattern recurs, consider making the evaluator's environment chain explicit and inspectable (debug tooling)
- The REPL and module evaluators sharing bindings could be formalized as a "module environment protocol"

---

**Document created**: 2026-02-19
**Last updated**: 2026-02-19
