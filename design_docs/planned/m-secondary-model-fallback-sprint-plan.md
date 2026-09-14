# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved, opt-in secondary-model fallback for cloud executor agents. A transport-class failure on a pinned model may run one compatible fallback inside the same Cloud Run execution, while terminal failures remain single-run and every attempt is visible in the completion payload.

**Duration:** 3 days (approximately 18 engineering hours)
**Target:** v1.2.0
**Dependencies:** Approved [design document](m-secondary-model-fallback.md); M-COORDINATOR-EXECUTION-TRUST M3 (landed)
**Risk Level:** High — this changes fleet-wide execution and cost control paths, although behavior remains opt-in.

## Current Status Analysis

### Completed

- `ClassifyFailure`, `ResolveModelChain`, `MaxTaskExecutions`, and stale-task re-dispatch bounds already exist with unit coverage.
- `TaskCompletion.ModelUsed` and `TaskCompletion.ChainLinkIndex` already exist, but are not populated.
- The design has been approved with explicit decisions for configuration ownership, failure taxonomy, loop/cost bounds, partial-work isolation, and observability.

### Velocity and Capacity

- The checkout exposes only one commit in the last seven days, so LOC/day cannot be calculated responsibly from local history.
- The design estimates approximately two days. This plan reserves three days (18 hours) to include cross-package integration tests and the full repository gates.
- Estimated change: 890 LOC total, including tests. The estimate is a sizing aid, not a delivery target.

### Remaining Work

- Add and validate explicit `fallback_models` configuration.
- Propagate the resolved chain and bounded job timeout through cloud dispatch.
- Execute at most one transport-class fallback in isolated attempt workdirs.
- Publish and bank terminal-link identity plus complete per-attempt history.
- Prove terminal classifications, cost bounds, back compatibility, and stale-detector invariants.

## Milestone 1: Configuration, Resolution, and Startup Validation

**Goal:** Make fallback intent explicit and reject unsafe chains before dispatch.
**Estimated:** 120 LOC implementation + 140 LOC tests = 260 LOC
**Duration:** 4 hours
**Files:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- new or existing registry validation test file under `internal/coordinator/`

**Tasks:**

- Add `FallbackModels []string` adjacent to the pinned model field.
- Append explicit fallbacks after the pin in `ResolveModelChain`; preserve existing role-chain behavior for unpinned agents.
- Add `MaxInContainerWalks = 1`, tail-length, duplicate, and pin-equality constraints.
- Resolve fallback entries through `modelreg`; validate same-harness compatibility and the 3× blended-price ceiling during registry validation/reload.
- Add the measured killed-process signature while pinning hard timeout, git, auth, refusal, and unknown errors as terminal.

**Acceptance Criteria:**

- An agent configured with pin A and fallback B resolves exactly `[A, B]`; an agent without fallbacks resolves exactly as it did before.
- Startup validation rejects unresolvable, cross-harness, duplicate, pin-equal, over-length, and over-price fallback entries with agent ID, entry, and reason in the error.
- Exact September strings prove idle mid-generation and killed-process failures walk, while hard timeout and git-push failures do not.
- Existing role-chain and chain-head parity tests remain green.

**Risk:** Model-registry identity and executor naming may not map one-to-one. Mitigation: use existing registry identity helpers and table-driven fixtures; do not add a second name resolver.

## Milestone 2: Dispatch Propagation and Bounded In-Container Walk

**Goal:** Run one fresh fallback attempt only for a transport-class executor failure, without re-dispatching the task.
**Estimated:** 170 LOC implementation + 150 LOC tests = 320 LOC
**Duration:** 7 hours
**Files:**

- `internal/coordinator/cloud_dispatcher.go`
- the concrete Cloud Run dispatcher and its tests under `internal/coordinator/`
- `cmd/ailang/coordinator_cloud.go`
- coordinator cloud execution tests under `cmd/ailang/`

**Tasks:**

- Carry the resolved chain in dispatch parameters and emit `AILANG_MODEL_CHAIN`; preserve `AILANG_MODEL` as the head for compatibility.
- Double the Cloud Run job timeout only when a fallback exists, bounded by Cloud Run's 24-hour maximum.
- Parse and validate the chain in the job wrapper before cloning.
- Extract a testable walk helper around executor invocation; call `ClassifyFailure` at the executor-error boundary.
- Give each model attempt a fresh `attempt{n}` workdir, discard failed partial work, and cap the loop with `MaxInContainerWalks`.
- Emit a structured `MODEL_WALK` decision without changing `AttemptCount` or invoking the stale-task dispatcher.

**Acceptance Criteria:**

- A deterministic fake executor returning 503 on A and success on B is invoked exactly twice in one execution and terminates on link 1.
- Git/auth/model/hard-timeout failures invoke only A; a third configured link is never invoked because the in-container walk budget is one.
- Attempt 2 uses a distinct fresh workdir and cannot observe files written by attempt 1.
- Empty fallback configuration preserves the prior execution path and effective environment, except `AILANG_MODEL_CHAIN` equals the pin.
- The compound ceiling remains two executions times two model invocations; `MaxTaskExecutions` and stale-task detector logic are unchanged.

**Risk:** `coordinator_cloud.go` has a deferred completion guard and many exit paths. Mitigation: keep one terminal publish, inject final attempt state into the existing closure, and test panic/early-error paths.

## Milestone 3: Completion Schema, Banking, and Operator Visibility

**Goal:** Make fallback use and cost attributable on every success and failure.
**Estimated:** 100 LOC implementation + 110 LOC tests = 210 LOC
**Duration:** 4 hours
**Files:**

- `internal/pubsub/topics.go`
- `internal/pubsub/pubsub_test.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- completion/finalization tests under `internal/coordinator/`

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts`, including model, link index, failure class, bounded error, duration, and cost.
- Populate `ModelUsed`, `ChainLinkIndex`, and attempts on every completion path, including no-walk failures and the deferred guard.
- Narrate walked failures with both links and the first failure class; retain attempt history on walked success.
- Bank the actual terminal model and expose `model_used` and `chain_link_index` in inbox completion notifications.
- Preserve JSON decode compatibility for older completion messages without the new field.

**Acceptance Criteria:**

- Pub/Sub round-trip tests preserve single- and two-attempt histories and accept legacy payloads.
- Every tested completion path supplies terminal model/link identity; walked success records A failed/transport then B succeeded.
- The banked execution result and completion notification name B and link 1 after fallback.
- A walked terminal failure names both attempted models and the classification that authorized the walk.

**Risk:** Attempt metrics can be accidentally double-counted or expose unbounded stderr. Mitigation: specify terminal metrics versus per-attempt metrics in tests and apply existing message-size/error-bounding conventions.

## Milestone 4: Fleet Regression Proof and Documentation

**Goal:** Demonstrate opt-in compatibility and close the sprint with repository-wide quality gates.
**Estimated:** 30 LOC implementation/docs + 70 LOC tests = 100 LOC
**Duration:** 3 hours
**Files:**

- registry/cloud regression tests under `internal/coordinator/` and `cmd/ailang/`
- `docs/docs/guides/coordinator.md` or the existing coordinator configuration reference, if fallback configuration is operator-facing there

**Tasks:**

- Add a registry-wide regression asserting all current agents without `fallback_models` retain their chain head and single-model behavior.
- Add a source/test invariant that the stale-task detector remains the sole re-dispatcher.
- Document `fallback_models`, one-walk limit, transport-only policy, price/harness boot validation, and completion observability.
- Run focused tests, then `make test`, `make lint`, and `make check-boundaries`.

**Acceptance Criteria:**

- The current agent registry produces no behavioral fallback unless an agent explicitly opts in.
- No production configuration is changed to enable a fallback in this sprint; rollout is a separately reviewable operator change.
- Deterministic tests cover 503 fallback success, idle versus hard-timeout classification, git terminal behavior, fresh workdirs, validation failures, completion fields, and the four-invocation absolute ceiling.
- `make test`, `make lint`, and `make check-boundaries` pass.
- Operator documentation states the cost bound and how to identify a fallback in completion data.

## Day-by-Day Plan

### Day 1

- Complete Milestone 1 with table-driven tests first.
- Start dispatch parameter/env propagation from Milestone 2.
- Run focused coordinator and model-registry tests.

### Day 2

- Complete the bounded execution loop and isolated workdirs.
- Land deterministic fake-executor integration coverage for walk and no-walk branches.
- Run focused `cmd/ailang` and coordinator tests.

### Day 3

- Complete payload, banking, notification, and compatibility tests.
- Add regression proof and operator documentation.
- Run formatting and all acceptance gates; update sprint JSON only with verified results.

## Success Metrics

- `ClassifyFailure` has a production call site at the executor-error boundary.
- Simulated 503 completes on link 1 within one execution; git and hard-timeout failures remain on link 0.
- `ModelUsed` and `ChainLinkIndex` are populated for all cloud completion paths covered by tests.
- Maximum model invocations are mechanically bounded at four across two executions.
- Current unconfigured agents retain single-model behavior.
- No `.ail` example is required: this is coordinator infrastructure, and the fake-executor integration test is the executable example.
- All repository gates pass.

## Dependencies and Sequencing

- Milestone 1 blocks Milestone 2 because dispatch must receive a validated chain.
- Milestone 2 blocks Milestone 3 because attempt history is produced by the walk loop.
- Milestone 4 follows all implementation milestones.
- The executor must not alter production agent YAML to opt in; implementation and rollout remain separate decisions.

## Open Questions

None. The approved design fixes the configuration location, failure boundary, walk budget, price ceiling, workdir policy, and observability contract.

