# M-CODEGEN-RECORD-TYPENAME-PRESERVATION: Fix TypeName Loss During Substitution

**Status**: Implemented
**Target**: v0.5.10
**Priority**: P0 (High) - Blocked stapledons_voyage compilation
**Actual Time**: ~5 minutes
**Dependencies**: None
**Created**: 2025-12-12
**Implemented**: 2025-12-12

## Problem Statement

Record literals compile to `map[string]interface{}` instead of typed structs because `TRecord.TypeName` is lost during substitution.

**Error from stapledons_voyage:**
```
panic: ConvertToPlanetSlice: element 0: expected *Planet, got map[string]interface {}
```

**Generated Go code (WRONG):**
```go
// In makePlanet_impl
return map[string]interface{}{"orbitDistance": dist, "radius": rad, ...}
```

**Expected Go code:**
```go
// In makePlanet_impl
return &Planet{OrbitDistance: dist, Radius: rad, ...}
```

## Root Cause Analysis

### Code Location: `internal/types/unification_substitution.go:141-162`

```go
case *TRecord:
    changed := false
    fields := make(map[string]Type)
    for name, fieldType := range typ.Fields {
        fields[name] = safeSubstitute(fieldType, sub, visited)
        if fields[name] != fieldType {
            changed = true
        }
    }
    var row Type
    if typ.Row != nil {
        row = safeSubstitute(typ.Row, sub, visited)
        if row != typ.Row {
            changed = true
        }
    }
    if !changed {
        return t
    }
    result := &TRecord{Fields: fields, Row: row}  // BUG: TypeName not preserved!
    visited[t] = result
    return result
```

**The bug:** When field types change during substitution, a NEW `TRecord` is created without copying `TypeName`. This causes:
1. Record literal gets type `TRecord{Fields: {...}, TypeName: "Planet"}` during type checking
2. Substitution runs to resolve type variables in field types
3. If ANY field changes, a new TRecord is created with `TypeName: ""`
4. CodeGen sees `TypeName == ""` and falls back to `map[string]interface{}`

### Chain of Events

1. **Type Checking**: `makePlanet() -> Planet` unifies return type with record literal
2. **Alias Expansion**: `expandAlias` correctly sets `TypeName: "Planet"` on the TRecord
3. **CoreTypeInfo Population**: TRecord with TypeName stored for record's NodeID
4. **Substitution**: `ApplySubstitution` called on CoreTypeInfo
5. **TypeName Lost**: If any field type changed, TypeName becomes ""
6. **CodeGen Fallback**: `generateRecord` checks `tRec.TypeName != ""` and falls through

## Solution

The fix required changes in **two** locations:

### Fix 1: Preserve TypeName in safeSubstitute

**Change `unification_substitution.go:160`:**

```go
// Before (BUG):
result := &TRecord{Fields: fields, Row: row}

// After (FIX):
result := &TRecord{Fields: fields, Row: row, TypeName: typ.TypeName}
```

This preserves TypeName when substitution creates a new TRecord with modified field types.

### Fix 2: Propagate TypeName during TRecord unification

**Change `unification_records.go:81-90`:**

The initial fix wasn't sufficient because TypeName was never SET on the record literal's type in the first place. When `expandAlias` creates a TRecord with TypeName, and this is unified with the record literal's TRecord (which has no TypeName), the TypeName wasn't being propagated.

```go
// After field unification, propagate TypeName
if t1.TypeName == "" && t2Rec.TypeName != "" {
    t1.TypeName = t2Rec.TypeName
} else if t2Rec.TypeName == "" && t1.TypeName != "" {
    t2Rec.TypeName = t1.TypeName
}
```

This ensures that when a record literal (no TypeName) is unified with a type alias expansion (has TypeName), the nominal type identity is propagated to the record literal.

**Why both fixes are needed:**
1. Fix 1 preserves TypeName through substitution (doesn't lose it)
2. Fix 2 sets TypeName in the first place during unification (propagates it)

## Success Criteria

- [x] `TRecord.TypeName` preserved through substitution
- [x] `makePlanet` generates `&Planet{...}` instead of `map[string]interface{}{...}`
- [x] stapledons_voyage compiles without panic
- [x] All existing tests pass

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/types/unification_substitution.go` | Add `TypeName: typ.TypeName` | +1 |
| `internal/types/unification_records.go` | Propagate TypeName during TRecord unification | +8 |
| `internal/types/unification_substitution_test.go` | Regression tests for TypeName preservation | +120 |
| `internal/gen/golang/codegen_ops.go` | Enhanced DEBUG_CODEGEN logging | +12 |

**Total:** ~141 LOC

## Test Plan

1. **Unit Test:** Add test case for TRecord TypeName preservation in substitution
2. **Integration:** Compile stapledons_voyage `sim/celestial.ail`
3. **Regression:** Run `make test`

## Pattern Analysis: Recurring Type Information Loss

This is the **fourth** codegen bug in v0.5.10 caused by type information being lost during AST transformation:

| Bug | Lost Information | Location |
|-----|------------------|----------|
| M-CODEGEN-CROSS-MODULE-IMPL | `_impl` version not used for imports | `codegen_expr_simple.go` |
| M-CODEGEN-ADT-TYPE-ASSERT | Nullary constructors misidentified | `codegen_decl.go` |
| M-CROSS-MODULE-RECORD-UNIFICATION | Nested type aliases not imported | `pipeline_module.go` |
| **This bug** | `TRecord.TypeName` lost in substitution | `unification_substitution.go` |

### Common Root Cause

All these bugs share a common pattern:
- **Type metadata is not preserved during transformation**
- Original type information exists but is lost when creating new type objects
- CodeGen relies on this metadata but silently falls back when missing

### Recommended Invariant

Add validation that critical type metadata is preserved:

```go
// In TRecord.Copy() or wherever types are cloned:
func (t *TRecord) Clone() *TRecord {
    // INVARIANT: TypeName must be preserved
    return &TRecord{
        Fields:   cloneFields(t.Fields),
        Row:      t.Row,
        TypeName: t.TypeName,  // Always preserve!
    }
}
```

### Future Prevention

1. **Add a centralized type clone function** that ensures all fields are copied
2. **Add validation tests** that check TypeName preservation through substitution
3. **Consider making TypeName mandatory** for record types in function return positions

## References

- Agent Inbox: stapledons_voyage message `fa81ba7c-f864-4ecc-8aa2-fa9afc9902d9`
- Related: M-CODEGEN-ADT-TYPE-ASSERT (v0.5.10)
- Related: M-CROSS-MODULE-RECORD-UNIFICATION (v0.5.10)
- Related: M-CODEGEN-CROSS-MODULE-IMPL (v0.5.10)

---

**Document created**: 2025-12-12
