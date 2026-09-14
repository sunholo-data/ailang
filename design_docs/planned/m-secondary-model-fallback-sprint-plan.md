# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Wire the existing failure classifier and model-chain machinery into the Cloud Run executor so an explicitly configured, same-harness fallback model can recover one transport failure inside the same execution. Preserve the stale-task detector as the sole re-dispatcher and make every attempt, model choice, and cost-affecting fallback visible in completion data.

**Design:** [m-secondary-model-fallback.md](m-secondary-model-fallback.md)  
**Duration:** 3 engineering days (about 20–24 hours, including integration and regression buffer)  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); model registry pricing and harness metadata  
**Risk Level:** High — this changes fleet execution, retry cost, and completion accounting, although activation remains per-agent opt-in

## Current Status Analysis

### Verified Baseline

- `ClassifyFailure` and `ResolveModelChain` exist in `internal/coordinator/retry_chain.go`, but the classifier has no production caller and an explicit model pin still produces a one-element chain.
- All current cloud agents are pinned, so existing role chains provide no fallback.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated by the cloud job.
- The stale-task detector already owns infrastructure re-dispatch and the two-execution cap; this sprint must not add another re-dispatch path.
- The worktree is clean. The available checkout is grafted to one recent commit, so recent LOC/day cannot be reconstructed reliably from git history. The design's two-day estimate is extended to three days to cover cross-package integration and regression work.

### Capacity and Estimate

- Planning capacity: approximately 300 changed LOC/day.
- Estimated total: 900 changed LOC (implementation, tests, and operator-facing example/config documentation).
- Buffer: roughly 25% is embedded in the three-day duration because executor seams and completion serialization are high-risk.

## Milestone 1: Explicit Chain Configuration and Startup Validation

**Goal:** Add opt-in per-agent fallback configuration and reject unsafe chains before dispatch.  
**Estimated:** 170 implementation + 180 tests = 350 LOC  
**Duration:** Day 1

**Files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- relevant agent-registry validation tests

**Tasks:**

- Add `fallback_models` to `AgentConfig`; append it only after an explicit pinned model.
- Add the one-walk/two-tail constants and the measured `signal: killed` transport signature while pinning hard-timeout and git failures as terminal.
- Validate chain length, duplicates, head repetition, registry resolution, same-harness compatibility, and the 3× blended-price ceiling at startup/reload.
- Prove empty fallback lists preserve every current agent's chain head and behavior.

**Acceptance Criteria:**

- [ ] `model: A` plus `fallback_models: [B]` resolves to `[A, B]`; an empty list still resolves to `[A]`.
- [ ] Invalid count, duplicate, head repetition, unknown model, harness mismatch, and price violation fail loudly with agent and model named.
- [ ] Exact measured strings distinguish 503/idle/signal-killed from hard-timeout/git failures.
- [ ] Registry tests cover all 37 current agents with no behavior change by default.

## Milestone 2: Dispatcher Contract and Bounded In-Container Walk

**Goal:** Pass the validated chain into the cloud job and execute at most one fallback attempt in isolated workdirs.  
**Estimated:** 180 implementation + 170 tests = 350 LOC  
**Duration:** Day 2

**Files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_executor.go` if an executor seam is required for deterministic tests
- associated dispatcher and cloud-job tests

**Tasks:**

- Emit `AILANG_MODEL_CHAIN` alongside the existing head model and double the job timeout only for multi-link chains, bounded by Cloud Run's maximum.
- Refactor the execution call behind the smallest testable seam and loop over attempts with `MaxInContainerWalks=1`.
- Give each attempt a fresh `attempt{n}` workdir and classify errors at their source.
- Publish once per execution; stop immediately on success, terminal class, exhausted chain, or spent walk budget.

**Acceptance Criteria:**

- [ ] A deterministic fake executor returning 503 on A succeeds on B in the same Cloud Run execution.
- [ ] Git and hard-timeout failures call the executor exactly once.
- [ ] No execution invokes more than two models, including when three models are configured.
- [ ] Attempt 2 cannot observe attempt 1's partial worktree.
- [ ] Existing `MaxTaskExecutions=2` and stale-task re-dispatch semantics remain unchanged.

## Milestone 3: Completion Accounting and Visible Fallback Evidence

**Goal:** Make the terminal model and every attempt observable on all completion paths.  
**Estimated:** 100 implementation + 80 tests = 180 LOC  
**Duration:** Day 3 morning

**Files to update:**

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- related Pub/Sub serialization and handler tests

**Tasks:**

- Add `CompletionAttempt` and `Attempts` with model, link, class, error, duration, and cost fields.
- Populate `ModelUsed` and `ChainLinkIndex` for success, terminal failure, walked success/failure, and deferred completion guards.
- Bank the terminal model and include model/link fields in inbox completion notifications.
- Emit a structured `MODEL_WALK` line and narrate both links in walked failure errors.

**Acceptance Criteria:**

- [ ] Every cloud completion carries terminal model and link index, including link 0 and deferred failure paths.
- [ ] Walked completions contain ordered attempt history; no-walk completions contain exactly one attempt.
- [ ] Banked results and inbox notifications expose the fallback model and link.
- [ ] JSON round-trip tests preserve attempt evidence without breaking older payload readers.

## Milestone 4: Opt-In Example, Full Regression, and Boundary Proof

**Goal:** Demonstrate safe configuration without silently enabling fleet-wide fallback, then close all repository gates.  
**Estimated:** 20 implementation/docs + 0–20 tests = 20 LOC (up to 40 if a fixture is needed)  
**Duration:** Day 3 afternoon

**Example/config files to update:**

- one test fixture or non-production example agent configuration showing `fallback_models`
- inline registry/config documentation adjacent to `AgentConfig`

**Tasks:**

- Add a validated example/fixture using two resolvable, same-harness models; do not modify a production agent unless separately approved.
- Run focused package tests, then repository test, lint, and architecture-boundary gates.
- Review the diff to confirm `internal/coordinator/stale_task_detector.go` is unchanged.

**Acceptance Criteria:**

- [ ] Example/fixture parses and passes startup validation.
- [ ] No production agent gains fallback configuration in this sprint.
- [ ] `go test ./internal/coordinator ./internal/pubsub ./cmd/ailang` passes.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.
- [ ] `git diff --exit-code -- internal/coordinator/stale_task_detector.go` succeeds relative to the sprint base.

## Day-by-Day Execution

1. **Day 1:** Write failing classifier/chain/validation tests; implement configuration and startup validation; close M1 on focused green tests.
2. **Day 2:** Write fake-executor integration tests; implement dispatch env, timeout scaling, isolated attempt workdirs, and bounded walk; close M2 without touching stale-task ownership.
3. **Day 3:** Add completion schema/accounting tests and implementation; add the non-production example/fixture; run full regression, lint, boundaries, and diff audit.

## Success Metrics

- A simulated transport failure on link 0 completes on link 1 with one execution and visible ordered attempt evidence.
- Git/auth and hard-timeout failures remain single-run terminal outcomes.
- Maximum remains two executions × two model invocations.
- 100% of cloud completion paths populate `ModelUsed` and `ChainLinkIndex`.
- Current agent registry behavior is unchanged unless `fallback_models` is explicitly present.
- All focused and repository-wide verification gates pass.

## Dependencies and Risks

- Model registry lookup must expose unambiguous wire-name, harness, and pricing data. If it does not, stop rather than add a silent default.
- Executor setup may be tightly coupled to process/env state. Keep the test seam local to the cloud-job package and avoid production-only fake modes.
- Completion schema consumers may assume absent attempt data. Preserve backward-compatible decoding and add round-trip coverage.
- Timeout multiplication can exceed provider/platform constraints. Clamp explicitly and test overflow/boundary cases.

## Non-Goals

- Cross-harness or local-GPU fallback.
- Activating fallback on any production agent.
- Making role chains the implicit tail of pinned models.
- New task states, CAS protocols, attempt fencing, or changes to stale-task re-dispatch.
- Quota-aware model selection beyond the approved transport classifier.

## Approval and Handoff

This plan creates execution artifacts only. Per repository gates, implementation begins only after the user says **execute sprint**; at that point hand off this plan and its JSON progress file to `sprint-executor`.
