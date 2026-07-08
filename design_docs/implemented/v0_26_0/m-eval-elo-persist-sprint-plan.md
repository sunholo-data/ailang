# Sprint Plan — M-EVAL-ELO-PERSIST (M-EVAL-RATING-EFFICIENCY part 2)

**Goal:** Persist ELO ratings and make the rig spend compute where it buys information — `ratings.db`, `eval-suite --benchmarks-by-confidence`, and a tier-saturation report — then emit ELO into `latest.json` to unblock the dashboard redesign.
**Design docs:** [m-eval-rating-efficiency.md](m-eval-rating-efficiency.md) (part 2), chains to [m-eval-dashboard-redesign.md](../m-eval-dashboard-redesign.md).
**Status going in:** part 1 done — `tools/eval-elo` has the converged ELO math; this sprint promotes it to a tested package + store + CLI surfaces.
**Estimate:** 3–4 days (~850 LOC + tests). Risk: medium (touches `cmd/ailang` flag wiring + observatory schema; no parser/runtime).
**Ships:** v0.26.0 (alongside the regrade + ELO part 1 already on dev).

## Velocity basis
Today shipped 3 harness features (CompareOutput normalize, eval-regrade, eval-elo) + 4 design docs. Harness-only work, no parser/type risk → conservative 3–4 day estimate holds with buffer.

## Milestones

### M1 — `ratings` package + backfill (≈220 LOC + tests)
Promote the ELO update from `tools/eval-elo` into `internal/eval_harness/ratings.go` as a tested package: `UpdateTrial(modelRating, benchRating, pass, k) (newM, newB)`, `FitFromTrials([]Trial) (modelRatings, benchRatings)` (converged static fit), and difficulty banding. `tools/eval-elo` becomes a thin caller. Backfill helper replays a baseline dir's (regraded) trials.
- **Acceptance:** unit tests over synthetic trial sequences (strong-beats-easy = small Δ; weak-beats-hard = large Δ; symmetric coin-flip); `tools/eval-elo` output unchanged on v0.25.0; `go test ./internal/eval_harness/` green.

### M2 — observatory `ratings` store + v16 migration (≈260 LOC + tests)
`internal/observatory/ratings.go`: `benchmark_ratings`, `model_ratings`, `trial_history` tables (schema from the design doc), upsert-on-trial + load. Versioned **v16 migration** (mirrors the v15 eval_baselines pattern) + `ValidateSchema` entry. Seed by replaying the regraded v0.25.0 baseline.
- **Acceptance:** `TestMigrateWithVersion_V16` backfills on a v15 DB; round-trip store/load; seeding v0.25.0 yields fable model-rating #1; regression test asserts no schema drift.

### M3 — `eval-suite --benchmarks-by-confidence` (≈200 LOC + tests)
New selection mode in `cmd/ailang/eval_suite.go`: load ratings, rank benchmarks by information-gain of one more trial, take top-N (`--max-benchmarks`), skip saturated (≥95%/≤5% over ≥5 trials at current compiler SHA) and unchanged-version cells. `--dry-run` lists the chosen set.
- **Acceptance:** on the v0.25.0 ratings, `--dry-run --max-benchmarks 5` surfaces the discriminating set (contract_*, run_length_encode, explicit_dataflow_ssa) and excludes the 9 Trivial; flag-parsing tests.

### M4 — saturation report + `latest.json` ELO emit (≈170 LOC)
`ailang eval-trend tier-saturation` renders the saturation report (saturated / discriminating per mode + recommendation). `eval-publish`/`update_dashboard` emit per-benchmark `elo`/`eloBand`/`saturated` and per-model `elo` into `latest.json` — the bridge that unblocks M-EVAL-DASHBOARD-REDESIGN.
- **Acceptance:** report runs on v0.25.0 and flags the saturated set; regenerated `latest.json` validates with the new fields present; `graderFlag` set for `contract_sorted_merge`/`decision_block_capture`.

## Status (2026-06-12)
- **M1 ✅ DONE** — `ratings` package + tests; `tools/eval-elo` refactored onto it.
- **M2 ✅ DONE** — observatory store + v16 migration + `--persist`; later **mode-separated** (standard/agent in the PK).
- **M3 ✅ DONE** — `eval-suite --benchmarks-by-confidence` selects the discriminating set (verified: picks the contract_*/dataflow benchmarks, excludes the 9 Trivial). Unit + dry-run tested.
- **M4 PARTIAL** — `eval-trend tier-saturation` ✅ DONE (per-mode report: standard 24% / agent 68% saturated). The **`latest.json` ELO emit is moved to the M-EVAL-DASHBOARD-REDESIGN sprint** (phase 1 of that doc is exactly "emit ELO/regrade/saturation into `latest.json`") so the schema lands with the React consumers, not split across two sprints.

## Out of scope (follow-on)
- The React/website work itself = **M-EVAL-DASHBOARD-REDESIGN** sprint (starts once M4 emits the fields).
- Cross-model "add a model in 8 trials" live workflow — useful but not blocking; fast-follow after M3.

## Success metrics
- `go test ./internal/eval_harness/ ./internal/observatory/ ./cmd/ailang/` green; `go vet` clean.
- `ratings.db` seeded from regraded v0.25.0; fable #1 on model-rating.
- `latest.json` carries ELO/difficulty fields (dashboard redesign unblocked).
- Changelog entry under `[Unreleased]`.
