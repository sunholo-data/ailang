# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

**Design doc:** [m-secondary-model-fallback.md](../m-secondary-model-fallback.md)  
**Target:** v1.2.0  
**Duration:** 3 days (~21 engineering hours)  
**Estimated change:** ~800 LOC  
**Risk:** High — cloud execution, retry cost, and completion accounting  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design doc

## Goal

Allow an explicitly configured cloud agent to make one same-harness fallback attempt after a transport-class model failure, inside the same Cloud Run execution. The walk is opt-in, cost-bounded, deterministic in tests, and visible in every completion payload. Git/auth failures, hard timeouts, and unclassified failures remain terminal.

## Planning Basis

- All 37 production agents currently have explicit one-link model pins.
- `ClassifyFailure` exists and is tested but has no production caller.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but are not populated.
- The stale-task detector remains the sole infrastructure re-dispatcher and is out of scope.
- The shallow checkout provides no responsible LOC/day signal. Estimates use the approved design's file inventory, current test surfaces, and a 25% integration reserve.
- No `.ail` example is appropriate; deterministic Go fixtures and registry YAML fixtures are the executable examples.

## Milestones

### M1 — Explicit Chain Configuration and Validation

**Estimate:** 80 implementation + 150 tests = **230 LOC**, 5 hours  
**Files:** `internal/coordinator/agent_registry.go`, `internal/coordinator/retry_chain.go`, and focused tests.

Tasks:

1. Add `FallbackModels` to `AgentConfig` and append it after the explicit pin in `ResolveModelChain`.
2. Add limits for two configured tail entries, one in-container walk, and a 3× blended-price ceiling.
3. At startup/reload, reject duplicate, pin-equal, unresolvable, cross-harness, over-price, and over-length entries.
4. Add the measured killed-process signature to transport classification while pinning hard timeout and git/auth failures as terminal.

Acceptance:

- A configured A→B→C chain resolves in order while `ResolveModel` remains A.
- The unconfigured 37-agent registry preserves current chain heads and behavior.
- Every invalid configuration fails loudly with agent, entry, and reason.
- Exact September failure strings distinguish walkable transport deaths from terminal hard-timeout and git failures.
- `go test ./internal/coordinator/...` passes.

### M2 — Dispatch the Validated Chain

**Estimate:** 55 implementation + 85 tests = **140 LOC**, 3.5 hours  
**Files:** `internal/coordinator/cloud_dispatcher.go`, `internal/coordinator/cloud_dispatch_test.go`, and job-spec tests.

Tasks:

1. Set `AILANG_MODEL_CHAIN` to the ordered comma-joined chain beside the head model.
2. Preserve one-link behavior, with the new chain variable equal to the pin.
3. Double job timeout only for multi-link chains, capped at 24 hours; keep per-attempt limits unchanged.
4. Test serialization, malformed input, and timeout boundaries against the final job spec.

Acceptance:

- One-link and multi-link environment payloads preserve exact order and namespace.
- Timeout is unchanged for one link and `min(2*taskTimeout,24h)` otherwise.
- Empty or malformed chain data fails loudly.
- Dispatcher tests pass.

### M3 — One Bounded In-Container Walk

**Estimate:** 150 implementation + 170 tests = **320 LOC**, 8 hours  
**Files:** `cmd/ailang/coordinator_cloud.go`, new focused walk tests, and `internal/pubsub/topics.go`.

Tasks:

1. Extract a testable loop around `executeCloudTask` with an injected fake executor.
2. Parse the dispatched chain and isolate each attempt in `attempt{n}` worktrees.
3. Walk only for transport failures when a next link exists and the one-walk budget remains.
4. Add ordered `CompletionAttempt` records containing model, link, class, error, duration, and cost.
5. Preserve exactly-once completion publication on success, failure, deferred, and panic paths.

Acceptance:

- Fake 503 on A then success on B makes two model calls in one execution and terminates at link 1.
- Git failure and hard timeout make exactly one model call.
- Final-link failure or spent walk budget never makes a third call.
- Attempt 2 cannot observe partial files from attempt 1; attempt 1 evidence remains in history.
- Every exit path publishes exactly one completion.

### M4 — Bank and Surface Fallback Observability

**Estimate:** 45 implementation + 65 tests = **110 LOC**, 4.5 hours  
**Files:** completion handler and tests, `internal/pubsub/topics.go`, stale-detector regression tests, and `docs/docs/guides/coordinator.md`.

Tasks:

1. Bank and notify `ModelUsed`, `ChainLinkIndex`, and structured attempt history.
2. Add bounded `MODEL_WALK` logs and narrative error context.
3. Test every completion variant with and without a walk.
4. Prove stale-task dispatch and the two-execution cap are unchanged.
5. Document opt-in configuration, limits, and observability.

Acceptance:

- Every completion identifies terminal model/link and includes at least one attempt.
- Inbox/portal payloads expose the model and link; walked failures name both models and the first class.
- Tests prove at most two executions × two model calls = four invocations per task.
- `make test`, `make lint`, and `make check-boundaries` pass.

## Day-by-Day Plan

- **Day 1:** M1 and M2, including registry validation and rendered job-spec tests.
- **Day 2:** M3 red-first; cover success-after-walk, terminal classes, isolation, exhaustion, and exactly-once completion.
- **Day 3:** M4 observability, regression tests, docs, full test/lint/boundary gates, and integration reserve.

## Success Metrics

- `ClassifyFailure` gains a production call site at the executor error boundary.
- A deterministic 503 fixture completes on link 1 without re-dispatch.
- Git/auth and hard-timeout fixtures never invoke link 1.
- 100% of cloud completions identify terminal model and chain link.
- Agents without `fallback_models` retain current behavior.
- Cost is structurally bounded to one fallback per execution and two executions per task.
- All focused and repository-wide gates pass.

## Deferred / Non-Goals

Cross-harness fallback, role-chain tails for pinned agents, quota-aware skipping, local GPU fallback, stale-task changes, and fleet-wide rollout are deferred. Enabling production fallbacks remains a separate operator decision because it changes cost.

## Handoff Gate

This plan is ready for human review. Per repository routing rules, implementation and the sprint-executor handoff begin only after the user explicitly says **“execute sprint.”**
