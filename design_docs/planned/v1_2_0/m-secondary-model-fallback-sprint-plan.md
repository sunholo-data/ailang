# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

**Design doc:** [m-secondary-model-fallback.md](../m-secondary-model-fallback.md)  
**Created:** 2026-09-14  
**Target:** v1.2.0  
**Duration:** 3 days (approximately 21 engineering hours)  
**Estimated change:** ~800 LOC (implementation, tests, and documentation)  
**Risk level:** High — this changes cloud execution, retry cost, and completion accounting  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design doc

## Goal

Allow an explicitly configured cloud executor to make one same-harness fallback attempt after a transport-class model failure, within the same Cloud Run execution. The walk must be opt-in, cost-bounded, deterministic under test, and visible in every completion payload. Git/auth failures, hard timeouts, and other unclassified failures remain terminal.

## Current Status and Planning Basis

- `ResolveModelChain` returns only the explicit pin for pinned agents; all 37 production agents are therefore one-link chains.
- `ClassifyFailure` exists and has tests, but has no production call site.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated.
- The stale-task detector already owns infrastructure re-dispatch and remains out of scope.
- The checkout exposes only one grafted commit in the last seven days, so LOC/day cannot be derived responsibly from history. The estimate below uses the approved design's file inventory, current file sizes, test surfaces, and a 25% integration/risk reserve.
- No `.ail` language example is appropriate: this is coordinator infrastructure. The working examples are deterministic Go test fixtures for a two-link chain and registry YAML fixtures.

## Fixed Implementation Order

M1 must precede M2 because dispatch cannot serialize an unvalidated chain. M2 must precede M3 because the job needs a stable chain contract. M3 must precede M4 because completion banking can only be verified once attempts and terminal-link data exist. Milestones are independently testable and should be committed in this order.

## Milestone 1: Explicit Chain Configuration and Startup Validation

**Goal:** Represent opt-in fallback tails and reject unsafe chains before dispatch.  
**Estimate:** 80 implementation + 150 tests = **230 LOC**, 5 hours  
**Files:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/agent_registry_test.go` or a focused new `internal/coordinator/fallback_models_test.go`
- `internal/coordinator/retry_chain_test.go`

**Tasks:**

1. Add `FallbackModels` to `AgentConfig`; append it after the explicit pin in `ResolveModelChain`.
2. Add `MaxFallbackChainTail=2`, `MaxInContainerWalks=1`, and `MaxFallbackPriceRatio=3.0` in the owning coordinator package.
3. Validate count, duplicates, pin duplication, model resolution, same-harness compatibility, and blended price ratio at startup/reload. Empty tails remain a no-op.
4. Add the measured killed-process signature to transport classification while pinning hard timeout and git/auth errors as terminal.

**Acceptance criteria:**

- An agent configured with pin A and fallbacks B/C resolves exactly `[A,B,C]`; the chain head still matches `ResolveModel`.
- The current 37-agent registry, with no fallback tails, preserves its chain heads and behavior.
- Unresolvable, cross-harness, over-price, duplicate, pin-equal, and over-length tails each fail loudly with agent, entry, and reason in the error.
- Exact September error strings prove idle/mid-generation death and killed-process errors walk, while hard timeout and git-push errors do not.
- Focused coordinator tests and `go test ./internal/coordinator/...` pass.

**Risk:** model registry names and executor identities may not share one namespace. Mitigate with table tests using real registry rows and fail closed on ambiguity.

## Milestone 2: Dispatch the Validated Chain and Bound Wall Time

**Goal:** Carry the validated ordered chain into the Cloud Run job without changing unconfigured behavior.  
**Estimate:** 55 implementation + 85 tests = **140 LOC**, 3.5 hours  
**Files:**

- `internal/coordinator/cloud_dispatcher.go`
- `internal/coordinator/cloud_dispatch_test.go`
- relevant Cloud Run job configuration tests

**Tasks:**

1. Set `AILANG_MODEL_CHAIN` to the ordered comma-joined chain alongside the existing head model.
2. Preserve the existing environment for one-link agents except for `AILANG_MODEL_CHAIN=<pin>`.
3. Double the job timeout only for multi-link chains, capped at Cloud Run's 24-hour maximum; leave per-attempt hard/idle timeouts unchanged.
4. Add parsing/serialization and timeout boundary tests.

**Acceptance criteria:**

- One-link and multi-link env payloads contain the expected ordered chain with no namespace conversion.
- A one-link agent retains the existing timeout; a multi-link agent receives `min(2*taskTimeout,24h)`.
- Empty/malformed chain data fails loudly in the job parser; it never silently falls back to another source.
- Dispatcher package tests pass.

**Risk:** timeout units or Cloud Run limits may be applied in a different layer. Mitigate by testing the final job specification, not only a helper return value.

## Milestone 3: One Bounded In-Container Model Walk

**Goal:** Invoke `ClassifyFailure` at the executor error boundary and retry once on the next model only for transport failures.  
**Estimate:** 150 implementation + 170 tests = **320 LOC**, 8 hours  
**Files:**

- `cmd/ailang/coordinator_cloud.go`
- focused new `cmd/ailang/coordinator_cloud_model_walk_test.go`
- `internal/pubsub/topics.go`

**Tasks:**

1. Extract a testable bounded walk helper around `executeCloudTask`; inject/fake the executor rather than requiring a real provider outage.
2. Parse the dispatched chain and run each attempted link in an isolated `attempt{n}` worktree.
3. Walk only when `ClassifyFailure` returns transport, a next link exists, and the one-walk budget remains.
4. Add `CompletionAttempt` and accumulate model, link, class, error, duration, and cost for every attempted link.
5. Ensure the existing completion guard still publishes exactly once on success, terminal failure, walked failure, and deferred/panic paths.

**Acceptance criteria:**

- A fake 503 on A followed by success on B yields one task execution, two model calls, `ModelUsed=B`, `ChainLinkIndex=1`, and two ordered attempt records.
- Git-push failure and hard timeout on A make exactly one executor call and terminate at link 0.
- A transport failure on the final link, or after one walk, terminates without a third model call even if a third configured link exists.
- Attempt 2 receives a fresh workdir and cannot observe partial files from attempt 1; attempt 1's error/diff summary remains in completion history.
- Every exit path publishes one completion, never zero or two.
- Focused `cmd/ailang` tests pass.

**Risk:** `executeCloudTask` currently mixes clone, execution, git, and completion concerns. Keep extraction narrow; do not broaden this sprint into completion-path refactoring.

## Milestone 4: Bank and Surface Fallback Observability; Prove No Re-dispatch Regression

**Goal:** Make every terminal model/link and all walks visible to storage and inbox consumers, then run repository gates.  
**Estimate:** 45 implementation + 65 tests = **110 LOC**, 4.5 hours  
**Files:**

- `internal/coordinator/pubsub_completion_handler.go`
- `internal/coordinator/pubsub_completion_handler_test.go` (create or extend)
- `internal/pubsub/topics.go`
- `internal/coordinator/stale_task_detector_test.go`
- `docs/docs/guides/coordinator.md`

**Tasks:**

1. Bank `ModelUsed` and `ChainLinkIndex` in the execution result and pass both through completion notifications.
2. Preserve structured attempt history and narrate walked failures in `ErrorMsg`; emit a bounded `MODEL_WALK` log event.
3. Add payload tests for success/failure with and without a walk.
4. Prove the stale-task detector and `MaxTaskExecutions=2` behavior are unchanged; combined ceiling is two executions times two model runs.
5. Document configuration, opt-in semantics, price/walk limits, and observable fields.

**Acceptance criteria:**

- Every cloud completion variant carries non-empty `ModelUsed`, a valid `ChainLinkIndex`, and at least one attempt record.
- Inbox/portal notification payloads include `model_used` and `chain_link_index`; a walked failure names both models and the first failure class.
- Stale-task tests still prove it is the sole infrastructure re-dispatcher and never exceeds two executions.
- A combined bound test proves at most four model invocations per task across two executions.
- `make test`, `make lint`, and `make check-boundaries` pass.
- The selected coordinator guide documents `fallback_models`, the 3× price ceiling, one-walk cap, and completion observability.

**Risk:** storage structs may omit new fields during conversion. Add round-trip/payload assertions at the handler boundary rather than relying only on JSON marshaling tests.

## Day-by-Day Plan

### Day 1

- Complete M1 with registry-backed validation tests.
- Complete M2 and inspect the rendered Cloud Run job specification.
- Run coordinator-focused tests and commit only if the boundary is green.

### Day 2

- Implement M3 red-first with fake executor call counts.
- Cover success-after-walk, terminal failures, exhausted budget, isolated workdirs, and exactly-once completion.
- Run `go test ./cmd/ailang/... ./internal/pubsub/...` and coordinator tests.

### Day 3

- Complete M4 storage/inbox observability and stale-detector regression tests.
- Update the existing coordinator documentation page.
- Run full test, lint, and architecture gates; reserve remaining time for integration fixes.

## Success Metrics

- `ClassifyFailure` has a production call site at the executor error boundary.
- A deterministic 503 fixture completes on link 1 without re-dispatch.
- Git/auth and hard-timeout fixtures never invoke link 1.
- 100% of cloud completion paths identify the terminal model and chain link.
- Default behavior for agents without `fallback_models` remains unchanged.
- Maximum cost is structurally bounded to one fallback per execution and two executions per task.
- All focused and repository-wide verification commands pass.

## Deferred / Non-Goals

- Cross-harness fallback, role-chain tails for pinned agents, local GPU fallback, quota-aware link skipping, and changes to stale-task re-dispatch.
- Fleet-wide fallback configuration or production rollout. This sprint supplies and validates the mechanism; enabling agents is a separate operator decision because it changes cost.

## Handoff Conditions

This plan is ready for human review. Per repository routing rules, implementation begins only after the user explicitly says **“execute sprint”**. The executor must use the JSON progress artifact and update milestone `passes` fields as work is verified.
