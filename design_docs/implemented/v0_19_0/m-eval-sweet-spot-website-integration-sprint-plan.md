# Sprint Plan: M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION

**Sprint ID**: M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION
**Design doc**: [m-eval-sweet-spot-website-integration.md](./m-eval-sweet-spot-website-integration.md)
**Target version**: v0.19.0
**Duration**: 3 working days (~22 hours)
**Risk level**: Low–Medium (data already exists end-to-end; this is a presentation layer integration)

## Goal

Surface every fact the CLI sweet-spot report produces (buckets, champions, $/pass, typed failures, Pareto frontier membership) onto the public benchmark dashboard, with numerical parity enforced by CI. Headline visibility goal: **the 12.4× deepseek-vs-gemma $/pass spread visible on first dashboard load.**

## Discovery (from prior session)

- M-EVAL-SWEET-SPOT shipped 11 commits this session including ASCII Pareto frontier
- `BuildSweetSpot` already produces every datum we need; this sprint is purely about wiring it into `latest.json` + 3 new JSX components
- The 56-run validation dataset (`eval_results/m_eval_sweet_spot_{validation,hard,stretch}/`) provides realistic test data
- M-BENCHMARK-DATA-INTEGRITY (sibling, planned) overlaps on Issue #4 — we coordinate via shared `ShouldExcludeFromCapability` semantics

## Velocity check

Recent comparable work this session: 11 commits in M-EVAL-SWEET-SPOT main sprint + M2-P1 follow-up = ~1400 LOC core, ~700 LOC tests over a single working day. Sustained Go velocity is ~250-400 LOC/day for eval-side code. JSX components are quicker (~300-500 LOC/day for plain Recharts/table components reading pre-computed JSON). Sprint target: ~600 LOC Go + ~450 LOC JSX = ~1050 LOC across 3 days = within velocity.

## Milestones

### M1: Embed Sweet-Spot in latest.json (~8h, ~300 LOC + 250 test LOC)

**Deliverables:**
- `internal/eval_analysis/export_json.go`: per-model `sweet_spot` block, top-level `sweet_spot_global`
- Hoist `BuildSweetSpot` call so MDX export and JSON export share one `SweetSpotReport`
- `export_json_sweet_spot_test.go`: golden test against the validation fixture data
- Backward-compat: pre-v0.19.0 result JSONs (no `finish_reason`) still load; bucket counts default to zero/empty

**Acceptance criteria:**
- `latest.json` per-model entry includes `sweet_spot: { pass_rate, median_tts_ms, p90_cost_per_success, speed_efficiency, dollars_per_pass, pareto_frontier, buckets, finish_reasons, error_categories }`
- Top-level `sweet_spot_global.champions[]` populated with cheapest/fastest pass per benchmark
- Golden test passes against the 56-run validation dataset
- Re-running `ailang eval-report` on a legacy baseline (no FinishReason fields) produces valid JSON (zero counts)
- `make test` clean

**Risks:**
- Existing dashboard JSX consumers assume the current `latest.json` shape — additive change minimizes risk, but verify nothing breaks

### M2: Numerical-Parity Test (~3h, ~150 test LOC)

**Deliverables:**
- `internal/eval_analysis/sweet_spot_parity_test.go`: asserts CLI sweet-spot output matches embedded JSON for every per-model field
- Operational drift guard: any change to `BuildSweetSpot` OR `export_json.go` that produces inconsistent numbers fails CI

**Acceptance criteria:**
- Test compares: pass_rate, median_tts_ms, p90_cost_per_success, speed_efficiency, all bucket counts, all champion entries
- All comparisons to 4 decimal places (floats), exact equality (ints / strings)
- Runs against the M-EVAL-SWEET-SPOT validation dataset checked into `eval_results/`
- Added to `make ci`

**Risks:**
- JSON marshaling could introduce float precision noise — mitigation: round to 4 decimals before comparing

### M3: $/Pass + Champions Tables (~6h, ~350 JSX LOC)

**Deliverables:**
- `docs/src/components/BenchmarkDashboard/DollarsPerPassTable.jsx`: sortable model × $/pass × pass-rate × total-spend table
- `docs/src/components/BenchmarkDashboard/BenchmarkChampionsTable.jsx`: per-benchmark cheapest/fastest pass winners
- Wire both into `BenchmarkDashboard/index.jsx`
- Optional "show as ratio vs cheapest" toggle on $/pass (renders the headline 12.4× gemma-vs-deepseek number)

**Acceptance criteria:**
- Both components render without console errors against the validation `latest.json`
- $/pass table default-sorted ascending (cheapest model first)
- Champions table sortable by benchmark name; rows show cheapest+cost and fastest+TTS
- `cd docs && npm run build` succeeds
- Visual inspection on `npm start` confirms headline number visible without scrolling

**Risks:**
- Recharts version compatibility — these are tables, not charts; risk minimal

### M4: Failure-Category Bars + QualityScatter Tier Filter (~5h, ~250 JSX LOC + ~40 JSX modified)

**Deliverables:**
- `docs/src/components/BenchmarkDashboard/FailureCategoryBars.jsx`: stacked horizontal bars per model. Color-coded by family (capability / budget / provider / success)
- Extend `docs/src/components/BenchmarkDashboard/QualityScatter.jsx`: bucket-color points + tier-filter dropdown
- Wire `FailureCategoryBars` into a new "Failure Modes" section in `index.jsx`

**Acceptance criteria:**
- FailureCategoryBars renders the categorized breakdown from `models[name].sweet_spot.error_categories`
- QualityScatter tier filter works (verified manually with v0.18.5 baseline data)
- Color legend visible and accessible
- `npm run build` succeeds

**Risks:**
- Per-tier sweet-spot data depends on M-BENCHMARK-DATA-INTEGRITY Issue #5 — for v0.19.0 if that doesn't ship, the tier filter falls back to "all" and we document the limitation

### M5: Docs + CHANGELOG (~3h, ~120 doc LOC)

**Deliverables:**
- `docs/docs/guides/evaluation/cost-and-speed-budgets.md`: "Reading the dashboard sweet-spot view" section
- 2-3 dashboard screenshots from the validation dataset embedded in the guide
- CHANGELOG entry under v0.19.0

**Acceptance criteria:**
- Docs guide describes each new component with a small annotated screenshot
- CHANGELOG entry under [Unreleased] / v0.19.0 references all 4 prior milestones
- `cd docs && npm run build` succeeds

**Risks:**
- Screenshots become stale — keep them generated from a stable fixture dataset committed to the repo

## Day-by-Day Plan

### Day 1 (8h)
- **AM (4h)**: M1 — JSON export extension + share BuildSweetSpot between MDX and JSON exporters
- **PM (4h)**: M1 cont'd — golden test against validation fixture; verify legacy baseline compatibility

### Day 2 (8h)
- **AM (3h)**: M2 — parity test + add to `make ci`
- **AM (1h)**: M3 start — `DollarsPerPassTable` scaffold
- **PM (4h)**: M3 cont'd — `BenchmarkChampionsTable` + wire both into index.jsx + ratio toggle

### Day 3 (6h)
- **AM (3h)**: M4 — `FailureCategoryBars` + QualityScatter tier-filter extension
- **AM (1h)**: M4 cont'd — wire into index.jsx
- **PM (2h)**: M5 — docs guide + CHANGELOG + screenshots

## Success Metrics

- [ ] `make test` clean; `make ci` clean
- [ ] `cd docs && npm run build` succeeds with no console errors
- [ ] `sweet_spot_parity_test` passes against the 56-run validation dataset
- [ ] Headline 12.4× $/pass ratio visible on first dashboard load (no clicks)
- [ ] All 7 issues from design doc Issue Index addressed (#A-#F; #G defers to sibling)
- [ ] CHANGELOG entry under v0.19.0

## Open Questions

1. **$/pass table placement**: top of `LanguageLeaderboard` or new dedicated section? (Proposal: top of leaderboard.)
2. **Color palette** for buckets: re-use existing chart palette or design new one? (Agent's call.)
3. **Per-tier sweet-spot data**: required for M4's tier filter. If M-BENCHMARK-DATA-INTEGRITY Issue #5 isn't ready, tier filter falls back to aggregate. Acceptable?

## Dependencies

- M2 depends on M1 (parity test needs JSON output to compare against)
- M3, M4 depend on M1 (JSX components read the new `sweet_spot` block)
- M5 depends on M3+M4 (screenshots need the components rendered)

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| JSON schema drift between CLI and dashboard | M2 parity test blocks merges with drift |
| Backward compat for pre-v0.19.0 baselines | Lazy default values; zero counts for missing fields |
| Per-tier data not available from sibling milestone | Tier filter falls back to "all"; documented in M5 |
| JS float precision noise | 4-decimal-place comparison in parity test |
| Dashboard bundle-size regression | New components are pure tables/bars reading existing JSON; minimal bundle impact |

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_19_0/m-eval-sweet-spot-website-integration-sprint-plan.md`
