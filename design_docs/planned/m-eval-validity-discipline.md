# M-EVAL-VALIDITY-DISCIPLINE: like-for-like, coverage-gated eval comparisons everywhere

**Status**: IN PROGRESS — coverage gating + per-model coverage landed 2026-07-11 (ratings block + ELO leaderboard). Remaining: uplift/delta like-for-like, cross-mode/harness labelling, tests.
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
2. **Coverage gating.** A model is only *ranked* if its coverage ≥ threshold (currently 50% of `maxCoverage`). Under-covered models render in a separate "not ranked — insufficient coverage" section with their count, never in the headline ranking. ✅ done (ELO leaderboard).
3. **Like-for-like deltas.** Any *cross-mode* (standard→agent) or *cross-run* delta is computed over the **intersection** of benchmarks the two cohorts both ran, and only across a **matching (model, harness) identity**. `or-X` (API-standard) vs `opencode-or-X` (agent-CLI) is a *harness* comparison and must be labelled as such, not presented as "uplift". ⏳ remaining.
4. **Label the axis.** Standard vs agent, API vs CLI-harness, per-language vs blended — every comparison states what it holds constant.
5. **Tests are the guardrail.** Distribution/validity invariants are unit-tested so a re-tier or a new cohort can't silently break them (the v0.29.2 re-tier already broke two drift detectors — caught late).

## Work items
- **W1 (done)** — `benchmarks` + `maxCoverage` in `ratings_export.go`; ELO leaderboard gates under-covered models into a separate section.
- **W2** — coverage-gate the other consumers that rank/compare: `eval-elo` CLI, the Model Leaderboard / gap-trend / dashboard tables (audit each — CLAUDE.md §3 systemic-fix rule).
- **W3** — like-for-like uplift: a shared-benchmark, matching-identity delta computation (backend helper) reused by any "vs standard"/"vs previous" view; harness differences explicitly labelled.
- **W4** — tests: coverage-gating unit test; a "no ranked model below coverage threshold" invariant; keep the tier-distribution detectors in sync with the corpus.
- **W5** — apply the same discipline to the merged local↔cloud board ([m-eval-local-cloud-unify](m-eval-local-cloud-unify.md)) — local only enters the ranking once `OS_FILLER_AILANG_FULL` grows its coverage.

## Acceptance criteria
1. No dashboard/leaderboard/CLI ranks a model against others on a materially different benchmark set without a coverage annotation + gate.
2. Every "uplift"/delta on the site is over shared benchmarks + matching identity, or is explicitly labelled as a harness/mode comparison.
3. Unit tests enforce the gating + distribution invariants; they fail loudly on a new cohort or a re-tier.

## Out of scope
- Changing the ELO math itself (it's fine within a cohort; the issue is *cross-cohort* display).
- The per-model correctness fixes tracked separately ([m-eval-reasoning-model-fairness](m-eval-reasoning-model-fairness.md)).
