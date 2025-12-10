# M-FIX-FLOAT-OP: Investigation Summary

**Status**: In Progress
**Target**: v0.5.10
**Priority**: P0 - High (blocks std/math usage)
**Related**: [m-fix-float-operator-dispatch.md](m-fix-float-operator-dispatch.md)

## Current State

We've made partial progress on fixing float operator dispatch. The issue has two distinct parts:

### Part 1: Pure Function Parameters (FIXED)

**Problem**: Float parameters in pure functions were defaulting to int.

```ailang
pure func square(x: float) -> float = x * x
-- Was failing because x * x dispatched to mul_Int
```

**Root Cause**: Parameter type annotations were lost during elaboration:
- Surface AST `FuncDecl` has typed parameters
- Elaboration converts to `core.Lambda` with just parameter names
- Type checker creates fresh type variables, ignoring annotations
- Type inference defaults ambiguous Num constraints to Int

**Fix Applied**:
1. Added `paramTypeAnnots` map to Elaborator and CoreTypeChecker
2. `funcToLambda` now captures parameter types from FuncDecl
3. `inferLambda` uses annotations instead of fresh type vars

**Result**: `square(3.0) = 9.0` now works correctly!

### Part 2: Operators on Function Results (NOT YET FIXED)

**Problem**: Operators on function call results still dispatch incorrectly.

```ailang
-- In main():
sin(PI() / 4.0)  -- STILL FAILS
-- PI() returns float, 4.0 is float literal
-- But / dispatches to div_Int
```

**Debug Output**:
```
[DEBUG M-DX4] NodeID 3: type=int, head=Int  -- wrong!
```

**Root Cause**: Different from Part 1!
- `PI()` has return type annotation → type checker infers `float`
- `4.0` is a float literal → type is `float`
- BinOp `/` should unify operand types → result should be `float`
- But CoreTI records `int` for the operator node

This suggests the issue is in:
1. How BinOp types are recorded in CoreTI, OR
2. Defaulting happening at the wrong stage, OR
3. Op lowering not reading CoreTI correctly

## Investigation Findings

### Debug Output Analysis

Running with `--debug-compile` shows:
```
[DEBUG M-DX4] NodeID 5: type=float, head=Float    // operand - correct
[DEBUG M-DX4] NodeID 14: type=float, head=Float   // operand - correct
[DEBUG M-DX4] NodeID 20: type=int, head=Int       // WRONG! Division should be float
```

The operands are correctly typed as `float`, but the division operator node is `int`.

### Root Cause Analysis

The issue is in `inferBinOp` at `typechecker_operators.go:79-92`:

```go
resultType = getType(leftNode)                      // Gets type before unification solved
cls := tc.mostSpecificNumericClass(ctx, resultType) // Checks for Fractional constraint
ctx.addConstraint(ClassConstraint{
    Class: cls,  // Returns "Num" if no Fractional found
    ...
})
```

**The problem:**
1. `PI()` returns concrete `float` type directly
2. `4.0` creates type variable with `Fractional` constraint
3. Division unifies them (correct)
4. `mostSpecificNumericClass` checks for `Fractional` constraint on `float`
5. No constraint found (constraints are on type variables, not concrete types)
6. Defaults to `Num` class
7. Later, `Num` defaults to `Int`

**The fix needed:**
When one operand is concrete `float`, the operator should recognize it without needing a constraint.

### Proposed Fix: Update mostSpecificNumericClass

The `mostSpecificNumericClass` function should also check if the type is already concrete `float`:

```go
func (tc *CoreTypeChecker) mostSpecificNumericClass(ctx *InferenceContext, t Type) string {
    // M-FIX-FLOAT-OP: If type is already concrete float, use Fractional
    if t == TFloat {
        return "Fractional"
    }
    if con, ok := t.(*TCon); ok && (con.Name == "float" || con.Name == "Float") {
        return "Fractional"
    }
    // ... existing logic
}
```

## Files Modified (Part 1 Fix)

| File | Change |
|------|--------|
| `internal/elaborate/core.go` | Added `paramTypeAnnots` field |
| `internal/elaborate/file.go` | Capture annotations in `funcToLambda` |
| `internal/types/typechecker_core.go` | Added `paramTypeAnnots` + setter |
| `internal/types/typechecker_functions.go` | Use annotations in `inferLambda` |
| `internal/pipeline/pipeline_single.go` | Wire up annotations |
| `internal/pipeline/pipeline_module.go` | Wire up annotations |

## Test Commands

```bash
# Test pure function params (WORKS)
./bin/ailang run --caps IO --entry main /tmp/test_float_pure.ail

# Test function result operators (FAILS)
./bin/ailang run --caps IO --entry main examples/math_trig.ail

# With debug output
DEBUG_PARAM_ANNOTS=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail

# Debug operator lowering
./bin/ailang run --caps IO --entry main --debug-compile examples/math_trig.ail
```

## Expert Feedback Analysis

### Key Insight: Return Type Annotations Not Constraining Inference

The fundamental issue is that **return type annotations are not being unified back into the type system**.

We fixed parameter annotations (λ-arguments), but:
- Surface: `pure func PI() -> float = 3.14159`
- Inference: Body `3.14159` gets fresh type variable `α` with `Num α` constraint
- We only check "is `α` compatible with float?" at the end
- But `α` gets defaulted to `Int` BEFORE the check!
- Result: `PI()` at call site is effectively `Num α => α`, which defaults to `int`

**The fix**: Mirror what we did for parameters - unify return type annotation with inferred body type:

```go
// In inferLambda / inferFuncDecl
inferredBodyType := inferExpr(body)
if hasReturnAnnotation {
    unify(inferredBodyType, annotatedReturnType)  // <-- This is missing!
}
```

### Three Failure Modes Identified

**(A) Return type annotations not constraining inference** (MOST LIKELY)
- We capture param annotations but NOT return type annotations
- `PI()` body `3.14159` infers to `Num α => α`, not `float`
- At call site, `PI()` resolves to `int` via defaulting

**(B) BinOp type defaulted too early/incorrectly**
- Even if operands are `float`, BinOp node might be defaulted to `int`
- Defaulting may be too aggressive - treating all `Num α` vars as ambiguous
- Should only default TVars that aren't already unified with `float`

**(C) Op lowering reading wrong node/notion of "head type"**
- Mapping might fold "numeric but not explicitly float" into `int`
- Less likely given (A) is probably the root cause

### Recommended Fix Order

1. **First**: Fix return type annotation unification (symmetric to param fix)
   - Capture `returnTypeAnnots` in elaborator (like `paramTypeAnnots`)
   - In `inferLambda`, unify body type with annotated return type

2. **Second**: Verify defaulting only touches truly ambiguous numeric TVars
   - Should ignore TVars already equated with `float` via annotation

3. **Third**: Verify op lowering sees final types after defaulting

### Key Invariant

Whatever op_lowering sees must be AFTER both:
- Solving unification constraints
- Applying defaulting

Pattern to follow (Pattern A - simpler):
1. Run HM inference → get constraint graph
2. Run defaulting → mutate type graph in place
3. Populate CoreTI **after** defaulting from final graph
4. Op_lowering only ever sees defaulted types

## Next Steps

1. **Confirm PI() type at call site**: Add logging to check if `Call(PI())` has type `float` or type var
2. **Fix return type unification**: Add `returnTypeAnnots` map, unify in `inferLambda`
3. **Instrument defaulting**: Log when TVars are defaulted to Int
4. **Verify op_lowering**: Ensure it sees concrete types

## Debug Commands

```bash
# Check what type PI() call has
DEBUG_CALL_TYPES=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail

# Trace defaulting decisions
DEBUG_DEFAULTING=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail

# Trace op lowering
DEBUG_OP_LOWER=1 ./bin/ailang run --caps IO --entry main examples/math_trig.ail
```

## Files to Modify (Part 2 Fix)

| File | Change |
|------|--------|
| `internal/elaborate/core.go` | Already has `returnTypeAnnots` field |
| `internal/elaborate/file.go` | Capture return type in `funcToLambda` |
| `internal/types/typechecker_functions.go` | Unify body type with return annotation |
| `internal/types/typechecker_defaulting.go` | Ensure explicit float paths aren't overridden |

---

**Created**: 2025-12-10
**Last Updated**: 2025-12-10
