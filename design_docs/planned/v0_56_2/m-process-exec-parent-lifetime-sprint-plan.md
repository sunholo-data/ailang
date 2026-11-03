# Sprint Plan: M-PROCESS-EXEC-PARENT-LIFETIME

**Design:** [parent lifetime](m-process-exec-parent-lifetime.md)  
**Issue:** #2034; evidence context #2069; excludes #2037  
**Status:** Planned, not started; planning authorized by the current handoff.  
**Duration:** 3 engineering days including approximately 25% integration buffer.  
**Risk:** Medium, with high-impact policy group isolation risk.

## Authorization and scope

The current user handoff states previous work is approved and explicitly requests sprint planning. Treat that as ratification of design D1/D3/D4/D6 despite the historical unchecked Design Freeze rows; do not reopen those decisions. No human design decisions remain open. Execution still requires a separate "execute sprint" instruction; this artifact does not authorize implementation, release or deployment. The original design remains unchanged for provenance.

Reuse WorkerOwner admission, OwnedWorker, runlife.Prepare and ConfigureContained. Preserve Result constructors, authorization order, per-call hard timeout, WaitDelay, signal subscription and exit codes. Runtime force is KillProcess in owner-group mode, KillGroup in standalone mode. SIGKILL of the runtime and setsid/daemon escape containment remain out of scope.

## Current status and estimates

Source inspection confirms ordinary exec still uses Background plus cmd.Run; managed/stream workers already implement the ownership/lifetime pattern. No implementation is complete. process.go is 364 lines; process_runtime_lifetime_test.go is 360. Put worker code and tests in companion files below rather than growing existing files toward 800 lines.

The 7-day velocity script and git history show only b043b93e (the design document); no implementation LOC/day or completion rate can be measured from this shallow checkout. Its archived changelog matches are not recent velocity. Use the design's 2–3 day estimate, with a provisional capacity of about 153 changed LOC/day: 460 LOC total (about 120 runtime, 290 tests, 50 documentation). This is a planning allowance, not an observed rate. Each milestone contains bounded executor sessions of 60–90 minutes; split M2/M3 by test family across sessions if necessary.

## Registry reuse audit

`ailang pkg search process` returned sunholo/duckdb, sunholo/external_backend and sunholo/audio. Inspected `ailang pkg info sunholo/external_backend` and `ailang pkg docs sunholo/external_backend`: it wraps std/process.exec and cannot own host shutdown. Decisions M1–M4: **none**; this is runtime-internal ownership wiring, regression tests and documentation, with existing Go mechanisms reused and no package dependency. The CLI warned its binary may be stale; these registry results are discovery evidence only, source inspection establishes the implementation gap.

## Sequencing clarification

WorkerAdmission.Complete may synchronously stop and Join on late admission. A literal Start → Complete → Wait on one goroutine would self-block. After successful Start and lifetime.AfterStart, start exactly one Wait/completion goroutine before Complete. It owns cmd.Wait, AfterWait, bounded receipt join, terminal error publication and runDone closure. Synchronize builtin result/output access through runDone; release registration after Complete returns to avoid a natural-completion-before-registration leak. Never return buffers while Wait may still be writing. Test the late-admission and fast-natural-completion races explicitly.

Use the existing managed lifetime deadline conventions, capped by WorkerShutdownTimeout and any earlier shutdown deadline. A timeout stays hard cancellation even for runtime children. Receipt failures are host cleanup failures, never substituted language errors. Do not add another signal handler, registry or kernel containment mechanism.

## Milestones

### M1_EXEC_WORKER: Register synchronous exec with the existing execution owner

**Estimate:** 140 LOC. **Schedule:** Day 1 morning; ~120 runtime + 20 test LOC. **Dependencies:** none.

**Files:** internal/proctree/proctree.go; internal/effects/process.go; new internal/effects/process_exec_worker.go; new internal/effects/process_exec_worker_test.go.

Acceptance criteria:

- [ ] BeginWorker follows authorization and confinement; every pre-start failure aborts admission and prepared lifetime resources.
- [ ] One Wait goroutine starts before Complete can synchronously Join; result/output reads occur only after completion.
- [ ] RequestStop uses cooperative lifetime OR context cancellation; force kills the group only outside owner-group mode.
- [ ] Late admission is stopped and reaped without waiting on the blocked evaluator; existing Result and timeout mapping remains intact.

### M2_LIFETIME_RACES: Exercise admission, nested lifetime, and receipt failure races

**Estimate:** 130 LOC. **Schedule:** Day 1 afternoon–Day 2 morning; ~130 test LOC. **Dependencies:** M1_EXEC_WORKER.

**Files:** new internal/effects/process_exec_lifetime_test.go; reuse process_runtime_lifetime_test.go harness.

Acceptance criteria:

- [ ] Deterministic tests cover close-before-admission, close-between-Start-and-Complete, normal release and repeated stop.
- [ ] Nested ordinary exec propagates remaining deadline and requires terminal receipt; missing/failed receipts surface as WorkerCleanupFailure.
- [ ] Per-call timeout remains hard cancellation; nonzero exit, authorization, output limit and confined behavior regressions pass.

### M3_SIGNAL_POLICY: Verify real runtime signals and preserve policy ownership

**Estimate:** 140 LOC. **Schedule:** Day 2 afternoon–Day 3 morning; ~140 test LOC. **Dependencies:** M2_LIFETIME_RACES.

**Files:** new internal/effects/process_exec_signal_unix_test.go and process_exec_policy_unix_test.go; reuse built-runtime harness.

Acceptance criteria:

- [ ] Readiness-synchronized POSIX subprocess tests send SIGTERM/SIGINT only to runtime and assert exits 143/130, no delayed marker and no surviving original group members.
- [ ] Isolated policy subprocess fixture verifies unchanged group placement, leader-only runtime force, supervisor survival and outer owner group cleanup.
- [ ] Tests own cleanup on all failure paths; platform gates preserve Windows leader-only behavior and build compatibility.

### M4_DOCS_VALIDATION: Document bounded cleanup and finish focused checks

**Estimate:** 50 LOC. **Schedule:** Day 3 afternoon; ~50 documentation LOC. **Dependencies:** M3_SIGNAL_POLICY.

**Files:** docs/docs/reference/effects.md; docs/LIMITATIONS.md; changelogs/unreleased/2026-11-03-process-exec-parent-lifetime.md.

Acceptance criteria:

- [ ] effects reference and LIMITATIONS describe standalone POSIX group cleanup, policy-owner responsibility and bounded cleanup failures.
- [ ] No SIGKILL/crash/power-loss or setsid/daemon containment guarantee; #2037 stays separate.
- [ ] Focused validation, file-size and changelog gates pass; unreleased fragment records the behavior.

## Examples and syntax scope

No new API or showcase module is needed; existing process examples remain the compatibility examples. All new regression fixtures are Go subprocess tests or reuse existing AILANG fixtures without editing them. No `.ail` changes are planned, so the AILANG syntax gate does not apply. If an executor needs a new `.ail` fixture, load `ailang prompt` first, record its active version, check the fixture, and record contracts/effects/inline-test include-or-skip decisions before adding it.

## Validation and success metrics

During M1/M2 use targeted `go test ./internal/effects ./internal/proctree ./internal/runlife -run 'Test(ProcessExec|SyncExec|Runtime)'` with concrete new names matching that filter. Run race-enabled targeted lifecycle/admission tests after concurrency wiring stabilizes. Real signals and policy group tests run only in isolated POSIX subprocesses with readiness handshakes and bounded polling, never by signaling the test runner's group. Readiness must establish that ordinary exec has started its command; a fixed startup sleep is insufficient. Assert marker absence after its scheduled write time and separately verify group absence.

Final gates: `make test-changed` (touched packages plus test-core), `make check-boundaries`, `make check-file-sizes`, `make check-changelog`, and focused package tests above. Use existing cross-platform build/CI gates to verify Windows compatibility. Do not require the full make test suite in the executor. No coverage percentage is invented; coverage target is every lifecycle branch and the signal/policy matrix. Changelog fragment begins with `### `. Record actual test results and skipped platform cases at execution time.

## Risks and handoff

Late admission and fast completion can strand a child or registration: the M1/M2 race tests are mandatory. Cooperative stop can be defeated by duplicate cancellation: mirror managed either/or handling and test it. Owner-group state is process-global: test it in subprocesses. Pipe-holding escaped descendants can exceed the host join bound: preserve WaitDelay and report bounded cleanup failure without promising stronger containment.

No executor handoff is sent by this planning task: the user requested planning only. Present plan and populated JSON for review; implementation starts only on a later execution instruction. No release, tag, deployment or issue closure is part of this sprint.
