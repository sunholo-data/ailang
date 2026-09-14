# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved in-container model-chain walk so a transport-class failure on a pinned cloud model can make one bounded attempt on an explicitly configured, validated fallback model. The sprint preserves the stale-task detector as the sole re-dispatcher and makes every fallback visible in completion telemetry.

**Design:** `design_docs/planned/m-secondary-model-fallback.md`

**Target:** v1.2.0

**Duration:** 3 engineering days (about 18 focused hours)

**Estimated change:** 1,200 LOC (implementation, tests, and documentation)

**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed)

**Risk level:** High — this changes cloud execution, cost bounds, model attribution, and completion reporting.

## Current Status Analysis

### Already available

- `ClassifyFailure`, `ResolveModelChain`, `MaxTaskExecutions`, and the stale-task re-dispatch tier exist in `internal/coordinator/retry_chain.go`.
- `TaskCompletion.ModelUsed` and `TaskCompletion.ChainLinkIndex` exist but are not populated by the cloud job.
- Cloud dispatch already passes the pinned `AILANG_MODEL`, and the execute-job path already has a single-completion guard.
- The design measured all 37 configured cloud agents and specifies an opt-in default: agents without `fallback_models` retain their current behavior.

### Remaining work

- Add and validate explicit per-agent fallback configuration.
- Propagate the resolved chain and expanded job timeout into Cloud Run.
- Execute a bounded, isolated second model attempt only for transport failures.
- Persist model/chain attribution and per-attempt history through completion handling.
- Pin unchanged behavior for hard timeouts, git/auth failures, unconfigured agents, and stale-task re-dispatch.

### Velocity and estimate basis

The checkout is a grafted shallow clone with one visible commit, so a defensible recent LOC/day figure cannot be calculated from local git history. The design estimates about two days. This plan reserves a third day for cross-package integration, regression tests, documentation, and the full repository gates. Capacity is therefore 400 changed LOC/day across implementation and tests, not a claim about historical team velocity.

## Milestone 1: Registry Schema and Fail-Loud Validation

**Goal:** Make fallback chains explicit, opt-in, and impossible to configure with a known-broken model lane.

**Estimated:** 140 implementation LOC + 210 test LOC + 30 documentation/example LOC = 380 LOC

**Duration:** Day 1 (6 hours)

**Dependencies:** Existing model registry resolution and pricing metadata.

### Files to update

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- `internal/coordinator/agent_registry_test.go`
- `internal/coordinator/daemon_tasks_init.go` if startup wiring is not already centralized in registry validation
- `docs/internal/cloud-coordinator-config.md` (configuration example using `fallback_models`)

### Tasks

- Add `FallbackModels []string` next to the pinned model field and append it only after an explicit pin in `ResolveModelChain`.
- Define the chain-tail, in-container-walk, and price-ratio constants specified by the design.
- Add `signal: killed` to the transport signatures while pinning hard timeout and git/auth outcomes as terminal.
- Validate count, duplicates, equality with the pin, model resolution, same-harness compatibility, and the 3× blended-price ceiling at startup and registry reload.
- Add a live-registry compatibility test proving all currently unconfigured agents retain the same chain head and behavior.
- Document one valid and representative invalid configuration in `docs/internal/cloud-coordinator-config.md`; validate the example through registry tests rather than adding a non-runnable `.ail` example.

### Acceptance criteria

- [ ] A pin plus one fallback resolves to `[pin, fallback]`; a pin without fallbacks remains a chain of one.
- [ ] Exact September error strings classify idle mid-generation and signal-killed as transport, but hard timeout and git-push failure as terminal.
- [ ] Startup errors name the agent, offending fallback, and reason for unresolvable, harness-incompatible, duplicate, overlong, or over-price chains.
- [ ] The 37-agent registry remains behaviorally unchanged while no `fallback_models` entries are configured.
- [ ] Focused coordinator registry and retry-chain tests pass.

### Risks

- Model identifiers have friendly-name and wire-name forms. Mitigation: reuse modelreg lookup rules and table-test both accepted forms and loud failures.
- Harness identity can be inferred through multiple fields. Mitigation: call the existing canonical provider/executor resolution rather than duplicating inference.

## Milestone 2: Chain Propagation and Bounded Walk Runtime

**Goal:** Run at most one clean fallback attempt inside the same Cloud Run execution when and only when the head fails transport-class.

**Estimated:** 190 implementation LOC + 210 test LOC = 400 LOC

**Duration:** Day 2 (7 hours)

**Dependencies:** Milestone 1 chain resolution and bounds.

### Files to update

- `internal/coordinator/cloud_dispatcher.go`
- `internal/coordinator/cloud_dispatcher_test.go` or the nearest existing dispatcher test file
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_executor.go`
- `cmd/ailang/coordinator_cloud_test.go` or a focused new `cmd/ailang/coordinator_cloud_walk_test.go`

### Tasks

- Pass comma-joined `AILANG_MODEL_CHAIN` alongside the unchanged `AILANG_MODEL` head.
- Double the Cloud Run job timeout only for multi-link chains, capped at the platform maximum; keep each model attempt's own timeout unchanged.
- Extract a testable bounded-walk helper around cloud execution and inject a fake executor/attempt function for deterministic tests.
- Give each attempt an isolated `attempt{n}` work directory based on the same base commit.
- Walk once only when `ClassifyFailure` returns transport and another link exists; stop immediately for success, terminal errors, an exhausted chain, or the walk budget.
- Keep `completionSent` authoritative so every execution publishes exactly one terminal completion.

### Acceptance criteria

- [ ] A deterministic fake 503 on A invokes B once and succeeds in the same execution without re-dispatch.
- [ ] Git-push and hard-timeout failures invoke only A, proven by the fake executor call count.
- [ ] A three-link chain can execute at most two model runs because `MaxInContainerWalks=1` is enforced in code.
- [ ] Attempt 2 starts in a clean work directory and cannot observe attempt 1's partial files.
- [ ] Single-link dispatch env and timeout behavior remain unchanged except for `AILANG_MODEL_CHAIN=A`.

### Risks

- `executeCloudTask` currently combines clone, execution, commit, and push behavior. Mitigation: introduce the smallest injectable boundary needed for deterministic tests and avoid broad executor refactoring.
- A deferred failure guard could report the wrong link. Mitigation: table-test early errors and the deferred guard as well as normal returns.

## Milestone 3: Completion Attribution, Attempt History, and System Regression

**Goal:** Make walked and non-walked outcomes fully attributable, cost-visible, and compatible with existing completion consumers.

**Estimated:** 150 implementation LOC + 220 test LOC + 50 documentation LOC = 420 LOC

**Duration:** Day 3 (5 hours)

**Dependencies:** Milestone 2 runtime results.

### Files to update

- `internal/pubsub/topics.go`
- `internal/pubsub/topics_test.go` or the nearest payload serialization test
- `internal/coordinator/pubsub_completion_handler.go`
- `internal/coordinator/pubsub_completion_handler_test.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/stale_task_detector_test.go`
- `docs/internal/cloud-coordinator-config.md`

### Tasks

- Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, failure class, error, duration, and cost.
- Populate `ModelUsed`, `ChainLinkIndex`, and attempt history on success, terminal failure, walked success/failure, and deferred-guard completion.
- Narrate walked failures with both links and the first failure class; emit the structured `MODEL_WALK` log record.
- Carry terminal model/link fields into the banked `ExecuteResult` and inbox completion notification.
- Prove the stale-task detector and `MaxTaskExecutions=2` behavior remain unchanged; combine that bound with the one-walk bound to assert at most four model invocations across two executions.
- Run focused tests, `make test`, `make lint`, and `make check-boundaries`.

### Acceptance criteria

- [ ] Every cloud completion path carries explicit terminal `ModelUsed` and `ChainLinkIndex`, including link 0.
- [ ] A walked completion serializes two ordered attempt records and retains the first attempt's error and cost.
- [ ] The completion handler banks the terminal model/link and forwards them to the agent inbox.
- [ ] A walked failure message names both models and the transport classification.
- [ ] The stale-task detector remains the sole re-dispatcher and the 2 executions × 2 runs ceiling is tested.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.

### Risks

- `omitempty` on integer link zero can hide required attribution. Mitigation: add payload-shape tests that require link 0 to be observable, adjusting the representation/tag if necessary.
- Attempt cost may be unavailable on early failures. Mitigation: record zero as unavailable without fabricating cost, while preserving model, class, error, and duration.

## Day-by-Day Execution Plan

### Day 1 — Configuration boundary

- Implement schema, chain resolution, failure signature changes, and validation.
- Write validation/classifier/back-compat tests first, then document the supported YAML configuration.
- Gate: focused coordinator tests and `go test ./internal/coordinator/...` pass.

### Day 2 — Execution boundary

- Implement dispatcher env/timeout changes and the isolated bounded walk.
- Build deterministic fake-executor coverage for success, transport walk, terminal failure, exhausted fallback, and deferred completion.
- Gate: `go test ./cmd/ailang ./internal/coordinator/...` passes.

### Day 3 — Observability and repository gates

- Add attempt payloads and completion-handler propagation.
- Add spend-bound and unchanged stale-detector regressions.
- Run formatting, full tests, lint, and architecture boundaries; update internal configuration docs with verified field names and observability behavior.

## Success Metrics

- `ClassifyFailure` gains a production call site.
- A deterministic 503 test completes on link 1 with terminal model B and link index 1.
- Hard-timeout and git/auth tests remain on link 0 with one model invocation.
- 100% of cloud completion shapes contain terminal model/link attribution.
- Maximum model invocations are mechanically bounded at four across the existing two-execution ceiling.
- The documented `fallback_models` example is parsed and validated by tests.
- Full repository tests, lint, and boundary checks pass.

## Dependencies and Sequencing

1. Milestone 1 must land before dispatcher propagation because it defines the trusted validated chain.
2. Milestone 2 must expose structured attempt outcomes before Milestone 3 can serialize them.
3. No production agent should receive `fallback_models` in this sprint; rollout is a separate, explicit operator change after implementation approval.

## Explicit Non-Goals

- Cross-harness or local-GPU fallback.
- Activating role chains as tails for pinned agents.
- New task states, CAS transitions, attempt IDs, or a second re-dispatcher.
- Raising `MaxTaskExecutions` or walking after model-quality, git, auth, or hard-timeout failures.

## Approval and Handoff

This plan creates no authorization to implement. After human approval, invoke `sprint-executor` with this plan and `.ailang/state/sprints/sprint_M-SECONDARY-MODEL-FALLBACK.json`; execution should use TDD and update only the progress fields allowed by the sprint-state contract.
