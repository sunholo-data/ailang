# M-DX19 Sprint Plan: Auto-Derive Eq for ADT Types

**Sprint ID**: DX19
**Duration**: 4-6 hours (single session)
**Risk Level**: Medium
**Dependencies**: None

## Summary

Implement `deriving (Eq)` syntax for ADT types, enabling automatic equality comparison via `==` operator. Monomorphic types only in v0.6.2.

## Current Status

- **Design doc**: Complete (`m-dx19-auto-derive-eq.md`)
- **Implementation**: Not started
- **Lexer**: No DERIVING token
- **TypeDecl AST**: No Deriving field

## Velocity Reference

Recent completions:
- M-RECORD-PATTERNS: ~350 LOC in 1 day
- DX-17 Phase 2: ~100 LOC in 1 day
- List[T] fix: ~10 LOC in 30 mins

Target velocity: ~150 LOC/hour for parser/type work

## Milestones

### M1: Lexer & Parser Support (~1.5 hours, ~80 LOC)

**Tasks:**
1. Add `DERIVING` token to `internal/lexer/token.go` (~5 LOC)
2. Add keyword recognition in `internal/lexer/lexer.go` (~5 LOC)
3. Add `DeriveKind` enum and `Deriving` field to TypeDecl in `internal/ast/ast_decl.go` (~20 LOC)
4. Parse `deriving (Eq)` after type definition in `internal/parser/parser_type.go` (~40 LOC)
5. Add parser tests (~10 LOC)

**Acceptance Criteria:**
- [ ] `type Color = Red | Green deriving (Eq)` parses without error
- [ ] TypeDecl.Deriving contains DeriveEq
- [ ] Parser test passes

**Files:**
- `internal/lexer/token.go`
- `internal/lexer/lexer.go`
- `internal/ast/ast_decl.go`
- `internal/parser/parser_type.go`
- `internal/parser/parser_type_test.go`

### M2: Type Checker Integration (~1.5 hours, ~100 LOC)

**Tasks:**
1. Track derived types in type environment (~20 LOC)
2. Validate type is monomorphic (reject polymorphic types) (~20 LOC)
3. Validate all fields are Eq-able (primitives, other derived types) (~30 LOC)
4. Accept `==` on derived types in typechecker (~20 LOC)
5. Add error messages for unsupported cases (~10 LOC)

**Acceptance Criteria:**
- [ ] `c1 == c2` type-checks when Color has `deriving (Eq)`
- [ ] `Option[T] deriving (Eq)` produces clear error
- [ ] `type Foo = Bar(SomeNonEq) deriving (Eq)` produces clear error

**Files:**
- `internal/types/check.go` or `internal/types/infer.go`
- `internal/types/env.go` (if needed)
- `internal/elaborate/elaborate.go`

### M3: Code Generation (~2 hours, ~120 LOC)

**Tasks:**
1. Generate internal equality function during elaboration (~60 LOC)
2. Handle nullary constructors (tag comparison) (~15 LOC)
3. Handle constructors with primitive fields (~25 LOC)
4. Handle recursive ADT types (~20 LOC)
5. Lower `==` on derived types to function call

**Acceptance Criteria:**
- [ ] `Red == Blue` evaluates to `false`
- [ ] `Leaf(1) == Leaf(1)` evaluates to `true`
- [ ] `Node(Leaf(1), 2, Leaf(3)) == Node(Leaf(1), 2, Leaf(3))` evaluates to `true`

**Files:**
- `internal/elaborate/elaborate.go`
- `internal/gen/golang/codegen_expr.go`
- `internal/eval/eval.go` (if interpreter changes needed)

### M4: Testing & Examples (~1 hour, ~70 LOC)

**Tasks:**
1. Create example file `examples/runnable/deriving_eq.ail`
2. Add integration tests for all cases
3. Test error messages for unsupported cases
4. Verify `make test` passes

**Acceptance Criteria:**
- [ ] Example file runs successfully
- [ ] All tests pass
- [ ] Error messages are clear and helpful

**Files:**
- `examples/runnable/deriving_eq.ail`
- `internal/parser/parser_type_test.go`
- `internal/types/check_test.go` (if exists)

## Total Estimates

| Milestone | LOC | Time |
|-----------|-----|------|
| M1: Lexer & Parser | 80 | 1.5h |
| M2: Type Checker | 100 | 1.5h |
| M3: Code Generation | 120 | 2h |
| M4: Testing | 70 | 1h |
| **Total** | **370** | **6h** |

## Success Criteria

- [ ] `deriving (Eq)` parses without errors
- [ ] `==` works on derived types
- [ ] Works with simple enums (Color, DeckID)
- [ ] Works with ADTs with primitive fields (Tree)
- [ ] Works with records
- [ ] Works with recursive types
- [ ] Clear error for polymorphic types
- [ ] Clear error for types with non-Eq fields
- [ ] Example file created and working
- [ ] `make test` passes

## Risk Factors

| Risk | Mitigation |
|------|------------|
| Codegen complexity | Start with simple enums, add fields later |
| Recursive type handling | Use structural recursion (safe for ADTs) |
| Integration with existing `==` | Check how primitives are handled first |

## Open Questions

1. Should records also support `deriving (Eq)`? → **Yes, per design doc**
2. Where to store derived type info? → Type environment or elaborator state
