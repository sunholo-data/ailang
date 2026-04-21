# M-EVAL-CATEGORY-ANALYSIS: Per-Category Benchmark Analysis & Curation

**Status**: Implemented (v0.14.0, M-EVAL-SUITE-PREP)
**Target**: v0.11.0
**Priority**: P2
**Estimated**: 3 days (~24 hours)
**Dependencies**: None (builds on existing eval infrastructure)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Tooling change, no language semantics affected |
| A2: Replayability | +1 | Tagged benchmarks enable reproducible category analysis across versions |
| A3: Effect Legibility | 0 | No language changes |
| A4: Explicit Authority | 0 | No language changes |
| A5: Bounded Verification | +1 | Per-category breakdowns bound analysis to focused areas |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Category metadata makes benchmarks machine-queryable; drives AI-focused roadmap |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-category cost analysis reveals which task types are expensive for AI |
| A10: Composability | 0 | No language changes |
| A11: Structured Failure | +1 | Refusal detection and error categorization improve failure diagnostics |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly optimises for machine analysis

## Problem Statement

AILANG's eval harness tracks per-benchmark, per-model, per-language results — but has no way to answer *design-level* questions like "which language features give AILANG an advantage over Python?" or "which task categories are saturated and should be retired?"

**Current State:**
- 51 benchmarks with `difficulty` and `expected_gain` fields, but only 5 have `category` tags
- No per-category analysis tooling — category breakdowns require ad-hoc Python scripts
- No automated detection of eval harness artefacts (model refusals, prompt injection failures)
- No systematic way to identify saturated benchmarks (100% pass rate across all models)
- The eval-analyzer skill focuses on individual failure analysis, not design-pattern-level insights

**Evidence from v0.9.1.1 analysis (April 2026):**

Ad-hoc per-category analysis revealed actionable design insights invisible in per-benchmark data:

| Task Category | AILANG | Python | Delta | Design Implication |
|---------------|--------|--------|-------|--------------------|
| Type Safety / Contracts | 55.6% | 27.8% | **+27.8%** | AILANG's `requires`/`ensures` is a proven differentiator |
| Effects / IO | 95.0% | 75.0% | **+20.0%** | Explicit effects prevent AI IO misordering |
| Records / Data | 100% | 93.8% | +6.2% | Structural typing works well for AI |
| Recursive Algorithms | 75.0% | 95.0% | **-20.0%** | Training data gap — needs stdlib + prompt work |
| Functional Patterns | 78.3% | 82.6% | -4.3% | Slight unfamiliarity penalty |

This analysis also revealed 5 Python refusals on Opus ("Apologies, but...") inflating AILANG's apparent advantage. Without automated refusal detection, headline numbers are unreliable.

**Impact:**
- Roadmap decisions are made without category-level data
- Saturated benchmarks waste eval budget without generating signal
- Eval harness artefacts pollute results without flagging
- The eval-analyzer skill can't answer "what language features should we build next?"

## Goals

**Primary Goal:** Make per-category benchmark analysis a standard, repeatable report that drives roadmap decisions.

**Success Metrics:**
- All 51 benchmarks tagged with 1-3 category tags
- `ailang eval-matrix --by-tags` produces per-category AILANG vs Python breakdown
- Saturated benchmarks identified and flagged in reports
- Refusal/artefact detection catches known patterns automatically
- Category analysis reproducible across versions for trend tracking

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Tag taxonomy (which tags, how many) | Determines all downstream analysis grouping | human | design | high |
| Saturated benchmark policy (retire vs keep) | Affects suite size, eval cost, trend continuity | human | design | med |
| Where category analysis lives (Go code vs script) | Affects maintenance burden and integration | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Tag taxonomy approved (see proposed tags below)
- [ ] Saturated benchmark policy: retire from default suite or keep but flag?

## Solution Design

### Overview

Three components: (1) add `tags` metadata to benchmark YAMLs, (2) extend eval-matrix with per-tag grouping, (3) add benchmark curation tooling (saturation detection, refusal flagging).

### Component 1: Benchmark Tags

**Add `tags` field to `BenchmarkSpec`:**

```go
// In internal/eval_harness/spec.go
type BenchmarkSpec struct {
    // ... existing fields ...
    Tags     []string `yaml:"tags,omitempty"` // Design-pattern categories
    Category string   `yaml:"category"`       // Already exists, keep for backward compat
}
```

**Proposed tag taxonomy (12 tags):**

| Tag | Description | Example benchmarks |
|-----|-------------|--------------------|
| `adt_pattern_match` | ADT construction + pattern matching | adt_option, exhaustive_pattern_matching |
| `recursion` | Recursive algorithms, tree traversal | recursion_fibonacci, binary_tree_sum, red_black_tree |
| `effects_io` | IO, FS, environment effects | effect_composition, effect_tracking_io_fs, cli_args |
| `contracts` | requires/ensures, invariant checking | contract_bst_validate, contract_roman_numeral |
| `data_transform` | JSON, CSV, config parsing/encoding | json_encode, json_parse, csv_to_json_converter |
| `records` | Record creation, update, access | nested_records, record_update, records_book |
| `functional` | HOFs, fold, pipeline, lambda | higher_order_functions, fold_reduce, pipeline |
| `type_safety` | Type-level correctness, unification | type_safe_record_access, type_unify, float_eq |
| `string_algo` | String processing, encoding | run_length_encode, balanced_parens |
| `state_machine` | Stateful computation, threading | state_machine_elevator, explicit_state_threading |
| `algorithmic` | General algorithms, sorting, search | merge_sort, graph_bfs, expression_evaluator |
| `error_handling` | Option types, Result, error recovery | error_handling, no_runtime_crashes_option |

**Principle: benchmarks can have 1-3 tags.** `expression_evaluator` gets `[adt_pattern_match, recursion, algorithmic]`. This allows cross-cutting analysis.

### Component 2: Per-Tag Eval Matrix

**Extend `ailang eval-matrix` with `--by-tags` flag:**

```
$ ailang eval-matrix eval_results/baselines/v0.9.1.1 v0.9.1.1 --by-tags

Category Analysis (top 4 models, AILANG vs Python):

  Tag               | AILANG  | Python  | Delta   | Benchmarks
  ──────────────────|─────────|─────────|─────────|───────────
  contracts         |  55.6%  |  27.8%  | +27.8%  | 5
  effects_io        |  95.0%  |  75.0%  | +20.0%  | 5
  records           | 100.0%  |  93.8%  |  +6.2%  | 4
  data_transform    |  58.3%  |  54.2%  |  +4.2%  | 6
  algorithmic       |  94.4%  |  91.7%  |  +2.8%  | 9
  adt_pattern_match | 100.0%  | 100.0%  |  +0.0%  | 4
  functional        |  78.3%  |  82.6%  |  -4.3%  | 6
  recursion         |  75.0%  |  95.0%  | -20.0%  | 5

  Saturated (100% both langs, all models): adt_option, records_book, fizzbuzz
  Refusals detected: 5 (claude-opus-4-6 Python, benchmarks: cli_args, float_eq, ...)
```

**Implementation:** Add grouping logic to `internal/eval_analysis/matrix.go`. Load tags from benchmark YAMLs at analysis time (not stored in result JSON — tags may change between versions).

### Component 3: Benchmark Curation

**Saturation detection:**
- A benchmark is "saturated" when pass rate = 100% across all models for both languages in the latest 2 baselines
- Saturated benchmarks are flagged in reports but NOT automatically removed
- New CLI flag: `ailang eval-matrix --show-saturated`

**Refusal detection:**
- Scan `stderr` and `stdout` for known refusal patterns: "Apologies", "I cannot", "I'm sorry, but", "As an AI"
- Add `refusal_detected: bool` to analysis output
- Flag in reports, exclude from headline numbers

**"AILANG-only wins" report:**
- For each benchmark × model: identify where AILANG passes and Python fails (and vice versa)
- Aggregate across models to find consistent AILANG advantages/disadvantages
- This is the analysis that revealed the design-choice correlation

**Benchmark rotation guidance (manual, not automated):**
- When >5 benchmarks are saturated, suggest replacements in the same category but harder difficulty
- When a category has <3 benchmarks, suggest additions
- Output as recommendations, not automatic changes — humans curate the suite

### Implementation Plan

**Phase 1: Benchmark Tags** (~8 hours)
- [ ] Add `Tags []string` field to `BenchmarkSpec` in `internal/eval_harness/spec.go`
- [ ] Tag all 51 benchmarks in `benchmarks/*.yml` using proposed taxonomy
- [ ] Add tag validation: warn on unrecognised tags, require at least 1 tag
- [ ] Update `benchmarks/README.md` with tag definitions

**Phase 2: Per-Tag Analysis** (~10 hours)
- [ ] Add `LoadBenchmarkTags(dir string) map[string][]string` to `internal/eval_analysis/`
- [ ] Add `GroupByTags(results []BenchmarkResult, tags map[string][]string)` grouping logic
- [ ] Implement `--by-tags` flag in `ailang eval-matrix` CLI command
- [ ] Add refusal detection: scan stderr/stdout for known patterns, add `RefusalDetected bool` to `BenchmarkResult`
- [ ] Add "AILANG-only wins" report: per-model, per-benchmark, where one lang passes and the other fails
- [ ] Add saturation detection: flag benchmarks at 100% across latest 2 baselines

**Phase 3: Eval-Analyzer Skill Update** (~6 hours)
- [ ] Update `.claude/skills/eval-analyzer/SKILL.md` with new `--by-tags` workflow
- [ ] Add `scripts/category_analysis.sh` script wrapping the new commands
- [ ] Add `scripts/benchmark_health.sh` for saturation + refusal report
- [ ] Update `resources/jq_queries.md` with tag-aware queries
- [ ] Document benchmark rotation guidelines in `benchmarks/CURATION.md`

### Files to Modify/Create

**Modified files:**
- `internal/eval_harness/spec.go` — Add `Tags` field (~5 LOC)
- `internal/eval_analysis/matrix.go` — Add tag grouping, saturation detection (~150 LOC)
- `internal/eval_analysis/types.go` — Add `RefusalDetected` field (~5 LOC)
- `internal/eval_analysis/loader.go` — Add refusal pattern detection (~30 LOC)
- `cmd/ailang/eval_matrix.go` — Add `--by-tags`, `--show-saturated` flags (~40 LOC)
- `benchmarks/*.yml` — Add `tags` field to all 51 files (~51 × 1 LOC)
- `.claude/skills/eval-analyzer/SKILL.md` — Document new workflows (~50 LOC)

**New files:**
- `internal/eval_analysis/tags.go` — Tag loading, grouping, AILANG-only-wins logic (~200 LOC)
- `.claude/skills/eval-analyzer/scripts/category_analysis.sh` — Wrapper script (~50 LOC)
- `.claude/skills/eval-analyzer/scripts/benchmark_health.sh` — Saturation + refusal report (~50 LOC)
- `benchmarks/CURATION.md` — Benchmark rotation guidelines (~100 LOC)

## Examples

### Example 1: Per-Tag Analysis After a Release

```bash
$ ailang eval-matrix eval_results/baselines/v0.11.0 v0.11.0 --by-tags

# Shows category breakdown with deltas
# Immediately visible: "recursion gap closed from -20% to -8% after stdlib additions"
```

### Example 2: Benchmark Health Check

```bash
$ ailang eval-matrix eval_results/baselines/v0.11.0 v0.11.0 --show-saturated

Saturated benchmarks (100% all models, both langs, last 2 baselines):
  fizzbuzz          — tags: [algorithmic]        — consider: harder variant
  adt_option        — tags: [adt_pattern_match]  — consider: multi-constructor ADT
  records_book      — tags: [records]            — consider: nested record update

Recommendations:
  - Category 'contracts' has only 5 benchmarks but +27.8% delta — add more to strengthen signal
  - Category 'string_algo' has only 2 benchmarks — add 2-3 for statistical reliability
```

### Example 3: AILANG-Only Wins Report

```bash
$ ailang eval-matrix eval_results/baselines/v0.11.0 v0.11.0 --by-tags --ailang-wins

AILANG-only wins (passes where Python fails, across top 4 models):

  Consistent (3+ models):
    float_eq          [type_safety]     — 6/6 models
    numeric_modulo    [type_safety]     — 6/6 models
    json_encode       [data_transform]  — 5/6 models
    cli_args          [effects_io]      — 4/6 models

  Python-only wins (3+ models):
    type_unify        [type_safety]     — 5/6 models
    config_file_parser [data_transform] — 5/6 models
```

## Success Criteria

- [ ] All 51 benchmarks have 1-3 tags from the approved taxonomy
- [ ] `ailang eval-matrix --by-tags` produces per-category table with AILANG vs Python deltas
- [ ] Refusal detection flags known patterns ("Apologies", etc.) and excludes from headline numbers
- [ ] Saturation report identifies benchmarks at 100% across all models
- [ ] AILANG-only-wins report identifies consistent per-model advantages
- [ ] Category analysis is reproducible: running on v0.9.1.1 data matches the manual analysis from this design doc
- [ ] All tests passing
- [ ] Eval-analyzer skill updated with new workflows
- [ ] `benchmarks/CURATION.md` documents rotation guidelines

## Testing Strategy

**Unit tests:**
- `internal/eval_analysis/tags_test.go` — Tag loading, grouping, AILANG-only-wins calculation
- Test with synthetic results: known pass/fail patterns should produce expected category deltas
- Refusal detection: test against known refusal strings and edge cases

**Integration tests:**
- Run `ailang eval-matrix --by-tags` against `eval_results/baselines/v0.9.1.1/` and verify output matches manual analysis
- Verify saturation detection against known-saturated benchmarks

**Manual testing:**
- Run full `--by-tags` report and verify category grouping makes sense
- Check that multi-tagged benchmarks appear in all relevant categories
- Verify refusal-adjusted numbers match manual calculation

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact refusal pattern list — agent may extend beyond "Apologies" based on data (start conservative, add patterns as discovered)
- Output format for category analysis (table vs JSON vs both) — agent may choose based on existing eval-matrix patterns
- Whether to add `--by-difficulty` in the same sprint or defer — agent may include if time allows

## Non-Goals

**Not attempted in this feature:**
- Automated benchmark generation — creating new benchmarks is human-curated, not automated
- Teaching-to-the-test — this feature analyses results to improve the *language*, not the *prompt*. Prompt changes that merely teach benchmark-specific patterns are out of scope.
- Changing the benchmark suite — this feature adds metadata and analysis. Actual additions/removals are a separate decision.
- Gemini-specific prompt tuning — the Gemini gap is an API reliability issue, not addressable through category analysis
- Cross-version trend charts — v1 produces per-version tables; visual trend tracking is future work

## Timeline

**Day 1** (~8 hours):
- Phase 1: Add tags field, tag all 51 benchmarks, update README

**Day 2** (~10 hours):
- Phase 2: Tag loading, grouping, per-tag matrix, refusal detection, AILANG-only-wins

**Day 3** (~6 hours):
- Phase 3: Skill update, scripts, curation doc, testing, verification

**Total: ~24 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tag taxonomy doesn't capture meaningful patterns | Med | Start with 12 tags from proven manual analysis; iterate based on results |
| Multi-tagging creates confusing overlapping categories | Low | Cap at 3 tags per benchmark; primary tag listed first |
| Refusal patterns evolve across model versions | Low | Keep patterns in a constant list; easy to extend |
| Saturated benchmarks are kept too long, wasting eval budget | Med | Curation doc provides guidelines; human reviews quarterly |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_10/M-DASH.md](../../implemented/v0_3_10/M-DASH.md) — Eval dashboard, performance matrix
- [benchmarks/VISION_BENCHMARKS.md](../../../benchmarks/VISION_BENCHMARKS.md) — Vision-aligned benchmark mapping

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-eval-cross-language-benchmark.md](m-eval-cross-language-benchmark.md) — M-EVAL-XLANG: third-party benchmark comparison
- [design_docs/planned/v0_11_0/m-cloud-eval-workers.md](m-cloud-eval-workers.md) — Cloud eval workers

## References

- [Design Axioms](/docs/references/axioms)
- v0.9.1.1 baseline data: `eval_results/baselines/v0.9.1.1/`
- Analysis methodology: `docs/talk-building-a-language.md` ("The Punchline" section)
- Eval-analyzer skill: `.claude/skills/eval-analyzer/SKILL.md`
- Benchmark specs: `internal/eval_harness/spec.go`

## Future Work

- **Per-category trend charts** — Track category deltas across versions visually
- **Automated benchmark suggestions** — When a category drops below threshold, suggest new benchmarks
- **Difficulty calibration** — Use actual pass rates to recalibrate difficulty labels
- **Cost-per-category** — Which task types cost the most tokens? Inform stdlib priorities.
- **Cross-language benchmark integration** — Apply tags to M-EVAL-XLANG results for unified analysis

---

**Document created**: 2026-04-09
**Last updated**: 2026-04-09
