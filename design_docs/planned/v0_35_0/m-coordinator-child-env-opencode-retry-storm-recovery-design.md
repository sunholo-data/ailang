# M-COORDINATOR-CHILD-ENV-OPENCODE-RETRY-STORM-RECOVERY: coherent routes, OpenCode-local pinning, generation-and-attempt-fenced dispatch

**Status**: Needs Human Review — D-50 approved execution, but the single permitted re-quorum remained BLOCKED
**Target**: v0.35.0
**Priority**: P0 — one incoherent route produced 41 dispatch failures and nine operator messages
**Estimated**: 5 days / 40 hours, including two-store transaction tests and mutation evidence
**Dependencies**: Fresh quorum PASS; isolated Firestore-emulator CI lane. Canary rollout has a
separate operator-authority gate.

## Recovery Authority and Quorum Trigger

This document is the self-contained canonical design for the recovery sprint. It supersedes
the implementation scope in the recovered broad design and sprint plan while preserving their
incident evidence. Human directive D-50 approves this narrowed recovery and authorizes
`execute sprint`; it does not waive the fresh-quorum condition recorded by iteration 305.

Fresh quorum is mandatory because this design changes shared SQLite and Firestore state
transitions, the coordinator-to-Cloud-Run boundary, and the completion wire contract. The
recovered round-2 quorum was explicitly BLOCKED. Neither its prose carve-out nor D-50 converts
that old verdict into a pass. Review this document as a new artifact; any present reviewer or
controller rejection blocks planning/execution.

The following recovered artifacts are evidence only:

- Commit `3500db0a7`: M1 route implementation candidate. Transplant production/test hunks only;
  do not reintroduce its superseded broad design and sprint-plan files.
- `/Users/voightkampff/dev/sunholo-data/.wt-iter302`: parked M2-M4 implementation evidence. Do
  not stage, cherry-pick, or use it as an implementation base.
- The old five-adapter executable design: superseded for this sprint by OpenCode-only M2a.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | +1 | One normalized agent route plus the current generation and unique reserved attempt identity determine every accepted completion. |
| A2: Replayability | +1 | Persisted generation, attempt count, current reserved attempt identity, timestamp, and last error explain every retry decision. |
| A3: Effect Legibility | 0 | No AILANG effect-system change. |
| A4: Explicit Authority | +1 | The selected OpenCode child uses one pinned absolute executable; dispatch effects require a durable reservation. |
| A5: Bounded Verification | +1 | Three reservations, fixed deadlines, and named transaction/race fixtures are finite gates. |
| A6: Safe Concurrency | +1 | Generation-and-attempt-aware terminal CAS prevents stale writers, prior runs, and superseded same-generation attempts from winning. |
| A7: Machines First | +1 | Typed permanent/retryable outcomes replace log-string inference and silent fallback. |
| A8: Minimal Syntax | 0 | No language syntax change. |
| A9: Cost Visibility | +1 | Coordinator dispatch attempts are persisted and bounded before paid Cloud Run effects. |
| A10: Composability | +1 | One immutable route feeds audit, image, model, environment, and budget selection. |
| A11: Structured Failure | +1 | Route, reservation, retry, lost-race, and terminal-winner outcomes are explicit. |
| A12: System Boundary | +1 | Config publication, RunJob, child health, and completion ingestion validate boundary data. |

**Net score: +10** — eligible for implementation after quorum.

### Hard Violation Check

- [x] A1 determinism: no implicit nondeterminism is introduced.
- [x] A3 effects: no hidden AILANG effect is introduced.
- [x] A4 authority: ambient executable and external-call authority are reduced, not widened.
- [x] A7 machines first: every failure/ownership decision has a structured outcome.

## Problem Statement

Task `task-a0628a5f` did not fail because a local daemon lost `PATH`. The coordinator selected
the provider and Cloud Run image from different authorities:

```text
agent sprint-planner: provider=codex, executor_variant=codex
coordinator default_provider: opencode

AILANG_PROVIDER       <- global default       = opencode
Cloud Run job variant <- per-agent variant    = codex

result: codex image starts the OpenCode executor, whose binary is absent
```

Every synchronous Cloud Run readiness failure then unconditionally reset the task from queued
to pending. The next daemon poll repeated the external call. Separately, stale detection and
completion handling performed read-then-unconditional terminal writes and notified without
owning an atomic transition. First-party incident evidence recorded 41 dispatch failures,
eight stale-terminalization attempts, and nine operator messages.

The recovered design correctly identified immutable route authority, pre-effect reservations,
bounded deadlines, and terminal CAS. Its executable milestone was too broad: it migrated
OpenCode, Codex, Pi, Motoko, and Claude despite the measured child failure involving OpenCode
only. Its retry premise was also incomplete:

- `RequeueTask` is not operator-only. Task-chain stage transitions call it automatically.
- `RetryAllFailedTasks` is a second deliberate retry path.
- A delayed completion has only `task_id`; after a requeue it can terminalize the new run.
- Generation alone is also insufficient: crash/timeout recovery can authorize a later RunJob in
  the same generation, and an earlier ambiguously accepted job may then complete out of order.
- A generic stale `UpdateTask` write must not refund durable dispatch state.

Therefore terminal authority is the pair **`{dispatch_generation,
dispatch_reservation_id}`**, not the task ID or generation alone. Every deliberate
re-execution starts a new generation; automated retries remain within the current generation,
receive a fresh unique reservation/attempt identity, and share its three-attempt bound.

## Goals

**Primary goal:** one coherent per-agent route reaches at most three reserved Cloud Run calls
per execution generation, uses the selected OpenCode binary consistently, and permits only the
currently authorized `{generation, reservation}` pair to win completion and attempt
notification.

**Success metrics:**

1. The historical global-opencode / agent-codex configuration dispatches `codex/codex`; every
   invalid provider/variant pair causes zero reservations, audit publishes, and RunJob calls.
2. OpenCode health and execution use the same absolute executable after `PATH` changes, while
   missing lookup results are not cached.
3. Twenty retryable polls, any daemon restart, and every named crash point produce at most
   three reservations and three RunJob calls in one generation.
4. Single retry, bulk retry, and task-chain stage advance start a new attempt budget and fence
   delayed completions from every older generation.
5. Within one generation, a later reservation transaction revokes the prior reservation's
   completion authority; conflicting completions from two attempts have a deterministic winner
   independent of arrival order.
6. Any stale/dispatch/completion race yields one terminal state/result and at most one
   notification attempt by the store-CAS winner.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1. Per-agent `{provider, executor_variant, model}` is the immutable route; global provider is inheritance only. | Prevents provider/image/model/budget split authority. | agent | design | high |
| D2. Validate the route at config publication and immediately before RunJob. | Invalid configuration must fail before spend even for non-daemon callers. | agent | design | high |
| D3. Pin only OpenCode in an OpenCode-package-local resolver. | Fixes the measured child boundary without freezing a shared five-adapter lifecycle/security API. | human via D-50 | design | high |
| D4. The retry budget is three durable reservations per dispatch generation. | Makes the external-call bound survive restarts and deliberate task reuse. | agent | design | high |
| D5. Completion authority is the exact current `{dispatch_generation, dispatch_reservation_id}` pair propagated end-to-end. | Prevents both prior-generation and superseded same-generation attempts from owning the result. | agent | design | high |
| D6. Dispatch-owned fields are mutated only by transaction APIs and deliberate new-generation APIs. | Generic stale writes must not refund attempts. | agent | design | high |
| D7. Terminal-CAS winner owns at most one notification attempt; delivery is not claimed exactly once. | Avoids duplicate messages without falsely claiming a transactional outbox. | agent | design | high |
| D8. Audit=10s, RunJob=60s, child health=15s, reservation lease=2m; parent cancellation is preserved. | A bounded call count is meaningless if a call outlives its lease or daemon shutdown. | agent | design | high |

### Design Freeze

- [x] Route authority and compatibility matrix are fixed by D1/D2.
- [x] Executable work is OpenCode-only and package-local by D3.
- [x] Attempt count is per generation; three means three total reservations, not three retries.
- [x] Requeue, bulk retry, and stage advance are deliberate new-generation operations.
- [x] Old-generation and superseded same-generation completions are rejected by exact
  generation-plus-reservation CAS.
- [x] Generic update paths preserve dispatch-owned fields.
- [x] Notification guarantee is winner-owned at-most-one attempt, not exactly-once delivery.
- [x] Deadline and lease constants are fixed by D8.

## Solution Design

### 1. Immutable route authority

Add `ExecutionRoute` and typed `PermanentDispatchError` in `internal/coordinator`. Resolve the
route once:

1. Find the agent named by `task.AgentID`; missing/mismatched registration is permanent.
2. Use `agent.Provider`, after config loading has inherited `default_provider` when absent.
3. Normalize empty executor variant only for legacy Claude to `default`.
4. Resolve the model through the existing `ResolveModel(agent)` path.
5. Validate this Cloud Run matrix:

| Provider | Allowed variants |
|---|---|
| `claude` | `default`, `go` |
| `gemini` | `gemini`, `gemini-go` |
| `codex` | `codex`, `codex-go` |
| `opencode` | `opencode` |
| `pi` | `pi` |
| `motoko` | `motoko` |

`eval`/`eval-go` remain dedicated job variants, not general provider names. Local-lane agents
remain outside the Cloud Run matrix. Unknown or off-diagonal pairs are permanent errors; no
fallback job is allowed.

Config publication applies defaults in memory and validates literal plus pipeline-expanded
cloud agents before the GCS CAS write. Default application and route validation must handle a
null agent without panic: normalization skips nil and validation returns a named error before
pipeline expansion dereferences it.

`dispatchTasksCloud` builds audit provider, `DispatchParams`, model, variant, and provider budget
from the route. `cloudrun.Dispatcher.Dispatch` repeats compatibility validation before job-name
construction and RunJob. A route error goes through terminal CAS fenced by the current generation
and empty pre-dispatch reservation before any reservation or external call.

### 2. OpenCode-only pinned executable and child-local preflight

Add an unexported, concurrency-safe resolver in `internal/executor/opencode`:

```go
type pinnedExecutable struct {
    configured string
    // mutex + cached successful absolute path
}

func (e *pinnedExecutable) Resolve() (string, error)
```

Contract:

- Empty configuration is an error.
- Absolute paths are cleaned, statted, required to be regular files and, on Unix, executable.
- Bare names use `exec.LookPath`; a non-absolute result is rejected and the same file checks
  apply.
- Only success is cached. A later install may recover after a failed lookup.
- OpenCode `HealthCheck` and `ExecuteStreaming` both call the same resolver instance and pass
  only its returned absolute path to `exec.CommandContext`.
- The global executor factory returns the same cached OpenCode executor to child preflight and
  later execution, so the health-approved path remains pinned.

The `execute-job` child obtains the executor and performs `HealthCheck` with a 15-second maximum
before credential mutation, plugin clone, repository clone, or AGENTS injection. Use
`context.WithTimeout(parent, 15s)`, preserving an earlier parent cancellation/deadline.

The boundary is deliberately stated precisely: missing OpenCode is a typed permanent failure
**inside the selected child before clone/plugin effects**. The coordinator has already made an
accepted RunJob call and consumed its reservation; missing-child acceptance must not claim zero
RunJob calls. Cloud Run `maxRetries=1` stays unchanged, so the preflight may execute twice inside
that one coordinator attempt, but both executions stop before repository/plugin effects.

No shared `internal/executor.Executable` API is introduced. Codex, Pi, Motoko, Claude, managed
agents, and git resolution do not change.

### 3. Generation-and-attempt-fenced durable dispatch state

Add backward-compatible fields to `TaskRecord`, SQLite, and Firestore:

```text
dispatch_generation       integer, default 0
dispatch_attempts         integer, default 0
dispatch_reservation_id   string, default ""
dispatch_reserved_at      timestamp, nullable
last_dispatch_error       string, default ""
```

`dispatch_reservation_id` is both the pre-effect lease token and, after RunJob acceptance, the
unique completion-authority token for that attempt. It is not cleared on a successful RunJob
response. `dispatch_reserved_at != nil` means the coordinator has not durably recorded whether
RunJob was accepted; `dispatch_reserved_at == nil` with a non-empty reservation ID means that
specific accepted child attempt is the only completion authority.

Generation zero is the migration/initial-task generation. `DispatchParams` adds
`DispatchGeneration`, `DispatchReservationID`, and `DispatchAttempt`. Every new coordinator
dispatch serializes them as `AILANG_DISPATCH_GENERATION`,
`AILANG_DISPATCH_RESERVATION_ID`, and `AILANG_DISPATCH_ATTEMPT`; the selected child copies all
three into `pubsub.TaskCompletion`. The generation-plus-random-ID pair is opaque authority;
the attempt ordinal is additionally required to equal the stored `dispatch_attempts` value for
nonlegacy completion, but it is never accepted as a substitute for the random ID.

Migration behavior is exact:

- an old task and old in-flight completion with no new fields are treated as
  `{generation: 0, reservation: "", attempt: 0}` and may match only while the stored task has
  that same legacy fence;
- a new generation-zero dispatch has a non-empty reservation, so a missing-ID legacy
  completion cannot match it;
- any completion with generation greater than zero and an empty reservation ID, or any
  non-empty reservation with attempt ordinal less than one, is malformed, acknowledged to avoid
  a poison retry loop, and logged as rejected;
- any deliberate re-execution increments generation and clears reservation, fencing every old
  pair before a new reservation is created.

Add these store operations with identical SQLite/Firestore semantics:

```go
ReserveDispatchAttempt(ctx, id, maxAttempts, now)
    (DispatchReservation{ID, Generation, Attempt, ReservedAt}, DispatchOutcome, error)

RecordDispatchOutcome(ctx, id, generation, reservationID, err, retryable)
    (DispatchOutcome, error)

RecoverExpiredDispatchReservation(ctx, id, generation, now, lease, maxAttempts)
    (DispatchOutcome, error)

FinishTaskIfActive(ctx, id, TerminalFence{Generation, ReservationID, Attempt}, finish)
    (won bool, error)
```

`ReserveDispatchAttempt` accepts only a pending task. In one transaction it checks the bound,
increments attempts, creates a random reservation ID, stores the timestamp, and changes status
to queued. Only after commit may audit publish or RunJob occur.

`RecordDispatchOutcome` requires queued state plus matching generation and reservation ID:

- success: clear only `dispatch_reserved_at`, retain `dispatch_reservation_id` as completion
  authority, and remain queued awaiting completion;
- retryable error with attempts below three: store error, clear reservation, return pending and
  `WillRetry`;
- permanent error or third failed attempt: terminalize failed and return `WonTerminal`;
- mismatched generation/reservation or non-queued state: no mutation, `LostRace`.

Expired reservation recovery also matches generation and reservation and consumes the recorded
attempt. Before expiry, that reservation remains the sole authority and a matching completion
may win. At expiry, the recovery transaction atomically revokes the ID: attempts one/two clear
the pair and return pending; attempt three terminalizes. Only after the revocation commit may a
later reservation create a fresh ID. From that commit onward, a completion from the ambiguous
old RunJob is ineligible even if no replacement reservation has yet been made. A later
reservation supersedes the old one by installing its new ID transactionally.

This gives deterministic completion ownership within one generation. If attempt 1 is
ambiguously accepted, expires, and attempt 2 is reserved, attempt 1's completion always loses
against the stored attempt-2 ID regardless of arrival order; only attempt 2 may win. The outbound
boundary remains bounded at-least-once rather than exactly once: physical work from the revoked
attempt may finish, but it cannot own task state, and the generation cannot exceed three RunJob
calls.

Dispatch fields have a single-writer contract:

- transaction APIs above own reservation/attempt/error mutation;
- `RequeueTask` and `RetryAllFailedTasks` atomically increment generation and clear all four
  per-generation fields because they intentionally start new work;
- task-chain stage transitions already call `RequeueTask`, so they receive the same fresh budget;
- local-lane `ResetTaskToPending` neither clears nor increments cloud dispatch state;
- generic SQLite/Firestore `UpdateTask` must omit/preserve all five dispatch-owned fields.

### 4. Bounded external effects and typed retry classification

The order is fixed:

```text
resolve + validate route
  -> reserve durable attempt
  -> audit publish under 10s maximum
  -> RunJob under 60s maximum
  -> record outcome for generation + reservation
```

Each timeout uses the caller as parent; do not use `context.Background` or
`context.WithoutCancel`. The reservation lease is two minutes, leaving one minute after the
maximum RunJob duration for result persistence and polling margin. Tests inject durations,
clocks, and blocking channel fakes; they do not sleep for production intervals.

Retry classification uses only `status.Code(err)`:

- retryable: `Unavailable`, `ResourceExhausted`, `DeadlineExceeded`, and the measured Cloud Run
  readiness `FailedPrecondition`;
- permanent: `InvalidArgument`, `Unauthenticated`, `PermissionDenied`, all unrecognized codes,
  route/config errors, and invalid local call state.

No substring matching and no untested REST-client inference are allowed. If a future caller uses
the generated REST client, it requires a separate typed `*googleapi.Error` design/test amendment.

### 5. Generation-and-reservation-aware terminal CAS and notification ownership

`FinishTaskIfActive` transactionally checks:

- expected generation equals stored generation;
- expected reservation ID exactly equals the stored current completion-authority ID;
- for a nonlegacy fence, expected attempt ordinal equals stored `dispatch_attempts`;
- current status is one of the caller's allowed active states;
- desired target is `failed`, `completed`, or `pending_approval`.

The stale detector passes the generation, reservation ID, and attempt it observed and notifies
only when the CAS returns `won=true`. Completion handling passes
`TaskCompletion.DispatchGeneration`, `TaskCompletion.DispatchReservationID`, and
`TaskCompletion.DispatchAttempt`; a duplicate, terminal, malformed-ordinal, prior-generation,
or superseded same-generation completion is acknowledged and logged without result overwrite or
notification. Dispatch outcome and expired-reservation recovery already match the pair, own
their terminal writes, and return `WonTerminal`; only that outcome attempts a route/dispatch
failure message.

A permanent route error occurs before the next reservation. It uses an explicit pre-dispatch
fence of the current generation, empty reservation ID, and the current stored attempt count
(which can be nonzero after an earlier retryable outcome), and may win only while the task
remains pending with that exact fence. Legacy completion compatibility uses the zero/empty/zero
fence only for generation zero and a queued/running legacy task; the allowed source-state sets
keep the two operations distinct.

The guarantee is **one store transition winner and at most one notification attempt**. Message
delivery remains at-least-once and can be zero if the process dies after CAS but before insertion.
A transactional outbox is outside this sprint; documentation and logs must not use
“exactly-once notification delivery.”

## Conflict Surface

This is not a parser/compiler change, but it crosses five concurrency and compatibility surfaces.

| Surface | Existing valid behavior | New rule | Regression guard |
|---|---|---|---|
| Config normalization | Empty provider inherits global; empty Claude variant means default; local agents may use host-specific values. | Inheritance happens once, then immutable cloud matrix validation; null configs fail without panic. | Complete matrix, local exemption, pipeline expansion, null-agent test. |
| Task reuse | `RequeueTask` is used by CLI retry and automatic task-chain stage transitions; `RetryAllFailedTasks` bulk-retries failures. | Each deliberate reuse increments generation and resets its attempt budget. | Single, bulk, and both task-chain stage fixtures. |
| Generic task update | SQLite updates selected fields; Firestore merges `taskToMap`. | Generic update cannot mutate any dispatch-owned field. | Stale snapshot update during an active reservation preserves generation/attempt/reservation/error. |
| Completion wire | Existing messages carry task ID but no generation, reservation identity, or attempt ordinal. | New jobs carry all three; all missing means only the legacy zero/empty/zero fence. | Legacy compatibility, old-generation rejection, ordinal validation, and conflicting same-generation attempt tests. |
| Terminal writers | Dispatch error resets pending; stale/completion paths write and notify independently. | Reservation outcome or exact generation-plus-reservation CAS names the sole terminal/notification owner. | Barrier-controlled stale/dispatch/completion races. |
| Child CLI selection | OpenCode health uses LookPath but execute uses the configured name again. | Same OpenCode-local absolute path is used for health and execution. | Poison-PATH and factory-instance tests. |
| Local lane | `daemon_tasks_exec_run.go` calls `ResetTaskToPending` specifically after `errors.Is(wtErr, ErrWorktreeLimitReached)`. The current `worktree_test.go` proves the manager returns a limit error but does not assert the daemon/store reset. | Local reset remains independent and cannot alter cloud state. | Retain the manager fixture and add a daemon-level reset/preservation test. |
| Deployment compatibility | Old Firestore documents and in-flight jobs lack new fields. | Missing task fields read as zero/empty; legacy completion is accepted only against the stored zero/empty pair. | Old-document conversion and rollout fixture. |

Deliberate incompatibility: publishing an off-diagonal or unknown cloud provider/variant pair now
fails instead of silently selecting a job. No local-lane configuration is rejected by this matrix.

## Examples

### Historical route split

```text
input:  global default=opencode, planner provider=codex, variant=codex
route:  {provider: codex, variant: codex, model: resolved planner model}
effect: audit provider=codex, job suffix=-codex, AILANG_PROVIDER=codex
```

Changing the global default cannot change any field after the route is resolved.

### Retry budget within one generation

```text
generation 4, attempt 1 -> FailedPrecondition -> pending
generation 4, attempt 2 -> DeadlineExceeded  -> pending
generation 4, attempt 3 -> FailedPrecondition -> failed, terminal winner notifies
next daemon polls        -> no reservation, no RunJob
```

An authorized retry or task-chain stage advance changes generation to 5 and starts attempts at
zero. Lease expiry never decrements an attempt.

### Delayed prior completion

```text
task completed {generation: 4, reservation: A}
RequeueTask -> pending {generation: 5, reservation: ""}
new dispatch -> queued {generation: 5, reservation: B}
redelivered completion {4, A} -> LostRace, ack, no overwrite or notification
completion {5, B}             -> may win terminal CAS
```

### Conflicting attempts within one generation

```text
generation 5, reservation A -> RunJob acceptance ambiguous
lease A expires             -> transaction revokes A, returns pending
generation 5, reservation B -> reserved and propagated to child B
completion A arrives first  -> LostRace (stored authority is B)
completion B arrives later  -> may win terminal CAS
```

Reversing the arrival order does not change the authoritative result: B is the only pair allowed
to win after its reservation transaction commits.

## Milestones

### M1 — immutable route authority (~1 day)

- Port code/test hunks from `3500db0a7`, excluding its design/sprint documents.
- Add nil-safe defaulting and publication validation.
- Use the route for audit, RunJob params, model, image variant, environment, and budget.
- Terminalize invalid routes through the current-generation/empty-reservation CAS path with zero
  external effects.

### M2a — OpenCode-local pinning and child preflight (~0.75 day)

- Implement and test the package-local success-only resolver.
- Use it in OpenCode health and execution.
- Perform 15s-max health preflight before clone/plugin work in `execute-job`.
- Add a source/diff guard proving no other executor adapter changed.

### M3 — two-store generation and reservation state machine (~1.5 days)

- Add five fields, SQLite migration/scans, and Firestore conversion.
- Implement reservation/outcome/recovery/terminal CAS transactions in both stores.
- Make single retry, bulk retry, and stage advance start a new generation.
- Make generic update and local reset preserve dispatch-owned fields.
- Add isolated Firestore-emulator CI with named PASS evidence; a CI skip is failure.

### M4 — bounded integration and notification ownership (~1.25 days)

- Propagate generation plus reservation/attempt identity through dispatch params, Cloud Run
  environment, child completion, and handler CAS.
- Reserve before audit/RunJob; enforce 10s/60s boundaries and typed retry classification.
- Route stale, completion, dispatch, and reservation-expiry writers through winner outcomes.
- Add 20-poll, restart, crash-point, prior-generation, superseded same-generation, and
  barrier-race fixtures.

### M5 — mutation evidence, regression gates, and rollout handoff (~0.5 day)

- Kill each named semantic mutant and restore from explicit temporary backups.
- Run targeted packages, full tests, lint, boundaries, and focused race tests.
- Update the current changelog.
- Prepare canary instructions; do not deploy or write live config without separate authority.

## Files to Modify/Create

### M1 route

- `cmd/ailang/coordinator_config.go`, `cmd/ailang/coordinator_config_test.go`
- `internal/coordinator/agent_config.go`
- `internal/coordinator/daemon.go`, `daemon_tasks_init.go`, `daemon_tasks_budget.go`,
  `daemon_tasks_exec.go`
- New `internal/coordinator/execution_route.go`, `execution_route_test.go`
- `internal/dispatch/cloudrun/dispatcher.go`, `dispatcher_test.go`

### M2a OpenCode only

- New `internal/executor/opencode/executable.go`, `executable_test.go`
- `internal/executor/opencode/opencode.go`, `opencode_test.go`, `opencode_streaming_test.go`
- `cmd/ailang/coordinator_cloud.go`, `coordinator_cloud_executor.go`
- New `cmd/ailang/coordinator_cloud_executor_test.go`

### M3/M4 durable state and completion fence

- `internal/coordinator/cloud_dispatcher.go`, `store.go`, `store_sqlite.go`,
  `store_sqlite_queries.go`
- New `internal/coordinator/dispatch_state.go`, `dispatch_state_test.go`,
  `store_sqlite_dispatch.go`, `store_sqlite_dispatch_test.go`,
  `cloud_dispatch_state_test.go`, `terminal_cas_test.go`
- `internal/coordinator/daemon.go`, `daemon_tasks_exec.go`, `stale_task_detector.go`,
  `pubsub_completion_handler.go`
- New `internal/coordinator/stale_task_detector_test.go`,
  `pubsub_completion_handler_test.go`, and `daemon_tasks_exec_run_test.go` (the repository has
  no existing test files for these three paths)
- `internal/coordinator/mock_store_test.go`, `approval_watcher_mock_test.go`
- `internal/storage/firestore/coordinator_convert.go`, `coordinator_tasks.go`,
  `coordinator_transitions.go`, `coordinator_approvals.go`
- New `internal/storage/firestore/coordinator_dispatch.go`, `coordinator_dispatch_test.go`
- `internal/pubsub/topics.go`, `pubsub_test.go`
- `.github/workflows/ci.yml`
- `changelogs/v0.32-current.md`

Files needed only to satisfy a mechanically expanded `Store` test mock may be added, but no
production package outside the listed surface may change without pausing for design review.

## Acceptance Criteria

### Route

- [ ] Historical `default_provider=opencode` plus planner `codex/codex` produces a codex job,
  `AILANG_PROVIDER=codex`, the resolved model, and codex-specific budget.
- [ ] Every allowed matrix cell passes; every off-diagonal/unknown cell returns typed permanent
  error naming agent/provider/variant before reservation, audit, or fake RunJob.
- [ ] Empty variant normalizes only for Claude; local-lane agents remain exempt.
- [ ] A config containing a null agent returns an error and does not panic in defaults,
  expansion, publication validation, or direct route validation.
- [ ] If configuration becomes invalid after one retryable outcome, the pending task
  terminalizes with its current-generation/empty-reservation/current-attempt fence and makes no
  additional reservation or external call.

### OpenCode M2a

- [ ] Resolve `A/opencode`, health-check it, switch PATH to `B`, execute, and prove only absolute
  `A/opencode` ran.
- [ ] Missing lookup then installation succeeds on retry; concurrent successful resolution pins
  one path without a data race.
- [ ] Empty path, directory, non-regular file, non-executable Unix file, and non-absolute
  LookPath result are rejected.
- [ ] Blocking child health exits on an internally applied 15s maximum (short duration injected
  in test), preserves earlier parent cancellation, and performs zero credential/plugin/repo effects.
- [ ] Missing OpenCode consumes the already accepted coordinator reservation/RunJob but produces
  a typed child-local permanent pre-clone failure; no test claims zero RunJob.
- [ ] Final diff contains no Codex, Pi, Motoko, Claude, managed-agent, or generic executor-resolver
  implementation change.

### Durable generation and attempts

- [ ] Old SQLite rows and Firestore documents read all new fields as zero values.
- [ ] SQLite migration/reopen and Firestore emulator preserve generation, attempts, current
  reservation/completion-authority ID, timestamp/phase, and last error.
- [ ] Twenty retryable polls yield exactly three reservations and at most three RunJob calls,
  final failed, attempts=3, and one notification attempt.
- [ ] Restart after attempt two permits exactly one further reservation.
- [ ] Crash after reservation/before audit, after audit/before RunJob, after RunJob error/before
  outcome, and after RunJob success/before outcome never exceeds three RunJob calls; expired
  attempts are never refunded.
- [ ] Successful RunJob outcome clears `dispatch_reserved_at` but retains the reservation ID;
  synchronous retryable error and lease-expiry recovery atomically revoke the old ID before
  returning pending.
- [ ] A stale generic `UpdateTask` snapshot cannot change any dispatch-owned field in either store.
- [ ] `RequeueTask`, `RetryAllFailedTasks`, design→sprint advance, and sprint→implementation
  advance each increment generation exactly once and begin a fresh three-attempt budget.
- [ ] `ResetTaskToPending` preserves generation and all cloud dispatch fields.

### Deadlines, completion, and notification

- [ ] Audit and RunJob side-effect fakes observe a matching durable reservation during the call.
- [ ] Fake RunJob environment and decoded completion contain the exact generation, reservation
  ID, and attempt ordinal returned by `ReserveDispatchAttempt`.
- [ ] Blocking audit/RunJob/health fakes return under injected 10s/60s/15s maxima; production
  constants equal those values, lease equals 2m, and parent cancellation wins earlier.
- [ ] Typed retry set is exactly `Unavailable`, `ResourceExhausted`, `DeadlineExceeded`, and
  `FailedPrecondition`; permanent and unknown codes do not retry.
- [ ] Generation-zero/empty-reservation legacy completion is accepted only against a stored
  generation-zero/empty-reservation queued or running task; it cannot match a new
  generation-zero dispatch with a non-empty ID.
- [ ] After any requeue, a delayed prior-generation or missing-pair completion cannot change
  state/result or notify; only the current generation with the exact current reservation can.
- [ ] Attempt 1 is ambiguously accepted, its lease expires, and attempt 2 is reserved in the same
  generation. Conflicting `{generation, reservation}` completions are delivered in both orders;
  attempt 1 always loses and only attempt 2 may own the result/notification.
- [ ] A completion with generation greater than zero and empty reservation ID is acknowledged,
  rejected, and logged without mutation.
- [ ] A completion with the correct generation/reservation but a different attempt ordinal is
  acknowledged, rejected, and logged without mutation.
- [ ] Stale-wins then dispatch-failure, completion-wins then stale, duplicate completion, and
  expiry-vs-completion barrier races each leave one terminal result and one notification attempt.
- [ ] Source/test guard proves cloud dispatch no longer calls `ResetTaskToPending` and no path uses
  `context.Background`/`context.WithoutCancel` for the bounded calls.

### Final gates

- [ ] Named Firestore transaction suite prints PASS in CI; emulator absence/skip fails CI.
- [ ] `go test ./internal/coordinator ./internal/dispatch/cloudrun ./internal/storage/firestore
  ./internal/executor/opencode ./internal/pubsub ./cmd/ailang` passes.
- [ ] `make test`, `make lint`, `make check-boundaries`, and focused race tests pass.
- [ ] Every mutation below compiles, turns its named test red for the intended reason, is restored
  without destructive git commands, and returns green.
- [ ] `changelogs/v0.32-current.md` (verified by `CHANGELOG.md` as the v0.32+ current file while
  `std/VERSION` is v0.34.0) documents route rejection, generation/attempt-authority semantics,
  three-attempt bound, explicit/bulk retry, and 10s/60s/15s/2m boundaries.

## Test and Mutation Plan

| Invariant | Positive/negative control | Build-preserving mutation that must fail |
|---|---|---|
| Agent route is authoritative | Historical job/env/model/budget fixture | Substitute global provider for route provider. |
| Invalid route is pre-effect | Matrix test with zero counters | Remove dispatcher validation. |
| Null config fails safely | Null-agent publication/direct-validator fixtures | Move nil check after default dereference. |
| OpenCode path is pinned | Poison-PATH fixture | Execute configured bare name instead of resolved path. |
| Failed lookup recovers | Missing→install→resolve | Cache the first error. |
| Attempt bound survives crash | Restart and four crash-point fixtures | Increment after RunJob or refund expiry. |
| Generic updates preserve accounting | Stale snapshot during reservation | Add dispatch fields to generic UpdateTask write. |
| New generation resets budget | Single/bulk/stage fixtures | Omit reset/increment from one deliberate retry API. |
| Old completion is fenced | Requeue then prior-generation completion | Remove generation predicate from terminal CAS. |
| Superseded same-generation attempt is fenced | Ambiguous attempt A, expiry, reservation B, conflicting completions in both orders | Remove reservation-ID predicate or clear B on RunJob success. |
| External waits stay within lease | Blocking fakes | Strip timeout or use `WithoutCancel`. |
| Terminal cannot resurrect | Barrier race | Restore cloud `ResetTaskToPending`. |
| Winner owns notification | Duplicate/stale/dispatch race | Notify outside the winning outcome branch. |

Tests are hermetic. They may use temporary files, in-process fakes, channels, SQLite, and an
isolated Firestore emulator. They must not contact production Firestore, GCS, Cloud Run, provider
APIs, or the public internet.

## Rollout and Rollback

1. Before deployment, validate/diff the live coordinator config without writing it.
2. Land additive storage fields and zero/empty-pair compatibility first; old documents remain
   readable.
3. Build coordinator and OpenCode executor image from one commit. Other executor images are not
   part of this sprint.
4. With separate operator authority, canary one low-cost coherent route and record generation,
   reservation/attempt identity, selected absolute child, terminal result, and notification count.
5. Exercise readiness failure only with fake/staging infrastructure, never by breaking a
   production job.

Rollback may restore the prior binary only after stopping new writers and inspecting tasks with
nonzero generation/attempt/reservation fields. Additive fields remain in place. Do not delete or
reset stored dispatch state automatically.

## Explicit Exclusions / Non-Goals

- No executable changes for Codex, Pi, Motoko, Claude, managed agents, or git.
- No exported/shared generic executable resolver.
- No image/Terraform changes and no installation of every CLI in every image.
- No change to Cloud Run `maxRetries=1`.
- No inner `TaskExecutor.ExecuteWithRetry` or eval-stream retry change.
- No transactional message outbox or exactly-once delivery claim.
- No provider signature/hash attestation, parent-directory ownership proof, immutable mounts, or
  complete filesystem TOCTOU prevention.
- No live multivac config write, paid canary, deployment, or production task mutation without
  separate operator authorization.
- No adoption of the parked `.wt-iter302` M2-M4 diff as an implementation base.
- No change to correlation/message-ingress dedup owned by `m-coord-dispatch-integrity`.

## Deferred Decisions

- Exact package-private helper names and test-fixture file grouping — implementation agent may
  choose within the listed packages.
- Whether SQLite uses a conditional single statement or explicit transaction for a particular
  CAS — implementation agent may choose if concurrency semantics and tests are identical.
- Firestore emulator launch mechanism in CI — implementation agent may reuse a repository
  facility if one lands before execution; named PASS/no-skip semantics are fixed.

No other architecture or lifecycle decision is deferred.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Completion-fence rollout rejects an old in-flight completion. | Medium | Both missing fields map to the legacy zero/empty pair and are accepted only against the same stored pair; deploy after inspecting/reconciling active tasks. |
| Firestore and SQLite drift. | High | One outcome table runs against both; CI emulator PASS is mandatory. |
| Generic update silently refunds attempts. | High | Dispatch fields are excluded from generic writes and covered by stale-snapshot mutation tests. |
| Retry API omitted. | High | Test single retry, bulk retry, and both task-chain stage transitions. |
| OpenCode-local helper later duplicates another adapter. | Low | Duplication is intentional scope containment; any shared extraction needs its own design/quorum. |
| Notification insertion fails after terminal CAS. | Medium | Log loudly and retain winner ownership; transactional outbox is explicitly not claimed. |
| Three attempts are insufficient during outage. | Medium | Failure is loud; deliberate retry starts a new generation. Infinite automated retry remains forbidden. |

## Verification Log

Repository claims were checked on 2026-08-30. Canonical code base is `origin/dev` at
`0b35abd5d`; the documentation worktree HEAD was `0b85f0388` and contains only the recovery-plan
commits beyond its older base. No `.ail` code or AILANG language-support claim is introduced, so
`ailang prompt`/`ailang check` is not applicable.

| ID | Claim | Command/source | Observed |
|---|---|---|---|
| V1 | Current origin descends from the recovery base. | `git merge-base --is-ancestor 3fc7be9b8 origin/dev` | Exit 0. |
| V2 | No relevant scoped source drift occurred between recovery base and current origin. | `git diff --stat 3fc7be9b8..origin/dev -- cmd/ailang/coordinator_cloud* cmd/ailang/coordinator_config.go internal/coordinator internal/dispatch/cloudrun internal/executor/opencode internal/storage/firestore .github/workflows/ci.yml` | Empty. |
| V3 | M1 remains mechanically conflict-free. | `git merge-tree $(git merge-base 3500db0a7 origin/dev) origin/dev 3500db0a7` searched for conflict markers | No conflict marker/`changed in both`; merge base `45503bac6`. |
| V4 | M1 commit contains superseded broad docs as well as code. | `git show --name-status 3500db0a7` | Adds old design and sprint plan; therefore transplant code/test hunks only. |
| V5 | Provider and image authorities are split at origin. | Read `internal/coordinator/daemon_tasks_exec.go` | Provider comes from `DefaultProvider`; variant later comes from `agent.ExecutorVariant`. |
| V6 | Cloud dispatch still resets on audit and RunJob errors. | `rg -n 'ResetTaskToPending' internal/coordinator/daemon_tasks_exec.go` | Two cloud-path calls. |
| V7 | Stale detection writes then notifies without CAS. | `rg -n 'MarkTaskFailed|postFailureNotification' internal/coordinator/stale_task_detector.go` | Unconditional mark followed by notification. |
| V8 | Completion uses pre-read plus unconditional terminal writes. | `rg -n 'MarkTaskFailed|MarkTaskCompleted|MarkTaskPendingApproval|postCompletionNotification' internal/coordinator/pubsub_completion_handler.go` | All three writes and notifications are separate from the pre-read. |
| V9 | OpenCode health and execution can select twice. | `rg -n 'LookPath|CommandContext' internal/executor/opencode/opencode.go` | Health uses LookPath/configured name; execution separately uses configured name. |
| V10 | No generic executable resolver exists at origin. | `test ! -e internal/executor/executable.go` plus resolver search | Confirmed absent. |
| V11 | Requeue is not operator-only. | `rg -n 'RequeueTask\(' internal/coordinator/task_chain.go cmd/ailang/coordinator_actions.go` | CLI call plus automatic design→sprint and sprint→implementation calls. |
| V12 | Bulk retry is a separate state-reset path. | Search `RetryAllFailedTasks` implementations | SQLite and Firestore implementations independently set failed tasks pending. |
| V13 | Completion has no generation fence. | Read `internal/pubsub/topics.go` `TaskCompletion` | Task/agent/status/result fields only; no dispatch generation/reservation. |
| V14 | Firestore generic update merges the complete task map. | Read `internal/storage/firestore/coordinator_tasks.go` | `taskToMap` + `Set(..., MergeAll)`. Adding dispatch fields to that map would make generic writes owners. |
| V15 | SQLite generic update currently omits dispatch fields. | Read `internal/coordinator/store_sqlite.go` `UpdateTask` | Its selected-field UPDATE contains no dispatch fields; recovery must preserve this property. |
| V16 | Recovered M1 nil check is ordered after defaulting. | `git show 3500db0a7:cmd/ailang/coordinator_config.go` and `:internal/coordinator/agent_config.go` | Validator calls defaults first; defaults dereference every agent; a null entry can panic. |
| V17 | Current source has no local audit/RunJob/health deadlines for this path. | Read daemon dispatch, Cloud Run dispatcher, and execute-job child | Long-lived caller contexts pass through; fixed boundaries must be added. |
| V18 | Current CI has no Firestore emulator lane. | Search `.github/workflows`, Makefile/make, internal, and tools for emulator setup | No `FIRESTORE_EMULATOR_HOST`/emulator provisioning hits. |
| V19 | The recovery plan records the old quorum as blocked and parked M2-M4 as evidence only. | `m-coordinator-child-env-opencode-retry-storm-recovery-plan.md` | `blocked_pending_fresh_quorum`; do not stage/cherry-pick parked diff. |
| V20 | Canonical cloud inbox was empty at designer session start. | `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list --unread --json` | `null`; nothing acknowledged. |
| V21 | `ResolveModel` exists with the assumed signature and order. | Read `internal/coordinator/model_resolution.go`; `rg -n 'ResolveModel\(' internal/coordinator cmd/ailang`; `go test ./internal/coordinator -run TestResolveModel -count=1 -v` | `ResolveModel(agent *AgentConfig) (string, error)` is pin-first, then cloud-lane registry role; no model/role returns empty; unknown role is loud. Current cloud/local call sites are `daemon_tasks_exec.go:194` and `daemon_tasks_exec_run.go:252`; all four focused tests PASS. |
| V22 | The local worktree-limit path calls `ResetTaskToPending`. | `sed -n '140,180p' internal/coordinator/daemon_tasks_exec_run.go`; `rg -n 'ResetTaskToPending\(' internal/coordinator internal/storage` | `errors.Is(wtErr, ErrWorktreeLimitReached)` logs the retry and calls `d.taskStore.ResetTaskToPending`; the only other production calls are the two cloud-reset defects in `daemon_tasks_exec.go`, which this sprint removes. SQLite and Firestore methods exist. |
| V23 | Existing worktree tests do not prove daemon reset preservation. | Search `ErrWorktreeLimitReached|ResetTaskToPending` in coordinator tests; `go test ./internal/coordinator -run TestWorktreeManagerMaxLimit -count=1 -v` | The manager cap fixture PASS proves the third allocation fails; no current daemon test invokes the reset path. Therefore `daemon_tasks_exec_run_test.go` is new and mandatory. |
| V24 | `cmd/ailang/coordinator_config.go` exists with the publication seams this design modifies. | `test -f cmd/ailang/coordinator_config.go && rg -n '^func (validateCoordinatorConfigBytes|writeConfigCAS)' cmd/ailang/coordinator_config.go` | `validateCoordinatorConfigBytes` at 55 and `writeConfigCAS` at 94. |
| V25 | `cmd/ailang/coordinator_config_test.go` exists and exercises validation-before-CAS. | `test -f cmd/ailang/coordinator_config_test.go && rg -n '^func TestConfigCAS' cmd/ailang/coordinator_config_test.go` | Current-generation write, stale generation, malformed YAML, missing workspace, and valid config tests at lines 49-134. |
| V26 | `internal/coordinator/agent_config.go` exists with the config-loading/default surface. | `test -f internal/coordinator/agent_config.go && rg -n '^func LoadCoordinatorConfigFrom' internal/coordinator/agent_config.go` | `LoadCoordinatorConfigFrom` at 553 applies current inline defaults; M1 extracts/reuses nil-safe normalization here. |
| V27 | `internal/coordinator/daemon.go` exists with the shared daemon wiring surface. | `test -f internal/coordinator/daemon.go && rg -n '^type Daemon struct' internal/coordinator/daemon.go` | `Daemon` at 66 owns store, Pub/Sub, dispatcher, registry, and test seams used by M1/M4. |
| V28 | `daemon_tasks_init.go` exists and owns Pub/Sub initialization. | `test -f internal/coordinator/daemon_tasks_init.go && rg -n '^func \(d \*Daemon\) initPubSub' internal/coordinator/daemon_tasks_init.go` | `initPubSub` at 364 and `initPubSubBroadcaster` at 391. |
| V29 | `daemon_tasks_budget.go` exists with provider-sensitive budget selection. | `test -f internal/coordinator/daemon_tasks_budget.go && rg -n '^func \(d \*Daemon\) checkBudgetBeforeExecution' internal/coordinator/daemon_tasks_budget.go` | `checkBudgetBeforeExecution` at 13; M1 adds route-provider lookup without inventing a second budget path. |
| V30 | `daemon_tasks_exec.go` exists with the cloud dispatch entry and both reset defects. | `test -f internal/coordinator/daemon_tasks_exec.go && rg -n 'dispatchTasksCloud|ResetTaskToPending' internal/coordinator/daemon_tasks_exec.go` | `dispatchTasksCloud` at 88; cloud reset calls at 121 and 255. |
| V31 | Cloud Run dispatcher source exists with job selection and effect entry point. | `test -f internal/dispatch/cloudrun/dispatcher.go && rg -n '^func (jobSuffixForVariant|\(d \*Dispatcher\) Dispatch)' internal/dispatch/cloudrun/dispatcher.go` | `jobSuffixForVariant` at 89 and `Dispatch` at 106. |
| V32 | Cloud Run dispatcher tests exist at the route/env/RunJob surface. | `test -f internal/dispatch/cloudrun/dispatcher_test.go && rg -n '^func TestDispatch' internal/dispatch/cloudrun/dispatcher_test.go` | Job name, env, error, auth, variant, and region tests exist; the new route and generation/reservation env cases extend this file. |
| V33 | OpenCode adapter source contains the constructor, execute, and health paths. | `test -f internal/executor/opencode/opencode.go && rg -n '^func (New|\(e \*OpenCodeExecutor\) ExecuteStreaming|\(e \*OpenCodeExecutor\) HealthCheck)' internal/executor/opencode/opencode.go` | `New` at 47, execute at 82, health at 566. |
| V34 | `opencode_test.go` exists with executor behavior fixtures. | `test -f internal/executor/opencode/opencode_test.go && rg -n '^func TestExecuteStreaming' internal/executor/opencode/opencode_test.go` | Persistent-system-prompt and fixture parsing tests exist; resolver construction assertions extend this file. |
| V35 | `opencode_streaming_test.go` exists with missing/fake-binary coverage. | `test -f internal/executor/opencode/opencode_streaming_test.go && rg -n '^func Test(ExecuteStreaming_BinaryNotFound|HealthCheck_)' internal/executor/opencode/opencode_streaming_test.go` | Binary-not-found at 119, missing health at 137, fake binary at 146; poison-PATH extends this file. |
| V36 | `coordinator_cloud.go` owns child orchestration and completion construction. | `test -f cmd/ailang/coordinator_cloud.go && rg -n 'publishCompletion :=|TaskCompletion\{|^func executeCloudTask' cmd/ailang/coordinator_cloud.go` | Completion closure at 110, `TaskCompletion` construction at 114, clone/orchestration entry at 218. |
| V37 | `coordinator_cloud_executor.go` owns factory lookup and execution. | `test -f cmd/ailang/coordinator_cloud_executor.go && rg -n '^func runExecutor' cmd/ailang/coordinator_cloud_executor.go` | `runExecutor` at 26; M2a adds the preflight seam beside this path. |
| V38 | `cloud_dispatcher.go` exists with the end-to-end parameter carrier. | `test -f internal/coordinator/cloud_dispatcher.go && rg -n '^type (CloudDispatcher|DispatchParams)' internal/coordinator/cloud_dispatcher.go` | Interface at 8 and `DispatchParams` at 16; generation/reservation fields belong here. |
| V39 | `store.go` exists with task persistence and full store contract. | `test -f internal/coordinator/store.go && rg -n '^type (TaskRecord|Store interface)' internal/coordinator/store.go` | `TaskRecord` at 9 and `Store` at 181. |
| V40 | `store_sqlite.go` exists with schema, generic update, requeue, and local reset. | `test -f internal/coordinator/store_sqlite.go && rg -n '^func \(s \*SQLiteStore\) (migrate|UpdateTask|RequeueTask|ResetTaskToPending)' internal/coordinator/store_sqlite.go` | Methods at 54, 276, 657, and 666; migration/state ownership changes belong here while generic update remains dispatch-field-free. |
| V41 | `store_sqlite_queries.go` exists with bulk retry. | `test -f internal/coordinator/store_sqlite_queries.go && rg -n '^func \(s \*SQLiteStore\) RetryAllFailedTasks' internal/coordinator/store_sqlite_queries.go` | Bulk retry at 148; it must increment generation/reset per-generation fields. |
| V42 | Stale detector source exists with the non-CAS writer. | `test -f internal/coordinator/stale_task_detector.go && rg -n '^func \(d \*StaleTaskDetector\) detectAndMarkStale|MarkTaskFailed|postFailureNotification' internal/coordinator/stale_task_detector.go` | Detector at 66; unconditional failure write and later notification at 84/94. No existing stale detector test file exists. |
| V43 | Completion handler source exists with the non-atomic terminal paths. | `test -f internal/coordinator/pubsub_completion_handler.go && rg -n '^func \(h \*CompletionHandler\) handleCompletion|MarkTaskCompleted|MarkTaskPendingApproval|MarkTaskFailed' internal/coordinator/pubsub_completion_handler.go` | Handler at 74; three unconditional transition methods at 123/130/141. No existing completion-handler test file exists. |
| V44 | Both full-Store test mocks exist and name retry/reset methods. | `test -f internal/coordinator/mock_store_test.go -a -f internal/coordinator/approval_watcher_mock_test.go`; search `RetryAllFailedTasks|ResetTaskToPending` in both | `MockStore`/`mockStore` exist; both implement bulk retry and local reset and must gain the expanded interface methods. |
| V45 | Firestore conversion source exists and owns task map hydration. | `test -f internal/storage/firestore/coordinator_convert.go && rg -n '^func (taskToMap|mapToTask)' internal/storage/firestore/coordinator_convert.go` | `taskToMap` at 10 and `mapToTask` at 84; missing fields must hydrate zero/empty. |
| V46 | Firestore generic task CRUD source exists. | `test -f internal/storage/firestore/coordinator_tasks.go && rg -n '^func \(s \*CoordinatorStore\) (CreateTask|UpdateTask)' internal/storage/firestore/coordinator_tasks.go` | Create at 18 and generic merge update at 38; UpdateTask must strip/preserve dispatch-owned fields. |
| V47 | Firestore transition source exists with deliberate single requeue and local reset. | `test -f internal/storage/firestore/coordinator_transitions.go && rg -n '^func \(s \*CoordinatorStore\) (RequeueTask|ResetTaskToPending)' internal/storage/firestore/coordinator_transitions.go` | Requeue at 156 and local reset at 170. |
| V48 | Firestore approvals source exists with bulk retry. | `test -f internal/storage/firestore/coordinator_approvals.go && rg -n '^func \(s \*CoordinatorStore\) RetryAllFailedTasks' internal/storage/firestore/coordinator_approvals.go` | Bulk retry at 283. |
| V49 | Pub/Sub completion wire source and decode tests exist. | `test -f internal/pubsub/topics.go -a -f internal/pubsub/pubsub_test.go`; search `type TaskCompletion|TestDecodeTaskCompletion` | `TaskCompletion` at `topics.go:47`; success/error decode fixtures at `pubsub_test.go:289/316`. |
| V50 | CI workflow exists with the ordinary Go test lane. | `test -f .github/workflows/ci.yml && rg -n '^name:|go test -timeout' .github/workflows/ci.yml` | Workflow `CI`; full `go test -timeout 300s ./...` lanes at 101 and 447; no emulator setup exists (V18). |
| V51 | `changelogs/v0.32-current.md` is the correct current changelog despite the v0.35 target. | `cat std/VERSION`; read `CHANGELOG.md`; `test -f changelogs/v0.32-current.md`; inspect its headings | Version is `v0.34.0`; root index says “For the latest version” use `v0.32-current.md`; file header says `v0.32+` and contains `[Unreleased]` followed by `[v0.34.0]`; no newer current-series file exists. |
| V52 | Fresh quorum was fully present and blocked specifically on identity and premise/file verification. | `.ailang/state/mission-quorum/m-coordinator-child-env-opencode-retry-storm-recovery-design-2026-08-30T17-59-13Z.json` | Three present reviewers rejected; no absent reviewer. This revision addresses each named objection once. |
| V53 | Every non-new file in the exact file surface exists. | Shell array over all 30 non-new paths in **Files to Modify/Create**, `test -f` each | `verified_non_new_files=30 missing=0`; V24-V51 name the depended-on symbol/content in each. |

## Related Documents

- [Recovery plan](m-coordinator-child-env-opencode-retry-storm-recovery-plan.md) — preservation,
  human/quorum gates, and recovered artifact disposition.
- `3500db0a7` recovered M1 commit — route candidate, not wholesale cherry-pick authority.
- [M-COORD-DISPATCH-INTEGRITY](m-coord-dispatch-integrity.md) — sibling inbound dispatch and
  correlation dedup; it does not own outbound reservation counts or completion-authority pairs.
- [M-GIT-BINARY-RESOLUTION-SWEEP](m-git-binary-resolution-sweep.md) — related absolute-path
  wording; its git-specific resolver is not reused here.
- [M-EXECUTOR-VARIANTS](../../implemented/v0_15_0/m-executor-variants.md) — establishes separately
  baked Cloud Run image variants.
- [M-EXEC-EXPAND-CODEX-OPENCODE](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) —
  original OpenCode adapter integration.
- [V1 mission queue](../../v1-mission.md) — incident routing and D-50.

## Fresh Quorum Disposition

Submit this document, not the recovered broad design, to a fresh quorum. The controller verdict
should be PASS only if reviewers accept exact generation-plus-reservation completion authority,
dispatch-field ownership, OpenCode-only scope, and the corrected child-local preflight boundary.
This is the single protocol-mandated revision after the fully present blocked quorum recorded in
V52. On rerun PASS, D-50 already supplies the human approval and execute instruction, so sprint
planning/execution may proceed without another human approval round. On rerun rejection, do not
invent a carve-out; park for the protocol's required disposition.

---

**Document created**: 2026-08-30
**Last updated**: 2026-08-30
