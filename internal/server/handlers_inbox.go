package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// GET /api/inbox - List inbox messages
// POST /api/inbox - Send a message
func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListInbox(w, r)
	case http.MethodPost:
		s.handleSendInbox(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/inbox/{id} - Get single message
// PUT /api/inbox/{id} - Update message (ack/unack)
func (s *Server) handleInboxMessage(w http.ResponseWriter, r *http.Request) {
	// Extract message ID from path
	msgID := r.URL.Path[len("/api/inbox/"):]
	if msgID == "" {
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetInboxMessage(w, r, msgID)
	case http.MethodPut:
		s.handleUpdateInboxMessage(w, r, msgID)
	case http.MethodDelete:
		s.handleDeleteInboxMessage(w, r, msgID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListInbox(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	opts := messaging.InboxListOptions{
		Inbox:      q.Get("inbox"),
		FromAgent:  q.Get("from"),
		UnreadOnly: q.Get("unread") == "true",
		Limit:      50,
	}

	if q.Get("status") != "" {
		opts.Status = q.Get("status")
	}

	messages, err := s.store.ListInboxMessages(opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list messages: %v", err), http.StatusInternalServerError)
		return
	}

	// Also get counts for the response
	counts, _ := s.store.CountInboxMessagesByStatus(opts.Inbox)

	response := struct {
		Messages []messaging.InboxMessage `json:"messages"`
		Counts   map[string]int64         `json:"counts"`
	}{
		Messages: messages,
		Counts:   counts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode inbox response: %v", err)
	}
}

func (s *Server) handleSendInbox(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ToInbox       string `json:"to_inbox"`
		FromAgent     string `json:"from_agent"`
		Title         string `json:"title"`
		Payload       string `json:"payload"`
		MessageType   string `json:"message_type"`
		CorrelationID string `json:"correlation_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.ToInbox == "" {
		http.Error(w, "to_inbox is required", http.StatusBadRequest)
		return
	}

	if body.Title == "" && body.Payload == "" {
		http.Error(w, "title or payload is required", http.StatusBadRequest)
		return
	}

	// Default values
	if body.FromAgent == "" {
		body.FromAgent = "dashboard"
	}
	if body.MessageType == "" {
		body.MessageType = messaging.InboxTypeNotification
	}
	if body.Title == "" {
		// Use truncated payload as title
		title := body.Payload
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		body.Title = title
	}

	msg := &messaging.InboxMessage{
		ToInbox:       body.ToInbox,
		FromAgent:     body.FromAgent,
		Title:         body.Title,
		Payload:       body.Payload,
		MessageType:   body.MessageType,
		CorrelationID: body.CorrelationID,
	}

	if err := s.store.InsertInboxMessage(msg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send message: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast new message via WebSocket
	s.wsServer.BroadcastInboxMessage(msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		log.Printf("Failed to encode inbox message response: %v", err)
	}
}

func (s *Server) handleGetInboxMessage(w http.ResponseWriter, r *http.Request, msgID string) {
	msg, err := s.store.GetInboxMessage(msgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get message: %v", err), http.StatusInternalServerError)
		return
	}

	if msg == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		log.Printf("Failed to encode inbox message response: %v", err)
	}
}

func (s *Server) handleUpdateInboxMessage(w http.ResponseWriter, r *http.Request, msgID string) {
	var body struct {
		Action string `json:"action"` // "ack" or "unack"
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	switch body.Action {
	case "ack", "read":
		err = s.store.MarkInboxMessageRead(msgID)
	case "unack", "unread":
		err = s.store.MarkInboxMessageUnread(msgID)
	default:
		http.Error(w, "Invalid action (use 'ack' or 'unack')", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update message: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated message
	msg, _ := s.store.GetInboxMessage(msgID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		log.Printf("Failed to encode inbox message response: %v", err)
	}
}

func (s *Server) handleDeleteInboxMessage(w http.ResponseWriter, r *http.Request, msgID string) {
	// Mark as deleted (soft delete via status change)
	// For now, we'll use cleanup instead
	http.Error(w, "Delete not implemented - use cleanup instead", http.StatusNotImplemented)
}

// POST /api/inbox/ack-all - Acknowledge all messages
func (s *Server) handleAckAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Inbox string `json:"inbox"` // Optional: filter by inbox
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Allow empty body
		body.Inbox = ""
	}

	count, err := s.store.MarkAllInboxMessagesRead(body.Inbox)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to ack messages: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Count   int64  `json:"count"`
		Message string `json:"message"`
	}{
		Count:   count,
		Message: fmt.Sprintf("%d message(s) marked as read", count),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode ack-all response: %v", err)
	}
}

// POST /api/inbox/cleanup - Clean up old messages
func (s *Server) handleInboxCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		OlderThanDays int  `json:"older_than_days"` // Default: 7
		ExpiredOnly   bool `json:"expired_only"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.OlderThanDays = 7
	}

	if body.OlderThanDays <= 0 {
		body.OlderThanDays = 7
	}

	olderThan := time.Duration(body.OlderThanDays) * 24 * time.Hour

	count, err := s.store.CleanupInboxMessages(olderThan, body.ExpiredOnly)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cleanup messages: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Count   int64  `json:"count"`
		Message string `json:"message"`
	}{
		Count:   count,
		Message: fmt.Sprintf("%d message(s) cleaned up", count),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode cleanup response: %v", err)
	}
}
