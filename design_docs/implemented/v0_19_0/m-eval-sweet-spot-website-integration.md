# M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION: Surface sweet-spot data on the public benchmark dashboard

**Status**: IMPLEMENTED
**Target**: v0.19.0
**Priority**: P1 (Medium — data already exists end-to-end; without dashboard surfaces operators can't see it without CLI access)
**Estimated**: ~3 working days (~22 hours)
**Dependencies**:
- M-EVAL-SWEET-SPOT (v0.19.0, implemented) — typed failure categories, `FinishReason`, sweet-spot bucketing, ASCII cost-vs-time frontier
- M-EVAL-SWEET-SPOT-FOLLOWUP-P1 (v0.19.0, implemented) — motoko cost-cap enforcement
- **M-BENCHMARK-DATA-INTEGRITY** (v0.19.0, planned) — sibling milestone; shares Issue #4 (api_error subcategory splitting). This doc layers ON TOP of integrity fixes; whichever ships first, the other consumes the cleaner data.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|------:|---------------|
| A1: Determinism | +1 | Same `eval_results/` always produces same dashboard render — pure function of inputs |
| A2: Replayability | +1 | Sweet-spot blocks embedded in `latest.json` are recomputable from the result JSONs |
| A3: Effect Legibility | 0 | n/a — dashboard is read-only |
| A4: Explicit Authority | 0 | n/a |
| A5: Bounded Verification | +1 | Per-(model × benchmark) bucket assignment is locally verifiable from a single result JSON |
| A6: Safe Concurrency | 0 | n/a |
| A7: Machines First | +2 | Machine-readable JSON shape AND machine-decidable bucket assignment. Downstream tools (blog generators, model-selection agents) can ingest without parsing visual charts |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +2 | Direct alignment — promotes $/pass economics from CLI-only to public-dashboard primary surface |
| A10: Composability | +1 | Reuses `BuildSweetSpot` (M4) and `EfficiencyByModel` (M4). No new server-side computation |
| A11: Structured Failure | +1 | Typed `error_category` breakdown becomes a public-facing chart (currently buried in CLI text) |
| A12: System Boundary | +1 | Dashboard JSON schema becomes the single source of truth for sweet-spot interpretation across CLI, MDX, and website |

**Net Score: +10** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Pure functions of result JSONs
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strict win — replaces visual-only Pareto with machine-decidable bucket assignment

## Problem Statement

M-EVAL-SWEET-SPOT shipped end-to-end this milestone — typed error categories, ASCII Pareto frontier, per-benchmark champion identification, $/pass economics — but the only surface today is the CLI (`ailang eval-sweet-spot`) and the auto-generated MDX export. The public dashboard at `docs/src/components/BenchmarkDashboard/` still consumes the v0.15.1 `efficiency` block and renders the old `QualityScatter` / `SpeedRadar` / `ValueScoreTable` charts — none of which know about:

- `finish_reason` (e.g. `cost_exhausted`, `step_exhausted`)
- Typed `ErrorCategory` values: `cost_killed`, `step_exhausted`, `timeout`, `quota_exhausted`, `rate_limit`
- Sweet-spot buckets: `fast_pass` / `slow_pass` / `budget_blocked` / `capability_blocked` / `provider_blocked`
- Per-benchmark champion data (cheapest pass × fastest pass)
- $/pass-success economics rolled up across tiers
- The Pareto-frontier-inversion phenomenon (deepseek dominates hard, claude+gemma share stretch — observed in 56-run validation)

Without dashboard integration, every blog post / external comparison / model-selection decision still has to be made from the CLI output by someone who knows where to look. The whole point of M-EVAL-SWEET-SPOT was operator visibility — but the operators are on the website.

**Current State:**

| Surface | Sweet-spot data present? | Status |
|---|---|---|
| Per-result JSON (`error_category`, `finish_reason`, `cost_killed_at`) | ✅ schema correct | Ships from v0.19.0 evals |
| `ailang eval-sweet-spot` CLI | ✅ full report (text/CSV/JSON/MDX) | Implemented M4-M5 |
| `latest.json` dashboard data file | ❌ no sweet-spot block | **Gap — this doc** |
| BenchmarkExplorer JSX | ❌ no bucket / champion / category charts | **Gap — this doc** |
| ValueScoreTable | partial — uses `efficiency` block, not sweet-spot | Update to read new shape |
| QualityScatter | partial — Pareto on cost OR speed; no bucket coloring or category filtering | Update to color by bucket |
| BenchmarkDashboard index | ❌ no top-level $/pass economics table | **Gap — this doc** |

**Impact:**

- **Who is affected?** Anyone reading the public benchmark dashboard, blog post authors comparing models, downstream model-selection agents that scrape `latest.json`, the AILANG vs competitor narrative
- **How significant?** The 56-run validation session showed deepseek-v4-flash is **12.4× cheaper per pass than gemma-4-26b** on OpenRouter — that headline number is currently invisible on the website. Every model-selection conversation has to be re-explained from CLI output, which doesn't scale.

## Goals

**Primary Goal:** Every fact the CLI sweet-spot report surfaces (buckets, champions, $/pass, typed failures, frontier membership) is also visible on the public benchmark dashboard with the same definitions and the same numbers.

**Success Metrics:**

1. **Numerical parity**: every per-model field shown in `ailang eval-sweet-spot --format=json` is also in `models[name].sweet_spot` in `latest.json`. A diff of "CLI numbers vs dashboard numbers" produces zero discrepancies.
2. **Visible $/pass economics**: a new "$/pass" sortable column appears in the leaderboard / explorer view, with the headline 12.4× deepseek-vs-gemma spread visible without expanding any row.
3. **Failure-category breakdown**: at least one chart on the dashboard shows the per-model split into typed categories (cost_killed / step_exhausted / timeout / quota / rate_limit / api_error). Currently the JSX collapses everything into "api_error" or counts only `compile_error`.
4. **Per-benchmark champions table**: a new section listing each benchmark's cheapest-pass model and fastest-pass model.
5. **Tier-aware Pareto**: the existing QualityScatter gains a tier filter so users can see the frontier-inversion phenomenon (deepseek wins hard, claude+gemma share stretch).
6. **Integrity audit**: every number on the published dashboard validated against re-running `ailang eval-sweet-spot` locally. Discrepancies documented as integrity issues.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `latest.json` schema extension shape (top-level `sweet_spot` vs per-model `sweet_spot` block) | Determines JSX consumer code path AND backward compat for legacy consumers reading `models[].aggregates` | human | design | high — every dashboard component must update if we change layout |
| Whether to compute sweet-spot data at **export time** (Go) or **render time** (JSX) | Compute-at-export: pre-computed buckets, smaller JSX, less in-browser CPU. Compute-at-render: live tier-filter recompute, but JSX must re-implement bucketing | human | design | med — affects test surface (Go golden vs Jest) |
| Per-pass cost column placement (leaderboard vs explorer vs new section) | Headline economics number — wherever it lives is where the first-impression comparison happens | human | design | low — just CSS/JSX wiring |
| Adjusted-rate denominator after typed categorization (this overlaps with M-BENCHMARK-DATA-INTEGRITY Issue #4) | Determines what "pass rate" actually means on the dashboard | human | design | high — published numbers shift |
| Backfill policy for pre-v0.19.0 result JSONs (no `finish_reason` / typed `error_category`) | Whether legacy baselines participate in sweet-spot bucketing or are excluded | human | design | med — categorizer can do offline backfill via stderr matching |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Schema layout**: proposed `models[name].sweet_spot = { pass_rate, median_tts_ms, p90_cost_per_success, buckets: {...}, finish_reasons: {...}, pareto_frontier: bool, dollars_per_pass }` — confirmed?
- [ ] **Compute strategy**: proposed = compute at export time in Go (`ExportDashboardJSON` calls `BuildSweetSpot`, embeds output). JSX reads pre-computed. Confirmed?
- [ ] **$/pass column location**: proposed = new column on existing `LanguageLeaderboard` table, default-sorted by `$/pass asc`. Confirmed?
- [ ] **Adjusted-rate coordination with M-BENCHMARK-DATA-INTEGRITY**: which doc owns the `successRateAdjusted` formula change? (Proposal: data-integrity doc owns it; this doc consumes the cleaned denominators.)
- [ ] **Backfill**: proposed = lazy categorization at read time in `loader.go`, mirroring M-EVAL-SWEET-SPOT M1's offline-recompute pattern. Pre-v0.19.0 baselines participate but with `finish_reason=""`. Confirmed?

## Solution Design

### Issue Index (parallel to M-BENCHMARK-DATA-INTEGRITY numbering)

| # | Topic | Severity | Sibling fix? |
|---|---|---|---|
| **#A** | `latest.json` missing sweet-spot block per model | High | New — this doc |
| **#B** | No public surface for typed failure categories | High | New — this doc |
| **#C** | No per-pass economics table on dashboard | High | New — this doc |
| **#D** | No per-benchmark champions list on dashboard | Med | New — this doc |
| **#E** | QualityScatter doesn't color by bucket / filter by tier | Med | New — this doc |
| **#F** | Numerical parity between CLI sweet-spot and dashboard not verified | Med | Audit, no code change |
| **#G** | Adjusted-rate formula needs typed categories (sibling Issue #4 in integrity doc) | High | **M-BENCHMARK-DATA-INTEGRITY owns** |

### Issue #A: Embed sweet-spot block in `latest.json`

**Problem**: dashboard JSX cannot render what isn't in the JSON. Today `latest.json` carries per-model `aggregates` and `efficiency` but no bucket counts, typed failure breakdown, or Pareto frontier membership.

**Proposed fix**: extend `internal/eval_analysis/export_json.go` to call `BuildSweetSpot(results, SweetSpotOpts{})` once at export time and embed the per-model rows into each `models[name]` entry as `sweet_spot`.

```json
"models": {
  "motoko-or-deepseek-v4-flash": {
    "aggregates": {...},
    "efficiency": {...},
    "sweet_spot": {
      "pass_rate": 0.857,
      "median_tts_ms": 46700,
      "p90_cost_per_success": 0.0422,
      "speed_efficiency": 0.681,
      "dollars_per_pass": 0.038,
      "pareto_frontier": true,
      "buckets": {
        "fast_pass": 4, "slow_pass": 5, "budget_blocked": 0,
        "capability_blocked": 1, "provider_blocked": 0
      },
      "finish_reasons": {
        "stop": 13, "cost_exhausted": 0, "step_exhausted": 0
      },
      "error_categories": {
        "none": 12, "runtime_error": 1, "logic_error": 0, "timeout": 0,
        "quota_exhausted": 0, "rate_limit": 0, "cost_killed": 0,
        "step_exhausted": 0, "api_error": 1
      }
    }
  }
}
```

**Top-level addition** for cross-model views:
```json
"sweet_spot_global": {
  "champions": [
    { "benchmark_id": "lambda_calc",
      "cheapest_model": "motoko-or-deepseek-v4-flash",
      "cheapest_cost": 0.0159,
      "fastest_model": "claude-haiku-4-5",
      "fastest_tts_ms": 29000 }
  ],
  "slow_threshold_ms": 60000,
  "total_runs": 56
}
```

### Issue #B: Typed failure-category chart

**Problem**: there's no JSX component that breaks down failures by typed category. The closest is the existing `apiErrors` count in `aggregates`, which lumps everything together.

**Proposed fix**: new `FailureCategoryBars` JSX component, stacked horizontal bars per model. Reads `models[name].sweet_spot.error_categories`. Color-coded:
- Capability (red): `compile_error`, `runtime_error`, `logic_error`, `timeout`
- Budget (orange): `cost_killed`, `step_exhausted`
- Provider (gray): `quota_exhausted`, `rate_limit`, `api_error`
- Success (green): `none`

Lives in a new "Failure Modes" tab next to the existing speed/cost tabs.

### Issue #C: Per-pass economics table

**Problem**: the headline "deepseek is 12× cheaper per pass than gemma" number is computable from existing data but not displayed anywhere on the dashboard.

**Proposed fix**: new `DollarsPerPassTable` component. Reads `models[name].sweet_spot.dollars_per_pass`. Default-sorted ascending (cheapest first). Shows: model, $/pass, pass rate, total spend in dataset. Optional toggle: "show as ratio vs cheapest" (so the gemma row reads "12.4×").

Lives in the `LanguageLeaderboard` section or replaces the current `ValueScoreTable` (the latter would be a deprecation; needs Design Freeze approval).

### Issue #D: Per-benchmark champions

**Problem**: the "cheapest model that passes this benchmark" data is a strong model-selection signal — "use deepseek for batch work on these benchmarks". It exists in CLI output, nowhere in the JSON or JSX.

**Proposed fix**: new `BenchmarkChampionsTable` JSX component. Reads `sweet_spot_global.champions[]`. Columns: benchmark, cheapest model + cost, fastest model + TTS. Sortable.

### Issue #E: Pareto frontier coloring + tier filter

**Problem**: existing `QualityScatter` already plots a Pareto frontier (good!), but every point is the same shape regardless of which sweet-spot bucket it's in, and there's no tier filter. The frontier-inversion phenomenon (deepseek dominates hard, claude+gemma share stretch) is invisible because the chart shows aggregate data.

**Proposed fix**:
1. Extend `QualityScatter` to optionally color points by `dominant_bucket` (the bucket with the largest count for that model).
2. Add a tier-filter dropdown above the chart (`all` / `core` / `stretch` / `vision`). When set, the JSX filters `models[name].sweet_spot.tiers[tier]` (requires per-tier sweet-spot data — see Issue #5 in M-BENCHMARK-DATA-INTEGRITY).

### Issue #F: Numerical-parity audit

**Problem**: we have CLI output for sweet-spot AND dashboard output for legacy metrics, but no cross-check that re-running `eval-sweet-spot` against the same data the dashboard renders produces the same numbers. Easy to drift.

**Proposed fix**: new `internal/eval_analysis/sweet_spot_parity_test.go` — golden test that asserts:
1. `BuildSweetSpot` output rendered as JSON equals the `models[].sweet_spot` block in the same run's `latest.json`
2. `Champions` array matches `sweet_spot_global.champions`
3. Per-model $/pass values agree to 4 decimal places

Failure mode: if the data shapes drift, the test fails and forces a deliberate update. **Acceptance test for this whole milestone.**

### Issue #G: Adjusted-rate denominator (defer to sibling)

**Problem**: M-BENCHMARK-DATA-INTEGRITY Issue #4 already proposes the right schema change (`error_subcategory` field, exclude only `quota_error` from adjusted-rate denominator). This doc's typed `error_category` values (`quota_exhausted`, `rate_limit`) provide the **same information at a finer grain** — they ARE the subcategory split that integrity doc proposes.

**Proposed coordination**:
- M-BENCHMARK-DATA-INTEGRITY Issue #4 schema becomes: use `error_category in {"quota_exhausted", "rate_limit"}` as the adjusted-rate exclusion set (matches `ShouldExcludeFromCapability` semantics from M-EVAL-SWEET-SPOT M3).
- This doc consumes the cleaner adjusted rates; no formula change needed here.

### Implementation Plan

**Phase 1: Embed sweet-spot in latest.json** (~8 hours)
- [ ] Add `sweet_spot` field to per-model entry in `internal/eval_analysis/export_json.go`
- [ ] Add `sweet_spot_global` (champions, slow_threshold_ms, total_runs) at top level
- [ ] Hoist `BuildSweetSpot` call to the export pipeline; share the `SweetSpotReport` between MDX export and JSON export
- [ ] Golden test: `internal/eval_analysis/export_json_sweet_spot_test.go`
- [ ] Manual regenerate: `ailang eval-report eval_results/baselines/v0.18.5 v0.18.5 --format=json` and inspect

**Phase 2: Numerical-parity audit** (~3 hours)
- [ ] `sweet_spot_parity_test.go` — JSON-vs-CLI round trip
- [ ] Run against the 56 M-EVAL-SWEET-SPOT validation result JSONs; document any discrepancies
- [ ] Add to `make ci`

**Phase 3: JSX surfaces** (~8 hours)
- [ ] `DollarsPerPassTable` — new component, slot into `LanguageLeaderboard` page
- [ ] `BenchmarkChampionsTable` — new component, slot below leaderboard
- [ ] `FailureCategoryBars` — new "Failure Modes" tab
- [ ] Extend `QualityScatter`: bucket coloring + optional tier filter
- [ ] Update `ValueScoreTable` to use `sweet_spot.dollars_per_pass` instead of computing locally

**Phase 4: Docs + screenshots** (~3 hours)
- [ ] `docs/docs/guides/evaluation/cost-and-speed-budgets.md` — add "Reading the dashboard sweet-spot view" section
- [ ] Screenshot the new charts using the M-EVAL-SWEET-SPOT validation dataset; embed in the guide
- [ ] CHANGELOG entry

### Files to Modify/Create

**Modified files:**

- `internal/eval_analysis/export_json.go` — add `sweet_spot` per-model + `sweet_spot_global` top-level. ~80 LOC.
- `internal/eval_analysis/export_json_sweet_spot_test.go` — **new**, golden test. ~200 LOC.
- `internal/eval_analysis/sweet_spot_parity_test.go` — **new**, parity test vs CLI. ~150 LOC.
- `docs/src/components/BenchmarkDashboard/DollarsPerPassTable.jsx` — **new**, ~150 LOC.
- `docs/src/components/BenchmarkDashboard/BenchmarkChampionsTable.jsx` — **new**, ~120 LOC.
- `docs/src/components/BenchmarkDashboard/FailureCategoryBars.jsx` — **new**, ~180 LOC.
- `docs/src/components/BenchmarkDashboard/QualityScatter.jsx` — bucket coloring + tier filter. ~40 LOC added.
- `docs/src/components/BenchmarkDashboard/ValueScoreTable.jsx` — read `sweet_spot.dollars_per_pass`. ~20 LOC changed.
- `docs/src/components/BenchmarkDashboard/index.jsx` — wire new tabs / sections. ~30 LOC added.
- `docs/docs/guides/evaluation/cost-and-speed-budgets.md` — "Reading the dashboard sweet-spot view" section. ~80 LOC.
- `CHANGELOG.md` — entry under [Unreleased] / v0.19.0.

**No new Go executable changes** — `BuildSweetSpot` already exists.

## Examples

### Example 1: Before/After — $/pass economics

**Before (current dashboard)**: model rows show `successRate`, `avgTokens`, `totalCostUSD` — but no $/pass aggregated. User has to mentally divide `totalCostUSD` by `successCount`.

**After**: leaderboard gains a `$/pass` column, default-sorted ascending:

| Model | $/pass | Pass rate | Total spend |
|---|---:|---:|---:|
| motoko-or-deepseek-v4-flash | **$0.038** | 86% | $0.46 |
| claude-haiku-4-5 | $0.089 | 100% | $1.24 |
| motoko-or-kimi-k2-6 | $0.109 | 50% | $0.76 |
| motoko-or-gemma-4-26b | $0.473 | 43% | $2.84 |

Plus optional "show as ratio" toggle for the headline 12.4× spread.

### Example 2: Typed failure breakdown

**Before**: dashboard `apiErrors` count = 51 across v0_18_* (the 76% of failures we couldn't categorize).

**After** (from `models[].sweet_spot.error_categories` after offline categorization):

```
gemma-4-26b · stretch    [████████░░] runtime_error=4  rate_limit=1
deepseek-v4-flash · all  [█░░░░░░░░░] runtime_error=1
kimi-k2-6 · all          [██░░░░░░░░] logic_error=2  api_error=2
```

Now the "model has issues" diagnosis ("provider noise" vs "couldn't write code") is visible at a glance.

### Example 3: Frontier-inversion visibility

**Before**: QualityScatter shows aggregate Pareto across all benchmarks; no way to see that the frontier reshapes by tier.

**After**: tier-filter dropdown. Selecting "stretch" shows the validation finding directly:

```
Stretch tier (6 benchmarks):
  fast ┤                          A claude (only fast pass)
       │     d gemma (cheap, rare wins — frontier!)
       │              B deepseek
       │                          C kimi
  slow ┘
       └─────────────────────────────
       cheap                expensive
```

Operators can see "for stretch tier, gemma is the cost-frontier when it passes; claude is the speed-frontier; deepseek loses dominance".

## Success Criteria

- [ ] `latest.json` from `ailang eval-report` includes `models[name].sweet_spot` blocks AND `sweet_spot_global.champions`
- [ ] `sweet_spot_parity_test` passes against the M-EVAL-SWEET-SPOT validation dataset (56 runs across 3 tiers)
- [ ] BenchmarkExplorer page renders the new `DollarsPerPassTable`, `BenchmarkChampionsTable`, `FailureCategoryBars` components without console errors
- [ ] QualityScatter tier filter works (manually verified on the v0.18.5 baseline)
- [ ] Headline number visible on first load: "deepseek-v4-flash is the cheapest model per pass at $0.038"
- [ ] `cd docs && npm run build` succeeds
- [ ] `make ci` passes
- [ ] CHANGELOG entry under v0.19.0

## Testing Strategy

**Unit tests:**
- `export_json_sweet_spot_test.go` — assert `sweet_spot` block shape against golden JSON
- `sweet_spot_parity_test.go` — assert numerical equality between CLI output and dashboard JSON for the same input dataset
- JSX: lightweight Jest snapshot tests for the 3 new components on a fixed `latest.json` fixture

**Integration tests:**
- Regenerate `latest.json` from M-EVAL-SWEET-SPOT validation data; build the docs site (`npm run build`); inspect rendered HTML for expected numbers
- Spot-check: take 3 specific values from the CLI sweet-spot output, find them in the rendered dashboard, confirm match

**Manual testing:**
- Open the dashboard locally (`cd docs && npm start`); click through each new component
- Confirm tier-filter on QualityScatter reproduces the frontier-inversion finding from the validation data
- Confirm headline $/pass row is sorted correctly and the 12.4× ratio is reachable in ≤2 clicks from landing page

## Numerical-Parity Audit (Issue #F detail)

The audit task: re-run `ailang eval-sweet-spot` on the same data set the website renders, then diff. The dashboard MUST agree with the CLI for these fields:

| Field | CLI source | Dashboard source |
|---|---|---|
| Pass rate | `report.Rows[i].PassRate` | `models[i].sweet_spot.pass_rate` |
| Median TTS | `report.Rows[i].MedianTTSMs` | `models[i].sweet_spot.median_tts_ms` |
| p90 $/pass | `report.Rows[i].P90CostPerSuccess` | `models[i].sweet_spot.p90_cost_per_success` |
| Speed eff. score | `report.Rows[i].SpeedEfficiency` | `models[i].sweet_spot.speed_efficiency` |
| Pareto membership | `frontierPoint.pareto` | `models[i].sweet_spot.pareto_frontier` |
| Bucket counts | `report.Rows[i].Buckets` | `models[i].sweet_spot.buckets` |
| Champion (cheapest) | `report.Champions[i].CheapestModel` | `sweet_spot_global.champions[i].cheapest_model` |

The parity test is the operational mechanism: any drift fails CI.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Color palette** for bucket coloring on QualityScatter — agent may choose, consistent with existing chart palettes
- **Tab placement** for the "Failure Modes" view (top-level tab vs nested under "Performance") — agent may choose
- **Per-tier sweet-spot** in `latest.json` (`sweet_spot.tiers[tier]`) — defer to follow-up if tier filter isn't needed in v0.19.0; depends on M-BENCHMARK-DATA-INTEGRITY Issue #5 shipping first
- **Backfill of historical baselines** with offline-categorized `error_category` values — recommend lazy at `loader.go` (mirrors M1 pattern); can re-run baseline regenerations separately

## Non-Goals

**Not attempted in this milestone:**

- Re-running any evaluations — this is a display/export integration; existing eval data is the source of truth
- Changing the sweet-spot bucketing thresholds (60s slow threshold, etc.) — those live in `SweetSpotOpts` and can be tuned later
- Adding new charts beyond the 3 named components — keep scope tight; experimental visualizations can follow if usage data shows demand
- Mobile-responsive design changes — the existing dashboard isn't mobile-first; this doc doesn't try to fix that

## Timeline

**Day 1** (8h): Phase 1 — embed sweet-spot in latest.json, golden test, manual regen
**Day 2** (8h): Phase 2 (parity test) + Phase 3 start (DollarsPerPassTable + BenchmarkChampionsTable)
**Day 3** (6h): Phase 3 cont'd (FailureCategoryBars, QualityScatter extension) + Phase 4 (docs, screenshots, CHANGELOG)

**Total: ~22 hours / 3 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Schema drift between CLI sweet-spot output and dashboard JSON | Med | `sweet_spot_parity_test` blocks merges that introduce drift |
| Backward compatibility with pre-v0.19.0 baselines | Low | Lazy categorization at load time; missing fields default to zero/empty |
| Numerical inconsistency due to JS float vs Go float64 | Low | Test asserts equality to 4 decimal places, not bit-exact |
| Sibling M-BENCHMARK-DATA-INTEGRITY changes the schema mid-flight | Med | Coordinate via shared Issue #G note; align on `ShouldExcludeFromCapability` as the canonical denominator filter |
| Build/render performance regression | Low | New components only render data already in JSON; no new network or compute. Bundle-size delta should be small |

## Related Documents

**Implemented (informs design):**
- [M-EVAL-SWEET-SPOT (v0.19.0)](../../implemented/v0_19_0/m-eval-sweet-spot.md) — typed categories, BuildSweetSpot, ASCII frontier
- [M-EVAL-COST-AND-SPEED-BUDGETS (v0.15.1)](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) — `efficiency` block schema this layers on top of

**Planned (sibling):**
- [M-BENCHMARK-DATA-INTEGRITY](./m-benchmark-data-integrity.md) — Issue #4 (adjusted-rate denominator) coordinates with this doc's typed-category exclusion set
- M-EVAL-SWEET-SPOT-FOLLOWUP (open items in CHANGELOG):
  - FormatMatrix efficiency-column extension
  - eval-compare ΔTTS / Δcost-killed deltas
  - step_exhausted emission from CLI subprocess executors

## References

- [Design Axioms](/docs/references/axioms)
- [internal/eval_analysis/sweet_spot.go](../../../internal/eval_analysis/sweet_spot.go) — `BuildSweetSpot`, `FormatSweetSpotJSON`
- [internal/eval_analysis/export_json.go](../../../internal/eval_analysis/export_json.go) — current `latest.json` export
- [docs/src/components/BenchmarkDashboard/QualityScatter.jsx](../../../docs/src/components/BenchmarkDashboard/QualityScatter.jsx) — existing Pareto chart
- [docs/src/components/BenchmarkDashboard/ValueScoreTable.jsx](../../../docs/src/components/BenchmarkDashboard/ValueScoreTable.jsx) — existing value-score table to replace
- M-EVAL-SWEET-SPOT validation session results (56 runs, 2026-05-11): `eval_results/m_eval_sweet_spot_{validation,hard,stretch}/`

## Future Work

- **Per-tier sweet-spot data** — once M-BENCHMARK-DATA-INTEGRITY Issue #5 ships per-tier model stats, layer per-tier sweet-spot on top so the QualityScatter tier filter works without re-aggregating in JS
- **Time-series of $/pass** — track how each model's economics evolve across baselines (gemma-4-26b $0.473/pass today; how does it move with prompt improvements?)
- **Local-vs-remote cost overlay** — for models like gemma-4-26b where local hosting is viable, show "OpenRouter $0.473/pass | Local $0.00 + compute" side-by-side as a model-selection decision aid
- **Auto-flag dominated models** — JSX annotation on the leaderboard when a model is strictly dominated on cost AND speed (currently visible only as "dominated" badge in CLI output)

---

**Document created**: 2026-05-11
**Last updated**: 2026-05-11

**DESIGN_DOC_PATH**: `design_docs/planned/v0_19_0/m-eval-sweet-spot-website-integration.md`
