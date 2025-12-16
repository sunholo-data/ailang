# DX-17: TList/TApp Type Unification Bug (Phase 1)

**Status:** ✅ IMPLEMENTED
**Version:** v0.5.11
**Priority:** High (blocked stdlib development)
**Actual LOC:** ~150 (estimated 50-100)
**Implementation Date:** 2025-12-16

## Summary

Fixed the TList/TApp type unification bug using Option B (shared helper + multi-site fixes). The parser creates `TList` for `[T]` syntax while the type builder creates `TApp("list", T)` for builtins. These two representations now unify correctly.

**Phase 2** (normalize `TList` to `TApp` at parse time) is planned for v0.5.12 - see [dx-17-phase2-tlist-normalization.md](../../planned/v0_5_12/dx-17-phase2-tlist-normalization.md).

## Decision

**Canonical internal representation for lists is `TApp(TCon("list"), [T])`.**

`TList` is syntactic sugar only. The `[T]` syntax in source code is convenient shorthand, but internally all list types should normalize to `TApp("list", T)` before type checking.

**Architectural Rule:** Syntactic sugar must normalize to canonical type constructors before unification. This applies to lists now and should be enforced for any future sugar (e.g., tuple syntax, record shorthand).

## Problem Statement

AILANG had two internal representations for list types that failed to unify:

1. **TList** - Created by the parser when it sees `[T]` syntax
2. **TApp("list", T)** - Created by the type builder (`T.List(T.X())`) used for builtins

This caused type errors when:
- Pattern matching on lists returned from builtins
- Passing builtin results to stdlib functions
- Using array conversion builtins on builtin results

### Root Cause

**Three issues found during implementation:**

1. **Case sensitivity in builder** (PRIMARY): The type builder in `internal/types/builder.go` was creating `TApp("List", ...)` with **uppercase** "List", while the canonical AILANG syntax is lowercase `list[T]`. This mismatch propagated through the entire type system.

2. **Inconsistent case checks**: Multiple locations checked for "List" (uppercase) instead of "list" (lowercase):
   - `internal/types/type_head.go` - Head() function
   - `internal/pipeline/op_lowering.go` - getTypeSuffixFromType()
   - `internal/types/unification_types.go` - unifyLists() and unifyTypeApps()

3. **Pattern type checker gap**: `internal/types/typechecker_patterns.go` only checked for `*TList`, not `TApp("list", ...)`.

## Implementation (Phase 1)

Implemented **Option B: Shared Helper + Multi-Site Fixes**.

### Changes Made

| File | Change | LOC |
|------|--------|-----|
| `internal/types/builder.go` | **ROOT FIX**: Changed `"List"` to `"list"` in List() constructor | +2/-1 |
| `internal/types/type_head.go` | Changed `"List"` to `"list"` in Head() function | +3/-2 |
| `internal/pipeline/op_lowering.go` | Changed `"List"` to `"list"` in getTypeSuffixFromType() | +4/-3 |
| `internal/types/helpers.go` | Added `AsList` helper function | +29 |
| `internal/types/helpers_test.go` | Added `TestAsList` with 8 test cases | +74 |
| `internal/types/unification_types.go` | Refactored `unifyLists` and `unifyTypeApps` to use `AsList` | +6/-18 |
| `internal/types/unification_substitution_test.go` | Added `TestTListTAppUnification` with 6 test cases | +61 |
| `internal/types/typechecker_patterns.go` | Updated list pattern checker to use `AsList` | +3/-4 |
| `internal/builtins/array.go` | Changed `T.App("List", a)` to `T.List(a)` for consistency | +2/-2 |
| `internal/builtins/list.go` | Changed `T.App("List", a)` to `T.List(a)` for consistency | +2/-2 |

### M0: Builder Root Fix (THE KEY FIX)

The root cause was in `internal/types/builder.go` - the `List()` constructor was creating uppercase "List":

```go
// BEFORE (broken)
func (b *Builder) List(elem Type) Type {
    return &TApp{
        Constructor: &TCon{Name: "List"},  // ❌ Uppercase!
        Args:        []Type{elem},
    }
}

// AFTER (fixed)
func (b *Builder) List(elem Type) Type {
    return &TApp{
        Constructor: &TCon{Name: "list"},  // ✅ Lowercase matches AILANG syntax
        Args:        []Type{elem},
    }
}
```

### M1: Type Head Fix

Updated `internal/types/type_head.go` to check for lowercase "list":

```go
case *TApp:
    // DX-17: Canonical form is lowercase "list" (from T.List())
    if con, ok := typ.Constructor.(*TCon); ok {
        if con.Name == "list" {  // Changed from "List"
            return HeadList
        }
    }
```

### M2: Op Lowering Fix

Updated `internal/pipeline/op_lowering.go` in `getTypeSuffixFromType()`:

```go
// DX-17: canonical form is lowercase "list"
case "list":  // Changed from "List"
    return "List"  // Return value stays uppercase (Go convention)

// Also in TApp check:
if con.Name == "list" {  // Changed from "List"
    return "List"
}
```

### M3: Builtins Consistency

Updated builtins to use `T.List()` instead of `T.App("List", ...)`:

**internal/builtins/array.go:**
```go
// BEFORE
listA := T.App("List", a)

// AFTER (DX-17)
listA := T.List(a)  // Uses T.List() for lowercase "list" constructor
```

**internal/builtins/list.go:**
```go
// BEFORE
listA := T.App("List", a)

// AFTER (DX-17)
listA := T.List(a)  // Uses T.List() for lowercase "list" constructor
```

### M4: AsList Helper Function

Added to `internal/types/helpers.go`:

```go
// AsList checks if a type is a list type (either TList or TApp("list", ...)).
// Returns the element type and true if it's a list, nil and false otherwise.
//
// This helper addresses DX-17: TList/TApp unification bug. The parser creates
// TList for [T] syntax, while the type builder creates TApp("list", T) for builtins.
// This helper recognizes both representations.
//
// Note: Case-sensitive - only matches lowercase "list" (not "List").
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

### M5: Unifier Fixes

Updated `internal/types/unification_types.go`:

```go
// unifyLists - now uses AsList helper
func (u *Unifier) unifyLists(t1 *TList, t2 Type, sub Substitution) (Substitution, error) {
    // Check if t2 is also a list (TList or TApp("list", ...))
    if elem, ok := AsList(t2); ok {
        return u.Unify(t1.Element, elem, sub)
    }
    // ... rest of function
}

// unifyTypeApps - now uses AsList helper
if t2List, ok := t2.(*TList); ok {
    if elem, ok := AsList(t1); ok {
        return u.Unify(elem, t2List.Element, sub)
    }
}
```

### M6: Pattern Type Checker Fix

Updated `internal/types/typechecker_patterns.go`:

```go
case *core.ListPattern:
    var elemType Type
    // DX-17: Try to extract list type (handles both TList and TApp("list",...))
    if elem, ok := AsList(scrutType); ok {
        elemType = elem
    } else {
        // Create fresh type variable and add constraint
        elemType = ctx.freshTypeVar()
        ctx.addConstraint(TypeEq{
            Left:  scrutType,
            Right: &TList{Element: elemType},
            Path:  []string{"list pattern"},
        })
    }
```

## Test Results

### New Tests Added

1. **TestAsList** (8 test cases):
   - TList returns element type ✅
   - TApp(list, int) returns element type ✅
   - TApp(List, int) uppercase returns false ✅ (case sensitive!)
   - TCon(string) returns false ✅
   - TApp(Option, int) returns false ✅
   - TVar2 returns false ✅
   - TList with polymorphic element ✅
   - TApp(list, TVar2) with polymorphic element ✅

2. **TestTListTAppUnification** (6 test cases):
   - TList{int} unifies with TApp(list, int) ✅
   - TApp(list, int) unifies with TList{int} ✅
   - TList{string} unifies with TApp(list, string) ✅
   - TList{int} does NOT unify with TApp(list, string) ✅
   - Nested list unification ✅
   - Polymorphic element unification ✅

### Verification

- ✅ All existing tests pass (`make test`)
- ✅ Linting passes (`make lint`)
- ✅ List pattern matching works with both TList and TApp
- ✅ Test file `/tmp/test_dx17.ail` compiles and runs successfully

## Acceptance Criteria Status

1. ✅ `[T]` syntax and `list[T]` syntax produce equivalent types
2. ✅ Pattern matching works on lists returned from builtins
3. ✅ `_array_from_list` accepts results from `_sharedindex_find_simhash`
4. ✅ All existing tests pass
5. ✅ No regressions in stdlib compilation
6. ⏳ Type printing consistency (deferred to Phase 2)
7. ✅ Type equality treats both forms as identical (via unification)
8. ✅ DX-16 M9 stdlib primitives compile without workarounds

## What's Left (Phase 2 - v0.5.12)

Phase 2 will eliminate `TList` entirely by normalizing `[T]` to `TApp("list", T)` at parse time:

1. Update parser to emit `TApp` instead of `TList`
2. Audit and remove `case *TList:` statements
3. Update type printer for consistency
4. Deprecate then remove `TList` struct

See: [design_docs/planned/v0_5_12/dx-17-phase2-tlist-normalization.md](../../planned/v0_5_12/dx-17-phase2-tlist-normalization.md)

## Lessons Learned

1. **Case sensitivity matters**: The bug was primarily a case mismatch - `builder.go` used uppercase "List" while AILANG syntax is lowercase `list[T]`. This propagated to multiple check sites.
2. **Fix at the source**: Initial attempt added workarounds in unification, but the root fix was changing the builder to emit lowercase "list". Always fix at the source, not downstream.
3. **Helper functions prevent fragmentation**: Adding `AsList` ensures all code paths handle both representations consistently
4. **Test both directions**: Unification must work symmetrically (TList→TApp AND TApp→TList)
5. **Audit all consumers**: When fixing a type representation, search for ALL places that check for that type name (grep for the string)

## Related: Dual Representations Elsewhere

This pattern (syntactic sugar creating different internal types) may exist elsewhere:

| Surface Syntax | Internal Representation | Status |
|----------------|------------------------|--------|
| `[T]` | `TList` vs `TApp("list", T)` | ✅ Fixed (Phase 1) |
| `{a: T, b: U}` | `TRecord` vs `TApp("Record", ...)` | Not yet encountered |
| `(A, B)` | `TTuple` vs `TApp("Tuple", ...)` | Not yet encountered |
| `! {IO, Net}` | Effect rows | Different system, likely OK |

**Prevention:** Apply the architectural rule:
> Syntactic sugar must normalize to canonical type constructors before unification.

## References

- Sprint Plan: [design_docs/planned/v0_5_11/dx-17-sprint-plan.md](../planned/v0_5_11/dx-17-sprint-plan.md)
- Sprint JSON: `.ailang/state/sprints/sprint_DX-17.json`
- Phase 2 Design: [design_docs/planned/v0_5_12/dx-17-phase2-tlist-normalization.md](../../planned/v0_5_12/dx-17-phase2-tlist-normalization.md)
- Type Builder: `internal/types/builder.go`
- AsList Helper: `internal/types/helpers.go`
- Unifier: `internal/types/unification_types.go`
