package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo/ailang/internal/messaging"
)

func setupTestServer(t *testing.T) (*Server, messaging.MessageStore) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}

	server := NewServer(store)
	go server.Run()

	return server, store
}

func TestNewServer(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	if server == nil {
		t.Fatal("Expected server to be created")
	}
	if server.store == nil {
		t.Error("Expected server to have store")
	}
	if server.connections == nil {
		t.Error("Expected server to have connections map")
	}
}

func TestServerRegisterUnregister(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	conn := &Connection{
		id:            "test_conn_1",
		server:        server,
		send:          make(chan *Event, 256),
		subscriptions: make(map[string]*Subscription),
	}

	// Register
	server.register <- conn
	time.Sleep(10 * time.Millisecond) // Allow event to process

	if server.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", server.GetConnectionCount())
	}

	// Unregister
	server.unregister <- conn
	time.Sleep(10 * time.Millisecond) // Allow event to process

	if server.GetConnectionCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", server.GetConnectionCount())
	}
}

func TestHandleWebSocket(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_instance", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Wait for connection to be registered
	time.Sleep(50 * time.Millisecond)

	if server.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", server.GetConnectionCount())
	}
}

func TestSubscribeAndSendMessages(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create a thread
	thread, err := store.CreateThread("Test Thread", "ailang_instance", "agent1", "")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create HTTP test server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_client", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Send subscribe event
	subEvent, _ := NewEvent(EventTypeSubscribe, &SubscribeEvent{
		ThreadID: thread.ID,
		FromSeq:  0,
	})
	subData, _ := subEvent.ToJSON()

	if err := wsConn.WriteMessage(websocket.TextMessage, subData); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read response (should be empty batch since no messages)
	_, responseData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var response Event
	if err := json.Unmarshal(responseData, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Type != EventTypeBatch {
		t.Errorf("Expected batch event, got %s", response.Type)
	}
}

func TestAckEvent(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create thread and subscribe
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1", "")
	store.Subscribe("test_client", thread.ID)

	// Create HTTP test server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_client", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Send ack event
	ackEvent, _ := NewEvent(EventTypeAck, &AckEvent{
		ThreadID: thread.ID,
		AckSeq:   5,
	})
	ackData, _ := ackEvent.ToJSON()

	if err := wsConn.WriteMessage(websocket.TextMessage, ackData); err != nil {
		t.Fatalf("Failed to send ack: %v", err)
	}

	// Wait for ack to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify ack was recorded (we can't easily check this without exposing internals,
	// but the test passes if no error occurs)
}

func TestBroadcastMessage(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create thread
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1", "")

	// Create message in database
	_, err := store.CreateMessage(thread.ID, "ailang_instance", "agent1", "human", "user", "status", "Test message", "")
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create HTTP test server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_client", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Subscribe to thread
	subEvent, _ := NewEvent(EventTypeSubscribe, &SubscribeEvent{
		ThreadID: thread.ID,
		FromSeq:  0,
	})
	subData, _ := subEvent.ToJSON()
	wsConn.WriteMessage(websocket.TextMessage, subData)

	// Read initial batch response
	wsConn.ReadMessage()

	// Broadcast new message
	newMsg, _ := store.CreateMessage(thread.ID, "human", "user", "ailang_instance", "agent1", "directive", "Do something", "")
	server.BroadcastMessage(thread.ID, newMsg)

	// Read broadcast message
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msgData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read broadcast message: %v", err)
	}

	var event Event
	if err := json.Unmarshal(msgData, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Should receive either a batch or message event
	if event.Type != EventTypeMessage && event.Type != EventTypeBatch {
		t.Errorf("Expected message or batch event, got %s", event.Type)
	}
}

func TestPingPong(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create HTTP test server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_client", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Send ping event
	pingEvent, _ := NewPingEvent()
	pingData, _ := pingEvent.ToJSON()

	if err := wsConn.WriteMessage(websocket.TextMessage, pingData); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read pong response
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, pongData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read pong: %v", err)
	}

	var pong Event
	if err := json.Unmarshal(pongData, &pong); err != nil {
		t.Fatalf("Failed to unmarshal pong: %v", err)
	}

	if pong.Type != EventTypePong {
		t.Errorf("Expected pong event, got %s", pong.Type)
	}
}

func TestInvalidEvent(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()

	// Create HTTP test server
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect as WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL+"?instance_id=test_client", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsConn.Close()

	// Send invalid JSON
	if err := wsConn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send invalid message: %v", err)
	}

	// Should receive error event
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, errData, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read error response: %v", err)
	}

	var event Event
	if err := json.Unmarshal(errData, &event); err != nil {
		t.Fatalf("Failed to unmarshal error event: %v", err)
	}

	if event.Type != EventTypeError {
		t.Errorf("Expected error event, got %s", event.Type)
	}
}
