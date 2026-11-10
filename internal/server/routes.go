package server

import (
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
	return s.corsMiddleware(s.readAuthMiddleware(mux.ServeMux)), mux.patterns, nil
}

// registerRoutes registers every route the dashboard serves. The dashboard is
// a headless API (design_docs/planned/m-dashboard-headless-api.md): every
// route below has a named consumer outside this package, listed in that doc's
// route table. Add a route only together with its consumer, its row in
// read_auth_test.go and its contract test (contract_test.go).
func (s *Server) registerRoutes(mux router) error {
	// WebSocket: the website builder's portal proxies it for live task events
	// (ailang-demos website_builder/portal/server.js, DASHBOARD_URL/ws?token=).
	// Fed by POST /api/coordinator/events, /api/telemetry and Pub/Sub.
	mux.HandleFunc("/ws", s.wsServer.HandleWebSocket)

	// Public benchmark data (M-EVAL-DATA-HOSTING-DECOUPLE) — read-through from the
	// private GCS bucket so the docs site fetches it at runtime, no site rebuild.
	mux.HandleFunc("/benchmarks/", s.handleBenchmarks)

	// Secret approvals: intake + status poll (cloud jobs via AILANG_APPROVAL_URL)
	// and the ntfy Approve/Deny decision (handlers_approvals.go).
	mux.HandleFunc("/api/approvals", s.handleApprovals)
	mux.HandleFunc("/api/approvals/", s.handleApproval)

	// Machine ingest: eval/pipeline self-reports, coordinator task events, and
	// exec sessions (claude_telemetry.sh, the resident executor).
	mux.HandleFunc("/api/telemetry", s.handleTelemetryReport)
	mux.HandleFunc("/api/coordinator/events", s.handleCoordinatorTaskEvents)
	mux.HandleFunc("/api/exec/sessions", s.handleExecSessions)
	mux.HandleFunc("/api/exec/events", s.handleExecEvents)

	// Reads behind the CLI: `ailang chains dashboard inbox|hierarchy|stats` and
	// the coordinator task browser (GET /api/coordinator/tasks/{id}/events).
	mux.HandleFunc("/api/inbox", s.handleInbox)
	mux.HandleFunc("/api/coordinator/tasks/", s.handleCoordinatorTaskEvents_)
	// Expensive reads wrapped in a response cache.
	mux.HandleFunc("/api/controlplane/exec-hierarchy", s.pollingCache.withCache(s.handleControlPlaneExecHierarchy))
	mux.HandleFunc("/api/controlplane/stats/breakdown", s.breakdownCache.withCache(s.handleControlPlaneStatsBreakdown))

	// Observatory API endpoints (if configured)
	if s.obsAPI != nil {
		s.obsAPI.RegisterRoutes(mux)
		log.Printf("Observatory API registered at /api/observatory/*")
	}
	if s.obsBackend != nil {
		// Claude Code telemetry: command hooks and the native HTTP hook protocol.
		mux.HandleFunc("/api/observatory/hooks", s.handleObservatoryHooks)
		mux.HandleFunc("/api/hooks/claude", s.handleClaudeHooks)
		log.Printf("Claude hook endpoints registered at /api/observatory/hooks, /api/hooks/claude")
		// Execution chains API endpoints (M-CHAINS-SIMPLIFY)
		s.registerChainRoutes(mux)
		log.Printf("Chains API registered at /api/chains/*")
		// OTLP receiver for standard trace/log/metrics ingestion
		otlpReceiver := observatory.NewOTLPReceiver(s.obsBackend)
		otlpReceiver.RegisterRoutes(mux)
		log.Printf("OTLP receiver registered at /v1/traces, /v1/logs, /v1/metrics")
	}

	// Health check and version
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)

	// No UI: the dashboard is a headless API (m-dashboard-headless-api). "/"
	// is the mux's catch-all, so it answers every path no other route claims:
	// a one-line pointer at "/" itself and a plain 404 for everything else —
	// no SPA fallback that would turn a mistyped API path into a 200.
	mux.HandleFunc("/", handleRoot)
	return nil
}

// rootBanner is what GET / answers. Plain text: nothing here is for a browser.
const rootBanner = "AILANG dashboard API (headless). Health: /health\n"

// handleRoot answers "/" and every path no other route is registered for.
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found\n"))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_, _ = w.Write([]byte(rootBanner))
}
