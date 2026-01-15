# M-GAP2: Fix Final Expression Lambda Arity Bug

## Status
- **Status:** Planned (Root Cause Identified)
- **Target:** v0.6.4
- **Priority:** P0 (Critical)
- **Estimated:** 4-8 hours
- **Dependencies:** None

## Problem Statement

**CRITICAL BUG:** Multi-param lambda type inference fails when the module's final expression is a bare identifier referencing a value computed with that lambda.

### Root Cause (Identified 2026-01-15)

**The bug is NOT path-dependent.** The actual issue:

| Final Expression | Result |
|------------------|--------|
| `sum` (bare identifier) | ❌ FAILS: "arity mismatch: 2 vs 1" |
| `(sum)` (parenthesized) | ✅ WORKS |
| `(sum, product)` (tuple) | ✅ WORKS |

### Reproduction

**Fails:**
```ailang
module test_bare
import std/list (foldl)
let numbers = [1, 2, 3]
let sum = foldl(\acc x. acc + x, 0, numbers)
sum  -- ❌ bare identifier fails
```

**Works (identical except final expression):**
```ailang
module test_parens
import std/list (foldl)
let numbers = [1, 2, 3]
let sum = foldl(\acc x. acc + x, 0, numbers)
(sum)  -- ✅ parenthesized works
```

### Why `examples/runnable/no_loops_fold.ail` Works

The original example returns a **tuple** `(sum, product, countEvens)` not a bare identifier:
```ailang
-- This works because of the tuple!
(sum, product, countEvens)
```

### Impact
- Users cannot use bare identifiers as final expressions when lambdas are involved
- Forces wrapping return values in parentheses or tuples
- Confusing error message doesn't indicate the actual problem
- Breaks AILANG's principle of least surprise

## Goals

**Primary Goal:** Bare identifier final expressions should work identically to parenthesized ones

**Success Metrics:**
- `sum` and `(sum)` produce identical type checking behavior
- All lambda patterns work regardless of final expression style
- Clear error messages if actual type errors exist

## Technical Analysis

### Hypothesis: Type Generalization Timing

The issue appears to be in how type variables are generalized when the final expression is a bare identifier vs a compound expression:

1. **Bare identifier path:** The type checker may be trying to generalize the lambda's type variables before the final usage, causing the arity to be miscounted.

2. **Parenthesized path:** The parentheses create a compound expression that delays generalization, allowing correct type inference.

### Investigation Points

1. **`internal/types/infer.go`** - Check `InferModule` handling of final expressions
2. **`internal/elaborate/elaborate.go`** - Check if bare identifiers are elaborated differently
3. **`internal/types/generalize.go`** - Check generalization timing for top-level bindings

### Likely Fix Location

```go
// internal/types/infer.go - InferModule function
// The issue is likely in how the module's return expression is typed

// Current (buggy): Bare identifier may trigger early generalization
case *core.Var:
    // Look up type without proper instantiation?

// Fixed: Ensure all expression paths have consistent typing
```

## Solution Design

### Phase 1: Diagnosis (~2 hours)

1. Add debug logging to type inference for final expressions
2. Compare type state between `sum` and `(sum)` paths
3. Identify where generalization/instantiation differs

### Phase 2: Fix (~2-4 hours)

Based on diagnosis, fix will likely be:
- Ensure bare identifiers in final position are treated identically to parenthesized ones
- May need to delay type generalization for module-level bindings
- Or ensure proper instantiation when referencing polymorphic bindings

### Files to Modify

| File | Change |
|------|--------|
| `internal/types/infer.go` | Fix final expression type inference |
| `internal/types/generalize.go` | Fix generalization timing (if needed) |

## Testing

### Minimal Test Cases

```ailang
-- test_gap2_bare.ail (should pass after fix)
module test_gap2_bare
import std/list (foldl)
let xs = [1, 2, 3]
let sum = foldl(\acc x. acc + x, 0, xs)
sum

-- test_gap2_parens.ail (already passes)
module test_gap2_parens
import std/list (foldl)
let xs = [1, 2, 3]
let sum = foldl(\acc x. acc + x, 0, xs)
(sum)
```

### Edge Cases
- [ ] Bare identifier referencing non-lambda binding (should work)
- [ ] Bare identifier referencing simple lambda (1-param)
- [ ] Bare identifier referencing multi-param lambda (2+ params)
- [ ] Multiple let bindings with bare identifier return

## Success Criteria

- [ ] `sum` works identically to `(sum)`
- [ ] All existing tests pass
- [ ] New regression tests for bare identifier final expressions
- [ ] No "arity mismatch" errors for correct code

## Timeline

**Day 1:** Diagnosis (2 hours) + Fix (2-4 hours)

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Consistent behavior regardless of syntax |
| A7: Machines First | +1 | AI-generated code works reliably |
| A8: Syntax Is Liability | +1 | Removes surprising syntax sensitivity |

**Net Score:** +3 (Accept - Critical fix)

## Related Documents

- [GAPS_DISCOVERED.md](../../../internal/dashboard_transforms/GAPS_DISCOVERED.md) - Discovery context
- GAP-3 (now obsolete) - The "path-dependent" symptom was incorrect

## Workaround

Until fixed, wrap final expressions in parentheses:
```ailang
-- Instead of:
sum

-- Use:
(sum)
```
