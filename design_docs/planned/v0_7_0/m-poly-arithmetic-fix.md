# M-POLY-ARITH: Fix Polymorphic Arithmetic Operators in Lambdas

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Medium-High) - Blocks common functional patterns
**Estimated Time**: 4-8 hours
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
