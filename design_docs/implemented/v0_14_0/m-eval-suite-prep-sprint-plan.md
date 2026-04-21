# Sprint Plan: M-EVAL-SUITE-PREP

**Status**: Implemented (v0.14.0)
**Target**: v0.14.0
**Estimated**: 5–6 days (~40–48 hours)
**Priority**: P2 (pre-work for M-EVAL-EXPAND)
**Role**: Pre-work for [M-EVAL-EXPAND](../v0_13_0/m-eval-expand-harnesses-languages.md) (new languages + harness dimensions)

## Related Design Docs (source-of-truth)

- [M-EVAL-CATEGORY-ANALYSIS](../v0_13_0/m-eval-category-analysis.md) — tags taxonomy, refusal detection, saturation, AILANG-only-wins
- [M-BENCHMARK-SUITE-TIERS](../v0_13_0/m-benchmark-suite-tiers.md) — smoke/core/stretch/vision tier structure + assignments

This sprint plan **combines the scope of both** into a single execution plan. The two source docs remain the design record; this plan drives execution.

---

## Why This Sprint (and Why Now)

The eval suite is the primary metric we'll use to justify adding **more languages and new harness dimensions** in M-EVAL-EXPAND. Before that expansion, two structural problems make every added dimension more expensive:

1. **Saturation**: 16/47 benchmarks pass 100% on every frontier model — they cost eval budget but produce no signal. Running these against N new languages/models multiplies waste by N.
2. **No categorical slicing**: v0.9.1.1 analysis revealed +27.8% AILANG advantage on `contracts`, -20% gap on `recursion` — but the harness has no built-in way to produce these breakdowns. Adding languages without this slice gives bigger tables with the same blindness.

**Doing this pre-work first** means M-EVAL-EXPAND lands on a suite that already knows how to report "AILANG vs Python, by category, with refusals excluded, per tier" — the analysis quality compounds.

---

## Milestone Breakdown

Seven milestones, structured as: **schema → data → CLI → reporting → new benchmarks → dashboard → docs**.

| # | ID | Title | Est. LOC | Depends on |
|---|----|-------|----------|-----------|
| M1 | M1_SCHEMA | Extend BenchmarkSpec with Tier + Tags | 80 | — |
| M2 | M2_TAG_BENCHMARKS | Tag all 51 benchmarks (tier + 1–3 tags) | 100 (YAML) | M1 |
| M3 | M3_CLI_FLAGS | CLI filter flags (`--tier`, `--by-tags`, `--show-saturated`, `--ailang-wins`) | 120 | M1, M2 |
| M4 | M4_REPORTING | Per-tier aggregates + refusal detection + saturation + AILANG-only-wins | 260 | M1, M2 |
| M5 | M5_MAKE_AND_NEW_BENCHMARKS | Makefile tier targets + 2 new benchmark YAMLs | 150 | M1, M2, M3 |
| M6 | M6_DASHBOARD | Per-tier toggle in BenchmarkDashboard + latest.json publication | 150 | M4 |
| M7 | M7_DOCS_AND_SKILL | CURATION.md + eval-analyzer skill update + evaluation guide | 200 | M1–M6 |

**Total estimate**: ~1,060 LOC + 51 YAML updates + 2 new benchmark files
**Schedule**: 5–6 days of focused work

---

## M1 — Schema & BenchmarkSpec

**Goal**: Extend `BenchmarkSpec` with `Tier` and `Tags` fields; validate both on load.

**Files modified:**
- `internal/eval_harness/spec.go` — Add `Tier string` and `Tags []string` fields (~15 LOC)
- `internal/eval_harness/spec.go` — Add `validateTierAndTags()` called from `LoadSpec` (~40 LOC)
- `internal/eval_harness/spec_test.go` (new or existing) — Parsing + validation tests (~100 LOC of tests)

**Validation rules:**
- Valid tiers: `smoke`, `core`, `stretch`, `vision`. Missing tier defaults to `core` (back-compat per tier doc §Implementation Plan Phase 1).
- Tags: 1–3 per benchmark from approved taxonomy (12 tags in category doc §Component 1).
- Unknown tags produce a warning (not a hard error) — agent may extend taxonomy.

**Acceptance criteria:**
- `BenchmarkSpec` has `Tier string` and `Tags []string` fields with YAML tags
- `LoadSpec` validates tier enum and tag count (1–3)
- Missing `tier:` field defaults to `"core"` with no error
- Test: loading benchmark with `tier: smoke` and `tags: [adt_pattern_match]` succeeds
- Test: loading benchmark with `tier: bogus` returns descriptive error
- Test: loading benchmark with 4+ tags returns descriptive error
- `make test` passes

**Dependencies**: none

---

## M2 — Tag all 51 Benchmarks

**Goal**: Every benchmark YAML has `tier:` and `tags:` fields per the assignments in the two source docs.

**Files modified:**
- `benchmarks/*.yml` — 51 files, add `tier:` + `tags:` to each (~2 lines per file)

**Authoritative assignments:**
- **Tier assignments**: M-BENCHMARK-SUITE-TIERS §Concrete Tier Assignments (T0 Smoke list, T1 Core list, T2 Stretch list, T3 Vision list)
- **Tag taxonomy**: M-EVAL-CATEGORY-ANALYSIS §Component 1 (12 tags, 1–3 per benchmark)

**New CI check:**
- Add a test in `internal/eval_harness/spec_test.go` that loads all YAMLs under `benchmarks/` and asserts every one has at least one tag and a valid tier.

**Acceptance criteria:**
- All 51 benchmarks in `benchmarks/*.yml` have `tier:` and `tags:` fields
- Tier distribution roughly matches the doc's plan: ~15 smoke, ~20 core, ~8 stretch, ~5 vision (actual counts may vary by ±2)
- Every benchmark has between 1 and 3 tags from the approved taxonomy
- New integration test `TestAllBenchmarksHaveTierAndTags` passes
- `make ci` passes

**Dependencies**: M1 (schema must exist first)

---

## M3 — CLI Filter Flags

**Goal**: Expose tier + tag filtering through the CLI on `eval-suite` and `eval-matrix`.

**Files modified:**
- `cmd/ailang/eval_suite.go` — Add `--tier` flag (~30 LOC)
- `cmd/ailang/eval_tools.go` (or wherever `eval-matrix` lives) — Add `--by-tags`, `--show-saturated`, `--ailang-wins` (~70 LOC)
- `cmd/ailang/eval_suite_test.go` — Flag parsing + filter behaviour (~50 LOC)

**Behaviour:**
- `ailang eval-suite --tier smoke` runs only T0 benchmarks
- `ailang eval-suite --tier smoke,core` runs T0 + T1
- `ailang eval-matrix <dir> <version> --by-tags` groups results by tag
- `--show-saturated` adds a saturation section to the report
- `--ailang-wins` adds an AILANG-only-wins section

**Acceptance criteria:**
- `ailang eval-suite --tier smoke --help` documents the flag
- `ailang eval-suite --tier smoke --benchmarks /dev/null` (dry-check) produces the correct filtered benchmark list
- `ailang eval-matrix <baseline-dir> <version> --by-tags` outputs per-tag table with AILANG/Python deltas
- Flag parsing tests pass
- Backwards compatible: existing commands without flags continue to work unchanged

**Dependencies**: M1 (schema), M2 (data to filter on)

---

## M4 — Reporting: Aggregates, Refusal Detection, Saturation, AILANG-Only-Wins

**Goal**: Add analysis primitives to `internal/eval_analysis/` that power all new CLI flags.

**Files modified:**
- `internal/eval_analysis/types.go` — Add `RefusalDetected bool` to result type (~5 LOC)
- `internal/eval_analysis/loader.go` — Scan stderr/stdout for refusal patterns (~40 LOC)
- `internal/eval_analysis/matrix.go` — Per-tier aggregates, saturation detection (~80 LOC)

**Files created:**
- `internal/eval_analysis/tags.go` — `LoadBenchmarkTags`, `GroupByTags`, AILANG-only-wins logic (~200 LOC)
- `internal/eval_analysis/tags_test.go` — Unit tests for grouping + refusal detection (~150 LOC of tests)

**Refusal patterns (start conservative, extend as discovered):**
- `"Apologies"`, `"I cannot"`, `"I'm sorry, but"`, `"As an AI"`
- Stored as a constant list in `loader.go`; easy to add patterns later.

**Saturation rule:**
- A benchmark is saturated when pass rate = 100% across all models for both languages in the **latest 2 baselines** (read from `eval_results/baselines/`).
- Saturated benchmarks are flagged, never auto-removed.

**AILANG-only-wins rule:**
- For each `benchmark × model`: count cases where AILANG passes and Python fails (and vice versa).
- Aggregate to find consistent patterns (3+ models agreeing).

**Acceptance criteria:**
- `LoadBenchmarkTags(dir)` returns `map[string][]string` of benchmark ID → tags
- `GroupByTags` produces per-tag aggregates with AILANG vs Python deltas
- Refusal detection flags at least the 4 known patterns; `RefusalDetected` is serialised in result JSON
- Saturation detection against `eval_results/baselines/v0.12.0` matches the known saturated list from the tier doc (±2 for tolerance)
- AILANG-only-wins report produces a list matching the v0.9.1.1 manual analysis within 1 benchmark of difference
- `make test` passes; new unit tests cover grouping + refusal detection edge cases

**Dependencies**: M1, M2

---

## M5 — Makefile Targets + New Benchmarks

**Goal**: Ergonomic invocation + add two genuinely hard benchmarks to re-expand the signal range.

**Files modified:**
- `Makefile` — Add `eval-smoke`, `eval-core`, `eval-stretch` targets (~20 LOC)

**Files created:**
- `benchmarks/polymorphic_ord_defaulting.yml` — T2 stretch; regresses if M-POLY-ORD fix is undone (~60 LOC)
- `benchmarks/typed_refusal.yml` — T3 vision; AI must refuse a request that can't be satisfied with granted effect row (~70 LOC)

**Deferred (require M-EFFECT-REFINEMENT, which is targeted at v1.0.0):**
- `effect_row_refinement` — deferred; author once parameterised effect rows ship
- `capability_budget_enforcement` — deferred; same reason

**Makefile target behaviour:**
- `make eval-smoke` → `ailang eval-suite --tier smoke --models claude-sonnet-4-6` (~90s target)
- `make eval-core` → full core tier across top-4 models
- `make eval-stretch` → stretch tier across all configured models

**Acceptance criteria:**
- `make eval-smoke` runs in <120s on a warm cache (target <90s per doc; allow a 30s buffer for CI)
- Both new benchmarks have YAML + `expected_stdout` + reference implementations (AILANG + Python)
- Both new benchmarks are **actually hard**: best frontier model scores <60% on stretch, <40% on vision (verify with one real eval run before marking M5 done)
- Makefile help includes the new targets

**Dependencies**: M1, M2, M3

---

## M6 — Dashboard Per-Tier Toggle

**Goal**: Surface tier breakdown on the website dashboard; publish per-tier aggregates to `latest.json`.

**Files modified:**
- `internal/eval_analysis/export_json.go` — Write per-tier aggregates into `docs/static/benchmarks/latest.json` under a `tiers` key (~40 LOC)
- `internal/eval_analysis/types.go` — Add `Tiers map[string]TierAggregate` to `DashboardJSON` (~10 LOC)
- `docs/src/components/BenchmarkDashboard/index.jsx` — Add tier toggle UI (~80 LOC)
- `docs/src/components/BenchmarkDashboard/*.jsx` — Wire the tier filter through any child components that render pass rates (~20 LOC)

**Acceptance criteria:**
- `docs/static/benchmarks/latest.json` has a `tiers` object with per-tier pass rates (smoke/core/stretch/vision)
- Dashboard renders a tier toggle; selecting a tier filters the displayed benchmarks
- `npm run build` in `docs/` succeeds (Docusaurus build green)
- Core tier pass rate is the primary headline metric on the dashboard
- No regressions in existing dashboard components (smoke-test all views)

**Dependencies**: M4 (the per-tier aggregates)

---

## M7 — Documentation + Skill Update

**Goal**: Make the new workflow discoverable for humans and for the `eval-analyzer` skill.

**Files created:**
- `benchmarks/CURATION.md` — Tier + tag guidelines; when to promote/retire/rotate (~100 LOC)
- `.claude/skills/eval-analyzer/scripts/category_analysis.sh` — Wrapper for `--by-tags` flow (~50 LOC)
- `.claude/skills/eval-analyzer/scripts/benchmark_health.sh` — Saturation + refusal report (~50 LOC)

**Files modified:**
- `.claude/skills/eval-analyzer/SKILL.md` — Document `--by-tags` / `--show-saturated` / `--ailang-wins` workflows (~60 LOC)
- `.claude/skills/eval-analyzer/resources/jq_queries.md` — Add tag-aware queries (~30 LOC)
- `docs/docs/guides/evaluation/*.md` — Document tier structure + release-metric conventions (~40 LOC)
- `CHANGELOG.md` / `changelogs/v0.10-current.md` — Add v0.14.0 entry for M-EVAL-SUITE-PREP

**Acceptance criteria:**
- `benchmarks/CURATION.md` exists with: tier definitions, tag taxonomy, rotation rules, promotion/demotion criteria
- `eval-analyzer` skill documents the new workflows and references the two scripts
- Both new scripts run end-to-end against `eval_results/baselines/v0.13.0`
- `docs/docs/guides/evaluation/` updated; Docusaurus build green
- CHANGELOG entry added under v0.14.0

**Dependencies**: M1–M6

---

## Success Metrics (measurable on v0.14.0 baseline)

- [ ] All 51 benchmarks have `tier:` + `tags:` fields
- [ ] 2 new benchmarks shipped (polymorphic_ord_defaulting, typed_refusal)
- [ ] `make eval-smoke` runs in <120s
- [ ] Core tier best/worst model spread ≥15pp (from ~6pp today)
- [ ] Stretch tier best model <60% pass
- [ ] `ailang eval-matrix --by-tags` reproduces v0.9.1.1 category analysis within ±1 benchmark
- [ ] Refusal detection flags ≥4 patterns
- [ ] Dashboard shows per-tier toggle; `latest.json` has `tiers` aggregates
- [ ] All tests passing (`make ci`)

---

## High-Impact Decisions (carried over from source docs)

| Decision | Resolved | Note |
|----------|----------|------|
| Tier taxonomy: 4 tiers (smoke/core/stretch/vision) | ✅ per tier doc | M1 enforces enum |
| Tag taxonomy: 12 tags, 1–3 per benchmark | ✅ per category doc | M1 enforces count |
| Missing `tier:` defaults to `core` | ✅ | M1 back-compat |
| Saturated benchmarks flagged, not auto-removed | ✅ per category doc | M4 implements |
| Vision benchmarks requiring M-EFFECT-REFINEMENT are deferred | ✅ (v1.0.0 target) | M5 defers 2 of 4 proposed new benchmarks |
| Core tier rate is the headline release metric | ✅ per tier doc | M6 surfaces on dashboard |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tier assignments in source doc don't perfectly match current benchmark set (doc said 47, we have 51) | Low | M2 resolves each new benchmark independently; no blocking issue |
| Refusal patterns evolve across model versions | Low | Constant list in `loader.go`; easy to extend in future sprint |
| Dashboard changes break existing views | Med | M6 smoke-tests all views; tier toggle defaults to core (current behaviour) |
| New stretch/vision benchmarks don't actually score <60%/<40% | Med | M5 acceptance criterion requires a real eval run to verify difficulty |
| `latest.json` schema change breaks downstream consumers | Low | Additive change (`tiers` key added); existing fields preserved |

---

## Timeline

- **Day 1**: M1 + M2 (schema + tagging; safe to land alone)
- **Day 2**: M3 + first half of M4 (CLI + refusal detection + saturation)
- **Day 3**: M4 finish + M5 (AILANG-only-wins + Makefile + new benchmarks)
- **Day 4**: M6 (dashboard tier toggle + latest.json)
- **Day 5**: M7 (CURATION.md + skill update + evaluation guide)
- **Day 6**: Buffer for fix-ups, re-baseline on v0.14.0, release prep

---

## Non-Goals

- **Not** adding new languages to the harness — that's M-EVAL-EXPAND (this sprint's follow-on)
- **Not** rewriting existing benchmarks — tiering is pure classification
- **Not** auto-promoting/demoting benchmarks based on pass rate — tier is editorial
- **Not** per-category trend charts (future work in M-EVAL-CATEGORY-ANALYSIS §Future Work)
- **Not** authoring the two deferred vision benchmarks — waits for M-EFFECT-REFINEMENT (v1.0.0)

---

## Follow-on Sprint

After this sprint lands, **M-EVAL-EXPAND** ([design doc](../v0_13_0/m-eval-expand-harnesses-languages.md)) becomes substantially easier: new languages plug in behind the existing tier + tag infrastructure; new harness dimensions emit into the same per-tier aggregates; no retrofit needed.

---

**Created**: 2026-04-21
**Author**: sprint-planner (Claude Opus 4.7)
