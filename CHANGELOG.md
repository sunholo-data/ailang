# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## v0.32.0 (Unreleased)

- Added experimental `std/ai.stepWithStreamRecorded`, originally authored by
  [@arniwesth](https://github.com/arniwesth). It preserves immediate stream
  callbacks while returning the exact ordered adapter-emitted chunk log and
  typed terminal outcome, including an explicit incomplete prefix on an
  unencodable chunk. See [#546](https://github.com/sunholo-data/ailang/issues/546)
  and [arniwesth/ailang#2](https://github.com/arniwesth/ailang/pull/2).

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
