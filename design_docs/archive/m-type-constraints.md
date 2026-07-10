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

AILANG type parameters use `[T]` but the constraint *annotation* `[a: Ord]` is not
written explicitly — it produces:
```
PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:14:12: expected ], got :
Suggestion: Add ']' to close type parameters
```

> **⚠️ CRITICAL CORRECTION (2026-06-03, after reading `docs/references/index.md`):**
> **AILANG HAS typeclasses** — it "implements type class elaboration via dictionary
> passing for overloaded operations" (Wadler/Blott style). The comparison operators
> `>`, `<`, `<=`, `>=`, `==` are already polymorphic via dictionaries. **The constraint
> is INFERRED automatically — you do not write it.** An earlier draft of this doc wrongly
> claimed AILANG lacks typeclasses and recommended an explicit-comparator workaround.
> That was wrong (caught by reading prior art). The real fix is simpler: **drop the
> `: Ord` annotation and use `>`/`<`/`compare` directly.** AILANG infers Ord from usage.

**VERIFIED (2026-06-03):**
```ailang
-- ✅ THIS COMPILES AND RUNS (Ord inferred from `>`):
pure func mymax[a](x: a, y: a) -> a = if x > y then x else y
-- mymax(3, 7) → 7    mymax("apple", "banana") → "banana"
```

**Evidence from eval:**
- `polymorphic_ord_defaulting` failures (recent: only ~1% of compile failures — this is a
  LOW-frequency gap; see RECENT-VERIFIED banner). Models add a `: Ord` annotation that AILANG
  doesn't need.

**Concrete failing code** (from `gemini-3-flash`):
```ailang
-- ❌ What models write (unnecessary constraint annotation → parse error):
func top3[a: Ord](xs: list[a]) -> list[a] = ...

-- ✅ Correct: just drop the constraint — AILANG infers it
func top3[a](xs: list[a]) -> list[a] = take(3, sortBy(\x. \y. compare(x, y), xs))
```

---

## Goals

**Primary goal:** Models know to write `[a]` (not `[a: Ord]`) and let AILANG infer typeclass
constraints from operator/`compare` usage.

**Success metrics:**
- `polymorphic_ord_defaulting` compile_error rate drops to ≤1/5 in frontier models
- No model generates `[a: Ord]` / `[T: Comparable]` syntax after prompt update

---

## Solution Design

### Phase 1: Prompt — "constraints are inferred, don't annotate them"

AILANG infers typeclass constraints (Ord, Num, Eq, Show) from usage via dictionary passing.
Models just write the bare type parameter and use the overloaded operators:

```ailang
-- ❌ WRONG: don't annotate the constraint — AILANG has no [a: Ord] syntax
func top3[a: Ord](xs: list[a]) -> list[a] = ...

-- ✅ CORRECT: bare type param; Ord is INFERRED from `>` / compare()
pure func mymax[a](x: a, y: a) -> a = if x > y then x else y

-- ✅ CORRECT: polymorphic sort — constraint inferred from compare()
pure func sorted[a](xs: list[a]) -> list[a] = sortBy(\x. \y. compare(x, y), xs)
```

**Teaching prompt addition** (under "Common Mistakes"):
```
| `func f[a: Ord](xs: list[a])`  | Don't annotate constraints — AILANG INFERS them via |
|                                  | dictionary passing. Write `func f[a](xs: list[a])` and |
|                                  | use `>`/`<`/`compare` directly; Ord/Num/Eq are inferred. |
```

### Phase 2 (only if a real gap remains): explicit constraint syntax

If users genuinely need to *constrain* a type parameter that isn't constrained by usage
(rare), explicit `forall a. Ord a => ...` syntax could be added. But given constraints are
inferred today, this is likely unnecessary. **Do NOT pursue the explicit-comparator
workaround the earlier draft proposed — it's verbose and unnecessary given AILANG's
dictionary-passing inference.**

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
