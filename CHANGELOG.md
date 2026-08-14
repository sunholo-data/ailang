# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## v0.32.0 (Unreleased)

- Added experimental `std/ai.stepWithStreamRecorded`, originally authored by
  [@arniwesth](https://github.com/arniwesth). It preserves immediate stream
  callbacks while returning the exact ordered adapter-emitted chunk log and
  typed terminal outcome, including an explicit incomplete prefix on an
  unencodable chunk. See [#546](https://github.com/sunholo-data/ailang/issues/546)
  and [arniwesth/ailang#2](https://github.com/arniwesth/ailang/pull/2).

### Eval cost accuracy — the pricing gate now covers vendor-direct rows (no user-facing language or CLI change)

- **Added Gemini 3.7 Flash** (`gemini-3-7-flash`, released 2026-08-13) and **gave it the
  `extended_suite` Flash-tier slot over `gemini-3-6-flash`**, which drops to opt-in as the
  longitudinal anchor (the role `gemini-3-5-flash` got when 3.6 displaced it 22 days
  earlier). Vertex GA name `gemini-3.7-flash` verified against the global endpoint on
  project `ailang-dev` (HTTP 200, `modelVersion` echoed, `finishReason: STOP`); the
  `-preview`, `gemini-3-7-flash`, `-flash-lite` and `3.7-pro` variants all 404. Thinks by
  default (552 thinking vs 464 content tokens on a bare prompt), `max_output_tokens: 65536`
  established by API rejection at 131072.
- **The swap is justified by cost and recency, NOT capability.** Pass rates are
  indistinguishable — smoke 23/23 = 23/23, core 23/23 = 23/23 (both at ceiling, so that
  tier ranks nothing), stretch 24/25 = 24/25. The two stretch benchmarks they split on were
  de-flaked at N=3 and both proved flaky for *both* models (`emit_exact_bytes_varied` 3.6
  1/3 vs 3.7 2/3; `run_length_encode` 3.6 1/3 vs 3.7 0/3 — 2/6 each, dead even). The N=1
  reading that 3.7 fixes `emit_exact_bytes_varied`, the standing failure of the Flash line,
  was **retracted** before it reached the annotation. What decides it: identical price
  ($0.75/$3.75 both, since the promo covers 3.6 too) and **42–48% fewer thinking tokens**
  (core 34,405 vs 59,250; stretch 71,468 vs 137,376), making 3.7 12–19% cheaper — $0.925 vs
  $1.147 on stretch, the tier release baselines actually run. No truncation on either
  (`finish=length` 0/50), so the 65536 ceiling is not confounding the comparison.
  `--tier frontier` was NOT measured; the 3.5→3.6 swap had it and this one does not.
- **Corrected five vendor-direct rows** whose prices no gate could see:
  `gemini-3-6-flash` was $1.50/$7.50 per 1M after Google extended a 50% introductory
  cut across the Flash line (2x overstatement); `opencode-haiku`,
  `pi-claude-haiku-4-5` and `motoko-claude-haiku-4-5` were still on Claude Haiku
  **3.5's** $0.25/$1.25 while billing Haiku 4.5 at $1.00/$5.00 (**4x understatement**,
  on exactly the rows that exist to compare harness cost); `opencode-gemini-2.5-flash`
  carried gemini-3-flash's rates. A sixth, `gemini-3-flash-preview`, had **no pricing
  block at all** and so banked real Vertex spend as $0.00.
  Banked results are NOT recomputed, per the precedent set below.
- **Widened the offline CI gates from `provider: "openrouter"` to every billed
  provider** (`TestModels_PricingIsSlugConsistent`, `TestModels_PricingIsPlausible`;
  both renamed). The openrouter-only scope was never a considered choice, and it is
  what hid all six rows above. `ollama` stays out: those 22 rows are legitimately $0.
  Rows are now keyed by provider **and** api_name, since the same model reached
  through a reseller legitimately costs something different.
- **Added `pricing.expires` / `pricing.next`** — a schema for rates known in advance
  to change on a date, with `TestModels_PricingScheduleIsHonoured` enforcing it
  offline. Both Gemini Flash rows are on introductory pricing that **doubles on
  2027-01-01**; before this that reversion lived only in a YAML comment, so on New
  Year's Day every Gemini Flash cost figure would have silently halved with nothing
  going red. Verified by backdating an expiry and watching the gate fail.
- **Extended `make verify-model-pricing` to cross-check google/openai/anthropic rows**
  against OpenRouter's listing of the same model (47 of 51 rows map; the 4 that don't
  are reported, not guessed at). Two designs were rejected on measurement and the
  reasons are recorded in the tool's package comment: a checked-in expected-price
  table only restates the number models.yml already holds, and a staleness date wide
  enough not to nag (90 days) would have sailed past the Gemini miss, which happened
  23 days after that row was last verified.
- **These findings are ADVISORY (exit 0; `--strict` promotes them).** OpenRouter runs
  its own promotions — measured 2026-08-13, it listed `google/gemini-3.7-flash` at
  exactly half Google's published rate, and gpt-5.6-terra/luna and claude-sonnet-5 at
  0.50x/0.50x/0.67x. We bill these models direct, so the vendor's page is the
  authority and OpenRouter is a second opinion. Failing the build on a disagreement
  you are supposed to ignore is how a checker gets ignored. What IS hard-failed for
  vendor rows is decidable offline and sits in CI instead.

### Eval cost accuracy — OpenRouter pricing drift (no user-facing language or CLI change)

- **Corrected 23 of 39 `provider: "openrouter"` rows in `internal/eval_harness/models.yml`**
  against the live OpenRouter models API (audited 2026-08-13). Drift ran from 0.89x to
  3.67x and in **both** directions, so it was not a uniform bias that comparisons could
  absorb: an understated rate silently flatters a model in every cost-per-pass table, an
  overstated one penalises it. Worst cases: `or-deepseek-v3` output 3.67x understated,
  `or-qwen-2-5-72b` input 3.0x, and `or-deepseek-v4-pro` / `opencode-or-deepseek-v4-pro`
  ~2.7x — the last because the row had been holding the price of what is now a *different*
  slug (`deepseek/deepseek-v4-pro-0813`, the newer GA snapshot at $0.435/$0.87 per 1M).
- **Banked results are NOT recomputed.** A banked cost measures what a run cost under the
  rates in force at the time; recomputing it at today's rate would fabricate a figure no
  one was ever charged. Affected baselines get annotated, following the v0.30.0 cost
  invalidation precedent and `m-eval-measurement-contract.md` ("the existing data stays
  as-is, only annotated"). Cross-model cost comparisons that span the correction date
  should be treated as incomparable rather than reconciled.
- **Added `make verify-model-pricing`** (`tools/verify-model-pricing`) — diffs every
  openrouter row against the live API, reports the drift table with per-row factors, and
  exits 1 on drift / 2 when the check *could not run*. The exit-2 distinction is the point:
  "prices are correct" and "I could not find out" are different answers. Deliberately **not**
  in `make ci`, which must not go red because a third party's API is down. Supports
  `--json` and `STRICT=1` (also fail on withdrawn slugs).
- **Added `TestModels_OpenRouterPricingIsSlugConsistent`** — the offline half of the gate,
  which *does* run in CI. Rows sharing an `api_name` must share a price: OpenRouter bills
  the slug and has no idea which harness sent the request, so when `or-glm-5`,
  `opencode-or-glm-5` and `motoko-glm-5` disagree, the harness-lift comparison those rows
  exist to enable is reading a bookkeeping artifact. The audit found 4 such splits, worst
  being `google/gemma-4-26b-a4b-it` with three different prices across four rows — none of
  them the live price. Also added `TestModels_OpenRouterPricingIsPlausible` (no negative
  rates; no non-free row priced at $0, which would report real spend as free).
- Aligned `cache_read_per_1k` across the four `deepseek-v4-flash-0731` rows (only
  `pi-or-deepseek-v4-flash` had declared it). The remaining ~30 rows still declare no cache
  rate, which bills cache reads at the **full input rate** — an overstatement, the safe
  direction, and now reported continuously by `verify-model-pricing` rather than left
  invisible.
- Flagged two rows whose slugs no longer resolve on the API — `motoko-or-qwen3-5-35b-a3b`
  (`qwen/qwen3.5-35b-a3b-20260224`, dated snapshot withdrawn) and `or-laguna-xs-2`
  (`poolside/laguna-xs.2:free`). These are *reachability* defects, not pricing ones; both
  are left un-repointed on purpose, since choosing a replacement revision is a
  model-selection decision rather than a drift correction.
- Documented the pricing policy in the `models.yml` header: `pricing:` is the single source
  of truth, `notes:` prose is dated narrative that must not be rewritten to match new
  prices, and `description:` must not embed a rate at all (it has nowhere to carry a
  verified-on date). Hand-verification decays fast — `or-glm-5-2` carried "verified live
  2026-08-06" and its output rate was already 1.79x stale seven days later.

### Mission infrastructure (no user-facing language or CLI change)

- The mission loop's sprint-planner now defaults to the ChatGPT-subscription codex
  lane instead of opus, so opus stays controller-only. The configured default is not
  the effective lane: a new `tools/launchd/derive-planner-lane.sh` reads the picked
  design doc and fails **closed** to opus unless the doc declares
  `**Planner-Lane**: codex-ok` and every path in its Files section is inside a narrow
  infrastructure allowlist. Rollback is one commented line in
  `~/.config/ailang/mission-<name>.env`. Design docs gain an optional
  `**Planner-Lane**` header field. See
  `design_docs/implemented/v1_0_0/m-planner-codex-lane.md`.

## Changelog Archives

The full changelog has been split into themed files for searchability and readability:

| File | Versions | Theme |
|------|----------|-------|
| [v0.18-current.md](changelogs/v0.18-current.md) | v0.18.0+ | Eval Harness, Extensions & Agent Loop |
| [v0.10-v0.17-bytecode-vm.md](changelogs/v0.10-v0.17-bytecode-vm.md) | v0.10.0–v0.17.0 | Bytecode VM & Runtime |
| [v0.9-cloud-pubsub.md](changelogs/v0.9-cloud-pubsub.md) | v0.9.0–v0.9.12 | Cloud Integration & Pub/Sub |
| [v0.8-cloud-features.md](changelogs/v0.8-cloud-features.md) | v0.8.0–v0.8.1.1 | Cloud Features & Advanced Coordinator |
| [v0.7-observatory.md](changelogs/v0.7-observatory.md) | v0.7.0–v0.7.3 | Observatory, Chains & Dashboard |
| [v0.6-coordinator.md](changelogs/v0.6-coordinator.md) | v0.6.0–v0.6.2 | Coordinator Daemon & Agents |
| [v0.5-ai-providers.md](changelogs/v0.5-ai-providers.md) | v0.5.0–v0.5.10 | AI Providers, Eval Harness & Search |
| [v0.4-monomorphization.md](changelogs/v0.4-monomorphization.md) | v0.4.0–v0.4.10 | Monomorphization & DX Improvements |
| [v0.3-core-language.md](changelogs/v0.3-core-language.md) | v0.3.0–v0.3.25 | Core Language Stabilization |
| [v0.0-v0.2-foundation.md](changelogs/v0.0-v0.2-foundation.md) | v0.0.1–v0.2.1 | Foundation (Initial Release through Modules) |

All archives are indexed by `ailang docs search` for full-text and neural search.
