# Sprint Plan: M-TEST-HARNESS-MODULE-SCOPED-ENVS

## Summary

Restore module-scoped private function lookup in every test-harness evaluation path so same-named helpers produce the same results under `ailang test` and `ailang run`.

**Status:** Planned; design approval supplied by coordinator handoff task-c8b74e6c. Implementation has not started. Execute through sprint-executor after the coordinator's sprint approval gate, then sprint-evaluator.
**Design:** [Approved design](m-test-harness-module-scoped-envs.md)
**Duration:** 2 working days, approximately 12 hours, within one calendar week.
**Estimate:** 140 implementation LOC + 340 test LOC + 20 documentation LOC = 500 LOC.
**Risk:** Medium, concentrated in root identity, recursive closures, re-exports, and fallback lookup.
**Dependencies:** No external dependencies; M1 → M2 → M3.

## Current Status and Velocity

The working tree was clean at planning start. Current branch is `coordinator/task-1f249fbe`; the handoff's `coordinator/task-c8b74e6c` is provenance, not a branch-switch instruction.

Inspection confirms unconditional shared bare-name writes in both Let and LetRec injection, closures capturing the shared environment, and CombinedResolver Case 2 falling back to shared bare names. All four executor paths still call injectModuleBindings. Runner already resolves module identity but needs to convey its canonical ID distinctly from the executor's source-file path.

The seven-day velocity script found one visible design-document commit and no usable implementation LOC metrics. This is a shallow checkout; its history cannot establish LOC/day or completion accuracy. Changelog records v0.51.0 named-test float-literal and transitive alias fixes, but these are regression context rather than measured velocity. Budget 250 LOC/day as a planning assumption, with verification time dominating the small implementation. The approved design's 12-hour estimate is retained; its inconsistent two-day versus one-week wording is resolved as two working days within a week.

`std/VERSION` is v0.51.0, beyond the design's v0.50.2 target. Keep artifact locations for coordinator continuity; release assignment must be reconciled at execution. Add the change to Unreleased in `changelogs/v0.32-current.md` unless a maintained v0.50.2 backport is explicitly selected. Do not rewrite a released entry or bump a version as part of this sprint.

## Registry Reuse Audit

Planning searches `ailang pkg search 'test harness'` and `ailang pkg search 'module'` both exited successfully with no packages found. The installed CLI warned that it may be stale; no registry candidates were available to inspect with pkg info/docs. This is an internal Go evaluator-environment correction, not a package capability. Every milestone therefore records **none**: reuse the existing eval.Environment, RefCell, module identity, resolver, and regression tooling; add no registry dependencies.

## Proposed Milestones

### M1: Module environments and root identity (~230 LOC)

**Estimate:** 90 implementation + 140 tests LOC; 4 hours on Day 1.
**Dependencies:** None.
**Files:** `internal/testing/executor.go`, `executor_helpers.go`, `runner.go`, new `executor_scoping_test.go`.
**Example fixtures:** Synthetic Core modules in executor_scoping_test.go exercising colliding Let functions, LetRec groups, and root bindings; no new public examples required for this bug fix.

Thread canonical root module ID from ResolveModuleIdentity, independently of the source path. Allocate one child environment per loaded module for each injection and capture it in Let closures. Preserve qualified entries in the shared environment. Parent each recursive group environment under its module environment, retain IndirectValue cells, and bind module-local functions there. Mirror root bare bindings into the shared environment conservatively for every harness path. Refresh module environments together with module-map/injection lifecycle.

**Acceptance criteria:**
- [ ] Two synthetic modules with same-named Let helpers each invoke their own helper in both sorted module orders.
- [ ] LetRec self/mutual recursion and calls between separate recursive groups retain module-local resolution.
- [ ] Only the canonical root module mirrors ordinary function bare names into the shared environment.
- [ ] Declared and module-less root identities work; missing identity does not trigger guessed or arbitrary root selection.
- [ ] Existing imported-helper and root harness tests remain green.

**Risk:** Child environments inherit root bare names from their shared parent. Test a qualified fallback missing its module binding against a same-named root binding; owning-module fallback must not accept an unrelated inherited value. Valid intra-module calls must remain local without modifying production eval.Environment.

### M2: Re-exports and owning-module resolver fallback (~150 LOC)

**Estimate:** 50 implementation + 100 tests LOC; 3 hours, Day 1 finish and Day 2 start.
**Dependencies:** M1.
**Files:** `internal/testing/executor_helpers.go`, `executor.go`, `executor_scoping_test.go`.
**Example fixtures:** Synthetic re-export and resolver cases in executor_scoping_test.go.

Store the owning module ID with each deferred VarGlobal. Resolve after function injection; populate owning module environments and preserve shared writes required by the approved compatibility design. Supply the same module-environment map to each CombinedResolver instance, including the resolver used during deferred evaluation. Prefer qualified keys; restrict Case 2 fallback to the requested module's bindings. Preserve builtin and ADT resolution. Test deferred write collisions explicitly because shared re-export aliases can otherwise reintroduce a leak. Retain existing deferral order rather than adding a new dependency algorithm.

**Acceptance criteria:**
- [ ] Qualified exports and aliased imports resolve exactly as before.
- [ ] Missing qualified keys fall back to the owning module's evaluated binding, never a different module or inherited root binding.
- [ ] Missing non-root bare names fail loudly using existing errors; no new diagnostic codes.
- [ ] Re-export consumers continue to work, including a same-named private helper in another module.
- [ ] Builtins and imported/root ADT constructors retain existing behavior.

**Risk:** Shared deferred alias compatibility writes need regression evidence. If those writes defeat module isolation in a valid program, report the concrete conflict with the approved design before expanding the approach.

### M3: Full harness regression matrix and documentation (~120 LOC)

**Estimate:** 100 tests + 20 documentation LOC; 5 hours on Day 2, including verification and buffer.
**Dependencies:** M1, M2.
**Files:** `internal/testing/executor_scoping_test.go` and/or `executor_regression_test.go`; `changelogs/v0.32-current.md`.
**Example fixtures:** Temporary package manifest, a.ail, b.ail, t_test.ail and main.ail created by integration tests; rename a→z in a second case. Inline/cluster and requires/ensures fixtures cover the same private-helper collision. Read `ailang prompt` before writing any embedded AILANG source and check fixture syntax before interpreting failures.

Run named-test integration through RunTestsFromFile, inline-test and cluster paths, and contract harnesses. Include a functionless named-test module and a module-less root case. CLI verification uses a freshly built binary; the installed binary's staleness warning makes it unsuitable as final proof. Test the reported consumer without its rename workaround if available, but absence is not a blocker.

**Acceptance criteria:**
- [ ] Both sort-order variants pass single-file and package named-test execution.
- [ ] Colliding helpers pass inline, cluster, and requires/ensures evaluation paths.
- [ ] Reproducer main returns true under interpreter and bytecode execution.
- [ ] TestAliasImportCollision, TestADTConstructorFromImportedModule, TestADTConstructorInCluster, TestClusterEvalWithImportedHelper, and TestFunctionlessNamedTestsResolveStdlibBuiltins pass.
- [ ] Full internal/testing suite, make test-core, make test, and make lint pass; all changed Go files are gofmt-clean.
- [ ] Existing CLI fixtures tests/record_update_regression_test.ail, tests/m_rt1_imported_constructor_pattern_test.ail, and std/trace_test.ail pass.
- [ ] Changelog cites the design and regression coverage; executable changes stay within internal/testing.

**Risk:** Full-suite infrastructure failures may exceed the estimate. Record baseline failures separately with concrete evidence, repair sprint-caused failures, and never mark unverified criteria passing.

## Daily Schedule

| Day | Work | Hours |
| --- | --- | --- |
| 1 | Establish focused regression failures; module environments and root identity; start re-export/fallback work | 6 |
| 2 | Finish fallback/re-exports; all-path integration coverage; CLI parity; full checks and changelog | 6 |

## Validation and Success Metrics

Before implementation, establish `go test ./internal/testing` baseline and record coverage with `go test ./internal/testing -coverprofile=<temporary-path>`. Coverage is not measured in this planning stage; require no decline from the recorded package baseline and direct assertions for every modified resolution branch, rather than inventing a numeric target.

During development run focused scoping tests, then `go test ./internal/testing`, `make test-core`, `make test`, and `make lint` once the final changes are in place. Use make build and the built bin/ailang for the CLI matrix; consult its help for exact flags. Run make check-boundaries if implementation becomes cross-cutting. Scope audit must show Go implementation/tests only in internal/testing, with planning artifacts and changelog as documentation exceptions. The design's literal “no files outside internal/testing” is interpreted as executable-code scope, since its own success criteria require a changelog update.

Completion requires all acceptance criteria, 500 estimated LOC across three milestones, verified temporary examples in both module orders, and no production linker/runtime changes. Do not add eval_projects fixtures, unify with link.Resolver, suppress MOD010, or repair unrelated ADT map ordering in this sprint.

## Execution Handoff

Sprint ID: `M-TEST-HARNESS-MODULE-SCOPED-ENVS`.
Progress: `.ailang/state/sprints/sprint_M-TEST-HARNESS-MODULE-SCOPED-ENVS.json`.
No GitHub issue number was supplied; github_issues stays empty rather than guessing from unrelated history. The coordinator consumes the final artifact markers to route sprint approval and execution. No executor was started by this planning task. After authorized execution, use sprint-evaluator against the approved design and this plan.
