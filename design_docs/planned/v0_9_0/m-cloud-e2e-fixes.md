# M-CLOUD-E2E-FIXES: Cloud End-to-End Wiring Fixes

**Status**: Planned
**Target**: v0.9.0
**Priority**: P0 (High) — Cloud coordinator deploys but cannot process messages
**Estimated**: 4 hours (~2h implementation + 1h testing + 1h cleanup)
**Dependencies**: M-CLOUD-E2E (implemented), M-PUBSUB (implemented), M-CLOUD-HEALTH (implemented)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Tooling/infrastructure fix — does not modify AILANG language semantics.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics change |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | +1 | Fixes env var naming to match Terraform contracts |
| A7: Machines First | +1 | Enables autonomous cloud coordinator to actually work |
| A11: Structured Failure | +1 | Removes silent fallbacks (wrong topic prefix, empty content) |
| A12: System Boundary | +1 | Notification → Firestore fetch boundary made explicit |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Enables machine-autonomous operation

## Problem Statement

M-CLOUD-E2E (commit `fe4a3802`) wired the coordinator's cloud mode path but left 3 critical gaps that prevent end-to-end flow from working on Cloud Run.

**Blocker 1: Env var name mismatch — coordinator subscribes to wrong topics**

Terraform sets `AILANG_TOPIC_PREFIX=ailang-dev` (cloud_run.tf:63, cloud_run_jobs.tf:44). Code reads `AILANG_PUBSUB_PREFIX` (daemon_tasks_init.go:291, coordinator_cloud.go:78). Since `AILANG_PUBSUB_PREFIX` is never set, prefix falls back to `"ailang"` (DefaultTopicPrefix). Result: coordinator subscribes to `ailang-messages-coordinator` instead of `ailang-dev-messages-coordinator`. **Zero messages received.**

**Blocker 2: PubSubInboxAdapter returns message ID as content, not actual message**

`pubsub_adapter.go:66` sets `Content: notification.MessageID` — just the UUID string. The Pub/Sub notification is intentionally minimal (only `message_id`); full content lives in Firestore. But the adapter has no access to a `MessageStore` to fetch it. Result: tasks are created with `Content: "29404032-74b3-..."` instead of the actual directive. **Agents execute with empty instructions.**

**Blocker 3: Dead code confusion — store_cloud.go is a 262-line unused stub**

`internal/coordinator/store_cloud.go` has 30+ methods all returning `"not yet implemented"`. The real Firestore coordinator store is `internal/storage/firestore/coordinator.go`. The stub is never instantiated but causes confusion during code exploration.

**Current State:**
- Coordinator starts on Cloud Run ✓
- Health endpoint works ✓
- ApprovalWatcher runs ✓
- Pub/Sub client initialized (but wrong prefix) ✗
- Messages received: 0 (wrong subscription name) ✗
- Task content: empty (no Firestore fetch) ✗

## Goals

**Primary Goal:** Fix the 3 blockers so end-to-end cloud flow works: publish to `ailang-dev-messages` → coordinator pulls → classifies → fetches from Firestore → dispatches via Pub/Sub.

**Success Metrics:**
1. Coordinator subscribes to `ailang-dev-messages-coordinator` (not `ailang-messages-coordinator`)
2. Task content is the actual message body from Firestore (not just the message ID)
3. `store_cloud.go` deleted — no more dead code confusion
4. All existing tests pass (`make test`)
5. No regression in local mode

## Solution Design

### Fix 1: Env var fallback chain (~5 min)

Read `AILANG_TOPIC_PREFIX` first (matches Terraform), fall back to `AILANG_PUBSUB_PREFIX` (matches CLAUDE.md docs), then default.

**Files:**
- `internal/coordinator/daemon_tasks_init.go:291` — `initPubSub()`
- `cmd/ailang/coordinator_cloud.go:78` — `executeJob()`

```go
// Before (broken):
prefix := os.Getenv("AILANG_PUBSUB_PREFIX")

// After (works with Terraform AND docs):
prefix := os.Getenv("AILANG_TOPIC_PREFIX")
if prefix == "" {
    prefix = os.Getenv("AILANG_PUBSUB_PREFIX")
}
```

Also update `CLAUDE.md` to document both env var names.

### Fix 2: PubSubInboxAdapter fetches full message from Firestore (~1h)

Inject `messaging.MessageStore` into the adapter. When a notification arrives, fetch the full `InboxMessage` from Firestore using `GetInboxMessage(messageID)`.

**File:** `internal/coordinator/pubsub_adapter.go`

```go
// Before:
type PubSubInboxAdapter struct {
    subscriber *pubsub.Subscriber
    subName    string
    inbox      string
    logger     *log.Logger
    // ...
}

// After:
type PubSubInboxAdapter struct {
    subscriber *pubsub.Subscriber
    msgStore   messaging.MessageStore  // NEW: for fetching full message content
    subName    string
    inbox      string
    logger     *log.Logger
    // ...
}
```

In the subscription callback:
```go
// After decoding notification:
fullMsg, err := a.msgStore.GetInboxMessage(notification.MessageID)
if err != nil {
    a.logger.Printf("PubSubInboxAdapter: failed to fetch message %s from store: %v",
        notification.MessageID, err)
    // Fall back to notification-only data (don't nack — message exists in Firestore
    // but may not be queryable yet due to eventual consistency)
} else {
    msg.Title = fullMsg.Title
    msg.Content = fullMsg.Content
    msg.From = fullMsg.From
    msg.Type = fullMsg.Type
}
```

**File:** `internal/coordinator/daemon_tasks_init.go` — pass `d.msgStore` to adapter constructor:
```go
d.cloudInboxAdapter = NewPubSubInboxAdapter(subscriber, subName, "coordinator", d.msgStore, d.logger)
```

### Fix 3: Delete store_cloud.go (~1 min)

```bash
git rm internal/coordinator/store_cloud.go
```

262 lines of dead code removed. No references in codebase (verified: `CloudStore` is never instantiated outside the file itself).

### Implementation Plan

**Phase 1: Env var fix + dead code cleanup** (~30 min)
- [ ] Update `daemon_tasks_init.go:291` to read `AILANG_TOPIC_PREFIX` first
- [ ] Update `coordinator_cloud.go:78` to read `AILANG_TOPIC_PREFIX` first
- [ ] Update `CLAUDE.md` to document `AILANG_TOPIC_PREFIX` alongside `AILANG_PUBSUB_PREFIX`
- [ ] Delete `internal/coordinator/store_cloud.go`
- [ ] Run `make test` and `make lint`

**Phase 2: Adapter Firestore fetch** (~1.5h)
- [ ] Add `msgStore messaging.MessageStore` field to `PubSubInboxAdapter`
- [ ] Update `NewPubSubInboxAdapter()` signature to accept `MessageStore`
- [ ] Fetch full message in subscription callback via `GetInboxMessage()`
- [ ] Graceful fallback if Firestore fetch fails (use notification-only data)
- [ ] Update `daemon_tasks_init.go` to pass `d.msgStore` to adapter
- [ ] Run `make test` and `make lint`

**Phase 3: Verification** (~1h)
- [ ] Verify subscription names match Terraform topic names
- [ ] Verify `GetInboxMessage` works on Firestore backend
- [ ] Run full test suite
- [ ] Update CHANGELOG.md

### Files to Modify

**Modified files:**
- `internal/coordinator/daemon_tasks_init.go` — Env var fix + pass msgStore (~5 LOC)
- `cmd/ailang/coordinator_cloud.go` — Env var fix (~3 LOC)
- `internal/coordinator/pubsub_adapter.go` — Add msgStore, fetch full message (~20 LOC)
- `CLAUDE.md` — Document AILANG_TOPIC_PREFIX env var (~2 LOC)
- `CHANGELOG.md` — Add entry (~3 LOC)

**Deleted files:**
- `internal/coordinator/store_cloud.go` — 262 LOC dead code removed

## Examples

### Example 1: Env Var Fix Effect

**Before (broken):**
```
# Terraform sets: AILANG_TOPIC_PREFIX=ailang-dev
# Code reads: AILANG_PUBSUB_PREFIX (empty)
# Result: prefix = "ailang" (default)
# Subscription: ailang-messages-coordinator  ← WRONG
# Actual topic: ailang-dev-messages          ← MISMATCH
```

**After (fixed):**
```
# Terraform sets: AILANG_TOPIC_PREFIX=ailang-dev
# Code reads: AILANG_TOPIC_PREFIX = "ailang-dev"
# Subscription: ailang-dev-messages-coordinator  ← CORRECT
```

### Example 2: Message Content Fix

**Before (broken):**
```json
// Task created by coordinator:
{
  "id": "task-29404032",
  "content": "29404032-74b3-40c6-acc3-23d6bbe14b68",
  "title": "Pub/Sub notification from user"
}
// Agent prompt: "Execute task task-29404032..."  ← EMPTY DIRECTIVE
```

**After (fixed):**
```json
// Task created by coordinator:
{
  "id": "task-29404032",
  "content": "Fix the null pointer bug in parser.go",
  "title": "Bug: Parser NPE"
}
// Agent prompt: "Fix the null pointer bug in parser.go"  ← REAL DIRECTIVE
```

## Success Criteria

- [ ] Coordinator subscribes to `{AILANG_TOPIC_PREFIX}-messages-coordinator` (verified by log output)
- [ ] Task content matches Firestore message body (not just UUID)
- [ ] `store_cloud.go` deleted, no compilation errors
- [ ] All existing tests pass (`make test`)
- [ ] No regression in local mode (`COORDINATOR_MODE=local` unchanged)
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests:**
- No new unit tests needed — changes are wiring fixes
- Existing tests verify PubSubInboxAdapter interface compliance

**Integration tests:**
- Deploy to Cloud Run staging → publish test message → verify coordinator log shows correct subscription name and message content

**Manual verification:**
- `grep -r "AILANG_TOPIC_PREFIX\|AILANG_PUBSUB_PREFIX" internal/ cmd/` → verify all occurrences use fallback chain
- `grep -r "CloudStore" internal/` → verify zero references after deletion

## Non-Goals

**Not in this feature:**
- Pub/Sub emulator integration tests — deferred to separate PR
- BigQuery observatory backend — deferred to M-CLOUD-ANALYTICS
- Dead-letter queue processing — topic exists, handler deferred
- HOME directory validation on Cloud Run — /tmp is writable, no issue

## Timeline

**Single session** (~4h):
- Phase 1: Env var fix + cleanup (30 min)
- Phase 2: Adapter Firestore fetch (1.5h)
- Phase 3: Verification + docs (1h)
- Buffer (1h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Firestore eventual consistency | Low | Graceful fallback to notification-only data if fetch fails |
| Breaking local mode | High | All changes gated on cloud mode; PubSubInboxAdapter only used when `COORDINATOR_MODE=cloud` |
| Missing env var on older deployments | Low | Fallback chain: AILANG_TOPIC_PREFIX → AILANG_PUBSUB_PREFIX → default |

## Related Documents

**Implemented (direct predecessor):**
- [design_docs/implemented/v0_9_0/m-cloud-e2e.md](design_docs/implemented/v0_9_0/m-cloud-e2e.md) — Parent design doc
- [design_docs/planned/v0_9_0/m-pubsub-messaging.md](design_docs/planned/v0_9_0/m-pubsub-messaging.md) — Pub/Sub architecture

**Planned (related):**
- [design_docs/planned/v0_9_0/m-cloud-health.md](design_docs/planned/v0_9_0/m-cloud-health.md) — Health endpoints

## References

- [Design Axioms](/docs/references/axioms)
- Terraform env vars: `ailang-multivac/terraform/cloud_run.tf:63`, `cloud_run_jobs.tf:44`
- Source message: `ailang messages read afc633f8-8a60-491c-afb1-6665d0d37c51`

## Future Work

- Pub/Sub emulator integration tests in CI
- Firestore message pre-fetch (batch fetch on adapter start for lower latency)
- Health endpoint reporting subscription name and prefix for debugging

---

**Document created**: 2026-03-06
**Last updated**: 2026-03-06
