# HANDOVER — message plane: approvals, staleness, dispatch (2026-09-23)

Session question: *"are messages working correctly? we have had some messages that
should have made design docs upon approval — is that workflow functioning?"*

Answer: delivery and execution work. **Approval was locked out, and the approval
queue is mostly a ledger of decisions that already happened.** One fix shipped;
three faults remain, each with its next step below.

---

## 1. SHIPPED — `97cc947b8` (ailang, on `dev`, unpushed at handover time)

`fix(approvals): approve a cloud task against the PLANE's registry, not the laptop's`

`coordinator approve --remote gcp` resolved the agent registry with
`coordinator.LoadAgentRegistry()` — **this machine's** config — while the task lives
on the plane. This laptop's `~/.ailang/config.yaml` declares exactly two agents
(`eval-rig`, `sprint-evaluator`), so every cloud task's agent failed to resolve and
`checkRegistryCanDispatch` refused the approval.

The guard itself is correct (`4e9466b28`, 2026-09-07): handoff topology comes only
from the registry, and an approval that dispatches nothing cannot be taken back. It
was asking the wrong registry.

- Landed 2026-09-07 → queue starts 2026-09-11 → bulk 09-13 → last handoff 09-17.
- The **dashboard was never affected** (it runs against `/etc/ailang-config`, which
  does declare the cloud agents). That is what made the fault read as intermittent.
- `d20772f7f` exempted *rejections* on 09-17 after `task-c063b6d2` sat seven days;
  approvals stayed blocked.

Fix: `resolveInboxRegistryForPlane(flagPath, plane)` — split out of
`resolveInboxRegistry`, which took its plane from the *message* store
(`messagesTarget()`); the approvals path's plane comes from `--remote`. Eight other
callers unchanged. New `approvalRegistry(bundleMode)` in
`cmd/ailang/coordinator_approvals_remote.go`.

**Verified live**: rejecting `task-98ed760b` printed
`handoff topology from: gs://ailang-multivac-ailang-config/config.yaml (the plane's own registry)`.

**Trap to preserve**: `bundle.Mode` is a *human label* —
`"gcp (project ailang-multivac, via config.yaml pubsub.project_id)"`. `storage.Mode`
of that equals no plane, falls through to the local config, and reinstates the bug
silently. I wrote that exact defect while fixing this and the whole `cmd/ailang`
suite passed with it in. Conversion + resolution now live in one function so one
test covers both; both mutations are caught (`-count=1`).

---

## 2. OPEN — staleness: nothing closes a card when its PR merges

**56 of 78 pending approval cards have an already-merged `coordinator/task-*` PR**,
most merged the same day the card was created. Two were created *today*:
`task-70a33b83` → daneel PR #182 (merged), `task-80e577ff` → daneel PR #179 (merged).

`reconcileTaskPR` runs *after* a decision — card → PR is wired. **PR → card is not.**
Same seam shape as the registry bug.

The harm is **not lost documents**. It is **stalled chains**: the doc lands via PR,
but approving is what dispatches `sprint-planner`, so the design exists and the
sprint never starts. That is the ~21 design-doc verdicts with nothing downstream.

**Next step**: close the card when its PR merges — and for a `merge_handoff`, still
fire the handoff (the merge happened, the dispatch did not). There is already an
`ApprovalWatcher` polling GitHub (`ailang coordinator watcher-status`); this belongs
there. Then a one-off reconcile over the ~56.

Related but distinct: `internal/coordinator/daemon_stranded_approvals.go` handles the
*opposite* case (approved elsewhere, never finalized here). Mirror it, do not extend it.

---

## 3. OPEN — dispatch into `design-doc-creator` dead since 2026-09-17

Two real work items from Daneel on 09-21 reached Firestore and never became tasks:

- `inbox_1789993952527_28db791a` — Feature: executor holds `sunholo/gemini_agents` (#1267)
- `inbox_1789993958333_e00b9074` — Bug: publish smoke gate runs in a temp copy, PUB015 (#1268)

`ailang messages health` reports this correctly and has been saying **DEGRADED** the
whole time (`routable but undelivered: 2 design-doc-creator, oldest 46.8h`). Run it
first — it answers the session's opening question in one command.

Cause: the cloud coordinator's intake is **Pub/Sub only**; a notification that never
published is invisible forever. The designed floor is the backstop sweep, which is in
`report` mode in all three environments and therefore only logs.

**The promotion to `dispatch` is now fully unblocked** — all four preconditions met:

| precondition | state |
|---|---|
| backlog read and small | ✓ 4 messages, all recent, all identifiable (prod sweep log, every 10 min) |
| sweep fix `e0b12bf5f` serving | ✓ coordinator runs `:latest` = `v0.41.0`, built 2026-09-21; fix shipped v0.35.0 |
| no re-dispatch loop | ✓ `markDispatchedMessageRead` at `daemon_tasks_polling.go:644` (2026-09-02) |
| no completion-notice loop | ✓ `isOutcomeNotice` filter + `Kind`/`CreatedAt` carried (`e0b12bf5f`) |

**Next step**: set `backstop_sweep_mode = "dispatch"` in
`ailang-multivac/terraform/environments/{dev,test,prod}/terraform.tfvars` (currently
`"report"` in all three), and **update the comment block above the setting** — it
still records the 2026-08-31 incident as the reason not to. dev → test → prod.

`origin/dev`, `origin/test`, `origin/prod` are **in sync (0 commits apart)**, so the
promotion carries only this change. NOTE: `terraform/docparse.tf` is modified and
uncommitted in that tree — **another session's work, do not commit it.**

Root cause of the non-delivery (send-side publish vs receive-side loss) is still
unknown. Promoting the sweep converts permanent silence into ≤10-minute latency
regardless; chase the root cause separately. Subscriptions carry no filters, and the
adapter's tag filter only applies to messages with a non-empty `requires` attribute,
so neither explains it.

---

## 4. OPEN — approving a triage note cannot start a design doc

`ailang-core-triage` has **no `trigger_on_complete`** — deliberately, per the comment
in `ailang-multivac/config/config.cloud.yaml`: *"A row recommending `design-doc` is a
proposal, and dispatching design-doc-creator on it is Mark's click."* Nothing was ever
wired as that click, so ~20 `design-doc` verdicts have piled up with nothing downstream.

**Correction to another agent's note on this**: the blocker is *not*
`auto_approve_handoffs: false`. That field should **stay false**.
`approval_handoff.go:56` shows non-auto edges are precisely the ones that fire **on
approval** — which is the desired behaviour, and it already works for
`design-doc-creator → sprint-planner`. The cause is the *absent* `trigger_on_complete`.

**Next step**: add `trigger_on_complete: [design-doc-creator]` to `ailang-core-triage`,
keeping `auto_approve_handoffs: false`. Ideally gated on the output saying
`RECOMMEND: design-doc` — duplicates and direct fixes must not trigger it. Multivac
config change; reaches prod only on a prod-branch push.

---

## Corrections to things said earlier in the session — do not carry these forward

- **"17 design docs exist nowhere but the approval queue."** WRONG. That came from
  file-existence in the ailang tree alone. Checked against git history and the daneel
  repo: **no work is lost**. The docs landed; several were then deliberately reverted
  or superseded. All 25 `merge_handoff` rows are stale — 19 already in tree, 6
  landed-then-removed. There is **no live handoff candidate in the queue**.
- **"test/prod are 92 commits behind."** WRONG — that was `git branch -vv` showing
  *local* refs behind their own upstreams. Real gap is 0. Always use
  `git rev-list --count origin/prod..origin/dev`.
- **Memory `project_message_plane_dispatch_works_2026_09_02`** says dispatched messages
  are never marked read. **Outdated** — fixed 2026-09-02 by M-COORDINATOR-EXECUTION-TRUST
  M4 (`internal/coordinator/dispatch_read_marking.go`).

## Done this session, for the record

- `task-98ed760b` (M-DANEEL-SOUL) **rejected as stale**: landed via PR #1245
  (`c10890054`), then reverted in `6f3a96683` as a duplicate of `sunholo-data/daneel#61`.
  Its twin `task-6863b5cc` (M-DANEEL-EXECUTOR-LONG-ANSWERS) was removed in the *same*
  revert and is **still pending** — same verdict applies.
- Remote `reject` does **not** re-dispatch (`RetriggerOnReject` is never set on that
  path), so dismissing is safe. `coordinator reopen` exists if a rejection needs undoing.

## Unrelated, but release-blocking

`make check-file-sizes` is **red on dev**: `internal/parser/parser_expr.go` is 817
lines, from `2122705e0` (2026-09-22, not this session's work).
