/*
Package server implements `ailang serve`: the dashboard, a headless HTTP API.

There is no UI (design_docs/planned/m-dashboard-headless-api.md). The server
ingests telemetry, serves the reads the CLI and scripts make, takes secret
approvals, and serves the docs site's benchmark data. That design doc holds the
route table — every route, its auth class and the client that calls it — and
routes.go registers exactly those routes.

  - Ingest: OTLP /v1/{traces,logs,metrics}; /api/hooks/claude and
    /api/observatory/hooks (Claude Code); /api/exec/{sessions,events};
    /api/coordinator/events; /api/telemetry.
  - Reads (gated on a non-loopback bind, read_auth.go): /api/observatory/*
    (internal/observatory/api.go), /api/chains/*, /api/inbox,
    /api/controlplane/{exec-hierarchy,stats/breakdown},
    /api/coordinator/tasks/{id}/events, and the /ws event stream.
  - Secret approvals: /api/approvals, /api/approvals/{id}[/approve|/reject].
  - Public: /health, /api/version, /benchmarks/*, and "/" (a one-line banner).

Stores: the observatory backend (SQLite observatory.db locally, Firestore in
the cloud), the messaging store (collaboration.db), and the coordinator store
(coordinator.db) — the same Go store methods the CLI uses directly.
*/
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
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

// Server represents the dashboard's HTTP server
type Server struct {
	store    messaging.MessageStore
	wsServer *websocket.Server
	httpAddr string
	dbPath   string
	version  string // AILANG version reported by /health and /api/version

	// Coordinator store: task events, exec sessions, chains
	coordStoreRaw coordinator.Store

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
		httpAddr:       httpAddr,
		dbPath:         dbPath,
		version:        "dev",                               // Default version
		pollingCache:   newResponseCache(30 * time.Second),  // 30s TTL for dashboard polling endpoints (M-COST2: was 5s)
		breakdownCache: newResponseCache(300 * time.Second), // 300s TTL for expensive breakdown queries (M-COST2: was 60s)
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
		log.Printf("WebSocket token authentication enabled for /ws")
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

		// Initialize workspace service with Firestore
		s.workspaceService = auth.NewWorkspaceService(fsClient)

		log.Printf("Firebase Auth initialized for project: %s", projectID)
		log.Printf("  - Token verification: enabled")
		log.Printf("  - Firestore access control: enabled")
		log.Printf("  - Workspace access control: enabled")
	}
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
			otelhttp.WithFilter(func(r *http.Request) bool { return shouldTraceRequest(r.URL.Path) }),
		)
	}

	log.Printf("Starting HTTP server on %s", s.httpAddr)
	log.Printf("WebSocket endpoint: ws://%s/ws", s.httpAddr)
	log.Printf("REST API: http://%s/api/", s.httpAddr)
	log.Printf("Health: http://%s/health (no UI: headless API)", s.httpAddr)

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

// shouldTraceRequest decides which requests the server's own OpenTelemetry
// instrumentation records. Skipped: health and the banner, the WebSocket,
// OTLP ingest (tracing it would loop), the high-frequency polling paths
// (M-DB-CLEANUP: these were 97% of ailang-server span noise), the observatory
// API (tracing our own trace reads is confusing) and static-file probes. The
// same rules as before the UI was removed, minus the paths of deleted routes.
func shouldTraceRequest(path string) bool {
	switch path {
	case "/", "/health", "/ws", "/api/approvals", "/api/version", "/api/inbox", "/api/coordinator/events":
		return false
	}
	for _, prefix := range []string{"/v1/", "/api/observatory/", "/api/controlplane/"} {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	for _, suffix := range []string{".png", ".ico", ".svg", ".woff2"} {
		if strings.HasSuffix(path, suffix) {
			return false
		}
	}
	return true
}
