# M-TASK-STATUS-TRUTH: a task's recorded outcome is what actually happened, and a handoff fires once

**Status**: PLANNED 2026-09-25 (attended). Implementation follows in the same session.
**Priority**: P0 for the message plane. Delivery is now trustworthy; the record of what delivery
*produced* is not, and every downstream reader (approvals, chains, ELO, the pipeline view) reads the record.
**Parent**: [m-message-plane-trust.md](m-message-plane-trust.md) — this is its D2 ("a dispatched job MUST
reach a terminal state") taken one step further: the terminal state must also be the *true* one.

---

## Problem statement

Measured against prod (`ailang-multivac`) coordinator logs, 2026-09-23 15:30 → 2026-09-25 10:40 UTC,
after the backstop sweep went to `dispatch`:

- **Delivery works.** 7/7 Pub/Sub messages became tasks in 1–34 s. `messages health --since 0`:
  0 routable-undelivered (was 22 in the prior 24 h).
- **The record lies in three independent ways**, and a fourth leaks work:

| # | Defect | Measured | Root cause (file) |
|---|---|---|---|
| S1 | Sweep-recovered tasks are marked failed ~12 s after dispatch; their real completion is then dropped | 3/3 on 09-23. `task-03bd17ad`: dispatched 15:23:37, "timed out (age=154h)" 15:23:49, `status=completed` arrived 15:26:22 → "already in terminal state failed, skipping". It had opened PR #1283. | Task inherits `msg.CreatedAt` (`backstop_sweep.go:202`, `daemon_tasks_polling.go:158/209/467/513`). A cloud task stays `queued` for its whole run (`MarkTaskRunning` is local-only), so `getTaskAge` (`stale_task_detector.go:233`) falls through to `CreatedAt` = the **message's** age. The timeout text says "of being queued"; nothing records when that was. |
| S2 | A `blocked` outcome is dropped, then reported as a 3 h timeout | `task-91337be6`, `task-95727fea`: "unknown completion status \"blocked\"" (15:25, 15:29) → "timed out … 3h0m0s" (18:26) | Producer learned `blocked` on 2026-09-14 (`coordinator_cloud.go:285`, `task_blocked.go`); the consumer's `completionOutcome` (`task_finalize_cloud_strategy.go:43`) and the finalize matrix (`task_finalize.go:82`) never did. Same concept, two implementations, one updated. |
| S3 | **Every approval with a handoff fires it twice** | 4/4 sampled (09-14, 09-16): a second `Handoff:` message 30–80 min after the first, at the next coordinator boot. | `dispatchApprovalHandoffs` (`approval_handoff.go:391`, the approve path since M-COMPLETION-PATH-PARITY) never calls `MarkApprovalHandoffsTriggered`. Boot-time `triggerMissedHandoffs` (`daemon.go:602`) then finds `handoffs_triggered == false` and fires again. Prod scale-to-zero reboots every ~1.5 h, so this is not a crash path — it is every approval. |
| S4 | `--landed --apply` printed "handoffs NOT fired" and 22 handoffs fired anyway | 09-23 15:17 boot: "Found 22 approved merge_handoff approval(s) without triggered handoffs" → 22 tasks, 17 ran sprint-planner on stale work (rejected 09-25). | `SkipHandoffs` (`approval_processor.go:214`) records nothing, so the same boot recovery cannot tell "deliberately suppressed" from "crashed before firing". Also: SQLite's recovery query is bounded to 7 days, Firestore's (`coordinator_approvals.go:198`) is unbounded — prod re-fires approvals of any age. |

**One pattern.** Each is a decision made in one place and read in another, with nothing recording
the decision: when a task was *queued*; that an outcome is *blocked*; that handoffs were *fired*
or *suppressed*. The reader then substitutes a default (message age / "unknown, ignore" / "never
fired") and acts on it.

## Design decisions

### D1 — The stale clock starts when the task is claimed, not when the message was written (S1)

Add `queued_at` to the task record, stamped by `MarkTaskQueued` in the **same write** that claims
the task (Firestore transaction; SQLite compare-and-set). `getTaskAge` order becomes
`StartedAt → QueuedAt → CreatedAt`.

`CreatedAt` keeps meaning "when the work was requested": `task_dedup.go` scopes by it, and the
landed-card cutoff (6540b5f6a) filters on it. Rewriting it to "now" would fix S1 and break both.

Rejected: stamping `started_at` at claim. It already has a meaning (execution began, shown in the
observatory) and `ResetTaskToPending` clears it; overloading it is the two-meanings-one-field shape
this doc exists to remove.

### D2 — `blocked` is a first-class finalize outcome (S2)

Add `OutcomeBlocked` to the finalize matrix, with the row the status tables already declare
(`task_status.go:66/90`): task → `blocked` (terminal), stage → failed, chain → failed, metrics
recorded, **no approval, no handoff** (the agent did not attempt the work; nothing to approve or
build on). `completionOutcome` maps `blocked` to it.

**Contract, not a case:** one exported list of the statuses the executor may publish, used by the
producer, and a test that every member is accepted by `completionOutcome`. Adding a status to the
producer without the consumer then fails a test instead of dropping completions in prod.

### D3 — A handoff is identified by (task, target), and every producer writes it first-write-wins (S3, S4)

*Revised after quorum round 1 (gpt6-astra): a boolean "handoffs fired" marker written after sending
cannot give exactly-once — a crash between send and marker replays the send, and a partial failure
across several targets either replays the ones that succeeded or loses the rest.*

The identity already exists: `HandoffMessageID(task, target)` (`task_finalize_approval.go:34`), used
by the completion path with `PutMessageIfAbsent` (first write wins; SQLite `ON CONFLICT DO NOTHING`,
Firestore `Create`). **Four** producers of the same event do not use it — the approve path
(`dispatchApprovalHandoffs`), the GitHub-label path (`task_chain.go:441`), the daemon auto-edge path
(`daemon_approval.go:82`), and boot recovery (`triggerHandoffsFromApprovalRecord`, which also builds
its own body and never notifies). All four route through the one `handoffSender`, which now:

1. writes the message under `HandoffMessageID(task.ID, target.ID)` with `PutMessageIfAbsent`;
2. notifies **only if it created the row**.

Every crash point is then safe: before the write → the retry writes it; after the write, before the
notify → the row exists unread and the backstop sweep (now `dispatch`) delivers it; after the notify →
any replay collides and does nothing. A partial failure retries exactly the missing targets.

The `handoffs_triggered` marker becomes an optimisation (stop re-scanning), no longer load-bearing
for anything that *fires*.

**Suppression is the one load-bearing marker, so it is written atomically with the approval**
*(quorum round 2, gpt6-astra)*. `SkipHandoffs` resolves the approval through
`ResolveApprovalSuppressingHandoffs`, which sets `status=approved`, `resolved_*`,
`handoffs_triggered=true` and `handoffs_suppressed=true` in **one** write — a single Firestore
document `Update` (`coordinator_approvals.go:167`) / a single SQLite `UPDATE`. There is no state in
which the approval reads `approved` and the suppression is absent, so no crash point lets boot
recovery fire a suppressed handoff. `handoffs_suppressed` exists for the audit trail; the recovery
query already excludes on `handoffs_triggered`.

**Structure, exactly** *(quorum round 4, oc-glm-5-3 — the doc named two functions without saying which
does what)*:

| Layer | Function | Does | Reached by |
|---|---|---|---|
| write | `handoffSender.send` (`approval_handoff.go`) | `PutMessageIfAbsent` under `HandoffMessageID(task, target)`; notify only if it created the row | **every** handoff producer except finalize: via `sendAgentHandoffMessage` (below) and via the daemon auto-edge entry `Daemon.sendHandoffMessage` (`daemon_approval.go:82`) |
| write | `finalizer.applyHandoff` (`task_finalize_approval.go`) | same `PutMessageIfAbsent`, id `HandoffMessageIDForWork(task, target, work)` with the work id computed from the completion's own diff (quorum round 8) | completion path, auto edges |
| approval-path entry | `sendAgentHandoffMessage` | reads `handoffs_suppressed`; refuses with `errHandoffSuppressed`; else calls `handoffSender.send` | approve (`dispatchApprovalHandoffs`), boot recovery (`triggerHandoffsFromApprovalRecord`), and `TaskChain.OnAgentApproved` — which has **no production caller** (V17) |

So one identity — `HandoffMessageIDForWork(task, target, work_id)`, degrading to `HandoffMessageID(task, target)` when no work id is known — and one write rule cover all producers; the suppression check covers every
producer that acts on an approval.

**No check-then-write race on suppression** *(quorum round 4, gpt6-astra)*. A race needs a producer
that sends for an approval it did not itself resolve, concurrently with that approval's resolution.
The live producers cannot: the approve path sends only after *winning* the resolve, and the resolve is
a compare-and-set on `status = 'pending'` in both stores — SQLite `UPDATE … WHERE status = 'pending'`;
Firestore re-reads the status **inside a transaction** before updating (added in this work: it was a
query-then-update, which two resolvers could both pass — quorum round 5). Exactly one resolver wins,
and it fixes `handoffs_suppressed` in that write. Boot recovery reads only *already-resolved* approvals, whose
suppression can never change after resolution (nothing writes `handoffs_suppressed=false`). The only
producer that could send for an approval it did not resolve is `OnAgentApproved`, which is unreachable
(V17); if it is ever wired, it must resolve first — recorded in D4.

**Recovery is bounded by resolving what it will not fire** *(quorum round 3, gemini-3-1-pro)*. Both
stores' recovery query now returns unmarked approved approvals **of any age** (SQLite's SQL-side
`created_at > now-7d` bound is removed — it hid stale rows from the expiry and made the stores diverge,
quorum round 4). `triggerMissedHandoffs` applies `HandoffRecoveryWindow` in one place: an approval
older than it fires nothing and is marked `handoffs_triggered=true, handoffs_expired=true`, so it is
fetched **at most once**. The candidate set holds only genuinely unresolved recent approvals and is
empty in steady state, since every approve branch now records its decision.

### D4 — Out of scope, recorded

- **A genuinely overrunning task's late completion is still dropped** after the detector fails it.
  With D1 this becomes rare (real overruns only). Accepting a late completion over a detector-imposed
  `failed` interacts with re-dispatch (attempt 2 may be running) — its own design.
- **The landed-card and stranded-approval sweeps never complete in prod** (75/75 `context canceled`):
  the instance lives ~15 min per wake and those sweeps first fire at shutdown. Needs a decision
  (run once on wake vs. min-instances) — separate doc.
- Cosmetic: `rejectd` in the remote reject output.
- `TaskChain.OnAgentApproved` (GitHub-label approvals) has no caller. If it is re-wired it must
  resolve the approval (CAS on pending) before sending, like the approve path — otherwise it is the one
  producer that could race a suppressing resolution.

## Milestones

| M | Scope | Proof |
|---|---|---|
| M1 | D1 `queued_at` (both stores, detector) | Unit: a task with `CreatedAt` 7 days ago and `QueuedAt` now is NOT stale; with `QueuedAt` past timeout it IS. Firestore convert round-trips the field. |
| M2 | D2 `OutcomeBlocked` + producer/consumer contract | Matrix row asserted behaviourally (status, stage, chain, no approval, no handoff). Contract test over the exported list. |
| M3 | D3 one handoff identity, first-write-wins; atomic suppression | Each asserts BOTH "exactly one row" AND "that row is delivered" — a test that passes on zero deliveries proves nothing *(quorum round 2, oc-glm-5-3)*. (a) approve → recovery replay → one row per target, notified once. (b) crash after write, before notify (simulate: `PutMessageIfAbsent` with no notify, no marker) → recovery → still one row, **and `BackstopSweep.SweepOnce` in `dispatch` mode enqueues it** into the drain. (c) partial failure (target 2 of 2 errors) → retry → target 1 not duplicated, target 2 written and notified. (d) `SkipHandoffs` → the approval row reads `approved` + `handoffs_suppressed` from the same read → recovery → zero rows. (e) recovery over an approval older than 7 days fires nothing and marks it expired; a second pass does not return it. (f) after `SkipHandoffs`, the GitHub-label entry (`sendAgentHandoffMessage`) writes zero rows. |
| M4 | Live proof on the **dev** plane | See below. Nothing reaches prod until M4 passes. |

**Every new test is mutation-checked**: revert the fix line, watch the test fail, restore.

### M4 — live proof (what should convince a sceptic)

On `ailang-multivac-dev`, after the dev coordinator serves a build containing M1–M3:

1. **S1:** leave an old routable message for the sweep (or send one and suppress its push); confirm
   the task survives past the detector's first two ticks and its real completion is *applied*
   (`CompletionHandler: task … -> …` line, not "already in terminal state").
2. **S2:** dispatch a task whose precondition is unmet; confirm the task reads `blocked` within
   minutes, not `failed` at 3 h.
3. **S3:** approve a task that owes a handoff; force a coordinator restart (new revision); confirm
   exactly ONE `Handoff:` message and no "Triggered missed handoff" line.
4. Record each as a V-row below with the log excerpt.

## Acceptance criteria

- [ ] All three defects reproduced by a failing test before the fix (mutation-checked).
- [ ] `make test` green for `internal/coordinator`, `internal/storage/firestore`, `cmd/ailang`; lint clean.
- [ ] M4 steps 1–3 observed on dev with log evidence.
- [ ] Prod: after release, 24 h of logs show 0 "unknown completion status", 0 stale kills with
      `age` ≫ timeout within a minute of dispatch, 0 "Triggered missed handoff" for approvals made
      after the deploy.

## Verification log

| # | Claim | Observed |
|---|---|---|
| V1 | S1 mechanism | prod log 09-23 15:23:30–15:26:22, `task-03bd17ad` (above) |
| V2 | S2 mechanism | prod log 09-23 15:25:56 / 18:26:16, `task-91337be6` |
| V3 | S3 is every approval, not a crash path | handoff messages per parent: `task-08032ebc` 13:00 + 13:33; `task-080f4657` 14:51 + 16:09; `task-38dcb44a` 18:03 + 18:34; `task-90bb931d` 17:30 + 18:19 |
| V4 | S4 mechanism | prod log 09-23 15:17:36 "Found 22 approved merge_handoff approval(s) without triggered handoffs" |
| V5 | D1 premise: the claim records no time | `MarkTaskQueued` updates only `status` (Firestore `coordinator_transitions.go:24-42`) / `status, finalization` (SQLite `store_sqlite_transitions.go:38`) |
| V6 | D1 premise: cloud tasks never reach `running` | `MarkTaskRunning` has one caller, `daemon_tasks_exec_run.go:49` (local execution). The cloud branch (`daemon_tasks_exec.go:350`) dispatches from `queued`. |
| V7 | D1 premise: age falls through to `CreatedAt` | `getTaskAge` (`stale_task_detector.go:233-238`): `StartedAt`, else `CreatedAt`. And the task inherits `CreatedAt` from the message: `daemon_tasks_polling.go:158,209,467,513` (`CreatedAt: msg.CreatedAt`). |
| V8 | D1 fix reproduced before it existed | `TestQueueClockNotMessageClock`, `TestMarkTaskQueuedStampsQueuedAt`, `TestRecoverStaleTasksAgesFromQueuedAt`, `TestTaskQueuedAtRoundTrips` all FAILED on `aaf7b08b5` with the field added and no writer/reader change. |
| V9 | D2 premise: the status tables already declare `blocked` | `task_status.go:44` (in `AllTaskStatuses`), `:66` (terminal), `:90` (observatory → failed); added `e03cbab40` 2026-09-14 alongside the producer (`coordinator_cloud.go:285`). The consumer `completionOutcome` (`task_finalize_cloud_strategy.go:43-53`) has no `blocked` case. So D2 is a matrix row, not a status-machine change. |
| V10 | D3 premise: the approve path never marks | `dispatchApprovalHandoffs` (`approval_handoff.go:391-439`) has no call to `MarkApprovalHandoffsTriggered`; the only caller is recovery (`approval_processor.go:728`). |
| V11 | D3 premise: store bounds differ | SQLite `ListApprovedMergeHandoffsWithoutTrigger` adds `created_at > now-7d` (`store_sqlite_approvals.go:326-337`); Firestore (`coordinator_approvals.go:198-205`) has three equality filters and no bound. |
| V12 | D3 premise: first-write-wins exists in both stores | Firestore `PutMessageIfAbsent` uses `Doc(id).Create` (`finalize_if_absent.go:109-122`); SQLite `ON CONFLICT(id) DO NOTHING` (`messaging/inbox.go:126-141`). Neither publishes: notify is a separate step (`approval_handoff.go:154-160`). |

| V13 | D3 premise: every handoff producer is enumerated | `grep InboxTypeHandoff\|MessageType: "handoff"` over `internal/` and `cmd/`: exactly three constructors — `approval_handoff.go:103` (`handoff.inboxMessage`, used by `handoffSender`, whose callers are `dispatchApprovalHandoffs` `:426`, `task_chain.go:441`, `daemon_approval.go:82`), `task_finalize_approval.go:316` (completion path, already `PutMessageIfAbsent` + `HandoffMessageID`), and `approval_processor.go:704` (boot recovery, plain insert, own body, no notify). After M3: two constructors, both first-write-wins under `HandoffMessageID`; recovery calls the shared sender. |
| V14 | D3 premise: an un-notified handoff row IS delivered by the sweep | prod 09-23 15:17:49, revision `00158-g57`: "backstop sweep: recovered inbox_1790176655955_877d700f (inbox=sprint-planner) into the normal drain" → 15:22:50 "Created task task-877d700f … from cloud message". That row was written by boot recovery with a plain insert and **no notify** (V13). `isOutcomeNotice` (`daemon_tasks_polling.go:67`) excludes only `completion` and `approval_request` kinds; `handoff` is routable. |
| V15 | Concurrency is real, not hypothetical | same log: revisions `00157-t6r` and `00158-g57` both ran boot work within 16 s of each other during one rollout — two recoveries over the same approvals. First-write-wins makes that safe; the boolean marker did not. |

| V16 | D1 premise: every cloud task passes through `MarkTaskQueued` | Tasks are created `pending` (`daemon_tasks_polling.go:205,509`); the sweep enqueues into the same drain (`backstop_sweep.go:194`) rather than writing tasks. `MarkTaskQueued` has exactly one caller, `dispatchTasksCloud` (`daemon_tasks_exec.go:123`), which lists `pending` and claims each. No other code writes `queued` (grep `TaskStatusQueued`: only the claim, the CAS allow-list, and readers). The stale detector's re-dispatcher is never wired: `WithReDispatcher` has no non-test caller, so there is no second path into `queued`. |

| V17 | `OnAgentApproved` is unreachable | `grep OnAgentApproved` over `internal/` and `cmd/`: the definition (`task_chain.go:385`) and comments only; no call site. |
| V18 | Auto edges are disjoint from what an approval owes | `approvalHandoffTargets` excludes `AutoApproveHandoffs` / `AutoApprovesHandoffTo` targets; pinned by `TestApprovalHandoffTargets_ExcludesAutoEdges` and `TestHandoffTargetsPartition` (`approval_handoff_test.go:10,59`). |
| V19 | The resolve is a compare-and-set on pending | SQLite `resolveApprovalByTask`: `UPDATE … WHERE task_id = ? AND status = 'pending'`, 0 rows → error — a real CAS. Firestore **was not**: `Where("status","==","pending")` then a plain `Update`, so two resolvers could both succeed (quorum round 5, correct). Fixed: the update runs in `RunTransaction` after re-reading `status == pending`, the pattern `MarkTaskQueued` already runs in prod. **Verified live against real Firestore** (`ailang-multivac-dev`, `TestResolveApprovalIsCompareAndSet_Live`, 8 concurrent resolvers, half suppressing): fixed code → exactly 1 winner, 3/3 runs, the document carries the winner's `resolved_by` and suppression. **Control** — same test against the pre-fix query-then-update: 8/8, 7/8, 7/8 resolvers "succeeded". The race was real in the shipped code. |
| V21 | `CreatedAt` has consumers that need the request time | `task_dedup.go:103-106` (`BlocksDuplicate` scopes by `t.CreatedAt`); `daemon_landed_cards.go:77` (`task.CreatedAt.Before(landedCardSweepSince)`, the 09-23 15:00 cutoff). Rewriting `CreatedAt` to dispatch time would re-open dedup for sweep-recovered work and let the landed-card sweep act on the pre-cutoff backlog. |
| V20 | Every fix line is guarded (superseded by V28) | Mutation run 2026-09-25: 13/13 mutations (one per fix line across M1–M3, incl. the batch bound and nothing-owed record) turn their test red; restored after each. Plus the live Firestore CAS control (V19). |

| V22 | `handoffs_triggered` is a scan latch, not a display field | Readers (grep over Go/TS/JS/Py): exactly the two `ListApprovedMergeHandoffsWithoutTrigger` queries (SQLite `store_sqlite_approvals.go:371`, Firestore `coordinator_approvals.go:277`). It is not on `ApprovalRequestRecord`, no CLI/dashboard renders it. So setting it on suppression/expiry cannot be misread as "fired"; the truthful audit is `handoffs_suppressed` / `handoffs_expired`. Interface comment renamed to "decision recorded". |
| V23 | Nothing un-suppresses | Writers of `handoffs_suppressed`: Firestore sets only `true` (`coordinator_approvals.go:225`); SQLite `resolveApprovalByTask` writes `?` (0 or 1) — but only inside the `WHERE status = 'pending'` CAS, i.e. on an approval that has never been resolved and so never suppressed. After resolution no code path writes it. |
| V24 | One boot's recovery is bounded | `HandoffRecoveryBatch = 100`: SQLite `LIMIT ?`, Firestore `.Limit(…)`. Every approval taken is decided (fired / nothing owed / expired), so a backlog drains across boots. `TestHandoffOnce_RecoveryWorkIsBoundedPerBoot` (101 stale → 1 left after one boot → 0 after two); mutation (drop LIMIT) caught. |

| V25 | One task can owe the same target more than one handoff | `ReopenApprovalForNewWork` (`finalize_if_absent.go:66`, SQLite `store_sqlite_approvals_if_absent.go:52`) re-opens an **approved** decision when a later run produces different work (`work_id`). So the approval-path identity is `HandoffMessageIDForWork(task, target, work_id)`; the re-open resets `handoffs_triggered/suppressed/expired` in both stores (a new decision), and every resolve writes `handoffs_suppressed` explicitly. Tests: `NewWorkAfterReopenOwesANewHandoff`, `ReopenClearsSuppression`, `ReopenedApprovalIsRecoverable`. |
| V26 | Deploy-time legacy duplicates | Rows written by the pre-fix approve path have random `inbox_…` ids the new identity cannot collide with. Recovery checks the target inbox (newest 500) for a legacy-form handoff row for the task written at/after this approval's resolution and treats it as sent. `LegacyRandomIDRowIsNotResent`. |
| V27 | Batch rate vs. load | Prod approvals are tens per day (09-23 backlog of 77 accumulated over ~14 days). One boot takes 100; prod boots every ≤1.5 h → ≥1,600/day of recovery capacity, and recovery is only the crash path — the approve path sends synchronously. A written-but-unnotified row does not wait for recovery at all: the sweep delivers it on its next 10-min pass (V14). |
| V28 | Mutation coverage after round 7 | 16/16 single-line mutations caught, plus a combined mutation removing both suppression guards (reopen reset + explicit resolve write), which are individually redundant by design. |

| V29 | Decision marks are compare-and-set on the work they describe | `MarkApprovalHandoffsTriggered/Expired(task, work)`: SQLite `WHERE status='approved' AND json_extract(context_json,'$.work_id') = ?`; Firestore re-reads status + `context_json` inside a transaction. A delayed mark for work A cannot stamp work B. `StaleMarkDoesNotHideNewDecision`; mutation caught. |
| V30 | Completion-path auto edges are per-work too | `applyHandoff` computes the work id from `strategy.DiffSource` (immutable SHAs → replay-stable). `TestFinalize_AutoHandoffIsPerWork`: run a.go, redeliver a.go, run b.go → 2 rows; mutation caught. |
| V31 | **Live proof S1 + S2 on `ailang-multivac-dev`** (build `c13557c`) | Seeded `inbox_1790076618356_m4s1` to `ailang-core`, `created_at` 72 h in the past, never notified. 11:37:15 sweep recovered it → 11:42:15 `task-5f31e226` created → 11:42:18 dispatched → **no stale-detector line for it** (old code: failed on first tick, age 72 h ≫ 30 m) → 11:43:25 `status=blocked` received and **applied** (`-> blocked`, not "unknown completion status"). Stored record: `created_at=2026-09-22T11:30:18Z` (message time kept), `queued_at=2026-09-25T11:42:15Z` (claim — proves the new code claimed it), `status=blocked`. |
| V32 | Mutation coverage after round 8 | 18/18 single-line mutations caught (a harness guard refuses to count a build failure as a catch — it caught one of its own mutations doing exactly that). |

## Quorum round 8 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | A delayed mark for old work can hide a reopened decision | Marks are CAS on (approved, work id) in both stores (V29). |
| gemini-3-1-pro, oc-glm-5-3 | Completion-path auto edges still keyed (task, target) | Now `HandoffMessageIDForWork` with the completion's own work id (V30). |

## Quorum round 7 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | Legacy random-id handoffs re-fired once at deploy | Legacy check in recovery (V26). |
| gemini-3-1-pro | 100/boot too slow under a failure spike | Measured volumes (V27); recovery is the crash path only, and unnotified rows are the sweep's, not recovery's. |
| oc-glm-5-3 | (task, target) may legitimately owe twice | Correct — `ReopenApprovalForNewWork`. Identity now includes the work id; re-open resets the decision (V25). |

## Quorum round 6 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | The current boot's recovery work is unbounded | Batch-bounded per boot (V24). |
| gemini-3-1-pro | Firestore CAS verified only by construction; un-suppression unverified | Now verified live with a control that shows the old code racing (V19); V23. |
| oc-glm-5-3 | `handoffs_triggered` overloaded without a reader enumeration | V22: two readers, both the recovery scan; comment renamed. |

## Quorum round 5 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra, oc-glm-5-3 | Firestore resolve is query-then-update, not a CAS | Correct — a real defect in the base code, not just the doc. Fixed with a transaction (V19); limitation of verification stated. |
| gemini-3-1-pro | `CreatedAt` consumers unverified | V21. |

## Quorum round 4 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | Check-then-write race between sender and a suppressing resolve | Impossible for live producers (V17, V19): suppression is fixed by the single winning resolve and never changes after; the only producer that could race is unreachable. Recorded in D4 as the condition for re-wiring it. |
| gemini-3-1-pro | Doc says SQLite bounds in SQL, so expiry never reaches it | Correct about the doc; the SQL bound is removed in the implementation, doc updated. V18 added for the auto-edge claim. |
| oc-glm-5-3 | Two functions named, roles unclear | Structure table above. |

## Quorum round 3 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | Suppression not authoritative for the GitHub-label / auto-edge producers | Checked inside the shared approval-path sender; auto edges are out of scope by construction (fire before any approval exists, disjoint targets). Test (f). |
| gemini-3-1-pro | In-process bound still fetches an ever-growing set | Recovery marks out-of-window approvals expired as it sees them — each fetched at most once; steady-state set empty. Test (e) runs two passes. |
| oc-glm-5-3 | No call-site enumeration for `MarkTaskQueued` | V16. |

## Quorum round 2 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | Suppression written "in the same call", not atomically | Now one write with the approval resolution (`ResolveApprovalSuppressingHandoffs`); test (d) reads both fields from one row. |
| gemini-3-1-pro | Producers-route-through-one-sender unverified | V13 enumerates all three constructors by grep and names what changes. |
| oc-glm-5-3 | Sweep delivery of un-notified rows unverified; test (b) passes on zero deliveries | V14 (prod evidence); every M3 test now asserts delivery as well as uniqueness, (b) drives `SweepOnce`. |

## Quorum round 1 (2026-09-25, BLOCKED) — responses

| Reviewer | Objection | Response |
|---|---|---|
| gpt6-astra | A marker written after sending leaves crash and partial-failure windows | Accepted; D3 rewritten around per-(task,target) first-write-wins. The marker is no longer load-bearing except for suppression, which is written before any send. M3 now tests crash-after-write and partial failure. |
| gemini-3-1-pro | The Firestore index question was left open | Decided: bound applied in-process; no new index. |
| oc-glm-5-3 | Code premises asserted, not verified | V5–V12 added with file:line evidence; V8 records the tests failing before the fix. |
