# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved, opt-in secondary-model fallback for cloud executor agents. A transport-class failure may make one visible, same-harness fallback attempt inside the existing Cloud Run execution, while hard-timeout, git, authentication, and model-class failures remain terminal.

**Design:** `design_docs/planned/m-secondary-model-fallback.md`  
**Duration:** 3 engineering days (approximately 20 hours)  
**Target:** v1.2.0  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed)  
**Risk Level:** Medium-high — the implementation crosses registry validation, Cloud Run dispatch, executor lifecycle, and completion persistence, and it can change inference cost when explicitly enabled.

## Current Status Analysis

### Completed Foundations

- `ResolveModelChain`, `ClassifyFailure`, the two-execution cap, and their baseline tests already exist.
- `TaskCompletion.ModelUsed` and `TaskCompletion.ChainLinkIndex` already exist, although cloud publishers do not populate them.
- The stale-task detector remains the sole infrastructure re-dispatcher and is intentionally outside this sprint's implementation surface.
- The approved design verified all 37 production agents are pinned and therefore currently resolve to one-link chains.

### Velocity and Capacity

- The checkout is shallow/grafted and exposes only one documentation commit in the last seven days, so a defensible recent LOC/day rate cannot be calculated.
- The approved design estimates about two days. This plan allows three days (about 20 hours), adding integration and regression-test buffer because the change crosses process and Pub/Sub boundaries.
- Estimated change volume is 900 LOC: about 425 implementation/configuration LOC and 475 test/documentation LOC.

### Remaining Work

- Add explicit fallback configuration and loud startup validation.
- Carry a validated chain into the Cloud Run job with a bounded timeout.
- Execute at most one transport-class fallback in isolated attempt workdirs.
- Persist and notify the terminal model/link plus complete attempt history.
- Prove default behavior, cost bounds, and stale-task behavior remain unchanged.

## Milestone 1: Configuration, Resolution, and Startup Validation

**Goal:** Make fallback chains explicit, opt-in, bounded, same-harness, resolvable, and price-limited before any task dispatches.  
**Estimated:** 110 implementation LOC + 150 test LOC = 260 LOC  
**Duration:** Day 1, approximately 6 hours  
**Dependencies:** None

**Example files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/agent_registry_test.go`
- `internal/coordinator/retry_chain_test.go`
- `internal/coordinator/dispatch_provider_test.go` or a focused fallback-validation test file

**Tasks:**

- Add `FallbackModels` beside the pinned `Model` field and append its entries in `ResolveModelChain` without changing the chain head.
- Define `MaxFallbackChainTail=2`, `MaxInContainerWalks=1`, and `MaxFallbackPriceRatio=3.0` in the owning coordinator package.
- Add startup/reload validation for resolvability, executor compatibility, price ratio, duplicates, pin repetition, and tail length.
- Extend the classifier for the measured signal-killed transport failure while pinning hard timeout, git, auth, and unknown errors as terminal.
- Test the unchanged one-link result for every registry agent without `fallback_models`.

**Acceptance criteria:**

- An unconfigured pinned agent still resolves byte-for-byte to `[pin]`.
- A configured agent resolves to `[pin, fallback...]`, with its head matching `ResolveModel`.
- Invalid fallbacks fail loudly with agent ID, entry, and reason.
- Exact September error strings distinguish idle/signal death from hard timeout and git failure.
- Package tests and formatting pass.

**Risk:** Model-registry names, executor variants, and raw wire model names use related but different namespaces. Mitigation: centralize resolution and use table tests containing current production-shaped entries.

## Milestone 2: Cloud Dispatch Contract and Bounds

**Goal:** Deliver the validated chain to the existing job while preserving default dispatch behavior and enforcing the wall-clock ceiling.  
**Estimated:** 75 implementation LOC + 85 test LOC = 160 LOC  
**Duration:** Day 1–2, approximately 3 hours  
**Dependencies:** Milestone 1

**Example files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- `internal/coordinator/cloud_dispatcher_test.go`
- `internal/coordinator/daemon_tasks_exec.go` if the resolved chain must cross the dispatch parameter seam

**Tasks:**

- Populate `AILANG_MODEL_CHAIN` from `ResolveModelChain` alongside the existing `AILANG_MODEL` head.
- Keep the unconfigured agent's environment identical except for a chain variable equal to its pin.
- Double the Cloud Run Job timeout only for multi-link chains, bounded by Cloud Run's 24-hour maximum; keep per-attempt executor timeouts unchanged.
- Add deterministic tests for single-link, multi-link, and maximum-timeout cases.

**Acceptance criteria:**

- Single-link agents receive `AILANG_MODEL_CHAIN=<pin>` and retain their existing effective timeout.
- Multi-link agents receive the ordered chain and a correctly bounded job timeout.
- No role-chain tail is silently attached to a pinned agent.
- Dispatch tests pass without requiring GCP.

**Risk:** String encoding could make a valid model name ambiguous. Mitigation: use one documented parser/serializer pair and round-trip tests rather than independently splitting strings at multiple call sites.

## Milestone 3: Bounded In-Container Walk and Attempt Isolation

**Goal:** Run one fallback attempt for transport failures inside a single execution and publish exactly one terminal completion.  
**Estimated:** 190 implementation LOC + 150 test LOC = 340 LOC  
**Duration:** Day 2, approximately 7 hours  
**Dependencies:** Milestones 1–2

**Example files to update:**

- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_test.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`

**Tasks:**

- Parse and validate the chain env at job startup, failing loudly on malformed or head-mismatched input.
- Extract a testable loop seam around `executeCloudTask` and enforce `MaxInContainerWalks=1` in code.
- Give each model attempt a fresh `attempt{n}` workdir based on the same base commit; never continue from a failed attempt's partial tree.
- Walk only on `FailureTransport`; stop on success, terminal classification, exhausted chain, or budget.
- Preserve the existing atomic single-completion guard across success, failure, panic, and deferred fallback paths.
- Emit structured `MODEL_WALK` output containing task, from/to link/model, and failure class.

**Acceptance criteria:**

- A fake executor returning 503 on A then success on B invokes two models within one task execution.
- Git failure and hard timeout invoke only A.
- Three configured links still cause at most two model invocations per execution.
- Each attempt starts in an isolated workdir; failed partial changes cannot reach the next attempt.
- The completion publisher is called exactly once on every tested path.

**Risk:** `executeCloudTask` currently combines clone, execution, evidence, artifact, and push behavior. Mitigation: introduce the narrowest injectable executor function needed for deterministic tests and avoid broad refactoring.

## Milestone 4: Completion Observability, Persistence, and Regression Gates

**Goal:** Make every fallback decision attributable in the durable completion path and prove global invariants.  
**Estimated:** 50 implementation LOC + 90 test/documentation LOC = 140 LOC  
**Duration:** Day 3, approximately 4 hours  
**Dependencies:** Milestone 3

**Example files to update:**

- `internal/pubsub/topics.go`
- `internal/coordinator/pubsub_completion_handler.go`
- `internal/coordinator/pubsub_completion_handler_test.go`
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_test.go`
- `internal/coordinator/stale_task_detector_test.go`
- `docs/docs/guides/coordinator.md` or the current coordinator operations guide discovered during implementation

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, class, bounded error text, duration, and cost.
- Populate `ModelUsed`, `ChainLinkIndex`, and one-or-more attempt records on successful, failed, walked, and non-walked completions, including defer-guard failures when model context is available.
- Bank the actual terminal model/link in `ExecuteResult` and include both values in the inbox completion notification.
- Narrate walked failures with both links and the first failure class while keeping payload/error sizes bounded.
- Add regression tests for the 2 executions × 2 model runs ceiling and unchanged stale-task re-dispatch behavior.
- Document configuration, visibility, price ceiling, and rollback (remove `fallback_models`).

**Acceptance criteria:**

- Every model execution completion has terminal model/link attribution; walked completions include both attempt records.
- A walked success banks B/link 1 and preserves A's failed attempt metadata.
- A walked failure names both links and the first failure class.
- `AttemptCount` remains 1 for an in-container walk; infrastructure re-dispatch remains capped at 2.
- `make test`, `make lint`, and `make check-boundaries` pass.

**Risk:** Completion schema changes may be consumed by older binaries. Mitigation: add only backward-compatible optional JSON fields and test decoding payloads with and without attempts.

## Day-by-Day Plan

### Day 1

- Complete Milestone 1 with focused coordinator tests.
- Begin Milestone 2 and confirm the single-link environment compatibility assertion.
- Run targeted coordinator and dispatcher packages plus formatting.

### Day 2

- Finish Milestone 2.
- Complete the isolated, bounded walk loop and deterministic fake-executor tests in Milestone 3.
- Run `go test ./cmd/ailang ./internal/coordinator ./internal/pubsub` and relevant race-sensitive tests.

### Day 3

- Complete Milestone 4 persistence, notification, payload, and documentation work.
- Run `make fmt`, `make test`, `make lint`, and `make check-boundaries`.
- Verify the live 37-agent fixture remains single-link by default and record rollout evidence before enabling any fallback in production configuration.

## Success Metrics

- `ClassifyFailure` has a production call site in the execute-job path.
- A deterministic 503 test succeeds on link 1 without re-dispatch.
- Git/auth/hard-timeout tests stop on link 0.
- 100% of model execution completions contain terminal `ModelUsed` and `ChainLinkIndex`.
- At most four model invocations per task across two capped executions, proven by tests.
- All existing agents preserve their chain head and behavior until explicitly configured.
- No new example `.ail` file is required: this is Go coordinator infrastructure with no AILANG syntax or user-language behavior. The runnable example is the deterministic fake-executor integration test.
- Documentation covers opt-in configuration and visible cost/attempt reporting.
- Full tests, lint, formatting, and architecture boundaries pass.

## Dependencies and Rollout

- M-COORDINATOR-EXECUTION-TRUST M3 must remain landed; it supplies the retry and completion foundations.
- Land code with all `fallback_models` lists empty. This produces no runtime fallback behavior.
- Enable one low-volume same-harness agent with one validated fallback and observe completion attempts and cost.
- Expand only after terminal-class non-walks and transport-class walks are visible in production payloads.
- Roll back operationally by removing `fallback_models`; roll back code independently because new payload fields are optional.

## Open Questions

- During implementation, confirm the authoritative coordinator operations guide path before updating documentation; do not create a duplicate guide.
- Confirm whether attempt `error_msg` needs a stricter limit than the existing completion payload cap; choose and test an explicit bound before shipping.

## Definition of Done

- All four milestones pass their acceptance criteria.
- The JSON tracker contains no placeholders and matches the 900 LOC / 3-day plan.
- No production agent is opted in as part of implementation.
- The sprint is evaluated against `design_docs/planned/m-secondary-model-fallback.md` before rollout.

