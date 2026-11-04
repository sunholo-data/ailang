# Sprint Plan: M-DX-EFFECTFUL-CALLBACK-DIAGNOSTIC

**Design:** [approved design](m-dx-effectful-callback-diagnostic.md)
**Status:** Implementation complete; validation recorded below; independent evaluation pending.
**Duration:** 2 working days, approximately 7.5 hours including 25% contingency.
**Scope:** Diagnostic changes only; no change to accepted programs or effect semantics.
**Risk:** Medium: nested unification context and local/deferred message parity.

## Current status and estimate

The approved design's systemic audit covers map/filter, lambda/named arguments, open/closed rows, record rows, annotation constraints, and both solver loops. Current code still attaches collision hints without inspecting failure cause. No implementation is present; the initial working tree was clean. Source version is v0.57.0.

The velocity script was run for seven days. This shallow checkout exposes only merge 37e5b48d and no usable diff/LOC history; its changelog matches are historical, not current velocity. No measured LOC/day can responsibly be inferred. Planning capacity is an explicit assumption: 270 LOC/day, 540 total (230 implementation + 310 tests/docs), six focused hours plus 1.5 hours buffer. This resolves the design's inconsistent two/three-day estimates as two working days for this bounded change. Each milestone is a 60–90 minute executor unit; split further if discovery exceeds that bound.

## Rulings and boundaries

D1–D6 in the approved design are rulings: typed sentinel in function effect unification, typed application context, one shared failure wrapper for both solvers, injected stdlib variant lookup, preserved raw solver text, and both plain lead and note suppression. No human-decision gates remain. Do not reopen those choices. Do not claim the callback invokes println: source provenance for the inner call is explicitly out of scope.

Use companion files for new helpers. `typechecker_functions.go` is already 773 lines: keep its edit to a short helper call and put context construction in `typechecker_app_diagnostics.go` or a companion. Current sizes: inference.go 597, inference_helpers.go 514, unification_types.go 295, stdlibindex/index.go 131. Keep all touched files under repository limits.

## Registry reuse audit

Ran `ailang pkg search effect` and `ailang pkg search diagnostic`. Inspected `sunholo/ail_diag@0.1.0` with `pkg info` and `pkg docs`: it parses emitted CLI diagnostics in AILANG and cannot classify Go type constraints, supply solver metadata, or wire the compiler. Effect search returned application packages, not compiler internals. Decision for every M1–M5: **none**, no dependency or contribution; implement at existing Go compiler seams. Reuse existing stdlibindex and regression fixtures within the repository.

## Milestones

### M1: Classify function effect failures (~100 LOC)

**Estimate:** 35 implementation + 65 tests; 60 minutes. **Dependencies:** none.

Create `internal/types/effect_row_mismatch.go` and `effect_row_mismatch_test.go`; update `unification_types.go`. Wrap only function effect-row failures, preserving Error/Unwrap behavior and the old text. Carry parameter index through parameter unification; explicitly test nested function arguments so an unrelated nested mismatch cannot be presented as the top-level callback's own effects.

- [x] errors.As finds the sentinel for both open/closed and closed/closed effect failures.
- [x] Parameter position is correct; direct function-row failure retains index -1.
- [x] Error text is byte-identical to the original solver chain, including existing wrappers.
- [x] Record-row failures contain no sentinel; focused internal/types tests pass.

Risk: incorrect context after recursive unification. Use fresh errors per failure and tests for nesting.

### M2: Carry typed application context (~85 LOC)

**Estimate:** 40 implementation + 45 tests; 60 minutes. **Dependencies:** M1.

Update `inference.go`, `typechecker_functions.go`, and `typechecker_app_diagnostics.go`; extend `typechecker_app_diagnostics_test.go` or a new `typechecker_app_effect_context_test.go`. Populate optional TypeEq.App with callee/module, call span, per-argument spans and lambda flags. Add note-free path construction without parsing Path strings.

- [x] Named, global, anonymous and aliased callees retain available context.
- [x] Non-application constraints keep App nil and existing diagnostic paths unchanged.
- [x] Argument metadata is aligned with the index used by unification; unavailable/out-of-range metadata never panics or invents spans.
- [x] Existing application diagnostic and collision-hint tests pass.

Risk: growing typechecker_functions.go; move helper bodies to the companion file.

### M3: Render effect leads through both solvers (~165 LOC)

**Estimate:** 85 implementation + 80 tests; 90 minutes. **Dependencies:** M1, M2.

Create `internal/types/application_effect_diagnostic.go` and `application_effect_diagnostic_test.go`; update both TypeEq wrap sites in `inference_helpers.go`. Define the injectable EffectfulVariantSuggester with nil-hook generic wording. Sort effect labels and preserve the complete original solver detail after a note-free path.

- [x] Pure callback rejection leads with callee, parameter, declared row, actual effects and available argument span; use lambda/argument wording correctly.
- [x] Both-extra-label and anonymous cases render deterministically; avoid calling nonempty declared rows pure.
- [x] Callee-own-row mismatch suppresses the note without inventing a callback lead.
- [x] Non-effect/type/record/non-app failures retain byte-for-byte prior messages.
- [x] Local and deferred solving yield exactly equal decorated errors; old solver substrings remain present.
- [x] Nil-hook library usage provides a generic remedy without a stdlib module claim.

Risk: row orientation and missing metadata. Exercise swapped labels and defensive cases in table tests; do not change Row.String().

### M4: Discover and wire honest effectful remedies (~80 LOC)

**Estimate:** 40 implementation + 40 tests; 60 minutes. **Dependencies:** M3.

Update `internal/stdlibindex/index.go`, `index_test.go`, `cmd/ailang/diagnostics_wiring.go`; add a CLI wiring test in `diagnostics_wiring_test.go` if needed. Use the existing index to find same-module name+E exports; restore any global hooks changed by tests.

- [x] std/list map/filter/foldl find their E variants; std/option.map and std/io return no false same-module variant.
- [x] Lookup output is sorted and stable; no hardcoded stdlib export table enters internal/types.
- [x] CLI-wired map diagnostic recommends std/list.mapE; nil hook behavior from M3 remains valid.
- [x] Focused stdlibindex/CLI tests and make check-boundaries pass.

Risk: pipeline tests do not load CLI init. Test lookup/wiring at the owning layers; explicitly inject and restore hooks in pipeline tests requiring remedies.

### M5: Integration regression coverage and release note (~110 LOC)

**Estimate:** 30 implementation/docs + 80 tests; 90 minutes. **Dependencies:** M4.

Create `internal/pipeline/effectful_callback_diagnostic_test.go`; add `examples/runnable/effectful_callback_map_e.ail` after checking local directory conventions; add `changelogs/unreleased/2026-11-03-effectful-callback-diagnostic.md` opening with `### `. Keep intentionally failing repros in Go test fixtures, not runnable examples. Record final validation in the sprint state. Move the design to implemented only upon completion under executor conventions and update artifact references if moved.

- [x] Reporter map lambda, filter lambda and named println argument still fail, lead with effects and omit the collision note.
- [x] Type-mismatch map case preserves the original collision note; complete solver tail is preserved for effect failures.
- [x] mapE remedy succeeds; existing #386 must-accept/must-reject cases and inferred row-polymorphic callback fixture retain their outcomes.
- [x] The new example checks/runs with IO and existing match_hof_lambda, poly_arith_lambda and effect_budget_demo remain accepted.
- [x] Focused validation below passes and the changelog describes the behavior change accurately.

Risk: tests that prove rejection without checking why. Assert lead, effect labels, note absence and solver suffix together, plus successful remedy.

## Day-by-day sequence

Day 1: M1, M2, M3 (3.5 focused hours). Day 2: M4, M5 (2.5 focused hours), then up to 1.5 hours contingency for integration findings. Milestones run sequentially; there is no external dependency.

## AILANG example syntax gate

AILANG prompt version loaded: v0.16.11 (`ailang prompt` loaded; `ailang prompt --version-active` reported this version). The installed binary warned it may be stale; source is v0.57.0. Before writing the example, build/use the checkout binary, reload its active teaching prompt and record the version. Use that binary for check/run verification.

Showcase `examples/runnable/effectful_callback_map_e.ail`:

| Requirement | Decision |
|---|---|
| Contracts | skip: this demonstration prints a traversal; no useful pure postcondition over console output |
| Effects | include: exported main returns unit with explicit ! {IO}, using std/list.mapE and std/io.println |
| Inline tests | skip: stdout behavior is covered by CLI/example verification and Go pipeline regression fixtures |

## Validation and success metrics

At each milestone run focused Go tests for touched behavior. Final executor gate: `make test-changed` (touched packages plus test-core), `make check-file-sizes`, `make check-changelog`, `make check-boundaries`, and targeted example check/run with the checkout binary. Run gofmt on touched Go files and scoped lint per repository tooling. Use `make verify-examples` when required by the example workflow. CI owns the full suite; `make test` is not an executor requirement, superseding the approved design's broad-suite instruction for cloud execution.

Success means every acceptance criterion above passes, exact local/deferred message parity holds, all selected acceptance/rejection controls retain outcomes, and no stdlib dependency enters internal/types. Coverage is behavioral over all classified branches, not an invented whole-repository percentage target.

## Handoff

Progress: `.ailang/state/sprints/sprint_M-DX-EFFECTFUL-CALLBACK-DIAGNOSTIC.json`.
No implementation has been authorized by this planning stage. The coordinator's plan review/merge triggers sprint-executor as described in the sprint-planner coordinator instructions. Historical #386 is regression context, not a newly linked issue to close. No report-specific GitHub issue was provided.

## Execution record (2026-11-04)

M1–M4 are committed on `coordinator/task-8e422620`; M5 adds pipeline controls, the runnable example, its manifest entry, and a release note. Core nodes expose `ast.Pos`, so application metadata records available start positions rather than inventing `ast.Span` ranges. Nested callback/return mismatches suppress collision hints without attributing inner effects to the outer argument.

AILANG prompt version loaded: v0.16.11, using the checkout binary. The showcase declares `main() -> () ! {IO}`. Contracts: skip, console output has no useful pure postcondition. Inline tests: skip, stdout is checked by example verification and pipeline fixtures. The example checks and runs, printing `a` then `b`.

Validation completed: focused type diagnostics and index tests (20 repetitions), CLI wiring, reporter/filter/named/type-mismatch/remedy pipeline cases, #386 must-accept/must-reject controls, existing `match_hof_lambda`, `poly_arith_lambda`, and `effect_budget_demo` checks; `make verify-examples` (266 passed, 10 skipped, zero failed), manifest sync, file-size/changelog/architecture/format gates, and Go vet for types/index/pipeline.

Environment notes: `make lint` cannot run because golangci-lint is absent. The initial broad type test's CLI subprocess exceeded its 120-second cold-build timeout while downloading dependencies; the checkout binary subsequently built successfully. No toolchains were installed. Cloud branch sync reported `skipped:no-task-id`; work is committed locally for coordinator publication. Design remains in planned pending the independent evaluator, per executor workflow.

Final broad validation: `make test-changed` passed the full types, pipeline, and stdlibindex packages, but failed three CLI policy tests (`TestRunPolicy_DeadlineStopsExecutionWithoutProtocolGrace`, `TestRunPolicy_TimeoutKillsDescendants`, `TestRunPolicy_SupervisorDeathStopsWorker`). All three reproduce on untouched starting commit `abd4dfd`; the first requires missing `ps`. `make test-core` additionally failed `TestRuntimeNestedStopUsesRemainingDeadline` and `TestProcessExec_TimeoutKillsGrandchildren`, both reproduced on that same baseline. The fast-loop pipeline process was killed, including on a subsequent isolated rerun; the earlier full pipeline run passed. These broad gates are not claimed green. All other emitted core-package results passed.

Evaluator handoff: Firestore message `inbox_1793785968442_04ca119c` filed in `sprint-evaluator`; not dispatched because this environment lacks Pub/Sub configuration and cannot read the cloud registry. Coordinator completion markers carry the implementation handoff. No routing configuration was changed.
