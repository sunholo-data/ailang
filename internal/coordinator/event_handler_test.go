package coordinator

import (
	"sync"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
)

func TestCoordinatorEventHandler_Basic(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	var mu sync.Mutex

	broadcaster := func(event *websocket.TaskStreamEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	handler := NewCoordinatorEventHandler("task-123", "thread-456", broadcaster)

	// Test turn start
	handler.OnTurnStart(1)

	// Test text
	handler.OnText("Hello, world!")

	// Test tool use
	handler.OnToolUse("Read", "file.txt")

	// Test tool result
	handler.OnToolResult("Read", "file contents here")

	// Test turn end
	handler.OnTurnEnd(1)

	// Verify events
	mu.Lock()
	defer mu.Unlock()

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	if events[0].StreamType != websocket.TaskStreamTurnStart {
		t.Errorf("expected turn_start, got %s", events[0].StreamType)
	}
	if events[0].TurnNum != 1 {
		t.Errorf("expected turn 1, got %d", events[0].TurnNum)
	}

	if events[1].StreamType != websocket.TaskStreamText {
		t.Errorf("expected text, got %s", events[1].StreamType)
	}
	if events[1].Text != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %s", events[1].Text)
	}

	if events[2].StreamType != websocket.TaskStreamToolUse {
		t.Errorf("expected tool_use, got %s", events[2].StreamType)
	}
	if events[2].ToolName != "Read" {
		t.Errorf("expected tool name 'Read', got %s", events[2].ToolName)
	}

	if events[3].StreamType != websocket.TaskStreamToolResult {
		t.Errorf("expected tool_result, got %s", events[3].StreamType)
	}

	if events[4].StreamType != websocket.TaskStreamTurnEnd {
		t.Errorf("expected turn_end, got %s", events[4].StreamType)
	}

	// Verify task and thread IDs
	for _, event := range events {
		if event.TaskID != "task-123" {
			t.Errorf("expected task ID 'task-123', got %s", event.TaskID)
		}
		if event.ThreadID != "thread-456" {
			t.Errorf("expected thread ID 'thread-456', got %s", event.ThreadID)
		}
	}
}

func TestCoordinatorEventHandler_RateLimiting(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	var mu sync.Mutex

	broadcaster := func(event *websocket.TaskStreamEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	handler := NewCoordinatorEventHandler("task-123", "", broadcaster)

	// Send 20 text events rapidly (rate limit is 10/sec)
	for i := 0; i < 20; i++ {
		handler.OnText("message")
	}

	mu.Lock()
	count := len(events)
	mu.Unlock()

	// Should have throttled some events
	if count >= 20 {
		t.Errorf("expected rate limiting to reduce events, got %d", count)
	}

	// Should have allowed at least some events (10 per second max)
	if count < 10 {
		t.Errorf("expected at least 10 events through, got %d", count)
	}

	// Verify throttled flag
	if !handler.IsThrottled() {
		t.Error("expected handler to be throttled after burst")
	}
}

func TestCoordinatorEventHandler_RateLimitReset(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	var mu sync.Mutex

	broadcaster := func(event *websocket.TaskStreamEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	handler := NewCoordinatorEventHandler("task-123", "", broadcaster)

	// Send burst to trigger throttling
	for i := 0; i < 15; i++ {
		handler.OnText("message")
	}

	// Wait for rate limit to reset
	time.Sleep(1100 * time.Millisecond)

	// Should be able to send more events now
	mu.Lock()
	countBefore := len(events)
	mu.Unlock()

	handler.OnText("after reset")

	mu.Lock()
	countAfter := len(events)
	mu.Unlock()

	if countAfter <= countBefore {
		t.Error("expected event to be sent after rate limit reset")
	}
}

func TestCoordinatorEventHandler_EventBuffer(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	broadcaster := func(event *websocket.TaskStreamEvent) {
		events = append(events, event)
	}

	handler := NewCoordinatorEventHandler("task-123", "", broadcaster)

	// Send some events
	handler.OnTurnStart(1)
	handler.OnToolUse("Edit", "file.go")
	handler.OnToolResult("Edit", "success")
	handler.OnTurnEnd(1)

	// Get buffer
	buffer := handler.GetEventBuffer()

	if len(buffer) != 4 {
		t.Errorf("expected 4 events in buffer, got %d", len(buffer))
	}

	// Verify buffer is a copy (modifying doesn't affect original)
	buffer[0] = nil
	newBuffer := handler.GetEventBuffer()
	if newBuffer[0] == nil {
		t.Error("buffer should be a copy, not original")
	}
}

func TestCoordinatorEventHandler_StatusEvent(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	broadcaster := func(event *websocket.TaskStreamEvent) {
		events = append(events, event)
	}

	handler := NewCoordinatorEventHandler("task-123", "thread-456", broadcaster)

	// Update metrics
	handler.UpdateMetrics(1000, 500, 0.05)

	// Emit status
	handler.EmitStatus("completed")

	if len(events) != 1 {
		t.Fatalf("expected 1 status event, got %d", len(events))
	}

	event := events[0]
	if event.StreamType != websocket.TaskStreamStatus {
		t.Errorf("expected status event, got %s", event.StreamType)
	}
	if event.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", event.Status)
	}
	if event.TokensIn != 1000 {
		t.Errorf("expected tokens_in 1000, got %d", event.TokensIn)
	}
	if event.TokensOut != 500 {
		t.Errorf("expected tokens_out 500, got %d", event.TokensOut)
	}
	if event.Cost != 0.05 {
		t.Errorf("expected cost 0.05, got %f", event.Cost)
	}
	if event.DurationSec < 0 {
		t.Errorf("expected non-negative duration, got %d", event.DurationSec)
	}
}

func TestCoordinatorEventHandler_Error(t *testing.T) {
	var events []*websocket.TaskStreamEvent
	broadcaster := func(event *websocket.TaskStreamEvent) {
		events = append(events, event)
	}

	handler := NewCoordinatorEventHandler("task-123", "", broadcaster)

	// Test error event
	handler.OnError(nil)

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}

	if events[0].StreamType != websocket.TaskStreamError {
		t.Errorf("expected error event, got %s", events[0].StreamType)
	}
}

func TestCoordinatorEventHandler_TruncateString(t *testing.T) {
	// Test truncation
	long := "This is a very long string that should be truncated"
	truncated := truncateString(long, 20)
	if len(truncated) > 20 {
		t.Errorf("expected string to be truncated to 20 chars, got %d", len(truncated))
	}
	if truncated[len(truncated)-3:] != "..." {
		t.Error("expected truncated string to end with ...")
	}

	// Test no truncation needed
	short := "Short"
	notTruncated := truncateString(short, 20)
	if notTruncated != short {
		t.Errorf("expected '%s', got '%s'", short, notTruncated)
	}
}

func TestCoordinatorEventHandler_NilBroadcaster(t *testing.T) {
	// Should not panic with nil broadcaster
	handler := NewCoordinatorEventHandler("task-123", "", nil)

	handler.OnTurnStart(1)
	handler.OnText("test")
	handler.OnToolUse("Read", "file.txt")
	handler.OnToolResult("Read", "content")
	handler.OnTurnEnd(1)
	handler.OnError(nil)
	handler.EmitStatus("completed")

	// Events should still be buffered
	buffer := handler.GetEventBuffer()
	if len(buffer) != 7 {
		t.Errorf("expected 7 events buffered, got %d", len(buffer))
	}
}
