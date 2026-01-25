# M-POLY-ADT Sprint Plan: Fix Polymorphic ADT Constructor Types

## Sprint Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-POLY-ADT |
| **Goal** | Fix polymorphic ADT constructor type inference for variants like `Err(string)` |
| **Duration** | 1 day (~4 hours) |
| **Risk Level** | Low (well-understood bug, isolated fix) |
| **Total LOC** | ~115 LOC |

## Design Doc Reference

- **Design Doc**: [m-poly-adt-option-inference.md](m-poly-adt-option-inference.md)
- **Bug Location**: `internal/pipeline/pipeline_module.go:427-440`
- **Root Cause**: Constructor type schemes assume field types map 1-to-1 with type params

## Current Status

### Verified Bug (2026-01-23)
```
Error: type unification failed at [return type annotation]:
failed to unify type argument 0: cannot unify type constructors: string vs int
```

### Workaround in Use
Monomorphic types work but are verbose:
```ailang
type IntResult = IntOk(int) | IntErr(string)  -- Works
type Result[a] = Ok(a) | Err(string)          -- Broken
```

## Milestones

### M1: Store Field Types in ConstructorInfo (~30 LOC)
**Duration**: 1 hour

**Files to modify**:
- `internal/elaborate/core.go` - Add `FieldTypes` field
- `internal/elaborate/file.go` - Capture field types during elaboration

**Tasks**:
1. Add `FieldTypes []types.Type` to `ConstructorInfo` struct
2. Create `RegisterConstructorWithFields()` method
3. Update `elaborateTypeDecl()` to capture field types from AST
4. Maintain backward compatibility with existing `RegisterConstructor()`

**Acceptance Criteria**:
- [ ] `ConstructorInfo` has `FieldTypes` field
- [ ] Field types captured for all ADT constructors
- [ ] Existing tests pass (no regression)

### M2: Use Actual Field Types in Pipeline (~40 LOC)
**Duration**: 1.5 hours

**Files to modify**:
- `internal/pipeline/pipeline_module.go` - Fix constructor scheme building
- `internal/pipeline/pipeline_single.go` - Same fix for single-file pipeline

**Tasks**:
1. Replace positional assumption with actual field type lookup
2. Handle type variables vs concrete types correctly
3. Map type variable names to ADT type params

**Acceptance Criteria**:
- [ ] `Err` gets type `∀a. string -> Result[a]` (not `∀a. a -> Result[a]`)
- [ ] `Ok` still gets type `∀a. a -> Result[a]`
- [ ] Minimal reproduction compiles successfully

### M3: Handle Imported Constructors (~25 LOC)
**Duration**: 45 minutes

**Files to modify**:
- `internal/iface/iface.go` - Add field types to `ConstructorScheme`
- `internal/pipeline/pipeline_module.go` - Read field types from imports

**Tasks**:
1. Add `FieldTypes` to `ConstructorScheme` in iface
2. Populate field types when building iface
3. Read field types when importing constructors

**Acceptance Criteria**:
- [ ] Cross-module ADT imports work correctly
- [ ] `Option[a]` from stdlib still works

### M4: Add Tests (~20 LOC)
**Duration**: 45 minutes

**Files to create/modify**:
- `internal/pipeline/pipeline_adt_test.go` - New test file

**Tasks**:
1. Add test for `Result[a] = Ok(a) | Err(string)` type schemes
2. Add integration test with `stringToInt` pattern
3. Verify no regression in existing Option tests

**Acceptance Criteria**:
- [ ] Unit test verifies constructor type schemes
- [ ] Integration test verifies full pattern matching scenario
- [ ] All existing tests pass

## Implementation Plan

### Hour 1: M1 - ConstructorInfo Enhancement

```go
// internal/elaborate/core.go
type ConstructorInfo struct {
    TypeName       string
    CtorName       string
    Arity          int
    TypeParamCount int
    IsImported     bool
    FieldTypes     []types.Type  // NEW
}

func (e *Elaborator) RegisterConstructorWithFields(
    typeName, ctorName string,
    fieldTypes []types.Type,
    typeParamCount int,
) {
    e.constructors[ctorName] = &ConstructorInfo{
        TypeName:       typeName,
        CtorName:       ctorName,
        Arity:          len(fieldTypes),
        TypeParamCount: typeParamCount,
        FieldTypes:     fieldTypes,
    }
}
```

### Hour 2: M2 - Pipeline Fix

```go
// internal/pipeline/pipeline_module.go (replace lines 427-440)
var paramTypes []types.Type
for i := 0; i < ctorInfo.Arity; i++ {
    if i < len(ctorInfo.FieldTypes) && ctorInfo.FieldTypes[i] != nil {
        fieldType := ctorInfo.FieldTypes[i]
        // Check if field type is a type variable
        if tvar, ok := fieldType.(*types.TVar2); ok {
            // Map type var name to ADT type param position
            found := false
            for j := 0; j < ctorInfo.TypeParamCount; j++ {
                // Type params are named a, b, c... or t0, t1, t2...
                if tvar.Name == fmt.Sprintf("t%d", j) ||
                   tvar.Name == string(rune('a'+j)) {
                    paramTypes = append(paramTypes, adtTypeVars[j])
                    found = true
                    break
                }
            }
            if !found {
                // Unknown type var - use as-is
                paramTypes = append(paramTypes, fieldType)
            }
        } else {
            // Concrete type (e.g., string in Err(string))
            paramTypes = append(paramTypes, fieldType)
        }
    } else {
        // Fallback: fresh type var
        varName := fmt.Sprintf("a%d", i)
        typeVars = append(typeVars, varName)
        paramTypes = append(paramTypes, &types.TVar2{Name: varName, Kind: types.Star})
    }
}
```

### Hour 3: M3 + M4 - Imports & Tests

1. Update `iface.ConstructorScheme` with `FieldTypes`
2. Write tests to verify fix

## Success Metrics

| Metric | Target |
|--------|--------|
| Minimal reproduction compiles | Yes |
| `error_handling` benchmark passes | Yes |
| All existing tests pass | Yes |
| Test coverage maintained | ≥ current |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing ADTs | Low | High | Extensive test coverage |
| Type serialization issues | Low | Medium | Use existing marshaling |
| Performance regression | Very Low | Low | Minimal data added |

## Dependencies

- None (self-contained fix)

## Verification Command

After implementation:
```bash
# Create test file
cat > /tmp/test_poly_adt.ail << 'EOF'
module test
import std/string (stringToInt, intToStr)
import std/io (println)

type Result[a] = Ok(a) | Err(string)

pure func parseIntResult(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => Ok(n),
    None => Err("Invalid integer")
  }

func main() -> () ! {IO} =
  let result = parseIntResult("42") in
  match result {
    Ok(n) => println("Parsed: " ++ intToStr(n)),
    Err(msg) => println("Error: " ++ msg)
  }
EOF

# Should succeed after fix
ailang run --caps IO --entry main /tmp/test_poly_adt.ail
```
