# M-POLY-ORD-DEFAULTING: Polymorphic `Ord`/`Eq` lambdas silently default to `Int`

**Status**: IMPLEMENTED (v0.11.4, 2026-04-14) — Phase 1 + cache fix shipped; Phase 2 (scheme instantiation leak) deferred
**Milestone target**: v0.11.4
**Priority**: High (regresses documented v0.4.0 behaviour)

## Summary

Polymorphic let-bound lambdas that use only comparison/equality operators
(`>`, `<`, `==`, …) are silently monomorphized to `Int -> Int -> Int`, making
them crash at runtime when called with `Float` or `String` arguments.

Reproducer (failing on `dev` @ ef6342b3):

```ailang
let max = \x. \y. if x > y then x else y in
max(3.14)(2.71)
-- runtime error: gt_Int: expected IntValue for arg 0, got *eval.FloatValue
```

Affected example files (both now broken despite being tagged "✅ WORKS (M-POLY-B Phase 1)"):

- [examples/runnable/polymorphic_comparison_simple.ail](../../../examples/runnable/polymorphic_comparison_simple.ail)
- [examples/runnable/polymorphic_lambdas_phase1.ail](../../../examples/runnable/polymorphic_lambdas_phase1.ail)

## Root cause

[internal/types/typechecker_defaulting.go:217-223](../../../internal/types/typechecker_defaulting.go#L217-L223) contains a branch in `pickDefault` that treats `Ord`/`Eq`/`Show` as default-to-`Int` classes:

```go
// Default to Int for Ord/Eq/Show when no numeric context
// This handles comparisons like `x > y` where x, y are already Int
if classes["Ord"] || classes["Eq"] {
    return &TCon{Name: "int"}, nil
}
```

Instrumentation at the defaulting boundary shows the following constraints reach defaulting for `let max = \x.\y. x > y in max(3.14)(2.71)`:

```
[DEBUG_DEFAULTING] Ambig: monotype=α6 ambig={α2, α5}
  c: Ord[α2]          ← scheme-bound var leaking into outer ctx
  c: Fractional[α5]   ← from 3.14
```

At the **outer** let-generalization boundary, `α2` (the scheme-quantified var from the inner `let gt`) is classified as ambiguous — meaning either (a) generalization leaves it free in the ambient constraint set, or (b) instantiation at the call site doesn't create a fresh variable. Either way, `defaultAmbiguities` then invokes `pickDefault({Ord})` on it and the Ord→Int branch forces `α2 → Int`. That substitution propagates to CoreTI *before* op-lowering, so `>` lowers to `gt_Int` inside the lambda body.

There are really **two** bugs layered here:

1. **Premature defaulting**: `pickDefault` defaults `Ord`-only constraints to `Int`. This is unsound for polymorphic functions — `Ord` should stay abstract and only get defaulted when genuinely ambiguous at the top level of a *complete program*, not at an inner let boundary.
2. **Scheme/instantiation leak**: An `α` that *should* have been quantified inside the inner `let gt`'s scheme (and replaced by a fresh β at each call site) surfaces as a free ambiguous var at the outer boundary. This suggests `generalizeWithConstraints` or scheme instantiation is not treating constraint vars correctly.

## History

- The `classes["Ord"] || classes["Eq"] → TInt` branch was introduced in **85d47647 "teschability and v0.2.0 work" (2025-10-02)**. It has been in-tree for ~6 months.
- Examples were labeled "✅ WORKS (M-POLY-B Phase 1) v0.4.0". They likely worked because M-POLY-B's **monomorphization specialized the lambda per call site**, so each copy's `α` was fresh and unified with the concrete arg type *before* defaulting ran.
- Suspect regressors (need bisection):
  - `7b1a91f1 M-PERF-DOCPARSE M1: defer CoreTI substitution` — deferring substitution may let defaulting mutate CoreTI before call-site unification lands.
  - `b29c391f M-TYPE-V2-MIGRATION: Delete legacy TFunc, unify on TFunc2`.
  - Current debug shows `0 specializations, 0 skipped` from monomorphization on the failing files — M-POLY-B's specializer no longer fires for non-module top-level `let` lambdas.

## Proposed fix (phased)

### Phase 1 — make `Ord`/`Eq` non-defaultable at generalization boundaries

Remove / gate the Ord/Eq → Int branch in [typechecker_defaulting.go:217-223](../../../internal/types/typechecker_defaulting.go#L217-L223). Only default when:

- the constraint is genuinely ambiguous **at the top of a program** (`defaultAmbiguitiesTopLevel`),
- and there is **no other plausible resolution**.

For intermediate `let` boundaries, leave `Ord α` unsolved and let generalization quantify it. This matches Haskell's `NoMonomorphismRestriction` behaviour and what the Phase 1 examples claim.

Acceptance:
- Both polymorphic example files pass `ailang run`.
- No new golden-test regressions; any eval-dependent goldens that relied on Ord→Int defaulting get updated or annotated.

### Phase 2 — fix scheme instantiation / generalization leak

Investigate why `Ord[α2]` appears in the outer constraint set rather than living inside the scheme. Likely fixes:

- Ensure `generalizeWithConstraints` removes the quantified vars' constraints from the ambient ctx.
- Ensure scheme instantiation rewrites both type *and* constraints with fresh vars.

Acceptance:
- `DEBUG_DEFAULTING=1 ailang run` on the reproducer shows `Ord[β]` (freshly instantiated) inside the call's scope, and the outer boundary sees only ground constraints (`Ord[Float]`).

### Phase 3 — regression tests

Add tests in `internal/types/defaulting_test.go` and/or `internal/pipeline/op_lowering_comparison_test.go`:

- `max` applied to `Float`, `Int`, and `String` — each must pick the right `gt_*` builtin.
- Polymorphic `eq` applied across types.
- Ensure `pickDefault` on `{Ord}` alone **returns an ambiguity error**, not `Int`.

## Gotcha surfaced during investigation: module-cache version skew

**This bit us during diagnosis** and deserves its own fix:

- `ModuleCacheKey` in [internal/pipeline/cache_key.go:22](../../../internal/pipeline/cache_key.go#L22) takes a `compilerVersion string` parameter…
- …but callers pass the format-version constant `cacheKeyVersion = "v1"` ([pipeline_module.go:219](../../../internal/pipeline/pipeline_module.go#L219)) as that argument.
- Result: **rebuilding `ailang` does not invalidate cached modules**. A bugfix to the type checker or op-lowering silently does nothing for files compiled before the rebuild, until someone manually nukes `.ailang/cache/compile/` or bumps the `"v1"` string.
- We hit this trying to instrument this bug: `[CACHE] tmp/debug_poly2: SKIP (cached 6m26s ago)` — my new debug prints never ran.

### Proposed fix

The Makefile already injects build info via ldflags ([Makefile:42](../../../Makefile#L42)):

```make
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"
```

Thread `main.Commit` (or a `main.Version + main.Commit` composite) into `ModuleCacheKey` calls instead of the literal `cacheKeyVersion` constant:

```go
// in pipeline_module.go (and other call sites)
moduleCacheKey = ModuleCacheKey(version.BuildCommit, sourceContent, depDigests)
```

Add a small `internal/pipeline/buildinfo.go` (or read from `runtime/debug.ReadBuildInfo` as a fallback for dev builds without ldflags):

```go
var BuildCommit = "unknown" // set via -ldflags from cmd/ailang main package
```

Acceptance:
- After `make quick-install`, the next run of any previously-cached module reports `CACHE MISS` (different commit → different key).
- Dev-mode builds without commit injection fall back to a stable "dev" marker, so repeated `go run` during iteration doesn't thrash the cache — but edits to source still trigger recompile via the source-hash component.
- CI uses a pinned commit, so CI cache behaviour is unchanged.

### Why this matters broadly

Any future bugfix to elaboration, type-checking, op-lowering, dictionary resolution, etc. suffers the same silent-skip if we leave this as-is. This is a foot-gun for every contributor and for CI debugging sessions. Recommend landing the cache-key fix **before or alongside** the polymorphic defaulting fix so the defaulting fix can be verified end-to-end.

## Out of scope

- Redesigning the defaulting rules more holistically (Haskell-style `default (...)` declarations). Keep the Num→Int and Fractional→Float behaviour.
- Re-enabling monomorphization for non-module top-level `let` lambdas — orthogonal to correctness.
