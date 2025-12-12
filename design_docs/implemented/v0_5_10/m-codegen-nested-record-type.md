# M-CODEGEN-NESTED-RECORD-TYPE: Fix Nested Record Type Selection

**Status**: Implemented
**Target**: v0.5.10
**Priority**: P0 (High) - Caused runtime panics in stapledons_voyage
**Dependencies**: M-CODEGEN-RECORD-TYPENAME-PRESERVATION
**Created**: 2025-12-12
**Implemented**: 2025-12-12
**GitHub Issue**: #38

## Problem Statement

Nested record literals within typed struct fields generate the wrong struct type when multiple types share the same field structure.

**Error from stapledons_voyage:**
```go
// Generated code (WRONG):
var tmp14 interface{} = &Vec3{X: float64(0), Y: float64(0), Z: float64(0)}
return &StarSystem{...Position: tmp14.(*SystemPos)...}
// Runtime: panic: interface conversion: interface {} is *sim_gen.Vec3, not *sim_gen.SystemPos
```

Both `Vec3` and `SystemPos` have identical structure `{x: float, y: float, z: float}` but are distinct Go types.

## Root Cause Analysis

The codegen used `GetRecordTypeByFields` which returns the **first** matching type by field structure. When multiple types share the same fields, it could return the wrong one.

**Code flow before fix:**
1. Nested record literal `{x: 0.0, y: 0.0, z: 0.0}` in `position` field
2. `generateTypedRecord` called with parent record info
3. For nested pointer fields (`*SystemPos`), called `generateExpr(nestedRec)` directly
4. `generateRecord` checked TypeName (empty!) then used `GetRecordTypeByFields`
5. Found `Vec3` (wrong type but same fields)

## Solution

When generating nested records within a typed struct field, use the **expected field type** directly instead of relying on `GetRecordTypeByFields`.

**Change in `codegen_ops.go:201-217`:**

```go
} else if nestedRec, isRec := value.(*core.Record); isRec {
    if strings.HasPrefix(goType, "*") {
        // M-CODEGEN-NESTED-RECORD-TYPE: Use expected field type, not GetRecordTypeByFields
        // This fixes Vec3/SystemPos confusion when multiple types have same structure
        expectedTypeName := strings.TrimPrefix(goType, "*")
        if expectedType, exists := g.recordTypes[expectedTypeName]; exists {
            if err := g.generateTypedRecord(nestedRec, expectedType); err != nil {
                return err
            }
        } else {
            // Fallback to default generation
            if err := g.generateExpr(nestedRec); err != nil {
                return err
            }
        }
    }
```

## Why TypeName Propagation Didn't Work

The TypeName propagation fix (M-CODEGEN-RECORD-TYPENAME-PRESERVATION) works for top-level record literals but not for nested records because:

1. Nested record literals are elaborated to temp variables before use
2. When the temp variable's record is type-checked, it's unified with the field type
3. The unification should propagate TypeName, but for nested records the TRecord object in CoreTypeInfo appears to be a different instance

This is a deeper issue with how nested record types are tracked through elaboration → type checking → CoreTypeInfo storage.

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/gen/golang/codegen_ops.go` | Use expected field type for nested records | +10 |

## Success Criteria

- [x] `{x: 0.0, y: 0.0, z: 0.0}` in `position: SystemPos` field generates `&SystemPos{...}`
- [x] stapledons_voyage sim_gen compiles without panic
- [x] All existing tests pass

## Related Issues

- M-CODEGEN-RECORD-TYPENAME-PRESERVATION (same root cause: type identity not preserved)
- Multiple duplicate type definitions in stapledons_voyage:
  - `Vec3` and `SystemPos` (both `{x, y, z: float}`)
  - `Planet` in bridge.ail vs celestial.ail (different fields but same name)

## Recommendations for stapledons_voyage

1. Remove duplicate type definitions with same structure
2. Either merge `Vec3` and `SystemPos` into one type, or differentiate their fields
3. Rename one of the `Planet` types (e.g., `DomePlanet`, `CelestialPlanet`)

---

**Document created**: 2025-12-12
