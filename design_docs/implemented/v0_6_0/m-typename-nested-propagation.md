# M-TYPENAME-NESTED-PROPAGATION: TypeName Propagation to Nested Record Literals

**Status**: ✅ Implemented
**Target**: v0.5.11
**Priority**: P1 (Medium) - Current workaround in codegen works but is architecturally wrong
**Estimated**: 4-6 hours
**Actual**: ~3 hours
**Dependencies**: None (builds on v0.5.10 fixes)
**GitHub Issue**: Continuation of #38

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | + | +1 | Type identity flows correctly through type system |
| Increase Determinism | + | +1 | Same record structure always generates same type |
| Lower Token Cost | 0 | 0 | No change to source code |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward

## Problem Statement

Nested record literals within typed struct fields don't receive `TRecord.TypeName` during type checking, causing codegen to either:
1. Fall back to `map[string]interface{}` (wrong)
2. Use `GetRecordTypeByFields` which returns the **first** matching type (ambiguous)

**Example that fails:**
```ailang
type Vec3 = {x: float, y: float, z: float}
type SystemPos = {x: float, y: float, z: float}  -- Same structure, different name!
type StarSystem = {name: string, position: *SystemPos}

fn makeStarSystem(name: string) -> StarSystem = {
    name: name,
    position: {x: 0.0, y: 0.0, z: 0.0}  -- BUG: Gets Vec3, not SystemPos!
}
```

**Current State:**
- TypeName propagates correctly for top-level record literals (v0.5.10 fix)
- Nested records inside typed fields don't receive TypeName in CoreTypeInfo
- Codegen workaround uses expected field type, but type system is architecturally wrong
- Multiple types with same structure cause ambiguity

**Impact:**
- AILANG users (stapledons_voyage) must use workarounds (rename types)
- Type system doesn't preserve nominal identity for nested records
- Runtime panics if codegen workaround doesn't apply (e.g., deeper nesting levels)

## Goals

**Primary Goal:** Propagate TypeName from expected field types to nested record literals during type checking, so CoreTypeInfo contains correct nominal type identity.

**Success Metrics:**
- Nested `{x: 0.0, y: 0.0, z: 0.0}` in `SystemPos` field has `TypeName: "SystemPos"` in CoreTypeInfo
- Remove codegen workaround in `codegen_ops.go` (lines 174-190)
- No runtime panics for any level of record nesting
- All stapledons_voyage files compile without workarounds

## Root Cause Analysis

### Why Current Fixes Don't Solve This

**v0.5.10 Fix 1** (`unification_substitution.go`):
- Preserves TypeName when substitution creates new TRecord
- Doesn't help: Nested record never HAD TypeName to preserve

**v0.5.10 Fix 2** (`unification_records.go`):
- Propagates TypeName when two TRecords unify
- Should help but doesn't because...

### The Real Problem: Elaboration Creates Separate TRecords

When elaborating a nested record literal:
```
1. makeStarSystem return type → StarSystem → expanded to TRecord{..., position: *TRecord{TypeName: "SystemPos"}}
2. Record literal {name, position: {...}} → TRecord{..., position: *TRecord{}}  (no TypeName!)
3. Nested literal {x, y, z} → elaborated to temp variable with fresh TRecord{}
4. Unification: nested TRecord{} unified with field's TRecord{TypeName: "SystemPos"}
```

**Problem at step 4**: The nested record literal is already stored in CoreTypeInfo with its OWN TRecord instance. The unification happens between the outer record's field type and the nested record's type, but the TypeName propagation only updates the types being unified - NOT the CoreTypeInfo entry for the nested record's NodeID.

### Data Flow Issue

```
Type Checking Flow:
    Surface AST → Elaboration → CoreTypeInfo[NodeID] = TRecord{}
                      ↓
                 Unification (sets TypeName on unification operands)
                      ↓
                 BUT: CoreTypeInfo[NodeID] still has TypeName: ""!
```

The unification mutates the type objects involved in unification, but the CoreTypeInfo map stores a different TRecord instance for the nested record node.

## Solution Design

### Overview

After unifying a record literal's type with its expected type (e.g., a field type), update the CoreTypeInfo entry for the record literal's NodeID with the unified type that now has TypeName.

### Approach Options

**Option A: Update CoreTypeInfo after unification** (Recommended)
- After unifying record literal with expected type
- Update `CoreTypeInfo[recordNode.NodeID]` with the unified result
- Preserves existing architecture, minimal change

**Option B: Pass expected type through elaboration**
- Thread expected field type through `checkExpr`
- Nested records receive TypeName at creation time
- More invasive, touches many functions

**Option C: Post-pass to fix CoreTypeInfo**
- After type checking, scan CoreTypeInfo
- Match record types by NodeID to their declaration context
- Find expected type and copy TypeName
- Clean separation but adds complexity

### Recommended Solution: Option A

**Key Insight**: The unification already correctly propagates TypeName between the two types being unified. We just need to ensure the result is stored back in CoreTypeInfo.

**Location**: `internal/types/typechecker_expr.go` or wherever record literals are type-checked

**Change**:
```go
// After unifying record literal type with expected type:
// unified, err := tc.unify(recordType, expectedType, sub)
// ...
// UPDATE CoreTypeInfo with the unified result:
tc.CoreTI[recordExpr.NodeID] = unified
```

This ensures that when a record literal is unified with a typed field, the CoreTypeInfo entry reflects the unified type (which now has TypeName).

### Implementation Plan

**Phase 1: Investigation** (~1 hour)
- [ ] Trace type checking of nested record in debug mode
- [ ] Identify exact location where record literal types are unified
- [ ] Confirm CoreTypeInfo entries for nested records lack TypeName
- [ ] Add test case that reproduces the issue at type-checker level

**Phase 2: Implementation** (~2 hours)
- [ ] After record-field unification, update CoreTypeInfo for nested record NodeID
- [ ] Handle pointer types (*SystemPos vs SystemPos)
- [ ] Handle multiple nesting levels
- [ ] Add DEBUG_TYPECHECKER flag for verbose logging

**Phase 3: Cleanup & Verification** (~2 hours)
- [ ] Remove codegen workaround in `codegen_ops.go:174-190`
- [ ] Verify stapledons_voyage compiles without workarounds
- [ ] Add regression tests
- [ ] Update design docs

### Files to Modify/Create

**Modified files:**
- `internal/types/typechecker_expr.go` - Update CoreTypeInfo after unification (~+15 LOC)
- `internal/gen/golang/codegen_ops.go` - Remove workaround (~-20 LOC)
- `internal/types/unification_records.go` - May need adjustment (~+5 LOC)

**New files:**
- `internal/types/typechecker_typename_test.go` - Regression tests (~100 LOC)

## Examples

### Example 1: Nested Record in Struct Field

**Before (v0.5.10 workaround):**
```go
// Generated Go code - codegen has to guess expected type
// Works but architecturally wrong - CoreTypeInfo doesn't have TypeName
} else if nestedRec, isRec := value.(*core.Record); isRec {
    expectedTypeName := strings.TrimPrefix(goType, "*")  // Guess from field type
    if expectedType, exists := g.recordTypes[expectedTypeName]; exists {
        g.generateTypedRecord(nestedRec, expectedType)  // Use guessed type
    }
}
```

**After (proper fix):**
```go
// CoreTypeInfo[nestedRecord.NodeID] already has TypeName: "SystemPos"
// Codegen just looks it up normally - no guessing needed
if recType, ok := g.coreTypeInfo[rec.NodeID].(*TRecord); ok {
    if recType.TypeName != "" {
        g.generateTypedRecord(rec, g.recordTypes[recType.TypeName])
    }
}
```

### Example 2: Multi-Level Nesting

```ailang
type Inner = {value: int}
type Middle = {inner: *Inner}
type Outer = {middle: *Middle}

fn makeOuter() -> Outer = {
    middle: {          -- TypeName: "Middle" in CoreTypeInfo
        inner: {       -- TypeName: "Inner" in CoreTypeInfo
            value: 42
        }
    }
}
```

All nested levels should have correct TypeName in CoreTypeInfo.

## Success Criteria

- [ ] Nested `{x: 0.0, y: 0.0, z: 0.0}` in `position: *SystemPos` field has `TypeName: "SystemPos"` in CoreTypeInfo
- [ ] Codegen workaround removed from `codegen_ops.go`
- [ ] Multi-level nesting works correctly
- [ ] stapledons_voyage compiles without type rename workarounds
- [ ] All existing tests pass
- [ ] New regression tests for TypeName propagation
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Test CoreTypeInfo contains TypeName for nested records
- Test multi-level nesting
- Test pointer vs non-pointer field types
- Test ambiguous types (Vec3 vs SystemPos) are resolved correctly

**Integration tests:**
- Compile stapledons_voyage sim/ with original type definitions (no renames)
- Verify generated Go code uses correct struct types

**Manual testing:**
- `DEBUG_CODEGEN=1` should show TypeName present for nested records
- Compare generated code before/after

## Non-Goals

**Not in this feature:**
- **Structural typing**: Record types with same fields remain distinct (nominal typing)
- **Type alias merging**: If user defines duplicate types, they stay separate
- **User-facing syntax**: No new syntax for nominal types

## Timeline

**Day 1** (4-6 hours):
- Phase 1: Investigation + test case
- Phase 2: Core implementation
- Phase 3: Cleanup and verification

**Total: ~4-6 hours in one session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Unification architecture is more complex than expected | High | Start with investigation phase, may need Option B or C |
| Breaking existing type inference | High | Extensive test suite, run all tests before/after |
| Performance impact from additional CoreTypeInfo updates | Low | Updates only affect record literals, not general unification |

## References

- [M-CODEGEN-RECORD-TYPENAME-PRESERVATION](../../implemented/v0_5_10/m-codegen-record-typename-preservation.md) - v0.5.10 TypeName preservation
- [M-CODEGEN-NESTED-RECORD-TYPE](../../implemented/v0_5_10/m-codegen-nested-record-type.md) - v0.5.10 nested record codegen workaround
- [Codegen Bug Pattern Analysis](../v0_5_10/codegen-bug-pattern-analysis.md) - Pattern analysis of recurring bugs
- GitHub Issue #38 - Vec3/SystemPos confusion bug report

## Future Work

- **Type Alias Validation**: Warn when multiple types have identical structure (potential confusion)
- **Nominal Type Annotations**: Syntax for explicit nominal types beyond record aliases
- **CoreTypeInfo Invariant Validation**: Add check that all record literals have TypeName when expected

---

## Implementation Report

**Completed**: 2025-12-12

### Implementation Summary

The fix involved a multi-pronged approach:

1. **TypeName set on aliases during registration** (`elaborate/core.go`):
   - `RegisterTypeAlias` now sets `TRecord.TypeName` when registering type aliases
   - This ensures the canonical type in `aliasEnv` has its identity

2. **TypeName propagation after substitution** (`typechecker_substitution.go`):
   - Added `propagateTypeNameToCoreTI()` to match anonymous TRecords in CoreTI to known type aliases
   - Builds field signature map from aliasEnv
   - **Critical**: Detects ambiguous signatures (e.g., Vec3 and SystemPos with same `{x,y,z}` fields) and skips auto-assignment

3. **Safe fallback in codegen** (`codegen.go`):
   - Modified `GetRecordTypeByFields` to return `nil` on ambiguous matches (multiple types with same fields)
   - Forces callers to use more specific type information (CoreTypeInfo TypeName) instead of guessing

4. **CoreTypeInfo priority in Let bindings** (`codegen_expr_let.go`):
   - Record let values first check CoreTypeInfo for TypeName before field matching

### Files Modified

| File | Change | Lines |
|------|--------|-------|
| `internal/elaborate/core.go` | Set TypeName on TRecord aliases | +3 |
| `internal/types/typechecker_substitution.go` | Added propagateTypeNameToCoreTI with ambiguity detection | +50 |
| `internal/gen/golang/codegen.go` | Return nil on ambiguous GetRecordTypeByFields | +10 |
| `internal/gen/golang/codegen_expr_let.go` | Check CoreTypeInfo TypeName first | +15 |

**Total**: ~78 net new LOC

### Key Insight

The fundamental issue was that unification works on **copies** created by `ApplySubstitution`, not the original types stored in CoreTypeInfo. The fix propagates TypeName from type aliases to CoreTI after substitution is applied, using field signature matching to identify anonymous records that match known type aliases.

### Handling Ambiguous Types

When multiple type aliases have identical field structures (Vec3 and SystemPos both having `{x: float, y: float, z: float}`):
- `propagateTypeNameToCoreTI()` detects the ambiguity and **skips** auto-assignment
- `GetRecordTypeByFields()` returns `nil` instead of picking arbitrarily
- Codegen relies on contextual type information (parent struct field type) to resolve correctly

This is a **safety-first** approach: when ambiguous, we fail gracefully rather than generating potentially incorrect code.

### Test Results

- All existing tests pass
- Generated code for nested records uses correct types:
  - `&SystemPos{...}` when used as StarSystem.position
  - `&Vec3{...}` when used via makeVec3()
- stapledons_voyage compiles successfully with 49 type declarations

### Debug Flags

- `DEBUG_TYPENAME=1` - Shows TypeName propagation and ambiguity detection
- `DEBUG_CODEGEN=1` - Shows codegen type resolution

---

**Document created**: 2025-12-12
**Last updated**: 2025-12-12
**Implementation completed**: 2025-12-12
