# Cons (`::`) as Expression Operator

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 - Medium (AI agent ergonomics)
**Estimated**: 2-4 hours
**Dependencies**: None (all infrastructure exists)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure list construction; no nondeterminism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Cons is pure; no effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type `(a, [a]) -> [a]` is locally checkable |
| A6: Safe Concurrency | 0 | Immutable list construction |
| A7: Machines First | +1 | AI agents already attempt `x :: xs` (ML/Haskell muscle memory); currently fails, wasting tokens on workarounds |
| A8: Minimal Syntax | 0 | `::` token already exists; no new syntax, just enabling it in expression position |
| A9: Cost Visibility | 0 | `::` has the semantic meaning of prepend. In the current runtime, prepend may still lower to copy-backed list construction (same cost as `[x] ++ xs`). Runtime optimization of prepend cost is a separate concern. |
| A10: Composability | +1 | Right-associative chaining `1 :: 2 :: 3 :: []` composes naturally |
| A11: Structured Failure | 0 | Type errors remain typed |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +2** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly improves machine ergonomics

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| >= +2 | Proceed to implementation |

## Problem Statement

The `::` (cons) operator works in pattern matching but fails as an expression. Every functional language that AILANG draws from (Haskell, OCaml, F#, Elm, Elixir) supports cons in both positions.

**Current State:**
- `::` works in patterns: `match xs { x :: rest -> ..., [] -> ... }`
- `::` fails as an expression: `let xs = 1 :: [2, 3]` produces an unresolved variable error
- Workaround: `let xs = [1] ++ [2, 3]` -- allocates an intermediate single-element list

**Discovery:** During the ai-coding-lang-bench benchmark, Claude Code repeatedly tried `x :: xs` to build lists (standard ML/Haskell idiom). Each attempt failed, forcing restructuring to `[x] ++ xs`. This wasted tokens and time on every list-building task.

**Impact:**
- AI agents (primary AILANG consumers) lose tokens on failed attempts and workarounds
- Symmetry violation: patterns and expressions should support the same constructors
- Missing from stdlib examples and teaching prompts, creating a blind spot

## Goals

**Primary Goal:** Make `x :: xs` work as an expression that prepends `x` to list `xs`.

**Success Metrics:**
- `1 :: [2, 3]` evaluates to `[1, 2, 3]`
- `1 :: 2 :: 3 :: []` evaluates to `[1, 2, 3]` (right-associative chaining)
- Type checks as `(a, [a]) -> [a]`
- All existing tests continue to pass

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Desugar in elaborator vs. keep as BinOp | Determines where `::` is resolved to builtin call | compiler | design | low |
| Reuse existing `::` builtin vs. new intrinsic | Affects eval path and type checker | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Desugar `::` expression to builtin call in elaborator (not a new Core IR node)
- [x] Reuse the existing `::` builtin registered in `internal/builtins/list.go`

## Solution Design

### Overview

The parser already handles `::` correctly -- `parseConsExpression` desugars `x :: xs` to `FuncCall{Func: Identifier{"::"}, Args: [x, xs]}`. The problem is in the elaborator: `normalizeFuncCall` does not recognize `::` as a known function, so it falls through to local variable resolution, producing `core.Var{Name: "::"}` which is unresolvable.

The fix: add a special case in `normalizeFuncCall` (or in `normalize` for identifiers) that recognizes `::` and emits a `core.App` calling the `::` builtin via `core.VarGlobal`.

### Operator Identity

In source, `::` is an **infix expression operator** (right-associative).
In elaborated core, it lowers to the existing `std/list` cons builtin.
It is **not** yet guaranteed to be usable as an ordinary first-class function value (e.g., `let f = (::)` or `map(::, xss)` are not required to work).

**Associativity guarantee:**
```
1 :: 2 :: 3 :: []  ≡  1 :: (2 :: (3 :: []))
```
This is a parser-level guarantee and must be directly tested.

### Expression Context Coverage

The elaborator fix must handle `::` in **all** expression contexts, not only top-level `let` bindings. The parser produces the same `FuncCall{"::", ...}` shape in every position, but the implementer must verify:
- Nested cons chains (`1 :: 2 :: 3 :: []`)
- Cons inside lambdas (`\x -> x :: xs`)
- Cons inside let-bindings (`let ys = x :: xs in ...`)
- Cons as arguments to other functions (`f(x :: xs)`)
- Cons under if/match branches (`if p then x :: xs else []`)

If all of these go through the same `FuncCall{"::", ...}` parse path (expected), then the single elaborator patch is sufficient. If not, `::` should be treated as a reserved expression-form operator that elaborates uniformly to the `std/list` cons builtin regardless of parse path.

### First-Class Function Access

Users cannot currently write `std/list.::` or `let f = (::)` to obtain `::` as a first-class function value. This is a known limitation and is deferred to future work.

### Architecture

**The pipeline today (broken):**
1. Parser: `x :: xs` -> `ast.FuncCall{Func: "::", Args: [x, xs]}` (correct)
2. Elaborator: `normalizeFuncCall` -> `core.App{Func: core.Var{"::"}, Args: [...]}` (broken -- `::` is not a local var)
3. Type checker: fails or evaluator: unbound variable

**The pipeline after fix:**
1. Parser: `x :: xs` -> `ast.FuncCall{Func: "::", Args: [x, xs]}` (unchanged)
2. Elaborator: `normalizeFuncCall` detects `::` -> `core.App{Func: core.VarGlobal{Module: "std/list", Name: "::"}, Args: [...]}` (fixed)
3. Type checker: resolves `::` builtin type `(a, [a]) -> [a]` (works)
4. Evaluator: calls `listConsImpl` (works)

**Components:**
1. **Elaborator** (`internal/elaborate/expr_calls.go`): Add `::` recognition before the general constructor check
2. **Tests**: Parser sugar test, elaborator unit test, end-to-end `.ail` example
3. **Prompt/docs**: Update teaching prompts to include `::` as expression

### Implementation Plan

**Phase 1: Elaborator fix** (~1 hour)
- [ ] In `normalizeFuncCall`, add a check: if `ident.Name == "::"`, emit `core.App` with `VarGlobal{Module: "std/list", Name: "::"}`
- [ ] Normalize both args to atomic (already handled by the existing code path)
- [ ] Verify the `::` builtin is registered and resolvable

**Phase 2: Tests** (~1 hour)
- [ ] Add elaborator unit test: `1 :: [2, 3]` elaborates to `App(VarGlobal("std/list", "::"), [Lit(1), List([2, 3])])`
- [ ] Add end-to-end test: `1 :: 2 :: 3 :: []` produces `[1, 2, 3]`
- [ ] Add type error test: `1 :: 2` fails (second arg must be list)
- [ ] Verify `make test` passes

**Phase 3: Docs and examples** (~30 min)
- [ ] Add `examples/cons_expression.ail`
- [ ] Update CHANGELOG.md
- [ ] Update teaching prompts in `prompts/` if they mention list construction

### Files to Modify/Create

**Modified files:**
- `internal/elaborate/expr_calls.go` - Add `::` special case in `normalizeFuncCall`, ~10 LOC
- `internal/elaborate/expr_calls_test.go` or new test file - Elaborator unit tests, ~40 LOC

**New files:**
- `examples/cons_expression.ail` - End-to-end example, ~20 LOC

## Examples

### Example 1: Basic prepend

**Before (fails):**
```ailang
let xs = 1 :: [2, 3]   // Error: unresolved variable ::
```

**Workaround (works but wasteful):**
```ailang
let xs = [1] ++ [2, 3]  // Allocates intermediate [1] list
```

**After (works):**
```ailang
let xs = 1 :: [2, 3]   // [1, 2, 3]
```

### Example 2: Right-associative chaining

```ailang
let xs = 1 :: 2 :: 3 :: []   // [1, 2, 3]
// Parses as: 1 :: (2 :: (3 :: []))
```

### Example 3: In recursive functions

```ailang
func map(f: (a) -> b, xs: [a]) -> [b] {
  match xs {
    [] => [],
    x :: rest => f(x) :: map(f, rest)
  }
}
```

### Example 4: Pattern + expression symmetry

```ailang
func duplicate_head(xs: [int]) -> [int] {
  match xs {
    [] => [],
    x :: rest => x :: x :: rest   // Prepend x twice
  }
}
```

## Success Criteria

- [ ] `1 :: [2, 3]` evaluates to `[1, 2, 3]`
- [ ] `1 :: 2 :: 3 :: []` evaluates to `[1, 2, 3]` (right-associative)
- [ ] `::` preserves right associativity in expression position
- [ ] `x :: xs` works in all expression contexts (lambdas, let-bindings, function args, if/match branches), not only top-level lets
- [ ] `f(x) :: map(f, rest)` works in recursive context
- [ ] Type error on `1 :: 2` (second arg not a list)
- [ ] REPL and file pipeline behave identically
- [ ] Existing pattern `::` behavior is unchanged
- [ ] All existing tests pass (`make test`)
- [ ] `make verify-examples` passes
- [ ] Documentation updated (CHANGELOG, example file)

## Testing Strategy

**Unit tests:**
- Elaborator: `::` FuncCall produces correct `core.App` with `VarGlobal`
- Parser: existing `sugar_test.go` tests already cover parse-level desugaring

**Integration tests:**
- End-to-end: `.ail` file with `::` expressions, checked via `ailang run`
- Type checking: verify `(a, [a]) -> [a]` is inferred
- Error case: second argument not a list
- Right associativity: `1 :: 2 :: 3 :: []` parses as `1 :: (2 :: (3 :: []))`
- Expression contexts: cons inside lambdas, let-bindings, function args, if/match branches
- REPL parity: same results in REPL and file pipeline

**Manual testing:**
- REPL: `1 :: [2, 3]` in interactive mode
- Benchmark: re-run failing ai-coding-lang-bench task to confirm fix

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether to also add `::` as a first-class function value (e.g., `map(:: 0, xss)`) -- agent may choose, but not required
- Whether to add an SMT encoding for cons in the verifier -- agent may defer to future work

## Non-Goals

**Not attempted in this feature:**
- Changing list runtime representation to cons cells - Lists remain slice-backed; `::` is an ergonomic fix, not a runtime optimization
- Guaranteeing O(1) prepend in the current evaluator - Prepend may still copy; runtime optimization is separate
- Making `::` first-class for partial application - `let f = (::)` or `map(::, xss)` are deferred
- Lazy cons / stream construction - Out of scope; AILANG lists are strict
- `snoc` operator (append to end) - Different semantics, different design
- Custom infix operator framework - `::` is special-cased, not a general mechanism

## Timeline

**Single session** (~2-4 hours):
- Phase 1: Elaborator fix (1h)
- Phase 2: Tests (1h)
- Phase 3: Docs and examples (30min)
- Buffer for unexpected issues (30min)

**Total: ~2-4 hours in a single session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `::` identifier conflicts with type annotation syntax in some contexts | Low | Parser already separates CONS token (expression) from DCOLON (type annotation); no ambiguity |
| Builtin resolution path differs between file pipeline and REPL | Med | Test both paths; the `::` builtin is registered in `init()` so available everywhere |
| Monomorphization of polymorphic `::` calls | Low | Same mechanism as `++` which already works polymorphically |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_14/list_pattern_spread.md](design_docs/implemented/v0_3_14/list_pattern_spread.md) - `::` in patterns
- [design_docs/implemented/v0_2_0/m_r3_pattern_matching.md](design_docs/implemented/v0_2_0/m_r3_pattern_matching.md) - Pattern matching foundation

**Planned (check for overlap):**
- (none with overlap)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/parser/parser_expr.go:583-613` - Existing `parseConsExpression` (S-CONS sugar)
- `internal/builtins/list.go:35-69` - Existing `::` builtin registration
- `internal/elaborate/expr_calls.go:12-88` - `normalizeFuncCall` (needs the fix)
- `internal/elaborate/patterns.go:105` - `::` handling in pattern elaboration (for reference)
- ai-coding-lang-bench - Benchmark that discovered the gap

## Future Work

- First-class `::` as a function value for partial application
- SMT encoding of cons for contract verification (`ensures result == x :: xs`)
- Tail-call optimization for cons-based recursive functions (accumulator transform)

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30 (incorporated review feedback: cost semantics, operator identity, expression context coverage, associativity guarantee)
