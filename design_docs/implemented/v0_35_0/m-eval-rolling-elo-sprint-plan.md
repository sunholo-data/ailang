# Sprint Plan: M-EVAL-ROLLING-ELO

**Design doc**: [m-eval-rolling-elo.md](m-eval-rolling-elo.md) (design freeze D1–D3 RATIFIED by Mark 2026-08-27, incl. amended 5-model bridge panel)
**Sprint ID**: M-EVAL-ROLLING-ELO
**Duration**: ~6 working days (5 milestones; M5 parallelizable)
**Risk**: medium — touches shared eval machinery; every override enumerated in the design doc's Conflict Surface with receipts
**Total LOC estimate**: ~800 Go + tests, ~120 shell/skill, ~210 frontend

## Non-negotiables (from Mark's steer)

1. Each milestone ends by EXECUTING its VERIFY block and recording the result in the design doc before the next milestone begins. A milestone without a recorded VERIFY result is not done.
2. Simplicity: no new parallel paths. M1 *removes* a fitting path (the dual-fit split); M3 *removes* a spend pattern (default full baselines).
3. `Band()` semantics and the `latest.json.ratings` shape are frozen (additive fields only) — consumers enumerated in design doc V20/V21.

## Milestones

### M1 — Anchored, single-path fit (~2 days, ~300 LOC)

Tasks:
1. `FitFromTrialsAnchored(trials, fixedBench, fixedModels)` in `internal/eval_harness/ratings.go` (~60 LOC): fixed entities keep their given rating (delta discarded); `FitFromTrials` delegates with `(nil, nil)` — behavior-preserving, proven by existing tests passing unchanged.
2. Generate `anchor_v1.json` from the clean corpus (v0.32.0 baseline standard/ + banked runs since): discriminating-band benchmark difficulties + provenance header; committed + `go:embed`.
3. Route both fitters through the anchored path with the same anchor: `cmd/ailang/eval_elo.go` (read-only projection) and `tools/eval-elo/main.go` (persist path).
4. Unify provisional threshold at 90%: one exported constant, emitted into the ratings block; `coverageGate.js` reads it with 0.9 fallback.
5. Fix stale seeding messages (`eval_saturation.go:87`, `eval_confidence.go:46`).
6. Tests (~120 LOC): anchor immobility; pool-composition invariance (THE regression test — same model rows ± easy-benchmark padding ⇒ ratings within ±25; this pins the 2763-vs-1995 artifact); unseen-entity flat-entry preserved (existing tests keep passing).

**VERIFY (mechanical)**: `go test ./internal/eval_harness/ ./internal/eval_analysis/ -count=1` green; pool-composition test demonstrably fails against the un-anchored fit (mutation check); refit current corpus anchored and record the band-change diff table in the design doc; `make ci` green.

### M2 — trial_history writer + agent-mode automation (~1.5 days, ~175 LOC)

Tasks:
1. `AppendTrialHistory` in `internal/observatory/ratings.go` (~60 LOC): `INSERT OR IGNORE` (PK `trial_id` = banked trial-file identity), columns per V17b DDL; `compiler_version` = `releaseTag()` of the banked version dir; `prompt_version` = row's own `PromptVersion` (V19).
2. Wire into `tools/eval-elo` persist path (~40 LOC): every persist appends its trials with before/after ratings.
3. Agent-mode automation: rotation/nightly persist hook (`tools/launchd/` +~15 LOC) so agent `model_ratings` stops going stale (currently 3 models @ 2026-07-31).
4. Tests (~60 LOC): idempotent re-persist (count unchanged); version stamping correct for a `-dirty` dev build (releaseTag normalization).

**VERIFY (mechanical)**: real persist run → `SELECT COUNT(*), compiler_version FROM trial_history GROUP BY compiler_version` shows correctly-stamped rows; second identical persist → count unchanged; agent `model_ratings.last_updated` advances after one rotation cycle (or a manually-triggered filler pass).

### M3 — Linking-run protocol (~1.5 days, ~100 LOC Go + ~60 shell)

Tasks:
1. Commit `direction_panel_v1` (fixed ~20-25 benchmark subset of the anchor panel, spanning tiers; exclude `graderFlag` artifacts).
2. Post-release skill: linking-run mode = direction panel × ratified bridge (`claude-sonnet-5`, `gpt5-6-terra`, `gemini-3-7-flash`, `or-glm-5-3-flash`, `or-deepseek-v4-flash`) × ailang+python × N=1, `--bank-by-version --skip-existing`; budget guard ≤ $25; missing panel cells fail LOUDLY (no partial index). Full baseline demoted to quarterly re-anchor.
3. Direction fit runs post-persist: bridge strengths fixed, panel difficulties refit from this release's trials only; index + input strengths stamped (consumed by M4).
4. Codify the new-model gate protocol (measured 2026-08-27: $3-15/model) as a model-manager skill reference section.

**VERIFY (mechanical + real)**: execute the linking run for the actual next release (or a dry-tag rehearsal on current dev): measured cost ≤ $25 recorded in the design doc; DB + projection updated; bridge ratings move < 50 pts vs prior fit; a deliberately-omitted panel cell makes the run fail loudly (negative test).

### M4 — The series becomes visible (~2 days, ~80 LOC Go + ~210 frontend)

Tasks:
1. `HistoryEntry` additive `ratings` summary: per-model anchored ELO + direction index (overall + per tier) + recorded bridge input strengths. JSON-shape test proves old entries round-trip.
2. One site chart (ELO-over-versions + direction index) inside `BenchmarkDashboard` trends — per the m-eval-os-version-trend-redesign rules verbatim: runtime fetch only, fold into proven component, change ONCE, Node 20 build.
3. Provenance stamps (version + timestamp) on the 5 pages missing them; extend the existing `benchmarkFetchWithSource` badge to the other surfaces (reuse; no new mechanism).

**VERIFY (mechanical + human)**: data-logic unit tests green; Docusaurus build green on Node 20; headless post-hydration `--dump-dom` shows chart rows; **Mark confirms visuals** (repo norm — agent never signs off frontend alone).

### M5 — Hygiene sweep (~1 day, parallelizable any time after M1)

Tasks: axiom scorecard staleness (wire into publish or banner); `BenchmarkMini` hardcoded "46 benchmarks" → data-driven; Explorer stale baseUrl link; delete dead `docs/docs/static/benchmarks/latest.json` + 4 orphaned components (git-history check first, per coding standards); replace hand-written v0.3.5 tables in `guides/benchmarking.md` with links; stamp `model-capability-threshold.md` as dated snapshot.

**VERIFY (mechanical)**: per deleted file, a grep proving zero consumers, recorded in the commit message; `make ci` + docs build green; link-check on touched pages.

## Dependencies

M1 → M2 → M3 → M4 (M4's chart needs ≥1 stamped release entry from M3; its provenance/badge tasks can start after M1). M5 independent, parallel.

## Velocity basis

Recent 7 days show the loop landing ~1 substantive milestone/day alongside docs traffic (prompt-freeze M1 `ed5600da6`, corpus M3, fmt pins). 5 milestones ≈ 6 working days with the 20-30% buffer already applied to per-milestone estimates.

## Success metrics (sprint-level) — status 2026-08-27

- [x] Pool-composition regression test in CI, failing against the old fit (mutation-checked)
      — anchored drift 31.2 vs unanchored 311.7 ELO (10x); the control fails if the scenario
      ever stops discriminating.
- [x] `trial_history` > 0 rows, idempotent, version-stamped — 1,711 rows stamped v0.32.0;
      identical re-persist leaves the count unchanged (PK constraint, not convention).
      ⚠️ *agent ratings auto-refresh*: code landed (filler step 8a) but NOT ACTIVE on the rig
      until its plist/script install cycle runs — mechanism proven by a manual run.
- [~] One real release measured ≤ $25 with the index stamped — protocol proven end to end
      (both refusal gates + a positive index of 1551.1 over 86 trials) and the v0.35.0 dry run
      costs **$2.88** against the $25 cap, but it has not yet run against an actual release tag.
- [~] ELO trend + provenance visible — implemented and CI `docs-build` green; **Mark's visual
      confirmation still outstanding by design** (no agent may sign this off).
- [x] Zero non-additive changes to `latest.json.ratings`; V20/V21 consumers untouched
      — additive fields only; legacy-history round-trip test proves the 47 published entries
      serialize with an identical key set.

Legend: [x] done · [~] done-with-a-named-gap (see the design doc's VERIFY blocks for detail).
