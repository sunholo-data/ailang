package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
)

// Server represents the HTTP server for the collaboration hub
type Server struct {
	store     *messaging.Store
	wsServer  *websocket.Server
	httpAddr  string
	dbPath    string
}

// NewServer creates a new HTTP server
func NewServer(dbPath string, httpAddr string) (*Server, error) {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	wsServer := websocket.NewServer(store)

	return &Server{
		store:    store,
		wsServer: wsServer,
		httpAddr: httpAddr,
		dbPath:   dbPath,
	}, nil
}

// Start starts the HTTP server and WebSocket server
func (s *Server) Start() error {
	// Start WebSocket event loop in background
	go s.wsServer.Run()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.wsServer.HandleWebSocket)

	// REST API endpoints
	mux.HandleFunc("/api/threads", s.handleThreads)
	mux.HandleFunc("/api/threads/", s.handleThread)
	mux.HandleFunc("/api/messages", s.handleMessages)
	mux.HandleFunc("/api/approvals", s.handleApprovals)
	mux.HandleFunc("/api/approvals/", s.handleApproval)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// CORS middleware
	handler := s.corsMiddleware(mux)

	log.Printf("Starting HTTP server on %s", s.httpAddr)
	log.Printf("WebSocket endpoint: ws://%s/ws", s.httpAddr)
	log.Printf("REST API: http://%s/api/", s.httpAddr)

	return http.ListenAndServe(s.httpAddr, handler)
}

// CORS middleware to allow cross-origin requests from the UI
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"connections": s.wsServer.GetConnectionCount(),
		"timestamp":   time.Now().Unix(),
	})
}

// GET /api/threads - List all threads
func (s *Server) handleThreads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement pagination and filtering
	// For now, return empty array (frontend uses mock data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// GET /api/threads/{id} - Get a specific thread
func (s *Server) handleThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract thread ID from path
	threadID := r.URL.Path[len("/api/threads/"):]
	if threadID == "" {
		http.Error(w, "Thread ID required", http.StatusBadRequest)
		return
	}

	thread, err := s.store.GetThread(threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thread)
}

// GET /api/messages?thread_id={id} - Get messages for a thread
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
	json.NewEncoder(w).Encode(messages)
}

// GET /api/approvals?status={status} - Get approvals by status
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	approvals, err := s.store.GetApprovalsByStatus(status, 50)
	if err != nil {
		http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}

// POST /api/approvals/{id}/approve - Approve an approval request
// POST /api/approvals/{id}/reject - Reject an approval request
func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract approval ID and action from path
	path := r.URL.Path[len("/api/approvals/"):]
	var approvalID, action string

	// Parse path: {id}/approve or {id}/reject
	for i, ch := range path {
		if ch == '/' {
			approvalID = path[:i]
			action = path[i+1:]
			break
		}
	}

	if approvalID == "" || (action != "approve" && action != "reject") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Parse request body for review notes
	var body struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform action
	var err error
	if action == "approve" {
		err = s.store.ApproveApproval(approvalID, "user", body.Notes, 24*time.Hour)
	} else {
		err = s.store.RejectApproval(approvalID, "user", body.Notes)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to %s approval: %v", action, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"action":  action,
		"message": fmt.Sprintf("Approval %s successfully", action+"d"),
	})
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.store.Close()
}
