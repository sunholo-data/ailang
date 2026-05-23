# M-EVAL-OS-LONGITUDINAL Phases 2–5 — Sprint Plan

**Sprint ID**: M-EVAL-OS-LONGITUDINAL-P2-P5
**Design Doc**: [m-eval-os-longitudinal.md](./m-eval-os-longitudinal.md)
**Target Version**: v0.23.0
**Estimated Duration**: 6 days (~46 hours)
**Total LOC**: ~570 (impl + tests + docs)
**Risk Level**: Low–Medium
**Phase 1 status**: ✅ Already shipped (commit `207a64c1`) — `-max-tokens-per-bench` flag + thrash_aborted category + tests

## Goal

Complete the remaining four phases of M-EVAL-OS-LONGITUDINAL, turning the local Ollama eval rig from "runs benchmarks" into "the closed feedback loop for AILANG language design with publishable longitudinal data."

The chain we're building:

```
N≥3 trials per benchmark per release          [Phase 3]
       ↓
Adaptive per-(model, benchmark) budget         [Phase 2]
       ↓
Auto-surfaced persistent-failure candidates    [Phase 4]
       ↓
Per-release Docusaurus pass-rate publication   [Phase 5]
```

After this sprint, a single `make eval-release v0.23.0` will: run N=3 trials of the canonical smoke set, populate the rolling baseline, generate a candidate list of language-level failures, and publish a Docusaurus page comparing this release to the last one. That's the killer-app artifact for AILANG's "designed for AI" positioning.

## Why this order

Dependency analysis from the design doc:
- **Phase 3 (N-trials)** is independent and unlocks Phases 4 + 5 (both need multi-trial data)
- **Phase 2 (adaptive baseline)** needs N≥5 passing data points to be meaningful — best fed by Phase 3
- **Phase 4 (candidates command)** consumes Phase 3 output to identify "persistent ≥2/N trials"
- **Phase 5 (publication)** consumes Phase 3 + 4 output for the per-release tables

So sequential ordering is: **Phase 3 → Phase 2 → Phase 4 → Phase 5.** Each phase ships independently and each later phase is more valuable with the previous one in place.

## Velocity context

- Phase 1 shipped in ~3 hours (estimated 1 day) — ahead of doc estimate
- Recent committed work: 1549 LOC across 14 files over the last 5 commits (very healthy pace)
- The M-EVAL-LOCAL-OBSERVABILITY sprint (4 milestones, ~700 LOC) shipped in ~3 hours total

Realistic: this sprint completes in 4–5 working days if focused, with the doc estimate of 6 as a buffer.

## Milestones

### M1 — Phase 3: N-trial release smoke flag (~100 LOC, 1 day)

The unlock for everything else. Add `--trials N` to `ailang eval-suite -agent` so each benchmark runs N times in a single invocation, and the output directory gains a `summary.json` aggregating pass-rate + token distribution per benchmark.

**Files**:
- `cmd/ailang/eval_suite.go` (+30): `--trials N` int flag, default 1; print "running N trials per benchmark" banner.
- `internal/eval_harness/agent_runner_multi.go` (+30): outer loop wrapping the existing per-benchmark execution; each trial writes its own result JSON with `trial: K` field added.
- `internal/eval_harness/metrics.go` (+40): new `SummarizeRotation(outputDir string) → RotationSummary` aggregator; writes `summary.json`.
- `internal/eval_harness/spec.go` or similar (+5): result JSON gains `trial: int` field (default 1 for backward-compat).
- `cmd/ailang/eval_suite_flags_test.go` or new test (+30): unit test for SummarizeRotation across multi-trial output.

**Acceptance criteria**:
- `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent --trials 3 -benchmarks fizzbuzz -output /tmp/test'` writes 3 fizzbuzz result JSONs + 1 `summary.json`
- `summary.json` contains `{benchmark_id, trials, passed, pass_rate, tokens_pass_mean, tokens_pass_stddev, tokens_fail_mean}` per benchmark
- Default `--trials 1` produces identical output to today's single-trial run (no regression)
- Test passes against synthetic multi-trial fixture data
- No regression in existing eval-harness or cmd/ailang tests

**Dependencies**: None
**Risk**: Low — outer loop on existing well-tested path; new aggregator is pure function over JSON

---

### M2 — Phase 2: adaptive per-model per-benchmark budget (~200 LOC, 2 days)

Replace the fixed `--max-tokens-per-bench` ceiling with a per-(model, benchmark) rolling mean+stddev maintained via Welford's online algorithm. The eval-suite reads the baseline before each benchmark and uses `mean + 2*stddev` as the abort threshold once N≥5 passing samples exist.

**Files**:
- `internal/observatory/schema.sql` (+10): new `eval_baselines` table (model_id, benchmark_id, n_pass_trials, mean_tokens, stddev_tokens, m2_tokens, last_updated)
- `internal/observatory/baselines.go` **NEW** (+120): Welford update function, GetBaseline/UpdateBaseline backend methods, abort-threshold computation
- `internal/observatory/baselines_test.go` **NEW** (+50): convergence test (10 synthetic samples → mean ≈ expected, σ ≈ expected); edge cases (n=0, n=1, fresh row)
- `internal/executor/opencode/opencode.go` (+20): replace fixed threshold lookup with `baselines.GetThreshold(model, benchmark, fallbackFixed)` call; after run completes, call `baselines.UpdatePass()` if PASS
- `internal/executor/opencode/opencode_test.go` (+20): regression test confirming the bootstrap path (n<5 falls back to fixed ceiling)

**Acceptance criteria**:
- After running fizzbuzz against gemma4:26b 5 times with PASS outcomes, the `eval_baselines` row has non-zero `mean_tokens` and `stddev_tokens`
- The 6th run computes an abort threshold from the baseline, not the fixed flag
- A run that exceeds `mean + 2*stddev` is aborted with `thrash_aborted` category
- Welford update is numerically stable (test asserts σ matches `numpy.std` to 4 decimal places)
- Bootstrap (n<5) gracefully falls back to the fixed `--max-tokens-per-bench` flag value
- Existing single-trial and multi-trial paths both work with adaptive baseline

**Dependencies**: M1 (adaptive needs N≥5 PASS samples; Phase 3 multi-trial mode is the natural source)
**Risk**: Medium — numerical stability of online algorithms, transactional baseline updates under concurrent runs

---

### M3 — Phase 4: persistent-failure candidates command (~70 LOC + skill update, 1 day)

New `ailang eval-trend candidates` subcommand that consumes Phase 3 `summary.json` and emits a structured candidate list for human triage. This is what closes the "failure → design doc" loop the user articulated.

**Files**:
- `cmd/ailang/eval_trend.go` **NEW** (+80): subcommand dispatcher; `candidates` action reads `summary.json` files under a rotation directory; filters to `pass_rate < 0.5 AND error_category consistent across trials`; emits structured table
- `cmd/ailang/eval_trend_test.go` **NEW** (+30): fixture-based test for candidate detection
- `.claude/skills/eval-analyzer/SKILL.md` (+20): add a "Triggered by `eval-trend candidates` output" section showing the deep-dive workflow

**Acceptance criteria**:
- `ailang eval-trend candidates --release v0.23.0` scans `eval_results/rotation/<date>/` directories for `summary.json` files
- Emits one row per (benchmark, error_category) tuple where ≥2/N trials failed with the same category
- Output format: `benchmark | category | n_fail/n_trials | example_token_count | example_session_id`
- Suggests `eval-analyzer` skill for per-candidate deep-dive
- Test covers: synthetic fixture with 3 trials × 5 benchmarks, asserts exactly the expected candidates surface

**Dependencies**: M1 (consumes `summary.json` format)
**Risk**: Low — pure read-only aggregation; failure case is "no candidates surfaced" which is just informational

---

### M4 — Phase 5: longitudinal Docusaurus publication (~200 LOC, 2 days)

`ailang eval-publish <release-tag>` generates the per-release Docusaurus page comparing pass rates across models and against the previous release.

**Files**:
- `cmd/ailang/eval_publish.go` **NEW** (+150): subcommand; reads `eval_results/rotation/<date>/summary.json` aggregates across the release window; computes trend deltas vs previous release; emits Markdown
- `cmd/ailang/eval_publish_test.go` **NEW** (+40): fixture-based test for table generation
- `docs/docs/reference/os-model-leaderboard/index.md` **NEW** (+30): static index page that lists per-release pages
- `docs/docs/reference/os-model-leaderboard/v0_22_0_seed.md` **NEW** (+30): backfilled seed data from the runs we've already done (allows the v0.23.0 release page to show a Δ row)
- `docs/sidebars.js` (+5): add `os-model-leaderboard` to the sidebar

**Acceptance criteria**:
- `ailang eval-publish v0.23.0` produces `docs/docs/reference/os-model-leaderboard/v0_23_0.md` with two tables:
  - Per-benchmark pass rate at N trials, one column per model
  - Trend deltas vs v0.22.0 for any benchmark where the gap is ≥10 percentage points
- `make docs-build` (or equivalent docusaurus build) succeeds with the new page rendered
- Test fixture covers: 2 releases × 2 models × 3 benchmarks, asserts trend table only includes ≥10pp deltas

**Dependencies**: M1 (consumes `summary.json`), M3 (uses `eval-trend` aggregations for the candidates highlight section)
**Risk**: Low–Medium — Markdown generation is straightforward; the only friction is docusaurus sidebar registration

---

## Day-by-Day Plan

| Day | Work | Cumulative LOC shipped |
|-----|------|------------------------|
| Day 1 | M1: `--trials N` + summary.json + tests + commit | ~100 |
| Day 2 | M2 part 1: `eval_baselines` table + Welford helper + tests | ~250 |
| Day 3 | M2 part 2: integrate into opencode executor + integration test + commit | ~370 |
| Day 4 | M3: `eval-trend candidates` command + skill wire-up + commit | ~440 |
| Day 5 | M4 part 1: `eval-publish` aggregator + Markdown emitter + tests | ~590 |
| Day 6 | M4 part 2: docusaurus integration + seed-data backfill + commit + final smoke verification | ~700 |

Buffer day: catch-up if M2 numerical-stability work bites.

## Success Metrics

When this sprint completes:

- [ ] `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent --trials 3 ...'` runs each benchmark 3 times and writes `summary.json`
- [ ] After 5 PASS runs of any (model, benchmark), the `eval_baselines` row has non-zero σ
- [ ] `ailang eval-trend candidates --release v0.23.0` surfaces at least one persistent failure (we already know `dense_operator_program` qualifies)
- [ ] `ailang eval-publish v0.23.0` produces a Docusaurus page that builds cleanly
- [ ] First v0.23.0 release uses the new infra end-to-end (rotation → trials → candidates → publication)
- [ ] CHANGELOG entries per phase
- [ ] All four phases tested + lint-clean + committed

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Welford updates race under concurrent eval runs | Medium | Wrap UpdateBaseline in a transaction; tests run sequentially; document "single-rig assumption" |
| `eval_baselines` table grows large across many models | Low | Schema is per-(model, benchmark) - bounded by their cross-product; ~50 models x ~50 benchmarks = 2.5K rows max |
| Docusaurus build breaks on the new page | Low | Test the build locally before committing |
| Phase 3 backward-compat (default --trials 1) | Low | Add a regression test that single-trial output matches today's format exactly |
| First eval-publish release has no previous release to diff against | Low | M4 includes a seed-data backfill (`v0_22_0_seed.md`) so v0.23.0 has a trend baseline |

## Dependencies & Open Questions

**Already decided** (from the design doc):
- Phase 1 fix shape: fixed flag first, adaptive layered on top (DONE)
- Baseline storage: observatory.db `eval_baselines` table (this sprint)
- Default trial count: N=1 (no behavior change), opt-in via `--trials N`

**Open for this sprint**:
- Threshold sensitivity: is `mean + 2*sigma` right, or do we want `+1.5*sigma`? Tunable via a config knob in M2.
- Should `--trials` interact with `-parallel`? At N=3 and `-parallel 2`, do we run 3 trials of benchmark-1 then 3 of benchmark-2, or interleave? **Recommendation**: keep simple — run all N trials of one benchmark before moving to the next; `-parallel 2` still applies between benchmarks.
- Should the eval-publish page include cost data alongside pass rate? **Recommendation**: yes — `pricing * tokens` is a useful "is the rotation getting cheaper per benchmark?" signal. Free local models will always be $0, but the per-token-time field tells a similar story.

## Acceptance / Done Definition

This sprint is DONE when:

1. All 4 milestones (M1-M4) merged
2. Manual verification: `ailang eval-suite -agent --trials 3 ... -benchmarks fizzbuzz` writes 3 result JSONs + summary.json
3. After 5 PASS runs of fizzbuzz against gemma4:26b, `eval_baselines` row populated with non-zero stddev
4. `ailang eval-trend candidates --release v0.23.0` returns at least one candidate (dense_operator_program is the known one)
5. `ailang eval-publish v0.23.0` produces a Docusaurus page; `make docs-build` (or equivalent) succeeds
6. CHANGELOG entries written
7. Sprint plan + design doc moved to `implemented/v0_23_0/`

## Handoff

After approval, sprint-executor takes over with the JSON progress file at `.ailang/state/sprints/sprint_M-EVAL-OS-LONGITUDINAL-P2-P5.json`.

## Sibling tracks (not in this sprint)

- **`M-PIPE-OPERATOR-WIRING`**: the one-line parser fix surfaced by the dense_operator_program candidate analysis. Independent milestone; can ship anytime. Not in this sprint because it's a language-design change, not eval-infrastructure work.
- **Future rotation expansion**: pulling new local models (qwen2.5-coder:32b, phi4:14b, etc.) is gated on this sprint completing so we have trustworthy N=3 data before comparing models.