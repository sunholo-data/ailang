# M-DASHBOARD-PUBSUB-EVENTS: Dashboard Pub/Sub Event Subscriber

**Status**: Planned
**Version**: v0.9.1
**Priority**: High — completes cloud mode event streaming story

## Problem

In cloud mode, the coordinator publishes real-time task stream events (text chunks, tool usage, status updates) to the `ailang-events` Pub/Sub topic via `PubSubBroadcaster`. The dashboard server has no code to consume these events. WebSocket clients connected to the dashboard see no live build progress.

**Current state:**

```
LOCAL MODE (works):
  Coordinator → HTTP POST → Dashboard /api/coordinator/events → WebSocket broadcast → Browser

CLOUD MODE (broken):
  Coordinator → Pub/Sub ailang-events topic → ??? → Dashboard has no subscriber
                                                     WebSocket clients get nothing
```

**Infrastructure that already exists:**
- `ailang-events` Pub/Sub topic (Terraform)
- `ailang-events-dashboard` pull subscription (Terraform, 1-hour retention)
- `PubSubBroadcaster` in coordinator publishes `TaskStreamEvent` JSON
- Dashboard WebSocket server with `BroadcastTaskEvent()` method
- `internal/pubsub/subscriber.go` with `Subscribe()` pull method
- Dashboard Cloud Run has `AILANG_CLOUD_PROJECT` and `AILANG_TOPIC_PREFIX` env vars

**Only missing piece:** A background goroutine in the dashboard server that pulls from the `ailang-events-dashboard` subscription and forwards events to WebSocket clients.

## Design

### Architecture

```
CLOUD MODE (after fix):
  Coordinator (Cloud Run)
    → PubSubBroadcaster.Broadcast(event)
      → Pub/Sub ailang-events topic
        → ailang-events-dashboard subscription (pull)
          → Dashboard PubSubEventSubscriber goroutine
            → json.Unmarshal → TaskStreamEvent
              → wsServer.BroadcastTaskEvent()
                → All connected WebSocket clients
```

### New Component: `PubSubEventSubscriber`

**File:** `internal/server/pubsub_events.go` (~80 lines)

A lightweight struct that:
1. Creates a `pubsub.Client` and `pubsub.Subscriber` from environment variables
2. Runs a blocking `Subscribe()` in a background goroutine
3. For each message: unmarshal `TaskStreamEvent` JSON, call `BroadcastTaskEvent()`
4. Stops cleanly on server shutdown via context cancellation

```go
type PubSubEventSubscriber struct {
    subscriber *pubsub.Subscriber
    wsServer   *websocket.Server
    subName    string
    logger     *log.Logger
    cancel     context.CancelFunc
}
```

**Message handler:**
```go
func (s *PubSubEventSubscriber) handleEvent(ctx context.Context, data []byte, attrs map[string]string) error {
    var event websocket.TaskStreamEvent
    if err := json.Unmarshal(data, &event); err != nil {
        // Ack malformed messages to prevent infinite retry
        log.Printf("PubSubEventSubscriber: bad event JSON (acking): %v", err)
        return nil
    }
    s.wsServer.BroadcastTaskEvent(&event)
    return nil // Ack
}
```

### Integration Points

**1. Server Option:**

```go
// WithPubSubEvents enables Pub/Sub event streaming for cloud mode.
func WithPubSubEvents(subscriber *pubsub.Subscriber, topicPrefix string) ServerOption
```

**2. Server startup (`cmd/ailang/server.go`):**

In cloud mode (when `AILANG_STORAGE != local`), create the Pub/Sub client and subscriber, then pass via `WithPubSubEvents`:

```go
if storageMode != storage.ModeLocal {
    // ... existing backend setup ...

    // Create Pub/Sub subscriber for event streaming
    psClient, err := pubsub.NewClient(ctx, project, topicPrefix)
    if err == nil {
        psSub := pubsub.NewSubscriber(psClient)
        serverOpts = append(serverOpts, server.WithPubSubEvents(psSub, topicPrefix))
        defer psClient.Close()
        defer psSub.Stop()
    }
}
```

**3. Server.Start():**

After starting the WebSocket event loop, start the Pub/Sub subscriber goroutine:

```go
func (s *Server) Start() error {
    go s.wsServer.Run()

    // Start Pub/Sub event subscriber if configured (cloud mode)
    if s.pubsubEventSub != nil {
        go s.pubsubEventSub.Start(context.Background())
    }
    // ... rest of Start() ...
}
```

**4. Server.Close():**

Stop the subscriber before closing other resources:

```go
func (s *Server) Close() error {
    if s.pubsubEventSub != nil {
        s.pubsubEventSub.Stop()
    }
    // ... existing close logic ...
}
```

### No Terraform Changes Required

The `ailang-events-dashboard` subscription already exists as a pull subscription. The dashboard service account already has Pub/Sub permissions. No infrastructure changes needed.

### Environment Variables

Already set on the dashboard Cloud Run service:

| Variable | Value | Used For |
|----------|-------|----------|
| `AILANG_CLOUD_PROJECT` | GCP project ID | Pub/Sub client creation |
| `AILANG_TOPIC_PREFIX` | `ailang-dev` (or similar) | Subscription name prefix |
| `AILANG_STORAGE` | `gcp` | Triggers cloud mode |

### Error Handling

- **Malformed JSON**: Ack (prevent infinite retry), log warning
- **Pub/Sub connection failure**: Log error, dashboard continues without live events (graceful degradation)
- **Subscriber goroutine crash**: Log error, no auto-restart (Cloud Run will restart the container)
- **Multiple dashboard instances**: All instances subscribe to the same pull subscription. Pub/Sub delivers each message to ONE subscriber instance. This means each event is broadcast to WebSocket clients on only one dashboard instance — which is correct since WebSocket connections are instance-local.

### Scaling Consideration

Dashboard scales to 3 instances. With pull subscriptions, Pub/Sub load-balances messages across instances. Each instance only broadcasts to its own WebSocket clients. This is the correct behavior — events reach all browsers because each browser connects to one instance.

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `internal/server/pubsub_events.go` | **New** — PubSubEventSubscriber | ~80 |
| `internal/server/server.go` | Add field + ServerOption + Start/Close hooks | ~20 |
| `cmd/ailang/server.go` | Create Pub/Sub client in cloud mode | ~15 |
| `internal/server/pubsub_events_test.go` | **New** — Unit tests | ~60 |

**Total: ~175 lines**

## Testing

```bash
# Unit tests (mock subscriber)
go test ./internal/server/ -run TestPubSubEvent -v

# Integration test (requires GCP project)
AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-dev ailang serve

# Verify in another terminal:
# 1. Start a coordinator task
# 2. Open dashboard WebSocket
# 3. Confirm events stream in real-time
```

## Follow-Up

1. **Integration guide update**: Add "Live Build Progress" section documenting WebSocket connectivity for external clients
2. **Push subscription option**: Could convert `events-dashboard` to push subscription (Pub/Sub POSTs to dashboard endpoint), eliminating the pull goroutine — but pull is simpler and already works
3. **Event persistence**: Currently events are broadcast-only (not stored by dashboard). Could persist to Firestore for historical replay if needed
