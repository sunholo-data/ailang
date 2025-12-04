# M-TYPE1: Array/TArray Type Unification Bug

**Status**: IMPLEMENTED
**Implemented**: 2025-12-04
**Commit**: 743f6a53
**Actual Time**: 15 minutes (the fix was simpler than expected)

**Original Estimate**: 4-6 hours
**Target**: v0.5.6
**Priority**: P0 - High (blocks game code generation use case)
**Dependencies**: None (v0.5.5 array infrastructure already in place)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables direct `#[...]` literals in ADT constructors |
| Preserve Semantic Clarity | + | +1 | Array types work consistently everywhere |
| Increase Determinism | + | +1 | Same code should always type-check the same way |
| Lower Token Cost | + | +1 | No workarounds needed (list→array conversion, etc.) |
| **Net Score** | | **+4** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

**Error Message:**
```
cannot unify type constructor Array with *types.TArray
```

**Reproduction:**
```ailang
type Direction = North | South | East | West

type AIBehavior =
  | PatternPatrol(Array[Direction])  -- ADT expects Array[Direction]
  | RandomWander

let patrol = PatternPatrol(#[North, East, South, West])  -- Array literal fails!
```

**Current State:**
- Arrays work in isolation: `A.length(#[1,2,3])` ✓, `A.getOpt(0, arr)` ✓
- Bug: Array literal `#[...]` cannot be passed to ADT constructor expecting `Array[T]`
- Internal type representations don't match during unification
- v0.5.5 introduced `TApp(Array, T)` unification with `TArray{Element: T}` but the fix is incomplete

**Impact:**
- Blocks AI game code generation (the primary demo use case)
- Users must work around with lists, losing O(1) access benefits
- Reported as `msg_20251204_120707`

## Goals

**Primary Goal:** Array literals `#[...]` should unify with `Array[T]` type parameters in all contexts.

**Success Metrics:**
- `PatternPatrol(#[North, East])` compiles and runs correctly
- All existing array tests continue to pass
- Game code generation benchmark works with arrays (not just lists)

## Solution Design

### Overview

The type unification algorithm needs to handle the bidirectional case where:
1. `TApp(TConst("Array"), T)` unifies with `TArray{Element: T}` (already implemented in v0.5.5)
2. `TArray{Element: T}` unifies with `TApp(TConst("Array"), T)` (may be missing)

The issue is likely asymmetric unification - the fix only handles one direction.

### Architecture

**Root Cause Hypothesis:**

The unifier in `internal/types/unify.go` handles:
```go
case *TApp:
    // TApp(Array, T) ~ TArray{Element: T}
```

But may not handle the reverse:
```go
case *TArray:
    // TArray{Element: T} ~ TApp(Array, T)
```

When an ADT constructor is called:
1. `PatternPatrol` expects `TApp(TConst("Array"), TConst("Direction"))`
2. Array literal `#[North, East]` infers to `TArray{Element: TConst("Direction")}`
3. Unification fails because `TArray` case doesn't handle `TApp(Array, T)` on the right

### Implementation Plan

**Phase 1: Diagnose** (~1 hour)
- [ ] Add debug logging to `unify.go` to trace the exact types being unified
- [ ] Confirm which direction of unification is failing
- [ ] Write minimal test case that reproduces the bug

**Phase 2: Fix Unification** (~2 hours)
- [ ] Add symmetric case to handle `TArray` on left, `TApp(Array, T)` on right
- [ ] Ensure element types are recursively unified
- [ ] Handle both `TConst("Array")` and `TVar` cases

**Phase 3: Test & Verify** (~2 hours)
- [ ] Add regression test for ADT constructor with array literal
- [ ] Test nested arrays: `Array[Array[int]]`
- [ ] Test polymorphic cases: `let f: Array[a] -> int = ...`
- [ ] Verify game code generation benchmark passes

### Files to Modify/Create

**Modified files:**
- `internal/types/unify.go` - Add symmetric TArray/TApp unification (~20 LOC)
- `internal/types/unify_test.go` - Add regression tests (~50 LOC)

**Test files:**
- `examples/array_adt_bug.ail` - Minimal reproduction case
- `benchmarks/game/` - Verify game generation works with arrays

## Examples

### Example 1: ADT Constructor with Array

**Before (v0.5.5 - FAILS):**
```ailang
type Direction = North | South | East | West
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander

let patrol = PatternPatrol(#[North, East, South, West])
-- Error: cannot unify type constructor Array with *types.TArray
```

**After (v0.5.6 - WORKS):**
```ailang
type Direction = North | South | East | West
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander

let patrol = PatternPatrol(#[North, East, South, West])  -- ✓ Compiles!
```

### Example 2: Function Taking Array Parameter

**Should also work:**
```ailang
let processPath: Array[Direction] -> int = \path. A.length(path)
let result = processPath(#[North, South])  -- Should work
```

## Success Criteria

- [ ] `PatternPatrol(#[North, East])` compiles without error
- [ ] `TArray{Element: T}` unifies bidirectionally with `TApp(Array, T)`
- [ ] All existing array tests pass (`go test ./internal/types/...`)
- [ ] Game code generation benchmark uses arrays successfully
- [ ] Documentation updated (CHANGELOG.md)
- [ ] Example added to `examples/`

## Testing Strategy

**Unit tests:**
- Unification: `TArray{Element: Int}` ~ `TApp(Array, Int)` both directions
- Unification: `TArray{Element: a}` ~ `TApp(Array, b)` with type variables
- Error case: `TArray{Element: Int}` ~ `TApp(List, Int)` should fail

**Integration tests:**
- ADT with array parameter: `type T = Foo(Array[int])`
- Function with array parameter: `let f: Array[a] -> int`
- Nested: `type T = Foo(Array[Array[int]])`

**Manual testing:**
- Run game code generation benchmark end-to-end
- Verify array operations in REPL with ADTs

## Non-Goals

**Not in this feature:**
- Array slicing syntax (`arr[1:3]`) - separate feature
- Mutable arrays - AILANG is pure functional
- Array comprehensions - future enhancement

## Timeline

**Day 1** (4-6 hours):
- Diagnose exact failure point (1h)
- Implement symmetric unification (2h)
- Write tests (1-2h)
- Update CHANGELOG, verify examples (1h)

**Total: ~4-6 hours, single day fix**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks existing unification | High | Comprehensive test suite, run `make test` |
| Multiple unification paths for arrays | Med | Canonicalize to single representation if needed |
| Performance regression | Low | Unification is already O(n), adding case is O(1) |

## References

- Bug report: `msg_20251204_120707` (agent inbox)
- v0.5.5 release: Fixed `TApp(Array, T)` unification with `TArray` (partial fix)
- Related: `internal/types/unify.go` - main unification logic
- Related: `internal/types/types.go` - `TArray` and `TApp` definitions

## Future Work

- Array slicing and range syntax
- Array pattern matching in `match` expressions
- Constant-time array builder syntax

---

## Implementation Notes

**Actual fix was just 8 lines in `internal/types/unification.go`:**

The TApp case already handled TList, just needed the same pattern for TArray:

```go
// Special case: TApp("Array", a) can unify with TArray{Element: a}
if t2Array, ok := t2.(*TArray); ok {
    h1, a1 := decomposeApp(t1)
    if headCon, ok := h1.(*TCon); ok && headCon.Name == "Array" && len(a1) == 1 {
        // TApp("Array", a) ~ TArray{Element: a}
        return u.Unify(a1[0], t2Array.Element, sub)
    }
}
```

The reverse direction (`TArray` on left, `TApp` on right) was already handled by the existing code at line 158-163.

## v0.5.6 Parser Fix (Additional)

**The v0.5.5 unification fix was necessary but not sufficient!**

The bug persisted because the **parser was discarding type arguments**. In `internal/parser/parser_type.go:27-50`:

```go
// OLD CODE (v0.5.5 - BUG):
_ = p.parseType() // first arg - DISCARDED!
typ = &ast.SimpleType{Name: name} // Just "Array", loses element type!
```

**Parser fix (v0.5.6):**
```go
// NEW CODE (v0.5.6 - FIX):
elemType := p.parseType() // Parse and KEEP element type

switch name {
case "Array":
    typ = &ast.ArrayType{Element: elemType, Pos: startPos}
case "List":
    typ = &ast.ListType{Element: elemType, Pos: startPos}
default:
    typ = &ast.SimpleType{Name: name, Pos: startPos} // Generic unchanged
}
```

**Files changed:**
- `internal/parser/parser_type.go` (~15 LOC)
- `internal/parser/type_test.go` (~80 LOC - regression tests)
- `examples/runnable/array_adt.ail` (integration test)
- `CHANGELOG.md` (documented fix)

**Verified working:**
```bash
ailang run --caps IO --entry main examples/runnable/array_adt.ail
# Output: Array in ADT works!
```

**Document created**: 2025-12-04
**Last updated**: 2025-12-04 (v0.5.6 parser fix added)
