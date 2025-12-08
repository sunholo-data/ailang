# M-DX4 Audit Findings: CoreTypeInfo Population Gaps

**Date**: 2025-10-22
**Sprint**: M-DX4 (Fix CoreTypeInfo Population Gaps)
**Phase**: 1 - Audit & Document

## Executive Summary

**Root Cause Identified**: CoreTypeInfo is populated with type variables BEFORE defaulting, but is NOT updated AFTER defaulting resolves them to concrete types (Int, Float, etc.).

**Impact**: Operator lowering sees type variables instead of concrete types, falls back to defaults (Int), causing wrong builtins to be called.

**Fix Complexity**: Medium - Need to apply substitution to CoreTI after defaulting

## Detailed Findings

### P1.1: Float Literal Inference Trace

**File**: `internal/types/typechecker_literals.go`
**Lines**: 25-34

**Current Behavior**:
```go
case core.FloatLit:
    // For float literals, create a type variable with Fractional constraint
    tv := ctx.freshType(Star)  // Creates TVar{Name: "t42"}
    ctx.addConstraint(ClassConstraint{
        Class:  "Fractional",
        Type:   tv,
        NodeID: lit.ID(),
    })
    typ = tv  // Returns type VARIABLE, not TFloat
```

**CoreTI Population** (line 433 in `typechecker_core.go`):
```go
tc.CoreTI.Set(expr.ID(), inferredType)  // Stores TVar{Name: "t42"}
```

**Defaulting** (line 195 in `typechecker_core.go`):
```go
defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguitiesTopLevel(finalType, unsolved)
// defaultingSub contains: {t42 → TFloat}
// BUT CoreTI is NOT updated with this substitution!
```

**Result**: CoreTI contains `TVar{Name: "t42"}` instead of `TFloat`

### P1.2: Var Inference for Let Bindings

**File**: `internal/types/typechecker_literals.go`
**Lines**: 59-105

**Current Behavior**:
```go
func (tc *CoreTypeChecker) inferVar(ctx *InferenceContext, v *core.Var) (*typedast.TypedVar, *TypeEnv, error) {
    typ, err := ctx.env.Lookup(v.Name)
    // ... instantiation ...
    monotype = scheme.Instantiate(ctx.freshType)  // May create NEW type variables

    return &typedast.TypedVar{
        TypedExpr: typedast.TypedExpr{
            Type: monotype,  // Could be type variable OR concrete type
        },
    }, ctx.env, nil
}
```

**CoreTI Population**: Same as literals - populated with type variable if applicable

**Issue**: If variable bound to float literal, gets type variable from scheme instantiation

### P1.3: All infer*() Methods

**Searched**: `internal/types/*.go` for all `infer*` methods

**Methods Found**:
1. ✅ `inferLit` - Populates CoreTI (line 433 caller)
2. ✅ `inferVar` - Populates CoreTI (line 433 caller)
3. ✅ `inferVarGlobal` - Populates CoreTI (line 433 caller)
4. ✅ `inferLambda` - Populates CoreTI (line 433 caller)
5. ✅ `inferLet` - Populates CoreTI (line 433 caller)
6. ✅ `inferLetRec` - Populates CoreTI (line 433 caller)
7. ✅ `inferApp` - Populates CoreTI (line 433 caller)
8. ✅ `inferIf` - Populates CoreTI (line 433 caller)
9. ✅ `inferBinOp` - Populates CoreTI (line 433 caller)
10. ✅ `inferUnOp` - Populates CoreTI (line 433 caller)
11. ✅ `inferRecord` - Populates CoreTI (line 433 caller)
12. ✅ `inferRecordAccess` - Populates CoreTI (line 433 caller)
13. ✅ `inferRecordUpdate` - Populates CoreTI (line 433 caller)
14. ✅ `inferMatch` - Populates CoreTI (line 433 caller)
15. ✅ `inferIntrinsic` - Populates CoreTI (line 433 caller)

**Conclusion**: ALL methods populate CoreTI via the common `inferCore` function (lines 428-435). The problem is NOT missing population calls.

### P1.4: Test Matrix - What Works vs What Doesn't

| Test Case | CoreTI Populated? | Type in CoreTI | Expected Type | Works? |
|-----------|-------------------|----------------|---------------|--------|
| `let x = 42 in x` | ✅ Yes | `TVar{t1}` → defaulted to `TInt` | `TInt` | ⚠️ Works after defaulting |
| `let f = 3.14 in f` | ✅ Yes | `TVar{t1}` → defaulted to `TFloat` | `TFloat` | ❌ CoreTI not updated |
| `let x = 42 in x > 0` | ✅ Yes | Intrinsic: `TBool`, Args: `TVar{t1}` | Intrinsic: `TBool`, Args: `TInt` | ⚠️ Works (M-DX3 fix uses arg type) |
| `let f = 3.14 in f > 0.0` | ✅ Yes | Intrinsic: `TBool`, Args: `TVar{t1}` | Intrinsic: `TBool`, Args: `TFloat` | ❌ CoreTI has type var for arg |
| `let b = true in b` | ✅ Yes | `TBool` (concrete) | `TBool` | ✅ Works |
| `let s = "hi" in s` | ✅ Yes | `TString` (concrete) | `TString` | ✅ Works |
| `let b = 5 > 3 in show(b)` | ✅ Yes | `TBool` (concrete) | `TBool` | ✅ Works |

**Pattern**:
- Concrete types (Bool, String, Unit) → Work ✅
- Type variables with constraints (Num, Fractional) → Stored as type vars, NOT updated after defaulting ❌

### P1.5: Design Decision - CoreTI Contract

**Contract** (to be documented in `docs/architecture/TYPE_INFERENCE.md`):

> **CoreTypeInfo Contract**
>
> After type inference completes (including constraint solving and defaulting), CoreTypeInfo MUST contain a mapping from every Core AST NodeID to its principal type.
>
> **Principal Type Definition**: The most general type that satisfies all constraints, with all ambiguous type variables defaulted to concrete types (Int, Float, etc.).
>
> **Requirements**:
> 1. Every Core expression node MUST have an entry in CoreTypeInfo
> 2. Types MUST be concrete (no unresolved type variables)
> 3. Defaulting substitutions MUST be applied to CoreTypeInfo
> 4. CoreTypeInfo MUST be populated BEFORE operator lowering
>
> **Rationale**: Operator lowering depends on CoreTypeInfo to choose correct builtins. Type variables cause fallback to defaults (Int), leading to wrong builtins and runtime panics.

## Root Cause Analysis

### The Bug

**Location**: `internal/types/typechecker_core.go` lines 188-207

**Current Code**:
```go
// Solve constraints and apply defaulting (proper way)
sub, unsolved, err := ctx.solveConstraints()

// Apply defaulting to unsolved constraints
defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguitiesTopLevel(finalType, unsolved)

// Compose substitutions if defaulting was applied
if len(defaultingSub) > 0 {
    sub = composeSubstitutions(defaultingSub, sub)
}

// Apply final substitution to type and effect row
finalType = sub.Apply(finalType).(Type)

// ❌ BUG: CoreTI is NOT updated with defaulting substitutions!
// CoreTI still has type variables (t1, t2, etc.)
// But finalType has concrete types (Int, Float, etc.)
```

**What Should Happen**:
```go
// After defaulting, apply substitution to ALL CoreTypeInfo entries
if len(defaultingSub) > 0 {
    sub = composeSubstitutions(defaultingSub, sub)

    // ✅ FIX: Update CoreTypeInfo with defaulting substitutions
    tc.CoreTI.ApplySubstitution(sub)
}
```

### Why This Causes Panics

1. Float literal `3.14` gets type variable `t1` with Fractional constraint
2. CoreTI.Set(nodeID, TVar{Name: "t1"})
3. Defaulting resolves `t1 → TFloat`
4. Final type is `TFloat`, but CoreTI still has `TVar{Name: "t1"}`
5. Operator lowering calls `CoreTI.Get(nodeID)` → gets `TVar{Name: "t1"}`
6. Type head extraction: `types.Head(TVar{...})` → returns `HeadUnknown`
7. Falls back to default: "Int"
8. Calls `gt_Int` on FloatValue → **PANIC**

## Solution Design

### Fix 1: Apply Substitution to CoreTypeInfo

**New Method** (`internal/types/typeinfo.go`):
```go
// ApplySubstitution applies a type substitution to all entries in CoreTypeInfo
func (cti *CoreTypeInfo) ApplySubstitution(sub Substitution) {
    for nodeID, typ := range cti.types {
        cti.types[nodeID] = sub.Apply(typ).(Type)
    }
}
```

**Update Caller** (`internal/types/typechecker_core.go` line 200):
```go
// Compose substitutions if defaulting was applied
if len(defaultingSub) > 0 {
    sub = composeSubstitutions(defaultingSub, sub)

    // ✅ NEW: Apply defaulting to CoreTypeInfo
    tc.CoreTI.ApplySubstitution(sub)
}
```

### Fix 2: Validation Pass (Catch Remaining Gaps)

Even with Fix 1, we should add validation to catch any remaining gaps:

**New File**: `internal/pipeline/validation.go`
```go
func ValidateCoreTypeInfo(coreAST core.CoreExpr, coreTI *types.CoreTypeInfo) error {
    var errors []error

    core.Walk(coreAST, func(node core.CoreExpr) {
        if !coreTI.Has(node.ID()) {
            errors = append(errors, fmt.Errorf(
                "Missing CoreTypeInfo for node %d at %s",
                node.ID(), node.Span()))
        }

        // Check for unresolved type variables
        if typ, ok := coreTI.Get(node.ID()); ok {
            if hasTypeVariables(typ) {
                errors = append(errors, fmt.Errorf(
                    "Unresolved type variable in CoreTypeInfo for node %d: %s",
                    node.ID(), typ))
            }
        }
    })

    if len(errors) > 0 {
        return fmt.Errorf("CoreTypeInfo validation failed:\n%v", errors)
    }
    return nil
}
```

## Next Steps

### P2.M1: Implement ApplySubstitution (Est: 2h)

1. Add `ApplySubstitution` method to CoreTypeInfo
2. Call it after defaulting in typechecker_core.go
3. Add unit tests for substitution application
4. Verify float literal test case works

### P2.M2: Test Var Nodes (Est: 1h)

After Fix 1, verify that variable references also get concrete types:
- `let x = 3.14 in x` → CoreTI has TFloat for both nodes
- `let x = 3.14 in let y = x in y` → All nodes have TFloat

### P2.M3: Validation Pass (Est: 3h)

1. Create `internal/pipeline/validation.go`
2. Implement `ValidateCoreTypeInfo`
3. Add to pipeline after type inference
4. Test with intentional gaps

### P2.M4: Debug Tooling (Est: 2h)

1. Add `--show-gaps` flag to `ailang debug types`
2. Show table of CoreTypeInfo coverage
3. Highlight type variables that should be concrete

## Estimated LOC

- Fix 1: ~20 LOC (ApplySubstitution method + caller)
- Fix 2: ~150 LOC (validation pass)
- Tests: ~300 LOC (unit + integration)
- **Total**: ~470 LOC

## Confidence Level

**High** - Root cause clearly identified, fix is straightforward (apply substitution).

The only risk is if there are other places where substitutions should be applied to CoreTI (e.g., after unification). Will discover this during testing.

---

**Audit completed**: 2025-10-22
**Time spent**: ~1.5 hours
**Ready to proceed**: Yes ✅
