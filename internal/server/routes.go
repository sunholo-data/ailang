package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// router is the part of *http.ServeMux that route registration uses. Every
// route the dashboard serves is registered through it, so recordingMux can
// list them for the read-auth table test (read_auth_test.go): a route that is
// registered but not classified there fails the build's tests.
type router interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// recordingMux is an http.ServeMux that remembers every pattern registered on it.
type recordingMux struct {
	*http.ServeMux
	patterns []string
}

func newRecordingMux() *recordingMux { return &recordingMux{ServeMux: http.NewServeMux()} }

func (m *recordingMux) Handle(pattern string, handler http.Handler) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.Handle(pattern, handler)
}

func (m *recordingMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.HandleFunc(pattern, handler)
}

// httpHandler builds the server's full HTTP handler — every route, behind the
// read-auth gate (read_auth.go), behind the origin policy (origin.go) — and
// returns the registered patterns alongside it.
//
// Order matters: the origin policy runs first so a refused 401 still carries
// the CORS headers an allowed cross-origin caller needs to read it, and an
// allowed preflight is answered before the gate (a preflight carries no
// credentials by design).
func (s *Server) httpHandler() (http.Handler, []string, error) {
	mux := newRecordingMux()
	if err := s.registerRoutes(mux); err != nil {
		return nil, nil, err
	}
	return s.corsMiddleware(s.ingestAuthMiddleware(mux.ServeMux, s.readAuthMiddleware(mux.ServeMux))), mux.patterns, nil
}

// registerRoutes registers every route the dashboard serves.
func (s *Server) registerRoutes(mux router) error {
	// WebSocket endpoint. /ws/observatory is the SAME hub: the observatory
	// hub it used to name (internal/observatory/websocket.go) had no producer
	// anywhere in the tree — nothing ever called its Broadcast* methods — so
	// the control-plane UI's socket only ever carried a heartbeat. One hub,
	// one origin/token check (M-V1-SIMPLIFY-S3 M5).
	mux.HandleFunc("/ws", s.wsServer.HandleWebSocket)
	mux.HandleFunc("/ws/observatory", s.wsServer.HandleWebSocket)

	// Public benchmark data (M-EVAL-DATA-HOSTING-DECOUPLE) — read-through from the
	// private GCS bucket so the docs site fetches it at runtime, no site rebuild.
	mux.HandleFunc("/benchmarks/", s.handleBenchmarks)

	// REST API endpoints - Threads
	mux.HandleFunc("/api/threads", s.handleThreads)
	mux.HandleFunc("/api/threads/", s.handleThread)
	// Workspaces endpoint uses OptionalAuth to get user role when authenticated
	mux.Handle("/api/workspaces", s.OptionalAuthMiddleware(http.HandlerFunc(s.handleWorkspaces)))
	mux.HandleFunc("/api/statistics", s.handleStatistics)

	// REST API endpoints - Messages
	// Thread messages. Was /api/messages, which the coordinator daemon ALSO
	// serves — as inbox messages, with a different body and auth. Two
	// servers, one path, two meanings; the thread one moves because it has
	// no external callers (the UI never used it, the coordinator's is a
	// documented REST contract). M-V1-SIMPLIFY-S3 M5.
	mux.HandleFunc("/api/thread-messages", s.handleMessages)

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
	// Claude Code HTTP hooks endpoint (native hook protocol, replaces command hooks)
	if s.obsBackend != nil {
		mux.HandleFunc("/api/hooks/claude", s.handleClaudeHooks)
		log.Printf("Claude hooks endpoint registered at /api/hooks/claude")
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
	return nil
}
