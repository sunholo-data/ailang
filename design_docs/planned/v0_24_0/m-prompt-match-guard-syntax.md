# Prompt Gap: Match Expression Guard Syntax

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Low-Medium)
**Estimated**: 0.5 day
**Dependencies**: None

## Problem Statement

Models generating AILANG solutions for benchmarks (`config_file_parser`, `merge_sort`) produce match expressions with guard patterns that the parser cannot handle. The errors are:

```
PAR_UNEXPECTED_TOKEN: expected => got IDENT
PAR_UNEXPECTED_TOKEN: expected } got IDENT
```

Both failures occur around match arms. Root cause: models trained on OCaml, Haskell, F#, and Rust naturally reach for guard clauses (`when`, `if`) after a match pattern because they are idiomatic in those languages.

**Current State:**
- AILANG match arms use the form `Pattern => expr` with no guard position
- Teaching prompts do not explicitly state that guard clauses are unsupported
- Models infer AILANG is ML-like and assume guard syntax exists
- Benchmark failure rate on tasks requiring conditional branching inside match: observed 2/17 failures on the Qwen 3.5 35B-A3B smoke/core run (2026-06-02)

**Impact:**
- AI models (primary consumers of AILANG teaching prompts) — hard failure, not a hint
- Benchmark tasks requiring multi-condition dispatch on a single value are disproportionately affected
- Cascading: model writes guard, parser errors, model retries with another unsupported form

## Goals

**Primary Goal:** Eliminate PAR_UNEXPECTED_TOKEN failures from guard-pattern attempts by making the constraint explicit and the workaround obvious in the teaching prompt.

**Success Metrics:**
- Zero guard-pattern parse errors on `config_file_parser` and `merge_sort` benchmark tasks after prompt update
- Teaching prompt includes one canonical before/after example of guard → nested-if rewrite
- Prompt clearly lists what match arm syntax IS supported (exact grammar)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Workaround is nested `if-then-else` inside arm body | Shapes how models rewrite; wrong advice causes secondary errors | human | design | med |
| No language change — prompt-only fix | Adding guards is a real feature request; this doc is prompt gap only | human | design | low |
| Show the pattern explicitly, not just a prohibition | Prohibition alone does not help models recover | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Confirm guards are not supported at any level (top-level match and nested match) — checked: parser does not implement guard productions
- [x] Confirm the idiomatic workaround is `if-then-else` inside the arm body, not a separate helper function
- [ ] Decide whether to add a dedicated "Unsupported Syntax" section to the teaching prompt or extend an existing example

## Deferred Decisions

- Exact placement within the prompt file — agent may choose (inline example vs. separate section)
- Whether to also add a compiler error hint (`did you mean if-then-else?`) — deferred to a future language-quality doc

## Solution Design

### Overview

The fix is entirely in the teaching/system prompt for AILANG. No parser or compiler changes. The prompt needs two additions:

1. An explicit statement that match arms do not support guard clauses (`when`, `if`, `where`, or any conditional suffix after the pattern).
2. A concrete before/after example showing how to rewrite a guard into a nested `if-then-else` expression inside the arm body.

### Architecture

Only `prompts/` files are changed. The relevant prompt is the one injected into model context when AILANG code generation is requested (typically the system prompt or the coding assistant prompt).

**Components:**
1. **Prohibition statement** — clear one-line statement that guards are not valid match arm syntax
2. **Rewrite example** — minimal two-arm match showing the guard pattern (wrong) and the nested-if equivalent (correct)
3. **Supported syntax summary** — explicit grammar fragment for match arms so models do not need to infer

### Implementation Plan

**Phase 1: Locate and update prompt** (~2 hours)
- [ ] Find the match-expression section in `prompts/` (check `ailang prompt` output for the relevant section)
- [ ] Add prohibition statement immediately after match syntax introduction
- [ ] Add before/after rewrite example

**Phase 2: Verify** (~1 hour)
- [ ] Run `ailang prompt` and confirm the new text appears
- [ ] Manually check the example compiles correctly (`make run FILE=examples/match_guard_workaround.ail` or equivalent)
- [ ] Update `examples/` with the workaround pattern if no existing example covers it

### Files to Modify/Create

**Modified files:**
- `prompts/<match-section>.md` or equivalent (+10/−0 LOC) — add guard prohibition + rewrite example

**New files (optional):**
- `examples/match_guard_workaround.ail` (~15 LOC) — runnable workaround example (if no existing example covers this)

## Examples

### Guard Pattern — NOT supported (models generate this)

```ailang
-- WRONG: parser rejects guard clauses
let classify = fun n ->
  match n with
  | x when x < 0 => "negative"
  | x when x == 0 => "zero"
  | _ => "positive"
```

Parser output:
```
PAR_UNEXPECTED_TOKEN: expected => got IDENT (when)
```

### Workaround — nested if-then-else inside arm body

```ailang
-- CORRECT: guard logic moves inside the arm expression
let classify = fun n ->
  match n with
  | x => if x < 0 then "negative"
         else if x == 0 then "zero"
         else "positive"
```

Or, when patterns are disjoint constructors with a conditional on the payload:

```ailang
-- CORRECT: combine constructor match + conditional
let describe = fun opt ->
  match opt with
  | None => "absent"
  | Some(v) => if v > 100 then "large" else "small"
```

## Success Criteria

- [ ] Teaching prompt explicitly states guards are not supported in match arms
- [ ] Teaching prompt shows the `if-then-else`-inside-arm workaround with a runnable example
- [ ] `config_file_parser` benchmark passes on Qwen 3.5 and equivalent-tier models after prompt update
- [ ] `merge_sort` benchmark passes on affected models after prompt update
- [ ] All tests passing (`make test`)
- [ ] Example file compiles cleanly (`make verify-examples`)

## Testing Strategy

**Manual testing:**
- Re-run `config_file_parser` and `merge_sort` benchmarks with updated prompt
- Confirm PAR_UNEXPECTED_TOKEN errors are gone from stderr

**Regression:**
- No existing match examples should be affected (prompt-only change)

## Non-Goals

- **Not** adding guard clause support to the parser/compiler (separate feature request)
- **Not** adding a compiler hint or error recovery for guard attempts (separate quality doc)
- **Not** covering all ML-ism mismatches — only the guard pattern specifically

## Timeline

**0.5 day (4 hours):**
- Locate and edit prompt (~2h)
- Write/verify example (~1h)
- Run targeted benchmarks to confirm fix (~1h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Prompt location unclear — multiple prompt files | Low | Run `ailang prompt` and grep for "match" to find canonical location |
| Example doesn't compile due to unrelated syntax issue | Low | Test with `make run` before committing |
| Guard prohibition discourages correct use of if-then-else | Low | Frame as "use if-then-else inside the arm" not just "guards are banned" |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language change |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | 0 | No impact |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No impact |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Reduces model error rate; machines are the primary prompt consumers |
| A8: Minimal Syntax | +1 | Affirms existing minimal syntax (no guards); teaches the minimal equivalent |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | 0 | No impact |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (model) success rate

## References

- **Evidence**: `eval_results/` — Qwen 3.5 35B-A3B core tier run 2026-06-02, tasks `config_file_parser` and `merge_sort`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Add guard clause syntax to the parser (separate feature, higher effort) — would eliminate the need for this workaround entirely
- Add a compiler error recovery hint: when `when` is seen after a pattern, emit `did you mean if-then-else inside the arm?`
