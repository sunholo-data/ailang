# M-EVAL-COST-AND-SPEED-BUDGETS — Sprint Plan

**Sprint ID**: M-EVAL-COST-AND-SPEED-BUDGETS
**Target**: v0.16.0
**Design doc**: [m-eval-cost-and-speed-budgets.md](m-eval-cost-and-speed-budgets.md)
**Estimated duration**: 2.5 working days (~22 hours)
**Total LOC estimate**: ~1290 LOC implementation + ~250 LOC tests
**Risk level**: Medium (5 executors to wire, but mostly mechanical; new cost-helper is the only novel logic)

## Sprint Goal

Replace wall-clock-as-cost-proxy with explicit cost budgets at the shared `Executor` interface so all 5 harnesses (claude/gemini/codex/opencode/pi) enforce uniform cost-vs-time semantics, AND surface speed metrics (TTFA, TTS, turns, tokens/sec) as first-class observables. Re-smoke 4 OS near-misses to validate the redesign — at least 1 should promote to PASS.

## Velocity Context

Recent sprint velocity (last 14 days, AILANG repo):
- **M-AI-EFFECT-MODES** (M1-M4): ~810 LOC across 4 milestones in ~3 days = ~270 LOC/day
- **M-AI-OPENROUTER** (M1-M4): ~1100 LOC across 4 milestones in ~3 days = ~370 LOC/day
- **M-EFFECT-REFINEMENT-PHASE1**: ~810 LOC + 332 zero-diff tests in ~3 days

Conservative target: **~520 LOC/day for tightly scoped Go-side work**. This sprint at 1290 LOC over 2.5 days = 516 LOC/day — matches recent velocity exactly. No buffer needed beyond the realistic estimates already in the design doc.

## Sprint Structure

```
Day 1 (~9h)
├── M1: Schema + cost helper (4h, ~250 LOC)
└── M2: Wire 5 executors (5h, ~150 LOC)

Day 2 (~8h)
├── M3: Result schema + JSON exporter (4h, ~200 LOC)
└── M4: Dashboard charts (4h, ~400 LOC React)
       └── (M4 + M5 can run in parallel via sub-agents)

Day 3 (~5h)
├── M5: Re-test + curation (3h, ~$2 eval cost)
└── M6: Docs + retro (2h, ~150 LOC docs)
```

## Milestone Breakdown

### M1 — Schema + Cost Helper (~4h, ~250 LOC)

**Owner**: sprint-executor (TDD)

**Tasks**:
1. Create `internal/executor/cost.go` (~150 LOC)
   - `CostBudget` struct with atomic counters
   - `NewCostBudget(maxUSD, inputPer1K, outputPer1K float64) *CostBudget`
   - `Add(inputDelta, outputDelta int) (current float64, exceeded bool)`
   - `Current() float64`
   - `KilledAt() float64` — first-write-wins via atomic.Value
2. Create `internal/executor/cost_test.go` (~100 LOC)
   - Token mix → cost calculation correctness (table-driven, ≥10 cases)
   - Race-condition test under `-race` with N=100 concurrent `Add()` goroutines
   - `KilledAt()` first-write-wins with concurrent triggers
   - Default-formula fallback when `MaxUSD == 0`
3. Add `budgets:` block parsing to `internal/eval_harness/models.go` (~60 LOC)
   - New `Budgets` struct: `MaxCostUSD`, `HardTimeoutSecs`, `ExpectedTTFTSecs`, `ExpectedTTFSolutionSecs`
   - Default-formula fallback: `min($0.50, input_per_1k × 64 + output_per_1k × 32)`
   - Parse 0+ existing models with omitted blocks (back-compat)

**Acceptance criteria**:
- [ ] `internal/executor/cost.go` exists with all 4 public methods
- [ ] Unit-test coverage ≥ 90% (`go test -coverprofile`)
- [ ] `go test -race ./internal/executor/cost...` passes
- [ ] `models.yml` parses with `budgets:` blocks AND with them omitted (back-compat)
- [ ] Default formula correct for `or-minimax-m2-7` (cheapest) and `claude-opus-4-7` (priciest)

**Risk**: Low. Cost-helper is pure Go with atomic primitives — well-trodden territory.

---

### M2 — Wire 5 Executors (~5h, ~150 LOC)

**Owner**: sprint-executor (TDD per executor; mechanical)

**Tasks** (one per executor, ~30 LOC each):
1. **`claude` executor** — hook at stream-json `usage` event
2. **`opencode` executor** — hook at `tool_result` event (use `usage` field if present, else fall back to wall-clock-extrapolated estimate using TTFTTimeout token rate)
3. **`gemini` executor** — hook at `result` event
4. **`codex` executor** — hook at turn boundary (codex emits per-turn usage)
5. **`pi` executor** — hook at Pi response

Each executor:
- Calls `budget.Add(in, out)` at the natural event point
- If `exceeded == true`, returns `Result{CostKilledAt: current, ...}` and stops reading the stream
- Records `FirstAttemptMs`, `SuccessAtMs`, `TokensPerSec`, `Turns` for the speed-metrics path

**Acceptance criteria**:
- [ ] All 5 executors call `budget.Add()` at the correct event boundaries
- [ ] When `MaxCostUSD == 0`, behaviour is byte-identical to pre-change baseline (back-compat)
- [ ] When budget exceeded mid-stream, executor exits cleanly with `Result.CostKilledAt > 0`
- [ ] Speed timestamps populate correctly per executor (verified via integration test)
- [ ] No new lint warnings in any executor file

**Risk**: Med — opencode `usage` event timing is the unknown. If incremental events don't arrive, fall back to wall-clock-extrapolated cost (flag in result metadata). This fallback is in the Deferred Decisions of the design doc — agent has latitude.

---

### M3 — Result Schema + JSON Exporter (~4h, ~200 LOC)

**Owner**: sprint-executor

**Tasks**:
1. Promote `Turns` from metadata to top-level `Result` field (~30 LOC)
2. Add new `Result` fields: `FirstAttemptMs`, `SuccessAtMs`, `TokensPerSec`, `CostKilledAt` (~20 LOC)
3. Update `internal/eval_analysis/export_json.go` to emit per-model `efficiency` block (~120 LOC):
   ```json
   "efficiency": {
     "median_time_to_first_attempt_ms": 8400,
     "median_time_to_success_ms":       42000,
     "median_turns_to_success":         3,
     "median_tokens_per_sec":           45.2,
     "p90_cost_per_success":            0.18,
     "speed_efficiency_score":          0.73
   }
   ```
4. Aggregate computation: median, p90, frontier scoring (~30 LOC)
5. New failure category in summary: `cost_killed` distinct from `api_error`/`timeout`/`logic_error`

**Acceptance criteria**:
- [ ] All result JSONs from existing baselines parse with new schema (additive only)
- [ ] `dashboard/latest.json` regenerates with new `efficiency` block per model
- [ ] `cost_killed` appears in `Top Error Codes` table when applicable
- [ ] `make test ./internal/eval_analysis/...` green
- [ ] Existing v0.14.x baselines replay with identical aggregates when re-exported (back-compat)

**Risk**: Low. Schema-additive change.

---

### M4 — Dashboard Charts (~4h, ~400 LOC React)

**Owner**: sprint-executor (can be parallelized with M5 via sub-agent)

**Tasks**:
1. New `docs/src/components/BenchmarkDashboard/SpeedRadar.jsx` (~200 LOC)
   - Median time-to-success per model (radar layout, mirror of existing success-rate radar)
   - Outlier-clipping at 5× median (same pattern as cost radar fix from v0.15.0 post-release)
   - Tooltip shows real values
2. New `docs/src/components/BenchmarkDashboard/CostSpeedFrontier.jsx` (~200 LOC)
   - Pareto scatter (x = cost/success log scale, y = sec/success log scale, marker = success rate)
   - Color: harness (claude/gemini/codex/opencode/pi)
   - Pareto frontier line through optimal points
   - Tooltip: real values + harness + cost-killed count
3. Augment `docs/src/components/BenchmarkDashboard/PerModelTrend.jsx` (~60 LOC)
   - Add speed regression line alongside success-rate trend
   - Flag versions where speed regresses ≥20%

**Acceptance criteria**:
- [ ] `npm run build` (Docusaurus) succeeds without warnings
- [ ] SpeedRadar renders with v0.15.0 data (no NaN/undefined values)
- [ ] CostSpeedFrontier shows Pareto line for ≥5 models
- [ ] PerModelTrend speed line uses same model colors as existing success-rate line (consistency)
- [ ] All 7 dashboard formatters use unified `formatModelName` from v0.15.0 hotfix (no regression)

**Risk**: Low. Recharts is well-known; outlier-clipping pattern already proven in v0.15.0 cost radar.

---

### M5 — Re-Test + Curation (~3h, ~$2 eval cost)

**Owner**: sprint-executor (can be parallelized with M4)

**Tasks**:
1. Add `budgets:` blocks to 4 OS near-miss models in `models.yml`:
   ```yaml
   opencode-or-deepseek-v4-flash:
     budgets:
       max_cost_usd: 0.30
       hard_timeout_secs: 600
   opencode-or-kimi-k2-6:
     budgets:
       max_cost_usd: 0.30
       hard_timeout_secs: 600
   opencode-or-glm-4-7-flash:
     budgets:
       max_cost_usd: 0.30
       hard_timeout_secs: 600
   opencode-or-gemma-4-26b:
     budgets:
       max_cost_usd: 0.30
       hard_timeout_secs: 600
   ```
2. Re-smoke each on the 3 smoke benchmarks (fizzbuzz, adt_option, csv_to_json):
   ```bash
   ailang eval-suite --agent --models <model> \
     --benchmarks fizzbuzz,adt_option,csv_to_json_converter \
     --langs ailang --agent-parallel 1 --output /tmp/smoke_<model>
   ```
3. Compare to 2026-05-04 / 2026-05-05 smoke results
4. Update `models.yml` `notes:` blocks with new smoke dates and pass/fail status
5. If ≥1 model promotes to 3/3 PASS, add to `agent_suite` (mirroring GLM 5 + MiniMax M2.7 v0.15.0 promotion)

**Acceptance criteria**:
- [ ] All 4 OS near-misses re-smoked with new budgets
- [ ] At least 1 model achieves 3/3 PASS (success metric from design doc)
- [ ] `models.yml` `notes:` blocks updated with new smoke dates
- [ ] If promoted, model added to `agent_suite` AND `extended_suite` (standard variant if applicable)

**Risk**: Med. Outcome is genuinely uncertain — that's why we're testing. If 0/4 promote, that's still useful signal (capability gap, not budget gap) and we update the design doc retro accordingly.

---

### M6 — Docs + Sprint Retro (~2h, ~150 LOC docs)

**Owner**: sprint-executor

**Tasks**:
1. New section in `docs/docs/guides/evaluation/cost-and-speed-budgets.md` (~100 LOC)
   - Conceptual overview (why cost > time)
   - `models.yml` `budgets:` block reference
   - Result-file new fields reference
   - Migration guide for legacy baselines
2. Update `docs/docs/guides/evaluation/README.md` to link new guide
3. Move design doc: `design_docs/planned/v0_16_0/m-eval-cost-and-speed-budgets.md` → `design_docs/implemented/v0_16_0/m-eval-cost-and-speed-budgets.md`
4. Move sprint plan: `design_docs/planned/v0_16_0/m-eval-cost-and-speed-budgets-sprint-plan.md` → `design_docs/implemented/v0_16_0/m-eval-cost-and-speed-budgets-sprint-plan.md`
5. Append implementation report to design doc (what was built, deviations, M5 outcome)
6. CHANGELOG entry under v0.16.0 with M-EVAL summary + curation outcome

**Acceptance criteria**:
- [ ] New eval guide linked from `docs/docs/guides/evaluation/README.md`
- [ ] Design doc + sprint plan moved to `implemented/v0_16_0/`
- [ ] CHANGELOG `[v0.16.0]` section references this milestone with link to design doc
- [ ] Implementation report includes: M5 outcome (which models promoted), any deviations from design, performance impact (cost-helper overhead negligible if true)

**Risk**: Low. Pure documentation.

## Day-by-Day Breakdown

### Day 1 — Foundations (~9h)

| Hours | Task |
|------:|------|
| 0:00 | Branch from `dev`: `git checkout -b sprint/m-eval-cost-and-speed-budgets` |
| 0:00–4:00 | **M1** — `cost.go` + `cost_test.go` + `models.go` budget parsing |
| 4:00–4:30 | M1 checkpoint: `make test`, `go test -race`, milestone checkpoint script |
| 4:30–9:00 | **M2** — wire 5 executors (one at a time, TDD) |
| 9:00 | M2 checkpoint: integration test all 5 executors observe `CostKilledAt` |
| EOD | Commit M1 + M2 with `refs M-EVAL-COST-AND-SPEED-BUDGETS` |

### Day 2 — Schema + Charts (~8h)

| Hours | Task |
|------:|------|
| 0:00–4:00 | **M3** — promote `Turns` to top-level Result; new fields; export_json.go aggregates |
| 4:00–8:00 | **M4** — `SpeedRadar.jsx` + `CostSpeedFrontier.jsx` + `PerModelTrend.jsx` augmentation |
| EOD | Commit M3 + M4. Verify dashboard locally (`cd docs && npm start`) |

### Day 3 — Validation + Docs (~5h)

| Hours | Task |
|------:|------|
| 0:00–3:00 | **M5** — add `budgets:` to 4 OS near-misses; re-smoke each; update `models.yml notes:` |
| 3:00–5:00 | **M6** — eval guide; move design doc; CHANGELOG; implementation report |
| EOD | Final commit. Run `make ci`. Send sprint-evaluator handoff. |

## Cross-Milestone Dependencies

```
M1 (schema + helper)
    ↓
M2 (wire executors) ← needs M1.CostBudget
    ↓
M3 (Result schema + exporter) ← needs M2.populated Result fields
    ↓
M4 (dashboard) ←──── parallel with ────→ M5 (re-test)
              ↓                                   ↓
              └──────────── M6 (docs) ←──────────┘
```

M4 + M5 can run in parallel via Task sub-agents on Day 2/Day 3 if useful, but the sequential plan also works at the stated cadence.

## Acceptance Criteria (Sprint-Level)

- [ ] All 6 milestones marked ✅ in this sprint plan
- [ ] `internal/executor/cost.go` ≥ 90% unit-test coverage
- [ ] All 5 executors pass `go test -race ./internal/executor/...`
- [ ] `make test`, `make lint`, `make verify-examples` green
- [ ] At least 1 of (DeepSeek V4 Flash, Kimi K2.6, GLM 4.7 Flash, Gemma 4 26B) promotes to PASS
- [ ] Dashboard renders SpeedRadar + CostSpeedFrontier locally without errors
- [ ] v0.14.x and v0.15.0 baselines replay identically (back-compat)
- [ ] CHANGELOG v0.16.0 entry references this design doc

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|----------:|------------|
| opencode `usage` events arrive only at task end (no incremental tally) | Med | Fallback in M2: extrapolate cost from TTFTTimeout token rate × elapsed; flag in result metadata. Listed in design doc Deferred Decisions. |
| Per-model `pricing` becomes stale (vendor price changes mid-sprint) | Low | OpenRouter `ResolvedRoute` already records authoritative cost — prefer it over calculated cost when present (decision in design doc) |
| Cost-kill races completion (race condition) | Low | Atomic compare-and-swap on `killedAt`; `cost_test.go` includes `-race` test with N=100 concurrent goroutines |
| Re-tested OS models still all fail (M5 acceptance not met) | Med | Acceptable signal — document in models.yml `notes:` and stay on watchlist. Update sprint retro accordingly. The point of this sprint is fair measurement, not forced promotion. |
| Sprint-executor mis-locates token-tally event in opencode | Med | Sprint plan explicitly lists per-executor event in M2; pre-impl spike if uncertain (~30 min) |
| Existing baselines change under recomputation | Low | Back-compat preserved via `MaxCostUSD == 0` default — gated by absence of `budgets:` block in models.yml |

## Open Questions

None. All 6 high-impact decisions are resolved in the design doc's Design Freeze section. Sprint-executor may proceed without further human gates.

## Success Criteria (User-Visible)

After this sprint ships in v0.16.0:

1. **Eval suite curators** can add cheap-but-slow models without artificial timeout exclusion
2. **Dashboard users** can see Pareto frontier of cost vs speed vs success — pick optimal model by use-case
3. **Future eval-cost-related design docs** (e.g. AILANG-side `!{AI[budget=$X]}` parameterised effects) can build on this primitive
4. **Eval result files** carry richer metadata: `cost_killed_at`, `first_attempt_ms`, `success_at_ms`, `turns`, `tokens_per_sec` — enables Pareto analysis without re-running

## Handoff Notes

- Branch from `dev` (not `main`) per AILANG convention
- Use `refs M-EVAL-COST-AND-SPEED-BUDGETS` in milestone commits, `Closes` only in final M6 commit
- TDD recommended for M1 (failing tests for `CostBudget` first); mechanical work in M2-M4
- M4 + M5 are independent — fine to spawn sub-agent for parallel execution if desired
- CHANGELOG goes in `changelogs/v0.16-current.md` (or whichever active changelog file is current at sprint kickoff)

---

**Sprint Plan Created**: 2026-05-05
**Sprint Plan Last Updated**: 2026-05-05
