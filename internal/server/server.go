package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
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
	version  string // AILANG version for display in UI

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

	// Coordinator integration for task metrics
	resourceRegistry *coordinator.ResourceTrackerRegistry
	coordStore       CoordinatorStore
}

// NewServer creates a new HTTP server
func NewServer(dbPath string, httpAddr string, opts ...ServerOption) (*Server, error) {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	wsServer := websocket.NewServer(store)

	s := &Server{
		store:             store,
		wsServer:          wsServer,
		httpAddr:          httpAddr,
		dbPath:            dbPath,
		version:           "dev", // Default version
		agents:            make(map[string]*AgentProcess),
		externalTelemetry: make(map[int]*websocket.TelemetryEvent),
		previouslySeen:    make(map[int]ProcessStats),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// ServerOption configures the server
type ServerOption func(*Server)

// WithVersion sets the AILANG version displayed in the UI
func WithVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}

// WithResourceRegistry sets the resource tracker registry for coordinator metrics
func WithResourceRegistry(registry *coordinator.ResourceTrackerRegistry) ServerOption {
	return func(s *Server) {
		s.resourceRegistry = registry
	}
}

// WithCoordinatorStore sets the coordinator store for task statistics
func WithCoordinatorStore(store CoordinatorStore) ServerOption {
	return func(s *Server) {
		s.coordStore = store
	}
}

// CoordinatorStore provides coordinator statistics
type CoordinatorStore interface {
	GetCoordinatorStats() (*CoordinatorStats, error)
}

// CoordinatorStats holds coordinator daemon statistics
type CoordinatorStats struct {
	Running      bool
	PID          int
	Uptime       string
	TasksRun     int
	PendingTasks int
	RunningTasks int
	FailedTasks  int
	TotalCost    float64
	TotalTokens  int
}

// Start starts the HTTP server and WebSocket server
func (s *Server) Start() error {
	// Start WebSocket event loop in background
	go s.wsServer.Run()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.wsServer.HandleWebSocket)

	// REST API endpoints - Threads
	mux.HandleFunc("/api/threads", s.handleThreads)
	mux.HandleFunc("/api/threads/", s.handleThread)

	// REST API endpoints - Messages
	mux.HandleFunc("/api/messages", s.handleMessages)

	// REST API endpoints - Approvals
	mux.HandleFunc("/api/approvals", s.handleApprovals)
	mux.HandleFunc("/api/approvals/", s.handleApproval)
	mux.HandleFunc("/api/approvals/history", s.handleApprovalHistory)

	// REST API endpoints - Agents
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentStop)

	// REST API endpoints - Monitoring & Metrics
	mux.HandleFunc("/api/monitor", s.handleMonitor)
	mux.HandleFunc("/api/telemetry", s.handleTelemetryReport)
	mux.HandleFunc("/api/hierarchy", s.handleHierarchy)
	mux.HandleFunc("/api/agent-stats/", s.handleAgentStats)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/metrics/", s.handleMetricsScope)
	mux.HandleFunc("/api/instances/history", s.handleInstanceHistory)
	mux.HandleFunc("/api/coordinator/status", s.handleCoordinatorStatus)

	// REST API endpoints - Inbox Messages (unified messaging)
	mux.HandleFunc("/api/inbox", s.handleInbox)
	mux.HandleFunc("/api/inbox/", s.handleInboxMessage)
	mux.HandleFunc("/api/inbox/ack-all", s.handleAckAll)
	mux.HandleFunc("/api/inbox/cleanup", s.handleInboxCleanup)

	// REST API endpoints - Utility
	mux.HandleFunc("/api/select-folder", s.handleSelectFolder)

	// Health check and version
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)

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

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.store.Close()
}
