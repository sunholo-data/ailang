# M-BUG: List Length Returns Sum Instead of Count

**Status**: ✅ Resolved (by M-LETREC-SCOPING)
**Target**: v0.6.1
**Priority**: P1 (Medium)
**Estimated**: 2-4 hours
**Actual**: 0 hours (already fixed)
**Dependencies**: None
**Resolution Commit**: `752cf226` (Dec 18, 2025)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Fixes incorrect semantics |
| Increase Determinism | + | +1 | Functions return expected results |
| Lower Token Cost | 0 | 0 | No change to token usage |
| **Net Score** | | **+2** | **Decision: Move forward (bug fix)** |

## Problem Statement

When using recursive ADT pattern matching, the `length` function returns incorrect values. Testing shows it returns 15 instead of 5 for a 5-element list.

**Current State:**
```ailang
type IntList = INil | ICons(int, IntList)

letrec length = \list.
    match list {
        INil => 0,
        ICons(_, tail) => 1 + length(tail)
    }
in
let numbers = ICons(1, ICons(2, ICons(3, ICons(4, ICons(5, INil))))) in
length(numbers)
-- Expected: 5
-- Actual: 15 (same as sum!)
```

**Impact:**
- Recursive functions may be executing incorrectly
- Pattern matching with wildcards (`_`) may not be working as expected
- This affects any user-defined recursive ADT operations

## Goals

**Primary Goal:** Fix recursive pattern matching so length returns 5, not 15.

**Success Metrics:**
- `length(numbers)` returns 5 for a 5-element list
- `sum(numbers)` still returns 15
- All existing tests pass
- New regression test added

## Investigation Plan

### Hypothesis 1: Wildcard Pattern Not Matching

The `_` in `ICons(_, tail)` may not be correctly ignoring the first element.

**Test:**
```ailang
letrec length = \list.
    match list {
        INil => 0,
        ICons(x, tail) => 1 + length(tail)  -- Try explicit binding
    }
```

### Hypothesis 2: Multiple `letrec` Interference

Having two `letrec` blocks might be causing variable shadowing or incorrect closures.

**Test:**
- Try with single letrec
- Check if `length` is somehow using `sum`'s body

### Hypothesis 3: Core AST Elaboration Bug

The elaboration from Surface AST to Core AST might be incorrectly handling:
- Wildcard patterns in constructor positions
- letrec scoping
- Closure capture

## Files to Investigate

- `internal/eval/eval.go` - Evaluator logic for pattern matching
- `internal/elaborate/elaborate.go` - Surface to Core transformation
- `internal/core/core.go` - Core AST representation
- `internal/dtree/dtree.go` - Decision tree compilation for patterns

## Reproduction Steps

```bash
# Create test file
cat > /tmp/test_length.ail << 'EOF'
type IntList = INil | ICons(int, IntList)

letrec length = \list.
    match list {
        INil => 0,
        ICons(_, tail) => 1 + length(tail)
    }
in
let numbers = ICons(1, ICons(2, ICons(3, ICons(4, ICons(5, INil))))) in
length(numbers)
EOF

# Run and observe
./bin/ailang run /tmp/test_length.ail
# Expected: 5
# Actual: 15
```

## Success Criteria

- [x] `length` function returns correct count (5 for 5-element list)
- [x] `sum` function still works correctly (15 for [1,2,3,4,5])
- [x] Wildcard patterns work correctly in all positions
- [x] Multiple letrec blocks work independently
- [x] All existing tests pass
- [x] Regression test added (`internal/pipeline/specialize_integration_test.go`)

## Testing Strategy

**Unit tests:**
- Test wildcard pattern matching in isolation
- Test multiple letrec scoping

**Integration tests:**
- Test list_sum.ail example returns (15, 5)
- Test similar patterns in other ADTs

**Manual testing:**
- REPL evaluation of length function
- Step-through debugging of evaluator

## References

- `examples/runnable/list_sum.ail` - Reproduction case
- `internal/eval/` - Evaluator implementation
- `internal/elaborate/` - AST elaboration

---

## Resolution

**Root Cause:** Monomorphization cache key collision for lambdas.

The monomorphization system used a generic `"(lambda)"` key for ALL anonymous lambdas with the same type signature. When two lambdas had identical types (like `sum` and `length`, both `IntList -> int`), they would share the same cache key, causing the second lambda to incorrectly receive the first lambda's specialized body.

**The Fix (commit `752cf226`):**
```go
// Before (broken):
DefSym: "(lambda)"  // Same key for all lambdas with same type!

// After (fixed):
DefSym: fmt.Sprintf("(lambda@%d)", lambda.ID())  // Unique per lambda
```

**Verification:**
```bash
$ ./bin/ailang run examples/runnable/list_sum.ail
(15, 5)  # ✅ CORRECT: sum=15, length=5
```

**Related:**
- Design doc: `design_docs/implemented/v0_6_1/m-letrec-scoping-regression.md`
- Tests: `internal/pipeline/specialize_integration_test.go`

---

**Document created**: 2025-12-17
**Last updated**: 2025-12-18
