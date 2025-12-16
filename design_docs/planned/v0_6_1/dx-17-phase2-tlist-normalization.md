# DX-17 Phase 2: Normalize TList to TApp at Parse Time

**Status:** Planned
**Priority:** Medium (cleanup, not blocking)
**Estimated LOC:** 100-150
**Depends On:** DX-17 Phase 1 (v0.5.11, completed)
**Target:** v0.5.12

## Summary

Phase 1 (DX-17, v0.5.11) fixed the TList/TApp unification bug with a shared `AsList` helper. This works but leaves two internal representations for list types. Phase 2 eliminates `TList` entirely by normalizing `[T]` syntax to `TApp("list", T)` during parsing.

## Motivation

### Current State (After Phase 1)

The codebase has:
- `TList` struct in `internal/types/types.go`
- `TApp("list", ...)` created by `internal/types/builder.go`
- `AsList` helper that recognizes both forms
- Multiple `case *TList:` branches throughout the type system

This works but has drawbacks:
1. **Cognitive overhead**: Developers must remember both representations exist
2. **Code duplication**: Many switch statements handle `TList` separately from other types
3. **Inconsistency**: Lists are special-cased while `Option[T]`, `Result[T, E]` use `TApp`
4. **Future risk**: New type system features might forget to handle `TList`

### Target State (After Phase 2)

- `[T]` syntax in source code normalizes to `TApp("list", T)` during parsing
- `TList` struct is deprecated, then removed
- `AsList` helper becomes optional (can directly check for `TApp("list", ...)`)
- All container types use uniform `TApp` representation

## Design Decision

**Canonical internal representation for lists is `TApp(TCon("list"), [T])`.**

`TList` is syntactic sugar only. The `[T]` syntax in source code is convenient shorthand, but internally all list types should normalize to `TApp("list", T)` before type checking.

**Architectural Rule:** Syntactic sugar must normalize to canonical type constructors before unification. This applies to lists now and should be enforced for any future sugar (e.g., tuple syntax, record shorthand).

## Implementation Plan

### M1: Update Parser to Emit TApp (~30 LOC)
**File:** `internal/parser/parser_type.go`

**Change:**
```go
// Current (creates TList)
case lexer.LBRACKET:
    // ... parses element type ...
    return &types.TList{Elem: elemType}

// New (creates TApp)
case lexer.LBRACKET:
    // ... parses element type ...
    return &types.TApp{
        Constructor: &types.TCon{Name: "list"},
        Args:        []types.Type{elemType},
    }
```

**Acceptance Criteria:**
- [ ] `[int]` parses to `TApp("list", int)` not `TList{Element: int}`
- [ ] `[[string]]` parses to nested `TApp`
- [ ] Existing parser tests pass
- [ ] Add new parser tests for list type normalization

### M2: Audit and Update TList Switch Statements (~40 LOC)
**Files:** Multiple in `internal/types/`

**Task:** Find all `case *TList:` statements and either:
1. Remove them (if `TApp` case handles it)
2. Merge with `TApp` case
3. Keep only for backward compatibility (temporary)

**Locations to update (from Phase 1 analysis):**
```
internal/types/typechecker_defaulting.go:329
internal/types/unification_equality.go:59
internal/types/safe_string.go:86
internal/types/inference.go:694
internal/types/unification_occurs.go:63
internal/types/unification_substitution.go:57
internal/types/normalize.go:55, 282
internal/types/unification_core.go:206, 275
internal/types/type_head.go:52
internal/types/types_v2.go:391
internal/types/typechecker_substitution.go:147
```

**Strategy:**
- First, ensure `TApp("list", ...)` branch handles lists
- Then, add `case *TList:` that delegates to `TApp` handling (for safety)
- Run tests after each file
- Once all tests pass, consider removing `TList` cases

### M3: Update Type Printer (~15 LOC)
**File:** `internal/types/types.go` or `internal/types/safe_string.go`

**Task:** Ensure `TApp("list", T)` prints as `list[T]` (not `list[T]` vs `[T]`).

Currently:
- `TList{Element: int}` → `"[int]"`
- `TApp("list", int)` → `"list[int]"` or `"list(int)"`

After Phase 2:
- `TApp("list", int)` → `"list[T]"` (consistent)

**Consideration:** Error messages and debug output should show `list[T]` consistently.

### M4: Deprecate TList (~20 LOC)
**File:** `internal/types/types.go`

**Add deprecation comment:**
```go
// TList represents a list type.
//
// Deprecated: Use TApp("list", T) instead. The [T] syntax in source code
// normalizes to TApp("list", T) during parsing. TList will be removed in v0.6.0.
type TList struct {
    Element Type
}
```

**Keep `AsList` helper:** It's still useful for code that needs to check "is this a list?" without knowing the representation.

### M5: Add Migration Warning (Optional, ~30 LOC)

If external code uses `TList` directly (unlikely but possible), add a compile-time or runtime warning.

**Option A:** Add `//go:deprecated` comment (Go 1.23+)
**Option B:** Add runtime warning in `TList.String()` when `DEBUG_DEPRECATION=1`

### M6: Remove TList (Future, v0.6.0)

After one release cycle with deprecation:
1. Remove `TList` struct
2. Remove all `case *TList:` branches
3. Simplify `AsList` to only check `TApp`
4. Update documentation

## Verification Checklist

- [ ] Parser emits `TApp("list", T)` for `[T]` syntax
- [ ] All existing tests pass
- [ ] `make test` passes
- [ ] Type printing is consistent (`list[T]`)
- [ ] `AsList` helper still works
- [ ] No regressions in stdlib compilation
- [ ] Debug output shows consistent list representation

## Risk Assessment

**Low Risk:**
- Phase 1 already handles both representations
- Changes are incremental
- Each milestone is independently testable

**Potential Issues:**
- External code depending on `TList` (mitigated by deprecation period)
- Subtle differences in type equality (mitigated by comprehensive tests)

## Test Cases

```go
// Test: Parser normalization
func TestParserListTypeNormalization(t *testing.T) {
    // Parse "[int]" and verify it's TApp not TList
    typ := parseType("[int]")
    app, ok := typ.(*types.TApp)
    require.True(t, ok, "expected TApp, got %T", typ)

    con, ok := app.Constructor.(*types.TCon)
    require.True(t, ok)
    require.Equal(t, "list", con.Name)
    require.Len(t, app.Args, 1)
}

// Test: Nested list normalization
func TestParserNestedListNormalization(t *testing.T) {
    typ := parseType("[[string]]")
    // Should be TApp("list", TApp("list", string))
}

// Test: Type printing consistency
func TestListTypePrinting(t *testing.T) {
    T := types.NewBuilder()
    listInt := T.List(T.Int())
    require.Equal(t, "list[int]", listInt.String())
}
```

## Related Work

- **DX-17 Phase 1 (v0.5.11):** Added `AsList` helper, fixed unification
- **Similar patterns elsewhere:**
  - `[a: T, b: U]` record syntax → could normalize to `TApp("Record", ...)`
  - `(A, B)` tuple syntax → could normalize to `TApp("Tuple", ...)`
  - Effect rows already use uniform `Row` representation

## References

- DX-17 Phase 1: [design_docs/implemented/v0_5_11/dx-17-tlist-tapp-unification.md](../implemented/v0_5_11/dx-17-tlist-tapp-unification.md)
- Type Builder: `internal/types/builder.go`
- Parser: `internal/parser/parser_type.go`
- AsList helper: `internal/types/helpers.go`
