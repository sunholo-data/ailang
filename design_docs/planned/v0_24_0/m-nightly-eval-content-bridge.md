# Nightly-Eval → Cloud Coordinator Content Bridge

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium-High — silently corrupts every cross-store task; wastes agent compute nightly)
**Estimated**: 2 days
**Dependencies**: None (touches `internal/pubsub/`, `internal/coordinator/`, `internal/storage/firestore/`)

> **Origin:** Self-discovered while executing coordinator task `task-49237121`. The task's
> entire directive was the literal string `msg_20260605_035539_49237121` — a dangling message
> pointer, not the intended benchmark-failure report. Forensic analysis (see Problem Statement)
> showed the same corruption hit all 7 tasks in the 2026-06-05 nightly batch, and recurs every
> night. The benchmark-specific content for this task is unrecoverable; this doc fixes the
> pipeline so it never happens again. This is a **systemic infrastructure fix**, chosen over a
> fabricated benchmark-specific doc per the design-doc-creator verification gate.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is an infrastructure (coordinator/messaging) change, not a language feature. Axioms still
apply to the system's behavior — particularly A11 (Structured Failure) and A2 (Replayability).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language evaluation; message routing already deterministic |
| A2: Replayability | +1 | Embedding content in the notification makes a task self-describing and re-dispatchable from the Pub/Sub record alone |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | 0 | No capability/authority change |
| A5: Bounded Verification | 0 | No type-checking change |
| A6: Safe Concurrency | 0 | Ordering key on the messages topic is unchanged |
| A7: Machines First | +1 | An AI agent currently receives an undecodable pointer as its task; this restores a machine-actionable directive |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Stops burning ~20 min of agent compute per corrupted nightly task |
| A10: Composability | 0 | Producer/consumer contract becomes cleaner but composition is unchanged |
| A11: Structured Failure | +1 | Replaces a silent `Content = MessageID` fallback with a loud, quarantined failure |
| A12: System Boundary | +1 | Makes the rig→cloud store boundary an explicit, validated crossing instead of an implicit assumption |

**Net Score: +5** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strictly improves machine-actionability of the dispatched task

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

Net +5 → proceed.

## Problem Statement

The nightly eval rig publishes coordinator tasks via Pub/Sub. When a task is created from a
"Pub/Sub notification from nightly-eval", it arrives at the cloud coordinator with its
`Content` field set to a **message ID string** (e.g. `msg_20260605_035539_49237121`) rather
than the intended task body (a benchmark-failure report). The AI agent then receives a task
whose entire instruction is an opaque pointer it cannot dereference.

**Current State (verified in code, 2026-06-05):**

The failure is the intersection of three real defects:

1. **Content lives only in the producer's store.** The Pub/Sub notification payload is
   deliberately ID-only:
   ```go
   // internal/pubsub/topics.go:33-37
   // MessageNotification is published to the messages topic.
   // Intentionally minimal — full message content lives in Firestore.
   type MessageNotification struct {
       MessageID string `json:"message_id"`
   }
   ```
   ```go
   // internal/pubsub/publisher.go:42-46
   // The actual message content is already stored in Firestore — this notification
   // tells the coordinator ... that a new message is available.
   payload, err := json.Marshal(MessageNotification{MessageID: messageID})
   ```
   The publisher's contract **assumes the content is already in Firestore**. But the nightly
   rig is a bare-metal worker using a **local SQLite** message store
   (`internal/messaging/inbox.go`); it never writes to the cloud Firestore the coordinator
   reads. The body never crosses the rig→cloud boundary.

2. **The consumer's resolve attempt always misses.** The adapter does try to hydrate content:
   ```go
   // internal/coordinator/pubsub_adapter.go:187-202
   // Fetch full message content from Firestore.
   if a.msgStore != nil {
       fullMsg, fetchErr := a.msgStore.GetInboxMessage(notification.MessageID)
       if fetchErr != nil { /* log, fall through */ }
       else if fullMsg != nil { msg.Content = fullMsg.Payload ... }
   }
   ```
   But `GetInboxMessage` on the Firestore backend looks up by **exact document ID**:
   ```go
   // internal/storage/firestore/messaging_inbox.go:110-118
   func (s *MessagingStore) GetInboxMessage(id string) (*messaging.InboxMessage, error) {
       doc, err := s.client.Doc(collInbox, id).Get(context.Background())
       ...
   }
   ```
   and Firestore doc IDs use the `inbox_<unixMilli>_<short>` format (write path at
   `messaging_inbox.go:24`, with `MessageID == doc.ID` enforced at lines 34-36). The
   notification carries a **SQLite-format** ID, `msg_<timestamp>_<id8>`. These key namespaces
   can never collide → `NotFound` → no hydration.

3. **The miss is silently papered over.** When hydration fails (or `msgStore == nil`), the
   message keeps its construction-time fallback:
   ```go
   // internal/coordinator/pubsub_adapter.go:159
   Content: notification.MessageID, // Fallback: just the ID
   ```
   So a structurally-broken task is dispatched to an AI agent as if it were valid. This
   directly violates the project's **NO SILENT FALLBACKS** principle (CLAUDE.md §2): the
   fallback value (`Content = an ID`) feeds business logic (what the agent builds).

A contributing operational factor: the rig's `nightly-eval.sh` send path swallows failures
with `|| true`, so even a clean send error never surfaces.

**Impact:**

- **Who:** Every coordinator task created from a nightly-eval Pub/Sub notification on the
  cloud (`AILANG_STORAGE=gcp`) coordinator. Confirmed across **all 7 tasks** in the
  2026-06-05 batch.
- **Severity:** The agent receives an undecodable directive. In the best case it spends ~20
  minutes of compute diagnosing the pipeline (as `task-49237121` did) and produces no feature
  work; in the worst case it **fabricates** a plausible-but-wrong design doc by guessing the
  benchmark, polluting `design_docs/planned/`. The original benchmark report is permanently
  lost — it lived only in the rig's local SQLite.
- **Cadence:** Recurs every night the rig runs. Pure waste, compounding.

## Goals

**Primary Goal:** A coordinator task created from a nightly-eval notification arrives with its
full, intended body — and if it cannot, the task fails loudly and is quarantined instead of
dispatching an undecodable pointer to an AI agent.

**Success Metrics:**
- 0% of nightly-eval-originated tasks dispatched with `Content == MessageID`.
- A task whose content cannot be resolved is routed to a dead-letter/quarantine state with a
  human-readable reason, **not** executed.
- A replay of the same Pub/Sub message reproduces the same task body without any dependency on
  the producer's local store (self-describing notification).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Embed task content in the notification vs. mirror to a shared store | Determines whether the rig→cloud boundary stays store-coupled; changes the Pub/Sub wire format | human | design | med |
| Pub/Sub payload size budget (inline body vs. preview+pointer) | Pub/Sub caps at 10MB; large eval reports could blow a tiny budget | agent | design | low |
| Behavior when content is unresolved: quarantine vs. nack-retry vs. dead-letter inbox | Defines the loud-failure semantics; affects coordinator state machine | human | design | med |
| Whether to also fix the SQLite↔Firestore key-namespace mismatch in `GetInboxMessage` | A latent bug even for in-cloud sends; systemic vs. localized | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Wire-format decision**: embed full content in `MessageNotification`, or
      embed content for cross-host (rig) sends only while keeping ID-only for in-cloud sends.
      *(Recommendation: always embed `Title` + `Content` + a `ContentTruncated bool`; keep the
      ID for back-reference. See Solution Design.)*
- [ ] **Unresolved-content behavior**: confirm quarantine-and-alert (recommended) vs. retry.

## Solution Design

### Overview

Make the notification **self-describing** (carry the task body), and make the coordinator
**fail loudly** when a task still has no resolvable body. Three coordinated changes, mirroring
the three defects, so no single point silently loses content:

1. **Producer carries content** — extend `MessageNotification` with the actual `Title` and
   `Content` (plus a truncation flag), so a SQLite-backed rig no longer depends on the cloud
   coordinator being able to read the rig's local store.
2. **Consumer trusts embedded content, then falls back to fetch** — the adapter uses the
   embedded body first; the Firestore fetch becomes a secondary path, not the only one.
3. **No silent pointer dispatch** — if, after all resolution, `Content` is still just the
   `MessageID` (or empty), the adapter marks the task `quarantined` with a clear reason and
   does **not** buffer it for agent execution. Loud log + alert.

Additionally (systemic hardening): fix `GetInboxMessage` so the Firestore backend can resolve
by the `message_id` field, not only by exact doc ID — closing the `msg_…` vs `inbox_…` key
mismatch for any future in-cloud sender that uses the SQLite ID format.

### Architecture

**Components:**
1. **`pubsub.MessageNotification`** (`internal/pubsub/topics.go`): gains `Title string`,
   `Content string`, `ContentTruncated bool`. Comment updated to reflect that content is now
   carried, with Firestore as an optional fallback for oversized bodies.
2. **`pubsub.Publisher.PublishMessage`** (`internal/pubsub/publisher.go`): accepts the message
   body and populates the new fields; truncates to the payload budget and sets
   `ContentTruncated` when it must fall back to "fetch the rest from the store".
3. **`coordinator.PubSubInboxAdapter.HandleNotification`**
   (`internal/coordinator/pubsub_adapter.go`): prefers `notification.Content`; only attempts
   `GetInboxMessage` when content is empty or `ContentTruncated`; quarantines instead of
   dispatching when content remains a bare pointer.
4. **`firestore.MessagingStore.GetInboxMessage`**
   (`internal/storage/firestore/messaging_inbox.go`): when a direct doc-ID `Get` misses, fall
   back to a `Where("message_id", "==", id)` query (reusing the existing `FindMessageByPrefix`
   query pattern) so SQLite-format IDs resolve.

### Implementation Plan

**Phase 1: Self-describing notification (~3 hours)**
- [ ] Add `Title`, `Content`, `ContentTruncated` to `MessageNotification`
      (`internal/pubsub/topics.go`); update the doc comment.
- [ ] Update `PublishMessage` signature/body to populate them; define a payload budget
      constant (e.g. `maxInlineContentBytes = 256 * 1024`, well under Pub/Sub's 10MB) and set
      `ContentTruncated` when exceeded.
- [ ] Update all `PublishMessage` call sites to pass the body (search:
      `grep -rn PublishMessage internal/`).

**Phase 2: Consumer prefers embedded content + loud failure (~4 hours)**
- [ ] In `HandleNotification`, set `msg.Content = notification.Content` /
      `msg.Title = notification.Title` when present.
- [ ] Gate the Firestore fetch on `notification.Content == "" || notification.ContentTruncated`.
- [ ] After resolution, if `msg.Content == "" || msg.Content == notification.MessageID`,
      mark the task quarantined (new status or routed to a dead-letter inbox) with a
      human-readable reason; log at error level; do **not** append to `a.buffered`.
- [ ] Decide ack/nack: ack the Pub/Sub message (avoid redelivery storms) but persist the
      quarantine so it is visible to `ailang coordinator pending`/logs.

**Phase 3: Key-mismatch hardening + rig send (~3 hours)**
- [ ] `GetInboxMessage` Firestore fallback: on `NotFound`, query
      `Where("message_id", "==", id)`; return the single match or a clear not-found error.
- [ ] Remove the `|| true` swallow in `nightly-eval.sh` (or replace with explicit
      log-and-continue that records the send error).
- [ ] Add a coordinator-side log line distinguishing "content from notification" vs.
      "content from store fetch" vs. "quarantined: unresolved".

### Files to Modify/Create

**Modified files:**
- `internal/pubsub/topics.go` — extend `MessageNotification` (~10 LOC)
- `internal/pubsub/publisher.go` — populate + truncate content (~25 LOC)
- `internal/coordinator/pubsub_adapter.go` — prefer embedded content, quarantine on
  unresolved (~40 LOC)
- `internal/storage/firestore/messaging_inbox.go` — `GetInboxMessage` field-query fallback
  (~20 LOC)
- `scripts/nightly-eval.sh` (or equivalent rig send script) — stop swallowing send errors
  (~5 LOC)

**New files:**
- None required. (Quarantine state may reuse the existing coordinator task status enum; if a
  dedicated dead-letter inbox is chosen, that is a config entry, not new code.)

## Examples

### Example 1: Nightly-eval task dispatch

**Before:**
```text
# Coordinator task created from Pub/Sub notification
Title:   "Pub/Sub notification from nightly-eval"
Content: "msg_20260605_035539_49237121"   # <-- a dangling pointer

# Agent receives this as its entire instruction → cannot proceed,
# wastes ~20 min, or fabricates a wrong design doc.
```

**After:**
```text
# Notification now carries the body
Title:   "Benchmark failure: <benchmark-name> on <model>"
Content: "<full failure report: prompt, expected, actual, error trace>"

# Agent receives an actionable directive and produces the intended work.
```

### Example 2: Content genuinely unresolvable (loud failure)

**Before:**
```text
GetInboxMessage("msg_…") → NotFound → Content stays "msg_…" → dispatched anyway (silent).
```

**After:**
```text
GetInboxMessage("msg_…") → NotFound, and notification carried no content
  → task marked quarantined: "content unresolved: notification had no body and store
     lookup msg_20260605_035539_49237121 missed (SQLite-format id, Firestore backend)"
  → NOT dispatched; visible in `ailang coordinator pending`; error logged + alert.
```

## Success Criteria

- [ ] A nightly-eval notification published with a body arrives at the coordinator with that
      body intact (integration test: publish → `HandleNotification` → assert `msg.Content`).
- [ ] A notification with no body and an unresolvable ID is quarantined, **not** buffered for
      execution (unit test on `HandleNotification`).
- [ ] `GetInboxMessage` on the Firestore backend resolves a `msg_…`-format ID via the
      `message_id` field fallback (unit test).
- [ ] `nightly-eval.sh` surfaces send failures instead of swallowing them.
- [ ] All tests passing (`make test`).
- [ ] Documentation updated (agent-messaging + coordinator guides note the self-describing
      notification contract).
- [ ] CHANGELOG.md entry under v0.24.0.

## Testing Strategy

**Unit tests:**
- `HandleNotification`: (a) embedded content used verbatim; (b) empty content +
  `ContentTruncated` triggers store fetch; (c) unresolved content → quarantine, not buffered.
- `GetInboxMessage` (Firestore, against the emulator or a fake): doc-ID hit; `message_id`
  field-query fallback hit; genuine miss returns a clear error.
- `PublishMessage`: content within budget embedded fully; oversized content truncated with
  `ContentTruncated = true`.

**Integration tests:**
- End-to-end: publish a `MessageNotification` with a realistic eval-failure body through the
  pubsub test harness; assert the coordinator builds a `Message` with the full body and never
  with `Content == MessageID`.

**Manual testing:**
- Trigger one nightly-eval send against a staging coordinator; confirm the resulting task's
  content in `ailang coordinator logs <task-id>` is the failure report, not an ID.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Quarantine surfacing** — reuse an existing task status vs. a dedicated dead-letter inbox.
  *(agent may choose; both satisfy "loud + visible".)*
- **Exact inline-content byte budget** — `256KB` is a safe default; agent may tune against
  observed eval-report sizes.
- **Whether to backfill content for already-quarantined 2026-06-05 tasks** — likely
  unrecoverable (rig-local only); agent may decide whether to attempt rig-side recovery or
  mark them permanently lost.

## Non-Goals

- **Recovering the lost 2026-06-05 benchmark reports.** They were written only to the rig's
  local SQLite and the Pub/Sub notifications carried only IDs; they are not reconstructable
  from the cloud side. Out of scope.
- **Redesigning the rig's message store** (SQLite → Firestore). The self-describing
  notification removes the need for store unification; a full migration is a separate effort.
- **General Pub/Sub large-payload streaming** (chunking >10MB). The truncate + fetch-rest path
  covers the realistic eval-report size range.

## Timeline

**Day 1** (~7 hours):
- Phase 1 (self-describing notification) + Phase 2 (consumer prefer + loud failure)

**Day 2** (~5 hours):
- Phase 3 (key-mismatch hardening, rig send fix)
- Tests, docs, CHANGELOG, manual staging verification

**Total: ~12 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Embedding content enlarges Pub/Sub payloads | Low | 256KB budget is far under the 10MB cap; truncate + fetch-rest for outliers |
| Quarantining breaks a legitimate ID-only flow elsewhere | Med | Audit all `PublishMessage` callers; only quarantine when content is empty AND store fetch misses — never on a successful fetch |
| `message_id` field query needs a Firestore composite index | Low | Single-field equality needs no composite index; mirrors existing `FindMessageByPrefix` range query |
| Changing the wire format breaks in-flight messages during deploy | Med | New fields are additive/optional; old ID-only notifications still resolve via the (now-fixed) store fetch path |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_23_0/m-coord-tag-routing-lastmile.md` — coordinator Pub/Sub
  tag-routing context (same adapter surface)

**Planned (check for overlap):**
- `design_docs/planned/v0_24_0/m-msg-auto-triage-pipeline.md` — message triage routing; the
  quarantine path here should be compatible with its router
- `design_docs/planned/v0_24_0/m-eval-local-ollama.md` — defines the local/bare-metal eval rig
  that is the producer in this bug

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- CLAUDE.md §2 — **NO SILENT FALLBACKS - FAIL LOUDLY** (the principle this fix restores)
- `internal/coordinator/pubsub_adapter.go:159,187-202` — the fallback + the missing fetch
- `internal/storage/firestore/messaging_inbox.go:24-36,110-119` — Firestore key format + lookup
- `internal/pubsub/topics.go:33-37`, `internal/pubsub/publisher.go:42-46` — ID-only notification
- `.claude/rules/coordinator.md` — multi-host worker (rig) routing context

## Future Work

- Unify the bare-metal rig and cloud message stores behind one interface so "producer store ≠
  consumer store" can never reoccur for any message category (not just nightly-eval).
- Add a coordinator pre-dispatch validator that rejects any task whose `Content` fails a
  minimal sanity check (non-empty, not equal to its own ID, parseable directive) — a general
  guard beyond this specific pipeline.

---

**Document created**: 2026-06-05
**Last updated**: 2026-06-05
