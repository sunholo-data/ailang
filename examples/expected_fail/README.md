# Expected-to-Fail Examples

These examples demonstrate features that have known parser or runtime bugs. They are tracked in the design doc `design_docs/planned/v0_9_5/m-dx-expected-fail-examples.md`.

## Parser Bugs

- `contracts/hof_verify.ail` — Lambda syntax `\x -> x + 1` not parsed (expects `.` after param)
- `contracts/list_recursive_verify.ail` — Multiple `requires` blocks rejected (`PAR_DUPLICATE_REQUIRES`)
- `serve_api_webhook.ail` — `let` binding inside `@raw` route handler fails to parse

## Runtime Bugs

- `effect_budgets.ail` — `@limit=N` annotation breaks cap checking at runtime
- `effect_budgets_exhausted.ail` — Same budget cap checking bug
- `effect_budgets_multi.ail` — Same budget cap checking bug
- `effect_budgets_rand.ail` — Same budget cap checking bug

## When to move back

Move examples back to `examples/runnable/` once the underlying bug is fixed and the example passes `ailang run`.
