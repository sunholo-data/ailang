# M-CASCADE-ENVELOPE-HYDRATION: close the inbox→task cascade envelope-loss routes

**Status**: Planned
**Target**: v0.44.x (next minor after v0.47.0 tooling — content version follows the planned/ folder convention)
**Priority**: P0 (measured prod flood: 173 duplicate EMPTY-dispatch bug reports in the last 5000 store messages alone; ~180 queued reports behind them)
**Estimated**: 1 day implementation + 0.5 day tests/review
**Dependencies**: None (builds on shipped M-PKG-CASCADE-DETERMINISTIC-FIRST and M-PKG-AUTONOMOUS-CASCADE-SAFE M1/M2)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cascade dispatch becomes a pure function of (message, payload): same inbox message → same task fields, on every route and every redelivery |
| A2: Replayability | +1 | Backstop-sweep recovery and Pub/Sub redelivery now produce identical envelope-populated tasks, so replays are faithful |
| A3: Effect Legibility | +1 | The cascade "authoritative bump" effect is carried explicitly in `Source`/envelope fields instead of being silently dropped at a route boundary |
| A4: Explicit Authority | +1 | Fail-loud guard refuses to spend dispatch authority on a cascade directive with no envelope behind it; no silent fallback value reaches business logic |
| A5: Bounded Verification | +1 | Deterministic-bump-vs-AI-escalation decision becomes locally checkable from the task record alone |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The wrapper's template guard (`{{.Source}} == "cascade"`) finally sees what the publisher stamped, instead of filing a `[bug] cascade routing` issue it cannot act on |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Stops paying full AI-escalation cost for tasks whose directive rendered blank |
| A10: Composability | 0 | No impact |
| A11: Structured Failure | +1 | Empty-envelope cascade messages fail loudly (NACK/log/refuse) instead of silently producing an unactionable task |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — hydration is a pure decode-and-copy
- [x] A3 (Effects): No hidden side effects — existing effects unchanged; one effect (dispatch) gains a precondition
- [x] A4 (Authority): No ambient access granted — the guard *narrows* what gets dispatched
- [x] A7 (Machines First): Not optimizing for human convenience — removes agent-side bug-report busywork

## Problem Statement

A cascade directive reaches a `pkg:` agent through any route other than the happy
Pub/Sub embedded-envelope path and arrives **empty**: `Source` unset, all envelope
fields blank. `buildTemplateDirective` (`internal/coordinator/stage_execution.go:219`)
then renders `tools/cloud-config/templates/pkg-update.md` with
`{{.RootPackage}}`/`{{.FromInterfaceHash}}`/… as blank strings, and the wrapper-side
guard (`pkg-update.md` "Defense-in-depth note": *if `Source` below is empty or
anything other than `cascade`, the wrapper has misrouted you — file a `[bug] cascade
routing` issue and stop*) correctly fires. The agent files a bug report and stops.

**Measured impact (2026-09-02 → 2026-09-27, message store `ailang-multivac-test`):**
173 `[bug] cascade routing: EMPTY dispatch` reports in the last 5000 store messages
alone, each one an agent task that cost money to conclude "nothing to repair", plus
~180 more queued behind them. This bug family was independently diagnosed by at
least seven parallel coordinator tasks (task-e891c221, task-ec1494ab, task-e705df2e,
task-d3f4bd97, task-caeb36fe, task-d06768f9, task-dfa31910) and triaged by
task-e1e2ce90 → `design-doc`.

### Verified mechanism (three lossy routes + one copy gap)

All claims verified by reading source at `fa2b3fac`:

1. **Publisher is correct.** `PublishCascadeWithEnvelope`
   (`internal/pubsub/publisher.go:119-165`) always stamps `source=cascade` and embeds
   the full `CascadeEnvelopeFields` in the data field (and in Pub/Sub attributes).
2. **Route A — adapter silent-envelope-drop.** `PubSubInboxAdapter.HandleNotification`
   (`internal/coordinator/pubsub_adapter.go:137-155`) decodes the cascade envelope
   ONLY if the data parses as `CascadeMessageData` with non-empty `message_id`;
   otherwise it falls back to the legacy notification-only decode, which leaves ALL
   cascade fields empty. A legacy-shaped or envelope-less message on the cascade
   topic silently produces a task with `Source=""` — exactly the EMPTY dispatch. (It
   does not even reach the payload-hydration step: `fullMsg.Payload` is a
   `PackageMessageEnvelope` JSON from `EmitUpgradeAvailable`
   (`internal/messaging/pkg_events.go:37-65`, stored via `ToInboxMessage` →
   `InsertInboxMessage`), and nothing reads it back for cascade fields.)
3. **Route B — backstop sweep drops cascade fields.** `BackstopSweep.SweepOnce`
   re-enqueues recovered messages via `s.adapter.Enqueue(&Message{…})`
   (`internal/coordinator/backstop_sweep.go:194-203`), deliberately preserving
   `Kind` and `CreatedAt` (post-2026-08-31 notice-flood fix) but NOT `Source` or any
   envelope field. `Enqueue` feeds the same drain, so the recovered cascade message
   becomes an EMPTY-dispatch task.
4. **Copy gap — local task creation.** The cloud task-creation site propagates
   envelope fields (`internal/coordinator/daemon_tasks_polling.go:~507-520`), but the
   local-drain site in `pollAndProcessTasks` (`daemon_tasks_polling.go:~203`) copies
   `Source: msg.Source` and **omits all 11 envelope fields** — so even a fully
   hydrated Message loses its envelope when a task is created on the local path.
5. **Downstream is faithful to what it is given.** Task record inherits
   `msg.Source`/envelope (both creation sites); the dispatcher injects
   `AILANG_CASCADE_*` env vars only `if params.X != ""`
   (`internal/dispatch/cloudrun/dispatcher.go:331-395`); the wrapper's guard
   (`cmd/ailang/coordinator_cloud.go:468-494`, `config.CascadeRootPackage()`)
   classifies and falls through to the AI executor, where the rendered directive's
   empty `{{.Source}}` triggers the template's file-a-bug-and-stop clause. **The
   wrapper-side guard WORKED; the coordinator's silent-empty path is the defect** —
   it violates CLAUDE.md principle 2 ("no silent fallbacks").

**Secondary defects in the same incident family (in scope, smaller):**

6. **Dispatch-time absence of a hard gate.** Nothing between Message and Cloud Run
   Job refuses a cascade-kind message whose `Source != "cascade"` or whose envelope
   is empty. Defense-in-depth currently relies on the template text inside the
   agent's context window.
7. **Correlation threading is ambiguous at best.** Completion notices set
   `CorrelationID: task.MessageID` (`internal/coordinator/pubsub_completion_handler.go:225`),
   but the *dispatch* record's message id does not survive into the completion chain
   (measured: task-9d0b57b3's completion pointed at an unrelated dedup completion,
   `inbox_1788179644941_9d0b57b3`). Task-dfa31910 requested this fix; it is bundled
   here because the same incident needs it for forensics.

**Impact:** every cascade bump that hits Pub/Sub redelivery, an eventual-consistency
gap, or the backstop sweep dispatches an unactionable task; the agent burns a full
task budget, files a duplicate bug report, and the flood masks real signals in the
coordinator inbox (103k+ unread).

## Goals

**Primary Goal:** Every cascade directive that reaches a task carries a fully
populated envelope from data that is already durable, and no task is ever dispatched
with a cascade template and an empty envelope.

**Success Metrics:**
- Re-running the three lossy routes against a real `PackageMessageEnvelope` payload produces a task with `Source="cascade"` and all envelope fields set (tests prove it)
- Zero new `[bug] cascade routing: EMPTY dispatch` reports after deploy (observable in the message store)
- A cascade-topic message with an unpopulated envelope is NACKed/logged loudly at the adapter, never dispatched
- Correlation ID of the dispatch survives into its completion notice (spot-checkable via `ailang messages list`)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Hydrate from the durable payload (`PackageMessageEnvelope` in `InboxMessage.Payload`) rather than persisting a parallel cascade envelope on `InboxMessage` | Avoids a second Firestore+SQLite schema migration and a second copy of the same data; the payload already carries everything | agent (this design) | design | low |
| `hydrateCascadeFields(msg)` populates `Source` ONLY from verified cascade evidence (Pub/Sub attributes or envelope-bearing payload), never guesses | A wrong `Source="cascade"` stamp is worse than an empty one: it makes the template's misroute guard lie | agent | design | med |
| Dispatch-time hard gate: refuse (loud, BLOCKED, no job) a cascade-template task with `Source != "cascade"` or empty required envelope fields | Defense-in-depth; converts a silent-empty path into a structured failure (A11) | human (approving this doc) | design | med |
| NACK vs ack for a cascade-topic message with unpopulated envelope at the adapter | NACK retries forever on permanently-bad data; ack loses the forensics record. Choose: NACK bounded, then dead-letter/log loudly | human | design | low |
| Where the sweep fix lives: inside `BackstopSweep` (copy cascade fields + hydrate from payload) vs inside `Enqueue` | Centralizing in `hydrateCascadeFields` called from both call sites keeps one mechanism | agent | design | low |
| Correlation fix scope: thread the dispatch id through task record → completion, not rewrite the chains model | Minimal change that restores forensibility | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Hydration source = payload-derived `PackageMessageEnvelope` (no new persisted fields)
- [x] `Source` stamping policy: verified evidence only
- [x] Dispatch gate refuses cascade tasks with empty `Source`/envelope (fail loud)
- [ ] Adapter NACK-vs-ack policy for unpopulated cascade envelopes — **needs operator input** (proposed: NACK with bounded retries via existing Pub/Sub retry policy, then ack + loud structured log; matches the existing "fetch nil → ack and say so loudly" pattern at `pubsub_adapter.go:229-234`)
- [x] Sweep fix routed through the shared helper (no separate mechanism)

## Solution Design

### Overview

One helper, three wiring points, one gate. `hydrateCascadeFields(msg *Message) *Message`
reads the message's durable context — the Pub/Sub attributes already parsed by the
adapter, and the hydrated `fullMsg.Payload` when it parses as a
`PackageMessageEnvelope` (`internal/messaging.ExtractPackageEnvelope` already exists,
`internal/messaging/pkg_schema.go:291`) — and populates `Source`, `RootPackage`,
`RootChangeClass`, `FromVersion`, `ToVersion`, `FromInterfaceHash`,
`ToInterfaceHash`, `FromContentHash`, `ToContentHash`, `EffectsWidened`,
`PrevEffectCeiling`, `NewEffectCeiling`. It is called:

1. in `HandleNotification` after Firestore hydration (`pubsub_adapter.go`, post-`fullMsg`),
2. in `BackstopSweep` before `adapter.Enqueue` (`backstop_sweep.go`),
3. nowhere else — the task-creation sites stay dumb copiers, but the **local** site
   (`daemon_tasks_polling.go:~203`) gets the same envelope-field copy block the
   cloud site already has (mechanical parity fix).

Then a dispatch-time gate refuses cascade tasks with an empty envelope, loudly.

The publisher, template, dispatcher env-injection, and wrapper are **not** changed:
they are already correct or already guard correctly. This is a coordinator-internal
plumbing fix.

### Architecture

**Components:**

1. **`hydrateCascadeFields`** (new, `internal/coordinator/cascade_hydration.go`):
   pure function, `Message`-in/`Message`-out (or in-place), no store access — takes
   the already-hydrated payload + attributes as inputs. Order of evidence (first
   wins, never merged): (a) Pub/Sub attributes on the cascade topic (cheap path, already
   published: `root_package`, `change_class`, `from_version`, `to_version`,
   `effects_widened` — `publisher.go:137-148`); (b) payload parsed as
   `PackageMessageEnvelope` (full fidelity incl. hashes and effect ceilings, via
   `messaging.ExtractPackageEnvelope`); (c) none → return unchanged (caller decides
   policy). Sets `Source="cascade"` only when (a) or (b) yielded envelope data AND the
   topic/attrs assert cascade.
2. **Adapter wiring** (`pubsub_adapter.go`): after Firestore hydration, call the
   helper with `msgAttrs` + `fullMsg.Payload`. If the message arrived on the cascade
   topic (`msgAttrs.Source == pubsub.SourceCascade`) and after hydration
   `msg.Source == ""` or `RootPackage == ""` → loud log + return error (NACK) — the
   measured trap: proceeding empty is the defect.
3. **Sweep wiring** (`backstop_sweep.go`): in the `Enqueue` block, extend the
   hand-built `Message` with fields recovered from the `messaging.InboxMessage`:
   `Source` (persisted on InboxMessage — `internal/coordinator/store.go:27` stores
   it) + helper over `m.Payload`. If the result is a cascade message with empty
   envelope fields → do NOT enqueue as a task; log loudly and leave for triage
   (withhold-for-triage, option 2 from task-e705df2e — the sweep has no redelivery
   backpressure to bound a NACK loop).
4. **Local-copy parity** (`daemon_tasks_polling.go` `pollAndProcessTasks`): add the
   same envelope-field copy block the cloud site has (11 lines, mechanical).
5. **Dispatch gate** (`daemon_tasks_exec.go` or the dispatch params assembly): the
   cascade shape of a task is detectable at dispatch time from its Kind — the
   publisher stamps `MessageType: "upgrade-available"`
   (`pkg_publish.go:483-487`) and the agent's `TemplateByMessageType` maps
   message types to templates (`agent_registry.go:24-28`), so Kind
   `upgrade-available`/`interface-change-notice` on a `pkg:` agent identifies a
   cascade task. If (`Source != "cascade"` OR `RootPackage == ""` OR
   `ToVersion == ""`) → mark task BLOCKED with a loud, structured error naming the
   message id; no Cloud Run Job. Mirrors CLAUDE.md principle 2 at the dispatch
   boundary.
6. **Correlation threading** (small): ensure the completion notice's
   `CorrelationID` is the **dispatch** message id (the `task.MessageID` chain) and
   that `publishDedupCompletion` (`daemon_tasks_polling.go:661`) cannot shadow a real
   task id in correlation space (measured: a dedup completion's id was mistaken for
   the dispatch's). Concretely: completion `Title` already carries the task id; add
   the task id to the dedup payload's correlation only when it *is* the dispatch id.

### Implementation Plan

**Phase 1: Helper + tests** (~2 hours)
- [ ] Add `internal/coordinator/cascade_hydration.go` with `hydrateCascadeFields`
- [ ] Unit tests: attributes-only, payload-only, both, neither; never stamps `Source` without evidence; legacy-shaped cascade data (envelope nil) falls through to payload hydration
- [ ] Extend `pubsub_adapter_hydration_test.go`: cascade-topic + empty-envelope data → NACK + loud log; cascade-topic + envelope → populated Message

**Phase 2: Route fixes** (~2 hours)
- [ ] Wire helper into `HandleNotification` (post-hydration) + NACK-on-empty policy
- [ ] Wire helper into `BackstopSweep` (withhold-on-empty policy) + extend `backstop_sweep_test.go`
- [ ] Add envelope-field copy block to local `pollAndProcessTasks` site

**Phase 3: Gate + correlation** (~2 hours)
- [ ] Dispatch-time gate in the cloud dispatch path with BLOCKED status + structured log
- [ ] Correlation threading fix + test that a task's completion carries the dispatch's message id
- [ ] `make test` green; `make lint` clean

**Phase 4: Verification against the real flood** (~1 hour)
- [ ] Confirm on the test store: reprocessed cascade messages produce populated tasks
- [ ] Sweep the accumulated duplicate `[bug] cascade routing` reports (operator-approved batch ack) — post-merge housekeeping, not part of the code change

### Files to Modify/Create

**New files:**
- `internal/coordinator/cascade_hydration.go` (~80 LOC) — `hydrateCascadeFields` helper
- `internal/coordinator/cascade_hydration_test.go` (~150 LOC) — helper + policy tests

**Modified files:**
- `internal/coordinator/pubsub_adapter.go` (+15 LOC) — post-hydration call + empty-envelope NACK
- `internal/coordinator/backstop_sweep.go` (+12 LOC) — sweep copies Source/envelope; withholds empty-envelope cascade messages
- `internal/coordinator/daemon_tasks_polling.go` (+12 LOC local copy block, +10 LOC correlation) — parity with cloud site; correlation fix
- `internal/coordinator/daemon_tasks_exec.go` or dispatch assembly (+20 LOC) — cascade gate
- `internal/coordinator/pubsub_adapter_hydration_test.go` (+60 LOC) — new cases
- `internal/coordinator/backstop_sweep_test.go` (+50 LOC) — cascade recovery cases

## Examples

### Example 1: Backstop-sweep recovery of a cascade message (the measured bug)

**Before** (`backstop_sweep.go:194-203` + `pollAndProcessTasks`):
```
InboxMessage{ID: "inbox_1788...", MessageType: "upgrade-available",
             Payload: '{"schema":"pkg-msg/1","kind":"upgrade-available",
                        "package":{"name":"sunholo/auth","from_version":"1.2.0",
                        "to_version":"1.3.0","change_class":"C",...}}'}
  ↓ Enqueue(&Message{ID, From, Title, Content, Inbox, Type, Kind, CreatedAt})
Message{Source: "", RootPackage: "", ...}
  ↓ pollAndProcessTasks → TaskRecord{Source: "", RootPackage: ""}
  ↓ dispatcher: no AILANG_CASCADE_* env vars
  ↓ pkg-update.md renders: "The cascade root package `` had a class **`` change"
  → agent: "if Source below is empty ... file a [bug] cascade routing issue and stop"
```

**After:**
```
  ↓ hydrateCascadeFields (payload parses as PackageMessageEnvelope)
Message{Source: "cascade", RootPackage: "sunholo/auth", RootChangeClass: "C",
        FromVersion: "1.2.0", ToVersion: "1.3.0", FromInterfaceHash: "...", ...}
  ↓ TaskRecord inherits all fields → AILANG_CASCADE_* env vars set
  → wrapper classifies DispatchDeterministic-vs-AI with real data; PR titled [cascade]
```

### Example 2: Legacy-shaped data on the cascade topic (the adapter trap)

**Before** (`pubsub_adapter.go:143-155`): data = `{"message_id": "msg_...", ...}`
without an envelope (older publisher or a non-cascade publisher on the cascade
topic) → `cascadeData.Envelope == nil` → silently proceeds with an empty envelope →
EMPTY dispatch.

**After**: hydration falls through to the stored payload; if the payload is a
`PackageMessageEnvelope` the task is fully populated; if it is NOT (nothing durable
to hydrate from), the cascade-topic assertion contradicts the data → NACK + loud
log ("cascade topic delivered message %s with no envelope and no hydratable
payload — refusing to dispatch"), per the no-silent-fallbacks principle.

## Success Criteria

- [ ] All three lossy routes produce fully populated cascade tasks (unit tests on helper + adapter + sweep)
- [ ] Cascade-topic message with unpopulated envelope is never dispatched as a task (adapter test + dispatch gate test)
- [ ] Local task-creation site copies envelope fields identically to the cloud site (parity test)
- [ ] Dispatch gate marks BLOCKED (with structured error) instead of dispatching empty
- [ ] A task's completion notice carries the dispatch's correlation id (test)
- [ ] All tests passing (`make test`)
- [ ] Documentation updated (`docs/internal/message-plane-topology.md` cascade section, if it names these routes)
- [ ] Zero new EMPTY-dispatch reports on the test store after deploy (post-merge observation, not a merge gate)

## Testing Strategy

**Unit tests:**
- `hydrateCascadeFields`: evidence order (attributes > payload), never-guess `Source`, all 11 envelope fields, `EffectsWidened` bool, effect-ceiling slices
- Adapter: cascade attrs + empty envelope + hydratable payload → populated; cascade attrs + nothing hydratable → NACK (error return); non-cascade topic → unchanged behavior (legacy messages must not start failing)
- Sweep: recovered cascade message → populated Message handed to Enqueue; recovered cascade message with empty envelope → NOT enqueued, loud log, counted
- Dispatch gate: cascade template + empty Source → BLOCKED, no job params built; non-cascade tasks unaffected

**Integration tests:**
- Full drain path: `Enqueue` → `pollAndProcessTasks` → TaskRecord fields populated (covers the local-copy gap)
- Legacy-notification route (non-envelope data, non-cascade topic) still dispatches — regression guard for the tag-filter/ack behavior

**Regression-surface tests** (see Conflict Surface):
- Existing `TestPubSubAdapter_*` hydration tests stay green (fetch-error NACK, missing-message ack, empty-content ack, real-message dispatch)
- Existing `backstop_sweep_test.go` notice-flood protections stay green (Kind/CreatedAt preservation)

**Manual testing:**
- `ailang coordinator start` on the test project with a seeded cascade inbox message; observe populated dispatch in logs

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact error/log wording and log levels — agent may choose (keep the message id + reason in every line)
- Whether the gate lives in `daemon_tasks_exec.go` (daemon-level) or in the dispatch params builder — agent may choose, as long as the refusal happens before the Cloud Run Job is created
- Helper signature (in-place vs copy-return) — agent may choose
- Whether to also surface `RootContentHash`/`ToContentHash` env vars (publisher sends them in the envelope but the dispatcher only injects interface hashes today) — agent may choose; do not silently drop them from the Message→TaskRecord copy either way
- Dead-letter mechanics for the bounded-NACK policy — human at review (depends on the subscription's retry/DLQ configuration, which is infrastructure)

## Non-Goals

**Not attempted in this feature:**
- Rewriting the publisher or the `PackageMessageEnvelope` schema — publisher verified correct (`publisher.go:119-165`)
- Changing the wrapper's guard or `pkg-update.md` template text — the guard worked; it is the only reason this bug was visible at all
- Persisting cascade envelope as new Firestore/SQLite columns — payload-derived hydration makes it unnecessary; revisit only if payloads ever lack the envelope while attrs do too
- Fixing the docparse routing collision audit (task-dfa31910 item 4) and `pkg-update.md`'s hardcoded `sunholo-data/ailang-packages` repo line (task-d3f4bd97 item 3) — separate, smaller changes; noted here so they are not lost
- Replaying the ~180 queued duplicate bug reports or retriggering the original cascade publishes — operator action after merge, from the ORIGINAL Pub/Sub payloads (per task-e1014157: never synthesize envelopes by hand)
- The `create_planned_doc.sh` script bugs found while writing this doc (`grep -E "^\d+\."` never matches, so the related-doc search always reports none; and `set -euo pipefail` + empty grep aborts the script before file creation) — separate trivial fix, awaiting explicit OK per the work-routing gate

## Timeline

**Day 1** (~7 hours):
- Phase 1: helper + tests (2h)
- Phase 2: adapter + sweep + local-copy fixes (2h)
- Phase 3: dispatch gate + correlation (2h)
- Phase 4: verification + review (1h)

**Total: ~7 hours across 1 day** (2x-buffered estimate)

## Verification Log

Every negative-existence / mechanism claim this design relies on, verified before
submission (per design-doc-creator hard gate):

| # | Claim | Check | Result |
|---|-------|-------|--------|
| V1 | No `hydrateCascadeFields` (or any cascade hydration helper) exists today | `grep -rn "hydrateCascadeFields\|hydrateCascade" internal/ cmd/` | empty — confirmed, nothing to reuse |
| V2 | The local task-creation site omits envelope fields; the cloud site copies them | read `daemon_tasks_polling.go:~203` (no `RootPackage`) vs `:~507-520` (copies all 11) | confirmed |
| V3 | The sweep's `Enqueue` call omits `Source` and envelope fields | read `backstop_sweep.go:194-203` | confirmed — only ID/From/Title/Content/Inbox/Type/Kind/CreatedAt |
| V4 | The dispatcher injects no `AILANG_CASCADE_*_CONTENT_HASH` env vars (only interface hashes) | `grep -rn "AILANG_CASCADE_FROM_CONTENT\|AILANG_CASCADE_TO_CONTENT" internal/ cmd/` | empty — confirmed |
| V5 | The legacy inbox path stores the full `PackageMessageEnvelope` as the message payload (so payload-derived hydration is possible) | read `pkg_events.go:37-65` → `sendPackageMessage:265` → `ToInboxMessage:260-286` (payload = `PackageEnvelopeToJSON`) | confirmed |
| V6 | `ExtractPackageEnvelope` already exists for payload parsing | read `pkg_schema.go:291-300` | confirmed — returns `(nil, nil)` for non-package payloads, safe to call on anything |
| V7 | The publisher stamps `Source` unconditionally and embeds the envelope | read `publisher.go:119-165` | confirmed |
| V8 | The `Message`/`TaskRecord` structs already carry all 11 cascade fields (no struct changes needed) | read `watcher.go:40-52`, `store.go:85-98` | confirmed |
| V9 | The adapter's existing hydration-failure policy (error → NACK; missing message → ack loudly) is the precedent for the new empty-envelope policy | read `pubsub_adapter.go:216-248` | confirmed — pattern reused, not invented |
| V10 | The sweep is dedup-safe for recovered messages (`Collapsed: true`) | read `backstop_sweep.go:128-131` | confirmed — ListInboxMessages respects semantic dedup |
| V11 | Cascade Kind detection at dispatch: publisher uses `MessageType: "upgrade-available"` | read `pkg_publish.go:483-487` | confirmed |
| V12 | `isOutcomeNotice` already keeps completion/approval notices out of the drain (no new sweep-exclusion needed) | read `daemon_tasks_polling.go:67-77` | confirmed |

**Language-claim gate (ailang check):** N/A — this design touches only Go
coordinator plumbing; it asserts no AILANG syntax/semantics support claims.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| NACK-on-empty loops forever on permanently-bad cascade data | Medium | Bounded by Pub/Sub's own retry policy + dead-letter (existing infra); ack-and-log-loudly fallback if no DLQ — decide in Design Freeze |
| Legacy (pre-envelope) publishers on the cascade topic start failing after the gate | Medium | Hydration first (payload may still carry the envelope); only refuse when NOTHING durable can populate the envelope — matches the measured failure mode, where the payload existed and was simply never read |
| Sweep withholding cascade messages stalls a bump if push is down | Low | Accepted trade-off (stricter authority guarantee), flagged by task-e705df2e; the sweep's per-pass log line makes the stall visible; operator can re-trigger from the original publish |
| Double-stamping `Source` from attributes + payload disagreement | Low | Evidence order is exclusive (first wins), unit-tested; disagreement itself gets a loud log line |
| The ~180 queued duplicate reports keep flooding after the fix | Low | They are products of the old path; post-merge batch ack (operator-approved), and the dispatch gate turns any future recurrence into a BLOCKED task instead of an agent run |

## Related Documents

<!-- Auto-populated by Ollama neural search on "cascade envelope hydration"
     (search silently returned no matches due to a script bug — see Non-Goals;
     the docs below were located by grep of design_docs/ instead) -->

**Implemented (may inform design):**
- [m-pkg-cascade-deterministic-first](../implemented/v0_16_0/m-pkg-cascade-deterministic-first.md) — the embedded-envelope publish path and dispatcher env vars
- [m-pkg-autonomous-cascade-safe](../implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md) — `Source` stamping and the cascade-vs-public distinction (M1/M2)

**Planned (check for overlap):**
- [m-cascade-observability](v0_29_0/m-cascade-observability.md) — cascade span observability; no envelope-loss rules
- [m-pkg-quality-ladder](v0_40_0/m-pkg-quality-ladder.md) — package agent template derivation (M6), relevant to the routing audit follow-up

**Triage row:**
- `design_docs/planned/ailang-core-triage/cascade-directive-unpopulated.md` (task-e1e2ce90, branch `coordinator/task-e1e2ce90`) — recommends design-doc; this doc is its execution

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Root-cause report: message `inbox_1790551573874_7eb9c970` (task-e1014157) — verified evidence + secondary defects
- Converging coordinator diagnoses: tasks e891c221, ec1494ab, e705df2e, d3f4bd97, caeb36fe, d06768f9, dfa31910 (coordinator inbox, 2026-09-27)
- CLAUDE.md principle 2 ("No silent fallbacks — fail loudly") — the principle the silent-empty path violates
- CLAUDE.md "Look the question up, not the tool" table — the instrument guidance used to verify against OpenRouter/pubsub evidence rather than theorize

## Future Work

- Envelope-preserving dual-write: persist the cascade envelope onto `InboxMessage` (Firestore + SQLite) if payload-derived hydration ever proves insufficient
- Automated routing audit: registry-level check that every `pkg:<name>` inbox an actual publish names resolves to a derived or hand-written agent (kills the docparse misroute class)
- Un-drift `pkg-update.md`: parameterize the hardcoded `sunholo-data/ailang-packages` repo claim (task-d3f4bd97 item 3)

---

**Document created**: 2026-09-27
**Last updated**: 2026-09-27
