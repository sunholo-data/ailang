# Sprint Plan: M-PROCESS-RESULT-SEMANTICS

Date: 2026-10-17. Design: [process-result semantics](m-process-result-semantics.md). Issues: #1878, #1879 (P1), #1880 (P2).

## Summary and authorization

Prepare faithful process results with explicit startup, completion and truncation facts. Planning is authorized by the handoff; implementation and executor dispatch are not authorized by this task. Status: planned, not started. No merge, release, issue closure or consumer edits are included.

Duration: 4 working days after design freeze, approximately 10 hours including 2 hours buffer. Five implementation milestones each target 60–90 minutes. Estimated changed LOC: 560 including tests/docs/examples (planning estimate, not a measured productivity claim). Risk: high because of silent consumer pattern mismatches and cancellation races.

## Mandatory execution gate

The approved document still has unchecked design-freeze decisions. General design approval does not record which alternative was chosen. Before any implementation, the operator must record HD-2 (A widened errors or B status record), HD-4 (kill at cap or capture to completion), HD-1 confirmation and the HD-6 per-arm guarantee table in the design. Also record whether V7 shared stdout/stderr budget is included, and exact timeout/cap/exit precedence. Refine this plan and JSON to those decisions before executor dispatch.

Resolve these design contradictions at that gate:

- The universal “Err implies never started” claim is valid only for B. Under A, Timeout/AbnormalExit/limit errors can follow execution. Guarantee by constructor, not Result polarity.
- Under B, Ok means started; it need not mean normal completion. Status must distinguish completion, timeout, signal, cap termination and unknown wait outcome without inventing an exit code.
- HD-3 natural completion must be reconciled with HD-4 kill-at-limit using an explicit race/precedence rule. Observing a kill request alone does not prove why the process exited.
- Scope the never-started table to exec: spawnProcessResult documents admission failure after a child was started and reaped. Shared constructor names do not establish a global guarantee.
- No output loss is permitted on unclassified post-start wait/I/O failures. Freeze a representation for these alongside the named outcomes.

Release remains the next approved minor after v0.54.x, with coordinated migration of motoko and affected package consumers. The separate V8 pattern-arity defect is excluded.

## Current implementation and estimates

Source read confirms cmd.Run merges start/wait errors, cap and timeout return before preserving buffers, stderr gets a separate initial cap, and normal results hard-code truncated false. process.go is 364 lines, process_test.go 451, builtin process.go 263 and std/process.ail 144. Use companion test files below rather than growing a single regression file toward the 800-line limit. Existing containment, allowlist, sandbox, capability and effect boundaries remain required regressions.

This is a shallow checkout: git log exposes only the 2026-10-17 design-doc commit. The 7-day velocity script reports no usable LOC metrics and cites historical changelog entries; these cannot establish current throughput. Therefore no empirical LOC/day or historical completion rate is asserted. Capacity budget is 140 changed LOC/day over four days, with 25% time buffer over eight hours of work. Re-estimate if freeze expands scope, especially V7; do not silently defer a failing combined-cap contract.

Duplicate search of sprint state and planned docs found no matching process-result sprint beyond the supplied design. The design's systemic audit covers timeout, limits, signals, wait failures and pre-start errors, satisfying the systemic-analysis prerequisite. Existing code/tests were inspected; runtime baseline and coverage measurement are deferred to execution because this task changes only planning artifacts.

## Registry reuse audit

Ran `ailang pkg search process`, then `pkg info` and `pkg docs` for both matches: sunholo/external_backend@0.2.0 and sunholo/duckdb@0.2.1. They wrap std/process; neither can restore bytes/status already discarded in the host runtime. Decision for M1–M5: **none**, no package dependency or contribution implements this core correction. external_backend's describeProcessError is an additional ADT match consumer for the release migration audit; duckdb also depends on process behavior. External package and motoko changes require their own coordinated work and are release dependencies, not edits in this sprint.

AILANG prompt version loaded: v0.16.7 (active teaching prompt identifier; repository std/VERSION is v0.54.15). Loaded using `ailang prompt`; the CLI warns its binary may be stale. Executor builds current source before checking new syntax and reloads the prompt if its active version differs.

## Milestones

### M1: Separate startup from waiting (~100 LOC)

Dependencies: design freeze and execution authorization. Budget: 60–90 minutes. Registry reuse: none.

Files: internal/effects/process.go; new internal/effects/process_start_test.go.

Classify NotFound, PermissionDenied and SpawnFailed only from Start errors using errors.Is; retain containment, CommandContext cancellation and WaitDelay.

Acceptance:
- [ ] Missing executable and non-executable file return pre-start errors; no wait error becomes SpawnFailed.
- [ ] Timeout process-group, confinement and spawnProcessResult regression tests pass.

Risk: Start/Wait cancellation drift; retain existing group-kill regression fixtures.

### M2: Install the frozen result schema (~110 LOC)

Dependencies: M1. Budget: 60–90 minutes. Registry reuse: none.

Files: std/process.ail; internal/builtins/process.go; internal/effects/process.go; new internal/effects/process_result_test.go.

Implement only the operator-selected HD-2 representation atomically across declarations, builtin metadata and value builders; preserve raw bytes and represent unavailable exit status explicitly.

Acceptance:
- [ ] Timeout and signal outcomes retain both captured byte buffers and distinguish termination from normal completion.
- [ ] Known nonzero exits retain their real status; unknown post-start wait failures preserve output and never claim execution did not start.
- [ ] Tests evaluate exact constructors, arities and fields; type-checking alone is insufficient because of V8.

Risk: Schema divergence and silent matches; evaluate real values against explicit patterns.

### M3: Report pipe closure accurately (~80 LOC)

Dependencies: M2. Budget: 60–90 minutes. Registry reuse: none.

Files: internal/effects/process.go; new internal/effects/process_wait_test.go.

Map ErrWaitDelay with observed successful foreground exit to completed output with exit 0 and truncated true; retain nonzero-exit symmetry.

Acceptance:
- [ ] Held-pipe exit-0 and exit-nonzero fixtures preserve output and real foreground exit code.
- [ ] Uncapped normally drained output has truncated false; forced pipe closure has truncated true.
- [ ] Fixtures bound waits and clean up descendants even on assertion failure.

Risk: Timing-sensitive pipes; use bounded synchronization and platform guards.

### M4: Apply the frozen output budget policy (~140 LOC)

Dependencies: M2, M3. Budget: 60–90 minutes. Registry reuse: none.

Files: internal/effects/process.go; new internal/effects/process_output_limit_test.go; cmd/ailang/main_run.go if help changes.

Implement HD-4, shared-budget V7 handling if approved, and the frozen ordering of deadline, cap and observed completion; use existing containment for any approved kill.

Acceptance:
- [ ] Tests cover stdout-only, stderr-only, interleaved streams, exact cap and cap+1; assert the approved combined-budget rule.
- [ ] Natural completion versus cap-triggered termination follows the frozen policy and preserves observed exit status when available.
- [ ] Timeout plus cap and pipe closure precedence are specified and tested; no fabricated exit 1 or lost captured prefix.
- [ ] Any concurrent budget/cancellation state passes focused race tests and does not double-kill or leak children.

Risk: Concurrent limit/deadline races; test both orderings and clean up every child.

### M5: Publish contract and migration fixtures (~130 LOC)

Dependencies: M4. Budget: 60–90 minutes. Registry reuse: none.

Files: std/process.ail; docs/docs/reference/effects.md; internal/builtins/process.go; new examples/runnable/process_result_semantics.ail; examples manifest; changelogs/unreleased/2026-10-17-process-result-semantics.md; process interface goldens.

Document the selected per-arm guarantees, breaking release boundary and motoko/package migration; add a runnable showcase with assertions against real returned values.

Acceptance:
- [ ] Example demonstrates completed output plus timeout/cap semantics under explicit capabilities, and existing process_subcmd_allowlist example passes.
- [ ] Guarantee table agrees with selected shape: option A allows post-start Err; option B reserves Err for pre-start exec failures.
- [ ] Breaking-change fragment starts with ### and includes old/new consumer matches and migration guidance; no patch-release claim.
- [ ] Review the intended process interface diff before updating its golden with existing freeze tooling; unrelated golden changes are excluded.
- [ ] Focused validation gates below pass and remaining operator or consumer release dependencies are recorded.

Risk: Consumers may silently mis-handle new shapes; require migration evidence before release.

## Day-by-day schedule

| Day | Work | Deliverable |
|---|---|---|
| 1 | Operator freeze first, then M1 | Startup classification regressions |
| 2 | M2 and M3 | Frozen schema, partial output and held-pipe results |
| 3 | M4 | Approved cap/termination semantics and race checks |
| 4 | M5 and 2-hour buffer | Showcase, migration note, interface diff and focused gates |

No milestone starts until the execution gate is satisfied. If a milestone exceeds 90 minutes, split it at a coherent tested boundary and update progress before continuing.

## Example specification

New showcase: examples/runnable/process_result_semantics.ail, exported main() -> () ! {IO, Process}; it renders raw output via std/bytes and checks outcomes using the frozen schema. It needs an explicitly allowlisted POSIX shell and short configurable timeout/cap scenarios; portable happy-path coverage remains runnable elsewhere with existing OS guards for signal fixtures.

| Requirement | Decision |
|---|---|
| Contracts | skip: external execution and scheduling cannot be established by pure requires/ensures verification; assert observed results in runtime fixtures |
| Effects | include: main() -> () ! {IO, Process}; exec retains ! {Process} |
| Inline tests | skip: effectful subprocess timing is unsuitable for pure tests [(in,out)]; use Go subprocess fixtures and runnable assertions |

Register the example through existing manifest tooling; do not create ad-hoc launch scripts. Preserve examples/runnable/process_subcmd_allowlist.ail.

## Validation and success metrics

For each code milestone run focused tests in touched packages plus `make test-changed` (includes test-core), format touched Go files and run focused lint/vet as appropriate. For M4 run targeted `go test -race ./internal/effects -run 'Process.*(Output|Limit|Timeout|Wait)'` with actual test names verified to select tests. Do not count an empty selection as a pass. POSIX signal fixtures follow existing GOOS guards.

Final execution gates: `make test-changed`, `make check-file-sizes`, `make check-changelog`, `make check-boundaries`, `make verify-stdlib`, and `make verify-examples`. Build current source first. Review and deliberately refresh only the approved process interface golden using existing freeze tooling before verify-stdlib; this is an intentional breaking interface change. CI owns the full suite. These focused gates supersede the design's executor requirement to run make test.

Coverage target is the complete outcome matrix, not an invented repository coverage percentage: denied/missing/permission/other start failure; exits 0/nonzero; timeout with partial bytes; signal with partial bytes; cap and exact-boundary cases on both streams; held-pipe exits 0/nonzero; unknown post-start failure; deadline/cap precedence. Assert both raw buffers, completion/termination state, available exit/signal, resolved path and truncation. Keep process_timeout_group_test.go, process_confined*_test.go, process_spawn_result_test.go and CLI process-policy regressions passing.

## Completion and handoff

All milestone passes start null. Final evaluation compares results against the frozen design and this matrix using sprint-evaluator after authorized execution. Planning validation checks populated JSON, matching LOC totals/dependencies and linked issues; it does not prove runtime behavior. Do not dispatch sprint-executor from this planning-only task. Return both artifact paths to the coordinator for operator review, without approving or merging anything.
