# M-EVAL-OS-LONGITUDINAL — Longitudinal OS-model eval as language-design feedback

**Status**: Planned
**Target**: v0.23.0 (multi-phase; first phases can ship piecemeal)
**Priority**: P1 — Strategic infrastructure
**Estimated**: 2–3 weeks (multi-phase)
**Dependencies**: [M-EVAL-LOCAL-OLLAMA](../m-eval-local-ollama.md), [M-EVAL-LOCAL-OBSERVABILITY](../v0_22_0/m-eval-local-observability.md) (both shipped)

## TL;DR

The Mac Studio rig already runs gemma4:26b for free against the AILANG eval suite, and the M-EVAL-LOCAL-OBSERVABILITY stack gives us per-turn span data. This milestone turns that infrastructure from a *measurement substrate* into a **closed feedback loop for AILANG language design**:

```
failures → identify pattern → design doc → ship fix → measure improvement
                                                          on same (model, benchmark)
```

Three concrete components:

1. **Adaptive thrash abort** — track per-(model, benchmark) expected token distribution; kill runs that exceed mean + Nσ instead of letting them burn 2.88M tokens of broken AILANG (observed today in fizzbuzz).
2. **Failure → design doc workflow** — when a benchmark fails persistently (≥2 of 3 trials with same `error_category`), surface it as a candidate language-improvement target. Already-eval-analyzer-adjacent; this milestone formalizes the loop.
3. **Longitudinal pass-rate publication** — at each AILANG release, publish `(release, model, benchmark) → pass_rate, token_distribution` so external observers can see whether AILANG is becoming more or less tractable for OS models over time.

Together these let us **publish OS-model eval data across many more dimensions than the paid-models leaderboard** (Anthropic/OpenAI/Google rate-limit us; gemma4:26b doesn't). That's a competitive differentiator for AILANG's "designed for AI" positioning.

## Problem Statement

**Observation from the M-EVAL-LOCAL-OLLAMA investigation (May 2026):**

We have 9 runs of fizzbuzz against gemma4:26b. 6 PASS, 3 FAIL — 67% pass rate. Token counts swing 26× between the cleanest pass (110K) and the worst thrash (2.88M). When the model thrashes, it spends 10+ minutes emitting variations of broken AILANG code, eventually hitting the wall-clock timeout. Those thrashing runs are:

1. **Wasted compute**. ~13 minutes of GPU time producing no signal.
2. **Latency tax on the rotation**. The smoke tier wall clock is bounded by the slowest benchmark; one thrasher drags the whole batch.
3. **Lost as analytic signal**. We learn nothing from "model produced 2.88M tokens of broken AILANG" — we already knew it doesn't know AILANG well. We DO learn from "model produced 50K tokens of broken AILANG with this specific shape" — short failures are more diagnostic than long ones.

**What we want instead:**

For each (model, benchmark) pair, maintain a rolling expectation of solution-length tokens. When a run exceeds that expectation by Nσ, **abort early** and label the failure as "thrash-aborted." Free up the slot for the next benchmark immediately.

**Mission framing the user articulated:**

> "We are also running evals with the overall goal of identifying AILANG difficulties for AI models — failures are good if they can lead to new design docs we implement and can see improvements comparing to same model / benchmark."

This is the closed-loop. The rotation isn't just performance measurement — it's the AILANG language-design test bench. Every persistent failure is a candidate fix in the stdlib, prompt, or syntax. Every fix gets validated by re-running the same (model, benchmark) and watching the failure disappear or the token count drop.

**Why local models specifically:**

> "We can utilize the free token nature to really get a working feedback on the language and of course be able to publish them at each release across many more dimensions than paid models."

Cloud APIs cost real money per call. We get maybe 2–3 runs per model per release because of cost. Local gemma4:26b costs $0 — we can run 30 trials per (model, benchmark), produce real distributions, and republish on every release. **This is the killer differentiator** for OS-model evaluation. No cloud-only competitor can publish at this granularity.

## Goals

**Primary goal:** Make the local rotation a closed feedback loop for AILANG design improvements, with publishable longitudinal pass-rate data across releases.

**Success metrics:**

1. **Adaptive abort cuts smoke-tier wall clock by ≥30%** by killing thrashers early.
2. **Token cost per failed-run drops by ≥70%** for repeated thrashers on the same benchmark.
3. **At each release**, the project publishes a `(release, model, benchmark, n_trials, pass_rate, token_distribution)` table on the docs site.
4. **At least 2 design docs ship per quarter** that originated from a persistent eval failure — closing the failures→design loop.
5. **The longitudinal dashboard shows ≥1 benchmark per release where pass_rate moved** (up or down) by ≥10 percentage points — proving the rotation is sensitive enough to measure language-level changes.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Abort threshold: fixed `--max-tokens-per-bench N` flag vs adaptive mean+σ | Adaptive needs a baseline DB; fixed flag is trivial to ship now | human | design | low (fixed) / med (adaptive) |
| Where the per-(model, benchmark) baseline lives | observatory.db (queryable) vs separate `eval_baselines.db` (cleaner) | human | design | med |
| How many trials per benchmark in the canonical "release smoke" | 3? 5? 10? More = better σ, more compute | human | design | low (just a flag) |
| Whether to auto-create design docs from persistent failures | full automation invites noise; assistive workflow is safer | human | design | med |
| Publication format on docs site | static page (markdown table per release) vs interactive (Docusaurus plugin) | human | design | low / high |

### Design Freeze

- [ ] Decide fixed-flag vs adaptive abort for Phase 1 (recommendation: fixed `--max-tokens-per-bench` first; adaptive in Phase 2)
- [ ] Decide canonical trial count for release smoke (recommendation: N=3 default, `--trials N` flag overrides)
- [ ] Decide baseline storage location (recommendation: observatory.db, new `eval_baselines` table)

### Deferred Decisions

- Cross-model abort sharing (gemma4's mean used to abort qwen3:32b's run?) — probably never, per-model only.
- Whether to auto-file GitHub issues from persistent failures — out of scope; design doc workflow is enough.
- Whether to publish FAILED-trial output content (transcripts) or just statistics — privacy/space concerns; statistics-only for v1.

## Solution Design

### Phase 1: Fixed-flag thrash abort (~80 LOC, 1 day)

Smallest viable abort gate: a CLI flag that kills the executor subprocess if cumulative tokens exceed a fixed budget.

```bash
ailang eval-suite -agent -models opencode-gemma4-26b \
  -benchmarks fizzbuzz \
  -max-tokens-per-bench 500000   # NEW: abort if exceeded
```

**Implementation:**

- `cmd/ailang/eval_suite.go`: add `maxTokensPerBench int` flag plumbed into `AgentBenchmarkConfig`.
- `internal/executor/opencode/opencode.go`: in the streaming event loop where token deltas are observed, cumulative sum; if > limit, cancel the context (existing cancellation path already terminates the subprocess).
- New `error_category: "thrash_aborted"` for runs killed this way — distinguishable from `timeout` (wall-clock) and `compile_error` (completed but wrong).

**Acceptance:**

- A fizzbuzz run that would have thrashed to 2.88M tokens now aborts at the limit and writes a result with `error_category: thrash_aborted` in <2 minutes.
- Passing runs unaffected (limit is a ceiling, not a floor).
- Smoke tier wall clock drops measurably (validate by re-running the same smoke set with and without the flag on a sample of N=3 each).

### Phase 2: Adaptive abort with rolling baseline (~200 LOC, 2 days)

Replace the fixed flag with a per-(model, benchmark) running mean and σ.

**Storage** (recommended in observatory.db, new table):

```sql
CREATE TABLE eval_baselines (
    model_id      TEXT NOT NULL,
    benchmark_id  TEXT NOT NULL,
    n_pass_trials INTEGER NOT NULL DEFAULT 0,
    mean_tokens   REAL    NOT NULL DEFAULT 0,
    stddev_tokens REAL    NOT NULL DEFAULT 0,
    m2_tokens     REAL    NOT NULL DEFAULT 0,
    last_updated  TIMESTAMP NOT NULL,
    PRIMARY KEY (model_id, benchmark_id)
);
```

**Update on every PASSING run** (Welford's online algorithm — no need to store individual token counts):

```go
// After a PASS, update the baseline for (model, benchmark)
delta := newTokens - baseline.MeanTokens
baseline.NPassTrials++
baseline.MeanTokens += delta / float64(baseline.NPassTrials)
delta2 := newTokens - baseline.MeanTokens
baseline.M2 += delta * delta2
if baseline.NPassTrials > 1 {
    baseline.StddevTokens = math.Sqrt(baseline.M2 / float64(baseline.NPassTrials - 1))
}
```

**Abort decision at runtime** (during streaming):

```go
// In opencode executor event loop
abortThreshold := baseline.MeanTokens + 2.0 * baseline.StddevTokens
if baseline.NPassTrials < 5 {
    // Not enough data; fall back to fixed limit (Phase 1's flag value)
    abortThreshold = fixedMaxTokens
}
if cumulativeTokens > abortThreshold {
    cancel()
    return result.WithCategory("thrash_aborted")
}
```

**Bootstrap**: first 5 PASS runs of any (model, benchmark) pair use Phase-1's fixed flag as the abort gate. After 5 passes the σ is meaningful.

### Phase 3: N-trial release smoke (~100 LOC, 1 day)

Currently `ailang eval-suite -benchmarks X` runs once. Add:

```bash
ailang eval-suite -agent -models opencode-gemma4-26b \
  -benchmarks fizzbuzz \
  --trials 3                # NEW: run each benchmark 3 times
```

Output directory layout becomes:

```
eval_results/rotation/2026-05-23/0100_gemma4_smoke/
  agent/
    fizzbuzz_trial_1_opencode-gemma4-26b_<ts>.json
    fizzbuzz_trial_2_opencode-gemma4-26b_<ts>.json
    fizzbuzz_trial_3_opencode-gemma4-26b_<ts>.json
    ...
  summary.json   # NEW: pass_rate per benchmark, σ, etc.
```

`summary.json` is the publishable artifact:

```json
{
  "model": "opencode-gemma4-26b",
  "release": "v0.23.0",
  "rotation_date": "2026-05-23",
  "trials_per_benchmark": 3,
  "results": [
    {
      "benchmark_id": "fizzbuzz",
      "trials": 3,
      "passed": 2,
      "pass_rate": 0.667,
      "tokens_pass_mean": 145000,
      "tokens_pass_stddev": 30000,
      "tokens_fail_mean": 2100000,
      "abort_count": 1
    }
  ]
}
```

### Phase 4: Failure → design-doc workflow (assistive, no automation) (~50 LOC + skill update, 1 day)

When the same (model, benchmark, error_category) tuple fails ≥2 of N≥3 trials in a release smoke, emit a structured "candidate" record:

```bash
ailang eval-trend candidates --release v0.23.0 --min-trials 3
```

Output:

```
Persistent failure candidates (2/3+ same error_category, v0.23.0):
  benchmark              category        n_fail/n_trials  example_token_count
  ────────────────────────────────────────────────────────────────────────
  fizzbuzz               compile_error   3/3              2880000 (thrash)
  dense_operator_program compile_error   3/3              148000 (genuine syntax gap)
  numeric_modulo         logic_error     2/3              72000  (logic bug)

For each candidate, consider whether the failure suggests:
- a stdlib gap that a new function would close
- a teaching-prompt gap that an additional example would close
- a syntax footgun that a parser improvement could remove
- a benchmark spec that should be clarified (not a real failure)

Use `eval-analyzer` skill on a specific candidate for deep-dive.
```

This is assistive — not automated. It surfaces candidates; humans decide which become design docs. The `eval-analyzer` skill already covers the deep-dive.

### Phase 5: Longitudinal publication (~150 LOC + Docusaurus page, 2 days)

At each release, after `make eval-baseline EVAL_VERSION=v0.X.Y`:

1. `ailang eval-publish v0.23.0` aggregates the rotation data into a Docusaurus-ready markdown page.
2. Output: `docs/docs/reference/os-model-leaderboard/v0_23_0.md` with tables like:

```
## v0.23.0 OS-model smoke tier results (3 trials per benchmark)

| Benchmark              | gemma4:26b | qwen2.5-coder:32b | phi4:14b |
|------------------------|------------|-------------------|----------|
| fizzbuzz               | 67% (n=3)  | not run           | not run  |
| adt_option             | 33% (n=3)  | not run           | not run  |

### Trends since v0.22.0

| Benchmark      | v0.22.0 gemma4 | v0.23.0 gemma4 | Δ |
|----------------|----------------|----------------|---|
| fizzbuzz       | 67% (n=9)      | 67% (n=3)      | 0 |
| numeric_modulo | 0% (n=3)       | 67% (n=3)      | +67pp ← std/numeric refinement |
```

This is the **publishable longitudinal artifact** that closes the loop with external observers.

## Files to Modify

| Phase | File | LOC | Why |
|---|---|---|---|
| 1 | `cmd/ailang/eval_suite.go` | +10 | Add `-max-tokens-per-bench` flag |
| 1 | `internal/executor/opencode/opencode.go` | +20 | Token sum in streaming loop; cancel on exceed |
| 1 | `internal/eval_harness/agent_runner.go` | +5 | Plumb flag into AgentBenchmarkConfig |
| 1 | `internal/eval_harness/error_categorizer.go` | +5 | New `thrash_aborted` category |
| 1 | tests | +30 | Regression: thrash run is killed at limit |
| 2 | `internal/observatory/schema.sql` | +10 | New `eval_baselines` table |
| 2 | `internal/observatory/baselines.go` (new) | +100 | Welford online algorithm + getters |
| 2 | `internal/executor/opencode/opencode.go` | +20 | Adaptive threshold lookup |
| 2 | tests | +50 | Bootstrap path, threshold computation |
| 3 | `cmd/ailang/eval_suite.go` | +30 | `-trials N` flag |
| 3 | `internal/eval_harness/agent_runner_multi.go` | +30 | Loop N times per benchmark |
| 3 | `internal/eval_harness/metrics.go` | +40 | Summary aggregation |
| 4 | `cmd/ailang/eval_trend.go` (new subcommand) | +80 | `candidates` + `deep-dive` |
| 4 | `.claude/skills/eval-analyzer/` | +20 | Wire into candidates output |
| 5 | `cmd/ailang/eval_publish.go` (new) | +150 | Aggregate → Docusaurus markdown |
| 5 | `docs/docs/reference/os-model-leaderboard/` (new section) | +50 | Static index page |

**Total estimated: ~650 LOC + tests. Phaseable — each phase is independently shippable.**

## Conflict Surface

Not a parser/typechecker change. Eval-suite surface changes:

| Position | Existing | This change |
|---|---|---|
| `error_category` enum | compile_error, logic_error, runtime_error, api_error, timeout | + thrash_aborted |
| eval result JSON | one file per (benchmark, model, lang) | one per (benchmark, model, lang, **trial**) when `--trials N>1` |
| observatory.db schema | (no eval_baselines table) | + eval_baselines table (additive) |

**Programs that must still work:**
1. `make eval-suite` / `make eval-baseline` default flags — unchanged behavior (N=1, no abort) when new flags not specified
2. `ailang eval-summary` / `ailang eval-matrix` / `ailang eval-report` — handle the new `thrash_aborted` category gracefully
3. `ailang chains live` — unchanged
4. Coordinator-driven eval invocations — unchanged

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Doesn't change determinism story |
| A2: Replayability | +1 | Trial counts enable proper statistical replay |
| A3: Effect Legibility | 0 | No effect-system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Token-budget abort is a bounded-resource guarantee |
| A6: Safe Concurrency | 0 | No concurrency model changes |
| A7: Machines First | +2 | Machine-readable eval feedback; publication makes this externally measurable |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +2 | Token distribution per benchmark per model becomes first-class; thrash abort makes runaway cost impossible |
| A10: Composability | 0 | New subcommands compose with existing chains tooling |
| A11: Structured Failure | +1 | `thrash_aborted` is a structured, distinguishable failure mode |
| A12: System Boundary | 0 | Eval boundary unchanged |

**Net Score: +7** → **Proceed.**

**Hard violation check:** None.

## Success Criteria

- [ ] Phase 1: A run that thrashes to 2.88M tokens now aborts at the configured limit and writes `thrash_aborted` category in <2 min.
- [ ] Phase 2: After 5 passing runs of `gemma4:26b` × `fizzbuzz`, the eval_baselines row has non-zero mean+stddev and the abort threshold is computed automatically.
- [ ] Phase 3: `make eval-smoke ... --trials 3` produces 3 result files per benchmark + a summary.json with pass_rate.
- [ ] Phase 4: `ailang eval-trend candidates` identifies at least one persistent-failure candidate from a sample rotation.
- [ ] Phase 5: First release using this infra publishes a Docusaurus page comparing two releases' pass rates.
- [ ] CHANGELOG entries per phase.
- [ ] All existing eval-suite invocations continue to work unchanged.

## Timeline

| Phase | Day | Work |
|---|---|---|
| 1 | Day 1 | Fixed-flag thrash abort + tests |
| 2 | Days 2–3 | Adaptive baseline (Welford + schema + plumb) |
| 3 | Day 4 | `--trials N` flag + summary.json |
| 4 | Day 5 | `ailang eval-trend candidates/deep-dive` |
| 5 | Days 6–8 | Publication: `eval-publish` + Docusaurus pages |
| Buffer | Days 9–10 | First end-to-end release using the new infra |

Phases are independently shippable. Phase 1+3 alone deliver most of the user-facing value (thrash abort + trustworthy multi-trial signal). Phases 4+5 turn the infra into a release-loop story.

## Related Documents

- [M-EVAL-LOCAL-OLLAMA](../m-eval-local-ollama.md) — runs the rig
- [M-EVAL-LOCAL-OBSERVABILITY](../v0_22_0/m-eval-local-observability.md) + [FOLLOWUP](../v0_22_0/m-eval-local-observability-followup.md) — observes the rig
- This doc — closes the design-improvement loop on top of both
- `eval-analyzer` skill — does the per-candidate deep-dive

## Open Questions

1. Should `--trials N` ship alongside Phase 1's thrash abort? The user articulated trial-count as a need; might be worth bundling.
2. For the publication page, what cadence beats per-release? Maybe weekly snapshot too, so monthly rollups show trends within a release cycle.
3. Should `eval-analyzer` skill auto-link to `eval-trend candidates` output, so the design-doc creation step always starts from a candidates list?
4. Are there benchmarks we EXPECT to fail forever (intentional difficult benchmarks marking ceiling capability)? If so, mark them so they don't trigger design-doc candidacy.
