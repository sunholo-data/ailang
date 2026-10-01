# Sprint Plan: M-VM-VAR-PATTERN-DEFAULT-ARM

## Summary

Restore variable-pattern binding, source arm order, and guards across the strict bytecode VM and both Go emitters. Implement the approved [design](m-vm-var-pattern-default-arm.md) without changing syntax, Statement IR, or the bytecode compiler.

**Status:** Ready for coordinator review; implementation not started.
**Target:** v0.51.2 (retain the approved design target; current std/VERSION is v0.51.0).
**Duration:** 2 working days, approximately 12 hours including 25% contingency.
**Estimated total:** 440 LOC: 100 implementation, 310 tests/fixtures, 30 documentation.
**Risk:** Medium: semantic routing, duplicated defaults, and interaction with the nested-pattern sibling design.
**Dependencies:** None external. M2 follows M1; M3 follows M1 and M2.

## Current Status and Estimate Basis

Read-only inspection confirms the approved design still matches this checkout: `allConstructorPatterns` accepts catch-all arms in any position, constructor defaults omit bindings, and `lowerPatternBindings` lacks a top-level constructor case. The v1 generator and emitgo suppression changes remain required. Existing bytecode and codegen golden harnesses provide reusable infrastructure.

The seven-day velocity script found only the design-document commit and no implementation LOC metrics. Its historical changelog samples do not establish current velocity. The 220 LOC/day planning budget is an estimate, not measured throughput: the design's localized 1.25-day phases plus integration and contingency justify two working days. No coverage measurement or implementation tests were run during planning.

The design contains systemic analysis of the evaluator, lowering, bytecode compiler, v1 generator, and v2 emitter, backed by its 27-row verification log. External quorum reviewers were unavailable; this plan preserves that disclosure and relies on the user's explicit approval of the previous work, without claiming quorum clearance.

## Registry Reuse Audit

| Milestone | Search / inspection | Decision | Reason |
|---|---|---|---|
| M1 | Not applicable: compiler-internal lowering; inspected existing binding and switch machinery | none | Registry packages cannot repair Core-to-Statement IR semantics; reuse internal helpers |
| M2 | Not applicable: compiler-internal Go emission; inspected existing v1 suppression and if-chain helpers | none | Reuse emitter infrastructure; no package-like capability or dependency added |
| M3 | Not applicable: in-repository parity tests and language documentation | none | Extend existing golden harnesses rather than introducing a package or runner |

No registry search is required because no milestone introduces a package-like capability.

## Milestones

### M1: Sound lowering and binding (~230 LOC)

**Duration:** Day 1, 5 hours. **Dependencies:** None.
**Estimate:** 75 implementation + 155 tests/fixtures.

**Files:** `internal/gen/lower/match.go`, `internal/gen/lower/lower_match_test.go`; prepare shared `tests/golden/codegen/var_default.ail` example fixture in M3.

Restrict the constructor-switch fast path to constructor arms plus at most one final, unguarded variable/wildcard arm. Prepend existing pattern-binding statements to the default before flattening its body, including copies in literal-guard failure branches. Add depth-1 constructor bindings using positional `_j` fields for variable arguments; skip wildcard arguments. For routed literal/nested argument patterns, use the existing explicit panic-to-EvalOnly channel unless the sibling's recursive lowering has already landed. Inspect sibling status before editing and preserve any generalized implementation.

- [ ] Final named defaults bind the scrutinee before body evaluation, including duplicated guard-failure defaults.
- [ ] Non-final and guarded variable/wildcard arms produce ordered if-chains; multiple catch-alls cannot use the switch fast path.
- [ ] A guard can reference its variable binding; false guards continue to later arms.
- [ ] Top-level constructor arguments in routed if-chains bind through `_j` field access; unsupported literal/nested shapes report a precise lowering reason rather than silently mis-binding.
- [ ] Final wildcard and underscore defaults emit no binding; existing constructor-with-wildcard and literal-guard tests pass.

**Risk:** A routing fix exposes missing constructor binding support. Mitigate by landing routing and depth-1 bindings together and testing both guard outcomes.

### M2: Go emitter parity (~100 LOC)

**Duration:** Day 1 end / Day 2 start, 2 hours. **Dependencies:** M1.
**Estimate:** 25 implementation + 75 tests.

**Files:** `internal/gen/golang/codegen_match.go`, `internal/gen/golang/codegen_match_patterns.go`, `internal/gen/golang/codegen_match_test.go`, `internal/gen/emitgo/funcs.go`, `internal/gen/emitgo/emitter_test.go`. Shared runnable example: `tests/golden/codegen/var_default.ail` in M3.

Bind named variable defaults to `_scrutinee` in the v1 value-switch using existing naming and unused-variable helpers. Route non-final variable/wildcard arms through the existing v1 if-else chain; retain guard routing. Add unused-variable suppression after emitgo VarDecl emission and verify it for names both used and unused.

- [ ] V1 string/value-switch defaults bind their variable and generate Go that builds.
- [ ] V1 non-final variable/wildcard arms preserve source order; guarded catch-alls retain guard behavior.
- [ ] V2 default bindings inherited from M1 emit valid Go, including bound-but-unused variables.
- [ ] Focused golang and emitgo tests pass, including existing naming/scoping and declaration tests.

**Risk:** Global VarDecl suppression changes generated text. Update only affected expectations and require build/runtime verification.

### M3: Differential regression gate and documentation (~110 LOC)

**Duration:** Day 2, 3 hours plus 2 hours contingency. **Dependencies:** M1, M2.
**Estimate:** 80 tests/fixtures + 30 documentation.

**Files:** `tests/golden/codegen/var_default.ail`, `tests/golden/codegen/golden_test.go`, `tests/golden/bytecode/golden_test.go`, `examples/var_pattern_default.ail`, `docs/docs/reference/language-syntax.md`, and `changelogs/unreleased/2026-10-01-vm-var-pattern-default-arm.md` following `changelogs/unreleased/README.md`.

Use the codegen fixture directory for the shared fixture: the existing bytecode harness loads those files. Add an explicit v2 build/run differential subtest using existing helpers rather than inventing a separate v2 golden directory. Keep strict checks free of evaluator fallback and xfail allowances for supported cases. Read `ailang prompt` before writing any AILANG and check every new fixture/example.

- [ ] Reported cancel/name repro returns `committed` in evaluator, strict VM, v1 Go, and v2 Go.
- [ ] Differential cases cover first/middle catch-all order, guarded variable and wildcard arms (true/false), depth-1 constructor bindings, string defaults, used/unused bindings, and final wildcards.
- [ ] Literal constructor guard failure followed by a named default executes its binding correctly; all design-listed regression fixtures remain green.
- [ ] Routed literal/nested constructor subpatterns explicitly reject strict execution with the lowering reason when still unsupported; supported cases do not become EvalOnly.
- [ ] Generated v1/v2 Go is built and executed, with outputs compared against evaluator expectations; snapshot agreement alone is insufficient.
- [ ] Baseline and post-change corpus checks show no additional EvalOnly functions or behavioral regressions in std/examples; existing unsupported functions are recorded rather than assumed absent.
- [ ] New runnable example is type-checked and runs on the evaluator and strict VM; language reference and changelog describe parity and the remaining nested-pattern boundary accurately.
- [ ] Required repository checks pass: make build, make test, make fmt, make lint, make check-boundaries.

**Risk:** Corpus modules require different entry points/capabilities. Use existing CLI/test tooling, compare only runnable inputs with identical configuration, and record excluded modules and baseline failures.

## Execution and Validation

Day 1: record relevant parity/EvalOnly baselines; implement M1 with failing regressions first; run focused lower tests; start M2. Day 2: finish emitter tests, add M3 cross-route fixtures/example, run required checks, and evaluate against the approved design. Use the newly built binary on PATH for CLI golden tests; the preinstalled binary reports a different commit from this worktree.

Focused command: `go test ./internal/gen/lower ./internal/gen/golang ./internal/gen/emitgo ./tests/golden/bytecode ./tests/golden/codegen`. Before CLI goldens, run `make build` and select that binary. No compiler-test coverage percentage is invented: require exercised branches for every new gate/binding decision and runtime parity for each supported regression shape.

## Scope and Handoff

Do not extend recursive nested/literal pattern lowering, non-tail match-expression lowering, switch IR, VM instructions, or effects. The sibling `design_docs/planned/v0_49_1/m-bytecode-nested-pattern-lowering.md` overlaps binding helpers; whichever lands second must reuse/generalize the first. Preserve explicit unsupported-shape errors until recursive support exists.

Progress: `.ailang/state/sprints/sprint_M-VM-VAR-PATTERN-DEFAULT-ARM.json`. All milestones start with `passes: null`. No original bug issue number is supplied; design PR #1477 is provenance, not an issue to close. Coordinator plan approval/merge supplies the executor handoff; this planning task does not start implementation or self-approve the plan.

## Artifact Validation

The existing JSON generator created the progress file. Its jq validation could not run because jq is absent on this runner; Python independently validated JSON syntax, real milestone criteria, populated reuse decisions, dependency references, 440-LOC totals, two-day duration, and null execution status. Generator-derived totals and short dependency references were corrected before handoff.
