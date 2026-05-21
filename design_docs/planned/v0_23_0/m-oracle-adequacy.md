# M-ORACLE-ADEQUACY: Convergence Oracles and Evidence Bundles for the Eval Harness

**Status**: Planned
**Target**: v0.23.0
**Priority**: P1 - Medium
**Estimated**: 1 week
**Dependencies**: Eval harness tier system (shipped), effect type system (shipped)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Evidence bundle is deterministic for given code + test suite |
| A2: Replayability | +1 | Bundle is a structured artifact; any run is replayable |
| A3: Effect Legibility | +1 | Effect-safety check verifies declared vs. actual effects |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Property tests are bounded; confidence score is locally computable |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-readable evidence bundle; confidence score enables automated decisions |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Confidence score exposes oracle weakness as a visible metric |
| A10: Composability | +1 | Evidence bundles compose: overall confidence = min(component confidences) |
| A11: Structured Failure | +1 | Typed failure: `OracleFailure { kind, coverage, untested_regions }` |
| A12: System Boundary | +1 | Explicit boundary between "verified" and "unverified" regions |

**Net Score: +10** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Property tests use seeded randomness; bundles are reproducible
- [x] A3 (Effects): Effect-safety check is a core component of the bundle
- [x] A4 (Authority): No ambient access introduced
- [x] A7 (Machines First): Bundle is JSON; confidence score is a float, not a narrative

## Problem Statement

The AILANG eval harness uses pass/fail as its primary convergence criterion: if a benchmark's tests pass, the chain is considered successful. This binary signal has a well-known failure mode: code that exploits weak test suites passes visible tests while being semantically wrong.

**Current State:**
- Eval harness records: pass/fail, runtime, model, timestamp
- No coverage metric: a passing chain with 10% branch coverage looks identical to one with 90%
- No effect-safety check: a program that declares `!Pure` but makes file writes passes unless a test catches it
- No property-based testing for pure AILANG stdlib functions
- No confidence score: there is no machine-readable signal for "how much do we trust this result?"

**Impact:**
- "Oracle adequacy crisis" documented in SWE-Bench++ and PatchDiff: systems achieve high pass rates on visible test suites but fail on held-out tests
- AILANG's eval tier system (`core`/`stretch`) is the structural skeleton for addressing this — but the evidence layer is missing
- "Code as Agent Harness" (arXiv:2605.18747) §open-problems calls for "composable verification artifacts with explicit scope — each declaring what it verifies and what it cannot." The missing abstraction is the **evidence bundle**.

## Goals

**Primary Goal:** Attach an "evidence bundle" to every benchmark result in the eval harness, carrying test coverage, property-test coverage, effect-safety status, and a composite confidence score. Each bundle declares its scope and limitations.

**Success Metrics:**
- Every benchmark result in `eval_results/` includes an evidence bundle JSON field
- Effect-safety check catches ≥90% of programs that declare but violate effects (validated against fixture set)
- Property-based tests cover ≥20 pure stdlib functions by end of v1
- `ailang eval-* --confidence` flag filters results below a threshold
- `ailang dashboard` displays confidence scores alongside pass/fail

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Confidence score formula: min vs. weighted average of components | Min is conservative; weighted average is tunable | human | design | med |
| Property test generator: custom vs. QuickCheck port | Custom is minimal; port is richer but more work | agent | compile | med |
| Effect-safety check: static (type-checker) vs. dynamic (runtime monitor) | Static is compile-time; dynamic catches runtime violations | human | design | high |
| Coverage tool: AILANG-native vs. wrapping Go coverage | AILANG-native is cleaner; Go wrapping is faster to ship | agent | compile | med |

### Design Freeze

Before implementation begins:

- [ ] Confidence score formula confirmed (recommendation: `min` of components — conservative, no false confidence)
- [ ] Effect-safety check strategy: static type-checker extension (recommended for v1) vs. dynamic monitor

## Solution Design

### Overview

The evidence bundle is a structured artifact attached to each benchmark result:

```go
type EvidenceBundle struct {
  BenchmarkID     string
  ChainID         string
  Timestamp       time.Time

  // Components
  TestResult      TestEvidence      // pass/fail + branch coverage %
  PropertyResult  PropertyEvidence  // property tests run, passed, coverage %
  EffectSafety    EffectEvidence    // declared vs. actual effects, violations

  // Composite
  ConfidenceScore float64           // min(component scores), 0.0–1.0
  UntestedRegions []string          // code paths not covered by any check
  Limitations     []string          // explicit "this bundle does NOT verify: ..."
}
```

**Confidence Score Components:**

| Component | Score Contribution | Empty/Missing |
|-----------|-------------------|---------------|
| Test coverage ≥ 80% | 0.4 | 0.0 |
| Property tests: ≥5 passing | 0.3 | 0.0 |
| Effect-safety: no violations | 0.3 | 0.0 (assume unsafe) |
| **Total** | 1.0 | |

Final confidence = min of the three weighted component scores.

### Architecture

**Components:**

1. **CoverageCollector** (`internal/eval_harness/coverage.go`): Instruments the AILANG evaluator to record which AST nodes are reached during benchmark execution. Reports branch coverage as a float.

2. **PropertyTestRunner** (`internal/eval_harness/property.go`): A minimal property-based test framework for pure AILANG functions. For each function tagged `@property` with a generator, runs N random inputs (seeded from `AILANG_PROP_SEED`). Reports pass count and coverage.

3. **EffectSafetyChecker** (`internal/eval_harness/effect_safety.go`): Static analysis pass that compares a benchmark's declared effect row against effects inferred by the type-checker. Flags undeclared effects (definite violation) and declared-but-unused effects (possible missing implementation).

4. **EvidenceBundleWriter** (`internal/eval_harness/bundle.go`): Assembles the three component results into an `EvidenceBundle`, computes `ConfidenceScore`, populates `UntestedRegions` and `Limitations`, and serializes to JSON alongside the existing eval result.

5. **CLI integration**: `ailang eval-* --with-evidence` (default: on), `ailang eval-* --confidence-threshold 0.7` (filter low-confidence results).

### Implementation Plan

**Phase 1: EvidenceBundle schema + EffectSafetyChecker** (~2 days)
- [ ] Define `EvidenceBundle` struct in `internal/eval_harness/bundle.go`
- [ ] `internal/eval_harness/effect_safety.go` — static effect comparison
- [ ] Fixture set: 10 programs with known effect violations; verify checker catches ≥9/10
- [ ] Wire into eval harness: bundle written to `eval_results/` alongside existing JSON

**Phase 2: CoverageCollector** (~1.5 days)
- [ ] AST node instrumentation in `internal/eval/`
- [ ] `internal/eval_harness/coverage.go` — coverage aggregation
- [ ] Unit test: known-coverage program gives expected coverage %

**Phase 3: PropertyTestRunner + stdlib annotations** (~2 days)
- [ ] `internal/eval_harness/property.go` — minimal property framework
- [ ] Annotate 20 pure stdlib functions with `@property` + generator
- [ ] Run against existing benchmarks; verify property tests execute
- [ ] `AILANG_PROP_SEED` env var for reproducibility

**Phase 4: ConfidenceScore + Dashboard + CLI** (~1 day)
- [ ] Confidence formula in `bundle.go`
- [ ] `ailang dashboard` Confidence column in eval results table
- [ ] `--confidence-threshold` flag
- [ ] Documentation: `docs/docs/guides/evaluation/evidence-bundles.md`

### Files to Modify/Create

**New files:**
- `internal/eval_harness/bundle.go` — EvidenceBundle schema + writer (~150 LOC)
- `internal/eval_harness/effect_safety.go` — EffectSafetyChecker (~120 LOC)
- `internal/eval_harness/coverage.go` — CoverageCollector (~130 LOC)
- `internal/eval_harness/property.go` — PropertyTestRunner (~150 LOC)
- `internal/eval_harness/bundle_test.go` — unit + fixture tests (~200 LOC)
- `docs/docs/guides/evaluation/evidence-bundles.md` (~80 LOC)

**Modified files:**
- `internal/eval_harness/runner.go` — wire EvidenceBundleWriter into eval run (~30 LOC)
- `internal/eval/` — AST instrumentation for coverage (~50 LOC)
- `internal/dashboard/` — Confidence column in eval table (~40 LOC)
- `cmd/ailang/eval_*.go` — `--confidence-threshold` flag (~20 LOC)

## Examples

### Example 1: Evidence Bundle Output

```json
{
  "benchmark_id": "stdlib-list-map",
  "chain_id": "chain_abc123",
  "confidence_score": 0.73,
  "test_result": {
    "passed": true,
    "branch_coverage_pct": 87.5,
    "score": 0.40
  },
  "property_result": {
    "tests_run": 100,
    "tests_passed": 100,
    "functions_covered": 3,
    "score": 0.30
  },
  "effect_safety": {
    "declared_effects": ["Pure"],
    "inferred_effects": ["Pure"],
    "violations": [],
    "score": 0.30
  },
  "untested_regions": ["list.map with empty list + error callback"],
  "limitations": [
    "Property tests use seeded random inputs only; adversarial inputs not covered",
    "Effect safety is static only; dynamic runtime violations not checked in this run"
  ]
}
```

### Example 2: Effect Safety Violation Caught

```json
{
  "benchmark_id": "io-shadowed-pure",
  "confidence_score": 0.40,
  "effect_safety": {
    "declared_effects": ["Pure"],
    "inferred_effects": ["Pure", "FS"],
    "violations": [
      {
        "kind": "UNDECLARED_EFFECT",
        "effect": "FS",
        "location": "src/io_shadowed.ail:42"
      }
    ],
    "score": 0.0
  },
  "limitations": ["Effect safety violation: program declares Pure but accesses filesystem"]
}
```

### Example 3: Dashboard with Confidence

```
Eval Results — core tier — claude-sonnet-4-6 — 2026-05-21

BENCHMARK                    PASS   COVERAGE   PROP   EFFECT-SAFE   CONFIDENCE
stdlib-list-map              ✓      87.5%      ✓      ✓             0.73
stdlib-string-split          ✓      62.0%      ✓      ✓             0.47 ⚠
three-camps-eval-only        ✓      91.0%      ✓      ✓             0.82
io-shadowed-pure             ✓      78.0%      ✓      ✗ violation   0.40 ⚠
```

### Example 4: Filter by Confidence

```bash
$ ailang eval-suite --tier core --confidence-threshold 0.7
# Only shows benchmarks with confidence ≥ 0.70
# Low-confidence results written to eval_results/ but excluded from pass-rate calculation
```

## Success Criteria

- [ ] Every benchmark result in `eval_results/` includes an `evidence_bundle` JSON field
- [ ] `EffectSafetyChecker` catches ≥90% of known effect violations in fixture set
- [ ] ≥20 pure stdlib functions annotated with `@property` + generator and tested
- [ ] `ConfidenceScore` is deterministic: same code + same seed → same score
- [ ] `ailang dashboard` shows confidence column in eval results table
- [ ] `--confidence-threshold` flag correctly filters low-confidence results
- [ ] All tests passing (`make test`)
- [ ] Evidence bundles guide published

## Testing Strategy

**Unit tests:**
- `TestEffectSafetyChecker` — 10 fixture programs with known violations
- `TestCoverageCollector` — programs with known coverage %; verify accuracy
- `TestPropertyRunner_Reproducible` — same seed → same results

**Integration tests:**
- Run full `--tier core` eval with `--with-evidence`; verify every result has a bundle
- Verify `--confidence-threshold 0.7` filters expected low-confidence benchmarks

**Manual testing:**
- Review dashboard confidence column for plausibility across a full tier run
- Spot-check `untested_regions` and `limitations` for accuracy

## Deferred Decisions

- Dynamic effect monitoring (runtime violation detection) — agent may add as v2 after static checker ships
- Adversarial input generation for property tests — deferred; random is sufficient for v1
- Cross-benchmark confidence aggregation (overall suite confidence score) — agent may add to dashboard in v2

## Non-Goals

- **Formal proof of correctness** — evidence bundles describe empirical confidence, not proofs
- **100% coverage requirement** — confidence score penalizes low coverage; 100% is not mandated
- **Mutation testing** — out of scope for v1; may be added as an evidence component later

## Timeline

**Week 1** (~5 days):
- Phase 1: EvidenceBundle schema + EffectSafetyChecker (days 1–2)
- Phase 2: CoverageCollector (days 3–4)
- Phase 3: PropertyTestRunner (day 5)

**Week 2** (~2 days):
- Phase 4: ConfidenceScore + Dashboard + CLI (day 1)
- Documentation + final tests (day 2)

**Total: ~7 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Coverage instrumentation adds >10% eval runtime overhead | Med | Profile before/after; add `--no-coverage` flag to skip if too slow |
| Property generators require per-function authorship (high effort) | Med | Start with 5 core stdlib functions; expand over multiple sprints |
| EffectSafetyChecker has false positives (flags correct programs) | Med | Start with conservative rules; tune against existing benchmark corpus |

## Related Documents

**Planned (same cluster):**
- [design_docs/planned/v0_23_0/m-trace-feedback.md](design_docs/planned/v0_23_0/m-trace-feedback.md) — Doc 1: `WEAK_VALIDATOR` failure class is directly informed by low confidence scores from this doc
- [design_docs/planned/v0_23_0/m-harness-dsl.md](design_docs/planned/v0_23_0/m-harness-dsl.md) — Doc 4: workflow oracle can use confidence threshold as convergence criterion

**Implemented (may inform design):**
- [design_docs/planned/v0_13_0/m-provenance-tracing.md](design_docs/planned/v0_13_0/m-provenance-tracing.md) — provenance tracking patterns applicable to evidence bundles

## References

- **Ning et al. (2026).** Code as Agent Harness. arXiv:[2605.18747](https://arxiv.org/abs/2605.18747) — §open-problems "oracle adequacy crisis" (PatchDiff, SWE-Bench++); "composable verification artifacts with explicit scope"; QualityFlow finding (LLM-simulated execution achieves 98%+ precision predicting test outcomes — informs tier routing)
- [Design Axioms](/docs/references/axioms)
- [Evaluation Guide](../../../docs/docs/guides/evaluation/)

## Future Work

- **Mutation testing component**: add as a fourth evidence component; confidence penalizes surviving mutants
- **Cross-tier confidence rollup**: aggregate confidence across `core` + `stretch` tiers into a single per-model confidence score for the dashboard
- **Training data quality gate**: use confidence threshold as a filter for OTEL traces that feed model fine-tuning (see m-trace-feedback future work)
- **QuickCheck-style shrinking**: when a property test fails, shrink the counterexample to minimal form

---

**Document created**: 2026-05-21
**Last updated**: 2026-05-21
