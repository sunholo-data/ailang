# M-FIX-FLOAT-OP: Float Operator Dispatch in Pure Functions

**Status**: Planned
**Target**: v0.5.10
**Priority**: P0 - High (blocks std/math usage in user code)
**Estimated**: 4-8 hours
**Dependencies**: None (bug fix)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to syntax |
| Preserve Semantic Clarity | + | +1 | Float operations execute correctly as expected |
| Increase Determinism | + | +1 | Operators dispatch deterministically based on type |
| Lower Token Cost | + | +1 | Users don't need workarounds (e.g., `pow(x,2.0)` instead of `x*x`) |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

Float arithmetic operators (`+`, `-`, `*`, `/`) in user-defined pure functions dispatch to integer implementations (`mul_Int`, `add_Int`, etc.) instead of float implementations (`mul_Float`, `add_Float`, etc.), causing runtime panics.

**Current State:**
- Float operators work correctly in `main()` and effectful functions
- Float operators work correctly when one operand is a literal (e.g., `10.0 * x`)
- Float operators FAIL in pure functions even when parameters are explicitly typed as `float`
- Runtime panic: `interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue`

**Example of failure:**
```ailang
-- This panics at runtime
pure func distance(x1: float, y1: float, x2: float, y2: float) -> float =
    sqrt((x2 - x1) * (x2 - x1) + (y2 - y1) * (y2 - y1))
```

**Impact:**
- Blocks usage of std/math trig functions in real algorithms
- Forces users to use workarounds like `pow(x, 2.0)` instead of `x * x`
- Undermines type safety guarantees of the language
- Makes pure functions unreliable for numeric computation

## Goals

**Primary Goal:** Float arithmetic operators should dispatch correctly based on operand types in all contexts, including pure functions.

**Success Metrics:**
- `distance(0.0, 0.0, 3.0, 4.0)` returns `5.0` without runtime errors
- All 19 std/math functions can be used in user-defined pure functions
- No regression in existing float/int arithmetic behavior
- `examples/math_trig.ail` runs completely without panics

## Root Cause Analysis

### Hypothesis 1: Codegen Operator Lowering

The codegen phase (`internal/gen/golang/codegen_ops.go`) may not be propagating type information correctly when lowering operators in pure function bodies.

**Evidence:**
- Git status shows recent changes to codegen files:
  - `internal/gen/golang/codegen_block.go`
  - `internal/gen/golang/codegen_expr_app.go`
  - `internal/gen/golang/codegen_expr_simple.go`
  - `internal/gen/golang/codegen_ops.go`
- The bug manifests at runtime evaluation, suggesting incorrect operator selection at compile time

### Hypothesis 2: Type Inference in Pure Function Context

The type checker may be defaulting to `Int` for binary operators when type inference occurs in pure function scope.

**Evidence:**
- Parameters are explicitly typed as `float`
- Float literals like `10.0 * cos(x)` work in main
- Suggests the issue is with variable/parameter type propagation

### Hypothesis 3: Dictionary Linking for Operators

The operator dictionary linking (`internal/link/`) may be selecting the wrong overload when operating on variables (vs. literals).

**Evidence:**
- Literals carry their type directly
- Variables require lookup from the type environment
- Pure functions may have different type context handling

## Solution Design

### Overview

Investigate and fix the operator dispatch mechanism to correctly select float variants when operand types are floats, regardless of function purity.

### Architecture

**Investigation Path:**
1. Add debug logging to trace operator lowering
2. Compare AST/Core IR for working vs. failing cases
3. Identify where type information is lost
4. Fix the root cause in codegen/link/types

**Components:**
1. **Operator Lowering** (`internal/gen/golang/codegen_ops.go`): Ensure type info available
2. **Dictionary Linking** (`internal/link/`): Verify correct overload selection
3. **Type Context** (`internal/types/`): Ensure pure function context preserves types

### Implementation Plan

**Phase 1: Investigation** (~2 hours)
- [ ] Add `DEBUG_OPERATOR_LOWERING=1` tracing to codegen
- [ ] Compare Core IR for `10.0 * cos(x)` (works) vs `x * x` in pure func (fails)
- [ ] Trace type environment in pure function elaboration
- [ ] Identify exact location where wrong operator is selected

**Phase 2: Fix** (~3 hours)
- [ ] Implement fix based on investigation findings
- [ ] Add regression test for float operators in pure functions
- [ ] Verify no regression in integer operators
- [ ] Test with complex expressions (nested operations)

**Phase 3: Validation** (~2 hours)
- [ ] Update `examples/math_trig.ail` to use `distance()` without workarounds
- [ ] Run full test suite
- [ ] Verify all std/math examples work
- [ ] Document any limitations discovered

### Files to Modify/Create

**Investigation files (likely culprits):**
- `internal/gen/golang/codegen_ops.go` - Operator lowering (~50 LOC changes)
- `internal/gen/golang/codegen_expr_app.go` - Function application codegen
- `internal/link/linker.go` - Dictionary linking for operators
- `internal/elaborate/elaborate_expr.go` - Expression elaboration

**New test files:**
- `internal/gen/golang/codegen_ops_test.go` - Add float operator tests (~100 LOC)

## Examples

### Example 1: Distance Calculation (Currently Fails)

**Before (workaround required):**
```ailang
-- Must use pow() to avoid operator dispatch bug
pure func distance(x1: float, y1: float, x2: float, y2: float) -> float =
    sqrt(pow(x2 - x1, 2.0) + pow(y2 - y1, 2.0))
```

**After (natural syntax works):**
```ailang
-- Standard arithmetic operators work correctly
pure func distance(x1: float, y1: float, x2: float, y2: float) -> float =
    let dx = x2 - x1 in
    let dy = y2 - y1 in
    sqrt(dx * dx + dy * dy)
```

### Example 2: Compound Float Expression

**Should work:**
```ailang
pure func quadratic(a: float, b: float, c: float, x: float) -> float =
    a * x * x + b * x + c
```

### Example 3: Mixed with Math Functions

**Should work:**
```ailang
pure func circleArea(r: float) -> float =
    PI() * r * r
```

## Success Criteria

- [ ] `pure func f(x: float) -> float = x * x` compiles and runs correctly
- [ ] `distance(0.0, 0.0, 3.0, 4.0)` returns `5.0`
- [ ] `examples/math_trig.ail` runs completely with distance/angle functions
- [ ] Float operators work with let-bound variables in pure functions
- [ ] No regression in integer arithmetic
- [ ] All tests passing
- [ ] Documentation updated (note any limitations)

## Testing Strategy

**Unit tests:**
- Codegen test for float binary operators
- Codegen test for mixed float expressions
- Type checker test for operator overload resolution

**Integration tests:**
- End-to-end test with `ailang run` on math_trig.ail
- REPL test for float pure functions

**Manual testing:**
- Verify debug output shows correct operator selection
- Test edge cases: nested operations, mixed int/float

## Non-Goals

**Not in this feature:**
- Adding new operators - Just fixing existing dispatch
- Performance optimization - Focus on correctness
- Changing operator semantics - Maintain current type rules

## Timeline

**Day 1** (4 hours):
- Phase 1: Investigation
- Phase 2: Implement fix

**Day 2** (4 hours):
- Phase 3: Validation
- Update examples
- Document findings

**Total: ~8 hours across 1-2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Root cause is in type inference | High | May need deeper changes; scope carefully |
| Fix breaks integer operators | High | Comprehensive regression tests |
| Fix is complex/invasive | Med | Consider targeted workaround if needed |
| Issue is in recent codegen changes | Med | May need to bisect commits |

## References

- [M-STD-MATH-TRIG](m-std-math-trig.md) - std/math implementation (exposed this bug)
- [M-DX10 Nullary Functions](../v0_4_6/m-dx10-nullary-function-calls.md) - Related type dispatch issue
- `internal/gen/golang/codegen_ops.go` - Operator codegen
- `internal/link/linker.go` - Dictionary linking

## Debug Commands

```bash
# Enable operator lowering debug output
DEBUG_OPERATOR_LOWERING=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail

# Check Core IR for a function
./bin/ailang debug core examples/math_trig.ail

# Compare working vs failing ASTs
./bin/ailang debug ast examples/math_trig.ail --show-types
```

## Future Work

- Unified operator dispatch mechanism
- Better error messages for type mismatches in operators
- Static analysis to catch dispatch issues at compile time

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10
