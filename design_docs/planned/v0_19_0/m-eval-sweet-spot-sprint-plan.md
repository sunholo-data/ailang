# Sprint Plan: M-EVAL-SWEET-SPOT

**Sprint ID**: M-EVAL-SWEET-SPOT
**Design doc**: [m-eval-sweet-spot.md](./m-eval-sweet-spot.md)
**Target version**: v0.19.0
**Duration**: 3 working days (~24 hours)
**Risk level**: Medium (touches eval JSON schema; dashboard JS consumers)

## Goal

Eliminate the undifferentiated `api_error` bucket (76% of recent agent failures) by introducing typed failure categories, then surface the cost-vs-time-vs-success Pareto frontier in the CLI and MDX dashboard. Enables operators to answer "would this cheap model pass if I gave it more turns / budget?"

## Discovery (from design doc + codebase audit)

- v0.15.1 already captures `first_attempt_ms`, `success_at_ms`, `tokens_per_sec`, `cost_killed_at`, `agent_turns` per result, and `ComputeEfficiency` aggregates them. The JS dashboard renders Pareto charts. The CLI does not.
- 51 of 67 agent failures in `eval_results/v0_18_*` lump into `api_error` (OpenRouter quota kills + timeouts + 429s all indistinguishable).
- Motoko emits `cost_exhausted` finish_reason but it's dropped at the `RunMetrics` boundary.
- `CostKilledAt` schema field is unused by any executor in the current dataset.
- ~10 inline `r.ErrorCategory == "api_error"` checks across `internal/eval_analysis/` filter capability data incorrectly.

## Velocity check

Recent comparable work (M-AI-STEP-STREAMING M1–M4, completed ~7 days): 4 milestones, ~200–400 LOC/day for eval-adjacent Go code. This sprint targets ~270 LOC/day across 3 days for ~810 LOC total — within recent velocity.

## Milestones

### M1: Failure Category Foundation (~8 hours, ~250 LOC + 200 test LOC)

**Deliverables:**
- 5 new `ErrorCategory` constants in `internal/eval_harness/metrics.go`: `Timeout`, `QuotaExhausted`, `RateLimit`, `CostKilled`, `StepExhausted`
- `FinishReason string` field added to `RunMetrics`
- `CategorizeAgentError(err error, finishReason string) string` in `internal/eval_harness/error_categorizer.go`
- Table-driven tests against real error strings sampled from `eval_results/v0_18_*/agent/*.json`

**Acceptance criteria:**
- All 5 new constants exported and documented
- Categorizer correctly classifies: OpenRouter "Key limit exceeded", "429", "context deadline exceeded", motoko `cost_exhausted`, agent `step_exhausted`, unknown → `api_error` fallback
- Re-running categorizer offline against the current `v0_18_5_core_3harness` dataset reclassifies ≥90% of `api_error` rows into typed categories
- All existing tests pass; `make test` clean

**Risks:**
- Provider error strings change over time → mitigation: keep `api_error` as fallback; tests use frozen real samples

### M2: Wire FinishReason Through Executors (~6 hours, ~150 LOC + 100 test LOC)

**Deliverables:**
- Motoko: promote `motoko_finish_reason` from `ProviderData` to `Result.FinishReason` in `internal/executor/motoko/motoko.go`
- Agent runners: emit `FinishReason = "step_exhausted"` on max-turns exit in `agent_runner_streaming.go` and claude/gemini/opencode equivalents
- `cmd/ailang/eval_benchmark.go:134-225` calls `CategorizeAgentError(err, result.FinishReason)` instead of blind `ErrorCategoryAPI`
- `FinishReason` plumbed into `RunMetrics` written to disk

**Acceptance criteria:**
- New agent runs produce JSON with non-empty `finish_reason` when applicable
- Motoko `cost_exhausted` fixture round-trips into `cost_killed` ErrorCategory
- Existing JSON files without `finish_reason` load fine (backward compatible)
- `make test` passes including motoko parser tests

**Risks:**
- Each runner has different lifecycle hooks → mitigation: M2 ships even if only motoko + streaming runner emit it; claude/gemini/opencode added best-effort

### M3: Replace api_error Filter with Helper (~3 hours, ~80 LOC + 60 test LOC)

**Deliverables:**
- New `internal/eval_analysis/capability.go` with `ShouldExcludeFromCapability(cat string) bool`
- Replace ~10 inline `r.ErrorCategory == "api_error"` checks in `dashboard_io.go`, `export_json.go`, `export_json_executors.go`, `export_json_matrix.go`
- Unit tests for the helper

**Acceptance criteria:**
- After M1+M2+M3, dashboard JSON `excluded` count matches `quota_exhausted + rate_limit` (not blanket `api_error`)
- `timeout`, `cost_killed`, `step_exhausted` rows are **included** in capability stats (visible as failures, not hidden)
- `make ci` clean

**Risks:**
- JS dashboard may rely on current exclusion semantics → mitigation: only the *capability filter* changes, not aggregate counts; visual changes documented in CHANGELOG

### M4: CLI + Sweet-Spot Report (~6 hours, ~400 LOC + 250 test LOC)

**Deliverables:**
- `internal/eval_analysis/efficiency.go`: extract `EfficiencyByModel(results)` helper from `export_json.go:384-391`
- Extend `FormatMatrix` in `internal/eval_analysis/formatter.go` "By Model" table with: Median TTS, Tokens/s, p90 $/success, cost-killed, step-exhausted
- `internal/eval_analysis/sweet_spot.go`: `BuildSweetSpot(results, opts) []SweetSpotRow` with bucketing (`fast_pass`, `slow_pass`, `budget_blocked`, `capability_blocked`, `provider_blocked`)
- Text/markdown/CSV/JSON formatters for sweet-spot output
- `cmd/ailang/eval_tools.go`: `runEvalSweetSpot()` + dispatcher case
- `cmd/ailang/help.go`: help entry for `eval-sweet-spot`
- `internal/eval_analysis/comparison.go`: ∆TTS / ∆cost-killed / ∆step-exhausted / ∆p90$ per model in `FormatComparison`

**Acceptance criteria:**
- `ailang eval-sweet-spot eval_results/v0_18_5_core_3harness` produces a ranked text table + per-benchmark "cheapest pass" footer
- `--format=csv` and `--format=json` produce valid output with identical data
- `ailang eval-compare <baseline> <new>` shows ∆TTS column for models present in both
- Golden tests in `sweet_spot_test.go` and `formatter_test.go` pass

**Risks:**
- Bucket thresholds need real-data validation → mitigation: default `slow_pass = TTS > 60s`, flag-overridable

### M5: MDX Export + Docs (~3 hours, ~120 LOC + 0 test LOC)

**Deliverables:**
- New "Sweet Spot" section in `internal/eval_analysis/export_mdx.go` emitting markdown table
- `docs/docs/guides/evaluation/cost-and-speed-budgets.md`: "Reading the sweet-spot report" section + failure-category reference table
- CHANGELOG.md entry under v0.19.0
- Example file: `examples/eval_sweet_spot_example.md` showing real CLI output

**Acceptance criteria:**
- `cd docs && npm run build` succeeds
- Generated dashboard MDX contains "Sweet Spot" section
- Docs page documents all 5 new error categories with example error strings
- Example file referenced from cost-and-speed-budgets.md

**Risks:**
- Docusaurus rebuild may fail on new markdown syntax → mitigation: keep MDX section to plain tables (no custom components)

## Day-by-Day Plan

### Day 1 (8h)
- **AM (4h)**: M1 — constants, categorizer, table-driven tests against real samples
- **PM (4h)**: M2 — motoko `FinishReason` promotion + streaming-runner `step_exhausted` emission

### Day 2 (8h)
- **AM (3h)**: M2 cont'd — claude/gemini/opencode runner `FinishReason`; wire `CategorizeAgentError` into `cmd/ailang/eval_benchmark.go`
- **AM (3h)**: M3 — `ShouldExcludeFromCapability` helper + replace ~10 inline checks
- **PM (2h)**: M4 start — `EfficiencyByModel` extract, extend `FormatMatrix`

### Day 3 (8h)
- **AM (4h)**: M4 cont'd — `BuildSweetSpot` + formatters + CLI wiring
- **PM (2h)**: M4 cont'd — `eval-compare` deltas + golden tests
- **PM (2h)**: M5 — MDX export section + docs + CHANGELOG + example file

## Success Metrics

- [ ] All `make test` passes; `make ci` clean
- [ ] ≥90% of `api_error` rows in v0_18_5 dataset reclassify into typed categories
- [ ] `ailang eval-sweet-spot eval_results/standard` runs successfully and outputs text + CSV + JSON
- [ ] `ailang eval-compare` shows ∆TTS column
- [ ] Dashboard MDX rebuild succeeds with new "Sweet Spot" section
- [ ] CHANGELOG entry under v0.19.0
- [ ] Example file `examples/eval_sweet_spot_example.md` created
- [ ] Test coverage on new files ≥80%

## Open Questions

1. **Backfill policy**: re-categorize shipped JSONs in place or lazy-at-read-time? (Design doc proposes lazy; confirm during M3)
2. **Bucket thresholds**: confirm `slow_pass = 60s` is the right default (could also be model-relative, e.g. 3× median)
3. **CLI flag bikeshed**: `--slow-threshold=60s` vs `--slow-ms=60000` — agent's call

## Dependencies

- M2 depends on M1 (constants needed before runners can emit them)
- M3 depends on M1 (helper categorizes against new constants)
- M4 depends on M1+M3 (sweet-spot bucketing uses typed categories + helper)
- M5 depends on M4 (MDX section uses same BuildSweetSpot)

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| Provider error strings drift | Frozen real-sample fixtures; `api_error` fallback preserved |
| Backward compatibility with old JSONs | `FinishReason` is `omitempty`; categorizer handles empty `finish_reason` |
| JS dashboard regressions | Phase 1 doesn't change aggregate definitions, only failure attribution; CHANGELOG flags visual changes |
| Sprint overruns 3 days | M5 (docs) is scoped to defer cleanly to a follow-up commit if needed |

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_19_0/m-eval-sweet-spot-sprint-plan.md`
