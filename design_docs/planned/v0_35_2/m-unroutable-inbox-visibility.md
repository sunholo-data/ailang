# M-UNROUTABLE-INBOX-VISIBILITY: Unroutable sends must be surfaced, not stored silently

**Status**: Planned
**Target**: v0.35.2
**Priority**: P0
**Estimated**: 1 day
**Dependencies**: None (builds on `BackstopSweep`, `AgentRegistry.IsUndeclaredUnrouted`, `triage_only_inboxes` — all already landed)

## Problem Statement

A message addressed to an inbox that no agent serves is accepted by the store, marked
unread, and never dispatched — with **zero signal to the sender or to any operator**.

**Measured 2026-09-07 (prod, project `ailang-multivac`):** five well-formed feedback
reports were sent to `pkg:sunholo/email`. No agent in `config.cloud.yaml` served that
inbox, and no wildcard pattern (e.g. `pkg:sunholo/motoko_ext_*`) matched it. All five
were accepted, stored, and marked unread. They produced **zero tasks, no bounce, no
warning, no error** and sat for ~6 hours until found by a human auditing the store by
hand.

**Mechanism:** cloud dispatch routes on the message's `Inbox` attribute via
`AgentRegistry.GetAgentForInbox` (exact match, then longest-prefix wildcard;
`internal/coordinator/agent_registry.go:376`). A nil result is a refusal
(`resolveInboxAgent`, `internal/coordinator/daemon_tasks_polling.go:34`), and the
message is left unread "for triage". The cloud path logs one line
(`daemon_tasks_polling.go:389`) and the local path logs a `CONFIG GAP` line — but a
log line on the coordinator host is not a report: nobody's job is reading that log,
and the sender sees a successful send either way.

**Impact:** this plane is meant to run unsupervised. A send that silently goes
nowhere is the worst failure shape available — indistinguishable from success at the
sender, discoverable only by auditing the store. User feedback (the exact input this
program exists to consume) is the measured casualty.

## Current State (what already exists — do not rebuild)

- **`GetAgentForInbox` / `HasInbox`** — exact + longest-prefix wildcard matching.
  Wildcards mean *"no exact match" ≠ "unroutable"*.
- **`triage_only_inboxes`** (`agent_config.go:37`, `SetTriageOnlyInboxes`) — an inbox
  can be *declared* human-triaged. Three states exist by design: **served**,
  **declared triage-only**, **undeclared unrouted** (a config gap). Only the third
  is a defect.
- **`IsUndeclaredUnrouted`** (`agent_registry.go:729`) — already computes exactly the
  predicate this fix needs: no agent (exact or wildcard) AND not declared
  triage-only.
- **`BackstopSweep`** (`internal/coordinator/backstop_sweep.go`) — a periodic pass
  (startup + ticker) over all unread messages that already finds "routable but not
  delivered". It today **skips** undeclared-unrouted inboxes (`GetAgentForInbox ==
  nil → continue`). The gap is a skipped branch in an existing sweep, not a missing
  subsystem.

## Options Considered

### (a) Validate the target inbox at send time against the registry

Reject the send with an error if `HasInbox(to) && !IsTriageOnly(to)` is false.

- **Pro:** fails loudly at the sender; message never enters limbo.
- **Con:** the sender does not have the registry. Sends go straight to the Firestore
  store from arbitrary machines; the routing table lives in `config.cloud.yaml` on
  the coordinator. A send-side check needs a cached/synced copy of the routing table,
  which is **stale by construction** — a newly added inbox (or a new wildcard
  pattern, or a new package in a wildcard family) would false-bounce legitimate
  mail, inverting the defect from silent-accept to loud-*wrong*-reject. It also
  changes send semantics for every caller of the store, including automated agents.
- **Verdict:** rejected. Wrong-place validation; staleness makes it a new defect.

### (b) Periodic unroutable-message report (extend the backstop sweep)

Make the existing `BackstopSweep` report the third state it currently skips: group
unread, non-outcome-notice messages on undeclared-unrouted inboxes, and emit a
single digest — log line plus a message posted to a human-served inbox
(e.g. `controlplane`) — naming the inbox, the count, and the two remedies
(register an agent, or declare it in `triage_only_inboxes`).

- **Pro:** small; reuses the sweep, the predicate, and the three-state model that
  already exist. Cannot false-positive on wildcards (uses `GetAgentForInbox`).
  Detects the 2026-09-07 incident within one sweep interval regardless of which
  machine sent the mail. Does not change send or dispatch semantics at all.
- **Con:** detection latency = sweep interval, not instant. Acceptable: the defect
  class is config drift, and the measured discovery time went from ~6 h (human
  audit) to one interval (minutes).
- **Verdict:** **recommended.**

### (c) Dead-letter inbox

Rewrite unroutable messages' `ToInbox` to a `dead-letter` inbox served by a human
triage flow.

- **Pro:** one place to look.
- **Con:** moves the message out of its original inbox, destroying the "left unread
  for triage" invariant that `resolveInboxAgent` documents as load-bearing — a
  later config fix (registering the agent) would no longer find the mail where it
  looks. Requires new query tooling and a new convention for re-delivery. Larger
  than the defect.
- **Verdict:** rejected. (b) reports in place; nothing moves.

## Proposed Design (option b)

### Sweep change

In `BackstopSweep.SweepOnce` (`internal/coordinator/backstop_sweep.go`), partition
the currently-skipped messages further:

1. `IsTriageOnly(m.ToInbox)` → skip (declared, by design — unchanged).
2. `GetAgentForInbox(m.ToInbox) != nil` → existing recoverable path (unchanged).
3. **New:** otherwise (`IsUndeclaredUnrouted(m.ToInbox)`), after the outcome-notice
   filter, group by `ToInbox` into `unrouted map[string][]InboxMessage`.

### Report

If `unrouted` is non-empty, emit **one** digest per sweep pass:

- **Log:** `backstop sweep: %d unread message(s) on %d undeclared-unrouted inbox(es): <inbox>(n), ... — register an agent or declare in triage_only_inboxes`.
- **Message:** post a `feedback`-kind digest to the human-triaged `controlplane`
  inbox titled `Unroutable mail: <inbox> (<n> unread)` (one per unrouted inbox),
  body listing message IDs/titles and the two remedies. The report itself is
  addressed to a **declared triage-only** inbox, so it can never recurse into the
  same report.

**Dedup / non-spam:** the sweep is periodic; the same unread message must not
re-report every interval. Key the report on the set of message IDs: post the digest
message with `Collapsed`/semantic-dedup semantics already used by the store (title +
inbox + ID list), so an unchanged backlog yields no new message. If the store's
dedup cannot express this, gate instead: post only when the ID set for that inbox
changed since the last pass (state kept in memory; a coordinator restart re-posts
once, which is acceptable and self-limiting).

### Cloud skip-path log parity

In `pollAndProcessTasksCloud` (`daemon_tasks_polling.go:389`), mirror the local
path: use `IsUndeclaredUnrouted` to log `CONFIG GAP` vs "human-triage inbox, no
agent by design" distinctly, so the log signal and the sweep report agree.

### Explicitly out of scope

- No send-time validation, no new store API, no dead-letter inbox, no changes to
  dispatch, routing, or `resolveInboxAgent`.
- No bounce-to-sender protocol (senders are not necessarily reachable; the report
  goes to the humans who own the plane).

## Acceptance Criteria

1. **AC1 — report fires:** with an unread message on an inbox that has no agent
   (exact or wildcard) and no `triage_only_inboxes` entry, one `SweepOnce` pass
   produces (a) a log line naming the inbox and count and (b) a digest message in
   `controlplane` naming the inbox, count, and remedies.
2. **AC2 — wildcard is routed:** a message on `pkg:sunholo/motoko_ext_foo` with only
   the pattern `pkg:sunholo/motoko_ext_*` registered produces **no** unrouted
   report (guards the "no exact match ≠ unroutable" rule).
3. **AC3 — declared triage-only is silent:** a message on a `triage_only_inboxes`
   inbox produces no report.
4. **AC4 — no re-report spam:** a second `SweepOnce` pass over an unchanged backlog
   posts no new digest message.
5. **AC5 — no recursion:** the digest message itself (on `controlplane`,
   triage-only) is never counted as unrouted.
6. **AC6 — outcome notices excluded:** completion/result notices on an unrouted
   inbox are not counted (mirrors the existing `isOutcomeNotice` guard).
7. **AC7 — cloud log parity:** the cloud skip path logs `CONFIG GAP` for
   undeclared inboxes and the "by design" line for declared ones.
8. **AC8 — regression test for the measured incident:** a test seeds five messages
   to `pkg:sunholo/email` with the prod-like registry (wildcard present, inbox
   absent) and asserts exactly one digest naming `pkg:sunholo/email` and count 5.

## Testing Plan

- Unit tests on `BackstopSweep.SweepOnce` with an in-memory `msgStore` and a
  registry covering: exact agent, wildcard pattern, triage-only, undeclared, and
  outcome-notice messages (AC1–AC6, AC8).
- Unit test on `pollAndProcessTasksCloud` logging or a factored helper (AC7).
- Existing backstop-sweep and registry tests must pass unchanged.

## Related Documents

- `internal/coordinator/backstop_sweep.go` (existing sweep and its header doc)
- `internal/coordinator/agent_registry.go` (`GetAgentForInbox`, `HasInbox`,
  `IsUndeclaredUnrouted`)
- `internal/coordinator/daemon_tasks_polling.go` (`resolveInboxAgent`,
  M-MESSAGE-PLANE-FAIL-LOUD M2 / D2 three-state model)
- `docs/internal/message-plane-topology.md`
