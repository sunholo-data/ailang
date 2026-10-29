# Sprint Plan: M-TEST-STRIP-BRACE-LITERALS

**Status**: Superseded — fixed in v0.54.13 by b6c9f592e (#1821). `testAndPropertySkipRanges` now takes block ends from the lexer token stream (`braceTokens`/`blockEndLine` in `internal/testing/source_strip.go`), so braces inside strings and comments no longer count (option (b) below). Regression test: `internal/testing/source_strip_brace_test.go`. This plan was not executed.

## Summary
Make named-test and property stripping use parser-recorded closing-brace positions so braces in literals and comments cannot corrupt the compiled module.

**Design:** [Approved design](m-test-strip-brace-literals.md)
**Target:** v0.54.10
**Duration:** 1 day (three sequential 60–90 minute executor runs; 4 hours including buffer)
**Estimated total:** 220 LOC (25 product, 185 tests/fixtures, 10 changelog)
**Risk:** Low; AST/parser/testing integration warrants focused regression checks.
**Approval:** Design approval supplied by task-4047d865 handoff. Sprint execution follows coordinator plan approval/merge; this artifact does not self-approve execution.

## Current Status and Estimate

The checkout is clean on coordinator/task-06e5ccbb; std/VERSION is v0.54.9. TestDecl and PropertyDecl still lack End, and source_strip.go still counts raw braces. The design's systemic audit covers named batches, property batches and per-body fallback compilation; all use the shared strip helper. No implementation is complete.

The seven-day velocity script found no usable LOC metrics; shallow history exposes one documentation commit, so an observed LOC/day rate cannot be calculated. Use the design's 0.5–1 day estimate and verified small product diff, budgeting 3 hours work plus 1 hour buffer. Planning capacity is 220 LOC/day, not measured velocity. No Go executable is installed in this planning environment; executor must provision Go via the existing cloud-setup workflow before validation and build its own binary.

Touched Go files currently have 334, 166, 362, 130 and 199 lines respectively (AST declarations, parser declarations, parser tests, strip helper and strip tests). None approaches 800 lines; no companion split is needed.

## Registry Reuse Audit

M1, M2 and M3: **none**. No package-like capability is being added: AST token positions, compiler source stripping and regression fixtures are repository internals. Registry search is not applicable under the gate's package-like scope; dependencies cannot change compiler AST nodes or the harness. No registry packages inspected or dependencies introduced.

## AILANG Syntax Gate

AILANG prompt version loaded: v0.16.7 (reported by `ailang prompt --version-active`; full `ailang prompt` loaded during planning). Installed binary reports v0.54.9, so executor must reload the prompt and use the freshly built repository binary to avoid version drift. This plan writes no AILANG source itself.

For `examples/tests/brace_string.ail` and the strip fixtures:

| Requirement | Decision |
|---|---|
| Contracts | skip: regression modules exercise named tests/property bodies, not exported API contracts |
| Effects | skip: string predicates and comparisons are pure, with no effectful function signature needed |
| Inline tests | include: named boolean test blocks for both brace directions, JSON fragments, interpolation and comments; property block using a pure forall predicate |

## Proposed Milestones

### M1: Record declaration end positions (~50 LOC)

**Dependencies:** None
**Duration:** 60 minutes
**Estimate:** 10 product + 40 Go-test LOC

Add End Pos to TestDecl and PropertyDecl in internal/ast/ast_decl.go. Record p.curPos() after validating RBRACE in both constructors in internal/parser/parser_test_decl.go, leaving token advancement unchanged. Extend internal/parser/parser_topleveltests_test.go with exact End.Line and End.Column checks for single/multi-line test and property declarations. Exercise existing contextual keyword entry paths without introducing grammar changes.

**Examples:** No standalone example in this metadata-only milestone; M3 supplies the user-facing regression example.

- [ ] Both successfully parsed declaration types carry the actual closing-brace position.
- [ ] Single-line and multi-line end coordinates are asserted, including lexical braces that are not closing tokens.
- [ ] Existing parser tests pass; focused `go test ./internal/ast ./internal/parser` and `make test-changed` pass.
- [ ] Go formatting is clean and parser token advancement/grammar remain unchanged.

**Risk:** Position off by one; assert both line and column from source rather than only nonzero values.

### M2: Consume AST ranges and cover lexical contexts (~130 LOC)

**Dependencies:** M1
**Duration:** 90 minutes
**Estimate:** 15 product + 115 tests/fixtures LOC

Update internal/testing/source_strip.go to prefer End.Line for both declaration types. Preserve the approved raw-brace fallback only for End.Line == 0, explicitly documenting its compatibility purpose. Extend internal/testing/source_strip_test.go and add named_test_brace_string.ail, named_test_closebrace_string.ail and named_test_interp_brace.ail under internal/testing/testdata/strip/. Additional property/character/comment cases may use Go table-driven source strings in that test file.

Cover unmatched opening/closing braces, balanced braces, escaped quotes/JSON fragments, interpolation containing braces, valid character literals and supported line comments, plus property bodies. Check both stripTestBlocks and stripWithLineMap paths. Keep surrounding declarations and assert exact line maps so a premature or overlong range cannot hide behind successful parsing. Construct a zero-End AST directly for a dedicated legacy-fallback test.

**Examples:** Internal strip fixtures above; public example arrives in M3.

- [ ] Parser-created declarations use End.Line, bypassing raw scanning.
- [ ] Named tests and properties strip their entire bodies and closing brace in every lexical case.
- [ ] Surrounding source and original-line mapping remain correct for multiple declarations.
- [ ] Zero-End manually constructed AST retains documented fallback behavior.
- [ ] `go test ./internal/testing` and `make test-changed` pass; Go formatting is clean.

**Risk:** Source stripping operates on whole lines; preserve that granularity and avoid expanding into unrelated strip refactors.

### M3: Verify CLI regression and document fix (~40 LOC)

**Dependencies:** M2
**Duration:** 90 minutes (includes buffer)
**Estimate:** 30 example LOC + 10 changelog LOC

Create examples/tests/brace_string.ail with both failure directions, the crew_core JSON fragment, interpolation/comments, a second ordinary test demonstrating module-wide recovery, and an unbalanced-brace property. Load the current teaching prompt before writing source; type-check the module. Build the repository CLI with make build, then use that binary for checks and test runs.

Create changelogs/unreleased/2026-10-13-test-strip-brace-literals.md beginning with `### `, explaining the check/test divergence and parser-authoritative fix. Record focused validation outcomes for review. Broader diagnostic rendering and function-strip fallback redesign remain deferred.

- [ ] `ailang check examples/tests/brace_string.ail` succeeds using the freshly built CLI.
- [ ] All named tests and the property pass in interpreter, `--bytecode`, and `--bytecode --strict-bytecode` modes, with zero skipped tests or strict-bytecode fallback.
- [ ] Conflict fixtures retain counts: named_helper_purity 2/2, record_update_regression_test 3/3, stdlib/list_drop_test 3/3, yaml_bridge_test 1/1.
- [ ] `make test-changed`, `make check-file-sizes`, `make check-changelog`, `make check-boundaries`, and `make verify-examples` pass.
- [ ] New example passes formatting check; changelog fragment describes user-visible behavior without introducing new diagnostics.

**Risk:** Prebuilt CLI can falsely suggest the fix failed; explicitly run the binary produced by make build. Check interpolation/character/comment syntax with the current prompt and parser before using it as a regression fixture.

## Day 1 Sequence and Validation

Hour 1: M1 positions and parser tests. Hours 2–2.5: M2 strip change and corpus. Hours 2.5–4: M3 build, three-mode regression, conflict fixtures and changelog, with buffer for setup or fixture corrections. Dependencies are strictly sequential.

Use targeted package tests and `make test-changed` (touched packages plus core checks), followed by the size/changelog/boundary gates above. If this checkout lacks a focused make target, report that and run the existing equivalent package/core checks; do not add unrelated tooling. Do not require `make test` in executor runs: the planner skill's updated focused-gate policy supersedes the design's older full-suite instruction. CI runs the full suite. No stdlib change is planned, so verify-stdlib is unnecessary; run `make verify-examples` for the example change, alongside explicit CLI validation of the new example and named conflict fixtures.

## Success Metrics and Handoff

All lexical regression cases and existing conflict fixtures pass. End coordinates and preserved source maps have direct assertions. Product change stays near 25 LOC, with no grammar, effect or capability changes. No artificial coverage percentage is claimed; tests cover both declaration kinds, both strip entry points and the explicit legacy fallback.

Progress artifact: `.ailang/state/sprints/sprint_M-TEST-STRIP-BRACE-LITERALS.json`. All milestones start unexecuted. No GitHub issue number was supplied; do not interpret numbered verification-log references as issue IDs. Coordinator artifact markers provide the handoff; merging/approving this plan triggers sprint-executor under the existing coordinator workflow.
