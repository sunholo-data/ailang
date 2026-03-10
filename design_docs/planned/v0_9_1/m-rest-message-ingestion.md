# M-REST-INGESTION: REST API for Client Message Ingestion

**Status**: Planned
**Priority**: High
**Estimated Effort**: 3 hours
**Target Version**: v0.9.1
**Depends On**: M-PUBSUB (v0.9.0, complete)

## Context

The cloud messaging integration guide (docs/docs/guides/cloud-messaging-integration.md) currently instructs external clients to perform a **dual-write**: store the message in Firestore, then publish a notification to Pub/Sub. This requires clients to integrate two GCP SDKs (Firestore + Pub/Sub), handle two failure modes, and understand AILANG's internal notification format.

The coordinator already runs on Cloud Run with HTTP endpoints (`/pubsub/push`, `/health`, `/status`). Adding a `POST /api/messages` endpoint lets clients send messages with a single HTTP call. The coordinator handles Firestore storage and Pub/Sub notification internally.

## Problem Statement

1. **Two SDKs, two failure modes** — clients must use both Firestore and Pub/Sub client libraries
2. **Leaky abstraction** — clients must know AILANG's internal notification format (`{"message_id": "..."}` + attributes)
3. **Error-prone** — if the client writes Firestore but fails Pub/Sub publish, the message is stored but never processed
4. **High barrier to entry** — a simple `curl` POST should be enough to send a message

## Proposed Solution

Add `POST /api/messages` to the coordinator's HTTP server. The endpoint:

1. Accepts a JSON body with message fields
2. Stores the message via `msgStore.InsertInboxMessageWithContext()`
3. Publishes a Pub/Sub notification via `pubsubPublisher.PublishMessage()`
4. Returns the created message ID

### Request Format

```
POST /api/messages
Authorization: Bearer <COORDINATOR_API_KEY>
Content-Type: application/json

{
  "inbox": "design-doc-creator",       // required
  "title": "Feature: Semantic Caching", // required
  "content": "Full description...",      // required
  "from": "my-client",                  // required
  "category": "feature",                // optional, default: "general"
  "message_type": "request",            // optional, default: "request"
  "github_issue": 42,                   // optional
  "github_repo": "owner/repo"           // optional
}
```

### Response Format

**Success (201 Created):**
```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "inbox": "design-doc-creator",
  "status": "unread"
}
```

**Validation Error (400):**
```json
{
  "error": "missing required field: inbox"
}
```

**Auth Error (401):**
No body (existing `requireAPIKey` pattern).

**Store Error (500):**
```json
{
  "error": "failed to store message: <detail>"
}
```

### Architecture

```
Client
  │
  POST /api/messages { inbox, title, content, from, ... }
  │
  ▼
Coordinator HTTP Handler (handlePostMessage)
  │
  ├── 1. Validate required fields (inbox, title, content, from)
  ├── 2. Build InboxMessage struct
  ├── 3. d.msgStore.InsertInboxMessageWithContext() → Firestore/SQLite
  ├── 4. d.pubsubPublisher.PublishMessage() → Pub/Sub notification (if publisher configured)
  ├── 5. Trigger d.pollAndProcessTasks() + d.executeTaskQueue() (cloud mode)
  │
  ▼
  201 Created { message_id, inbox, status }
```

### Key Design Decisions

**Always registered (not cloud-only)**: Unlike `/pubsub/push` which is cloud-mode only, `/api/messages` is useful in both local and cloud modes. Local mode skips the Pub/Sub publish but still stores via msgStore.

**Auth via `requireAPIKey`**: Uses existing middleware. In local mode (no `COORDINATOR_API_KEY` set), requests pass through unauthenticated — matching existing behavior.

**Pub/Sub publish is best-effort**: If the publisher is nil (local mode) or publish fails, the message is still stored. A warning is logged but the request succeeds (201). The message will be picked up by the next poll cycle. This follows the "Firestore first, Pub/Sub second" principle.

**Trigger immediate processing**: In cloud mode, after storing + publishing, the handler calls `pollAndProcessTasks()` and `executeTaskQueue()` synchronously — same pattern as `handlePushMessage`. This means the message starts processing immediately rather than waiting for the next poll tick.

## Implementation Plan

### File: `internal/coordinator/daemon_http.go`

1. **Add handler registration** (in `startHealthServer`):
   ```go
   mux.HandleFunc("/api/messages", d.requireAPIKey(d.handlePostMessage))
   ```

2. **Add request struct**:
   ```go
   type postMessageRequest struct {
       Inbox       string `json:"inbox"`
       Title       string `json:"title"`
       Content     string `json:"content"`
       From        string `json:"from"`
       Category    string `json:"category"`
       MessageType string `json:"message_type"`
       GitHubIssue *int   `json:"github_issue,omitempty"`
       GitHubRepo  string `json:"github_repo,omitempty"`
   }
   ```

3. **Add response struct**:
   ```go
   type postMessageResponse struct {
       MessageID string `json:"message_id"`
       Inbox     string `json:"inbox"`
       Status    string `json:"status"`
   }
   ```

4. **Implement `handlePostMessage`** (~60 lines):
   - Check `r.Method == POST`
   - Decode JSON body into `postMessageRequest`
   - Validate required fields: inbox, title, content, from
   - Default category to "general", message_type to "request"
   - Check `d.msgStore != nil` (return 503 if not)
   - Build `messaging.InboxMessage` from request
   - Call `d.msgStore.InsertInboxMessageWithContext()`
   - If `d.pubsubPublisher != nil`, call `PublishMessage()` (log warning on failure, don't fail request)
   - Trigger `pollAndProcessTasks()` + `executeTaskQueue()` if in cloud mode
   - Return 201 with `postMessageResponse`

### File: `internal/coordinator/daemon_http_test.go`

5. **Add tests**:
   - `TestHandlePostMessage_Success` — valid request, verify stored + response
   - `TestHandlePostMessage_MissingFields` — missing inbox/title/content/from → 400
   - `TestHandlePostMessage_WrongMethod` — GET → 405
   - `TestHandlePostMessage_NoStore` — msgStore is nil → 503
   - `TestHandlePostMessage_DefaultValues` — verify category defaults to "general"

### File: `docs/docs/guides/cloud-messaging-integration.md`

6. **Update integration guide**:
   - Add new section "Option 1: REST API (Recommended)" before the current Firestore+Pub/Sub approach
   - Move current dual-write approach to "Option 2: Direct Firestore + Pub/Sub (Advanced)"
   - Update all client examples (Python, Node.js, Go, curl) to use single REST call
   - Simplify prerequisites (only needs HTTP client, no GCP SDKs)

## Testing Strategy

1. **Unit tests**: Mock `msgStore` and `pubsubPublisher`, verify handler logic
2. **Integration test**: Start coordinator locally, `curl POST /api/messages`, verify message appears in `ailang messages list`
3. **Cloud verification**: Deploy to Cloud Run, send message via curl, verify agent picks it up

```bash
# Local test
ailang coordinator start &
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -d '{"inbox":"user","title":"Test","content":"Hello","from":"curl-test"}'

# Verify
ailang messages list --unread
```

## Metrics & Success Criteria

- Single HTTP call to send a message (no Firestore/Pub/Sub SDK required)
- Message appears in `ailang messages list` within 1 second
- Agent picks up and processes message within normal coordinator poll cycle
- All existing Pub/Sub push flow continues to work (no regression)

## Related Work

- `handlePushMessage` in `daemon_http.go` — same pattern (receive, store, trigger)
- `PubSubNotifier.Notify()` in `messaging/pubsub_notifier.go` — dual-write pattern to reuse
- `InsertInboxMessageWithContext()` in `messaging/inbox.go` — auto-generates ID, status, simhash
