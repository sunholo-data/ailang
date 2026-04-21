# M-BENCHMARK-SUITE-TIERS: Restructure the Eval Suite into Smoke / Core / Stretch Tiers

**Status**: Implemented (v0.14.0, M-EVAL-SUITE-PREP)
**Target**: v0.13.0
**Priority**: P2 — Medium (eval-infra; not a language blocker but drives all other prioritisation)
**Estimated**: 2–3 days (~16–24 hours, mostly YAML + analysis scripts)
**Dependencies**: None (uses existing `ailang eval-suite`, `eval-matrix`, `eval-summary`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval tooling; no language semantics affected |
| A2: Replayability | +1 | Tier metadata preserved in results → reproducible category analysis across versions |
| A3: Effect Legibility | 0 | No language changes |
| A4: Explicit Authority | 0 | No language changes |
| A5: Bounded Verification | +1 | Per-tier breakdowns bound analysis to focused areas; faster feedback |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Suite curated around machine-synthesis gaps, not human intuition; each tier answers a concrete question for an AI reader |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-tier cost analysis exposes which task types are expensive for AI |
| A10: Composability | 0 | No language changes |
| A11: Structured Failure | +1 | Fewer saturated benchmarks → failure categories become meaningful again |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly optimises for machine analysis — sharper signal per dollar

---

## Related Documents

**Prior art (must read before implementing):**

- [`design_docs/archive/2025-10/analysis/AGENT_BENCHMARK_WORK_SUMMARY.md`](../../archive/2025-10/analysis/AGENT_BENCHMARK_WORK_SUMMARY.md) — October 2025 audit proposing a 4-tier structure (Smoke / Differentiators / Vision / JSON-new). This doc operationalises those recommendations with v0.12.0 data.
- [`design_docs/archive/2025-10/analysis/BENCHMARK_AUDIT_ANALYSIS.md`](../../archive/2025-10/analysis/BENCHMARK_AUDIT_ANALYSIS.md) — 500+ line audit of 38 benchmarks with difficulty + capability matrices.
- [`design_docs/planned/v0_11_0/m-eval-category-analysis.md`](../v0_11_0/m-eval-category-analysis.md) — Planned: per-category tagging + refusal detection. Complementary; tiering is the structural axis, categories are the semantic axis.
- [`design_docs/planned/v0_13_0/m-eval-expand-harnesses-languages.md`](m-eval-expand-harnesses-languages.md) — Planned: expand harness with more models/languages. This doc is about **which** benchmarks we run; that one is about **across what** we run them.

**Supersedes**: nothing (first attempt to land the 2025-10 recommendations).

---

## Problem Statement

### The Suite Has Saturated

v0.12.0 baseline (`eval_results/baselines/v0.12.0/summary.jsonl`, 8 AILANG runs per benchmark across 8 frontier models: GPT-5.2-codex, GPT-5.4, Claude Sonnet 4.5, Claude Sonnet 4.6, Claude Opus 4.7, Claude Haiku 4.5, Gemini-3 Flash, Gemini-3.1 Pro):

| Bucket | Count | Meaning |
|--------|-------|---------|
| **AILANG 100% pass** | 16/47 benchmarks | Every frontier model solves it every time — **no signal** |
| **AILANG + Python 100%** | 9/47 benchmarks | Saturated across both languages — pure smoke value only |
| **AILANG 85–99%** | 20/47 benchmarks | One flaky model; effectively solved |
| **AILANG <50%** | 7/47 benchmarks | The actual signal — language / prompt / capability gaps |
| **AILANG <25%** | 2/47 benchmarks | Hardest frontier (contract_sorted_merge, log_file_analyzer) |

**Concrete 100%-both-languages list** (candidates for smoke tier, run cheaply every commit):
`adt_option, balanced_parens, canonical_normalization, explicit_state_threading, gcd_lcm, numeric_modulo, records_book, recursion_fibonacci, type_safe_record_access`

**Concrete AILANG 100% but Python flaky** (differentiator candidates — AILANG advantage is real):
`effect_pure_separation, exhaustive_pattern_matching, immutable_data_structures, inline_tests, json_encode, record_update`

**Concrete <50% list** (core signal, keep and invest):
`contract_sorted_merge (16%), log_file_analyzer (25%), contract_matrix_determinant (33%), contract_rle_roundtrip (33%), contract_roman_numeral (33%), red_black_tree (37%), type_unify (37%), config_file_parser (50%), json_transform (50%), lambda_calc (50%)`

### Why This Matters

Running 368 benchmark executions (~$0.50–1.00, ~15–20 min) to learn that 16 of 47 benchmarks are at 100% wastes budget and obscures signal. A release that moves all the hard benchmarks by +5pp looks identical in the headline to a release that does nothing — because saturated benchmarks dominate the denominator.

Worse, when we add a language feature specifically to unlock a hard benchmark (e.g. M-CONCAT-DISAMBIG for `type_unify`), the +1 on `type_unify` gets diluted by 16 benchmarks stuck at 100%.

### Why Not Just Delete Saturated Benchmarks?

Because they still catch regressions. A future refactor that breaks `fizzbuzz` is a 🚨 — we need to see it. Smoke tier preserves that guarantee at ~10× lower cost.

---

## Goals

**Primary Goal**: Restructure the benchmark suite into explicit tiers so that each tier answers a specific question, and each release can be measured against the tier most sensitive to the change.

**Success metrics** (measurable on v0.13.0 baseline):

1. **Core tier shows ≥15pp spread** between best and worst model (today: ~6pp because saturation compresses the range).
2. **Smoke tier runs in <90 seconds** across all 8 models (AILANG only), suitable for pre-push validation.
3. **Stretch tier shows <60% pass on best model** (i.e. always some headroom).
4. **At least 3 new stretch benchmarks** added targeting AILANG-specific strengths (effect tracking, capability budgets, ADT exhaustiveness).
5. **No regression in total benchmark coverage of language features** — every feature currently exercised is still exercised somewhere.

---

## Solution Design

### Tier Structure

| Tier | Name | Size | Purpose | When run |
|------|------|------|---------|----------|
| **T0** | Smoke | ~15 | Regression guard — 100%-both-languages benchmarks | Every commit (pre-push hook); AILANG only |
| **T1** | Core | ~20 | Primary release metric — where frontier models split | Every release (`make eval-baseline`) |
| **T2** | Stretch | ~8 | Forward-looking — hard today, guide language roadmap | Every release; tracked separately in dashboard |
| **T3** | Vision | ~5 | AILANG-only wins — effect rows, capability budgets, typed refusals | Every release; "AILANG advantage" metric |

Total: ~48 benchmarks (current: 47 standard + 6 contract = 53). Net change is reclassification, not deletion.

**Tagging mechanism**: add `tier: smoke|core|stretch|vision` to each benchmark YAML. `ailang eval-suite --tier core` filters.

### Concrete Tier Assignments (proposed)

**T0 Smoke** (saturated across AILANG + Python; preserve as regression guards):
- `adt_option`, `balanced_parens`, `canonical_normalization`, `explicit_state_threading`, `gcd_lcm`, `numeric_modulo`, `records_book`, `recursion_fibonacci`, `type_safe_record_access`
- Plus `fizzbuzz` (traditional smoke), `immutable_data_structures`, `record_update`, `effect_pure_separation`, `exhaustive_pattern_matching`, `inline_tests`
- **Total: 15**. Target runtime: <90s, 1 run per model, no agent mode.

**T1 Core** (the benchmarks that actually separate models — where language/prompt/capability improvements land):
- Hard contracts: `contract_sorted_merge`, `contract_matrix_determinant`, `contract_rle_roundtrip`, `contract_roman_numeral`, `contract_bst_validate`
- Compile-error signal: `type_unify`, `lambda_calc`, `red_black_tree`, `config_file_parser`, `json_transform`, `expression_evaluator`
- Multi-file / effect-heavy: `csv_to_json_converter`, `log_file_analyzer`, `api_call_json`, `effect_composition`, `effect_tracking_io_fs`, `error_handling`
- Algorithm + control flow: `merge_sort`, `graph_bfs`, `pipeline`, `tree_transformation_pipeline`
- **Total: ~20**. Agent mode + standard mode both.

**T2 Stretch** (today's best models score <60%):
- `contract_sorted_merge` (stretch-goal marker — currently 16%)
- New benchmarks targeting known gaps:
  - `effect_row_refinement` (uses M-EFFECT-REFINEMENT syntax once it lands)
  - `crypto_rand_policy` (uses CryptoRand capability)
  - `polymorphic_ord_defaulting` (exercises M-POLY-ORD fix)
- Existing hard benchmarks: `symbolic_diff`, `run_length_encode`, `mini_interpreter`

**T3 Vision** (AILANG-only wins; Python has no equivalent or fails by design):
- `effect_pure_separation` (promote from smoke — Python can't express it)
- `capability_budget_enforcement` (NEW — requires M-EFFECT-REFINEMENT)
- `typed_refusal` (NEW — agent must refuse to call a non-permissioned effect)
- `exhaustive_pattern_matching` (keep one canonical example here vs smoke)
- `no_runtime_crashes_option` (currently core-ish; show the AILANG story)

### Retirement List (move to T0 smoke or drop entirely)

From today's agent suite (46 benchmarks), demote to smoke-only:
`adt_option, balanced_parens, canonical_normalization, effect_pure_separation (→ T3), exhaustive_pattern_matching (→ T3), explicit_state_threading, fizzbuzz, float_eq, fold_reduce, gcd_lcm, higher_order_functions, immutable_data_structures, inline_tests, json_parse, nested_records, no_runtime_crashes_option (→ T3), numeric_modulo, pattern_matching_complex, record_update, records_book, recursion_fibonacci, type_safe_record_access`

That's 23 benchmarks. Removing them from the agent tier alone cuts the hot-path eval cost roughly in half with no loss of signal on the remaining 23.

### New Benchmark Proposals (stretch + vision tiers)

Three benchmarks tied to shipping language features:

1. **`effect_row_refinement`** (T2) — Verify AI can use the refined effect row syntax from M-EFFECT-REFINEMENT to distinguish `Rand[mode=crypto]` from `Rand[mode=prng]`.
2. **`capability_budget_enforcement`** (T3) — AI is given a capability budget (e.g. max 10 FS reads) and must write code that the compiler can prove stays under budget.
3. **`typed_refusal`** (T3) — AI must recognise a request that cannot be satisfied with the granted effect row and return a typed error rather than fabricate a workaround. Measures "does AI understand what AILANG *can't* do".

One benchmark tied to a v0.12.0 gap:

4. **`polymorphic_ord_defaulting`** (T2) — Sprint M-POLY-ORD just landed. Add a benchmark that regresses if the fix is undone; today there's no agent-level regression test for this class of inference.

### Dashboard + Reporting Changes

- `ailang eval-matrix` grows a `--tier` filter.
- `ailang eval-report` emits per-tier aggregates to `docs/static/benchmarks/latest.json`.
- Website dashboard shows three stacked sparklines (smoke/core/stretch) instead of one headline number.
- Release notes use core-tier rate as the primary metric; smoke is mentioned only if it regressed.

### Implementation Plan

**Phase 1 — Tagging (4h)**
- Add `tier:` field to benchmark YAML schema (`internal/eval_harness/benchmark.go`).
- Tag all 47 existing benchmarks per the assignments above.
- Backward-compatible: missing `tier:` defaults to `core`.

**Phase 2 — Filtering + runner (4h)**
- Add `--tier` flag to `ailang eval-suite` and `eval-matrix`.
- Update `make eval-baseline` to run all tiers by default, `make eval-smoke` for T0 only.
- Update post-release script to tag result directories with tier breakdown.

**Phase 3 — Reporting (4h)**
- Per-tier aggregates in `latest.json`.
- Website dashboard: add tier toggle.
- CHANGELOG template: mention core tier rate prominently.

**Phase 4 — New benchmarks (4–8h)**
- Author the four new benchmark YAMLs + expected outputs.
- Verify they are actually hard (best model <60% for T2, <40% for T3).

### Files to Modify

| File | Change | LOC (est) |
|------|--------|-----------|
| `internal/eval_harness/benchmark.go` | Add `Tier` field, parse from YAML | +20 |
| `internal/eval_harness/suite.go` | Add tier filter | +15 |
| `internal/eval_analysis/matrix.go` | Per-tier aggregates | +40 |
| `internal/eval_analysis/report.go` | Tier breakdown in JSON output | +30 |
| `benchmarks/*.yml` | Add `tier:` to all 47 | ~47 × 1 line |
| `benchmarks/effect_row_refinement.yml` | NEW | +60 |
| `benchmarks/capability_budget_enforcement.yml` | NEW | +70 |
| `benchmarks/typed_refusal.yml` | NEW | +55 |
| `benchmarks/polymorphic_ord_defaulting.yml` | NEW | +50 |
| `docs/src/components/BenchmarkDashboard/*` | Tier toggle | +100 |
| `Makefile` | `eval-smoke`, `eval-core`, `eval-stretch` targets | +15 |
| `.claude/skills/post-release/scripts/run_eval_baseline.sh` | Tier-aware output | +20 |
| **Total** | | **~560 LOC + 47 YAML tags + 4 new benchmark files** |

---

## High-Impact Decisions

| # | Decision | Who | Change cost if wrong |
|---|----------|-----|---------------------|
| 1 | Tier is a single field, not composable tags | Design | Low — rename field later |
| 2 | Promote `effect_pure_separation` from smoke to vision (not both) | Design | Low — dual-tier possible later |
| 3 | Core tier rate is the headline release metric | Product | Medium — one release cycle of dashboard churn if reverted |
| 4 | New vision benchmarks require M-EFFECT-REFINEMENT shipped first | Scheduling | Low — defer 2 of 4 new benchmarks to v0.14.0 if M-EFFECT-REFINEMENT slips |

---

## Design Freeze (check before implementation)

- [ ] Tier names finalised (smoke/core/stretch/vision)
- [ ] Retirement list approved (23 benchmarks → smoke-only)
- [ ] New benchmark list approved (4 benchmarks)
- [ ] Headline metric = core tier rate
- [ ] Backward compat: missing `tier:` defaults to `core`

---

## Deferred Decisions (sprint executor has latitude)

- Exact UI of the dashboard tier toggle
- Whether smoke runs pre-commit or pre-push (both are fine)
- Sort order within each tier in the dashboard
- Whether the contract_* family all goes in core or some go stretch

---

## Non-Goals

- **Not** auto-promoting/demoting benchmarks based on pass rate. The tier is editorial — changing it is a design decision, not a statistical one.
- **Not** rewriting existing benchmarks. Tiering is pure classification.
- **Not** adding per-category tagging (that's M-EVAL-CATEGORY-ANALYSIS, which is complementary).
- **Not** touching the agent vs standard mode split. Tiers are orthogonal to mode.

---

## Success Criteria

- [ ] All 47 existing benchmarks have a `tier:` tag
- [ ] 4 new benchmarks land with YAML + Python reference + AILANG reference
- [ ] `ailang eval-suite --tier smoke` runs in <90s across 8 models
- [ ] Core tier best/worst spread ≥ 15pp on v0.13.0 baseline
- [ ] Dashboard shows per-tier breakdown
- [ ] Release notes use core tier as primary metric
- [ ] All tests passing (`make ci`)
- [ ] Documentation updated (`docs/docs/guides/evaluation/` + `docs/LIMITATIONS.md` if relevant)

---

## Timeline

Week 1, days 1–2: Phase 1 + Phase 2 (tagging + filtering). Safe to land standalone.
Week 1, day 3: Phase 3 (reporting). Dashboard update.
Week 2, days 1–2: Phase 4 (new benchmarks). Vision-tier benchmarks depend on M-EFFECT-REFINEMENT — defer if not shipped.

Realistic total: 3 days of focused work, schedule for 5 to account for dashboard + docs.

---

## Open Questions

1. Should smoke tier run on every commit (slow CI) or every push (faster local loop)? Recommendation: pre-push, opt-in pre-commit.
2. Should the dashboard show historical tier rates for versions before v0.13.0? Requires back-tagging — propose: no, start fresh at v0.13.0 and note the discontinuity.
3. Does the `contract_*` family need its own tier? Currently scattered across core and stretch. Recommendation: keep in core unless the family grows past 10 benchmarks.
