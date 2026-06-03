# M-TYPE-CONSTRAINTS: Type parameter constraints (`[a: Ord]`) — explicit-comparator workaround

> **📊 RECENT-VERIFIED: only 1% of recent compile failures (3/230). Lower than first thought — stale-inflated by older runs.** (verified 2026-06-03 against Apr-Jun 2026 data only — not all-time aggregate.)

**Status**: Planned
**Target**: v0.24.0
**Priority**: P3 (Low) — see recent-data banner above
**Estimated**: 0.5 day (prompt only) + separate large feature doc if implementing syntax
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification change (prompt-only Phase 1) |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Removes a typeclass-syntax compile-failure pattern for AI |
| A8: Minimal Syntax | 0 | No new syntax (explicit-comparator uses existing syntax) |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | Explicit comparators compose better than implicit dispatch |
| A11: Structured Failure | 0 | No error-handling change |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Proceed** (no −1 on A1/A3/A4/A7)

---

## Problem Statement

When models need a generic function that works on any orderable/comparable type, they write type constraints using familiar syntax:

- Haskell: `sort :: Ord a => [a] -> [a]`
- Rust: `fn top3<T: Ord>(items: Vec<T>) -> Vec<T>`
- TypeScript: `function top3<T extends Comparable>(items: T[]): T[]`

In AILANG, models write:
```ailang
-- ❌ What models attempt (PARSE ERROR):
func top3[a: Ord](xs: list[a]) -> list[a] = ...
```

AILANG type parameters use `[T]` but do **not** support constraint syntax. The `[a: Ord]` form produces:
```
PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:14:12: expected ], got :
Suggestion: Add ']' to close type parameters
```

**Evidence from eval (June 2026):**
- `polymorphic_ord_defaulting` fails in **5 frontier models** (40% compile_error)
- Affects every model family (Claude, GPT-5.2-Codex, Gemini) — universal false assumption

**Concrete failing code** (from `gemini-3-flash`, April 2026):
```ailang
-- ❌ PARSE ERROR:
func top3[a: Ord](xs: list[a]) -> list[a] =
  take(3, sortBy(\a. \b. compare(a, b), xs))
```

---

## Goals

**Primary goal:** Models know how to write polymorphic ordered functions in AILANG using the explicit-comparator pattern, without type constraint syntax.

**Success metrics:**
- `polymorphic_ord_defaulting` compile_error rate drops from 5/5 to ≤1/5 in frontier models
- No model generates `[a: Ord]` or `[T: Comparable]` syntax after prompt update

---

## Solution Design

### Phase 1: Prompt — the "explicit comparator" pattern

AILANG's idiomatic approach is to **pass the comparator as an explicit function parameter**. This is more explicit (A1), more type-safe (A4), and expressible today:

```ailang
-- ❌ WRONG: Type constraints not supported
func top3[a: Ord](xs: list[a]) -> list[a] =
  take(3, sortBy(\x. \y. compare(x, y), xs))

-- ✅ CORRECT: Concrete type + explicit comparator
pure func top3Ints(xs: list[int]) -> list[int] =
  take(3, sortBy(\a. \b. a - b, xs))

-- ✅ CORRECT: Pass cmp as a parameter for flexibility
pure func top3With(xs: list[int], cmp: (int, int) -> int) -> list[int] =
  take(3, sortBy(cmp, xs))

-- Usage:
top3With(myList, \a. \b. a - b)   -- ascending
top3With(myList, \a. \b. b - a)   -- descending
```

**Teaching prompt addition** (under "Common Mistakes"):
```
| `func f[a: Ord](xs: list[a])`  | NOT SUPPORTED — no type constraints. Use concrete types |
|                                  | or pass comparator explicitly: `func f(xs: list[int], |
|                                  | cmp: (int, int) -> int)` |
```

### Phase 2 (Future, separate doc): Implement type constraints

A multi-sprint effort requiring typeclass declaration syntax, constraint checking in elaboration, and interaction with row polymorphism. Explicit-comparator is not a compromise — for an effects-aware language it may be the right permanent design, since comparators can themselves have effects.

---

## Files to Modify (Phase 1 only)

| File | Change | LOC |
|---|---|---|
| `prompts/v0.17.0.md` | Add constraint prohibition + explicit-comparator example | +12 |

---

## Success Criteria

- [ ] Prompt v0.17.0+ explicitly prohibits `[a: Ord]` / `[T: Comparable]` syntax
- [ ] At least one worked sort-with-comparator example in the prompt
- [ ] `polymorphic_ord_defaulting` compile_error rate ≤1/5 frontier models on next eval run

---

## Conflict Surface

**Phase 1 (prompt only):** No compiler changes — no conflict surface.

---

## Related Documents

- `design_docs/planned/v0_24_0/m-import-alias.md` — companion P1 prompt-gap fix
- `design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md` — similar FP-language assumption pattern
