# typesIdentical Performance Bug (Large Records)

**Status**: Implemented
**Version**: v0.5.7
**Priority**: P0 - High (Blocking for stapledons_voyage project)
**Estimated**: 2 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax changes |
| Preserve Semantic Clarity | + | +1 | Fixes broken type checking |
| Increase Determinism | + | +1 | Types correctly compared without String() overhead |
| Lower Token Cost | 0 | 0 | Internal fix, no token impact |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

**Type checking hangs indefinitely when a function calls another function that returns a record with 9+ fields.**

**Current State:**
- 8 fields: Compiles in <1 second
- 9+ fields: Hangs indefinitely
- Reported by stapledons_voyage project (ArrivalState with 13 fields)
- Root cause: `typesIdentical()` uses `String()` comparison for complex types
- Called in a loop during `CoreTypeInfo.ApplySubstitution()` for every node

**Impact:**
- Users cannot use records with more than 8 fields in functions that call other functions
- Blocks development of stapledons_voyage game
- Makes AILANG impractical for real-world applications

## Root Cause Analysis

### Minimal Reproduction

```ailang
module test/minimal

type State = {
  f1: int, f2: int, f3: int, f4: int, f5: int,
  f6: int, f7: int, f8: int, f9: int  -- 9 fields
}

func makeState() -> State =
  { f1: 1, f2: 2, f3: 3, f4: 4, f5: 5, f6: 6, f7: 7, f8: 8, f9: 9 }

func test() -> State =
  makeState()  -- THIS CAUSES HANG
```

### Stack Trace

```
github.com/sunholo/ailang/internal/types.(*Row).String
github.com/sunholo/ailang/internal/types.(*TRecord).String
github.com/sunholo/ailang/internal/types.(*Row).String
github.com/sunholo/ailang/internal/types.(*TRecord).String
  ... (infinite recursion between Row.String and TRecord.String)
github.com/sunholo/ailang/internal/types.typesIdentical
github.com/sunholo/ailang/internal/types.CoreTypeInfo.ApplySubstitution
github.com/sunholo/ailang/internal/types.(*CoreTypeChecker).InferWithConstraints
```

### The Bug

In `internal/types/typeinfo.go:137`:

```go
func typesIdentical(t1, t2 Type) bool {
    // Quick check: same pointer
    if t1 == t2 {
        return true
    }

    switch v1 := t1.(type) {
    case *TVar2:
        // ... simple cases handled correctly
    case *TCon:
        // ... simple cases handled correctly
    default:
        return t1.String() == t2.String()  // BUG: O(fields) for each call!
    }
    return false
}
```

For complex types like `TRecord`, `Row`, `TFunc2`, etc., the function falls through to the default case and calls `String()` on both types. This is:

1. **O(fields)** for each String() call
2. **Called in a loop** until fixed point is reached
3. **Called for every node** in CoreTypeInfo

With 9 fields × many nodes × many iterations = exponential slowdown → infinite hang

## Goals

**Primary Goal:** Fix type checking to handle records with any number of fields efficiently.

**Success Metrics:**
- 9-field records compile in <2 seconds
- 13-field records compile in <2 seconds
- 20-field records compile in <2 seconds
- No performance regression for existing examples

## Solution Design

### Overview

Replace the `String()` comparison in `typesIdentical()` with proper structural comparison using the existing `Equals()` method.

### The Fix

**Change** `internal/types/typeinfo.go:117-140`:

```go
// typesIdentical checks if two types are identical (same structure, same names)
// Used to detect fixed points in substitution application
func typesIdentical(t1, t2 Type) bool {
    // Quick check: same pointer
    if t1 == t2 {
        return true
    }

    // Use Equals() method which handles all type cases correctly
    return t1.Equals(t2)
}
```

This is correct because:
1. `Equals()` is already implemented for all Type interfaces
2. `Equals()` uses structural comparison, not string conversion
3. `Equals()` handles cycles via pointer equality checks

### Implementation Plan

**Phase 1: Fix** (~30 minutes)
- [ ] Replace String() comparison with Equals() in typesIdentical()
- [ ] Add comment explaining why Equals() is used

**Phase 2: Test** (~1 hour)
- [ ] Create test file with 9-field record
- [ ] Create test file with 20-field record
- [ ] Verify all existing tests still pass
- [ ] Add regression test for this specific pattern

**Phase 3: Cleanup** (~30 minutes)
- [ ] Remove test files from examples/
- [ ] Update CHANGELOG.md
- [ ] Send response to stapledons_voyage

### Files to Modify

**Modified files:**
- `internal/types/typeinfo.go` - Replace typesIdentical implementation (~5 LOC change)
- `internal/types/typeinfo_test.go` - Add regression test (~30 LOC)

## Examples

### Example 1: Before Fix (Hangs)

```ailang
module game/state

type GameState = {
  player: int, enemy: int, score: int, lives: int, level: int,
  x: int, y: int, velocity: int, health: int
}

func createState() -> GameState = { ... }
func useState() -> int = (createState()).score  -- HANGS!
```

### Example 2: After Fix (Works)

Same code compiles in <1 second.

## Success Criteria

- [ ] 9-field record test compiles in <2 seconds
- [ ] 20-field record test compiles in <2 seconds
- [ ] All existing tests passing (make test)
- [ ] Regression test added
- [ ] stapledons_voyage ArrivalState module compiles

## Testing Strategy

**Unit tests:**
- Test typesIdentical with TRecord2 containing 10+ fields
- Test typesIdentical with nested records
- Test that Equals() is used correctly

**Integration tests:**
- Compile minimal reproduction case
- Compile stapledons_voyage arrival_sequence module

**Manual testing:**
- Verify stapledons_voyage project compiles

## Non-Goals

**Not in this feature:**
- Optimizing String() methods - Not needed if Equals() is used
- Adding memoization - Simple fix is sufficient
- Changing type representation - Too invasive for a bug fix

## Timeline

**Day 1** (2 hours):
- Fix implementation (30 min)
- Testing (1 hour)
- Documentation and messaging (30 min)

**Total: ~2 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Equals() has subtle bugs | High | Existing tests + new regression tests |
| Equals() is slower than String() | Low | Equals() is O(fields) vs O(fields × string_length) |
| Other code depends on String() behavior | Low | typesIdentical is internal function |

## References

- Bug reports from stapledons_voyage (message inbox)
- Stack trace showing infinite recursion between Row.String() and TRecord.String()
- Minimal reproduction: 9+ fields + function calling function

## Future Work

- Consider adding cycle detection in String() methods as defensive measure
- Profile type checking for other potential bottlenecks
- Add performance benchmarks for type checking

---

**Document created**: 2025-12-06
**Last updated**: 2025-12-06
