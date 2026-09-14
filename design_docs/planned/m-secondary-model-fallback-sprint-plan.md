# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement an explicit, same-harness secondary-model fallback for cloud executor agents. A transport-class failure may make one visible, cost-bounded in-container walk to a configured fallback, while git/auth/model failures remain terminal and infrastructure failures retain the existing stale-task re-dispatch path.

**Duration:** 2.5 engineering days (about 20 hours)
**Dependencies:** Approved `design_docs/planned/m-secondary-model-fallback.md`; M-COORDINATOR-EXECUTION-TRUST M3 (landed)
**Risk Level:** High — this changes fleet execution, cost accounting, and completion semantics
**Estimated change:** ~840 LOC (implementation, tests, configuration example, and docs)

## Current Status Analysis

### Verified baseline

- All 37 production agents have an explicit model pin, so `ResolveModelChain` currently returns a one-element chain.
- `ClassifyFailure` exists but has no production call site.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated.
- The stale-task detector is the sole infrastructure re-dispatcher and is explicitly out of scope.
- No live agent has `fallback_models`, so rollout can remain opt-in and behavior-preserving.

### Velocity and estimate basis

- The checkout exposes only one recent commit, so a defensible recent LOC/day average cannot be calculated.
- The approved design estimates roughly two days. This plan adds a half-day integration and regression buffer because the work crosses configuration, dispatch, execution, Pub/Sub payloads, and banking.
- Capacity target: ~840 LOC over 2.5 days, including ~430 LOC of tests and fixtures.

## Milestones

### M1: Explicit chain configuration and startup validation

**Goal:** Add opt-in per-agent fallback configuration and reject unsafe chains before dispatch.
**Estimated:** 140 LOC implementation + 150 LOC tests = 290 LOC
**Duration:** 0.75 day
**Dependencies:** None

**Files:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- agent-registry validation tests (existing test file or a focused new `_test.go`)

**Tasks:**

- Add `fallback_models` to `AgentConfig` and append it after the explicit pin in `ResolveModelChain`.
- Add and enforce `MaxFallbackChainTail = 2`, `MaxInContainerWalks = 1`, and `MaxFallbackPriceRatio = 3.0`.
- Resolve pin and fallback entries through `modelreg`; reject unresolved, duplicate, head-equal, harness-incompatible, over-length, and over-price entries with agent-specific errors.
- Extend classifier coverage for the exact measured failure strings: killed and mid-generation failures walk; hard timeout and git/auth failures remain terminal.
- Prove that agents without `fallback_models` retain the same chain head and single-link behavior.

**Acceptance criteria:**

- Startup validation loudly names the agent, fallback entry, and reason for every invalid case.
- Empty fallback configuration is a no-op; the current 37-agent registry validates without dispatch behavior changes.
- `ResolveModelChain` returns `[pin, fallback...]` in stable order and never more than three entries.
- Exact September failure strings have table-driven classifier tests, including hard-timeout versus idle-stall separation.

**Risk:** Model registry identifiers and executor/harness names may use different namespaces. Mitigation: test against real registry rows and keep all conversion inside one validation helper.

### M2: Dispatch chain and bounded in-container walk

**Goal:** Carry the validated chain into the Cloud Run job and execute at most one fallback attempt on transport failure.
**Estimated:** 150 LOC implementation + 170 LOC tests = 320 LOC
**Duration:** 0.9 day
**Dependencies:** M1

**Files:**

- `internal/coordinator/cloud_dispatcher.go`
- `cmd/ailang/coordinator_cloud.go`
- focused dispatcher/job execution tests

**Tasks:**

- Emit `AILANG_MODEL_CHAIN` while preserving `AILANG_MODEL` as the head; scale the job timeout only for multi-link chains and retain the platform maximum.
- Parse the chain in the job binary and place each attempt in an isolated `attempt{n}` work directory.
- Wrap executor invocation in a testable bounded loop; invoke `ClassifyFailure` at the executor error boundary.
- Walk only on `FailureTransport`, only when a tail exists, and only once. Publish exactly one terminal completion.
- Preserve the stale-task infrastructure path and `MaxTaskExecutions` unchanged.

**Acceptance criteria:**

- A fake executor returning 503 on A then success on B runs twice in one execution and terminates on link 1.
- Git failure and hard timeout on A invoke the fake executor exactly once and terminate on link 0.
- The loop cannot exceed two model invocations per execution, including with a three-entry chain.
- Attempts use separate work directories; attempt 2 cannot observe attempt 1's partial tree.
- Single-link dispatch remains behaviorally identical apart from `AILANG_MODEL_CHAIN=A`.

**Risk:** The current executor call is tightly coupled to process setup. Mitigation: introduce the smallest injectable runner boundary needed by deterministic tests; do not refactor unrelated cloud execution code.

### M3: Completion observability and accounting

**Goal:** Make every attempted model and every walk decision visible in completion state, inbox notifications, logs, and banked results.
**Estimated:** 90 LOC implementation + 80 LOC tests = 170 LOC
**Duration:** 0.5 day
**Dependencies:** M2

**Files:**

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- completion payload and handler tests

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, class, error, duration, and cost fields.
- Populate `ModelUsed`, `ChainLinkIndex`, and attempt history on success, failure, panic, and deferred-guard completion paths.
- Narrate walked failures in `ErrorMsg` and emit a structured `MODEL_WALK` stdout record.
- Bank the terminal model and include model/link fields in the agent inbox completion notification.

**Acceptance criteria:**

- Every cloud completion identifies its terminal model and link; walked completions contain both attempts.
- Walked success remains visibly a fallback success, not an indistinguishable link-0 success.
- Walked failure names both links and the first failure class.
- Payload round-trip and completion-handler tests cover walked success, walked failure, terminal link-0 failure, and deferred/panic paths.

**Risk:** Cost may not be available at every failure boundary. Mitigation: represent unavailable cost explicitly as zero/omitted according to the existing payload convention; never estimate silently.

### M4: Fleet fixture, regression gates, and documentation

**Goal:** Prove opt-in rollout safety and close the sprint with repository-wide validation.
**Estimated:** 20 LOC configuration/example + 40 LOC tests/docs = 60 LOC
**Duration:** 0.35 day
**Dependencies:** M1, M2, M3

**Files:**

- an existing non-production registry fixture or test fixture (do not enable fleet fallback by default)
- `design_docs/planned/m-secondary-model-fallback.md` status/reality notes after implementation
- `changelogs/v0.18-current.md`

**Tasks:**

- Add a test-only two-link agent configuration demonstrating the supported YAML shape.
- Add a compound-bound test proving the ceiling of two executions times two model runs.
- Run focused packages, full tests, lint, and architecture-boundary checks.
- Record implementation deviations and final metrics; move design and sprint docs according to repository convention only after acceptance passes.

**Acceptance criteria:**

- No production agent gains `fallback_models` as part of this sprint; rollout remains a separate operator action.
- `make test`, `make lint`, and `make check-boundaries` pass.
- The stale-task detector is unchanged (verified by review/diff) and its existing tests pass.
- Changelog and design status accurately describe the opt-in behavior, one-walk bound, and observable cost change.

**Risk:** Repository-wide tests may expose unrelated baseline failures. Mitigation: capture the baseline before implementation and distinguish pre-existing failures without weakening any gate.

## Execution schedule

### Day 1

- Complete M1 and its focused tests.
- Begin M2 with chain env propagation and the injectable runner boundary.

### Day 2

- Complete M2's bounded loop, isolated work directories, and deterministic tests.
- Complete M3 payload and completion-handler propagation.

### Day 3 (half day)

- Complete M4, run all gates, resolve integration defects, and update design/changelog artifacts.

## Success metrics

- `ClassifyFailure` has a production call site at the executor error boundary.
- Deterministic 503 simulation completes on link 1 without re-dispatch.
- Git/auth and hard-timeout simulations never run link 1.
- All cloud completions carry terminal model/link; walked runs carry complete attempt history.
- Maximum work is mechanically bounded to two model invocations per execution and four per task across the existing two-execution ceiling.
- Current registry remains single-link and behavior-compatible until explicitly configured.
- Full test, lint, and architecture-boundary gates pass.

## Non-goals

- Cross-harness fallback, role-chain activation for pinned agents, local GPU fallback, or coordinator retry-state changes.
- Changing `MaxTaskExecutions`, stale-task detector ownership, or enabling fallback on the production fleet.
- Treating hard timeout, git, credentials, auth, refusal, or wrong-answer outcomes as transport failures.

## Open questions for execution

- Confirm the exact existing field used to derive an agent's effective harness during registry validation; do not introduce a second source of truth.
- Confirm whether failed executor results expose measured cost. If absent, leave attempt cost unavailable rather than synthesizing it.

