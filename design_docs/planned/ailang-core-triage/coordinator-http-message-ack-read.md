# Coordinator HTTP routes: mark message read/unread, read one message, forward

- **Date**: 2026-10-07
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `design_docs/` for "api/messages", "REST", "ack"; read `design_docs/implemented/v0_9_2/m-rest-message-ingestion.md` (the doc that shipped GET/POST /api/messages — covers listing and ingestion only, does not rule on mutating routes); grepped the backlog `ailang-core-backlog.md` and `ailang-core-triage/` siblings (none cover this); verified the storage layer already supports every operation.
- **Estimate**: n/a (design-doc)

The mechanism checks out as described. `internal/coordinator/daemon_http.go` (`handleMessages`) dispatches only GET→`handleGetMessages` and POST→`handlePostMessage`; `handleGetMessages` caps at `limit` (default 50, caller uses 200), so older messages are indeed unreachable today. But the daemon's `msgStore` is the `messaging.MessageStore` interface (`internal/messaging/message_store.go`), which **already has** `GetInboxMessage(id)`, `FindMessageByPrefix(prefix)`, `MarkInboxMessageRead(id)`, `MarkInboxMessageUnread(id)`, `MarkAllInboxMessagesRead(inbox)` and `ForwardInboxMessage(id, toInbox)` — implemented by both SQLite and Firestore backends, storage-agnostically. The HTTP layer is therefore thin handlers; the real work is the API-shape decisions, which is why this is a design doc and not a patch:

1. **Verb/shape**: PATCH /api/messages/{id} {"status": read|unread} vs POST /{id}/ack + /{id}/unack — the report itself offers both, and a batch form (POST /api/messages/ack {"ids":[...]}) is a third axis. At least three acceptable designs; someone could reasonably disagree.
2. **Idempotency semantics**: the store's `MarkInboxMessageRead` errors on "not found **or already read**" (`internal/messaging/inbox.go`). An HTTP route must decide whether re-acking returns 200, 409, or 404 — this also decides whether the batch form is atomic or per-id results. Silent-success-vs-error here is exactly the failure class the report measured (launchd acking to the wrong store and reporting success), so it deserves an explicit ruling.
3. **Public surface**: this grows the API-key-protected REST contract consumed by the Sunholo platform (row 4 of the rubric on its own).
4. **ID resolution**: decide whether {id} accepts the short prefix via the existing `FindMessageByPrefix` (the CLI does) or exact ids only.

Two inputs worth folding into the doc: Go 1.26's ServeMux supports method+wildcard patterns (`mux.Handle("PATCH /api/messages/{id}", ...)`) so routing is one line per route; and the design doc should note the strategic payoff — HTTP routes hit the coordinator's own configured store, permanently eliminating the CLI's wrong-store ack class the report documented (local SQLite ack under launchd while prod Firestore stayed unread). Forward (`POST /{id}/forward {"to": inbox}`) is cheap once {id} routing exists since `ForwardInboxMessage` is on the interface; recommend including it in the same doc.

Related but not covering: `m-rest-message-ingestion.md` (GET/POST only, implemented), `docs/internal/message-plane-topology.md` (store topology and the env-var traps, no HTTP surface), `m-message-plane-trust.md` (cloud push plane trust, no REST mutations).
