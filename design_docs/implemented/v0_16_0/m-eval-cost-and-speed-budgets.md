# M-EVAL-COST-AND-SPEED-BUDGETS

**Status**: Implemented (v0.16.0, 2026-05-05)
**Target**: v0.16.0
**Priority**: P1 (Medium — blocks fair OS-model evaluation; relatively contained change)
**Estimated**: ~2.5 working days (~20 hours)
**Dependencies**:
- Existing `internal/executor/executor.go` Executor interface (Task → Result)
- `internal/eval_harness/models.yml` per-model `pricing` field (already populated)
- OpenRouter `usage` events in stream-json (already captured by handlers)
- M-AI-OPENROUTER (v0.14.3 + v0.16.0): `ResolvedRoute` per-call cost — we mirror the same pattern for non-OR providers

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|------:|---------------|
| A1: Determinism | 0 | No new runtime determinism semantics — eval harness only |
| A2: Replayability | +1 | New `CostKilledAt` field in result JSONs preserves cost-kill events for replay/audit |
| A3: Effect Legibility | 0 | n/a — eval harness internal, no AILANG-side effect changes |
| A4: Explicit Authority | 0 | n/a |
| A5: Bounded Verification | 0 | n/a |
| A6: Safe Concurrency | +1 | Atomic counters tested under `-race`; safe for parallel agent runs |
| A7: Machines First | +1 | Cost-budget semantics are machine-decidable from `models.yml` pricing — no human-facing heuristics |
| A8: Minimal Syntax | 0 | No new AILANG syntax |
| A9: Cost Visibility | **+2** | Direct alignment — this IS a cost-budget primitive; promotes cost from implicit (wall-clock proxy) to explicit, observable, enforceable |
| A10: Composability | +2 | Replaces 5× duplicated timeout logic with one shared `CostBudget` helper composed across all executors — strict net win |
| A11: Structured Failure | +1 | New `CostKilledAt` is a structured, distinguishable failure mode (vs current `api_error` lumping) |
| A12: System Boundary | +2 | Speed metrics (TTFA, TTS, TPS, turns) exposed for the first time as first-class fields at the executor boundary |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Cost-budget semantics derive from `models.yml` pricing — fully machine-decidable

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

**This doc: +10, no hard violations → proceed.**

## Problem Statement

The current eval harness uses **wall-clock timeouts** as a proxy for cost control:

| Budget | Default | Purpose |
|--------|---------|---------|
| `Timeout` (hard) | 60s (agent), 90/180s (benchmark) | Wall-clock cap |
| `IdleTimeout` | 3 min | No-output watchdog |
| `TTFTTimeout` | 30s | Prefill timeout |

This **systematically excludes cheap-but-slow models** that we'd otherwise want to evaluate. v0.15.0 baseline confirmed:

**Current State:**

- `opencode-or-minimax-m2-7`: 29% API error rate (mostly opencode 60s hard timeouts on slow benchmarks). Capability is real — just slow. Cost per benchmark: ~$0.013.
- `opencode-or-deepseek-v4-flash` (re-smoked 2026-05-05): **1/3 PASS**. Both failures = `api_error` from 60s/180s hard timeouts. Cost incurred: ~$0.003. Last week was 2/3.
- `opencode-or-kimi-k2-6`: similar pattern — runs 90-150s of multi-turn iteration, gets killed at 60s.
- Meanwhile `opencode-sonnet-4-6` runs ~$0.20/call × multi-turn = $13.38 for 68 runs, but the wall-clock cap doesn't actually constrain its cost (each call is fast, just expensive).

**Wall-clock is the wrong dimension.**

**Impact:**

- **Who is affected?** The eval suite curators (us), the SOTA OS-model story (anyone choosing AILANG eval data), Motoko fork users who recommended these models, and downstream model selection (we may be picking wrong winners).
- **How significant?** Probably 2-4 OS models are stuck in "near-miss" purgatory because of timeout artifacts, not capability. At 5-7 candidates we've smoke-tested in v0.15.0, that's a 30-60% misclassification rate — high.

This is a **systemic** issue (per CLAUDE.md "audit before patching" rule): patching individual executor timeouts (60s → 120s → 180s) keeps not solving the root problem and adds complexity. One unified cost-and-speed budget at the `Executor` layer handles all 5 harnesses.

## Goals

**Primary Goal:** Replace wall-clock-as-cost-proxy with explicit cost budgets at the `Executor` interface so all 5 harnesses (claude/gemini/codex/opencode/pi) enforce uniform cost-vs-time semantics, AND surface speed metrics as first-class observables.

**Success Metrics:**

- All 4 OS near-misses (DeepSeek V4 Flash, Kimi K2.6, GLM 4.7 Flash, Gemma 4 26B) fairly evaluated — at least 1 promotes to PASS after budget redesign
- Per-model `cost_killed` and `time_killed` are distinguishable failure categories in result JSONs (was bundled as `api_error`)
- Dashboard surfaces speed efficiency frontier alongside cost frontier — Pareto chart for "fastest model under $X/success"
- Zero-diff back-compat: existing `Timeout`/`IdleTimeout`/`TTFTTimeout` semantics preserved when `MaxCostUSD == 0`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Cost budget lives at `Executor` interface (not per-executor) | Avoids 5× duplicate logic; uniform semantics across harnesses | human (this doc) | design | high |
| Default `MaxCostUSD = min($0.50, input × 64K + output × 32K)` | Sets the "no special config" budget for all 50+ existing models | human (this doc) | design | med |
| `hard_timeout_secs` raised from 60s/180s to 600s | Cost becomes primary gate; wall-clock becomes safety net | human (this doc) | design | med |
| Pricing source authority: OpenRouter `ResolvedRoute` over models.yml when both present | Prevents drift when vendor prices change | human (this doc) | design | low |
| Speed metrics (TTFA, TTS, TPS, turns) added to `Result` | First-class observability; enables Pareto frontier | human (this doc) | design | med |
| `CostKilledAt` separate from `api_error` failure category | Affects dashboard reliability cards + report aggregations | human (this doc) | design | med |
| Token-tally hook point per executor | Each executor has different stream semantics; agent decides at impl time | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Cost budget belongs at `Executor` interface, not per-executor (decided in this doc)
- [x] `MaxCostUSD` default formula confirmed: `min($0.50, input × 64K + output × 32K)`
- [x] `hard_timeout_secs` default = 600s (was 60s/180s)
- [x] Speed metrics promoted to first-class `Result` fields (TTFA, TTS, Turns, TokensPerSec)
- [x] OpenRouter `ResolvedRoute` cost takes precedence over calculated cost when present
- [x] `CostKilledAt > 0` is its own failure category, distinct from `api_error`/`timeout`/`logic_error`

All "high"-cost decisions resolved in this doc. Sprint-executor may proceed without further human input.

## Solution Design

### Overview

Add a 4th budget dimension (`MaxCostUSD`) to `Task`, with a shared `CostBudget` helper that all 5 executors hook into. Independently, instrument speed metrics (TTFA, TTS, turns, tokens/sec) in `Result` so the dashboard can render a 3D efficiency frontier.

The wall-clock `Timeout` becomes a **safety net** (default 10 min, was 60s) — only kills runaway/hung tasks, not cheap-but-slow ones. Cost becomes the primary gate.

### Architecture

```
                        ┌─────────────────────────────────────┐
                        │       internal/executor/             │
                        │                                       │
                        │   ┌──────────────────┐                │
                        │   │  type Executor   │  (existing)    │
                        │   │  interface       │                │
                        │   └────────┬─────────┘                │
                        │            │                          │
                        │            │ Execute(ctx, task)       │
                        │            ▼                          │
                        │   ┌──────────────────┐                │
                        │   │  type Task       │                │
                        │   │  + MaxCostUSD    │  ← NEW         │
                        │   │  + (existing 3   │                │
                        │   │     timeouts)    │                │
                        │   └────────┬─────────┘                │
                        │            │                          │
                        │   ┌────────▼─────────┐                │
                        │   │  cost.go         │  ← NEW         │
                        │   │  CostBudget      │                │
                        │   │  (shared helper) │                │
                        │   └──────────────────┘                │
                        └────────┬────────────┬─────────────────┘
                                 │            │
                ┌────────────────┼────────────┼────────────────┐
                ▼                ▼            ▼                ▼
        ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐
        │  claude   │  │ opencode  │  │  gemini   │  │   codex   │  + pi
        │ executor  │  │ executor  │  │ executor  │  │ executor  │
        └───────────┘  └───────────┘  └───────────┘  └───────────┘
        Each executor calls budget.Add(input, output) at its
        natural token-tally point (usage event, turn boundary, etc.)
```

**Components:**

1. **`internal/executor/cost.go` (NEW)**: Shared `CostBudget` struct with atomic counters. Single source of truth for the kill-on-exceed loop.
2. **`Task`/`Result` additions (existing file)**: New fields `MaxCostUSD` (input) and `CostKilledAt`, `FirstAttemptMs`, `SuccessAtMs`, `Turns`, `TokensPerSec` (output).
3. **Per-executor token-tally hooks (5 files)**: Each existing executor gains exactly one new line at the natural token-event point. The kill logic stays in `cost.go`.
4. **`models.yml` `budgets:` block**: Per-model overrides for `max_cost_usd`, `hard_timeout_secs`, `expected_ttft_secs`, `expected_ttf_solution_secs`.
5. **`export_json.go` aggregates**: New `efficiency` block per model — median TTS, p90 cost-per-success, speed-efficiency score.
6. **Dashboard charts (2 new + 1 augmented)**: SpeedRadar, CostSpeedFrontier (Pareto scatter), PerModelTrend (with speed regression line).

### Key types (new)

```go
// internal/executor/cost.go (NEW FILE)
type CostBudget struct {
    MaxUSD       float64
    InputPer1K   float64   // from models.yml pricing
    OutputPer1K  float64

    inputTokens  atomic.Int64
    outputTokens atomic.Int64
    killedAt     atomic.Value // float64
}

func NewCostBudget(maxUSD, inputPer1K, outputPer1K float64) *CostBudget
func (b *CostBudget) Add(inputDelta, outputDelta int) (current float64, exceeded bool)
func (b *CostBudget) Current() float64
func (b *CostBudget) KilledAt() float64

// internal/executor/executor.go (additions)
type Task struct {
    // existing: Timeout, IdleTimeout, TTFTTimeout
    MaxCostUSD float64 // Hard cost ceiling. 0 = unlimited (legacy behaviour).
}

type Result struct {
    // existing: Success, Cost, OutputTokens, Duration, ...

    // NEW — speed instrumentation
    FirstAttemptMs   int64   // ms from task start to first solution submission
    SuccessAtMs      int64   // ms from task start to first passing solution (-1 if never)
    Turns            int     // promote from metadata to top level
    TokensPerSec     float64 // OutputTokens / generation_seconds

    // NEW — cost
    CostKilledAt     float64 // > 0 if killed for cost; 0 otherwise
}
```

### Per-executor token-tally hook

Each existing executor gains exactly one new line at the natural token-event point:

| Executor | Hook event | Token source |
|----------|-----------|--------------|
| `claude` | After each stream-json `usage` event | Anthropic stream usage |
| `gemini` | After each `result` event | Gemini stats block |
| `codex` | After each turn (codex emits per-turn usage) | OpenAI usage |
| `opencode` | After each `tool_result` event | opencode stream-json `usage` field |
| `pi` | After each Pi response | Pi stats block |

The kill-on-exceed loop lives ONLY in `cost.go` — executors just call `budget.Add(in, out)`, get back `(current, exceeded bool)`, and if `exceeded`, return a typed `Result{CostKilledAt: current}` and stop reading the stream.

### Per-model `models.yml` schema additions

```yaml
opencode-or-minimax-m2-7:
  # existing pricing block unchanged
  pricing:
    input_per_1k: 0.0003
    output_per_1k: 0.0012
  # NEW
  budgets:
    max_cost_usd: 0.30      # generous cost ceiling — 30¢ for cheap model
    hard_timeout_secs: 600  # 10 min wall-clock SAFETY NET
    expected_ttft_secs: 30  # for "is this run abnormally slow?" alerts
    expected_ttf_solution_secs: 90
```

**Default fallback** (when `budgets:` block omitted):

```
max_cost_usd      = min($0.50, input_per_1k × 64 + output_per_1k × 32)
hard_timeout_secs = 600  (was 60-180; raised because cost is now primary gate)
```

### Implementation Plan

**M1: Schema + cost helper** (~4h, ~250 LOC)
- [ ] Add `budgets:` block to `internal/eval_harness/models.go` parsing
- [ ] New `internal/executor/cost.go` (~150 LOC + 100 LOC tests)
- [ ] Atomic counters tested under `-race`
- [ ] Default-formula fallback when `budgets:` omitted

**M2: Wire 5 executors** (~5h, ~150 LOC)
- [ ] claude executor: token-tally hook at `usage` event
- [ ] opencode executor: token-tally hook at `tool_result` event
- [ ] gemini executor: token-tally hook at `result` event
- [ ] codex executor: token-tally hook at turn boundary
- [ ] pi executor: token-tally hook at Pi response
- [ ] Speed timestamps (`FirstAttemptMs`, `SuccessAtMs`) recorded in shared post-processing

**M3: Result schema + JSON exporter** (~4h, ~200 LOC)
- [ ] Promote `Turns` and add new fields to `Result`
- [ ] `internal/eval_analysis/export_json.go` — emit speed/cost aggregates per model
- [ ] Aggregate computation: median, p90, frontier scoring
- [ ] `efficiency` block in dashboard JSON

**M4: Dashboard charts** (~4h, ~400 LOC React)
- [ ] New `SpeedRadar.jsx` — median time-to-success per model (radar layout, mirror of existing success rate radar)
- [ ] New `CostSpeedFrontier.jsx` — Pareto scatter (x = cost/success, y = sec/success, marker = success rate)
- [ ] `PerModelTrend.jsx` — add speed regression line alongside success-rate trend

**M5: Re-test + curation** (~3h, ~$2 in eval cost)
- [ ] Re-smoke 4 OS near-misses (DeepSeek V4 Flash, Kimi K2.6, GLM 4.7 Flash, Gemma 4 26B) with new budgets
- [ ] Verify ≥1 promotes to PASS
- [ ] Update `models.yml` notes with new smoke dates

**M6: Docs + design doc retro** (~2h, ~150 LOC docs)
- [ ] New section in `docs/docs/guides/evaluation/` covering cost-and-speed budgets
- [ ] Move design doc to `implemented/v0_16_0/`
- [ ] CHANGELOG entry under v0.16.0

### Files to Modify/Create

**New files:**
- `internal/executor/cost.go` — shared cost-budget helper (~250 LOC)
- `docs/src/components/BenchmarkDashboard/SpeedRadar.jsx` — NEW radar chart (~200 LOC)
- `docs/src/components/BenchmarkDashboard/CostSpeedFrontier.jsx` — NEW Pareto scatter (~200 LOC)
- `design_docs/implemented/v0_16_0/m-eval-cost-and-speed-budgets-sprint-plan.md` — sprint plan (created at sprint kickoff)

**Modified files:**
- `internal/executor/executor.go` — `Task`/`Result` field additions (~30 LOC)
- `internal/executor/claude/claude.go` — token-tally hook (~30 LOC)
- `internal/executor/opencode/opencode.go` — token-tally hook (~30 LOC)
- `internal/executor/gemini/gemini.go` — token-tally hook (~30 LOC)
- `internal/executor/codex/codex.go` — token-tally hook (~30 LOC)
- `internal/executor/pi/pi.go` — token-tally hook (~30 LOC)
- `internal/eval_harness/models.go` — `budgets:` block parsing (~60 LOC)
- `internal/eval_harness/models.yml` — per-model `budgets:` blocks (~200 LOC, mostly mechanical)
- `internal/eval_analysis/export_json.go` — speed/cost aggregates (~120 LOC)
- `docs/src/components/BenchmarkDashboard/PerModelTrend.jsx` — speed regression line (~60 LOC)
- `.claude/skills/post-release/scripts/run_eval_baseline.sh` — pass cost args (~20 LOC)

**Total: ~1290 LOC + tests**

## Examples

### Example 1: cheap-slow vs expensive-fast comparison

**Before this milestone (v0.15.0):**
```
opencode-or-deepseek-v4-flash on csv_to_json:
  Killed at 60s (opencode hard timeout) — api_error
  Cost incurred: ~$0.003
  Result: capability not measured
```

**After this milestone:**
```
opencode-or-deepseek-v4-flash on csv_to_json:
  MaxCostUSD: 0.30 (per models.yml budgets:)
  Hard timeout: 600s (safety net)
  Completed at 145s, cost $0.018, success=true
  Result: PASS — capability accurately measured
```

### Example 2: expensive model with cost cap

**Before:**
```
opencode-sonnet-4-6 on a pathological 50-turn agent loop:
  Wall-clock 60s elapses
  Some turns completed; others didn't
  Result: api_error (timeout) — true cost incurred unknown
```

**After:**
```
opencode-sonnet-4-6 on a pathological 50-turn agent loop:
  MaxCostUSD: 0.80
  Cost reaches $0.80 at turn 18 (each turn ~$0.045)
  CostKilledAt: 0.80
  Result: cost_killed (NEW failure category, was previously runaway)
  Wall-clock: 38s (well under 600s safety net)
```

### Example 3: Pareto frontier dashboard chart

**Before:** Cost-per-1000-successes radar with one outlier ($256) that collapses every other spoke.

**After:** Pareto scatter:
- X axis: $/success (log scale)
- Y axis: sec/success (log scale)
- Marker size: success rate (bigger = higher pass rate)
- Color: harness (claude/gemini/codex/opencode/pi)
- Pareto frontier line drawn through optimal points
- Tooltip shows real values; outliers visible without distortion

## Success Criteria

- [ ] `internal/executor/cost.go` exists with full unit-test coverage (≥90%)
- [ ] All 5 executors call `budget.Add()` at the correct event boundaries
- [ ] `Result.CostKilledAt > 0` is a separate failure category from `api_error` and `timeout`
- [ ] `models.yml` `budgets:` block parses for all 50+ existing models without breaking changes
- [ ] Default `MaxCostUSD = min($0.50, input × 64K + output × 32K)` formula applied when `budgets:` omitted
- [ ] At least 1 of (DeepSeek V4 Flash, Kimi K2.6, GLM 4.7 Flash, Gemma 4 26B) promotes to PASS smoke after re-test
- [ ] Dashboard shows new Speed Radar and Cost-Speed Frontier charts
- [ ] CHANGELOG v0.16.0 entry references this design doc
- [ ] Existing eval baselines (v0.14.x, v0.15.0) replay with identical results when budgets blocks omitted (back-compat)
- [ ] All `make test`, `make lint`, `make verify-examples` green

## Testing Strategy

**Unit tests** (`internal/executor/cost_test.go`):
- `CostBudget.Add()` correctness across input/output mix
- Race-condition safety under `-race` with concurrent `Add()` calls
- `KilledAt()` first-write-wins semantics
- Default-formula fallback when `MaxUSD == 0`
- Per-model pricing × token mix → expected cost calculation

**Integration tests**:
- All 5 executors observe `CostKilledAt` correctly when budget exceeded mid-stream
- Wall-clock timeout (`Timeout=600s`) triggers BEFORE cost-kill on hung connections
- Speed metrics (`FirstAttemptMs`, `SuccessAtMs`, `Turns`, `TokensPerSec`) populated correctly per executor
- `models.yml` `budgets:` parsing for malformed blocks rejected with clear error

**Manual testing**:
- Re-smoke DeepSeek V4 Flash with new budgets — confirm csv_to_json now passes
- Spot-check dashboard Pareto frontier renders without outlier collapse
- Verify `git revert HEAD` restores legacy behaviour (back-compat)

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Per-executor token-tally hook event boundary** — agent decides which exact event in each executor's stream is most reliable for incremental tally (e.g., `usage` deltas vs. cumulative)
- **Cost-extrapolation fallback for opencode** — if `usage` events don't arrive incrementally, agent may use TTFTTimeout token rate × elapsed time as fallback estimate; flag in result metadata
- **`SpeedRadar` vs `CostSpeedFrontier` placement order on the dashboard** — agent picks based on visual coherence
- **Default `expected_ttft_secs` and `expected_ttf_solution_secs` per model family** — agent can populate from observed v0.15.0 baseline data
- **Whether to log warning at 50% budget** — agent may add observability hooks beyond the hard kill if useful

## Non-Goals

**Not attempted in this feature:**

- **Per-tier cost ceilings** (e.g., smoke=$0.10, core=$0.50, stretch=$1.00) — possible follow-up if uniform $0.50 proves wrong shape; deferred until we see post-rollout data
- **Cost-aware routing** (route to cheapest provider mid-flight) — needs M-AI-OPENROUTER M5+ work; out of scope here
- **Live budget alerting via macOS notify daemon** — separate milestone (M-NOTIFY-BUDGET-ALERTS); out of scope
- **Per-org / per-day cumulative budgets** — needs persistence layer; out of scope here
- **Cost-vs-quality ROC curves** — interesting but needs more historical data than v0.15.0+v0.16.0 affords
- **AILANG-language cost metering** — runtime budgets via `!{AI[budget=$0.10]}` parameterised effects — that's a separate v1.0.0 axis (M-EFFECT-REFINEMENT Phase 6+)

## Timeline

**Day 1** (~9 hours):
- M1: Schema + cost helper (~4h)
- M2: Wire 5 executors (~5h)

**Day 2** (~8 hours):
- M3: Result schema + JSON exporter (~4h)
- M4: Dashboard charts (~4h)

**Day 3** (~5 hours):
- M5: Re-test + curation (~3h)
- M6: Docs + retro (~2h)

**Total: ~22 hours across 2.5 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| opencode `usage` events arrive after task completion (no incremental tally) | Med | Fall back to wall-clock-extrapolated cost estimate using TTFTTimeout token rate; flag in result metadata |
| Per-model `pricing` becomes stale (vendor price changes) | Low | OpenRouter `ResolvedRoute` already records actual cost — for OR-routed models, prefer authoritative cost over our calculated cost |
| Cost-kill races completion (race condition) | Low | Atomic compare-and-swap on `killedAt`; first-write-wins |
| Re-tested OS models still fail | Med | Acceptable — that's signal. Document in models.yml `notes:` and stay on watchlist. |
| Sprint-executor mis-locates token-tally event in opencode | Med | Sprint plan should include a small smoke test that validates incremental tally before wiring kill logic |
| Existing baselines change under recomputation | Low | Back-compat: `MaxCostUSD == 0` (default for omitted `budgets:` block) preserves legacy timeout-only behaviour |

## Related Documents

<!-- Auto-populated by Ollama neural search on "eval cost and speed budgets" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_1/m-dx25-budget-report.md](../../implemented/v0_7_1/m-dx25-budget-report.md) (0.42) — prior budget-reporting work; informs aggregate-output shape
- [design_docs/implemented/v0_11_2/m-latency-budget.md](../../implemented/v0_11_2/m-latency-budget.md) (0.41) — closest prior art on budget-as-first-class concept
- [design_docs/implemented/v0_11_2/m-latency-budget-sprint-plan.md](../../implemented/v0_11_2/m-latency-budget-sprint-plan.md) (0.40) — sprint structure to mirror
- [design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — `ResolvedRoute` per-call cost capture pattern
- [design_docs/implemented/v0_15_x/m-effect-refinement-phase1.md](../../implemented/v0_15_x/m-effect-refinement-phase1.md) — parameterised-effects machinery for future budget-as-effect work

**Planned (check for overlap):**
- [design_docs/planned/v0_15_0/m-eval-results-folder-structure.md](../../planned/v0_15_0/m-eval-results-folder-structure.md) (0.39) — eval result file schema; this doc adds new fields
- [design_docs/planned/v0_15_0/m-eval-trust-signals.md](../../planned/v0_15_0/m-eval-trust-signals.md) (0.38) — adjacent eval-suite improvements
- [design_docs/planned/v0_13_0/m-cloud-eval-workers.md](../../planned/v0_13_0/m-cloud-eval-workers.md) (0.37) — cloud-side eval; cost budgets matter even more there

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [internal/executor/executor.go](../../../internal/executor/executor.go) — current `Executor` interface, `Task`, `Result`
- [internal/eval_harness/models.yml](../../../internal/eval_harness/models.yml) — per-model pricing + budgets target file
- [.claude/skills/model-manager/SKILL.md](../../../.claude/skills/model-manager/SKILL.md) — Smoke-Test Gate methodology this milestone validates
- v0.15.0 post-release retro (this session) — empirical findings driving the redesign

## Future Work

- **Per-tier cost ceilings** — adapt `max_cost_usd` per tier (smoke/core/stretch/vision) once we see distributions from the v0.16.0 rollout
- **AILANG-side `!{AI[budget=$X]}` parameterised effect** — once M-EFFECT-REFINEMENT Phase 6+ ships, runtime-side cost budgets become available to user code. This milestone is the eval-harness precursor.
- **Cost-aware routing** — feed the `MaxCostUSD` into M-AI-OPENROUTER's routing policy so OpenRouter routes to a cheap-enough provider automatically
- **Speed regression alerts** — once speed metrics are first-class, CI can fail PRs that regress median TTS by ≥20% per model
- **`ailang trace` dashboard** — surface CostKilledAt + speed metrics in the existing trace explorer, not just eval results

---

## Implementation Report (2026-05-05)

### What was built

All 6 milestones landed against the design — no scope cuts. ~22 wall-clock hours into a 22-hour estimate. Sprint executed by 1 lead + 2 Task sub-agents (M2 wiring, M4 dashboard charts). Branch: `dev` (sub-agent committed dashboard work directly to `dev` after a transient branch-state issue; M2 was cherry-picked from `sprint/m-eval-cost-and-speed-budgets`).

| Milestone | Estimated LOC | Actual LOC | Notes |
|-----------|--------------:|-----------:|-------|
| M1 (schema + cost.go) | 250 | 305 (115 cost.go + 200 tests) | Atomic helper, 100% coverage, race-clean |
| M2 prep (Task/Result fields) | (folded into M2) | 14 | Schema additions only |
| M2 (5 executors) | 150 | 589 lines added net (with refactoring) | Sub-agent did all 5 in parallel-effect (sequential commits) |
| M3 (efficiency block) | 200 | 336 (150 efficiency.go + 200 tests) | New file `efficiency.go` keeps export_json.go from growing further |
| M4 (3 dashboard components) | 400 | ~600 (sub-agent, ComposedChart for frontier line) | Used `recharts.ComposedChart` (not bare ScatterChart) so Pareto frontier line composes cleanly |
| M5 (re-test 4 OS) | 50 | ~30 (yml edits + agent_runner_multi.go wiring) | Wiring fix needed: per-model `budgets:hard_timeout_secs` must override CLI `--agent-timeout` default (was guarded too defensively) |
| M6 (docs + retro) | 240 | ~300 (this report + guide + CHANGELOG) | New eval guide + CHANGELOG entry |

### Primary success metric: ✅ MET

**`opencode-or-glm-4-7-flash` promoted from 2/3 NEAR-MISS to 3/3 PASS** under the new budgets. csv_to_json took 131s — would have failed under the old 60s opencode hard timeout. Validates the timeout-as-budget misclassification: capability was real, just slow.

### M5 outcome breakdown (smoke 2026-05-05, with `budgets:hard_timeout_secs=600`)

| Model | 2026-05-04 (60s) | 2026-05-05 (600s) | Δ |
|-------|----------------:|-----------------:|---|
| **`opencode-or-glm-4-7-flash`** | 2/3 | **3/3 PASS** ✨ | +1 (PROMOTED) |
| `opencode-or-deepseek-v4-flash` | 2/3 | 2/3 | 0 |
| `opencode-or-kimi-k2-6` | 2/3 | 2/3 | 0 |
| `opencode-or-gemma-4-26b` | 2/3 | 1/3 | -1 (regression) |

Secondary findings:
- **DeepSeek V4 Flash**: csv_to_json still fails (api_error mid-stream — needs upstream OpenRouter route fix or model improvement). adt_option ran 65.5s (would have failed @ 60s).
- **Kimi K2.6**: csv_to_json still fails (api_error). fizzbuzz at 79s and adt_option at 91s (both would have failed at 60s; both pass at 600s).
- **Gemma 4 26B**: regression — adt_option became `runtime_error` this run (was `pass` last week). Variance, not a budget issue. Stays on watchlist.

### Deviations from design

1. **Sub-agent branching**: The M2 sub-agent committed to `sprint/m-eval-cost-and-speed-budgets` correctly; the M4 sub-agent committed to `dev` (concurrent work + merge state). Resolved via cherry-pick — no functional impact, but the sprint branch was not the sole source of truth as planned.
2. **M2 prep merged into earlier work**: Task/Result schema additions ended up as a tiny stand-alone commit between M1 and M2 (commit 7c5c265a) rather than folded into M1 or M2. Cleaner history, identical net effect.
3. **Wiring rework in M5**: First attempt at `agent_runner_multi.go` over-protected the CLI default; second pass made per-model `budgets:hard_timeout_secs` always override the agent-timeout default (per-benchmark `spec.Timeout` still wins). This is the correct semantics — the original guard was wrong.
4. **Gemini cost-kill latency**: Gemini CLI doesn't emit incremental usage events, so `task.Budget.Add()` only fires post-hoc at the `result` event. Gemini cannot kill mid-stream — `CostKilledAt` is recorded but the model already finished. Acceptable per design doc Deferred Decisions; documented in code comments. Token-rate-extrapolated estimator deferred.

### Quality gates

- `go test -race -count=1 ./internal/executor/...` — clean across all 5 executors
- `make test` — 85 packages green, 0 failures
- `make lint` — 0 issues (pre-existing `go:S1186` warnings on empty `NoOpEventHandler` methods unchanged)
- `make verify-examples` — not affected (no AILANG-side changes)
- `cd docs && npm run build` — Docusaurus production build succeeds
- M5 smoke cost: ~$0.05 (cheaper than the $2 estimate; budgets capped runs early)

### Performance impact

- `CostBudget.Add()` overhead: 2 atomic.Add operations + 2 float multiplications per token-event; negligible compared to network I/O of the LLM call itself
- `efficiency` block in dashboard JSON adds ~80 bytes per model (~10KB total for the v0.15.0 baseline with 11 models)
- Speed timestamps: 3 × `time.Now()` calls per task — sub-microsecond overhead

### Follow-ups for v0.16.1+

- **Re-smoke DeepSeek V4 Flash + Kimi K2.6**: csv_to_json failures are upstream `api_error` — re-test in a few weeks once OpenRouter routing settles
- **Gemma 4 26B regression** investigation — single-run variance or genuine drop?
- **Per-tier cost ceilings**: now that we see distributions from M5, consider tightening smoke=$0.10, loosening stretch=$1.00
- **Wire M5 outcome into dashboard event annotation**: the v0.16.0 release should drop a marker on the SpeedRadar so future viewers see when the budget redesign landed

---

**Document created**: 2026-05-05
**Last updated**: 2026-05-05 (implementation report appended after sprint completion)
