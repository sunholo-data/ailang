# Sprint Plan — M-PATTERN-AND-INVOCATION-REPAIR

**Sprint ID**: `M-PATTERN-AND-INVOCATION-REPAIR`
**Target**: v0.22.0
**Created**: 2026-05-20
**Estimated duration**: 2.5 days
**Risk level**: Medium (M1 has investigation uncertainty; M2 is well-defined)

## Goal

Repair two reported regressions in v0.20.x that re-introduce previously-fixed bug classes, both bit production code at sunholo-demos/cognitive_commons (May 2026):

1. **M-MATCH-ADT-XCHECK regression** — pattern checker silently accepts foreign-ADT constructors when the scrutinee is a function call.
2. **Zero-arg invocation surfaces** — 0-arg WASM exports return the FunctionValue instead of invoking; same bug class fixed twice before in apiserver.

Point fix for #2 already landed in this branch. This sprint completes the systemic fix and tackles the #1 P0.

## Source design docs

- [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md) — P0
- [m-zero-arg-invocation-surfaces.md](m-zero-arg-invocation-surfaces.md) — P1 (point fix landed; systemic fix pending)

## Milestones

### M1 — MATCH-XCHECK regression: investigate + fix + tests (P0, ~1.5 days, ~250 LOC)

**Day 1 AM — investigation**
- Add a temporary `DEBUG_MATCH_XCHECK=1` printf in [internal/types/typechecker_patterns.go:166](../../internal/types/typechecker_patterns.go#L166) capturing `scrutType`, `extractADTName(scrutType)`, `p.Name`, `adtTypeName` for the reproducer at [/tmp/foreign_ctor_repro.ail](/tmp/foreign_ctor_repro.ail).
- Confirm hypothesis: is `scrutType` a TVar (hypothesis #1) or is `Ok` missing from `constructorTypes` (hypothesis #2)?
- Also test a let-bound-but-typed variant: `let x: Option[int] = decode(...) in match x { Ok(_) => ... }` — does the v0.18.10 check even still work for that case? Establishes whether this is a *regression* or always-was-a-hole.

**Day 1 PM — fix**
- If hypothesis #1: apply `ctx.applySubst(scrutType)` (or whatever the in-flight substitution accessor is) before extracting the ADT name. Verify the substitution is populated at pattern-check time.
- If insufficient (constraints not yet solved): implement post-solve pass — walk `TypedMatch` nodes in the elaborated tree, re-extract each scrutinee's final type, re-check each arm's constructor. Suppress generic unification errors when both sides are differently-ADT'd TApps so the post-solve pass produces the better message.
- If hypothesis #2: trace the gap in `ImportedCtorTypes` population for explicitly-imported constructors and fix in [internal/pipeline/pipeline_module_compile.go:175](../../internal/pipeline/pipeline_module_compile.go#L175).

**Day 2 AM — tests**
- Add `internal/pipeline/match_foreign_constructor_regression_test.go` covering:
  - Function-call scrutinee returning concrete ADT (the reproducer).
  - Nested match (outer Result, inner Option from `getNumber`).
  - Cross-module ADT: scrutinee is from `std/option`, foreign ctor from `std/result`.
  - **Negative test**: legitimate polymorphic scrutinee (e.g. `\opt -> match opt { Some(x) => x, None => 0 }`) must still type-check cleanly. Guards against false positives.
- Verify all existing M-MATCH-ADT-XCHECK tests in [internal/pipeline/match_foreign_constructor_test.go](../../internal/pipeline/match_foreign_constructor_test.go) still pass.

**Acceptance**
- Reproducer at [/tmp/foreign_ctor_repro.ail](/tmp/foreign_ctor_repro.ail) fails `ailang check` with `MatchForeignConstructorError` naming `Ok` and listing `Option`'s constructors.
- New regression tests pass.
- All existing tests pass.
- `go build ./internal/... ./cmd/ailang/...` clean.

### M2 — Zero-arg invocation: systemic fix (P1, ~1 day, ~150 LOC)

**Day 2 PM — flag + central injection**
- Add `IsZeroArgExport bool` field to `eval.FunctionValue` (in [internal/eval/value.go](../../internal/eval/value.go) or wherever FunctionValue lives).
- Set the flag in the elaborator wherever the parser injects the implicit unit param for `export func name() -> T`. Likely in [internal/elaborate/](../../internal/elaborate/). Reuse the existing `isUnitParam` detection from [internal/gen/golang/codegen_type_analysis.go:493](../../internal/gen/golang/codegen_type_analysis.go#L493) (extract into a shared helper).
- Update `eval.CoreEvaluator.CallFunction` at [internal/eval/eval_evaluator.go:255](../../internal/eval/eval_evaluator.go#L255): when `len(args) == 0 && len(fn.Params) == 1 && fn.IsZeroArgExport`, inject `&UnitValue{}` before binding.

**Day 3 AM — surface cleanup**
- Delete the apiserver retry-on-error in [internal/apiserver/routes.go:469-474](../../internal/apiserver/routes.go#L469-L474).
- Delete the point-fix branch in [internal/repl/module_registry_prelude.go:77-95](../../internal/repl/module_registry_prelude.go#L77-L95) (the one we just added) — `CallFunction` now handles it transparently.
- Verify `TestInvokeExportZeroArg` still passes via the new path.
- Verify apiserver 0-arg health-check tests still pass.

**Day 3 — CI guard**
- Add a `make` target or test that greps the codebase for the string `"expects 1 arguments, got 0"` outside `internal/eval/`. If it reappears in any surface package, fail — that means someone reintroduced a local workaround instead of using the central machinery.

**Acceptance**
- `&FunctionValue{IsZeroArgExport: true}` invokes correctly via `CallFunction(fn, [])`.
- All three surfaces (apiserver, WASM/REPL, bytecode entrypoint) work without surface-local injection code.
- Existing `TestInvokeExportZeroArg`, apiserver health-check tests, and bytecode bench harness all pass.
- New test in `internal/eval/` exercises `CallFunction(fn, [])` directly with no surface wrapper.
- Grep-guard for `"expects 1 arguments, got 0"` passes.

## Velocity check

Recent 7d: 114 commits, 270 files, +59334/-12393 LOC. Sprint asks for ~400 LOC + ~150 LOC test code over 2.5 days. Well within velocity.

## Risks

- **M1 hypothesis uncertainty**: if both #1 and #2 are partially true, the fix is larger. Mitigation: timebox investigation to Day 1 AM; if no clear hypothesis by lunch, switch to the post-solve pass approach which is correct regardless of root cause.
- **M2 elaborator gap**: I haven't yet located the precise elaborator site that injects the unit param. May need to search beyond elaborate/ — could be in the parser or in a sugar-rewriter pass. Mitigation: M2 sub-task #1 starts with `grep` to find the injection site before writing code.
- **CI grep guard scope**: too aggressive a grep could flag legitimate error message tests in `internal/eval/`. Mitigation: scope to surface packages only (`internal/apiserver/`, `internal/repl/`, `internal/runtime/`).

## Out of scope

- Changing parser behavior to stop emitting implicit unit params (would ripple through codegen).
- Exhaustiveness checking for missing arms.
- Refinement type matching.

## Success metrics

- 2 P0/P1 bugs closed, both with regression tests.
- Net code reduction in M2 (delete 2 surface-local workarounds, add 1 central handler + flag).
- No new failing tests; existing test count grows.
- Acks sent on msg_20260520_105856_e3b99ca6 and msg_20260520_111521_44c38751.

## Day-by-day

| Day | AM | PM |
|-----|----|----|
| 1 | M1 investigation + hypothesis confirmation | M1 fix (apply substitution OR post-solve pass) |
| 2 | M1 tests + verify existing tests | M2 flag + central injection in CallFunction |
| 3 | M2 surface cleanup + CI grep guard | Buffer / docs / final commit |
