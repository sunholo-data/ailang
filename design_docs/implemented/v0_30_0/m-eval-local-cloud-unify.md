# M-EVAL-LOCAL-CLOUD-UNIFY: local models in the same tables + rolling trends as cloud

**Status**: PLANNED — surfaced 2026-07-11 investigating why the local-rig pages looked frozen and couldn't be compared to cloud.
**Target**: v0.30.x (eval infrastructure)
**Priority**: P2 (we can't answer "does an AILANG release help local models vs cloud" — the whole point of the on-device roster)
**Estimated**: 2–3 days (+ GPU wall-clock for the coverage runs)
**Dependencies**: none in AILANG core. Touches the rotation scripts (`tools/launchd/os-rotation-filler.sh`, `tools/os-release-snapshot.sh`), `eval-publish`/`eval-report`, and the dashboard components.
**Author**: Claude Opus 4.8 (requested by Mark, 2026-07-11)

---

## Problem

The on-device (GPU) models live on a **separate data island** and can't be compared with cloud models:

1. **Separate data source.** Local results publish to `/benchmarks/os/latest.json` + `os/history.json`; cloud results live in `/benchmarks/latest.json`. The ELO leaderboard, gap-trend, per-model-trend and dashboard all read the *cloud* file, so local models never appear next to cloud.
2. **Nowhere near frontier.** The rotation pool (`os-rotation-filler.sh`) is filtered to **4-language + `grade_entrypoint`** benchmarks — **41/91**, of which only **2/16 frontier** (docx/markdown reimplements). gauntlet_10, quine, legal_obligation_engine, the contract set, etc. are excluded because they're AILANG/Python-focused, not 4-language. So local models have **never** been measured on the hard benchmarks.
3. **Round-robin lag + `--skip-existing`.** 3 benchmarks/cycle, skip-existing → a version's coverage fills slowly and never re-measures optimization within a release (scores look frozen).
4. **Publish bugs (fixed 2026-07-11).** The version-trend `history.json` was frozen at v0.25.0 (never appended for v0.26–v0.29); `os-release-snapshot.sh` read the *mixed multi-version root*, cross-contaminating entries. Backfilled + fixed to read the per-version subdir.

Net: we cannot see how local models track cloud across releases — which is the on-device roster's entire justification.

## Goals
1. Local models measured on the **same benchmarks as cloud, up to frontier** (at least in AILANG).
2. Local + cloud appear **in the same tables** (ELO leaderboard, gap-trend, dashboard) — no separate island.
3. A **rolling recent-trend** view: local vs cloud over recent runs, not just per-release snapshots.

## Design

### D1 — Bank local runs into the shared baseline (the key move)
Point the on-device agent runs at `eval_results/baselines/<version>/agent` (the same dir cloud uses) instead of `eval_results/rotation/os-rolling`. Then `eval-report … --format=json` folds local rows into `latest.json` — models leaderboard, `byLang` ELO, gap-trend and dashboard render them **automatically, no component changes** (the tables already draw whatever's in the data). `getProvider` already buckets qwen/gemma/deepseek/glm as `other` (open-source), so the provider-grouped charts group them sensibly.

### D2 — Full-tier AILANG coverage for local models
Add an AILANG-only full-tier mode to the rotation: run local models over `core,stretch,frontier` in **AILANG** (drop the 4-language filter for this pass — the 4-lang cross-language view stays a separate, optional artifact). Keep `--skip-existing --bank-by-version` for cost, but ensure the pool includes the frontier benchmarks. The two mega-reimplements (docx/markdown, ~50 min each) run on a slower cadence so they don't starve the round-robin.

### D3 — Rolling recent-trend view (local vs cloud)
The existing gap-trend/per-model-trend are **per-release**. Add a **rolling window** view (last N runs by date, across models) so intra-release movement + local-vs-cloud tracking is visible without waiting for a release. Frame it as "local vs cloud" (the `other` provider group vs the frontier-lab groups) reusing the provider grouping already built.

### D4 — Keep the OS cross-language page for JS/Go
The `os/latest.json` + `OSLocalLeaderboard` view (AILANG/Python/JS/Go per harness) is still useful for the *cross-language* story that the cloud baseline doesn't cover. Keep it, fed by the 4-language subset — but it's no longer the *only* place local numbers live.

## Risks / unknowns
- **GPU wall-clock.** Local agent runs are serial on one GPU with model reloads; a full core+stretch+frontier × 3 models AILANG pass is many hours (docx/markdown alone dominate). Mitigate: exclude the mega-reimplements from the fast pass; run them on a slow cadence.
- **Rig contention.** The shared-baseline local run competes with the continuous rotation for the single GPU (rig lock serializes). Decide whether the rotation *is* the baseline-filler or a separate loop.
- **Version banking correctness.** The rig binary must be on the released version so `--bank-by-version` attributes correctly (past os-rolling staleness came from this).

## Acceptance criteria
1. qwen3.6 + gemma4 appear in the **ELO leaderboard, gap-trend, and dashboard** alongside cloud, covering `core,stretch,frontier` (AILANG).
2. `latest.json` contains local model rows (agent mode) banked from the shared baseline.
3. A rolling recent-trend view shows local vs cloud tracking over recent runs.
4. `os/history.json` continues to update per release (regression test the `os-release-snapshot.sh` per-version fix).

## Out of scope
- Cross-language (JS/Go) coverage for the *hard* tiers on local models — keep JS/Go on the existing 4-language subset.
- New GPU hardware / parallelism — this is a single-GPU rig.
- Cloud-model changes — cloud already lands in the shared baseline.

## Progress (2026-07-11)
- ✅ Fixed the version-trend freeze (`os/history.json` backfilled v0.26–v0.29.2; `os-release-snapshot.sh` reads per-version subdir).
- 🏃 Proving D1+D2: a first `opencode-qwen3-6-35b-a3b-mxfp8` run over 54 core+stretch+frontier benchmarks (AILANG, minus docx/markdown) is banking into `eval_results/baselines/v0.29.2` — when `latest.json` regenerates, qwen3.6 should appear in the shared cloud tables.
