# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved, opt-in secondary-model fallback for cloud executor agents. A transport failure may trigger one isolated in-container retry, while model, timeout, Git, credential, and infrastructure failures retain their designed terminal or re-dispatch behavior.

**Design:** `design_docs/planned/m-secondary-model-fallback.md`  
**Target:** v1.2.0  
**Duration:** 3 working days (approximately 21 hours)  
**Estimated change:** 820 LOC (implementation, tests, and documentation)  
**Risk:** High — the work crosses registry validation, Cloud Run dispatch, executor lifecycle, completion accounting, and persistent observability.

## Current Status and Planning Basis

- `ResolveModelChain` and `ClassifyFailure` exist, but pinned agents resolve to one model and production execution does not call the classifier.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated; per-attempt history does not exist.
- The stale-task detector remains the sole infrastructure re-dispatcher and is explicitly outside the implementation scope.
- The checkout is shallow and contains one recent commit, so it provides no defensible LOC/day history. The design's two-day estimate is extended to three days for cross-package integration and full repository gates.
- Capacity target is approximately 273 LOC/day, including tests and docs.

## Milestones

### M1 — Explicit chain configuration and fail-loud validation

**Estimate:** 210 LOC; 5.25 hours  
**Files:** `internal/coordinator/agent_registry.go`, `internal/coordinator/retry_chain.go`, their focused test files, and representative agent registry fixtures.

Tasks:

- Add `fallback_models` to `AgentConfig` and append its entries after an explicit pin in `ResolveModelChain`.
- Enforce at most two tail entries, no duplicates/head repetition, model-registry resolution, matching harness, and the 3× blended-price ceiling.
- Add the measured killed-process transport signature while pinning hard timeout and Git/auth failures as terminal.

Acceptance criteria:

- Empty `fallback_models` preserves the current one-link behavior.
- Invalid count, duplicates, unresolved models, harness mismatch, or excessive price fail startup/reload with agent and model named.
- Classifier tables cover 429/503, mid-generation death, killed process, hard timeout, and Git/auth errors.
- Chain-head compatibility tests remain green.

### M2 — Bounded isolated in-container walk

**Estimate:** 290 LOC; 7 hours  
**Dependencies:** M1  
**Files:** `internal/coordinator/cloud_dispatcher.go`, `cmd/ailang/coordinator_cloud.go`, and focused dispatcher/executor tests.

Tasks:

- Pass the resolved chain through `AILANG_MODEL_CHAIN`; scale the Cloud Run job timeout only when a fallback is configured.
- Extract a testable walk helper around `executeCloudTask` and permit at most one fallback attempt on `FailureTransport`.
- Give every attempt a fresh `attempt{n}` worktree from the same base commit; discard partial work while retaining its evidence.
- Preserve one execution/one completion and leave stale-task re-dispatch logic unchanged.

Acceptance criteria:

- Fake-executor tests prove transport-fail→fallback-success, transport-fail→fallback-fail, and all terminal no-walk cases.
- No task performs more than two model runs per execution, even if a three-link chain is configured.
- Each attempt starts from an isolated clean worktree at the same base commit.
- Existing completion guard still publishes exactly once; stale-task detector tests remain unchanged and green.

### M3 — Attempt accounting, banking, and notifications

**Estimate:** 210 LOC; 5.25 hours  
**Dependencies:** M2  
**Files:** `internal/pubsub/topics.go`, `cmd/ailang/coordinator_cloud.go`, `internal/coordinator/pubsub_completion_handler.go`, and their tests.

Tasks:

- Add `CompletionAttempt` history with model, link, failure class, error, duration, and cost.
- Populate `ModelUsed`, `ChainLinkIndex`, and attempts on success, failure, and deferred completion paths.
- Bank the terminal model identity and expose model/link fields in agent inbox notifications.
- Emit a structured `MODEL_WALK` line and narrate walked failures without hiding the first attempt.

Acceptance criteria:

- Zero-walk completions contain one attempt; walked completions contain two ordered attempts.
- Success and failure payloads identify the terminal model/link, including deferred-guard publication.
- Banked results and inbox notifications preserve `model_used` and `chain_link_index` end to end.
- Payload round-trip and backward-compatibility tests pass for completions without the new field.

### M4 — Registry proof, documentation, and repository gates

**Estimate:** 110 LOC; 3.5 hours  
**Dependencies:** M1–M3  
**Files:** one representative agent registry entry/fixture, coordinator documentation, changelog, and the design document status after all gates pass.

Tasks:

- Add one representative opt-in configuration and prove unconfigured agents remain unchanged.
- Document bounds: one walk per execution, two executions maximum, four model invocations worst case, price ceiling, and visible fallback accounting.
- Run focused tests, full tests, formatting, lint, architecture boundaries, and example verification.

Acceptance criteria:

- Representative configured agent resolves and validates; all unconfigured agents retain one-link chains.
- `go test ./internal/coordinator ./internal/pubsub ./cmd/ailang` passes.
- `make fmt`, `make test`, `make lint`, `make check-boundaries`, and `make verify-examples` pass, or any known unrelated failure is recorded with reproducible evidence.
- Design status and changelog are updated only after implementation and verification succeed.

## Day-by-Day Plan

### Day 1 — Configuration boundary

- Establish focused red tests for chain resolution, startup validation, and classifier behavior.
- Implement M1 and run coordinator tests.
- Start dispatcher env/timeout tests for M2.

### Day 2 — Execution and observability

- Implement the isolated bounded walk and its fake-executor matrix.
- Add completion-attempt schema and populate every publication path.
- Verify single-completion and unchanged stale-task behavior.

### Day 3 — Persistence and integration

- Complete banking, inbox notification, structured logs, and compatibility tests.
- Add representative opt-in config and operational documentation.
- Run repository-wide gates, fix in-scope failures, and record final evidence.

## Success Metrics

- A configured transport failure advances exactly once and can complete on link 1 without re-dispatch.
- Hard timeout, model-quality, Git/auth, and unknown failures never walk.
- Every completion identifies its terminal model/link and retains ordered per-attempt evidence.
- Default blast radius is zero: agents without `fallback_models` behave byte-for-byte equivalently at the configuration boundary.
- All focused and repository quality gates pass.

## Dependencies and Risks

- M-COORDINATOR-EXECUTION-TRUST M3 is landed and supplies the existing chain and completion primitives.
- Model-registry price and harness metadata must be available during startup validation; missing data must fail loudly.
- Fresh attempt isolation may expose assumptions in clone/worktree setup. Keep the executor injectable so this is proven without live provider calls.
- Cloud Run timeout scaling and deferred completion paths are regression-sensitive; both require explicit tests before registry rollout.

## Execution Handoff

Implementation requires explicit user approval under the repository gate. After approval, invoke `sprint-executor` with `.ailang/state/sprints/sprint_M-SECONDARY-MODEL-FALLBACK.json`; do not implement from this planning task alone.
