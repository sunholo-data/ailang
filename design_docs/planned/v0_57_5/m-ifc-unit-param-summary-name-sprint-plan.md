# Sprint Plan: M-IFC-UNIT-PARAM-SUMMARY-NAME

## Summary
Restore check/run acceptance of unit-pattern parameters by synthesizing positional names at the IFC summary mint site, preserving strict decode validation and module-wide analysis.

- Design: [approved design](m-ifc-unit-param-summary-name.md)
- Duration: 1 day, approximately 5 hours including 25% contingency.
- Estimated total: 160 LOC (10 implementation, 130 tests, 20 release notes).
- Risk: low implementation scope; security-sensitive invariants require focused regression coverage.
- Status: planned; implementation awaits the coordinator's sprint-plan approval gate.

## Current Status and Rulings

The current `AnalyzeModuleIFC` still copies `p.name` verbatim and reports every validation failure as an export. The design audits both summary producers, positional consumers, codec corruption checks, hash projection, wildcard cases, and local-callee fixed-point dependencies; this is a systemic producer fix rather than a validator relaxation.

The coordinator handoff explicitly states that previous work is approved. Treat that as approval of the recommended D2 choice in the design's High-Impact Decisions / Design Freeze: retain validation for all functions; use `function` for private functions and `export` for exports. D1 and D3 remain rulings: synthesize only at the mint site using `arg%d`. No human-decision gates remain inside execution; sprint-plan approval still precedes execution. Do not alter parser names, flow-analysis environments, codecs, schema, or the empty-name sentinel.

Velocity analysis ran for seven days, but this shallow checkout has only one merge commit and no usable LOC/day history. The script's changelog matches were historical, not measured recent velocity. The 160 LOC/day capacity is a task estimate, not an observed rate. Two milestones fit within separate 60–90 minute implementation runs, with remaining time for validation and contingency.

## Registry Reuse Audit

M1: `none`; M2: `none`. No package-like capability is being introduced, so registry searches and package inspection are not applicable. An AILANG package cannot replace compiler-internal Go summary construction or its compile-path regression tests. Reuse `IFCSummaryFromScheme`'s existing naming convention, existing codec tests and pipeline test helpers. No new dependency.

## Syntax and Examples

AILANG prompt version loaded: v0.16.11 (`ailang prompt` loaded; `ailang prompt --version-active` reported this version). The repository version is v0.57.4; the installed CLI's teaching prompt is older. Build the checkout binary for all executor validation; validate every fixture with that binary rather than treating the installed binary as authoritative.

No new showcase/demo module is required for this bug fix. Put source snippets in Go regression tests and temporary CLI fixtures, preserving the reported syntax.

| Fixture | Contracts | Effects | Inline tests |
|---|---|---|---|
| Original private-function repro and exported variant | skip: compiler acceptance regression, no new contract behavior | include: main has `! {IO}` as in the report | skip: Go compile/run regression assertions cover acceptance |
| Mixed, multiple anonymous, wildcard and named-collision cases | skip: exercise summary metadata only | skip: pure fixture functions need no effect row | skip: table-driven Go assertions inspect names and arity |

## Proposed Milestones

### M1: Canonical names and invariant tests (~100 LOC)

**Dependencies:** none. **Time:** 60–90 minutes. **Estimate:** 10 implementation + 90 test LOC.

Update `internal/types/ifc_summary.go` to index parameters, synthesize `arg<i>` only for an empty surface name, and select the accurate validation error prefix by `fn.IsExport`. Add table-driven tests in new companion `internal/types/ifc_summary_unit_param_test.go`; keep existing validation tests unchanged. Existing producer and test files are 296 and 359 lines, comfortably below 800; use the companion to keep the case matrix isolated.

- [ ] Private and exported unit params produce non-empty `arg0`; mixed named/unit params produce the correct positional `arg1`.
- [ ] Multiple anonymous params, wildcard/unit combinations, typed wildcard, and a named `arg0` adjacent to an anonymous param preserve arity, existing names and positional metadata.
- [ ] Repeat analysis gives identical summaries; synthesized names survive gob encode/decode and Validate.
- [ ] Existing empty-name corruption rejection, shadowing/local-summary coverage and IFC enforcement tests pass unchanged.
- [ ] Synthesis does not modify `buildIFCSig`, environment keys, or fixed-point behavior; diagnostics retain export/private scope accurately.

Risk: moving synthesis earlier changes flow-analysis bindings. Mitigate by keeping it inside the summary construction loop and reviewing the diff explicitly for scope.

### M2: Compile-path regression and release record (~60 LOC)

**Dependencies:** M1. **Time:** 60–90 minutes plus validation. **Estimate:** 40 test + 20 documentation LOC.

Create `internal/pipeline/ifc_unit_param_test.go` using existing module compilation helpers, ensuring it reaches interface construction rather than only the parser or AST analyzer. If helpers dictate an existing test file, keep additions in this companion. Assert private functions are analyzed but absent from persisted exports, while exported unit-param summaries carry the synthesized name. Add `changelogs/unreleased/2026-11-06-ifc-unit-param-summary-name.md`, opening with `### `; do not edit the historical consolidated changelog proposed by the older design instructions.

- [ ] Original private repro and exported variant compile through the module pipeline; persisted interface export membership is correct.
- [ ] Locally built checkout binary passes check and run on original/exported variants; save before/after exit status and diagnostics in the completion report.
- [ ] `make test-changed TEST_CHANGED_BASE=5d7b2efe` passes, including touched packages and test-core; run after implementation commits so diff-based selection sees all changes.
- [ ] Targeted summary, codec, hash projection and enforcement regressions pass; formatting, `make check-file-sizes`, `make check-changelog` and `make check-boundaries` pass.
- [ ] Compare baseline/fixed check output and exit status for existing `examples/` and `std/` modules using the same fixture set, explicitly including the three design-cited secrets fixtures; report actual module count and expected failures without claiming the historical 530 count.
- [ ] Release fragment records restored unit-parameter acceptance and accurate diagnostic scope; inspect LIMITATIONS for a conflicting statement and amend only if needed.

Risk: a builder-only test misses the original pipeline failure. Mitigate with a pipeline entry-point test and checkout-built CLI check/run evidence. Baseline fixtures may intentionally fail IFC checks; preserve their output and status rather than requiring all to succeed.

## Day 1 Schedule

1. Capture baseline CLI failure and existing fixture outputs; implement M1 with regression tests (90 minutes).
2. Implement M2 pipeline regression and release fragment (90 minutes).
3. Run focused gates, compare baseline/fixed fixture outputs, record evidence and contingency (2 hours).

## Validation and Success Metrics

Acceptance requires all new regression cases, unchanged malformed-summary rejection, preserved named-parameter summaries, correct persisted export membership, and no IFC fixture output/status drift. No global coverage percentage is asserted for a small targeted fix; coverage must exercise synthesis, mixed positioning and codec persistence. Use `make test-changed` and the targeted gates above; the full test suite belongs to CI. No new imports are anticipated. No external dependency or unresolved design decision blocks planning.

## Execution Handoff

Progress: `.ailang/state/sprints/sprint_M-IFC-UNIT-PARAM-SUMMARY-NAME.json`. All milestones start with `passes: null`. The coordinator's normal approval/merge handoff starts sprint-executor; creating these artifacts does not authorize implementation or self-approval. Original report task: task-08797c12. The design's #1719 is historical foundation context, not an issue to close for this regression; no current issue number was supplied.
