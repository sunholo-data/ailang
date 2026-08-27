# M-COORD-DISPATCH-INTEGRITY: accept-then-work, dedup, and no silent handoffs

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1 — three dispatch-plane defects measured in one afternoon of live sprint
routing; one produced duplicate paid work, one nearly did, one silently dropped the
pipeline's quality gate
**Estimated**: ~2.5 days
**Dependencies**: None (sibling of [m-coord-cli-shared-store](m-coord-cli-shared-store.md))

## Quorum triggers

All four skip conditions hold: no freeze items, fixes (not overrides) shared machinery, no
cost/KPI/banked-data schema surface, premises in-repo (mechanisms marked TO-DIAGNOSE are
scoped as milestone work, not assumed). **Quorum skipped per checklist.**

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Idempotent dispatch: the same directive cannot become two tasks |
| A2: Replayability | 0 | — |
| A3: Effect Legibility | 0 | — |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | 0 | — |
| A6: Safe Concurrency | +1 | Accept-then-work removes the timeout/retry double-dispatch race |
| A7: Machines First | +1 | Agents driving dispatch need machine-readable acks, not inference from timeouts |
| A8: Minimal Syntax | 0 | — |
| A9: Cost Visibility | +1 | Duplicate tasks are duplicate spend; measured $0.98 + in-flight duplicate on 2026-08-27 |
| A10: Composability | 0 | — |
| A11: Structured Failure | +1 | A handoff that produces no task becomes an alert instead of silence |
| A12: System Boundary | 0 | — |

**Net Score: +5** → Move forward. Hard-violation check: none.

## Problem Statement

Three defects, all measured live on 2026-08-27 while routing the M-EVAL-ROLLING-ELO sprint
through the coordinator (Verification Log below):

1. **The dispatch endpoint does not respond before the work is underway.** A `POST
   /api/messages` (tag-routed sprint dispatch) timed out client-side awaiting headers — yet
   the message was stored at 14:09:11 and the task was created and *executing* by 14:09:14.
   The client cannot distinguish "failed" from "accepted"; any client that retries on timeout
   double-dispatches paid agent work. (Root cause of the slow response is deliberately
   TO-DIAGNOSE in M1 — the *contract* defect is proven regardless of mechanism.)
2. **Nothing dedups dispatch.** `correlation_id` exists in every `plan_ready` payload and is
   used by exactly nothing in task polling. When the rejection-feedback retrigger
   (`task-5f0bad80`, 14:31:50) and an operator revision dispatch (`task-78a9624b`, 14:33:48)
   both landed, the coordinator happily ran both — duplicate ~$1 executor runs, two approval
   cards for one piece of work.
3. **A handoff can vanish silently.** At 14:19:18 the daemon logged `Auto-approved handoff
   from eval-rig to sprint-evaluator` — meaning `sendHandoffMessage` returned success — and
   then *no evaluator task was ever created* (none in the following 2.5h; the most recent
   sprint-evaluator task in the log is from the previous day, which also proves the
   inbox→task path works when fed directly). The pipeline's quality gate simply did not run,
   and nothing said so. Also observed in the same window: `Embedding handoff to [] in merge
   approval` — an empty-target embed logged as routine. This is the exact "alive, healthy,
   doing nothing" class M-MESSAGE-PLANE-FAIL-LOUD (2026-08-26) was built to kill, one layer
   up.

Secondary observation, same incident: the rejection→retrigger loop works but took ~12 minutes
with zero intermediate logging, which is what invited the duplicate dispatch — an operator
watching for 60s concluded (wrongly) that no retrigger existed.

**Impact:** dispatch is the entry point for ALL delegated work (mission loops, cloud lanes,
attended steering). Every one of these defects either spends money twice, drops a gate, or
teaches operators wrong lessons about the system.

## Goals

**Primary Goal:** dispatch is acknowledged, idempotent, and observable end-to-end: a client
knows its send landed, the same work cannot run twice, and every promised follow-on (retrigger,
handoff) either materializes or alerts.

**Success Metrics:**
- `POST /api/messages` responds < 2s with the created message id (accept-then-work), under a
  test that makes the downstream work artificially slow.
- Re-sending an identical `correlation_id` within the dedup window yields the *existing*
  task's id, not a second task (and says so).
- A handoff whose message does not become a task within N poll cycles produces a loud log +
  controlplane alert; the empty-target embed logs as a warning, not routine.
- Retrigger emits an immediate log line at rejection-processing time naming the future work,
  so a watching operator sees "retrigger queued" within seconds even if execution follows
  minutes later.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Accept-then-work: the messages endpoint persists + responds, execution follows asynchronously | HTTP contract for every dispatch client | agent | design | med |
| D2: Dedup key = `correlation_id` (when present) with a bounded window; absent key = today's behavior | Defines what "the same work" means; opt-in via the field that already exists | agent | design | med |
| D3: Handoff watchdog lives in the daemon (poll-cycle check), not a new service | Smallest mechanism that closes the seam | agent | design | low |

### Design Freeze

None — all decisions agent-resolvable. (D2 is additive/opt-in: payloads without
`correlation_id` behave exactly as today, so no consumer needs migration.)

## Solution Design

### Overview

Three milestones, one per defect, each independently landable. All are fail-loud/contract
fixes in `internal/coordinator`; none touch executor, storage schema, or billing semantics.

### Conflict Surface

1. **Dispatch clients** (`ailang messages send --requires`, mission loops, dashboard) — D1 is
   contract-tightening (faster response, same success shape). Clients that already tolerate
   the slow path keep working; document the response body.
2. **Task polling** (`daemon_tasks_polling.go`) — D2 adds a pre-create lookup keyed on
   correlation metadata. Messages without the key are untouched; retriggers built from
   rejection feedback must CARRY the original correlation_id so retrigger-vs-manual-revision
   collapses to one task (this is the 2026-08-27 duplicate, fixed by construction).
3. **`daemon_approval.go` handoff path** — D3 adds observation, changes no routing; the
   auto-approve and merge-embedded branches keep their semantics.
4. **M-MESSAGE-PLANE-FAIL-LOUD (M1-M4, 2026-08-26)** — same family; reuse its alerting
   conventions (controlplane messages, CONFIG GAP-style logging), do not invent new channels.
5. **Programs that MUST still work**: tag-routed dispatch end-to-end (proven 2026-08-27),
   plain inbox sends, GitHub-driven approvals, the rejection feedback loop (including its
   retrigger, now with logging), `pkg-*` cloud-agent task flow.

### Implementation Plan

**M1 — accept-then-work + diagnosis (~1 day)**
- First DIAGNOSE the measured 30s+ non-response (read/trace `handlePostMessage`'s downstream;
  record the mechanism in this doc — the design deliberately does not assume it).
- Restructure: persist message → write response (id, `accepted`) → hand execution to the
  existing poll/exec machinery. No behavioral change to what eventually runs.
- **VERIFY**: integration test with an artificially slow executor: response < 2s, work still
  completes; the 2026-08-27 curl reproduction (send + immediate client timeout at 10s) now
  returns cleanly inside the window.

**M2 — correlation dedup (~1 day)**
- Pre-create lookup on `correlation_id` within a bounded window (existing non-terminal task
  with the same id → return it, log `dedup hit`); retrigger messages inherit the original
  correlation_id.
- **VERIFY**: (a) same payload sent twice → one task, second response names the first task;
  (b) replay of the 2026-08-27 sequence (retrigger + manual revision, same correlation_id)
  → one task; (c) two DIFFERENT correlation_ids → two tasks (no over-dedup).

**M3 — handoff/retrigger observability (~0.5 day)**
- Watchdog: handoff message id recorded at send; if no task references it after N poll
  cycles, log loudly + controlplane alert. Empty-target embed (`handoff to []`) logs as a
  warning. Rejection processing logs `retrigger queued (message <id>)` at decision time.
- Then DIAGNOSE the 14:19 dropped evaluator handoff with the new instrumentation (root cause
  currently unknown — the watchdog is the fix for the *class*; the specific bug gets found
  and fixed under it).
- **VERIFY**: (a) induced dropped handoff (send to an inbox no agent serves) alerts within
  N cycles; (b) a real eval-rig → sprint-evaluator handoff produces an evaluator task, or the
  alert fires and the root cause is recorded in this doc; (c) rejection via dashboard shows
  the retrigger-queued line within seconds.

### Files to Modify/Create

- `internal/coordinator/daemon_http.go` — accept-then-work (+~60 LOC)
- `internal/coordinator/daemon_tasks_polling.go` — dedup lookup (+~50 LOC)
- `internal/coordinator/daemon_approval.go`, `approval_processor.go` — retrigger correlation
  inheritance + decision-time logging (+~40 LOC)
- `internal/coordinator/daemon_handoff_watchdog.go` (new, +~80 LOC)
- tests across the four (+~250 LOC)

## Success Criteria

- [ ] Every milestone's VERIFY executed and recorded in this doc before the next begins
- [ ] The 2026-08-27 duplicate-dispatch sequence is a regression test and cannot recur
- [ ] The dropped-handoff class alerts; the specific 14:19 root cause found and fixed
- [ ] `make ci` green; no changes to executor/storage/billing semantics

## Non-Goals

- CLI store wiring ([m-coord-cli-shared-store](m-coord-cli-shared-store.md) owns it)
- The cloud sprint-executor image gap (multivac repo; controlplane ticket
  `inbox_1787832097692_40affb0c`)
- Message-plane routing semantics (M-MESSAGE-PLANE-FAIL-LOUD owns those; this doc reuses its
  conventions)
- Any approval-policy change (who approves what is untouched)

## Verification Log (first-party, 2026-08-27)

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Dispatch accepted but client timed out awaiting headers | 14:09:11 `curl` POST → client timeout; daemon log 14:09:11 message `inbox_1787832551135_08f450a0` created, 14:09:12 task `task-08f450a0` created + execution started, 14:09:14 observatory synced |
| V2 | The work proceeded to completion despite the "failed" client | task-08f450a0 reached "awaiting approval" 14:19:18 ($0.9776, 32,673 tokens) |
| V3 | No correlation dedup exists | `grep -rn correlation internal/coordinator/daemon_tasks_polling.go` → empty |
| V4 | Duplicate dispatch ran as two tasks | `task-5f0bad80` (14:31:50, retrigger, directive "Task rejected with feedback:", chain `f5443be7`) + `task-78a9624b` (14:33:48, manual revision) both created and executed for the same revision work |
| V5 | Retrigger works but is slow and silent | rejection POST accepted ~14:19-14:21 window closed with no retrigger logged; retrigger task appeared 14:31:50 (~12 min), no intermediate log lines |
| V6 | Handoff logged success, no task materialized | 14:19:18 `Auto-approved handoff from eval-rig to sprint-evaluator` (i.e. `sendHandoffMessage` returned nil per daemon_approval.go:79-86); zero `agent: sprint-evaluator` task creations in the following 2.5h |
| V7 | The evaluator inbox→task path itself works when fed directly | 2026-08-26 21:13:07 `Created task task-b88b68e3 … (agent: sprint-evaluator)` from a direct inbox message |
| V8 | Empty-target embed logs as routine | 14:19:17 `Embedding handoff to [] in merge approval for task task-08f450a0` |
| V9 | Handoff send path and its success-return shape | `daemon_approval.go` read: auto-approve branch calls `sendHandoffMessage`, logs success, no follow-up tracking of the message's fate |

## Related Documents

- [m-coord-cli-shared-store](m-coord-cli-shared-store.md) — sibling, same incident
- M-MESSAGE-PLANE-FAIL-LOUD (changelog [Unreleased] 2026-08-26) — the fail-loud family and
  alerting conventions this doc extends to the dispatch plane
- [m-eval-rolling-elo](m-eval-rolling-elo.md) — the sprint whose routing surfaced all of this
