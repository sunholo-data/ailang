# M-BENCHMARK-SECTION: Multi-Page Benchmark Section for AILANG Website

**Status**: Planned
**Target**: v0.15.x
**Priority**: P2 (Strategic — needed as dimension count grows)
**Estimated**: 4 days
**Dependencies**: M-EVAL-CROSS-HARNESS (model_family field + harness_suite), M-OLLAMA-LOCAL-EVAL (local model results)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a **website/display layer** change. Most axioms are neutral as it doesn't affect AILANG language semantics.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language runtime |
| A2: Replayability | 0 | No change to eval pipeline |
| A3: Effect Legibility | 0 | Website code, no AILANG effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | **+2** | Structured benchmark data makes AILANG quality legible to AI researchers; cross-harness display reveals which harness best serves machine synthesis |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | **+1** | Harness cost-per-run is surfaced as a first-class metric; cloud vs local model cost delta now visible |
| A10: Composability | **+1** | Dimension-based pages compose — a new language or harness appears automatically via dynamic JSON data |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No new boundary crossings |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Explicitly improves machine-readability of benchmark quality signal

---

## Problem Statement

**Current State:**

The AILANG website has a single benchmark page (`docs/docs/benchmarks/performance.md`) built for a 2-language (AILANG + Python), 2-harness (claude + gemini) world. It loads a 484KB static JSON and renders all dimensions on one page through 15 React components.

As of v0.14.1, the eval system has expanded to:
- **4 languages**: ailang, python, javascript, go — with more planned
- **4+ executor harnesses**: claude, gemini, codex, opencode
- **30+ model variants** across cloud and local (Ollama) providers
- **2 eval modes**: standard (0-shot + repair) and agent
- **Dual timeout model** for local vs cloud models (ttft_timeout + generation_timeout)

**Concrete problems today:**
- The model comparison table already overflows on laptop screens with 8 models; 30+ is unworkable
- Local Ollama models can't be compared fairly against cloud models on the same table (scaled timeouts, different cost model)
- `opencode-haiku` and `claude-haiku` represent the same model via different harnesses but appear as unrelated rows
- JavaScript and Go language results are ready to stream in but the page has no place to display them alongside Python and AILANG
- The page has no way to answer "which harness is best for this model?" or "which language is easiest for AI to write?"

**Impact:**
- Benchmark data becomes less actionable as dimension count grows
- External researchers can't compare across dimensions without downloading and parsing `latest.json` manually
- The page will become unnavigable before v0.16

---

## Goals

**Primary Goal:** Replace the single benchmark page with a multi-page benchmark section organised by dimension (language, model, harness), where each view gives a focused, comparable answer to one question.

**Success Metrics:**
- 4 new pages render with real data from `latest.json` (overview, by-language, by-model, by-harness)
- JS and Go appear in the language view automatically once results land in baselines — zero code change required
- Same model running under two harnesses is grouped and shows a delta row on the by-harness page
- Local (Ollama) models are visually distinguished from cloud models with a badge
- Docusaurus build passes with no regressions to existing pages

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `latest.json` extended vs new file | Determines backward compat; extending is safer but risks schema bloat | human | design | med |
| `by-harness` page scope — cloud only or cloud+local | Local models (Ollama) have scaled timeouts; mixing them with cloud baseline may mislead | human | design | med |
| `performance.md` fate — fold into overview or keep | Affects URL stability (existing external links may break if removed) | human | design | low |
| Dashboard JSON `harnesses` key — flat or nested under `executors` | Determines how components query the data; restructuring later is expensive | agent | design | med |
| Language filter UI — dropdown or tab | Visual scope changes all charts vs just filtering a table; tab is simpler for 4 languages | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **`latest.json` extension strategy**: extend existing `latest.json` schema (add `harnesses` key, `provider_type` on model entries) vs produce a new `benchmarks/multi.json`. **Recommendation: extend `latest.json`** — components already load it and it avoids a second fetch.
- [ ] **`by-harness` page local model inclusion**: cloud-only harness comparison (apples-to-apples), or include Ollama with a `⏱×N` timeout-scale badge. **Recommendation: include local with badge** — excluding them hides real data; the badge makes the caveat visible.

---

## Solution Design

### Overview

The current benchmark section (2 pages) becomes 6 pages. Each page answers one comparative question via focused charts and tables. All pages share the same `latest.json` static file — no new data pipeline runs needed. The backend adds two new JSON keys (`harnesses`, `provider_type`) to the existing export; the frontend adds 4 new React components and 4 new `.md` page stubs.

### Architecture

```
docs/static/benchmarks/latest.json  (extended — +harnesses, +provider_type)
         │
         ├── overview.md ──────── BenchmarkOverview.jsx
         │                          LanguageSummaryCard × N  (dynamic)
         │                          HarnessSummaryCard × N   (dynamic)
         │                          ModelTopN.jsx (top 5, any tier)
         │                          SuccessTrend.jsx (existing)
         │
         ├── by-language.md ───── LanguageLeaderboard.jsx (NEW)
         │                          rows: languages, cols: models, cells: pass rate heatmap
         │                          DimensionSelector.jsx (NEW) — tier + tag filter
         │
         ├── by-model.md ──────── ModelLeaderboard.jsx (extends ModelComparisonTable)
         │                          ModelRadarComparison.jsx (existing)
         │                          DimensionSelector.jsx — language + harness filter
         │
         ├── by-harness.md ────── HarnessComparisonTable.jsx (NEW)
         │                          groups by model_family, shows harness delta rows
         │                          LocalCloudBadge.jsx (NEW)
         │
         ├── performance.md ───── BenchmarkDashboard.jsx (existing, renamed "Standard Eval Details")
         │
         └── codebase-stats.mdx ─ CodebaseStats.jsx (unchanged)
```

### Components

**New components** (`docs/src/components/`):

1. **`BenchmarkOverview/index.jsx`** — Landing page. Shows headline metrics (core tier success rate, token efficiency), one card per language, one card per harness, link to detail pages. Reads `languages`, `harnesses`, `tiers.core` from JSON.

2. **`LanguageLeaderboard/index.jsx`** — Heatmap table: rows = languages, columns = model families, cells = pass rate (colour-coded green→red). Includes a row for each language registered in JSON (zero config for new languages). Reuses `DimensionSelector` for tier/tag filtering.

3. **`HarnessComparisonTable/index.jsx`** — Table grouped by `model_family`. For each family with ≥2 harness entries, renders a cluster: one row per harness + a Δ row (pass diff, cost diff, duration diff). Uses `harnesses` section from JSON. Shows `LocalCloudBadge` for Ollama entries.

4. **`LocalCloudBadge/index.jsx`** — Tiny indicator: `☁ Cloud` or `🖥 Local (×N)` where N is timeout_scale. Inlined in tables and model cards wherever `provider_type` is present.

5. **`DimensionSelector/index.jsx`** — Shared filter bar (tier toggle + language multi-select + harness multi-select). Props: `dimensions`, `selected`, `onChange`. Used by by-language, by-model, and by-harness pages.

**Extended existing components:**

- **`ModelComparisonTable.jsx`** — Add `groupByFamily` prop; when true, collapse entries with the same `model_family` into collapsible groups.
- **`BenchmarkDashboard/index.jsx`** — Add `view` prop (`"overview" | "language" | "model" | "harness" | "standard"`) to select which sub-components render, avoiding loading all 15 on every page.

### Data Pipeline Changes

**File: `internal/eval_analysis/export_json.go`**

Add `harnesses` section to `DashboardJSON` struct and populate in `BuildDashboard()`:

```go
type DashboardJSON struct {
    // ... existing fields ...
    Harnesses map[string]*HarnessAggregate `json:"harnesses"`
}

type HarnessAggregate struct {
    Name          string                       `json:"name"`          // "claude", "gemini", "opencode", "codex"
    DisplayName   string                       `json:"display_name"`  // "Claude Code CLI"
    Models        []string                     `json:"models"`        // model keys using this harness
    TotalRuns     int                          `json:"total_runs"`
    SuccessRate   float64                      `json:"success_rate"`
    AvgCostUSD    float64                      `json:"avg_cost_usd"`
    AvgDurationMs float64                      `json:"avg_duration_ms"`
    Languages     map[string]*HarnessLangStats `json:"languages"`     // per-language breakdown
}
```

**File: `internal/eval_analysis/export_json.go`** — add `provider_type` to model entries:

```go
// In existing model aggregate struct:
ProviderType string `json:"provider_type"` // "cloud" | "local"
TimeoutScale float64 `json:"timeout_scale,omitempty"` // >1.0 for Ollama
```

Populated by reading `ModelConfig.Provider` from `models.yml` (`provider: "ollama"` → `"local"`, all others → `"cloud"`).

**File: `internal/eval_analysis/export_json_executors.go`** — Surface timing data:

```go
type ExecutorLangStats struct {
    // ... existing fields ...
    AvgTTFTSeconds        float64 `json:"avg_ttft_seconds,omitempty"`
    AvgGenerationSeconds  float64 `json:"avg_generation_seconds,omitempty"`
}
```

### Implementation Plan

**Phase 1: Dashboard JSON extensions** (~1 day, ~120 LOC Go)
- [ ] Add `HarnessAggregate` struct to `internal/eval_analysis/types.go`
- [ ] Populate `harnesses` map in `BuildDashboard()` — group model results by `agent_cli` field from `ModelConfig`
- [ ] Add `provider_type` + `timeout_scale` to model aggregate entries (read from `ModelConfig.Provider`)
- [ ] Surface `avg_ttft_seconds`, `avg_generation_seconds` in executor/harness stats (from result `DurationMs` breakdown — if not already split, use `CompileMs` as proxy or skip for now)
- [ ] Run `ailang eval-report` and verify new keys appear in `docs/static/benchmarks/latest.json`
- [ ] Tests: `TestBuildDashboard` verifies `harnesses` key present and counts correct

**Phase 2: New page scaffolds + navigation** (~0.5 day)
- [ ] Create `docs/docs/benchmarks/overview.md` (stub importing `BenchmarkOverview`)
- [ ] Create `docs/docs/benchmarks/by-language.md` (stub importing `LanguageLeaderboard`)
- [ ] Create `docs/docs/benchmarks/by-model.md` (stub importing `ModelLeaderboard`)
- [ ] Create `docs/docs/benchmarks/by-harness.md` (stub importing `HarnessComparisonTable`)
- [ ] Update `docs/sidebars.js` — add 4 new items, rename `performance` label to "Standard Eval Details"
- [ ] Verify `cd docs && npm run build` passes with stubs

**Phase 3: New React components** (~2 days, ~600 LOC JSX)
- [ ] `DimensionSelector/index.jsx` — shared filter bar, used by 3 pages
- [ ] `LanguageLeaderboard/index.jsx` — heatmap table from `languages` + `models` JSON sections
- [ ] `HarnessComparisonTable/index.jsx` — grouped table from `harnesses` section, delta rows
- [ ] `LocalCloudBadge/index.jsx` — inline badge from `provider_type` field
- [ ] `BenchmarkOverview/index.jsx` — landing summary from all top-level sections
- [ ] Extend `ModelComparisonTable.jsx` with `groupByFamily` prop
- [ ] Wire all new components into their respective page stubs

**Phase 4: Connect + verify** (~0.5 day)
- [ ] Run `ailang eval-report` with latest baselines; confirm `latest.json` has `harnesses` key
- [ ] Verify language heatmap shows AILANG + Python; confirm JS/Go rows appear with `--` cells (no results yet)
- [ ] Verify harness table shows grouped rows for `claude-sonnet-4-6` / `opencode-sonnet-4-6` pair (once M-EVAL-CROSS-HARNESS runs provide result data)
- [ ] Check existing `/docs/benchmarks/performance` page still renders correctly
- [ ] `cd docs && npm run build` clean

### Files to Modify/Create

**New files:**
- `docs/docs/benchmarks/overview.md` — page stub, ~20 LOC
- `docs/docs/benchmarks/by-language.md` — page stub, ~20 LOC
- `docs/docs/benchmarks/by-model.md` — page stub, ~20 LOC
- `docs/docs/benchmarks/by-harness.md` — page stub, ~20 LOC
- `docs/src/components/BenchmarkOverview/index.jsx` — ~120 LOC
- `docs/src/components/LanguageLeaderboard/index.jsx` — ~180 LOC
- `docs/src/components/HarnessComparisonTable/index.jsx` — ~180 LOC
- `docs/src/components/LocalCloudBadge/index.jsx` — ~40 LOC
- `docs/src/components/DimensionSelector/index.jsx` — ~80 LOC

**Modified files:**
- `internal/eval_analysis/types.go` — add `HarnessAggregate`, `HarnessLangStats` structs (~40 LOC)
- `internal/eval_analysis/export_json.go` — populate `harnesses`, `provider_type`, `timeout_scale` (~80 LOC)
- `internal/eval_analysis/export_json_executors.go` — add timing fields (~20 LOC)
- `docs/sidebars.js` — add 4 items, rename existing label (~10 LOC)
- `docs/src/components/BenchmarkDashboard/index.jsx` — add `view` prop, conditional rendering (~30 LOC)
- `docs/src/components/ModelComparisonTable.jsx` — add `groupByFamily` prop (~40 LOC)

---

## Examples

### Example 1: By-Language page

The heatmap table at `/docs/benchmarks/by-language`:

```
Language     | claude-sonnet | gemini-flash | gpt5-mini | opencode-gemma4
─────────────|───────────────|──────────────|───────────|────────────────
ailang       |    86%  ████  |   74%  ███   |   78% ███ |  42% ██  🖥 ×5
python       |    86%  ████  |   76%  ███   |   81% ████|  38% █   🖥 ×5
javascript   |     —         |    —         |    —      |   —
go           |     —         |    —         |    —      |   —
```

As JS/Go baselines land, their rows populate automatically — zero frontend code change.

### Example 2: By-Harness page

The grouped table at `/docs/benchmarks/by-harness` (after M-EVAL-CROSS-HARNESS produces paired results):

```
Model Family: claude-sonnet-4-6
  Harness          | AILANG | Python | Avg   | Cost/run | Avg duration
  Claude CLI       |  86%   |  86%   | 86%   | $0.012   | 45s
  opencode         |  71%   |  86%   | 79%   | $0.009   | 78s
  Δ (opencode−CLI) |  −15%  |   0%   | −7%   | −$0.003  | +33s

Model Family: gemini-3-flash
  Harness          | AILANG | Python | Avg   | Cost/run | Avg duration
  Gemini CLI       |  74%   |  76%   | 75%   | $0.001   | 38s
  opencode         |  78%   |  79%   | 79%   | $0.001   | 52s
  Δ (opencode−CLI) |  +4%   |  +3%   | +4%   | $0.000   | +14s
```

### Example 3: Dashboard JSON schema additions

New keys in `docs/static/benchmarks/latest.json`:

```json
{
  "harnesses": {
    "claude": {
      "name": "claude",
      "display_name": "Claude Code CLI",
      "models": ["claude-sonnet-4-6", "claude-haiku-4-5", "claude-opus-4-7"],
      "total_runs": 264,
      "success_rate": 0.82,
      "avg_cost_usd": 0.014,
      "avg_duration_ms": 48200,
      "languages": {
        "ailang": { "success_rate": 0.86, "total_runs": 132 },
        "python": { "success_rate": 0.79, "total_runs": 132 }
      }
    },
    "opencode": {
      "name": "opencode",
      "display_name": "opencode CLI",
      "models": ["opencode-haiku", "opencode-sonnet-4-6", "opencode-gemma4-e4b"],
      "total_runs": 48,
      "success_rate": 0.71,
      "avg_cost_usd": 0.007,
      "avg_duration_ms": 76400
    }
  },
  "models": {
    "claude-sonnet-4-6": {
      "provider_type": "cloud",
      "timeout_scale": 1.0,
      "...": "..."
    },
    "opencode-gemma4-e4b": {
      "provider_type": "local",
      "timeout_scale": 5.0,
      "...": "..."
    }
  }
}
```

---

## Success Criteria

- [ ] `ailang eval-report` output contains `harnesses` top-level key with ≥1 harness entry
- [ ] `provider_type: "cloud"` on all non-Ollama model entries; `"local"` on Ollama entries
- [ ] `docs/sidebars.js` includes overview, by-language, by-model, by-harness items
- [ ] `/docs/benchmarks/overview` renders with real headline metrics (not placeholder)
- [ ] `/docs/benchmarks/by-language` shows AILANG and Python rows with live data; JS/Go rows show `—` (not an error)
- [ ] `/docs/benchmarks/by-harness` shows grouped clusters when `model_family` matches exist in results
- [ ] `/docs/benchmarks/performance` renders without regression
- [ ] `cd docs && npm run build` clean, no broken imports
- [ ] `make test ./internal/eval_analysis/...` green (new struct tests)

---

## Testing Strategy

**Unit tests (Go):**
- `TestBuildDashboardHarnesses` — verify `harnesses` key populated from mock result set with 2 harnesses
- `TestProviderType` — verify `provider_type: "local"` for Ollama models, `"cloud"` for all others
- `TestHarnessAggregateEmpty` — verify no panic when `harnesses` section has 0 results

**Integration tests:**
- Run `ailang eval-report eval_results/baselines/latest` and grep `latest.json` for `harnesses` key
- Verify JS/Go entries appear in `languages` section of `latest.json` once any JS/Go result files exist

**Manual testing (browser):**
- Open each new page and confirm charts render with real data
- Resize to mobile width — confirm tables don't overflow
- Confirm `LocalCloudBadge` appears on Ollama model rows
- Confirm `DimensionSelector` filter updates all charts on the page
- Confirm existing `/docs/benchmarks/performance` page unchanged

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Heatmap colour scale** (LanguageLeaderboard) — agent may choose scale breakpoints (e.g., red <50%, yellow 50–75%, green >75%)
- **Delta row colour** (HarnessComparisonTable) — agent may use green for positive Δ, red for negative, or plain text
- **Overview hero metric selection** — agent may choose which 3–4 metrics to feature on the overview landing (e.g., core-tier AILANG pass rate, best harness, cheapest model)
- **Mobile layout for heatmap** — agent may choose to collapse to a scrollable table or a simplified list on narrow screens
- **`timeout_scale` display precision** — agent may round to nearest integer or show one decimal

---

## Non-Goals

**Not attempted in this feature:**
- **Live/dynamic data** — the site remains statically generated; `latest.json` is regenerated on each `ailang eval-report` run, not live
- **Drill-down to individual benchmark results** — `BenchmarkGallery` already handles this on the existing performance page; not duplicated on new pages
- **Statistical significance / confidence intervals** — separate sprint (needs repeated run data)
- **Prompt-level diff between harnesses** — observability sprint; out of scope here
- **Adding new harnesses or languages** — this doc is purely display; new harnesses/languages are added via `models.yml` and `langreg`

---

## Timeline

**Day 1** (~8 hours):
- Phase 1: Go data pipeline changes, new JSON keys, tests

**Day 2** (~4 hours):
- Phase 2: Page scaffolds, sidebars.js, build check

**Days 3–4** (~8 hours):
- Phase 3: New React components
- Phase 4: Connect + verify, fix regressions

**Total: ~4 days (~20 hours)**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `harnesses` section empty if results lack `executor` field | High | Executor field has been populated since M-EXEC-OPENCODE; fallback: omit empty harnesses, show "no data" card |
| `model_family` not yet populated (M-EVAL-CROSS-HARNESS not done) | Med | HarnessComparisonTable degrades gracefully to ungrouped list when `model_family` absent |
| `latest.json` schema change breaks existing components | Med | Only additive keys — no existing keys renamed or removed; existing components ignore unknown keys |
| Docusaurus MDX import of new components fails | Low | Test `npm run build` after each component add; keep page stubs minimal until components are wired |
| Local Ollama result files present but incomplete (TTFT timeout) | Low | Partial results still have `lang`, `model`, `executor` — they contribute to harness pass rate correctly |

---

## Related Documents

**Dependencies (must precede or run concurrently):**
- [m-eval-cross-harness-comparison.md](m-eval-cross-harness-comparison.md) — provides `model_family` in result JSON and `harness_suite` composite; required for HarnessComparisonTable delta rows
- [m-eval-cross-harness-sprint-plan.md](m-eval-cross-harness-sprint-plan.md) — sprint plan for the above
- [m-ollama-local-eval.md](m-ollama-local-eval.md) — provides local Ollama results with timeout_scale metadata

**Implemented (inform design patterns):**
- [design_docs/implemented/v0_7_0/m-control-plane-interactive-filtering.md](../../implemented/v0_7_0/m-control-plane-interactive-filtering.md) — interactive filtering pattern in the Collaboration Hub UI; DimensionSelector follows same approach
- [design_docs/implemented/v0_8_1/m-perf3-performance-quick-wins.md](../../implemented/v0_8_1/m-perf3-performance-quick-wins.md) — benchmark data structure patterns

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-dashboard-simplification.md](../../planned/v0_13_0/m-dashboard-simplification.md) — Collaboration Hub UI simplification; independent but shares `view` prop decomposition pattern
- [design_docs/planned/v0_13_0/m-ui-refactor-ai-friendly.md](../../planned/v0_13_0/m-ui-refactor-ai-friendly.md) — file size targets for React components (≤500 lines); new components in this doc must comply

---

**Document created**: 2026-04-23
**Last updated**: 2026-04-23
