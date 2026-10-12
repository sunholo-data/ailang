### Fixed — motoko and opencode bank registry-priced cost, not their own meters

The follow-up to the pi fix: motoko and opencode also set `Result.CostUSD` from a
figure the harness computed itself.

- **motoko** banked `run_summary.total_cost_usd`, whose meter
  (`step_cost_millicents` in motoko core) prices input + output only. Measured
  2026-10-12 on banked run_summary rows: Haiku 5.5 on direct Anthropic reported
  $0.01329 against $0.05455 at registry rates (2.55M cache reads and 125K cache
  writes unpriced); GLM-5.3-Flash via OpenRouter reported $2.00187 against
  $0.44314 (12.999M of its 13.272M input tokens were cache hits, billed at the
  full input rate). `Result.CostUSD` is now registry rates × tokens via
  `executor.PriceTokens`; motoko's figure is kept as
  `ProviderData["motoko_self_reported_cost_usd"]`.
- **motoko input tokens are now FRESH input on cache-inclusive routes.** motoko
  passes `ai.Response.InputTokens` through, which is disjoint from cache reads on
  the direct Anthropic wire but includes them on every OpenAI/Gemini-shaped wire
  (`openrouter/`, `openai/`, `chatgpt/`, `google/`, `ollama/`). The adapter now
  subtracts the cache reads there (the executor contract codex already follows),
  so banked `input_tokens` for those motoko rows drop; the raw figure is kept as
  `ProviderData["motoko_reported_input_tokens"]`. A cache read larger than the
  input is banked unpriced (`$0`, `unknown`), not clamped.
- **opencode** banked the sum of `step_finish.part.cost`, from opencode's own
  price table. Now registry rates × tokens (cache is already exclusive of input
  on that wire); opencode's figure is `ProviderData["opencode_self_reported_cost_usd"]`.
  Provenance now uses `executor.AuthLaneForModel`, so an Ollama Cloud route is
  `list-price-equivalent` rather than `metered`.
- The `provider/model` → registry-row resolution pi introduced is now shared as
  `executor.HarnessCostModel` (pi, motoko, opencode). The bare-id fallback still
  requires the row's provider to match, so `openrouter/z-ai/glm-5.3-flash` can
  never pick up the Lyceum row and an `ollama/` route never borrows a cloud price.

Unchanged: `CostKilledAt` for motoko stays motoko's own figure (what its cap
compared against), and motoko's internal cost cap still meters input + output
only — that is a motoko-core fix, not an adapter one.
