# M-ZERO-ARG-INVOCATION-SURFACES — Unify zero-arg export invocation across all surfaces

**Status**: Partially implemented (point fix landed; systemic fix planned)
**Target**: v0.22.0
**Priority**: P1 — recurring footgun, third report in 6 months
**Estimated**: ~0.5 day (point fix done; ~150 LOC + tests for unified helper)
**Dependencies**: None

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change |
| A2: Replayability | 0 | No trace impact |
| A5: Bounded Verification | **+1** | One invocation path is easier to reason about and test exhaustively than N |
| A7: Machines First | **+1** | A 0-arg export from JS/HTTP/bytecode should "just work" — surface inconsistency is exactly the kind of accidental complexity that wastes agent context |
| A11: Structured Failure | **+1** | Replaces a silent "returns the FunctionValue" failure with explicit invocation |
| A12: System Boundary | 0 | Boundary contract unchanged |

**Net Score: +3** → **Decision: Proceed**

## Problem Statement

AILANG compiles `export pure func f() -> T` to a `FunctionValue` with a single implicit unit parameter (param name `_`, type `()`). The arrow-less type signature reports arity 0, but the evaluator demands 1 argument. Every external invocation surface must paper over this mismatch independently — and they don't all do it the same way, which is how this bug keeps reappearing.

### Recurrence history

| Date | Commit | Surface | Symptom | Fix |
|------|--------|---------|---------|-----|
| 2024-09 | `2ccf3d4e` | Statement-level (M-S-CALL0) | Zero-arg call expression failed at compile time | Parser injects `_: ()` param |
| 2026-03-18 | `8cc21027` | `internal/apiserver` (HTTP serve-api) | Evaluator errored "expects 1 arguments, got 0" | Pre-injected unit arg when arity==0 |
| 2026-03-18 | `4075a402` | `internal/apiserver` (HTTP serve-api, again) | Type signature shows arity -1, not 0; previous check never fired | Retry-on-error catching `"expects 1 arguments, got 0"` |
| 2026-05-20 | *(this doc)* | `internal/repl/module_registry_prelude.go:InvokeExport` (WASM `ailangCall`) | Returned the FunctionValue itself instead of invoking — silent, not even an error | Inject unit when `len(args)==0 && len(fn.Params)==1` |

Three fixes, same root cause, three different recipes. The WASM variant was especially nasty because it failed silently — `ailangCall` returned `{success: true, result: <function>}` from JS, so callers got `"<function>"` rendered as a string instead of `"hi"`.

### Bug report

From sunholo-demos/cognitive_commons (msg_20260520_105856_e3b99ca6):

> WASM: ailangCall (InvokeExport) returns the FunctionValue itself for 0-arg functions instead of invoking them. […] Probably affects every 0-arg WASM-exported function across the demos.

Workaround the user applied: add a dummy parameter. Not acceptable as a long-term ergonomic.

## Invocation surface inventory

| Surface | File | Path |
|---------|------|------|
| Bytecode entrypoint | `internal/runtime/entrypoint.go:CallEntrypoint` | Used by `ailang run`, `serve-api`, bench harness |
| HTTP serve-api | `internal/apiserver/routes.go:callFunction` | Wraps `engine.CallPreserveFloats` |
| REPL / WASM `ailangCall` | `internal/repl/module_registry_prelude.go:InvokeExport` | Used by `cmd/wasm` exports |
| REPL string eval | `internal/repl/module_registry_prelude.go:CallExport` | Formats a call expression, evaluates as source — naturally injects `()` |
| Direct closure | `internal/repl/apply_closure.go` | Internal helper |

Each surface has its own answer to "what do I do when the caller passes 0 args?". The string-eval and apply-closure paths happen to do the right thing by going through expression evaluation. The other three needed bespoke patches.

## Point fix (landed in this commit)

`InvokeExport` now checks `len(args) == 0 && len(fn.Params) == 1` before the apply loop and invokes with a `&eval.UnitValue{}`. Regression test in `module_registry_invoke_test.go:TestInvokeExportZeroArg`.

This unblocks the cognitive_commons demo and matches the apiserver fix.

## Systemic fix (proposed)

Push the unit-injection one layer down so all surfaces inherit it.

### Option A — Inject in `eval.CoreEvaluator.CallFunction` (preferred)

When `len(args) == 0 && len(fn.Params) == 1 && fn.IsZeroArgExport()`, inject the unit value before binding. Detection of "this slot is the synthetic unit" is the only nontrivial piece. Two candidate markers:

1. Param name == `"_"` (the parser's actual convention — already used by `isUnitParam` in Go codegen).
2. Param type == `()` recorded on `FunctionValue` (requires plumbing types through, which we currently drop on the floor at eval time).

Option 1 is small and matches existing convention. Risk: a user-written `\_. ...` lambda with a single arg would also match. Mitigation: only apply the heuristic when the function came from an `export` declaration. We can record this with a single bool field `FunctionValue.IsExport` set by the elaborator.

### Option B — Single shared `InvokeExport` helper in `internal/runtime`

Move the unit-injection wrapper into `runtime.InvokeExport(inst, name, args)` and have `apiserver`, `repl` (WASM), and `cmd/ailang` (bytecode) all call it. Keeps `CallFunction` neutral; trades one boundary cost for one stable contract.

**Recommendation:** Option A. The mismatch between "arity 0 in the signature" and "arity 1 in the closure" is an internal compiler artifact that `CallFunction` is the natural place to hide. Every other surface should just work.

### Acceptance criteria (systemic fix)

1. `FunctionValue` gains `IsZeroArgExport bool`, set during elaboration when the parser injected the implicit unit param for an exported 0-arg func.
2. `CoreEvaluator.CallFunction(fn, [])` succeeds when `IsZeroArgExport` is true, returning the body result.
3. The apiserver retry-on-error in `routes.go:469` is deleted.
4. The point fix branch in `InvokeExport` is deleted (loop handles it via `CallFunction`).
5. Existing tests pass; new test in `eval/` exercises `CallFunction(fn, [])` directly with no surface wrapper.
6. `ailang test --strict` or an equivalent CI check: grep that no surface contains the string `"expects 1 arguments, got 0"` — if it reappears, someone has reintroduced a surface-local workaround.

### Out of scope

- Variadic 0-arg detection for unexported lambdas. The `IsZeroArgExport` flag is intentionally narrow — it only papers over the *parser's* implicit unit injection, not user intent.
- Changing the parser to stop injecting the unit param. That would ripple through the type system and codegen and is a much larger project (M-S-CALL0 chose injection for good reasons).

## Implementation order

1. **Done**: Point fix in `InvokeExport` + regression test.
2. Add `IsZeroArgExport` field to `FunctionValue`; set in elaborator wherever the implicit unit slot is injected.
3. Update `CallFunction` to inject unit when flag set and args empty.
4. Delete the apiserver retry and the `InvokeExport` branch.
5. Add a single test in `internal/eval/` that asserts `CallFunction(fn, [])` works for any function with `IsZeroArgExport=true`.

## Risks

- Mis-flagging: if `IsZeroArgExport` is set on a function that genuinely needs a unit arg from the caller, callers passing `[]` would get an injected unit they didn't ask for. Mitigation: only set during elaboration of `export func name() -> T` syntax, never elsewhere.
- Codegen interaction: `internal/gen/golang/codegen_type_analysis.go:isUnitParam` already special-cases this slot for Go emission. The systemic fix should reuse the same detection logic — extract a shared helper to keep them in lockstep.
