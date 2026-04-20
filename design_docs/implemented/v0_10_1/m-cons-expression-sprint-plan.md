# Sprint Plan: Cons (`::`) as Expression Operator

## Summary
Enable `::` (cons) as an expression operator by adding a single elaborator special-case that routes `::` function calls to the existing `std/list` cons builtin. This restores pattern/expression symmetry and eliminates AI agent token waste on workarounds.

**Duration:** 1 session (~2-3 hours)
**Dependencies:** None (parser, builtin, and evaluator all exist)
**Risk Level:** Low
**Design Doc:** `design_docs/planned/v0_10_0/m-cons-expression.md`

## Current Status Analysis

### Completed Recently
- v0.10.0 released with bitwise operators, charCode, exit(), serve-api improvements
- Recent velocity: ~200-400 LOC/day across multiple features

### What Already Works
- Parser: `parseConsExpression` desugars `x :: xs` to `FuncCall{"::", [x, xs]}` (right-associative)
- Builtin: `::` registered in `std/list` module with `(a, [a]) -> [a]` type
- Evaluator: `listConsImpl` handles prepend correctly
- Patterns: `::` works in pattern matching

### What's Broken
- Elaborator: `normalizeFuncCall` doesn't recognize `::` as a builtin, falls through to `core.Var{"::"}`
- Result: unresolved variable error at type check or eval time

## Proposed Milestones

### Milestone 1: M1_ELABORATOR_FIX — Elaborator Special Case
**Goal:** Make `::` resolve to the `std/list` cons builtin in expression position
**Estimated:** ~15 LOC implementation
**Duration:** 30 min

**Tasks:**
- Add `::` check in `normalizeFuncCall` before the general constructor check
- Emit `core.App{Func: core.VarGlobal{Module: "std/list", Name: "::"}, Args: [...]}`
- Normalize both args to atomic (reuse existing code path)
- Run `make test` to verify no regressions

**Acceptance Criteria:**
- `::` in `normalizeFuncCall` emits `VarGlobal` to `std/list` builtin
- All existing tests pass (`make test`)
- `make lint` passes

### Milestone 2: M2_TESTS — Comprehensive Test Coverage
**Goal:** Add tests covering all expression contexts, associativity, type errors, and REPL parity
**Estimated:** ~80 LOC tests
**Duration:** 45 min

**Tasks:**
- Add elaborator unit test in `internal/elaborate/` for `::` FuncCall → VarGlobal
- Add end-to-end `.ail` test: basic cons, chained cons, cons in lambdas/match/if/let/args
- Add type error test: `1 :: 2` (second arg not a list)
- Verify REPL and file pipeline both work
- Run `make test` and `make verify-examples`

**Acceptance Criteria:**
- `1 :: [2, 3]` evaluates to `[1, 2, 3]`
- `1 :: 2 :: 3 :: []` evaluates to `[1, 2, 3]` (right-associative)
- `::` works in all expression contexts (lambda, let, match, if, function arg)
- `1 :: 2` produces a type error
- REPL and file pipeline behave identically
- Existing pattern `::` behavior unchanged

### Milestone 3: M3_DOCS — Documentation and Examples
**Goal:** Add example file, update CHANGELOG, update teaching prompts
**Estimated:** ~40 LOC docs/examples
**Duration:** 30 min

**Tasks:**
- Create `examples/cons_expression.ail` with comprehensive examples
- Update CHANGELOG.md with the feature
- Update teaching prompts in `prompts/` if they mention list construction
- Run `make verify-examples`

**Acceptance Criteria:**
- `examples/cons_expression.ail` exists and passes verification
- CHANGELOG.md updated
- `make verify-examples` passes

## Success Metrics
- All tests passing: `make test`
- Examples passing: `make verify-examples`
- Linting clean: `make lint`
- Documentation: CHANGELOG.md, example file
- Total LOC: ~135 (15 impl + 80 tests + 40 docs)

## Risks
- `::` identifier conflict with type annotation DCOLON — **Mitigated**: parser already separates these tokens
- REPL vs file pipeline path divergence — **Mitigated**: builtin registered in `init()`, available everywhere
- Monomorphization of polymorphic `::` — **Mitigated**: same mechanism as `++` which already works

## Open Questions
- None — all design decisions frozen in design doc
