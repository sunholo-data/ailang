## M-BUDGET-SCOPING-BUG: `@limit`/`@min` effect budgets are cumulative across the call chain, not per-function

**Status**: PLANNED (bug)
**Target**: v0.27.x / v0.28.0
**Priority**: P2 (Medium — makes per-function budget annotations misleading; surfaced a shipped broken example)
**Estimated**: 0.5–1 day (semantics decision + scoping fix + test migration)
**Dependencies**: None.

**Found during**: M-SNAKE-FEEDBACK (migrating `effect_budget_demo.ail` off `++`). Verified on v0.26.2.

---

## Verdict: the implementation contradicts the annotation's documented intent

A function annotated `! {IO @limit=3}` is documented to mean "this function may perform at
most 3 IO operations" — [docs/docs/reference/effects.md](../../docs/docs/reference/effects.md)
and the `effect_budget_demo.ail` comments both say so. The implementation instead enforces a
**cumulative** budget over the whole dynamic call chain: a callee's `@limit` is charged the
IO already spent by its callers. A function that makes only 2 of its own IO calls fails if
the caller already made 3.

This is either a real scoping bug (most likely — the annotation is per-function in both the
docs and the user's mental model) or an undocumented design choice that needs to be documented
and the annotation re-described. Either way the current behavior is a trap.

## Minimal reproduction

```ailang
module repro
import std/io (println)

export func limited(x: int) -> () ! {IO @limit=3} {
  println("a");   -- only 2 of limited's own IO calls
  println("b")
}

export func main() -> () ! {IO} {
  println("p1"); println("p2"); println("p3");  -- 3 IO in main (no @limit)
  limited(0)                                      -- limited makes 2 → should be fine
}
```

```
$ ailang run --caps IO --entry main repro.ail
p1
p2
p3
a
Error: execution failed: effect 'IO' budget exhausted: semantic limit=3, used=3 (physical: 4)
```

`limited`'s `@limit=3` is exhausted at **physical=4** = main's 3 prints + `limited`'s first
print. The limit is being applied to the global running total, not to `limited`'s own effects.

This is exactly why `examples/effect_budget_demo.ail` cannot run to completion: its narrative
`main()` preamble (`println("=== Demo ===")` …) spends the budget before the first budgeted
helper is even entered. (Migrated off `++` in M-SNAKE-FEEDBACK; left runtime-broken and
skipped-with-reason in `verify-examples-toplevel`.)

## Investigation pointers

- Budget composition note: "[for nested scopes] limits are summed" — [internal/effects/budget.go:293](../../internal/effects/budget.go#L293).
- `EnterFunction` attribution — [internal/effects/budget.go:399](../../internal/effects/budget.go#L399).
- Budget context preserved across scopes — [internal/effects/context.go:348](../../internal/effects/context.go#L348) (`BudgetReport` shared).
- Existing tests encode "physical: N" cumulative numbers — [internal/effects/budget_test.go:255](../../internal/effects/budget_test.go) — so a fix must reconcile these (they may have been written to the buggy behavior).

## Proposed direction (decision required)

**Option A (recommended): per-function scoping.** On `EnterFunction` with a `@limit`/`@min`,
push a fresh budget *frame* that counts only effects performed within that function's dynamic
extent (and, by composition, its callees up to their own frames). The annotation then means
what the docs say. Update `budget_test.go` expectations to the corrected semantics.

**Option B: keep cumulative, fix the docs + annotation name.** If cumulative-from-entry is
intended (e.g. a whole-program ceiling), rename/clarify so `@limit` reads as "max total IO
from here down" and fix `effect_budget_demo.ail` + the effects guide to match. Less surprising
than today only if clearly documented.

Pick A unless there's a concrete reason the budget must be a global ceiling.

## Acceptance Criteria

- [ ] A function's `@limit=N` bounds **its own** effects (per the chosen semantics), with a
      regression test mirroring the repro above.
- [ ] `examples/effect_budget_demo.ail` runs to completion under `--caps IO` and is **removed
      from the `verify-examples-toplevel` run-skip list**.
- [ ] `internal/effects/budget_test.go` updated to the corrected semantics (no stale cumulative
      assumptions).
- [ ] `docs/docs/reference/effects.md` "Capability Budgets" describes the scoping precisely.
