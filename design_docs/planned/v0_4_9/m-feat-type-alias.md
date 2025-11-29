# M-FEAT-TYPE-ALIAS: Support Type Aliases

**Status**: Planned
**Priority**: P2
**Estimated LOC**: ~200
**Target Version**: v0.4.9+

## Problem Statement

Type aliases (`type Foo = Bar`) are parsed but not processed by the type checker, causing runtime errors:

```
Error: failed to process type declaration Subst: unknown type definition: *ast.TypeAlias
Error: failed to process type declaration Env: unknown type definition: *ast.TypeAlias
```

### Impact

- 6 compile errors in v0.4.8 eval baseline
- Affects `type_unify`, `symbolic_diff`, and other benchmarks
- Limits expressiveness - can't create readable type synonyms

## Current State

The parser recognizes type alias syntax:
```ailang
type Subst = [(string, Expr)]  -- List of (variable, expression) pairs
type Env = {bindings: Subst}
```

But the type checker rejects `*ast.TypeAlias` nodes with "unknown type definition".

## Proposed Solution

### Phase 1: Simple Type Aliases (v0.4.9)

Support non-recursive type aliases as syntactic sugar:

```ailang
type Point = {x: int, y: int}
type Matrix = [[float]]

-- Expands during type checking to:
-- Point -> {x: int, y: int}
-- Matrix -> [[float]]
```

Implementation:
1. In type checker, when encountering `TypeAlias`:
   - Register alias name in environment
   - When referenced, expand to underlying type
2. No runtime representation needed - purely compile-time

### Phase 2: Recursive Type Aliases (v0.5.0+)

Support recursive types like:
```ailang
type Expr =
  | Const(int)
  | Add(Expr, Expr)
  | Mul(Expr, Expr)
```

This requires more work (equi-recursive or iso-recursive types).

## Implementation Plan

### Phase 1 Tasks

1. **Add TypeAlias case to type checker** (`internal/types/typechecker.go`)
   - Handle `*ast.TypeAlias` in type declaration processing
   - Store alias → expansion mapping
   - Expand on reference

2. **Add alias expansion** to type operations
   - When unifying, expand aliases first
   - When printing types, optionally show alias names

3. **Add tests** for basic aliases

## Acceptance Criteria (Phase 1)

- [ ] `type Foo = Bar` parses and type-checks
- [ ] Alias types can be used in function signatures
- [ ] Alias types expand correctly during unification
- [ ] Error messages show alias names where helpful
- [ ] 6 failing benchmarks no longer fail due to this issue

## Files to Modify

- `internal/types/typechecker.go` (~80 LOC)
- `internal/types/unify.go` (~40 LOC expansion)
- `tests/types/alias_test.go` (~80 LOC new)

## References

- v0.4.8 eval analysis showing 6 type alias failures
- `internal/ast/ast.go` - TypeAlias node definition
- Haskell type synonyms for inspiration
