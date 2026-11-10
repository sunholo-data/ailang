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
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/sunholo-data/ailang/internal/approvaltoken"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/platform/originpolicy"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/server/auth"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"github.com/sunholo-data/ailang/internal/websocket"
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

	// Secret-approval token auth (M-SECRET-EFFECT). When set, the approve/reject
	// endpoints accept a signed single-use ?token= (used by the iPhone ntfy
	// action buttons) as an alternative to dashboard/IAM auth. nil = disabled.
	secretTokenSigner *approvaltoken.Signer
	secretTokenGuard  *approvaltoken.SingleUseGuard

	// approvalPublisher publishes secret-approval notifications to the approvals
	// topic for the ntfy push bridge (M-SECRET-REMOTE-APPROVAL-WIRING). nil =
	// no push (local mode); the executor still learns the decision by polling.
	approvalPublisher ApprovalPublisher

	// idTokenValidate verifies approval callers' ID tokens
	// (approval_intake_auth.go). nil = idtoken.Validate; tests inject a fake.
	idTokenValidate idTokenValidator

	// Coordinator task event store for historical replay
	taskEventStore CoordinatorTaskEventStore

	// Observatory for telemetry, traces, and metrics visualization
	obsBackend observatory.Backend
	obsAPI     *observatory.API

	// Response cache for expensive polling endpoints (prevents 400% CPU from dashboard polls)
	pollingCache   *responseCache
	breakdownCache *responseCache // Longer TTL for expensive breakdown queries (M-PERF-OBSERVATORY)

	// Firebase authentication and authorization
	tokenVerifier    *auth.TokenVerifier
	accessControl    *auth.AccessControlCache
	workspaceService *auth.WorkspaceService

	// Ingest authentication (ingest_auth.go): hookToken is AILANG_HUB_TOKEN
	// (hook/exec/telemetry routes), otlpIngestToken AILANG_OTLP_INGEST_TOKEN
	// (/v1/*), ingestAuthMode AILANG_INGEST_AUTH.
	hookToken       string
	otlpIngestToken string
	ingestAuthMode  string
	ingestWarnings  *ingestWarnLog

	// apiKey is the dashboard's machine credential (COORDINATOR_API_KEY): the
	// bearer token that authenticates /api/ reads and the ?token= external
	// WebSocket clients present. read_auth.go.
	apiKey string

	// Cross-origin policy (origin.go): --cors-origin allowlist, built in NewServer
	// and shared by the HTTP middleware and the WebSocket hub.
	corsOrigins []string
	origins     *originpolicy.Policy

	// Pub/Sub event subscriber for cloud mode (bridges events to WebSocket).
	// pubsubEventSub is created in NewServer after wsServer is initialized.
	pubsubEventSub     *PubSubEventSubscriber
	pubsubSubscriber   *pubsub.Subscriber // set via option, used to create pubsubEventSub
	pubsubEventSubName string             // subscription base name (e.g., "events-dashboard")
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
		pollingCache:      newResponseCache(30 * time.Second),  // 30s TTL for dashboard polling endpoints (M-COST2: was 5s)
		breakdownCache:    newResponseCache(300 * time.Second), // 300s TTL for expensive breakdown queries (M-COST2: was 60s)
	}

	// Apply options (may set s.store and s.obsBackend via WithBackends)
	for _, opt := range opts {
		opt(s)
	}

	if err := originpolicy.Validate(false, s.corsOrigins); err != nil {
		return nil, err
	}
	s.origins = originpolicy.New(false, s.corsOrigins, hubAllowMethods)

	// If no messaging store was injected, open one from dbPath (local SQLite)
	if s.store == nil {
		store, err := messaging.OpenStore(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open store: %w", err)
		}
		s.store = store
	}

	s.wsServer = websocket.NewServer(s.store)
	s.wsServer.SetOriginPolicy(s.origins)
	if s.apiKey != "" {
		s.wsServer.SetToken(s.apiKey)
		log.Printf("WebSocket token authentication enabled for /ws and /ws/observatory")
	}

	// Create Pub/Sub event subscriber if configured (cloud mode).
	// Must happen after wsServer is created since it broadcasts to WebSocket clients.
	if s.pubsubSubscriber != nil && s.pubsubEventSubName != "" {
		s.pubsubEventSub = NewPubSubEventSubscriber(
			s.pubsubSubscriber,
			s.wsServer,
			s.pubsubEventSubName,
			log.Default(),
		)
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
	}
}

// WithHookToken sets a bearer token for authenticating hook requests.
// When set, requests to /api/hooks/* must include Authorization: Bearer <token>.
// When empty (default), no authentication is required (local mode).
func WithHookToken(token string) ServerOption {
	return func(s *Server) {
		s.hookToken = token
	}
}

// WithAPIKey sets the dashboard's machine credential (COORDINATOR_API_KEY).
// On a non-loopback bind every /api/ route outside the public allow-list, and
// the WebSocket, require it as `Authorization: Bearer <key>` (the WebSocket
// also takes ?token=<key>) unless the caller has a Firebase Viewer/Approver
// session. With neither configured those routes fail closed (read_auth.go).
func WithAPIKey(key string) ServerOption {
	return func(s *Server) {
		s.apiKey = key
	}
}

// WithPubSubEvents enables Pub/Sub event streaming for cloud mode.
// The subscriber and subName are stored; the actual PubSubEventSubscriber
// is created in NewServer after the WebSocket server is initialized.
func WithPubSubEvents(subscriber *pubsub.Subscriber, subName string) ServerOption {
	return func(s *Server) {
		s.pubsubSubscriber = subscriber
		s.pubsubEventSubName = subName
	}
}

// WithApprovalPublisher wires the approvals-topic publisher used by the secret
// approval ntfy push bridge (M-SECRET-REMOTE-APPROVAL-WIRING).
func WithApprovalPublisher(pub ApprovalPublisher) ServerOption {
	return func(s *Server) {
		s.approvalPublisher = pub
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
// The observatory is local-only; the GCP Trace / composite remote backends were
// deleted in M-V1-SIMPLIFY-S1 M4 (nothing ever set AILANG_ENABLE_GCP_TRACE).
func WithObservatoryDB(dbPath string) ServerOption {
	return func(s *Server) {
		sqliteBackend, err := observatory.NewSQLiteBackendFromPath(dbPath)
		if err != nil {
			log.Printf("Warning: Failed to initialize observatory: %v", err)
			return
		}
		log.Printf("Observatory: Local-only mode")

		s.obsBackend = sqliteBackend
		s.obsAPI = observatory.NewAPI(sqliteBackend)
	}
}

// WithFirebaseAuth initializes Firebase authentication and Firestore-based access control.
// The projectID should match your Firebase/GCP project (e.g., "ailang-dev").
// Requires GOOGLE_APPLICATION_CREDENTIALS environment variable to be set with service account key.
//
// If initialization fails, the server continues without Firebase sessions: on a
// non-loopback bind the /api/ reads then accept only the COORDINATOR_API_KEY
// bearer, and fail closed with neither (read_auth.go).
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
			log.Printf("Dashboard will run without Firebase sessions")
			return
		}

		// Get Firebase Auth client
		authClient, err := app.Auth(ctx)
		if err != nil {
			log.Printf("Warning: Failed to get Firebase Auth client: %v", err)
			log.Printf("Dashboard will run without Firebase sessions")
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

	// Start Pub/Sub event subscriber if configured (cloud mode).
	// Bridges ailang-events topic → WebSocket broadcast.
	if s.pubsubEventSub != nil {
		go s.pubsubEventSub.Start(context.Background())
		log.Printf("Pub/Sub event streaming: ENABLED")
	}

	// Log auth status
	if s.tokenVerifier != nil {
		log.Printf("Firebase authentication: ENABLED")
		log.Printf("  - Approval routes require Approver role")
	} else {
		log.Printf("Firebase authentication: DISABLED (API reads take the COORDINATOR_API_KEY bearer; see read_auth.go)")
	}

	// Routes behind the read-auth gate (read_auth.go) and the origin policy.
	handler, _, err := s.httpHandler()
	if err != nil {
		return err
	}

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

// Close closes the server and releases resources
func (s *Server) Close() error {
	if s.pubsubEventSub != nil {
		s.pubsubEventSub.Stop()
	}

	if s.obsBackend != nil {
		if err := s.obsBackend.Close(); err != nil {
			log.Printf("Warning: Failed to close observatory backend: %v", err)
		}
	}
	return s.store.Close()
}
