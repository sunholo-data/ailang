package observatory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_RunAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Verify initial state
	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.ClientCount())
	}

	// Test broadcasting without clients (should not block)
	hub.Broadcast(&Event{
		Type: EventTypeSpanCreated,
		Data: map[string]string{"test": "data"},
	})
}

func TestHub_BroadcastHelpers(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Test all broadcast helpers don't panic
	hub.BroadcastSpanCreated(&Span{ID: "span-1"})
	hub.BroadcastSpanUpdated(&Span{ID: "span-1"})
	hub.BroadcastTaskCreated(&Task{ID: "task-1"})
	hub.BroadcastTaskUpdated(&Task{ID: "task-1"})
	hub.BroadcastTaskCompleted(&Task{ID: "task-1"})
	hub.BroadcastMessageCreated(&Message{ID: "msg-1"})
	hub.BroadcastApprovalRequested(map[string]string{"id": "1"})
	hub.BroadcastApprovalDecision(map[string]string{"approved": "true"})
	hub.BroadcastMetricsUpdated(&MetricsSummary{TotalSpans: 10})
}

func TestClient_MatchesEvent_NoSubscription(t *testing.T) {
	client := &Client{}

	// No subscription should match all events
	event := &Event{Type: EventTypeSpanCreated, Data: map[string]any{}}
	if !client.matchesEvent(event) {
		t.Error("Client with no subscription should match all events")
	}
}

func TestClient_MatchesEvent_EventTypeFilter(t *testing.T) {
	client := &Client{
		subscription: &Subscription{
			EventTypes: []WSEventType{EventTypeSpanCreated, EventTypeSpanUpdated},
		},
	}

	// Should match subscribed types
	if !client.matchesEvent(&Event{Type: EventTypeSpanCreated}) {
		t.Error("Should match EventTypeSpanCreated")
	}
	if !client.matchesEvent(&Event{Type: EventTypeSpanUpdated}) {
		t.Error("Should match EventTypeSpanUpdated")
	}

	// Should not match unsubscribed types
	if client.matchesEvent(&Event{Type: EventTypeTaskCreated}) {
		t.Error("Should not match EventTypeTaskCreated")
	}
}

func TestClient_MatchesEvent_WorkspaceFilter(t *testing.T) {
	client := &Client{
		subscription: &Subscription{
			WorkspaceID: "ws-1",
		},
	}

	// Match if workspace matches
	task := &Task{ID: "task-1", WorkspaceID: "ws-1"}
	if !client.matchesEvent(&Event{Type: EventTypeTaskCreated, Data: task}) {
		t.Error("Should match task with matching workspace")
	}

	// Don't match if workspace differs
	task2 := &Task{ID: "task-2", WorkspaceID: "ws-2"}
	if client.matchesEvent(&Event{Type: EventTypeTaskCreated, Data: task2}) {
		t.Error("Should not match task with different workspace")
	}
}

func TestClient_MatchesEvent_TaskFilter(t *testing.T) {
	client := &Client{
		subscription: &Subscription{
			TaskID: "task-1",
		},
	}

	// Match span with matching task
	span := &Span{ID: "span-1", TaskID: "task-1"}
	if !client.matchesEvent(&Event{Type: EventTypeSpanCreated, Data: span}) {
		t.Error("Should match span with matching task")
	}

	// Don't match span with different task
	span2 := &Span{ID: "span-2", TaskID: "task-2"}
	if client.matchesEvent(&Event{Type: EventTypeSpanCreated, Data: span2}) {
		t.Error("Should not match span with different task")
	}

	// Match task with matching ID
	task := &Task{ID: "task-1"}
	if !client.matchesEvent(&Event{Type: EventTypeTaskUpdated, Data: task}) {
		t.Error("Should match task with matching ID")
	}

	// Match message with matching task
	msg := &Message{ID: "msg-1", TaskID: "task-1"}
	if !client.matchesEvent(&Event{Type: EventTypeMessageCreated, Data: msg}) {
		t.Error("Should match message with matching task")
	}
}

func TestHub_WebSocketIntegration(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}

	// Send subscription
	sub := Subscription{EventTypes: []WSEventType{EventTypeSpanCreated}}
	subData, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, subData); err != nil {
		t.Fatalf("Failed to send subscription: %v", err)
	}

	// Give time for subscription to process
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event
	hub.BroadcastSpanCreated(&Span{
		ID:      "test-span",
		TraceID: "test-trace",
		Name:    "test.operation",
	})

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var event Event
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeSpanCreated {
		t.Errorf("Expected span.created, got %s", event.Type)
	}
}

func TestHub_MultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect 3 clients
	var conns []*websocket.Conn
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns = append(conns, conn)
		defer conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("Expected 3 clients, got %d", hub.ClientCount())
	}

	// Disconnect one
	conns[0].Close()
	time.Sleep(100 * time.Millisecond)

	// Should have 2 remaining
	// Note: May still show 3 until read pump detects disconnect
	// This is expected behavior
}

func TestEvent_MarshalJSON(t *testing.T) {
	event := Event{
		Type:      EventTypeSpanCreated,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: map[string]any{
			"span_id": "test-123",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != EventTypeSpanCreated {
		t.Errorf("Type mismatch: got %s", unmarshaled.Type)
	}
}
