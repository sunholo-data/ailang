package server

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/sunholo/ailang/internal/websocket"
)

// TestPubSubEventSubscriber_HandleEvent verifies that valid TaskStreamEvent
// JSON is deserialized and returns nil (ack).
func TestPubSubEventSubscriber_HandleEvent(t *testing.T) {
	event := &websocket.TaskStreamEvent{
		TaskID:     "task-abc123",
		StreamType: websocket.TaskStreamText,
		Text:       "hello world",
		TurnNum:    1,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	// Test the handler directly. We can't call BroadcastTaskEvent without
	// a running WebSocket server, so we test deserialization + ack logic.
	sub := &PubSubEventSubscriber{
		logger: log.Default(),
	}

	// handleEvent requires wsServer for BroadcastTaskEvent.
	// Create a minimal websocket server (no actual connections).
	sub.wsServer = websocket.NewServer(nil)

	err = sub.handleEvent(context.Background(), data, nil)
	if err != nil {
		t.Fatalf("handleEvent returned error (should ack): %v", err)
	}
}

// TestPubSubEventSubscriber_HandleEvent_MalformedJSON verifies that
// malformed JSON is acked (returns nil) to prevent infinite Pub/Sub retries.
func TestPubSubEventSubscriber_HandleEvent_MalformedJSON(t *testing.T) {
	sub := &PubSubEventSubscriber{
		logger: log.Default(),
	}

	err := sub.handleEvent(context.Background(), []byte("not valid json"), nil)
	if err != nil {
		t.Fatalf("expected nil (ack) for malformed JSON, got error: %v", err)
	}
}

// TestPubSubEventSubscriber_Stop verifies Stop is safe to call
// when subscriber hasn't started (cancel is nil).
func TestPubSubEventSubscriber_Stop_NilCancel(t *testing.T) {
	sub := &PubSubEventSubscriber{
		logger: log.Default(),
	}
	// Should not panic.
	sub.Stop()
}
