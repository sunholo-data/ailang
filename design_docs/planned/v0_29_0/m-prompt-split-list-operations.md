# Prompt Gap: Split Returns List, Not String

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Low)
**Estimated**: 0.5 day
**Dependencies**: None

## Problem Statement

Models solving `csv_to_json_converter` (and similar text-processing tasks) pass the result of `split()` directly to functions expecting a `string`. The type unification error is:

```
type unification failed: list[string] vs string
```

Root cause: in many languages (`Python`, `JavaScript`, `Ruby`), `split` and `join` are methods on the string object and the split→transform→join pipeline is idiomatic. Models know the pipeline conceptually but misremember AILANG's type boundary: `split(str, delim)` returns `list[string]`, not a cursor or lazy string.

**Current State:**
- `split` return type is `list[string]` — correct and unambiguous in the type system
- Teaching prompt shows `split` in isolation but does not show the complete `split → map/filter → join` pipeline
- Models correctly recall they need to call `join` eventually but attempt to pass the `list[string]` through a `string`-typed intermediate, producing a unification error before reaching `join`
- Frequency: observed on every text-normalisation/conversion benchmark attempted with models below ~70B parameters

**Impact:**
- AI models producing AILANG solutions — type error on the first plausible approach
- High frequency: any task involving CSV, log parsing, or string normalization hits this pattern
- Effort to fix per-task is low but cumulative across the benchmark suite

## Goals

**Primary Goal:** Eliminate the `list[string] vs string` unification error on split-based tasks by making the full `split → map/filter → join` pipeline explicit in the teaching prompt.

**Success Metrics:**
- `csv_to_json_converter` benchmark passes on Qwen 3.5 and equivalent models after prompt update
- Teaching prompt contains at least one end-to-end `split → map → join` example
- The return type annotation for `split` is stated inline in the example, not inferred from context

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Show full pipeline in one example, not split across multiple snippets | Fragmented examples allow models to miss the type connection | agent | implementation | low |
| Annotate return type explicitly in example (`split(...) : list[string]`) | Inline annotation removes ambiguity for models that skip prose | agent | implementation | low |
| Prompt-only fix, no new stdlib functions | Adding a `splitjoin` convenience would paper over the type gap | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Confirm `split(str, delim) -> list[string]` is the only signature (no overload returning a string)
- [x] Confirm `join(list[string], delim) -> string` is the standard inverse and is already documented
- [ ] Confirm which prompt file is the canonical location for string operation examples

## Deferred Decisions

- Whether to add a `string-operations` section to the prompt or extend an existing one — agent may choose
- Whether to include a `filter` step in the example or keep it minimal (just `split → map → join`) — agent may choose based on example clarity

## Solution Design

### Overview

The fix is entirely in the teaching/system prompt. A single canonical pipeline example — showing `split`, `map`, and `join` chained with explicit intermediate types — will close the gap. No compiler or stdlib changes.

### Architecture

Only `prompts/` files are affected. The example should appear in or near the existing string-operations section of the prompt.

**Components:**
1. **Type annotation comment** — inline `-- : list[string]` comment on the `split` call result
2. **Full pipeline example** — `split → map → join` in a single readable let-chain
3. **Common mistake callout** — one-line note that `split` does NOT return a string

### Implementation Plan

**Phase 1: Locate and update prompt** (~2 hours)
- [ ] Find the string-operations section in `prompts/` (check output of `ailang prompt`)
- [ ] Add the pipeline example with inline type annotation
- [ ] Add explicit note: "`split` returns `list[string]`, not `string`"

**Phase 2: Verify** (~1 hour)
- [ ] Run `ailang prompt` and confirm new content appears
- [ ] Verify example compiles (`make run` or equivalent)
- [ ] Add `examples/split_pipeline.ail` if no existing example covers this

### Files to Modify/Create

**Modified files:**
- `prompts/<string-section>.md` or equivalent (+12/−0 LOC) — add pipeline example and note

**New files (optional):**
- `examples/split_pipeline.ail` (~20 LOC) — runnable split→map→join example

## Examples

### Common mistake — treating split result as string

```ailang
-- WRONG: split returns list[string], not string
let normalize = fun row ->
  let parts = split(row, ",") in
  trim(parts)  -- ERROR: trim expects string, got list[string]
```

Error:
```
type unification failed: list[string] vs string
```

### Correct pattern — split → map → join pipeline

```ailang
-- CORRECT: process each element, then rejoin
let normalize_csv_row = fun row ->
  let parts : list[string] = split(row, ",") in
  let trimmed = map(parts, fun s -> trim(s)) in
  join(trimmed, ",")

-- Step by step:
-- split("a , b , c", ",")  =>  ["a ", " b ", " c"]
-- map(_, trim)              =>  ["a", "b", "c"]
-- join(_, ",")              =>  "a,b,c"
```

### With filter (drop empty fields)

```ailang
let clean_fields = fun row ->
  let parts : list[string] = split(row, ",") in
  let non_empty = filter(parts, fun s -> length(s) > 0) in
  map(non_empty, fun s -> trim(s))
-- returns list[string], not string — use join if a string is needed
```

## Success Criteria

- [ ] Teaching prompt states explicitly that `split` returns `list[string]`
- [ ] Teaching prompt shows a complete `split → map → join` pipeline with inline type annotation
- [ ] `csv_to_json_converter` benchmark passes on affected models after prompt update
- [ ] All tests passing (`make test`)
- [ ] Example file compiles cleanly (`make verify-examples`)

## Testing Strategy

**Manual testing:**
- Re-run `csv_to_json_converter` benchmark with updated prompt
- Confirm `list[string] vs string` unification errors are gone

**Regression:**
- Prompt-only change; existing examples unaffected

## Non-Goals

- **Not** adding a `splitjoin` convenience builtin
- **Not** changing `split`'s return type
- **Not** covering all list-vs-string confusion patterns — only the `split` case specifically

## Timeline

**0.5 day (4 hours):**
- Locate and edit prompt (~2h)
- Write/verify example (~1h)
- Run targeted benchmarks to confirm fix (~1h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Multiple prompt files; update lands in non-canonical one | Low | Run `ailang prompt` and verify the section appears in output |
| Example uses a builtin (`trim`, `join`) that has a different signature than expected | Low | Test with `make run` before committing |

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
| A7: Machines First | +1 | Reduces model error rate on a high-frequency pattern |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Pipeline example reinforces composable list operations |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (model) success rate

## References

- **Evidence**: `eval_results/` — Qwen 3.5 35B-A3B core tier run 2026-06-02, task `csv_to_json_converter`; error: `type unification failed: list[string] vs string`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Audit all other stdlib functions that return `list[T]` for similar prompt gaps (`lines`, `keys`, etc.)
- Consider a prompt section specifically for "functions that return lists" to group these in one teachable unit
