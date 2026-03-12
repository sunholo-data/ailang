// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSEventType represents the type of real-time WebSocket event.
type WSEventType string

const (
	EventTypeSpanCreated       WSEventType = "span.created"
	EventTypeSpanUpdated       WSEventType = "span.updated"
	EventTypeTaskCreated       WSEventType = "task.created"
	EventTypeTaskUpdated       WSEventType = "task.updated"
	EventTypeTaskCompleted     WSEventType = "task.completed"
	EventTypeMessageCreated    WSEventType = "message.created"
	EventTypeMessageRead       WSEventType = "message.read"
	EventTypeAgentAssigned     WSEventType = "agent.assigned"
	EventTypeAgentCompleted    WSEventType = "agent.completed"
	EventTypeApprovalRequested WSEventType = "approval.requested"
	EventTypeApprovalDecision  WSEventType = "approval.decision"
	EventTypeMetricsUpdated    WSEventType = "metrics.updated"
)

// Event represents a real-time event for WebSocket broadcast.
type Event struct {
	Type      WSEventType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      any         `json:"data"`
}

// Subscription represents a client's subscription preferences.
type Subscription struct {
	// Filter events by workspace
	WorkspaceID string `json:"workspace_id,omitempty"`
	// Filter events by task
	TaskID string `json:"task_id,omitempty"`
	// Filter events by type (empty = all types)
	EventTypes []WSEventType `json:"event_types,omitempty"`
}

// Client represents a connected WebSocket client.
type Client struct {
	id           string
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	subscription *Subscription
	mu           sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts events.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound events to broadcast
	broadcast chan *Event

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for clients map
	mu sync.RWMutex

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Authentication token for external WebSocket clients.
	// When set, non-same-origin connections must provide ?token=<value>.
	token string
}

// SetToken sets the authentication token for external WebSocket clients.
func (h *Hub) SetToken(token string) {
	h.token = token
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				// TODO: Configure allowed origins for production
				return true
			},
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run starts the hub's main event loop.
func (h *Hub) Run() {
	for {
		select {
		case <-h.ctx.Done():
			// Shutdown: close all clients
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			message, err := json.Marshal(event)
			if err != nil {
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				if client.matchesEvent(event) {
					select {
					case client.send <- message:
					default:
						// Client buffer full, disconnect
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully shuts down the hub.
func (h *Hub) Stop() {
	h.cancel()
}

// Broadcast sends an event to all matching clients.
func (h *Hub) Broadcast(event *Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	select {
	case h.broadcast <- event:
	default:
		// Broadcast channel full, drop event
	}
}

// BroadcastSpanCreated sends a span.created event.
func (h *Hub) BroadcastSpanCreated(span *Span) {
	h.Broadcast(&Event{
		Type: EventTypeSpanCreated,
		Data: span,
	})
}

// BroadcastSpanUpdated sends a span.updated event.
func (h *Hub) BroadcastSpanUpdated(span *Span) {
	h.Broadcast(&Event{
		Type: EventTypeSpanUpdated,
		Data: span,
	})
}

// BroadcastTaskCreated sends a task.created event.
func (h *Hub) BroadcastTaskCreated(task *Task) {
	h.Broadcast(&Event{
		Type: EventTypeTaskCreated,
		Data: task,
	})
}

// BroadcastTaskUpdated sends a task.updated event.
func (h *Hub) BroadcastTaskUpdated(task *Task) {
	h.Broadcast(&Event{
		Type: EventTypeTaskUpdated,
		Data: task,
	})
}

// BroadcastTaskCompleted sends a task.completed event.
func (h *Hub) BroadcastTaskCompleted(task *Task) {
	h.Broadcast(&Event{
		Type: EventTypeTaskCompleted,
		Data: task,
	})
}

// BroadcastMessageCreated sends a message.created event.
func (h *Hub) BroadcastMessageCreated(msg *Message) {
	h.Broadcast(&Event{
		Type: EventTypeMessageCreated,
		Data: msg,
	})
}

// BroadcastApprovalRequested sends an approval.requested event.
func (h *Hub) BroadcastApprovalRequested(data any) {
	h.Broadcast(&Event{
		Type: EventTypeApprovalRequested,
		Data: data,
	})
}

// BroadcastApprovalDecision sends an approval.decision event.
func (h *Hub) BroadcastApprovalDecision(data any) {
	h.Broadcast(&Event{
		Type: EventTypeApprovalDecision,
		Data: data,
	})
}

// BroadcastMetricsUpdated sends a metrics.updated event.
func (h *Hub) BroadcastMetricsUpdated(summary *MetricsSummary) {
	h.Broadcast(&Event{
		Type: EventTypeMetricsUpdated,
		Data: summary,
	})
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket handles a WebSocket connection upgrade request.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Token authentication for external clients.
	// Same-origin browser connections (embedded React UI) are exempt.
	if h.token != "" {
		origin := r.Header.Get("Origin")
		isSameOrigin := origin != "" && (origin == "http://"+r.Host || origin == "https://"+r.Host)
		if !isSameOrigin {
			qToken := r.URL.Query().Get("token")
			if qToken != h.token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		id:   r.RemoteAddr,
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Start reader and writer goroutines
	go client.readPump()
	go client.writePump()
}

// matchesEvent checks if a client's subscription matches an event.
func (c *Client) matchesEvent(event *Event) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// No subscription means accept all events
	if c.subscription == nil {
		return true
	}

	// Check event type filter
	if len(c.subscription.EventTypes) > 0 {
		found := false
		for _, t := range c.subscription.EventTypes {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check workspace filter
	if c.subscription.WorkspaceID != "" {
		// Extract workspace ID from event data if possible
		if data, ok := event.Data.(map[string]any); ok {
			if wsID, ok := data["workspace_id"].(string); ok {
				if wsID != c.subscription.WorkspaceID {
					return false
				}
			}
		}
		// For typed structs, check common fields
		if task, ok := event.Data.(*Task); ok && task.WorkspaceID != c.subscription.WorkspaceID {
			return false
		}
	}

	// Check task filter
	if c.subscription.TaskID != "" {
		if data, ok := event.Data.(map[string]any); ok {
			if taskID, ok := data["task_id"].(string); ok {
				if taskID != c.subscription.TaskID {
					return false
				}
			}
		}
		if span, ok := event.Data.(*Span); ok && span.TaskID != c.subscription.TaskID {
			return false
		}
		if task, ok := event.Data.(*Task); ok && task.ID != c.subscription.TaskID {
			return false
		}
		if msg, ok := event.Data.(*Message); ok && msg.TaskID != c.subscription.TaskID {
			return false
		}
	}

	return true
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle subscription updates
		var sub Subscription
		if err := json.Unmarshal(message, &sub); err == nil {
			c.mu.Lock()
			c.subscription = &sub
			c.mu.Unlock()
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
