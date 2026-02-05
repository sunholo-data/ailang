# M-POLY-ARITH: Fix Polymorphic Arithmetic Operators in Lambdas

**Status**: IMPLEMENTED
**Target**: v0.7.2
**Priority**: P1 (Medium-High) - Blocks common functional patterns
**Implemented**: 2026-02-05
**Dependencies**: None (continuation of archived M-POLY-B Phase 2)

## Problem Statement

Arithmetic operators (`+`, `-`, `*`, `/`, `%`) inside polymorphic lambdas panic at runtime when called with floats:

```ailang
-- This panics:
let add = \x. \y. x + y in
add(3.14)(2.71)  -- panic: FloatValue, not IntValue
```

**Expected**: `5.85`
**Actual**: `panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue`

## Root Cause

Type inference defaults arithmetic operators to `int` via Num typeclass defaulting. Monomorphization runs *after* defaulting and sees the lambda already typed as `int -> int -> int`, so it cannot correct the operator dispatch.

**Comparison operators work** because they remain polymorphic (`Ord a => a -> a -> Bool`) through inference.

## Prior Work

- **M-POLY-A (v0.4.0)**: Var→Lam resolution infrastructure - COMPLETE
- **M-POLY-B Phase 1 (v0.4.0)**: Fixed comparison operators - COMPLETE
- **M-POLY-B Phase 2**: Deferred, archived at `design_docs/archive/v0_4_1_m-poly-b-operator-relinking.md`

## Proposed Solution

Two options to explore:

### Option A: Defer Num Defaulting Until Monomorphization

Move arithmetic defaulting to happen *during* monomorphization when concrete types are known.

**Pros**: Clean solution, operators get correct types at specialization time
**Cons**: Requires type system refactoring, may affect other defaulting behavior

### Option B: Re-link Operators During Monomorphization

When specializing a lambda for concrete types, scan for arithmetic operators and re-link them to the correct type-specific builtins.

**Pros**: Localized fix, similar to what M-POLY-B Phase 1 did for comparison operators
**Cons**: More surgical, may miss edge cases

## Workarounds (Current)

```ailang
-- Option 1: Named functions with explicit types
func addFloat(x: float, y: float) -> float = x + y
addFloat(3.14, 2.71)  -- Works

-- Option 2: Direct arithmetic without lambdas
let x = 3.14 + 2.71 in x * 2.0  -- Works
```

## Success Criteria

1. `let add = \x. \y. x + y in add(3.14)(2.71)` returns `5.85`
2. All arithmetic operators (`+`, `-`, `*`, `/`, `%`) work in polymorphic lambdas
3. Existing tests pass
4. No performance regression

## References

- [Limitations doc](/docs/reference/limitations#polymorphic-arithmetic-operators-in-lambdas)
- Archived design: `design_docs/archive/v0_4_1_m-poly-b-operator-relinking.md`

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`

---

## Implementation Report (2026-02-05)

### Approach: Runtime Type Correction in Evaluator (Option C)

Neither Option A (defer defaulting) nor Option B (re-link in specializer) was used.
Instead, the fix corrects the DictRef.TypeName at evaluation time based on actual
runtime argument types.

**Root cause chain:**
1. `defaultAmbiguitiesTopLevel` defaults type variable α → Int (Num constraint)
2. Dictionary elaboration creates `DictRef(Num, "Int")` using the defaulted type
3. Evaluator's `evalDictRef` uses `DictRef.TypeName` to select Int implementation
4. Int implementation receives Float arguments → "expected int arguments" panic

**Fix location:** `internal/eval/eval_patterns.go` — `evalDictApp()`

**How it works:**
1. Evaluate arguments first (pure functional, order doesn't matter)
2. Check if first arg's runtime type matches `DictRef.TypeName`
3. If mismatch (e.g., FloatValue but DictRef says "Int"), create corrected DictRef
4. Evaluate the corrected dictionary → gets Float implementations
5. Apply method with correct type implementations

**Why this approach:**
- The specializer never updates CoreTypeInfo (it only reads, never writes)
- CoreTypeInfo has defaulted types (Int) that can't be corrected at compile time
- Runtime values are ground truth — we always know the actual type
- Works for all pipelines (file, REPL, WASM) without resolver setup

### Files Modified
| File | Change | LOC |
|------|--------|-----|
| `internal/eval/eval_patterns.go` | Fix `evalDictApp` + add `valueTypeName` helper | ~30 |
| `internal/pipeline/poly_arithmetic_test.go` | 12 integration tests | ~180 |

### Test Results
- 12/12 poly arithmetic tests pass
- All existing tests pass (0 failures)
- Lint clean
