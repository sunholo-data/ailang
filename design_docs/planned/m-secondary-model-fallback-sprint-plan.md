# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement an explicit, same-harness secondary-model fallback for cloud executor agents. Transport failures may make one bounded in-container walk to an operator-configured fallback, while git/auth/model outcomes remain terminal and every attempt is visible in completion data.

**Duration:** 3 engineering days (about 20 hours)
**Target:** v1.2.0
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design `design_docs/planned/m-secondary-model-fallback.md`
**Risk Level:** High — this changes cloud execution control flow, timeout/cost bounds, and completion accounting.

## Current Status Analysis

### Verified baseline

- All 37 cloud agents have explicit model pins, and `ResolveModelChain` currently returns a one-element chain for every pinned agent.
- `ClassifyFailure` exists with tests but has no production call site.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated.
- The stale-task detector already owns infrastructure re-dispatch and must remain unchanged.
- The repository is at v0.38.6. Only one non-merge commit is visible in the last 14 days, so recent git history does not support a reliable LOC/day estimate.

### Capacity and estimate

- Planned work: about 1,010 changed LOC, including about 570 LOC of implementation and 440 LOC of tests/fixtures.
- Planning rate: about 335 changed LOC/day over 3 days. This is a task decomposition estimate, not an inference from the sparse recent history.
- The approved design estimates about 2 days; this plan reserves a third day for cross-package integration, boundary checks, and failure-path verification.

## Milestone 1: Explicit chain configuration and startup validation

**Goal:** Add opt-in fallback configuration and reject unsafe chains before dispatch.
**Estimated:** 150 implementation + 110 tests = 260 LOC
**Duration:** 5 hours

**Files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- relevant agent-registry validation test file

**Tasks:**

- Add `fallback_models` to `AgentConfig` and append its entries after an explicit pin in `ResolveModelChain`.
- Enforce at most two configured tails, no duplicates, and no tail equal to the head.
- Resolve head and tails through `modelreg`; reject unresolved, cross-harness, or greater-than-3x blended-price tails with agent-specific errors.
- Add the measured killed-process transport signature and pin hard timeout, git, and auth outcomes as terminal.
- Prove empty fallback configuration preserves every existing chain head and behavior.

**Acceptance criteria:**

- An agent configured as head A with tail B resolves exactly `[A, B]`; an agent without tails resolves exactly as before.
- Startup validation rejects count, duplicate, head-repeat, unresolved, harness, and price violations with the agent, entry, and reason in the error.
- Exact September failure strings distinguish idle/killed transport failures from hard-timeout and git terminal failures.
- `MaxFallbackChainTail=2`, `MaxFallbackPriceRatio=3.0`, and `MaxInContainerWalks=1` are named and directly tested.

**Risk:** Model registry names and executor/harness identity may use different representations. Mitigate with table tests built from real registry rows rather than duplicated lookup logic.

## Milestone 2: Dispatch contract and bounded job envelope

**Goal:** Deliver the validated chain to the Cloud Run job and provide enough wall-clock budget for one fallback without changing no-tail dispatch behavior.
**Estimated:** 90 implementation + 90 tests = 180 LOC
**Duration:** 3.5 hours
**Dependencies:** Milestone 1

**Files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- corresponding cloud dispatcher tests

**Tasks:**

- Add `AILANG_MODEL_CHAIN` to the job environment while retaining `AILANG_MODEL` as the head.
- Scale the job timeout to twice the task timeout only when a tail exists, bounded by Cloud Run's 24-hour maximum.
- Keep the stale-task detector, execution count, and chain-head resolution untouched.

**Acceptance criteria:**

- A two-link configuration dispatches the ordered chain and doubles only the enclosing job timeout.
- A no-tail configuration has the same effective model and timeout behavior as the baseline; the new chain variable contains only the pin.
- Timeout scaling is capped and invalid chain configuration cannot reach dispatch.
- No diff is made to `internal/coordinator/stale_task_detector.go`.

**Risk:** Environment serialization could corrupt model names or ordering. Mitigate with round-trip tests using representative OpenRouter model strings.

## Milestone 3: Single-execution walk with isolated attempts

**Goal:** Execute at most one fallback model after a transport-class failure, with fresh workdirs and one final completion.
**Estimated:** 220 implementation + 140 tests = 360 LOC
**Duration:** 7 hours
**Dependencies:** Milestones 1 and 2

**Files to update:**

- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_executor.go` if executor injection needs a narrow seam
- cloud coordinator/executor test files

**Tasks:**

- Parse the chain and wrap `executeCloudTask` in a loop bounded by `MaxInContainerWalks`.
- Call `ClassifyFailure` at the executor-error boundary and walk only on transport class with a remaining link.
- Give each attempt a fresh `attempt{n}` workdir rooted at the same base commit; never continue from partial output.
- Preserve the existing one-completion guard on success, terminal failure, and deferred/error paths.
- Add a deterministic fake executor fixture that fails link 0 with a selected error and records invocation count and workdir.

**Example/test fixture:** the fake executor scenario is the runnable example for this infrastructure feature: 503 then success walks once; git failure and hard timeout run once; repeated transport failure stops after link 1.

**Acceptance criteria:**

- A deterministic 503 on A followed by success on B completes in the same execution with exactly two executor calls and distinct workdirs.
- Git-push and hard-timeout failures invoke only A; an idle/killed transport failure may invoke B.
- A chain with more available links still runs at most two models per execution.
- Partial files from attempt 0 are absent from attempt 1, while attempt history preserves its error and diffstat.
- Every path publishes exactly one final completion.

**Risk:** Existing execution setup may combine cloning and invocation too tightly for injection. Mitigate by extracting only the smallest testable loop/helper and retaining the current production executor path.

## Milestone 4: Completion accounting, notifications, and end-to-end verification

**Goal:** Make fallback use and cost visible in Pub/Sub, banked results, logs, and inbox notifications.
**Estimated:** 110 implementation + 100 tests = 210 LOC
**Duration:** 4.5 hours
**Dependencies:** Milestone 3

**Files to update:**

- `internal/pubsub/topics.go`
- `internal/coordinator/pubsub_completion_handler.go`
- `cmd/ailang/coordinator_cloud.go`
- corresponding Pub/Sub and completion-handler tests
- `docs/docs/guides/coordinator.md` or the current coordinator operations guide, after verifying the canonical path

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, class, bounded error, duration, diffstat, and cost fields supported by the executor result.
- Populate `ModelUsed` and `ChainLinkIndex` on every cloud completion, including failures and the deferred guard.
- Bank the terminal model and expose model/link/attempt history in completion notifications.
- Emit a structured `MODEL_WALK` log for each transition and narrate walked failures without hiding either link.
- Run focused tests, full tests, lint, and architecture boundary checks.

**Acceptance criteria:**

- Success, terminal failure, walked success, and walked failure payloads all identify the terminal model and link.
- A walked payload contains both attempts in order; notifications expose `model_used` and `chain_link_index`.
- Cost/duration fields reflect each attempt without double-counting the terminal result.
- `make test`, `make lint`, and `make check-boundaries` pass.
- The coordinator documentation states that fallback is opt-in, same-harness, one-walk maximum, price-bounded, and observable.

**Risk:** Existing banking may have only one model/cost slot. Mitigate by preserving the terminal summary fields and treating `Attempts` as the auditable per-run breakdown.

## Day-by-day execution

### Day 1 — configuration and dispatch contract

- Complete Milestone 1 with table-driven registry and classifier tests.
- Complete Milestone 2 and verify no-tail back-compat.
- Run focused coordinator tests and lint touched packages.

### Day 2 — execution loop

- Build the minimal executor injection seam and isolated attempt workdirs.
- Complete the deterministic walk/no-walk matrix in Milestone 3.
- Verify the one-completion and one-walk invariants under all terminal paths.

### Day 3 — observability and integration

- Complete Milestone 4 payload, banking, notification, logging, and docs work.
- Run the full verification suite and inspect the final diff for any stale-detector or default-behavior drift.
- Update only milestone `passes` fields in the sprint JSON as acceptance criteria are met.

## Success metrics

- `ClassifyFailure` has a production call site at the executor failure boundary.
- Simulated 503-to-success completes on link 1 without re-dispatch; hard-timeout and git failures remain on link 0.
- Every cloud completion identifies the terminal model/link and provides a non-placeholder attempt history.
- No configured agent changes runtime behavior until `fallback_models` is explicitly present.
- At most 2 executions x 2 model invocations per task is enforced and tested.
- Deterministic fake-executor examples/tests pass; no real provider outage is required.
- Full test, lint, and boundary gates pass.

## Dependencies and assumptions

- The approved design is authoritative; role chains are not made implicit tails.
- Same-harness fallback is the only supported lane in this sprint.
- The existing stale-task detector remains the sole infrastructure re-dispatcher.
- No GitHub issue is linked (`#0` in the handoff means no issue).
- Sprint execution requires the user's separate explicit “execute sprint” approval under repository policy.

## Deferred work

- Cross-harness and local-GPU fallback.
- Quota-aware tail skipping beyond the approved transport classifier.
- More than one in-container walk or changes to `MaxTaskExecutions`.
