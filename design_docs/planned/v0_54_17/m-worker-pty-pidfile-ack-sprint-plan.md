# Sprint Plan: M-WORKER-PTY-PIDFILE-ACK

**Status:** Proposed plan; execution not authorized by this planning task.
**Design:** [Approved design](m-worker-pty-pidfile-ack.md)
**Issue:** #1892 · **Route:** product-test PR, never fleet repair.
**Duration:** 1 working day, approximately 6 hours plus 2 hours contingency; CI queue time additional.
**Risk:** Low implementation risk; medium validation/environment risk.

## Scope and current status

Make the worker PTY fixture publish a complete PID atomically and read it with a bounded content poll. Preserve all lifecycle assertions and deadlines. Only implementation file allowed: `internal/runner/worker_terminal_pty_test.go` (currently 82 lines, well below the 800-line limit; no companion required). No runtime, sibling-test, CI workflow, coverage threshold, or fleet files change. Planning artifacts and verification evidence are administrative exceptions to the implementation-file boundary.

Planner inspected HEAD `267cf2fd` on `coordinator/task-5755c3e3`, version v0.54.16, with a clean starting tree. The producer still uses open/write and the reader still waits only for existence. Search of planned designs and sprint state found the supplied design and no duplicate sprint for this acknowledgement repair. The approved design already audits sibling fixtures and identifies atomic-publish/content-poll precedents; retain its single-file scope. Recheck current source and active work before implementation, since this checkout is not proof of current remote status.

The design records 4/400 baseline races and 0/200 proposed-protocol trials at d75ef77. Those are inherited evidence, not planner measurements. Failed CI run 39544923639 has UNMEASURED total coverage and 18 skipped downstream steps (16 gates, 2 cleanup). GO-2026-6809 and the other CI run are separate; no current CI status is asserted here.

## Estimate and reuse audit

The velocity script was run for seven days. The shallow checkout exposes one documentation commit and no usable changed-LOC history, so measured LOC/day is unavailable. Use task estimates, not an invented historical rate: 25 changed test-fixture lines plus 15 verification-document lines, 40 LOC total. Validation dominates elapsed time. Capacity target is 40 LOC/day, not observed velocity.

All milestones have registry decision `none`: no package-like capability is introduced; this is an existing Go test's embedded Python fixture and its verification. Registry search/info/docs are not applicable, and no package dependency or shared helper will be added. No .ail files or examples are changed; the syntax gate, showcase contracts/effects/inline tests, stdlib and example gates are not applicable. Existing PTY tests are the executable examples.

## Milestones and day-one schedule

### M1: Refresh baseline and deduplicate (~0 LOC)

**Time:** 60 minutes. **Dependencies:** none. **Files:** read-only review of the target, sibling fixtures and existing design/sprint state; temporary reproduction scripts outside the repository.

- Recheck HEAD, uncommitted work and overlapping plans/PRs before repair. Park conflicting work rather than overwrite it.
- Extract the actual current producer/reader protocol into a temporary pair reproduction; run at least 200 sequential and 200 eight-way-concurrent pairs before editing. Record revision, exact commands, counts and failure signatures.
- Require a reproduced empty-PID read before repair. If ordinary trials are inconclusive, use a separately labelled controlled pause between open and write to demonstrate the current-source interleaving; never present the controlled count as a natural race rate. If the protocol has already changed or reproduction contradicts the diagnosis, stop repair and report evidence.

**Acceptance:**
- [ ] Current-source baseline and deduplication recorded before any fixture edit.
- [ ] Empty-PID failure reproduced, with natural and controlled observations clearly distinguished.
- [ ] No product lifecycle or coverage-threshold conclusion inferred from this fixture failure.

### M2: Repair and stress the complete-PID protocol (~25 LOC)

**Time:** 90 minutes. **Dependencies:** M1. **Files:** `internal/runner/worker_terminal_pty_test.go` only; temporary stress harness outside the repository.

- Write the complete PID to a same-directory temporary file, close it, then publish with os.replace. Prefer a context-managed write to ensure all PID bytes are written and flushed before close/replace. No stdout/stderr changes.
- Replace existence polling with content polling at the existing 10ms cadence and 5s deadline. Retry only absent/empty/unparseable content; require PID > 1 and preserve os.kill(pid, 0). Do not silently swallow unrelated OS errors. Atomic publication is essential: parsing alone cannot distinguish a positive truncated PID from a complete PID.
- Stress the exact new strings: at least 200 sequential plus 200 eight-way-concurrent pairs, zero malformed acknowledgements and liveness verified for each exact PID. Check absent, empty and malformed content waits remain bounded and fail with the startup-ack diagnostic.
- Review the diff against the saved baseline: ps disappearance, exit codes, termios/PENDIN, escape sequences, OVER_BUDGET assertion, worker_absent bookkeeping and SIGKILL fallback remain byte-identical. Preserve 5s/12s/30s/120s deadlines.

**Acceptance:**
- [ ] Both atomic producer publication and bounded reader content polling implemented.
- [ ] Zero failures in at least 400 fixed-protocol pairs with the stated concurrency split; exact-PID liveness checked.
- [ ] Timeout cases bounded; original lifecycle assertions and deadlines unchanged.
- [ ] Implementation diff confined to the target fixture; no permanent reproduction helper added.

### M3: Validate lifecycle matrix and record handoff (~15 LOC)

**Time:** 90 minutes of executor work, then up to 2 hours validation observation; 2 hours contingency across the day. **Dependencies:** M2. **Files:** existing design Verification Log and sprint progress JSON. No additional code files.

- [ ] On Linux or macOS with python3, ps and the repository-supported Go toolchain, run `go test ./internal/runner -run '^TestWorkerTerminalPTYLifecycle$' -count=1`; all 11 cells execute without skips and pass.
- [ ] Run `go test ./internal/runner -run '^TestWorkerTerminalPTYLifecycle$/^eval$/^sigint$' -count=20`; all repetitions pass.
- [ ] Run `go test ./internal/runner -run '^(TestTerminalCLIPTYLifecycle|TestWorkerCLIProcessExitLifecycle)$' -count=1`; sibling lifecycle checks pass unchanged.
- [ ] Run `make test-changed`, `make lint`, `make fmt-check`, `make check-file-sizes` and `make check-changelog`; record results. No local full `make test` requirement: CI owns the full suite, superseding the design's older local-full-suite wording.
- [ ] Update the design Verification Log with baseline, fixed-protocol counts, commands, host/toolchain and exact fix revision. Record failures or skips as unmeasured, never passes.
- [ ] Observe CI at the exact fix revision: PTY matrix and coverage step succeed; record measured coverage and status of all 18 previously skipped downstream steps. Attribute independent govulncheck failures separately. Do not claim complete downstream validation while any relevant gate is skipped or unmeasured.

Changelog default: none for the fixture-only repair, preserving the approved source-file scope. Run check-changelog; if repository policy demands a fragment, report the scope conflict for review rather than editing legacy aggregate changelogs or silently widening scope. The current fragment convention is `changelogs/unreleased/YYYY-MM-DD-<slug>.md`, opening with `### `.

## Dependencies, risks and completion

This planning environment has no Go executable. No Go test or fresh coverage measurement was attempted by the planner. Executor needs a POSIX host and supported Go toolchain; missing prerequisites block validation, not justify skips. Temporary workers must be cleaned by exact PID in reproduction teardown. A zero observed stress failure rate supports the atomic-publish invariant but is not statistical proof of all schedules.

Milestones are sized for 60–90 minute work sessions; long commands/CI observation may resume separately. Documentation LOC is distinct from test LOC: 0 runtime + 25 test fixture + 15 evidence = 40 total. No new unit tests that mirror the fixture, new abstractions, AILANG demos, or package dependencies are planned.

Success requires M1–M3 evidence, unchanged lifecycle checks, focused local gates and revision-matched CI observations. Unrelated CI failures must remain explicit release blockers where applicable rather than be repaired under this sprint. The plan and progress JSON are reviewable handoff artifacts only; this task does not approve or launch sprint execution. Coordinator approval/execution routing remains pending.
