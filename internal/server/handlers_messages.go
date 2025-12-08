package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// GET /api/messages?thread_id={id} - Get messages for a thread
// POST /api/messages - Send a message
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetMessages(w, r)
	case http.MethodPost:
		s.handleSendMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	threadID := r.URL.Query().Get("thread_id")
	if threadID == "" {
		http.Error(w, "thread_id required", http.StatusBadRequest)
		return
	}

	// Get messages from sequence 0 with limit 100
	messages, err := s.store.GetMessagesFromSeq(threadID, 0, 100)
	if err != nil {
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("Failed to encode messages response: %v", err)
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ThreadID     string `json:"thread_id"`
		FromType     string `json:"from_type"`
		FromID       string `json:"from_id"`
		ToType       string `json:"to_type"`
		ToID         string `json:"to_id"`
		Kind         string `json:"kind"`
		Content      string `json:"content"`
		MetadataJSON string `json:"metadata_json"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.ThreadID == "" || body.Content == "" {
		http.Error(w, "thread_id and content are required", http.StatusBadRequest)
		return
	}

	// Default values
	if body.FromType == "" {
		body.FromType = "human"
	}
	if body.FromID == "" {
		body.FromID = "user"
	}
	if body.ToType == "" {
		body.ToType = "ailang_instance"
	}
	if body.ToID == "" {
		body.ToID = "default"
	}
	if body.Kind == "" {
		body.Kind = "directive"
	}

	message, err := s.store.CreateMessage(
		body.ThreadID,
		body.FromType, body.FromID,
		body.ToType, body.ToID,
		body.Kind,
		body.Content,
		body.MetadataJSON,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create message: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast message to WebSocket subscribers
	s.wsServer.BroadcastMessage(body.ThreadID, message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(message); err != nil {
		log.Printf("Failed to encode message response: %v", err)
	}
}
