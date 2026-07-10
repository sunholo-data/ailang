# M-PROMPT-NESTED-ADT-PATTERNS: Clarify Nested ADT Match Support and Prevent Over-Verbose Balance Functions

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Medium)
**Estimated**: 1 day (prompt update + eval verification)
**Dependencies**: None

## Problem Statement

Models generating solutions for `red_black_tree` (and similar tree-with-rebalancing tasks) produce
files that are truncated mid-function with unexpected EOF errors. The diagnosis is:

1. **Root cause is token-limit truncation**, not unsupported syntax. AILANG does support deep nested
   ADT pattern matching (confirmed by `ailang check` tests). The match construct
   `B(Black, Node(Red, Node(Red, a, x, b), y, c2), z, d)` is valid AILANG.
2. **Secondary root cause**: models are uncertain whether AILANG supports nested ADT patterns, so
   they add extra helper functions, intermediate booleans, and guard checks to avoid nesting. This
   makes their `balance` function expand to 80-120 lines instead of the idiomatic 30-40 lines,
   consuming the token budget.

**Current State:**
- `red_black_tree` benchmark shows unexpected-EOF compile errors across multiple models.
- Measured generated `balance` function length: 80-120 lines (verbose) vs. ~35 lines (idiomatic).
- No teaching prompt states that AILANG supports nested ADT patterns — models conservatively
  avoid them.
- The pattern `match expr with | Ctor(Ctor2(a, b), c) -> ...` is syntactically valid but never
  demonstrated in the teaching prompt.

**Impact:**
- **Affected benchmarks**: `red_black_tree`, any benchmark using recursive ADT rebalancing or
  deep structural match.
- **Severity**: Hard compile failure from truncated output (unexpected EOF).
- **Frequency**: Occurs reliably when models generate verbose workarounds for assumed limitations.

## Goals

**Primary Goal:** Add examples to the teaching prompt confirming AILANG supports deep nested ADT
pattern matching, so models use the concise idiomatic style and avoid token-limit truncation.

**Success Metrics:**
- Zero unexpected-EOF failures on `red_black_tree` in the next eval rotation.
- Model-generated `balance` functions are ≤50 lines (down from observed 80-120).
- At least one nested ADT match example added to the teaching prompt showing 2-level nesting.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Whether to show 2-level or 3-level nesting in example | Too shallow = model still adds guards; too deep = example hard to read | agent | implementation | low |
| Whether to combine with m-prompt-concise-recursive-solutions fix | Both address token truncation; may share prompt real estate | agent | implementation | low |
| Whether to add a note that `ailang check` accepts nested patterns | Removes ambiguity for future models | agent | implementation | low |

### Design Freeze

No high-change-cost decisions — all are low. No design freeze items required.

## Solution Design

### Overview

Add a "Nested ADT Pattern Matching" section to the AILANG teaching prompt. The section should:

1. Explicitly state that AILANG supports deep nested constructor patterns in `match` expressions.
2. Show a concrete 2-level nested example (e.g., tree rotation or red-black balance case).
3. Show the before/after: verbose helper-function approach vs. concise nested match.

This is a prompt-only change. No compiler modifications are needed.

### Implementation Plan

**Phase 1: Write and verify examples** (~2 hours)
- [ ] Write a 2-level nested ADT match example in AILANG (tree rotation style)
- [ ] Write the verbose helper-function version of the same code
- [ ] Verify both examples compile with `ailang check`
- [ ] Verify both produce correct output with `ailang run`

**Phase 2: Insert into teaching prompt** (~1 hour)
- [ ] Locate all teaching prompt / eval system prompt injection files
- [ ] Add "Nested ADT Patterns" section with before/after and explicit confirmation note
- [ ] Confirm the section fits within prompt length budget

**Phase 3: Eval verification** (~3 hours)
- [ ] Re-run `red_black_tree` benchmark after prompt update
- [ ] Inspect generated code: confirm nested match usage and line counts
- [ ] Confirm no unexpected-EOF failures

### Files to Modify/Create

**Modified files:**
- Teaching/eval prompt file(s) (paths TBD after Phase 2 survey) (+35 LOC)

## Examples

### Verbose approach — triggers token truncation

```ailang
-- ❌ VERBOSE: Model avoids nested match, uses helper functions instead
-- Result: ~90 lines for balance, hits token limit

type Color = Red | Black
type Tree[a] = Leaf | Node(Color, Tree[a], a, Tree[a])

pure func is_double_red_left(t: Tree[int]) -> bool =
  match t with
  | Node(Red, Node(Red, _, _, _), _, _) -> true
  | _                                   -> false
  end

pure func is_double_red_left_right(t: Tree[int]) -> bool =
  match t with
  | Node(Red, Node(_, _, _, Node(Red, _, _, _)), _, _) -> true
  | _                                                  -> false
  end

-- ... 60 more lines of helpers before balance is even defined ...
```

### Concise nested match — idiomatic AILANG (~35 lines)

```ailang
-- ✅ CONCISE: Nested ADT patterns work in AILANG — use them directly

type Color = Red | Black
type Tree[a] = Leaf | Node(Color, Tree[a], a, Tree[a])

pure func balance(t: Tree[int]) -> Tree[int] =
  match t with
  | Node(Black, Node(Red, Node(Red, a, x, b), y, c), z, d) ->
    Node(Red, Node(Black, a, x, b), y, Node(Black, c, z, d))
  | Node(Black, Node(Red, a, x, Node(Red, b, y, c)), z, d) ->
    Node(Red, Node(Black, a, x, b), y, Node(Black, c, z, d))
  | Node(Black, a, x, Node(Red, Node(Red, b, y, c), z, d)) ->
    Node(Red, Node(Black, a, x, b), y, Node(Black, c, z, d))
  | Node(Black, a, x, Node(Red, b, y, Node(Red, c, z, d))) ->
    Node(Red, Node(Black, a, x, b), y, Node(Black, c, z, d))
  | t -> t
  end
```

### Prompt note to add

```
NOTE: AILANG supports deep nested ADT constructor patterns in match expressions.
You can match `Ctor1(Ctor2(a, b), c)` directly without decomposing into helpers.
Prefer nested patterns for tree operations (balance, rotate, rebalance) — they
produce concise 30-40 line implementations.
```

## Success Criteria

- [ ] Teaching prompt explicitly states that AILANG supports nested ADT patterns in `match`
- [ ] At least one 2-level nested pattern match example is in the prompt
- [ ] Before/after comparison shows the verbose vs. concise approach
- [ ] All examples compile with `ailang check`
- [ ] `red_black_tree` benchmark: zero unexpected-EOF failures in next eval run
- [ ] Model-generated `balance` function ≤50 lines in post-fix eval
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Compile verification:**
- `ailang check` on the nested match example — must pass with no errors
- `ailang run` on the balance example with a sample tree — must produce correct output

**Eval verification:**
- Re-run `red_black_tree` after prompt update
- Inspect generated solutions: confirm nested patterns used, count lines in `balance`

**Manual testing:**
- Read the updated prompt section; confirm the note is unambiguous

## Deferred Decisions

- Whether to use a red-black tree example or a simpler binary search tree example for the prompt —
  agent may choose (simpler may be clearer, but RBT is the failing benchmark)
- Whether this section should be merged with the "Concise Recursive Patterns" section from
  `m-prompt-concise-recursive-solutions.md` — agent may decide based on prompt length
- Exact wording of the confirmation note — agent may choose

## Non-Goals

- Changing how AILANG parses nested ADT patterns — no compiler change needed
- Adding a new stdlib tree type — out of scope
- Fixing other tree-related benchmarks beyond red_black_tree — deferred

## Timeline

- Day 1: Write examples, verify compilation, locate prompt files, insert section, run eval

**Total: ~1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Model still generates verbose helpers despite prompt note | Medium | Add explicit statement "do NOT decompose into is_double_red helpers" |
| Nested match example is complex — model misreads it | Low | Use a simpler 2-level example first; red-black as a comment |
| Prompt section too long — crowds out other rules | Low | Keep example ≤20 lines; use a single before/after pair |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Nested match reduces code surface; fewer paths to verify |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Directly reduces token usage in model-generated tree algorithms |
| A8: Minimal Syntax | +1 | Teaches models to use existing syntax; no new syntax needed |
| A9: Cost Visibility | +1 | Shorter solutions lower per-eval token cost (measurable) |
| A10: Composability | +1 | Nested match patterns compose with all existing ADT definitions |
| A11: Structured Failure | 0 | No change to error handling |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Teaching nested patterns reduces model verbosity directly

## Related Documents

- `design_docs/planned/v0_24_0/m-prompt-concise-recursive-solutions.md` — companion doc (same
  root cause: token-limit truncation from verbose solutions)
- `design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md` — related match-expression prompt fix

## References

- **Failing benchmark**: `benchmarks/red_black_tree/`
- **Verification**: `ailang check` confirms nested ADT patterns are supported (confirmed in tests)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- If token truncation persists after this fix, consider adding a "solution length hint" to the
  eval system prompt: "your solution should be under 80 lines for tree benchmarks"
- A broader "AILANG pattern matching guide" example file in `examples/` covering all match forms
  (wildcard, nested, guard, list destructure) would benefit all future model evaluations

---

**Document created**: 2026-06-03
**Last updated**: 2026-06-03
