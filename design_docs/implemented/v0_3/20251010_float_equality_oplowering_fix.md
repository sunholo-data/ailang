# AILANG v0.3.3: Float Equality OpLowering Fix

**Implemented**: v0.3.3 (2025-10-10)
**Commit**: 86f3c75
**Issue**: Float equality with variables incorrectly called `eq_Int` instead of `eq_Float`
**Priority**: P0 (Critical - Runtime Crash)
**Impact**: HIGH - Fixed `adt_option` benchmark, +10% absolute improvement

---

## Problem Statement

AILANG programs with float equality operations (`b == 0.0`) were **crashing at runtime** with the error:
```
Error: execution failed: builtin eq_Int expects Int arguments
```

This occurred when float variables were compared to float literals inside control flow (if/match expressions), despite the program passing type checking. The issue was **not** with literals (e.g., `0.0 == 0.0` worked fine), but with **variables** of float type.

### Example Failure

```ailang
func divide(a: float, b: float) -> Option[float] {
  if b == 0.0  // ← Runtime crash: "eq_Int expects Int arguments"
  then None
  else Some(a / b)
}
```

**Expected**: Use `eq_Float` builtin
**Actual**: Used `eq_Int` builtin → crash

---

## Root Cause

The **OpLowering pass** selected builtins using **literal inspection heuristics** instead of the type checker's **resolved constraints**.

### How OpLowering Worked (Broken)

```go
// OLD: internal/pipeline/op_lowering.go (pre-v0.3.3)
func (l *OpLowerer) lowerIntrinsic(e *core.Intrinsic) core.CoreExpr {
    // Heuristic: inspect arguments
    if isFloatLiteral(e.Args[0]) || isFloatLiteral(e.Args[1]) {
        return &core.Call{Func: "eq_Float", Args: e.Args}
    }
    return &core.Call{Func: "eq_Int", Args: e.Args}  // ← Wrong default!
}
```

**Problem**: Variables are **not** literals, so heuristic defaulted to `eq_Int` even when the type checker resolved both operands to `float`.

### Why Type Checker Didn't Catch It

The type checker **correctly** resolved:
- `b: float`
- `0.0: float`
- `==: forall a. Eq a => a -> a -> bool`
- Constraint: `Eq float` → **resolved to `eq_Float`**

But OpLowering **ignored** this information and made its own (incorrect) choice based on AST inspection.

---

## Solution: Use Type Checker's Resolved Constraints

### Changes in v0.3.3

**1. Wire resolved constraints into OpLowering**

```go
// NEW: internal/pipeline/pipeline.go
func RunPipeline(...) {
    // ...
    constraints := typeResult.ResolvedConstraints  // ← From type checker
    opLowerer.SetResolvedConstraints(constraints)  // ← Pass to OpLowering
    // ...
}
```

**2. OpLowering uses constraints for operator selection**

```go
// NEW: internal/pipeline/op_lowering.go (v0.3.3)
func (l *OpLowerer) lowerIntrinsic(e *core.Intrinsic) core.CoreExpr {
    // Look up resolved constraint for this node
    if rc, ok := l.resolvedConstraints[e.NodeID()]; ok {
        // Use the type checker's decision
        if rc.Instance == "Eq_Float" {
            return &core.Call{Func: "eq_Float", Args: e.Args}
        }
        if rc.Instance == "Eq_Int" {
            return &core.Call{Func: "eq_Int", Args: e.Args}
        }
    }
    // Fallback: old heuristic (for untyped code paths)
    return l.lowerWithHeuristic(e)
}
```

**3. Added regression tests**

```go
// NEW: internal/pipeline/op_lowering_test.go
func TestOpLowering_FloatEqualityWithVariables(t *testing.T) {
    // Test case from adt_option benchmark
    code := `
        func divide(a: float, b: float) -> bool {
            b == 0.0
        }
    `
    // Should generate eq_Float, not eq_Int
}
```

---

## Implementation Details

### Files Changed

| File | LOC | Description |
|------|-----|-------------|
| `internal/pipeline/op_lowering.go` | +50 | Use resolved constraints for builtin selection |
| `internal/pipeline/pipeline.go` | +2 | Wire constraints from type checker |
| `internal/pipeline/op_lowering_test.go` | +160 | Comprehensive regression tests |
| `internal/eval/value.go` | +12 | Fixed float display (`5.0` not `5`) |
| `internal/eval/eval_simple.go` | +9 | Float formatting in `show()` |
| `internal/types/typechecker_core.go` | -50 | Cleanup unused code |

**Total**: ~180 LOC net (implementation + tests)

### Float Builtin Implementation

The `eq_Float` builtin was **already implemented** in `internal/eval/builtins.go` with proper NaN handling:

```go
Builtins["eq_Float"] = &BuiltinFunc{
    Name:    "eq_Float",
    NumArgs: 2,
    IsPure:  true,
    Impl: func(a, b *FloatValue) (*BoolValue, error) {
        // NaN is not equal to anything, including itself
        if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
            return &BoolValue{Value: false}, nil
        }
        return &BoolValue{Value: a.Value == b.Value}, nil
    },
}
```

**The bug was NOT in the builtin**, but in **selecting the wrong builtin**.

---

## Testing & Validation

### M-EVAL Benchmark Results

**Comparison**: v0.3.2 → v0.3.3

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| **adt_option status** | runtime_error | **PASSING** ✅ | **FIXED** |
| **Overall success rate** | 40% (4/10) | 50% (5/10) | **+10% absolute** |
| **Relative improvement** | - | - | **+25%** |

### Specific Test Cases

```ailang
// ✅ Works: Literal comparisons (always worked)
0.0 == 0.0  // → true

// ✅ Works: Variable comparisons in modules (NOW FIXED)
func divide(a: float, b: float) -> Option[float] {
  if b == 0.0  // ← No longer crashes!
  then None
  else Some(a / b)
}

// ⚠️ Known limitation: REPL type annotation persistence
let b: float = 0.0  // Type annotation lost during elaboration
b == 0.0            // Still fails in REPL (type becomes α)
                     // Fix planned: M-REPL1 (v0.3.5)
```

### Regression Tests

Added comprehensive tests in `internal/pipeline/op_lowering_test.go`:
- Float equality with variables
- Float equality with literals
- Mixed float/int operations (should fail at type check)
- Float comparisons in control flow (if/match)
- Cross-module float equality

All tests pass ✅

---

## Known Limitations

### 1. REPL Type Annotation Persistence (M-REPL1)

**Issue**: User type annotations are lost during elaboration in REPL.

```ailang
λ> let b: float = 0.0  // ← Type annotation given
λ> :type b
b :: α                  // ← Type lost! Becomes type variable
λ> b == 0.0
Error: eq_Int expects Int arguments  // ← Wrong builtin selected
```

**Workaround**: Use direct literals in REPL:
```ailang
λ> 0.0 == 0.0  // ← Works
true :: Bool
```

**Fix**: Planned in M-REPL1 (v0.3.5 or v0.4.0) - Make elaboration preserve type schemes.

### 2. Float Display Formatting

**Secondary fix**: Float `show()` formatting was inconsistent.

```ailang
// Before v0.3.3
show(5.0)  // → "5"   (missing .0)

// After v0.3.3
show(5.0)  // → "5.0" (correct)
```

This caused benchmark output mismatches and has been fixed.

---

## Impact Assessment

### Benchmarks Fixed
- ✅ `adt_option` - runtime_error → **PASSING**

### Benchmarks Still Failing (unrelated issues)
- ⚠️ `recursion_factorial` - logic_error (AI variance, not language bug)
- ❌ `pipeline`, `numeric_modulo`, `json_parse`, `float_eq`, `cli_args` - compile errors (different issues)

### Success Rate Improvement
- **Absolute**: +10% (4/10 → 5/10)
- **Relative**: +25% improvement
- **Critical bugs fixed**: 1 (runtime crash)

---

## Lessons Learned

### What Worked Well
1. **M-EVAL detected the bug** - Benchmark suite caught the issue before user reports
2. **Type checker was correct** - Problem was in compilation pipeline, not inference
3. **Clear error messages** - Users could diagnose "eq_Int expects Int" easily
4. **Comprehensive tests** - Regression tests ensure fix stays fixed

### What Could Be Improved
1. **Earlier detection** - OpLowering should have asserted consistency with type checker
2. **Better separation of concerns** - OpLowering shouldn't have had its own type inference
3. **Documentation** - Known limitations (REPL) should be more prominently documented

### Architecture Insights
- **Trust the type checker** - Don't re-infer types in later passes
- **Wire constraints through pipeline** - Each pass should use type checker's decisions
- **Test at boundaries** - Test that compilation passes agree with each other

---

## References

### Related Docs
- [M-EVAL Case Study](../../docs/guides/evaluation/case-study-oplowering-fix.md) - Detailed investigation
- [CHANGELOG v0.3.3](../../CHANGELOG.md#v033---2025-10-10) - Release notes
- [REPL Stabilization](./M-REPL0_basic_stabilization.md) - Related REPL fixes

### Related Issues
- **Float Equality Investigation**: [design_docs/planned/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md](../planned/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md)
- **M-REPL1**: [design_docs/planned/M-REPL1_persistent_bindings.md](../planned/M-REPL1_persistent_bindings.md)

### Commits
- **86f3c75**: Fix float equality bug and improve eval harness (Oct 10, 2025)
- **77f81ce**: Fix tests/binops_float.ail: remove invalid module syntax (Oct 10, 2025)

---

## Conclusion

The float equality bug was a **critical runtime crash** caused by OpLowering ignoring the type checker's resolved constraints. The fix was **surgical** (~180 LOC) and **effective** (+10% benchmark success rate).

**Key takeaway**: Compilation passes should use the type checker as the single source of truth for type information, not re-infer types independently.

The remaining REPL limitation is tracked separately in M-REPL1 and requires elaboration to preserve type schemes across inputs.

---

*Implemented: v0.3.3 (2025-10-10)*
*Documented: 2025-10-13*
*Author: Mark + Claude*
