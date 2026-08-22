# Cost-Per-Verified-Success KPI

**Status**: IMPLEMENTED (all milestones; M4b baseline fired 2026-08-22, iteration 253)
**Target**: v1.0.0
**Priority**: P1 (clause-5 headline KPI)
**Estimated**: 1.5 days (up to 2 days including the measured baseline run)
**Dependencies**: m-mission-cost-chains (LANDED iter-100), ValueDashboard, latest.json benchmark data

## Problem Statement

### Current State

Mark's ratified design freeze makes cost-per-success the headline KPI for the v1.0 claim: AILANG is "the verified AI-orchestration language" in which an AI author gets a verified-correct program at the lowest cost. The current product surface does not yet make that claim measurable as stated.

- The cost substrate is shipped. `internal/observatory/cost_classify.go` classifies every chain stage at read time as **reported**, **estimated**, **quota**, or **unknown**; `CostRollup.TotalKnownCost()` is reported plus estimated cost, quota lanes are `$0` by design, and unknown token-bearing stages make the rollup incomplete. `internal/observatory/mission_rollup.go` reuses that classifier for `ailang chains stats --by-mission`, while `ailang chains post-iteration` ingests mission evidence including the existing `metered=$` field.
- A live repository-local probe on 2026-07-27, using `ailang chains stats --hours 336`, returned 226 chains and a classified known cost of **$180.3808**: $180.3808 reported across 2,750 stages, $0 estimated across 2,378 stages, 654 quota stages, and 1 unknown stage. The command's legacy `Total Cost` line was $179.7148, so this KPI must use the classifier's known-cost total rather than the older raw SQL total. The same probe with `--by-mission` showed `mission:v1` at $0 metered with three Opus quota stages and one Sonnet quota stage; mission development spend is therefore visible, but it is not interchangeable with benchmark-run spend.
- `docs/static/benchmarks/latest.json` is currently version `v0.30.0`, timestamped 2026-07-22T18:44:18+02:00, with 1,810 runs, $98.32670983 aggregate benchmark cost, 0.808839779 aggregate final success, 17 standard models, and 7 agent models. It contains no `verify_*` or `verified` keys.
- `docs/src/components/BenchmarkDashboard/ValueScoreTable.jsx` already derives a **per-model pass-cost**. Standard mode computes `totalCostUSD / (finalSuccess × totalRuns)`; agent mode computes `(avgCost × runs) / (successRate × runs)`. That value appears as `$/success` and as one denominator inside the weighted Value Score. It counts ordinary benchmark passes, not verified successes.
- `docs/src/components/ValueDashboard/index.jsx` places the Quality-vs-Cost scatter and Value Score below the mode toggle. Its explanatory formula is `pass_rate^N / (cost_per_success × (1 + median_TTS/60))`; therefore cost-per-success exists only as an input to a secondary value lens, not as the top-line metric promised by the v1.0 claim. Data arrives through `@site/src/lib/benchmarkFetch`, which fetches the runtime GCS-backed `/benchmarks/latest.json` route and falls back to the build-time static copy.
- The verification substrate exists but is not carried into the published data. `EvalAssessment` and `RunMetrics` both define `verify_ok`, `verify_verified`, `verify_counterexample`, `verify_skipped`, and `verify_errors`. `internal/eval_harness/verify.go` populates them from `ailang ai-check`/Z3 when verification is available. However, both standard and agent `EvalAssessment` constructors in `cmd/ailang/eval_benchmark.go` currently omit those fields, and the current `latest.json` consequently cannot distinguish a golden-output pass from an independently verified program. The checked-in `eval_results` also contain zero files with `verify_ok: true` or a positive `verify_verified` count, so there is no repo-banked v1.0 verified baseline to publish yet.

### Impact

This is a release-credibility gap, not a cosmetic dashboard gap. Prospective users, release reviewers, and Mark cannot reproduce the headline claim from banked evidence. The current per-model `$/success` can reward a program that matches expected output without proving its contracts, while the stronger wording "verified-correct" implies an independent verification step. Until the denominator and cost provenance are explicit, the dashboard can show attractive cost numbers without demonstrating the v1.0 product thesis.

## Goals

**Primary Goal**: Publish one reproducible v1.0 top-line number equal to the total attributable metered dollars for a frozen AILANG agent benchmark cohort divided by the number of runs in that cohort that both pass execution grading and complete contract verification with no counterexamples, skipped obligations, or verifier errors.

### Success Metrics

1. A canonical stats result exposes `cost_per_verified_success_usd`, numerator dollars, verified-success count, total cohort runs, cost-provenance counts, and an explicit completeness flag.
2. The v1.0 headline uses real banked runs from a frozen, documented cohort; no synthetic or hand-entered numerator or denominator is accepted.
3. The primary dashboard surface displays **Cost per verified success** before the Value Score and QualityScatter lenses, with the baseline timestamp, cohort size, and verification definition visible.
4. `ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict` reproduces the published numerator, denominator, and quotient; `--strict` fails when any cohort stage has unknown cost or missing verification evidence.
5. Existing per-model `$/success`, ELO, Pareto, speed, standard/agent toggle, runtime GCS fetch, and fallback behavior remain unchanged.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By (human/agent/compiler) | Deadline (design/compile/runtime) | Change Cost |
|---|---|---|---|---|
| Define a **VERIFIED success** as an AILANG agent run where `compile_ok && runtime_ok && stdout_ok && verify_ok && verify_verified > 0 && verify_counterexample == 0 && verify_skipped == 0 && verify_errors == 0` | A benchmark pass alone proves observed behavior, not the v1.0 "verified-correct" claim. Requiring an actual proved obligation prevents `verify_ok` with zero proved functions, skipped obligations, or unavailable Z3 from entering the denominator. | agent, subject to Mark ratification | design | Medium if changed after baseline banking |
| Freeze the v1.0 baseline cohort to agent mode, language `ailang`, contract-bearing benchmarks, declared models/executors, seeds, prompt version, and a bounded run window recorded in baseline metadata | The single AILANG headline is otherwise sensitive to model mix, reruns, and cherry-picked benchmark subsets. The existing per-model table remains the comparison surface; the headline is an aggregate over one immutable release cohort. | human (Mark) | design, before baseline run | High after publication |
| Use `reported_cost + estimated_cost` for every run in the frozen cohort, including failures; exclude quota dollars but disclose quota-stage counts; refuse publication when unknown stages exist | Charging only successful runs would hide failed-attempt spend. Using the shipped classifier preserves provenance. Quota lanes are intentionally `$0`, while unknown costs would bias the KPI downward. | agent | runtime | Medium |
| Keep mission-loop development spend separate from benchmark KPI spend by filtering the exact eval cohort/source reference | `metered=$`, `post-iteration`, and `--by-mission` correctly track the cost of building AILANG, but dividing mission-development spend by benchmark successes would mix unlike units and make the release KPI depend on sprint activity. | agent | compile/runtime | Low |
| Place the headline on the main benchmark/leaderboard or landing surface and mirror it at the top of ValueDashboard, rather than leaving it only inside Value Score | "Headline KPI" requires a primary release-facing placement. The exact main-page slot is a product judgment, but ValueDashboard-only placement would remain too buried. | human (Mark) | design, before M2 | Low before implementation; Medium after launch |

## Proposed Design

### 1. Canonical Metric Definition

For one frozen v1.0 baseline cohort `C`:

`cost_per_verified_success_usd = total_known_metered_cost_usd(C) / verified_success_count(C)`

The numerator is the sum of `StageCost.CostUSD` for all benchmark stages in `C` classified as reported or estimated by the existing `ClassifyStageCost` logic. It includes spend from passing and failing attempts. Quota stages contribute `$0` but are counted and disclosed. If any stage is unknown, or if the denominator is zero, the result is `available=false`; the dashboard must show an incomplete/unavailable state rather than `$0`, infinity, or a stale prior value.

The denominator is not `stdout_ok` alone. A row counts only when it passes compile/runtime/stdout grading and has affirmative `ailang ai-check` evidence with at least one verified obligation, zero counterexamples, zero skipped obligations, and zero verifier errors. Runs without contract verification are outside the headline denominator and are reported as `unverified_passes` for auditability; they continue to count normally in the existing pass-rate and per-model `$/success` views.

The headline is a **single AILANG release-baseline number**, aggregated over the Mark-approved frozen agent cohort. It is distinct from:

1. **Per-model `$/success`** in `ValueScoreTable`: current benchmark pass-cost, retained for model comparison and the Value Score.
2. **Per-model cost per verified success**: optional rows derivable from the same canonical fields, useful for comparison but not required for this 1–2 day item.
3. **Mission development cost** in `chains stats --by-mission`: retained as operational budget evidence and explicitly excluded from the benchmark numerator.

### 2. Capture Verification Evidence

Reuse the existing `RunMetrics` and `EvalAssessment` fields; no schema migration is required because `eval_assessment` is JSON. The eval-to-chain writers copy all five verification fields for both standard and agent modes. The agent `RunMetrics` construction also carries the verification values already returned by `AgentBenchmarkResult`, so JSON result banking, chain assessment, and later aggregation agree.

Baseline eligibility additionally requires verification to have been attempted on a contract-bearing AILANG benchmark. A missing/default all-zero verification block is `verification_missing`, not a failed verification and not a success. This preserves historical data without falsely interpreting pre-feature zero values.

### 3. Observatory and Stats Layer

Add one narrow observatory rollup that queries assessed eval stages for a supplied immutable baseline filter and returns a structured result such as:

- `baseline_id`, `generated_at`, `language`, `eval_mode`, and frozen cohort metadata/hash;
- `total_runs`, `passed_runs`, `verified_successes`, `unverified_passes`, and `verification_failures`;
- `reported_cost_usd`, `estimated_cost_usd`, `known_cost_usd`, `quota_stages`, `unknown_stages`, and `incomplete_data`;
- `cost_per_verified_success_usd` and `available`.

The implementation stays **decoupled from the eval package**: outcome filtering queries the Observatory's **own** chains store and decodes the newly banked `eval_assessment` JSON column — via the existing `(*observatory.Store).QueryEvalResults` method (`internal/observatory/store_chains_eval.go`, an Observatory-local `Store` method, NOT an `internal/eval_harness` function — so there is no cross-package import and no import cycle). The numerator reuses `ClassifyStageCost`/`CostRollup`. It must not use `GetChainStatusCounts.TotalCost`, because the 2026-07-27 probe showed that legacy total diverging from the classified known total. Filtering is by the frozen eval source reference/cohort metadata, not by `mission:v1`.

Extend `internal/server/handlers_chains.go` so `GET /api/chains/stats` can return the canonical KPI for a named baseline without duplicating metric logic in JavaScript. Add the equivalent thin CLI presentation in `cmd/ailang/chains_stats.go` so the release baseline is reproducible locally with:

`ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict`

The CLI and HTTP handler call the same observatory method and serialize the same fields. The strict command exits non-zero for unknown cost, missing verification evidence in an expected-to-verify cohort, a zero denominator, or a cohort metadata mismatch.

### 4. Publish Through `latest.json`

After the v1.0 baseline run is banked, the existing benchmark publication step snapshots the canonical result into `latest.json` under one additive object, for example `headlineKpis.costPerVerifiedSuccess`. The object includes the value and all audit fields listed above; it is not a bare floating-point number.

This preserves the existing dashboard delivery architecture: `benchmarkFetch('latest.json')` continues to use the runtime GCS-backed route with the static fallback. The server stats response remains the canonical computation and reproducibility surface; `latest.json` is the immutable published snapshot consumed by the public dashboard.

### 5. Dashboard Presentation

Add a compact headline KPI card before the Quality-vs-Cost section in `ValueDashboard` and on the Mark-selected primary benchmark/landing surface. The card shows:

- **Cost per verified success: `$X.XXXX`**;
- `verified_successes / total_runs` and total known metered dollars;
- `Reported + Estimated`, quota-stage count, baseline timestamp/version, and a short definition tooltip;
- a visible **Incomplete** state when `available=false` or `incomplete_data=true`;
- a visible **Fallback / stale data** badge when `benchmarkFetch` serves the build-time static fallback instead of the runtime GCS copy. This source warning is distinct from the KPI-computation Incomplete state: `benchmarkFetch` must expose a small additive signal identifying which source was used so the card can surface degraded delivery. The fallback is retained for availability, but it is never silent because stale release evidence can affect user decisions.

The existing Value Score and scatter remain secondary diagnostic lenses. Their current `costPerSuccess` derivation and labels are not silently redefined; changing them to verified-success semantics would break historical comparisons and is outside this item. A short label distinguishes them as "cost per benchmark pass" where ambiguity exists.

## Milestones

1. **M1 — Capture and compute (0.5 day):** propagate verification fields and add the strict observatory rollup; acceptance check: fixtures distinguish pass-only, verified success, verifier failure, skipped proof, quota, estimated cost, and unknown cost.
2. **M2 — Expose and publish (0.25–0.5 day):** add matching CLI/server output and the additive `latest.json` headline object; acceptance check: CLI JSON, HTTP JSON, and published snapshot have identical numerator, denominator, quotient, and completeness fields.
3. **M3 — Headline dashboard (0.25 day):** render the KPI card on the Mark-approved primary surface and atop ValueDashboard; acceptance check: the metric appears before Value Score/QualityScatter and handles available, zero-denominator, and incomplete states.
4. **M4 — Measure v1.0 baseline.** SPLIT (2026-07-27) once BF-1 showed the write side could not
   produce a non-zero denominator at all:
   - **M4a — Cohort freeze mechanism (LANDED, sprint `M-COST-KPI-M4A`, 0.5 day).** `eval-suite
     --baseline <id>` writes the cohort-prefixed `source_ref` the M1 filter matches; a data-driven
     `cohort_manifest.json` + `cohort_hash` records the cohort (models resolved from `models.yml`,
     nothing hardcoded in Go); the baseline-id charset is pinned on both sides; and agent-mode
     contract verification is moved onto the live multi-executor path so `verify_verified > 0` is
     reachable at all (see BF-1 below). Zero benchmark executions, zero metered spend.
   - **M4b — Measured baseline (NOT STARTED, Mark-gated on spend).** Actually run the frozen cohort,
     publish the measured number, and record the command/output in release documentation; acceptance
     check: a clean invocation of the documented strict command reproduces the displayed value from
     banked data. Real metered exposure on the OpenRouter lanes. **Blocked until M4a landed — now
     satisfied — and still awaiting the human's spend approval.**

## Conflict Surface

**No compiler surface (parser/types/elaborate/core/codegen) touched — N/A.** There is no parser, typechecker, elaboration, core IR, interpreter, or code-generation conflict surface for this dashboard and metric-definition item.

The real conflict surface is limited to:

- **Observatory cost semantics:** `ClassifyStageCost`, `CostRollup.TotalKnownCost()`, quota `$0` handling, and unknown-cost fail-loud behavior are authoritative and must not be reimplemented differently in the handler, CLI, exporter, or React code. Reported cost remains authoritative; estimated cost remains read-side only.
- **Eval assessment JSON:** existing historical rows omit verification values. New logic must treat absent/all-zero verification as missing evidence, not as success. Adding JSON fields must remain backward-compatible with stored rows and `QueryEvalResults`.
- **`latest.json` schema:** the headline object is additive. Existing `models`, `agentModels`, `aggregates`, `ratings`, and efficiency fields must remain stable so other dashboard components do not regress.
- **`ValueScoreTable` derivation:** its current per-model `costPerSuccess` is cost per benchmark pass, with separate standard and agent calculations. This design does not replace or silently alter that value; it only clarifies its label relative to the new strict headline.
- **Dashboard runtime fetch:** `benchmarkFetch` must continue remote GCS fetch first and local static fallback second. The KPI must render from either copy, surface the static-fallback state visibly so stale data is never rendered silently, and must not introduce a second client-side cost calculation.
- **Server stats compatibility:** existing `/api/chains/stats` fields and `by_agent` behavior remain unchanged; the baseline KPI response is additive and calls observatory logic rather than duplicating SQL in the handler.
- **Mission cost ingest:** `metered=$`, `ailang chains post-iteration`, mission rollups, budget checks, and `--by-mission` output must not change. They provide cost credibility and provenance but are excluded from the benchmark cohort calculation.

## Acceptance Criteria

- [ ] The repository defines VERIFIED success exactly once as compile/runtime/stdout pass plus available, positive, complete `ai-check`/Z3 proof: `verify_ok`, `verify_verified > 0`, zero counterexamples, zero skipped obligations, and zero verifier errors.
- [ ] Both standard and agent eval paths bank the existing verification fields into run JSON and chain `eval_assessment` without a database schema migration.
- [ ] The numerator uses real classified reported plus estimated dollars for all frozen-cohort runs, including failures; quota and unknown counts are disclosed separately.
- [ ] Any unknown cost, missing expected verification evidence, zero denominator, or cohort mismatch makes the KPI unavailable and causes the documented strict command to exit non-zero.
- [ ] `ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict` reproduces the baseline from banked data and emits the cohort identity, numerator, denominator, quotient, and provenance counts.
- [ ] The same result is published additively in `docs/static/benchmarks/latest.json` and its runtime GCS copy, with a timestamp and baseline/cohort identifier.
- [ ] The dashboard's primary headline visibly reads **Cost per verified success** and shows the measured v1.0 number before the existing Value Score and scatter lenses.
- [ ] The KPI card shows a visible **Fallback / stale data** badge when the static copy is served, verified by a component/data-logic fixture.
- [ ] The existing per-model `$/success` remains cost per benchmark pass and continues to produce the same rows and values for the pre-change `latest.json` fixture.
- [ ] Standard/agent mode separation, ELO ratings, Pareto flags, speed metrics, runtime GCS fetch, static fallback, mission cost rollups, and existing chain stats fields do not regress.
- [ ] The published v1.0 baseline is accompanied by the exact frozen cohort manifest and command output; a reviewer can independently recompute it without dashboard JavaScript.

## Risks & Open Questions

- **BF-1 (FIXED in M4a, was an absolute M4b blocker) — agent-mode verification was on a DEAD code
  path.** The only agent-mode `RunAICheck` call lived in `RunAgentBenchmark()`
  (`internal/eval_harness/agent_runner.go`), which had **no caller**: `cmd/ailang/eval_benchmark.go`
  always used `RunAgentBenchmarkWithExecutor`, and `grep -n Verify agent_runner_multi.go` returned
  zero matches. So no agent run could ever bank `verify_verified > 0`, `isVerifiedSuccess()` was
  permanently false, and the KPI would have returned `available:false, reason:"zero_denominator"`
  **forever** — meaning an M4b metered run would have spent real money for an unavailable KPI. This
  also explains why no banked run has ever had positive verification: a harness gap, not a lack of
  opportunity. Fixed in M4a-3 by moving verification onto the live path and deleting the dead runner.
  **Any future claim that "verification is wired" must be checked against the LIVE call graph, not
  the presence of the code.**
- **BF-3 (FIXED in M4a):** `--verify-timeout` never reached the agent verifier (hardcoded `5s`), so
  the cohort manifest's `verify_timeout` field would have documented a timeout the run never used.
- **No currently banked verified cohort:** the present `latest.json` has no verification fields, and checked-in eval results have no positive verification outcomes. M4 must run and bank a new strict cohort; historical pass data cannot be relabeled as verified.
- **Verification coverage:** only AILANG contract-bearing benchmarks can meet the strict denominator. A small eligible set may make the headline volatile or unrepresentative. The card must display `verified_successes / total_runs` and the frozen benchmark count.
- **Unknown/unattributed cost:** one unknown stage appeared in the 2026-07-27 14-day chain probe. If an unknown stage falls inside the baseline cohort, publishing a quotient would understate cost; strict mode therefore blocks publication.
- **Quota versus metered spend:** quota lanes are `$0` by design in the shipped classifier, not free compute. The headline must say "metered cost per verified success" and disclose quota-stage counts so subscription usage is not mistaken for no resource cost.
- **Legacy total divergence:** the live probe's raw total ($179.7148) differed from classified known cost ($180.3808). Any path that uses the old aggregate instead of `CostRollup` can publish a contradictory numerator.
- **Small-sample and model-mix volatility:** aggregating different AI authors into one AILANG number can move the KPI when the mix changes. The v1.0 cohort must be immutable after Mark approves it; later data publishes as a new baseline ID rather than rewriting v1.0.
- **Open — headline placement (Mark):** approve the recommended main benchmark/leaderboard placement mirrored at the top of ValueDashboard, or name another primary landing surface. ValueDashboard-only placement does not satisfy the ratified "headline" directive.
- **Open — cohort manifest (Mark):** approve the exact agent models/executors, contract benchmark IDs, seeds, prompt version, and run count before M4. The implementation supplies a proposed manifest from the current v1 suite; it must not infer or silently change the product-comparison mix at runtime.

## Testing Strategy

### M1 — Metric Logic

- Add table-driven unit tests around the canonical observatory computation for: failed paid run; stdout-only pass; verified success; `verify_ok` with zero verified functions; counterexample; skipped obligation; verifier error; reported cost; estimated cost; quota stage; unknown model; and zero denominator.
- Assert the numerator includes failed-run cost and equals `reported_cost_usd + estimated_cost_usd` exactly.
- Add eval mapping tests proving agent and standard assessment JSON retain all five verification fields and that omitted historical fields deserialize as missing evidence, never success.

### M2 — Reproducibility and Publishing

- Use a temporary Observatory database fixture with a frozen baseline source reference. Compare the observatory result, `/api/chains/stats` response, CLI `--json --strict` output, and generated `latest.json` headline object field-for-field.
- Verify strict mode exits non-zero for unknown cost, missing verification, a zero denominator, and a mismatched baseline/cohort hash.
- Record the final v1.0 command and JSON output in the release evidence so the measured number can be rerun independently.

### M3 — Dashboard

- Run the existing docs Babel/build check to catch JSX/schema errors.
- Add a focused data-logic/component test or fixture check for available, incomplete, and absent-headline objects; the absent field must degrade to "baseline unavailable", not break the page.
- Add a fallback-source fixture check proving that serving the static copy renders the visible **Fallback / stale data** badge.
- Confirm the pre-change fixture yields unchanged `ValueScoreTable` per-model `$/success`, ELO, Pareto, and speed values.
- Verify `benchmarkFetch` still loads the runtime GCS `latest.json` and falls back to `docs/static/benchmarks/latest.json` with the same headline rendering.

### M4b — Measured Baseline (Mark-gated spend; M4a's mechanism tests are in the sprint plan)

- Run the Mark-approved frozen agent/AILANG/contract cohort with verification enabled and bank every result.
- Run `ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict`; require exit 0 and archive its full output.
- Independently recompute `known_cost_usd / verified_successes` from the archived JSON and compare it to the dashboard value at full stored precision before display rounding.

## Quorum Verification Log

**Author (designer):** `codex:gpt-5.6-sol` (rotation designer, iter-103). Reviewers: `gemini-3-1-pro` (independent of the codex author, per generator≠judge) + Claude controller (opus) in-session.

- **Round 1 (2026-07-27T05:16Z) — BLOCKED.** gemini-3-1-pro objection: the Dashboard Presentation reused `benchmarkFetch`'s GCS→static fallback but did not surface a stale/degraded render, violating the NO-SILENT-FALLBACKS axiom. Controller: pass. → Routed to the designer for one revision.
- **Designer revision (codex):** added a mandatory visible **Fallback / stale data** badge + a `benchmarkFetch` source signal across §5, Conflict Surface, Acceptance Criteria, and M3 testing. R1 objection resolved.
- **Round 2 / re-quorum (2026-07-27T05:19Z) — BLOCKED (new, distinct objection).** gemini-3-1-pro: §3 "reuses `QueryEvalResults`" reads as a cross-package call into the eval harness → alleged boundary violation / import cycle. Proposed fix: "the Observatory must remain decoupled from the Eval package and perform outcome filtering by directly querying its own chains store and decoding the newly banked `eval_assessment` JSON column." Controller: pass.
- **NARROW-REFINEMENT CARVE-OUT applied (controller, ratified iter-98).** The R2 objection is direction-preserving with a concrete reviewer-authored fix, so it is carve-out-eligible. On repo verification the objection's *premise* is a phantom: `QueryEvalResults` is **already** an Observatory-local `(*Store)` method (`internal/observatory/store_chains_eval.go:68`) that decodes the observatory's own `eval_assessment` JSON — there is no eval-package dependency and no import cycle (observatory imports `eval_harness` only for the `models.yml` pricing path, `internal/observatory/pricing.go:11`). The reviewer's intent is thus already the design; the controller applied it verbatim as a §3 clarifying edit making `QueryEvalResults`'s observatory-local nature explicit, preempting the naming-induced misread. No behavior change, no controller-invented resolution. Design direction accepted by all parties in both rounds.

**Status:** quorum-cleared (carve-out). Ready to route to sprint-planner.
