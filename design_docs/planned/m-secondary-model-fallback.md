# M-SECONDARY-MODEL-FALLBACK: a cloud executor agent whose pinned model is unavailable walks to a next model instead of failing

**Status**: Planned
**Target**: v1.2.0
**Priority**: P1 — September's eight executor failures include at least three that a working fallback would have converted from task-death into task-completion, and the retry module that would do it (`ClassifyFailure`, `ResolveModelChain`) is written, tested, and unwired.
**Estimated**: ~2 days
**Dependencies**: [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) M3 (landed — this doc implements its stated "model/provider class — in-container chain walk" tier, which was declared in `retry_chain.go` and never built). **Supersedes §6 of [m-cloud-plane-fallback-lanes.md](m-cloud-plane-fallback-lanes.md)** per that doc's own stopping-point recommendation: "re-derive §6 from scratch… a fresh derivation by an author without seven rounds of investment in this particular shape is more likely to succeed than an eighth patch." This is that re-derivation, deliberately narrower in scope.

**Created**: 2026-09-14
**Problem**: every agent on the cloud plane is a chain of ONE, so a 429 or a mid-generation death ends the task. The classifier that distinguishes "walk" failures from "terminal" failures exists and is never called.

---

## 1. The measurement (verified against HEAD 2026-09-14, prod plane `ailang-multivac`, 37 agents)

Every claim below was re-verified in source this session, not carried forward from the prior doc.

1. **Every agent is a chain of one.** `ResolveModelChain` (`internal/coordinator/retry_chain.go:115`) returns `[]string{agent.Model}` whenever an explicit pin exists. All 37 registry entries carry a `model:` pin — 33 on `openrouter/z-ai/glm-5.3-flash`, 2 on `openrouter/z-ai/glm-5.3`, 2 on `gpt-5.6-sol`. No cloud agent has any fallback.

2. **Role chains are dead config in this path.** `internal/modelreg/models.yml` declares role chains (e.g. `evaluator: [pi-or-minimax-m3, pi-claude-sonnet-4-6, opencode-or-minimax-m3]`), consulted only when `model` is empty — which is never. Worse, the declared tails name `opencode` harnesses while the agents run in `pi` containers: a chain resolving to a harness the container cannot run is worse than no chain.

3. **`ClassifyFailure` is unwired.** `retry_chain.go:70` defines it; grep at HEAD confirms nothing outside its own test calls it. The only thing that advances `ChainLinkIndex` today is the infrastructure class (timeout with no completion, via the stale-task detector, `stale_task_detector.go:130`). A provider returning 429 or 503 — exactly the "unavailable and fails" case — does not walk.

4. **What the failures actually look like** (September, live plane):
   - `pi exceeded hard timeout (15m0s)`
   - `pi idle for 3m mid-generation (no output)`
   - `pi exited with error: signal: killed [stderr: Warning: Model "minimax/..."]`
   - `codex task failed: exit status 1`
   - `git push failed` / deploy key errors — **not model failures; must NOT trigger a model walk.**

5. **The observability fields exist and are never written.** `pubsub.TaskCompletion` carries `ModelUsed` and `ChainLinkIndex` (`internal/pubsub/topics.go:63-64`), added by M-COORDINATOR-EXECUTION-TRUST M3 for exactly this purpose; grep at HEAD shows neither is populated anywhere. The completion payload cannot say which chain link ran.

**Impact**: a single provider hiccup on the fleet's dominant pin (`glm-5.3-flash`, 33/37 agents) kills every in-flight task simultaneously, and each death costs a full re-clone + re-dispatch when it retries at all. The walk this doc designs costs no re-clone and no re-dispatch — that is the point.

---

## 2. Why this shape and not the prior doc's shape

The prior proposal (`m-cloud-plane-fallback-lanes.md` §6) failed seven quorum rounds, and the review history's diagnosis is structural: it accumulated multiple paths doing one job — a coordinator-side `retry_requested` state, an `AttemptID` fencing primitive, a second eligibility predicate on the detector, cross-harness re-dispatch — and each round found a defect in the joins between them. Its author's stopping recommendation was to re-derive from scratch against: single application per failure, class persisted or re-derivable by every caller, no phase shipping a known-broken lane, one accounting model.

This design satisfies all four by **moving the walk inside the execution that failed**:

- **Single application per failure**: the walk is a `for` loop in one function in the job binary (`cmd/ailang/coordinator_cloud.go`). No coordinator state machine, no second dispatcher, no fencing primitive — there is exactly one execution and it publishes exactly one completion (the existing `completionSent` guard already enforces that). The V23 invariant (stale-task detector is the sole re-dispatcher) is untouched.
- **Class re-derivable**: the classifier runs where the error string is born, at the moment it is born.
- **No known-broken lane**: the fallback list is validated at coordinator startup for harness compatibility and price (§4.3), so a lane that cannot run is a boot error, not a runtime surprise.
- **One accounting model**: one execution → one completion → one `ModelUsed`/`ChainLinkIndex`, plus a per-attempt record list (§6).

The coordinator's contribution shrinks to two env vars and a startup validator. Everything that made the prior doc hard — CAS transitions, superseded-attempt completions, a detector that cannot observe the failure class — does not exist in a single-execution walk, because there is nothing to race with.

---

## 3. High-impact decisions

### D1(a) — Where the fallback list lives: a new `fallback_models:` field on `AgentConfig`

```yaml
model: openrouter/z-ai/glm-5.3-flash
fallback_models:
  - openrouter/z-ai/glm-5.3
  - openrouter/deepseek/deepseek-v4-flash-0731
```

`ResolveModelChain` appends them after the pin. Entries use **the same string format as `model:`** — a raw per-harness wire string, not a registry friendly name.

**Why the alternative — making the existing role chain the tail of a pinned chain — is worse:**

1. **It silently re-routes 37 agents the moment someone edits `models.yml`.** An operator who pins `model:` has made a deliberate choice; attaching a role tail means editing the evaluator role row in the registry changes the fallback behavior of pinned agents who never opted in. That is a silent fallback affecting cost — the exact failure mode CLAUDE.md Critical Principle 2 forbids. `fallback_models:` is per-agent and explicit: no agent's behavior changes until an operator writes its list.
2. **Namespace mismatch.** The pin is a raw harness string (`openrouter/z-ai/glm-5.3-flash`); role entries are friendly names resolving to `(Executor, ModelName)` pairs. A head/tail chain mixing the two needs a conversion layer where head-vs-tail comparison is apples-to-oranges — and the conversion is many-to-one in the wrong direction (roles.go's own header records why).
3. **Executor compatibility is structurally avoided, not validated at runtime.** The measured evaluator tail names `opencode-or-minimax-m3` while sprint-evaluator runs in a pi container. A friendly-name tail carries its own `agent_cli` which may disagree with the container it lands in — the prior doc needed a P4 capability-derivation phase to patch this. A `fallback_models:` wire string is executed by the **same harness, in the same container, through the same code path** as the pin (it becomes `AILANG_MODEL` for attempt 2). If the harness cannot run it, that is a startup validation error (§4.3), not a mid-outage dispatch to a dead lane.

### D2 — Which failure classes advance the chain

| class | advances? | signatures |
|---|---|---|
| **Transport** (provider unavailable, connection, mid-generation death) | **yes — walk** | existing `transportSignatures` (`status 429/500/502/503/504`, `rate limit`, `connection reset`, `mid-generation (no output)`, `returned zero bytes`, `empty completion`, …) **plus one addition: `exited with error: signal: killed`** — a measured September failure that currently classifies as model-class and would stay terminal. Substring is safe: it cannot appear in a git or auth error. |
| **Model** (refusal, wrong answer, uncompilable output, scope-guard refusal) | **no — terminal** | the default: `ClassifyFailure` returns model-class for anything unrecognized. This default direction is the classifier's most important property — the safe answer to "was this a walkable failure?" is **no**. |
| **Git / credentials / auth** | **no — terminal** | `git push failed`, deploy-key errors, 401s. None match a transport signature, so they fall through to model-class terminal **by construction, not by enumeration**. A fallback that fired on the git-credential class would burn a second model run for nothing; the boundary survives new git error strings because it never had to know them. |
| **Infrastructure** (container died, no completion at all) | **re-dispatch** (existing tier, unchanged) | the stale-task detector path. The job cannot walk a chain from inside a container that is gone. |

**Hard timeout is terminal, deliberately.** `pi exceeded hard timeout (15m0s)` is a budget outcome — the model had its full allocation and did not finish — not provider unavailability. A fallback attempt would buy another full budget for a task that may simply be too big, doubling spend on the most expensive failure class. (The idle-stall case — "working but stalled" — IS walk-eligible via `mid-generation (no output)`, which is the correct distinction: the stall says the provider hung; the hard timeout says the work is too large.) The signature `exceeded hard timeout` is therefore NOT added to `transportSignatures`, and a test pins that.

### D3 — Where `ClassifyFailure` is called, and what happens to partial work

**Call site: the job binary, on the executor error, before `publishCompletion`.** In `cmd/ailang/coordinator_cloud.go`, the single call `executeCloudTask(...)` becomes a bounded loop:

```
chain := parseChainFromEnv()          // AILANG_MODEL_CHAIN, §4.2
for link, model := range chain {
    branch, result, evidence, err := executeCloudTask(..., model, ...)
    if err == nil { break }           // success on this link
    class := ClassifyFailure(err.Error())
    log walk decision (link, model, class)
    if class != FailureTransport || link == len(chain)-1 || walkBudgetSpent() {
        break                          // terminal — publish failed with the full attempt history
    }
    // transport failure with a link remaining: continue loop
}
publishCompletion(status, errMsg-with-attempt-history, ..., modelUsed=currentLink, chainLinkIndex=currentLink)
```

**Partial work is discarded, deliberately, by construction.** Each attempt runs in its own fresh workdir (`/workspace/{taskID}/attempt{n}` — the workdir is already per-task at `coordinator_cloud.go:261`, so this is a one-line parameterization). A model that died mid-generation leaves a half-written tree behind; letting attempt 2 "continue" from a dead model's partial output is contamination, not progress — and it makes attempt 2's outcome unattributable to its model. A fresh clone of the same base commit is deterministic and matches the repo's own determinism principle. Nothing is silently lost: the first attempt's error text and diffstat ride in the completion payload (§6).

**Costs nothing extra per walk**: no re-clone of the coordinator's view, no re-dispatch, no second Cloud Run execution — the prior doc's measured justification for the in-container tier ("costs nothing extra: no re-clone, no pull", `retry_chain.go:23`).

### D4 — Cost and loop bounds (stated, not derived at runtime)

The 33 flash agents mean a wrong default is a fleet-wide cost change. The bounds:

1. **Opt-in blast radius: zero until configured.** No agent's behavior changes until an operator writes `fallback_models:` on it. The 37-agent fleet gains nothing it did not ask for. This is the direct answer to "the blast radius is the whole fleet": the default IS no change.
2. **Chain length ≤ 3.** Registry startup validation errors if `fallback_models` has more than 2 entries. The walk tries at most `min(len(chain), walkBudget+1)` links.
3. **At most ONE walk per execution.** The loop budget is 1 fallback attempt, not "walk until success" — a walk that could visit 3 links in one execution is 3× spend on one task. State: `MaxInContainerWalks = 1`.
4. **Executions still ≤ 2** (`MaxTaskExecutions`, unchanged — the persisted cap the stale-task detector enforces). Worst case is the compound one: an execution that walked and then died infrastructure-class gets one re-dispatch, which itself may walk: **≤ 2 executions × ≤ 2 model runs = ≤ 4 model invocations per task, ceiling; typical added cost = exactly 1 extra run**, and only on a transport failure.
5. **Price ceiling at validation time, not runtime.** Every `fallback_models` entry must resolve through `modelreg` (matching `agent_model_name`/`api_name`), and its blended price (`pricing.input_per_1k + output_per_1k` from the row) must be **≤ 3× the pinned model's blended price**. A flash→flagship hop is allowed (that is the point — availability over cheapness at the margin); a flash→$50/M row is a boot error naming the agent and both rows. The ratio is checked once at startup where a human sees it, never in the outage path. Ratio constant `MaxFallbackPriceRatio = 3.0` is config-visible.
6. **Wall-clock bound.** When `fallback_models` is non-empty, the dispatcher scales the Cloud Run Job timeout to `2 × task timeout` (bounded by the 24h Cloud Run max); each attempt still gets the agent's full configured hard/idle timeout. A walk can never run longer than the job it lives in.

### D5 — How the fallback is OBSERVED

Rule: **a run that completed on link 1 is not the same result as one on link 0, and the system must never let you mistake them.**

1. **Populate the fields that exist but are never written**: `TaskCompletion.ModelUsed` and `.ChainLinkIndex` (topics.go:63-64) are set by the job on every publish — success or failure — to the link that produced the terminal outcome. The completion handler already banks `ExecuteResult`; `ModelUsed` flows into the banked row so per-task cost attribution names the model that actually ran.
2. **Per-attempt history on the payload**: a new `Attempts []CompletionAttempt` field on `TaskCompletion`, each `{model, chain_link_index, failure_class, error_msg, duration_ms, cost_usd}`. A two-link run shows both attempts; a no-walk run shows one.
3. **The error message narrates the walk**: on a walked failure, `ErrorMsg` reads `model walk: link 0 openrouter/z-ai/glm-5.3-flash failed (transport: status 503); retried link 1 openrouter/z-ai/glm-5.3, failed (model: ...)`. On a walked success, the completion still carries `Attempts` — "it fell back, and to what" is in the payload, not the logs. This satisfies the repo rule directly: no fallback affecting cost is invisible.
4. **Fell-back runs are tagged**: `ModelUsed != chain[0]` is the tag; the completion-notification payload posted to the agent inbox (which the portal and sidecar poll, `pubsub_completion_handler.go:188`) includes `model_used` and `chain_link_index` verbatim.
5. **Job stdout prints each walk decision** — the existing `COMPLETION_*` stderr / stdout conventions gain `MODEL_WALK|task=...|from=link0/model|to=link1/model|class=transport`.

---

## 4. Solution design

### 4.1 Architecture

```
coordinator (daemon, per-dispatch)                     execute-job (Cloud Run, per-execution)
─────────────────────────────────                      ─────────────────────────────────────────
ResolveModelChain (pin + fallback_models)   ──env──▶   parse AILANG_MODEL_CHAIN
startup validation (harness, price, ≤2)                loop { executeCloudTask(link)
                                                          ClassifyFailure on error
dispatcher sets AILANG_MODEL_CHAIN,                      walk ≤1 on transport }
  scales job timeout ×2                                publish ONE completion with
stale-task detector: UNCHANGED                            ModelUsed, ChainLinkIndex, Attempts
  (still sole re-dispatcher; re-dispatch
   runs the chain from the head)
```

### 4.2 Files to modify/create

| file | change |
|---|---|
| `internal/coordinator/agent_registry.go` | add `FallbackModels []string \`yaml:"fallback_models" json:"fallback_models,omitempty"\`` to `AgentConfig`, adjacent to `Model` |
| `internal/coordinator/retry_chain.go` | `ResolveModelChain` appends `agent.FallbackModels` after the pin; add `MaxInContainerWalks`, `MaxFallbackChainTail = 2`; add `exited with error: signal: killed` to `transportSignatures`; add a test pinning that `exceeded hard timeout` is NOT a signature |
| `internal/coordinator/agent_registry.go` (validation) | new `ValidateFallbackModels(agent)` — resolve each entry in `modelreg`, check `agent_cli` equals the agent's harness, check price ratio ≤ 3.0, check count ≤ 2, no dupes, none equal the pin; loud startup error naming agent + entry on any failure. Empty list = no-op (back-compat). |
| `internal/coordinator/cloud_dispatcher.go` | set `AILANG_MODEL_CHAIN` (comma-joined chain) in the job env alongside `AILANG_MODEL`; when chain length > 1, scale the Job timeout ×2 |
| `cmd/ailang/coordinator_cloud.go` | the walk loop around `executeCloudTask` (per-attempt workdir `attempt{n}`), `ClassifyFailure` call site, `Attempts` population, `ModelUsed`/`ChainLinkIndex` on every `publishCompletion` path including the deferred guard |
| `internal/pubsub/topics.go` | add `Attempts []CompletionAttempt` to `TaskCompletion` |
| `internal/coordinator/pubsub_completion_handler.go` | bank `ModelUsed` into `ExecuteResult`; include `model_used`/`chain_link_index` in the inbox completion notification payload |
| tests | classifier table (incl. the four September messages verbatim — two walk, hard-timeout terminal, git terminal); walk-loop integration test with a fake executor failing once then succeeding; validation tests for harness/price/count; a test that the stale-task detector path is unchanged |

### 4.3 Startup validation detail (the "no known-broken lane" gate)

Validation runs at coordinator startup and at registry reload, per agent with non-empty `fallback_models`:

- **resolvable**: entry matches a `modelreg` row via `agent_model_name` or `api_name`; unresolvable → error (the string would otherwise die at runtime, after a clone, inside an outage).
- **harness-compatible**: the resolved row's `agent_cli` equals the agent's executor (its provider/image-variant resolution). This kills the measured defect where a role tail names `opencode` for a pi-container agent — at boot, where it is a config fix, not an incident.
- **price-bounded**: blended ≤ 3× pin's blended (D4.5).
- **well-formed**: ≤ 2 entries, no duplicates, none equal to the pin.

Loud on first failure, naming agent and entry — the registry already has this pattern (`ResolveRole`'s three loud places; `unresolved_model.go`'s error naming the agent AND the valid options).

### 4.4 What is deliberately NOT changed

- The stale-task detector, `ShouldReDispatch`, `MaxTaskExecutions`, and the V23 single-dispatcher invariant — untouched. An infra-class re-dispatch runs the chain from the head; the execution cap bounds cost.
- `ResolveModel` — still the chain head; `TestChainHeadMatchesResolveModel` keeps guarding.
- Role chains in `models.yml` — left alone; they remain dead config for pinned agents, now harmlessly so (this doc does not adopt them as tails, D1).
- No new task states, no `AttemptID`, no `retry_requested`, no CAS transitions — the single-execution walk needs none of them.

---

## 5. Goals and success metrics

**Primary goal:** a cloud executor task whose pinned model fails with a transport-class error completes on its fallback model within the same Cloud Run execution, visibly.

**Success metrics:**
- `ClassifyFailure` has ≥ 1 non-test call site (currently 0).
- `ModelUsed`/`ChainLinkIndex` populated on 100% of cloud completions (currently 0%).
- A simulated 429 on a two-link agent completes on link 1 with `ChainLinkIndex=1` banked — deterministically testable, no real outage needed (acceptance 4).
- Git-credential and hard-timeout failures still terminate on link 0, provable by test.
- No agent's dispatch behavior changes unless it carries `fallback_models:` (back-compat test over the live registry).

---

## 6. Acceptance criteria

1. An agent with `model: A` and `fallback_models: [B]`, whose executor returns a 503 on A, completes with `ModelUsed=B` and `ChainLinkIndex=1` in the same execution — no re-dispatch, `AttemptCount` still 1.
2. The same scenario with a git-push failure on A terminates failed on link 0 with `ChainLinkIndex=0` and NO second model run — assert via a counter in the fake executor.
3. `pi exceeded hard timeout` and `pi idle for 3m mid-generation` classify differently (terminal vs walk) — table test with the exact September strings.
4. The outage is simulable deterministically: a fake/flag executor mode (or test double) makes link 0 fail transport-class on demand; no acceptance test waits for a real provider outage.
5. Coordinator startup refuses to boot an agent whose `fallback_models` entry is unresolvable, harness-incompatible, or >3× the pin's price — error message names the agent, the entry, and the reason.
6. Every completion payload — success, failed, walked, not-walked — carries `ModelUsed` and `ChainLinkIndex`; a walked failure's `ErrorMsg` names both links and the class of the first failure.
7. Registry back-compat: the 37 live agents, none of which have `fallback_models`, produce byte-identical dispatch env (minus the new `AILANG_MODEL_CHAIN` var, present but equal to the pin) and identical chain heads — no behavior change by default.
8. Worst-case spend is bounded and tested: a task can never exceed 2 executions × 2 model runs; `MaxInContainerWalks=1` is enforced by the loop, not by convention.
9. The stale-task detector's behavior is unchanged by a diff-assertion on its file (or equivalent test): this work must not re-open V23.
10. `make test`, `make lint`, `make check-boundaries` pass.

---

## 7. Non-goals

- Cross-harness fallback (running link 1 in a different container image) — the prior doc's P4. Ruled out: the same-harness constraint is what makes validation a boot-time check and the walk a one-loop change. If a model family ever needs a different harness, the operator pins a fallback the harness can run.
- Making role chains live for pinned agents (D1 — rejected with reasons).
- Local-lane (GPU rig) fallback — the local path has no Cloud Run execution structure; separate concern.
- Changing `MaxTaskExecutions` or the re-dispatch tier.

---

## 8. Review trail

- Builds on the verified §§1–5 of [m-cloud-plane-fallback-lanes.md](m-cloud-plane-fallback-lanes.md) (its own review history records the measurement as "never challenged on substance across seven rounds").
- Adopts that review's four carry-forward constraints verbatim: single application per failure; class re-derivable at the point of decision; no phase ships a known-broken lane (→ startup validation, §4.3); one accounting model (→ one execution, one completion, one `Attempts` list).
- The quota-vs-rate-limit distinction from prior round 3 is deliberately NOT re-introduced as a third runtime class: with per-agent `fallback_models`, the operator picks the tail, and the startup price/harness checks plus the transport default are the whole runtime story. A future doc can add quota-aware skipping if a measured incident demands it.
