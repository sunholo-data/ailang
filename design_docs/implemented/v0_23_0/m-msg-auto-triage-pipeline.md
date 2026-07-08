# M-MSG-AUTO-TRIAGE-PIPELINE: Autonomous Inbound Triage + Central Notification Bus

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium)
**Estimated**: ~4 weeks (phased — each milestone independently shippable)
**Dependencies**: M-MSG-AGENT-MESSAGING (v0.5.6, shipped), M-PKG-AUTONOMOUS-CASCADE-SAFE (v0.16.0, shipped), M-COORD-MULTI-HOST-WORKERS (v0.22.0, shipped), M-NOTIFY-CHANNELS-FRAMEWORK (companion — owns channel delivery)

> **Scope split:** this doc owns triage + auto-draft + the notification **bus** (the dispatcher that decides *what* to notify and *which channel name(s)* to route to). The channel **framework** (adapter interface, registry, transports like Discord/Google Chat/email, and the channel *choice*) lives in [m-notify-channels-framework.md](m-notify-channels-framework.md). The seam: the dispatcher calls `registry.Get(name).Send(intent)`.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is orchestration/infrastructure work (coordinator, Pub/Sub, notification fan-out). It does not change language semantics, so most axioms are neutral; the positives come from keeping authority, effects, and failures explicit at the new system boundaries.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Triage clustering/threshold is deterministic; the *drafting* step is an LLM agent (nondeterministic) but is gated behind human approval — no nondeterminism enters language semantics |
| A2: Replayability | +1 | Every triage decision and notification fan-out is an event on the `events` topic; the message → doc → approval chain is fully reconstructable from the event log |
| A3: Effect Legibility | +1 | Notification sends are an explicit `Msg`/channel effect; no channel emits implicitly — each fan-out is logged |
| A4: Explicit Authority | +1 | New channels (email, cloud routes) require explicit config + IAM, mirroring the cascade topic's authenticated-source guard; no ambient send authority |
| A5: Bounded Verification | 0 | No change to local verification |
| A6: Safe Concurrency | 0 | Reuses existing coordinator worktree isolation + Pub/Sub at-least-once semantics |
| A7: Machines First | +1 | Structured triage envelopes + design docs are machine-legible artifacts, not free-text; reduces human-in-loop token cost |
| A8: Minimal Syntax | 0 | No new language syntax |
| A9: Cost Visibility | +1 | Each auto-drafted doc carries the executor's token/cost metrics (already on `TaskCompletion`); triage surfaces cost before promotion |
| A10: Composability | +1 | Composes the existing `triage`, `forward`, ApprovalWatcher, and dev-cycle pieces rather than replacing them |
| A11: Structured Failure | +1 | Notification + dispatch failures route to the existing `ailang-dead-letter` topic; no silent drops |
| A12: System Boundary | 0 | Boundary crossings (email, cloud) remain explicit channel config |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism in language; LLM drafting is human-gated
- [x] A3 (Effects): Notification sends are explicit, logged channel effects
- [x] A4 (Authority): New channels require explicit config + IAM; no ambient send
- [x] A7 (Machines First): Structured artifacts (envelopes, docs, events), not human-convenience shortcuts

## Problem Statement

The repo already has every part needed to turn an inbound agent message into an approved, scheduled design doc — but the parts are **not connected end-to-end**, and notifications can't reach the developer when they're away from a Claude Code session.

**Current State:**

- **Triage → design doc is manual.** Commit `d6990e03` ("triage 2 inbound docparse reports into design docs") is the proof: two `sunholo/docparse` bug/feature reports arrived via `ailang messages`, and a human had to read each one, judge priority, and hand-run `design-doc-creator` for both. The `dev-cycle` agent ([.claude/agents/dev-cycle.md](.claude/agents/dev-cycle.md)) automates the *chain* once started, but Stage 1 (TRIAGE) requires a human to say "start dev cycle" and pick a message.
- **Semantic triage exists but isn't wired to action.** `ailang messages triage --cluster-by intent|code|context|skill|resolution` ([cmd/ailang/messages.go](cmd/ailang/messages.go)) clusters unread messages by semantic envelope slot, but the output is informational — nothing promotes a cluster into a `design-doc-creator` task.
- **The approval chain only starts mid-stream.** `ApprovalWatcher` ([internal/coordinator/approval_watcher.go](internal/coordinator/approval_watcher.go)) drives `design-approved` → `sprint-approved` → `merge-approved` handoffs between configured agents, but the *first* hop (raw message → `design-doc-creator` inbox) has no automation.
- **No notification reaches the developer off-session.** Notifications today are: the SessionStart hook (only fires when a Claude Code session starts), GitHub issues (only if the dev is watching), and the dashboard (only if it's open). There is **no email and no central place to configure routing** — confirmed: zero SMTP/SendGrid/Mailgun/Gmail integration in the codebase.
- **Pub/Sub bus exists but isn't used for notifications.** [internal/pubsub/topics.go](internal/pubsub/topics.go) defines 6 topics including `ailang-events` (task start/complete/error, handoff, approval, message, session) with `events-dashboard` and `events-laptop` subscriptions — but nothing fans those events out to human-reachable channels.

**Impact:**

- **Who:** The developer (single maintainer running parallel agents on a Mac Studio, increasingly from a laptop / GitHub / cloud) and the autonomous agents that file reports into `ailang messages`.
- **How significant:** Inbound reports sit unread until a session starts; triage is a recurring manual chore (every new external-project report repeats the `d6990e03` flow); and the dev has no way to be paged when something needs approval while away from a terminal. This is the bottleneck that caps how autonomous the system can run.

## Goals

**Primary Goal:** A new inbound message classified as a clear bug/feature is automatically triaged and drafted into a `design_docs/planned/` doc that sits at `pending_approval`, with the developer notified through their configured channel(s) — no human action required until the approval decision.

**Success Metrics:**

- Manual steps from inbound message → draft design doc awaiting approval: **3+ → 0** (human's first touch is the approve/reject decision).
- Notification reach: from **session-only** → **email + Claude Code + dashboard + GitHub**, all configured in **one** config block.
- Triage precision: ≥ 90% of auto-promoted messages are accepted (not rejected as "shouldn't have been a doc") in the first month, measured from approval/rejection labels.
- Time-to-notify for an approval-pending item: from **unbounded (until next session)** → **< 2 min** via the events bus.
- Zero new manual orchestration code: M1 reuses `triage`, `forward`, the coordinator agent runner, and `ApprovalWatcher` — the connective tissue is config + one router, not a new pipeline.

## Build Order & Sequencing

This milestone and its companion [m-notify-channels-framework.md](m-notify-channels-framework.md) ship in this order. The two docs meet only at the `registry.Get(name).Send(intent)` seam, so steps 1 and 2 can run as **parallel agents**.

| Step | Milestone | Doc | Depends on | Parallel? |
|------|-----------|-----|------------|-----------|
| 1 | Auto-triage → auto-draft | this (M1) | nothing new | ‖ with step 2 |
| 2 | Channel framework + first webhook channel | companion (M1) | channel pick | ‖ with step 1 |
| 3 | Notification bus / dispatcher | this (M2) | steps 1 + 2 | — (the join) |
| 4 | Second channel + dead-letter | companion (M2) | step 3 | — |
| 5 | Open up routing | this (M3) | step 3 | — |

**Rationale:** value-first (1 delivers autonomous draft-to-approval using existing surfaces), then close the missing egress (2 + 3 page the dev off-session), then harden + scale (4, 5). The riskiest integration (3) happens only after both halves exist and are individually tested. Each step is independently shippable.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Human approval gate sits at the **design doc** (`design-approved` label), pipeline auto-runs triage→draft then STOPS | Defines the trust boundary; reuses existing `ApprovalEventDesign` path; full-auto-to-PR would need a different gate and far more trust | human | design | high |
| **Pub/Sub `events` topic is THE central notification bus**; all channels (email, Claude Code, dashboard, GitHub) are fan-out subscribers configured in one registry | Architectural — determines that channels never publish directly to each other and that one config block governs routing | human | design | high |
| Triage **promotion criteria**: only clear `bug`/`feature` category messages above a confidence threshold auto-draft; ambiguous/low-confidence stay queued for human pick | Sets the auto vs human-pick line; too loose floods `planned/` with noise, too tight defeats the purpose | human | design | med |
| **Notification dispatcher runs in the coordinator daemon (local, Mac Studio)** — the Studio is always-on, so a local dispatcher is the primary path; a cloud dispatcher is an optional redundant extra | Determines the always-on assumption; if the Studio were ever off, only the optional cloud path would notify | human | design | med |
| Triage **trigger cadence**: debounced schedule (batch related messages) vs per-message on `ailang-messages` arrival | Affects clustering quality; debounce lets related reports group into one doc | agent | implementation | low |

> Channel delivery (email/Discord/Google Chat) and the channel *choice* are decided in [m-notify-channels-framework.md](m-notify-channels-framework.md), not here.

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Approval gate at the design doc (`design-approved`), not full-auto-to-PR — **confirmed**
- [x] Pub/Sub `events` topic as the central notification bus — **confirmed**
- [x] Dispatcher runs in the coordinator daemon (Studio always-on); cloud dispatcher is an optional extra — **confirmed**
- [ ] Promotion criteria: exact category set + confidence threshold value + what "ambiguous" routes to (dashboard queue vs `user` inbox)
- [ ] (Channel choice + email provider are resolved in [m-notify-channels-framework.md](m-notify-channels-framework.md), not here)

## Solution Design

### Overview

Three phases, each independently shippable, that **connect existing parts** rather than build new machinery. (Channel delivery — the actual transports — is a fourth concern owned by the companion doc [m-notify-channels-framework.md](m-notify-channels-framework.md).)

1. **M1 — Auto-triage → auto-draft (the connective tissue).** A triage router runs the existing `ailang messages triage` clustering on unread `user`/`claude-code`-inbox messages, scores each cluster, and `forward`s promoted ones to the `design-doc-creator` inbox. The coordinator's existing agent runner picks them up, runs `design-doc-creator` headlessly in a worktree, produces a `planned/` doc + linked GitHub issue, and stops at `pending_approval`. The human's first touch is applying `design-approved` — which the **existing** `ApprovalWatcher` already turns into the `sprint-planner` → `sprint-executor` chain.

2. **M2 — Central notification bus.** Promote the `ailang-events` topic to the canonical notification source and add a **channel registry config** in `~/.ailang/config.yaml`. A notification dispatcher (running in the always-on coordinator daemon) subscribes to `events`, matches each event against channel filters, and fans out by calling `registry.Get(name).Send(intent)` — where the channels come from the companion framework. Existing surfaces (Claude Code hook, dashboard, GitHub) become registered channels; the bus is the single routing-config point. A cloud-side dispatcher is an optional redundant extra.

3. **M3 — Open up routing.** Extend inbound sources (the packages/publish system, GitHub, future routes) to feed `ailang messages`, and let the triage→doc pipeline run on cloud workers (multivac Cloud Run) as well as the local Mac Studio, routed by the existing `requires` worker tags.

### Architecture

**Components:**

1. **Triage Router** (new, M1): reads unread messages from configured "intake" inboxes, runs envelope clustering + a promotion classifier, and forwards promotions to `design-doc-creator`. Lives alongside the coordinator. Deterministic given the same message set + thresholds.
2. **Promotion Classifier** (new, M1): maps a message/cluster → `{promote | hold | drop}` using `category` (bug/feature/general), envelope confidence, and dedupe state (`dup_of`). `hold` → surfaced for human pick; `drop` → archived (e.g., eval-suite noise).
3. **Coordinator Agent Runner** (exists): already runs configured agents on their inboxes in worktrees. `design-doc-creator` is already a configured agent (`inbox: design-doc-creator`, `trigger_on_complete: [sprint-planner]`). No change needed beyond ensuring it produces a `design-approved`-able issue.
4. **ApprovalWatcher** (exists): `ApprovalEventDesign` ("design-approved") → `sprint-planner`; `ApprovalEventSprint` → `sprint-executor`; `ApprovalEventMerge` → merge. Unchanged.
5. **Notification Bus** (M2): the `ailang-events` topic + a **Notification Dispatcher** (in the coordinator daemon) that subscribes once, matches events to channels, and fans out by calling `registry.Get(name).Send(intent)`. Failures → `ailang-dead-letter`.
6. **Channel Registry config** (new config, M2): declarative list of channels in `~/.ailang/config.yaml`, each `{transport, filter}`. The dispatcher reads this to decide *which* channels fire for an event. The transports themselves (`discord`, `googlechat`, `email`, `claude-code`, `dashboard`, `github`) are implemented by the **companion framework**, not here.
7. **Channel framework** (companion doc): the `Channel` interface + registry + concrete transports. See [m-notify-channels-framework.md](m-notify-channels-framework.md). The dispatcher depends only on `registry.Get(name).Send(...)`.

### Data Flow

```
                          intake inboxes (user, claude-code, pkg:*, github-imported)
                                          │
                                          ▼
                              ┌───────────────────────┐
                              │   Triage Router (M1)   │  ailang messages triage --cluster-by
                              │   + Promotion Classifier│  (deterministic: category+envelope+dedupe)
                              └───────────┬───────────┘
                         promote │        │ hold            │ drop
                                 ▼        ▼                 ▼
                    forward → design-doc-  surfaced in    archived
                    creator inbox          dashboard /
                                 │         user inbox
                                 ▼
                    ┌────────────────────────┐
                    │ Coordinator Agent Runner│  (exists) headless design-doc-creator
                    │ → worktree → planned/doc│  → GitHub issue, status=pending_approval
                    └───────────┬────────────┘
                                │ emits events  ─────────────────────────────┐
                                ▼                                            ▼
                    [HUMAN: design-approved label]              ailang-events topic (M2 bus)
                                │                                            │
                                ▼                  Notification Dispatcher (M2, coordinator daemon)
                        ApprovalWatcher (exists)   matches event→channels; for each: registry.Get(name).Send
                        → sprint-planner → executor                       │
                                                   ┌────────┬────────┬────┴───────────────┐
                                                   ▼        ▼        ▼                     ▼
                                              claude-code dashboard github       discord/googlechat/email
                                                  └─── channel transports (companion framework) ────┘
```

### Implementation Plan

**Phase M1: Auto-triage → auto-draft** (~1.5 weeks) — **shipped (M-MSG-TRIAGE-ROUTER, v0.23.0)**
- [x] Define promotion criteria config (`coordinator.triage`) — intake inboxes, category allow-list, threshold, defaults
- [x] Implement Triage Router: classify (promote/hold/drop) + `forward` promotions to `design-doc-creator` inbox (reuses `ForwardInboxMessage`)
- [x] Wire trigger: debounced coordinator tick (`pollLoop`, default 120s), opt-in via `coordinator.triage.enabled`
- [ ] Ensure headless `design-doc-creator` run produces a `design-approved`-able GitHub issue + `DESIGN_DOC_PATH` marker (the skill already emits this marker) — relies on existing coordinator agent runner; verify end-to-end with the router enabled
- [ ] `hold` path: surface ambiguous clusters in the dashboard triage view / `user` inbox — deferred (currently no-op; holds stay visible in the intake inbox)
- [x] Integration test: synthetic bug message → router → design-doc-creator inbox (real SQLite store)

**Phase M2: Central notification bus** (~1 week) — **mostly shipped (M-NOTIFY-FANOUT, v0.23.0)**
> **Discovery:** the dispatcher already existed as `internal/daemon` (a laptop-side daemon wired in `cmd/ailang/daemon.go`), *not* the coordinator daemon. It subscribes to the `events`/`messages` Pub/Sub subs, maps task `pending_approval`/`completed`/`failed` + inbox messages → notifications, dedups, excludes, and nacks-on-failure. Step 3 generalized its single macOS notifier into a multi-channel fan-out.
- [x] Fan-out dispatcher: `internal/daemon` now delivers over a `notify.Registry` (macOS + env-gated Discord) via `SendAll` — local best-effort, remote authoritative
- [x] Reliable retry: `dedup.forget` on nack paths so a failed remote send is actually redelivered (was eaten by the pre-fire dedup)
- [ ] Per-channel event filters / `notifications.channels` config (`[{transport, filter}]`) — deferred; today all relevant events fan out to all registered channels (the exclude-substring filter still applies)
- [ ] Explicit `ailang-dead-letter` routing on total failure — deferred to Step 4 (Pub/Sub nack/redelivery is the current safety net)

> Channel transports (Discord today; Google Chat/email later) live in the companion milestone [m-notify-channels-framework.md](m-notify-channels-framework.md). The dispatcher depends only on the `Channel.Send` seam.

**Phase M3: Open up routing** (~1 week)
- [ ] Add packages/publish system as an intake source (cascade `upgrade-available` → triage where appropriate)
- [ ] Allow triage→doc pipeline to run on cloud workers via `requires` tags (route heavy drafting to multivac when Mac Studio is busy)
- [ ] Document adding a new inbound route (extension guide)

### Files to Modify/Create

**New files:**
- `internal/coordinator/triage_router.go` (~250 LOC) — Triage Router + Promotion Classifier
- `internal/coordinator/triage_router_test.go` (~200 LOC) — promotion decisions, dedupe handling, hold/drop routing
- `internal/notify/dispatcher.go` (~200 LOC) — events subscriber + channel fan-out (calls the companion framework's `Registry.Get(name).Send`)
- `internal/notify/dispatcher_test.go` (~150 LOC) — filter matching, dead-letter on failure

> The `Channel` interface, `Registry`, and concrete transports live in [m-notify-channels-framework.md](m-notify-channels-framework.md) (`internal/notify/channel.go`, `registry.go`, `discord.go`, etc.). This doc adds only the dispatcher that drives them.

**Modified files:**
- `internal/messaging/config.go` (+60 LOC) — `coordinator.triage` + `notifications.channels` config structs
- `internal/coordinator/daemon_tasks_init.go` (+40 LOC) — register Triage Router tick
- `internal/pubsub/topics.go` (+20 LOC) — notification event types / channel attribute helpers if needed
- `cmd/ailang/messages.go` (+30 LOC) — `ailang messages triage --auto-route` flag to run the router on demand (dry-run + apply)
- `~/.ailang/config.yaml` (config, not code) — `coordinator.triage` + `notifications.channels` blocks

## Examples

### Example 1: Inbound docparse report (the d6990e03 flow, automated)

**Before (manual — what actually happened in commit d6990e03):**
```
1. Agent files report → ailang messages (cli inbox)
2. Human starts a session, sees unread in SessionStart hook
3. Human reads each message, judges priority, decides "this is a P1 feature"
4. Human runs design-doc-creator skill, fills template, picks version folder
5. Repeat for the second report
6. Human commits both docs
```

**After (M1 — automated up to the approval decision):**
```
1. Agent files report → ailang messages (cli inbox)
2. Triage Router tick: clusters unread, classifies "feature, confidence 0.91" → promote
3. forward → design-doc-creator inbox
4. Coordinator runs design-doc-creator headlessly → planned/v0_24_0/m-….md + GitHub issue,
   status = pending_approval
5. Notification Dispatcher → email: "1 design doc awaiting approval: callJsonSimpleResult"
6. Human (from phone) reviews issue, applies `design-approved`
7. ApprovalWatcher (existing) → sprint-planner → sprint-executor
```

### Example 2: Notification channel registry

**After (M2 config in `~/.ailang/config.yaml`):**
```yaml
notifications:
  channels:
    - transport: email
      to: mark@aitanalabs.com
      filter:
        event_types: [approval_pending, task_failed]
    - transport: claude-code        # existing SessionStart behavior, now a registered channel
      filter:
        event_types: [message, approval_pending]
    - transport: github
      filter:
        event_types: [approval_pending]   # comment on the linked issue
    - transport: dashboard
      filter: {}                     # everything (real-time feed)
```

### Example 3: Ambiguous message (hold, not auto-drafted)

```
Inbound: "the thing is slow sometimes" (category: general, envelope confidence 0.38)
→ Promotion Classifier: hold (below threshold, category not in allow-list)
→ surfaced in dashboard triage queue + left unread in `user` inbox
→ no design doc auto-created; human decides
```

## Success Criteria

- [ ] A synthetic clear bug/feature message reaches `pending_approval` with a `planned/` doc and linked GitHub issue, with **zero** manual steps
- [ ] Ambiguous/`general` messages are held (not auto-drafted) and surfaced for human pick
- [ ] `design-approved` on the auto-drafted issue triggers the existing `sprint-planner` → `sprint-executor` chain unchanged
- [ ] One config block (`notifications.channels`) governs all channel routing
- [ ] An `approval_pending` event reaches a registered push channel within 2 minutes (mocked transport in tests; real channel via the companion framework)
- [ ] Notification send failures land in `ailang-dead-letter`, never silently dropped
- [ ] All tests passing (≥ 85% coverage on new dispatcher + `triage_router`)
- [ ] Documentation updated (CHANGELOG, agent-messaging guide; channel how-to lives in the companion doc)

## Testing Strategy

**Unit tests:**
- Promotion Classifier: promote/hold/drop for each category × confidence band; dedupe (`dup_of`) suppresses re-promotion
- Notification Dispatcher: event matches only channels whose filter passes; non-matching skipped; matched → `registry.Get(name).Send` called (mock the registry/channels)

**Integration tests:**
- End-to-end M1: seed unread message → router tick → assert message lands in `design-doc-creator` inbox with correct category (mock the agent runner)
- End-to-end M2: publish synthetic `approval_pending` event → assert the matched channel names get `Send` called (mocked transports)
- Dead-letter: force a `Send` error → assert event lands on `ailang-dead-letter`

**Manual testing:**
- Real inbound message on the Mac Studio coordinator → observe auto-draft + a real push notification (via the companion framework's channel)
- Verify off-session reach: coordinator running, no Claude Code session open → push notification still arrives (Studio always-on)

## Deferred Decisions

- Promotion confidence threshold tuning (start conservative, e.g., 0.75) — **agent may choose** initial value; revisit after first-month precision data
- Whether `hold` items also generate a daily digest notification — **human at review**
- Dispatcher retry/backoff policy before dead-lettering — **agent may choose**
- Internal naming of the dispatcher types — **agent may choose**
- (Channel rendering, transport choice, and email styling are deferred in the companion doc)

## Non-Goals

- **Full-auto to PR** (drafting + planning + execution without a human gate) — explicitly out; the human gate at `design-approved` is the chosen trust boundary. [Revisit once triage precision is proven.]
- **Channel transports + channel choice** — out of *this* doc; owned by the companion [m-notify-channels-framework.md](m-notify-channels-framework.md). This doc stops at the `registry.Get(name).Send` seam.
- **Two-way approval over a channel** (reply-to-approve) — out; approval stays via GitHub label / dashboard. [Companion doc's inbound phase.]
- **Replacing the `dev-cycle` agent** — it remains the manual/interactive path; M1 is the autonomous path. Both coexist.
- **New language syntax or `Msg` effect changes** — this is orchestration over the existing transport.

## Timeline

**Week 1** (M1, ~12h): triage config + Triage Router + Promotion Classifier + forward wiring + tests
**Week 2** (M1 finish + M2 start, ~12h): headless design-doc-creator integration + hold path; channel registry config + dispatcher core
**Week 3** (M2 finish, ~8h): re-express existing surfaces through dispatcher; dead-letter wiring + tests
**Week 4** (M3, ~8h): packages intake source + cloud worker routing + extension docs

**Total: ~40 hours across 4 weeks** (each milestone shippable independently; channel transports tracked separately in the companion ~15h doc)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Triage over-promotes → floods `planned/` with low-value docs | Med | Conservative threshold + category allow-list; `--auto-route --dry-run` to validate before enabling; precision metric tracked from approve/reject |
| Mac Studio is off → local dispatcher can't notify | Low | Studio is treated as always-on (confirmed). For belt-and-braces, the optional cloud-side dispatcher (M2 extra) survives a Studio outage |
| Auto-drafted docs are low quality and waste approval cycles | Med | Human gate at design doc catches this; `needs-revision` label already feeds the iteration loop (max 3) in ApprovalWatcher |
| Notification spam (too many events fan out) | Low | Per-channel `event_types` filter; default push filter is just `approval_pending` + `task_failed` |
| Duplicate promotion of the same report | Med | Reuse existing SimHash `dup_of` dedupe before promotion; idempotent forward (skip if already in design-doc-creator inbox) |

## Related Documents

**Companion milestone (owns channel delivery):**
- [m-notify-channels-framework.md](m-notify-channels-framework.md) — the outbound channel adapter framework (Discord/Google Chat/email transports) this bus delivers through, ported from Aitana's proven Channels Framework

**Implemented (directly inform this design):**
- [design_docs/implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md](../../implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md) — the cascade topic's "authoritative source + deterministic-bump vs AI-escalation" pattern is the model for triage promotion + IAM-gated channels
- [design_docs/implemented/v0_5_6/m-msg-agent-messaging-improvements.md](../../implemented/v0_5_6/m-msg-agent-messaging-improvements.md) — unified messaging store, inboxes, envelope/dedupe that triage builds on

**Code this connects (verified locations):**
- [internal/pubsub/topics.go](../../../internal/pubsub/topics.go) — 6 topics incl. `events`; `MessageAttributes` (Inbox/Category/Source/Requires)
- [internal/coordinator/approval_watcher.go](../../../internal/coordinator/approval_watcher.go) — `ApprovalEventDesign` → `sprint-planner` handoff (unchanged)
- [internal/messaging/inbox.go](../../../internal/messaging/inbox.go) — inbox_messages (category, simhash, `dup_of`, envelope)
- [cmd/ailang/messages.go](../../../cmd/ailang/messages.go) — `triage --cluster-by`, `forward`, `import-github`
- [.claude/agents/dev-cycle.md](../../../.claude/agents/dev-cycle.md) — the manual 4-stage chain this automates Stage 1 of

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [Agent Messaging Guide](../../../docs/docs/guides/agent-messaging.md)
- [Coordinator Guide](../../../docs/docs/guides/coordinator.md)
- Triage proof-of-concept: commit `d6990e03` ("triage 2 inbound docparse reports into design docs")

## Future Work

- **Two-way approval over a channel** (reply "approve" → apply `design-approved`) — owned by the companion framework's inbound phase
- **Inbound email/messages as new intake sources** (M3 starts this with the packages system)
- **Confidence-aware autonomy ramp**: as triage precision proves out, optionally extend auto-run past the design-doc gate for the highest-confidence, lowest-risk classes

---

**Document created**: 2026-05-28
**Last updated**: 2026-05-28
