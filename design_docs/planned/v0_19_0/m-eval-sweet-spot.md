# M-EVAL-SWEET-SPOT: Eval Sweet-Spot Reporting & Failure-Category Distinguishability

**Status**: Planned
**Target**: v0.19.0
**Priority**: P1 (Medium — unblocks honest cross-model comparison, especially OpenRouter cheap-but-slow models)
**Estimated**: ~3 working days (~24 hours)
**Dependencies**:
- M-EVAL-COST-AND-SPEED-BUDGETS (v0.15.1, implemented) — provides `FirstAttemptMs`, `SuccessAtMs`, `TokensPerSec`, `CostKilledAt`, `EfficiencyAggregates`
- M-AI-OPENROUTER (v0.14.3 / v0.16.0) — error strings we categorize against
- Motoko executor `finish_reason` stream events (already emitted)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|------:|---------------|
| A1: Determinism | 0 | Eval-harness reporting only — no runtime semantics change |
| A2: Replayability | +1 | New `FinishReason` + sharper `ErrorCategory` enrich result JSONs for replay/audit |
| A3: Effect Legibility | 0 | n/a |
| A4: Explicit Authority | 0 | n/a |
| A5: Bounded Verification | 0 | n/a |
| A6: Safe Concurrency | 0 | n/a (read-side aggregations) |
| A7: Machines First | +2 | Failure-category disambiguation makes eval data **machine-decidable**: `quota_exhausted` ≠ `timeout` ≠ `step_exhausted`. Downstream agents (model-selectors, post-release dashboards) can now ingest cause without parsing stderr strings |
| A8: Minimal Syntax | 0 | No language syntax change |
| A9: Cost Visibility | +2 | Direct alignment — surfaces cost-vs-time-vs-success Pareto frontier in CLI + MDX (today only in JS dashboard); separates "model couldn't do it" from "we capped the budget" |
| A10: Composability | +1 | Reuses existing `ComputeEfficiency` and `CategorizeError` machinery; refactors duplicated `r.ErrorCategory == "api_error"` checks behind one helper |
| A11: Structured Failure | +2 | Replaces the conflated `api_error` bucket (51/67 agent failures in v0_18_*) with 5 distinct, typed reasons. Failure cause becomes a first-class observable |
| A12: System Boundary | +1 | Promotes `motoko_finish_reason` from opaque `ProviderData` into the canonical `RunMetrics.FinishReason` boundary field |

**Net Score: +9** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (all aggregations are pure functions of result JSONs)
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strict net win — failure cause becomes machine-decidable instead of stderr-string-decidable

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

**This doc: +9, no hard violations → proceed.**

## Problem Statement

We've been running evaluations against many OpenRouter-routed models and observe a recurring pattern: weak/cheap models pass at **~100%** when given enough wall-clock and turns; strong models pass faster but cost much more. The *interesting* boundary — **the sweet spot** — is invisible in our current reports.

Two compounding problems:

### Problem A — The reporting surfaces don't expose the data we already capture

v0.15.1's `M-EVAL-COST-AND-SPEED-BUDGETS` already records `first_attempt_ms`, `success_at_ms`, `tokens_per_sec`, `cost_killed_at`, `agent_turns` per result and computes `EfficiencyAggregates` per model. The JS dashboard (`SpeedRadar`, `QualityScatter`, `ValueScoreTable`) renders it. **But:**

| Surface | Status |
|---|---|
| Per-result JSON fields | ✅ Captured (v0.15.1) |
| Dashboard JSON `models[].efficiency` block | ✅ Emitted (`export_json.go:506`) |
| Docusaurus dashboard charts | ✅ Render Pareto frontier |
| **CLI `FormatMatrix` (text/markdown)** | ❌ No efficiency rows |
| **`eval-compare` text output** | ❌ No speed/cost-killed deltas |
| **"Sweet-spot" / bucketed view** (impossible / slow-pass / fast) | ❌ Does not exist anywhere |
| **Cross-model boundary table** ("cheapest model that passes if N turns / $X allowed") | ❌ Does not exist |

### Problem B — The failure taxonomy is broken, so any sweet-spot report built on it would lie

Audit of 67 agent-mode failures across `eval_results/v0_18_*/agent`:

| What actually happened | Where it lives today | Why this corrupts the analysis |
|---|---|---|
| **OpenRouter monthly quota kill** ("Key limit exceeded") | Lumped into `api_error` (51 of 67) with `duration_ms=0, agent_turns=0` | Indistinguishable from network timeout; pollutes "cheap model failed" narrative |
| HTTP timeout / 429 / 500 | Also `api_error` via `internal/eval_harness/ai_agent.go:110-137` `isRetryableError` | Same bucket as quota kill |
| Motoko **`cost_exhausted`** finish_reason | Captured in `internal/executor/motoko/parser.go:293` as `ProviderData["motoko_finish_reason"]` | `RunMetrics` doesn't carry `ProviderData` → signal lost; ends up as generic non-success |
| Wall-clock budget kill (v0.15.1 `CostKilledAt`) | Field exists in `metrics.go:80` | **Never set** in any agent result JSON in current dataset — the cost-budget primitive isn't wired through the agent executors |
| Agent ran out of turns | Currently appears as `logic_error` / `runtime_error` whenever last attempt fails | No `step_exhausted` category to identify "model gave up vs. harness cut it off" |
| Hard timeout (no API error, just clock ran out) | Lumped into `api_error` if it surfaces as an error string | Indistinguishable from quota |

Until A and B are both fixed, a sweet-spot report cannot honestly answer **"is this model cheap-but-needs-more-time?"** vs **"is this model out of capability?"**

**Current State:**

- 51 of 67 recent agent failures (76%) classify as `api_error` — undifferentiated.
- `CostKilledAt` schema field is unused in any executor.
- Motoko fixture `internal/executor/motoko/testdata/session_cost_exhausted.jsonl` exists but the signal is dropped at the `RunMetrics` boundary.
- Dashboard excludes all `api_error` indiscriminately (~10 inline `r.ErrorCategory == "api_error"` checks across `internal/eval_analysis/`) — quota-killed cheap models look better than they are.

**Impact:**

- **Who is affected?** Eval-suite curators, anyone consuming the public benchmark dashboard, downstream model-selection logic, blog posts about "AILANG vs Python".
- **How significant?** With 76% of failures unclassifiable, every sweet-spot ranking and cost-vs-speed comparison today is decorated noise. A user *cannot* tell from current reports whether bumping the cost budget would change a model's score.

## Goals

**Primary Goal:** Make it trivial — from any `eval_results/` run — to see (a) which model is the cheapest one that passes each benchmark, (b) which models would likely pass if given more turns/budget, and (c) where the cost-vs-speed Pareto frontier sits.

**Success Metrics:**

1. **Failure categorization recall**: After re-categorizing the existing `v0_18_5_core_3harness` dataset, fewer than 10% of failures remain in the catch-all `api_error` bucket (current: ~76%).
2. **CLI parity with dashboard**: `ailang eval-matrix` and `ailang eval-sweet-spot` text output expose every field that the JSON `efficiency` block contains.
3. **Operator-actionable bucketing**: `ailang eval-sweet-spot eval_results/<dir>` shows each model classified into `{passing_fast, passing_slow, budget_blocked, capability_blocked, provider_blocked}` — a human can read the table and decide "bump the turn budget on model X" in under 30 seconds.
4. **Offline recompute on existing data**: Re-categorization works on existing result JSONs without re-running expensive evals (categorizer is a pure function of `stderr` + `finish_reason`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Five new `ErrorCategory` constants (`timeout`, `quota_exhausted`, `rate_limit`, `cost_killed`, `step_exhausted`) | Becomes the canonical failure vocabulary; downstream dashboards, post-release tooling, and blog stats key off these strings | human | design | high (renames cascade to JSX dashboard, docs, public dashboard JSON consumers) |
| Add `FinishReason` field to `RunMetrics` (not nested in `ProviderData`) | Promotes executor-level finish signals to a first-class JSON column; alternative ("keep in ProviderData and parse on read") would force every consumer to re-implement extraction | human | design | med (one schema field; all writers/readers must update) |
| `ShouldExcludeFromCapability(cat)` helper | Replaces ~10 inline `== "api_error"` checks; if helper semantics drift the dashboard misreports model capability | human | design | med (single helper, called from many sites) |
| Sweet-spot bucket definitions (thresholds for `slow_pass`, `budget_blocked`) | Bucketing is the headline UX of this feature; threshold defaults shape the recommendation users read | human | design | low (CLI flag overrides; only default value at stake) |
| Whether to backfill `error_category` in already-shipped result JSONs | Backfill changes historical dashboards; not backfilling means v0.15.x baselines stay in `api_error` purgatory | human | design | high (file rewrites on shipped baselines; audit trail concerns) |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Final set of `ErrorCategory` constants (names, JSON spelling) approved
- [ ] `FinishReason` placement (top-level field vs. nested) chosen — proposal: top-level `finish_reason` string
- [ ] Backfill policy: re-categorize in place on next dashboard rebuild, OR leave shipped JSONs untouched and re-categorize lazily at read time. (Proposal: lazy at read time in `loader.go` — preserves the on-disk audit trail and lets us re-run categorization with newer rules without rewriting history.)
- [ ] Bucket thresholds: `slow_pass = TTS > 60s`, `budget_blocked = step_exhausted OR cost_killed`, `provider_blocked = quota_exhausted OR rate_limit` — confirmed defaults

## Solution Design

### Overview

Two-phase fix:

- **Phase 1 (foundation)** — Distinguish failure causes. Introduce typed `ErrorCategory` constants, plumb executor `finish_reason` into `RunMetrics`, classify agent errors against real error strings, and replace the indiscriminate `api_error` filter with a semantic `ShouldExcludeFromCapability` helper.
- **Phase 2 (reporting)** — Surface what we now have. Extend `FormatMatrix` and `eval-compare` with efficiency + failure-cause columns; add a new `ailang eval-sweet-spot` command; add a "Sweet Spot" section to the MDX dashboard export.

The data is already captured per result (v0.15.1). This work is almost entirely **aggregation + categorization + reporting** — no new executor instrumentation beyond passing through `finish_reason`.

### Architecture

**Components:**

1. **Failure Categorizer** (`internal/eval_harness/error_categorizer.go`) — Pure function `CategorizeAgentError(err error, finishReason string) ErrorCategory` matching against real error strings sampled from `eval_results/`.
2. **Sweet-Spot Builder** (`internal/eval_analysis/sweet_spot.go`) — Pure function `BuildSweetSpot([]*BenchmarkResult, opts) []SweetSpotRow` that groups results by `(model, harness)`, calls `ComputeEfficiency`, and buckets benchmarks per model into the 5-way classification.
3. **CLI surface** (`cmd/ailang/eval_tools.go`) — New `ailang eval-sweet-spot` subcommand with `--format=text|json|csv`, `--slow-threshold`, `--by-benchmark` flags.
4. **MDX exporter extension** (`internal/eval_analysis/export_mdx.go`) — New "Sweet Spot" section in the auto-generated dashboard MDX.
5. **Inclusion helper** (`internal/eval_analysis/capability.go`) — New `ShouldExcludeFromCapability(cat string) bool` to replace ~10 inline `api_error` checks.

### Implementation Plan

**Phase 1: Distinguish failure categories** (~10 hours)

- [ ] Add `Timeout`, `Quota`, `RateLimit`, `CostKilled`, `StepExhausted` constants to `internal/eval_harness/metrics.go`
- [ ] Add `FinishReason string` field to `RunMetrics`
- [ ] Implement `CategorizeAgentError(err, finishReason)` in `internal/eval_harness/error_categorizer.go`
- [ ] Table-driven tests against real error strings sampled from `eval_results/v0_18_*/agent/*.json` (quota, rate-limit, timeout, motoko cost_exhausted)
- [ ] Promote `motoko_finish_reason` → `Result.FinishReason` in `internal/executor/motoko/motoko.go`
- [ ] Emit `FinishReason = "step_exhausted"` on max-turns exit in `internal/eval_harness/agent_runner_streaming.go` (and claude/gemini/opencode runners)
- [ ] Replace blind `ErrorCategoryAPI` assignment in `cmd/ailang/eval_benchmark.go:134-225` with `CategorizeAgentError` call
- [ ] Introduce `ShouldExcludeFromCapability` helper in `internal/eval_analysis/capability.go`
- [ ] Replace inline `r.ErrorCategory == "api_error"` checks in `dashboard_io.go`, `export_json.go`, `export_json_executors.go`, `export_json_matrix.go` (~10 sites)
- [ ] Re-categorize existing dataset: run categorizer offline against `eval_results/v0_18_5_core_3harness/` and verify ≥90% of current `api_error` rows split into typed categories

**Phase 2: Reporting** (~10 hours)

- [ ] Extract `EfficiencyByModel(results) map[string]EfficiencyAggregates` helper in `internal/eval_analysis/efficiency.go` from `export_json.go:384-391`
- [ ] Extend `FormatMatrix` "By Model" table with: Median TTS, Tokens/s, p90 $/success, cost-killed count, step-exhausted count
- [ ] Implement `BuildSweetSpot` + bucketing in `internal/eval_analysis/sweet_spot.go`
- [ ] Implement text/markdown/CSV/JSON formatters for sweet-spot output
- [ ] Wire `ailang eval-sweet-spot` subcommand in `cmd/ailang/eval_tools.go`; update `cmd/ailang/help.go`
- [ ] Add "Sweet Spot" section emitter to `internal/eval_analysis/export_mdx.go`
- [ ] Add per-model ∆TTS / ∆cost-killed / ∆step-exhausted / ∆p90$/success to `ComparisonReport` and `FormatComparison`

**Phase 3: Tests, docs, validation** (~4 hours)

- [ ] `internal/eval_analysis/sweet_spot_test.go` — golden output table tests
- [ ] Extend `formatter_test.go` with golden tests for new matrix columns
- [ ] Update `docs/docs/guides/evaluation/cost-and-speed-budgets.md` — add "Reading the sweet-spot report" + failure-category reference
- [ ] CHANGELOG entry under v0.19.0
- [ ] Manual verification on real eval data (see Success Criteria)

### Files to Modify/Create

**New files:**

- `internal/eval_harness/error_categorizer.go` — Already exists with basic `CategorizeError`; add `CategorizeAgentError` (~80 LOC)
- `internal/eval_harness/error_categorizer_test.go` — New table-driven tests against real error strings (~200 LOC)
- `internal/eval_analysis/sweet_spot.go` — `BuildSweetSpot` + bucketing + text/csv/json formatters (~350 LOC)
- `internal/eval_analysis/sweet_spot_test.go` — golden tests (~250 LOC)
- `internal/eval_analysis/capability.go` — `ShouldExcludeFromCapability` helper (~30 LOC)

**Modified files:**

- `internal/eval_harness/metrics.go` — 5 new constants + `FinishReason` field (~15 LOC)
- `internal/eval_harness/agent_runner_streaming.go` (+ claude/gemini/opencode equivalents) — emit `FinishReason = "step_exhausted"` on max-turns (~10 LOC each × 4 files)
- `internal/executor/motoko/motoko.go` — promote `motoko_finish_reason` (~10 LOC)
- `cmd/ailang/eval_benchmark.go:134-225` — call `CategorizeAgentError`; pass `FinishReason` through (~30 LOC)
- `internal/eval_analysis/{dashboard_io,export_json,export_json_executors,export_json_matrix}.go` — replace ~10 inline `api_error` checks with helper (~30 LOC total)
- `internal/eval_analysis/efficiency.go` — extract `EfficiencyByModel` helper (~20 LOC)
- `internal/eval_analysis/formatter.go` — extend `FormatMatrix` "By Model" (~40 LOC)
- `internal/eval_analysis/comparison.go` — add per-model ∆ fields (~40 LOC)
- `internal/eval_analysis/export_mdx.go` — new "Sweet Spot" MDX section (~80 LOC)
- `cmd/ailang/eval_tools.go` — new `runEvalSweetSpot()` + dispatcher case (~120 LOC)
- `cmd/ailang/help.go` — help text for new command (~20 LOC)
- `docs/docs/guides/evaluation/cost-and-speed-budgets.md` — "Reading the sweet-spot report" section (~80 LOC)
- `CHANGELOG.md` — entry under v0.19.0

## Examples

### Example 1: Reading the new sweet-spot report

**Before** (today, `ailang eval-matrix eval_results/v0_18_5_core_3harness`):

```
By Model
─────────────────────────────────────────
Model                       0-shot     Final   Tokens
claude-haiku-4-5             95.7%     95.7%  124318
opencode-or-deepseek-v4...   33.3%     33.3%   89221
opencode-or-kimi-k2-6        14.3%     14.3%   67882
```

User cannot tell whether deepseek/kimi failed due to capability or budget — both show as "67% failure" identically.

**After**:

```
Sweet-Spot Report
─────────────────────────────────────────────────────────────────────────────────────
Model                        Pass% | Med TTS | Tokens/s |  p90 $/win | Cost-killed | Step-exh | Quota | Score
claude-haiku-4-5             95.7% |    33s  |   124    |    $0.08   |       0     |     0    |   0   | 0.78
opencode-or-deepseek-v4...   33.3% |   142s  |    62    |    $0.003  |       0     |     8    |   0   | 0.21
opencode-or-kimi-k2-6        14.3% |   189s  |    41    |    $0.002  |       0     |    12    |   0   | 0.12
motoko-claude-haiku-4-5      —     |    —    |    —     |     —      |       0     |     0    |  22   | excl.

Buckets:
  fast_pass:        claude-haiku-4-5 (21/21)
  slow_pass:        opencode-or-deepseek-v4 (7/21 take >60s), opencode-or-kimi-k2-6 (3/21)
  budget_blocked:   opencode-or-deepseek-v4 (8 step_exhausted), opencode-or-kimi-k2-6 (12 step_exhausted)
  capability_blocked: opencode-or-deepseek-v4 (6 logic_error)
  provider_blocked: motoko-claude-haiku-4-5 (22 quota_exhausted — excluded from scoring)

Cheapest pass per benchmark:
  balanced_parens         opencode-or-deepseek-v4    $0.001  62s
  cli_args                opencode-or-deepseek-v4    $0.002  120s
  state_machine_vending   claude-haiku-4-5           $0.079  43s   (no cheaper model passed)
  ...
```

The user can now read: "deepseek would likely pass more if I bumped max-turns; kimi is borderline; motoko's data is meaningless this run (quota)."

### Example 2: Re-categorization on existing data

```bash
$ ailang eval-sweet-spot eval_results/v0_18_5_core_3harness --recategorize
Re-categorizing 467 result files (in-memory, files unchanged)...
  api_error → quota_exhausted:      44  (OpenRouter "Key limit exceeded")
  api_error → rate_limit:            5  (429)
  api_error → timeout:               2  (context deadline exceeded)
  api_error → api_error (unknown):   0
  unchanged:                       416

Sweet-spot bucketing now reliable. Run report? [Y/n]
```

## Success Criteria

- [ ] **Categorization recall**: ≥90% of `api_error` rows in `eval_results/v0_18_5_core_3harness` split into a typed category after re-categorization
- [ ] **Schema**: `RunMetrics.FinishReason` populated in all new agent runs; backward-compatible (older JSONs without the field load fine)
- [ ] **CLI parity**: `ailang eval-matrix` output contains every numeric field the JSON `models[].efficiency` block contains
- [ ] **New command**: `ailang eval-sweet-spot eval_results/standard` produces text, CSV, and JSON output identically populated
- [ ] **MDX export**: Generated dashboard MDX contains a "Sweet Spot" section with the same bucketing as the CLI
- [ ] **Dashboard non-regression**: Pareto chart in QualityScatter still renders correctly; the only visible change is that `quota_exhausted` / `rate_limit` rows now appear as `excluded` rather than just being silently dropped
- [ ] All `make test` passes; `make ci` clean
- [ ] CHANGELOG entry under v0.19.0
- [ ] `docs/docs/guides/evaluation/cost-and-speed-budgets.md` has a "Reading the sweet-spot report" section with the bucket taxonomy

## Testing Strategy

**Unit tests:**

- `error_categorizer_test.go` — table-driven with real strings sampled from `eval_results/v0_18_*`:
  - `{err: "Key limit exceeded (monthly limit)", finish: ""}` → `quota_exhausted`
  - `{err: "429 rate limit exceeded", finish: ""}` → `rate_limit`
  - `{err: "context deadline exceeded", finish: ""}` → `timeout`
  - `{err: "", finish: "cost_exhausted"}` → `cost_killed`
  - `{err: "", finish: "step_exhausted"}` → `step_exhausted`
  - `{err: "500 internal server error", finish: ""}` → `api_error` (fallback)
- `sweet_spot_test.go` — golden tests covering: all-pass model, never-pass model, mixed slow-pass, cost-killed bucket, empty input, missing `success_at_ms` (mirror existing `ComputeEfficiency` fallback to `DurationMs`)
- `formatter_test.go` — golden test for new matrix columns

**Integration tests:**

- Run `ailang eval-sweet-spot eval_results/v0_18_5_core_3harness --format=json` and assert it parses + contains expected models + buckets
- Run `ailang eval-compare <baseline> <new>` end-to-end and assert the ∆TTS column appears

**Manual testing:**

- Inspect output against an OpenRouter cheap model (e.g. `opencode-or-deepseek-v4-flash`) and verify it lands in `slow_pass` or `budget_blocked` bucket — not `capability_blocked`
- Confirm `make ci` passes
- Build the docs site (`cd docs && npm run build`) and visually inspect the auto-generated dashboard MDX

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact CLI flag names** (`--slow-threshold` vs `--slow-ms` vs `--slow=60s`) — agent may choose
- **Whether `eval-sweet-spot` is also wired into `post-release` automation** — defer; can be added as a follow-up once humans use the CLI version for a release or two
- **JSX dashboard additions** for the new categories (`step_exhausted` badge, `budget_blocked` filter) — out of scope here; track separately as a docs-side follow-up
- **Color palette / ANSI styling** for the CLI text output — agent may choose, consistent with existing `formatter.go`
- **Which Pareto-frontier algorithm** to use in the sweet-spot text output (currently the JSX uses a simple max-y-min-x sweep) — agent may reuse or simplify

## Non-Goals

**Not attempted in this feature:**

- **Re-running evaluations to backfill `FinishReason`** in shipped result JSONs — categorization is offline-recomputable from `stderr`; `FinishReason` only matters for new runs
- **Adding new executor cost-budget enforcement** — `CostKilledAt` will be set if and only if the executor produces a budget-kill signal; we will not invent one. If unset for the current dataset, the bucket simply shows 0
- **Changing the JSON schema of `EfficiencyAggregates`** — v0.15.1's shape is stable and consumed by dashboards
- **Adding a new dashboard page** — MDX section + CLI is sufficient for this milestone; full JSX page is a follow-up if needed
- **Cross-language comparison rework** — sweet-spot is per-(model, harness), language remains an orthogonal dimension

## Timeline

**Day 1 — Categorization foundation** (~8 hours):
- Phase 1 tasks 1–6: constants, `FinishReason` field, categorizer, motoko + agent runner plumbing
- Categorizer unit tests against real error strings

**Day 2 — Reporting surfaces** (~8 hours):
- Phase 1 tasks 7–10: replace inline `api_error` filters, run re-categorization sanity check
- Phase 2 tasks 1–4: `EfficiencyByModel`, `FormatMatrix` extension, `BuildSweetSpot` + formatters

**Day 3 — CLI + docs + validation** (~8 hours):
- Phase 2 tasks 5–7: CLI wiring, MDX export, `eval-compare` deltas
- Phase 3: tests, docs, CHANGELOG, end-to-end validation on real data

**Total: ~24 hours across 3 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Categorizer string matching is fragile across provider error formats | Med | Table-driven tests pinned to real samples; `api_error` remains the safe fallback; lazy categorization at read time means we can iterate categorizer rules without rewriting JSONs |
| Dashboard JS still references the old `api_error` filter semantics | Med | Phase 1 task 10 audits all reads; JSX `QualityScatter` already reads typed `error_category` strings, so new categories surface as additional series rather than breaking the chart |
| Backfill changes historical baselines if we re-categorize in place | High | Default to **lazy categorization at read time** in `loader.go`; shipped JSONs remain byte-identical; offer `--rewrite` only as an explicit opt-in flag |
| Motoko / opencode runner doesn't actually emit `step_exhausted` finish in current versions | Low | Phase 1 task 6 is "best-effort"; bucket simply shows 0 step-exhausted for executors that don't emit it. Document in the user guide |
| Net effect on the public dashboard could surprise users (numbers shift) | Med | Frame the release note as "improved categorization — old `api_error` slice now resolves to quota/rate/timeout"; do not change historical aggregate definitions, only failure-cause attribution |

## Related Documents

**Implemented (must inform design):**

- [design_docs/implemented/v0_15_1/m-eval-cost-and-speed-budgets.md](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) — direct predecessor; this doc consumes its `EfficiencyAggregates` and `CostKilledAt`
- [design_docs/implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md](../../implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md) — local-model eval infra, error path patterns
- [design_docs/implemented/v0_3/dashboard-workflow-improvements.md](../../implemented/v0_3/dashboard-workflow-improvements.md) — historical dashboard architecture

**Planned (check for overlap):**

- [design_docs/planned/v0_15_0/m-eval-results-folder-structure.md](../v0_15_0/m-eval-results-folder-structure.md) — folder layout; may interact with where sweet-spot writes its CSV/JSON
- [design_docs/planned/v0_15_0/m-eval-trust-signals.md](../v0_15_0/m-eval-trust-signals.md) — adjacent concept (signal quality); sweet-spot bucketing is a specialization
- [design_docs/planned/v0_19_0/m-benchmark-data-integrity.md](./m-benchmark-data-integrity.md) — sibling milestone in v0.19.0 around the same data pipeline

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [M-EVAL-COST-AND-SPEED-BUDGETS implementation report](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) — what shipped in v0.15.1
- [docs/docs/guides/evaluation/cost-and-speed-budgets.md](../../../docs/docs/guides/evaluation/cost-and-speed-budgets.md) — existing user-facing schema doc
- [internal/eval_harness/metrics.go](../../../internal/eval_harness/metrics.go) — `RunMetrics` schema
- [internal/eval_analysis/efficiency.go](../../../internal/eval_analysis/efficiency.go) — `ComputeEfficiency` (will be reused)
- [internal/executor/motoko/parser.go:293](../../../internal/executor/motoko/parser.go) — `motoko_finish_reason` capture site

## Future Work

- **JSX dashboard surface** for the new categories — a stacked "failure cause" bar per model, plus a `budget_blocked` filter on the explorer
- **Auto-tune turn budgets**: once `step_exhausted` is a first-class signal, post-release tooling can recommend increasing `max_turns` for models whose `step_exhausted` rate is high and `capability_blocked` rate is low
- **Cost-budget enforcement at eval level**: actually wire `CostKilledAt` to a per-benchmark $ cap in the agent path (M-EVAL-COST-AND-SPEED-BUDGETS designed it but the agent path doesn't enforce yet)
- **Per-prompt-version bucketing**: the same model can move between buckets across prompt versions; expose the diff so prompt iterations can be evaluated for their effect on cost/speed, not just pass-rate

---

**Document created**: 2026-05-11
**Last updated**: 2026-05-11

**DESIGN_DOC_PATH**: `design_docs/planned/v0_19_0/m-eval-sweet-spot.md`
