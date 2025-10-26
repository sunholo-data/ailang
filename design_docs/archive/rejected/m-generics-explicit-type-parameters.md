# M-GENERICS: Explicit Type Parameters (Syntax Sugar)

**Status**: Planned
**Target**: v0.4.1 (or defer - low priority)
**Priority**: P2 (Low - Syntactic sugar only, inference already works)
**Estimated**: 1 week (~30 hours)
**Dependencies**: None (type inference already handles polymorphism)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | − | −1 | Adds optional syntax where inference already works |
| Preserve Semantic Clarity | + | +1 | Makes polymorphic constraints explicit and self-documenting |
| Increase Determinism | 0 | 0 | No change - inference is already deterministic |
| Lower Token Cost | 0 | 0 | Neutral - optional syntax |
| **Net Score** | | **0** | **Decision: Borderline - implement only if documentation value justifies it** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- Hindley-Milner type inference handles polymorphism automatically
- Generic functions work without explicit type parameters
- `quicksort.ail` example uses syntax that doesn't exist yet: `func quicksort[a: Ord](list: [a])`

**Impact:**
- **Example file blocked**: `examples/experimental/quicksort.ail` uses `[a: Ord]` syntax
- **Documentation clarity**: Polymorphic constraints are implicit, not self-documenting
- **API design**: No way to explicitly declare type parameters for documentation purposes

**Current working code:**
```ailang
-- This already works! No [a: Ord] syntax needed
pure func quicksort(list: [a]) -> [a] {
  match list {
    [] => [],
    [pivot, ...rest] => {
      let less = filter(\x. x < pivot, rest);     -- Ord constraint inferred!
      let greater = filter(\x. x >= pivot, rest);
      quicksort(less) ++ [pivot] ++ quicksort(greater)
    }
  }
}

-- Type system infers: ∀a. [Ord a] => [a] -> [a]
```

**Problem:** The syntax `func quicksort[a: Ord]` doesn't exist, but it would be nice for:
1. **Documentation** - Makes type constraints visible in function signature
2. **Error messages** - Clearer errors when wrong type is passed
3. **Teaching** - Helps humans understand polymorphic code

## Goals

**Primary Goal:** Add optional explicit type parameter syntax for documentation and clarity purposes

**Success Metrics:**
- `quicksort.ail` example works with `[a: Ord]` syntax
- Type inference still works without explicit parameters (backward compatible)
- Error messages reference explicit type parameter names
- AI models can generate code with explicit type parameters (M-EVAL)

## Solution Design

### Overview

Add **optional** explicit type parameter syntax to function declarations:

```ailang
-- Explicit type parameters (NEW)
pure func quicksort[a: Ord](list: [a]) -> [a] { ... }

-- Equivalent to implicit (EXISTING)
pure func quicksort(list: [a]) -> [a] { ... }
```

Key design principle: **This is purely syntactic sugar.** Type inference already handles everything - this just makes it explicit in the source code.

### Architecture

**Components:**

1. **Parser Extensions** (`internal/parser/parser.go`)
   - Parse `[TypeVar: Constraint, ...]` after function name
   - Build TypeParameterList AST node
   - Attach to FuncDecl

2. **Type Checker** (`internal/types/typechecker.go`)
   - Validate explicit type parameters match inferred types
   - Use explicit names in error messages
   - No change to inference algorithm (already works)

3. **Error Reporting** (`internal/errors/`)
   - Reference explicit type parameter names in errors
   - Show constraint violations clearly

### Syntax Design

**Function declarations with type parameters:**

```ailang
-- Single type parameter with constraint
pure func map[a, b](f: a -> b, list: [a]) -> [b] { ... }

-- Multiple constraints
pure func sort[a: Ord, b: Show](pairs: [(a, b)]) -> [(a, b)] { ... }

-- No constraints (pure polymorphism)
pure func id[a](x: a) -> a { x }
```

**Constraints:**
- `Num` - Numeric operations (+, -, *, /)
- `Eq` - Equality (==, !=)
- `Ord` - Ordering (<, >, <=, >=)
- `Show` - String conversion (show)

**Equivalent to Haskell:**
```haskell
-- Haskell
quicksort :: Ord a => [a] -> [a]

-- AILANG
pure func quicksort[a: Ord](list: [a]) -> [a]
```

### Implementation Plan

**Phase 1: Parser** (~10 hours)
- [ ] Add `[` `]` tokens for type parameters
- [ ] Parse type parameter lists
- [ ] Parse type constraints (`: Ord`, `: Num`, etc.)
- [ ] Build TypeParameterList AST node
- [ ] Unit tests for parser

**Phase 2: Type Checker Integration** (~15 hours)
- [ ] Extract explicit type parameters from FuncDecl
- [ ] Validate against inferred type variables
- [ ] Ensure constraints match usage
- [ ] Use explicit names in error messages
- [ ] Unit tests for type checking

**Phase 3: Documentation & Examples** (~5 hours)
- [ ] Update `quicksort.ail` to work
- [ ] Add generics examples to docs
- [ ] Update teaching prompt
- [ ] CHANGELOG entry

### Files to Modify/Create

**New files:**
- `internal/ast/typeparameters.go` - TypeParameterList AST node (~100 LOC)

**Modified files:**
- `internal/parser/parser.go` - Parse type parameters (~150 LOC)
- `internal/ast/ast.go` - Add TypeParameterList to FuncDecl (~20 LOC)
- `internal/types/typechecker.go` - Validate type parameters (~100 LOC)
- `internal/errors/errors.go` - Use explicit names in errors (~50 LOC)

**Total new code: ~420 LOC**

## Examples

### Example 1: Generic Map Function

**Before (implicit - already works):**
```ailang
pure func map(f: a -> b, list: [a]) -> [b] {
  match list {
    [] => [],
    [x, ...xs] => [f(x), ...map(f, xs)]
  }
}

-- Type inferred: ∀a b. (a -> b) -> [a] -> [b]
```

**After (explicit - more documentation-friendly):**
```ailang
pure func map[a, b](f: a -> b, list: [a]) -> [b] {
  match list {
    [] => [],
    [x, ...xs] => [f(x), ...map(f, xs)]
  }
}

-- Same type, but now explicitly declared
```

### Example 2: Constrained Quicksort

**Before (implicit constraint):**
```ailang
pure func quicksort(list: [a]) -> [a] {
  match list {
    [] => [],
    [pivot, ...rest] => {
      let less = filter(\x. x < pivot, rest);  -- Ord inferred from <
      quicksort(less) ++ [pivot] ++ quicksort(greater)
    }
  }
}
```

**After (explicit constraint):**
```ailang
pure func quicksort[a: Ord](list: [a]) -> [a] {
  match list {
    [] => [],
    [pivot, ...rest] => {
      let less = filter(\x. x < pivot, rest);  -- Ord explicit in signature
      quicksort(less) ++ [pivot] ++ quicksort(greater)
    }
  }
}

-- Error if called with non-Ord type:
-- quicksort([func1, func2])
-- Error: Type parameter 'a' requires Ord constraint, but (int -> int) does not implement Ord
```

### Example 3: Multiple Constraints

```ailang
-- Print sorted list of comparable items
pure func printSorted[a: Ord & Show](list: [a]) -> () ! {IO} {
  let sorted = quicksort(list);  -- Needs Ord
  let _ = map(\x. _io_println(show(x)), sorted);  -- Needs Show
  ()
}

-- Works: printSorted([1, 2, 3])
-- Works: printSorted(["a", "b", "c"])
-- Error: printSorted([\x.x, \y.y])  -- Functions don't implement Show
```

## Success Criteria

- [ ] `quicksort.ail` example runs with `[a: Ord]` syntax
- [ ] Type inference still works without explicit parameters (backward compatible)
- [ ] Error messages reference type parameter names
- [ ] Parser handles all constraint types (Num, Eq, Ord, Show)
- [ ] All existing examples work unchanged (backward compatibility)
- [ ] AI models can generate code with explicit type parameters
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Parser correctly handles `[a]`, `[a: Ord]`, `[a, b]`, `[a: Ord & Show]`
- Type checker validates explicit parameters match inferred types
- Error messages use explicit type parameter names

**Integration tests:**
- `quicksort.ail` runs correctly
- Mixed code (some with explicit params, some without) works
- Constraint violations produce clear error messages

**Backward compatibility tests:**
- All existing examples still work without modifications
- Inference works identically with or without explicit parameters

## Non-Goals

**Not in this feature:**
- **Higher-kinded types** - No `[f: * -> *]` (functor-like abstractions)
- **Type-level computation** - No type families or associated types
- **User-defined type classes** - Only built-in constraints (Num, Eq, Ord, Show)
- **Rank-N types** - Only Rank-1 polymorphism
- **Existential types** - No `∃a. ...` syntax
- **Dependent types** - No value-dependent types

## Timeline

**Week 1** (30 hours):
- Phase 1: Parser (10 hours)
- Phase 2: Type Checker Integration (15 hours)
- Phase 3: Documentation & Examples (5 hours)

**Total: ~30 hours across 1 week**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Adds complexity without benefit** | Medium | Make it purely optional - no semantic change |
| **Inference confusion** | Low | Validate explicit parameters match inferred types |
| **Backward compatibility** | Low | All existing code works unchanged |
| **AI model adoption** | Low | Update teaching prompt with examples |

## References

- **Type parameter syntax**:
  - Haskell: `quicksort :: Ord a => [a] -> [a]`
  - Rust: `fn quicksort<T: Ord>(list: Vec<T>) -> Vec<T>`
  - TypeScript: `function quicksort<T extends Comparable>(list: T[]): T[]`

- **Example files requiring this feature**:
  - `examples/experimental/quicksort.ail`

- **Related design docs**:
  - [v0.4-roadmap.md](../v0.4-roadmap.md) - Overall v0.4 plan

## Decision: Implement or Defer?

**Recommendation**: **Defer to v0.5.0 or later** - Low priority syntactic sugar.

**Rationale:**
- **Net score: 0** (borderline by AI-first criteria)
- **Low priority**: Only 1 example file blocked
- **Adds syntax**: Violates "reduce syntactic noise" principle
- **Alternative exists**: Type inference already works perfectly
- **Documentation value**: Could be achieved with comments instead

**Arguments for implementation:**
- Makes type constraints self-documenting
- Clearer error messages
- Familiar syntax for developers from other languages

**Arguments against:**
- Adds syntax where none is needed
- Inference already works perfectly
- Goes against "minimal syntax" philosophy
- Could confuse beginners (when to use explicit vs implicit?)

**If implemented:**
- Make it optional and discouraged (prefer inference)
- Only use for documentation in stdlib/examples
- Add linter rule to discourage overuse

**Alternative approach:**
- Use comments for documentation instead:
```ailang
-- Type: ∀a. [Ord a] => [a] -> [a]
pure func quicksort(list: [a]) -> [a] { ... }
```

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26
