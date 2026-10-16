# Sprint Plan: M-RUN-ENTRY-NO-EXPORTS-FAIL-LOUD

**Status:** Planned; implementation awaits sprint approval through the coordinator.
**Design:** [Approved design](m-run-entry-no-exports-fail-loud.md)
**Duration:** 0.5 day (4 hours including review/buffer)
**Risk:** Low; deliberate change for private-only modules previously returning success.
**Estimate:** 180 LOC total: 20 implementation, 140 tests, 20 documentation.

## Goal and current state

Every module with zero exports must fail with exit 1 when run, identifying the module and requested entry and explaining the `export func` repair. Scripts, exported entrypoints, `check`, and inline-test execution retain their behavior.

The checkout reports std/VERSION v0.54.12; v0.54.8 is the design's historical target, not a version to restore. Commit b6c9f592 already added `internal/runner/entry_unexported.go`: a declared private entry errors, but absent entries still receive a note and exit 0. The export-count dispatch guard remains in `internal/runner/run.go`. Finish the approved systemic design rather than duplicating that partial fix. Its comment deliberately permits test-only modules via run; this sprint supersedes that policy per the approved design, while preserving `ailang test`.

The design audits single/batch/auto-caps dispatch, module-interface construction, scripts, testing and embedding callers, and other export-count uses. No parser or type-system changes are needed.

## Velocity and capacity

Ran the skill's seven-day velocity script. History contains only one visible commit and no parent diff (N/A); historical changelog LOC entries cannot establish current LOC/day. No defensible measured velocity is available. Use bottom-up estimates: two milestones of 90 minutes plus one hour integration/review buffer. Planning capacity is 360 LOC/day, an estimate rather than measured throughput.

## Registry reuse audit

`ailang pkg search entrypoint` returned no packages. Both milestones have decision `none`: dispatch and diagnostics belong in the Go runner; CLI regression coverage and repository documentation cannot be supplied by an AILANG package. No candidate requires info/docs inspection and no dependency is added.

## Milestones and day-one schedule

### M1: Complete module dispatch and entry diagnostics (~80 LOC)

**Duration:** 90 minutes. **Dependencies:** none.
**Files:** `internal/runner/run.go`, `internal/runner/entrypoint.go`, `internal/runner/entry_unexported.go`, new `internal/runner/entry_no_exports_test.go`.
**Estimate:** 20 implementation + 60 test LOC.

First capture the remaining failing case. Dispatch on the existing `isModuleFile` decision and make zero-export resolution fail with the approved module/entry/export hint. Consolidate the existing helper rather than leaving an unreachable second policy; remove it if obsolete after checking callers. Keep exported-entry auto-selection unchanged. Ensure batch errors even with zero inputs: inspect batch preconditions and validate the module before an empty loop can return success. Retain no private execution.

`run.go` is 739 lines: keep its change small and put added validation helpers in `entry_unexported.go` (or companion `entry_no_exports.go` if replacement is clearer), never grow the main file beyond 800 lines.

- [ ] Private main, missing main, custom missing entry, and a module with no functions all fail with exit 1 and module/entry/export repair anchors.
- [ ] Single execution, batch with inputs, batch with zero inputs, and caps auto cannot return success for zero-export modules.
- [ ] Existing exported-entry fallback behavior remains unchanged; no private function executes.
- [ ] Focused runner tests pass; changed Go files are gofmt clean.

**Risk:** moving dispatch may expose batch input checks or module initialization before entry validation; validate early enough to preserve the promised diagnostic and avoid executing private code.

### M2: CLI regression matrix and user documentation (~100 LOC)

**Duration:** 90 minutes. **Dependencies:** M1.
**Files:** new `cmd/ailang/run_no_exports_test.go`, `docs/docs/reference/modules.md`, new `changelogs/unreleased/2026-10-16-run-entry-no-exports-fail-loud.md`.
**Estimate:** 80 test + 20 documentation LOC.

Use the existing CLI binary test harness, with temporary `.ail` fixtures for the reporter's module and a bare expression. Assert stdout, stderr and exit status separately; use semantic message substrings rather than an exact multiline snapshot. Document the run-only export requirement and migration; changelog fragment begins with `### `.

- [ ] CLI tests cover default main, explicit main, arbitrary entry, batch, and caps auto on zero-export modules, asserting exit 1 and an export hint.
- [ ] Exported main prints its expected output; bare script with --print prints 42; missing entry with other exports retains its existing error.
- [ ] `ailang check` accepts a private-only module and `ailang test` retains success for existing inline-test fixtures.
- [ ] Modules reference and unreleased changelog explain the repair and distinguish run from check/test.
- [ ] `make test-changed`, `make check-file-sizes`, and `make check-changelog` pass.

**Risk:** examples may depend on the old successful no-op. Inspect affected references and report unrelated failures without broadening this change.

## AILANG fixtures and syntax gate

AILANG prompt version loaded: v0.16.7 (`ailang prompt` loaded; active version queried). Executor refreshes the prompt before writing embedded fixtures. No new showcase module is necessary for a runner-policy bug; reuse `examples/hello_world.ail`, `examples/inline_tests_nullary.ail`, `examples/pattern_matching_adt.ail`, and `examples/inline_tests_arithmetic.ail`.

| Fixture | Contracts | Effects | Inline tests |
|---|---|---|---|
| Temporary private-main / exported-main pair | skip: dispatch regression, no algorithmic contract | include: main() -> () ! {IO} | skip: CLI status/output assertions exercise the behavior |
| Temporary missing-entry / empty module | skip: no executable contract | skip: no executed function | skip: negative runner fixture |
| Temporary bare expression 41 + 1 | skip: expression-only script | skip: pure expression | skip: CLI asserts 42 |
| Existing inline-test examples | skip: preserve existing contracts unchanged | skip: preserve existing effect signatures | include: existing inline tests run through ailang test |

Validate generated fixture syntax with `ailang check`; missing exports are valid syntax and types.

## Validation and completion

Run focused runner and CLI tests first (new tests with a precise -run filter), then `make test-changed` for touched packages plus test-core, `make check-file-sizes`, and `make check-changelog`. Build/use the checkout CLI for manual reporter reproduction and the four existing examples above. Use repository lint tooling for touched Go files. Do not require the full `make test` suite on the executor; CI owns that broader gate, superseding the older design's full-suite instruction. No coverage percentage is invented: success is explicit coverage of each changed dispatch branch and compatibility case.

No external dependency, version bump, or unresolved design choice blocks planning. Total estimate includes integration buffer. This handoff creates planning artifacts only; the coordinator's approval/merge starts sprint-executor, followed by sprint-evaluator against the approved criteria.
