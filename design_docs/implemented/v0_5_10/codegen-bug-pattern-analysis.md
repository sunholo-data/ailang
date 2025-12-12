# Codegen Bug Pattern Analysis: v0.5.10

**Status**: Analysis Document
**Created**: 2025-12-12

## Executive Summary

v0.5.10 has seen **four** codegen bugs with the same root cause: **type metadata is not preserved during AST transformations**. This document analyzes the pattern and proposes systematic fixes.

## The Four Bugs

| ID | Bug | Root Cause | Fix Location |
|----|-----|------------|--------------|
| 1 | M-CODEGEN-CROSS-MODULE-IMPL | Cross-module functions not calling `_impl` | `codegen_expr_simple.go` |
| 2 | M-CODEGEN-ADT-TYPE-ASSERT | Nullary ADT constructors not recognized | `codegen_decl.go` |
| 3 | M-CROSS-MODULE-RECORD-UNIFICATION | Nested type aliases not imported | `pipeline_module.go`, `unification_records.go` |
| 4 | M-CODEGEN-RECORD-TYPENAME-PRESERVATION | `TRecord.TypeName` lost in substitution | `unification_substitution.go` |

## Common Pattern: Silent Fallback on Missing Metadata

All four bugs share a common anti-pattern:

```go
// Anti-pattern: Silent fallback when metadata is missing
if metadata != nil {
    // Use typed version
    return generateTypedCode(metadata)
}
// SILENT FALLBACK: Generate generic version
return generateGenericCode()  // BUG HIDES HERE
```

This pattern appears in:

### Bug 1: Cross-module function calls
```go
// codegen_expr_simple.go:62-67
if _, isTopLevel := g.topLevelFuncs[e.Ref.Name]; isTopLevel {
    g.write(ToGoVarName(e.Ref.Name) + "_impl")  // Uses _impl
    return nil
}
g.write(ToPascalCase(e.Ref.Name))  // FALLBACK: typed wrapper (wrong in _impl context!)
```

### Bug 2: ADT constructor detection
```go
// codegen_decl.go (before fix)
if app, ok := expr.(*core.App); ok {
    // Check for ADT constructor
}
// FALLBACK: Not ADT, add type assertion (wrong for nullary constructors!)
```

### Bug 3: Type alias lookup
```go
// unification_records.go (before fix)
expanded := u.expandAlias(t2Con)
if expanded != t2Con {
    return u.Unify(t1, expanded, sub)
}
// FALLBACK: Can't expand, error (wrong when alias not imported!)
```

### Bug 4: Record type preservation
```go
// codegen_ops.go:76-83
if recType, ok := g.coreTypeInfo[rec.NodeID]; ok {
    if tRec, ok := recType.(*TRecord); ok && tRec.TypeName != "" {
        return g.generateTypedRecord(rec, info)
    }
}
// FALLBACK: map[string]interface{} (wrong when TypeName was lost!)
```

## Why These Bugs Are Hard to Catch

1. **Tests pass**: Most tests don't exercise cross-module or complex type scenarios
2. **Happy path works**: Simple cases (single module, simple types) work correctly
3. **Error is distant**: Bug manifests at runtime (Go compile error or panic), not at AILANG compile time
4. **Fallback is plausible**: `map[string]interface{}` is a valid Go type, just not the RIGHT type

## Root Cause: Type Information Flow

The type information flows through multiple stages:

```
Surface AST → Type Check → CoreTypeInfo → Substitution → CodeGen
                  ↓              ↓              ↓           ↓
            TypeName set   Stored in CTI  Lost here!   Fallback triggers
```

The problem is that each transformation stage can lose metadata:

1. **Type Checking**: Sets metadata (TypeName, constructor info)
2. **CoreTypeInfo Storage**: Correctly preserves metadata
3. **Substitution**: LOSES metadata when creating new type objects
4. **CodeGen**: Falls back silently when metadata missing

## Systematic Fixes

### Fix 1: Preserve TypeName in Substitution (Immediate) ✅ IMPLEMENTED

```go
// unification_substitution.go:160 - ONE LINE FIX
result := &TRecord{Fields: fields, Row: row, TypeName: typ.TypeName}
```

### Fix 2: Propagate TypeName During Unification (Immediate) ✅ IMPLEMENTED

Fix 1 alone wasn't sufficient - TypeName also needs to be propagated when two TRecords are unified:

```go
// unification_records.go:81-90
// After field unification, propagate TypeName
if t1.TypeName == "" && t2Rec.TypeName != "" {
    t1.TypeName = t2Rec.TypeName
} else if t2Rec.TypeName == "" && t1.TypeName != "" {
    t2Rec.TypeName = t1.TypeName
}
```

**Why both fixes are needed:**
- Fix 1 preserves TypeName through substitution (doesn't lose it)
- Fix 2 sets TypeName in the first place during unification (propagates it from type alias to record literal)

### Fix 3: Add Type Clone Helper (Medium-term)

Create a centralized clone function that guarantees all metadata is preserved:

```go
// types/clone.go
func CloneTRecord(t *TRecord) *TRecord {
    return &TRecord{
        Fields:   cloneFields(t.Fields),
        Row:      t.Row,
        TypeName: t.TypeName,  // INVARIANT: Always preserve
    }
}
```

### Fix 3: Add Validation (Long-term)

Add a validation pass that checks metadata preservation:

```go
// pipeline/validate_type_metadata.go
func ValidateTypeMetadata(cti types.CoreTypeInfo, expectedReturnTypes map[string]string) error {
    for nodeID, typ := range cti {
        if rec, ok := typ.(*types.TRecord); ok {
            // If this node is a function return, TypeName should be set
            if expectedTypeName, expected := expectedReturnTypes[nodeID]; expected {
                if rec.TypeName != expectedTypeName {
                    return fmt.Errorf("TypeName lost for node %d: expected %s, got %s",
                        nodeID, expectedTypeName, rec.TypeName)
                }
            }
        }
    }
    return nil
}
```

## Locations to Audit

All places where new `TRecord` is created should be audited:

| Location | TypeName Preserved? | Action |
|----------|---------------------|--------|
| `unification_substitution.go:160` | YES | **Fixed** - Added `TypeName: typ.TypeName` |
| `unification_records.go:406` | N/A | TRecord2 doesn't have TypeName field |
| `typechecker_data.go:37` | N/A | Initial creation, TypeName set later |
| `unification_core.go:68` | YES | OK |
| `types.go:272` | YES | OK (Copy method) |

## Prevention Checklist

For future type transformation code:

- [x] When creating new type objects, copy ALL fields from original
- [x] Add tests that verify metadata preservation through transformation
  - Added `unification_substitution_test.go` with 3 tests for TypeName preservation
- [x] Avoid silent fallbacks - log warnings or fail explicitly
  - Added `DEBUG_CODEGEN=1` warning in `codegen_ops.go` when record falls back to map
- [ ] Add cross-module test cases for complex type scenarios
- [ ] Consider making critical metadata (TypeName) non-optional with validation

## Conclusion

The recurring bugs stem from a systemic issue: **type metadata is treated as optional throughout the codebase**. The fix requires:

1. **Immediate**: ✅ Patch the substitution bug (1 line)
2. **Short-term**: ✅ Audit all TRecord creation sites
3. **Long-term**: Add validation and make metadata mandatory where appropriate

The AILANG principle of "fail loudly" (CLAUDE.md section 2) should apply to type metadata - missing metadata should cause explicit errors, not silent fallbacks to generic types.
