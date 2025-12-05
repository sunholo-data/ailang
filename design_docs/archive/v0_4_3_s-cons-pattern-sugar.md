# S-CONS Pattern Sugar (x :: xs in Patterns)

**Status**: Planned
**Target**: v0.4.3
**Priority**: P1 (Medium-High - fixes 12% of parse failures)
**Estimated**: 3-5 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Allows familiar ML-style pattern syntax without ceremony |
| Preserve Semantic Clarity | + | +1 | Bijective desugaring to canonical ::(x, xs); semantics unchanged |
| Increase Determinism | 0 | 0 | Pure surface sugar; no semantic changes |
| Lower Token Cost | + | +1 | Fewer tokens in patterns (~15% reduction in list patterns) |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- `x :: xs` syntax works in **expressions** but fails in **patterns**
- Results in 36 PAR_001 parse errors (12% of all failures in v0.4.2 eval baseline)
- Inconsistent with ML-family language expectations (OCaml, Haskell, F#)
- Forces verbose canonical form `::(x, xs)` in patterns only

**Current behavior:**
```ailang
-- ✅ WORKS (expression)
let list = x :: xs :: [] in ...

-- ❌ FAILS (pattern) - PAR_001 error
match list {
  x :: xs => ...  -- Parse error!
}

-- ✅ MUST USE (pattern) - verbose
match list {
  ::(x, xs) => ...  -- Canonical form
}
```

**Impact:**
- **Users affected:** All AI models, human developers familiar with ML languages
- **Significance:** 12% of parse failures are avoidable DX friction
- **Principle violation:** Violates "least surprise" - sugar exists for expressions but not patterns

## Goals

**Primary Goal:** Allow `x :: xs` syntax in pattern position, desugaring to `::(x, xs)` exactly like expressions

**Success Metrics:**
- 36 PAR_001 errors eliminated (12% reduction in parse failures)
- Pattern/expression syntax parity achieved
- Zero regression in existing pattern matching
- Test coverage: 100% for new pattern forms
- Documentation updated with examples and strict-mode behavior

## Solution Design

### Overview

Extend the pattern parser to support right-associative `::` operator, mirroring the existing expression-level implementation. This is **pure syntactic sugar** - the canonical form `::(x, xs)` remains unchanged, and the AST representation is identical.

**Key principle:** Bijective desugaring - `x :: xs` in patterns means exactly the same as in expressions.

### Architecture

**Components:**
1. **Pattern Parser Extension**: Add right-associative loop for `::` after base pattern parsing
2. **AST Representation**: Desugar immediately to canonical `CtorPattern(name="::", args=[head, tail])`
3. **Strict Mode Support**: In `--strict-syntax` mode, reject sugar and suggest canonical form
4. **Formatter**: Keep printing canonical `::(x, xs)` unless future `--format:sugar=cons` flag

**No lexer changes needed** - `::` token already exists for expressions.

### Implementation Plan

**Phase 1: Parser Extension** (~2 hours)
- [ ] Add right-associative `::` loop in `parsePattern()` function
- [ ] Handle precedence correctly (atomic patterns on left, any pattern on right)
- [ ] Desugar to `ConsPattern` or canonical `CtorPattern(name="::", args=[head, tail])`
- [ ] Support parenthesized patterns: `(x :: xs)`
- [ ] Support wildcards: `(_ :: xs)`, `(x :: _)`
- [ ] Right-associative nesting: `a :: b :: c` → `::(a, ::(b, c))`

```go
// Pseudocode for parser change
func (p *Parser) parsePattern() Pattern {
    pat := p.parseBasePattern() // Ident, _, literal, ctor, tuple, etc.

    // Right-associative :: loop
    for p.curTokenIs(lexer.CONS) { // '::'
        p.nextToken()
        rhs := p.parsePattern() // Allow full pattern recursion
        pat = ConsPattern{Head: pat, Tail: rhs} // or CtorPattern("::", [pat, rhs])
    }

    return pat
}
```

**Phase 2: Strict Mode Support** (~30 minutes)
- [ ] In `--strict-syntax` mode, emit error on `x :: xs` in patterns
- [ ] Error message: `"Use canonical ::(x, xs) instead of x :: xs in patterns"`
- [ ] Ensure strict mode does not break existing `::(x, xs)` usage

**Phase 3: Testing** (~2 hours)
- [ ] Happy paths: `x :: xs`, `1 :: rest`, `_ :: _`, right-associativity
- [ ] Mixed forms: `::(x, xs)` and `x :: xs` in same match
- [ ] Guards: `x :: xs if p(x) => ...`
- [ ] Empty list: `x :: []`
- [ ] Nested: `x :: (y :: ys)`
- [ ] Strict mode: error with helpful message
- [ ] Negatives: `(:: x xs)` remains illegal, spacing ambiguities caught

**Phase 4: Documentation** (~30 minutes)
- [ ] Update teaching prompt (`prompts/`) to document sugar
- [ ] Update "Pattern Matching" guide in docs/
- [ ] Update "Common Mistakes" section
- [ ] Note canonical form remains `::(x, xs)`; sugar is optional

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser.go` - Add `::` loop in `parsePattern()` (~20 LOC)
- `internal/parser/parser_test.go` - Add pattern sugar tests (~150 LOC)
- `internal/ast/ast.go` - Ensure `ConsPattern` exists or use `CtorPattern` (~0-10 LOC)
- `prompts/v0.4.3.md` - Document pattern sugar (~10 LOC)
- `docs/docs/guides/pattern-matching.md` - Add examples (~20 LOC)

**No new files needed** - this is a parser-only change.

## Examples

### Example 1: Basic Pattern Matching

**Before (v0.4.2):**
```ailang
-- ❌ Parse error: x :: xs not allowed in patterns
match list {
  x :: xs => x
}

-- ✅ Must use verbose canonical form
match list {
  ::(x, xs) => x
}
```

**After (v0.4.3):**
```ailang
-- ✅ Both forms work!
match list {
  x :: xs => x         -- Sugar (preferred by ML-family users)
}

match list {
  ::(x, xs) => x       -- Canonical (required in strict mode)
}
```

### Example 2: Right-Associativity

**Before:**
```ailang
match list {
  ::(a, ::(b, c)) => a + b  -- Verbose nesting
}
```

**After:**
```ailang
match list {
  a :: b :: c => a + b      -- Natural right-associative syntax
}
-- Desugars to: ::(a, ::(b, c))
```

### Example 3: Guards and Empty Lists

**Before:**
```ailang
match list {
  ::(x, []) if x > 0 => x
}
```

**After:**
```ailang
match list {
  x :: [] if x > 0 => x     -- Cleaner with sugar
}
```

### Example 4: Wildcards

**Before:**
```ailang
match list {
  ::(_, xs) => length(xs)   -- Ignore head
}
```

**After:**
```ailang
match list {
  _ :: xs => length(xs)     -- More idiomatic
}
```

### Example 5: Strict Mode

```bash
# With --strict-syntax flag
ailang check --strict-syntax module.ail

# Error output:
# [PAR_SUGAR_001] Use canonical ::(x, xs) instead of x :: xs in patterns
#   at line 10, col 5
#   Hint: In strict mode, only canonical forms are allowed
```

## Success Criteria

- [ ] `x :: xs` parses correctly in all pattern positions
- [ ] Desugars to identical AST as `::(x, xs)` (bijective)
- [ ] Right-associativity works: `a :: b :: c` → `::(a, ::(b, c))`
- [ ] Wildcards work: `_ :: xs`, `x :: _`
- [ ] Guards work: `x :: xs if p(x) => ...`
- [ ] Mixed forms work: `::(x, xs)` and `x :: xs` in same match
- [ ] Strict mode rejects sugar with helpful message
- [ ] All existing pattern tests pass (zero regression)
- [ ] New tests: 15+ test cases covering happy/edge/negative paths
- [ ] Documentation updated (prompt, guides, changelog)
- [ ] Examples added to `examples/pattern_sugar.ail`

## Testing Strategy

**Unit tests (`internal/parser/parser_test.go`):**
- Parse `x :: xs` → verify AST matches `::(x, xs)`
- Parse `a :: b :: c` → verify right-associative nesting
- Parse `1 :: rest`, `_ :: _` → verify literals/wildcards work
- Parse `x :: []` → verify empty list terminator
- Parse `(x :: xs)` → verify parenthesized patterns
- Negative: `(:: x xs)` → verify spacing ambiguity caught
- Negative: `x ::` at EOF → verify helpful error message

**Integration tests (`internal/pipeline/pipeline_test.go`):**
- Full compilation: `match xs { x :: xs => ... }` → Core AST → evaluated
- Type checking: ensure inferred types match canonical form
- Pattern exhaustiveness: sugar doesn't affect exhaustiveness checker

**Manual testing (`examples/pattern_sugar.ail`):**
```ailang
-- Test file for v0.4.3 pattern sugar
module examples/pattern_sugar

def main: !{IO} unit =
  let list = [1, 2, 3] in
  match list {
    x :: xs => print(show(x))  -- Should print "1"
  | [] => print("empty")
  }
```

**Strict mode test:**
```bash
ailang check --strict-syntax examples/pattern_sugar.ail
# Expected: Error suggesting canonical ::(x, xs) form
```

## Non-Goals

**Not in this feature:**
- Type-level syntax sugar (`::`  in types remains as-is) - No user demand, types are already concise
- Expression precedence changes - Already correct, no regressions
- Formatter changes to print sugar by default - Canonical form remains default output
- LSP/IDE integration - AILANG doesn't prioritize IDE tooling (AI-first DX)
- Custom infix operators in patterns - Out of scope, requires broader design

## Timeline

**Day 1** (3 hours):
- Phase 1: Parser extension (2 hours)
- Phase 2: Strict mode support (30 minutes)
- Phase 3: Core tests (30 minutes)

**Day 2** (2 hours):
- Phase 3: Complete testing (1 hour)
- Phase 4: Documentation (30 minutes)
- Integration testing (30 minutes)

**Total: ~5 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Regression in existing pattern matching | High | Run full test suite; add regression tests for all existing pattern forms |
| Ambiguity with other syntax | Medium | Parser already handles `::` in expressions; pattern context is unambiguous |
| Performance impact | Low | Desugaring is O(1) per pattern; no runtime cost (happens at parse time) |
| Breaking change for strict mode users | Low | Strict mode is opt-in; most users won't be affected; document clearly |

## References

- **Related Design Docs:**
  - [v0.4.2 Eval Baseline Analysis](./ACTUAL_STATUS_AND_ACTION_PLAN.md) - Identified PAR_001 failures
  - [Pattern Matching](../../implemented/v0_2_0/pattern-matching.md) - Original pattern matching design

- **Prior Art:**
  - OCaml: `x :: xs` in patterns (standard syntax)
  - Haskell: `x : xs` in patterns (standard syntax)
  - F#: `x :: xs` in patterns (standard syntax)
  - Standard ML: `x :: xs` in patterns (standard syntax)

- **Implementation Examples:**
  - Expression-level cons sugar: `internal/parser/parser.go` (existing implementation)
  - Pattern parsing: `internal/parser/parser.go:parsePattern()`

## Future Work

**Post-v0.4.3 enhancements (not required for this feature):**
- Formatter flag `--format:sugar=cons` to print sugar instead of canonical form
- Additional infix operators in patterns (e.g., custom operators if user-defined operators land)
- LSP integration for auto-conversion between forms (low priority - AI-first DX)

---

**Document created**: 2025-11-11
**Last updated**: 2025-11-11
