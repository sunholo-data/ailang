# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Add an explicit, bounded, and fully observable secondary-model fallback to cloud executor agents. A transport-class failure may walk once from an agent's pinned model to an operator-declared compatible fallback inside the same Cloud Run execution; auth, git, scope, hard-timeout, and other unrecognized failures remain terminal.

**Duration:** 3 engineering days (about 18–21 hours; the approved design estimates 2 implementation days plus one integration/verification day)
**Dependencies:** Approved `design_docs/planned/m-secondary-model-fallback.md`; M-COORDINATOR-EXECUTION-TRUST M3 is already landed
**Risk Level:** High — this changes fleet-wide cost and retry behavior, so fail-closed classification, cost bounds, and completion attribution are release-critical

## Current Status Analysis

### Repository findings verified against HEAD

- `ResolveModelChain` still returns only `agent.Model` for explicit pins; the configured role chain is consulted only when the pin is empty.
- `ClassifyFailure` still has no production caller. Only tests exercise it, while infrastructure re-dispatch remains in the stale-task detector.
- `TaskCompletion.ModelUsed` and `TaskCompletion.ChainLinkIndex` exist, but the cloud executor does not populate them.
- `AgentConfig` has a single `Model` field and no explicit fallback list. `DispatchParams` likewise carries only one model.
- The cloud job invokes `executeCloudTask` once, writes one artifact set, and publishes through a guarded completion closure. This is the correct single-owner surface for an in-container walk.
- `m-provider-failover.md` is eval-only, same-weight route failover and explicitly excludes coordinator/agent-mode changes; it does not overlap this sprint's cross-model cloud executor scope.
- Recent velocity cannot be calculated reliably from this shallow checkout: only one seven-day commit is visible and no usable LOC stat is available. Estimates therefore use the approved design's two-day range plus a conservative integration day.

### Scope boundary

This sprint adds per-agent `fallback_models`, validates every configured tail before dispatch, transports the resolved chain to the cloud job, walks at most once on a narrowly classified transport failure, and banks every attempt. It does not activate role chains for pinned agents, alter the stale-task detector, add cross-harness fallback, or configure the production fleet's 37 agents.

## Proposed Milestones

### M1: Explicit chain schema and fail-closed startup validation

**Goal:** Make fallback an operator-visible per-agent declaration and reject chains that could silently change harness or exceed the cost ceiling.
**Estimated:** 120 implementation LOC + 180 test LOC = 300 LOC
**Duration:** 5 hours

**Example files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/daemon_tasks_init.go`
- `internal/coordinator/agent_registry_test.go`
- `internal/coordinator/retry_chain_test.go`

**Tasks:**

- Add `FallbackModels []string` adjacent to `AgentConfig.Model`; append it after an explicit pin in `ResolveModelChain` without making role chains live for pinned agents.
- Define `MaxFallbackChainTail = 2`, `MaxInContainerWalks = 1`, and `MaxFallbackPriceRatio = 3.0` in the coordinator policy surface.
- Implement validation for pin presence, maximum tail length, duplicates, pin repetition, registry resolution by supported identity, matching executor/harness, and the blended price ratio.
- Run validation at coordinator startup and registry reload through the existing loud validation path; errors name the agent, offending model, and reason.
- Add `signal: killed` to the transport signatures and pin exact September classifications, including idle-stall walk, hard-timeout terminal, and git/auth terminal.

**Acceptance Criteria:**

- [ ] `model: A` plus `fallback_models: [B]` resolves exactly `[A, B]`; a pinned agent with no tail still resolves exactly `[A]`.
- [ ] Role chains remain unused when an explicit pin exists.
- [ ] Unresolvable, cross-harness, duplicate, over-length, pin-repeating, and >3x-price tails prevent startup with actionable errors.
- [ ] The exact measured idle and killed errors classify transport; hard timeout, git/deploy-key, auth, and scope refusals classify terminal.
- [ ] The shipped agent registry with no fallback declarations retains identical chain heads and passes validation.

### M2: Chain dispatch and bounded in-container execution loop

**Goal:** Carry the validated chain into the job and perform at most one fresh-worktree fallback attempt under explicit wall-clock and invocation bounds.
**Estimated:** 170 implementation LOC + 210 test LOC = 380 LOC
**Duration:** 7 hours
**Dependencies:** M1

**Example files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- the Cloud Run dispatcher implementation and tests identified by `DispatchParams` usage
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_test.go`

**Tasks:**

- Extend dispatch parameters with the resolved model chain and emit `AILANG_MODEL_CHAIN`; preserve `AILANG_MODEL` as the head for backward compatibility.
- Double the Cloud Run job timeout only when the chain has a tail, bounded by the platform maximum; keep each model attempt's existing hard/idle limits.
- Extract a testable walk-loop seam around executor invocation. Call `ClassifyFailure` on the executor error before completion publication.
- Run each model attempt in `/workspace/{taskID}/attempt{n}` from the same base commit. Never reuse a failed attempt's partial tree.
- Continue only when the class is transport, a next link exists, and `MaxInContainerWalks` has not been spent; otherwise publish the terminal result.
- Preserve the one-completion guard and leave `ShouldReDispatch`, `MaxTaskExecutions`, and stale-task ownership unchanged.

**Acceptance Criteria:**

- [ ] A deterministic fake executor returning 503 on A and success on B invokes exactly two models in one execution and does not increment task `AttemptCount`.
- [ ] A git-push, auth, scope, hard-timeout, or unknown failure invokes only A.
- [ ] A three-link chain still invokes at most two models because the loop enforces `MaxInContainerWalks=1`.
- [ ] Every attempted link gets a fresh worktree; partial files from A are absent from B.
- [ ] The compound upper bound remains two Cloud Run executions times two model invocations, with an automated assertion.
- [ ] A regression test proves stale-task detector behavior and its sole-re-dispatcher role are unchanged.

### M3: Completion provenance, attempt history, and banking

**Goal:** Make every model attempt and every cost-changing walk visible in the completion payload, stored result, inbox notification, and logs.
**Estimated:** 130 implementation LOC + 170 test LOC = 300 LOC
**Duration:** 5 hours
**Dependencies:** M2

**Example files to update:**

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- associated Pub/Sub and completion-handler tests

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts`, recording model, link index, failure class, bounded error text, duration, and attempt cost.
- Populate `ModelUsed` and `ChainLinkIndex` on every completion path, including preflight failures and the deferred panic/unknown-exit guard. Use the head/link 0 when no model invocation occurred and describe that state explicitly in attempts or error context.
- Aggregate terminal executor metrics without losing per-attempt cost; ensure the reported total cost cannot undercount a failed first run.
- Format walked failures so `ErrorMsg` names both links and the first failure class; walked successes still expose both attempts.
- Bank the actual terminal model and include `model_used` and `chain_link_index` in the completion inbox payload.
- Emit one structured `MODEL_WALK` log per transition without secrets or unbounded provider output.

**Acceptance Criteria:**

- [ ] Success, failure, walked, and non-walked completion fixtures all carry deterministic model/link attribution.
- [ ] A 503→success fixture reports A as a failed transport attempt, B as the serving attempt, `ModelUsed=B`, and `ChainLinkIndex=1`.
- [ ] A walked failure's message names A, B, and the transport classification; a walked success remains visible without relying on logs.
- [ ] Total completion cost includes both attempts and each attempt retains its own duration/cost.
- [ ] The banked execute result and agent inbox notification name the model that actually produced the terminal outcome.

### M4: Fleet-safe regression suite and operator documentation

**Goal:** Close the sprint with proof that fallback is opt-in, bounded, compatible, and observable before any production agent is configured.
**Estimated:** 35 implementation/docs LOC + 85 test/docs LOC = 120 LOC
**Duration:** 3–4 hours
**Dependencies:** M3

**Example files to update:**

- coordinator configuration documentation located during execution
- registry configuration examples/tests
- `CHANGELOG.md`
- focused integration test fixtures

**Tasks:**

- Add a configuration example documenting `fallback_models`, same-harness rules, maximum chain size, 3x price ceiling, and one-walk behavior.
- Add a registry-wide compatibility test proving agents without the field have unchanged dispatch behavior except the explicit head-only chain env.
- Run deterministic simulated-outage tests; do not wait for or manufacture a real provider outage.
- Run formatting, focused tests, `make test`, `make lint`, and `make check-boundaries`.
- Record production rollout as a separate attended configuration decision; this sprint must not silently add tails to all 37 agents.

**Acceptance Criteria:**

- [ ] Documentation states that fallback is opt-in and can increase spend, and shows where the serving model/attempt history is observed.
- [ ] No production agent gains a fallback declaration in this sprint.
- [ ] The full 503→success and terminal git/hard-timeout scenarios pass deterministically.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.

## Day-by-Day Plan

### Day 1

1. Complete M1 schema, validation, and failure-classification tests.
2. Start M2 by carrying the validated chain through dispatch and implementing the injectable execution-loop seam.
3. Run focused coordinator/model registry tests.

### Day 2

1. Complete M2 fresh-worktree walk and hard bounds.
2. Complete M3 attempt schema, completion attribution, cost aggregation, handler banking, and inbox visibility.
3. Run deterministic integration tests for walk success, walk failure, and terminal failures.

### Day 3

1. Complete M4 documentation and registry-wide backward-compatibility checks.
2. Audit every `publishCompletion` path and every `TaskCompletion` consumer for attribution handling.
3. Run formatting, full tests, lint, and architecture-boundary checks; capture any production configuration proposal separately for human approval.

## Success Metrics

- `ClassifyFailure` has a production call site at the executor-error decision point.
- A deterministic 503 on A completes on B inside one execution with two visible attempts and correct cumulative cost.
- Git/auth/scope/hard-timeout failures run no second model.
- Every cloud completion identifies the terminal model and link; every walked completion exposes both attempts.
- The runtime enforces one in-container walk and the existing system enforces two executions, bounding a task at four model invocations.
- All existing agents remain head-only until an operator explicitly configures a tail.
- `make test`, `make lint`, and `make check-boundaries` pass.

## Dependencies and Risks

- **Classifier false positives:** default all unknown errors to terminal and use exact measured fixtures; do not enumerate broad auth/git patterns as retryable.
- **Under-counted cost:** carry per-attempt metrics and test cumulative totals before enabling any tails.
- **Harness identity mismatch:** resolve registry identity and executor variant at startup; fail closed before dispatch.
- **Partial-work contamination:** use a fresh clone/worktree per link and retain only bounded diff/error evidence from failed attempts.
- **Completion schema fan-out:** audit all producers and consumers of `TaskCompletion`; additive JSON fields preserve compatibility, but attribution must not disappear in the handler.
- **Cloud Run timeout semantics:** centralize scaling and test the platform cap rather than multiplying unvalidated duration strings at call sites.
- **Fleet blast radius:** leave all registry tails empty in this implementation sprint; any rollout requires an explicit, reviewed config change with price impact visible.

## Executor Handoff Notes

- Begin with M1 and TDD the fail-closed validation/classifier boundaries before modifying execution behavior.
- Preserve the stale-task detector byte-for-byte if feasible; if a necessary compile-only change appears, stop and explain why the approved boundary cannot hold.
- Prefer an injected executor function/interface for deterministic loop tests over an environment-only production backdoor.
- Do not configure fallback models for the live fleet as part of implementation.
- A user must say **execute sprint** before implementation begins.
