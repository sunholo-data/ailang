# M-BENCHMARK-DATA-INTEGRITY: Benchmark Dashboard Data Integrity Audit

**Status**: Planned
**Target**: v0.19.0
**Priority**: P0 — High (blocks publishing motoko results; current numbers untrustworthy)
**Estimated**: 5 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Dedup + intersection logic makes dashboard fully reproducible from same input |
| A2: Replayability | +1 | Traceable methodology means same eval data always yields same dashboard |
| A3: Effect Legibility | 0 | No impact — dashboard is read-only |
| A4: Explicit Authority | 0 | No impact |
| A5: Bounded Verification | +1 | Per-benchmark provenance enables local spot-checking |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Correct denominators make automated regression detection reliable |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +1 | Per-run cost numbers become trustworthy |
| A10: Composability | 0 | No impact |
| A11: Structured Failure | +1 | Error categories split into harness_crash vs quota_error |
| A12: System Boundary | 0 | No impact |

**Net Score: +6** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

The benchmark dashboard (`docs/static/benchmarks/latest.json` + BenchmarkExplorer UI) contains at least 7 data integrity issues discovered during the v0.18.4 motoko publishing session. The numbers are not trustworthy enough to publish.

**Current State:**

| Issue | Symptom | Root Cause |
|-------|---------|-----------|
| #1 (Fixed) | motoko showed 52% pass rate; actual was 95% | `LoadResults` included all 209 runs (debug + final); no dedup |
| #2 (Fixed) | claude-haiku baseline in CrossHarnessTable was 45% | `successRate` blends standard+agent; should use `agentSuccessRate` |
| #3 (Open) | models ran 38/49/76 benchmarks; deltas compare apples to oranges | No intersection-based comparison within a harness family |
| #4 (Open) | Gemini stretch AILANG shows 100% adjusted from 9% raw | `successRateAdjusted = raw / (1 - apiErrorRate)` applied to harness crashes (91% error rate = divide by 0.09) |
| #5 (Open) | Tier filter shows correct harness rows but stale model sub-rows | No per-tier data in `models[name].languages[lang].tiers` |
| #6 (Open) | Avg cost/run column doesn't update on tier filter | `hr.avg_cost_usd` is always the overall average |
| #7 (Open) | `latest.json` write fails if binary runs from wrong dir | Output path is hardcoded relative: `docs/static/benchmarks/latest.json` |

**Impact:**
- Publishing incorrect numbers would undermine AILANG's benchmarking credibility
- The 100% adjusted rate for Gemini stretch (computed from 91% harness crashes) is the most visible symptom
- Cross-harness comparisons are biased toward harnesses that only ran easier benchmarks

## Goals

**Primary Goal:** Every number shown in the benchmark dashboard is traceable to a documented, correct methodology.

**Success Metrics:**
- Cross-harness pass rate deltas only computed over shared benchmark sets (or intersection clearly noted in UI)
- `successRateAdjusted` only shown when `apiErrorRate < 0.5`; harness crashes excluded from api_error count
- Tier filter updates ALL visible numbers (aggregate rows AND model sub-rows AND cost column)
- A "data provenance" tooltip on each metric documents what runs are included
- `ailang eval-report` works correctly regardless of current working directory

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Error category split schema | Changes eval result JSON schema; all parsers must update | human | design | high |
| Intersection benchmark set definition | How we define "comparable" across harness variants (same model family?) | human | design | med |
| Adjusted rate threshold (hide vs warn at X% api_error) | Affects trust signals for all harnesses | human | design | low |
| Output path for latest.json (flag vs env var vs config) | CLI interface contract | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Error category schema**: which field name? (`error_subcategory`? `harness_error`?) and what values (`harness_crash`, `quota_error`, `other`)
- [ ] **Intersection strategy**: is "family" defined by model (haiku variants together) or by exact benchmark set? Do we show intersection count in UI?

## Solution Design

### Issue #3: Intersection-Based Cross-Harness Comparison

**Problem**: opencode-haiku ran 49 benchmarks; motoko-haiku ran 38; haiku-direct ran 76. A harness that only ran smoke benchmarks looks better than one that ran all tiers.

**Proposed fix:**
1. In `buildHarnessAggregates`, group harness variants by model family (extract base model from `r.Model`)
2. Compute the intersection of benchmark IDs run by all variants in the family
3. Compute both overall rate AND intersection-restricted rate
4. Export both as `successRateAll` and `successRateIntersection` (with `intersectionSize: N`)
5. UI: CrossHarnessTable uses `successRateIntersection` when available; shows footnote "N benchmarks in common"

**Research needed**: How MLPerf and OpenLLM handle this (different eval subsets per system).

### Issue #4: API Error Adjustment Formula

**Problem**: `successRateAdjusted = successRate / (1 - apiErrorRate)`. When 91% of runs are api_errors (harness crashes), this gives `0.09 / 0.09 = 1.0` — mathematically trivial and meaningless.

**Root cause**: The eval harness doesn't distinguish harness crashes (agent binary failed to start / segfault / OOM) from quota/rate-limit errors (OpenRouter 429). Both get `error_category: api_error`.

**Proposed fix — schema change:**
Add `error_subcategory` field to `BenchmarkResult`:
```go
ErrorSubcategory string `json:"error_subcategory,omitempty"` // "quota_error" | "harness_crash" | ""
```

Detection heuristics (applied at categorization time in eval_harness):
- Exit code 1 + stderr contains "rate limit" / "quota" / "429" → `quota_error`
- Exit code != 0 + no meaningful stdout/stderr, or signal kill → `harness_crash`
- Otherwise when error_category=api_error → `other`

`successRateAdjusted` formula changes to exclude only `quota_error`:
```go
quotaErrors := // count where error_subcategory == "quota_error"
nonQuota := runs - quotaErrors
if nonQuota > 0 && quotaErrors > 0 {
    entry["successRateAdjusted"] = float64(success) / float64(nonQuota)
}
```

**UI short-term fix (before schema change):** Hide `successRateAdjusted` when `apiErrorRate > 0.5` — the formula is unreliable at that threshold regardless of category.

### Issue #5: Per-Tier Model Sub-Row Stats

**Problem**: `models[haiku].languages[ailang].tiers` doesn't exist, so when tier filter is active, model sub-rows show all-tier numbers.

**Current workaround**: Hide model sub-rows when tier filter active.

**Proper fix**: In `ExportBenchmarkJSON`, add per-tier accumulation to the model/language loop (similar to what `buildHarnessAggregates` does for harnesses).

### Issue #6: Avg Cost/Run on Tier Filter

`hr.avg_cost_usd` is the overall average. When tier filter is active, `activeTierData.avgCost` should be used instead. Already available in `hr.tiers[activeTier][lang].avgCost`.

### Issue #7: Relative Output Path

```go
// internal/eval_analysis/export_json.go
// Change from:
outputPath := "docs/static/benchmarks/latest.json"
// To: accept as parameter with default resolved to absolute
```

Add `--output` flag to `ailang eval-report` defaulting to `$(pwd)/docs/static/benchmarks/latest.json` with early validation.

### Implementation Plan

**Phase 1: Immediate display fixes** (~4 hours)
- [ ] Hide `successRateAdjusted` in UI when `apiErrorRate > 0.5`
- [ ] Show tier-specific cost in Avg Cost/Run column when tier filter active
- [ ] Add `--output` flag to `eval-report` command; validate path is writable before running

**Phase 2: Error category split** (~1 day)
- [ ] Add `ErrorSubcategory` to `BenchmarkResult` struct
- [ ] Add detection heuristics in eval_harness result writer
- [ ] Update `buildHarnessAggregates` to use `quota_error` as the adjustment denominator
- [ ] Backfill heuristic in `LoadResult` for historical baselines (best-effort)
- [ ] Update tests

**Phase 3: Intersection-based cross-harness** (~1 day)
- [ ] Implement benchmark intersection grouping in `buildHarnessAggregates`
- [ ] Export `successRateIntersection` and `intersectionSize` per harness family
- [ ] Update CrossHarnessTable to use intersection rate + show footnote
- [ ] Research: read MLPerf approach to unequal benchmark sets

**Phase 4: Per-tier model stats** (~4 hours)
- [ ] Add tier accumulation to the model-language loop in `ExportBenchmarkJSON`
- [ ] Export `models[name].languages[lang].tiers[tier]`
- [ ] Un-hide model sub-rows in tier-filter mode; use tier-specific data

**Phase 5: Data provenance tooltips** (~4 hours)
- [ ] Add tooltip component to BenchmarkExplorer for each metric header
- [ ] Document: what runs are included, what denominator is used, what date range

### Files to Modify/Create

**Modified files:**
- `internal/eval_analysis/loader.go` — already has dedup (Issue #1 fixed)
- `internal/eval_analysis/export_json.go` — per-tier model stats, output path flag
- `internal/eval_analysis/export_json_executors.go` — intersection logic, quota_error adjustment
- `internal/eval_harness/result_writer.go` — `ErrorSubcategory` detection
- `internal/eval_analysis/types.go` — `ErrorSubcategory` field on `BenchmarkResult`
- `cmd/ailang/eval.go` — `--output` flag on `eval-report`
- `docs/src/components/BenchmarkExplorer/index.jsx` — adjusted rate threshold, tier cost, tooltips

## Examples

### Issue #4: Before/After Adjusted Rate

**Before (broken):**
```
Gemini Stretch AILANG: raw 9%, adjusted 100%  (api_error_rate=91%)
```

**After (fixed):**
```
Gemini Stretch AILANG: 9%  (no adjusted shown — api_error_rate 91% is above threshold)
Note: 10 of 11 runs were harness crashes (not quota errors); raw rate is accurate
```

### Issue #3: Before/After Cross-Harness Delta

**Before:**
```
haiku-claude: 87%  (76 benchmarks including hard stretch)
opencode-haiku: 78%  (49 benchmarks, mostly smoke/core)
Delta: -9pp  — misleading, different sets
```

**After:**
```
haiku-claude: 84%  }  both computed over 38 shared benchmarks
opencode-haiku: 81%  }
Delta: -3pp  ✓  (38 benchmarks in common)
```

## Success Criteria

- [ ] Gemini stretch no longer shows "100%" adjusted rate
- [ ] CrossHarnessTable deltas computed over intersection (or footnoted when sets differ)
- [ ] Tier filter updates cost column
- [ ] `ailang eval-report` with wrong CWD prints actionable error (not silent wrong path)
- [ ] Model sub-rows show tier-specific data when tier filter is active
- [ ] All existing eval_analysis tests pass
- [ ] `make ci` passes

## Testing Strategy

**Unit tests:**
- `export_json_executors_test.go`: intersection grouping with 3 harness variants of different sizes
- `export_json_executors_test.go`: adjusted rate formula with subcategory filtering
- `loader_test.go`: dedup preserves latest, drops older re-runs

**Integration tests:**
- Regenerate `latest.json` from known fixture inputs; golden-file diff against expected output

**Manual testing:**
- Select Stretch tier: Gemini row should show raw rate, no adjusted
- Deselect tier: model sub-rows reappear with all-tier numbers
- Run `ailang eval-report` from `/tmp`: should error with clear message

## Deferred Decisions

- **Tooltip UX design** (wording, placement) — agent may choose
- **Backfill strategy for historical `error_subcategory`** — agent may use best-effort heuristics; don't block on perfect historical data
- **Intersection footnote formatting** — agent may choose (e.g., "38 shared" vs "⌂ intersection: 38")

## Non-Goals

- **Re-running any benchmarks** — this is a display/export fix only; existing eval data is the source of truth
- **Changing the eval harness scoring logic** — we only split categories, not change pass/fail criteria
- **Retroactively re-categorizing all historical runs perfectly** — heuristic backfill is good enough

## Timeline

**Day 1** (Phase 1, 4h): Immediate UI fixes — hide bad adjusted rates, tier-cost column, output path flag
**Day 2** (Phase 2, 8h): Error subcategory schema + harness detection
**Day 3** (Phase 3, 8h): Intersection cross-harness logic + research
**Day 4** (Phase 4, 4h): Per-tier model sub-row stats
**Day 5** (Phase 5 + polish, 4h): Tooltips, cleanup, `make ci`

**Total: ~5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Historical baselines can't be backfilled reliably | Med | Use heuristics; mark uncertain rows as `error_subcategory: ""` (unknown); don't adjust those |
| Intersection logic groups wrong model variants together | Med | Test with known fixture; let human approve grouping key before shipping |
| Adjusted rate threshold (0.5) is arbitrary | Low | Document the threshold; add a `(⚠ high error rate)` tooltip instead of silently hiding |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_11_0/m-eval-cross-language-benchmark-sprint-plan.md](design_docs/implemented/v0_11_0/m-eval-cross-language-benchmark-sprint-plan.md) — Cross-language eval infrastructure
- [design_docs/implemented/v0_14_0/m-eval-category-analysis.md](design_docs/implemented/v0_14_0/m-eval-category-analysis.md) — Error category analysis prior art

**Planned (check for overlap):**
- [design_docs/planned/v0_15_0/m-benchmark-section-redesign.md](design_docs/planned/v0_15_0/m-benchmark-section-redesign.md) — Broader dashboard redesign
- [design_docs/planned/v0_15_0/m-eval-trust-signals.md](design_docs/planned/v0_15_0/m-eval-trust-signals.md) — Trust signal framework (directly related)

## References

- [Design Axioms](/docs/references/axioms)
- MLPerf Results Methodology: https://mlcommons.org/en/policies/mlperf-results-policies/ (research needed)
- OpenLLM Leaderboard methodology notes (research needed)

## Future Work

- Automated regression alerts when adjusted rate diverges from raw by > 20pp
- Per-benchmark provenance page (show which runs contributed to each data point)

---

**Document created**: 2026-05-09
**Last updated**: 2026-05-09
