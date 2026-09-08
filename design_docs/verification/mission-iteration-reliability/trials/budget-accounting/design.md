# Trial A: Budget accounting for durable iterations

Status: frozen task brief under approved reliability M4; not yet executed.
Designer provenance: attended session, registry model `gpt6-astra`.
Priority: bounded adoption evidence. Scope: one documentation section, ~180–300 words.

## Problem and outcome

The guide names positive token/time/cost limits and a retry example but does not
explain their accounting. The real evaluator exhausted its cumulative allowance
mostly reading input. Add a concise **Budget accounting** section that lets an
operator interpret limits and recorded usage without mistaking subscription price
estimates for billed spend. Place it near the work-item/run explanation.

## Frozen criteria

- **A1-fresh-tokens:** Explain `max_tokens` as cumulative executor-reported fresh
  input plus output across turns, not output-only. Cache-read and cache-creation
  counters remain separate from this runtime sum; do not infer free provider billing
  from exclusion from the token guard. Give one small arithmetic example.
- **A2-stage-total:** Explain explicit per-stage and whole-item limits. Each new
  stage is capped by its own token/cost allowance and the remaining item allowance
  after accepted local stages; imported prerequisite work is not retrospectively
  charged to this item's execution budget. A stage's persisted deadline is bounded
  by the original item deadline; resume does not reset either deadline.
- **A3-cost-provenance:** Distinguish `metered`, `list-price-equivalent`, `free-local`
  and `unknown`. Only metered cost is deducted from the whole-item dollar balance;
  subscription list-price-equivalent arithmetic is not actual billed spend. State
  that per-stage reported-cost validation and pricing-based executor guards can
  still stop execution: do not promise a subscription lane ignores cost limits.
- **A4-unknown-accounting:** State precisely that OpenRouter work requires metered
  cost provenance for acceptance and unknown metered accounting blocks further
  work. Do not generalize this transport-specific rule to every unknown-cost lane.
- **A5-scope-quality:** Change only the guide, preserve existing instructions and
  examples, use source links for the accounting details, and keep the new section
  concise. No runtime behavior, tests, pricing tables or live configuration changes.

## Verified implementation evidence

Read at preparation on 2026-09-08; evaluator rechecks the frozen baseline.

| Source | Audited mechanism |
| --- | --- |
| `internal/mission/iteration/runtime_stage.go`, `request` | Subtracts accepted `InputTokens + OutputTokens`; subtracts `CostUSD` only for metered provenance; caps next request; prerequisites contribute author identity rather than retrospective usage |
| `internal/mission/iteration/runtime.go`, `validateUsage` | Rejects invalid/over-cap token and cost values; OpenRouter requires metered provenance; reported stage cost check is not restricted to metered provenance |
| `internal/mission/dispatch/run.go`, `taskFor` and `executionError` | Carries cumulative token allowance and pricing-based CostBudget; typed stop flags prevent acceptance |
| `internal/executor/cost.go`, `CostProvenance` | Defines all four provenance meanings |
| `internal/coordinator/mission_work_item_stage.go`, `BeginMissionStage` | Persists the first deadline bounded by the parent deadline |
| `docs/docs/guides/mission-iteration.md` | Existing run/recovery guide lacks a dedicated resource-accounting explanation |

No language semantics are proposed. Axiom effects: A9 cost visibility +1, A11
structured failure interpretation +1; remaining axioms unchanged. Related work:
`design_docs/planned/m-mission-iteration-reliability.md` and its M4 sprint plan.
