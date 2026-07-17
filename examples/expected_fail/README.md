# Expected-to-Fail Examples

These examples exist to **fail on purpose**. They are either code the
type-checker MUST reject, or runtime demonstrations of an enforcement mechanism
firing. If any of them ever starts passing `ailang check` / `ailang run`
cleanly, a real regression has shipped.

Examples that were previously parked here as "parser bugs" have been fixed to
canonical AILANG syntax and promoted to `examples/runnable/` as regression
guards (`contracts/hof_verify.ail`, `contracts/list_recursive_verify.ail`,
`serve_api_webhook.ail`).

## Type-Checker Rejections (correctly fail to type-check)

These are NOT bugs — they're examples of code the typechecker MUST
reject. Acceptance: each one fails `ailang check` with a clear,
specific error message.

- `match_foreign_constructor_option.ail` — `match Option { Err(_) ... }` (M-MATCH-ADT-XCHECK v0.18.10)
- `match_foreign_constructor_result.ail` — `match Result { Some(_) ... }` (M-MATCH-ADT-XCHECK v0.18.10)

## Budget Enforcement Demonstrations (correctly fail at runtime)

These files carry `@limit=N` effect budgets and **intentionally exceed them** so
that runtime budget enforcement fires. The budget-exhausted error IS the
demonstration — a clean run would mean enforcement broke.

- `effect_budgets.ail` — `limitedPrints()` has `IO @limit=3` and prints 4 times
- `effect_budgets_exhausted.ail` — budget exhaustion path
- `effect_budgets_multi.ail` — multiple effect budgets in one function
- `effect_budgets_rand.ail` — budgeted `Rand` capability

**Correct invocation — put `--caps` BEFORE the filename:**

```bash
ailang run --caps IO,Clock examples/expected_fail/effect_budgets.ail
# → Error: execution failed: effect 'IO' budget exhausted: semantic limit=3, used=3
```

Flags placed AFTER the filename are ignored, so `ailang run <file> --caps IO`
silently drops the caps and you won't see enforcement — that is an invocation
mistake, not a bug in `@limit=N`.

> **Open question (backlog, not a task here):** in `effect_budgets.ail`, the
> `@limit=3` on `limitedPrints()` reports `physical: 4` at the point of
> exhaustion — the parent scope's `IO` operations appear to count toward the
> child function's budget. Whether that accounting is intended (physical vs.
> semantic counting across call boundaries) is unresolved and should be filed
> separately.

## When to move an example back to runnable/

Move an example to `examples/runnable/` once it is expected to pass
`ailang check` and (if it has a `main`) `ailang run` cleanly. The budget and
type-rejection demos above are **not** candidates — they fail by design.
