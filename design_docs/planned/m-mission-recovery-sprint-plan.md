# M-MISSION-RECOVERY — repository identity and durable attempts

**Status:** Approved for attended execution by Mark, 2026-09-07: “can you take care of that?”
**Parent:** [Mission runtime contract](m-mission-runtime-contract.md), slice 2 increment.
**Priority:** P0. **Estimate:** 3–4 days / approximately 1,200 implementation and test lines;
planning allowance, not an individual velocity measurement. Prior dispatch slice delivered
1,559 lines including tests and documentation in several commits. Work is isolated from live loops.

## Observed baseline and ownership

Canonical inbox read without acknowledgement: 20 unread. V1 owns #1074 (exec.go size) and
#1073 (notification regression, decision D-60). Motoko owns #1076/#1077. Do not duplicate
those changes or reinterpret their pending human decisions.

Current live thread pointers, confirmed open on GitHub: V1 ailang#1072, World
ailang-world#129, Motoko ailang#1078, Docs ailang#979. Thread rotation is communication
routing, not execution identity or approval authority. The older mission-gh-issue=745
is stale; production V1 now reads mission-v1-gh-issue. Do not rewrite any pointer here.

World's AILANG_DRIVER_PIN=0 is an intentional mitigation for a confirmed wrong-repository
redirect, not an installation mistake to erase. #1075 supplies a reviewed handoff and
acceptance criteria, including its correction to the vacuous flag test. Keep mitigation
until the pinned ref contains the fix and a controlled fire proves the target repository.

Recent slot tails (not an unbiased fleet rate): V1 4/6 crashed/killed; World latest completed;
Docs latest four completed; Motoko last three completed. These describe harness termination,
not accepted product progress. Threads corroborate parked work and provider transport issues.
Quota has two stages recorded, capacity unknown; measurement warmup is not a quota defect.

## Frozen contract

### Repository identity repair (#1075)

Driver code remains pinned. `_set_pin_workdir "$wt" "$src" || return 1` runs in the
current shell before setting the successful pin markers. It preserves a different repository's
incoming MISSION_WORKDIR and redirects a same-origin clone to the pin worktree. No incoming
workdir means the driver source is the work repository. Explicit de-fork flag 0/1 overrides
inference; invalid values fail loudly. Normalize URL transports and host case while preserving
repository path case. Unknown origins fail STALE without modifying the incoming path.
Tests cover opposite flag/inference answers, missing origins, SSH/HTTPS equivalents, distinct
clones of the same origin, and real re-exec. No installed env, launchd job or driver ref changes.

### Durable opt-in attempt state

Extend the existing coordinator SQLiteStore with additive mission-attempt tables and methods;
no generic tasks/status migration and no automatic coordinator finalization. A stage key is
(mission_id, work_item_id, stage_id). Exactly one attempt can occupy an unreconciled stage;
a different receipt path cannot bypass this admission. Request digest binds immutable inputs.

States: prepared -> running -> execution_completed | execution_failed. A lease expiring in
prepared can be reclaimed with a new fencing token because dispatch has not started. Expiry
in running becomes needs_reconciliation: never automatically run it again. Cancellation is a
terminal fence; stale completion cannot override it. Terminal duplicate completion is accepted
only when its owner token and exact outcome digest match. No state means artifact acceptance.

Persist state before executor dispatch and completion with a version-checked conditional write.
Separate heartbeat renews a finite lease. Use database time for lease decisions. Start marks
running before any executor is invoked; a crash between those two events is conservatively
ambiguous. No atomicity claim between SQLite and filesystem/remote side effects.

Expose opt-in role-run --state-db FILE integration and `mission attempt` status/reconcile/cancel
commands against an explicit DB. Existing role-run without --state-db retains slice-1 semantics.
Reconciliation marks expired running work ambiguous; it does not retry or accept artifacts.
Canonical weekly threads remain untouched; no untrusted inbox/event can approve or advance work.

### Reuse and verified premises

Read coordinator/store_sqlite.go (existing WAL DB, serialized connection pool), task_status_cas.go
(conditional status updates), finalization_ledger_store.go (task finalization is separate),
mission/dispatch/run.go (recorder before dispatch and completion), and launchd/lib/pin-root.sh
(unconditional workdir replacement). New mission records live alongside existing coordinator
records with their own additive schema, not in a competing external state store. Cloud store
implementation, resource reservations, approval migration and artifact acceptance are later
increments. This plan does not claim the full parent slice-2 fault matrix is complete.

## Milestones and acceptance

1. **M1 — Pin repository identity** (~250 lines): helper and regression fixtures; same-origin,
   cross-origin, explicit override, ambiguity, production re-exec. Run bounded pin suite.
2. **M2 — Durable coordinator attempts** (~600 lines): schema, prepare/claim/start/renew/finish/
   cancel/reconcile APIs; two-connection races, reopen recovery, stale tokens, duplicate/conflicting
   completion and stage conflicts. No live coordinator DB used during tests.
3. **M3 — CLI integration and engineering evaluation** (~350 lines): explicit state DB, live
   heartbeat, durable recorder ordering; abrupt subprocess store-recovery fixture plus in-process CLI ordering tests; status/reconcile/cancel,
   docs/example and independent review. Test interruption before/after start and duplicate runs.

M1 is independent of M2; M3 depends on M2. Use independent review after implementation, then
bounded full tests/lint/build/race/boundaries. Report inherited failures separately. All test
subprocesses have wall-clock ceilings and synthetic homes/databases. No provider spend.

Terminal stages remain occupied; this increment exposes no retry/reset operation. Cancellation
fences completion and requests cooperative stop on the next heartbeat. Store crash tests prove
process-exit persistence, not power-loss durability or distributed execution.

## Success and remaining work

- [x] World cannot be redirected into the driver repository when origins differ.
- [x] Duplicate role execution cannot bypass a persisted active stage with a new receipt path.
- [x] Expired running work, cancellation and stale completion retain their fences after restart.
- [x] CLI lifecycle fixture, tests, documentation and independent review pass.

Live canary requires an explicit controlled invocation using the tested artifact and an isolated
worktree, with spend and rollback set before activation. Do not silently cut over launchd or
restore World's pin using a local commit absent from its configured ref. Mission onboarding,
verified artifact handoffs, quota reservations and remote workers remain subsequent milestones.

## Delivery evidence

M1/M2/M3 complete in the isolated recovery branch, 2026-09-07. Independent engineering
review PASS after adding a real lease-renewal/lost-ownership cancellation regression.
81 pin shell assertions, full make test/lint, build, architecture boundaries, race and vet
pass. File-size check reports only inherited exec.go at 807 lines (V1 owns #1074).
Built binary with a fake Pi executable completed one role, persisted matching receipt/state,
refused a second receipt path, and kept dry-run free of state writes. No provider invoked.
See [dated fleet baseline](../verification/mission-recovery-2026-09-07/baseline.md).

This is an implemented opt-in increment, not completion of the parent mission runtime.
World mitigation and live thread pointers remain intact. The reviewed pin commit must
reach the configured driver ref before World can verify and restore pinning.
