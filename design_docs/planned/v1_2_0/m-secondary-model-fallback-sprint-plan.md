# Sprint Plan — M-SECONDARY-MODEL-FALLBACK

**Design doc**: [../m-secondary-model-fallback.md](../m-secondary-model-fallback.md)  
**Sprint ID**: `M-SECONDARY-MODEL-FALLBACK`  
**Target**: v1.2.0 · P1  
**Planned at**: HEAD `c608de32`, 2026-09-14  
**Duration**: 3 days, 4 milestones  
**Estimated change**: ~1,140 LOC (implementation, tests, and fixtures)  
**Risk**: High — this changes fleet-wide execution, cost bounds, failure classification, and completion accounting.  
**Dependencies**: M-COORDINATOR-EXECUTION-TRUST M3 (landed); approved design doc.

## 1. Summary

Add an explicit, same-harness `fallback_models` tail to pinned cloud agents and walk at most once inside a Cloud Run execution when the current model fails with a transport-class error. Validate every configured tail before dispatch, isolate partial attempt work, and make the chosen link and complete attempt history visible in every completion.

The design estimated about two days. This plan budgets three because the implementation crosses registry validation, model-registry identity/pricing, Cloud Run job construction, executor lifecycle, Pub/Sub schema, result banking, and notification payloads. The extra day is verification and integration buffer, not added scope.

## 2. Current status and evidence

- `AgentConfig` has `Model` but no fallback tail (`internal/coordinator/agent_registry.go`).
- `ResolveModelChain` returns only the explicit pin when `Model` is set (`internal/coordinator/retry_chain.go`).
- `ClassifyFailure` is referenced only by its tests; `exited with error: signal: killed` is not a transport signature.
- `coordinator_cloud.go` reads one `AILANG_MODEL`, calls `executeCloudTask` once, and publishes one terminal result.
- `TaskCompletion.ModelUsed` and `ChainLinkIndex` exist in `internal/pubsub/topics.go`, but the cloud publish closure does not populate them.
- `MaxTaskExecutions=2` and stale-task redispatch remain out of scope and must not change.
- Recent-history velocity is not measurable in this shallow planning checkout: the 14-day script sees one documentation commit and no diff base. Estimate therefore uses the approved design's two-day estimate plus a 50% integration/risk buffer.
- Planner baseline limitation: this workspace has neither `go` nor `make`; focused tests, lint, and boundary gates returned command-not-found and must be established by the executor before editing.

## 3. Milestone map

| Milestone | Outcome | Estimate | Dependency |
|---|---|---:|---|
| M1 | Explicit chain configuration and loud startup validation | 330 LOC | none |
| M2 | Chain delivery and bounded Cloud Run timeout | 180 LOC | M1 |
| M3 | Isolated, one-walk in-container execution loop | 370 LOC | M1, M2 |
| M4 | Completion accounting, banking, notifications, and end-to-end regression | 260 LOC | M3 |

Total: **1,140 LOC**, including approximately 500–600 LOC of tests.

## 4. Milestones

### M1 — Explicit chain configuration and startup validation

**Goal**: Make fallback behavior opt-in and refuse invalid, incompatible, duplicate, overlong, or over-price tails before the coordinator accepts work.  
**Estimate**: 140 implementation + 190 tests = **330 LOC** (0.75 day).

**Files to update**:

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- `internal/coordinator/agent_registry_test.go` or a focused new `internal/coordinator/fallback_validation_test.go`
- `internal/modelreg/` only if a small exported identity/pricing lookup is required

**Tasks**:

1. Add `FallbackModels []string` with YAML/JSON names adjacent to `Model`.
2. Append the explicit tail in `ResolveModelChain` without changing the head or unpinned role-chain behavior.
3. Add `MaxFallbackChainTail=2`, `MaxInContainerWalks=1`, and `MaxFallbackPriceRatio=3.0` as named bounds.
4. Validate resolution by agent-model/API identity, same `agent_cli`, blended price ratio, count, uniqueness, and pin exclusion at startup and registry reload.
5. Add the measured killed-process signature; pin hard timeout and git/auth errors as terminal.

**Acceptance criteria**:

- [ ] `model: A` + `fallback_models: [B, C]` resolves exactly `[A, B, C]`; the chain head still equals `ResolveModel`.
- [ ] Empty tails preserve all live registry agents' effective chain and behavior.
- [ ] Unknown, cross-harness, duplicate, pin-equal, over-two, and over-3× entries fail loudly with agent, entry, and reason.
- [ ] Exact September strings prove idle mid-generation and killed-process errors walk, while hard timeout and git-push errors terminate.
- [ ] No role-chain tail is attached implicitly to a pinned agent.

**Risk**: Wire model strings may map to aliases. Mitigation: use one model-registry identity lookup and table-test friendly name, `agent_model_name`, and API-name forms.

### M2 — Dispatch chain and bound the Cloud Run job

**Goal**: Deliver the validated chain to the existing executor container without changing the no-tail job shape beyond a single-value chain variable.  
**Estimate**: 70 implementation + 110 tests = **180 LOC** (0.4 day).

**Files to update**:

- `internal/coordinator/cloud_dispatcher.go`
- existing Cloud Run job/env tests near `cloud_dispatcher.go`

**Tasks**:

1. Set comma-joined `AILANG_MODEL_CHAIN` beside the existing head `AILANG_MODEL`.
2. Retain `AILANG_MODEL` for compatibility and assert it equals chain link 0.
3. Double the job timeout only when a tail exists, preserving each attempt's current timeout and respecting Cloud Run's 24-hour ceiling.

**Acceptance criteria**:

- [ ] No-tail dispatch has `AILANG_MODEL_CHAIN=A`, retains `AILANG_MODEL=A`, and does not scale timeout.
- [ ] Tail dispatch has `AILANG_MODEL_CHAIN=A,B` and scales job timeout exactly 2×, capped at 24 hours.
- [ ] Invalid chain configuration cannot reach job construction.
- [ ] Existing provider/image/env tests remain unchanged and green.

**Risk**: Cloud Run duration types and caps can introduce rounding. Mitigation: table-test below, exact, and above-cap durations.

### M3 — Isolated bounded walk in the cloud job

**Goal**: Execute link 1 only after a transport failure on link 0, in a fresh attempt worktree, while publishing exactly one completion.  
**Estimate**: 180 implementation + 190 tests = **370 LOC** (1.0 day).

**Files to update/create**:

- `cmd/ailang/coordinator_cloud.go`
- focused tests in `cmd/ailang/coordinator_cloud_test.go` or new `cmd/ailang/coordinator_cloud_fallback_test.go`
- `internal/coordinator/retry_chain.go` only for reusable loop policy helpers

**Tasks**:

1. Parse and cross-check `AILANG_MODEL_CHAIN`; fail loudly if malformed or inconsistent with `AILANG_MODEL`.
2. Extract a testable execution-loop seam around `executeCloudTask`.
3. Run each link in `/workspace/{taskID}/attempt{n}` from the same immutable base revision; never reuse partial files.
4. Call `ClassifyFailure` at the executor-error boundary and continue only for transport class, one remaining link, and unused walk budget.
5. Preserve the existing deferred `completionSent` guard, panic behavior, and one-completion invariant.
6. Emit structured `MODEL_WALK` stdout with task, links, models, and class.

**Acceptance criteria**:

- [ ] A fake executor returning 503 on A then success on B runs twice inside one execution and publishes once.
- [ ] Git failure, hard timeout, malformed chain, and model-class failure invoke only A.
- [ ] Three configured links still invoke at most two models because `MaxInContainerWalks=1` is enforced by code.
- [ ] Attempt 2 starts from a fresh clone/workdir and cannot observe attempt 1's partial file mutation.
- [ ] Panic and early validation paths still publish exactly one failed completion.
- [ ] Stale-task detector code and its redispatch semantics remain unchanged.

**Risk**: `executeCloudTask` currently owns clone and execution together. Mitigation: inject only the narrow function seam needed by tests; do not create a second dispatcher or task state machine.

### M4 — Observable completion and end-to-end accounting

**Goal**: Make every terminal result name its final model/link and preserve all attempt evidence through Pub/Sub, banking, and inbox notification.  
**Estimate**: 100 implementation + 160 tests = **260 LOC** (0.85 day).

**Files to update/create**:

- `internal/pubsub/topics.go`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- corresponding Pub/Sub schema, cloud completion, handler, and notification tests

**Tasks**:

1. Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, failure class, bounded error, duration, and cost.
2. Populate `ModelUsed` and `ChainLinkIndex` on success, failure, panic, and deferred-guard completions.
3. Narrate walked terminal errors with both links and the first failure class without losing structured attempts.
4. Bank the terminal model/link into `ExecuteResult` and include both fields in completion notifications.
5. Add a deterministic end-to-end fake-executor test for 503→success and terminal controls.

**Acceptance criteria**:

- [ ] Every cloud completion carries a non-empty `ModelUsed`, a valid `ChainLinkIndex`, and at least one attempt.
- [ ] A 503→success completion reports B/link 1 and contains ordered attempts for A and B.
- [ ] A walked failure's `ErrorMsg` names both links and classifies A's failure as transport.
- [ ] Notification payload and banked result expose the actual terminal model and link.
- [ ] Attempt error text and history are bounded so Pub/Sub payload size cannot grow without limit.
- [ ] Worst case remains two executions × two invocations; tests cover the compound ceiling without changing `MaxTaskExecutions`.

**Risk**: Existing consumers may assume absent attempt history. Mitigation: use `omitempty`, retain old fields, and add round-trip/backward-compatibility tests.

## 5. Day-by-day execution

### Day 1 — Configuration and dispatch

- Establish executor-machine baselines: focused packages, `make lint`, and `make check-boundaries`.
- Complete M1 test-first, including alias/harness/price refusal tables.
- Complete M2 and verify no-tail registry dispatch parity.
- Checkpoint: coordinator and model-registry packages green; no live agent configured with a tail.

### Day 2 — Walk loop and isolation

- Build the fake executor seam and red tests for 503→success, terminal controls, one-walk ceiling, and fresh workdirs.
- Implement M3 without touching stale-task redispatch.
- Checkpoint: job tests prove one execution, one completion, and at most two model runs.

### Day 3 — Accounting and full verification

- Complete M4 schema, publisher, banking, notification, and round-trip tests.
- Run the deterministic end-to-end walk simulation and refusal controls.
- Run `make test`, `make lint`, `make check-boundaries`, formatting, and diff checks.
- Verify the registry remains opt-in; do not add fleet `fallback_models` rollout entries in this sprint.

## 6. Verification gates

The executor must first record base results because this planner environment cannot run Go tooling.

```bash
go test ./internal/coordinator ./internal/pubsub ./cmd/ailang
make test
make lint
make check-boundaries
gofmt -l internal/coordinator internal/pubsub cmd/ailang
git diff --check
```

Additional invariants:

- `rg "ClassifyFailure\("` shows at least one non-test call site.
- `git diff -- internal/coordinator/stale_task_detector.go` is empty.
- No placeholder milestone IDs or acceptance criteria remain in the progress JSON.
- A no-tail fixture proves head and execution behavior are unchanged.
- A fake transport outage proves fallback without any real provider call.

## 7. Success metrics

- 100% of cloud completion paths identify terminal model and link.
- 503/idle/killed transport failures can walk once; git/auth/hard-timeout/model failures cannot.
- Maximum remains 4 invocations per task across the existing two-execution cap.
- Invalid or >3× fallback configuration prevents coordinator startup.
- No behavior change for agents without `fallback_models`.
- All focused and repository gates pass on an executor with Go tooling.

## 8. Dependencies, assumptions, and non-goals

- The design doc is approved; this plan does not reopen its D1–D5 decisions.
- Model pricing and `agent_cli` metadata in `internal/modelreg/models.yml` remain the validation source of truth.
- Rollout configuration for the 37 production agents is a separate, explicit operator decision after implementation.
- Cross-harness and local-GPU fallback, role-chain activation for pinned agents, new task states, and changes to stale-task redispatch are excluded.
- No new `.ail` files or AILANG syntax changes are planned, so `ailang prompt` is not required for execution.

## 9. Executor handoff gate

The artifacts are ready for human review. Per repository routing, implementation starts only after the user explicitly says **“execute sprint”**; at that point invoke `sprint-executor` with this plan and `.ailang/state/sprints/sprint_M-SECONDARY-MODEL-FALLBACK.json`.
