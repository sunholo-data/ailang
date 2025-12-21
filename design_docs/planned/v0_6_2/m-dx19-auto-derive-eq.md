# M-DX19: Auto-Derive Eq for ADT Types

**Status**: Planned
**Target**: v0.6.2
**Priority**: P1 (Medium - significant boilerplate reduction)
**Estimated**: 4-6 hours
**Dependencies**: None
**Reporter**: stapledons_voyage (agent message)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Derived equality is deterministic syntactic elaboration |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Pure function generation |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Reduces boilerplate that obscures real logic |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents can use derived equality without manual implementation |
| A8: Minimal Syntax | 0 | Adds `deriving` keyword but reduces code overall |
| A9: Cost Visibility | 0 | Generated code has same cost as manual |
| A10: Composability | +1 | ADTs with Eq compose correctly in collections, map keys, sets |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Generated equality is deterministic
- [x] A3 (Effects): Pure function, no side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Improves AI agent experience

## Problem Statement

ADT types require manual equality functions, creating 10-15 lines of boilerplate per enum type.

**Current State:**
```ailang
type DeckID = Bridge | ObservationDeck | Engineering | Medical | Cargo | Crew | Recreation

-- Manual equality function: 15+ lines of boilerplate
pure func deckIDEquals(a: DeckID, b: DeckID) -> bool =
    match a {
        Bridge => match b { Bridge => true, _ => false }
        ObservationDeck => match b { ObservationDeck => true, _ => false }
        Engineering => match b { Engineering => true, _ => false }
        Medical => match b { Medical => true, _ => false }
        Cargo => match b { Cargo => true, _ => false }
        Crew => match b { Crew => true, _ => false }
        Recreation => match b { Recreation => true, _ => false }
    }
```

**Impact:**
- ~10-15 lines per enum type
- Error-prone: easy to miss a case or typo
- Obscures real business logic with mechanical code
- stapledons_voyage has 13 ADT types, potentially 130-195 lines of boilerplate
- Name collision risk: `eqColor`, `colorEq`, `colorEquals`?

## Goals

**Primary Goal:** Automatically generate equality for ADT types via `==` operator.

**Success Metrics:**
- `deriving (Eq)` syntax supported on type declarations
- `==` operator works on derived types (no separate named function)
- Works with nullary constructors (enum-like ADTs)
- Works with constructors with primitive fields
- Clear errors for unsupported cases (polymorphic types)

## Design Decisions

### Decision 1: Desugar `==`, Don't Generate Named Functions

**Problem:** If we generate `eqType` functions, we risk:
- Name collisions (`eqColor`, `colorEq`, `colorEquals`?)
- Split mental model: sometimes `x == y`, sometimes `eqX(x, y)`

**Solution:** `deriving (Eq)` tells the compiler to desugar `==` on that type.

```ailang
-- User writes:
type Color = Red | Green | Blue
deriving (Eq)

-- User uses:
if c1 == c2 then ...  -- Works!

-- Compiler internally generates (private, mangled name):
-- __eq$module$Color(a, b) -> match tree
```

**Benefits:**
- `==` is the universal surface
- No public naming game
- Compiler-synthesized name is stable and internal

### Decision 2: Monomorphic Only in v0.6.2

**Problem:** Generic deriving requires constraint solving:
```ailang
type Option[T] deriving (Eq) = None | Some(T)
-- Implicitly requires: Eq[T] must exist
-- This is basically typeclass territory
```

**Solution for v0.6.2:** Reject polymorphic types with clear error.

```
Error: cannot derive Eq for polymorphic type Option[T] without Eq constraints
  Hint: This feature requires typeclass-style constraints (deferred to v0.7+)
  Workaround: Define a manual equality function for each concrete instantiation
```

**Allowed types for deriving:**
- Primitives: `int`, `bool`, `string`
- `float` (with documented IEEE semantics)
- Records whose fields are all Eq-able
- ADTs that themselves derive Eq (all constructors, all fields Eq-able)
- **Not allowed:** `[T]`, `Option[T]`, any type with type parameters

### Decision 3: Float Equality Uses IEEE Semantics

**Decision:** Float equality follows IEEE 754 semantics:
- `NaN == NaN` is `false`
- `-0.0 == 0.0` is `true`

**Rationale:**
- Matches Go backend behavior
- Deterministic across backends
- Well-documented, predictable

**Documentation note:** Users who need total equality for floats can use explicit comparison functions.

### Decision 4: Syntax - Post-Definition Deriving

**Chosen syntax (Option B):**
```ailang
type DeckID = Bridge | ObservationDeck | Engineering
deriving (Eq)
```

**Rationale:**
- Easier to extend without grammar ambiguity around `=`
- Clear visual separation between type definition and derived instances
- Extensible to multiple derives: `deriving (Eq, Show)`

## Solution Design

### Syntax

```ailang
-- Simple enum
type Color = Red | Green | Blue
deriving (Eq)

-- ADT with fields (all fields must be Eq-able)
type Tree = Leaf(int) | Node(Tree, int, Tree)
deriving (Eq)

-- Record type
type Point = {x: float, y: float}
deriving (Eq)
```

### Implementation Shape

**AST Changes:**
```go
type TypeDecl struct {
    Name       string
    TypeParams []string    // For generic types
    Definition Type
    Deriving   []DeriveKind // NEW: enum like DeriveEq, DeriveShow
}

type DeriveKind int
const (
    DeriveEq DeriveKind = iota
    // DeriveShow - future
    // DeriveOrd - future
)
```

**Elaborator:**
- When it sees `deriving (Eq)` for a closed (monomorphic) type
- Registers an internal equality implementation
- Creates lowered rule for `==` on that type

**Typechecker:**
- `==` requires both operands same type
- If type has derived Eq (or built-in Eq), accept
- Else error: "type X does not support equality; add `deriving (Eq)`"

**Lowering/Codegen:**
- `==` becomes either:
  - Direct tag/field compare chain (inline)
  - Call to compiler-synthesized `__eq$module$Type`

### Generated Equality Logic

**For nullary constructors (enums):**
```go
// type Color = Red | Green | Blue deriving (Eq)
// Generates tag comparison:
func __eq$module$Color(a, b *Color) bool {
    return a.Tag == b.Tag
}
```

**For constructors with fields:**
```go
// type Tree = Leaf(int) | Node(Tree, int, Tree) deriving (Eq)
func __eq$module$Tree(a, b *Tree) bool {
    if a.Tag != b.Tag {
        return false
    }
    switch a.Tag {
    case TagLeaf:
        return a.Leaf.Value == b.Leaf.Value
    case TagNode:
        return __eq$module$Tree(a.Node.Left, b.Node.Left) &&
               a.Node.Value == b.Node.Value &&
               __eq$module$Tree(a.Node.Right, b.Node.Right)
    }
    return false
}
```

### Implementation Plan

**Phase 1: Parser Support** (~2 hours)
- [ ] Add `DERIVING` keyword to lexer
- [ ] Extend TypeDecl AST with `Deriving []DeriveKind`
- [ ] Parse `deriving (Eq)` after type definition
- [ ] Error on `deriving` before `=` (Option A syntax)

**Phase 2: Validation & Typechecker** (~1.5 hours)
- [ ] Validate type is monomorphic (no type parameters)
- [ ] Validate all fields are Eq-able (primitives or derived Eq types)
- [ ] Register derived types in type environment
- [ ] Accept `==` on derived types in typechecker

**Phase 3: Lowering/Codegen** (~2 hours)
- [ ] Generate internal equality function during elaboration
- [ ] Lower `==` on derived types to function call
- [ ] Handle recursive types correctly

**Phase 4: Testing** (~1 hour)
- [ ] Unit tests for parser
- [ ] Integration tests for generated equality
- [ ] Error message tests for unsupported cases

### Files to Modify

**Modified files:**
- `internal/lexer/token.go` - Add DERIVING token (~5 LOC)
- `internal/lexer/lexer.go` - Recognize `deriving` keyword (~5 LOC)
- `internal/ast/ast.go` - Add Deriving field to TypeDecl (~15 LOC)
- `internal/parser/parser_type.go` - Parse deriving clause (~40 LOC)
- `internal/types/check.go` - Accept `==` on derived types (~20 LOC)
- `internal/elaborate/elaborate.go` - Generate eq implementation (~60 LOC)
- `internal/gen/golang/codegen_expr.go` - Generate equality calls (~30 LOC)

## Examples

### Example 1: Simple Enum

```ailang
type Color = Red | Green | Blue
deriving (Eq)

pure func isRed(c: Color) -> bool = c == Red

-- Works: comparison returns true/false
```

### Example 2: ADT with Fields

```ailang
type Tree = Leaf(int) | Node(Tree, int, Tree)
deriving (Eq)

pure func sameTree(a: Tree, b: Tree) -> bool = a == b

-- Works: recursive structural comparison
```

### Example 3: Record Type

```ailang
type Point = {x: float, y: float}
deriving (Eq)

pure func samePoint(a: Point, b: Point) -> bool = a == b

-- Works: field-by-field comparison (IEEE float semantics)
```

### Example 4: Polymorphic Type (Error)

```ailang
type Option[T] = None | Some(T)
deriving (Eq)

-- Error: cannot derive Eq for polymorphic type Option[T] without Eq constraints
--   Hint: This feature requires typeclass-style constraints (deferred to v0.7+)
```

## Success Criteria

- [ ] `deriving (Eq)` parses without errors
- [ ] `==` works on derived types
- [ ] Works with simple enums (DeckID, Color, etc.)
- [ ] Works with ADTs with primitive fields
- [ ] Works with records
- [ ] Works with recursive types (Tree)
- [ ] Clear error for polymorphic types
- [ ] Clear error for types with non-Eq fields
- [ ] stapledons_voyage types compile with deriving
- [ ] All existing tests pass
- [ ] `make test` passes

## Go/No-Go Conditions

**Go if:**
- [x] Monomorphic only in v0.6.2
- [x] `==` is the surface (no public naming game)
- [x] Clear diagnostics for unsupported derives
- [x] Float equality documented as IEEE semantics

**No-go only if:**
- Planning to ship polymorphic `deriving (Eq)` without constraints
- Would leak typeclass-like behavior and create unsound corners

## Semantic Edge Cases

### Float Equality
- **Decision:** IEEE 754 semantics (`NaN == NaN` is false)
- **Documented:** Users expecting total equality should use explicit functions

### Recursive ADTs
- Derived equality is recursive and structurally decreasing
- No cycles possible in pure ADTs (no references)
- No special runtime needed

### Performance
- Match-based equality is optimal and predictable
- No hashing tricks needed
- Compiler can inline for small types

## Non-Goals

**Not in v0.6.2:**
- Polymorphic deriving (`Option[T]`, `List[T]`) - requires constraints
- Other derivable type classes (`Show`, `Ord`) - separate feature
- User-defined deriving mechanisms
- Instance constraints (`Eq a => Eq [a]`)

## Future Work

**v0.7+:**
- `deriving (Show)` for debug printing
- `deriving (Ord)` for ordering/comparison
- Polymorphic deriving with Eq constraints
- Container equality (`[T]` where `T: Eq`)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Parser ambiguity | Medium | Use post-definition syntax |
| Recursive types | Low | Structural recursion is safe |
| Float edge cases | Low | Document IEEE semantics |
| Scope creep to typeclasses | High | Strict monomorphic-only for v0.6.2 |

## Open Questions

1. **Container equality:** Should `[int]` support `==` with order-sensitive comparison in v0.6.2, or defer?
   - **Recommendation:** Defer, handle as primitive equality only

2. **Future `deriving (Show)`:** Should it follow same pattern?
   - **Recommendation:** Yes, separate design doc when needed

## Related Documents

- [M-DX18: Function Namespacing](../v0_6_1/m-dx18-codegen-function-namespacing.md) - Recent codegen fix
- Type classes (deferred to v0.7.0+)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage DX Feedback (agent message)
- Haskell's deriving mechanism
- IEEE 754 floating-point standard

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20
