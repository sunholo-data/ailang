# M-MESSAGE-PLANE-TRUST: make "a message was sent" mean "a job ran and reported back"

**Status**: IN PROGRESS — M1/M2/M3 landed 2026-08-31 (`91716b092`, `f83fbd0b1`, `cc5cdd736`) on top of `a4098b6a0`, `23066345e`, `8131b4101`. M4/M5 open and BLOCKED (see below).
**Target**: this week.
**Priority**: P1. Everything downstream — provenance chains, ELO routing, cost KPIs — reads data this path is supposed to produce.
**Related**: [m-feature-provenance-chains.md](m-feature-provenance-chains.md) (consumes this), [message-plane-topology.md](../../docs/internal/message-plane-topology.md) (the map).

---

## Problem statement

The cross-provider setup is built and, part by part, works: 34 agents registered, push
subscriptions correctly configured, 17 executor images built and deployed, a live coordinator, jobs
that do run and sometimes succeed. What does not work is the **whole**. Every seam between the
parts fails silently, and each silent seam is individually cheap to fix but collectively fatal to
trust: a message is "sent", and nothing happens, and nothing says so.

Measured first-party 2026-08-31 against prod (`ailang-multivac`):

| Seam | Symptom | Root cause | State |
|---|---|---|---|
| send → Pub/Sub | 3 `pkg:sunholo/ailang_parse` reports sat unread, no task, no job | notify gated on the SENDER's local config, error discarded, `✓ Message sent` printed anyway | **FIXED** `a4098b6a0` |
| dispatch → container | 88% of 92 job executions failed since April | variant picks the image, provider picks the binary, nothing checked they agree | **FIXED** `23066345e` |
| hooks → sessions | 2 tool rows since April against 6,508 sessions | `session_tools` FK-rejected events for sessions predating the server; endpoint still returned 200 | **FIXED** `8131b4101` |
| stage → chain counter | all 99 prod chains read "0 stages" while holding stages | Firestore never incremented `stages_completed`; its list reports that field | **FIXED** `8131b4101` |
| Firestore → coordinator | a missed notification is permanent, not slow | cloud intake reads the Pub/Sub adapter ONLY, never Firestore | **FIXED** `91716b092` (report-only by default) |
| job → completion | 92 of 99 chains stuck `active`, oldest since April | stale detector marked the TASK failed and never touched the chain | **FIXED** `f83fbd0b1` |
| gate → visibility | `ANTHROPIC_API_KEY not set: heuristic-flagged submissions filed, never dispatched` | fail-closed posture announced once at startup, in a log nobody reads | **PARTLY — `cc5cdd736`** (`messages health`; gate row still owed) |

**The pattern is one thing, not seven.** Each seam reports success on the happy path and silence on
the failure path. `|| true`, a discarded error, a denormalized counter nobody increments, a `200
{"status":"ok"}` over a rejected insert. The system is not unreliable so much as **unobservable**,
and unobservable is indistinguishable from unreliable when you are deciding whether to trust it.

## Verification Log

| # | Claim | Observed |
|---|---|---|
| V1 | Routing for the reported inbox is correct | `pkg-sunholo-ailang-parse` → `pkg:sunholo/ailang_parse` present in the coordinator's own startup registry line (34 agents), correct underscore spelling |
| V2 | The path works once announced | probe to an unregistered inbox: publish → push → Cloud Run **cold start** → `received 1 message(s) from Pub/Sub` → routed, in **23s** |
| V3 | Job failure rate | 81 of 92 executions since 2026-04-27 did not succeed (**88%**); `ailang-agent-executor` 77/85 |
| V4 | Task outcomes agree | 83 of 103 prod tasks `failed`, 7 `completed`; newest task 4 days old |
| V5 | The mismatch is real, with a named victim | `task-a0628a5f`: job `…-codex`, log `running opencode executor (unified path)`, `exec: "opencode": executable file not found in $PATH`, after `Updating files: 100% (24032/24032)` |
| V6 | Config is internally consistent | every agent's provider matches its variant (pi/pi ×31, codex/codex, codex/codex-go, motoko/motoko) — so V5 predates the current config, and the GUARD is what stops it recurring |
| V7 | `pi` has never run | `ailang-agent-executor-pi` has no `latestCreatedExecution` — the 31 pi agents have never dispatched |
| V8 | Coordinator is healthy, not stuck | heartbeat `status: idle` updated within minutes; `minScale=0` + push subscriptions is a correct pairing, and cold start cost 22s |

## Open finding — the fail-loud guard for this exact class is RED at HEAD

`TestRun_FailsLoudlyWhenTaskProcessingCannotInit` fails on a clean checkout (verified by reverting
every change in this session). It guards M-MESSAGE-PLANE-FAIL-LOUD, whose own comment describes the
problem this doc exists to solve:

> *standby mode is indistinguishable from idle from the outside: `coordinator status` reports
> "running", the health port answers, and the log prints "Checking for new tasks..." on every tick.
> Measured 2026-08-26 on the rig: 20 standby startups against 11 healthy ones ... production claimed
> nothing for three months.*

Probed directly: `initTaskProcessing()` now returns **nil** when CWD is not a git repository, so the
fatal branch never runs and `Run()` falls through to the poll loop — which is what the test times out
on.

**This needs a ruling, not a unilateral fix.** Either the guard is stale (the cloud coordinator
demonstrably processes messages from a non-repo CWD — we watched it route a probe in 23s, so
"CWD is not a repo" is no longer sufficient to mean "broken"), or the fail-loud condition was
weakened and the standby regression can recur. Deleting a three-month-outage guard on the first
reading would be exactly the mistake the guard was written to prevent.

## Design decisions

### D1 — Push stays the fast path; Firestore becomes the backstop

Cloud intake reads the Pub/Sub adapter only, so a lost or unpublished notification is permanent.
A periodic sweep for unread messages **whose inbox has a registered agent** converts that class from
data loss into latency. Push remains primary; the sweep is a floor, not a replacement. It must be
idempotent against the existing dedup (fingerprint / `dup_of`) so a swept message cannot double-run.

### D2 — A dispatched job MUST reach a terminal state, or be swept into one

92 chains stuck `active` since April is not a display bug; nothing reconciles a chain whose job
died. The completion handler already exists (`/pubsub/completions`); what is missing is the
reconciler for jobs that never publish one. A stale-task detector already runs at 2m intervals —
it should own chain closure too, and record WHY (`job_failed`, `no_completion`, `timeout`), because
"active forever" and "failed at startup" are different facts that currently look identical.

### D3 — Fail-closed postures are health, not startup logs

The feedback gate without `ANTHROPIC_API_KEY` files heuristic-flagged submissions and never
dispatches them. That is a defensible posture and an indefensible way to announce it. Every
fail-closed switch reports into a health surface that a human or a morning report reads.

### D4 — Refuse before spending, not after

Landed in `23066345e` and stated as a rule: any precondition knowable before a container starts is
checked before dispatch. The measured counter-example cost a full image pull plus a 24,032-file
clone per occurrence to discover a static config fact.

### D5 — Stale data is cleaned only AFTER the writers are correct

Explicitly sequenced (Mark, 2026-08-31). Cleaning first would destroy the evidence that tells us
whether the fixes worked, and would make a broken writer look healthy.

## Blocked on the operator

1. **Two Firestore composite indexes do not exist**, and each reads as a query failure at every call
   site. `obs_chain_stages(chain_id, stage_number)` — makes `chains view --remote gcp` fail.
   `inbox_messages(dup_of, status, created_at DESC)` — makes the unread sweep and `messages health`
   fail. Creating them is a cloud infrastructure mutation, deliberately not performed here:

   ```
   gcloud firestore indexes composite create --project=ailang-multivac \
     --collection-group=obs_chain_stages \
     --field-config=field-path=chain_id,order=ascending \
     --field-config=field-path=stage_number,order=ascending

   gcloud firestore indexes composite create --project=ailang-multivac \
     --collection-group=inbox_messages \
     --field-config=field-path=dup_of,order=ascending \
     --field-config=field-path=status,order=ascending \
     --field-config=field-path=created_at,order=descending
   ```

   These belong in `firestore.indexes.json` in the terraform repo (D6) so they stop being
   rediscovered as outages.

2. **Nothing here is live in cloud.** The coordinator runs a deployed image; these commits are on
   local `dev` and unpushed. The backstop sweep, chain closure and dispatch guard take effect only
   after push + image rebuild + deploy.

3. **M4 and M5 are gated on a measurement that needs 1 + 2 first.** Run the sweep in `report` mode
   for a day; its count is what decides whether `dispatch` is worth enabling and whether re-announcing
   the stranded messages is safe.

## Milestones

**M1 — Firestore backstop sweep** (D1): periodic unread scan restricted to inboxes with a
registered agent; idempotent against dedup; logs how many it recovered, so a recovering sweep is
itself a signal that push is dropping.

**M2 — Terminal-state reconciliation** (D2): the stale-task detector closes chains for dead jobs
with a recorded reason; a chain may not sit `active` past a bounded age without being reported.

**M3 — Health surface** (D3): `ailang chains health` (and the morning report) gain message-plane
rows — coordinator reachable, last push received, unread-with-agent backlog, fail-closed gates
active, jobs succeeded/failed in the window.

**M4 — Re-announce the stranded** : the 3 `ailang_parse` reports (and any siblings) are re-published
once M1/M2 make the outcome observable. NOT before — dispatching into an 88% failure rate produces
noise, not progress.

**M5 — Cleanup** (D5): close or delete the 92 stuck-active chains and the pre-fix task corpus, once
M1–M3 have produced a clean window to compare against.

## Acceptance criteria

| AC | Claim |
|----|----|
| AC1 | A message written to Firestore with NO notification is picked up by the sweep within one interval, and exactly one task is created |
| AC2 | A job that dies without publishing a completion leaves its chain `failed` with a reason, not `active` |
| AC3 | `chains health` reports RED when the unread-with-agent backlog exceeds a bound, or no push has been received in N hours |
| AC4 | A variant/provider mismatch is refused before the Cloud Run API is called (landed; wiring-arm asserted) |
| AC5 | `messages send` to a GCP store without pubsub config warns FILED, NOT DISPATCHED (landed) |
| AC6 | No chain in prod is older than N hours in state `active` |

**Mutation arms**: MU-1 make the sweep ignore dedup → AC1 RED (double-dispatch). MU-2 make the
reconciler skip jobs with no completion → AC2 RED. MU-3 hard-code the health rows green → AC3 RED.

## Risks

- **The sweep double-runs work.** The whole risk of D1. Dedup is the mitigation and AC1 is the arm;
  ship the sweep in report-only mode first and compare what it *would* have dispatched against what
  push already did.
- **88% has more than one cause.** The variant/provider mismatch is one confirmed mechanism, not
  proof of the whole rate. M3's per-window job stats are what tell us the remainder.
- **Cold start inside an ack deadline.** Push ack deadline is 60s and a cold start measured 22s.
  Comfortable now, not obviously comfortable under load with a bigger image.
