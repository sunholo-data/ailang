# DX-17: TList/TApp Type Unification - Sprint Plan

**Sprint ID:** DX-17
**Status:** Planned → In Progress
**Estimated Duration:** 2-3 hours
**Estimated LOC:** 80-100
**Blocking:** DX-16 M9 stdlib primitives

## Sprint Summary

Fix the TList/TApp unification bug that prevents pattern matching on lists returned from builtins. The bug has two root causes:

1. **Case sensitivity mismatch**: Builder creates `TApp("list", ...)` (lowercase) but unifier checks for `"List"` (uppercase)
2. **Pattern type checker gap**: Only checks for `*TList`, not `TApp("list", ...)`

**Solution:** Implement Option B from design doc (shared helper + multi-site fixes) with the case sensitivity fix.

## Current Status Analysis

### Completed (Pre-Sprint)
- ✅ Design doc created with comprehensive analysis
- ✅ Root cause identified (case sensitivity + pattern checker gap)
- ✅ Code locations mapped (13 `case *TList:` in types package)
- ✅ Acceptance criteria defined

### Velocity Reference
- Recent M-DX11-PHASE2: ~220 LOC/day
- Recent M-DX15 semantic caching: ~150 LOC/day
- This sprint: ~80-100 LOC total (single focused fix)

## Milestones

### M1: Add AsList Helper Function (~20 LOC)
**File:** `internal/types/helpers.go`

**Task:** Add `AsList` helper that recognizes both `TList` and `TApp("list", ...)`.

```go
// AsList checks if a type is a list (either TList or TApp("list", ...))
// Returns the element type and true if it's a list, nil and false otherwise.
func AsList(t Type) (elem Type, ok bool) {
    switch tt := t.(type) {
    case *TList:
        return tt.Element, true
    case *TApp:
        h, args := decomposeApp(tt)
        if con, ok := h.(*TCon); ok && con.Name == "list" && len(args) == 1 {
            return args[0], true
        }
        return nil, false
    default:
        return nil, false
    }
}
```

**Acceptance Criteria:**
- [ ] `AsList(*TList{Element: int})` returns `(int, true)`
- [ ] `AsList(TApp("list", [int]))` returns `(int, true)`
- [ ] `AsList(TApp("List", [int]))` returns `(nil, false)` - case sensitive!
- [ ] `AsList(*TCon{Name: "string"})` returns `(nil, false)`
- [ ] Unit tests cover all cases

**Dependencies:** None

---

### M2: Fix Case Sensitivity in Unifier (~10 LOC)
**File:** `internal/types/unification_types.go`

**Task:** Change `"List"` to `"list"` in two locations where unifier checks for list TApp.

**Changes:**
1. Line 107: `headCon.Name == "List"` → `headCon.Name == "list"`
2. Line 199: `headCon.Name == "List"` → `headCon.Name == "list"`

Also use the new `AsList` helper for cleaner code:

```go
// In unifyLists (line 104-111)
if t2App, ok := t2.(*TApp); ok {
    if elem, ok := AsList(t2App); ok {
        return u.Unify(t1.Element, elem, sub)
    }
}

// In unifyTypeApps (line 196-203)
if t2List, ok := t2.(*TList); ok {
    if elem, ok := AsList(t1); ok {
        return u.Unify(elem, t2List.Element, sub)
    }
}
```

**Acceptance Criteria:**
- [ ] `TList{Element: int}` unifies with `TApp("list", [int])`
- [ ] `TApp("list", [int])` unifies with `TList{Element: int}`
- [ ] Symmetric: both directions work
- [ ] Existing unification tests pass

**Dependencies:** M1 (AsList helper)

---

### M3: Update Pattern Type Checker (~15 LOC)
**File:** `internal/types/typechecker_patterns.go`

**Task:** Update `checkPattern` for `*core.ListPattern` to handle both `TList` and `TApp("list", ...)`.

**Current code (line 274):**
```go
if listTy, ok := scrutType.(*TList); ok {
    elemType = listTy.Element
}
```

**Updated code:**
```go
if elem, ok := AsList(scrutType); ok {
    elemType = elem
}
```

**Also update the constraint generation (line 281):**
```go
// Keep using TList for consistency in constraint generation
// The unifier will handle TList ↔ TApp("list",...) equivalence
ctx.addConstraint(TypeEq{
    Left:  scrutType,
    Right: &TList{Element: elemType},
    Path:  []string{"list pattern"},
})
```

**Acceptance Criteria:**
- [ ] Pattern `[]` matches `TApp("list", [int])` (empty list)
- [ ] Pattern `[x, ...rest]` matches `TApp("list", [int])` (cons pattern)
- [ ] Pattern bindings have correct types
- [ ] Error messages are clear for non-list types

**Dependencies:** M1 (AsList helper), M2 (unifier fix)

---

### M4: Add Comprehensive Tests (~40 LOC)
**Files:**
- `internal/types/helpers_test.go` (AsList tests)
- `internal/types/unification_types_test.go` (TList/TApp unification tests)

**Test Cases:**

```go
// Test 1: AsList helper
func TestAsList(t *testing.T) {
    T := NewBuilder()

    // TList case
    tlist := &TList{Element: T.Int()}
    elem, ok := AsList(tlist)
    require.True(t, ok)
    require.Equal(t, T.Int(), elem)

    // TApp("list", int) case
    tapp := T.List(T.Int())
    elem, ok = AsList(tapp)
    require.True(t, ok)
    // elem is T.Int()

    // Non-list cases
    _, ok = AsList(T.String())
    require.False(t, ok)

    _, ok = AsList(T.App("Option", T.Int()))
    require.False(t, ok)
}

// Test 2: TList/TApp unification
func TestUnifyTListTApp(t *testing.T) {
    T := NewBuilder()
    u := NewUnifier()

    // TList{int} ~ TApp("list", int)
    t1 := &TList{Element: T.Int()}
    t2 := T.List(T.Int())
    sub, err := u.Unify(t1, t2, make(Substitution))
    require.NoError(t, err)

    // Symmetric
    sub, err = u.Unify(t2, t1, make(Substitution))
    require.NoError(t, err)
}
```

**Acceptance Criteria:**
- [ ] All new tests pass
- [ ] No regressions in existing tests (`make test`)
- [ ] Coverage for edge cases (nested lists, polymorphic lists)

**Dependencies:** M1, M2, M3

---

### M5: Verify DX-16 M9 Compilation (~5 min)
**Files:** `std/sem.ail`, examples/

**Task:** Verify that `std/sem.ail` and related code now compiles without the TList/TApp type errors.

**Verification Steps:**
1. Run `ailang check std/sem.ail`
2. Run `make test`
3. Test pattern matching on builtin results:
   ```ailang
   let results = _sharedindex_find_simhash(ns, hash, 10, 0, true)
   in match results {
     [] => None,
     [best, ..._] => Some(best)
   }
   ```

**Acceptance Criteria:**
- [ ] `std/sem.ail` compiles without type errors
- [ ] Pattern matching on builtin results works
- [ ] `_array_from_list` accepts builtin results
- [ ] All existing tests pass

**Dependencies:** M1-M4

---

## Success Metrics

- [ ] All 5 milestones completed
- [ ] `make test` passes (no regressions)
- [ ] DX-16 M9 unblocked (std/sem.ail compiles)
- [ ] ~80-100 LOC added (within estimate)

## Risk Assessment

**Low Risk:**
- Small, focused change
- Well-understood root cause
- Comprehensive test coverage planned

**Mitigation:**
- Run `make test` after each milestone
- Keep changes minimal and targeted

## Files to Modify

1. `internal/types/helpers.go` - Add AsList helper (~20 LOC)
2. `internal/types/unification_types.go` - Fix case sensitivity (~10 LOC)
3. `internal/types/typechecker_patterns.go` - Use AsList (~15 LOC)
4. `internal/types/helpers_test.go` - Add AsList tests (~20 LOC)
5. `internal/types/unification_types_test.go` - Add unification tests (~20 LOC)

**Total: ~85 LOC**

## References

- Design Doc: [dx-17-tlist-tapp-unification.md](dx-17-tlist-tapp-unification.md)
- DX-16 M9 blocked by this issue
- Type Builder: `internal/types/builder.go:90-95` (creates lowercase "list")
- Unifier: `internal/types/unification_types.go:107,199` (checks uppercase "List")
