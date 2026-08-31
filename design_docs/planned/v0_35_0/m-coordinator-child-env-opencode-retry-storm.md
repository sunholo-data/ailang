# M-COORDINATOR-CHILD-ENV-OPENCODE-RETRY-STORM: coherent execution routes, pinned executables, bounded terminalization

**Status**: Planned
**Target**: v0.35.0
**Priority**: P0 — a planner task was sent to the wrong Cloud Run image/provider pair,
remained non-terminal across 41 dispatch failures, and emitted nine operator messages
**Estimated**: 4 days
**Dependencies**: None; coordinate with
[m-git-binary-resolution-sweep](m-git-binary-resolution-sweep.md) so the generic executor
resolver and the git-specific resolver have the same security wording without sharing APIs

## Quorum triggers

Quorum is required before approval because the design depends on two external first-party
systems (Cloud Run execution/log metadata and the GCS-backed multivac coordinator config)
and changes shared coordinator storage transitions. No decision is left for a reviewer to
freeze; quorum tests whether the evidence supports the frozen design. See **Quorum
Verification** below.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One agent config resolves to one provider/image/model tuple before dispatch |
| A2: Replayability | +1 | Persisted attempt count and last dispatch error explain every retry decision |
| A3: Effect Legibility | 0 | No language-effect change |
| A4: Explicit Authority | +1 | Child binaries are resolved once to an absolute path rather than re-selected from ambient `PATH` at execution |
| A5: Bounded Verification | +1 | Three attempts and one terminal notification are finite, testable invariants |
| A6: Safe Concurrency | +1 | Compare-and-set terminal transitions prevent a losing writer from resurrecting a task |
| A7: Machines First | +1 | Config validation rejects incoherent routes before an agent spends a Cloud Run attempt |
| A8: Minimal Syntax | 0 | No AILANG syntax change |
| A9: Cost Visibility | +1 | Dispatch attempts become stored data; repeated Cloud Run starts are bounded |
| A10: Composability | +1 | One route validator and one executable resolver serve all coordinator CLI providers |
| A11: Structured Failure | +1 | Permanent route errors and retryable dispatch errors are typed, not parsed from strings |
| A12: System Boundary | +1 | Coordinator→Cloud Run and Go→child-process boundaries validate their inputs explicitly |

**Net Score: +10** → Move forward. Hard-violation check: none.

## Problem Statement

Task `task-a0628a5f` (`inbox_1787857916125_a0628a5f`) did not fail because a local launchd
child lost `PATH`. The actual execution was a Cloud Run Job selected from two inconsistent
configuration sources:

```text
agent sprint-planner: provider=codex, executor_variant=codex
coordinator default_provider: opencode

HEAD dispatchTasksCloud:
  AILANG_PROVIDER       <- coordinator default_provider  # opencode
  Cloud Run job variant <- agent executor_variant        # codex

result:
  ailang-agent-executor-codex starts runExecutor("opencode")
  opencode child lookup fails: exec: "opencode": executable file not found in $PATH
```

The code still has this split at HEAD `45503bac6`: `daemon_tasks_exec.go:114-115` chooses
the global default provider, while `:214-215` independently chooses the agent's executor
variant. `LoadCoordinatorConfigFrom` already fills a missing `agent.Provider` from the
default (`agent_config.go:575-588`), so ignoring the resolved per-agent provider has no
compatibility rationale.

The same incident exposed a second defect. A Cloud Run Job readiness error returned the task
from `queued` to `pending` on every daemon poll. The stale-task detector independently wrote
`failed` and posted a timeout notice, but a racing dispatch error then unconditionally reset
the task to `pending`. Live logs contain 41 failed dispatches and eight stale-terminalization
attempts before one dispatch finally succeeded. The child then failed twice under Cloud Run's
bounded `maxRetries=1`; the completion handler emitted one failure message. Across the whole
incident, the task produced eight timeout notices plus one executor-failure notice.

This is not the inner `TaskExecutor.ExecuteWithRetry` loop: that loop is already bounded and
does not classify “executable file not found” as retryable. The storm lives in persistent
cloud dispatch state and a non-atomic terminal transition.

Finally, the child adapters (`opencode`, `codex`, `pi`, `motoko`, and `claude`) do not share a
security contract for executable resolution. Several health checks use `LookPath` but later
execute the original bare name, allowing execution-time `PATH` to make a second selection.
Correct route selection prevents the incident; absolute, pinned resolution makes the child
boundary correct even if process environment changes later.

### Impact

- A provider/default change can silently select an image that does not contain that
  provider's CLI, even when the agent entry itself is coherent.
- A transiently unhealthy Cloud Run Job can be retried for the lifetime of the daemon and a
  terminal task can be resurrected.
- Every stale cycle can post another operator message for the same logical failure.
- A health check can approve one binary and execution can run another binary selected later
  from a changed `PATH`.

## Goals

**Primary Goal:** Resolve and validate one coherent execution route before dispatch, execute
the intended child through a pinned absolute path, and make all automated dispatch retries
and terminal notifications finite and race-safe.

**Success metrics:**

1. An agent with `provider=codex, executor_variant=codex` produces `AILANG_PROVIDER=codex`
   regardless of a different global default; every unsupported provider/variant pair is
   rejected before `RunJob`.
2. A configured bare executor is resolved once to an absolute path and that exact path is
   passed to `exec.CommandContext`; changing `PATH` after construction cannot redirect it.
3. One logical task can reserve at most **three total coordinator dispatch attempts** without
   an explicit operator requeue. Every `RunJob` call requires a durable reservation first,
   so crashes cannot raise the call count above three.
4. Any number of racing stale/completion/dispatch callbacks produces at most **one terminal
   transition winner and one operator completion/failure message**.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Resolve `{provider, executor_variant, model}` as one immutable `ExecutionRoute` from the agent; the global provider is inheritance only | Removes the source split that created the incident and defines cloud routing authority | agent | design | high |
| D2: Enforce the provider/variant compatibility matrix both at config publication and immediately before `RunJob` | Bad config must fail before spend, while defensive dispatch validation covers programmatic callers | agent | design | high |
| D3: Resolve CLI executables to an absolute path once per executor instance, cache successes only, and execute that path | Defines the child-process security boundary and preserves recovery if a binary appears later | agent | design | high |
| D4: Transactionally reserve each of at most three coordinator dispatch attempts before any publish/`RunJob` side effect; route errors fail before reservation | Bounds actual external calls across daemon crashes/restarts while preserving short recovery from Cloud Run readiness failures | agent | design | high |
| D5: All terminal writes are compare-and-set from active states; only the winning writer may notify | Prevents stale/dispatch/completion races and message duplication | agent | design | high |
| D6: Keep Cloud Run Job `maxRetries=1` unchanged | It is already bounded, protects against container-level transients, and duplicate completions become harmless under D5 | agent | design | med |
| D7: Enforce 10s audit-publish, 60s `RunJob`, and 15s child-health deadlines internally; keep reservation lease at 2m | A retry bound is meaningless if an external call can block forever or outlive its lease | agent | design | high |

### Design Freeze

All high-change-cost decisions are resolved by this design:

- [x] D1 — per-agent resolved route is authoritative
- [x] D2 — publish-time plus dispatch-time validation
- [x] D3 — absolute success pin; failed lookups are not cached
- [x] D4 — three durable pre-side-effect reservations, persistent across crashes/restarts
- [x] D5 — compare-and-set transition winner owns notification
- [x] D7 — fixed internal deadlines, never caller-background defaults

No sprint pause is required unless implementation evidence disproves a Verification Log
premise.

## Solution Design

### 1. One coherent execution route

Add `ExecutionRoute` and `ResolveExecutionRoute` in `internal/coordinator`. Resolution order
is deliberately narrow:

1. Find the agent by `task.AgentID`; absence is a permanent error.
2. Use `agent.Provider`. Config loading already inherits `default_provider` into this field.
3. Use `agent.ExecutorVariant`; an empty value is normalized only for legacy Claude agents
   to `default`.
4. Resolve the model with the existing `ResolveModel(agent)` path.
5. Validate the tuple against this compatibility table:

| Provider | Allowed Cloud Run variants |
|----------|----------------------------|
| `claude` | `default`, `go` (and legacy empty, normalized to `default`) |
| `gemini` | `gemini`, `gemini-go` |
| `codex` | `codex`, `codex-go` |
| `opencode` | `opencode` |
| `pi` | `pi` |
| `motoko` | `motoko` |

`eval` and `eval-go` are job variants, not general provider names; their dedicated eval
entry path remains out of this matrix. Unknown providers/variants and mismatched known pairs
return `PermanentDispatchError` naming agent, provider, and variant. They must not fall back
to a default job.

`dispatchTasksCloud` constructs `DispatchParams` from the returned route, not from separate
global and agent reads. `cloudrun.Dispatcher.Dispatch` calls the same compatibility validator
before computing a job name or invoking `RunJob`, protecting tests and future non-daemon
callers. `validateCoordinatorConfigBytes` applies config defaults in memory and validates all
cloud-lane agents before the GCS compare-and-swap write. Local-lane agents are exempt from
the Cloud Run variant matrix.

The budget lookup must use `route.Provider`, closing the same split for provider-specific
budgets. Logs record all three route fields in one line.

### 2. Pinned executable resolution

Add `internal/executor/executable.go` with a small, dependency-free resolver used by the five
CLI executor adapters:

```go
type Executable struct {
    configured string
    resolved   string // guarded; successful absolute resolution only
}

func (e *Executable) Resolve() (string, error)
```

The contract is:

- Empty configuration is an error.
- An absolute configured path is cleaned, `stat`ed, required to be a regular file, and on
  Unix required to have at least one execute bit.
- A bare name is resolved with `exec.LookPath`; the result must be absolute and pass the same
  file checks.
- Only successful resolution is cached. A missing binary can be installed and a later health
  check can recover without rebuilding the process.
- `HealthCheck` and `ExecuteStreaming` call the same resolver; `exec.CommandContext` receives
  the returned absolute path. No adapter performs a second bare-name lookup.
- Claude's existing NVM discovery remains an input candidate, but the final candidate passes
  through this resolver. Motoko's health-check resolution and execute path are unified.

This protects against late `PATH` substitution and makes the selected child observable. It
does **not** prove that the binary's parent directory is unwritable by an attacker, verify
code signatures/hashes, or eliminate filesystem TOCTOU between `stat` and `execve`. Those
stronger ownership/provenance controls are separate work. This wording intentionally matches
the corrected scope of `m-git-binary-resolution-sweep`; the git resolver stays git-specific
and is not imported into the general executor package.

The Cloud Run `execute-job` path performs `HealthCheck` immediately after factory lookup and
before clone/plugin setup. That produces a cheap, structured `PermanentDispatchError` for a
missing baked-in CLI. It complements route validation; it is not a substitute for choosing
the correct image.

### 3. Bounded persistent dispatch attempts

Add the following task fields to SQLite and Firestore conversion:

```text
dispatch_attempts       integer, default 0
dispatch_reservation_id string, default ""
dispatch_reserved_at    timestamp, nullable
last_dispatch_error     string, default ""
```

The bound is `MaxCloudDispatchAttempts = 3` total reservations, not three retries. Every
audit publish and coordinator→Cloud Run `RunJob` call must hold a reservation created before
the side effect, so three reservations imply at most three `RunJob` calls even if the daemon
crashes at the least convenient instruction. The existing poll cadence supplies spacing;
this sprint adds no independent retry goroutine.

Route/model/variant validation and local-lane exclusion happen before reservation. Then
replace `MarkTaskQueued` plus cloud `ResetTaskToPending` with three transactional store
operations:

```go
ReserveDispatchAttempt(ctx, id, maxAttempts, now) (Reservation, DispatchOutcome, error)
RecordDispatchOutcome(ctx, id, reservationID, err, retryable) (DispatchOutcome, error)
RecoverExpiredDispatchReservation(ctx, id, now, lease, maxAttempts) (DispatchOutcome, error)
```

`ReserveDispatchAttempt` may mutate only a currently `pending` task. In one transaction it
checks the bound, increments `dispatch_attempts`, writes a random reservation ID and
`dispatch_reserved_at`, and moves the task to `queued`. Only after commit may the daemon
publish the audit event and call `RunJob`.

`RecordDispatchOutcome` may mutate only the matching current reservation:

- on success, clear reservation fields and leave the task `queued` awaiting completion;
- on `retryable && attempts < maxAttempts`, store `last_dispatch_error`, clear the
  reservation, set `pending`, and return `WillRetry`;
- otherwise set `failed`, set `completed_at`, and return `WonTerminal`;
- if the task is terminal, no longer `queued`, or carries a different reservation ID, return
  `LostRace` without changing state.

The daemon wraps synchronous audit publication in `context.WithTimeout(..., 10s)` and
`RunJob` in a separate `context.WithTimeout(..., 60s)`, regardless of its caller context.
The Cloud Run child wraps executable health preflight in `context.WithTimeout(..., 15s)`.
No path falls back to `context.Background` after timeout. An unresolved reservation has a
fixed `CloudDispatchLease = 2m`, leaving 60 seconds after the maximum `RunJob` duration for
outcome persistence/poll margin. The poller/startup recovery transaction consumes (never refunds)
an expired reservation: if attempts remain it clears the lease and returns the task to
`pending`; at attempt 3 it terminalizes. The stale detector skips an unexpired reservation
and races through the same CAS after expiry. Tests inject the clock; there are no sleeps.

This deliberately gives the outbound task→Cloud Run boundary **bounded at-least-once**
semantics, not exactly-once execution. A crash after Cloud Run accepted a job but before the
success record can cause one later redispatch if no completion wins during the lease, but
the prior reservation remains consumed and total calls stay ≤3. Task ID plus reservation ID
are logged for reconciliation. The sibling `m-coord-dispatch-integrity` owns inbound
message/correlation dedup; it neither counts nor reserves outbound Cloud Run calls, so the
two designs do not share state or override each other.

The HEAD dispatcher uses `cloud.google.com/go/run/apiv2.NewJobsClient`, whose default client
is gRPC, and the live incident error was exactly `rpc error: code = FailedPrecondition` (V19).
Cloud Run transport/readiness errors are retryable only through `status.Code(err)` for the
explicit set `Unavailable`, `ResourceExhausted`, `DeadlineExceeded`, and the observed
job-readiness `FailedPrecondition`. `InvalidArgument`, `Unauthenticated`, `PermissionDenied`,
and all unrecognized codes are permanent. Route/config errors and missing executables occur
before this classifier and are permanent. There is no substring matching. If a future caller
constructs the generated REST client instead, it must add typed `*googleapi.Error` tests and
classification before use; the sprint must not silently infer REST codes.

`RequeueTask` is the explicit operator action and resets all four dispatch fields.
`ResetTaskToPending` remains for worktree-limit recovery in the local execution lane, but may
not be used by cloud dispatch.

### 4. Atomic terminalization and exactly-once notification ownership

Introduce `FinishTaskIfActive`, implemented as a SQLite transaction/conditional update and a
Firestore transaction. It accepts the desired terminal state/result and succeeds only from
`pending`, `queued`, or `running` as allowed by the caller. It returns `won=false` when
another actor already changed the state.

- The stale detector calls `FinishTaskIfActive(... failed ...)` and posts its timeout message
  only if `won=true`.
- The Pub/Sub completion handler performs the same atomic transition to `failed`,
  `pending_approval`, or `completed`, and posts only if `won=true`.
- `RecordDispatchOutcome` and expired-reservation recovery own their terminal transitions
  and return `WonTerminal`; the daemon uses that outcome to post one route/dispatch failure
  notification through the existing completion-notification plumbing.
- A completion for an already terminal task is acknowledged and logged as an idempotent
  no-op. A late dispatch failure can never reset it.

Exactly-once here means one transition/notification decision in the task store. Delivery by
the external message transport remains at-least-once; changing the collaboration message
store to a transactional outbox is a non-goal. Notification payloads retain `task_id`, which
is the consumer dedup key if transport redelivery occurs.

## Alternatives Considered

1. **Fix only the live multivac config.** The current config is already back to
   `codex/codex`, but HEAD can recreate the split whenever the global default differs. Rejected
   because configuration repair does not close the code path.
2. **Install `opencode` in every executor image.** This hides route mistakes, expands image
   authority/size, and could run the wrong paid provider successfully. Rejected.
3. **Copy the agent provider into the task at creation.** This creates another stale copy and
   complicates config changes between enqueue and dispatch. Rejected; resolve one route at
   dispatch from the loaded agent registry and store it when execution starts.
4. **Count only after `RunJob` returns.** A crash after the external call and before the write
   evades the bound. Rejected; reservation is durable and precedes every external attempt.
5. **In-memory retry counters/backoff.** A daemon restart erases the bound, recreating an
   infinite distributed retry. Rejected; attempt state is persistent.
6. **Make every dispatch error permanent.** Safe but turns brief Cloud Run readiness failures
   into avoidable operator work. Rejected in favor of a typed three-attempt allowance.
7. **Set Cloud Run `maxRetries=0`.** It would remove one bounded recovery layer but does not
   fix the 41 coordinator attempts or terminal race. Deferred unless deployment evidence
   shows container retries violate the new idempotency contract.
8. **Trust `LookPath` in HealthCheck and execute the bare name later.** Rejected because the
   two operations can select different files after `PATH` changes.

## Implementation Plan

### M1 — route coherence and preflight (~1 day)

- Implement `ExecutionRoute`, the provider/variant matrix, and typed permanent errors.
- Use the route in cloud daemon params, budget lookup, config publication validation, and
  defensive Cloud Run dispatch validation.
- Run executor health check before cloud clone/setup and keep its error structured.
- Add the exact historical `default=opencode + planner=codex/codex` regression fixture.

### M2 — shared executable resolver (~1 day)

- Implement and test absolute success pinning, non-cached failures, file/execute-bit checks,
  and concurrent calls.
- Migrate `opencode`, `codex`, `pi`, `motoko`, and `claude` health/execute paths while
  preserving Claude NVM candidate discovery.
- Add poison-`PATH` tests proving a health-approved binary cannot be swapped before execute.

### M3 — bounded storage state machine (~1.5 days)

- Add task fields and backward-compatible SQLite migration/scan plus Firestore conversion.
- Add transactional reservation/outcome/recovery operations and `FinishTaskIfActive` to both
  stores.
- Replace cloud reset, stale terminalization, and completion terminalization with outcome-aware
  calls; only the terminal winner posts.
- Add deterministic crash-point and race tests with barriers/injected clocks, not timing
  sleeps.

### M4 — integration, mutation, rollout evidence (~0.5 day)

- Run the historical route fixture through a fake `jobRunner` and the 20-poll plus
  reserve/crash/recover fixtures through both store semantics.
- Run targeted packages, `make test`, `make lint`, and `make check-boundaries`.
- Deploy to a canary coordinator, dispatch one `codex/codex` no-op task, and record route,
  absolute executable, attempt count, terminal state, and notification count.

## Exact Files

### New

- `internal/coordinator/execution_route.go` — immutable route, compatibility matrix, typed
  route/dispatch classification (~140 LOC)
- `internal/coordinator/execution_route_test.go` — full provider/variant table and historical
  regression (~220 LOC)
- `internal/coordinator/dispatch_state.go` — attempt bound and store outcomes (~100 LOC)
- `internal/coordinator/dispatch_state_test.go` — 20-poll and deterministic race fixtures
  (~250 LOC)
- `internal/executor/executable.go` — shared absolute resolver (~110 LOC)
- `internal/executor/executable_test.go` — pinning, recovery, permissions, concurrency, and
  poison-`PATH` tests (~220 LOC)

### Modified

- `internal/coordinator/daemon_tasks_exec.go` — consume one route and replace cloud reset with
  bounded outcome handling
- `internal/coordinator/daemon_tasks_budget.go` — use the same resolved provider for budgets
- `internal/coordinator/store.go` — task fields and transactional transition interface
- `internal/coordinator/store_sqlite.go` — schema migration and SQLite transitions
- `internal/coordinator/store_sqlite_queries.go` — scan/persist new task fields
- `internal/coordinator/store_sqlite_test.go` — restart persistence and CAS transition tests
- `internal/coordinator/stale_task_detector.go` — notify only after winning terminal CAS
- `internal/coordinator/pubsub_completion_handler.go` — atomic idempotent completion and shared
  notification ownership
- `internal/coordinator/mock_store_test.go`,
  `internal/coordinator/approval_watcher_mock_test.go` — satisfy the expanded store contract
- `internal/storage/firestore/coordinator_convert.go` — attempt field conversion
- `internal/storage/firestore/coordinator_transitions.go` — Firestore transactional transitions
- `internal/storage/firestore/feedbackgate_stores_test.go` — emulator-backed transition/race
  cases (or skip loudly when the emulator is unavailable in local runs; CI must run them)
- `internal/dispatch/cloudrun/dispatcher.go`, `internal/dispatch/cloudrun/dispatcher_test.go` —
  defensive route check before `RunJob`
- `cmd/ailang/coordinator_config.go`, `cmd/ailang/coordinator_config_test.go` — reject mismatched
  cloud routes before GCS write
- `cmd/ailang/coordinator_cloud_executor.go` — early executor health preflight
- `internal/executor/opencode/opencode.go`, `internal/executor/opencode/opencode_streaming_test.go`
- `internal/executor/codex/codex.go`, `internal/executor/codex/codex_test.go`
- `internal/executor/pi/pi.go`, `internal/executor/pi/pi_test.go`
- `internal/executor/motoko/motoko.go`, `internal/executor/motoko/healthcheck.go`,
  `internal/executor/motoko/motoko_test.go`
- `internal/executor/claude/claude.go`, `internal/executor/claude/claude_test.go`
- `CHANGELOG.md` — operator-visible route validation, retry bound, and migration note

No multivac repository file is part of this implementation: its live config is already
coherent, and this design prevents a future incoherent config from being published.

## Acceptance Criteria (must be able to fail)

- [ ] Historical fixture: global `default_provider=opencode`, planner
  `provider=codex, executor_variant=codex` yields route `codex/codex`, and the fake RunJob
  request contains job suffix `-codex` plus `AILANG_PROVIDER=codex`.
- [ ] Negative fixture: `provider=opencode, executor_variant=codex` fails config validation
  and dispatcher validation; fake `RunJob` call count remains **0**.
- [ ] All allowed matrix cells pass and every off-diagonal/unknown cell fails with a typed
  permanent error naming the agent/provider/variant.
- [ ] With two fake binaries `A/opencode` and `B/opencode`, resolve while `PATH=A`, switch to
  `PATH=B`, execute, and prove only absolute `A/opencode` ran.
- [ ] Resolve a missing name (failure), install it in the test directory, retry, and prove
  success; this fails if lookup failures are cached.
- [ ] A non-regular file, relative `LookPath` result, and non-executable Unix file are each
  rejected; the test suite does not claim directory ownership or signature validation.
- [ ] Twenty daemon poll cycles against a retryable fake Cloud Run readiness error produce
  exactly **3** durable reservations and **3** RunJob calls, final `failed`,
  `dispatch_attempts=3`, and exactly **1** failure notification.
- [ ] Restart after normal attempt 2 outcome; only one further reservation/RunJob call occurs.
- [ ] Crash injection after reservation/before `RunJob`, after `RunJob` error/before outcome,
  and after `RunJob` success/before outcome never produces more than 3 RunJob calls; each
  expired reservation stays consumed.
- [ ] Blocking audit publisher, `RunJob`, and health-check fakes return by their respective
  10s/60s/15s injected deadlines; the reservation cannot expire while a permitted `RunJob`
  call is still live, and three timed-out dispatches terminalize.
- [ ] A permanent route error produces **0** reservations, **0** RunJob calls, terminal
  `failed`, and one notification.
- [ ] Barrier race A: stale detector wins terminal CAS, then dispatch failure returns; final
  state remains failed, attempts do not resurrect it, one notification.
- [ ] Barrier race B: completion wins, then stale detector and duplicate Cloud Run completion
  arrive; final result is unchanged, one notification.
- [ ] Explicit `RequeueTask` clears attempt/error fields and permits a fresh three-attempt
  budget; no automated path clears them.
- [ ] Existing local worktree-limit recovery still passes and does not consume cloud dispatch
  attempts.
- [ ] `go test ./internal/coordinator ./internal/dispatch/cloudrun ./internal/storage/firestore
  ./internal/executor/... ./cmd/ailang`, `make test`, `make lint`, and
  `make check-boundaries` pass.

## Test and Mutation Plan

Tests use fakes/emulators; they must not create paid Cloud Run work until rollout canary.

| Invariant | Test | Deliberate mutation that must turn it red |
|-----------|------|-------------------------------------------|
| Agent provider is authoritative | Historical config fixture inspects job/env pair | Replace `agent.Provider` with `cfg.DefaultProvider`; expected provider becomes wrong |
| Matrix blocks mismatch before side effect | Table test + zero fake `RunJob` calls | Remove the compatibility check; call count becomes 1 |
| Absolute path is pinned | Poison-`PATH` executable records which file ran | Pass `configured` instead of `resolved` to `CommandContext`; B runs |
| Failed lookup can recover | Missing→install→resolve test | Cache the first error; second lookup still fails |
| Retry is persistent and crash-bounded | 20 cycles plus three crash-point fixtures | Move increment after `RunJob` or refund expired lease; call count exceeds 3 |
| External waits are bounded below lease | Blocking fakes + injected clock/context assert 10s/60s/15s | Pass daemon background context directly; test remains blocked or live call overlaps lease |
| Terminal state cannot be resurrected | Barrier-controlled stale-wins race | Restore unconditional `ResetTaskToPending`; final state becomes pending |
| One winner owns notification | Duplicate completion + stale race | Notify outside `if won`; count exceeds 1 |

The executor package tests run with test-controlled directories and do not depend on a real
Codex/OpenCode/Pi/Motoko/Claude installation. Firestore tests use the existing emulator CI
pattern; a local skip is allowed only when it states that the emulator is absent, while CI
must exercise the transaction tests.

## Rollout and Rollback

1. Run `ailang coordinator config diff` against the live GCS generation and validate every
   cloud agent against the matrix. Do not write config as part of this sprint.
2. Land additive SQLite columns and Firestore-compatible fields. Old Firestore documents read
   missing attempt fields as zero.
3. Build all executor variants from the same commit, then deploy the coordinator. The current
   live sprint-planner route is already `codex/codex`, so validation should be a no-op.
4. Canary one bounded, low-cost codex task. Verify one route log, an absolute child path,
   `dispatch_attempts=1` after successful dispatch, cleared reservation fields, and one
   completion message.
5. Induce a fake/staging readiness failure (not production job breakage) and verify 3/1
   attempt/message counts before broad rollout.

Rollback may restore the previous binary; additive columns/Firestore fields are ignored by
old code. Before rollback, stop the new coordinator to avoid mixed writers, and inspect tasks
with nonzero `dispatch_attempts`. Do not delete the fields or reset task state automatically.

## Deferred Decisions

None. Constants, state transitions, compatibility matrix, and security boundary are frozen.
Any newly discovered provider/image family requires a design-doc verification-log amendment,
not an implementation-time fallback.

## Non-Goals

- Changing Cloud Run Terraform images, job `maxRetries=1`, or installing every CLI in every
  image.
- Fixing local launchd `PATH`; first-party evidence proves this incident ran in Cloud Run.
- Binary signature/hash attestation, immutable filesystem mounts, parent-directory ownership
  checks, or complete TOCTOU prevention.
- Replacing the collaboration message system with an exactly-once transactional outbox.
- Changing inner agent execution retry (`TaskExecutor.ExecuteWithRetry`) or eval-stream retry
  semantics owned by [m-eval-stream-health-retry](../v0_29_0/m-eval-stream-health-retry.md).
- General coordinator dispatch dedup/handoff work owned by
  [m-coord-dispatch-integrity](m-coord-dispatch-integrity.md).
- Editing the live multivac config or retroactively deleting historical timeout messages.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Legitimate legacy config uses an undocumented pair | Deploy-time rejection | Audit live config before deploy; matrix table is exhaustive and test-backed; no fallback |
| Firestore/SQLite transition semantics drift | Race remains in one store | Same outcome contract and fixtures; Firestore emulator CI plus SQLite unit tests |
| Early health check is expensive | Slower cold start | CLI `--version` is bounded by context and occurs once per job before clone; successful path is cached |
| Three attempts are too few during a long Cloud Run outage | More operator intervention | Failure is loud and explicit requeue resets the budget; an infinite retry is not acceptable |
| Resolver description overstates security | False assurance | Acceptance tests and non-goals explicitly limit the guarantee to absolute selection/pinning |

## Verification Log (first-party, 2026-08-29)

Base for all repository claims: HEAD and `origin/dev` were both `45503bac6`. Commands used a
fresh HEAD binary built into a temporary directory; no installed stale binary was trusted.

| # | Claim | Command / source | Observed |
|---|-------|------------------|----------|
| V1 | Queue evidence commissions this exact incident | `git show origin/dev:design_docs/v1-mission.md \| sed -n '512,517p'` | Names `task-a0628a5f`, the missing `opencode` error, pending state, repeated notices, and refutes local PATH stripping/dangling symlink |
| V2 | The child failure happened in a codex Cloud Run Job while provider was opencode | `gcloud logging read 'resource.type="cloud_run_job" AND (textPayload:"task-a0628a5f" OR jsonPayload.message:"task-a0628a5f")' --project ailang-multivac ...` | Execution `ailang-agent-executor-codex-6nrnx`; log says `running opencode executor`; then `exec: "opencode": executable file not found in $PATH` |
| V3 | Container retry was bounded and duplicated the child attempt, not the notification | `gcloud run jobs executions describe ailang-agent-executor-codex-6nrnx --region europe-west1 --project ailang-multivac` plus completion logs | `retriedCount: 1`, job `maxRetries=1`; second completion was skipped because task was terminal |
| V4 | HEAD reproduces the child boundary's missing-binary failure | `go test ./internal/executor/opencode -run 'TestExecuteStreaming_BinaryNotFound\|TestHealthCheck_MissingBinary' -count=1 -v` | Both tests PASS by observing the expected missing-binary failures |
| V5 | HEAD creates provider/variant from different authorities | `sed -n '90,225p' internal/coordinator/daemon_tasks_exec.go` | provider from `coordConfig.DefaultProvider` at 114-115; variant from `agent.ExecutorVariant` at 214-215 |
| V6 | Per-agent provider is already resolved during config load | `sed -n '570,595p' internal/coordinator/agent_config.go` | Empty agent provider inherits global default before registry use |
| V7 | Historical config created the exact split | In `ailang-multivac`: `git show d14998c -- config.yaml` and `git show c1fe8d0 -- config.yaml` | `d14998c` set global/default planner to opencode; `c1fe8d0` changed planner provider+variant to codex but retained global opencode |
| V8 | Current live config is coherent but code vulnerability remains | fresh HEAD `ailang coordinator config get <tempfile>` and inspect sprint-planner | Current planner is `provider: codex`, `executor_variant: codex`; this explains why a new paid reproduction is unnecessary and would not recur without recreating bad config |
| V9 | Retry/message storm counts are exact | Cloud coordinator/job logs and GCP message-store JSON filtered by task/message IDs | 41 dispatch failures, 8 stale terminal attempts, 1 successful dispatch; 8 timeout notices + 1 executor-failure notice = 9 messages |
| V10 | Unbounded cloud loop is an unconditional reset | `sed -n '245,260p' internal/coordinator/daemon_tasks_exec.go`; SQLite/Firestore `ResetTaskToPending` implementations | Every dispatch error resets to pending with no counter; both stores overwrite status unconditionally |
| V11 | Stale writer can lose and still notify | `sed -n '70,100p' internal/coordinator/stale_task_detector.go` | Unconditional `MarkTaskFailed`, then notification; no compare-and-set or winner result |
| V12 | Completion pre-read is not an atomic transition | `sed -n '1,175p' internal/coordinator/pubsub_completion_handler.go` and both store implementations | Later `MarkTaskFailed/Completed/PendingApproval` writes are unconditional; a concurrent writer can intervene |
| V13 | Inner retry is not the storm | `sed -n '145,245p' internal/coordinator/task_executor.go`; `daemon_tasks_exec_run.go:333` | Loop is bounded (`maxRetries=2`, three total); missing-executable text is not in retryable classes |
| V14 | Child adapters re-resolve bare names inconsistently | `rg -n 'LookPath\|CommandContext' internal/executor/{opencode,codex,pi,motoko,claude}` | opencode/codex/pi health checks look up a name but execute the configured string; motoko and Claude use separate resolution paths |
| V15 | No shared generic executor resolver exists | `rg -n 'type Executable|func .*Resolve.*Executable|LookPath' internal/executor` with known-positive `LookPath` hits | Adapter-local hits only; no generic resolver. The planned git resolver is git-specific and not yet implemented at this HEAD |
| V16 | Existing-tools check preceded design | `make help`; `find tools -maxdepth 2 -type f`; read `CLAUDE.md` workflows | Existing make targets, coordinator CLI, gcloud, and design creation script were used; no helper script proposed |
| V17 | Related-doc neural search found no duplicate | `create_planned_doc.sh m-coordinator-child-env-opencode-retry-storm v0_35_0` | Highest implemented similarity 0.43; highest planned 0.44, both below the 0.45 warning threshold and semantically distinct |
| V18 | Canonical inbox was clear before work | `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list --unread --json` | `null`; nothing acknowledged |
| V19 | The incident and HEAD use the gRPC error shape named by the classifier | `rg -n 'cloud.google.com/go/run/apiv2\|RunJob\\(' internal/dispatch/cloudrun/dispatcher.go`; `gcloud logging read '"task-a0628a5f"' ...` | HEAD imports `cloud.google.com/go/run/apiv2` and calls its `RunJob`; incident logs repeatedly contain `rpc error: code = FailedPrecondition desc = Job ... cannot be run because is in an error state` |
| V20 | HEAD does not currently enforce a dispatch-call deadline | `sed -n '105,265p' internal/dispatch/cloudrun/dispatcher.go`; `sed -n '105,130p' internal/coordinator/daemon_tasks_exec.go` | `Dispatch` passes its caller `ctx` directly to `RunJob`; daemon passes long-lived `d.ctx` to synchronous `PublishTask`; no local timeout exists, so D7 is required rather than presumed |
| V21 | Provider-specific budget selection shares the provider split | `sed -n '1,45p' internal/coordinator/daemon_tasks_budget.go`; `sed -n '225,250p' internal/coordinator/daemon_tasks_exec.go` | Both select budget provider from `coordConfig.DefaultProvider`; cloud dispatch must instead use the resolved route provider |

No `.ail` code or AILANG semantic claim is introduced, so `ailang prompt`/`ailang check` is
not applicable to this design.

## Related Documents

- [m-executor-variants](../../implemented/v0_15_0/m-executor-variants.md) — establishes that
  variant selects a separately baked Cloud Run image
- [m-exec-expand-codex-opencode](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) —
  original Codex/OpenCode adapter expansion
- [m-cloud-health](../../implemented/v0_9_0/m-cloud-health.md) — prior cloud health contracts
- [m-git-binary-resolution-sweep](m-git-binary-resolution-sweep.md) — related absolute-path
  security boundary, deliberately separate API
- [m-model-registry-single-source-sprint-plan](m-model-registry-single-source-sprint-plan.md) —
  current model-role resolution; this design composes its result into one route
- [v1 mission queue](../../v1-mission.md) — originating incident and acceptance direction

## Quorum Verification

**Round 1 — rejected, revised.** Artifact:
`.ailang/state/mission-quorum/m-coordinator-child-env-opencode-retry-storm-2026-08-29T07-04-38Z.json`.
Both reviewers were present. GPT-5.6 objected that post-failure counting was not crash-safe;
Gemini 3.1 Pro objected that the gRPC retry classifier lacked transport/error-shape evidence.
The revision reserves every attempt transactionally before side effects, consumes expired
reservations, adds three crash-point mutants, distinguishes bounded at-least-once outbound
dispatch from the sibling's inbound dedup scope, and adds V19 proving both the HEAD apiv2
client and incident `FailedPrecondition` error shape.

**Round 2 — rejected; quorum remains BLOCKED.** Artifact:
`.ailang/state/mission-quorum/m-coordinator-child-env-opencode-retry-storm-2026-08-29T07-07-53Z.json`.
Both reviewers agreed the pre-reservation and gRPC evidence fixed round 1, but independently
found that the draft presumed an unverified sub-lease `RunJob` deadline. V20 confirms HEAD
has no such deadline. D7 and the blocking-fake acceptance/mutation cases now require 10s
audit, 60s RunJob, and 15s child-health deadlines under a 2m lease; V21 also records the
budget-provider split requested by Gemini. Per the design skill's one-rerun limit, this
post-round-2 revision has not been resubmitted. Approval must wait for a fresh quorum pass.

## Future Work

- Binary provenance/ownership verification for all child tools.
- A transactional outbox if transport-level exactly-once notification becomes a requirement.
- Metrics/alert thresholds for fleet-wide dispatch-attempt distributions after the stored
  counter has enough production history.

---

**Document created**: 2026-08-29
**Last updated**: 2026-08-29
