# Prompt Gap: Option/None Requires Explicit Import

> **📊 RECENT-VERIFIED: 6% of recent compile failures (15/230). Current.** (verified 2026-06-03 against Apr-Jun 2026 data only — not all-time aggregate.)

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Low)
**Estimated**: 0.5 day
**Dependencies**: None

## Problem Statement

Models solving `graph_bfs` (and any task involving optional values) emit `undefined variable: None`. The error occurs because `None` is NOT a keyword — it is a value constructor exported by `std/option` and must be imported explicitly.

```
undefined variable: None
```

Root cause: `None` and `Some(x)` are universal idioms across Python, OCaml, Haskell, Rust, and Swift. Models write them fluently but treat them as ambient (always in scope). AILANG requires an explicit import from `std/option`. The teaching prompt mentions `Option` as a type but does not show the required import line with the value constructors.

**Current State:**
- `None` and `Some(x)` ARE the correct AILANG constructors — the semantics are right
- The fix is a single import line: `import std/option (Option, None, Some)`
- Teaching prompt references `Option` type but does not show the import that brings `None` and `Some` into scope
- Models operating from training memory assume `None` is a keyword and omit the import
- Observed in `graph_bfs` benchmark (Qwen 3.5 35B-A3B, 2026-06-02); likely affects any task using optional values

**Impact:**
- AI models — hard failure on first use of any optional value
- High value to fix: the change is a single import line in the prompt, but the error is a complete blocker
- Affects a broad class of tasks (graph algorithms, lookup functions, parsers with optional results)

## Goals

**Primary Goal:** Eliminate `undefined variable: None` errors by making the `std/option` import explicit and prominent in every teaching prompt example that uses optional values.

**Success Metrics:**
- `graph_bfs` benchmark passes on Qwen 3.5 and equivalent models after prompt update
- Every `Option`-using example in the teaching prompt includes the `import std/option (Option, None, Some)` line
- Prompt contains a callout: "`None` and `Some` are constructors, not keywords — import them"

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Import must list `None` and `Some` explicitly, not `import std/option (..)` wildcard | Wildcard imports hide what's in scope from models; explicit imports are self-documenting | human | design | low |
| No language change — `None` stays a constructor, not a keyword | Making `None` a keyword would break module-system semantics; out of scope | human | design | low |
| Callout placed before first Option example, not buried at end | Models read prompts top-to-bottom; placement determines whether they see it | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Confirm `None` is NOT a keyword and will not become one in v0.24.0
- [x] Confirm the canonical import form is `import std/option (Option, None, Some)`
- [ ] Confirm whether `std/option` is auto-imported in any execution mode (REPL vs file) — if REPL auto-imports it, the callout should clarify file-mode requires explicit import

## Deferred Decisions

- Whether to also mention `Option.map`, `Option.unwrap_or`, etc. in the same section — agent may choose (keep minimal for this doc, additive later)
- Exact callout phrasing — agent may choose, provided it uses the word "constructor" or "not a keyword"

## Solution Design

### Overview

The fix is in the teaching/system prompt only. Two additions are needed:

1. A short callout stating that `None` and `Some` are constructors from `std/option`, not keywords.
2. Every Option-related example in the prompt must include the import line at the top.

No compiler or stdlib changes.

### Architecture

Only `prompts/` files are affected.

**Components:**
1. **Callout block** — placed before the first Option example: `None` and `Some` are not keywords; they must be imported
2. **Import-annotated examples** — each existing Option example gets the import prepended

### Implementation Plan

**Phase 1: Locate and update prompt** (~2 hours)
- [ ] Find all Option/None/Some examples in `prompts/` (grep for `None\|Some\|Option` in prompts/)
- [ ] Prepend `import std/option (Option, None, Some)` to each example that uses these constructors
- [ ] Add callout block before the first Option example

**Phase 2: Verify** (~1 hour)
- [ ] Run `ailang prompt` and confirm callout and import lines appear
- [ ] Verify examples compile with `make run` or equivalent
- [ ] Check `examples/` — ensure any Option examples there also have the import (for consistency)

### Files to Modify/Create

**Modified files:**
- `prompts/<option-section>.md` or equivalent (+8/−0 LOC) — add callout + import lines to Option examples

**Possibly also modified:**
- `examples/*.ail` files using `None`/`Some` without imports (audit with `grep -l "None\|Some(" examples/`) — add missing imports if found

## Examples

### Common mistake — constructors used without import

```ailang
-- WRONG: None and Some are not in scope without importing std/option
let find_first = fun lst pred ->
  match lst with
  | [] => None          -- ERROR: undefined variable: None
  | x :: rest => if pred(x) then Some(x) else find_first(rest, pred)
```

Error:
```
undefined variable: None
```

### Correct pattern — explicit import brings constructors into scope

```ailang
-- CORRECT: import std/option to get Option type and its constructors
import std/option (Option, None, Some)

let find_first : forall a. list[a] -> (a -> bool) -> Option[a]
let find_first = fun lst pred ->
  match lst with
  | [] => None
  | x :: rest => if pred(x) then Some(x) else find_first(rest, pred)
```

### Callout for teaching prompt

> **`None` and `Some` are constructors, not keywords.**
> They must be imported from `std/option`:
> ```ailang
> import std/option (Option, None, Some)
> ```
> Without this import, `None` and `Some` are undefined variables.

### BFS example (the failing benchmark pattern)

```ailang
import std/option (Option, None, Some)

-- visited: map[node, bool], queue: list[node]
let bfs_step = fun graph visited queue ->
  match queue with
  | [] => None                          -- no more nodes
  | node :: rest =>
      if map_get(visited, node) then
        bfs_step(graph, visited, rest)  -- already seen
      else
        Some({ node = node, rest = rest })
```

## Success Criteria

- [ ] Teaching prompt explicitly states `None` and `Some` are constructors requiring import
- [ ] Every Option example in the prompt includes `import std/option (Option, None, Some)`
- [ ] `graph_bfs` benchmark passes on affected models after prompt update
- [ ] All tests passing (`make test`)
- [ ] Any `examples/*.ail` files using `None`/`Some` without imports are fixed
- [ ] `make verify-examples` passes

## Testing Strategy

**Manual testing:**
- Re-run `graph_bfs` benchmark with updated prompt
- Confirm `undefined variable: None` errors are gone

**Audit:**
- `grep -rn "None\|Some(" examples/` — verify all examples include the import

**Regression:**
- Prompt-only change with possible example file fixes; no compiler changes

## Non-Goals

- **Not** making `None` a keyword (separate language design question)
- **Not** adding `std/option` to an implicit prelude
- **Not** covering all `std/*` import gaps in one doc — only `std/option` specifically

## Timeline

**0.5 day (4 hours):**
- Locate and audit prompt + examples (~1.5h)
- Edit prompt and examples (~1h)
- Run targeted benchmarks to confirm fix (~1.5h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| REPL auto-imports `std/option`; callout confuses REPL users | Low | Clarify "in `.ail` files" in the callout |
| Other `std/*` constructors have the same gap (e.g., `Result`, `Either`) | Low | Note in Future Work; fix only `std/option` now |
| Example audit finds many files missing import | Med | Fix them all in the same PR — low mechanical effort |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language change |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | 0 | No impact |
| A4: Explicit Authority | +1 | Reinforces explicit import = explicit authority; `None` entering scope only when declared |
| A5: Bounded Verification | 0 | No impact |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +2 | High-value, easy fix; directly unblocks model success on optional-value tasks |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | 0 | No impact |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): Reinforces, not weakens, explicit authority
- [x] A7 (Machines First): Directly improves machine (model) success rate

## References

- **Evidence**: `eval_results/` — Qwen 3.5 35B-A3B core tier run 2026-06-02, task `graph_bfs`; error: `undefined variable: None`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Audit `std/result` (constructors `Ok`, `Err`) for the same import-visibility gap
- Consider a prompt section "Standard library constructors you must import" grouping `Option`, `Result`, and any other commonly-missed modules
- Investigate whether a linter hint (`did you mean to import std/option?` when `None` is undefined) would eliminate the runtime surprise entirely
