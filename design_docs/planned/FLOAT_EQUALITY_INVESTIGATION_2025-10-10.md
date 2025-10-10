# Float Equality Bug - Deep Investigation Report

**Date**: 2025-10-10
**Status**: ❌ Bug Confirmed - Root Cause Identified - Implementation Pending
**Priority**: P1 (High)
**Affected Benchmarks**: adt_option, float_eq (22.6% of all failures)

## Executive Summary

After deep investigation, we've determined that the Float equality bug (`b == 0.0` calling `eq_Int` instead of `eq_Float`) is **NOT** a type inference or defaulting issue. The type system is working correctly.

**The real problem**: AILANG's operator architecture has two conflicting code paths (dictionary-based vs shim-based), and `Intrinsic` nodes aren't properly integrated with the dictionary-passing system.

## What We Verified Works Correctly ✅

1. **Eq[Float] instance exists** and is properly registered ([internal/types/instances.go:178-186](../internal/types/instances.go))
2. **eq_Float builtin exists** and is implemented ([internal/eval/builtins.go:219](../internal/eval/builtins.go))
3. **Float literal typing is correct** - `0.0` creates Fractional constraint ([internal/types/typechecker_core.go:483-492](../internal/types/typechecker_core.go))
4. **Defaulting logic is correct** - `pickDefault` chooses Float for Fractional+Eq ([internal/types/typechecker_core.go:2208-2267](../internal/types/typechecker_core.go))
5. **Type annotations work** - `let b: float` correctly types b as Float

## The Actual Problem ❌

### Architectural Conflict

AILANG has **two operator implementations** that conflict:

```
Path 1 (Old - Dictionary-based):
  BinOp nodes → Type class constraints → DictApp → Correct dictionary lookup → eq_Float ✅

Path 2 (New - Shim-based):
  Intrinsic nodes → Experimental shim → Direct pattern matching → ??? → eq_Int ❌
```

### What Happens

```
Source:     b == 0.0  (where b: float)
    ↓
Elaboration: Creates core.Intrinsic{Op: OpEq, Args: [var_b, lit_0.0]}
    ↓
Type Check: Adds Eq constraint, correctly resolves to Float type
    ↓
Constraint Resolution: Creates ResolvedConstraint{Class: "Eq", Type: Float}
    ❌ BUT: Intrinsic node's ID doesn't match constraint's NodeID!
    ↓
FillOperatorMethods: Tries to find constraint for Intrinsic node ID
    ❌ NOT FOUND (NodeID mismatch)
    ↓
Runtime: Falls back to... something that calls eq_Int
    ❌ ERROR: builtin eq_Int expects Int arguments, but received float
```

### Root Cause: Missing OpLowering Pass

Comments throughout the codebase mention an "OpLowering pass" that should convert `Intrinsic` nodes to `DictApp` nodes:

- [internal/elaborate/elaborate.go:1429](../internal/elaborate/elaborate.go): "Intrinsic nodes pass through - they'll be handled by OpLowering pass"
- [internal/eval/eval_core.go:787](../internal/eval/eval_core.go): "This should typically be handled by OpLowering pass"

**This pass was never implemented.**

## Test Results

### Simple Case: WORKS ✅
```ailang
0.0 == 0.0  // Returns: true
```
**Why**: Both literals, shim handles it directly

### Variable Case: FAILS ❌
```ailang
let b: float = 0.0;
b == 0.0  // Error: eq_Int expects Int, got float
```
**Why**: Variable introduces indirection, constraint resolution fails

### Parameter Case: FAILS ❌
```ailang
func test(b: float) -> bool {
  b == 0.0  // Error: eq_Int expects Int, got float
}
```
**Why**: Same issue - constraint/Intrinsic NodeID mismatch

## Changes Made During Investigation

### 1. Fixed FillOperatorMethods for Intrinsic Nodes
**File**: `internal/types/typechecker_core.go:2656-2676`
**What**: Added case to handle Intrinsic nodes, set method name
**Impact**: Necessary but not sufficient - no constraint found to fill

### 2. Added intrinsicOpToString Helper
**File**: `internal/types/typechecker_core.go:2723-2761`
**What**: Maps IntrinsicOp enum to operator string ("==", "+", etc.)
**Impact**: Support function for Intrinsic handling

### 3. Cleaned Up Broken Defaulting
**File**: `internal/types/typechecker_core.go:222-248`
**What**: Removed duplicate defaulting that always chose Int for Eq/Ord
**Impact**: Code cleanup, didn't fix the issue

### 4. Fixed InferWithConstraints Pipeline
**File**: `internal/types/typechecker_core.go:174-215`
**What**: Made it use proper defaulting (defaultAmbiguitiesTopLevel)
**Impact**: Improves module type checking, but doesn't fix Intrinsic issue

### 5. Improved Error Messages
**File**: `internal/eval/builtins.go:760-780`
**What**: Added type mismatch error with hints for Float equality
**Impact**: Better diagnostics, helps users understand the bug

## Recommended Solution

### Approach A: Implement OpLowering Pass (RECOMMENDED)

**Complexity**: ~400 LOC, 2-3 days
**Quality**: HIGH - Proper architectural fix

**Steps**:
1. Create `internal/oplowering/oplowering.go`
2. Walk Core AST after type checking
3. For each Intrinsic node with resolved constraint:
   - Look up constraint type (Float, Int, etc.)
   - Create DictApp with correct dictionary reference
   - Replace Intrinsic with DictApp
4. Integrate into pipeline between type checking and evaluation

**Example Transformation**:
```go
// BEFORE
core.Intrinsic{
  Op: OpEq,
  Args: [var_b, lit_0.0]
}

// AFTER
core.DictApp{
  Dict: DictRef{Class: "Eq", Type: "Float"},
  Method: "eq",
  Args: [var_b, lit_0.0]
}
```

### Approach B: Fix Experimental Shim (WORKAROUND)

**Complexity**: ~100 LOC, 0.5 days
**Quality**: MEDIUM - Band-aid fix

**Steps**:
1. Add tracing to find where eq_Int is called
2. Ensure Intrinsic nodes always use shim path
3. Verify shim correctly handles Float/Float comparison

**Issue**: The shim already has Float handling code that SHOULD work:
```go
if lFloat, lOk := left.(*FloatValue); lOk {
    if rFloat, rOk := right.(*FloatValue); rOk {
        case "==": return &BoolValue{Value: lFloat.Value == rFloat.Value}, nil
    }
}
```

Something is bypassing this code path.

## Impact

**Current State** (v0.3.0-40):
- Success rate: 40% (4/10 benchmarks)
- adt_option: ❌ Failing
- float_eq: ❌ Failing

**After Fix**:
- Success rate: 50-60% (+2 benchmarks)
- Eliminates 22.6% of runtime failures
- 5-10% reduction in token usage (fewer retries)

## Next Steps

1. **Decide on approach**: OpLowering (proper) vs Shim fix (quick)
2. **Implement chosen approach**
3. **Test with benchmarks**: `ailang eval-validate adt_option float_eq`
4. **Run full test suite**: `make test`
5. **Update CHANGELOG.md**
6. **Move this doc to** `design_docs/implemented/v0_3/`

## Files to Review

- `internal/types/typechecker_core.go` - Constraint resolution
- `internal/elaborate/elaborate.go` - Intrinsic node creation
- `internal/eval/eval_core.go` - Runtime evaluation paths
- `internal/core/core.go` - Core AST definitions

## Related Issues

- **Capability auto-grant**: Separate issue, easier fix
- **Compile errors in other benchmarks**: Different root causes (prompt quality)

---

**Investigation completed**: 2025-10-10
**Time spent**: ~3 hours
**Conclusion**: Complex architectural issue requiring either OpLowering implementation or shim debugging
