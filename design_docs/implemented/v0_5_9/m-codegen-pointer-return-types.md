# M-CODEGEN-POINTER-RETURN-TYPES: Fix Type Assertion Failures for Record Returns

**Status**: Implemented
**Target**: v0.5.9
**Priority**: P0 - Critical (Blocking runtime integration)
**Estimated**: 1 hour
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix - no syntax change |
| Preserve Semantic Clarity | + | +1 | Generated code now correctly reflects AILANG semantics |
| Increase Determinism | + | +1 | Consistent pointer types for all record returns |
| Lower Token Cost | 0 | 0 | No impact on token cost |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

**Bug Report Source**: stapledons_voyage project (4 messages, Dec 9, 2025)

### Problem 1: Function Return Types (FIXED)

When compiling AILANG code with exported functions that return record types, the generated Go code produces runtime panics:

```
panic: interface conversion: interface {} is *sim_gen.World, not sim_gen.World
```

**Root Cause:**
The `ailangTypeToGo` function in `cmd/ailang/compile.go` registered function return types as value types (e.g., `World`), but `generateTypedRecord` in the codegen always generates pointer literals (`&World{...}`).

### Problem 2: Struct Field Types (FIXED)

After fixing Problem 1, a new compile error emerged:

```
cannot use NewArrivalPhasePhaseBlackHole() (value of type *ArrivalPhase) as ArrivalPhase value in struct literal
```

**Root Cause:**
The `mapASTType` function in `internal/gen/golang/adt.go` generated struct fields as VALUE types (e.g., `Phase ArrivalPhase`), but ADT constructors return POINTERS (e.g., `*ArrivalPhase`).

## Goals

**Primary Goal:** Fix type consistency so all user-defined types are pointers throughout codegen.

**Success Metrics:**
- [x] Generated typed wrappers use pointer types (`*World`)
- [x] Generated struct fields use pointer types (`*ArrivalPhase`)
- [x] Type assertions match actual runtime values
- [x] No compile errors for ADT fields in records
- [x] All tests pass

## Solution Design

### Overview

Make ALL user-defined types (records, ADTs) use pointer types consistently:
1. ✅ Function signatures (ailangTypeToGo) - DONE
2. ✅ Struct field types (mapASTType) - DONE

### Architecture

The AILANG codegen generates user-defined types in multiple places:

| Location | Function | Before Fix | After Fix |
|----------|----------|------------|-----------|
| Function signatures | `ailangTypeToGo` | `World` | `*World` ✅ |
| Struct fields | `mapASTType` | `ArrivalPhase` | `*ArrivalPhase` ✅ |
| List elements | `mapASTType` | `[]*DrawCmd` | `[]*DrawCmd` ✅ |
| ADT constructors | Various | `*ArrivalPhase` | `*ArrivalPhase` ✅ |
| Record literals | `generateTypedRecord` | `&World{...}` | `&World{...}` ✅ |

**Key Insight:** ADT constructors return pointers. All references to user-defined types must use pointers to match.

### Implementation Plan

**Phase 1: Fix Function Signatures** (~5 minutes) ✅ COMPLETE
- [x] Change `ailangTypeToGo` default case to return `*capitalize(typ.Name)`
- [x] Update comment to explain pointer semantics

**Phase 2: Fix Struct Field Types** (~30 minutes) ✅ COMPLETE
- [x] Modify `mapASTType` SimpleType case to add `*` for user-defined types
- [x] Update ListType/ArrayType cases to handle already-pointer elements
- [x] Track base types (without `*`) in adtSliceTypes for converter generation
- [x] Update tests to expect pointer types

**Phase 3: Fix Type Assertions for interface{} Values** (~10 minutes) ✅ COMPLETE
- [x] Add type assertions for user-defined pointer types in `generateTypedRecord`
- [x] Use `exprProducesInterface()` to determine when assertion is needed
- [x] Skip type assertions for ADT constructors (they return typed pointers, not interface{})

**Phase 4: Fix Double-Pointer in Slice Type Assertions** (~5 minutes) ✅ COMPLETE
- [x] Fix ListType case in `ailangTypeToGo` to check for existing `*` prefix
- [x] Fix ArrayType case similarly
- [x] Prevent `[]**TypeName` when element type is already `*TypeName`

### Files Modified

**Phase 1:**
- `cmd/ailang/compile.go` - Line 649-653, ~4 LOC changed

**Phase 2:**
- `internal/gen/golang/adt.go` - `mapASTType` function, ~20 LOC changed
- `internal/gen/golang/adt_test.go` - Updated test expectations, ~6 LOC changed

**Phase 3:**
- `internal/gen/golang/codegen_ops.go` - `generateTypedRecord` else branch, ~5 LOC changed

**Phase 4:**
- `cmd/ailang/compile.go` - ListType and ArrayType cases in `ailangTypeToGo`, ~8 LOC changed

**Phase 2 Code Changes:**

1. **SimpleType case** - Add pointer for user-defined types:
```go
case *ast.SimpleType:
    goType := g.mapNamedType(typ.Name)
    // M-CODEGEN-POINTER-RETURN-TYPES: User-defined types need pointers
    if isUserDefinedType(goType) {
        return "*" + goType
    }
    return goType
```

2. **ListType/ArrayType cases** - Handle already-pointer elements:
```go
if isUserDefinedType(elemType) {
    baseType := strings.TrimPrefix(elemType, "*")
    g.adtSliceTypes[baseType] = true
    if strings.HasPrefix(elemType, "*") {
        return fmt.Sprintf("[]%s", elemType)  // Already a pointer
    }
    return fmt.Sprintf("[]*%s", elemType)
}
```

## Examples

### Example 1: Exported Function Returning Record

**Generated Go (After Fix):**
```go
func InitWorld(seed int64) *World {
    return initWorld_impl(seed).(*World)  // Works!
}
```

### Example 2: Record with ADT Field

**Generated Go (After Fix):**
```go
type ArrivalState struct {
    Phase *ArrivalPhase  // POINTER type - correct!
}

return &ArrivalState{
    Phase: NewArrivalPhasePhaseBlackHole(),  // Works: *ArrivalPhase == *ArrivalPhase
}
```

### Example 3: Record with ADT Slice Field

**Generated Go (After Fix):**
```go
type FrameOutput struct {
    Draw []*DrawCmd  // Pointer slice - correct!
}
```

### Example 4: Slice Type Assertions (Phase 4)

**Before Phase 4 Fix (wrong):**
```go
tmp58.([]**CrewPosition)  // Double pointer - wrong!
patrolPath.([]**Direction)
```

**After Phase 4 Fix (correct):**
```go
tmp58.([]*CrewPosition)   // Single pointer - correct!
patrolPath.([]*Direction)
```

## Success Criteria

- [x] `ailangTypeToGo` returns `*TypeName` for user-defined types
- [x] `mapASTType` returns `*TypeName` for user-defined SimpleTypes
- [x] List/Array types generate `[]*TypeName` (not `[]**TypeName`)
- [x] Slice type assertions use `[]*TypeName` (not `[]**TypeName`)
- [x] Build passes
- [x] All existing tests pass (with updated expectations)
- [x] Documentation updated

## Testing Strategy

**Unit tests updated:**
- `TestMapASTType_Primitives` - expects `*Tree` for user-defined types
- `TestGenerateSumType_Tree` - expects `*Tree` for recursive fields
- `TestMapASTType_ADTSlices` - verifies `[]*DrawCmd` format preserved
- `TestGenerateRecordWithADTSlice` - verifies `[]*DrawCmd` in records

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Double-pointer for list elements | High | Fixed: Check for existing `*` prefix before adding |
| Test failures | Medium | Updated test expectations |
| Cascading issues | Medium | Full test suite passes |

## References

- Bug reports from stapledons_voyage (agent messages Dec 9, 2025)
- `mapASTType` in `internal/gen/golang/adt.go:283-330`
- `ailangTypeToGo` in `cmd/ailang/compile.go:635-677`

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
**Phase 1 complete**: 2025-12-09
**Phase 2 complete**: 2025-12-09
**Phase 3 complete**: 2025-12-09
**Phase 4 complete**: 2025-12-09
