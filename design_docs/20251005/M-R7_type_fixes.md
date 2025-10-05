# M-R7: Type System Fixes (Integral & Float Comparison)

**Status**: 📋 Planned
**Priority**: P0 (CRITICAL - MUST SHIP)
**Estimated**: 300 LOC (200 impl + 100 tests)
**Duration**: 2 days
**Dependencies**: None
**Blocks**: Arithmetic operators, numeric algorithms

## Problem Statement

### Problem 1: Modulo Operator (`%`) Broken

**Current State**: `%` operator fails type inference with ambiguous constraints.

```ailang
-- ❌ BROKEN in v0.2.0
5 % 3  -- ERROR: Ambiguous type variable α with classes [Num, Ord]
```

**Root Cause**:
- `%` operator elaborates to both `mod` (Num) and comparison (Ord)
- Type checker cannot pick between Int/Float when constraints conflict
- No Integral type class to distinguish integer division from float division

### Problem 2: Float Comparison Uses Wrong Dictionary

**Current State**: `==` on floats calls `eq_Int` instead of `eq_Float`.

```ailang
-- ❌ BROKEN in v0.2.0
0.0 == 0.0        -- Uses eq_Int, wrong behavior
let x = 5.0 in x == 0.0  -- Dictionary resolution bug
```

**Root Cause** (discovered in adt_option benchmark):
- Dictionary elaboration doesn't respect concrete float types
- Defaulting picks Int even when operands are known to be Float
- Builtin resolution uses type-agnostic lookup

**Evidence from CHANGELOG v0.2.0**:
```
**Fixed Comparison Operators** (internal/types/typechecker_core.go, +9 LOC)
- Modified pickDefault() to default Ord, Eq, Show constraints to int
- ⚠️ But this breaks when operands are concrete floats!
```

## Goals

### Primary Goals (Must Achieve)
1. **Integral type class**: Distinguish integer ops from float ops
2. **Modulo works on Int**: `5 % 3` returns 2
3. **Modulo fails on Float**: Clear error message suggesting `div`
4. **Float comparison fixed**: `0.0 == 0.0` uses `eq_Float`
5. **All type class tests pass**: No regressions in Num, Eq, Ord

### Secondary Goals (Nice to Have)
6. Audit other operators for similar bugs (deferred to v0.3.1)
7. Integer division operator `//` (deferred to v0.4.0)

## Design

### Part 1: Integral Type Class

**Type Class Definition**:
```ailang
class Integral a {
  div: (a, a) -> a,  -- Integer division (truncates toward zero)
  mod: (a, a) -> a   -- Modulo (remainder)
}

instance Integral Int {
  div = \x, y. _int_div(x, y),
  mod = \x, y. _int_mod(x, y)
}

-- Note: No instance for Float (not Integral)
```

**Elaboration Rule**:
```
% operator → mod method call (requires Integral constraint)
```

**Implementation**:
- Add Integral to `internal/types/typeclass.go`
- Update `internal/elaborate/elaborate.go` to emit `mod` calls
- Update `pickDefault()` to default Integral to Int (only valid instance)

### Part 2: Float Comparison Fix

**Root Cause Analysis**:
1. Defaulting happens too early (before concrete types known)
2. Dictionary lookup ignores actual operand types
3. Builtin resolution doesn't check type-specific dictionaries

**Solution Options**:

**Option A: Fix defaulting (Conservative)**
- Delay defaulting until after constraint solving
- Check if operands have concrete types (Int vs Float)
- Only default ambiguous type variables, not concrete types

**Option B: Fix dictionary elaboration (Correct)**
- During elaboration, check actual operand types
- Select `eq_Int` vs `eq_Float` based on concrete type
- No defaulting needed if types are known

**Recommended: Option B** (more principled)

**Implementation**:
```go
// internal/elaborate/elaborate.go
func (e *Elaborator) elaborateBinOp(op string, left, right Expr) (Expr, error) {
    // After elaborating operands, check their types
    leftType := e.getType(left)
    rightType := e.getType(right)

    // If both are concrete floats, use float-specific dictionary
    if isConcreteFloat(leftType) && isConcreteFloat(rightType) {
        switch op {
        case "==":
            return Call(Var("eq_Float"), [left, right])
        case "!=":
            return Call(Var("neq_Float"), [left, right])
        // ... other operators
        }
    }

    // Otherwise, use type class elaboration
    return e.elaborateTypeClassOp(op, left, right)
}
```

### Error Messages

**Modulo on Float**:
```
Type Error: Modulo (%) requires Integral type
  Got: Float

  Float is not an instance of Integral.
  Hint: Use 'div' for floating-point division: x / y
```

**Float Comparison Debug** (if still broken):
```
Type Error: Comparison operator '==' failed
  Left operand:  0.0 :: Float
  Right operand: 0.0 :: Float
  Expected dictionary: eq_Float
  Got dictionary: eq_Int (BUG!)

  This is an internal error. Please report this issue.
```

## Implementation Plan

### Day 1: Integral Type Class (~150 LOC)

**Files to Modify**:
- `internal/types/typeclass.go` (~50 LOC)
- `internal/elaborate/elaborate.go` (~50 LOC)
- `stdlib/std/prelude.ail` (~20 LOC)
- `internal/types/typechecker_core.go` (~30 LOC)

**Tasks**:
1. Define Integral type class with `div`, `mod` methods
2. Add Integral[Int] instance in prelude
3. Update elaboration: `%` → `mod` method call
4. Update defaulting: Integral constraints → Int
5. Add test: `5 % 3` returns 2

**Test Cases**:
```go
// internal/types/typeclass_test.go
func TestIntegralTypeClass(t *testing.T) {
    tests := []struct {
        name string
        expr string
        want string
        err  string
    }{
        {"mod_ints", "5 % 3", "2", ""},
        {"div_ints", "5 `div` 3", "1", ""},
        {"mod_floats", "5.0 % 3.0", "", "Float is not an instance of Integral"},
    }
    // ...
}
```

### Day 2: Float Comparison Fix (~150 LOC)

**Files to Modify**:
- `internal/elaborate/elaborate.go` (~80 LOC)
- `internal/types/typechecker_core.go` (~40 LOC)
- `internal/eval/builtins.go` (~20 LOC, if needed)
- `internal/elaborate/float_test.go` (~10 LOC new file)

**Tasks**:
1. Debug dictionary elaboration for `==` operator
2. Add concrete type checking before defaulting
3. Emit `eq_Float` when operands are float
4. Test: `0.0 == 0.0`, `let x = 5.0 in x == 0.0`
5. Test: ADT with float comparison (adt_option benchmark)

**Test Cases**:
```go
// internal/elaborate/float_test.go
func TestFloatComparison(t *testing.T) {
    tests := []struct {
        name string
        expr string
        want bool
    }{
        {"literal_eq", "0.0 == 0.0", true},
        {"literal_neq", "1.0 == 0.0", false},
        {"var_eq", "let x = 5.0 in x == 0.0", false},
        {"adt_guard", "match Some(0.0) { Some(x) if x == 0.0 => true, _ => false }", true},
    }
    // ...
}
```

**Integration Test**:
```ailang
// examples/test_float_comparison.ail
module examples/test_float_comparison

import std/option (Option, Some, None)

export func test_float_eq() -> bool {
  0.0 == 0.0  -- Should use eq_Float
}

export func test_float_adt() -> int {
  match Some(0.0) {
    Some(x) if x == 0.0 => 1,  -- Guard with float comparison
    Some(x) => 2,
    None => 3
  }
}

export func main() -> bool {
  test_float_eq() && (test_float_adt() == 1)
}
```

## Acceptance Criteria

### Functional Requirements
- [ ] `5 % 3` returns 2 (Int modulo works)
- [ ] `5.0 % 3.0` errors with "Float is not Integral" message
- [ ] `0.0 == 0.0` returns true (uses `eq_Float`)
- [ ] `let x = 5.0 in x == 0.0` returns false (variable float comparison)
- [ ] ADT guard with float comparison works (adt_option benchmark)

### Code Quality
- [ ] 100% test coverage for Integral type class
- [ ] 100% test coverage for float comparison paths
- [ ] No regressions in existing type class tests
- [ ] Clear error messages for type mismatches

### Performance
- [ ] No measurable performance regression (<1%)
- [ ] Dictionary lookup overhead negligible

## Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Float comparison fix breaks other operators** | Medium | Low | Confine changes to Eq type class only; test >, <, +, -, etc. |
| **Integral breaks existing code** | Low | Low | Only affects `%` which is currently broken anyway |
| **Defaulting regression** | Medium | Low | Comprehensive type class test suite; check all operators |
| **Scope creep** (fixing other operators) | Low | Medium | **Strict scope**: Only `%` and `==`. Other operators in v0.3.1 |

## Testing Strategy

### Unit Tests (~100 LOC)
- `internal/types/typeclass_test.go`
  - Integral[Int] instance works
  - Integral[Float] rejected
  - div, mod methods type-check
  - Edge cases: mod by zero (runtime error, not type error)

- `internal/elaborate/float_test.go`
  - Float literal comparison
  - Float variable comparison
  - Float in ADT patterns
  - Float in guards

### Integration Tests
- `examples/test_integral.ail` - Integral type class demo
- `examples/test_float_comparison.ail` - Float comparison cases

### Regression Tests
- Re-run all existing type class tests
- Re-run adt_option benchmark (should pass now)

## Success Metrics

| Metric | Target |
|--------|--------|
| **Operators fixed** | 2 (% modulo, == on float) |
| **Examples fixed** | +2-3 (using % or float comparison) |
| **Test coverage** | 100% for new code paths |
| **Regressions** | 0 |

## Future Work (Deferred)

**v0.3.1 - Operator Audit**:
- Check all arithmetic operators for similar bugs
- Float division `/` vs integer division `div`
- Comparison operators on other types (String, custom Ord)

**v0.4.0 - Integer Division Operator**:
- Add `//` operator for integer division (Python-style)
- `//` → `div` method call (Integral)
- `/` remains for float division (Fractional)

**v0.4.0 - Numeric Tower**:
- Fractional type class (extends Num)
- Real type class (extends Ord + Fractional)
- Proper numeric type hierarchy

## Implementation Notes

### Integral Type Class Hierarchy

```
Num
 ├─ Int (Integral)
 └─ Float (Fractional, not Integral)

Integral requires Num (superclass)
Fractional requires Num (superclass)
```

### Dictionary Resolution Order

1. Check for explicit type annotations
2. Check for concrete types (Int, Float, String)
3. Apply constraint solving
4. Default ambiguous type variables
5. Error if still ambiguous

### Error Code Taxonomy

- `TC_INTEGRAL_001` - Type not an instance of Integral
- `TC_INTEGRAL_002` - Ambiguous Integral constraint
- `TC_EQ_001` - Comparison operator type mismatch
- `TC_EQ_002` - Eq dictionary resolution failed

## References

- **CHANGELOG**: v0.2.0 - Float comparison bug discovered
- **Benchmark**: `benchmarks/adt_option.yaml` - Float guard failure
- **Design Doc**: `design_docs/planned/v0_3_0_implementation_plan.md`
- **Type Classes**: `internal/types/typeclass.go`
- **Prior Art**: Haskell's Integral class, Python's `//` and `%`
