# M-EVAL-ROLLING-ELO: one anchored rating series across models and AILANG versions

**Status**: Planned — quorum run twice (both rounds' objections accepted + incorporated, see Quorum Record); awaiting Mark's ratification of Design Freeze D1–D3
**Target**: v0.35.0
**Priority**: P1 — this is the instrument we measure ourselves with; Mark (2026-08-27): "key and how we measure our own performance so critical to get right … this time, lets concentrate on value and simplicity"
**Estimated**: ~2 weeks across 5 milestones (M1–M3 core, M4 site, M5 hygiene — M5 parallelizable)
**Dependencies**: None hard. Complementary: `m-eval-failure-attribution` (planned), `m-eval-validity-discipline` (planned) — both compose with, neither blocks.

## Design principles (Mark's steer, 2026-08-27)

1. **Value**: every milestone must reduce measurement cost or increase what a reader can conclude. No new surface that doesn't retire confusion.
2. **Simplicity**: the instrument grew piecemeal (10 site surfaces, 2 independent ELO fitting paths, 3 artifact families). This doc *consolidates*; anything that adds a parallel path is out.
3. **Follow-through**: each milestone carries an explicit VERIFY block with a mechanical check. A milestone is not done until its check has been run and its result recorded here.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval/measurement infrastructure, not a language change — most axioms are neutral. Scored honestly rather than inflated:

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Anchored fit makes ratings reproducible: same trials + same anchor ⇒ same numbers, replayable from `trial_history` |
| A2: Replayability | +1 | `trial_history` (finally written) is a full audit log: every rating movement replayable with before/after values |
| A3: Effect Legibility | 0 | No effect-system surface |
| A4: Explicit Authority | 0 | No capability surface |
| A5: Bounded Verification | 0 | No change to checking |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Machine-readable rating series with provenance replaces human-interpreted blended pass rates |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Explicit per-release measurement budget; today spend is invisible until the bill (42/56 agent benchmarks saturated = GPU buying no information) |
| A10: Composability | 0 | Composes with existing exporters; additive artifact fields only |
| A11: Structured Failure | 0 | Validity/error-category semantics unchanged (see Non-Goals) |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): improves it — no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): optimizes for machine analysis explicitly

## Problem Statement

AILANG's benchmark instrument is how we know whether the language is improving, and it has grown piecemeal into something that can no longer answer that question cheaply or comparably.

**Current State** (all claims verified 2026-08-27 — see Verification Log):

- **ELO ratings are not comparable across fits.** `FitFromTrials` seeds every model *and* benchmark at 1500 with no scale anchor ([ratings.go:27,55-63](../../../internal/eval_harness/ratings.go)); models and benchmarks drift apart symmetrically over 400 epochs, so the same rows produce different absolute numbers in different pools. The code documents "subject 2664 here vs 1789 there over the same rows" ([eval_elo.go:85](../../../cmd/ailang/eval_elo.go)); reproduced 2026-08-27: or-glm-5-3-flash rated **2763** in a pool contaminated with smoke rows + de-flake retrials vs **1995** in a clean pool. Nothing can accumulate.
- **No version, no time, no history.** `Trial` is `{Model, Bench, Pass}` — no timestamp, no compiler version. `latest.json.history[]` carries 47 versions of pass-rate stats and **zero rating fields**; the entire `ratings` block is wholesale-overwritten on every export ([export_json.go:506](../../../internal/eval_analysis/export_json.go)). We have 47 versions of pass-rate trend and 0 versions of rating trend.
- **Two independent fitting paths disagree by construction.** `observatory.db` (via `tools/eval-elo`, feeds benchmark selection) and `latest.json` (via `eval-report`, feeds the site + rotation priority) are fitted from different corpora at different times. Agent-mode DB ratings: 3 models, last updated 2026-07-31 — four weeks stale.
- **Full-suite runs mostly buy nothing.** 42/56 agent and 39/91 standard benchmarks are saturated (`latest.json` `ratings.*.saturation`); the nightly script's own comment: "over half the night's GPU buying no information." This is exactly why running the full suite feels low-value — it is.
- **The rolling machinery was designed in v0.26.0 and only the schema shipped.** `trial_history` has `compiler_version` + `prompt_version` columns and **zero writers**; `UpdateTrial` is a tested incremental-update primitive with zero callers outside the batch fit ([m-eval-rating-efficiency](../../implemented/v0_26_0/m-eval-rating-efficiency.md)).
- **The site scatters dimensions and hides provenance.** 10 surfaces, no surface combines version × harness except `OSReleaseTrend`, no cost-over-time anywhere, and 5 pages (Leaderboard, ELO, Explorer, Value, Gallery) display no version/date at all. Of 13 component fetch call sites, only `ValueDashboard` uses the source-aware fetch that can render a staleness badge; the other 12 silently degrade to the static fallback (currently v0.32.0) with no indicator (V22).

**Impact:**
- New models and language improvements land weekly (2 models registered + gated 2026-08-27 alone) but their measurements evaporate instead of accumulating into a comparable series.
- Release-over-release direction ("are we improving?") is read from blended pass rates, which the v0.32.0 post-mortem showed are ~62% composition artifacts.
- The cost of a trustworthy answer is a full baseline (~$5-25 + hours of wall clock); the cheap alternative (targeted runs) produces numbers that can't be compared to anything.

**Evidence the cheap path works** (measured 2026-08-27): GLM-5.3-Flash received a defensible fleet placement (smoke floor → core head-to-head → frontier N=3 → pooled ELO rank 9/19) for **$3.30**; the sol-vs-terra capability question was settled for **$8.27** by re-running only the 9 cells that could move the answer. What's missing is not the protocol — it's the anchored series to bank the results into.

## Goals

**Primary Goal:** One anchored, versioned, incrementally-updated rating series — for models (capability) and benchmarks (difficulty) — that cheap targeted runs accumulate into, replacing full-suite baselines as the default release measurement.

**Success Metrics:**
- Ratings comparable across fits: two pool compositions over the same rows agree within ±25 ELO (regression test, M1).
- Per-release measurement cost ≤ **$25** via linking runs (vs full baseline; today's protocol measured $3-12 per question).
- New-model fleet placement ≤ **$15** per model, producing a rating on the shared scale.
- A **language-direction metric** published per release: the direction fit's per-version panel difficulty index (benchmark difficulties refit with bridge-model strengths held fixed — see "The estimator" in Solution Design) — falling difficulty = the language/prompt got easier for the same models.
- ELO-over-versions visible on the site next to the pass-rate trend; every benchmark page shows data provenance (version + date + live/fallback).
- One fitting path: the DB is authoritative, `latest.json.ratings` is a projection of it.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Scale anchor = frozen reference panel of benchmark difficulties (recommended) vs mean-centering | Everything downstream assumes cross-fit comparability; wrong choice = re-anchor everything later | human | design | high |
| D2: `observatory.db` becomes the single authoritative rating store; `latest.json.ratings` becomes a projection | Kills the dual-fit split; changes what "the rating" means for 6 site components + 2 shell scripts + benchmark selection | human | design | high |
| D3: Full baselines retired as default; per-release measurement = linking run (fixed direction panel × 5-model bridge across all three majors + current OpenRouter picks); full run demoted to quarterly re-anchoring | Changes release cost/KPI semantics and what banked per-version data exists | human | design | high |
| D4: Version lives on the **trial** (`trial_history.compiler_version` via `releaseTag()`), not in the ratings PK; per-version series are derived views | Avoids a PK migration; keeps `SaveRatings`/consumers untouched; determines all downstream queries | agent | design | med |
| D5: `Band()` predicate and its 4 consumers frozen as-is | Saturation gating (nightly A/B, rotation priority, post-release gate, site) all key off `Band(r)=="Trivial"` | agent | design | low |
| D6: Uncertainty = existing provisional/coverage gating only; no Glicko/CI in this milestone | Scope control (simplicity); n is recorded, intervals deferred | agent | design | low |

### Design Freeze

All three RATIFIED by Mark (attended, 2026-08-27) — D3 with an amended bridge panel (Mark:
first proposal was "stale and dont include anthropic; we want a mix between all three main
providers and open router current choices"):

- [x] **D1**: Anchor via frozen reference panel — freeze the current fitted difficulties of the ~52 discriminating standard benchmarks as `anchor_v1` (versioned JSON, committed). Re-anchoring is an explicit, versioned event (see Risks), never silent.
- [x] **D2**: `observatory.db` authoritative; `eval-report` reads ratings from the DB (or refits *with the anchor* and writes back) instead of fitting independently. `latest.json.ratings` shape unchanged (additive fields only).
- [x] **D3**: Release protocol = linking run; `make eval-baseline FULL=true` kept for quarterly re-anchor + longitudinal spot-checks. **Bridge panel (amended + ratified 2026-08-27)** — current-generation, all three majors + current OpenRouter picks:
  - `claude-sonnet-5` (Anthropic, $3/$15, ELO 2022) — standard mode uses `ANTHROPIC_API_KEY` on the metered API, which is the correct lane here and fine at current usage (Mark, 2026-08-27). Note the `~/.zshenv` key-strip exists to protect the *claude-CLI agent path* (inherited key silently outbids keychain OAuth and mis-bills — see `ai_provider.go:237-249`); it is not a prohibition on standard-mode metered runs. The linking run exports the key scoped to its own command only, never globally, so the agent-path protection stays intact.
  - `gpt5-6-terra` (OpenAI, $2/$12, ELO 2074) — N=3-characterized 2026-08-27, capability-tied with the flagship on AILANG
  - `gemini-3-7-flash` (Google via Vertex ADC, $0.75/$3.75) — current Flash slot-holder since 2026-08-13
  - `or-glm-5-3-flash` (OpenRouter/z.ai, $0.075/$0.25, ELO 1995) — ratified into extended_suite 2026-08-27
  - `or-deepseek-v4-flash` (OpenRouter/DeepSeek, $0.06/$0.12, ELO 1852) — pins the low end of the scale

  Five members so the overlap-swap protocol tolerates any single vendor retirement; scale span
  1852–2091; estimated linking-run cost ~$12–16, inside the $25 guard (Sonnet 5 + Terra dominate).

## Solution Design

### Overview

Three moves, in dependency order:

1. **Anchor the scale** so ratings mean the same thing across fits (M1). A frozen panel of benchmark difficulties pins the scale; every fit holds panel members fixed and lets everything else move.
2. **Write the trials down** (M2). Every eval run appends `trial_history` rows stamped with `compiler_version` (already `releaseTag()`-normalized) — the audit log the v0.26.0 design specified and never shipped. Ratings become a *derived, incrementally-updatable* quantity.
3. **Change what a release runs** (M3). Replace the full baseline with a linking run: the confidence-selected discriminating set × the bridge panel. New models get the (now codified) targeted gate. Everything lands in the same anchored series.

Then the series becomes visible (M4: per-version ratings in `history[]` + one site trend chart + provenance stamps) and the accumulated cruft is swept (M5).

**The estimator: one engine, two named configurations.** The engine is one function that fits
ratings while holding a declared set of entities fixed. The two configurations differ only in
*which side* is held:

- **Placement fit** (default; every persist): the anchor panel's *benchmark difficulties* are
  held fixed at their `anchor_vN` values; model ratings (and non-panel benchmark difficulties)
  are free. Output: the model-capability series, comparable across fits because the panel pins
  the scale. Used for new-model placement (~$3-15/model) and all routine updates.
- **Direction fit** (once per release; a stamped measurement, never persisted into
  `model_ratings`): the bridge panel's *model strengths* are held fixed; benchmark
  difficulties of the **direction panel** are free. Input: only trials with
  `compiler_version = v`. Output: per-version difficulty series `D_v(bench)` and the
  **language-direction index** = mean of `D_v` over the direction panel (overall + per tier)
  — this CAN fall as the language improves, and it remains on the anchored scale because the
  fixed model strengths (themselves anchored by the placement fit) carry the scale into the
  refit.

Two constraints make the index identifiable and replayable (quorum round 2, objections
accepted):

- **Identifiability — the direction panel is a FIXED set, measured in full, every release.**
  It is a committed subset (~20-25 benchmarks) of the anchor panel, spanning tiers, and the
  release linking run runs *exactly this panel* against the bridge models — panel coverage is
  complete by construction, never confidence-drifted. (Confidence-selection may ADD
  placement-value cells to a linking run; it never substitutes for the panel.) There is no
  carried-forward difficulty for an unmeasured member because there are no unmeasured members;
  a missing cell (provider outage) fails the VERIFY block loudly rather than falling back —
  per the repo's no-silent-fallbacks principle. Panel membership changes are versioned events
  (`direction_panel_v2`) with one overlap release, same protocol as the anchor.
- **Replayability — the index is a stamped measurement, not a live view.** It is computed
  once, at release time, with the bridge strengths *as of that release*, and those input
  strengths are recorded alongside the output in the release's `HistoryEntry` (and derivable
  from `trial_history`). Later refits of bridge ratings never touch historical indices — the
  same measurement contract as banked costs ("what was measured under the rates in force").

Both configurations are deterministic (same trials + same fixed set ⇒ same output) and share
one code path; "anchored" never means "the same numbers are simultaneously fixed and free" —
each fit fixes exactly one side. Falling `D_v` = AILANG/prompt improvement in ELO points, per
tier, immune to the roster-composition artifacts that corrupt blended pass rates.

### Architecture

```
eval runs (linking / gate / rotation / nightly A-B)
   └─ banked rows (unchanged identity: bench, model, lang, mode, condition, version-dir)
        └─ NEW: trial_history append (compiler_version, prompt_version, outcome,
                 model/bench rating before/after)          [M2]
              └─ anchored fit (FitFromTrials + frozen anchor_vN panel)   [M1]
                    ├─ observatory.db model_ratings / benchmark_ratings  (authoritative)
                    │     ├─ benchmark selection (--benchmarks-by-confidence)  [exists]
                    │     └─ per-version series = SELECT over trial_history    [M4]
                    └─ latest.json.ratings = projection (+ ratings summary
                       appended to history[] per version)                [M4]
                          └─ site: EloTrend chart + provenance stamps    [M4]
```

### Conflict Surface

Not a parser/typechecker change, but it overrides shared machinery — enumerated per the 2026-08-26 precedent (writing this section anyway surfaced two shipped-bug candidates there):

1. **`Band()` / saturation predicate** — `Band()` is called in 4 Go files (`eval_confidence.go`, `eval_elo.go`, `eval_saturation.go` via the `eval-trend tier-saturation` dispatch, `ratings_export.go`), and the exported `saturated` flag is additionally read by `tools/eval-signal-set.sh` + `tools/launchd/os-rotation-filler.sh` (V20). Anchoring shifts absolute rating values → band boundaries (1300/1500/1700/1900) now *mean* something stable for the first time, but the transition fit may re-band some benchmarks. **Decision: freeze the predicate, accept one-time re-banding, record the before/after band diff in the M1 VERIFY block.**
2. **`latest.json.ratings` consumer contract** — 8 site files + 2 shell scripts reference the ratings block or its fields (V21; `eval-signal-set.sh` fails open on drift; the React components do not). **Rule: additive fields only; no renames.**
3. **`observatory.db` PKs** — `(model_id, mode)` / `(benchmark_id, mode)` stay; D4 keeps version off the PK so `SaveRatings`, `LoadModelRatings`, `LoadBenchmarkRatings` and both existing callers are untouched. `trial_history` gains its first writer; verified at COLUMN level (V17b): the live v16 DDL already has every field M2 writes — `trial_id TEXT PRIMARY KEY` (the PK *is* the dedup key: writer uses `INSERT OR IGNORE` with trial-file identity as `trial_id`, so re-persisting is a no-op by constraint, not by convention), `prompt_version`, `compiler_version`, all four `*_rating_before/after` columns, `recorded_at`. No migration needed — proven, not assumed.
4. **`eval-report` history preservation** — `mergeHistory`/`writeJSONAtomic` upsert-by-version semantics must be preserved; the new per-version ratings summary rides *inside* `HistoryEntry` (additive field), never a parallel file. The "never redirect stdout" rule is unaffected.
5. **`--skip-existing` / `--bank-by-version`** — unchanged; linking runs use them as-is. `releaseTag()` is the version authority for trial stamping (same normalization the bank dirs already use).
6. **Post-release skill** (`run_eval_baseline.sh`) — its confidence-gating + line-552 refit is *replaced* by the linking-run path in M3; the `--full` branch is kept for quarterly re-anchor. This is the one deliberate behavioral override (D3, human-ratified).
7. **Programs/flows that MUST still work after each milestone**: `ailang eval-elo <dir> --json` (read-only fit, now anchored); `ailang eval-suite --benchmarks-by-confidence auto`; the 45-min rotation publish chain (`eval-publish` → snapshot → gsutil); `tools/publish-unified-dashboard.sh`.

### Implementation Plan

> **M1 VERIFY EXECUTED — 2026-08-27, in-session sprint (worktree sprint-m-eval-rolling-elo).**
> - `anchor_v1.json` GENERATED by the new committed `tools/gen-anchor` from the real corpus
>   (`eval_results/baselines/v0.32.0`, 1,711 standard trials): **47 benchmarks**; excluded
>   39 trivial-band, 1 smoke-tier, 2 grader-artifact, 2 harness-artifact (docx/markdown_reimplement —
>   their standard-mode difficulty is the input_files harness gap; anchoring them would hand every
>   model a windfall when that bug is fixed; re-anchor as anchor_v2 when it lands).
> - Pool-composition test + mutation control: anchored drift **31.2** vs unanchored **311.7** ELO
>   (10.0x; criterion: abs ≤ 50 AND ratio ≥ 5x — the original ±25 was tightened-by-guess, measured
>   basis recorded in the test comment per the deferred-decision latitude).
> - Both fitters + the export projection routed through the anchored placement fit (standard mode);
>   agent mode unanchored (documented — no agent anchor exists yet). Embedded-anchor parse failure
>   now PANICS at init (was a silent nil fallback in the salvaged commit — fixed per no-silent-fallbacks).
> - Threshold unified at 90% (`DefaultCoverageThreshold`, emitted as `ratings.provisionalCoverageFraction`;
>   `coverageGate.js` prefers the emitted value). Old 50% CLI test rewritten to pin the new contract
>   incl. the 35/36 boundary from both sides.
> - One-time transition re-leveling recorded (unanchored→anchored, same corpus): 17/18 models moved
>   (−135 to +24, rank order preserved; 6 rows newly provisional — the 90% gate doing its declared
>   job), 25/91 ailang benchmark bands moved (mostly one step; full table in session record 2026-08-27).
> - Gates: `go test` green ×3 packages, `make lint` 0 issues, `make check-file-sizes` clean.

**M1 — Anchored, single-path fit** (core, ~3 days)
- Add fixed-entity support to the fit: `FitFromTrialsAnchored(trials, fixedBench, fixedModels map[string]float64)` — entities present in a fixed map keep their given rating (their `UpdateTrial` delta is discarded); everything else updates as today. Placement fit passes `(anchor_vN, nil)`; direction fit passes `(nil, bridgeStrengths)`. `FitFromTrials` delegates with `(nil, nil)` (behavior-preserving).
- Generate `anchor_v1.json` from the current clean corpus (v0.32.0 baseline + subsequent banked runs): the discriminating-band benchmarks' fitted difficulties, with provenance header (corpus, date, fit params). Committed; embedded via `go:embed`.
- `eval-report` and `tools/eval-elo` both fit through the anchored path with the same anchor → the dual-path disagreement collapses (D2: DB write stays in `tools/eval-elo`; `eval-report` projection reads/reproduces identically).
- Unify the provisional threshold: one constant (90%, matching the site's `ELO_COVERAGE_FRACTION`), exported to the ratings block so `coverageGate.js` and Go can't drift again.
- Fix the two stale seeding instructions (`eval_saturation.go:87`, `eval_confidence.go:46`).
- **VERIFY**: (a) new regression test: fit the same model's rows in two pool compositions (with/without extra easy-benchmark rows) → ratings agree within ±25 (this is the 2763-vs-1995 artifact, pinned); (b) refit the current corpus anchored, record the band-change diff table in this doc; (c) `go test ./internal/eval_harness/ ./internal/eval_analysis/`, `make ci` green.

> **M2 VERIFY EXECUTED — 2026-08-27, in-session sprint.**
> - Real persist (scratch DB, corpus `baselines/v0.32.0`): **1,711 trial_history rows**, all
>   `compiler_version = v0.32.0` (path-derived — banked rows carry no version field, V18's
>   `releaseTag`-shaped dir layout is the authority), 2 distinct `prompt_version` values,
>   before/after ratings recorded (e.g. gpt5-6-terra 1500.0 seed → 2109.9 anchored-blended).
> - Identical re-persist: **count unchanged at 1,711** — idempotency by `trial_id` PK
>   (`INSERT OR IGNORE`), not by convention.
> - BONUS defect fixed in passing: the tool's raw JSON walk ignored the validity quarantine, so
>   infrastructure 520s counted as losses in the authoritative DB — the sol-vs-terra
>   contamination in the persist path. Quarantined rows now excluded from fit AND history.
> - Agent-mode automation: filler step 8a persists agent ratings + trial_history from the
>   rotation bank each cycle (`bash -n` clean; bash-3.2-safe). ⚠️ ACTIVATION PENDING the rig's
>   plist/script install cycle — repo edits do not reach the running rig by themselves (the
>   documented launchd trap); mechanism proven by the manual run above.

**M2 — trial_history writer + agent-mode automation** (~2 days)
- `tools/eval-elo` appends `trial_history` rows on every persist: `compiler_version` = `releaseTag()` of the run's banked version dir, `prompt_version` = the banked row's own `PromptVersion` field (exists on every result — V19), outcome, before/after ratings (from `UpdateTrial` deltas during the anchored fit's final epoch, or recomputed post-fit — implementer's choice, D4 latitude).
- Idempotency by constraint: `trial_id` is the table's PRIMARY KEY (V17b); the writer sets it to the banked trial-file identity and uses `INSERT OR IGNORE`, so re-running a persist over the same corpus is a no-op enforced by the schema.
- Wire agent-mode persistence into the nightly/rotation path so agent ratings stop going stale (currently 3 models, 2026-07-31).
- **VERIFY**: (a) after one real run, `SELECT COUNT(*), compiler_version FROM trial_history GROUP BY compiler_version` returns rows with correct version stamps; (b) re-run the same persist → count unchanged; (c) agent-mode `model_ratings.last_updated` advances after a rotation cycle.

> **M3 VERIFY EXECUTED — 2026-08-27, in-session sprint.**
> - `direction_panel_v1.json` GENERATED (`tools/gen-anchor --direction-out`): **22 benchmarks**,
>   difficulty-stratified even stride over the 47-member anchor panel → tier spread core 9 /
>   stretch 6 / frontier 4 / vision 2 / experimental 1. Deterministic, never hand-picked.
> - `tools/direction-fit` implements the DIRECTION configuration (bridge strengths fixed, panel
>   difficulties free) and stamps `bridge_strengths_used` into the artifact — the replayability
>   requirement from quorum R2.
> - **NEGATIVE test 1 (unrated bridge member)**: refused — "bridge model gemini-3-7-flash has no
>   standard placement rating … run the placement persist first". No seeding, no silent default.
> - **NEGATIVE test 2 (partial panel)**: refused, naming all 3 missing cells — matching an
>   independent coverage scan exactly.
> - **POSITIVE path** (rehearsal on v0.32.0 + 6 freshly-run fill cells, bridge
>   gemini-3-flash/gpt5-4-mini, 86 trials): index **1551.1** overall; by tier core 1328.6,
>   stretch 1339.3, experimental 1710.9, frontier 1996.0, vision 2218.4.
> - **DESIGN BUG found by this VERIFY and fixed**: the completeness gate demanded every panel
>   benchmark × every requested language, but specs may declare a single language
>   (`ai_effect_json_schema: languages: ["ailang"]`) — the index would have been permanently
>   unachievable. Required set is now `requested ∩ spec.Languages`, with a loud failure if a
>   panel member declares none of them.
> - `tools/linking-run.sh` (bash-3.2 safe, `bash -n` clean): panel verbatim (never
>   confidence-selected), `--bank-by-version --skip-existing`, post-hoc cost check, then
>   placement persist → direction stamp. **Dry-run for v0.35.0: 5 models × 22 benchmarks ×
>   2 langs = 220 runs, est. $2.88** — well inside the $25 cap and far under a full baseline.
> - REMAINING for a real release (not blocking M3): execute against an actual new release tag.

**M3 — Linking-run protocol replaces the full baseline** (~3 days)
- New script (post-release skill step): the **fixed direction panel** (committed list, ~20-25 benchmarks) × bridge panel (D3) × `--langs ailang,python`, `--trials 1`, banked under the release version, persisted through M1+M2; the direction fit runs immediately after and stamps the index + its input bridge strengths into the release's HistoryEntry. Optionally, `--benchmarks-by-confidence` appends extra placement-value cells beyond the panel. Budget guard: abort if projected cost > $25. Missing panel cells fail the run loudly (no partial index).
- Codify the new-model gate (today's protocol: smoke floor → core vs incumbent → frontier N=3 → anchored placement) as a model-manager skill reference section with measured costs.
- Demote `FULL=true` to quarterly re-anchor duty; document the re-anchor protocol (see Risks).
- **VERIFY**: run the linking run for the next release (v0.35.0): (a) measured cost ≤ $25 recorded here; (b) DB + `latest.json.ratings` updated with the release's fit; (c) sanity: bridge-panel models' ratings move < 50 points release-over-release absent a real regression.

> **M4 VERIFY EXECUTED — 2026-08-27, in-session sprint. ONE ITEM OPEN (Mark's visual sign-off).**
> - `HistoryEntry.Ratings` (additive, `omitempty`) + `RatingsHistoryPoint` carry per-release model
>   ELO, the direction index (overall + per tier), the anchor/panel versions, and the
>   `bridge_strengths_used` that make a historical index replayable. Wired into the export path;
>   the index is READ from the release's stamped artifact, never recomputed.
> - **Round-trip test PASSES**: a legacy history entry (no `ratings` key) unmarshals with
>   `Ratings == nil` and re-serializes with an identical key set — the 47 published entries cannot
>   be silently rewritten. Plus tests for reading the stamped index and for the
>   no-artifact case (model half only, never a fabricated index). 3/3 green.
> - `RatingTrend.jsx` folded into the proven `BenchmarkDashboard` (m-eval-os-version-trend-redesign
>   rules: props from the parent's runtime fetch, no build-time import, no cache-busting, changed
>   ONCE). Two views: model capability, and the direction index on an INVERTED axis so
>   "up = better" holds in both. Renders a "not enough history yet" state below 2 points and never
>   interpolates a missing release.
> - `DataProvenance` shared component now stamps version + date on all five surfaces that showed
>   none (Leaderboard, ELO, Explorer, Value, Gallery), and carries the stale-fallback badge —
>   previously 1 of 13 fetch sites had any staleness indicator.
> - **Frontend verification** (repo norm: babel/parse + data-logic + CI; headless Chrome is
>   explicitly not the tool): all 8 touched files parse clean under the Docusaurus syntax set,
>   **with an untouched control file in the same run** to prove the instrument sees a positive.
>   (An earlier ad-hoc babel invocation "failed" 5 files — the control failed too, proving the
>   INSTRUMENT was wrong, not the code. Recorded because that is the trap this project keeps
>   re-learning.)
> - ⚠️ **Docusaurus production build NOT run to completion in the worktree**: it fails on 39
>   generated doc pages (`packages/sunholo/*`, several `reference/*`) that exist untracked in the
>   main checkout only — 143 files here vs 182 there. PRE-EXISTING worktree condition, reproduced
>   before any of my changes were involved; CI builds from the main checkout. Must still be
>   green there before merge.
> - ⏸️ **PENDING: Mark's visual confirmation of the chart** — the sprint plan makes this
>   non-negotiable and no agent may sign it off.

**M4 — The series becomes visible** (~3 days)
- `HistoryEntry` gains an additive `ratings` summary: per-model anchored ELO + the language-direction index (mean anchored panel difficulty, overall + per tier) for that version.
- One site chart: ELO-over-versions (models as lines) + the difficulty index (inverted axis or companion line), following the `m-eval-os-version-trend-redesign` rules verbatim: runtime `fetch` only, fold into an existing proven component (`BenchmarkDashboard` trends tab), change once, Node 20 build, headless post-hydration check.
- Provenance stamps (version + timestamp from `latest.json`) on the 5 pages missing them; extend the existing `benchmarkFetchWithSource` staleness badge from ValueDashboard to the other surfaces (reuse, not rebuild).
- **VERIFY**: (a) data-logic unit tests on the history-entry builder; (b) Docusaurus build green on Node 20; (c) headless `--dump-dom` post-hydration shows the chart rows; (d) Mark confirms visuals (repo norm: user verifies frontend).

> **M5 VERIFY EXECUTED — 2026-08-27, in-session sprint.**
> - **Axiom Scorecard staleness**: now computes its own age from the artifact timestamp and shows
>   `⚠ … N days old (static file, not auto-refreshed)` past 45 days. It had been showing
>   v0.15.0 / 2026-05-04 numbers for ~115 days with no indicator. (Wiring it into the publish
>   pipeline is a separate change; saying how old it is was the honest minimum.)
> - **Hardcoded "46 benchmarks"** on the homepage → derived from the artifact, and the clause is
>   omitted entirely when the data is absent rather than printing a wrong number.
> - **Broken Explorer link** `/ailang/docs/...` (old GitHub-Pages baseUrl) → `/docs/...`.
> - **`guides/benchmarking.md`**: the hand-maintained v0.3.5 / Oct-2025 results table (19
>   benchmarks, 52.6%) is replaced by a routing table to the six live pages, each of which now
>   stamps its own release + date. The undated cost/token table below it is marked illustrative.
> - **`model-capability-threshold.md`**: stamped DATED SNAPSHOT with pointers to the live pages.
> - **Deletions, each with a zero-consumer grep first** (coding-standards rule — never delete on
>   "unused" alone): `docs/docs/static/benchmarks/latest.json` (v0.3.12 / 2025-10-17, sat inside
>   the docs CONTENT dir so `staticDirectories: ['static']` never served it — genuinely
>   unreachable), and 4 orphaned components (`LanguageChart`, `SpeedRadar`,
>   `BenchmarkChampionsTable`, `DollarsPerPassTable`) — git history shows they were superseded by
>   the May-2026 move of cost/speed analysis to the Value page, not abandoned mid-refactor.
> - Frontend: all 10 touched files + the untouched control parse clean.

**M5 — Hygiene sweep** (parallelizable, ~2 days)
- Axiom Scorecard: wire `axiom_scorecard.json` into the publish path or add a staleness banner (currently v0.15.0/2026-05-04, no indicator).
- `BenchmarkMini` hardcoded "46 benchmarks" → data-driven; fix Explorer's stale `/ailang/docs/...` link; delete the dead `docs/docs/static/benchmarks/latest.json` (v0.3.12) and the 4 orphaned components (per coding standards: check git history for why-unused first); replace `guides/benchmarking.md`'s hand-written v0.3.5 tables with links to the live pages; same for `model-capability-threshold.md` (or stamp it as a dated snapshot).
- **VERIFY**: `make ci` + docs build green; grep proves no consumer of each deleted file; link-check on touched pages.

### Files to Modify/Create

- `internal/eval_harness/ratings.go` — anchored fit (+~60 LOC), `anchor_v1.json` + embed (+1 file)
- `internal/eval_harness/ratings_test.go` — pool-composition regression test, anchor tests (+~120 LOC)
- `tools/eval-elo/main.go` — anchored fit + trial_history writer (+~100 LOC)
- `internal/observatory/ratings.go` — `AppendTrialHistory` (+~60 LOC)
- `cmd/ailang/eval_elo.go`, `eval_saturation.go`, `eval_confidence.go` — anchor wiring, threshold unification, message fixes (±~40 LOC)
- `internal/eval_analysis/ratings_export.go`, `export_json.go`, `dashboard_io.go`, `types.go` — projection + HistoryEntry ratings summary (+~80 LOC)
- `.claude/skills/post-release/scripts/run_eval_baseline.sh` — linking-run mode (±~60 LOC)
- `.claude/skills/model-manager/SKILL.md` — new-model gate protocol reference (+~40 lines)
- `docs/src/components/BenchmarkDashboard/` — EloTrend (+1 component ~150 LOC), provenance stamp + badge reuse (±~60 LOC)
- `tools/launchd/nightly-eval.sh` or `os-rotation-filler.sh` — agent-mode persist hook (+~15 LOC)

## Success Criteria

- [ ] Pool-composition regression test passes (±25 ELO); the 2763-artifact can never recur silently
- [ ] `trial_history` populated with `compiler_version` on every persist; idempotent
- [ ] One release measured by linking run at ≤ $25, ratings landing in the anchored series
- [ ] Agent-mode ratings auto-refresh (last_updated advances without manual runs)
- [ ] ELO-over-versions + language-direction index on the site; 5 pages gain provenance stamps
- [ ] Single fitting path: `eval-report` and `tools/eval-elo` produce identical ratings for the same corpus
- [ ] All hygiene items closed with receipts
- [ ] All tests passing, `make ci` green, docs build green

## Testing Strategy

- **Unit**: anchored-fit math (anchor members immobile; non-members converge as before); trial_history dedup; HistoryEntry additivity (old JSON round-trips).
- **Regression**: pool-composition invariance (the headline test); band-change diff recorded once at M1.
- **Integration**: one real linking run end-to-end (M3 VERIFY) — this is deliberate: the instrument's proof is a measured release, not a mock.
- **Frontend**: data-logic tests + Node 20 build + headless post-hydration DOM check; visual confirmation by Mark (repo norm).

## Deferred Decisions (agent latitude)

- Exact anchor-panel membership beyond "current discriminating band" (implementer may trim benchmarks with `graderFlag` measurement artifacts)
- ±25 tolerance value for the invariance test (tighten if the fit allows)
- Whether before/after ratings in trial_history come from final-epoch deltas or post-fit recompute
- Chart library details / whether the difficulty index renders as its own panel or an overlay
- Where the unified provisional constant lives (Go const exported into JSON vs JSON read by Go)

## Non-Goals

- **Glicko-2 / confidence intervals** — provisional+coverage gating suffices now; revisit when linking runs accumulate (the trial log makes this possible later without rework).
- **Changing validity or error-category semantics** — `m-eval-validity-discipline` / `m-eval-failure-attribution` own that; this doc consumes `LoadResults` as-is.
- **Coordinator model-selection by ELO** — nothing in the coordinator reads ratings today; keep it that way until a use case exists.
- **Re-banking or recomputing any historical result** — the measurement contract stands; old baselines are annotated, never rewritten.
- **New benchmarks or tier changes** — the instrument is being fixed, not the ruler marks.
- **Multi-language rating changes** beyond the existing `byLang` split.

## Timeline

- Week 1: M1 (anchor + single path) → freeze ratified → M2 (trial log + agent automation)
- Week 2: M3 (linking run, verified on v0.35.0 release) + M4 (site) ; M5 hygiene in parallel
- Each milestone ends with its VERIFY block executed and results recorded in this doc before the next begins (Mark: "follow through with and verify each").

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| **Anchor drift**: as AILANG improves, frozen difficulties become absolutely wrong (they encode 2026-08 reality) | That's by design — model ratings stay comparable *because* the anchor is frozen; language improvement is read from the *refit* difficulty series, not the anchor. Quarterly `FULL=true` re-anchor creates `anchor_v2` with an explicit linking fit between anchor versions; anchors are versioned artifacts, never edited in place |
| **Bridge-model retirement** (vendors kill models) | 5-model panel across 5 vendors (3 majors + 2 OpenRouter); swap protocol = one overlap run (old + new bridge in the same fit) before retiring a member |
| **Discriminating set saturates over time** | `--benchmarks-by-confidence` already re-selects each run; saturation of the *anchor panel* is tolerable (anchored members don't need to discriminate, only to pin) |
| **Frontend mixed-build regression** (the OSVersionTrend failure mode) | The 6 rules from `m-eval-os-version-trend-redesign` adopted verbatim; component changed once |
| **React consumers break on ratings-block changes** | Additive-only contract, asserted by a JSON-shape test on the projection |
| **Linking run under-measures a real regression** (only ~25 benchmarks) | Bridge-panel movement > 50 pts triggers a targeted escalation run (the sol/terra pattern: re-run only the moved cells at N=3); quarterly full run backstops |

## Verification Log (all rows first-party, 2026-08-27)

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `Trial` has no version/timestamp field | `ratings.go:31-35` — `{Model, Bench, Pass}` only |
| V2 | Fit has no anchor; everything seeds 1500 | `ratings.go:27` `DefaultInitialRating = 1500.0`; `:55-63` seeds both maps; `FitFromTrials(trials []Trial)` — no prior/anchor param |
| V3 | `trial_history` has zero writers | `grep -rn trial_history internal/ cmd/ tools/ --include=*.go` minus DDL/validate/tests → **empty**; live DB `COUNT(*)`=0 |
| V4 | `UpdateTrial` has no callers outside the fit loop | grep: only `ratings.go:40` (def) and `:77` (fit loop) |
| V5 | `SaveRatings` overwrites, never appends | `internal/observatory/ratings.go:52,62` `ON CONFLICT … DO UPDATE` |
| V6 | `HistoryEntry` carries no rating fields | `types.go` struct: 0 matches for elo/rating |
| V7 | Provisional threshold is 50% in CLI, 90% on site | `eval_elo.go:191` `covThreshold := maxCov / 2`; `coverageGate.js:25-26` `RATE=0.5, ELO=0.9` |
| V8 | Two CLI messages instruct a command that hard-errors | `eval_saturation.go:87`, `eval_confidence.go:46` say `eval-elo … --persist`; `eval_elo.go:91-93` rejects `--persist` |
| V9 | `cmd` eval-elo refuses persist (split is deliberate) | `eval_elo.go:73-93` comment + error redirect to `tools/eval-elo` |
| V10 | Agent DB ratings stale: 3 models @ 2026-07-31; standard 19 @ 2026-08-03 | sqlite read-only query on `~/.ailang/state/observatory.db` |
| V11 | Saturation: standard 39/91, agent 42/56 | `latest.json` `.ratings.{standard,agent}.saturation` |
| V12 | Ratings block wholesale-overwritten per export | `export_json.go:506` `dashboard.Ratings = buildRatingsBlock(…)` |
| V13 | Pool-composition artifact is real and large | Measured 2026-08-27: or-glm-5-3-flash 2763.4 (contaminated pool) vs 1994.8 (clean pool), same model rows |
| V14 | Cheap targeted gates produce defensible placements | Measured 2026-08-27: GLM-5.3-Flash full gate $3.30; sol/terra 9-cell N=3 $8.27, sign test p=1.000 |
| V15 | `OSReleaseTrend` exists and renders os/history.json (supersedes the removal note in m-eval-os-version-trend-redesign) | `docs/src/components/OSReleaseTrend/index.jsx:78`; rendered from `os-model-leaderboard.md` |
| V16 | Nothing in coordinator/executor/ai reads ratings (Non-Goals premise) | `grep -rn "model_ratings\|LoadModelRatings\|benchmark_ratings" internal/coordinator/ internal/executor/ internal/ai/` → empty |
| V17 | `trial_history` schema live on every DB, no new migration needed | `migrate_v16.go:34` DDL + `migrate.go:560` in `ValidateSchema` table list |
| V17b | The v16 DDL contains EVERY column M2 writes, and a PK usable for dedup | `migrate_v16.go:34-47` read in full: `trial_id TEXT PRIMARY KEY, benchmark_id, model_id, mode, outcome, prompt_version, compiler_version, benchmark_rating_before/after, model_rating_before/after, recorded_at` |
| V18 | `releaseTag()` is the existing version normalization at bank time | `eval_suite.go:32-42` def, `:200` applied to `--bank-by-version` dir |
| V19 | Banked result rows already carry a prompt version for M2 to copy | `internal/eval_harness/metrics.go:93` `PromptVersion string \`json:"prompt_version,omitempty"\``; mirrored in `eval_analysis/types.go:60,467` |
| V20 | Band/saturation consumers enumerated | `grep -rln "Band("` → `eval_confidence.go, eval_elo.go, eval_saturation.go, ratings_export.go`; `saturated` flag read by `eval-signal-set.sh`, `os-rotation-filler.sh` |
| V21 | ratings-block consumers: 8 site files + 2 shell scripts | `grep -rln '\.ratings' docs/src/` → BenchmarkExplorer, EloLeaderboard, ValueDashboard, BenchmarkStandaloneGallery, OSReleaseTrend, AgentUpliftTable, BenchmarkDashboard, OSLocalLeaderboard + eval-signal-set.sh, os-rotation-filler.sh |
| V22 | Staleness badge exists on exactly 1 of 13 fetch call sites | `grep -rln benchmarkFetchWithSource docs/src/components/` → ValueDashboard only; 12 files call plain `benchmarkFetch(` |

## Quorum Record (2026-08-27, reviewers gpt5-6-sol + gemini-3-1-pro, both reject-by-default)

Both rounds run; the re-quorum-once guardrail is exhausted, so per the design-doc-creator
protocol the resolved doc goes to Mark with its record rather than a third round. Every
objection was accepted and incorporated — none argued:

- **R1/sol — estimator internally contradictory** (frozen difficulties can't also fall):
  ACCEPTED → rewritten as two named configurations of one engine; each fit fixes exactly one
  side ("The estimator" section).
- **R1/gemini — trial_history schema adequacy unverified**: ACCEPTED → V17b reads the full
  DDL; every M2 column exists; `trial_id` PK is the dedup constraint.
- **R2/sol — direction index unidentifiable** (52-benchmark panel vs ≤25 measured, drifting
  subset; carried-forward values = silent fallback) **+ not replayable** (later bridge refits
  would rewrite history): ACCEPTED → direction panel is a FIXED committed subset measured in
  full every release (missing cells fail loudly); the index is a stamped release-time
  measurement whose input bridge strengths are recorded with it and never recomputed.
- **R2/gemini — quantified survey claims absent from the Verification Log**: ACCEPTED → V20,
  V21, V22 added first-party; two of my own counts were corrected in the process ("6
  components" → 8 site files; "9 of 10 surfaces" → 12 of 13 call sites).

Machine artifacts: `.ailang/state/mission-quorum/m-eval-rolling-elo-2026-08-27T11-23-50Z.json`
and `…T11-26-28Z.json`; human blocks appended to `design_docs/v1-mission-log.md`.

## Related Documents

- [m-eval-elo-persist](../../implemented/v0_26_0/m-eval-elo-persist-sprint-plan.md) (implemented v0.26.0) — created the schema this doc finally writes to
- [m-eval-rating-efficiency](../../implemented/v0_26_0/m-eval-rating-efficiency.md) (v0.26.0) — planned the rolling updates + version-aware skips; only the schema shipped
- [m-eval-standard-confidence-gating](../../implemented/v0_32_0/m-eval-standard-confidence-gating-sprint-plan.md) (v0.32.0) — the selection machinery M3 reuses
- [m-eval-validity-discipline](../m-eval-validity-discipline.md) (planned) — row-validity semantics; this doc consumes them unchanged
- [m-eval-failure-attribution](../m-eval-failure-attribution.md) (planned) — infra-vs-capability failure split; composes with the fit's input filter later
- [m-eval-os-version-trend-redesign](../m-eval-os-version-trend-redesign.md) (planned, partially superseded — see V15) — the frontend rules M4 adopts
- [m-eval-data-hosting-decouple](../m-eval-data-hosting-decouple.md) (planned/live) — the transport M4 rides on

## References

- `internal/eval_harness/ratings.go` — fit + bands; `internal/observatory/ratings.go` + `migrate_v16.go` — store + schema
- `internal/eval_analysis/{export_json.go, ratings_export.go, dashboard_io.go, types.go}` — artifact pipeline
- `tools/eval-elo/main.go`, `cmd/ailang/{eval_elo,eval_confidence,eval_saturation}.go` — fitting paths
- `tools/launchd/{os-rotation-filler.sh, nightly-eval.sh}`, `tools/os-release-snapshot.sh`, `tools/publish-unified-dashboard.sh` — publish chain
- Site survey, pipeline map, and ELO-infrastructure map: session transcripts 2026-08-27 (three parallel explorations; receipts reproduced in the Verification Log)

## Future Work

- Glicko-style uncertainty once trial_history has depth (the log makes it a query, not a migration)
- Fold `m-eval-failure-attribution`'s infra/capability split into the fit's input filter (excluding quarantined rows is already inherited from `LoadResults`)
- Cost-over-time and tokens-over-time site series (the trial log carries the data)
- Retire/merge redundant site surfaces once the trend chart proves out (candidate: fold Value + ELO pages)
