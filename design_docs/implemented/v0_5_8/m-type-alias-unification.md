# M-TYPE-ALIAS: Type Alias Expansion in ADT Variant Parameters

**Status**: Implemented
**Version**: v0.5.8
**Priority**: P1 (Medium - workaround exists but impacts DX)
**Estimated**: 2-3 hours
**Actual**: 1.5 hours
**Reported by**: stapledons_voyage project
**Completed**: 2025-12-09

## Problem Statement

When an ADT variant has a type alias as a parameter, passing a value of that aliased type fails with a unification error:

```ailang
type Coord = { x: int, y: int }

type DrawCmd =
  | Sprite(x: int, y: int, ...)
  | IsoTile(tile: Coord, ...)    -- Type alias parameter

-- This fails:
let tileCoord: Coord = { x: 1, y: 2 }
IsoTile(tileCoord, ...)  -- ERROR: cannot unify type constructor Coord with *types.TRecord
```

Even with explicit type annotation, the unification fails.

**Error Message:**
```
cannot unify type constructor Coord with *types.TRecord
```

## Root Cause Analysis

The unification code in `internal/types/unification.go` handles `TCon` (type constructors) but doesn't expand type aliases before unification.

When the type checker encounters:
1. The ADT variant `IsoTile(tile: Coord, ...)` - `Coord` is stored as `TCon{Name: "Coord"}`
2. The argument `tileCoord` - inferred as `TRecord{Fields: {x: int, y: int}}`

At unification (line 84-96 of unification.go):
```go
case *TCon:
    // Type constructor unification
    if t2Con, ok := t2.(*TCon); ok {
        if t1.Name == t2Con.Name {
            return sub, nil
        }
        return nil, fmt.Errorf("cannot unify type constructors: %s vs %s", t1.Name, t2Con.Name)
    }
    // ...
    return nil, fmt.Errorf("cannot unify type constructor %s with %T", t1.Name, t2)
```

The unifier sees `TCon{Coord}` vs `TRecord{...}` and fails because there's no expansion of type aliases.

## Proposed Solution

Add type alias expansion during unification. When encountering a `TCon` that names a type alias, expand it to the actual type before continuing unification.

### Option A: Expand During Unification (Preferred)

Add alias expansion in the unifier:

```go
func (u *Unifier) Unify(t1, t2 Type, sub Substitution) (Substitution, error) {
    // Apply current substitution
    t1 = ApplySubstitution(sub, t1)
    t2 = ApplySubstitution(sub, t2)

    // NEW: Expand type aliases
    t1 = u.expandAlias(t1)
    t2 = u.expandAlias(t2)

    // ... rest of unification
}

func (u *Unifier) expandAlias(t Type) Type {
    if con, ok := t.(*TCon); ok {
        if alias, exists := u.aliasEnv[con.Name]; exists {
            return alias.Target
        }
    }
    return t
}
```

**Pros:**
- Clean, localized fix
- Transparent to rest of type system
- Easy to test

**Cons:**
- Requires passing alias environment to Unifier

### Option B: Expand at ADT Construction Time

When registering ADT constructors, expand aliases in parameter types:

```go
func (tc *TypeChecker) registerADTConstructor(name string, fields []*ast.ConstructorField) {
    for _, field := range fields {
        fieldType := tc.resolveType(field.Type)
        // NEW: Expand if alias
        if con, ok := fieldType.(*TCon); ok {
            if alias, exists := tc.aliasEnv[con.Name]; exists {
                fieldType = alias.Target
            }
        }
        // ... register constructor with expanded type
    }
}
```

**Pros:**
- One-time expansion
- No unifier changes

**Cons:**
- Loses alias name for error messages
- More invasive change

### Recommendation: Option A

Option A is preferred because:
1. Keeps alias information in AST/types for better error messages
2. Localized change in unification
3. Easy to add alias environment parameter to Unifier

## Implementation Plan

### Phase 1: Add Alias Environment to Unifier (1 hour)

1. Add `aliasEnv map[string]Type` field to `Unifier` struct
2. Create `NewUnifierWithAliases(aliases map[string]Type) *Unifier`
3. Add `expandAlias()` method
4. Call `expandAlias()` at start of `Unify()`

### Phase 2: Wire Alias Environment (0.5 hours)

1. Pass alias environment from TypeChecker to Unifier
2. Collect aliases during type declaration processing
3. Test with simple alias expansion

### Phase 3: Handle Parameterized Aliases (1 hour)

Type aliases can be parameterized:
```ailang
type Pair[a, b] = { first: a, second: b }
type IntPair = Pair[int, int]
```

Need to handle:
1. Simple aliases: `type Coord = { x: int, y: int }`
2. Parameterized aliases: `type Pair[a, b] = ...`
3. Applied aliases: `type IntPair = Pair[int, int]`

### Phase 4: Testing (0.5 hours)

1. Add test for alias in ADT variant
2. Add test for alias in function parameter
3. Add test for parameterized aliases
4. Verify stapledons_voyage IsoTile/IsoEntity work

## Files to Modify

| File | Changes |
|------|---------|
| `internal/types/unification.go` | Add aliasEnv, expandAlias() |
| `internal/types/typechecker.go` | Pass aliases to Unifier |
| `internal/types/unification_test.go` | Add alias tests |

## Workaround

Until this is fixed, users can use the expanded type directly in ADT variants:

```ailang
-- Instead of using alias:
type DrawCmd =
  | IsoTile(tile: Coord, ...)

-- Use expanded type:
type DrawCmd =
  | IsoTile(tile: { x: int, y: int }, ...)
```

Or use different constructors that don't use aliases:
```ailang
-- stapledons_voyage workaround: use Sprite instead of IsoTile
```

## Acceptance Criteria

- [x] Type alias in ADT variant parameter unifies correctly
- [x] Type alias in function parameter works
- [ ] Parameterized type aliases work (NOT IMPLEMENTED - see Non-Goals)
- [x] Error messages still show alias names when helpful
- [x] No regression in existing unification tests
- [x] stapledons_voyage IsoTile/IsoEntity constructors work

## Related Messages

- `msg_20251209_102218_5633bd38` - Bug report: type alias unification failure

---

## Implementation Report

### What Was Built

Implemented Option A (Expand During Unification) with the following components:

**1. Alias Environment in Unifier (~30 LOC)**
- Added `aliasEnv map[string]Type` field to `Unifier` struct
- Created `NewUnifierWithAliases(aliases map[string]Type) *Unifier`
- Added `expandAlias(t Type) Type` method

**2. Alias Registration in TypeChecker (~20 LOC)**
- Added `RegisterTypeAlias(name string, target Type)` to `CoreTypeChecker`
- Aliases are collected during type declaration processing

**3. AST to Internal Type Conversion (~60 LOC)**
- Added `astTypeToInternalType(ast.Type) types.Type` helper in `elaborate/file.go`
- Handles `SimpleType`, `RecordType`, `TupleType`, `FunctionType`, `ListType`, `GenericType`
- Called when processing `*ast.TypeAlias` in `elaborateTypeDecl`

**4. Pipeline Integration (~10 LOC)**
- Wired alias registration from elaborator to type checker in `pipeline_single.go`

### Non-Goals (Deferred)

Parameterized type aliases were NOT implemented in this sprint:
```ailang
-- NOT SUPPORTED YET:
type Pair[a, b] = { first: a, second: b }
type IntPair = Pair[int, int]
```

This is tracked as future work - simple aliases like `type Coord = { x: int, y: int }` work now.

### Code Locations

**Modified files:**
- `internal/types/unification.go` - Added aliasEnv, expandAlias (~45 LOC)
- `internal/types/typechecker.go` - RegisterTypeAlias method (~20 LOC)
- `internal/elaborate/file.go` - astTypeToInternalType helper, TypeAlias case (~70 LOC)
- `internal/pipeline/pipeline_single.go` - Wiring (~5 LOC)
- `internal/types/record_unification_test.go` - Tests for alias expansion (~50 LOC)

### Verification

- All existing unification tests pass (TestTRecord2Unification, TestOpenClosedInteractions, etc.)
- New tests verify alias expansion: `TCon(Coord) ~ TRecord{x,y}` succeeds
- stapledons_voyage IsoTile constructor works with Coord alias

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
