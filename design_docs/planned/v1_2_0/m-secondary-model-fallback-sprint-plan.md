# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement an explicit, bounded, same-harness fallback model chain for cloud executor agents. Transport failures may make one visible in-container walk to a configured secondary model, while hard-timeout, git, authentication, and model-result failures remain terminal.

**Duration:** 3 working days (approximately 20–24 engineering hours)

**Dependencies:** Approved `design_docs/planned/m-secondary-model-fallback.md`; M-COORDINATOR-EXECUTION-TRUST M3 (landed)

**Risk Level:** High — this changes fleet-wide execution, cost accounting, timeout behavior, and completion semantics, though behavior remains opt-in.

## Current Status Analysis

### Completed Recently

- The execution-trust work already introduced `ClassifyFailure`, `ResolveModelChain`, `MaxTaskExecutions`, and completion fields for `ModelUsed`/`ChainLinkIndex`.
- The approved design verified the production fleet and all affected paths against HEAD on 2026-09-14.
- Existing retry-chain tests cover role chains, chain heads, and the safe terminal default, providing a base for extension.

### Velocity and Capacity

- The checkout's 14-day history is shallow and contains only one documentation commit, so it does not support a credible LOC/day calculation.
- The design estimate is approximately 2 days. This plan reserves 1 additional day for cross-package integration, boundary checks, and deterministic failure-path tests.
- Planned capacity is approximately 1,050 changed LOC, including roughly 580 implementation/configuration LOC and 470 test LOC.

### Remaining from the Design Doc

- Add explicit per-agent fallback configuration and loud startup/reload validation.
- Carry the resolved chain into Cloud Run and scale the job timeout only for configured chains.
- Wire failure classification into one bounded in-container execution loop with isolated attempt workdirs.
- Populate terminal-link and per-attempt observability through completion handling and banking.
- Prove cost bounds, back compatibility, terminal failure boundaries, and the unchanged stale-task re-dispatch tier.

## Proposed Milestones

### M1: Configuration, chain resolution, and validation (~300 LOC)

**Goal:** Make fallback intent explicit and reject unsafe chains before dispatch.

**Estimated:** 140 LOC implementation + 160 LOC tests = 300 LOC

**Duration:** Day 1 morning

**Dependencies:** None

**Files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- Registry validation tests adjacent to the existing agent-registry tests

**Tasks:**

- Add `FallbackModels []string` to `AgentConfig` and make `ResolveModelChain` return the pin followed by the explicit tail.
- Add the two-entry tail limit, one-walk limit, price-ratio constant, duplicate/head checks, registry resolution, and same-harness compatibility validation.
- Invoke validation at both startup and registry reload, using errors that name the agent, fallback entry, and violated rule.
- Extend the classifier table with the exact measured killed-process, hard-timeout, idle-stall, and git failure strings.

**Acceptance Criteria:**

- [ ] An empty `fallback_models` field preserves the current single-link chain and chain head.
- [ ] Unresolvable, harness-incompatible, duplicate, head-equal, over-count, and over-price entries fail validation loudly.
- [ ] `pi idle for 3m mid-generation` and `signal: killed` are transport-class; hard timeout and git/auth failures are terminal.
- [ ] Validation and chain-resolution unit tests pass and lint is clean.

**Risks:** Model registry names and raw harness wire strings can be confused. Mitigation: resolve both the pin and each tail through the registry in tests using representative pi and codex rows.

### M2: Dispatch contract and bounded execution loop (~330 LOC)

**Goal:** Execute at most one fallback model run inside the same Cloud Run execution after a transport-class failure.

**Estimated:** 170 LOC implementation + 160 LOC tests = 330 LOC

**Duration:** Day 1 afternoon through Day 2 morning

**Dependencies:** M1

**Files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- Dispatcher tests adjacent to `cloud_dispatcher.go`
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_test.go` or a focused new `cmd/ailang/coordinator_cloud_fallback_test.go`

**Tasks:**

- Emit `AILANG_MODEL_CHAIN` alongside the existing head model and double the Cloud Run job timeout only for a multi-link chain, capped at the platform maximum.
- Parse the chain in the job binary and wrap `executeCloudTask` in a loop capped by `MaxInContainerWalks=1`.
- Parameterize the workdir with `attempt{n}` so a failed attempt's partial tree cannot contaminate the fallback attempt.
- Introduce a narrow executor seam/test double that deterministically fails link 0 and records invocation count; avoid live provider calls.
- Preserve the existing single-completion guard and ensure infrastructure re-dispatch always restarts at chain head.

**Acceptance Criteria:**

- [ ] A simulated 503 on model A invokes model B once and succeeds within the same task execution.
- [ ] A git failure or hard timeout invokes only model A and terminates.
- [ ] A three-link configured chain still runs at most two model invocations per execution.
- [ ] Single-link agents retain the current timeout and execution behavior.
- [ ] The stale-task detector and `MaxTaskExecutions=2` behavior remain unchanged and covered by existing/new regression tests.

**Risks:** The current execution function owns cloning, branch creation, and final evidence collection, so looping it may expose assumptions about one workdir per task. Mitigation: isolate only the attempt workspace while keeping task-level completion and branch state outside the loop, and exercise both failed-first and first-link-success paths.

### M3: Completion observability and banking (~250 LOC)

**Goal:** Make every attempted model, walk decision, terminal link, duration, error class, and known cost visible to operators and downstream consumers.

**Estimated:** 130 LOC implementation + 120 LOC tests = 250 LOC

**Duration:** Day 2 afternoon

**Dependencies:** M2

**Files to update:**

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- Completion payload/handler tests adjacent to those packages

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts`, with model, link index, failure class, error, duration, and cost fields.
- Populate `ModelUsed`, `ChainLinkIndex`, and `Attempts` on success, failure, panic/deferred completion, walked, and non-walked paths.
- Bank the terminal model/link and include them in agent inbox completion notifications.
- Add structured `MODEL_WALK` stdout records and narrate both links in terminal walked errors without relying on logs as the source of truth.

**Acceptance Criteria:**

- [ ] Every cloud completion carries the terminal `ModelUsed` and `ChainLinkIndex`, including deferred failure paths.
- [ ] A walked success carries two attempt records and identifies link 1 as terminal.
- [ ] A walked failure names both models and the first failure class in its payload.
- [ ] Completion handler tests prove the model/link fields survive publishing, banking, and inbox notification.

**Risks:** `omitempty` on zero-valued link indexes can hide link 0 in serialized JSON. Mitigation: add wire-format tests for link 0 and link 1 and use field types/tags that preserve the required distinction.

### M4: Fleet fixture, executable scenarios, and quality gates (~170 LOC)

**Goal:** Demonstrate opt-in compatibility and close the sprint with repository-wide checks.

**Estimated:** 40 LOC configuration/documentation + 130 LOC tests = 170 LOC

**Duration:** Day 3

**Dependencies:** M1, M2, M3

**Files to update/create:**

- The checked-in coordinator agent registry fixture/config used by cloud agents, adding one clearly documented test/canary fallback only if an existing non-production fixture is available
- Focused integration fixtures under the relevant coordinator test package
- `design_docs/planned/m-secondary-model-fallback.md` only if implementation discoveries require an approved-design clarification

**Tasks:**

- Run a deterministic end-to-end fixture: A returns 503, B succeeds, exactly one completion is emitted, and the attempt history is complete.
- Run terminal fixtures for hard timeout and git/auth errors and a ceiling fixture for 2 executions × 2 model runs.
- Compare the existing 37-agent registry resolution before/after: no configured fallback means identical chain head and execution behavior; do not silently enable fleet fallback in this sprint.
- Run formatting, focused tests, full tests, lint, and architecture boundaries.

**Acceptance Criteria:**

- [ ] Deterministic fallback and terminal-path fixtures pass without network/provider access.
- [ ] Existing agents without `fallback_models` show no behavioral change.
- [ ] Worst-case model invocations are mechanically bounded at four per task across two executions.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.
- [ ] No `.ail` example is required because this is coordinator infrastructure; the deterministic integration fixture is the executable example and is verified working.

**Risks:** Full repository checks may expose unrelated concurrent failures. Mitigation: record focused-test evidence separately, identify unrelated failures precisely, and do not weaken gates or silently ignore failures.

## Day-by-Day Plan

### Day 1 — Safe configuration and dispatch contract

- Complete M1 with table-driven validation/classification tests first.
- Implement chain env propagation and timeout scaling from M2.
- Establish the fake executor seam and verify a first-link success remains single-run.

### Day 2 — Walk loop and observability

- Complete the bounded walk and attempt-isolation behavior in M2.
- Implement M3 payloads, structured walk record, banking, and notification propagation.
- Run focused race/error-path tests, including deferred completion.

### Day 3 — Integration, compatibility, and gates

- Complete M4 deterministic scenarios and fleet back-compat comparison.
- Run `make fmt`, focused package tests, `make test`, `make lint`, and `make check-boundaries`.
- Reconcile implementation discoveries with the approved design and update progress evidence.

## Success Metrics

- `ClassifyFailure` has a production call site in the cloud execution job.
- Simulated transport failure completes on link 1 with one task execution and two model invocations.
- Hard-timeout, git, credentials, auth, and scope/model failures never walk.
- `ModelUsed`, `ChainLinkIndex`, and attempt history are present and tested for all cloud completion paths.
- No existing agent changes behavior without explicit `fallback_models` configuration.
- Maximum spend is structurally bounded to two model invocations per execution and four per task.
- Deterministic integration fixtures replace any need for a live outage or an AILANG-language example.
- Full tests, lint, formatting, and architecture-boundary checks pass.

## Dependencies and Assumptions

- The approved design is authoritative; semantic changes require returning through design approval.
- Registry pricing and harness metadata are complete enough to validate both the pin and fallback entries. Missing metadata must fail loudly, not bypass validation.
- The executor can accept a per-attempt model without mutating process-global state; if it currently reads only `AILANG_MODEL`, pass the model explicitly through the execution seam.
- The plan does not add fallback entries to the production fleet. Operator rollout and canary selection are a separately approved configuration action.

## Deferred Work

- Cross-harness fallback and local GPU-lane fallback.
- Making role chains implicit tails for explicitly pinned agents.
- Quota-aware skipping beyond the approved transport classifier.
- Production rollout policy, canary agent selection, and fleet-wide fallback configuration.
