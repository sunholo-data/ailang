# Sprint Plan — M-SECONDARY-MODEL-FALLBACK

**Design doc:** [m-secondary-model-fallback.md](m-secondary-model-fallback.md)  
**Target:** v1.2.0  
**Duration:** 3 days (about 21 engineering hours)  
**Estimated change:** ~820 LOC (implementation, tests, and operator documentation)  
**Risk:** High — this changes fleet-wide retry, cost, worktree isolation, and completion accounting  
**Dependencies:** M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design `task-08032ebc`

## Goal

Let an explicitly configured cloud executor retry one transport-failed model on one compatible
fallback model inside the same Cloud Run execution, while keeping git/auth/model failures terminal,
bounding spend, isolating partial work, and exposing every attempted model in the completion record.

## Verified starting point

- `AgentConfig` has a single `Model` pin and no fallback list.
- `ResolveModelChain` returns only that pin; role chains are consulted only without a pin.
- `ClassifyFailure` is called only by tests. `exited with error: signal: killed` is not transport-class;
  hard timeout and git failures correctly fall through to terminal model-class today.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist but the cloud job does not populate them.
- `executeCloudTask` owns clone, execution, commit, and push in one function and uses
  `/workspace/{taskID}`. The retry seam must therefore be extracted/testable rather than duplicating
  the full command path.
- The stale-task detector already owns infrastructure re-dispatch and caps executions at two; it is
  outside this sprint's implementation surface.
- The last-seven-day checkout history is shallow and documentation-dominated, so it is not a usable
  LOC velocity measure. The design's two-day estimate is expanded to three days for test seams,
  startup validation, and cross-component completion plumbing.
- Targeted Go baselines could not be executed in this planner environment because `go` is absent.
  The executor must establish them before the first RED test and stop if failures are unrelated.

## Planning discrepancy resolved

The design requires each attempt to use a fresh isolated worktree, but also says a walk has “no
re-clone.” Both cannot be true with the current `executeCloudTask`, which clones into its workdir.
This plan keeps the safety invariant: attempt 1 uses `/workspace/{taskID}/attempt1` and performs its
own clean clone. The added clone/time is included in the cost and wall-clock budget. Reusing a
transport-failed attempt's tree is out of scope unless a later design supplies a proven clean reset.

## Milestones

### M1 — Explicit chain configuration and fail-loud validation (~230 LOC, 0.75 day)

Add `FallbackModels []string` to `AgentConfig`; append it after the explicit pin in
`ResolveModelChain`. Add one validation seam that resolves pin and tails through `modelreg`, rejects
more than two tails, duplicates, the head repeated as a tail, missing registry rows, executor/harness
mismatch, and a fallback whose effective price exceeds 3× the head. Empty tails preserve current
behavior. Carry the validated chain through `DispatchParams` and emit `AILANG_MODEL_CHAIN`; keep
`AILANG_MODEL` as the byte-identical head for compatibility.

**Example files to update:** `internal/coordinator/agent_registry.go`,
`internal/coordinator/retry_chain.go`, the Cloud Run dispatcher implementation and its tests,
`internal/modelreg/models.go` only if a small lookup helper is required.

**Acceptance:**

- A pin `A` plus `[B, C]` resolves exactly to `[A, B, C]`; no tail resolves exactly as before.
- Every invalid list fails startup/reload with agent ID, rejected entry, and reason.
- Price comparison uses the registry's effective-price semantics, not an ad hoc unweighted sum.
- Existing 37-agent registry produces unchanged heads and no extra model invocation.
- Dispatcher tests assert the exact chain env and 2× job timeout only for a non-empty tail, bounded
  by Cloud Run's 24-hour maximum.

### M2 — Bounded, isolated in-container walk (~270 LOC, 1 day)

Add the measured killed-process signature and pin exact classifier cases for 429/503, idle
mid-generation death, killed process, hard timeout, git/deploy-key/auth, refusal, and unknown errors.
Extract a dependency-injected attempt runner around `executeCloudTask`; parse the validated chain;
run at most two models (`MaxInContainerWalks = 1`); and walk only after `FailureTransport`.
Parameterize the workdir with the link index so each attempt starts from a clean clone. Preserve the
single existing completion publication guard.

**Example files to update:** `internal/coordinator/retry_chain.go`,
`internal/coordinator/retry_chain_test.go`, `cmd/ailang/coordinator_cloud.go`, and a focused
`cmd/ailang/coordinator_cloud_test.go` test seam.

**Acceptance:**

- Fake A returns 503 and fake B succeeds: exactly two calls, B is terminal, one completion is sent.
- Git failure, auth failure, hard timeout, refusal, and unknown error: exactly one call on A.
- A and B transport-fail: exactly two calls, never C, with a terminal failed completion.
- Attempt workdirs differ (`attempt0`, `attempt1`) and attempt 1 cannot see a sentinel from attempt 0.
- Existing infrastructure re-dispatch tests remain green and `stale_task_detector.go` is unchanged.

### M3 — Attempt accounting, banking, and notifications (~230 LOC, 0.75 day)

Add `CompletionAttempt` and `TaskCompletion.Attempts`; record model, link, class, sanitized error,
duration, and available cost for each attempt. Populate `ModelUsed` and `ChainLinkIndex` on every
completion path, including preflight/deferred failures with deterministic link-0 semantics. Aggregate
token/cost metrics across attempts while retaining the terminal attempt as `ModelUsed`. Bank the
terminal model and include model/link/attempts in the inbox completion notification. Emit a structured
`MODEL_WALK` stdout line.

**Example files to update:** `internal/pubsub/topics.go`, `cmd/ailang/coordinator_cloud.go`,
`internal/coordinator/pubsub_completion_handler.go`, and their serialization/handler tests.

**Acceptance:**

- Success, failure, walked success, and walked failure payloads always identify terminal model/link.
- A walked payload contains both ordered attempts and its narrative names both models and class.
- Banked `ExecuteResult` and inbox JSON expose terminal model/link; total cost/tokens include all runs.
- Error text is bounded/sanitized consistently with existing completion payload constraints.
- JSON round-trip remains backward-compatible when `attempts` is absent.

### M4 — Registry proof, documentation, and repository gates (~90 LOC, 0.5 day)

Add a no-tail compatibility fixture over the production registry, document `fallback_models`, its
same-harness rule, 3× price ceiling, one-walk cap, second-clone cost, and operator-visible completion
fields. Do not add fallbacks to the 37 live agents in this sprint: rollout is a separate attended
configuration decision after code approval.

**Example files to update:** the canonical coordinator configuration/help documentation identified
by the executor's CLI-doc audit, registry fixtures, and changelog. No `.ail` example is applicable:
this is coordinator behavior, demonstrated by deterministic Go integration tests.

**Acceptance:**

- `go test ./internal/coordinator ./internal/pubsub ./internal/modelreg ./cmd/ailang -count=1` passes.
- `make test`, `make lint`, and `make check-boundaries` pass.
- Documentation states opt-in behavior and worst case: 2 executions × 2 model runs = 4 invocations.
- No production agent gains `fallback_models` as an incidental code change.

## Day-by-day execution

- **Day 1:** establish baselines; implement M1 RED→GREEN; begin classifier and attempt-runner seam.
- **Day 2:** complete M2; prove terminal boundaries, isolation, one-walk cap, and unchanged stale path.
- **Day 3:** implement M3; finish M4 docs/compatibility audit; run targeted and full repository gates.

## Risks and controls

- **False-positive fallback burns money:** unknowns remain terminal; exact negative fixtures cover
  git/auth/hard-timeout cases.
- **Metrics undercount first attempt:** attempt list and aggregate-cost tests cover walked success and
  failure; zero/unknown cost stays explicit rather than fabricated.
- **Dirty partial work contaminates fallback:** separate workdirs and a sentinel isolation test.
- **Cloud timeout API differs from task timeout:** dispatcher contract tests pin both values and cap.
- **Model registry namespace ambiguity:** resolve both head and tails through one helper and test raw
  wire name plus friendly-name rejection/acceptance according to the approved design.
- **Large cloud-job function makes tests brittle:** extract only the bounded loop/attempt interface;
  do not refactor unrelated clone/push behavior in this sprint.

## Scope boundaries

No cross-harness fallback, role-chain activation for pinned agents, local GPU fallback, new task
states, stale-detector changes, production fallback-list rollout, or changes to `MaxTaskExecutions`.

## Approval and handoff

This plan is ready for human review. Implementation begins only after the user explicitly says
`execute sprint`; the executor should update only milestone progress fields in the JSON artifact.
