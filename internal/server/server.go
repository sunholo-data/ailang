/*
Package server implements the HTTP server for the Collaboration Hub dashboard.

# API Namespace Reference

The dashboard exposes several API namespaces:

## /api/observatory/* (Core Telemetry - 44 endpoints)

Primary telemetry data: spans, traces, tasks, sessions, messages.
Source: internal/observatory/api.go

Key endpoints:
  - GET  /api/observatory/spans           - List spans with filters
  - GET  /api/observatory/spans/enriched  - Spans with display names from session_tools
  - GET  /api/observatory/traces          - List traces
  - GET  /api/observatory/sessions        - List Claude Code sessions
  - GET  /api/observatory/sessions/{id}/tools - Session tool usage

## /api/controlplane/* (Aggregated Analytics - 6 endpoints)

Unified stats combining Observatory + Coordinator data.
Source: internal/server/handlers_controlplane.go

Key endpoints:
  - GET  /api/controlplane/stats          - Unified statistics with filters (USE THIS)
  - GET  /api/controlplane/stats/breakdown - Detailed breakdown by dimension
  - GET  /api/controlplane/exec-hierarchy - Execution tree (Messages→Execs→Turns→Tools)
  - GET  /api/controlplane/heatmap        - Activity heatmap

## /api/coordinator/* (Task Execution - ~8 endpoints)

Task management and approval workflow.
Source: internal/server/handlers_coordinator.go

Key endpoints:
  - POST /api/coordinator/approve/{id}    - Approve a task
  - POST /api/coordinator/reject/{id}     - Reject a task
  - GET  /api/coordinator/events          - SSE stream of task events

## Legacy Endpoints (Deprecated)

  - GET  /api/statistics                  - DEPRECATED: Use /api/controlplane/stats
  - GET  /api/observatory/metrics/summary - DEPRECATED: Use /api/controlplane/stats

# Data Source Architecture

All endpoints pull from the same underlying data stores:

	┌─────────────────────────────────────────┐
	│           SQLite Databases              │
	├─────────────────────────────────────────┤
	│ ~/.ailang/state/collaboration.db       │
	│ ├── spans (OTEL telemetry)             │
	│ ├── traces, tasks, sessions            │
	│ └── session_tools, messages            │
	├─────────────────────────────────────────┤
	│ ~/.ailang/state/coordinator.db         │
	│ └── tasks, approvals, agent_state      │
	└─────────────────────────────────────────┘
	              │
	              ▼
	   Same Go store methods used by:
	   - API handlers (this package)
	   - CLI commands (cmd/ailang/dashboard.go)
	   - UI hooks (ui/src/features/controlplane/hooks/)
*/
package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/server/auth"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/websocket"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	store    messaging.MessageStore
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
	coordStoreRaw    coordinator.Store // Full store interface for Control Plane queries

	// Coordinator approval store for approval workflow
	approvalStore CoordinatorApprovalStore

	// Coordinator task event store for historical replay
	taskEventStore CoordinatorTaskEventStore

	// Observatory for telemetry, traces, and metrics visualization
	obsBackend observatory.Backend
	obsAPI     *observatory.API
	obsHub     *observatory.Hub

	// Response cache for expensive polling endpoints (prevents 400% CPU from dashboard polls)
	pollingCache   *responseCache
	breakdownCache *responseCache // Longer TTL for expensive breakdown queries (M-PERF-OBSERVATORY)

	// Firebase authentication and authorization
	tokenVerifier    *auth.TokenVerifier
	accessControl    *auth.AccessControlCache
	workspaceService *auth.WorkspaceService
}

// NewServer creates a new HTTP server.
// If a messaging store was set via WithMessagingStore, dbPath is only used for display.
func NewServer(dbPath string, httpAddr string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		httpAddr:          httpAddr,
		dbPath:            dbPath,
		version:           "dev", // Default version
		agents:            make(map[string]*AgentProcess),
		externalTelemetry: make(map[int]*websocket.TelemetryEvent),
		previouslySeen:    make(map[int]ProcessStats),
		pollingCache:      newResponseCache(5 * time.Second),  // 5s TTL for dashboard polling endpoints
		breakdownCache:    newResponseCache(60 * time.Second), // 60s TTL for expensive breakdown queries
	}

	// Apply options (may set s.store and s.obsBackend via WithBackends)
	for _, opt := range opts {
		opt(s)
	}

	// If no messaging store was injected, open one from dbPath (local SQLite)
	if s.store == nil {
		store, err := messaging.OpenStore(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open store: %w", err)
		}
		s.store = store
	}

	s.wsServer = websocket.NewServer(s.store)

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

// WithMessagingStore sets a pre-created messaging store, skipping local SQLite open.
// Use this when running with cloud backends (AILANG_STORAGE=gcp).
func WithMessagingStore(store messaging.MessageStore) ServerOption {
	return func(s *Server) {
		s.store = store
	}
}

// WithObservatoryBackend sets a pre-created observatory backend.
// Use this instead of WithObservatoryDB when running with cloud backends.
func WithObservatoryBackend(backend observatory.Backend) ServerOption {
	return func(s *Server) {
		s.obsBackend = backend
		s.obsAPI = observatory.NewAPI(backend)
		s.obsHub = observatory.NewHub()
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

// WithObservatoryDB sets up the observatory backend with SQLite at the given path.
// If GCP project is configured (via GOOGLE_CLOUD_PROJECT or OTLP_GOOGLE_CLOUD_PROJECT),
// it also adds a GCP Trace remote backend for federated trace queries.
func WithObservatoryDB(dbPath string) ServerOption {
	return func(s *Server) {
		// Create local SQLite backend
		sqliteBackend, err := observatory.NewSQLiteBackendFromPath(dbPath)
		if err != nil {
			log.Printf("Warning: Failed to initialize observatory: %v", err)
			return
		}

		// Check for GCP project configuration
		gcpProject := getGCPProject()
		var backend observatory.Backend

		// GCP Trace federation disabled until M-GEMINI-TRACE investigation is complete
		// See: design_docs/planned/v0_6_4/m-gemini-trace-investigation.md
		// Issue: Gemini CLI exports to Cloud Logging, not Cloud Trace
		if gcpProject != "" && getEnv("AILANG_ENABLE_GCP_TRACE") == "1" {
			// Create GCP Trace remote backend
			gcpBackend, err := observatory.NewGCPTraceBackend(observatory.GCPConfig{
				ProjectID: gcpProject,
			})
			if err != nil {
				log.Printf("Warning: Failed to initialize GCP Trace backend (will use local only): %v", err)
				backend = sqliteBackend
			} else {
				// Create composite backend with local + GCP remote
				compositeBackend, err := observatory.NewCompositeBackend(observatory.CompositeConfig{
					Local:   sqliteBackend,
					Remotes: []observatory.Backend{gcpBackend},
				})
				if err != nil {
					log.Printf("Warning: Failed to create composite backend: %v", err)
					backend = sqliteBackend
				} else {
					backend = compositeBackend
					log.Printf("Observatory: Composite backend enabled (local + GCP Trace project=%s)", gcpProject)
				}
			}
		} else {
			backend = sqliteBackend
			if gcpProject != "" {
				log.Printf("Observatory: Local-only mode (GCP Trace disabled, set AILANG_ENABLE_GCP_TRACE=1 to enable)")
			} else {
				log.Printf("Observatory: Local-only mode")
			}
		}

		s.obsBackend = backend
		s.obsAPI = observatory.NewAPI(backend)
		s.obsHub = observatory.NewHub()
	}
}

// getGCPProject returns the GCP project ID from environment variables.
func getGCPProject() string {
	// Check OTLP-specific variable first (for dual-export scenarios)
	if project := getEnv("OTLP_GOOGLE_CLOUD_PROJECT"); project != "" {
		return project
	}
	// Fall back to standard GCP variable
	return getEnv("GOOGLE_CLOUD_PROJECT")
}

// getEnv returns an environment variable value.
func getEnv(key string) string {
	return os.Getenv(key)
}

// WithFirebaseAuth initializes Firebase authentication and Firestore-based access control.
// The projectID should match your Firebase/GCP project (e.g., "ailang-dev").
// Requires GOOGLE_APPLICATION_CREDENTIALS environment variable to be set with service account key.
//
// If initialization fails, the server will continue without authentication (all routes public).
// Check logs for "Firebase Auth initialized" to confirm successful setup.
func WithFirebaseAuth(projectID string) ServerOption {
	return func(s *Server) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Initialize Firebase app
		config := &firebase.Config{ProjectID: projectID}
		app, err := firebase.NewApp(ctx, config)
		if err != nil {
			log.Printf("Warning: Failed to initialize Firebase app: %v", err)
			log.Printf("Dashboard will run without authentication (all routes public)")
			return
		}

		// Get Firebase Auth client
		authClient, err := app.Auth(ctx)
		if err != nil {
			log.Printf("Warning: Failed to get Firebase Auth client: %v", err)
			log.Printf("Dashboard will run without authentication (all routes public)")
			return
		}

		// Get Firestore client
		fsClient, err := app.Firestore(ctx)
		if err != nil {
			log.Printf("Warning: Failed to get Firestore client: %v", err)
			log.Printf("Dashboard will run with token verification only (no role-based access)")
			s.tokenVerifier = auth.NewTokenVerifier(authClient)
			return
		}

		// Initialize both verifier and access control
		s.tokenVerifier = auth.NewTokenVerifier(authClient)
		s.accessControl = auth.NewAccessControlCache(fsClient)

		// Initialize workspace service with Firestore and config
		workspacesConfig := coordinator.LoadWorkspacesConfig()
		s.workspaceService = auth.NewWorkspaceService(fsClient, workspacesConfig)

		log.Printf("Firebase Auth initialized for project: %s", projectID)
		log.Printf("  - Token verification: enabled")
		log.Printf("  - Firestore access control: enabled")
		log.Printf("  - Workspace access control: enabled")
	}
}

// CoordinatorStore provides coordinator statistics
type CoordinatorStore interface {
	GetCoordinatorStats() (*CoordinatorStats, error)
	GetCostByProvider() (map[string]float64, error)
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

// requireApprover wraps a handler to require Approver role if auth is enabled.
// If Firebase auth is not configured, passes through without auth check.
func (s *Server) requireApprover(handler http.HandlerFunc) http.Handler {
	if s.tokenVerifier == nil {
		// No auth configured - pass through
		return handler
	}
	// Chain: AuthMiddleware -> RequireApprover -> handler
	return s.AuthMiddleware(RequireApprover(handler))
}

// Start starts the HTTP server and WebSocket server
func (s *Server) Start() error {
	// Start WebSocket event loop in background
	go s.wsServer.Run()

	// Log auth status
	if s.tokenVerifier != nil {
		log.Printf("Firebase authentication: ENABLED")
		log.Printf("  - Approval routes require Approver role")
	} else {
		log.Printf("Firebase authentication: DISABLED (all routes public)")
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.wsServer.HandleWebSocket)

	// REST API endpoints - Threads
	mux.HandleFunc("/api/threads", s.handleThreads)
	mux.HandleFunc("/api/threads/", s.handleThread)
	// Workspaces endpoint uses OptionalAuth to get user role when authenticated
	mux.Handle("/api/workspaces", s.OptionalAuthMiddleware(http.HandlerFunc(s.handleWorkspaces)))
	mux.HandleFunc("/api/statistics", s.handleStatistics)

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
	mux.HandleFunc("/api/coordinator/pending", s.handleCoordinatorPendingApprovals)
	// Protected routes - require Approver role when auth is enabled
	mux.Handle("/api/coordinator/approve/", s.requireApprover(s.handleCoordinatorApproval))
	mux.Handle("/api/coordinator/reject/", s.requireApprover(s.handleCoordinatorApproval))
	mux.HandleFunc("/api/coordinator/events", s.handleCoordinatorTaskEvents)
	mux.HandleFunc("/api/coordinator/tasks/", s.handleCoordinatorTaskEvents_)
	mux.HandleFunc("/api/coordinator/running", s.handleCoordinatorRunningTasks)

	// REST API endpoints - Tasks (cleaner alias)
	// GET /api/tasks/{id}/events - same as /api/coordinator/tasks/{id}/events
	mux.HandleFunc("/api/tasks/", s.handleTasksAlias)

	// REST API endpoints - Exec Sessions (for Claude Code hooks and ailang exec)
	mux.HandleFunc("/api/exec/sessions", s.handleExecSessions)
	mux.HandleFunc("/api/exec/sessions/", s.handleExecSessionEvents)
	mux.HandleFunc("/api/exec/events", s.handleExecEvents)

	// REST API endpoints - Inbox Messages (unified messaging)
	mux.HandleFunc("/api/inbox", s.handleInbox)
	mux.HandleFunc("/api/inbox/", s.handleInboxMessage)
	mux.HandleFunc("/api/inbox/ack-all", s.handleAckAll)
	mux.HandleFunc("/api/inbox/cleanup", s.handleInboxCleanup)

	// REST API endpoints - Utility
	mux.HandleFunc("/api/select-folder", s.handleSelectFolder)
	mux.HandleFunc("/api/telemetry/config", s.handleTelemetryConfig)

	// REST API endpoints - Control Plane
	// Expensive endpoints wrapped with 5s response cache to prevent CPU burn from dashboard polling
	mux.HandleFunc("/api/controlplane/heatmap", s.pollingCache.withCache(s.handleControlPlaneHeatmap))
	mux.HandleFunc("/api/controlplane/topology/observed", s.handleControlPlaneTopologyObserved)
	mux.HandleFunc("/api/controlplane/topology", s.handleControlPlaneTopology)
	mux.HandleFunc("/api/controlplane/stats", s.pollingCache.withCache(s.handleControlPlaneStats))
	mux.HandleFunc("/api/controlplane/stats/breakdown", s.breakdownCache.withCache(s.handleControlPlaneStatsBreakdown))
	mux.HandleFunc("/api/controlplane/exec-hierarchy", s.pollingCache.withCache(s.handleControlPlaneExecHierarchy))
	mux.HandleFunc("/api/controlplane/span-hierarchy", s.pollingCache.withCache(s.handleSpanHierarchy))
	mux.HandleFunc("/api/controlplane/task-hierarchy", s.pollingCache.withCache(s.handleTaskHierarchy))

	// REST API endpoints - Analytics (Phase 2+)
	mux.HandleFunc("/api/controlplane/task-evolution", s.handleTaskEvolution)
	mux.HandleFunc("/api/controlplane/usage-timeseries", s.handleUsageTimeSeries)
	mux.HandleFunc("/api/controlplane/token-distribution", s.handleTokenDistribution)
	mux.HandleFunc("/api/controlplane/outliers", s.handleOutliersAnalysis)

	// REST API endpoints - Budget (AILANG dogfooding)
	mux.HandleFunc("/api/budget/status", s.handleBudgetStatus)
	mux.HandleFunc("/api/budget/check", s.handleBudgetCheck)

	// REST API endpoints - Claude Code Conversation History (JSONL-based)
	mux.HandleFunc("/api/claude-history/projects", s.handleClaudeHistoryProjects)
	mux.HandleFunc("/api/claude-history/sessions", s.handleClaudeHistorySessions)
	mux.HandleFunc("/api/claude-history/session/", s.handleClaudeHistorySession)
	mux.HandleFunc("/api/claude-history/by-span/", s.handleClaudeHistoryBySpan)
	mux.HandleFunc("/api/claude-history/search", s.handleClaudeHistorySearch)

	// REST API endpoints - Claude Code Conversation History (DB-backed, M-CHAT-HISTORY-DB)
	mux.HandleFunc("/api/claude-history/sync", s.handleClaudeHistorySync)
	mux.HandleFunc("/api/claude-history/sync-status", s.handleClaudeHistorySyncStatus)
	mux.HandleFunc("/api/claude-history/db/session/", s.handleClaudeHistoryDBSession)
	mux.HandleFunc("/api/claude-history/db/by-span/", s.handleClaudeHistoryDBBySpan)

	// Observatory API endpoints (if configured)
	if s.obsAPI != nil {
		s.obsAPI.RegisterRoutes(mux)
		log.Printf("Observatory API registered at /api/observatory/*")
	}
	// Observatory hooks endpoint for Claude Code telemetry
	if s.obsBackend != nil {
		mux.HandleFunc("/api/observatory/hooks", s.handleObservatoryHooks)
		log.Printf("Observatory hooks endpoint registered at /api/observatory/hooks")
	}
	// Execution chains API endpoints (M-CHAINS-SIMPLIFY)
	if s.obsBackend != nil {
		s.registerChainRoutes(mux)
		log.Printf("Chains API registered at /api/chains/*")
	}
	// OTLP receiver for standard trace/log/metrics ingestion
	if s.obsBackend != nil {
		otlpReceiver := observatory.NewOTLPReceiver(s.obsBackend)
		otlpReceiver.RegisterRoutes(mux)
		log.Printf("OTLP receiver registered at /v1/traces, /v1/logs, /v1/metrics")
	}
	if s.obsHub != nil {
		go s.obsHub.Run()
		mux.HandleFunc("/ws/observatory", s.obsHub.HandleWebSocket)
		log.Printf("Observatory WebSocket registered at /ws/observatory")
	}

	// Health check and version
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)

	// Auth endpoint (returns user info if authenticated)
	if s.tokenVerifier != nil {
		mux.Handle("/api/auth/whoami", s.AuthMiddleware(http.HandlerFunc(WhoAmIHandler)))
	}

	// Serve static UI files from embedded dist folder
	distFS, err := fs.Sub(uiAssets, "dist")
	if err != nil {
		return fmt.Errorf("failed to load UI assets: %w", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("/", fileServer)

	// CORS middleware
	handler := s.corsMiddleware(mux)

	// OpenTelemetry HTTP instrumentation (if enabled via OTEL_EXPORTER_OTLP_ENDPOINT)
	if telemetry.IsEnabled() {
		log.Printf("OpenTelemetry instrumentation enabled")
		handler = otelhttp.NewHandler(handler, "ailang-server",
			otelhttp.WithFilter(func(r *http.Request) bool {
				path := r.URL.Path
				// Skip tracing for:
				// - Health checks
				// - WebSocket endpoints
				// - Static assets (JS, CSS, images, fonts)
				// - Root path (just serves UI)
				// - OTLP receiver endpoints (would cause infinite loop)
				// - High-frequency polling endpoints (create noise)
				// - Observatory API (tracing our own traces is confusing)
				if path == "/health" || path == "/" {
					return false
				}
				if path == "/ws" || path == "/ws/observatory" {
					return false
				}
				if strings.HasPrefix(path, "/assets/") {
					return false
				}
				if strings.HasPrefix(path, "/v1/") { // OTLP endpoints
					return false
				}
				// Skip high-frequency polling endpoints (UI polls these constantly)
				// M-DB-CLEANUP: These endpoints generate 97% of ailang-server span noise
				if path == "/api/approvals" || path == "/api/hierarchy" ||
					path == "/api/statistics" || path == "/api/version" ||
					path == "/api/monitor" || path == "/api/telemetry/config" ||
					path == "/api/inbox" {
					return false
				}
				// Skip control plane polling endpoints (dashboard polls these every few seconds)
				// M-DB-CLEANUP: /api/controlplane/* generated 140K+ spans
				if strings.HasPrefix(path, "/api/controlplane/") {
					return false
				}
				// Skip coordinator events SSE endpoint (long-lived connection, creates noise)
				if path == "/api/coordinator/events" {
					return false
				}
				// Skip budget status polling
				if strings.HasPrefix(path, "/api/budget/") {
					return false
				}
				// Skip observatory API (tracing our own traces would be confusing)
				if strings.HasPrefix(path, "/api/observatory/") {
					return false
				}
				// Skip metrics polling
				if strings.HasPrefix(path, "/api/metrics") {
					return false
				}
				// Skip common static files
				if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".ico") ||
					strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".woff2") {
					return false
				}
				return true
			}),
		)
	}

	log.Printf("Starting HTTP server on %s", s.httpAddr)
	log.Printf("WebSocket endpoint: ws://%s/ws", s.httpAddr)
	log.Printf("REST API: http://%s/api/", s.httpAddr)
	log.Printf("UI: http://%s/", s.httpAddr)

	return http.ListenAndServe(s.httpAddr, handler)
}

// CORS middleware to allow cross-origin requests from the UI
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log ALL POST requests to help debug Gemini telemetry
		if r.Method == "POST" {
			log.Printf("POST request: %s (Content-Type: %s, UA: %s)", r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"))
		}
		// Debug logging for OTLP paths or any POST to non-API paths
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			log.Printf("OTLP request: %s %s (Content-Type: %s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		// Catch any other telemetry paths that might be used
		if strings.Contains(r.URL.Path, "trace") || strings.Contains(r.URL.Path, "log") || strings.Contains(r.URL.Path, "metric") {
			log.Printf("Potential telemetry request: %s %s (Content-Type: %s, UA: %s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"))
		}

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
	if s.obsHub != nil {
		s.obsHub.Stop()
	}
	if s.obsBackend != nil {
		if err := s.obsBackend.Close(); err != nil {
			log.Printf("Warning: Failed to close observatory backend: %v", err)
		}
	}
	return s.store.Close()
}
