# Executor shadowing for AILANG-lane tasks

**Status**: Planned. **Ratified by Mark 2026-10-13** (D1–D7 as written, after quorum rounds 1–2).
**Target**: v0.55.0
**Priority**: P1
**Estimated**: 4–5 days (~3 days implementation, 1 day grading and report, 1 day verification + buffer)
**Dependencies**: v0.54.8 (motoko image with a warm compile cache, cache-stable compaction, and OpenRouter routing parity with pi); a dedicated, capped OpenRouter key for shadow runs.
**Quorum**: triggers 1 (design-freeze items D1–D4) and 3 (new banked-data schema: shadow task records and `ShadowComparison`) fire, so run `ailang design-quorum` before sprint planning. Round 1 (2026-10-13): gemini-3-1-pro rejected on unverified API claims; rows V17–V25 were added, and the observatory tagging premise turned out false and was redesigned. Round 2: gemini-3-1-pro rejected on the grader running `make test` in the control plane without a hard timeout; grading moved to a policy-confined, time-bounded Cloud Run job. gpt6-1-sol was unreachable both rounds (N−1). The re-quorum-once guardrail was used up; **Mark ratified D1–D7 and this revision on 2026-10-13.**

## Related Documents

- [m-exec-multi-executor-support](../../implemented/v0_6_1/m-exec-multi-executor-support.md): the executor abstraction this builds on (neural 0.43, distinct: one executor per task).
- [m-motoko-executor-adapter](../../implemented/v0_18_4/m-motoko-executor-adapter.md): the motoko executor (0.39).
- [m-completion-path-parity](../m-completion-path-parity.md): the completion path that shadow finalization branches from.
- [HANDOVER-m-motoko-ailang-only-lane](../HANDOVER-m-motoko-ailang-only-lane.md): the pi↔motoko lane comparison whose manual process this automates.
- Search 2026-10-13 (`ailang docs search --neural`, three queries): no planned or implemented doc ≥ 0.45, so this is not a duplicate.

## Problem Statement

We have two harnesses that can run the `ailang_only` lane, pi and motoko. Deciding which writes better AILANG on **real work** (Mark, 2026-10-02: "we will see it run on real tasks over time") currently needs a person to:

- pick real tasks by hand;
- write an acceptance check for each;
- rerun both harnesses side by side.

The 2026-10-12/13 comparison did exactly that for six tasks. It found **nine harness defects**, and fixing them took motoko on GLM-5.3-Flash from 2/6 to parity with pi:

- no AILANG teaching prompt in the lane;
- system role dropped on Anthropic and Gemini;
- dead repetition guard;
- cost cap pricing cached tokens wrong;
- no session affinity;
- host-pool mismatch;
- compaction summary busting the prompt cache;
- a summariser sent the agent's tools;
- a regression in compile-cache size.

None of these defects showed up in benchmarks; all of them showed up on real tasks. But six hand-graded tasks is a snapshot, and building it cost a day of work.

We want this to run continuously: every real AILANG-lane task also runs, unseen, on the other harness, and the gaps surface themselves.

## Goals

**Primary goal:** for a configurable sample of real AILANG-lane coordinator tasks, run the same task on a second executor with no side effects. Record both outcomes and grade them, so a weekly report shows where the harnesses differ and why.

**Success metrics:**
- **Coverage:** at least 80% of eligible tasks in a week get a completed shadow run.
- **Isolation:** zero pushes, PRs, approval cards or handoffs from shadow runs, enforced and tested.
- **Cost:** a weekly spend cap, enforced through a dedicated key; the expected cost is about $5–10 a week at today's volume (about 10 eligible tasks a week).
- **Signal:** every task where one harness passes and the other fails is listed with links to both transcripts and both diffs.

## High-Impact Decisions

| # | Decision | Default | Why |
|---|---|---|---|
| D1 | How a shadow is represented | A **child task** `<primary>-shadow` with a new field **`ShadowOf=<primary id>`** | Every pipeline stage is keyed by task ID: directive store, dedup, branch name, the 2-execution cap, completion. Reusing the primary's ID would collide everywhere. **Not** `Kind` (V7: `Kind=="question"` changes execution mode) and **not** `ParentTaskID` (V8: it links handoff chains in the observatory); both would change the meaning of existing fields. |
| D2 | Which executor shadows | **The other lane harness**: pi agents are shadowed by motoko, motoko agents by pi | `ailang-only-executor` already runs motoko, and the `pkg:*` agents run pi, so the comparison works both ways. |
| D3 | Model | **Same as the primary**, overridable per agent | Measures the harness, not the model. A `model:` override allows a "Sol 6.1 on subscription" arm later at no marginal cost. |
| D4 | What a shadow may write | **Nothing outside its own run directory**: no push, no PR, no approval card, no handoff | Shadowing must be invisible to the fleet. |
| D5 | Picking the better fix | **Report only** by default; `shadow.offer_alternative: true` opts an agent in to a draft PR when the shadow wins and the primary failed | Avoids surprise PRs; Mark decides when to trust it. |
| D6 | Grading | Three layers, cheapest first (see Grading) | Real tasks have no acceptance checks. |
| D7 | Budget isolation | A separate OpenRouter key (`OPENROUTER_API_KEY_SHADOW`) with its own monthly cap, plus a per-shadow `max_cost_usd` | On 2026-10-13 a local test key hit its $100 cap mid-run; shadows must never drain the fleet's key. |

### Design Freeze
D1–D4 are frozen once this doc is approved. D5–D7 can change per agent through config.

## Solution Design

### Overview

```
message → primary task (unchanged) ─────────────────────────────────▶ PR / approval (unchanged)
              │ dispatch hook: agent has shadow.enabled and sample hit
              └─▶ child task <id>-shadow, ShadowOf set, executor = shadow.provider
                     │ Cloud Run job (variant job), SHADOW=1:
                     │   no write-key preflight, no push, no PR, no git credential to the agent
                     └─▶ TaskCompletion (diff, metrics, transcript path)
                            └─▶ FinalizeTaskCompletion: store on the child record; no card, no handoff
grader (on child completion, and again when the parent's PR settles) ─▶ ShadowComparison record
weekly: `ailang coordinator shadow report` ─▶ markdown/JSON + message to the user inbox
```

### Configuration (agent registry)

A new optional block on `AgentConfig` (`internal/coordinator/agent_registry.go:102-292`), validated alongside `ValidateAgentProviders` (`dispatch_provider.go:123`):

```yaml
- id: pkg-sunholo-ailang-parse
  provider: pi
  tool_policy: ailang_only
  shadow:
    enabled: true
    provider: motoko             # required; must differ from provider
    executor_variant: motoko
    model: ""                    # "" = the primary's model (D3)
    sample_rate: 1.0             # 0..1, per eligible task
    max_cost_usd: 1.00           # per shadow run
    offer_alternative: false     # D5
```

Validation, added to `ValidateAgentProviders` (`dispatch_provider.go:123`, V19):
- `shadow` is only allowed when `tool_policy: ailang_only`, at least for v1;
- `provider` must differ from the primary's provider;
- the variant and provider must agree, using the same rule as the cloud dispatcher's `checkVariantProviderAgreement` (`internal/dispatch/cloudrun/dispatcher.go:146`, V20);
- `sample_rate` must be in [0, 1].

### Dispatch

- **Cloud:** in `dispatchTasksCloud` (`daemon_tasks_exec.go:101`), after the primary is claimed (`MarkTaskQueued`, :123), check the shadow config and the sample. If both pass:
  1. create the child task (`ShadowOf=<primary>`, the same directive, the same agent tool policy and policy file);
  2. build `DispatchParams` with the shadow provider, variant and model, plus `Shadow: true`;
  3. dispatch through the same `cloudDispatcher.Dispatch` (`cloud_dispatcher.go:382`) to the variant's job.
- **Local:** `executeTask` (`daemon_tasks_exec_run.go:26`). Local shadowing is **off in v1**: worktree slots (`MaxWorktrees` = 3) would compete with real work.
- **Base commit:** both jobs clone `--branch <base> --depth 1`, i.e. the branch tip at clone time (V10). Pinning isn't free, because the primary's base is only known after its own clone. v1 therefore **records** each run's `BaseCommit` (already on `TaskCompletion`, V11) and the comparison marks `base_drift` when they differ. Both jobs are dispatched in the same tick, so drift needs a merge to land within seconds; the report counts it. Pinning (dispatch-time `ls-remote`, passed to both jobs) is deferred unless drift is observed.

### The no-side-effects guarantee (D4)

`Shadow` travels as an env var next to `config.AcknowledgeOnly()` (`internal/config/job.go:144`). In `executeCloudTask` (`cmd/ailang/coordinator_cloud_task.go`), when it's set:
- **skip** the deploy-key write preflight (~:140), the push (step 5c, ~:405) and `openCascadePullRequest` (step 5d, ~:437);
- **still** run `collectGitEvidence` (~:449) and `writeTaskArtifacts`, so the diff and transcript are kept;
- **withhold** `childGitCredential` (`coordinator_cloud_executor.go:418`), so the agent can't push even if it tries.

At finalization, `wantsApproval` and `wantsHandoff` (`task_finalize.go:374/388`, V4) return false when `ShadowOf != ""`. The landed-card sweep only visits tasks with pending cards, so shadows, having none, are skipped without a change (V12).

### Storage

- **The child task record:** the existing store (Firestore in cloud) holds the shadow's diff (capped, as for primaries), cost, tokens, turns, finish reason, session ID, transcript artifact path and executor version.
- **Not in the observatory's chain metrics.** `applyMetrics` writes stage metrics only when `Task.StageID` is set and recomputes chain aggregates only when `Task.ChainID` is set (V18). `SetStageMetrics` has no tag parameter (V17), so a shadow can't be marked as such there. And if it shared the primary's chain, `RecomputeChainAggregates` would add its cost to the primary's chain total (V18). The shadow child is therefore created **without** `StageID` and `ChainID`, `applyMetrics` is a no-op for it, and chain costs, `ailang chains` and fleet cost views are unchanged. Shadow cost and tokens are read from the child record and `ShadowComparison` only.
- **A new `ShadowComparison` record**, keyed by the parent task ID, holds the grades and the verdict (see Grading).

### Grading (D6)

Graders run when the child completes, and again when the parent's PR settles:

1. **Mechanical**, on child completion, **never in the coordinator daemon**. The coordinator dispatches a **grade job**: a Cloud Run Job on the agent executor image, with no git credential and no push. The job applies each diff to the run's base commit in a scratch clone and runs **only AILANG gates under the agent's own restricted policy**:
   - `ailang check` on changed modules;
   - `ailang test --package` on changed packages, via `ailang policy-tool` (the same confinement the lane's tools use).

   Each gate has a hard timeout (default 300 s), and the job has a wall-clock bound (900 s). A timeout is recorded as `ungraded(timeout)`, not as a failure of either harness. Repo `make` targets are **not** run in v1, because they would execute arbitrary repo code outside the AILANG policy.
2. **Against the merged fix**, when the parent's PR settles. `settledPRLookup` (`daemon_landed_cards.go:50`, which points at `lookupSettledPRREST`, :176), called on the parent's branch `BranchForTask(parent)` (`pr_reconcile.go:178`), returns `LandedPR{Number, State, MergedAt, MergedBy}`, with no merge SHA (V21). Fetch its diff and record:
   - whether each candidate touches the same files and functions;
   - whether the merged PR's own new or changed tests pass against each candidate.

   A merged PR that came from the primary itself is noted as such, since that grade is then circular. The merged diff is fetched with the PR number (GitHub `pulls/{n}/files`), because `LandedPR` carries no SHA.
3. **Review**, only when (1) and (2) disagree or tie. An off-Anthropic reviewer, in the `design-quorum` style and reject-by-default, compares the two diffs against the task directive. It must name a winner, give its strongest objection to that winner, and give a confidence. Reviewer cost counts against the shadow budget.

`ShadowComparison.verdict` is one of:
- `primary_better`;
- `shadow_better`;
- `equivalent`;
- `both_failed`;
- `ungraded` (with a reason).

### Weekly report

A new command, `ailang coordinator shadow report [--since 7d] [--json]`, with a scheduled weekly run that posts to the `user` inbox. It contains:

- per-agent and per-harness pass rates, cost, wall time and turns;
- **every disagreement**, i.e. one harness passed and the other didn't, with both transcript links (`ailang chains chat`), both diffs and the reviewer note;
- running totals since the feature was enabled, so a pattern over weeks is visible (the "over the next few weeks" ask).

### Offering the alternative (D5, opt-in)

When `offer_alternative: true`, the verdict is `shadow_better`, and the primary failed the mechanical gate, the grader opens a **draft** PR from a `coordinator/<parent>-shadow` branch. The PR body explains it's a shadow alternative and links both runs. It never auto-merges.

## Files to Modify/Create

| File | Change |
|---|---|
| `internal/coordinator/agent_registry.go` | `ShadowConfig` on `AgentConfig`; validation |
| `internal/coordinator/daemon_tasks_exec.go` | Shadow dispatch hook (cloud) |
| `internal/coordinator/provider.go`, `store.go`, `internal/storage/firestore/coordinator_tasks.go` | `ShadowOf` on the task and its record; `ShadowComparison` collection |
| `internal/coordinator/task_finalize.go` | No approval card or handoff when `ShadowOf != ""` (`applyMetrics` needs no change: shadows have no StageID/ChainID) |
| `internal/coordinator/cloud_dispatcher.go` | `DispatchParams.Shadow` (env to the job) |
| `internal/config/job.go` | `AILANG_TASK_SHADOW` (registry var) |
| `cmd/ailang/coordinator_cloud_task.go`, `coordinator_cloud_executor.go` | No-side-effects gates; withhold the git credential |
| `internal/coordinator/shadow_grade.go` (new) | Dispatches the grade job; merged-fix comparison; reviewer call (all bounded) |
| `cmd/ailang/coordinator_shadow_grade_job.go` (new) | The grade job: apply diffs, run policy-confined `ailang check`/`test` with timeouts, report back |
| `cmd/ailang/coordinator_shadow.go` (new) | `shadow report` |
| `ailang-multivac/config/config.cloud.yaml` | Enable on two agents first (see Rollout) |

## Success Criteria

- [ ] A sampled `pkg:*` task produces a primary PR and a completed shadow child with diff, metrics and transcript, and **no** extra PR, push, card or handoff.
- [ ] A shadow run whose agent tries `git push` fails to push, because no credential is provided.
- [ ] The mechanical grade runs on both diffs and is stored.
- [ ] When the parent's PR merges, the merged-fix comparison runs and updates the verdict.
- [ ] `ailang coordinator shadow report --since 7d` lists every disagreement with links.
- [ ] Shadow spend goes through the shadow key and stays under its cap; exceeding the per-run `max_cost_usd` stops the run with `cost_exhausted`, banked, not retried.

## Testing Strategy

- **Unit:**
  - config validation;
  - the sampling decision;
  - `wantsApproval` / `wantsHandoff` for `ShadowOf != ""`;
  - the dispatch-parameter mapping;
  - the grading verdict table.
- **Integration (fake dispatcher):** a primary plus its shadow, where the shadow's completion produces no card and the comparison record is created.
- **Cloud smoke on the dev plane:** enable on one agent and send one known task, then check that no PR exists for the `-shadow` branch and that the transcript artifact is present.
- **Negative test:** the shadow job, given a write key by mistake, still doesn't push (the gate runs before the push step).
- **Mutation-test every guard** (revert it and watch its test fail), per the repo rule.

## Rollout

1. Dev plane: `ailang-only-executor` (motoko primary, pi shadow) and `pkg-sunholo-ailang-parse` (pi primary, motoko shadow), `sample_rate: 1.0`, for one week.
2. Review the first weekly report with Mark, and tune the graders.
3. Prod: enable on all `ailang_only` agents.

## Non-Goals

- Shadowing non-AILANG-lane agents (sprint-executor, design-doc-creator, codex/claude lanes). v1 is the `ailang_only` lane only.
- Local daemon shadowing (worktree contention).
- Automatic switching of an agent's executor based on shadow results. The report informs Mark's decision (the fleet switch rule); it doesn't make it.
- More than one shadow per task.

## Deferred Decisions

- A third arm (for example Sol 6.1 on the ChatGPT subscription) once v1 is stable.
- Whether a reviewer-only verdict (with no mechanical or merged-fix signal) should count in pass rates, or be reported separately.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A shadow accidentally pushes or opens a PR | Three independent gates: env flag, skipped push/PR steps, withheld git credential. Plus a negative test and a dev-plane smoke check. |
| Shadow spend drains the fleet's budget | A dedicated key with its own cap (D7), and a per-run `max_cost_usd`. |
| The base commit drifts, so the shadow starts from different code | Both base commits are recorded; drift is flagged and counted in the report (pinning deferred). |
| The grade is circular (the merged fix *is* the primary's PR) | Recorded as such; that comparison weighs only the mechanical and reviewer layers. |
| The two harnesses land on different OpenRouter hosts | Routing parity (`openrouter_provider`, v0.54.8). The report records the serving host from the provider trace. |
| Grading executes repo code | Grading runs only AILANG gates, under the agent's restricted policy, in a separate Cloud Run job with per-gate and job timeouts. Nothing runs in the coordinator daemon; `make` targets are excluded in v1. |
| Doubling cloud executions hits job quotas | Variant jobs have separate quotas, and v1 volume is about 10 extra runs a week. |

## Verification Log

Every load-bearing claim, checked against the code on 2026-10-13 (`dev` at `45f483efd`).

| # | Claim | Evidence | Result |
|---|---|---|---|
| V1 | Cloud dispatch claims a task with `MarkTaskQueued`, then dispatches through `cloudDispatcher.Dispatch` | `daemon_tasks_exec.go:101-123`, `store.go:230`, `cloud_dispatcher.go:382` | Confirmed |
| V2 | Cloud dispatch batches at most 5 tasks per tick | `daemon_tasks_exec.go:106` `Limit: 5` | Confirmed |
| V3 | The job clones, pushes and opens a PR in fixed steps inside `executeCloudTask` | `cmd/ailang/coordinator_cloud_task.go:128` (clone), ~:405 (push), ~:437 (PR) | Confirmed |
| V4 | Approval cards and handoffs are decided by `finalizer.wantsApproval` / `wantsHandoff` | `task_finalize.go:374`, `:388` | Confirmed |
| V5 | The agent's git push credential comes from one function | `cmd/ailang/coordinator_cloud_executor.go:418` `childGitCredential` | Confirmed |
| V6 | A job-level boolean env switch already exists to follow (`AcknowledgeOnly`) | `internal/config/job.go:144` | Confirmed |
| V7 | `Task.Kind` is taken and changes execution | `provider.go:30` ("directive" or "question"); `provider_executor.go:141` branches on `Kind == "question"` | Confirmed, so not reused |
| V8 | `ParentTaskID` is taken by handoff chains | `provider.go:33`; `observatory_sync.go:61` | Confirmed, so not reused |
| V9 | **No** shadow or variant config exists on `AgentConfig` today | `grep -in shadow internal/coordinator/agent_registry.go` → no hits | Confirmed (negative) |
| V10 | The job clones the branch tip, not a pinned commit | `coordinator_cloud_task.go:128` `clone --branch baseBranch --depth 1` | Confirmed |
| V11 | Completions already carry `BaseCommit` | `pubsub_completion_handler.go:215`; `task_finalize_cloud_strategy.go:29` | Confirmed |
| V12 | The landed-card sweep only visits tasks in `PendingApproval` | `daemon_landed_cards.go:76` `ListTasks(... Status: []TaskStatus{TaskStatusPendingApproval})`. **Corrected 2026-10-19 (M4):** `wantsApproval` being false does not keep a shadow out of that state; `nextTaskStatus` sets `PendingApproval` independently for any completed, non-skip-approval task. M4 makes a completed shadow `Completed`, with a test. | Confirmed (read); premise corrected in M4 |
| V13 | **No** existing re-run or A/B path would conflict: `ClassifyFailure` has no non-test callers | `grep -rn "ClassifyFailure(" internal cmd` → only the definition (`retry_chain.go:70`) and tests | Confirmed (negative) |
| V14 | Each executor variant has its own Cloud Run job, so shadows draw on separate job quotas | `internal/dispatch/cloudrun/dispatcher.go:182` `jobSuffixForVariant` | Confirmed |
| V15 | Eligible volume is about 10 tasks a week | 300 most recent prod tasks (09-23 to 10-12): `pkg-*` AILANG agents 32, `ailang-only-executor` 0 | Measured; re-measure at rollout |
| V16 | A motoko GLM-Flash run costs about $0.2–1.0 with the v0.54.8 fixes | Real-task reruns 2026-10-12/13 (OpenRouter traces) | Measured; noisy (n = 6 tasks) |
| V17 | **No** tag or label parameter exists on observatory stage metrics | `internal/observatory/backend.go:160` `SetStageMetrics(ctx, stageID, cost, tokensIn, tokensOut, turns, toolCalls, durationMs, costProvenance)` | Confirmed (negative), so the design does not tag; it keeps shadows out |
| V18 | `applyMetrics` writes only for a set `StageID` and recomputes aggregates only for a set `ChainID` | `internal/coordinator/task_finalize.go:311-330` (`if f.in.Task.StageID != ""`, `if f.in.Task.ChainID != ""`) | Confirmed |
| V19 | Agent provider validation has a single entry point to extend | `internal/coordinator/dispatch_provider.go:123` `func ValidateAgentProviders(agents []*AgentConfig) []error` | Confirmed |
| V20 | The variant/provider agreement rule exists, in the dispatcher | `internal/dispatch/cloudrun/dispatcher.go:146` `func checkVariantProviderAgreement(variant, provider string) error` | Confirmed; the registry check calls it or mirrors it |
| V22 | The dispatch parameters live in one struct to extend with `Shadow` | `internal/coordinator/cloud_dispatcher.go:32` `type DispatchParams struct` | Confirmed |
| V23 | The cloud job's write and evidence steps are separate named functions, so each can be gated independently | `cmd/ailang/coordinator_cloud_sshkey.go:203` `verifyDeployKeyWrite`; `cmd/ailang/coordinator_cloud_github.go:36` `openCascadePullRequest`; `cmd/ailang/coordinator_cloud_evidence.go:50` `collectGitEvidence`; `cmd/ailang/coordinator_cloud_github.go:505` `writeTaskArtifacts` | Confirmed |
| V24 | Local worktree slots default to 3 (why local shadowing is off in v1) | `internal/coordinator/daemon.go:47/54` `MaxWorktrees: 3` | Confirmed |
| V25 | Local execution and cloud dispatch have single entry points | `internal/coordinator/daemon_tasks_exec_run.go:26` `executeTask`; `daemon_tasks_exec.go:101` `dispatchTasksCloud` | Confirmed |
| V21 | A settled PR can be found from a task's branch, without a merge SHA | `internal/coordinator/daemon_landed_cards.go:50` (`var settledPRLookup = lookupSettledPRREST`), `:176`; `pr_reconcile.go:178` `BranchForTask`; `LandedPR{Number, State, MergedAt, MergedBy}` (`pr_landed.go:69`) | Confirmed |

## Conflict Surface

This design adds no language syntax. It does extend **shared coordinator machinery**. Each row below shows what else lives there, and whether the design reuses or overrides it:

| Position | Existing users | Decision |
|---|---|---|
| Task identity and record schema | Every pipeline stage (dedup, directive store, branch naming, 2-execution cap) | **Reuse** IDs; **add** `ShadowOf`. `Kind` and `ParentTaskID` are left untouched (V7, V8). |
| `finalizer.wantsApproval` / `wantsHandoff` | Every completed task | **Override** for `ShadowOf != ""` only, with a test that a non-shadow task's behaviour is unchanged. |
| `executeCloudTask` push and PR steps | Every cloud task | **Override** behind the `Shadow` env flag (default off). Non-shadow jobs take the same path as today, guarded by a regression test. |
| Observatory stage metrics, chain aggregates, `ailang chains` | Cost dashboards, eval/fleet reports | **Not touched.** Shadow children carry no `StageID`/`ChainID`, so `applyMetrics` skips them (V17, V18) and fleet cost views are unchanged by construction. |
| Cloud dispatch batch (5 per tick) | All cloud agents | **Reuse**. Shadows count against the batch, so at most +1 per eligible primary; v1 volume is about 10 a week. |

**Programs that must still work:**
- a normal `pkg:*` task, with PR, card and handoff unchanged;
- a `question`-kind task;
- an acknowledge-only job;
- a handoff chain, whose `ParentTaskID` links must stay untouched.

## Axiom Compliance

This is coordinator tooling, not a language feature, so most axioms are untouched.

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Sampling uses a recorded per-task decision (stored on the child) and is replayable from the record. |
| A2: Replayability | +1 | Both runs' transcripts, diffs and base commits are banked, so any verdict can be re-derived. |
| A3: Effect Legibility | +1 | A shadow's side effects are confined and **declared**: three gates and a withheld credential. |
| A4: Explicit Authority | +1 | The shadow is denied write authority explicitly; there is no ambient push. |
| A5: Bounded Verification | 0 | No impact. |
| A6: Safe Concurrency | 0 | Separate task IDs and variant jobs; no shared mutable state with the primary. |
| A7: Machines First | +1 | Grading is mechanical first (repo gates, merged-fix tests), with a model reviewer only on ties. |
| A8: Minimal Syntax | 0 | No syntax. |
| A9: Cost Visibility | +1 | Per-run `max_cost_usd`, a dedicated capped key, and shadow cost kept out of fleet chain totals (V18) and reported separately. |
| A10: Composability | +1 | Reuses the dispatcher, executors, completion path and evidence collection. |
| A11: Structured Failure | +1 | Verdicts are a closed set, including `ungraded` with a reason and `base_drift`. |
| A12: System Boundary | 0 | No boundary change. |

**Net score: +8.** Hard-violation check: A1, A3, A4 and A7 are all non-negative.
