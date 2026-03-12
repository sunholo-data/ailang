package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo/ailang/internal/messaging"
)

const (
	// Message batching
	MaxBatchSize = 50

	// Throttling
	MaxMessagesPerSecond = 100

	// Backpressure threshold
	MaxLagMessages = 1000

	// WebSocket configuration
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = (PongWait * 9) / 10
	MaxMessageSize = 10 * 1024 // 10KB

	// Database polling for agent messages (agents write directly to DB)
	PollInterval = 500 * time.Millisecond
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin header
		return true
	},
}

// Server manages WebSocket connections and message broadcasting
type Server struct {
	store       messaging.MessageStore
	connections map[string]*Connection // connectionID -> Connection
	mu          sync.RWMutex
	broadcast   chan *BroadcastMessage
	register    chan *Connection
	unregister  chan *Connection

	// Polling state: track last seen seq for each subscribed thread
	threadSeqMu sync.RWMutex
	threadSeq   map[string]int // threadID -> last seen message_seq

	// Authentication token for external WebSocket clients.
	// When set, non-same-origin connections must provide ?token=<value>.
	token string
}

// SetToken sets the authentication token for external WebSocket clients.
func (s *Server) SetToken(token string) {
	s.token = token
}

// Connection represents a WebSocket client connection
type Connection struct {
	id            string
	server        *Server
	conn          *websocket.Conn
	send          chan *Event
	subscriptions map[string]*Subscription // threadID -> Subscription
	mu            sync.RWMutex
	instanceID    string // AILANG instance ID or "user" for human
}

// Subscription tracks a client's subscription to a thread
type Subscription struct {
	ThreadID     string
	FromSeq      int
	LastAckSeq   int
	SubscribedAt time.Time
}

// BroadcastMessage represents a message to be broadcast to subscribers
type BroadcastMessage struct {
	ThreadID string
	Message  *messaging.Message
}

// NewServer creates a new WebSocket server
func NewServer(store messaging.MessageStore) *Server {
	return &Server{
		store:       store,
		connections: make(map[string]*Connection),
		broadcast:   make(chan *BroadcastMessage, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		threadSeq:   make(map[string]int),
	}
}

// Run starts the WebSocket server event loop
func (s *Server) Run() {
	// Start database polling goroutine for agent messages
	go s.pollDatabaseForNewMessages()

	for {
		select {
		case conn := <-s.register:
			s.mu.Lock()
			s.connections[conn.id] = conn
			s.mu.Unlock()
			log.Printf("WebSocket: registered connection %s", conn.id)

		case conn := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.connections[conn.id]; ok {
				delete(s.connections, conn.id)
				close(conn.send)
				log.Printf("WebSocket: unregistered connection %s", conn.id)
			}
			s.mu.Unlock()

		case msg := <-s.broadcast:
			s.broadcastToThread(msg.ThreadID, msg.Message)
		}
	}
}

// pollDatabaseForNewMessages polls the database for new messages in subscribed threads
// This is needed because agents write directly to the database, bypassing the REST API
func (s *Server) pollDatabaseForNewMessages() {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.checkForNewMessages()
	}
}

// checkForNewMessages checks all subscribed threads for new messages
func (s *Server) checkForNewMessages() {
	// Collect all subscribed threads across all connections
	subscribedThreads := make(map[string]int) // threadID -> min lastAckSeq

	s.mu.RLock()
	for _, conn := range s.connections {
		conn.mu.RLock()
		for threadID, sub := range conn.subscriptions {
			if existing, ok := subscribedThreads[threadID]; !ok || sub.LastAckSeq < existing {
				subscribedThreads[threadID] = sub.LastAckSeq
			}
		}
		conn.mu.RUnlock()
	}
	s.mu.RUnlock()

	if len(subscribedThreads) == 0 {
		return // No subscriptions, nothing to poll
	}

	// For each subscribed thread, check for new messages since last seen
	for threadID, minAckSeq := range subscribedThreads {
		s.threadSeqMu.RLock()
		lastSeen := s.threadSeq[threadID]
		s.threadSeqMu.RUnlock()

		// Use the higher of lastSeen or minAckSeq
		fromSeq := lastSeen
		if minAckSeq > fromSeq {
			fromSeq = minAckSeq
		}

		// Query for new messages
		messages, err := s.store.GetMessagesFromSeq(threadID, fromSeq, MaxBatchSize)
		if err != nil {
			continue // Silently skip errors in polling
		}

		if len(messages) == 0 {
			continue
		}

		// Update last seen seq
		maxSeq := fromSeq
		for _, msg := range messages {
			if msg.MessageSeq > maxSeq {
				maxSeq = msg.MessageSeq
			}
			// Broadcast each new message
			s.broadcastToThread(threadID, &msg)
		}

		s.threadSeqMu.Lock()
		if maxSeq > s.threadSeq[threadID] {
			s.threadSeq[threadID] = maxSeq
		}
		s.threadSeqMu.Unlock()
	}
}

// broadcastToThread sends a message to all subscribers of a thread
func (s *Server) broadcastToThread(threadID string, msg *messaging.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		conn.mu.RLock()
		sub, subscribed := conn.subscriptions[threadID]
		conn.mu.RUnlock()

		if subscribed && msg.MessageSeq > sub.LastAckSeq {
			// Send message to this connection
			event, err := NewMessageEvent(&MessageEvent{
				ID:           msg.ID,
				ThreadID:     msg.ThreadID,
				MessageSeq:   msg.MessageSeq,
				CreatedAt:    msg.CreatedAt.UnixMilli(),
				FromType:     msg.FromType,
				FromID:       msg.FromID,
				ToType:       msg.ToType,
				ToID:         msg.ToID,
				Kind:         msg.Kind,
				Content:      msg.Content,
				MetadataJSON: msg.MetadataJSON,
			})
			if err != nil {
				log.Printf("WebSocket: failed to create message event: %v", err)
				continue
			}

			select {
			case conn.send <- event:
			default:
				// Connection is slow, skip this message (it will be replayed on reconnect)
				log.Printf("WebSocket: connection %s is slow, skipping message", conn.id)
			}
		}
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Token authentication for external clients.
	// Same-origin browser connections (embedded React UI) are exempt.
	if s.token != "" {
		origin := r.Header.Get("Origin")
		isSameOrigin := origin != "" && (origin == "http://"+r.Host || origin == "https://"+r.Host)
		if !isSameOrigin {
			qToken := r.URL.Query().Get("token")
			if qToken != s.token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket: upgrade failed: %v", err)
		return
	}

	// Extract instance ID from query params or auth
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		instanceID = "user" // Default to user if not specified
	}

	// Create connection
	connection := &Connection{
		id:            fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		server:        s,
		conn:          conn,
		send:          make(chan *Event, 256),
		subscriptions: make(map[string]*Subscription),
		instanceID:    instanceID,
	}

	// Register connection
	s.register <- connection

	// Start read and write pumps
	go connection.writePump()
	go connection.readPump()
}

// readPump handles incoming messages from the WebSocket connection
func (c *Connection) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket: read error: %v", err)
			}
			break
		}

		// Parse event
		event, err := ParseEvent(message)
		if err != nil {
			log.Printf("WebSocket: failed to parse event: %v", err)
			c.sendError("PARSE_ERROR", fmt.Sprintf("Failed to parse event: %v", err))
			continue
		}

		// Handle event
		c.handleEvent(event)
	}
}

// writePump handles outgoing messages to the WebSocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// Channel closed
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write event as JSON
			data, err := event.ToJSON()
			if err != nil {
				log.Printf("WebSocket: failed to serialize event: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleEvent processes incoming events from the client
func (c *Connection) handleEvent(event *Event) {
	switch event.Type {
	case EventTypeSubscribe:
		c.handleSubscribe(event)
	case EventTypeAck:
		c.handleAck(event)
	case EventTypePing:
		c.handlePing()
	default:
		c.sendError("UNKNOWN_EVENT", fmt.Sprintf("Unknown event type: %s", event.Type))
	}
}

// handleSubscribe handles subscribe events
func (c *Connection) handleSubscribe(event *Event) {
	sub, err := ParseSubscribe(event)
	if err != nil {
		c.sendError("INVALID_SUBSCRIBE", err.Error())
		return
	}

	// Create subscription in database
	if err := c.server.store.Subscribe(c.instanceID, sub.ThreadID); err != nil {
		c.sendError("SUBSCRIBE_FAILED", err.Error())
		return
	}

	// Track subscription in connection
	c.mu.Lock()
	c.subscriptions[sub.ThreadID] = &Subscription{
		ThreadID:     sub.ThreadID,
		FromSeq:      sub.FromSeq,
		LastAckSeq:   sub.FromSeq,
		SubscribedAt: time.Now(),
	}
	c.mu.Unlock()

	// Initialize thread sequence tracking for polling (prevents re-sending old messages)
	c.server.threadSeqMu.Lock()
	if existing := c.server.threadSeq[sub.ThreadID]; sub.FromSeq > existing {
		c.server.threadSeq[sub.ThreadID] = sub.FromSeq
	}
	c.server.threadSeqMu.Unlock()

	log.Printf("WebSocket: connection %s subscribed to thread %s from seq %d", c.id, sub.ThreadID, sub.FromSeq)

	// Send missed messages (resume from last ack)
	c.sendMissedMessages(sub.ThreadID, sub.FromSeq)
}

// sendMissedMessages sends messages that the client missed since last ack
func (c *Connection) sendMissedMessages(threadID string, fromSeq int) {
	// Fetch messages from database
	messages, err := c.server.store.GetMessagesFromSeq(threadID, fromSeq, MaxBatchSize)
	if err != nil {
		c.sendError("FETCH_FAILED", err.Error())
		return
	}

	if len(messages) == 0 {
		// No missed messages, send empty batch to confirm subscription
		event, _ := NewBatchEvent(threadID, []MessageEvent{}, false)
		c.send <- event
		return
	}

	// Convert to MessageEvents
	var msgEvents []MessageEvent
	for _, msg := range messages {
		msgEvents = append(msgEvents, MessageEvent{
			ID:           msg.ID,
			ThreadID:     msg.ThreadID,
			MessageSeq:   msg.MessageSeq,
			CreatedAt:    msg.CreatedAt.UnixMilli(),
			FromType:     msg.FromType,
			FromID:       msg.FromID,
			ToType:       msg.ToType,
			ToID:         msg.ToID,
			Kind:         msg.Kind,
			Content:      msg.Content,
			MetadataJSON: msg.MetadataJSON,
		})
	}

	// Send batch
	hasMore := len(messages) == MaxBatchSize
	event, err := NewBatchEvent(threadID, msgEvents, hasMore)
	if err != nil {
		c.sendError("BATCH_FAILED", err.Error())
		return
	}

	c.send <- event
}

// handleAck handles ack events
func (c *Connection) handleAck(event *Event) {
	ack, err := ParseAck(event)
	if err != nil {
		c.sendError("INVALID_ACK", err.Error())
		return
	}

	// Update subscription in database
	if err := c.server.store.UpdateAckSeq(c.instanceID, ack.ThreadID, ack.AckSeq); err != nil {
		c.sendError("ACK_FAILED", err.Error())
		return
	}

	// Update local subscription
	c.mu.Lock()
	if sub, ok := c.subscriptions[ack.ThreadID]; ok {
		sub.LastAckSeq = ack.AckSeq
	}
	c.mu.Unlock()

	log.Printf("WebSocket: connection %s acked thread %s up to seq %d", c.id, ack.ThreadID, ack.AckSeq)
}

// handlePing handles ping events
func (c *Connection) handlePing() {
	event, _ := NewPongEvent()
	c.send <- event
}

// sendError sends an error event to the client
func (c *Connection) sendError(code, message string) {
	event, err := NewErrorEvent(code, message)
	if err != nil {
		log.Printf("WebSocket: failed to create error event: %v", err)
		return
	}

	select {
	case c.send <- event:
	default:
		log.Printf("WebSocket: failed to send error (channel full)")
	}
}

// BroadcastMessage broadcasts a message to all subscribers of a thread
func (s *Server) BroadcastMessage(threadID string, msg *messaging.Message) {
	s.broadcast <- &BroadcastMessage{
		ThreadID: threadID,
		Message:  msg,
	}
}

// GetConnectionCount returns the number of active connections
func (s *Server) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// BroadcastTelemetry broadcasts a telemetry update to ALL connected clients
// This is used for real-time process monitoring updates
func (s *Server) BroadcastTelemetry(telem *TelemetryEvent) {
	event, err := NewTelemetryEvent(telem)
	if err != nil {
		log.Printf("WebSocket: failed to create telemetry event: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		select {
		case conn.send <- event:
		default:
			// Connection is slow, skip (telemetry is best-effort)
			log.Printf("WebSocket: connection %s is slow, skipping telemetry", conn.id)
		}
	}
}

// BroadcastInboxMessage broadcasts an inbox message to ALL connected clients
// This is used for real-time notification of new async messages
func (s *Server) BroadcastInboxMessage(msg *messaging.InboxMessage) {
	event, err := NewInboxMessageEvent(&InboxMessageEvent{
		ID:            msg.ID,
		MessageID:     msg.MessageID,
		CorrelationID: msg.CorrelationID,
		FromAgent:     msg.FromAgent,
		ToInbox:       msg.ToInbox,
		MessageType:   msg.MessageType,
		Title:         msg.Title,
		Payload:       msg.Payload,
		Status:        msg.Status,
		CreatedAt:     msg.CreatedAt.UnixMilli(),
	})
	if err != nil {
		log.Printf("WebSocket: failed to create inbox message event: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		select {
		case conn.send <- event:
		default:
			// Connection is slow, skip (message can be fetched via REST)
			log.Printf("WebSocket: connection %s is slow, skipping inbox message", conn.id)
		}
	}
}

// BroadcastTaskEvent broadcasts a task streaming event to ALL connected clients
// This is used for real-time task execution monitoring
func (s *Server) BroadcastTaskEvent(stream *TaskStreamEvent) {
	event, err := NewTaskStreamEvent(stream)
	if err != nil {
		log.Printf("WebSocket: failed to create task stream event: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.connections {
		select {
		case conn.send <- event:
		default:
			// Connection is slow, skip (streaming is best-effort)
			log.Printf("WebSocket: connection %s is slow, skipping task event", conn.id)
		}
	}
}
