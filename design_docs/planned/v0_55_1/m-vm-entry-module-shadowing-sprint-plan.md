# Sprint Plan: M-VM-ENTRY-MODULE-SHADOWING

## Summary

Make bytecode entry dispatch prefer the requested module's function, preserve exact and unique qualified-suffix resolution, and reject ambiguous dependency matches before VM execution.

**Design:** [Approved design](m-vm-entry-module-shadowing.md), task task-582a4a2b.
**Target:** v0.55.1. **Duration:** 3 working days, approximately 8 hours including 25% contingency. **Estimate:** 400 added/changed LOC (100 implementation, 270 tests, 30 documentation). **Risk:** medium.
**Status:** planned; implementation has not started. Coordinator approval/merge routes the artifacts to sprint-executor; this planning run does not start execution.

## Current Status and Evidence

The clean checkout is v0.55.0. Code inspection confirms `FindEntryProto` still returns the first exact/dot/underscore suffix match, and all four production/test helper callers remain unmigrated. `RootModuleID` is not exposed. The approved design includes a systemic audit of batch, policy, REPL, select_best and named-test dispatch; compiler lowering and image naming remain outside scope.

The seven-day velocity script found only design commit f00663ef and no usable LOC measurements in this shallow history. An empirical LOC/day rate cannot be calculated. Planning capacity is provisionally 134 LOC/day, based on the design's bounded scope rather than a claimed historical average. Four 60–90 minute milestones total 6 hours, with 2 hours contingency across three days. If the incident fixture exceeds that budget, commit completed work and report the remaining fixture work explicitly.

No Go toolchain is available here; no baseline test coverage or runtime reproduction was measured. The installed CLI warns it may be stale. Executor must use a source-built binary and record actual results.

## Rulings and Scope

The approved High-Impact Decisions are rulings: root-module exact match, then exact name, then unique dot suffix; ambiguity errors enumerate candidates deterministically; underscore suffix matching is removed. Preserve unique dependency matching. Add `pipeline.Result.RootModuleID` from the existing canonical root; do not change compiler names, image order, or evaluator dispatch. No human-decision gates remain open. Test placement and wrapper removal are delegated implementation choices in the design, not approval gates.

CLI entrypoint validation consults root exports before VM dispatch and evaluates module initialization. Therefore a root without the requested export may never reach resolver ambiguity through the CLI. Test ambiguity through the resolver and a focused VM dispatch harness, asserting no entry invocation; do not broaden evaluator export semantics or claim no module initialization effects. Keep CLI coverage for the actual two-main incident. This refines the test placement, preserving the approved resolution semantics.

## Registry Reuse Audit

Executed `ailang pkg search bytecode` and `ailang pkg search 'entry resolution'`. The first returned sunholo/explainer@0.1.1; `pkg info` and `pkg docs` confirm it provides narration/audio processing, not compiler entry dispatch. The second returned no packages. M1–M4 each choose **none**: this is internal Go pipeline/runner correctness, regression coverage and documentation; an AILANG package cannot replace these internals. No dependency is added.

## Milestones

### M1: Root identity and resolver policy (~160 LOC)

**Budget:** 1.5 hours, Day 1. **Dependencies:** none. **Estimate:** 70 implementation + 90 tests.

Modify `internal/pipeline/pipeline.go`, `pipeline_module_phases.go`, and `internal/runner/vm.go`; add `internal/runner/vm_entry_test.go` and focused root-ID tests in a new `internal/pipeline/pipeline_root_module_test.go`. Implement `ResolveEntryProto` alongside the old function to keep this intermediate commit buildable. Source root identity from `st.rootCanonical`; retain empty identity for non-module runs.

- [ ] Tests pin RootModuleID to the actual entry key in Result.Modules, including a package/relative-import root; non-module results remain empty.
- [ ] Resolver tests cover root precedence with dependency first, exact bare and fully qualified names, unique suffix, empty root, not found, underscore rejection and ambiguity candidate order.
- [ ] Reorder prototypes and assert successful root resolution is unchanged; exact bare matches beat earlier suffix matches.
- [ ] Focused pipeline/runner tests pass and milestone is committed.

Risk: different canonical IDs for roots and imports. Assert identity against the loaded Modules key rather than reconstructing it in tests.

### M2: Migrate dispatch callers (~80 LOC)

**Budget:** 1.5 hours, Day 1. **Dependencies:** M1. **Estimate:** 30 implementation + 50 tests.

Update `internal/runner/vm.go`, `vm_bench_test.go`, `vm_quickwins_test.go`, and `internal/testing/bytecode_engine.go` to pass RootModuleID and propagate resolver errors before VM entry invocation. Add `internal/testing/bytecode_entry_resolution_test.go` for named-test dispatch. Audit `internal/testing/named_batch.go`; retain its intentionally single-module resolver unless sharing the new resolver preserves its semantics.

- [ ] All repository FindEntryProto callers are migrated; remove the function unless an evidenced compatibility caller requires a deprecated wrapper that uses the new error-safe policy.
- [ ] Runner ambiguity returns an error before invoking any candidate; direct resolver/harness tests cover both zero and multiple matches.
- [ ] Named-test synthesized entries and empty-root unique-suffix compatibility pass focused tests.
- [ ] `make test-changed` passes; commit milestone.

Risk: returning nil after swallowing ambiguity restores silent failure. Propagate errors and assert their candidate lists in tests.

### M3: Incident and batch regression (~130 LOC)

**Budget:** 1.5 hours, Day 2. **Dependencies:** M2. **Estimate:** 130 tests.

Add `cmd/ailang/vm_entry_shadowing_test.go`, following the existing buildAilang/runWithStdin pattern. Generate temporary package fixtures with a verifier taking a two-field record and a generator taking five fields; include a data module importing the generator helper to mirror the reported dependency graph. Assert the compiled generator prototype precedes the verifier, so the regression cannot accidentally pass because of favorable sort order. Include a compatible-record variant or distinguishable marker assertion to detect silent wrong-function success.

- [ ] The regression fails against the old resolver and passes after migration; record the dependency/root prototype names used to establish order.
- [ ] Interpreter, `--bytecode`, and `--strict-bytecode` produce identical verifier output with JSON record arguments, with no generator marker.
- [ ] Batch dispatch of the same entry produces the expected verifier result for multiple inputs.
- [ ] Ambiguity is tested at a reachable dispatch boundary per the scope note; no CLI-export behavior changes are introduced to manufacture coverage.
- [ ] Focused CLI regression tests and `make test-changed` pass; commit milestone.

Examples: temporary modules gen.ail, data.ail, and verify.ail in this Go regression test are the executable examples; no public syntax feature or standalone showcase is added.

### M4: Documentation and focused parity (~30 LOC)

**Budget:** 1.5 hours, Day 3. **Dependencies:** M3. **Estimate:** 30 documentation.

Add `changelogs/unreleased/2026-11-02-vm-entry-module-shadowing.md`, starting with `### `, and update `docs/docs/reference/limitations.md` with module-scoped entry behavior and ambiguity handling. Reconcile the design's completion checklist with actual evidence, retaining the downstream acceptance step as pending.

- [ ] Reference text states root/exact/unique-suffix order and removal of underscore matching without promising CLI imported-export support beyond existing validation.
- [ ] Run focused compiler multimodule tests, named-test tests, and the existing strict-VM parity tests relevant to runner changes.
- [ ] Run the design's multi_module/main, intra_package_imports/service (entry greet with package-dir), and imports symbol_alias/module_alias/combined_alias examples under strict VM using their existing documented arguments/capabilities; compare expected output.
- [ ] `make test-changed`, `make lint`, `make check-file-sizes`, and `make check-changelog` pass; run `make verify-examples` only if tracked examples change. Commit milestone.
- [ ] Record actual commands/results and remaining downstream verification; no full-suite requirement on the executor.

## Syntax and Example Checklist

AILANG prompt version loaded: v0.16.8 (`ailang prompt` and `ailang prompt --version-active`). The installed prompt version is older than the source version; executor reloads the prompt from its source-built CLI before creating fixture strings and checks temporary `.ail` modules before running them.

| Temporary regression modules | Decision |
|---|---|
| contracts | skip: compiler dispatch regression, no new domain contract; Go assertions pin selected function and output |
| effects | include: main with named `! {IO}` for distinguishable output; pure generator/data helpers use `! {}` |
| inline tests | skip: CLI JSON arguments and byte-for-byte engine assertions are the regression oracle; synthesized named-test dispatch is covered separately in M2 |

## Validation, Size and Resumption

Use focused tests per milestone and `make test-changed` (changed packages plus test-core); full `make test` belongs to CI, superseding the design's executor full-suite instruction. Source files currently measure 284 lines (vm.go), 199 (pipeline.go), 466 (pipeline_module_phases.go), and 215 (bytecode_engine.go), below the 800-line limit. New tests go in the named companion files above; avoid inflating existing large CLI test files. No architectural boundaries change.

Progress state: `.ailang/state/sprints/sprint_M-VM-ENTRY-MODULE-SHADOWING.json`. Read plan and JSON at each executor session; commit after each milestone and record test evidence before marking passes. Keep unresolved tests visibly incomplete. No production work was performed during planning.

## Dependencies and Downstream Acceptance

Executor needs Go, a source-built CLI and normal test dependencies. The game clone and snapshot are unnecessary for repository regression tests. After landing, the reporter must bump the game's binary and run the original whole-data exoplanets-check plus native5 parity tests on the committed snapshot. This external acceptance remains pending until its result arrives; do not assert the full reported workload passed from a synthetic fixture. Do not rename downstream functions or introduce an interpreter production fallback.
