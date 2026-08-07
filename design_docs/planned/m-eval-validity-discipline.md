# M-EVAL-VALIDITY-DISCIPLINE: like-for-like, coverage-gated eval comparisons everywhere

**Status**: IN PROGRESS — coverage gating + per-model coverage landed 2026-07-11 (ratings block + ELO leaderboard). Remaining: uplift/delta like-for-like, cross-mode/harness labelling, tests, and **W8 (P0, added 2026-08-07): the publisher scores `validity.valid=false` harness errors as capability failures — a live ~4× understatement on the v0.33.0 OS board.**
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
2. **Coverage annotation, not hiding.** Every model stays **visible** in the leaderboard (the local models are the point of the server). Models below the coverage threshold (currently 50% of `maxCoverage`) are marked **provisional** — dimmed, italic, no medal, with a coverage badge — so a 6-benchmark ELO is visible but can't be misread as beating a 55-benchmark one. Full ranking (medal, undimmed) is earned as coverage fills in. ✅ done (ELO leaderboard).
3. **Like-for-like deltas.** Any *cross-mode* (standard→agent) or *cross-run* delta is computed over the **intersection** of benchmarks the two cohorts both ran, and only across a **matching (model, harness) identity**. `or-X` (API-standard) vs `opencode-or-X` (agent-CLI) is a *harness* comparison and must be labelled as such, not presented as "uplift". ⏳ remaining.
4. **Label the axis.** Standard vs agent, API vs CLI-harness, per-language vs blended — every comparison states what it holds constant.
5. **Tests are the guardrail.** Distribution/validity invariants are unit-tested so a re-tier or a new cohort can't silently break them (the v0.29.2 re-tier already broke two drift detectors — caught late).

## Work items
- **W1 (done)** — `benchmarks` + `maxCoverage` in `ratings_export.go`; ELO leaderboard marks under-covered models provisional (dimmed, no medal, coverage badge).
- **W2 (CLI ✅ · dashboards mostly ✅)** — `ailang eval-elo` tracks per-model AILANG coverage + flags `provisional` below 50% of `max_coverage` (Cov column + `--json`; 7abbb3218). **Frontend:** a shared `coverageGate.js` helper (`buildCoverage` → `id → {benchmarks, provisional}` + `maxCoverage`, reads the W1 ratings block) is threaded through both dashboards. Gated + **CI-docs-build-verified** (tip 9b4f4f890): `ModelChart` (dimmed bars, ⚠, sorted last — kills the live bogus-#1 where the 3 local qwen at 9/56 ranked as headline), `ModelComparisonTable` (true-coverage badge replaces the run-count heuristic; dimmed rows), `ValueScoreTable` (provisional models get **no medal**, sort last, dimmed), `QualityScatter` (provisional excluded from the Pareto frontier), `ModelDeltaTrend` (provisional models filtered from the delta trend + provider average). `DollarsPerPassTable` is **dead code** (not mounted on any page) — no gating needed. `ComparisonTable`/`BenchmarkChampionsTable`/`OSLocalLeaderboard` were already safe. **All live dashboard rank/compare views are now coverage-gated.** **Verification note:** local full build is blocked by a pre-existing `sidebars.js` quirk (Node-26 rig), so verify via per-file babel compile + coverage-logic-on-real-data + the CI docs build (never trust a background exit code — read the log / CI check).
- **W3 (done ✅)** — `ComputeUplift` (`internal/eval_analysis/uplift.go`): shared-benchmark, matching-identity standard→agent delta; wired into the ratings block as `ratings.uplift`, and **surfaced on the site** via `AgentUpliftTable` on the Agent Harness Explorer page. Verified on v0.29.2 (haiku +51.8%, sonnet +25.4%, luna −32.6%). Mismatched identity (`or-X` vs `opencode-or-X`) is excluded as a harness comparison, not uplift. (5 unit tests.) Gotcha fixed: the unified-publish script preferred a stale repo `bin/ailang` over PATH, silently dropping the uplift block — removed it.
- **W4 (partial ✅)** — done: `ComputeUplift` tests (shared-benchmark/identity/macro-avg/lang-scope) + `eval-elo` coverage-gating test (full/sparse/at-threshold). Remaining: a "no ranked model below threshold" invariant on the dashboard board once W2-frontend lands; keep the tier-distribution detectors in sync with the corpus.
- **W5** — apply the same discipline to the merged local↔cloud board ([m-eval-local-cloud-unify](m-eval-local-cloud-unify.md)) — local only enters full (non-provisional) ranking once its AILANG coverage matches. The rig is now **AILANG-first**: `os-rotation-filler.sh` fills every core+stretch+frontier AILANG benchmark for the current version *first* (default), then auto-hands-off to the cross-language pass; a new release resets coverage so AILANG-first resumes. Completeness = every full-tier bench banked for every local model, OR one full AILANG lap (deadlock-safe against benchmarks a weak model can't pass).

- **W8 (NEW 2026-08-07, P0 — ailang#619)** — **the OS leaderboard publisher counts harness errors as capability failures.** Same class as the fable "60% Python = 16 API refusals counted as capability fails" face above, but in the *publisher* rather than the display. `cmd/ailang/eval_publish.go` computes `PassRate = Passed / Trials` over **every banked row** and never reads `validity` — even though the harness already writes `validity: {valid: false, reason: "harness_error"}` on exactly these rows (`internal/eval_harness/validity_backstop*`). Concretely on 2026-08-07: 30 `api_error` rows from the ollama 300s-timeout cascade ([m-ollama-v1-streaming-idle-timeout](m-ollama-v1-streaming-idle-timeout.md), ailang#618) put motoko-local's published v0.33.0 **frontier at exactly `3/22 = 0.13636363636363635`** — bit-for-bit the published value — where **17 of the 22 were harness timeouts**. True figure ≈ 60% (n=5): a ~4× understatement, live on the dashboard and synced to the bucket. Frozen wrong, too, because `--skip-existing` treats a banked `api_error` as done, so those combos never re-run for that version.
  - **Fix:** exclude `validity.valid == false` rows from BOTH numerator and denominator at publish time, and surface the excluded count (`n=5 (17 invalid excluded)`) rather than silently shrinking `n` — a silent drop trades one invisible bug for another. Per Critical Principle 2, a harness error must never be scored as a capability failure.
  - **Also:** `--skip-existing` should not treat an invalid row as satisfying a combo (otherwise every harness outage permanently poisons that version's bank). Deleting the rows is the current manual workaround — done 2026-08-07, 30 rows removed after backup.
  - **Tests:** a banked `validity.valid=false` row must not move a published pass rate; the excluded-count must appear.

## Acceptance criteria
1. No dashboard/leaderboard/CLI ranks a model against others on a materially different benchmark set without a coverage annotation + gate.
2. Every "uplift"/delta on the site is over shared benchmarks + matching identity, or is explicitly labelled as a harness/mode comparison.
3. Unit tests enforce the gating + distribution invariants; they fail loudly on a new cohort or a re-tier.

## Out of scope
- Changing the ELO math itself (it's fine within a cohort; the issue is *cross-cohort* display).
- The per-model correctness fixes tracked separately ([m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).
