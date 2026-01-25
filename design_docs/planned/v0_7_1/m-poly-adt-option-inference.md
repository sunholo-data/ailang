# M-POLY-ADT: Polymorphic ADT Type Inference with Option

## Status
- **Priority**: High (affects error handling patterns)
- **Discovered**: 2026-01-06 via eval gap analysis
- **Verified**: 2026-01-23 (bug still present in v0.7.0)
- **Affects**: Result types, Either types, any polymorphic ADT where constructor fields don't match type parameters

## Problem Statement

Polymorphic ADT types like `Result[a]` fail to unify correctly when a constructor's field type doesn't match the type parameter position.

### Minimal Reproduction

```ailang
module test
import std/string (stringToInt)

type Result[a] = Ok(a) | Err(string)

-- This FAILS with: "cannot unify type constructors: string vs int"
pure func parseIntResult(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => Ok(n),
    None => Err("Invalid integer")
  }
```

### Error Message

```
Error: type error in test (decl 0): type unification failed at
  [return type annotation at test.ail:6:6]:
  failed to unify type argument 0: cannot unify type constructors: string vs int
```

### Working Workaround

Use monomorphic types instead:

```ailang
-- This WORKS
type IntResult = IntOk(int) | IntErr(string)

pure func parseIntResult(s: string) -> IntResult =
  match stringToInt(s) {
    Some(n) => IntOk(n),
    None => IntErr("Invalid integer")
  }
```

## Impact

- **Error handling patterns**: Can't use standard `Result[a]` pattern
- **API consistency**: Forces verbose type definitions per concrete type
- **Agent benchmarks**: `error_handling` benchmark fails due to this issue

## Root Cause Analysis (CONFIRMED)

### Location of Bug

**File:** `internal/pipeline/pipeline_module.go`, lines 427-440

### The Bug

When building constructor type schemes, the code **incorrectly assumes** that constructor field types correspond 1-to-1 with ADT type parameters:

```go
// BUGGY CODE at pipeline_module.go:427-440
// For constructor fields, use the same type vars for the first TypeParamCount fields
// (This assumes common case where field types are the ADT's type params)
for i := 0; i < ctorInfo.Arity; i++ {
    if i < ctorInfo.TypeParamCount {
        // Use the ADT type var
        paramTypes = append(paramTypes, adtTypeVars[i])  // ❌ WRONG!
    } else {
        // Create additional type var for extra fields
        varName := fmt.Sprintf("a%d", i)
        typeVars = append(typeVars, varName)
        paramTypes = append(paramTypes, &types.TVar2{Name: varName, Kind: types.Star})
    }
}
```

### Why It's Wrong

For `type Result[a] = Ok(a) | Err(string)`:

**What the code generates:**
- `Ok` → `∀t0. t0 -> Result[t0]` ✅ CORRECT (field type IS the type param)
- `Err` → `∀t0. t0 -> Result[t0]` ❌ WRONG (field type is `string`, not `t0`)

**What it should generate:**
- `Ok` → `∀t0. t0 -> Result[t0]` ✅
- `Err` → `∀t0. string -> Result[t0]` ✅ (field type is concrete `string`)

### The Failure Chain

When we call `Err("Invalid integer")`:
1. Instantiate scheme with fresh var `t1` → `t1 -> Result[t1]` (WRONG!)
2. Unify argument `"Invalid integer" : string` with param `t1`
3. Solver binds `t1 = string`
4. Result type becomes `Result[string]`
5. Later, unify `Result[string]` with return annotation `Result[int]`
6. **FAIL**: `string vs int` on type argument

### Root Problem

`ConstructorInfo` struct only stores:
```go
type ConstructorInfo struct {
    TypeName       string // e.g., "Result"
    CtorName       string // e.g., "Err"
    Arity          int    // e.g., 1
    TypeParamCount int    // e.g., 1
    // ❌ MISSING: actual field types!
}
```

**It does NOT store the actual field types** from the AST. The code guesses that fields match type parameters positionally, which is wrong for `Err(string)`.

## Proposed Fix

### Phase 1: Store Field Types in ConstructorInfo

**File:** `internal/elaborate/core.go`

```go
type ConstructorInfo struct {
    TypeName       string
    CtorName       string
    Arity          int
    TypeParamCount int
    IsImported     bool
    FieldTypes     []types.Type  // NEW: actual field types from AST
}
```

**File:** `internal/elaborate/file.go`, in `elaborateTypeDecl`:

```go
// Process each constructor in the ADT
for _, ctor := range def.Constructors {
    // Convert AST field types to internal types
    fieldTypes := make([]types.Type, len(ctor.Fields))
    for i, field := range ctor.Fields {
        fieldTypes[i] = e.astTypeToInternalType(field.Type)
    }

    // Register constructor with actual field types
    e.RegisterConstructorWithFields(typeName, ctor.Name, fieldTypes, typeParamCount)
}
```

### Phase 2: Use Actual Field Types in Pipeline

**File:** `internal/pipeline/pipeline_module.go`, around line 427:

```go
// FIXED: Use actual field types instead of assuming positional match
var paramTypes []types.Type
for i := 0; i < ctorInfo.Arity; i++ {
    if i < len(ctorInfo.FieldTypes) {
        fieldType := ctorInfo.FieldTypes[i]
        // Check if field type uses a type parameter
        if tvar, ok := fieldType.(*types.TVar2); ok {
            // Find which ADT type param this corresponds to
            for j, paramName := range adtTypeParamNames {
                if tvar.Name == paramName {
                    paramTypes = append(paramTypes, adtTypeVars[j])
                    break
                }
            }
        } else {
            // Concrete type (like `string` in Err(string))
            paramTypes = append(paramTypes, fieldType)
        }
    }
}
```

### Phase 3: Handle Imported Constructors

**File:** `internal/iface/iface.go`

Add field types to `ConstructorScheme`:

```go
type ConstructorScheme struct {
    TypeName   string
    CtorName   string
    FieldTypes []types.Type  // Actual field types
    ResultType types.Type
    Arity      int
}
```

Update serialization/deserialization to include field types.

## Test Cases

### Unit Test (Phase 1)

```go
// In internal/pipeline/pipeline_adt_test.go
func TestPolymorphicADTConstructorTypes(t *testing.T) {
    src := `
    module test
    type Result[a] = Ok(a) | Err(string)
    `
    // Verify:
    // - Ok has type ∀a. a -> Result[a]
    // - Err has type ∀a. string -> Result[a]
}
```

### Integration Test (Phase 2)

```go
func TestPolymorphicADTWithOption(t *testing.T) {
    src := `
    module test
    import std/string (stringToInt)
    type Result[a] = Ok(a) | Err(string)
    pure func parse(s: string) -> Result[int] =
      match stringToInt(s) {
        Some(n) => Ok(n),
        None => Err("bad")
      }
    `
    // Should type check successfully
}
```

### Benchmark Test

```bash
ailang eval-suite --benchmarks error_handling
# Should pass after fix
```

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/elaborate/core.go` | Add `FieldTypes` to `ConstructorInfo` | +5 |
| `internal/elaborate/file.go` | Capture field types during elaboration | +15 |
| `internal/pipeline/pipeline_module.go` | Use actual field types | +20 |
| `internal/pipeline/pipeline_single.go` | Same fix for single-file pipeline | +20 |
| `internal/iface/iface.go` | Add field types to `ConstructorScheme` | +5 |
| `internal/pipeline/pipeline_adt_test.go` | New test file | +50 |

**Estimated total:** ~115 LOC

## Success Criteria

1. ✅ Minimal reproduction compiles without error
2. ✅ `error_handling` benchmark passes with standard `Result[a]` pattern
3. ✅ Existing polymorphic ADT tests continue to pass
4. ✅ `Option[a]` from stdlib still works correctly
5. ✅ Cross-module ADT imports work (field types serialized in iface)

## Risks and Mitigations

### Risk: Breaking existing ADT patterns
**Mitigation:** Extensive test coverage before/after. The current behavior is clearly buggy, so "breaking" it means fixing it.

### Risk: Field type serialization complexity
**Mitigation:** Field types are already serializable as `types.Type`. Use existing JSON marshaling.

### Risk: Performance regression from storing more data
**Mitigation:** Minimal impact - only adds a slice of types per constructor, not per call site.

## References

- Bug location: `internal/pipeline/pipeline_module.go:427-440`
- Prompt workaround: `prompts/v0.6.5.md` line 498 (Note about monomorphic types)
- Benchmark: `benchmarks/error_handling.yml`
- Eval results: `eval_results/v0.6.5-g3-haiku/` (error_handling failures)
