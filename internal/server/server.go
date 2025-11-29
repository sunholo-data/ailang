package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
)

//go:embed dist
var uiAssets embed.FS

// AgentProcess tracks a running agent process
type AgentProcess struct {
	InstanceID string    `json:"instance_id"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	cmd        *exec.Cmd
}

// Server represents the HTTP server for the collaboration hub
type Server struct {
	store    *messaging.Store
	wsServer *websocket.Server
	httpAddr string
	dbPath   string

	// Process management for spawned agents
	agentsMu sync.RWMutex
	agents   map[string]*AgentProcess

	// External telemetry from self-reporting processes (eval suite, etc.)
	externalTelemetryMu sync.RWMutex
	externalTelemetry   map[int]*websocket.TelemetryEvent // keyed by PID

	// Process history for recently completed/failed processes
	processHistoryMu sync.RWMutex
	processHistory   []ProcessStats // Recent history, oldest first

	// Track previously seen processes for history detection
	previouslySeenMu sync.RWMutex
	previouslySeen   map[int]ProcessStats // keyed by PID
}

// NewServer creates a new HTTP server
func NewServer(dbPath string, httpAddr string) (*Server, error) {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	wsServer := websocket.NewServer(store)

	return &Server{
		store:             store,
		wsServer:          wsServer,
		httpAddr:          httpAddr,
		dbPath:            dbPath,
		agents:            make(map[string]*AgentProcess),
		externalTelemetry: make(map[int]*websocket.TelemetryEvent),
		previouslySeen:    make(map[int]ProcessStats),
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
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentStop)
	mux.HandleFunc("/api/monitor", s.handleMonitor)
	mux.HandleFunc("/api/telemetry", s.handleTelemetryReport)
	mux.HandleFunc("/api/select-folder", s.handleSelectFolder)

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

// GET /api/agents - List known agent IDs and running agents
// POST /api/agents - Spawn a new agent process
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetAgents(w, r)
	case http.MethodPost:
		s.handleSpawnAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	// Get known agents from database
	knownAgents, err := s.store.GetKnownAgents()
	if err != nil {
		http.Error(w, "Failed to get agents", http.StatusInternalServerError)
		return
	}

	// Get running agents
	s.agentsMu.RLock()
	runningAgents := make([]*AgentProcess, 0, len(s.agents))
	for _, agent := range s.agents {
		runningAgents = append(runningAgents, agent)
	}
	s.agentsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"known":   knownAgents,
		"running": runningAgents,
	}); err != nil {
		log.Printf("Failed to encode agents response: %v", err)
	}
}

func (s *Server) handleSpawnAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	// Check if agent already running
	s.agentsMu.RLock()
	if _, exists := s.agents[body.InstanceID]; exists {
		s.agentsMu.RUnlock()
		http.Error(w, "Agent with this instance_id is already running", http.StatusConflict)
		return
	}
	s.agentsMu.RUnlock()

	// Spawn the agent process
	cmd := exec.Command("ailang-agent", "--instance-id", body.InstanceID, "--db", s.dbPath)

	// Create telemetry parser to capture and broadcast stdout telemetry
	// We don't know the PID yet, so use 0 and update after Start()
	telemetryParser := NewTelemetryParser(body.InstanceID, 0, s.wsServer)

	// Tee stdout to both os.Stdout (for logging) and telemetry parser
	cmd.Stdout = NewTeeWriter(os.Stdout, telemetryParser)
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to spawn agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Update telemetry parser with actual PID
	telemetryParser.pid = cmd.Process.Pid

	// Track the process
	agent := &AgentProcess{
		InstanceID: body.InstanceID,
		PID:        cmd.Process.Pid,
		StartedAt:  time.Now(),
		cmd:        cmd,
	}

	s.agentsMu.Lock()
	s.agents[body.InstanceID] = agent
	s.agentsMu.Unlock()

	// Start goroutine to clean up when process exits
	go func() {
		exitErr := cmd.Wait()
		if exitErr != nil {
			log.Printf("Agent %s (PID %d) exited with error: %v", body.InstanceID, agent.PID, exitErr)
		} else {
			log.Printf("Agent %s (PID %d) exited normally", body.InstanceID, agent.PID)
		}

		// Broadcast final telemetry update
		telemetryParser.MarkComplete(exitErr)

		s.agentsMu.Lock()
		delete(s.agents, body.InstanceID)
		s.agentsMu.Unlock()
	}()

	log.Printf("Spawned agent %s with PID %d", body.InstanceID, agent.PID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		log.Printf("Failed to encode agent response: %v", err)
	}
}

// DELETE /api/agents/{id} - Stop a running agent
func (s *Server) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract instance ID from path
	instanceID := r.URL.Path[len("/api/agents/"):]
	if instanceID == "" {
		http.Error(w, "Instance ID required", http.StatusBadRequest)
		return
	}

	// First, try to find in tracked agents (UI-spawned)
	s.agentsMu.Lock()
	agent, exists := s.agents[instanceID]
	if exists {
		// Kill the tracked process
		if err := agent.cmd.Process.Signal(os.Interrupt); err != nil {
			// Try harder with SIGKILL
			_ = agent.cmd.Process.Kill()
		}
		delete(s.agents, instanceID)
		s.agentsMu.Unlock()
		log.Printf("Stopped tracked agent %s (PID %d)", instanceID, agent.PID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Agent %s stopped", instanceID),
		})
		return
	}
	s.agentsMu.Unlock()

	// Not a tracked agent - try to extract PID from instance_id pattern
	// Patterns: agent_<pid>, eval_<pid>, run_<pid>, process_<pid>, external_<pid>
	var pid int
	for _, prefix := range []string{"agent_", "eval_", "run_", "process_", "external_"} {
		if len(instanceID) > len(prefix) && instanceID[:len(prefix)] == prefix {
			pidStr := instanceID[len(prefix):]
			// Handle patterns like eval_IMP001_abc12345 (take last numeric part)
			parts := splitLast(pidStr, "_")
			if p, err := strconv.Atoi(parts); err == nil {
				pid = p
				break
			}
		}
	}

	if pid == 0 {
		http.Error(w, "Agent not found or not running", http.StatusNotFound)
		return
	}

	// Kill the discovered process by PID
	proc, err := os.FindProcess(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Process %d not found: %v", pid, err), http.StatusNotFound)
		return
	}

	// Try SIGTERM first, then SIGKILL
	if err := proc.Signal(os.Interrupt); err != nil {
		if err := proc.Kill(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to kill process %d: %v", pid, err), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Stopped discovered process %s (PID %d)", instanceID, pid)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Process %s (PID %d) stopped", instanceID, pid),
	})
}

// splitLast returns the last segment after underscore, or the whole string if no underscore
func splitLast(s, sep string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return s[i+1:]
		}
	}
	return s
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.store.Close()
}

// handleSelectFolder opens a native folder picker dialog and returns the selected path
// GET /api/select-folder
func (s *Server) handleSelectFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use osascript on macOS to open a native folder picker
	// This displays a Finder dialog and returns the selected path
	script := `tell application "System Events"
		activate
		set folderPath to POSIX path of (choose folder with prompt "Select workspace directory")
		return folderPath
	end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// User cancelled or error occurred
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cancelled": true,
			"path":      "",
		})
		return
	}

	// Clean up the path (remove trailing newline)
	path := strings.TrimSpace(string(output))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cancelled": false,
		"path":      path,
	})
}
