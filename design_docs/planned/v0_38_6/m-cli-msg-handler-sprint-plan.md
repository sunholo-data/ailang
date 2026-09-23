# Sprint Plan: M-CLI-MSG-HANDLER

## Summary

Make the existing `Msg` effect usable from native `ailang run` by adapting the selected message-plane store to `effects.MsgHandler`, wiring it only when `Msg` is granted, and preserving deterministic poll-once/drain-once batch semantics.

**Design:** `design_docs/planned/v0_38_6/m-cli-msg-handler.md`

**Issue:** #1130

**Duration:** 2.5 engineering days

**Estimated change:** ~550 LOC (implementation, tests, example, and docs)

**Risk:** Medium

**Dependencies:** Existing `effects.MsgHandler`, `messaging.MessageStore`, and storage-plane configuration; no new external dependency

## Current Status and Planning Evidence

- The design is approved and explicitly audits the systemic surface: single-run, batch, REPL/serve parity, capability denial, WASM/stub handlers, store selection, and `ailang messages` regressions.
- A fresh inspection at v0.42.0 confirms the defect remains: native contexts grant `Msg` but never assign `EffContext.Msg`.
- Since the v0.38.6 design was written, common execution moved into `internal/runner`; message-plane resolution still lives in `cmd/ailang/messages.go`. The implementation must share that resolver rather than copy it into the runner.
- Recent history spans only one active commit day, so raw LOC/day is not representative. The estimate uses the approved 2–3 day design estimate and reserves roughly 30% for integration and architecture drift.
- The worktree was clean when this plan was created.

## Architecture Constraints

1. Keep `--caps Msg` / `--caps auto` as the authority boundary. Do not open a store or install a handler without the granted capability.
2. Extract the message-plane target/open logic into a dependency-safe internal package used by both `ailang messages` and the runner. Do not duplicate environment precedence.
3. Preserve the selection contract: `AILANG_STORAGE_MESSAGING` > `AILANG_STORAGE` > local, with `AILANG_MESSAGES_PROJECT` taking project precedence. Retired variables remain hard errors.
4. Do not change inbox persistence semantics under `internal/messaging`; the adapter consumes the interface.
5. Native `Recv` is one non-blocking oldest-unread poll; native `Subscribe` synchronously drains current unread rows and returns a no-op cancel.
6. Keep WASM and `StubMsgHandler` behavior unchanged. Preserve `ErrNoMsgHandler` identity if its guidance text is updated.
7. Ensure every opened SQLite/Firestore store is closed exactly once, including startup and batch error paths.

## Milestones

### M1 — Shared plane opener and store-backed adapter

**Estimate:** 0.75 day; ~120 implementation + ~130 test LOC

**Dependencies:** None

**Files (expected):**

- New shared resolver/opener under `internal/` (exact package chosen to preserve import boundaries)
- `cmd/ailang/messages.go`
- New native adapter near the runner/effects boundary
- Adapter and resolver tests

**Tasks:**

- Move the current message target resolution and store opening into a shared, error-returning helper; keep command output descriptions and selection behavior stable.
- Implement `StoreMsgHandler` over `messaging.MessageStore` with the approved field mapping.
- Implement send, oldest-unread receive plus read-marking, and ordered drain-once subscription.
- Define ownership/cleanup explicitly so adapter or caller closes the store once.
- Add hermetic SQLite-backed tests for mapping, ordering, empty mailbox, callback order, read marking, and backend errors.

**Acceptance criteria:**

- [ ] `ailang messages` and native runtime resolve the same backend for every local/hybrid/GCP configuration fixture.
- [ ] Empty receive returns `effects.ErrNoMsgAvailable` without waiting.
- [ ] Subscribe visits each initially unread message once in oldest-first order, marks it read, and returns a safe cancel function.
- [ ] Send produces a store row with correct sender, inbox, payload, ID, and millisecond clock mapping.
- [ ] Bad storage configuration is returned as an error; no fallback store is opened.
- [ ] Focused resolver and adapter tests pass with `go test`.

**Risk:** Extracting command-local storage code can accidentally change CLI behavior. Mitigate with table tests over environment precedence and a before/after command regression fixture.

### M2 — Capability-gated runner wiring across execution paths

**Estimate:** 0.75 day; ~90 implementation + ~100 test LOC

**Dependencies:** M1

**Files (expected):**

- `internal/runner/run.go`
- `internal/runner/batch.go`
- `internal/runner/handlers.go` or a focused new Msg handler setup file
- `internal/effects/msg.go` (guidance/comment only)
- Runner tests

**Tasks:**

- Add an injectable message-store factory/setup path to the runner so production uses the shared opener and tests remain hermetic.
- Install `effects.NewMsgContext(handler, "cli_run")` only after capability grants and only when `Msg` is present.
- Cover single execution and every per-item batch context; avoid reopening the backend unnecessarily and close it exactly once.
- Keep denied-capability behavior ahead of any store access.
- Update stale native-handler guidance while preserving the sentinel error and typed classification.

**Acceptance criteria:**

- [ ] Single and batch runs with granted `Msg` receive a configured handler.
- [ ] A run without `Msg` neither opens the store nor installs the handler, and Msg operations remain capability-denied.
- [ ] `--caps auto` initializes the handler when the inferred entrypoint row includes `Msg`.
- [ ] Store-open failure occurs before user code executes, names the relevant storage/project configuration, and exits non-zero.
- [ ] Batch setup shares or safely scopes the backend without leaks or double-close.
- [ ] Existing WASM and stub-handler tests remain unchanged and green.

**Risk:** Setup duplicated between single and batch execution could drift. Mitigate by using one setup helper and testing it through both paths.

### M3 — CLI parity, executable example, documentation, and end-to-end gates

**Estimate:** 1 day; ~35 implementation/docs/example + ~75 integration-test LOC

**Dependencies:** M2

**Files (expected):**

- Native REPL setup and any `serve` capability path discovered by the executor
- `cmd/ailang/help.go` / run help tests
- `std/cognition.ail` comments
- `examples/msg_roundtrip.ail` or a send/receive pair
- CLI integration tests

**Tasks:**

- Wire native REPL/serve paths where they grant `Msg`; document any surface intentionally excluded with a test proving current behavior.
- Update Msg help and cognition API comments with poll-once/drain-once semantics and message-plane selection.
- Before editing `.ail`, run `ailang prompt`; type-check the example with `ailang check`.
- Add a hermetic local-store end-to-end test: send through `ailang run`, observe through the messages store/CLI, then receive through `ailang run`.
- Add loud-failure and capability-denial integration cases.
- Run focused tests, `make test-core`, `make test`, `make lint`, and `make check-boundaries`.

**Acceptance criteria:**

- [ ] `ailang run --caps IO,Msg` sends a row visible through the same local message plane.
- [ ] A second native run receives that row; an empty mailbox yields typed `no_message` and does not hang.
- [ ] Invalid store mode and missing GCP project/credentials fail before program execution with non-zero status.
- [ ] The checked-in Msg example type-checks and runs against an isolated local store.
- [ ] `ailang messages` output/behavior regression fixture is unchanged.
- [ ] REPL/serve parity is tested, or any deliberate exclusion is explicitly documented and guarded.
- [ ] Full tests, lint, and architecture boundaries pass.

**Risk:** GCP credential failures are environment-dependent. Unit-test injected opener failures and keep any real-project smoke manual/non-CI.

## Execution Order

### Day 1

- Complete M1 with tests first.
- Start M2 shared setup and prove no-capability/no-store-access behavior.

### Day 2

- Finish M2 single/batch wiring and cleanup tests.
- Implement M3 REPL/serve parity, help, cognition comments, and the checked example.

### Day 2.5

- Complete end-to-end local-store tests and loud-failure coverage.
- Run repository gates and record deviations in the sprint JSON rather than silently reducing scope.

## Success Metrics

- Issue #1130 reproduction succeeds under explicit and auto-inferred Msg capability.
- Zero store opens occur without Msg authority.
- Poll/drain tests are bounded and deterministic; no test relies on sleeps.
- At least one checked `.ail` example demonstrates the native message round trip.
- No behavior change to existing `ailang messages`, WASM Msg, or stub Msg paths.
- All focused and repository-wide quality gates are green.

## Deferred Scope

- Live subscription/Pub/Sub listening and blocking receive timeouts.
- A `--msg-self` flag; native sender remains the stable `cli_run` identity.
- New message operations, envelope fields, Lamport-clock persistence, or message-store schema changes.
- Budget plumbing beyond the existing honest unbounded/default representation.

## Handoff

After human approval, invoke `sprint-executor` with `.ailang/state/sprints/sprint_M-CLI-MSG-HANDLER.json`. The executor should begin by rechecking import boundaries and current REPL/serve wiring because these were the principal areas of drift from the v0.38.6 design.
