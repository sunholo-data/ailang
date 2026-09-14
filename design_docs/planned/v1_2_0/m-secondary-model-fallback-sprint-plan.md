# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Wire the existing failure classifier and chain resolver into cloud execution so an opted-in agent can retry one transport failure on a validated, same-harness fallback model within the same Cloud Run execution. Preserve the stale-task detector as the sole re-dispatcher and make every attempt and cost-affecting fallback visible in completion data.

**Duration:** 2 focused engineering days (approximately 14 hours)  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design in `design_docs/planned/m-secondary-model-fallback.md`  
**Risk Level:** High — cloud execution, timeout, cost, and completion-accounting paths are coupled

## Current Status Analysis

### Completed Recently

- `ClassifyFailure`, `ResolveModelChain`, execution caps, and completion fields already exist from M-COORDINATOR-EXECUTION-TRUST.
- The current branch contains the stage-boundary coordinator regression fix (`ae4422b8`), providing relevant handoff coverage.

### Velocity and Capacity

- The seven-day history contains one focused coordinator commit and no reliable LOC/day metric; the velocity script reports no usable recent LOC data.
- Capacity is therefore derived conservatively from the approved design's two-day estimate, with integration work split into independently testable milestones and a final verification buffer.
- Estimated change: about 860 LOC including tests. This is an estimate for sizing, not a target.

### Remaining from the Design

- Add explicit per-agent fallback configuration and loud registry validation.
- Send the validated chain and sufficient timeout headroom to the cloud job.
- Execute at most one in-container model walk for transport-class failures.
- Persist terminal-link identity and per-attempt history through completions and notifications.
- Prove default behavior, cost ceilings, terminal failure classes, and stale-task behavior remain unchanged.

## Proposed Milestones

### M1: Registry Schema, Chain Resolution, and Validation

**Goal:** Make fallback configuration explicit, opt-in, bounded, and invalid at startup when unsafe.  
**Estimated:** 120 LOC implementation + 150 LOC tests = 270 LOC  
**Duration:** 3 hours

**Files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- Corresponding `internal/coordinator/*_test.go` files

**Tasks:**

- Add `fallback_models` to `AgentConfig`; append it only after an explicit model pin.
- Add the measured killed-process transport signature while pinning hard timeout and git/auth errors as terminal.
- Validate a maximum tail of two, no duplicates/head repetition, model resolution, same harness, and a maximum 3× blended-price ratio.
- Ensure validation runs at startup and registry reload and names the agent, entry, and reason on failure.

**Acceptance Criteria:**

- Empty fallback lists preserve current chain heads and behavior for all live registry entries.
- Invalid count, duplicate, unresolved, harness-incompatible, and over-price entries fail loudly.
- Exact September failure strings classify as specified by the design.

### M2: Dispatch Chain and Timeout Plumbing

**Goal:** Deliver the validated chain to the existing job without introducing another dispatch path.  
**Estimated:** 70 LOC implementation + 80 LOC tests = 150 LOC  
**Duration:** 2 hours

**Files to update:**

- `internal/coordinator/cloud_dispatcher.go`
- Cloud dispatcher tests and job-spec fixtures

**Tasks:**

- Set `AILANG_MODEL_CHAIN` to the comma-joined resolved chain alongside the current head model.
- Double the job timeout only for multi-link chains, bounded by Cloud Run's 24-hour maximum; keep each attempt's configured timeout unchanged.
- Assert a single-link agent's job configuration is unchanged except for the chain variable equaling the pin.

**Acceptance Criteria:**

- Opted-in jobs receive an ordered, validated chain and adequate wall-clock headroom.
- Non-opted-in jobs remain behaviorally identical.
- No new task state, dispatcher, or re-dispatch condition is introduced.

### M3: Bounded In-Container Walk and Completion Evidence

**Goal:** Retry exactly once on a transport-class executor failure and publish one attributable completion.  
**Estimated:** 180 LOC implementation + 160 LOC tests = 340 LOC  
**Duration:** 5 hours

**Files to update:**

- `cmd/ailang/coordinator_cloud.go`
- `internal/pubsub/topics.go`
- Cloud execution and Pub/Sub serialization tests

**Tasks:**

- Parse the chain and wrap `executeCloudTask` in a loop bounded by `MaxInContainerWalks = 1`.
- Use a fresh `attempt{n}` worktree for each link and retain failure, duration, diffstat, and available cost evidence.
- Add `CompletionAttempt` history and populate `ModelUsed` and `ChainLinkIndex` on every success, failure, and deferred-guard completion path.
- Emit structured `MODEL_WALK` output and a narrated terminal error without publishing intermediate completions.

**Acceptance Criteria:**

- A fake 503 on A then success on B yields one completion, `ModelUsed=B`, `ChainLinkIndex=1`, two attempt records, and no re-dispatch.
- Git-push and hard-timeout failures invoke only A and terminate at link 0.
- Even a three-link configured chain invokes at most two models per execution.

### M4: Banking, Notifications, and System Verification

**Goal:** Make fallback use visible to downstream consumers and prove coordinator invariants did not regress.  
**Estimated:** 50 LOC implementation + 50 LOC tests = 100 LOC  
**Duration:** 4 hours including repository-wide verification

**Files to update:**

- `internal/coordinator/pubsub_completion_handler.go`
- Completion handler and stale-task detector regression tests
- `docs/` only if operator-facing configuration is not already fully described by generated/config documentation

**Tasks:**

- Bank the terminal model identity and include model/link fields in inbox completion notifications.
- Add end-to-end fake-executor coverage for walked success and walked failure.
- Prove the stale-task detector, `ShouldReDispatch`, and `MaxTaskExecutions` behavior is unchanged.
- Run focused tests, then `make test`, `make lint`, and `make check-boundaries`.

**Acceptance Criteria:**

- All completion variants carry terminal model/link identity; walked variants carry full attempt history.
- The deterministic outage simulation passes without a live provider.
- Worst case remains two executions × two model invocations.
- Repository test, lint, and boundary gates pass.

## Day-by-Day Plan

### Day 1

- Complete M1 using table-driven validation and classifier tests first.
- Complete M2 and verify generated job specs for single- and two-link agents.
- Begin M3 with fake-executor seams and the successful A→B walk.

### Day 2

- Finish all M3 terminal/deferred completion paths and attempt evidence.
- Complete M4 banking, notification, and unchanged-stale-detector coverage.
- Run full verification and resolve only regressions caused by this sprint.

## Success Metrics

- `ClassifyFailure` gains a production call site.
- 100% of cloud completion paths populate `ModelUsed` and `ChainLinkIndex`.
- A deterministic 503 simulation succeeds on link 1 with one published completion.
- Exact git/auth and hard-timeout cases remain terminal on link 0.
- All 37 currently non-opted-in agents retain their existing dispatch behavior.
- Focused tests plus `make test`, `make lint`, and `make check-boundaries` pass.

## Dependencies and Risks

- Model-registry price/harness lookup APIs may not expose the required normalized data at the intended layer. Mitigation: extend the existing registry boundary narrowly; do not duplicate model metadata or silently skip validation.
- Completion publication has deferred/error branches. Mitigation: centralize completion construction and test every exit class.
- Fresh per-attempt workdirs can expose cleanup or credential assumptions. Mitigation: use the current per-task setup helper with an explicit attempt suffix and retain failed-attempt evidence until publication.
- Timeout multiplication could exceed platform limits. Mitigation: validate and cap before creating the job spec.

## Scope Boundaries

- No cross-harness or local-GPU fallback.
- No activation of role chains for pinned agents.
- No new coordinator task state, CAS protocol, or dispatcher.
- No changes to `MaxTaskExecutions`, stale-task eligibility, or the V23 sole-re-dispatcher invariant.
- No fleet configuration rollout in this sprint; fallback remains opt-in until separately approved.

## Approval and Handoff

This plan and its JSON progress artifact are ready for human review. Per repository routing, implementation must begin only after the user explicitly says **execute sprint**; that approval triggers the `sprint-executor` workflow.
