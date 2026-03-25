# Expected-to-Fail Examples

These examples are known to fail and are tracked for future implementation. They live here (not in `runnable/`) so `verify-examples` reports 100% pass rate without noise.

## Categories

### Contracts (verify-only, no main entrypoint)
- `contracts/per_function_depth_verify.ail` — Per-function `@verify(depth: N)` attribute (verify-only, no main)
- `contracts/quantifier_verify.ail` — Bounded quantifier `forall` in ensures (verify-only, no main)

### Contracts (unimplemented parser features)
- `contracts/hof_verify.ail` — Lambda arrow syntax `\x -> x + 1` in HOF args
- `contracts/list_recursive_verify.ail` — Multiple `requires` blocks per function

### Effect Budgets (physical vs semantic counting bug)
- `effect_budgets.ail` — Basic effect budget tracking
- `effect_budgets_exhausted.ail` — Budget exhaustion behavior
- `effect_budgets_multi.ail` — Multi-effect budgets
- `effect_budgets_rand.ail` — Random effect budgets

### Package Demo (package management not implemented)
- `package_demo/` — Multi-module package import demo

### API (parser limitation)
- `serve_api_webhook.ail` — `match` with `let` bindings in route handler

## When to move back

Move examples back to `examples/runnable/` once the underlying feature is implemented and the example passes `ailang run`.
