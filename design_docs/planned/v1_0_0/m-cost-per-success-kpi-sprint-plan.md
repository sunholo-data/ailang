# Sprint Plan: Cost-Per-Verified-Success KPI (M1–M3)

**Design doc**: [m-cost-per-success-kpi.md](m-cost-per-success-kpi.md)
**Sprint ID**: M-COST-PER-SUCCESS-KPI
**Status**: Planned (quorum-cleared iter-103, carve-out)
**Target**: v1.0.0
**Risk level**: Medium
**Scope**: **M1–M3 only. M4 (Measured Baseline) is GATED ON MARK and is NOT in this sprint.**

---

## Sprint Summary

**Goal**: Ship the plumbing, computation, publishing path, and dashboard presentation for the
Cost-Per-Verified-Success KPI so that the moment Mark ratifies the verified-success definition,
the v1.0 cohort manifest, and headline placement (M4 gate), the measured baseline can be run and
published with zero further code work.

**What this sprint delivers (M1–M3):**
1. **M1** — Verification-field propagation into both eval assessment constructors + a strict
   observatory rollup that computes the canonical KPI from banked `eval_assessment` JSON, reusing
   `ClassifyStageCost`/`CostRollup.TotalKnownCost()`.
2. **M2** — Matching CLI (`ailang chains stats --cost-per-verified-success --baseline <id> --json --strict`)
   and HTTP (`GET /api/chains/stats`) surfaces, plus the additive `latest.json` `headlineKpis.costPerVerifiedSuccess`
   publish path. All three emit field-for-field identical output.
3. **M3** — Headline KPI card in `ValueDashboard` (before Value Score / QualityScatter) with
   available / zero-denominator / incomplete states and a visible **Fallback / stale data** badge
   driven by a `benchmarkFetch` source signal.

**What this sprint does NOT deliver (deferred to M4, Mark-gated):**
- Freezing the v1.0 cohort manifest (models/executors/seeds/prompt version/run count).
- Running the actual v1.0 baseline eval cohort and banking verified results.
- Publishing a *measured* headline number.
- Mark's decision on the exact primary landing-surface slot (the doc's second Mark-gated decision).
  → M3 renders the card **atop ValueDashboard** (the doc-sanctioned, non-Mark-gated location).
  The additional "main benchmark/leaderboard/landing" mirror placement is Mark's call and is left
  as a follow-up wiring task, not blocked-on but not force-placed.

**Total estimated effort**: **1.0 day** (M1 0.5d + M2 0.25–0.5d + M3 0.25d), plus buffer → **plan as ~1.5 days**.

---

## Ground-Truth Verification (planner probe, 2026-07-27)

The design doc's implementation facts were re-verified against the repo on branch `dev`. All confirmed:

| Doc claim | Verified? | Evidence |
|---|---|---|
| `verify_*` fields DEFINED on `EvalAssessment` but OMITTED by constructors | ✅ | `internal/observatory/models_chains.go:157-161` defines all 5; `cmd/ailang/eval_benchmark.go:349` (agent) & `:618` (standard) omit them |
| Agent + standard `RunMetrics`/`AgentBenchmarkResult` carry verify values | ✅ | `metrics.go:58-62`, `agent_runner.go:103-107,353-357` |
| `QueryEvalResults` is an Observatory-local `(*Store)` method (no eval import) | ✅ | `internal/observatory/store_chains_eval.go:68` |
| `ClassifyStageCost` / `CostRollup.TotalKnownCost()` authoritative | ✅ | `internal/observatory/cost_classify.go:78,148` |
| CLI `chains stats` already has `--strict` + `TotalKnownCost()` wiring | ✅ | `cmd/ailang/chains_stats.go:116-122,197` (extension point, not new surface) |
| HTTP `GET /api/chains/stats` handler exists | ✅ | `internal/server/handlers_chains.go:429` (extend, don't duplicate) |
| Dashboard components + `benchmarkFetch` exist | ✅ | `docs/src/components/ValueDashboard/index.jsx`, `.../BenchmarkDashboard/ValueScoreTable.jsx`, `docs/src/lib/benchmarkFetch.js` |

**Premises surfaced that DIVERGE from the design doc text (not silently worked around):**

1. **`latest.json` metadata drift (cosmetic).** The doc records `latest.json` as version `v0.30.0`,
   timestamp `2026-07-22T18:44:18+02:00`, 1,810 runs, `$98.32670983` aggregate cost. On probe the
   version is still `v0.30.0` and 1,810 runs, but the timestamp is now `2026-07-25T12:53:29+02:00`.
   This is the expected rig data-refresh cadence, not a contradiction. **No action** — M2's publish
   step is additive and version-agnostic.

2. **`EvalQueryOptions` has no cohort/baseline filter (expected M1 work, but the doc is slightly imprecise).**
   The doc's §3 says filtering is "by the frozen eval source reference/cohort metadata." Today
   `EvalQueryOptions` (`models_chains.go:175`) exposes `Model/Language/BenchmarkID/Condition/EvalMode/SuccessOnly/FailureOnly`
   but **no baseline-id / source-reference / created-window filter**. M1 must add the cohort filter
   dimension (a `BaselineID`/source-ref + time-window or explicit benchmark-set) to `EvalQueryOptions`
   OR resolve the cohort inside the new rollup. This is *implied* by the doc but not called out as a
   struct change; it is folded into M1 as an explicit task. **No cohort is frozen in this sprint** —
   the filter is parameterized and the `--baseline v1.0` value stays un-materialized until M4.

3. **Numerator source of stages.** The KPI numerator sums `StageCost.CostUSD` over *benchmark* stages
   in the cohort. `QueryEvalResults` returns `*ChainStage` rows carrying the `eval_assessment`; the
   rollup must classify each via `ClassifyStageCost` and sum reported+estimated. This is consistent
   with the doc; flagged only so the executor does not reach for `GetChainStatusCounts.TotalCost`
   (the divergent legacy total, $179.71 vs $180.38) — explicitly forbidden by the Conflict Surface.

---

## Velocity Basis

Recent landed mission items of comparable shape (single-lane, observatory/CLI + tests, additive):
- `m-mission-cost-chains` (iter-100, PR #478) — the direct substrate this builds on; observatory
  rollup + `chains` CLI + tests, landed inside a ~0.5–1d mission lane.
- `m-check-strict-fallbacks` (iter-101, PR #479) — single detector + tests, ~0.5d.
- `m-budget-scoping-bug` (iter-98, PR #474) — effects + tests, ~1d.

These are Go-side observatory/CLI/handler changes with table-driven tests — the same profile as M1+M2.
M3 is a small JSX card + data-logic fixture (docs Babel check, no headless browser). The design doc's
own estimate (1.5d incl. baseline) minus M4 lands at ~1.0d of build; **plan 1.5d with buffer.**

---

## Milestone Breakdown (TDD-first ordering)

### M1 — Capture and Compute (0.5 day)

**Goal**: verification evidence reaches banked `eval_assessment` for both modes, and a strict
observatory rollup computes the canonical KPI from banked data + the shipped cost classifier.

**Estimated LOC**: ~220 impl + ~180 test = ~400

**Tasks (test-first):**

1. **[TEST]** Add table-driven tests for the new observatory rollup (`cost_per_verified_success_test.go`)
   covering the 12 cases from the doc's Testing Strategy §M1: failed paid run; stdout-only pass;
   verified success; `verify_ok` with zero verified functions; counterexample; skipped obligation;
   verifier error; reported cost; estimated cost; quota stage; unknown model/unknown cost; zero
   denominator. Assert numerator == `reported_cost_usd + estimated_cost_usd` **including failed-run
   cost**, and that unknown cost / zero denominator ⇒ `available=false` / `incomplete_data=true`.
2. **[TEST]** Eval-mapping tests: agent + standard `EvalAssessment` JSON round-trip retains all 5
   verify fields; historical rows with omitted/all-zero verify block deserialize as
   `verification_missing` (never success).
3. **[IMPL]** Propagate the 5 verify fields into **both** `EvalAssessment` constructors in
   `cmd/ailang/eval_benchmark.go` (agent `:349`, standard `:618`) from `AgentBenchmarkResult` /
   `RunMetrics`. No schema migration (`eval_assessment` is JSON).
4. **[IMPL]** Extend `EvalQueryOptions` (or the rollup input) with the cohort/baseline filter
   dimension (source-ref + window / benchmark-set). Parameterized only — no v1.0 cohort materialized.
5. **[IMPL]** Add the narrow observatory rollup (new file, e.g. `internal/observatory/cost_per_verified_success.go`)
   that: queries via `(*Store).QueryEvalResults`; applies the VERIFIED-success predicate
   (`compile_ok && runtime_ok && stdout_ok && verify_ok && verify_verified>0 && counterex==0 && skipped==0 && errors==0`);
   classifies each stage via `ClassifyStageCost`; sums reported+estimated via `CostRollup`; returns
   the full structured result (`baseline_id, generated_at, language, eval_mode, cohort hash,
   total_runs, passed_runs, verified_successes, unverified_passes, verification_failures,
   reported_cost_usd, estimated_cost_usd, known_cost_usd, quota_stages, unknown_stages,
   incomplete_data, cost_per_verified_success_usd, available`).
6. **[VERIFY]** `make check-boundaries` (observatory stays dashboard-layer, no core import).
   `go test ./internal/observatory/...` green.

**Acceptance:**
- Fixtures distinguish pass-only, verified success, verifier failure, skipped proof, quota,
  estimated cost, and unknown cost.
- Numerator = reported+estimated exactly, includes failed-run cost.
- VERIFIED-success predicate defined **exactly once** (in the rollup), matching the doc's decision row.
- Both eval paths bank all 5 verify fields; no DB migration.

---

### M2 — Expose and Publish (0.25–0.5 day)

**Goal**: identical KPI output from CLI, HTTP handler, and the additive `latest.json` snapshot; strict
mode fails loudly.

**Estimated LOC**: ~180 impl + ~140 test = ~320

**Tasks (test-first):**

1. **[TEST]** Temp Observatory DB fixture with a frozen baseline source-ref. Assert the observatory
   result, `/api/chains/stats` JSON, and CLI `--json --strict` output are **field-for-field identical**
   (numerator, denominator, quotient, provenance counts, completeness).
2. **[TEST]** Strict-mode failure tests: exit non-zero for (a) unknown cost, (b) missing verification
   evidence in an expected-to-verify cohort, (c) zero denominator, (d) cohort/baseline hash mismatch.
3. **[TEST]** `latest.json` publish test: generated headline object matches the observatory result and
   existing `models/agentModels/aggregates/ratings`/efficiency fields remain byte-stable.
4. **[IMPL]** Extend `cmd/ailang/chains_stats.go` with `--cost-per-verified-success`, `--baseline <id>`,
   reusing the existing `--json` / `--strict` flags. Thin presentation only — calls the M1 rollup.
5. **[IMPL]** Extend `internal/server/handlers_chains.go` `handleChainsStats` to return the additive
   baseline KPI when requested. No SQL duplication — calls the same observatory method.
6. **[IMPL]** Extend the benchmark publication step so it snapshots the canonical result into
   `latest.json` under `headlineKpis.costPerVerifiedSuccess` (additive object with value + all audit
   fields, never a bare float). **No baseline is materialized in this sprint** — the publish code path
   is exercised via fixture; wiring to a real cohort is M4.
7. **[VERIFY]** `go test ./internal/server/... ./cmd/... ./internal/observatory/...` green;
   `make check-boundaries`.

**Acceptance:**
- CLI JSON, HTTP JSON, and published snapshot have identical numerator/denominator/quotient/completeness.
- Strict mode exits non-zero on unknown cost, missing verification, zero denominator, cohort mismatch.
- `latest.json` headline object is additive; existing fields stable.

---

### M3 — Headline Dashboard (0.25 day)

**Goal**: render the KPI card atop `ValueDashboard`, before Value Score / QualityScatter, with
available / zero-denominator / incomplete states and a visible fallback-source badge.

**Estimated LOC**: ~140 impl + ~90 test = ~230

**Tasks (test-first):**

1. **[TEST]** Data-logic / component fixture check for three `headlineKpis.costPerVerifiedSuccess`
   states: available, incomplete (`available=false` / `incomplete_data=true`), absent (degrade to
   "baseline unavailable" — must NOT break the page).
2. **[TEST]** Fallback-source fixture: serving the static copy renders the visible **Fallback / stale
   data** badge (driven by the `benchmarkFetch` source signal).
3. **[TEST]** Regression fixture: pre-change `latest.json` still yields unchanged `ValueScoreTable`
   per-model `$/success`, ELO, Pareto, and speed values (existing per-model `$/success` remains
   "cost per benchmark pass", not silently redefined).
4. **[IMPL]** Add a small additive source signal to `docs/src/lib/benchmarkFetch.js` identifying
   whether the runtime GCS copy or the build-time static fallback was served. GCS-first / static-second
   order unchanged; no second client-side cost calculation.
5. **[IMPL]** Add the compact headline KPI card component; render it in
   `docs/src/components/ValueDashboard/index.jsx` **before** the Quality-vs-Cost section. Card shows:
   `Cost per verified success: $X.XXXX`, `verified_successes / total_runs`, total known metered dollars,
   `Reported + Estimated`, quota-stage count, baseline timestamp/version, definition tooltip, Incomplete
   state, and Fallback/stale badge.
6. **[VERIFY]** Docs Babel/build check (JSX + schema). **No headless-browser verification** — describe
   expected visuals and let the user confirm.

**Acceptance:**
- Card appears before Value Score / QualityScatter and handles available, zero-denominator, incomplete states.
- Fallback badge verified by a component/data-logic fixture.
- Existing per-model `$/success`, ELO, Pareto, speed, mode toggle, GCS fetch + fallback unchanged.

---

## Conflict Surface Guardrails (carry into every commit)

- **Never** reimplement cost semantics: `ClassifyStageCost`, `CostRollup.TotalKnownCost()`, quota `$0`,
  unknown-cost fail-loud are authoritative in one place (observatory). Handler/CLI/publisher/React
  must call it, not recompute.
- **Never** use `GetChainStatusCounts.TotalCost` for the numerator (divergent legacy total).
- Treat absent/all-zero verify block as `verification_missing`, never success (backward-compat with
  stored rows + `QueryEvalResults`).
- `latest.json` headline object is **additive**; `models/agentModels/aggregates/ratings`/efficiency stable.
- `benchmarkFetch`: GCS-first, static-fallback-second, fallback state **visible** (never silent).
- Mission cost ingest (`metered=$`, `post-iteration`, `--by-mission`, budget checks) unchanged and
  excluded from the benchmark cohort.
- `make check-boundaries` on every cross-cutting change (CI gate; observatory must not import core).

---

## Success Metrics

- [ ] `go test ./internal/observatory/... ./internal/server/... ./cmd/...` green.
- [ ] `make check-boundaries` passes.
- [ ] Docs Babel/build check passes for the new card.
- [ ] VERIFIED-success defined exactly once; both eval paths bank verify fields (no migration).
- [ ] CLI == HTTP == published snapshot, field-for-field, on the fixture.
- [ ] Strict mode fails loudly on unknown cost / missing verification / zero denom / cohort mismatch.
- [ ] Dashboard card renders atop ValueDashboard with all three states + fallback badge; existing
      lenses unregressed.

---

## Out of Scope (M4 — Mark-gated, DO NOT start)

- Freeze + run the v1.0 contract-bearing agent cohort; bank verified results.
- Publish the *measured* headline number and record command/output in release evidence.
- Mark's ratification of: the verified-success definition, the cohort manifest, and the primary
  landing-surface placement.
- The additional "main benchmark/leaderboard/landing" mirror placement (Mark's product judgment).

**Handoff signal for M4**: once M1–M3 land, the code is ready. The measured baseline is a
data-and-ratification task, not a code task.

---

## Handoff Note (planner → executor)

- This is a **substrate-complete** sprint: every cost/verify primitive already exists and is
  authoritative. The work is *propagation + one rollup + thin surfaces + one card*, NOT new cost math.
- Start M1 with the table-driven rollup tests — they pin the VERIFIED-success predicate and the
  numerator-includes-failures rule before any wiring.
- The one non-obvious struct change is the **cohort/baseline filter dimension** on `EvalQueryOptions`
  (or rollup input) — the doc implies it but doesn't spell it out (planner premise #2). Keep it
  parameterized; do NOT hardcode a v1.0 cohort (that's M4/Mark).
- Do not touch mission-cost ingest or the legacy `TotalCost` path.
- Verify frontend via Babel + data-logic fixtures only; do not use headless Chrome — describe the
  card and let the user confirm.
- Stop at the end of M3. M4 is Mark-gated.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-cost-per-success-kpi-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-COST-PER-SUCCESS-KPI.json`
