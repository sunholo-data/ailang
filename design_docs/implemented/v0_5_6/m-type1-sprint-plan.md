# M-TYPE1 Sprint Plan: Array/TArray Type Unification Bug Fix

**Sprint ID**: M-TYPE1
**Duration**: 1 day (~4 hours)
**Target Version**: v0.5.6
**Risk Level**: Low (localized parser fix)

## Sprint Summary

**Goal**: Fix the parser to preserve type arguments when parsing `Array[T]` and `List[T]` in type annotations, enabling array literals to unify with ADT constructor parameters.

**Root Cause**: The parser in `internal/parser/parser_type.go:27-50` discards type arguments when parsing type applications like `Array[Direction]`. It returns `SimpleType{Name: "Array"}` instead of `ArrayType{Element: Direction}`.

## Current Status Analysis

### Bug Reproduction
```ailang
type Direction = North | South | East | West
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander

let patrol = PatternPatrol(#[North, East, South, West])
-- Error: cannot unify type constructor Array with *types.TArray
```

### Bug Flow
1. **Parser** (`parser_type.go:27-50`): `Array[Direction]` → `SimpleType{Name: "Array"}` ❌
2. **Type conversion** (`iface/builder.go:44`): `SimpleType{Name: "Array"}` → `TCon{Name: "Array"}`
3. **Constructor scheme**: FieldTypes includes `TCon{Name: "Array"}`
4. **Unification**: `TCon("Array")` vs `TArray{Element: Direction}` → FAIL

### Existing Infrastructure
- `ast.ArrayType` exists for array types ✅
- `ast.ListType` exists for list types ✅
- `astTypeToInternalType` handles both correctly ✅
- Unification handles `TArray ~ TApp("Array", T)` bidirectionally ✅ (v0.5.5)

## Milestones

### M1: Fix Parser Type Application (~2 hours)

**Description**: Modify `parseType()` to create `ArrayType` or `ListType` when parsing `Array[T]` or `List[T]`.

**File**: `internal/parser/parser_type.go`

**Current code** (lines 27-50):
```go
if p.peekTokenIs(lexer.LBRACKET) {
    // ... parses type args but discards them
    typ = &ast.SimpleType{Name: name}  // BUG: loses type args
}
```

**Fixed code**:
```go
if p.peekTokenIs(lexer.LBRACKET) {
    p.nextToken() // consume IDENT
    p.nextToken() // consume LBRACKET

    elemType := p.parseType() // Parse element type

    for p.peekTokenIs(lexer.COMMA) {
        p.nextToken() // move to COMMA
        p.nextToken() // move past COMMA
        _ = p.parseType() // Skip additional args (not supported yet)
    }

    if !p.expectPeek(lexer.RBRACKET) {
        return nil
    }

    // Special-case Array and List to preserve element types
    switch name {
    case "Array":
        typ = &ast.ArrayType{Element: elemType, Pos: startPos}
    case "List":
        typ = &ast.ListType{Element: elemType, Pos: startPos}
    default:
        // Generic type application - return SimpleType for now
        // TODO: Add ast.TypeApp for proper generic support
        typ = &ast.SimpleType{Name: name, Pos: startPos}
    }
    goto checkArrow
}
```

**Acceptance Criteria**:
- [ ] `Array[int]` parses to `ast.ArrayType{Element: SimpleType{Name: "int"}}`
- [ ] `List[string]` parses to `ast.ListType{Element: SimpleType{Name: "string"}}`
- [ ] `Option[int]` still parses to `ast.SimpleType{Name: "Option"}` (unchanged)

**Estimated LOC**: ~15 lines changed

### M2: Add Parser Tests (~1 hour)

**Description**: Add unit tests for type application parsing.

**File**: `internal/parser/parser_type_test.go` (new or existing)

**Test Cases**:
1. `Array[int]` → `ArrayType{Element: SimpleType{Name: "int"}}`
2. `Array[Direction]` → `ArrayType{Element: SimpleType{Name: "Direction"}}`
3. `List[string]` → `ListType{Element: SimpleType{Name: "string"}}`
4. `Option[int]` → `SimpleType{Name: "Option"}` (generic, unchanged)
5. ADT with array field: `type T = Foo(Array[int])` parses correctly

**Estimated LOC**: ~50 lines

### M3: Integration Test & Verification (~1 hour)

**Description**: Verify the end-to-end fix works.

**Tasks**:
- [ ] Create `examples/runnable/array_adt.ail` test file
- [ ] Run `ailang run` on the game benchmark with arrays
- [ ] Verify unification succeeds
- [ ] Run full test suite: `make test`
- [ ] Update CHANGELOG.md

**Example file** (`examples/runnable/array_adt.ail`):
```ailang
module examples/runnable/array_adt

type Direction = North | South | East | West

type AIBehavior =
  | PatternPatrol(Array[Direction])
  | RandomWander

let main: () !{IO} -> () = \.
  let patrol = PatternPatrol(#[North, East, South, West])
  print("Array in ADT works!")
```

**Estimated LOC**: ~30 lines (example + changelog)

## Success Criteria

- [ ] `PatternPatrol(#[North, East])` compiles without error
- [ ] Parser tests pass for Array and List type applications
- [ ] All existing tests pass (`make test`)
- [ ] Example file `array_adt.ail` runs successfully
- [ ] CHANGELOG.md updated

## Velocity Estimate

Based on recent commits:
- v0.5.5 fix for `TApp(Array, T)` unification: ~2 hours
- Similar parser fixes (M-DX26): ~3 hours

**Estimated total**: 4 hours

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/parser/parser_type.go` | Fix type application parsing | ~15 |
| `internal/parser/parser_type_test.go` | Add parser tests | ~50 |
| `examples/runnable/array_adt.ail` | Integration test example | ~15 |
| `CHANGELOG.md` | Document fix | ~10 |

**Total**: ~90 LOC

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking other type applications | High | Keep `Option[T]` etc. unchanged (return SimpleType) |
| Parser test coverage gaps | Med | Add comprehensive test cases |
| Edge cases (nested arrays) | Low | Test `Array[Array[int]]` explicitly |

---

**Sprint created**: 2025-12-04
**Author**: Claude (sprint-planner)
