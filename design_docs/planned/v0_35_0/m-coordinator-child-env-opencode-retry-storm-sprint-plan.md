# Sprint Plan: M-COORDINATOR-CHILD-ENV-OPENCODE-RETRY-STORM

**Design doc**: [m-coordinator-child-env-opencode-retry-storm.md](m-coordinator-child-env-opencode-retry-storm.md)  
**Target**: v0.35.0  
**Planned at**: 2026-08-29, mission iteration 302  
**Base**: `45503bac6` (`HEAD == origin/dev`)  
**Duration**: 4.25 engineering days (34 hours), plus an operator-controlled canary  
**Estimated change**: ~1,170 implementation LOC + ~1,870 test/fixture/doc LOC = ~3,040 LOC  
**Risk**: High — shared state transitions span SQLite and Firestore and guard paid Cloud Run effects  
**Dependencies**: No code dependency; coordinate security wording with
[m-git-binary-resolution-sweep](m-git-binary-resolution-sweep.md), without sharing APIs

## 0. Planning authority and scope

The design's round-2 quorum record says BLOCKED, but mission iteration 302 supplied the
ratified narrow-refinement carve-out: round 2 was blocked only on unverified external-call
deadlines, and the designer incorporated both reviewers' requested fixes verbatim as D7,
V20, and V21. This plan therefore treats the design as approved without altering its design
document or the mission log.

The derived planner lane output was VERBATIM `opus fail-closed:env-pin`. The driver reports
`MISSION_ANTHROPIC_AVAILABLE=0`, so this plan was produced through the documented Agent-tool
fallback `codex:gpt-5.6-sol`.

This is one sprint, split into five independently reviewable milestones. The executor must
not weaken or reinterpret these frozen invariants:

1. The resolved agent route `{provider, executor_variant, model}` is the sole cloud dispatch
   authority. `default_provider` participates only through config-load inheritance.
2. Route/config errors fail before a durable reservation and before every external side
   effect.
3. A reservation is committed before audit publication or `RunJob`; at most three
   reservations, and therefore at most three `RunJob` calls, survive crashes and restarts.
4. Terminal writes use compare-and-set. Only the winning transition may request an operator
   notification; message transport itself remains at-least-once.
5. The coordinator enforces separate internal deadlines of 10 seconds for audit publication,
   60 seconds for `RunJob`, and 15 seconds for child health. The reservation lease remains
   two minutes and cannot be replaced by caller/background context.
6. `ResetTaskToPending` remains local-worktree recovery only. No automated cloud path may use
   it or clear dispatch accounting; only explicit `RequeueTask` resets that budget.

No `.ail` source, parser/type semantics, public example program, live multivac config, Cloud
Run image, or job `maxRetries` setting is in scope. Internal fake executables and state-machine
fixtures are the appropriate examples for this operational feature.

## 1. Velocity and sizing

The sprint-planner velocity script found no trustworthy changelog LOC metric. The seven-day
git total (88,953 insertions across 918 files) is dominated by bulk/generated work and is not
a useful capacity signal. Recent comparable changes are smaller and narrower:

| Comparable work | Measured churn | Relevance |
|---|---:|---|
| `5cb003683` coordinator git-exec sweep | 276 insertions, 94 deletions across 13 files | Cross-cutting coordinator refactor |
| `45503bac6` pi install distribution completion | 360 insertions, 101 deletions across 10 files | Multi-package contract + tests |
| `83d306bff` pi install/uninstall/status | 1,441 insertions across 15 files | Recent one-day large feature lane |

This sprint touches more interfaces and two transactional backends, so the design's four-day
estimate receives a small integration buffer. Test LOC is intentionally larger than
implementation LOC because crash points, race barriers, deadline fakes, and two-store
semantic parity are the product.

| Milestone | Implementation | Tests / fixtures / docs | Estimate |
|---|---:|---:|---:|
| M1 — immutable route authority | ~220 | ~320 | 0.75d / 6h |
| M2 — pinned executable + health deadline | ~180 | ~360 | 0.75d / 6h |
| M3 — durable state machine in two stores | ~420 | ~520 | 1.25d / 10h |
| M4 — bounded dispatch + CAS notification ownership | ~300 | ~450 | 1.0d / 8h |
| M5 — integration, mutation, gates, rollout evidence | ~50 | ~220 | 0.5d / 4h |
| **Total** | **~1,170** | **~1,870** | **4.25d / 34h** |

## 2. Pristine-base measurements

All repository commands below were re-run in this worktree at `45503bac6`, with
`HEAD == origin/dev`. The only pre-existing untracked file was the approved design doc, which
does not affect compilation or tests.

| Acceptance command from the design | Base observation |
|---|---|
| `go test ./internal/coordinator ./internal/dispatch/cloudrun ./internal/storage/firestore ./internal/executor/... ./cmd/ailang` | **rc=0**, 43.87s |
| `make test` | **rc=0**, 84.59s |
| `make lint` | **rc=0**, 7.64s; 0 issues |
| `make check-boundaries` | **rc=0**, 0.45s |

These are regression gates, not proof of the new behavior: all are green before the feature
exists. The feature tests and mutation controls in §5 provide the falsifiable acceptance.

Additional base facts, measured rather than copied from the design:

| Fact | Base |
|---|---:|
| Cloud daemon references to `ResetTaskToPending` | 2 |
| Shared `internal/executor/executable.go` | absent |
| Local dispatch `WithTimeout` uses in daemon/Cloud Run dispatcher | 0 |
| Dispatch-attempt/reservation/error fields | 0 |
| `FinishTaskIfActive` implementations | 0 |
| `DefaultProvider` reads in cloud dispatch/budget files | 4 |
| Toolchain | Go 1.26.6, darwin/arm64 |

## 3. Lane constraints and hard gates

### 3.1 Ordinary test lane

The targeted suite and full `make test` pass on this host. The CI test job pre-downloads Go
modules, then poisons `HTTP_PROXY`/`HTTPS_PROXY` and sets `GOPROXY=off`; implementation tests
must therefore use fakes and cached modules and must not contact Cloud Run, GCS, provider
APIs, or the public internet. Live Codex/OpenCode/Pi/Motoko/Managed Agents tests are opt-in
and currently skip.

### 3.2 Loopback

The targeted package set contains seven existing `httptest.NewServer` files, and the pristine
targeted suite passed because loopback is available here. CI deliberately exempts
`localhost,127.0.0.1` via `NO_PROXY`. A more restrictive executor lane that cannot bind
loopback cannot honestly run the complete target or full suite; use the ordinary host/CI
lane. New route, reservation, race, and deadline tests should use in-process interfaces and
channels, not new listeners.

### 3.3 Firestore

`FIRESTORE_EMULATOR_HOST` is unset. A verbose pristine run of
`go test -count=1 -v ./internal/storage/firestore` passed only pure/cache/conversion tests;
it ran no coordinator transaction against Firestore. Repository search found no existing
Firestore emulator setup in `.github/workflows`, `Makefile`, `make/`, `internal/`, or `tools/`.
That contradicts the design's phrase “existing emulator CI pattern.”

This is a **hard M3 execution gate**, not permission for a silent skip:

- Local runs may skip emulator cases only with an explicit `FIRESTORE_EMULATOR_HOST is not
  set` diagnostic.
- CI must provision an isolated Firestore emulator, run the new transaction tests verbosely,
  and assert named PASS lines. A green pure-unit run is insufficient.
- Tests must use a unique project/collection namespace and delete only that namespace.
- Production Firestore is never a test target.

If the executor cannot provision the emulator in CI without expanding infrastructure scope,
it must pause after SQLite completion and request the required lane; it may not mark M3
passing.

### 3.4 Cloud/GCS and rollout

Unit/integration acceptance uses fake audit publishers and fake `jobRunner`; it creates zero
paid Cloud Run jobs and performs no GCS config write. The config-publication validator is
tested by calling the validation seam on bytes/in-memory config, not by publishing config.

The canary requires network, ADC with Cloud Run/log-read authority, a built executor image,
and explicit deployment/operator approval. It is rollout evidence after code gates, not a
reason to call cloud services during implementation. An unavailable cloud lane blocks the
canary record, not the mergeable code artifact.

## 4. Milestones

### M1: Immutable route authority (~540 LOC, 0.75 day)

**Goal:** Make one resolved per-agent route authoritative from config validation through job
name, provider environment, model, and provider-specific budget selection.

**Dependencies:** None

**Primary files:**

- New `internal/coordinator/execution_route.go`
- New `internal/coordinator/execution_route_test.go`
- Modify `internal/coordinator/daemon_tasks_exec.go`
- Modify `internal/coordinator/daemon_tasks_budget.go`
- Modify `internal/dispatch/cloudrun/dispatcher.go` and `_test.go`
- Modify `cmd/ailang/coordinator_config.go` and `_test.go`

**Tasks:**

1. Define immutable `ExecutionRoute` and `PermanentDispatchError`; resolve agent, inherited
   provider, legacy-Claude empty variant, and existing `ResolveModel(agent)` once.
2. Encode the complete provider/variant matrix as one validator shared by route resolution,
   config publication, and defensive dispatcher validation. Keep eval variants and local
   agents outside the cloud matrix exactly as the design specifies.
3. Construct `DispatchParams`, job suffix, `AILANG_PROVIDER`, model, and budget lookup from
   the route. Remove cloud-path reads of `cfg.DefaultProvider` after route resolution.
4. Validate config after defaults are applied and before the GCS CAS write. Tests call the
   validation seam only; they do not publish.
5. Log agent ID, provider, variant, and model together without secrets.

**Acceptance:**

- [ ] Historical `default_provider=opencode` + planner `codex/codex` produces a `-codex` job
  and `AILANG_PROVIDER=codex`; budget selection is also `codex`.
- [ ] `opencode/codex`, every off-diagonal known pair, and unknown values fail with typed
  permanent errors naming agent/provider/variant before reservation or fake `RunJob`.
- [ ] Every allowed matrix cell passes; legacy empty variant normalizes only for Claude.
- [ ] Config-validation and dispatch-validation negative fixtures each observe zero fake
  external calls. Local-lane agents remain exempt.

**Review stop:** Any new provider or variant not frozen in the design requires a design
amendment; do not add a fallback cell.

### M2: Pinned executables and enforced child-health deadline (~540 LOC, 0.75 day)

**Goal:** Resolve each CLI to one validated absolute path, reuse it for health and execution,
and bound cloud health preflight to 15 seconds.

**Dependencies:** M1 for structured permanent cloud preflight errors

**Primary files:**

- New `internal/executor/executable.go` and `_test.go`
- Modify `internal/executor/{opencode,codex,pi}/...`
- Modify `internal/executor/motoko/{motoko.go,healthcheck.go}` and tests
- Modify `internal/executor/claude/claude.go` and tests
- Modify `cmd/ailang/coordinator_cloud_executor.go` and tests

**Tasks:**

1. Implement the dependency-free, concurrency-safe `Executable` resolver: empty refusal,
   clean/stat absolute inputs, absolute `LookPath` result for bare names, regular-file and
   Unix execute-bit checks, and success-only caching.
2. Migrate all five CLI adapters so `HealthCheck` and execution share the resolver and
   `exec.CommandContext` receives only its returned absolute path. Preserve Claude NVM
   discovery as candidate selection and remove Motoko's second resolution path.
3. Run cloud executor health before clone/plugin setup. Wrap the health call internally in
   `context.WithTimeout(..., 15*time.Second)` even when the caller supplies background or a
   longer deadline; return a structured permanent dispatch error on missing/non-executable
   CLI.
4. Provide an injectable duration seam for fast deterministic tests while separately
   asserting the production constant is exactly 15 seconds.

**Acceptance:**

- [ ] Poison-`PATH`: resolve `A/tool`, switch `PATH` to `B`, health/execute, and prove only
  absolute `A/tool` runs.
- [ ] Missing then installed binary recovers, proving failed lookups are not cached; two
  successful concurrent resolves choose and cache one absolute result without a race.
- [ ] Empty, relative lookup result, directory, and non-executable Unix file are rejected.
- [ ] A channel-blocked health fake observes an internal deadline, exits on `ctx.Done`, and
  performs no clone/plugin setup; the production default is exactly 15 seconds.
- [ ] Tests make no provenance, directory-ownership, signature, hash, or full-TOCTOU claim.

### M3: Durable reservation and terminal CAS parity (~940 LOC, 1.25 days)

**Goal:** Implement the same crash-bounded dispatch state machine and terminal compare-and-set
semantics in SQLite and Firestore.

**Dependencies:** M1 outcome/error types; Firestore emulator lane from §3.3

**Primary files:**

- New `internal/coordinator/dispatch_state.go` and `_test.go`
- Modify `internal/coordinator/store.go`, `store_sqlite.go`,
  `store_sqlite_queries.go`, and tests
- Modify `internal/storage/firestore/coordinator_convert.go`,
  `coordinator_transitions.go`, and `feedbackgate_stores_test.go` (or a narrowly named new
  emulator test file)
- Update every mock/adapter required by the expanded `Store` contract, including
  `mock_store_test.go` and `approval_watcher_mock_test.go`
- Add the minimal CI emulator provisioning/assertion needed by §3.3 if no shared repository
  facility appears before execution

**Tasks:**

1. Add backward-compatible task fields with zero-value reads:
   `dispatch_attempts`, `dispatch_reservation_id`, `dispatch_reserved_at`, and
   `last_dispatch_error`. Add an additive SQLite migration and symmetric scan/persist plus
   Firestore conversion.
2. Implement `ReserveDispatchAttempt`, `RecordDispatchOutcome`, and
   `RecoverExpiredDispatchReservation` transactionally in both stores. Reservation increments
   before setting queued; expired leases are consumed, never refunded; mismatched reservation
   IDs and non-active states return `LostRace` without mutation.
3. Implement `FinishTaskIfActive` as SQLite conditional transaction/update and Firestore
   transaction, preserving result/error metadata for failed, completed, and pending-approval
   targets. Return `won=false` for a terminal or otherwise disallowed source state.
4. Make explicit `RequeueTask` clear all four dispatch fields; ensure no other API clears the
   attempt budget. Keep local `ResetTaskToPending` behavior independent of cloud accounting.
5. Drive identical semantic fixtures against both backends: reservation limit, restart,
   expired lease at attempts 1/2/3, stale reservation ID, CAS winner, requeue, and missing-field
   compatibility. Use injected times and barriers, never sleeps.

**Acceptance:**

- [ ] SQLite migration/reopen preserves attempt/error/reservation state and reads old rows as
  zero values.
- [ ] Firestore conversion reads old documents as zero values and emulator transactions pass
  the same outcome table as SQLite.
- [ ] Twenty reserve/outcome cycles permit exactly three reservations; a fourth has no write.
- [ ] Recovery after attempt 2 permits exactly one more reservation; attempt-3 expiry
  terminalizes and never refunds.
- [ ] `FinishTaskIfActive` admits one winner from allowed active states and no terminal
  resurrection under barrier-controlled competitors.
- [ ] `RequeueTask` alone clears all dispatch fields and enables a fresh three-attempt budget;
  local reset does not consume or clear cloud attempts.
- [ ] CI output contains named PASS evidence for the Firestore emulator transaction suite;
  a skip is a failure in CI.

### M4: Pre-side-effect dispatch, bounded waits, and one notification owner (~750 LOC, 1 day)

**Goal:** Compose M1/M3 into the daemon so every external call is reservation-backed, bounded,
and unable to resurrect or duplicate-notify a terminal task.

**Dependencies:** M1 and M3; M2 for the cloud child preflight path

**Primary files:**

- Modify `internal/coordinator/daemon_tasks_exec.go`
- Modify `internal/coordinator/stale_task_detector.go`
- Modify `internal/coordinator/pubsub_completion_handler.go`
- Modify daemon/completion/stale tests and fake publisher/notifier seams
- Modify `internal/dispatch/cloudrun/dispatcher.go` and tests for the 60-second call boundary

**Tasks:**

1. Resolve and validate route before reservation. Permanently fail an invalid route through
   the terminal CAS path with one winner-owned notification.
2. Commit `ReserveDispatchAttempt` before audit publication. A fake publisher must be able to
   read the already-queued task and matching reservation during its callback. Record audit
   failure as the reserved attempt's outcome; never refund it.
3. Wrap audit in a fresh internal 10-second context and `RunJob` in a distinct fresh internal
   60-second context. Do not pass the daemon's long-lived/background context through. Assert
   `CloudDispatchLease == 2*time.Minute` and `lease > RunJobDeadline`.
4. Classify only typed gRPC status codes: retry
   `Unavailable`, `ResourceExhausted`, `DeadlineExceeded`, and `FailedPrecondition`; all
   unrecognized/authority/config failures are permanent. No substring classifier and no
   untested REST inference.
5. Record every outcome against its reservation. Only `WillRetry` returns pending; only
   `WonTerminal` notifies; `LostRace` is an idempotent no-op. Remove cloud uses of
   `ResetTaskToPending`.
6. Route stale detection and Pub/Sub completion through terminal CAS. A completion for an
   already terminal task is acknowledged/logged without notification or result overwrite.
7. Inject short test durations and channel-blocked fakes for no-sleep deadline tests, while
   separately asserting production defaults are exactly 10s/60s/15s.

**Acceptance:**

- [ ] A side-effect-order fake asserts durable matching reservation state inside both audit
  publish and `RunJob`; route failures observe 0 reservations, 0 audit calls, and 0 RunJob.
- [ ] Twenty retryable readiness polls yield exactly 3 reservations, 3 RunJob calls, final
  failed state, `dispatch_attempts=3`, and one failure notification.
- [ ] Crash fixtures after reservation/before RunJob, after RunJob error/before outcome, and
  after RunJob success/before outcome never exceed three calls; expired reservations stay
  consumed.
- [ ] Blocking audit and RunJob fakes exit from the internally injected deadlines; production
  constants assert 10 seconds and 60 seconds, and three timed-out attempts terminalize.
- [ ] Barrier A: stale wins, late dispatch failure returns `LostRace`; final state/result and
  one notification are unchanged. Barrier B: completion wins, then stale and duplicate
  completions are idempotent; one notification.
- [ ] Permanent route error reaches failed with one notification but 0 reservations and 0
  external calls.
- [ ] Source scan/test guard proves cloud dispatch no longer calls `ResetTaskToPending`.

### M5: Integration, mutation evidence, regression gates, and canary handoff (~270 LOC, 0.5 day)

**Goal:** Prove the composed contract, kill the deliberate mutants, update operator-facing
notes, and prepare—but do not silently perform—the controlled rollout.

**Dependencies:** M1–M4

**Primary files:**

- Integration fixtures in the packages above
- `changelogs/v0.32-current.md` under `## [Unreleased]` (root `CHANGELOG.md` remains index-only)
- Optional operator evidence artifact only if the authorized canary lane is available

**Tasks:**

1. Run every named positive/negative fixture and every mutation in §5. Restore each mutant
   from an explicit temporary backup; never use destructive git restoration.
2. Run the exact targeted command, `make test`, `make lint`, and `make check-boundaries`.
3. Document additive migration, route rejection, retry bound, deadlines, and explicit-requeue
   recovery in the current changelog.
4. For an authorized rollout only: diff/validate live config without writing, build all
   variants from one commit, deploy coordinator, canary one low-cost `codex/codex` no-op, and
   record route, absolute child path, one attempt, cleared reservation, terminal state, and
   one completion message. Exercise readiness failure only in fake/staging, not production.

**Acceptance:**

- [ ] All mutation controls in §5 build, land, turn their named test red for the intended
  semantic reason, restore cleanly, and return green.
- [ ] All four pristine-green regression commands remain rc=0.
- [ ] No live config, production task, or Cloud Run job is changed without explicit operator
  authorization.
- [ ] Changelog names the compatibility rejection, three-attempt bound, explicit requeue,
  and 10s/60s/15s deadlines.

## 5. Falsifiability and mutation protocol

For every code mutant: copy the file to an explicit temporary directory, apply one semantic
mutation, assert the file hash changed, assert the mutant still compiles, run only the named
test and require nonzero status, restore from the backup, and require the test to pass.
A compile failure does not count as a killed mutant.

| Invariant | Positive / negative control | Build-preserving mutation that must turn it red |
|---|---|---|
| Agent route is authoritative | Historical default-opencode/planner-codex fixture inspects job/env/budget | Substitute `cfg.DefaultProvider` for `route.Provider` |
| Matrix blocks before effect | Mismatch fixture asserts 0 reservations/audit/RunJob | Bypass compatibility validation |
| Dispatcher has its own guard | Call dispatcher directly with mismatch; RunJob count 0 | Remove defensive dispatcher validation only |
| Absolute executable stays pinned | Poison `PATH` A→B and inspect marker | Pass configured bare name to `CommandContext` |
| Failed lookup recovers | Missing→install→resolve | Cache the first error |
| Reservation precedes effects | Fake reads matching persisted reservation inside callback | Move reservation after audit or after `RunJob` |
| Three-call bound survives crash | Three crash points + 20-poll fixture | Increment after `RunJob`, or refund expired lease |
| Reservation identity is CAS authority | Stale reservation outcome returns `LostRace` | Ignore reservation-ID mismatch |
| Terminal cannot resurrect | Stale-wins and completion-wins barriers | Restore unconditional reset/write |
| Winner alone notifies | Duplicate completion/stale/dispatch competitors; count 1 | Move notify outside `if won`/`WonTerminal` |
| Audit is bounded to 10s | Deadline-capturing + channel-blocked fake | Pass caller/daemon context directly |
| RunJob is bounded to 60s | Deadline-capturing + channel-blocked fake | Pass caller context directly |
| Health is bounded to 15s | Deadline-capturing health fake, clone count 0 | Remove health wrapper or start clone first |
| Lease cannot undercut RunJob | Constant relation test: 2m > 60s | Set lease to `<= RunJobDeadline` |
| Requeue is explicit reset | Requeue clears all fields; automated outcomes retain count | Clear attempts in retry/recovery/local reset |
| Firestore matches SQLite | Shared semantic table against SQLite + emulator | Make one Firestore transaction unconditional |

Deadline tests use two independent controls:

- A deadline-capturing fake returns immediately and asserts the production context deadline is
  approximately now+10s/60s/15s, proving the configured values without waiting 85 seconds.
- A channel-blocked fake exits only on `ctx.Done()` under injected millisecond durations,
  proving cancellation behavior without sleeps. An outer test timeout prevents a broken
  mutant from hanging the suite.

## 6. Final command gates

Run in this order after M5, with Firestore emulator PASS evidence captured separately:

```bash
go test ./internal/coordinator ./internal/dispatch/cloudrun ./internal/storage/firestore ./internal/executor/... ./cmd/ailang
make test
make lint
make check-boundaries
```

Also run focused race detection where the new concurrency lives:

```bash
go test -race ./internal/coordinator ./internal/executor ./internal/storage/firestore
```

`-race` is a regression gate, not a substitute for barrier-controlled semantic races. If
the Firestore emulator cannot run under `-race`, keep its named transaction suite as a
separate required CI command and run `-race` over the in-process packages.

## 7. Success metrics and handoff

- One route source controls provider, image variant, model, job env, and budget.
- Every external dispatch call is preceded by a durable reservation; automated calls are
  bounded at three across restarts and crash points.
- Every terminal race yields one stored winner and at most one notification decision.
- Internal deadline defaults are exactly audit=10s, RunJob=60s, health=15s under lease=2m.
- SQLite and Firestore emulator pass the same state-machine fixtures.
- No new public example is required; fake binaries, fake job runners, and shared store
  fixtures are created and verified.
- The sprint remains `not_started` until the user explicitly says **execute sprint**. No
  sprint-executor handoff or cloud action occurs during planning.

## 8. Blockers and pause conditions

There is no blocker to approving this sprint plan. There is one known execution-lane blocker:
the repository currently has no Firestore emulator CI facility. M3 cannot be called complete
until one is provisioned and its named transaction tests visibly pass. The authorized cloud
canary is separately gated on operator credentials/deployment approval and may remain pending
after the mergeable code sprint completes.

Pause and return to design only if implementation evidence disproves the gRPC transport shape,
requires a new provider/variant matrix cell, or cannot preserve the frozen 10s/60s/15s/2m
relationship. Ordinary interface fallout, CI emulator provisioning, and mock updates are
implementation work already covered by this plan.
