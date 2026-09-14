# Sprint Plan: M-SECONDARY-MODEL-FALLBACK

## Summary

Implement the approved, opt-in same-harness model fallback for cloud executor agents. A transport-class failure may make one bounded in-container walk from the pinned model to an explicitly configured fallback, while terminal failures remain on link 0 and every attempt is observable in the completion payload.

**Duration:** 3 engineering days (about 22 hours)
**Dependencies:** Approved [M-SECONDARY-MODEL-FALLBACK design](m-secondary-model-fallback.md); M-COORDINATOR-EXECUTION-TRUST M3 (landed)
**Risk Level:** High — this changes unattended execution, cost bounds, registry startup validation, and completion accounting across a Cloud Run boundary

## Current Status Analysis

### Completed Recently

- The trust work already introduced `ClassifyFailure`, `ResolveModelChain`, `MaxTaskExecutions`, and completion fields `ModelUsed`/`ChainLinkIndex`.
- The approved design verified all 37 production agents are explicitly pinned and therefore currently resolve to one-link chains.
- The current branch includes a recent coordinator stage-boundary test/fix, showing active test coverage around cross-stage behavior.

### Velocity and Capacity

- The seven-day velocity script found only one commit in this isolated coordinator worktree and no reliable LOC/day baseline; estimates therefore use file-level inspection and include a 25% integration buffer.
- Planned capacity is approximately 1,150 changed LOC over 3 days: about 650 implementation/configuration LOC and 500 test LOC.
- The work is split at independently testable boundaries so execution can stop safely after any milestone.

### Remaining from the Design Doc

- Add explicit `fallback_models` configuration and startup validation.
- Carry the resolved chain into Cloud Run with a bounded timeout adjustment.
- Execute one same-container fallback on transport failures with isolated attempt workdirs.
- Populate terminal-link and per-attempt observability through completion banking and notifications.
- Prove back-compatibility, terminal classification, cost bounds, and unchanged stale-task behavior.

## Proposed Milestones

### M1: Configuration, chain resolution, and startup validation (~300 LOC)

**Goal:** Make fallback configuration explicit, opt-in, same-harness, price-bounded, and invalid at startup rather than during an outage.

**Estimated:** 140 implementation LOC + 160 test LOC = 300 LOC
**Duration:** 6 hours
**Dependencies:** None

**Example files to update:**

- `internal/coordinator/agent_registry.go`
- `internal/coordinator/retry_chain.go`
- `internal/coordinator/retry_chain_test.go`
- relevant registry validation test file under `internal/coordinator/`

**Tasks:**

- Add `FallbackModels []string` beside `AgentConfig.Model` with YAML/JSON tags.
- Append explicit fallbacks after the pin in `ResolveModelChain`; preserve existing role-chain behavior for unpinned agents.
- Add chain-tail, price-ratio, duplicate, pin-equality, resolvability, and harness-compatibility validation.
- Wire validation into startup and registry reload using the existing loud validation path.
- Add the measured killed-process transport signature and pin hard timeout and git/auth errors as terminal.

**Acceptance Criteria:**

- [ ] `model: A` plus `fallback_models: [B]` resolves exactly to `[A, B]`; an empty list preserves the existing chain head and behavior.
- [ ] Startup rejects unresolved, cross-harness, duplicate, pin-equal, over-length, and over-3x-price fallbacks with agent, model, and reason in the error.
- [ ] Exact September error strings prove idle/killed failures walk while hard timeout and git failures remain terminal.
- [ ] Existing role-routing and chain-head tests pass without weakening assertions.

**Risks:** Registry model names and executor identities have multiple representations. Mitigation: resolve through `modelreg` once and use table-driven tests containing live registry shapes.

### M2: Dispatch chain transport and bounded execution loop (~360 LOC)

**Goal:** Carry the validated chain into the job and perform at most one isolated retry inside the same Cloud Run execution.

**Estimated:** 190 implementation LOC + 170 test LOC = 360 LOC
**Duration:** 7 hours
**Dependencies:** M1

**Example files to update/create:**

- `internal/coordinator/cloud_dispatcher.go`
- dispatcher tests under `internal/coordinator/`
- `cmd/ailang/coordinator_cloud.go`
- `cmd/ailang/coordinator_cloud_fallback_test.go` (new focused test file)

**Tasks:**

- Set `AILANG_MODEL_CHAIN` from `ResolveModelChain`; keep `AILANG_MODEL` as the head for compatibility.
- Scale the Cloud Run job timeout only for configured chains, bounded by the platform maximum.
- Parse and validate the chain in the job binary, then wrap `executeCloudTask` behind a fakeable attempt runner.
- Run each model in an `attempt{n}` workdir and discard failed-attempt filesystem state.
- Call `ClassifyFailure` at the executor error site; walk only on transport class, a remaining link, and `MaxInContainerWalks=1`.
- Accumulate metrics and failure details without publishing an intermediate completion.

**Acceptance Criteria:**

- [ ] A deterministic fake 503 on A invokes B once in the same execution; git and hard-timeout errors invoke no second model.
- [ ] The loop enforces one walk even when a three-link chain is supplied.
- [ ] Each attempt receives a distinct workdir derived from the same task/base input.
- [ ] Exactly one completion is published and `AttemptCount` remains the Cloud Run execution count, not the model-run count.
- [ ] Single-link agents retain their prior timeout and execution path aside from the chain env containing the pin.

**Risks:** `coordinator_cloud.go` couples execution, artifacts, evidence, and completion guards. Mitigation: extract only a narrow attempt-runner seam and retain the existing single-publish guard.

### M3: Completion observability and banking (~260 LOC)

**Goal:** Make every terminal model link and every paid attempt visible to operators and downstream consumers.

**Estimated:** 120 implementation LOC + 140 test LOC = 260 LOC
**Duration:** 5 hours
**Dependencies:** M2

**Example files to update:**

- `internal/pubsub/topics.go`
- Pub/Sub serialization tests under `internal/pubsub/`
- `cmd/ailang/coordinator_cloud.go`
- `internal/coordinator/pubsub_completion_handler.go`
- completion-handler tests under `internal/coordinator/`

**Tasks:**

- Add `CompletionAttempt` and `TaskCompletion.Attempts` with model, link, class, error, duration, and cost.
- Populate `ModelUsed`, `ChainLinkIndex`, and attempts on success, failure, panic/deferred-guard, walked, and non-walked completions.
- Narrate both links and the first failure class in walked terminal errors; emit structured `MODEL_WALK` output.
- Bank the actual terminal model in `ExecuteResult` and include model/link fields in inbox completion notifications.
- Define aggregation behavior for task-level token, duration, and cost fields so paid attempts are not hidden or double-counted.

**Acceptance Criteria:**

- [ ] All completion paths carry the terminal `ModelUsed` and `ChainLinkIndex`; walked runs carry two ordered attempts.
- [ ] A walked success is visibly distinguishable from a link-0 success without consulting logs.
- [ ] A walked failure names both links and the initial transport class in `ErrorMsg`.
- [ ] Banked execution and inbox notification data identify the model that actually produced the terminal result.
- [ ] JSON round-trip tests preserve attempt history and remain compatible with older payloads lacking the new field.

**Risks:** Existing metrics describe one executor result. Mitigation: document and test that top-level cost/tokens aggregate all paid attempts while terminal-model identity remains separate.

### M4: Fleet back-compatibility and end-to-end guardrails (~230 LOC)

**Goal:** Prove the feature is inert until configured and cannot reopen the stale-task re-dispatch lane or exceed its stated spend ceiling.

**Estimated:** 40 implementation/docs LOC + 190 test LOC = 230 LOC
**Duration:** 4 hours
**Dependencies:** M1, M2, M3

**Example files to update/create:**

- `internal/coordinator/retry_chain_test.go`
- coordinator dispatch/completion integration tests
- `docs/docs/guides/coordinator.md` or the current coordinator configuration reference
- production agent registry YAML only if a separately approved rollout is requested (not part of this sprint)

**Tasks:**

- Add a registry-wide test proving current agents have no fallback and preserve chain heads/default execution.
- Exercise the compound ceiling: two executions times at most two model invocations.
- Assert stale-task detector ownership and behavior remain unchanged using behavioral tests, not a brittle source-file diff.
- Document the opt-in field, startup failure conditions, price ratio, one-walk bound, and observability fields.
- Run focused tests, full tests, lint, formatting, and architecture boundary checks.

**Acceptance Criteria:**

- [ ] Current live registry entries produce no fallback behavior by default.
- [ ] Tests prove a task cannot exceed 2 executions x 2 model runs.
- [ ] Existing stale-task detector tests pass and no second re-dispatch owner is introduced.
- [ ] Coordinator configuration documentation explains cost and compatibility validation before showing an example.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.

**Risks:** Full-suite failures may be unrelated in a fast-moving branch. Mitigation: record focused green evidence first, then report any unrelated full-suite failure with exact command and ownership rather than weakening gates.

## Day-by-Day Plan

### Day 1 — Safe configuration boundary

- Complete M1 with table-driven validation/classification tests.
- Begin M2 by carrying the chain and timeout into the job environment.
- Pause point: configuration may merge only when invalid fallback lanes fail before dispatch and empty fallback lists are behaviorally inert.

### Day 2 — Execution and attribution

- Complete the fakeable bounded walk in M2.
- Complete M3 payload, banking, notification, and aggregation behavior.
- Pause point: the deterministic 503-to-success scenario publishes exactly one completion with two attempts and terminal link 1.

### Day 3 — System guardrails and verification

- Complete M4 back-compatibility, compound-bound, docs, and stale-detector tests.
- Run formatting, focused tests, `make test`, `make lint`, and `make check-boundaries`.
- Do not add production `fallback_models` entries in this sprint; fleet rollout requires an explicit operator-reviewed configuration change.

## Success Metrics

- `ClassifyFailure` has a production call site at the executor failure boundary.
- A deterministic transport failure succeeds on link 1 with one Cloud Run execution and two recorded model attempts.
- Git, auth, scope refusal, and hard-timeout failures remain terminal on link 0.
- `ModelUsed` and `ChainLinkIndex` are populated for every cloud completion path.
- The current 37-agent registry has no behavior or spend change by default.
- Worst-case task spend is mechanically capped at four model invocations across two executions.
- Focused fallback tests, full tests, lint, and boundary checks pass.

## Dependencies and Sequencing

| Milestone | Depends on | Parallelism |
|---|---|---|
| M1 | Approved design | Starts first |
| M2 | M1 chain contract | Dispatch tests can begin while validation tests finish |
| M3 | M2 attempt result shape | Pub/Sub type/round-trip tests can begin early |
| M4 | M1-M3 | Final integration gate |

## Open Questions for Execution

- Confirm the canonical existing coordinator configuration documentation path before editing docs.
- Choose the smallest existing registry-reload hook that can reuse startup validation; do not create a second validator.
- Confirm whether top-level completion token/cost fields already mean total execution cost. The implementation must make the chosen aggregation explicit in tests.

## Explicit Non-Goals

- No cross-harness fallback.
- No role-chain tail for pinned agents.
- No production fleet rollout or new fallback entries.
- No changes to `MaxTaskExecutions`, stale-task ownership, or local GPU fallback.

## Estimate Summary

| Milestone | Implementation/docs LOC | Test LOC | Total LOC | Hours |
|---|---:|---:|---:|---:|
| M1 | 140 | 160 | 300 | 6 |
| M2 | 190 | 170 | 360 | 7 |
| M3 | 120 | 140 | 260 | 5 |
| M4 | 40 | 190 | 230 | 4 |
| **Total** | **490** | **660** | **1,150** | **22** |

