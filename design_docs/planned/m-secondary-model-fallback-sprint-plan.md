# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved, opt-in secondary-model fallback for cloud executor agents. A transport-class failure may make one same-container retry on an explicitly configured compatible model, while terminal failures remain single-run and every attempt is visible in the completion payload.

**Duration:** 3 engineering days (approximately 20 hours)
**Target:** v1.2.0
**Dependencies:** Approved [M-SECONDARY-MODEL-FALLBACK](m-secondary-model-fallback.md); M-COORDINATOR-EXECUTION-TRUST M3 is already landed
**Risk Level:** Medium
**Estimated change:** 690 LOC (approximately 335 implementation/configuration + 355 tests)

## Current Status Analysis

### Verified baseline

- `AgentConfig` has a single `Model` pin and no explicit fallback list.
- `ResolveModelChain` returns a one-element chain whenever `Model` is set.
- `ClassifyFailure` is implemented but has no production call site.
- `TaskCompletion.ModelUsed` and `TaskCompletion.ChainLinkIndex` exist but the cloud completion publisher does not populate them.
- Cloud execution calls `executeCloudTask` exactly once, using `AILANG_MODEL`.
- The stale-task detector owns infrastructure re-dispatch and remains out of scope.
- Repository version is v0.38.6. The worktree had no pre-existing changes when planning began.

### Velocity and capacity

The last-seven-day checkout contains only one visible documentation commit, so it is not a credible code-velocity sample. The approved design estimates about two days. This plan reserves a third day for integration, negative-path tests, and repository-wide gates: approximately 230 changed LOC/day with low velocity-confidence.

### Baseline limitation

This workspace does not contain the Go toolchain (`go: command not found`), so Go tests and architecture gates could not be executed by the planner. The executor must establish a green baseline before M1 and stop if an in-scope package is already red.

## Milestone 1: Explicit chain configuration and fail-loud validation

**Goal:** Make fallback configuration opt-in, bounded, resolvable, same-harness, and price-limited before any job is dispatched.
**Estimated:** 115 implementation LOC + 125 test LOC = 240 LOC
**Duration:** 0.9 day
**Dependencies:** None

**Files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- Agent-registry validation tests (existing colocated test file, or a focused new `internal/coordinator/agent_registry_fallback_test.go`)

**Tasks:**

1. Add `FallbackModels []string` with `yaml:"fallback_models"` and omitting empty JSON output.
2. Append explicit fallbacks after a pinned head in `ResolveModelChain`; preserve existing role-chain behavior for unpinned agents.
3. Add the fixed bounds `MaxFallbackChainTail=2`, `MaxInContainerWalks=1`, and `MaxFallbackPriceRatio=3.0`.
4. Wire registry startup/reload validation for resolvability, duplicate/head equality, tail length, executor compatibility, and blended price ratio. Errors must name the agent, model, and reason.
5. Add table tests, including the current 37-agent registry producing unchanged chain heads with empty fallback lists.

**Acceptance criteria:**

- An explicit `model: A` plus `fallback_models: [B, C]` resolves to `[A, B, C]` in order.
- Empty fallback configuration preserves current behavior for pinned and role-routed agents.
- More than two fallbacks, duplicates, a repeated head, unknown models, incompatible harnesses, and a price ratio above 3× each fail loudly in tests.
- `go test -count=1 ./internal/coordinator` passes.

**Risk:** Model-registry names may not map one-to-one to wire model strings. Mitigation: use the registry's existing resolution API and add fixtures for both accepted name forms; do not add a silent resolution fallback.

## Milestone 2: Dispatch contract and bounded in-container walk

**Goal:** Carry the validated chain to Cloud Run and perform at most one fresh-workdir retry only after a transport failure.
**Estimated:** 125 implementation LOC + 130 test LOC = 255 LOC
**Duration:** 1.1 days
**Dependencies:** M1

**Files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- The concrete Cloud Run dispatcher/env builder and its tests (located from `DispatchParams`/`AILANG_MODEL` call sites during execution)
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_test.go` or a focused new `cmd/ailang/coordinator_cloud_walk_test.go`
- `internal/coordinator/retry_chain.go` and `internal/coordinator/retry_chain_test.go`

**Tasks:**

1. Extend dispatch parameters with the resolved chain and emit `AILANG_MODEL_CHAIN`; retain `AILANG_MODEL` as the head for compatibility.
2. Double the Cloud Run job timeout only when a fallback exists, capped by the platform maximum; do not change each executor attempt's configured timeout.
3. Add the measured killed-process signature to transport classification. Pin exact September strings proving idle stall and provider errors walk, while hard timeout and git/auth errors terminate.
4. Extract a small testable walk controller around executor invocation. Parse and validate the env chain, cap walks at one, and run each attempt in `attempt{n}` under the task workspace.
5. Preserve the single-completion guard and keep the stale-task detector, `ShouldReDispatch`, and `MaxTaskExecutions` unchanged.

**Acceptance criteria:**

- A deterministic fake executor returning 503 on link 0 and success on link 1 is invoked twice in one execution and never re-dispatched.
- Git-push, auth, scope refusal, and hard-timeout failures invoke only link 0.
- A three-link chain still invokes no more than two models because `MaxInContainerWalks=1` is enforced in code.
- Fresh attempt workdirs prevent link 1 from reading link 0's partial tree.
- A no-fallback dispatch is behaviorally unchanged except for `AILANG_MODEL_CHAIN` containing the head.
- `go test -count=1 ./cmd/ailang ./internal/coordinator` passes.

**Risk:** `executeCloudTask` currently combines clone, execution, git evidence, and push behavior. Mitigation: introduce the narrowest injectable executor seam needed for deterministic tests; do not broaden this sprint into a general cloud-runner refactor.

## Milestone 3: Attempt accounting and observable completion

**Goal:** Attribute every terminal result to its actual model and expose the full bounded attempt history to all completion consumers.
**Estimated:** 70 implementation LOC + 70 test LOC = 140 LOC
**Duration:** 0.6 day
**Dependencies:** M2

**Files to update:**

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- Associated Pub/Sub serialization, completion-handler, and cloud-execution tests

**Tasks:**

1. Add `CompletionAttempt` and `TaskCompletion.Attempts`, including model, link index, failure class, sanitized error, duration, and cost.
2. Populate `ModelUsed`, `ChainLinkIndex`, and a non-empty attempt list on success, failure, walked, non-walked, early terminal, and deferred-guard completion paths.
3. Aggregate top-level cost/duration consistently across attempts while keeping `ModelUsed` tied to the terminal link; document the chosen aggregation in code and tests.
4. Include `model_used` and `chain_link_index` in inbox completion notifications and emit the structured `MODEL_WALK` log line.
5. Format walked errors to name both links and the first failure class without leaking credentials.

**Acceptance criteria:**

- Every cloud completion has the actual terminal model/link and at least one attempt record.
- A walked success has two ordered attempt records and remains visibly distinguishable from a link-0 success.
- A walked failure narrates both links; a terminal link-0 failure has one attempt.
- Completion JSON round-trip and inbox payload tests pin the new fields.
- Attempt costs sum to the top-level cost, preventing invisible fallback spend.
- `go test -count=1 ./internal/pubsub ./internal/coordinator ./cmd/ailang` passes.

**Risk:** Existing consumers may treat omitted/zero link fields ambiguously. Mitigation: populate fields unconditionally in the cloud producer and retain backward-compatible JSON tags.

## Milestone 4: Fleet fixture, documentation, and release gates

**Goal:** Demonstrate opt-in behavior on a representative registry entry, document operator controls, and close with full quality gates.
**Estimated:** 25 configuration/docs LOC + 30 test LOC = 55 LOC
**Duration:** 0.4 day
**Dependencies:** M3

**Files to update/create:**

- The coordinator agent-registry fixture/config selected by the approved operator during execution
- Relevant coordinator operations documentation under `docs/docs/guides/`
- `changelogs/v0.18-current.md`
- No `.ail` examples are required: this is coordinator infrastructure, not language syntax or stdlib behavior.

**Tasks:**

1. Add one reviewed same-harness fallback fixture or test-only registry example; do not silently enable all 37 production agents.
2. Document opt-in configuration, 3× price validation, one-walk/two-execution bounds, observability fields, and terminal failure classes.
3. Record the feature in the current changelog.
4. Run targeted and repository-wide tests, lint, boundaries, and file-size checks.
5. Verify by diff that `internal/coordinator/stale_task_detector.go` is unchanged.

**Acceptance criteria:**

- No production agent receives a fallback without an explicit reviewed configuration change.
- The documented worst case is no more than 2 Cloud Run executions × 2 model invocations.
- `git diff -- internal/coordinator/stale_task_detector.go` is empty.
- `make test`, `make lint`, `make check-boundaries`, and `make check-file-sizes` pass.
- The working tree contains no temporary attempt directories or generated test debris.

## Day-by-day execution

- **Day 1:** Establish green baseline; complete M1 and start dispatch-contract tests in M2.
- **Day 2:** Complete the execution loop and negative-path coverage in M2; implement M3 payload/accounting changes.
- **Day 3:** Finish completion consumers, M4 documentation/configuration, repository-wide gates, and review against all ten design acceptance criteria.

## Success metrics

- `ClassifyFailure` has a production call site.
- A deterministic 503 simulation succeeds on link 1 in the same execution.
- Git/auth/scope/hard-timeout tests prove no second model invocation.
- `ModelUsed`, `ChainLinkIndex`, and `Attempts` are populated for 100% of tested cloud completion paths.
- No fallback is enabled by default; startup refuses incompatible or over-price configurations.
- All repository test, lint, architecture-boundary, and file-size gates pass.

## Dependencies and stop conditions

- The design document is approved; implementation still requires the user/coordinator to approve this sprint plan and explicitly start `sprint-executor`.
- Stop and return to design if same-harness compatibility cannot be determined at startup from authoritative registry data.
- Stop and request an operator decision before modifying the production fleet registry: model choice changes cost and availability policy.
- Stop if preserving one-completion semantics would require changing stale-task detector ownership or adding a second dispatcher.

## Planning assumptions

- The concrete Cloud Run dispatcher already has a single env-construction seam adjacent to `AILANG_MODEL`; the executor should confirm its exact file before editing.
- Attempt cost and duration are aggregated at the top level; the terminal model remains `ModelUsed`.
- No AILANG source is changed, so `ailang prompt` is not required for this sprint.
- Estimates include tests and documentation but exclude rollout observation during a real provider outage.
