# M-EVAL-VALIDITY-DISCIPLINE: like-for-like, coverage-gated eval comparisons everywhere

**Status**: IN PROGRESS — coverage gating + per-model coverage landed 2026-07-11 (ratings block + ELO leaderboard). Remaining: uplift/delta like-for-like, cross-mode/harness labelling, tests, and **W9 (added 2026-08-11): the coverage gate compares counts, not benchmark-set identity**. **W8 SPLIT OUT 2026-08-13** to [m-eval-w8-harness-errors-as-capability-failures.md](m-eval-w8-harness-errors-as-capability-failures.md) — Mark took the pending human decision so it can route without waiting on W9; this doc no longer blocks it.
**Quorum**: 2 rounds run 2026-08-11 (iter-178), artifacts `m-eval-validity-discipline-2026-08-11T17-46-03Z.json` and `…T17-49-21Z.json`. Both rounds **BLOCKED**, both reviewers **present** (`absent_reviewers: []` — no N−1 hole), metered **$0.0955** total. Every objection from both rounds was measured rather than forwarded and its `proposed_fix` adopted VERBATIM (R1→W9 ACs, R2→AC-W8.3 + the Conflict Surface). **The doc is NOT cleared to route**: round 2's surviving R1 objection disputes the design *direction* of **W9**, so per the mission-control carve-out this parks `needs-human-review` rather than taking a controller-authored third round. **W8 is untouched by either objection** — whether W8 may route on its own, in its own scoped doc, is the human decision on the bookkeeping issue.
**Target**: v0.30.x (eval infrastructure + dashboard)
**Priority**: P1 — every benchmark run has surfaced a *new manifestation of the same class of bug* (invalid cross-cohort comparison). This is the fix that stops the cycle.
**Author**: Claude Opus 4.8 (requested by Mark, 2026-07-11 — "every benchmark run we need to fix all this data and reliability")

---

## The recurring bug (one class, many faces)
The eval **display** has no validity discipline, so each run surfaces a new invalid comparison:
- **Sparse local models posted a bogus #1** — qwen ran 6 benchmarks, cloud ran 55; the merged ELO ranked qwen top because ELO across *disjoint benchmark sets* isn't comparable.
- **"Agent uplift" was mostly not like-for-like** — standard included the 2 impossible reimplements (0%) that agent excluded; and several "uplifts" compared `or-deepseek` (standard API) vs `opencode-or-deepseek` (agent CLI) — different harness/ID; haiku had no standard baseline at all.
- **fable's 60% Python** was 16 genuine-but-spurious API refusals counted as capability fails.
- **glm-5.2's low score** was reasoning-token accounting + un-logged code (see [m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).

None were random — they're all *comparing things that aren't comparable*, then publishing the result as a headline.

## Principle
**A number is only comparable to another number if it was measured the same way, on the same benchmarks.** The display must enforce this — not the reader, and not a per-run manual audit.

## Rules (enforced in code, not convention)
1. **Coverage is first-class.** Every model in the ratings block carries `benchmarks` (distinct count) + the block carries `maxCoverage`. ✅ done.
2. **Coverage annotation, not hiding.** Every model stays **visible** in the leaderboard (the local models are the point of the server). Models below the coverage threshold are marked **provisional** — dimmed, italic, no medal, with a coverage badge — so a 6-benchmark ELO is visible but can't be misread as beating a 55-benchmark one. Full ranking (medal, undimmed) is earned as coverage fills in. ✅ done (ELO leaderboard). **Corrected 2026-08-11 (iter-178, measured — this rule read "currently 50%" and that was stale):** there are **two** thresholds, deliberately, in `docs/src/components/BenchmarkDashboard/coverageGate.js` — `RATE_COVERAGE_FRACTION = 0.5` and `ELO_COVERAGE_FRACTION = 0.9`. A pass *rate* degrades gracefully with coverage; a *rating* does not, so ELO demands near-identical sets. Anyone quoting "50%" against the ELO board is quoting this doc, not the code.
3. **Like-for-like deltas.** Any *cross-mode* (standard→agent) or *cross-run* delta is computed over the **intersection** of benchmarks the two cohorts both ran, and only across a **matching (model, harness) identity**. `or-X` (API-standard) vs `opencode-or-X` (agent-CLI) is a *harness* comparison and must be labelled as such, not presented as "uplift". ⏳ remaining.
4. **Label the axis.** Standard vs agent, API vs CLI-harness, per-language vs blended — every comparison states what it holds constant.
5. **Tests are the guardrail.** Distribution/validity invariants are unit-tested so a re-tier or a new cohort can't silently break them (the v0.29.2 re-tier already broke two drift detectors — caught late).

## Work items
- **W1 (done)** — `benchmarks` + `maxCoverage` in `ratings_export.go`; ELO leaderboard marks under-covered models provisional (dimmed, no medal, coverage badge).
- **W2 (CLI ✅ · dashboards mostly ✅)** — `ailang eval-elo` tracks per-model AILANG coverage + flags `provisional` below 50% of `max_coverage` (Cov column + `--json`; 7abbb3218). **Frontend:** a shared `coverageGate.js` helper (`buildCoverage` → `id → {benchmarks, provisional}` + `maxCoverage`, reads the W1 ratings block) is threaded through both dashboards. Gated + **CI-docs-build-verified** (tip 9b4f4f890): `ModelChart` (dimmed bars, ⚠, sorted last — kills the live bogus-#1 where the 3 local qwen at 9/56 ranked as headline), `ModelComparisonTable` (true-coverage badge replaces the run-count heuristic; dimmed rows), `ValueScoreTable` (provisional models get **no medal**, sort last, dimmed), `QualityScatter` (provisional excluded from the Pareto frontier), `ModelDeltaTrend` (provisional models filtered from the delta trend + provider average). `DollarsPerPassTable` is **dead code** (not mounted on any page) — no gating needed. `ComparisonTable`/`BenchmarkChampionsTable`/`OSLocalLeaderboard` were already safe. **All live dashboard rank/compare views are now coverage-gated.** **Verification note:** local full build is blocked by a pre-existing `sidebars.js` quirk (Node-26 rig), so verify via per-file babel compile + coverage-logic-on-real-data + the CI docs build (never trust a background exit code — read the log / CI check).
- **W3 (done ✅)** — `ComputeUplift` (`internal/eval_analysis/uplift.go`): shared-benchmark, matching-identity standard→agent delta; wired into the ratings block as `ratings.uplift`, and **surfaced on the site** via `AgentUpliftTable` on the Agent Harness Explorer page. Verified on v0.29.2 (haiku +51.8%, sonnet +25.4%, luna −32.6%). Mismatched identity (`or-X` vs `opencode-or-X`) is excluded as a harness comparison, not uplift. (5 unit tests.) Gotcha fixed: the unified-publish script preferred a stale repo `bin/ailang` over PATH, silently dropping the uplift block — removed it.
- **W4 (partial ✅)** — done: `ComputeUplift` tests (shared-benchmark/identity/macro-avg/lang-scope) + `eval-elo` coverage-gating test (full/sparse/at-threshold). Remaining: a "no ranked model below threshold" invariant on the dashboard board once W2-frontend lands; keep the tier-distribution detectors in sync with the corpus.
- **W5** — apply the same discipline to the merged local↔cloud board ([m-eval-local-cloud-unify](m-eval-local-cloud-unify.md)) — local only enters full (non-provisional) ranking once its AILANG coverage matches. The rig is now **AILANG-first**: `os-rotation-filler.sh` fills every core+stretch+frontier AILANG benchmark for the current version *first* (default), then auto-hands-off to the cross-language pass; a new release resets coverage so AILANG-first resumes. Completeness = every full-tier bench banked for every local model, OR one full AILANG lap (deadlock-safe against benchmarks a weak model can't pass).

- **W8 (NEW 2026-08-07, P0 — ailang#619) — SPLIT OUT 2026-08-13** — **the OS leaderboard publisher counts harness errors as capability failures.** Moved verbatim to its own scoped doc so it can route independently of W9's disputed direction: **[m-eval-w8-harness-errors-as-capability-failures.md](m-eval-w8-harness-errors-as-capability-failures.md)**.
  - Mark took the pending human decision on 2026-08-13 ("whether W8 may route on its own, in its own scoped doc") — **yes**. W8 was untouched by both quorum rounds' objections; only W9's direction is disputed, and it was the bundling that blocked W8.
  - That doc carries the full V1-V11 reality-check, AC-W8.1 through AC-W8.6, the `summary.json` Conflict Surface table, and the verbatim R1/R2 `proposed_fix` text. **Do not maintain a second copy here.**
  - Added at the split (V12): W8's target rows are concentrated in agent/motoko rotation data — 147 invalid rows tree-wide, 80 in `motoko_full_core_matrix` alone — so W8 delivers on its own for the OS board. On *standard-mode* baselines only 4 of 877 rows carry `validity.valid=false`, which is why it pairs with the producer-side [m-eval-failure-attribution.md](m-eval-failure-attribution.md).

- **W9 (NEW 2026-08-11, iter-178 — raised by quorum R1, `gpt5-6-sol`, and MEASURED before adoption)** — **the coverage gate compares COUNTS, never benchmark-set IDENTITY, so two models on disjoint sets of equal size are ranked as comparable.** This is the doc's own headline defect surviving inside the fix for it: Rule 1 makes "coverage" a scalar, and a scalar cannot express "measured on the same benchmarks". Measured at base `5f471b2b7`:

  | # | Claim | Command | Result |
  |---|---|---|---|
  | V12 | The gate is count-based | `coverageGate.js` | `buildCoverage` builds `id → n` from `m.benchmarks` (an integer); `isProvisional`/`isProvisionalForElo` compare that integer to a fraction of `maxCoverage`. **No benchmark ID is read anywhere in the file**: `grep -ci 'benchmark_id\|benchmarkId\|benchIds'` = **0**, controls `benchmarks` = **6** and `maxCoverage` = **13** firing in the same call |
  | V13 | The producer HAS the set but exports only its size | `internal/eval_analysis/ratings_export.go:107-117` | `maxCoverage = len(bs)`; `"benchmarks": len(modelBenches[id])` — `modelBenches[id]` **is** the set, and only `len` crosses the boundary |
  | V14 | The reviewer's threshold detail was quoting this doc, not the code | `coverageGate.js:26-27` | `RATE_COVERAGE_FRACTION = 0.5`, `ELO_COVERAGE_FRACTION = 0.9` — so "promotes at 50% into the full ELO ranking" is **FALSE** for ELO and TRUE for rates. The objection's *core* (count ≠ set) survives both thresholds |

  **W9 acceptance criteria — the round-1 sketch was REPLACED at round 2 by `gpt5-6-sol`'s own
  text, adopted VERBATIM.** The reviewer's round-2 objection is sound and is the reason the first
  draft could not stand: a *pairwise* shared-intersection rating *"can produce different samples
  and ratings for every model pair, so it cannot support one deterministic, transitive
  leaderboard"* — i.e. the round-1 sketch would have traded an invalid comparison for a
  non-transitive ordering, which is the same class of bug one layer down. Adopted:

  > *"Each ratings export includes sorted completed benchmark IDs and a deterministic required-set
  > ID computed from the benchmark corpus/version and tier policy. Full ELO ranking is partitioned
  > by exact required-set ID, and a model is medal-eligible only when its completed set equals
  > that cohort's required set. Models with missing, extra, or differently composed sets remain
  > visible and provisional but are not ordered against the full cohort. Count thresholds are
  > annotations only. Optional intersection ratings are separate, explicitly labelled artifacts
  > for one fixed, named intersection shared by every model shown; pair-specific intersections
  > never contribute to a global leaderboard. Corpus or re-tier changes create a new required-set
  > ID and cannot silently reuse the prior cohort."*

  Plus, per the same objection: verification rows locating the existing ELO cohort/grouping code
  and an inventory of whether it can enforce exact-set cohorts; and tests for equal-count disjoint
  sets, same-count different-composition sets, subsets, extras, release/re-tier changes, stable
  set-ID generation, and the absence of cross-cohort ordering. **Explicit no-data behaviour is
  required rather than silently retaining the count-based gate.**

  ⚠ **W9 IS PARKED FOR HUMAN RATIFICATION, NOT ROUTED.** The controller adopted the reviewer's
  text but has NOT verified the ELO cohort/grouping inventory it calls for, and partitioning the
  public leaderboard by required-set ID is a visible product change — see the DECISIONS ask on the
  bookkeeping issue.

  **Scope:** NOT part of the W8 sprint (W8 is one guard at one aggregation point in
  `eval_harness`; W9 is a schema + display change in `eval_analysis` + the dashboard). Queued
  separately so W8's P0 is not held behind it. **AC1 below is consequently NOT satisfied today**
  — see the note there.

## Acceptance criteria
1. No dashboard/leaderboard/CLI ranks a model against others on a materially different benchmark set without a coverage annotation + gate. ⚠ **NOT SATISFIED as of 2026-08-11 (iter-178)** — the shipped gate annotates on benchmark *count*, so "materially different benchmark set" is not actually tested; equal-count disjoint sets pass it. Closing this AC is **W9**, not W1/W2. Recorded rather than quietly claimed, because a doc that marks its own headline AC done is how the next reader stops checking.
2. Every "uplift"/delta on the site is over shared benchmarks + matching identity, or is explicitly labelled as a harness/mode comparison.
3. Unit tests enforce the gating + distribution invariants; they fail loudly on a new cohort or a re-tier.

## Out of scope
- Changing the ELO math itself (it's fine within a cohort; the issue is *cross-cohort* display).
- The per-model correctness fixes tracked separately ([m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).
