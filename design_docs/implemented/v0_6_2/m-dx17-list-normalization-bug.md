# M-DX17-LIST-NORMALIZATION-BUG: List[T] Not Normalized to Lowercase

**Status:** ✅ Implemented
**Target:** v0.6.2
**Priority:** P1 (High - breaks examples)
**Estimated:** 30 minutes
**Actual:** 15 minutes
**Dependencies:** DX-17 Phase 2
**Created:** 2025-12-25
**Implemented:** 2025-12-26

## Problem Statement

DX-17 Phase 2 normalized `[T]` syntax to `TypeApp("list", T)` at parse time, but failed to normalize explicit `List[T]` syntax to the same form.

### Reproduction

```ailang
-- This works:
pure func head1(xs: [int]) -> int = match xs { x :: _ => x, [] => 0 }

-- This FAILS:
pure func head2(xs: List[int]) -> int = match xs { x :: _ => x, [] => 0 }
```

**Error:**
```
Error: type unification failed at [list pattern]: cannot unify type application with *types.TList
```

### Root Cause

In `internal/parser/parser_type.go`:

- Line 109: `[int]` creates `TypeApp{Constructor: "list"}` (lowercase)
- Line 57: `List[int]` creates `TypeApp{Constructor: "List"}` (capital L)

The `AsList()` helper in `internal/types/helpers.go` only checks for lowercase:
```go
if con, ok := h.(*TCon); ok && con.Name == "list" && len(args) == 1 {
```

## Solution

Normalize `List[T]` to lowercase `"list"` at parse time, matching the `[T]` normalization.

### Implementation

**File:** `internal/parser/parser_type.go`

Change line 51-58 from:
```go
switch name {
case "Array":
    typ = &ast.ArrayType{Element: elemType, Pos: startPos}
default:
    typ = &ast.TypeApp{Constructor: name, Args: typeArgs, Pos: startPos}
}
```

To:
```go
switch name {
case "Array":
    typ = &ast.ArrayType{Element: elemType, Pos: startPos}
case "List":
    // DX-17 Phase 2: Normalize List[T] to lowercase "list" for consistency
    typ = &ast.TypeApp{Constructor: "list", Args: typeArgs, Pos: startPos}
default:
    typ = &ast.TypeApp{Constructor: name, Args: typeArgs, Pos: startPos}
}
```

## Test Cases

1. `List[int]` in function parameter types
2. `List[T]` with type variables
3. Pattern matching on `List[int]` parameters
4. Nested types like `Option[List[int]]`

## Success Criteria

- [x] `examples/runnable/pattern_sugar.ail` passes
- [x] `List[int]` and `[int]` are fully interchangeable
- [x] All existing tests pass (parser, types)
