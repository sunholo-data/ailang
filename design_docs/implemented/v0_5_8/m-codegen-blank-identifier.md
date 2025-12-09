# M-CODEGEN-BLANK: Fix Blank Identifier and Return Type Codegen Bugs

**Status**: Implemented
**Version**: v0.5.8
**Priority**: P0 (Blocking - stapledons_voyage sprint is waiting)
**Estimated**: 3-4 hours
**Actual**: 2 hours
**Reported by**: stapledons_voyage project
**Completed**: 2025-12-09

## Problem Statement

The Go codegen has multiple bugs that produce invalid or incorrect Go code:

### Bug 1: Blank Identifier Used as Value (High Priority)

When an AILANG function has an unused parameter (common pattern: `\_ . expr`), the codegen generates:

```go
func TileFloor(_ int64) int64 {
    return tileFloor_impl(_).(int64)  // ERROR: cannot use _ as value
}
```

**Root Cause**: In `generateTypedWrapper()`, when building `callArgs`:
```go
callArgs = append(callArgs, ToGoVarName(p))  // ToGoVarName("_") returns "_"
```

The blank identifier `_` is valid as a parameter name (to discard), but CANNOT be used as a value in function calls.

### Bug 2: Incorrect Return Types for Exported Functions (Medium Priority)

Exported functions have wrong return types that don't match the implementation:

```go
// Generated:
func InitBridge(_ int64) struct{} { ... }

// But _impl returns:
func initBridge_impl(...) interface{} {
    return &BridgeState{...}  // Actually returns *BridgeState
}
```

**Impact**:
- External Go code cannot use the typed wrapper correctly
- Forces consumers to call unexported `_impl` functions (not possible from other packages)
- Breaks the type-safe API promise of the codegen

### Bug 3: Unexported Slice Converters (Medium Priority)

Slice converter functions are generated as unexported:

```go
func convertToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
```

But the wrapper tries to use them:
```go
func RenderBridge(...) []*DrawCmd {
    return convertToDrawCmdSlice(renderBridge_impl(...))
}
```

This works internally but consumers cannot access the converter if they need it.

## Proposed Solutions

### Fix 1: Generate Placeholder Names for Unused Parameters

Replace `_` with a numbered placeholder that IS used in the call:

```go
// In generateTypedWrapper():
for i, p := range lam.Params {
    paramName := ToGoVarName(p)
    if paramName == "_" {
        // Generate a placeholder name that can be used as a value
        paramName = fmt.Sprintf("_unused%d", i)
    }
    params = append(params, fmt.Sprintf("%s %s", paramName, pType))
    callArgs = append(callArgs, paramName)
}
```

**Result:**
```go
func TileFloor(_unused0 int64) int64 {
    return tileFloor_impl(_unused0).(int64)  // Valid Go!
}
```

**Alternative (Zero-arg Functions)**: If ALL parameters are `_`, generate a parameterless wrapper:
```go
func TileFloor() int64 {  // No params at all
    return tileFloor_impl(0).(int64)  // Pass dummy value
}
```

### Fix 2: Improve Return Type Inference

The `getTypedSignature()` function needs to trace through type aliases and ADT constructors:

1. For struct-returning functions: Use CoreTypeInfo to get the actual struct type
2. For ADT returns: Map to the generated Go ADT pointer type
3. Fall back to `interface{}` only when truly polymorphic

```go
func (g *Generator) getTypedSignature(lam *core.Lambda) ([]GoType, GoType) {
    // Existing lookup...

    // For struct types, use the registered Go struct name
    if structName, ok := g.structTypes[typeName]; ok {
        return paramTypes, GoType("*" + structName)
    }

    // For ADT types, use the registered ADT name
    if adtName, ok := g.adtTypes[typeName]; ok {
        return paramTypes, GoType("*" + adtName)
    }

    return paramTypes, returnType
}
```

### Fix 3: Export Slice Converters

Change converter naming to be exported when the element type is exported:

```go
func (g *Generator) getSliceConversion(retType string) string {
    // Extract element type
    elemType := strings.TrimPrefix(retType, "[]")
    elemType = strings.TrimPrefix(elemType, "*")

    // Generate exported converter name
    return "ConvertTo" + elemType + "Slice"
}
```

**Result:**
```go
// Exported converter (usable from other packages)
func ConvertToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
```

## Implementation Plan

### Phase 1: Blank Identifier Fix (1 hour)
1. Modify `generateTypedWrapper()` in `codegen_decl.go`
2. Also check `generateImplFunc()` for same issue
3. Add test case with `_` parameter
4. Verify with stapledons_voyage `TileFloor` function

### Phase 2: Return Type Fix (1.5 hours)
1. Improve `getTypedSignature()` type resolution
2. Add struct and ADT type tracking during declaration pass
3. Add test case for struct-returning functions
4. Verify with stapledons_voyage `InitBridge` function

### Phase 3: Converter Export Fix (0.5 hours)
1. Change `convertTo*Slice` to `ConvertTo*Slice`
2. Update all references
3. Add test for exported converters

### Phase 4: Verification (1 hour)
1. Run full test suite
2. Regenerate stapledons_voyage code
3. Verify compilation succeeds
4. Unblock sprint Session 4

## Files to Modify

| File | Changes |
|------|---------|
| `internal/gen/golang/codegen_decl.go` | Fix blank identifier, improve type inference |
| `internal/gen/golang/codegen_test.go` | Add tests for `_` param and return types |
| `internal/gen/golang/naming.go` | Maybe add `IsBlankIdentifier()` helper |

## Acceptance Criteria

- [x] `\_ . expr` functions compile to valid Go
- [x] Exported functions have correct typed signatures matching `_impl`
- [x] Slice converters are exported when element types are exported
- [x] stapledons_voyage bridge code compiles and runs
- [x] All existing tests pass
- [x] No regressions in other codegen output

## Risk Assessment

**Low Risk** - These are targeted bug fixes with clear scope:
- Blank identifier fix is surgical (one location)
- Return type fix requires tracing types but uses existing infrastructure
- Changes are additive (better output, not breaking existing valid output)

## Blocking

This bug is **blocking** the stapledons_voyage bridge-interior-v1 sprint:
- Session 4 (Go BridgeView implementation) cannot proceed
- Sessions 5-8 depend on Session 4
- Timeline impact until fixed: entire sprint paused

## Related Messages

- `msg_20251209_103805_395798e0` - Bug report: blank identifier
- `msg_20251209_103342_cdaab7c8` - Bug report: wrong return types
- `msg_20251209_103700_2ac6ab3c` - DX feedback: blocked sprint

---

## Implementation Report

### What Was Built

**M1 - Blank Identifier Fix (113 LOC)**
- Modified `generateImplFunc()` and `generateTypedWrapper()` in `codegen_decl.go`
- Parameters named `_` are now renamed to `_unused0`, `_unused1`, etc.
- Prevents "cannot use _ as value" Go compile errors

**M2 - Return Type Fix (85 LOC)**
- Added `RecordTypeLookup` callback to `TypeMapper` in `internal/gen/golang/types.go`
- Wired up in `codegen.go` `New()` function
- Updated `TRecord` case to lookup by fields and return proper struct pointer types
- `InitBridge` now correctly returns `*BridgeState` instead of `struct{}`

**M3 - Export Converters (15 LOC)**
- Changed `convertTo*Slice` to `ConvertTo*Slice` (exported)
- Updated in `codegen.go` and `codegen_runtime.go`
- Converters now accessible from external packages

### Code Locations

**Modified files:**
- `internal/gen/golang/codegen_decl.go` - Blank identifier fix (~30 LOC)
- `internal/gen/golang/types.go` - RecordTypeLookup callback (~20 LOC)
- `internal/gen/golang/codegen.go` - Return type wiring, converter export (~25 LOC)
- `internal/gen/golang/codegen_runtime.go` - Converter export (~3 LOC)
- `internal/gen/golang/codegen_test.go` - Tests (~83 LOC)

### Verification

- stapledons_voyage bridge code compiles successfully
- All 5 sprint milestones passed
- Total actual LOC: 213 (estimated: 370)
- Sprint completed in 1 session (estimated: 1 day)

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
