# Sprint Plan: spawnProcessResult

**Sprint ID:** M-SPAWN-PROCESS-RESULT
**Design:** [spawn-process-result.md](spawn-process-result.md)
**Status:** Implementation completed; ready for independent evaluation. Baseline container cleanup failures documented below.
**Duration:** 2 working days, about 7 hours including 90 minutes contingency.
**Risk:** Medium: worker admission/cleanup and persisted bytecode compatibility.

## Goal and scope

Add spawnProcessResult returning Result[ProcessHandle, ProcessError] while retaining spawnProcess's current fail-fast behavior. Reuse ProcessError, authorization, managed-process ownership, and existing Result constructors. No parser/type-system work or package dependencies.

The approved design's default is adopted: asyncExecProcessResult remains follow-up work. No stream API is added. The original v0.54.7 directory is retained for traceability; this checkout is already v0.54.9, so use unreleased metadata and a changelog fragment, not a historical release edit.

## Current status and estimate basis

Verified at 79ede6ee: the new symbols are absent; process_spawn.go contains the existing shared-policy path, and the denial-to-Result helper already exists. The design includes the required systemic audit of exec, spawnProcess and asyncExecProcess. Its implementation remains outstanding.

The 7-day velocity script found only one visible coordinator commit and no usable LOC metrics. This is a shallow checkout; a measured LOC/day rate cannot be inferred. Estimate is bottom-up: 500 changed/added lines (roughly 230 implementation, 200 tests, 70 documentation/example), 250 LOC/day planning capacity, with 25–30% time contingency. LOC is effort, not a delivery requirement. No coverage measurement was run during planning; exercise every new return path rather than invent a baseline percentage.

## Registry reuse audit

Ran `ailang pkg search process`, then `ailang pkg info` and `ailang pkg docs` for both returned candidates: sunholo/external_backend@0.2.0 and sunholo/duckdb@0.2.1. Both consume std/process.exec; neither can intercept a fatal native spawn error or extend the capability runtime. Decision for M1, M2, M3 and M4: **none**; reuse existing repository runtime/stdlib mechanisms, with no external dependency or package contribution required.

## Syntax and example gate

AILANG prompt version loaded: v0.16.7 (full `ailang prompt` loaded; `ailang prompt --version-active` reports this teaching-prompt version, distinct from std/VERSION v0.54.9).

| Module | Gate | Decision |
|---|---|---|
| examples/runnable/process_spawn_result.ail | contracts | skip: OS authorization/spawn outcomes are effectful and cannot be proved by a pure requires/ensures contract |
| examples/runnable/process_spawn_result.ail | effects | include: main() -> () ! {IO, Process}; spawnProcessResult(cmd: string, args: [string]) -> Result[ProcessHandle, ProcessError] ! {Process} |
| examples/runnable/process_spawn_result.ail | inline tests | skip: effectful process behavior is checked by CLI output/exit assertions and hermetic Go tests |

The std/process wrapper retains explicit ! {Process}; its contract is tested at runtime. Use the teaching prompt again during execution and check imports/constructor visibility with `ailang check` before publishing any snippet.

## Milestones

### M1: Share spawn lifecycle without changing legacy behavior

**Estimate:** 100 LOC; 60–90 minutes. **Dependencies:** none.

**Files:** internal/effects/process_spawn.go; new internal/effects/process_spawn_legacy_test.go. Example reference: existing examples/runnable/process_stdin_write.ail (unchanged).

**Acceptance criteria:**

- [x] Extract the shared authorization/admission/spawn lifecycle while preserving legacy errors and prefixes, including unprefixed admission errors.
- [x] Regression tests cover missing command, allowlist/confinement denial, success, and admission failure; no leaked workers or handles.

**Validation:** Focused tests for the changed package(s), then `make test-changed`.

### M2: Expose typed spawn through stdlib and VM

**Estimate:** 170 LOC; 60–90 minutes. **Dependencies:** M1.

**Files:** new internal/effects/process_spawn_result.go; new internal/builtins/process_spawn_result.go and process_spawn_result_test.go; std/process.ail; internal/bytecode/effect_names.go; new internal/vm/process_spawn_result_test.go. Example delivered in M4.

**Acceptance criteria:**

- [x] Register Process.spawnProcessResult and _process_spawn_result with Result[ProcessHandle, ProcessError] metadata and wrapper.
- [x] NotFound, NotAllowed including narrowed subcommands and confinement, OS SpawnFailed, and BeginWorker/Complete failures return typed Err after capability checking.
- [x] Append __process_spawn_result without changing existing bytecode indices; focused compiler/VM dispatch and builtin type tests pass.
- [x] Successful Ok handle supports stdin write/close and cancellation; capability absence retains policy failure.

**Validation:** Focused tests for the changed package(s), then `make test-changed`.

### M3: Lock down failure and lifecycle regressions

**Estimate:** 130 LOC; 60–90 minutes. **Dependencies:** M2.

**Files:** internal/effects/process_spawn_result_test.go; new cmd/ailang/run_process_spawn_result_test.go. CLI fixtures live in test temporary directories; showcase delivered in M4.

**Acceptance criteria:**

- [x] Hermetic effects tests exercise each typed failure, including shutdown during admission completion; assert cleanup and distinguish no-child from child-started-then-cleaned-up cases.
- [x] CLI integration regression proves missing-command fallback prints started and after and exits zero; legacy spawn still aborts with its original error.
- [x] Targeted race tests for changed lifecycle tests pass without sleeps as synchronization.

**Validation:** Focused tests for the changed package(s), then `make test-changed`.

### M4: Publish example and document process contracts

**Estimate:** 100 LOC; 60–90 minutes. **Dependencies:** M3.

**Files:** examples/runnable/process_spawn_result.ail; examples/manifest.json; docs/docs/reference/effects.md; docs/docs/guides/streaming.md; changelogs/unreleased/2026-10-15-spawn-process-result.md.

**Acceptance criteria:**

- [x] examples/runnable/process_spawn_result.ail and manifest entry demonstrate typed fallback under IO,Process with the missing command allowlisted; expected output includes after.
- [x] Effects reference and streaming guide document the new API, literal allowlist keys, and WorkerHandleInvalid after normal reaping.
- [ ] Unreleased changelog fragment begins with ###; make test-changed, make check-file-sizes, make check-changelog, make verify-stdlib and make verify-examples pass.

**Validation:** Focused tests for the changed package(s), then `make test-changed`.

## Day-by-day execution

Day 1: M1 extracts the lifecycle with regression coverage; M2 adds the native op, registration, stdlib wrapper and append-only VM name. Keep refactor and new behavior separately reviewable.

Day 2: M3 hardens failure/cleanup and CLI regressions; M4 ships the runnable example and docs, then runs the final gates. Reserve 90 minutes across both days for admission races, VM fixtures, or portability adjustments. Stop and report an unanticipated semantic decision rather than changing the API contract.

## Technical risks and validation

Preserve resolveCommand's confined-mode and narrowed allowlist checks. Convert ProcessDenial using its constructor/detail; map spawn/admission errors to SpawnFailed. Capability absence remains a policy error. Malformed builtin arguments are internal misuse, outside the typed AILANG call contract.

The design says SpawnFailed means a child never existed, but admission.Complete can fail after NewManagedProcess started it. Preserve cleanup and document the precise guarantee: no usable child handle is returned; a started child must be stopped/reaped. Do not weaken lifecycle policy or claim no child ever existed in that race.

Preserve legacy admission errors exactly, not just authorization prefixes. Use errors.As or a typed denial channel to avoid string parsing. Tests use temporary executables/controlled PATH for NotFound and OS spawn failure, with no dependency on crew or a particular shell path. Synchronize lifecycle tests on completion channels, not arbitrary sleep durations. Run focused `go test -race` for the touched admission/lifecycle cases.

Append the bytecode builtin name and assert existing indices remain stable. Follow native !js build tags; this sprint does not add browser process support. Current touched Go files are far below 800 lines (builtins/process.go 263, vm/worker_effect_test.go 167); new companion files above keep additions isolated. Run `make check-file-sizes` at completion.

Executor gates: `make test-changed` (changed packages plus test-core), `make check-file-sizes`, `make check-changelog`, `make verify-stdlib`, `make verify-examples`, and formatting/lint checks appropriate to changed Go files. Use repository make help/target definitions for required arguments. CI owns the full suite; do not require `make test` on the cloud executor. Capture any pre-existing gate failures separately. Run simplicity audit if a baseline is available and report the single added builtin surface point without masking regressions.

## Completion and handoff

Success requires all milestone criteria, runnable fallback output, no legacy behavior drift, lifecycle cleanup coverage, and both docs observations. Existing stdin/cancel/policy examples remain regression references. No release/tag/version bump is part of this sprint. No issue number was supplied; do not infer a GitHub issue from task IDs.

This plan and its JSON are review artifacts. Under the coordinator workflow, merging the sprint-plan PR approves it and triggers sprint-executor; do not separately dispatch execution before that approval. See `.agents/skills/sprint-planner/resources/coordinator.md`.

## Execution notes (2026-10-15)

Recovered the exact approved planner artifacts from commit c1751499 because the executor checkout omitted them. Implementation stays on coordinator/task-431ee02d. M1 preserves legacy error text and ownership. M2 adds the builtin, stdlib wrapper and append-only VM dispatch; ProcessError needed a scoped VM constructor mapping, now checked against declaration order. M3 adds portable Go child-process lifecycle tests, typed failures, capability and missing-context checks, and the original CLI regression. The example prints started, a fallback message, and after with exit zero.

AILANG prompt version loaded: v0.16.7. Example contracts: skipped because OS startup/authorization is effectful; explicit IO/Process effects included; inline tests skipped in favor of hermetic effects tests and CLI assertions. The example type-checks and runs using the rebuilt binary.

The approved process interface snapshot is intentionally updated for the one additive spawnProcessResult export; no other stdlib interface changes. Targeted race tests, builtin validation, formatting, file-size, changelog and architecture-boundary checks pass.

Optional simplicity audit exits 2 against 2026-09-30: nonleaf closure 32→33 and tracked files 25,731→28,563 at measurement. The starting commit already has 28,555 tracked files, and this sprint introduces no package or dependency edges; this is existing repository-wide drift plus the expected additive files. The audit's timed core probe reports FAILED (also FAILED in its baseline); the required make test-changed gate independently runs test-core.

### Final validation and handoff

All four implementation milestones are complete. The final M4 all-green-gates checkbox remains open as an explicit validation caveat, rather than hiding a baseline failure. `make test-changed` and `make test-core` were run and are not green in this container: TestRuntimeNestedStopUsesRemainingDeadline, TestRunPolicy_TimeoutKillsDescendants and TestRunPolicy_SupervisorDeathStopsWorker reproduce on untouched starting commit 79ede6ee. Actual procps inspection shows the reported survivors are zombies adopted by PID 1. No cleanup semantics were changed to accommodate the container.

The missing ps utility was supplied via temporary extracted Debian procps packages. The deadline test then passed. Deep-recursion tests pass with GOMEMLIMIT=512MiB/GOGC=30; the initial process kills were resource-related. The entire internal/pipeline suite passes when run alone (73.812s), after intentionally adding the one new builtin signature to builtin_types.golden; its first concurrent runs were resource-killed. All other core-package results passed. The new spawn CLI regression and all targeted lifecycle/race checks pass.

`make verify-stdlib`: all 48 interfaces stable after the intentional additive process snapshot update. `make verify-examples`: 247 passed, 0 failed, 9 existing skips; manifest modules have no drift (one existing stale missing-file entry is logged). Builtin doctor, formatting, file-size, changelog and architecture checks pass.

Lint: the initial run exited 137, and a 512MiB low-memory retry hit the default 5m timeout while reporting 0 issues. The same full configured linter scope passed with 0 issues using `GOMEMLIMIT=768MiB GOGC=30 GOMAXPROCS=2 golangci-lint run --timeout 10m --concurrency 1 ./cmd/... ./internal/... ./serveapi/...`.

Added/deleted implementation and documentation lines vs starting commit (excluding sprint JSON and imported plan): see sprint JSON velocity metrics. The larger-than-planned test portion covers the full failure/lifecycle matrix, portable child fixture, exact legacy behavior and CLI regression. No stream API, parser/type-system change, release or new package dependency was added. Independent sprint-evaluator handoff follows this final checkpoint; merge/release remains outside this execution.

Evaluator handoff was filed in canonical Firestore as `inbox_1792054443306_f6f1777f`. Direct dispatch did not occur: this executor cannot read the cloud routing configuration (GCS 403), and its local Pub/Sub notifications are disabled. Independent evaluation has not run in this session.
