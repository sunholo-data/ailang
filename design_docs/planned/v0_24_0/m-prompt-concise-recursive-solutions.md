# M-PROMPT-RUN-LENGTH-PATTERNS: Concise Recursive Solutions to Prevent Token-Limit Truncation

**Status**: Planned (⚠️ ROOT CAUSE CORRECTED — see note below)
**Target**: v0.24.0
**Priority**: P3 (LOWERED — root cause was misdiagnosed)
**Estimated**: 0.5 day (concise examples still useful, but not for the stated reason)
**Dependencies**: m-prompt-string-concat-plusplus (the actual cause)

> **⚠️ CORRECTION (2026-06-03):** This doc originally attributed the EOF/truncation
> failures in `run_length_encode`, `type_unify`, `red_black_tree` to models hitting
> `max_output_tokens`. **That was wrong.** Investigation showed the failing outputs
> were only 161–442 tokens (nowhere near the 8192 limit) — the models were NOT
> truncated by the harness. The real cause: those solutions used **`++` for string
> concatenation** (a type error since v0.13.0), and the parser bailed after the type
> error, producing an EOF-looking error later in the file. The `++` reflex appears in
> **46% of all compile failures** — see `m-prompt-string-concat-plusplus.md` (P0).
> The concise-example content below is still mildly useful (shorter solutions = fewer
> places to misuse `++`), but it is NOT the primary fix. Demoted to P3.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Shorter solutions have smaller, more verifiable proof obligations |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Directly teaches models to generate code within token budget; fewer truncated outputs |
| A8: Minimal Syntax | +1 | Prompt demonstrates concise AILANG idioms — less boilerplate |
| A9: Cost Visibility | +1 | Reduces model output tokens per eval run (measurable cost) |
| A10: Composability | +1 | foldl/recursion patterns compose with existing std functions |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Teaching conciseness benefits machine generation, not just humans

## Problem Statement

Models generating solutions for `run_length_encode` and `type_unify` benchmarks produce files that
are truncated mid-function with unexpected EOF errors. The root cause is model output-token
exhaustion: the model generates a long recursive solution, runs out of tokens, and delivers a
syntactically incomplete file.

**Current State:**
- `run_length_encode` and `type_unify` show `unexpected EOF` compile errors in eval results.
- The `max_output_tokens` budget is 8192, but verbose multi-helper recursive solutions for these
  tasks routinely exceed 200-300 lines — more than the model can produce within the budget.
- The model correctly understands the algorithm; the problem is solution verbosity, not reasoning.
- The teaching prompt has no examples showing AILANG-idiomatic accumulator/fold patterns for these
  classic recursive problems.

**Impact:**
- **Affected benchmarks**: `run_length_encode`, `type_unify`, and any benchmark requiring recursive
  accumulator-style traversal with helper functions.
- **Severity**: Hard compile failure from truncated output (unexpected EOF mid-function-body).
- **Models affected**: Occurs across model families — any model that defaults to a verbose imperative
  recursive style rather than a compact fold/accumulator style.

## Goals

**Primary Goal:** Add concise AILANG-idiomatic recursive examples to the teaching prompt so models
generate compact (~20-line) solutions for accumulator-style problems, staying within the token budget.

**Success Metrics:**
- Zero unexpected-EOF compile failures on `run_length_encode` and `type_unify` in the next eval run.
- Reference solutions demonstrate RLE and accumulator patterns in ≤25 lines each.
- Model-generated solutions for these benchmarks are ≤150 lines (down from observed 250+).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Whether to also raise `max_output_tokens` | Token limit increase costs money; prompt fix is free | human | design | low |
| Which examples to include (RLE, type unifier, or both) | Too many examples wastes prompt space; too few doesn't cover the pattern | agent | implementation | low |
| Where in the prompt to insert examples | Placement affects model attention to the pattern | agent | implementation | low |

### Design Freeze

Before implementation begins:
- [ ] Human confirms: do NOT raise max_output_tokens (prompt fix only)

## Solution Design

### Overview

Add a "Concise Recursive Patterns" section to the AILANG teaching prompt. Show that run-length
encoding and similar accumulator problems are idiomatic in AILANG as a single recursive function
with an accumulator parameter, leveraging pattern matching — not as multiple helpers with verbose
base-case scaffolding.

The goal is not to give models the answer to the benchmark, but to demonstrate the *style* of
concise AILANG recursion so models apply it when generating their own solutions.

### Implementation Plan

**Phase 1: Draft prompt examples** (~2 hours)
- [ ] Write a concise RLE-style accumulator function in AILANG (~20 lines)
- [ ] Write a concise type-unifier style recursive pattern in AILANG (~25 lines)
- [ ] Verify both examples compile with `ailang check`
- [ ] Verify both examples produce correct output with `ailang run`

**Phase 2: Insert into teaching prompt** (~1 hour)
- [ ] Locate all prompt injection files used by the eval harness
- [ ] Add "Concise Recursive Patterns" section with examples and inline commentary
- [ ] Add a note: "AILANG foldl/recursion expresses accumulator patterns in ~20 lines;
  do not use multiple helper functions when one recursive function with an accumulator suffices"

**Phase 3: Eval verification** (~3 hours)
- [ ] Re-run `run_length_encode` and `type_unify` benchmarks
- [ ] Confirm no unexpected-EOF failures
- [ ] Check solution line counts are below 150

### Files to Modify/Create

**Modified files:**
- Teaching/eval prompt file(s) (paths TBD after locating injection points) (+40 LOC)
- Possibly `benchmarks/run_length_encode/reference.ail` (+25 LOC) — add a concise reference solution

## Examples

### RLE — verbose pattern (triggers truncation)

```ailang
-- ❌ TOO VERBOSE — triggers token-limit truncation in eval

pure func rle_encode(xs: list[int]) -> list[(int, int)] =
  rle_helper(xs, [])

pure func rle_helper(xs: list[int], acc: list[(int, int)]) -> list[(int, int)] =
  match xs with
  | [] -> reverse(acc)
  | [x] ->
    match acc with
    | [] -> reverse([(x, 1)])
    | [(hv, hc), ...rest] ->
      if hv == x then
        reverse([(hv, hc + 1), ...rest])
      else
        reverse([(x, 1), (hv, hc), ...rest])
    end
  | [x, ...rest2] ->
    -- ... 40 more lines of verbose matching ...
```

### RLE — concise AILANG idiom (stays within token budget)

```ailang
-- ✅ CONCISE: single recursive function with accumulator (~20 lines)

pure func rle_encode(xs: list[int]) -> list[(int, int)] =
  reverse(rle_go(xs, []))

pure func rle_go(xs: list[int], acc: list[(int, int)]) -> list[(int, int)] =
  match (xs, acc) with
  | ([], _)                      -> acc
  | ([x, ...rest], [])           -> rle_go(rest, [(x, 1)])
  | ([x, ...rest], [(v, c), ..t]) ->
    if x == v
    then rle_go(rest, [(v, c + 1), ..t])
    else rle_go(rest, [(x, 1), (v, c), ..t])
  end
```

### Accumulator pattern — general teaching note

```ailang
-- Idiomatic AILANG recursive accumulator:
-- 1. One function with an explicit acc parameter
-- 2. match on (input, acc) simultaneously
-- 3. reverse at the end if order matters
-- This covers RLE, group-by, run-detection, and similar patterns in ~20 lines.
```

## Success Criteria

- [ ] Teaching prompt contains "Concise Recursive Patterns" section with at least one RLE-style example
- [ ] Both examples compile with `ailang check` (no errors)
- [ ] `run_length_encode` benchmark: zero unexpected-EOF failures in next eval run
- [ ] `type_unify` benchmark: zero unexpected-EOF failures in next eval run
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Compile verification:**
- `ailang check` on each new example in the prompt — must pass

**Eval verification:**
- Re-run `run_length_encode` and `type_unify` after prompt update
- Inspect generated solutions: confirm line count < 150

**Manual testing:**
- Run the RLE example with a sample input, verify output is correct pairs

## Deferred Decisions

- Exact wording of the "concise patterns" callout — agent may choose
- Whether to include a `type_unify` example or rely on the RLE example as sufficient — agent may decide based on prompt length
- File path for prompt injection — agent determines during Phase 2

## Non-Goals

- Raising `max_output_tokens` — not in this doc; prompt fix is the approach
- Teaching all possible accumulator patterns — only RLE-style is in scope here
- Adding new stdlib functions for RLE — out of scope

## Timeline

- Day 1: Draft examples, verify compilation, locate prompt files, insert examples
- Day 2 (morning): Run eval, measure improvement, minor adjustments if needed

**Total: ~1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Concise example is too abstract — model doesn't generalise it | Medium | Include inline comments explaining the "accumulator with reverse" pattern |
| Prompt examples inflate prompt length — reduces model attention to other rules | Low | Keep examples to ≤40 lines total; use comment annotations, not prose paragraphs |
| Model still generates verbose solution for type_unify specifically | Medium | Add a second example targeting unifier-style mutual recursion if RLE alone is insufficient |

## Related Documents

- `design_docs/planned/v0_24_0/m-prompt-nested-adt-patterns.md` — companion doc covering nested ADT match truncation (same token-limit root cause)
- `design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md` — related prompt-gap fix

## References

- **Failing benchmarks**: `benchmarks/run_length_encode/`, `benchmarks/type_unify/`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- If token truncation continues after this fix, consider a "solution length budget" hint in the eval system prompt: "your solution should be under 100 lines"
- Investigate whether `type_unify` needs its own dedicated example or if the RLE accumulator pattern generalises

---

**Document created**: 2026-06-03
**Last updated**: 2026-06-03
