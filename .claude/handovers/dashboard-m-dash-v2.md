# Handover: M-DASH-V2 — Dashboard API reliability + tier/tag filtering

**Branch**: `dev` (uncommitted)
**Started**: 2026-04-21 session continuation
**Status**: Implementation complete; backend tests unverified; needs smoke-test + commit

---

## What this work does

Three interlocking dashboard enhancements the user asked for after the v0.13.0
post-release review:

1. **API reliability surfaced as first-class** — 13/33 `api_error` runs for
   gemini-3-1-pro in v0.13.0 were silently plotted as 0% alongside real
   code-quality failures. Now exposed as counts + rate, per-model, per-tier.
2. **Tier/tag filters propagate to every chart** — the TierToggle only
   affected 4 hero cards + the gallery; now it also swaps the bar charts,
   trend lines, and gap chart data sources. Added a TagFilter (12 canonical
   tags) that does the same filtering when no tier is selected.
3. **Suite-change events as annotations** — replaced the hardcoded
   `+5 contract benchmarks` ReferenceLine (duplicated across 3 chart files)
   with a data-driven system sourced from `benchmarks/events.yml`. Every
   taxonomy/add/remove change is now a chart annotation.

User's original asks (key quotes):
- "can we perhaps add to our website benchmarks a count on API errors … if
  we get 0% … can we omit from plot so it doesnt look werird?"
- "the core and stretch update numbers but perhaps they can update more
  numbers down the page? and allow us to filter graphs?"
- "can you also add like we did with contract benchmark additions when we
  update things like this tiering as events we can track in the plots"

---

## Backend changes (all done)

| File | Change |
|------|--------|
| [internal/eval_analysis/types.go](../../internal/eval_analysis/types.go) | Added `ModelDimensionStats`, `TierHistoryPoint`, `SuiteEvent`. Extended `TierAggregate` with `ModelStats`, API-error counters, repair-delta, avg-cost. Added `Tags`, `Events` to `DashboardJSON`. Added `Tiers` per `HistoryEntry`. YAML tags added to `SuiteEvent`. |
| [internal/eval_analysis/export_json_matrix.go](../../internal/eval_analysis/export_json_matrix.go) (NEW, 513 lines) | `buildTierModelMatrix`, `buildTagModelMatrix`, `computeTierExtras`, `computeReliability`, `buildHistoricalTierPoints`, `buildTierAggregates`, `buildTagAggregates`. |
| [internal/eval_analysis/events.go](../../internal/eval_analysis/events.go) (NEW) | `LoadSuiteEvents(path) ([]SuiteEvent, error)`. Missing file returns `(nil, nil)`. |
| [internal/eval_analysis/export_json.go](../../internal/eval_analysis/export_json.go) | Delegates to helpers. Loads events.yml. Adds `apiErrorCount/apiErrorRate/refusalCount/refusalRate` to aggregates. Adds `Tags`, `Events` to dashboard struct. Wires historical tier points into both historic and current baseline loops. Net +5 lines (821 total, ~21 over soft limit). |
| [benchmarks/events.yml](../../benchmarks/events.yml) (NEW) | Canonical suite-change log. Seeded with v0.9.1.1 contract benchmarks + v0.14.0 taxonomy + v0.14.0 stretch additions. |
| [internal/eval_analysis/events_test.go](../../internal/eval_analysis/events_test.go) (NEW) | 3 tests (missing file, valid YAML, missing required field). |
| [internal/eval_analysis/export_json_matrix_test.go](../../internal/eval_analysis/export_json_matrix_test.go) (NEW) | 5 tests covering reliability counts, tier matrix, tier extras, historical points, tag smoke. |

**Dashboard JSON regenerated** — verified via jq that new keys exist:
- `.aggregates.apiErrorCount` = 18
- `.aggregates.refusalCount` = 0
- `.tiers.core.model_stats.gemini-3-1-pro.ailang` has `apiErrorCount: 13`
- `.tags` populated (12 tags)
- `.events` array populated from events.yml
- `.history[-1].tiers.core.modelStats` populated retroactively

---

## Frontend changes (all done)

**New components:**
- [docs/src/components/BenchmarkDashboard/useEvents.js](../../docs/src/components/BenchmarkDashboard/useEvents.js) — shared hook; replaces 3 duplicate `VERSION_ANNOTATIONS` consts. Filters by `kinds` and `selectedTier`.
- [docs/src/components/BenchmarkDashboard/TagFilter.jsx](../../docs/src/components/BenchmarkDashboard/TagFilter.jsx) — mirrors TierToggle; disabled while a tier is selected.
- [docs/src/components/BenchmarkDashboard/ReliabilityCard.jsx](../../docs/src/components/BenchmarkDashboard/ReliabilityCard.jsx) — headline reliability % + expandable per-model breakdown.

**Modified:**
- [index.jsx](../../docs/src/components/BenchmarkDashboard/index.jsx) — added `selectedTag` state, `buildTierScopedModels`/`buildTierScopedLanguages` helpers, rendered TagFilter + ReliabilityCard, swapped ModelChart/LanguageChart/ModelTokenChart/ModelComparisonTable to use `scopedModels`/`scopedLanguages`, added tier headlines to section titles, filtered gallery by tier+tag.
- [PerModelTrend.jsx](../../docs/src/components/BenchmarkDashboard/PerModelTrend.jsx) — tier-scoped history via `entry.tiers[t].modelStats`. **API-error 0% gate**: when `apiErrorCount/totalRuns >= 0.5`, point = `null` (connectNulls bridges). Tooltip shows "— (API errors: N/M)".
- [ModelDeltaTrend.jsx](../../docs/src/components/BenchmarkDashboard/ModelDeltaTrend.jsx) — tier-scoped + same API-error gate. Filters events to add/remove/prompt kinds (taxonomy events don't shift per-model delta).
- [SuccessTrend.jsx](../../docs/src/components/BenchmarkDashboard/SuccessTrend.jsx) — prefers `baseline.tiers[t]` when selectedTier is set.
- [styles.module.css](../../docs/src/components/BenchmarkDashboard/styles.module.css) — added `.tierToggleDisabled` + full `.reliabilityCard*` block.

**Build verified**: `cd docs && npm run build` passes with only a pre-existing broken-anchor warning on serve-api#binary-response.

---

## Docs changes

- [benchmarks/CURATION.md](../../benchmarks/CURATION.md) — added **§8 Suite-change events** (`events.yml`) — when to record events, kind table, schema, authoring workflow, rule-of-thumb.

---

## What's LEFT before commit

### 1. Backend tests unverified
The user interrupted `go test ./internal/eval_analysis/...` so I never
confirmed the 8 new tests pass end-to-end. **First thing** in the next
session:
```bash
go test ./internal/eval_analysis/... -count=1 -v 2>&1 | tail -40
make test 2>&1 | tail -20
```
If anything fails, check:
- `TestLoadSuiteEventsValid` — needed yaml tags on `SuiteEvent`; if it
  still fails, something else is wrong with the yaml parser wiring.
- `TestComputeReliability` — hand-tuned against the 13/33 gemini-3-1-pro
  pattern; should pass.

### 2. `make check-file-sizes`
`export_json.go` is 821 lines (was 826 before this work). **Check if the
soft limit fails**; if so, the extra logic in `buildTierAggregates` /
`buildTagAggregates` inside export_json.go (currently inlined at the call
site) can be moved fully into `export_json_matrix.go`.

```bash
make check-file-sizes
```

### 3. Manual click-through
Not yet done. Start `npm start` in `docs/`:
```bash
cd docs && npm start
```
Verify:
- TierToggle updates the ModelChart, LanguageChart, ModelTokenChart, and
  both trend charts (not just the hero row + gallery).
- TagFilter narrows the gallery; disabled while a tier is selected.
- ReliabilityCard shows 18 total API errors, expandable to per-model
  breakdown (gemini-3-1-pro = 13 ailang, 0 python).
- PerModelTrend: switch to AILANG, look at v0.13.0 — gemini-3-1-pro should
  have a GAP, not a 0% dot. Hover tooltip should say
  "gemini-3-1-pro: — (API errors: 13/33)".
- Every trend chart shows the v0.9.1.1 `+5 contract benchmarks` line AND
  the v0.14.0 `Tier + tag taxonomy` line, sourced from events.yml (not
  the removed hardcoded const).

### 4. Commit
Once tests + UI verified:
```bash
git add internal/eval_analysis/ benchmarks/events.yml benchmarks/CURATION.md \
  docs/src/components/BenchmarkDashboard/ docs/static/benchmarks/latest.json
git status  # verify nothing unexpected
git diff --stat HEAD
```
Suggested commit message:
```
feat(dashboard): API reliability card + tier/tag filtering + suite events

- Backend: api_error/refusal counters in aggregates + per-tier + per-model.
  New tiers.{t}.model_stats cross-section lets charts filter retroactively.
  New events.yml canonical suite-change log (benchmark_add/remove, taxonomy).
- Frontend: ReliabilityCard surfaces 18 API errors with per-model expand;
  TagFilter mirrors TierToggle for the 12-tag taxonomy (disabled under tier).
  Bar/line/trend charts all re-render when TierToggle changes, not just the
  hero row. PerModelTrend gates 0% dots when majority-infra-failure.
- Docs: CURATION.md §8 documents the events.yml authoring workflow.
```

Tag to track this sprint: `M-DASH-V2`.

---

## Open questions for user confirmation

1. Should ReliabilityCard show reliability % even when `apiErrorCount == 0`
   (i.e. the non-warn, green "100%" state)? Currently yes — it renders as a
   small success-state card. If user prefers to hide the card entirely when
   nothing to report, change the `hasIssues` check to `return null`.

2. The tier-scoped model cost in ModelTokenChart uses the *global*
   `aggregates.totalCostUSD` as a fallback since per-tier per-model cost
   isn't in the aggregate. This is labelled "close-enough proxy" in the
   code comment. If accuracy matters, we'd need to expand
   `ModelDimensionStats` to carry `totalCostUSD`.

3. TagFilter is disabled under an active tier (per original plan — to
   avoid the 4-way tier×tag cross-product which the backend doesn't
   support yet). If user wants tag+tier combined, we'd need to add
   `tiers[t].tags[tag]` aggregates in `export_json_matrix.go`.

---

## Design doc

No separate design doc was written — work was driven by three in-session
ExitPlanMode approvals from the user. If this should be a retroactive
design doc (suggest `design_docs/implemented/v0_15_0/m-dash-v2.md`),
source material is:
- This handover
- The original plan in `/Users/mark/.claude/plans/memoized-dazzling-lantern.md`
- The user's three quoted asks above
