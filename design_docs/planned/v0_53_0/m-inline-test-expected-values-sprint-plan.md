# Sprint Plan: Inline-test expected values

**Refs #495**
**Design:** [Approved design](m-inline-test-expected-values.md)
**Status:** Ready for sprint-plan review; implementation not started
**Target:** v0.53.0
**Duration:** 2.5 working days (20 hours, including 4 hours buffer)
**Risk:** Medium

## Summary

Evaluate composite expected values through the input arm's existing AST-to-Core and harness-evaluator machinery. Share a bounded recursive grammar gate between the pipeline and runner so unsupported row expressions produce TST001/TST002 instead of false-green checks or panics. Option A is approved by the incoming design handoff.

## Current status and velocity

Read issue #495 and both comments through the GitHub REST API on 2026-10-08 (gh is unavailable). The August comment confirms ADT failures; October triage adds the list case. F1 is already fixed. The approved design's F3 live re-verification shows composed contracts verifying and genuine solver errors exiting 1; the original 213-line sketch was never supplied, so this is bounded evidence, not proof that every composed contract works.

Source inspection confirms runner.go still calls EvaluateLiteral, harness.go still has the non-error-returning converter, and the surface-AST effect-ceiling stage lives in pipeline_module_compile.go. No implementation milestone is complete. Current std/VERSION is v0.52.5; actual workspace branch is coordinator/task-771ee84a, distinct from the incoming design branch. Initial working tree was clean.

The existing analyze_velocity.sh ran for 7 days, but this shallow checkout exposes only one documentation commit and reports no usable LOC metrics. Do not treat its historical changelog snippets as measured throughput. Planning assumption: 300 changed/added LOC per implementation day, 600 total LOC over two days, plus 0.5 day (25%) integration buffer. No coverage baseline was measured during planning; verify targeted branch coverage during execution rather than invent a percentage.

## Registry reuse audit

Ran `ailang pkg search inline-test` (no candidates) and `ailang pkg search testing` (sunholo/testing_utils@0.1.1). Inspected that candidate with `pkg info` and `pkg docs`: it exports pure user-level assertions, not AST conversion, compiler diagnostics or evaluator integration. Every milestone chooses **none**: these changes belong to the Go compiler/runtime and its fixtures. No dependency or contribution is appropriate. The installed CLI warned that it may be stale; these searches support package reuse decisions only, not source-behavior verification.

## Milestones

### M1: Harden conversion and evaluate composite expected values (~230 LOC)

**Estimate:** 90 implementation/docs + 140 tests/examples LOC; 7 hours.
**Dependencies:** None.

Update internal/testing/harness.go, executor.go, runner.go and their tests. Make conversion recursively error-returning, then audit every caller with rg, including runner.go requires generation (~line 462), both harness builders, property AST wrappers and round-trip tests. Keep converter behavior needed by property wrappers separate from the stricter inline-row grammar. Add EvaluateExpectedExpr using newHarnessEvaluator and the existing source-module constructor environment; remove EvaluateLiteral after migrating all callers/tests. Preserve equalValues and inline compilation memoization.

Create internal/testing/testdata/inline_expected_values.ail during execution, after reading `ailang prompt`; cover local nullary/applied ADTs, empty/nested lists, records, tuples, scalars and mismatch cases. Risk: environment drift or lost constructor injection; verify with actual module-scoped fixtures and existing round-trip tests.

**Acceptance criteria:**

- [ ] All astExprToCore callers propagate errors, including requires generation and AST property wrappers; unsupported nodes never panic.
- [ ] EvaluateExpectedExpr replaces EvaluateLiteral using the existing module-scoped harness evaluator and constructor injection.
- [ ] Nullary/applied local ADTs, lists, nested list-of-records, tuples, records and scalar negatives compare correctly; mismatches still fail.
- [ ] Inline compile memoization adds no pipeline runs and requires/ensures regressions pass.

### M2: Share row grammar and diagnostics between check and test (~240 LOC)

**Estimate:** 120 implementation/docs + 120 tests/examples LOC; 7 hours.
**Dependencies:** M1.

Create internal/ast/test_row_expr.go and table tests; create internal/pipeline/validate_test_rows.go and tests, integrate with the surface-AST module pipeline, and register codes in internal/errors/codes.go using existing diagnostics conventions. Update runner.go to validate both arms before actual evaluation/cluster extraction. Walk nested tuples/lists/records/calls/unary operands. Default-reject unknown kinds; accept unary minus only unless a premise test justifies widening.

Create negative fixtures internal/testing/testdata/inline_rows_binary.ail and inline_rows_unsupported.ail, plus cmd/ailang CLI tests for check/test/ai-check. Test nested comparisons, arithmetic and lambda forms. Grammar tests use bound identifiers and supported calls: grammar acceptance does not prove name resolution, type correctness or call purity. Direct runner tests must cover fast-failure even when compilation is bypassed. Risk: pipeline errors may abort test compilation before report creation; assert the CLI report contract explicitly and use the existing report mechanism to preserve it.

**Acceptance criteria:**

- [ ] Recursive RowExprSupported in internal/ast accepts supported shapes and rejects every BinaryOp and unsupported nested form on both row arms.
- [ ] TST001 and TST002 are registered with row locations and offending constructs; check exits 1 and ai-check includes check.errors.
- [ ] Runner validates before evaluating any row arm and returns matching codes without process crashes; rejected-row CLI tests assert rc=1 and a test report.
- [ ] Pipeline validation reaches module check, ai-check and LSP without importing internal/testing into the language closure.
- [ ] Table-driven grammar/evaluation tests cover every AST expression kind with valid bindings for accepted identifiers/calls, plus nested rejection paths.

### M3: Verify regressions and document row limits (~130 LOC)

**Estimate:** 20 implementation/docs + 110 tests/examples LOC; 6 hours.
**Dependencies:** M1, M2.

Create examples/inline_tests_expected_values.ail; update docs/docs/reference/language-syntax.md and the current CHANGELOG section. Preserve the existing five inline examples. Add capability-gating and multi-argument tuple-input regressions; no row-arity redesign. Run focused Go tests for internal/ast, internal/testing, internal/pipeline, internal/errors and cmd/ailang first, then the required full checks. Verify LSP diagnostic propagation through its shared pipeline path (existing integration harness or attended manual check).

F3 adds no implementation task: retain the design verification log and residual float-division encoder limitation; do not close #495 solely from this sprint. Risk: broad suite/environment failures; record exact command and failure, distinguish baseline failures, and do not silently waive checks.

**Acceptance criteria:**

- [ ] New examples/inline_tests_expected_values.ail checks and all composite rows pass; negative fixtures assert TST001/TST002 on check and test.
- [ ] All five existing inline_tests examples retain outcomes; effectful inline functions retain capability gating and successful execution with --caps IO.
- [ ] make test, make lint and make check-boundaries pass; formatting is clean.
- [ ] Language syntax documentation explains supported expected shapes, excluded operators and residual name-resolution/imported-constructor limits.
- [ ] F3 design verification is carried forward as fixed/cannot-reproduce with its evidence limits; no SMT change or new issue is included.

## Day-by-day execution

- Day 1 (8 hours): M1 red fixtures, conversion caller audit, evaluator switch and focused regressions (7h); start shared grammar table (1h).
- Day 2 (8 hours): finish M2 pipeline/runner/CLI parity (6h); begin M3 examples and documentation (2h).
- Day 3 (half day, 4 hours): M3 broad checks and diagnostic/report integration buffer; resolve failures and collect evidence.

Total: 600 estimated LOC (230 + 240 + 130). Estimates include tests and examples, not only net additions; deleting EvaluateLiteral reduces net LOC.

## Validation and success metrics

Run `ailang prompt` before writing any .ail fixture, then use the newly built binary for check/test/ai-check. Verify composite fixtures check at rc=0 and test green; negative fixtures check/test at rc=1 with matching codes, row positions and reports; ai-check JSON contains the check errors. Unknown identifiers and imported constructors remain documented runtime limitations. Confirm all existing scalar examples and requires/ensures outcomes, capability enforcement, and unchanged compile memoization. Run make test, make lint, make check-boundaries, and formatting checks after targeted tests pass. No extra benchmark campaign is needed.

## Dependencies, scope and handoff

M1 precedes M2, and both precede M3 completion. No external package or Z3 dependency is required for this F2 sprint. Reuse the design's F3 evidence; request the original sketch only if F3 is reopened. Do not implement named-test-body/properties check visibility, imported constructors, arithmetic elaboration, comparator TypeName fixes, or input-row shape/arity changes. The stale multiarg design owns those arity decisions; its future checks should join this pipeline stage rather than duplicate it.

The coordinator receives the plan and JSON paths below; review/merge of this sprint-plan PR triggers the executor under the documented coordinator workflow. This task only creates planning artifacts; direct executor dispatch and implementation await that gate. All milestone passes remain null. Future PR body must contain **Refs #495** and describe F2's fixed behavior plus F3's bounded re-verification. Use Refs rather than an automatic closing keyword; create no new issue.

Suggested PR body:

> Refs #495. Plan the approved composite expected-value evaluator, shared check/test row diagnostics, and regression sweep in three milestones over 2.5 days. F3 is carried forward as fixed/cannot-reproduce based on the design's live checks; this PR contains planning artifacts only.
