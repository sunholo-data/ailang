package server

// Authentication for the dashboard's data routes — DEFAULT DENY.
//
// The prod dashboard is a public Cloud Run service (allUsers invoker), and
// until this gate every /api/ route answered anonymous callers: the unified
// inbox (message bodies), approvals, chains, observatory traces and spans,
// Claude conversation history. Anonymous reads were observed in prod logs.
//
// The rule is now: on a non-loopback bind, every path under /api/ and the
// WebSocket (/ws, /ws/observatory) require a credential, EXCEPT the explicit
// allow-list in publicAPIRoute — machine ingest (hooks, exec sessions,
// coordinator events, eval telemetry), routes that carry their own auth
// (secret-approval intake/poll, the signed ntfy approve/deny link) and
// /api/version. A route added later is gated until someone classifies it, and
// TestReadAuth_EveryRouteClassified fails until they do.
//
// Paths outside /api/ and /ws stay public: the static UI, /health,
// /benchmarks/* (docs-site data) and OTLP ingest /v1/* (its own optional
// AILANG_OTLP_INGEST_TOKEN, observatory/otlp_auth.go).
//
// Credentials accepted, and nothing else:
//
//  1. `Authorization: Bearer <COORDINATOR_API_KEY>` — the dashboard's existing
//     machine credential (already provisioned on the Cloud Run service for the
//     WebSocket), compared in constant time. The WebSocket also takes
//     ?token=<key>, its documented external-client contract (browsers cannot
//     set headers on a WebSocket).
//  2. A Firebase ID token whose holder has the Viewer or Approver role, when
//     --firebase-project is configured. A signed-in Guest gets 403.
//
// Fail closed: on a non-loopback bind with neither configured, gated routes
// return 503 — never fall open.
//
// Loopback exemption: a server bound to 127.0.0.0/8, ::1 or localhost (the
// default `ailang serve` on the rig) skips the gate — only local processes can
// reach it. Cloud Run binds 0.0.0.0 (PORT set), which is never exempt.

import (
	"context"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
)

// dashboardAuthRealm is sent in WWW-Authenticate on the gate's 401s. Its realm
// differs from the approval intake's ("ailang-approvals") so a client — and
// the route table test — can tell which check refused it.
const dashboardAuthRealm = `Bearer realm="ailang-dashboard"`

// readAuthMiddleware gates mux's data routes. It looks up the pattern a
// request will be routed to, so the allow-list is keyed by registered pattern
// (the same strings the route table test enumerates), not by a path guess.
func (s *Server) readAuthMiddleware(mux *http.ServeMux) http.Handler {
	if isLoopbackBind(s.httpAddr) {
		log.Printf("Dashboard API auth: EXEMPT (bound to loopback %s)", s.httpAddr)
		return mux
	}
	switch {
	case s.apiKey == "" && s.tokenVerifier == nil:
		log.Printf("SECURITY: dashboard API auth is NOT configured on non-loopback bind %s — every gated /api/ route and /ws will return 503 until COORDINATOR_API_KEY or --firebase-project is set", s.httpAddr)
	default:
		log.Printf("Dashboard API auth: ENFORCED on %s (COORDINATOR_API_KEY bearer: %t, Firebase Viewer/Approver: %t)",
			s.httpAddr, s.apiKey != "", s.tokenVerifier != nil)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !needsDashboardAuth(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}
		if _, pattern := mux.Handler(r); s.publicAPIRoute(pattern, r) {
			mux.ServeHTTP(w, r)
			return
		}
		r, ok := s.authorizeDashboardRequest(w, r)
		if !ok {
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// needsDashboardAuth reports whether a path is in the gated namespace: /api
// and /ws, judged on the cleaned path so "/x/../api/inbox" is not a way round.
func needsDashboardAuth(p string) bool {
	if p == "" {
		return false
	}
	c := path.Clean("/" + p)
	return c == "/api" || strings.HasPrefix(c, "/api/") || c == "/ws" || strings.HasPrefix(c, "/ws/")
}

// publicAPIRoute is the allow-list: the gated-namespace routes that answer
// without the dashboard credential. pattern is what the mux will route r to.
// Every entry is ingest (writes in, a status out — no stored data back) or
// carries its own authentication.
func (s *Server) publicAPIRoute(pattern string, r *http.Request) bool {
	switch pattern {
	case "/api/version":
		return r.Method == http.MethodGet || r.Method == http.MethodHead

	// Machine ingest. Each handler is POST-only and answers {"status":"ok"} or
	// the id it created.
	case "/api/hooks/claude", // Claude Code HTTP hooks; optional AILANG_HUB_TOKEN
		"/api/observatory/hooks",  // command hooks + resident executor sessions
		"/api/exec/sessions",      // exec session start (claude_telemetry.sh, resident)
		"/api/exec/events",        // exec transcript events
		"/api/coordinator/events", // coordinator http_broadcaster
		"/api/telemetry":          // eval-suite self-reports
		return r.Method == http.MethodPost

	// Secret-approval intake: its own Google ID token / AILANG_APPROVAL_TOKEN
	// check, fail closed (approval_intake_auth.go). GET (the list) is gated.
	case "/api/approvals":
		return r.Method == http.MethodPost

	case "/api/approvals/":
		rest := strings.TrimPrefix(r.URL.Path, "/api/approvals/")
		switch r.Method {
		case http.MethodGet:
			// Status poll by the executor that filed the request — same own
			// auth as the intake (handleSecretApprovalStatus).
			return rest != "" && !strings.Contains(rest, "/")
		case http.MethodPost:
			// The iPhone ntfy Approve/Deny link: a signed single-use ?token=
			// the handler verifies for this exact id + action, and rejects
			// rather than falling through when invalid. Without the token (or
			// with token auth off) the decision needs the dashboard credential.
			return s.secretTokenSigner != nil && r.URL.Query().Get("token") != ""
		}
	}
	return false
}

// authorizeDashboardRequest checks the caller's credential. On success it
// returns the request (with the Firebase user in context when that is how
// they authenticated); otherwise it has written 401/403/503.
func (s *Server) authorizeDashboardRequest(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	if s.apiKey == "" && s.tokenVerifier == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "auth_not_configured",
			"Dashboard API authentication is not configured on this server (set COORDINATOR_API_KEY or --firebase-project)")
		return nil, false
	}

	token := presentedDashboardToken(r)
	if token == "" {
		refuseDashboardRequest(w, r, "missing_token", "Authentication required: send Authorization: Bearer <COORDINATOR_API_KEY> (CLI readers: set AILANG_DASHBOARD_TOKEN)")
		return nil, false
	}
	if s.apiKey != "" && constantTimeEqual(token, s.apiKey) {
		return r, true
	}
	if s.tokenVerifier != nil {
		if user, err := s.firebaseUser(r, token); err == nil {
			if !user.IsViewer() && !user.IsApprover() {
				log.Printf("dashboard auth: refused %s %s: Firebase user has role %q, needs Viewer or Approver", r.Method, r.URL.Path, user.Role)
				writeErrorResponse(w, http.StatusForbidden, "forbidden", "This dashboard requires the Viewer or Approver role")
				return nil, false
			}
			return r.WithContext(context.WithValue(r.Context(), UserContextKey, user)), true
		}
	}
	refuseDashboardRequest(w, r, "invalid_token", "Invalid or expired token")
	return nil, false
}

func refuseDashboardRequest(w http.ResponseWriter, r *http.Request, code, msg string) {
	log.Printf("dashboard auth: refused %s %s from %s: %s", r.Method, r.URL.Path, r.RemoteAddr, code)
	w.Header().Set("WWW-Authenticate", dashboardAuthRealm)
	writeErrorResponse(w, http.StatusUnauthorized, code, msg)
}

// presentedDashboardToken is the caller's bearer token. The WebSocket also
// accepts ?token= (browsers cannot set headers on one); REST routes do not,
// because a token in a URL lands in access logs.
func presentedDashboardToken(r *http.Request) string {
	if tok := strings.TrimSpace(extractBearerToken(r)); tok != "" {
		return tok
	}
	if c := path.Clean("/" + r.URL.Path); c == "/ws" || strings.HasPrefix(c, "/ws/") {
		return r.URL.Query().Get("token")
	}
	return ""
}

// isLoopbackBind reports whether the server's listen address only accepts
// local connections. An empty host (":1957") listens on every interface.
func isLoopbackBind(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
