package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
)

//go:embed dist
var uiAssets embed.FS

// Server represents the HTTP server for the collaboration hub
type Server struct {
	store    *messaging.Store
	wsServer *websocket.Server
	httpAddr string
	dbPath   string
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

	// Serve static UI files from embedded dist folder
	distFS, err := fs.Sub(uiAssets, "dist")
	if err != nil {
		return fmt.Errorf("failed to load UI assets: %w", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("/", fileServer)

	// CORS middleware
	handler := s.corsMiddleware(mux)

	log.Printf("Starting HTTP server on %s", s.httpAddr)
	log.Printf("WebSocket endpoint: ws://%s/ws", s.httpAddr)
	log.Printf("REST API: http://%s/api/", s.httpAddr)
	log.Printf("UI: http://%s/", s.httpAddr)

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
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"connections": s.wsServer.GetConnectionCount(),
		"timestamp":   time.Now().Unix(),
	}); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// GET /api/threads - List all threads
// POST /api/threads - Create a new thread
func (s *Server) handleThreads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetThreads(w, r)
	case http.MethodPost:
		s.handleCreateThread(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetThreads(w http.ResponseWriter, r *http.Request) {
	// Get all threads with status filter
	status := r.URL.Query().Get("status")

	threads, err := s.store.GetThreadsByStatus(status, 100)
	if err != nil {
		http.Error(w, "Failed to get threads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(threads); err != nil {
		log.Printf("Failed to encode threads response: %v", err)
	}
}

func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title         string `json:"title"`
		CreatedByType string `json:"created_by_type"`
		CreatedByID   string `json:"created_by_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	// Default values
	if body.CreatedByType == "" {
		body.CreatedByType = "human"
	}
	if body.CreatedByID == "" {
		body.CreatedByID = "user"
	}

	thread, err := s.store.CreateThread(body.Title, body.CreatedByType, body.CreatedByID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create thread: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(thread); err != nil {
		log.Printf("Failed to encode thread response: %v", err)
	}
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
	if err := json.NewEncoder(w).Encode(thread); err != nil {
		log.Printf("Failed to encode thread response: %v", err)
	}
}

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
		ThreadID string `json:"thread_id"`
		FromType string `json:"from_type"`
		FromID   string `json:"from_id"`
		ToType   string `json:"to_type"`
		ToID     string `json:"to_id"`
		Kind     string `json:"kind"`
		Content  string `json:"content"`
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
	if err := json.NewEncoder(w).Encode(approvals); err != nil {
		log.Printf("Failed to encode approvals response: %v", err)
	}
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
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"action":  action,
		"message": fmt.Sprintf("Approval %s successfully", action+"d"),
	}); err != nil {
		log.Printf("Failed to encode approval response: %v", err)
	}
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.store.Close()
}
