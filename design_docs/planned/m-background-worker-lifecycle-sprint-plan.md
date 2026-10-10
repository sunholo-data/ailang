# Sprint Plan: M-BACKGROUND-WORKER-LIFECYCLE

**Status:** Implementation complete locally; M5 independent final gates pending. User approved execution on 2026-10-10.
**Design:** [Owned subprocess lifecycle](m-background-worker-lifecycle.md)
**Duration:** 8 engineering days including integration buffer; 3,400 LOC total (implementation plus tests).
**Risk:** High: runtime ownership, pipe/Wait ordering, signals, embedding concurrency.
**Branch:** `sprint/background-worker-lifecycle` in an isolated checkout.

## Current Status and Velocity

The 7-day velocity script was run. Its broad changelog scan includes historical entries and its diff is only HEAD~1, so it does not justify a daily rate. The immediately preceding terminal runtime sprint added 1,388 production lines and 2,090 test lines across effects/runner/VM/bytecode/builtins. That sprint is useful scope evidence, not a reliable human-day throughput measurement. Use 425 LOC/day as a conservative planning assumption with the eight-day budget; record actual time and LOC separately.

The leak is verified in official v0.54.0. Current baseline is `37001e839` (stdlib v0.54.1), plus the design commit. Shared ownership, public cancellation and host cleanup are implemented locally; release delivery remains pending. All five design-freeze choices are approved by the current user instruction.

## Registry Reuse Audit

Searches: `ailang pkg search process` and `ailang pkg search terminal`. Inspected `pkg info` and `pkg docs` for `sunholo/external_backend@0.2.0` and `sunholo/terminal_ui@0.2.0`. The former wraps synchronous exec and excludes long-lived workers; the latter declares IO/Env and relies on core terminal lifecycle. Neither can own runtime subprocess shutdown. Every milestone therefore records `none` for an AILANG package dependency, while reusing existing core `internal/proctree`, capability dispatch, Stream.Child, host interfaces and test helpers. No new package or external dependency is planned.

## Execution Waves

M1 establishes the shared supervisor and its API. M2 (managed process files) and M3 (async source files) then run in parallel with explicit file ownership. M4 integrates both into host boundaries; M5 validates the combined tree. The integration agent owns context/runner/runtime/embed/REPL/backend files, shared artifacts and final commits. Worker agents must not overwrite others' changes. Independent evaluation uses a separate agent after implementation.

## Milestones

### ✅ M1: Shared execution ownership (~500 LOC)
**Dependencies:** None
**Estimate:** 500 LOC including regression tests.

- [x] Execution owner shares a two-second shutdown deadline across workers and rejects late admission safely.
- [x] Budget views share ownership; independent clones preserve policy with fresh Process/Stream registries.
- [x] Opaque non-reused IDs prevent cross-owner registry collisions; shutdown failures remain structured.

Files: new `internal/effects/worker_owner*.go`, context ownership hooks, policy-copy helper and supervisor tests.

### ✅ M2: Managed worker cancellation (~650 LOC)
**Dependencies:** M1
**Estimate:** 650 LOC including regression tests.

- [x] Closed stdin retains worker ownership until reaping; cooperative queued writes drain.
- [x] cancelProcess returns typed results, enforces Process authority, stops owned POSIX groups and joins writer/Wait.
- [x] Natural exit, repeated cancellation, concurrent write/close, stale/foreign handles and Windows/WASM unsupported results are tested.

Files: managed Process handlers/tracking, `std/process.ail`, cancellation builtin and tests. Example: `examples/runnable/process_cancel.ail`.

### ✅ M3: Async process source cancellation (~650 LOC)
**Dependencies:** M1
**Estimate:** 650 LOC including regression tests.

- [x] cancelProcessSource enforces Stream and Process authority and rejects non-process sources.
- [x] Async workers have one natural-exit Wait owner; final chunks drain and blocked consumers do not prevent cancellation.
- [x] Source selection borrows sources; cancellation stops descendants and joins owned readers without closing borrowed stdin/connections.

Files: async process source/handlers and source registry, `std/stream.ail`, source cancellation builtin and tests. Example: `examples/runnable/stream_process_cancel.ail`.

### ✅ M4: Runner and embedding lifecycle integration (~850 LOC)
**Dependencies:** M2, M3
**Estimate:** 850 LOC including regression tests.

- [x] CLI/batch normal, error, exit(7), budget failure and cancellation clean up owned workers while preserving primary outcomes.
- [x] SIGINT/SIGTERM restore terminal first, clean workers within the deadline, and retain 130/143 without signal-watcher deadlock.
- [x] Embedded requests, engine initialization/close and REPL reset/quit have explicit independent ownership; WebSocket close-code and borrowed-transport behavior is preserved.
- [x] Strict VM supports the new Process operations through explicit effect dispatch; unsupported generated/platform paths fail visibly.

Files: runner/batch, runtime request boundary, embed close/call, REPL lifecycle, terminal signal integration, VM/bytecode effect support and host regression tests. Preserve WebSocket-owned transports.

### M5: Regression evidence and independent evaluation (~750 LOC)
**Dependencies:** M4
**Estimate:** 750 LOC including regression tests.

- [x] Both original host-exit leak reproductions pass with exact PID/reaping evidence on the sprint build.
- [x] Provider-free blocking AI/effect, descendant, bystander, admission-race, 20-worker deadline and repeated-run resource controls pass.
- [ ] Design conflict fixtures, examples/manifest, goldens, relevant race/PTy tests, full tests, lint, formatting, file sizes and architecture boundaries pass.
- [ ] Independent evaluator reports a passing verdict; artifacts, changelog, documentation and release/consumer handoff are complete; no push or release is performed.

Files: integration/PTY fixtures, `examples/manifest.json`, affected stdlib/builtin goldens, docs/effects reference, changelog fragment, validation report and sprint retrospective.

## Day-by-Day Breakdown

- Day 1: failing supervisor tests, ownership/admission/deadline implementation, fresh policy-preserving clones.
- Days 2–3: managed stdin/cancellation and async output/reaping in parallel; failing tests first, focused race verification.
- Days 4–5: runner/batch and request/engine/REPL lifetimes; signal/terminal ordering; strict VM/backend behavior.
- Days 6–7: provider-free blocking-effect and exact PID/process-group integration evidence, repeated-run resource tests and docs/examples.
- Day 8: full quality gates, independent evaluation, fix feedback within the skill's bounded review rounds, final local commits.

## AILANG Syntax and Showcase Gate

AILANG prompt version loaded: **v0.16.7** (`ailang prompt` loaded before `.ail` edits).

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| `process_cancel.ail` | skip: OS cancellation is an external-effect guarantee, exercised by PID integration tests | include: main `! {Process, IO}` | include: a pure Result-to-status classifier with inline cases |
| `stream_process_cancel.ail` | skip: real subprocess stop/join is not a pure SMT property | include: main `! {Stream, Process, IO}` | include: a pure Result-to-status classifier with inline cases |

Existing conflict fixtures: process_stdin_write, stream_process_source, stream_multi_source, terminal_keys and process_demo. Exercise their intended behavior through focused fixture controls and existing PTY tests; do not mistake type-checking for live lifecycle validation.

## Quality Gates and Delivery Boundary

Baseline prerequisites: isolated clean tree, full `make test` and `make lint` before implementation. Milestone checkpoints use focused relevant tests and lint; final verification includes full repository tests, race tests, `make verify-examples`, `make fmt-check`, `make check-file-sizes` and `make check-boundaries`. Retain actual gate outputs and disclose baseline/environment limitations rather than declaring unrun checks passed.

Complete runtime cleanup and local consumer feasibility tests in this sprint. The official published-runtime consumer retest necessarily follows a future push/release and remains explicitly pending in the delivery handoff. Do not release or push during this execution: the user said the work will be pushed to dev once fixed. The current task includes local implementation, verification and independent evaluation.

## Progress

No milestone is complete until its acceptance criteria and verification evidence pass. Update only progress/timing/evidence fields in sprint JSON during execution; preserve requirements. The design and plan move together to implemented after an independent passing evaluation, with a truthful unreleased-delivery annotation.
