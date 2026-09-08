# M-COMPLETION-PATH-PARITY: a cloud task must end the same way a local task does

**Status**: **M0, M0b, M1 (cloud half), M2, M3, M4, M5 LANDED on `dev` 2026-09-03.** Not yet deployed — this is a coordinator-image change and needs the dev → test → prod rollout. M1b (daemon consolidation) deliberately outstanding; see the sprint plan. Originally planned — attended, standalone critical-infrastructure work, sibling to [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md). Written 2026-09-03 after a live end-to-end trial (`task-88a9fa95`) showed the agent-to-agent handoff chain has **never fired in production**.
**Revision**: **r14** (last completed full quorum: round 12 — blocked 3/3; rounds 0–12 blocked, one pass recorded in round 10)
**Rulings**: **D2, D3 and D5 ruled by Mark 2026-09-03 (attended). Design freeze CLOSED; sprint may proceed.** D1 and D4 recorded below. See [the sprint plan](m-completion-path-parity-sprint-plan.md). — rounds 0–5 each blocked 3/3. Every objection accepted; none argued. Round 5 landed the most embarrassing finding of the series: **r6 verified the SQLite stores — the daemon path nobody runs — and asserted the Firestore ones, on a doc whose entire thesis is that the cloud path is the only one that matters.** Checked properly, Firestore differs from SQLite on two of three effects, and differs in a way that is **worse**, not better (V31fs, V34fs): it overwrites silently, so a redelivered completion would reset an already-resolved approval to `pending`.
**Scope**: The coordinator has **two** task-completion paths. The daemon path performs ten side effects; the cloud path — the only one prod uses — performs two. Everything in the difference is dead in prod: agent handoffs, chain/stage progression, GitHub stage routing, and the approval record itself.
**Target**: v0.35.0
**Priority**: **P0.** This is the load-bearing link for unsupervised AILANG package-ecosystem work. The pipeline is designed, configured and coded — and has produced zero handoffs in four months.
**Estimated**: ~5 days (M1–M3). 2d→3d→5d→4d→**5d**: r6 adds three idempotent store primitives (`CreateApprovalIfAbsent`, `PutMessageIfAbsent`, absolute metric setters) in both backends, because V31/V32/V34 showed none of them exists today. + ~0.5d reconciliation + rollout
**Dependencies**: none blocking. Builds on [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) M1a–M8, which made dispatch and execution trustworthy. This doc fixes the layer **after** execution.

---

## Corrections to r1 (recorded, not quietly edited)

r1 was blocked 3/3. Two of its load-bearing claims did not survive the work the objections demanded:

1. **r1 claimed "0 of 404 chains has ever had a stage row."** **False.** The chain-detail endpoint
   shows `task-88a9fa95`'s stage exists: `3e88a511`, `stage_number: 1`, `agent_id:
   design-doc-creator`. Stages **are** created, on both paths. The true defect is narrower and
   sharper: **the stage is created and then never updated again** — `status: "pending"`, all metrics
   zero, twelve hours after the task finished and opened a PR (V15).
2. **r1 claimed A6 "no concurrency change."** **Unsupported**, as gpt5-6-sol said. Pub/Sub push is
   at-least-once with a 60s ack deadline and 10s minimum backoff (V22), and the existing terminal-state
   guard does not cover `pending_approval` (V23) — so concurrent and repeated finalisation is a live
   case, not a hypothetical.

3. **r2 claimed the `ExecutionStrategy` cure for silent fallbacks was "every strategy implements
   every method."** **False by construction**, as oc-glm-5-2 said: a `Cleanup` whose cloud body is
   `return nil` is structurally identical to the empty-string branching r2 withdrew — the caller
   cannot tell "nothing to clean up" from "cleanup silently skipped". r2's Hard Violation Check
   ("Silent fallback introduced? No") was therefore wrong. Fixed in D1 by branching explicitly on
   `strategy.Kind()`, and the check is restated honestly below.
4. **r2 asserted the cloud artifact source is available because "the payload already carries it."**
   That was a code-shape claim standing in for a data claim. `cmd/ailang/coordinator_cloud.go:110`
   documents the opposite in its own comment: *"artifactPath is the GCS path prefix where raw
   artifacts were uploaded (**may be empty**)"* (V27).

5. **r3 claimed each effect could commit "atomically with its own ledger row" in one
   `RunTransaction`.** **False.** Co-location is not transaction participation, as gpt5-6-sol said.
   No store method accepts a caller-supplied transaction: `CreateApprovalRequest(ctx, req) error`,
   `UpdateStageStatus(ctx, stageID, status) error`, `UpdateChainMetrics(ctx, id, ...) error` (V30).
   Those atomic units **do not exist** and must be built.
6. **r3 claimed effect 10's external effect is "a GitHub label write" that is "separately
   idempotent".** **False**, as gemini-3-1-pro and oc-glm-5-2 both said. `ProcessStageCompletion`
   reaches `TaskChain`, which makes **twelve `PostCommentInRepo` calls with no dedup and no marker**
   (V29). A comment POST duplicates on retry. r3's claim-after-effect protocol was unsafe, and its
   "no outbox" stance contradicted its own Risks table.

7. **r4's V28 claimed the daemon path is single-store.** **False**, as oc-glm-5-2 said — and it was
   asserted at the same confidence as V25 while carrying only a directory-name inference.
   `internal/storage/backend.go NewSQLiteBackends` opens **three separate database files**:
   `coordinator.db`, `collaboration.db`, `observatory.db`, each its own `*sql.DB` (V28). No `*sql.Tx`
   spans them. The SQLite half of r4's `FinalizationTx` was impossible as specified.
8. **r4 tried to make effect 10 safe by growing the protocol** (outbox → marker probe → per-operation
   rows → leases). gpt5-6-sol was right that one ledger row and one marker cannot cover twelve
   non-idempotent POSTs, and gemini-3-1-pro was right that none of that work appeared in any
   milestone or file list. r5 removes the cause instead: **effect 10 leaves scope** (V29b — it is
   already inert for every task this doc exists to enable).

9. **r5 asserted four per-effect idempotency properties without verifying any of them.** All three
   reviewers flagged it; **three of the four were false.** Approval creation and message insertion are
   bare `INSERT`s that raise on a duplicate id, and stage *and* chain metrics both **increment**
   (V31, V32, V34). Only the status writes were as claimed (V33). r5's protocol would have crash-looped
   on approvals and double-counted every retried cost.

10. **r6 verified the wrong backend.** V31/V32/V34 cited only `store_sqlite_approvals.go`,
    `store_chains.go` and `internal/messaging/inbox.go`. The cloud path uses Firestore. oc-glm-5-2
    caught it, and the correction matters: Firestore's approval and message writes use
    `Doc(id).Set(...)`, which does **not** error on a duplicate — it **silently overwrites**, so the
    danger is not a crash loop but a resolved approval being reset to `pending` (V31fs, V34fs).
    Metrics, however, double-count identically on both backends (V32fs).
11. **r6's M3 diff source was replay-unsafe.** `TaskCompletion` carries `BranchName` but **no commit
    SHA** (V35), and a branch can move or be deleted between attempts, so `base_commit..branch_name`
    is not a deterministic input — gpt5-6-sol's point.

12. **r7 said `base_commit` was "already recorded at dispatch".** **False, and worse than the
    objection assumed.** `BaseCommit` is worktree-derived (`worktree.go:138` →
    `daemon_tasks_exec_run.go:410`) and its only persistence is `MarkTaskPendingApproval` — which the
    cloud path calls with `""` (`pubsub_completion_handler.go:132`). **Every cloud task has an empty
    `base_commit`** (V37). gemini-3-1-pro caught the unverified premise; the check showed the field is
    not merely unverified but reliably absent on the path M3 targets.
13. **r7's Files list named a file the V-log never inspected.** `PutMessageIfAbsent` was pointed at
    `observatory_messages.go`, while V34fs cites `messaging_inbox.go`. oc-glm-5-2 caught it; V38
    records what each file actually is. Clerical, but the same class as correction 10 — the plan must
    cite the code that was read.

14. **r8 had the wrong component computing the diff.** The cloud coordinator is a Cloud Run service
    with no clone and no worktree — it passes `""` for `worktreePath` (V37) — so it cannot run
    `git diff` at all. gemini-3-1-pro named the entity that can: **the executor**, which already does
    (V39). M3 is now smaller: the executor publishes the diff, the coordinator reads it.
15. **r8's `SetChainMetrics` was a read-then-write race.** Deriving `SUM(stage.cost)` and then writing
    the chain total in a separate operation lets two finalizers compute different snapshots, with the
    stale writer landing last — gpt5-6-sol's point, and it would have broken the very convergence A6
    claims. Fixed by making the recompute atomic on both backends (C1).
16. **r8 advanced its revision label past the last completed full-quorum round.** Round 6 was
    degraded to N-1 on budget, and r8 was submitted before round 7 ran. oc-glm-5-2 was right to
    refuse it. Round 7 has since run at a raised cap with all three reviewers present; the header now
    states which round last completed, and the counter will not run ahead of it again.

17. **r9 said `HeadCommit` was "captured at push".** **False** — the push is
    `exec.Command("git", "-C", workDir, "push", "origin", branchName)` and the success path returns
    without ever asking for a SHA (V40). Same class as corrections 4 and 12: a convenience assumed
    into existence. M3 now adds an explicit `git rev-parse HEAD`, and the push file joins the list.
18. **r9's ledger had no verified home.** The `tasks` table is column-per-field, extended by
    `ALTER TABLE tasks ADD COLUMN` (V41); nothing accommodates `finalization.<effect>`. M1 carries a
    migration. oc-glm-5-2 caught it.
19. **r9's SQLite recompute covered one metric of three.** V32 named `total_cost`, `total_tokens` and
    `total_turns`; the SQL summed only cost, leaving the other two on the unsafe increment path —
    gemini-3-1-pro's catch, and it would have half-fixed the bug.

20. **r10 said the orchestrator performs effects "1–9 unconditionally". Wrong, and dangerously so.**
    The daemon's effects sit inside `if <success> { … } else { <failed path> }`, with a nested
    `skipApproval` branch (V43). `handleAgentHandoffs` is in the **success** arm. Extracting them as
    unconditional would **dispatch a handoff off a failed task** — starting sprint-planner from a
    design run that errored. gpt5-6-sol caught it; D1 now carries an outcome × effect matrix.
21. **r10 said the executor "already computes the diff".** It computes a **file list**:
    `discoverChangedFilesFromCommit` returns `[]string` from `ArtifactDiscovery.DiscoverChangedFiles()`
    (V42). A diffstat and a patch are different git invocations. M3 does **not** shrink; oc-glm-5-2
    caught the conflation.
22. **r10 said Firestore "needs no migration" for the ledger.** True of Firestore, false of the Go
    layer: `TaskRecord` is populated by a hand-written map converter (`coordinator_convert.go:40,114`),
    so a field absent from it is **silently dropped on read** — the orchestrator would see an empty
    ledger on every redelivery and re-run every effect, destroying the protocol it exists to provide.
    gemini-3-1-pro caught it.

23. **r11's matrix deferred auto-edge handoffs to approval. Wrong.** `handleAgentHandoffs` sits at
    `daemon_tasks_exec_run.go:633` — **after** the `if skipApproval {…} else {…}` block closes at
    `:629` and **inside** the success arm that closes at `:637` (V45). It therefore fires at
    *completion* for normal completions too, not on approval. r11 claimed "column-for-column"
    behaviour preservation and then silently changed a column. gemini-3-1-pro and oc-glm-5-2 both
    caught it.

24. **r12's M2 contradicted its own C1.** M2 said "cost routes through the single writer identified
    in V19… no double-count risk" — but V32 proved *that* writer increments. M2 must use the absolute
    setters C1 introduces. gemini-3-1-pro caught the doc arguing against itself.
25. **r12's Goals overclaimed parity.** "Identical completion effects" is false while effect 10 is
    deliberately daemon-only. V29b bounds the *consequence*, not the claim. gpt5-6-sol was right that
    a scoping decision cannot be laundered into a goal; the goal is now stated with its exception.

26. **r13's "absolute writes are idempotent" was too strong.** An absolute write is *repeatable*,
    but not idempotent once another step has legitimately advanced the record: a stale replay could
    write `pending_approval` over a task the approval processor has already approved, regressing it.
    gpt5-6-sol found the hole in C1's core safety claim. Status writes are now **conditional and
    monotonic**, not blind sets.
27. **r13's recovery could not actually recover.** The stale-task detector reads the database and has
    no `TaskCompletion` payload — no metrics, no branch, no diff — so it could never "take over" a
    finalisation. And telling a Pub/Sub redelivery (which *does* carry the payload, at 60s, V22) to
    wait out a 10-minute lease is an unbounded-wait violation. gemini-3-1-pro caught both. The claim
    now persists the payload, and a redelivery proceeds immediately.
28. **r13's matrix omitted an effect entirely.** The failure arm also calls `updateStageError`
    (V48) — a 13th effect that appeared in no table across thirteen revisions.

**Method note:** no reviewer caught r1's stage error. Both r1 errors surfaced from *acting on* the objections —
the list-view number that anchored r1 was wrong, and only the detail endpoint disclosed it. This is
[M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) V40 again — *verify the artefact,
not the aggregate* — and it cost a whole revision to relearn.

---

## Problem statement

M-MESSAGE-PLANE-TRUST fixed **delivery**. M-COORDINATOR-EXECUTION-TRUST fixed **execution** — and it
worked: measured 2026-09-02, a message from an attended session produced a Cloud Run job in 8 seconds,
ran on the new 2h ceiling, wrote a real design doc, pushed a branch and opened **PR #1033** against the
correct repo. The commit-and-push half is genuinely deterministic: wrapper code, not a model choosing
to push.

Then the pipeline stopped dead, silently, at the seam this doc is about.

`design-doc-creator` carries `trigger_on_complete: [sprint-planner]` in live prod config. Its task
completed. **No sprint-planner task was ever created** — not because the handoff logic is wrong (it is
one of the better-built things in the coordinator, V2–V4) but because the cloud completion path never
creates the approval record the handoff hangs off (V7–V9).

**The unifying defect: two completion paths diverged, and only the unused one is complete.**

| Side effect on task completion | daemon (`daemon_tasks_exec_run.go`) | **cloud (`pubsub_completion_handler.go`) — prod** |
|---|---|---|
| Mark task status | ✅ | ✅ |
| Post a message to an inbox | ✅ approval message | ✅ agent's own inbox |
| `CreateApprovalRequest` | ✅ `:602` | ❌ **never called** |
| Embed `handoff_targets` | ✅ `:555` | ❌ |
| `handleAgentHandoffs` (auto edges) | ✅ `:633` | ❌ |
| `updateChainStageStatus` | ✅ `:503,:523,:654` | ❌ |
| `updateChainStatus` | ✅ | ❌ |
| `updateStageMetrics` / `updateChainMetrics` | ✅ | ❌ |
| `updateStageSession` | ✅ | ❌ |
| `ProcessStageCompletion` (GitHub routing) | ✅ `:532` | ❌ |
| Persist approval diff (#921) | ✅ `:568-577` | ❌ |

Ten to two. Prod runs the two.

**Impact, measured against prod 2026-09-03:**

- **No handoff has fired in 30 days of coordinator logs** (V5), instrument proven (V6).
- **Every cloud stage is frozen at `pending` with zero metrics** (V15), because every stage-update
  helper lives only on the daemon completion path (V12).
- **315 of 404 chains are stuck `active`**, oldest since **2026-04-27** — four months (V18). Chain
  status is therefore not a usable health signal.
- The data model's own doc-comment gives its example as `"design-doc-creator -> sprint-planner ->
  sprint-executor"` (V17). No chain has ever advanced past stage 1.

Same shape as both predecessors, one layer up: **the parts work, the composition is unobservable, and
one number says everything is fine.**

**Why this blocks the stated goal.** The intent (Mark, 2026-09-03) is unsupervised work for the AILANG
package ecosystem: *"work is passed off correctly, work is done and committed and pushed to the right
repo."* Push works (V14). Pass-off has never executed once.

---

## Verification Log

Checked against code or prod (`ailang-multivac`) on 2026-09-03. Negatives carry positive controls.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | The pipeline is already declared in prod config | multivac `config/config.cloud.yaml`: `design-doc-creator`→`[sprint-planner]` (:107), `sprint-planner`→`[sprint-executor]` (:135), `sprint-executor`→`[sprint-evaluator]` (:174) | **Confirmed** — 3 of 34 agents carry a non-empty `trigger_on_complete` |
| V2 | Handoff targets are **registry config**, not model output and not sender-supplied | `daemon_tasks_exec_run.go:542-556` reads `agent.TriggerOnComplete` off the registry record for `task.AgentID` | **Confirmed — deterministic by construction.** The property the ecosystem goal needs already holds |
| V3 | The handoff message is coordinator-authored to a registry-resolved inbox | `approval_processor.go:641-660`: `FromAgent: "coordinator"`, `ToInbox: targetAgent.Inbox` via `GetAgentByID`, carrying `ParentTaskID` and `ChainID` | **Confirmed** |
| V4 | A handoff-replay guard exists — **and its scope is handoffs only** | `MarkApprovalHandoffsTriggered` sets `approval_requests.handoffs_triggered = 1` per task; the catch-up query filters `type='merge_handoff' AND status='approved'` (`store_sqlite_approvals.go:320-340`) | **Confirmed, and it REFUTES r1's C1.** The flag is post-approval and handoff-scoped; it cannot guard approval creation, stage updates or metrics, which all happen *before* an approval exists |
| V5 | **No handoff has ever fired in prod** | `gcloud logging read`, `service_name="ailang-coordinator"`, `--freshness=30d`, `textPayload:"handoff"` → **0 rows** | **Confirmed (negative)** |
| V6 | …and the instrument was proven before the negative was believed | The first query (bare `resource.type`) returned empty **including a control that must match**. Adding `resource.labels.service_name` made the control return rows | **Method note.** The wrong answer was indistinguishable from the right one |
| V7 | The cloud path never creates an approval record | `grep -rn "CreateApprovalRequest"` (non-test) → 3 sites: `daemon_tasks_exec_run.go:602`, `daemon_tasks_budget.go:106`, `approval_checkpoint.go:370`. `pubsub_completion_handler.go` has **none** | **Confirmed (negative), instrument proven by 3 positives** |
| V8 | `MarkTaskPendingApproval` does not create one either | `internal/storage/firestore/coordinator_transitions.go:39-75` — a single `Doc(collTasks, id).Update(...)` over status/metrics fields | **Confirmed.** The status claims a control that does not exist |
| V9 | Therefore the handoff gate can never pass for a cloud task | `approval_processor.go:608` returns early unless `approvalReq.Type == "merge_handoff"`; there is no `approvalReq` at all | **Confirmed — root cause** |
| V10 | The **auto-approved** edge is dead by a second, independent route | `grep -rn "handleAgentHandoffs"` → exactly one call site, `daemon_tasks_exec_run.go:633` | **Confirmed (negative).** Both handoff mechanisms are daemon-only |
| V11 | The divergence is far wider than handoffs | `daemon_tasks_exec_run.go:470-640` vs `pubsub_completion_handler.go:111-165` | **Confirmed** — 10 effects vs 2 |
| V12 | Every stage/chain update helper guards on IDs and is called only from the daemon completion path | `daemon_tasks_chain.go:10,22,34,46,67` — each returns early on `task.StageID == ""` (or `ChainID == ""`); `updateChainStageStatus` has 4 call sites, all in `daemon_tasks_exec_run.go`; `ProcessStageCompletion` has 1, same file | **Confirmed (negative)** |
| V13 | `observatory_sync.go` does **not** cover the gap | Its surface is `SyncTask`, `SyncAgentAssignment`, `CompleteAgentAssignment` — no stage or chain-status function | **Confirmed (negative)** |
| V14 | A live task reproduces it | `task-88a9fa95` (design-doc-creator, `trigger_on_complete: [sprint-planner]`) → PR #1033 open, +157 lines, correct repo, no sprint-planner task | **Confirmed — reproduction in hand** |
| **V15** | **Cloud tasks DO get a chain and a stage — the stage is then never updated** | `GET /api/chains/87f3840e…` → `stages: [{id: 3e88a511, stage_number: 1, agent_id: "design-doc-creator", task_id: "task-88a9fa95", status: "pending", cost: 0, turns: 0, started_at: 20:26:20}]`; chain `status: active`, ~12h after completion | **Confirmed — and this CORRECTS r1**, which claimed no stage existed. Creation works (`daemon_tasks_polling.go:546-566`); progression is what is missing |
| V16 | Stage creation is silent-on-success, which is why logs looked empty | `daemon_tasks_polling.go:546-566` logs only on failure; the parallel local-message branch at `:276-303` logs on success | **Confirmed.** Absence of "Created chain stage" was not evidence of absence |
| V17 | The data model documents the pipeline that has never advanced | `internal/observatory/models_chains.go:251` — `AgentFlow string // e.g., "design-doc-creator -> sprint-planner -> sprint-executor"` | **Confirmed** |
| V18 | 315 chains leak as `active`, oldest 2026-04-27 | `GET /api/chains?limit=500` → 404 chains: 315 `active`, 84 `failed`, 5 `completed`; all 404 with `stages_completed: 0` | **Confirmed** — >4 months |
| **V19** | **Chain cost is written by exactly one function, on the daemon path only** (gemini-3-1-pro's objection, resolved) | `UpdateChainMetrics` is called only from `daemon_tasks_chain.go:67 updateChainMetrics`, guarded on `task.ChainID`; stage writes are guarded on `task.StageID` instead. Consistent with the 5 cost-bearing chains all falling on 2026-08-26 | **Resolved.** The mutation path is identified by file:line, so M2 can route cost through one call with no double-count risk |
| **V20** | **The GCP observatory backend silently no-ops every chain/stage write** | `backend_gcp.go:677-730` — `ListChains`→`(nil,nil)`, `CreateStage`→`(nil,nil)`, every `UpdateStage*`→`nil`. No error, ever | **Confirmed — a latent landmine and a direct violation of CLAUDE.md §2 ("no silent fallbacks").** Prod is *not* on this backend (V15 proves stages persist), but any deploy that selects it would erase the chain model with zero signal |
| **V21** | **Effect-by-effect worktree classification** (oc-glm-5-2's objection, resolved) | Read each call site | See the classification table below — **9 of 11 are worktree-independent**, 1 needs a cloud strategy, 1 is correctly daemon-only |
| **V22** | **Pub/Sub redelivery is real, not hypothetical** | `gcloud pubsub subscriptions list` → `ailang-completions-coordinator`, `ackDeadlineSeconds: 60`, `retryPolicy.minimumBackoff: 10s`; push delivery is at-least-once | **Confirmed** |
| **V23** | **The existing idempotency guard does not cover the state cloud tasks land in** | `pubsub_completion_handler.go:84-86` skips only if the task is *already terminal*. `pending_approval` is **not** terminal — and per M-COORDINATOR-EXECUTION-TRUST V20 that check is a literal string comparison over `completed/failed/cancelled`, so `no_changes` escapes it too | **Confirmed.** A redelivered completion re-runs finalisation in full — the concurrency case r1 wrongly dismissed |
| V24 | The system directs the user to an affordance that shows a frozen stage | PR #1033 body: *"View execution chain: `ailang chains view task-88a9fa95`"* | **Confirmed** |
| **V25** | **In cloud, every finalisation effect writes to ONE Firestore database through ONE client** | `internal/storage/backend.go:154-168` — `fsClient, err := fsstore.NewClient(ctx)` with the comment *"Firestore client (shared by coordinator and messaging)"*, then `NewCoordinatorStore(fsClient)`, `NewMessagingStore(fsClient)`, `NewObservatoryStore(fsClient)`. `observatory.go:16` compile-time asserts `var _ obs.Backend = (*ObservatoryStore)(nil)` | **Confirmed — and it ANSWERS gpt5-6-sol.** Tasks, approvals, chains, stages and inbox messages are co-resident. No cross-store boundary exists on the cloud path |
| **V26** | **Firestore transactions are available and already wired** | `internal/storage/firestore/client.go:76-79` — `RunTransaction(ctx, fn)` delegating to `fs.RunTransaction`; `client.go:66` `Batch()` | **Confirmed.** The ledger can commit atomically with its effect using an existing API, not a new one |
| **V27** | **The cloud artifact path is NOT guaranteed** | `cmd/ailang/coordinator_cloud.go:110` — *"artifactPath is the GCS path prefix where raw artifacts were uploaded (may be empty)"*; it is passed straight through to `TaskCompletion.ArtifactGCSPath` at `:122` | **Confirmed — REFUTES r2's claim.** M2 must log an explicit reason when it is empty, never skip silently |
| **V29** | **Effect 10's external surface is NOT idempotent** | `stage_execution.go:508` → `taskChain.OnDesignDocComplete` → `task_chain.go` makes **12 `PostCommentInRepo` calls** (:99,:148,:166,:203,:235,:286,:304,:339,:371,:412,…) plus `AddLabelInRepo` (:209,:345). `github_poster.go:55 PostCommentInRepo` has **no dedup and no marker**; only `AddLabel` (:66) is idempotent ("Ensure label exists first") | **Confirmed — REFUTES r3.** A retried comment duplicates. Effect 10 needs a real outbox, not claim-after-effect |
| **V29b** | **Effect 10 does not apply to the ecosystem pipeline at all** | `stage_execution.go:437` — `if task.GithubIssue == 0 \|\| task.Stage == TaskStageNone \|\| d.taskChain == nil { return nil }`. Message-sourced tasks (the whole package-ecosystem path) carry `GithubIssue == 0` | **Confirmed.** Effect 10 is inert for the tasks this doc exists to enable — it is a GitHub-linked-pipeline concern only, which bounds its risk sharply |
| **V30** | **No store method accepts a caller-supplied transaction** | `internal/coordinator/store.go:242` `CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error`; `internal/observatory/backend.go:131` `UpdateChainMetrics(ctx, id, cost, tokens, turns) error`; `:149` `UpdateStageStatus(ctx, stageID, status) error` — none takes a `*firestore.Transaction` or `*sql.Tx` | **Confirmed — REFUTES r3.** Shared client (V25) gives the *possibility* of atomicity; the API does not yet expose it. Building `FinalizationTx` is now in M1's scope |
| **V28** | **The daemon path is NOT single-store — it is three separate databases** | `internal/storage/backend.go NewSQLiteBackends`: `coordinator.NewSQLiteStore(dir/"coordinator.db")`, `messaging.OpenStore(dir/"collaboration.db")`, `observatory.NewSQLiteBackendFromPath(dir/"observatory.db")` — three files, three `*sql.DB` | **REFUTES r4's V28**, which inferred co-residency from directory names. No `*sql.Tx` can span them. **Cross-store atomicity is therefore not available on the daemon path, and the design must not depend on it** — depending on it would re-create the very path divergence this doc exists to fix |

| **V31** | **`CreateApprovalRequest` is a bare INSERT — a retry ERRORS, it is not a no-op** | `store_sqlite_approvals.go:35-38` — `INSERT INTO approval_requests (id, …) VALUES (?, …)`, no `ON CONFLICT` / `OR IGNORE` / `OR REPLACE` | **Confirmed — REFUTES r5.** With the deterministic `apr-<hash>` id, redelivery hits a UNIQUE violation. Needs a new `CreateApprovalIfAbsent` primitive |
| **V32** | **Both metrics writers INCREMENT — a retry double-counts cost** | `internal/observatory/store_chains.go:237-244` `UPDATE execution_chains SET total_cost = total_cost + ?, total_tokens = total_tokens + ?, total_turns = total_turns + ?`; `:661-672` `UPDATE chain_stages SET cost = cost + ?, tokens_in = tokens_in + ?, … duration_ms = duration_ms + ?` | **Confirmed — REFUTES r5's "absolute set, never increment".** Exactly the failure gemini-3-1-pro predicted. Needs absolute setters |
| **V33** | The status and session writes **are** absolute sets, as claimed | `store_chains.go` `UPDATE chain_stages SET status = ?`, `UPDATE execution_chains SET status = ?, updated_at = ?, completed_at = COALESCE(?, completed_at)`, `UPDATE chain_stages SET session_id = ?`; `store_sqlite.go:633+` `UPDATE tasks SET …` | **Confirmed.** Effects 1, 6, 7 and 9 are already idempotent; no new primitive needed |
| **V34** | **The messaging store accepts a caller-supplied id but its INSERT is bare** | `internal/messaging/inbox.go:105-106` — `if msg.ID == "" { msg.ID = uuid.New().String() }` (so a deterministic id *can* be supplied); `:183-185` — `INSERT INTO inbox_messages (…) VALUES (…)` with no conflict clause | **Confirmed — REFUTES r5's "a re-insert collides rather than duplicating".** Exactly what oc-glm-5-2 asked to check. Needs `PutMessageIfAbsent` |

| **V31fs** | **Firestore approval creation OVERWRITES rather than erroring** | `internal/storage/firestore/coordinator_approvals.go:43` — `s.client.Doc(collApprovals, req.ID).Set(ctx, data)` | **Confirmed — and it differs from SQLite (V31).** No duplicate error, but a redelivered completion would **reset an approved/rejected approval back to `pending`**. `CreateApprovalIfAbsent` is required on *both* backends, for opposite reasons |
| **V32fs** | **Firestore metrics increment too — identical double-count** | `internal/storage/firestore/observatory_chains.go:176-180` — `Update(ctx, []firestore.Update{{Path:"total_cost", Value: firestore.Increment(cost)}, …})` | **Confirmed.** Both backends double-count on retry; the absolute setter is needed on both |
| **V34fs** | **Firestore message insert overwrites rather than erroring** | `internal/storage/firestore/messaging_inbox.go:47` — `s.client.Doc(collInbox, msg.ID).Set(ctx, inboxToMap(msg))`; caller-supplied ids accepted (`:23-24`), and its own comment notes `MessageID` must equal `ID` because the write is `Doc(...)`-addressed | **Confirmed — differs from SQLite (V34).** A re-insert silently overwrites, which would reset a message already marked read. `PutMessageIfAbsent` required on both |
| **V35** | **The completion payload carries no commit SHA** | `internal/pubsub/topics.go` `TaskCompletion` — has `BranchName`, `ChangedFiles`, `ArtifactGCSPath`, metrics; **no head/base commit field** | **Confirmed — REFUTES r6's M3.** `base_commit..branch_name` is not replay-safe: a branch can move or be deleted between attempts. M3 must add an immutable `HeadCommit` to the payload, published by the executor |
| **V36** | **`execution_chains` has no `reason` column** | `internal/observatory/store_chains.go:41-43` — insert columns are `id, source_type, source_ref, github_repo, github_issue_number, status, current_stage, workspace_id, workspace_path, created_at` | **Confirmed.** D3's "terminate with an explicit reason" needs a migration in M4 or the phrase must go |

| **V37** | **`base_commit` is empty for every cloud task** | It originates from the worktree — `worktree.go:138 BaseCommit: baseCommit`, read at `daemon_tasks_exec_run.go:410 baseCommit = worktree.BaseCommit`. Its only persistence path is `MarkTaskPendingApproval` (`store_sqlite.go:637`, `coordinator_transitions.go:45`), which the cloud handler calls as `MarkTaskPendingApproval(ctx, id, "", completion.BranchName, "", "", execResult)` (`pubsub_completion_handler.go:132`) | **Confirmed — REFUTES r7's M3.** The field exists in both schemas but the cloud path writes `""`. M3 must have the executor supply **both** `BaseCommit` and `HeadCommit` |
| **V38** | The two "messages" files are different subsystems | `internal/storage/firestore/messaging_inbox.go:47` is the inbox writer (`Doc(collInbox, msg.ID).Set`, V34fs). `internal/storage/firestore/observatory_messages.go:18` is the observatory mirror — `ObservatoryStore.CreateMessage/GetMessage/ListMessages/UpdateMessage/DeleteMessage` over `obs.Message` | **Confirmed.** r7's Files list named the wrong one; corrected below |

| **V39** | **The executor already computes the diff, from the base commit, with a real git tree** | `cmd/ailang/coordinator_cloud.go:505` — `changedFiles := discoverChangedFilesFromCommit(workDir, clonePoint)`; the result is published on the payload at `:121` (`ChangedFiles: changedFiles`), and `:109` documents it as "discovered via git diff". `clonePoint` **is** the base commit, held by the executor | **Confirmed.** The coordinator never needs a clone. M3 shrinks to: executor also publishes `DiffStat`, a capped patch, `BaseCommit` (= `clonePoint`) and `HeadCommit`; the cloud `DiffSource` reads the payload |

| **V33fs** | **The Firestore status and session writes ARE absolute — verified, not assumed** | `internal/storage/firestore/observatory_chains.go:321-326` `Doc(collObsChainStages, stageID).Update(…{Path:"status", Value: …}, {Path:"completed_at", …})`; `:165-171` chain status the same; `:357-358` `{Path:"session_id", Value: sessionID}`. No `firestore.Increment` on any of them | **Confirmed.** Effects 1, 6, 7 and 9 are idempotent on **both** backends. gpt5-6-sol was right that r9 asserted this for Firestore; checked, the claim holds |
| **V40** | **The executor captures no head SHA at push** | `cmd/ailang/coordinator_cloud.go:529-533` — `exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", branchName)`; the success path returns `branchName, execResult, …` with no `rev-parse` | **Confirmed — REFUTES r9's "captured at push".** M3 must add an explicit `git rev-parse HEAD` after a successful push, and the push code path joins the Files list |
| **V41** | **The task record has no home for the ledger** | `internal/coordinator/store_sqlite.go:197-204` — the `tasks` table is extended column-per-field (`ALTER TABLE tasks ADD COLUMN …`, JSON-encoded into TEXT where structured); there is no map/JSON field for `finalization.<effect>` | **Confirmed.** M1 carries an `ALTER TABLE tasks ADD COLUMN finalization TEXT` migration (JSON-encoded). Firestore needs none — a map field is schemaless |

| **V42** | **The executor discovers changed FILES, not a diff** | `cmd/ailang/coordinator_cloud_github.go:174-188` — `discoverChangedFilesFromCommit(workDir, clonePoint) []string` wraps `coordinator.NewArtifactDiscovery(...).DiscoverChangedFiles()`, returning filenames | **Confirmed — REFUTES r10's "same code path".** M3 adds a `git diff --stat` and a capped `git diff <base>..<head>` in the executor; the estimate rises accordingly |
| **V43** | **The daemon's completion effects are OUTCOME-CONDITIONAL, not unconditional** | `daemon_tasks_exec_run.go`: the block is `if <success> { … } else { … }`; inside success, a further `if skipApproval { MarkTaskCompleted … } else { MarkTaskPendingApproval, ProcessStageCompletion, CreateApprovalRequest … }`; `handleAgentHandoffs` sits at `:633` **inside the success arm**; the failure arm does `MarkTaskFailed` + `updateChainStageStatus(StageStatusFailed)` + `updateChainStatus(ChainStatusFailed)` (:654-655) | **Confirmed — REFUTES r10's D1.** Unconditional extraction would create approvals and dispatch handoffs for failed tasks. The orchestrator must be driven by the outcome matrix below |
| **V44** | **A ledger field absent from the converter is silently dropped** | `internal/storage/firestore/coordinator_convert.go:40` (`"base_commit": t.BaseCommit`) and `:114` (`BaseCommit: getString(data, "base_commit")`) — `TaskRecord` is mapped by hand in both directions, not by `DataTo` reflection over unknown fields | **Confirmed.** The ledger needs a `Finalization` field on `TaskRecord` **and** entries in both converter directions, or every redelivery reads an empty ledger and re-runs every effect |

| **V45** | **`handleAgentHandoffs` fires at completion for ALL successful tasks, not on approval** | `daemon_tasks_exec_run.go`: the `if skipApproval {…} else {…}` block closes at `:629`; `handleAgentHandoffs` is called at `:633`; the success arm closes at `:637` with `} else {` opening the failed arm | **Confirmed — REFUTES r11's matrix.** Auto-approved edges (`AutoApproveHandoffTo`) dispatch immediately on completion for both sub-branches; only **non-auto** targets are embedded in the merge approval and wait for it (consistent with the comment at `:545-550`) |
| **V46** | A 2-minute sweep already runs in prod and can host ledger reconciliation | Prod coordinator log, 2026-09-02T20:26:19: `stale_task_detector.go:73: stale task detector: started (interval=2m0s)` | **Confirmed.** C1's recovery needs no new scheduler |

| **V47** | `max_concurrent_tasks: 1` is genuinely set on all three pipeline agents | multivac `config/config.cloud.yaml` — `:111` (design-doc-creator), `:139` (sprint-planner), `:179` (sprint-executor) | **Confirmed.** The claim held, but it was a prod-config assertion propping up a safety argument and had never been checked — oc-glm-5-2 was right to demand it |

| **V48** | **The failure arm writes metrics, session AND a stage error; and the daemon has no `no_changes` arm at all** | `daemon_tasks_exec_run.go` failure arm: `MarkTaskFailed`, `updateChainStageStatus(Failed)`, `updateChainStatus(Failed)`, `updateStageSession`, `updateStageMetrics`, `updateChainMetrics`, **`updateStageError`**. `grep -n "TaskStatusNoChanges\|no_changes" daemon_tasks_exec_run.go` → **empty** | **Confirmed.** r13's `failed`/effect-8 cell was right after all — but it was unverified, and the check surfaced **effect 13 (`updateStageError`), missing from every prior matrix**. It also shows `no_changes` is cloud-only (introduced by the sibling doc's M2), so its column defines new behaviour rather than preserving daemon behaviour — and the doc must say so |

### OPEN (bounded — no milestone depends on it)

**O1 — the chain *list* view under-reports stages.** `GET /api/chains` returns `stage_count: 0` and
`agent_flow: ""` for all 404 chains, while `GET /api/chains/{id}` returns the stage rows (V15).
`Store.ListChains` (`store_chains.go:151-160`) has a correct `COUNT(s.id)` / `GROUP_CONCAT` over a
`LEFT JOIN`, and `handleListChains` calls `obsBackend.ListChains` directly — so the discrepancy is in
which backend serves the dashboard, not in the SQL. **This is a read-path reporting defect. Nothing in
M1–M5 depends on it**; it is scoped as M6 and must not be conflated with the write-path work. It is
recorded because it is what made r1 draw a false conclusion.

### V21 — effect classification

| # | Effect | Class | Note |
|---|---|---|---|
| 1 | Mark task status | **(a) portable** | Cloud already does this |
| 2 | Post completion/approval message | **(a) portable** | Both paths already post; only the recipient differs |
| 3 | `CreateApprovalRequest` | **(a) portable** | Pure store write |
| 4 | Embed `handoff_targets` | **(a) portable** | Registry lookup only (V2) |
| 5 | `handleAgentHandoffs` | **(a) portable** | Registry + message store |
| 6 | `updateChainStageStatus` | **(a) portable** | Needs `task.StageID`, which cloud tasks **have** (V15) |
| 7 | `updateChainStatus` | **(a) portable** | Needs `task.ChainID` — present |
| 8 | `updateStageMetrics` / `updateChainMetrics` | **(a) portable** | Needs `ExecuteResult`; the cloud handler already builds one (`pubsub_completion_handler.go:92-110`) |
| 9 | `updateStageSession` | **(a) portable** | `completion.SessionID` is present |
| 10 | `ProcessStageCompletion` | **(d) OUT OF SCOPE — daemon-only, unchanged** | r2/r3 classed this (b). r5 removes it: its external surface is 12 non-idempotent comment POSTs (V29), and it is already inert for every message-sourced task (V29b). Left exactly as today; its own doc |
| 11 | Persist approval diff (#921) | **(b) cloud strategy needed** | `GetWorktreeDiff` is worktree-only; cloud must diff `base_commit..branch_name` from the pushed branch — this is M3's `DiffSource` |
| 12 | Worktree removal | **(c) daemon-only, correctly absent** | No cloud equivalent; must stay behind the daemon strategy |

**Conclusion:** the extraction is mechanically sound. Nine effects port directly; two need a named
strategy; one stays daemon-only. **No effect is blocking.** r1's "pass `WorktreePath: \"\"` and branch
on emptiness" framing was a silent fallback and is withdrawn — see D1.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A task's completion effects become a function of the task, not of which executor ran it. |
| A2: Replayability | +1 | Stage progression and metrics make a chain replayable; today every cloud stage is frozen at `pending` (V15). |
| A3: Effect Legibility | +1 | "handed off to sprint-planner" becomes a recorded stage transition, not an inference from two unrelated inbox messages. |
| A4: Explicit Authority | +1 | The cloud path reaches `pending_approval` with **no approval object** (V7/V8) — authority is neither granted nor withheld, merely absent. M1 makes reject-by-default actually hold. |
| A5: Bounded Verification | +1 | M5 lands a parity arm that fails RED when an effect is added to one path only — the exact defect class this doc exists for. |
| A6: Safe Concurrency | **+1** | **Raised from r1's 0, which was unsupported.** Redelivery is real (V22) and the current guard misses `pending_approval` (V23). M1 adds a per-effect finalisation ledger where today there is none, and makes every effect idempotent so correctness needs no cross-store transaction (V28/V30) — the two paths therefore converge on one protocol instead of two. |
| A7: Machines First | +1 | Every consumer is a machine: approval processor, handoff trigger, dashboard, sweep. A frozen stage is unanalysable. |
| A8: Minimal Syntax | 0 | No language surface touched. |
| A9: Cost Visibility | +1 | V19 identified the single cost-writing call site, so pipeline cost becomes attributable without double-count. |
| A10: Composability | +1 | Unification by **extraction + strategy**, not a second implementation — the mechanism that produced the divergence. |
| A11: Structured Failure | +1 | A failed cloud task currently leaves its chain `active` forever (V18); M4 makes termination explicit. |
| A12: System Boundary | 0 | No boundary change; handoffs already stop at the registry (V2/V3). |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- **Silent fallback introduced?** **r2 answered "No" and was wrong** — its `Cleanup` no-op was one
  (oc-glm-5-2). r3 removes it: `Cleanup` leaves the interface, the orchestrator branches on
  `strategy.Kind()` and logs the cloud case explicitly, and `ArtifactSource` returns
  `ErrNoArtifactSource` rather than a silent nil (V27). With that, the answer is No, and three
  pre-existing fallbacks are *removed*: `pending_approval` asserting a nonexistent control (V8),
  r1's `WorktreePath == ""` branching, and r2's `Cleanup`. V20 records a fourth that this doc names
  but does not fix.
- **Backward compatibility carried?** No. The 315 stuck chains are reconciled (M4), not preserved.
- **New routing table?** No. `trigger_on_complete` stays the single source of pipeline topology.

---

## Goals

1. **One completion path**, because both callers run the same code — for **every effect in scope**.
   Effect 10 (`ProcessStageCompletion`) is a deliberate, named exception: it stays daemon-only and
   unchanged, with its own doc (Non-Goals). V29b bounds the consequence — it is inert for every
   message-sourced task — but it does not make the paths identical, and this doc does not claim they
   will be.
2. **The configured pipeline actually runs** end to end in prod on real work.
3. **A chain is legible while running and terminal when done** — stages advance past `pending`,
   metrics are non-zero, no chain outlives its task's terminal state.
4. **Approval means something on the cloud path** — a real record carrying a real diff.

### The goal behind the goal

Trust for **unsupervised** package-ecosystem work. Note the asymmetry this creates for review:
because nothing hands off, nothing compounds, so **no incorrect handoff has ever been observed — the
absence of failures here is not evidence of correctness.**

---

## High-Impact Decisions

### Rulings — 2026-09-03, Mark, attended

- **D2 — RULED: per-edge configuration.** *"we configure that per edge."* Reject-by-default stays
  the global posture; `auto_approve_handoff_to` is the instrument, set per edge as each one earns it.
  The existing `sprint-executor → sprint-evaluator` edge (`config.cloud.yaml:176`) is the precedent.
  **C3 still binds:** an auto-approved edge behind a blind approval card is the worst available
  combination, so M3 lands before any new edge is opened.
- **D3 — RULED: abandon.** *"yes abandon."* The 315 stuck chains are marked terminal; **no synthetic
  stage transitions are created.** Per V36 `execution_chains` has no `reason` column, so M4 carries a
  one-column migration rather than dropping the reason.
- **D5 — RULED: terminal, nothing follows.** A `no_changes` completion writes task/stage/chain
  terminal plus metrics and session, and creates **no approval, no handoff, and no retry**. Rationale:
  handing a sprint-planner an empty design doc is worse than stopping the chain, and `no_changes` is a
  semantic outcome, not a transport failure — so the sibling doc's re-dispatch path (M3,
  `MaxTaskExecutions = 2`) must not catch it. The chain is marked terminal so it does not join the
  `active` leak this doc exists to close.
- **D1 — author's recommendation stands** (orchestrator + explicitly-kinded strategy); no reviewer
  objected to the shape across thirteen rounds, only to its details, all of which are now fixed.
- **D4 — deferred to the sibling doc.** Whether `sprint-*` agents take `work_tier: tier1` belongs to
  [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) M1a, which owns the tier model.



**D1 — How do the two paths become one?** *(restructured r2 after oc-glm-5-2; corrected r3 after
oc-glm-5-2 and gemini-3-1-pro)*

Not "one big function". **An orchestrator plus an explicitly-kinded strategy**, justified by V21:
nine effects are shared and unconditional; two are strategy-supplied; one is daemon-only and is
branched on **by kind**, never by a no-op body.

```go
type StrategyKind int
const (StrategyKindDaemon StrategyKind = iota; StrategyKindCloud)

type ExecutionStrategy interface {
    Kind() StrategyKind

    // Supplied by both kinds, with genuinely different implementations.
    DiffSource(ctx context.Context, task *TaskRecord) (DiffResult, error)

    // ArtifactSource returns ErrNoArtifactSource when this executor produced
    // none — an explicit, logged outcome. It is never a silent nil (V27).
    ArtifactSource(ctx context.Context, task *TaskRecord) (ArtifactSource, error)
}

// Cleanup is NOT on the interface. Worktree removal is daemon-only (V21 class c),
// so the orchestrator branches on kind and records why:
//
//   switch strategy.Kind() {
//   case StrategyKindDaemon: err = removeWorktree(ctx, task)
//   case StrategyKindCloud:  logger.Printf("finalize: no cleanup applicable for cloud executor (task %s)", task.ID)
//   }
```

`FinalizeTaskCompletion(ctx, deps, task, result, strategy)` is **driven by the completion outcome**,
not applied unconditionally — r10 said "1–9 unconditionally" and V43 refutes it. Effect 10 is not
performed at all; it stays daemon-only (V29b).

| Effect | `completed` (skip_approval) | `completed` (normal) | `no_changes` ¹ | `failed` |
|---|---|---|---|---|
| 1 Task status | `completed` | `pending_approval` | `no_changes` | `failed` |
| 2 Completion message | ✅ | ✅ | ✅ | ✅ |
| 3 Approval + 4 handoff targets + 11 diff | — | ✅ | — | — |
| 5 Handoff dispatch | ✅ auto edges, **at completion** | ✅ auto edges **at completion** (V45); non-auto edges embedded in the approval and dispatched on approval | — | **never** |
| 6 Stage status | `completed` | `awaiting_approval` | terminal | `failed` |
| 7 Chain status | `completed` | `pending_approval` | terminal | `failed` |
| 8 Metrics + 9 session | ✅ | ✅ | ✅ | ✅ — **verified** (V48): the failure arm calls `updateStageMetrics`, `updateChainMetrics` and `updateStageSession` |
| **13 Stage error** (`updateStageError`) | — | — | — | ✅ (V48) |
| 12 Cleanup (daemon) | ✅ | keep worktree | ✅ | ✅ |

¹ **`no_changes` has no daemon behaviour to preserve** — the daemon path has no such arm (V48); the
status exists only on the cloud path, added by the sibling doc's M2. That column therefore *defines*
intended behaviour rather than mirroring existing behaviour, and D5 below asks for a ruling on it.

This matrix **is** the parity contract: M5 asserts both call sites produce exactly it. Preserving the
daemon's current behaviour column-for-column is what makes the extraction provably
behaviour-preserving on the path that works today.

The rejected alternatives — cloud calling the daemon method directly (drags worktree assumptions
in) and an internal event hop (adds an async gap in the very property we are making trustworthy) —
are recorded here rather than re-litigated. **Needs a ruling.**

**D2 — Does the ecosystem pipeline stay reject-by-default?**
The per-edge mechanism already exists (`auto_approve_handoff_to`, used once at
`config.cloud.yaml:176`), so this is a **configuration** ruling, not a code one.
*Recommendation: keep reject-by-default globally; widen per edge only after M3 lands* — an
auto-approved edge behind a blind approval card (#921) is the worst available combination.
**Needs a ruling.**

**D3 — What happens to the 315 stuck `active` chains?**
*Recommendation: mark them `abandoned` with an explicit reason and create no stage transitions.* Note `execution_chains` has **no `reason` column** (V36), so M4 carries a one-column migration — or the ruling drops the reason and records termination in the status alone.
Backfilling synthetic progression would invent history and poison the first dataset anyone uses to
check whether M1 worked. **Needs a ruling.**

**D5 — What should `no_changes` do?** It has no daemon precedent (V48): the status exists only on
the cloud path. The matrix currently marks task/stage/chain terminal, metrics and session written, and
no approval or handoff — i.e. "it ran, it cost money, it changed nothing, nothing follows." That is
the author's reading of the sibling doc's intent, not a preserved behaviour. **Needs a ruling.**

**D4 — Do the `sprint-*` pipeline agents get `work_tier: tier1`?**
Cross-doc with [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) M1a. A pipeline
amplifies the intermittent-ack problem: four chances to stall instead of one. **Needs a ruling — may
belong wholly to the sibling doc.**

---

## Conflict Surface

**C1 — Effectively-once finalisation.** *(r2 redesigned it; r3 and r4 each rested on an atomicity
premise that turned out to be false; r5 stops requiring atomicity at all.)*

`handoffs_triggered` cannot serve as the guard: it is post-approval and handoff-scoped (V4), while
approval creation, stage updates and metrics all precede any approval. Redelivery is real (V22) and
the existing terminal-state guard does not cover `pending_approval` (V23) — so today a redelivered
completion re-runs finalisation in full.

**Why the previous two designs failed.** r3 assumed the store methods could join a caller-supplied
transaction; they cannot (V30). r4 assumed the daemon's stores shared one database; they do not
(V28) — three files, three `*sql.DB`. Cross-store atomicity is unavailable on the daemon path and
would require new API on the cloud path. **Designing around it would also re-create path
divergence**, which is the defect this doc exists to remove.

**The protocol that does not need it.** Every effect is made **individually idempotent**, and the
ledger records progress rather than guaranteeing atomicity. At-least-once delivery plus idempotent
effects gives effectively-once, uniformly, on both paths:

- The ledger is `finalization.<effect> = {state, at, attempt}` on the **task record** — one store on
  either path (Firestore tasks; `coordinator.db`). It never spans stores.
- Each effect is claim-then-apply-then-mark. A crash between claim and mark re-applies the effect,
  which is safe **because the effect itself is idempotent** — that is the load-bearing property, not
  the transaction.
- **Status writes are conditional, not blind.** An absolute write is *repeatable* but not idempotent
  once another step has legitimately advanced the record: a stale replay writing `pending_approval`
  over a task the approval processor has already approved would **regress** it. So every status
  effect is a compare-and-set against the state the finalizer expects to find, and status values are
  ordered so a write can only advance, never move backwards. A CAS that does not match is recorded as
  `superseded` — a normal outcome, not an error.
- **The claim carries the payload.** The ledger row stores the `TaskCompletion` fields the effects
  need (metrics, branch, SHAs, diff). Without this, recovery cannot recover: the stale-task detector
  reads only the database and has no completion payload, so it could never re-apply an effect it
  took over.
- Reconciliation sweeps ledger rows left `pending`, on **bounded, specified terms**:

| Property | Value |
|---|---|
| Claim record | `{state, at, attempt, owner}` — `owner` is the coordinator instance id |
| Stale threshold | 10 minutes, for the sweep only. Every effect is a single-store write taking milliseconds, so a claim older than this has certainly lost its owner |
| Sweep cadence | the existing 2-minute stale-task detector (V46) — no new scheduler |
| Ownership | **A Pub/Sub redelivery always proceeds immediately** — it carries the payload and every write is CAS-guarded, so it cannot corrupt; it may simply find the effect already done. Waiting out a lease would be an unbounded wait against a 60s redelivery (V22). The threshold governs only the *sweep*, which takes over claims whose owner is demonstrably gone |
| Attempt bound | 3. On the third failure the row goes to `state: failed` — a terminal, operator-visible state, never a silent skip |

**Leases are an efficiency measure here, not a safety one.** Because every effect is idempotent
(table above) and every status write is CAS-guarded, a takeover that races the original owner either
re-applies a no-op or is recorded as `superseded`. That is
the property doing the work; ownership only avoids wasted calls. A `pending` row is therefore never
stranded (the sweep collects it) and never silently dropped (attempts are bounded and terminal
failure is recorded).

| # | Effect | Store | SQLite today | **Firestore today (prod)** | Safe? | Primitive needed |
|---|---|---|---|---|---|---|
| 1 | Mark task status | coordinator | `UPDATE tasks SET status = ?` — absolute (V33) | `Doc.Update` with absolute field values (V8) | **Yes** | — |
| 2 | Post completion message | messaging | bare `INSERT` → duplicate errors (V34) | `Doc(id).Set` → **silently overwrites** (V34fs) | **No** (both, differently) | `PutMessageIfAbsent` |
| 3 | `CreateApprovalRequest` | coordinator | bare `INSERT` → UNIQUE violation (V31) | `Doc(id).Set` → **overwrites; resets a resolved approval to `pending`** (V31fs) | **No** (both, differently) | `CreateApprovalIfAbsent` |
| 4 | Embed `handoff_targets` | — | pure computation over the registry (V2); written inside effect 3 | same | **Yes** | — |
| 5 | Handoff dispatch | messaging | as effect 2 (V34) | as effect 2 (V34fs) | **No** | `PutMessageIfAbsent` |
| 6 | Stage status | observatory | `SET status = ?` — absolute (V33) | absolute `Update` | **Yes** | — |
| 7 | Chain status | observatory | `SET status = ?, completed_at = COALESCE(?, completed_at)` (V33) | absolute `Update` | **Yes** | — |
| 8 | Stage + chain metrics | observatory | `SET cost = cost + ?` — increment (V32) | `firestore.Increment(cost)` — **increment** (V32fs) | **No** (both, identically) | `SetStageMetrics` / `SetChainMetrics` |
| 9 | Stage session | observatory | `SET session_id = ?` — absolute (V33) | absolute `Update` | **Yes** | — |
| 11 | Approval diff | coordinator | carried inside effect 3's `context_json` | same | **Yes** (with effect 3) | — |
| 12 | Cleanup (daemon only) | filesystem | removing an absent directory is a no-op | n/a | **Yes** | — |

**Both backends were checked; they do not behave identically.** SQLite errors where Firestore
overwrites — so the same three primitives are needed on both, but for opposite reasons: on SQLite to
avoid a crash loop, on Firestore to avoid silently clobbering state that has since moved on (a
resolved approval, a read message). That asymmetry is precisely the kind of thing that produced this
doc's original defect, and it is why the primitives must be defined by **behaviour**, not by
whichever backend was inspected first.

**`SetChainMetrics` must be a single atomic recompute — not a read-modify-write, and not a
read-then-write.** Computing `new_total = current + delta` in the application reproduces V32's
double-count across a crash; deriving `SUM(stages)` and *then* writing the chain total in a separate
operation lets two concurrent finalizers compute different snapshots, with the stale one landing last.
Both are ruled out:

- **SQLite:** one statement, atomic by construction, covering **all three** metrics V32 named —
  `UPDATE execution_chains SET (total_cost, total_tokens, total_turns) = (SELECT COALESCE(SUM(cost),0), COALESCE(SUM(tokens_in)+SUM(tokens_out),0), COALESCE(SUM(turns),0) FROM chain_stages WHERE chain_id = ?) WHERE id = ?`
  (row-value assignment, SQLite ≥ 3.15). r9's version summed only `total_cost` and would have left
  tokens and turns on the unsafe increment path.
- **Firestore:** there is no such construct, so the read of the stage docs and the write of the chain
  doc **must** happen inside `RunTransaction` (V26), which gives read-then-write atomicity with
  automatic retry under contention. This is the one place the design uses a transaction, and it is
  within a single store.

Stage metrics need neither: they are an absolute set of the task's own values, immutable once the run
has finished.

Effect 10 is absent by design — see below.

**Seven of eleven effects are already idempotent on both backends; three writes are not, and M1 adds
exactly three primitives** (`CreateApprovalIfAbsent`, `PutMessageIfAbsent`, absolute metric setters),
each implemented and tested in **both** backends. The increment-based metric writers stay for their
existing callers; finalisation uses the absolute setters.

**No outbox, no lease, no two-phase commit** — none is needed once every effect is idempotent and the
one non-idempotent effect is out of scope (below). Where a store *can* commit an effect and its ledger
row together (Firestore, V25/V26), it may do so as an optimisation, but **correctness does not depend
on it**, so the two paths stay identical in behaviour.

**Effect 10 leaves scope.** `ProcessStageCompletion` reaches twelve `PostCommentInRepo` calls with no
dedup and no marker (V29) — genuinely non-idempotent, and the only effect that escapes the store.
r4 tried to contain it with an outbox and a marker probe; gpt5-6-sol showed one ledger row and one
marker cannot cover twelve distinct POSTs, and gemini-3-1-pro showed none of that work was scoped.
**It is removed rather than patched, on evidence:** effect 10 already returns at
`stage_execution.go:437` for every message-sourced task (`GithubIssue == 0`, V29b) — i.e. for the
entire package-ecosystem path this doc exists to enable. It stays daemon-only and unchanged, exactly
as today; making it cloud-safe is its own doc (see Non-Goals). This is status quo, not a regression.

**Tests:** a per-effect retry arm asserting a second application changes nothing — covering **all
eleven effects**, not only the four r5 listed; crash injection on both sides of every ledger write;
and a concurrent daemon/Pub-Sub finalisation test proving convergence to exactly one approval, one
stage transition, one metrics value and one handoff. The metrics arm must assert the **value**, not
merely that a write happened — V32's double-count is invisible to a write-count assertion.

**C2 — Strategy boundaries, not empty-string branching.** Resolved by D1 + V21: every strategy
implements every method; a cloud no-op is written as a no-op, not inferred from an empty field.

**C3 — #921 was written for the daemon path.** Persisting the diff at creation time fixed an approval
card that rendered a confident "Files (0)" — and Mark approved two merges blind off it. Creating cloud
approval records **without** a diff reintroduces that failure on a path where approval also releases a
handoff. **M3 is a prerequisite for D2, not polish.**

**C4 — Cost double-count.** Closed by V19: one writer, identified by file:line.

**C5 — Chain volume changes the cost profile.** A 4-stage pipeline is 4 dispatched tasks. At the 2h
ceiling that is up to 8h wall-clock and 4× per-message spend. Unsupervised operation needs a
chain-level budget, which does not exist today.

**C6 — Backend selection can erase the chain model silently.** V20: the GCP backend no-ops every
chain/stage write and returns no error. Out of scope to fix here, but M5 should assert the configured
backend implements chain writes, so this cannot regress into the path M1 depends on.

---

## Solution Design

### M1 — One finalisation path, with a ledger (D1, C1)

`internal/coordinator/task_finalize.go`: the orchestrator, the `ExecutionStrategy` interface, and the
finalisation ledger. Both call sites route through it. Idempotency is enforced inside, per effect —
never by callers.

**In M1 (r6), because none of them exists today:** `CreateApprovalIfAbsent` (V31),
`PutMessageIfAbsent` (V34) and absolute `SetStageMetrics` / `SetChainMetrics` (V32) — each with a
SQLite and a Firestore implementation and a retry test asserting a second application changes
nothing. Without these the ledger is decorative: r5's protocol would have crash-looped on approvals
and double-counted every retried cost.

**Not in M1 (r5):** a cross-store `FinalizationTx`. V30 showed the store methods take no
transaction and V28 showed the daemon's three databases cannot share one anyway — so M1 instead
makes each effect idempotent (C1) and keeps the ledger in the single store that already holds the
task. Firestore may still commit an effect with its ledger row atomically as an optimisation, but no
test asserts it and no behaviour depends on it.

### M2 — Cloud stages advance (V15, V19)

Effects 6–9 move into the orchestrator so a cloud task's stage leaves `pending` and carries metrics.
**Cost routes through the absolute setters C1 introduces — not through `updateChainMetrics`.** V19
identified that function as the *only* current writer, which is why it was easy to reason about; V32
then showed it **increments**, so using it would double-count on redelivery. r12 said "route cost
through the single writer identified in V19… no double-count risk", which contradicted C1 outright.
`ArtifactSource` still returns `ErrNoArtifactSource` when the cloud executor reported no artifact
path — `cmd/ailang/coordinator_cloud.go:110` documents it "may be empty" (V27) — and the orchestrator
logs that explicitly rather than skipping silently. (Its only consumer, effect 10, is out of scope in
r5; the explicit-error contract stays so the next consumer inherits it.)

**Acceptance:** a cloud task's stage reaches a terminal status with non-zero metrics — which has never
happened.

### M3 — A cloud approval carries its diff (C3, #921 parity)

**The coordinator does not compute the diff — the executor does, because it is the only component
with a git tree.** The cloud service has no clone and passes `""` for `worktreePath` (V37), so
`GetWorktreeDiff` is not available to it at any price. The executor already runs
`discoverChangedFilesFromCommit(workDir, clonePoint)` and publishes `ChangedFiles` (V39).

M3 therefore extends the completion payload rather than adding a fetch:

- `BaseCommit` (= `clonePoint`, already in hand) and `HeadCommit` — **which requires new code**: the
  push at `coordinator_cloud.go:529` captures no SHA today, so M3 adds a `git rev-parse HEAD`
  immediately after a successful push (V40). Two immutable SHAs, so the value is identical on replay
  (V35, V37).
- `DiffStat` and a capped `Diff`, computed by the same code path that already produces
  `ChangedFiles`.

The cloud `DiffSource` then reads these fields. If any is missing, the approval records an explicit
`diff_unavailable` reason — it never renders a confident "Files (0)", which is the failure #921 was
written for (C3).

### M4 — Reconcile the leak (D3)

One-shot termination of the 315 pre-fix `active` chains with an explicit reason, plus a recurring
check flagging any chain still `active` whose task is terminal — so the next occurrence is caught in
hours, not four months.

### M5 — The anti-drift arm

A test enumerating the effects the orchestrator performs, asserting both call sites route through it,
and asserting the configured backend implements chain writes (C6). **This defect existed because
nothing could see the two paths side by side.**

### M6 — The chain list view (O1, separate)

Fix `stage_count` / `agent_flow` under-reporting in the list endpoint. Scoped separately and
explicitly **not** a dependency of M1–M5, because conflating a read-path defect with the write-path
work is what produced r1's false headline.

### Files to Modify/Create

- `internal/coordinator/task_finalize.go` (**new**) — orchestrator + strategy interface — M1
- `internal/coordinator/task_finalize_ledger.go` (**new**) — per-effect idempotency — M1/C1
- `internal/coordinator/store_sqlite.go` — `ALTER TABLE tasks ADD COLUMN finalization TEXT` migration (V41)
- `internal/coordinator/store.go` + `internal/storage/firestore/coordinator_convert.go` — `Finalization` on `TaskRecord` and **both** converter directions, or the ledger is silently dropped on read (V44)
- `internal/coordinator/task_finalize_test.go` (**new**) — M5 + crash-injection + concurrency arms
- `internal/coordinator/daemon_tasks_exec_run.go` — call the orchestrator with the daemon strategy
- `internal/coordinator/pubsub_completion_handler.go` — call it with the cloud strategy
- `internal/coordinator/task_finalize_diff.go` (**new**) — M3 cloud `DiffSource`
- `internal/pubsub/topics.go` + `cmd/ailang/coordinator_cloud.go` — add `BaseCommit`, `HeadCommit`, `DiffStat` and a capped `Diff` to `TaskCompletion`, published by the executor beside the `ChangedFiles` it already sends (V35, V37, V39); **including a `git rev-parse HEAD` after the push at `coordinator_cloud.go:529`, which captures no SHA today** (V40)
- `internal/coordinator/store_sqlite_approvals.go` + `internal/storage/firestore/coordinator_approvals.go` — `CreateApprovalIfAbsent` (V31)
- `internal/messaging/inbox.go` + `internal/storage/firestore/messaging_inbox.go` — `PutMessageIfAbsent` (V34/V34fs/V38)
- `internal/observatory/store_chains.go` + `internal/storage/firestore/observatory_chains.go` — absolute metric setters (V32)
- `internal/coordinator/chain_reconcile.go` (**new**) — M4

---

## Testing Strategy

Each arm fails RED before its fix.

1. **C1 idempotency:** finalise the same task twice → exactly one approval, one stage transition, one
   metrics contribution, one handoff. RED today (V23: `pending_approval` is not terminal).
2. **C1 crash injection:** interrupt after each ledger effect; assert retry converges and no effect is
   skipped. RED today (no ledger exists).
3. **M1 handoff on the cloud path:** a completion for an agent with `trigger_on_complete` produces a
   `merge_handoff` approval with the right targets. RED today.
4. **M2 stage progression:** after a cloud completion, the stage is terminal with non-zero metrics.
   RED today (V15: frozen at `pending`).
5. **M3 diff presence:** a cloud approval record carries non-empty `changed_files`. RED today.
6. **M5 parity + backend capability:** both call sites route through the orchestrator; the configured
   backend implements chain writes (C6).
7. **Live prod proof (the only one that counts):** send a real message to `design-doc-creator`, approve
   the merge, observe a `sprint-planner` task with the correct `parent_task_id` and `chain_id`, and a
   chain whose stages read `design-doc-creator -> sprint-planner`. **Verify the artefact, not the
   pipeline** — and per the Corrections above, **not the aggregate either**: read the chain detail, not
   the list.

---

## Success Criteria

- [ ] A cloud task and a daemon task produce identical completion effects **for every effect in scope** — i.e. the D1 outcome matrix, which excludes effect 10 by name (M5 proves it mechanically)
- [ ] A prod chain advances past stage 1 — the first time in the system's history
- [ ] The full `design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator` chain
      completes once in prod on real work, artefact landing in the correct repo
- [ ] Redelivery of a completion changes nothing the first delivery already did
- [ ] No chain remains `active` more than 1h after its task reaches a terminal state
- [ ] Every cloud approval card shows a real file list before a human approves it

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| **Duplicate handoff on redelivery** (C1, V22/V23) | Every effect made individually idempotent (absolute sets, deterministic ids), with a per-effect ledger on the task record for reconciliation. **No outbox, no lease, no cross-store transaction** — none is needed once the one non-idempotent effect is out of scope (V29/V29b). Crash-injection and concurrent-finalisation arms |
| **Blind approval releasing a handoff** (C3) | M3 lands before D2 widens any auto-approved edge |
| **Unsupervised chain runs away** (C5) | `max_concurrent_tasks: 1` is already set on all three pipeline agents. A chain-level budget does **not** exist and is **not** scoped here (C5) — naming it as a mitigation would be false comfort; it is listed as a prerequisite for unsupervised operation instead |
| **Extraction changes daemon behaviour** | The daemon path is the *working* one; M5 pins its current effect set **before** extraction, so the refactor is provably behaviour-preserving there |
| **Silent backend swap erases chains** (C6, V20) | M5 asserts the configured backend implements chain writes |
| Prod images lag the fix | Coordinator-side change, so it ships on the coordinator image — but per [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) Rollout, verify the **deployed revision**, not the build status |

---

## Non-Goals

- **Reply-to-sender routing.** Completions land in the agent's own inbox and correlate home only via
  `correlation_id`. A real gap, a different doc.
- **The tier-1 permission work** — the sibling doc owns it; D4 only asks whether pipeline agents opt in.
- **Making `ProcessStageCompletion` cloud-safe** (effect 10). It has twelve non-idempotent comment
  POSTs with no dedup or marker (V29) and needs a per-operation outbox with leases and marker probes
  — real work, and worthless to the ecosystem path, which never reaches it (V29b). It stays
  daemon-only and unchanged, as today. **Own doc.**
- **Fixing the GCP backend's silent no-ops** (V20) — named, not fixed; M5 only prevents it reaching
  the path M1 depends on.
- **Any change to `trigger_on_complete` semantics.** The topology model is correct (V2) and stays.

---

## Quorum

**Round 0 (2026-09-03) — BLOCKED 3/3.** All three objections accepted, none argued.
Artifact: `.ailang/state/mission-quorum/m-completion-path-parity-2026-09-03T06-07-31Z.json`

| Reviewer | Objection | Disposition in r2 |
|---|---|---|
| gpt5-6-sol | `handoffs_triggered` cannot provide exactly-once across all finalisation effects; A6's "no concurrency change" is unsupported | **Accepted.** V4 re-scoped, V22/V23 added, C1 redesigned as a per-effect ledger with a transactional outbox, A6 raised to +1 with justification |
| gemini-3-1-pro | V19 deferred root-causing an active data-mutation path to implementation | **Accepted.** V19 resolved: `UpdateChainMetrics` via `daemon_tasks_chain.go:67`, one call site, daemon-only |
| oc-glm-5-2 | The ten effects were never classified worktree-dependent vs not; `WorktreePath: ""` branching is a silent fallback; M3 forks rather than unifies | **Accepted.** V21 classifies all 12 effects (9 portable, 2 strategy, 1 daemon-only); D1 restructured as orchestrator + strategy interface; empty-string branching withdrawn |

**Round 1 (2026-09-03) — BLOCKED 3/3.** All three accepted, none argued.

| Reviewer | Objection | Disposition in r3 |
|---|---|---|
| gpt5-6-sol | "each claimed transactionally" is not an implementable protocol without naming the datastore and transaction API per effect; claim-before-effect skips, effect-before-claim duplicates | **Accepted.** V25/V26/V28 added — the cloud path turns out to be **single-store**, so C1 is now an effect-by-effect table naming datastore, idempotency key, atomic unit and recovery query, with claim+effect in one `RunTransaction` and the one external effect (GitHub) shown to be idempotent on the far side |
| gemini-3-1-pro | D1's "effects 1–10 unconditionally" contradicts V21, which classifies effect 10 as strategy-supplied | **Accepted.** D1 now reads 1–9 unconditional, 10–12 delegated |
| oc-glm-5-2 | `Cleanup`'s cloud no-op is a silent fallback dressed as an interface method, so r2's Hard Violation Check was false by construction; and `ArtifactGCSPath` was asserted available without a data claim | **Accepted.** `Cleanup` removed from the interface in favour of an explicit `Kind()` branch with a logged cloud case; `ArtifactSource` returns `ErrNoArtifactSource`; V27 records that the code itself documents the path "may be empty", and M2 now logs an explicit reason |

**Round 2 (2026-09-03) — BLOCKED 3/3.** All three accepted, none argued. Two converged on the same
unverified premise, which turned out to be false.

| Reviewer | Objection | Disposition in r4 |
|---|---|---|
| gpt5-6-sol | V25/V26 prove co-location and a generic entry point, not that the store methods can *participate* in a caller-supplied transaction; the atomic units may not exist | **Accepted — and they do not.** V30 lists the three signatures, none of which takes a transaction. `FinalizationTx`/`FinalizationStore` added to M1 with Firestore **and** SQLite implementations and abort-atomicity tests; estimate raised 3d → 5d |
| gemini-3-1-pro | C1 asserted effect 10's GitHub surface is an idempotent label write with no V-log entry proving it | **Accepted.** V29: 12 non-idempotent `PostCommentInRepo` calls, no dedup, no marker; only `AddLabel` is idempotent |
| oc-glm-5-2 | C1 ("no outbox") directly contradicts the Risks table ("transactional outbox"), and the resolution hinges on the unverified GitHub idempotency claim | **Accepted.** Contradiction resolved in favour of: no outbox for in-store effects, **outbox with a marker probe for effect 10 alone**, justified by V29. Both C1 and the Risks row now say the same thing. V29b additionally bounds it — effect 10 is inert for message-sourced ecosystem tasks |

**Round 3 (2026-09-03) — BLOCKED 3/3.** All three accepted, none argued. The round's net effect was
to make the design **smaller**.

| Reviewer | Objection | Disposition in r5 |
|---|---|---|
| oc-glm-5-2 | V28 asserted SQLite co-residency with only a directory-name inference, at the same confidence as V25's file:line proof — the exact mistake V30 had already burned | **Accepted; V28 REFUTED.** `NewSQLiteBackends` opens `coordinator.db`, `collaboration.db`, `observatory.db` — three `*sql.DB`. Cross-store atomicity dropped as a design basis; C1 rebuilt on idempotent effects, which works identically on both paths |
| gpt5-6-sol | One ledger row and one marker cannot cover twelve non-idempotent POSTs; concurrent finalizers both see `pending`, probe absent, and duplicate | **Accepted.** Rather than add per-operation rows and leases, effect 10 leaves scope (V29b): it is already inert for every task this doc serves |
| gemini-3-1-pro | The marker-probe work appears in no milestone and no file list — un-scoped vaporware | **Accepted.** Correct, and resolved by removal rather than by scoping work the ecosystem path never executes |

**Round 4 (2026-09-03) — BLOCKED 3/3.** All three converged on the same defect, and were right:
r5's idempotency claims were asserted, not verified. Three of four were false.

| Reviewer | Objection | Disposition in r6 |
|---|---|---|
| gemini-3-1-pro | Named the three specific DB behaviours C1 assumed — absolute metrics, graceful duplicate approvals, collision-safe message ids — and asked for proof | **Accepted; all three checked, all three false.** V31 (bare approval INSERT), V32 (both metric writers increment), V34 (bare message INSERT). Predicted the double-count exactly |
| oc-glm-5-2 | Effect 5's collision-idempotency was unverified; V4 covers only the catch-up query, not insert semantics | **Accepted.** V34 confirms a caller-supplied id is accepted but the INSERT has no conflict clause; `PutMessageIfAbsent` added to M1 |
| gpt5-6-sol | Same unverified-idempotency point, plus: the C1 table omitted effects 1, 2, 4, 9, 11 and 12 although the orchestrator runs them | **Accepted.** The table now covers all eleven, each with its verified mechanism, and the test strategy covers all eleven rather than four |

**Round 5 (2026-09-03) — BLOCKED 3/3.** All accepted. The round's headline finding was a
methodological one, and it was fair.

| Reviewer | Objection | Disposition in r7 |
|---|---|---|
| oc-glm-5-2 | V31/V32/V34 verify only SQLite — the daemon path — while the cloud path this doc exists to fix uses Firestore, which was asserted, not checked | **Accepted, and the miss was the doc's worst.** V31fs/V32fs/V34fs added: Firestore `Set`s rather than erroring, so the risk is a **resolved approval reset to `pending`**, not a crash. Metrics double-count on both. The C1 table now has a column per backend |
| gemini-3-1-pro | An application-level read-modify-write `SetChainMetrics` reproduces the double-count across a crash; also `execution_chains` has no `reason` column, and Risks cites a chain-level budget that does not exist | **Accepted on all three.** `SetChainMetrics` now derives the total in-store from `SUM(chain_stages.cost)`; V36 records the missing column and M4 carries the migration; the phantom budget mitigation is struck |
| gpt5-6-sol | M3's diff source is nondeterministic — no proof cloud finalisation has repo identity and immutable base/head commits, and a branch can move between attempts | **Accepted.** V35: `TaskCompletion` has `BranchName` and no SHA. M3 now adds `HeadCommit` to the payload and diffs SHA..SHA, with an explicit `diff_unavailable` reason if either is missing |

**Round 6 (2026-09-03) — BLOCKED 2/3; gpt5-6-sol absent (budget), quorum degraded to N-1 and
recorded by name.** Both present objections accepted. Both were narrow — the series has moved from
"the architecture is wrong" to "you cited the wrong filename", which is what convergence looks like.

| Reviewer | Objection | Disposition in r8 |
|---|---|---|
| gemini-3-1-pro | M3's "`base_commit` already recorded at dispatch" is a new unverified data claim — the same error as r2's `ArtifactGCSPath` | **Accepted, and the check found worse.** V37: `base_commit` is worktree-derived and the cloud handler persists `""` for every task. M3 now has the executor publish **both** SHAs |
| oc-glm-5-2 | The Files list points `PutMessageIfAbsent` at `observatory_messages.go`, while V34fs cites `messaging_inbox.go` — the plan targets a file the V-log never read | **Accepted.** V38 records that the two files are different subsystems (inbox writer vs observatory mirror); the Files list is corrected |
| gpt5-6-sol | — | **Absent: per-reviewer budget cap.** Re-run at a raised cap so round 7 is a full quorum |

**Round 7 (2026-09-03) — BLOCKED 3/3, full quorum** (gpt5-6-sol present at the raised cap). All accepted.

| Reviewer | Objection | Disposition in r9 |
|---|---|---|
| gemini-3-1-pro | The cloud coordinator has no clone or worktree, so it cannot compute a git diff at all; the executor is the component that can | **Accepted — and it makes M3 smaller.** V39: the executor already runs `discoverChangedFilesFromCommit(workDir, clonePoint)` and publishes `ChangedFiles`. M3 now extends that payload with `BaseCommit`/`HeadCommit`/`DiffStat`/capped `Diff`; the coordinator only reads |
| gpt5-6-sol | `SetChainMetrics` derives a sum then writes separately — two finalizers can snapshot differently and the stale writer wins, contradicting A6 | **Accepted.** The recompute is now atomic per backend: one `UPDATE … (SELECT SUM …)` statement on SQLite; inside `RunTransaction` on Firestore (V26) |
| oc-glm-5-2 | The doc claimed r8 while round 7 was pending and round 6 was degraded to N-1 — the load-bearing V37/V38 changes had never faced a full quorum | **Accepted.** Round 7 has now run with all three present; the header records the last completed full-quorum round, and the revision counter will not lead it again |

**Round 8 (2026-09-03) — BLOCKED 3/3, full quorum.** All accepted. Every objection was narrow and
mechanical; none touched the architecture.

| Reviewer | Objection | Disposition in r10 |
|---|---|---|
| gpt5-6-sol | V33's Firestore column ("absolute Update") was asserted — only SQLite was cited — while effects 6, 7 and 9 depend on it on the production backend | **Accepted; checked; the claim holds.** V33fs cites the three Firestore call sites, none using `Increment` |
| gemini-3-1-pro | The SQLite recompute summed only `total_cost`, leaving `total_tokens` and `total_turns` on the unsafe increment path | **Accepted.** Row-value assignment now covers all three |
| oc-glm-5-2 | `HeadCommit` "captured at push" repeats r7's mistake; and the ledger's home in the task record is unverified | **Accepted; both were gaps.** V40: the push captures no SHA, so M3 adds `git rev-parse HEAD` and the push file joins the Files list. V41: the `tasks` table is column-per-field, so M1 carries an `ALTER TABLE … ADD COLUMN finalization TEXT` migration |

**Round 9 (2026-09-03) — BLOCKED 3/3, full quorum.** All accepted. One was a latent correctness bug
in the proposed design, not merely an unverified claim.

| Reviewer | Objection | Disposition in r11 |
|---|---|---|
| gpt5-6-sol | "Effects 1–9 unconditionally" ignores that the daemon's call sites sit in different outcome branches; extracting them so would create approvals and fire handoffs for failed or cancelled tasks | **Accepted — the most consequential catch since round 0.** V43 maps the real control flow; D1 now carries an outcome × effect matrix, and that matrix is the parity contract M5 asserts |
| gemini-3-1-pro | "Firestore needs no migration" is false at the Go layer — a field missing from `TaskRecord`/its converter is silently dropped, so every redelivery would read an empty ledger | **Accepted.** V44 cites the hand-written converter in both directions; `TaskRecord.Finalization` and both converter entries added to the Files list |
| oc-glm-5-2 | "The executor already computes the diff" conflates file discovery with diff generation | **Accepted.** V42: `discoverChangedFilesFromCommit` returns `[]string`. M3 adds `git diff --stat` and a capped `git diff`; it does not shrink |

**Round 10 (2026-09-03) — BLOCKED 2/3. oc-glm-5-2 PASSED — the first pass in the series.** Both
remaining objections converged on the same five lines, and both were right.

| Reviewer | Objection | Disposition in r12 |
|---|---|---|
| gemini-3-1-pro | The matrix defers effect 5 to "on approval" for normal completions, but V43 showed `handleAgentHandoffs` runs in the general success arm — silently changing the behaviour r11 claimed to preserve column-for-column | **Accepted.** V45 pins the exact placement (`:629` closes the inner block, `:633` calls, `:637` opens the failed arm). The matrix now reads: auto edges at completion for both sub-branches, non-auto on approval |
| gpt5-6-sol | C1 gave no bounded recovery protocol — no stale threshold, sweep cadence, ownership rule, attempt bound or terminal failure state | **Accepted.** All five specified, reusing the 2-minute stale-task detector already running in prod (V46), plus the explicit argument that leases are an efficiency measure and idempotency is the safety property |
| oc-glm-5-2 | **PASS** — raised the same `handleAgentHandoffs` placement question as its strongest objection while passing | Resolved by V45 |

**Round 11 (2026-09-03) — BLOCKED 3/3, full quorum.** Two objections were the doc contradicting
itself; one was the last unverified prod-config claim.

| Reviewer | Objection | Disposition in r13 |
|---|---|---|
| gemini-3-1-pro | M2 routes cost through V19's writer while V32 proves that writer increments — reintroducing the A6 violation C1 exists to fix | **Accepted.** M2 now names the absolute setters explicitly and records why V19's writer must not be used |
| gpt5-6-sol | Goals and Success Criteria claim identical cloud/daemon effects while effect 10 is deliberately daemon-only; V29b bounds the consequence, not the claim | **Accepted.** Goal 1 and the success criterion now state the exception rather than eliding it |
| oc-glm-5-2 | `max_concurrent_tasks: 1` was asserted as a live safety mitigation and never verified — the same class as V27 and V37 | **Accepted; checked; it holds.** V47 cites `:111`, `:139`, `:179`. The claim survived, but it should not have been resting on assertion in a risk row |

**Round 12 (2026-09-03) — BLOCKED 3/3, full quorum.** Two objections found real holes in C1's
safety argument; the third demanded verification of a matrix cell and surfaced a missing effect.

| Reviewer | Objection | Disposition in r14 |
|---|---|---|
| gpt5-6-sol | An absolute status write is repeatable but not idempotent once another step has advanced the record — a stale replay regresses `approved` back to `pending_approval` | **Accepted; a real hole.** Status effects are now compare-and-set against the expected prior state, values are ordered so writes only advance, and a non-matching CAS records `superseded` |
| gemini-3-1-pro | The stale-task detector has no `TaskCompletion` payload so it cannot execute the effects it "takes over"; and making a 60s redelivery wait out a 10-minute lease is an unbounded wait | **Accepted; both.** The claim now persists the payload, and a redelivery proceeds immediately — safe because every write is CAS-guarded. The threshold governs the sweep only |
| oc-glm-5-2 | The `failed` and `no_changes` matrix cells were asserted, not verified | **Accepted.** V48: the `failed` cell was right, but the check found **effect 13 (`updateStageError`) missing from every matrix in thirteen revisions**, and showed the daemon has no `no_changes` arm at all — so that column defines behaviour rather than preserving it, and now carries ruling D5 |

**Round 13**: pending.

---

## Related Documents

- [M-COORDINATOR-EXECUTION-TRUST](m-coordinator-execution-trust.md) — the layer below; made execution
  trustworthy. This doc fixes what happens *after* a cloud task finishes.
- [M-MESSAGE-PLANE-TRUST](m-message-plane-trust.md) — delivery, the layer below that.
- [M-PACKAGE-PROTOCOL-MANIFESTS](m-package-protocol-manifests.md) — per-repo protocol declaration the
  ecosystem pipeline will need once it runs.
