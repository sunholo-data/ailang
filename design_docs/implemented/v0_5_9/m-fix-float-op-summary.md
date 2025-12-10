# M-FIX-FLOAT-OP: Complete Fix Summary

**Status**: COMPLETE
**Target**: v0.5.10
**Priority**: P0 - High (blocks std/math usage)
**Related**: [m-fix-float-operator-dispatch.md](m-fix-float-operator-dispatch.md)

## Executive Summary

Float arithmetic operators were dispatching to integer implementations in pure functions. The fix required changes across 7 files, with the **root cause** being that `ApplySubstitution` wasn't following chains in the substitution map.

**Before fix:**
```
distance(0,0,3,4) = 7    -- WRONG! Should be 5.0
pow(...) + pow(...)      -- dispatched to add_Int
```

**After fix:**
```
distance(0,0,3,4) = 5.0  -- CORRECT!
All trig functions work correctly
```

## The Fix Journey

### Part 1: Parameter Annotations (Completed Earlier)

**Problem**: Float parameters in pure functions defaulted to int.

```ailang
pure func square(x: float) -> float = x * x
-- Failed: x * x dispatched to mul_Int
```

**Root Cause**: Parameter type annotations lost during elaboration.

**Fix**: Added `paramTypeAnnots` map to capture annotations and use them in inference.

**Result**: `square(3.0) = 9.0` works.

---

### Part 2: Operators on Function Results (This Session)

**Problem**: Operators on function call results still dispatched incorrectly.

```ailang
sin(PI() / 4.0)           -- FAILED: / dispatched to div_Int
pow(x2-x1, 2.0) + pow(y2-y1, 2.0)  -- FAILED: + dispatched to add_Int
```

#### Attempt 1: Check Both Operands for Fractional

**Hypothesis**: `mostSpecificNumericClass` only checks the left operand.

**Fix Applied** (`typechecker_operators.go`):
```go
leftCls := tc.mostSpecificNumericClass(ctx, getType(leftNode))
rightCls := tc.mostSpecificNumericClass(ctx, getType(rightNode))
if leftCls == "Fractional" || rightCls == "Fractional" {
    cls = "Fractional"
}
```

**Result**: Fixed `PI() / 4.0` but NOT `pow(...) + pow(...)`.

#### Attempt 2: Check Concrete Float Types

**Hypothesis**: Concrete `float` types bypass constraint checking.

**Fix Applied** (`typechecker_operators.go`):
```go
if t == TFloat {
    return "Fractional"
}
if con, ok := t.(*TCon); ok && (con.Name == "float" || con.Name == "Float") {
    return "Fractional"
}
```

**Result**: Still didn't fix `pow(...) + pow(...)`.

#### Attempt 3: Return Type Annotation Unification

**Hypothesis**: Return type annotations not constraining inference.

**Fix Applied** (`typechecker_functions.go`):
```go
if returnAnnot := tc.returnTypeAnnots[lam.ID()]; returnAnnot != nil {
    bodyType := bodyNode.GetType().(Type)
    ctx.addConstraint(TypeEq{
        Left:  bodyType,
        Right: returnAnnot,
        Path:  []string{fmt.Sprintf("return type annotation at %s", lam.Span())},
    })
}
```

**Result**: Still didn't fix the issue!

#### Attempt 4: Class Upgrade at Resolution

**Hypothesis**: Resolved type is Float but class is still Num.

**Fix Applied** (`typechecker_substitution.go`):
```go
// M-FIX-FLOAT-OP: Upgrade class based on resolved type
resolvedClass := c.Class
if normalizedType.Name == "Float" && c.Class == "Num" {
    resolvedClass = "Fractional"
}
```

**Result**: Marginal improvement but still broken.

#### The Breakthrough: Debug Output Analysis

Added extensive debug output to trace the constraint flow:

```
[DEBUG] After defaulting:
  Constraint: Num α7
  Substitution: α3 -> α7
  Substitution: α7 -> float
```

**Key observation**: The substitution had `α3 -> α7` and `α7 -> float`, but when we applied the substitution to the constraint `Num α7`, we got `Num α7` instead of `Num float`!

#### ROOT CAUSE: Substitution Not Following Chains!

**The Bug** (`unification_substitution.go`):
```go
case *TVar2:
    if newType, ok := sub[typ.Name]; ok {
        return newType  // Returns α7, not float!
    }
```

When the substitution contains `α3 -> α7` and `α7 -> float`:
- Applying to `α3` returned `α7` (one step)
- Should have returned `float` (follow the chain!)

**The Fix**:
```go
case *TVar2:
    if newType, ok := sub[typ.Name]; ok {
        // M-FIX-FLOAT-OP: Follow chains in substitution
        // If α3 -> α7 and α7 -> float, we need to return float, not α7
        result := safeSubstitute(newType, sub, visited)
        visited[t] = result
        return result
    }
```

**Result**: ALL TESTS PASS! Float operators dispatch correctly!

## Files Modified

| File | Change |
|------|--------|
| `internal/types/unification_substitution.go` | **ROOT CAUSE FIX**: Follow chains in `ApplySubstitution` |
| `internal/types/typechecker_substitution.go` | Upgrade Num→Fractional when resolved type is Float |
| `internal/types/typechecker_operators.go` | Check BOTH operands for Fractional constraint |
| `internal/types/typechecker_functions.go` | Unify body type with return annotation |
| `internal/elaborate/core.go` | Added `returnTypeAnnots` field and getter |
| `internal/elaborate/file.go` | Capture return type in `funcToLambda` |

## DX Issues & Lessons Learned

### Issue 1: No Visibility Into Substitution Chains

**Problem**: Had to add 50+ lines of debug output to understand the substitution state.

**Symptom**: Constraint showed `Num α7` but substitution had `α7 -> float`. Why wasn't it resolved?

**Root Cause Discovery**: Only found the chain issue after printing EVERY substitution entry and manually tracing the chain.

**Recommendation**: Add a `--debug-types` flag that shows:
- Current substitution map (with chain resolution)
- Active constraints before/after defaulting
- Final resolved types per node

### Issue 2: Fragmented Type Information

**Problem**: Type information spread across multiple data structures:
- `InferenceContext.substitution` - the unification state
- `CoreTypeChecker.CoreTI` - types for code generation
- `CoreTypeChecker.resolvedConstraints` - which type class to use
- `InferenceContext.constraints` - pending constraints

**Impact**: Had to trace through 4+ data structures to understand why a type was wrong.

**Recommendation**: Add a `typeReport(nodeID)` debug function that shows ALL type info for a node in one place.

### Issue 3: No Tests for Substitution Chains

**Problem**: The chain-following bug existed because there were no tests for transitive substitutions.

**Example test that should exist**:
```go
func TestSubstitutionChains(t *testing.T) {
    sub := Substitution{
        "α": &TVar2{Name: "β"},
        "β": TFloat,
    }
    result := ApplySubstitution(sub, &TVar2{Name: "α"})
    // Must be TFloat, not TVar2{β}
    assert.Equal(t, TFloat, result)
}
```

**Recommendation**: Add property-based tests for substitution that verify chain resolution.

### Issue 4: Error Messages Don't Show Type Provenance

**Problem**: When we see `type=int` in debug output, we don't know:
- Where did this type come from?
- What constraint caused it?
- When was it defaulted?

**Recommendation**: Track type provenance - each type should know its "origin story".

### Issue 5: Multi-File Changes Required

**Problem**: The fix touched 7 files across 3 packages. This indicates tight coupling.

**Files involved**:
- `internal/elaborate/` - 2 files
- `internal/types/` - 4 files
- `internal/pipeline/` - already wired (from Part 1)

**Recommendation**: Consider a unified "type annotation" system that handles param AND return types in one place.

## Test Commands

```bash
# Test the full math_trig example
./bin/ailang run --caps IO --entry main examples/math_trig.ail

# Test distance calculation specifically
./bin/ailang run --caps IO --entry main /tmp/test_distance.ail

# With debug output
DEBUG_BINOP=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail
```

## Verification

```bash
$ ./bin/ailang run --caps IO --entry main examples/math_trig.ail

=== Distance Calculation ===
distance(0,0,3,4) = 5         # CORRECT! Was 7 before fix

=== Trigonometric Functions ===
sin(PI/4) = 0.7071067811865475
cos(PI/4) = 0.7071067811865476
tan(PI/4) = 0.9999999999999999

=== Powers and Logarithms ===
pow(2,10) = 1024
sqrt(2) = 1.4142135623730951
```

## Timeline

| Time | Activity | Result |
|------|----------|--------|
| T+0 | Check both operands | Fixed `PI()/4.0`, not `pow()+pow()` |
| T+1h | Check concrete float types | No improvement |
| T+2h | Return type unification | Enabled but didn't fix |
| T+3h | Class upgrade at resolution | Marginal improvement |
| T+4h | Added extensive debug output | Found chain issue |
| T+4.5h | Fixed substitution chains | **ALL TESTS PASS** |
| T+5h | Cleanup and documentation | Complete |

## Key Takeaway

**The bug was NOT in operator dispatch logic.** It was a fundamental issue in the type inference engine: substitution chains weren't being followed. This caused type variables to remain partially resolved, leading to incorrect defaulting.

The lesson: When debugging type inference issues, **always check if substitution is idempotent** - applying it twice should give the same result as applying it once. If not, you have a chain-following bug.

---

**Created**: 2025-12-10
**Completed**: 2025-12-10
