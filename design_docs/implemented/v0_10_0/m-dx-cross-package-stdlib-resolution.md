# M-DX-XPKG-RESOLVE: Cross-Package Stdlib Function Resolution Bug

**Status**: Implemented
**Target**: v0.9.5
**Priority**: P1 (High — breaks all package decode helpers that use std/json accessors)
**Estimated**: 2-3 days (1 day repro + investigation, 1-2 days fix)
**Dependencies**: None
**Milestone ID**: M-DX-XPKG-RESOLVE
**Created**: 2026-03-25
**Source**: DocParse agent messages `eafa7e06`, `08b28360`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes nondeterministic behavior (function works in one context, fails in another) |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Module-level function calls should behave identically regardless of call site |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Eliminates a class of silent failures that confuse AI agents |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +2 | Fundamental to composition — exported functions must work when composed cross-package |
| A11: Structured Failure | +1 | Silent empty return → correct value (or clear error) |
| A12: System Boundary | 0 | No change |

**Net Score: +6** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): This is a determinism FIX — same function should return same result regardless of call site
- [x] A10 (Composability): Cross-package function calls are the foundation of composability

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"resolver context leak" class of bugs**. The evaluator uses the caller's resolver when executing a called function's body. This means any function that:
1. Is defined as pure AILANG (not a Go builtin)
2. Is exported from module A
3. Internally references other symbols from module A
4. Is called from module B (which doesn't directly import A's internal dependencies)

...may silently fail or produce wrong results.

**Related work:**
- `m-type-cross-package-alias-unification.md` — Same class of bug but at the TYPE level (TCon vs TRecord). This is the VALUE level equivalent.
- `resolver.go` lines 96-103 — Existing fallback for transitive imports via `runtime.GetInstance()`. This fallback was added for exactly this scenario but may not cover all cases.
- `m-serve-api-transitive-imports.md` — serve-api-specific manifestation of the same root cause.

---

## Problem Statement

### The Bug

```ailang
-- In sunholo/firestore/fields (a package):
import std/json (get, asString, getOrElse)

export func asStr(fields: Json, fieldName: string) -> string {
  match get(fields, fieldName) {
    Some(field) => match get(field, "stringValue") {
      None => "",
      Some(sv) => getOrElse(asString(sv), "")  -- RETURNS EMPTY!
    },
    None => ""
  }
}
```

```ailang
-- Inline equivalent using getString works fine:
import std/json (get, getString)

let value = match get(fields, fieldName) {
  Some(field) => match getString(field, "stringValue") {
    Some(s) => s,
    None => ""
  },
  None => ""
}
-- Returns "test-user-123" correctly
```

### Key Observation

- `getString` (from `std/json`) calls `asString` internally and **works**
- `asString` called directly from a package function **fails** (returns `None`)
- Same input data, same function, different call sites -> different results

### Architecture Background

The AILANG evaluator has two types of variable references:

| Reference | Core IR | Resolution | Used For |
|-----------|---------|------------|----------|
| `core.Var` | Local | `e.env.Get(name)` — closure environment | Intra-module calls |
| `core.VarGlobal` | Cross-module | `e.resolver.ResolveValue(ref)` — module resolver | Imported function calls |

**Intra-module** (e.g., `getString` calling `asString`): The elaborator produces `core.Var{Name: "asString"}`. The function is found in the module's environment captured by the closure. This works because all module-level functions share the same child environment.

**Cross-module** (e.g., user package calling `asString`): The elaborator produces `core.VarGlobal{Ref: {Module: "std/json", Name: "asString"}}`. The resolver looks up `std/json` exports.

### Root Cause Hypotheses

**Hypothesis A: Resolver Context Leak**

When the calling package's evaluator calls `asString` (resolved via VarGlobal to a FunctionValue), it evaluates the function body using the CALLER's resolver and evaluator state. The function body contains `match j { JString(s) => Some(s), _ => None }`.

If pattern matching for `JString(s)` depends on the resolver context (e.g., to resolve the constructor identity), and the caller's resolver can't find `JString` because it's defined in `std/json`... the match always falls through to `_ => None`.

**Key code**: [eval_operations.go:118-131](internal/eval/eval_operations.go#L118-L131) — body evaluated with caller's environment/resolver.

**Hypothesis B: Constructor Module Path Mismatch**

`asString` matches against `JString(s)`. Pattern matching checks `TaggedValue.CtorName == "JString"`. But if the JSON parser creates `TaggedValue{ModulePath: "std/json", CtorName: "JString"}` and the pattern match also checks `ModulePath`... a cross-package call might have a different expected module path.

**Key code**: [eval_patterns.go](internal/eval/eval_patterns.go) — pattern matching logic for TaggedValue.

**Hypothesis C: asString Resolves to Wrong/Stale Binding**

The `GetExport("asString")` call on the `std/json` module instance might return a different value than what's in the module's closure environment. This would mean `getString`'s internal `asString` call (via Var/closure) uses the correct binding, but external calls (via VarGlobal/resolver) get a stale or different one.

**Key code**: [resolver.go:121-127](internal/runtime/resolver.go#L121-L127) — export lookup.

---

## Investigation Plan

Before implementing a fix, we need to reproduce and identify the exact root cause.

### Step 1: Minimal Reproduction

Create a test case in `internal/runtime/` that:
1. Defines module A with `asString(j)` matching on an ADT constructor
2. Defines module A's `getString(obj, key)` calling `asString` internally
3. Defines module B that imports and calls `asString` directly
4. Verifies both return the same result

```go
// internal/runtime/cross_package_stdlib_test.go
func TestCrossPackageStdlibResolution(t *testing.T) {
    // Module "mod_a" defines:
    //   type Wrapper = Wrap(string) | Empty
    //   func unwrap(w) = match w { Wrap(s) => Some(s), _ => None }
    //   func getAndUnwrap(obj, key) = match get(obj, key) { Some(w) => unwrap(w), None => None }
    //
    // Module "mod_b" imports mod_a.unwrap and calls it directly
    // Both should return Some("hello") for Wrap("hello")
}
```

### Step 2: Debug Tracing

Add `DEBUG_RESOLVER=1` tracing to `moduleGlobalResolver.ResolveValue()` to log:
- What module the resolver is bound to
- What ref is being resolved
- Whether it hits Case 1 (current), Case 2 (imported), or fallback

### Step 3: Pattern Match Tracing

Add `DEBUG_PATTERN=1` tracing to pattern matching to log:
- What value is being matched (`TaggedValue` details including `ModulePath`)
- What pattern is being checked
- Whether match succeeds or falls through

---

## Proposed Fix (Contingent on Root Cause)

### If Hypothesis A (Resolver Context Leak):

**Option A1**: Store the resolver in `FunctionValue` alongside `Env`:
```go
type FunctionValue struct {
    Params   []string
    Body     interface{}
    Env      *Environment
    Resolver GlobalResolver  // NEW: resolver from defining module
    ...
}
```

When calling a function, swap in its stored resolver:
```go
// eval_operations.go
oldResolver := e.resolver
if fn.Resolver != nil {
    e.resolver = fn.Resolver
}
// ... evaluate body ...
e.resolver = oldResolver
```

**Risk**: This changes the resolver semantics for ALL function calls. Need extensive testing.

**Option A2**: Only use function's resolver for VarGlobal lookups that fail with the caller's resolver (fallback approach). Less invasive but more complex.

### If Hypothesis B (Constructor ModulePath Mismatch):

Fix pattern matching to ignore `ModulePath` when matching constructors, or normalize module paths for stdlib types.

### If Hypothesis C (Stale Export Binding):

Ensure `GetExport` returns the same value reference as what's in the module environment.

---

## Test Plan

1. **Minimal repro test**: Cross-package ADT pattern match (described above)
2. **std/json specific test**: Import `asString` into a separate module, call with known `JString` value
3. **Regression**: All existing tests pass (`make test`)
4. **Integration**: `make verify-examples` — existing examples still work
5. **Package test**: Create a minimal test package that uses `asString` from `std/json`

---

## Key Files

| File | Purpose |
|------|---------|
| `internal/elaborate/expressions.go:38-63` | Var vs VarGlobal classification |
| `internal/runtime/resolver.go:56-129` | Module global resolver |
| `internal/runtime/runtime.go:227-383` | Module evaluation and environment setup |
| `internal/eval/eval_operations.go:118-131` | Function body evaluation (resolver context) |
| `internal/eval/eval_expressions.go:86-115` | Var and VarGlobal evaluation |
| `internal/eval/eval_patterns.go` | Pattern matching for TaggedValue |
| `std/json.ail:102-180` | asString and getString definitions |
