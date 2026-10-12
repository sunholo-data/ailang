### Fixed — pi executor banks registry-priced cost, not pi's own price table

The pi executor set `Result.CostUSD` by summing pi's self-reported
`message_end.usage.cost.total`, which comes from pi's built-in model price table
(pi 0.85.1). That table is wrong for models it does not know: six measured
`anthropic/claude-haiku-5-5` runs reported $15.49 against $0.275 at list price
(~56x over), so the mission stage cost cap (`CostUSD > max_cost_usd`) killed
healthy Haiku 5.5 runs and banked costs were wrong.

- `Result.CostUSD` is now registry rates × the accumulated tokens (input, output
  + reasoning, cache read, cache write), via a new shared
  `executor.PriceTokens` (Task.Pricing first, else the task's model resolved
  through `internal/modelreg`; pi's `provider/model` names fall back to the bare
  id only when the resolved row has the same provider).
- An unresolvable model banks $0 with `cost_provenance: unknown` — never pi's
  figure and never a metered $0. A zero-rate row is `free-local`.
- pi's own figure is kept as `ProviderData["pi_self_reported_cost_usd"]` and the
  `pi.self_reported_cost_usd` span attribute, for diagnosis only.
- `PiExecutor.CostModel()` returns the registry card (or an explicit Unpriced
  card) instead of an all-zero placeholder.

Known gap: the registry's scalar `pricing` is the short-context card; Haiku 5.5's
long-context tier (>100K prompt tokens, `claude_credit`) is only priced by the
credit-gateway settlement path, so a pi turn above 100K prompt tokens is
under-reported at short-context rates.
