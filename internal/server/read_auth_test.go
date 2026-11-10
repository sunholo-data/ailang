package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
	"github.com/sunholo-data/ailang/internal/server/auth"
)

// The dashboard's route table, classified. This is the regression guard for
// read_auth.go: every pattern the server registers must appear in exactly one
// of these two tables, and a route added later fails
// TestReadAuth_EveryRouteClassified until someone decides which.
//
// publicRoutes maps a pattern to the methods that answer WITHOUT the dashboard
// credential ("*" = every method: the path is outside /api/ and /ws).
// Everything else a public pattern serves, and every method of every
// gatedRoutes pattern, must be refused.
var publicRoutes = map[string][]string{
	"/":                {"*"}, // one-line banner; 404 for any unrouted path
	"/health":          {"*"},
	"/benchmarks/":     {"*"}, // docs-site data, intentionally public
	"POST /v1/traces":  {"*"}, // OTLP ingest (own optional AILANG_OTLP_INGEST_TOKEN)
	"POST /v1/logs":    {"*"},
	"POST /v1/metrics": {"*"},

	"/api/version":            {http.MethodGet},
	"/api/hooks/claude":       {http.MethodPost},
	"/api/observatory/hooks":  {http.MethodPost},
	"/api/exec/sessions":      {http.MethodPost},
	"/api/exec/events":        {http.MethodPost},
	"/api/coordinator/events": {http.MethodPost},
	"/api/telemetry":          {http.MethodPost},
	"/api/approvals":          {http.MethodPost}, // intake: own ID-token / AILANG_APPROVAL_TOKEN check
	"/api/approvals/":         {http.MethodGet},  // status poll by id: same own check (decisions: see below)
}

var gatedRoutes = []string{
	"/ws", "/ws/observatory",
	"/api/threads", "/api/threads/", "/api/workspaces", "/api/statistics", "/api/thread-messages",
	"/api/approvals/history",
	"/api/agents", "/api/agents/",
	"/api/monitor", "/api/hierarchy", "/api/agent-stats/", "/api/metrics", "/api/metrics/", "/api/instances/history",
	"/api/coordinator/status", "/api/coordinator/pending", "/api/coordinator/approve/", "/api/coordinator/reject/",
	"/api/coordinator/tasks/", "/api/coordinator/running", "/api/tasks/",
	"/api/exec/sessions/",
	"/api/inbox", "/api/inbox/", "/api/inbox/ack-all", "/api/inbox/cleanup",
	"/api/select-folder", "/api/telemetry/config",
	"/api/controlplane/heatmap", "/api/controlplane/topology/observed", "/api/controlplane/topology",
	"/api/controlplane/stats", "/api/controlplane/stats/breakdown", "/api/controlplane/exec-hierarchy",
	"/api/controlplane/span-hierarchy", "/api/controlplane/task-hierarchy", "/api/controlplane/task-evolution",
	"/api/controlplane/usage-timeseries", "/api/controlplane/token-distribution", "/api/controlplane/outliers",
	"/api/budget/status", "/api/budget/check",
	"/api/claude-history/projects", "/api/claude-history/sessions", "/api/claude-history/session/",
	"/api/claude-history/by-span/", "/api/claude-history/search", "/api/claude-history/sync",
	"/api/claude-history/sync-status", "/api/claude-history/db/session/", "/api/claude-history/db/by-span/",
	"GET /api/observatory/workspaces", "POST /api/observatory/workspaces", "GET /api/observatory/workspaces/{id}",
	"PUT /api/observatory/workspaces/{id}", "DELETE /api/observatory/workspaces/{id}", "GET /api/observatory/workspaces/{id}/stats",
	"GET /api/observatory/tasks", "POST /api/observatory/tasks", "GET /api/observatory/tasks/{id}",
	"PUT /api/observatory/tasks/{id}", "DELETE /api/observatory/tasks/{id}", "GET /api/observatory/tasks/{taskId}/agents",
	"POST /api/observatory/agents", "GET /api/observatory/agents/{id}", "PUT /api/observatory/agents/{id}",
	"DELETE /api/observatory/agents/{id}", "GET /api/observatory/agents/{id}/stats",
	"GET /api/observatory/spans", "GET /api/observatory/spans/enriched", "POST /api/observatory/spans",
	"GET /api/observatory/spans/{id}", "PUT /api/observatory/spans/{id}", "DELETE /api/observatory/spans/{id}",
	"GET /api/observatory/spans/{id}/events", "POST /api/observatory/spans/{id}/events", "GET /api/observatory/spans/{id}/enriched",
	"GET /api/observatory/traces", "GET /api/observatory/traces/{id}",
	"GET /api/observatory/messages", "POST /api/observatory/messages", "GET /api/observatory/messages/{id}",
	"PUT /api/observatory/messages/{id}", "DELETE /api/observatory/messages/{id}",
	"POST /api/observatory/messages/{id}/read", "POST /api/observatory/messages/{id}/archive",
	"GET /api/observatory/metrics/summary", "GET /api/observatory/metrics/providers",
	"GET /api/observatory/tasks/{id}/timeline", "GET /api/observatory/tasks/{id}/hierarchy",
	"GET /api/observatory/sessions", "GET /api/observatory/sessions/{id}", "GET /api/observatory/sessions/{id}/tools",
	"GET /api/observatory/sessions/{id}/tools/summary", "GET /api/observatory/sessions/{id}/metrics",
	"GET /api/observatory/telemetry/metrics",
	"POST /api/observatory/ingest/claude", "POST /api/observatory/ingest/otel",
	"/api/chains", "/api/chains/stats", "/api/chains/active", "/api/chains/pending", "/api/chains/by-message/",
	"/api/chains/by-task/", "/api/chains/by-github/", "/api/spans/", "/api/chains/",
	"/api/auth/whoami",
}

const testAPIKey = "test-dashboard-key"

// fullRouteServer builds a server with every optional route family registered
// (observatory, chains, hooks, OTLP, whoami), bound to addr.
func fullRouteServer(t *testing.T, addr string, opts ...ServerOption) *Server {
	t.Helper()
	dir := t.TempDir()
	opts = append([]ServerOption{WithObservatoryDB(filepath.Join(dir, "obs.db"))}, opts...)
	s, err := NewServer(filepath.Join(dir, "collab.db"), addr, opts...)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func registeredPatterns(t *testing.T, s *Server) []string {
	t.Helper()
	_, patterns, err := s.httpHandler()
	if err != nil {
		t.Fatalf("httpHandler: %v", err)
	}
	return patterns
}

// probe is one request against one registered pattern.
type probe struct {
	pattern, method, target string
	wantPublic              bool
}

var wildcard = regexp.MustCompile(`\{[^}]*\}`)

// probesFor turns each classified pattern into concrete requests: the
// pattern's own method (or every common method when it has none), with
// wildcards and trailing-slash subtrees filled in.
func probesFor(patterns []string) []probe {
	var out []probe
	for _, p := range patterns {
		method, path := "", p
		if i := strings.Index(p, " "); i > 0 {
			method, path = p[:i], p[i+1:]
		}
		path = wildcard.ReplaceAllString(path, "probe-id")
		if strings.HasSuffix(path, "/") && path != "/" {
			path += "probe-id"
		}
		methods := []string{method}
		if method == "" {
			methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
		}
		for _, m := range methods {
			out = append(out, probe{pattern: p, method: m, target: path, wantPublic: isWantPublic(p, m)})
		}
	}
	return out
}

func isWantPublic(pattern, method string) bool {
	for _, m := range publicRoutes[pattern] {
		if m == "*" || m == method {
			return true
		}
	}
	return false
}

// stubbedGate wraps a mux holding the SAME patterns as the real server, each
// answering 200 "reached", in the real gate. It lets every route be probed
// without running handlers that touch GCS, ~/.claude or spawn processes.
func stubbedGate(s *Server, patterns []string) http.Handler {
	stub := http.NewServeMux()
	for _, p := range patterns {
		stub.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("reached"))
		})
	}
	return s.readAuthMiddleware(stub)
}

func serve(h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		r.Header[k] = v
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func bearer(tok string) http.Header {
	return http.Header{"Authorization": {"Bearer " + tok}}
}

// Every registered pattern is classified, and every classified pattern is
// still registered (so the tables cannot rot into allowing a dead name).
func TestReadAuth_EveryRouteClassified(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	s.tokenVerifier = auth.NewTokenVerifier(nil) // registers /api/auth/whoami
	patterns := registeredPatterns(t, s)

	classified := map[string]bool{}
	for p := range publicRoutes {
		classified[p] = true
	}
	for _, p := range gatedRoutes {
		if classified[p] {
			t.Errorf("%q is in both publicRoutes and gatedRoutes", p)
		}
		classified[p] = true
	}

	registered := map[string]bool{}
	for _, p := range patterns {
		registered[p] = true
		if !classified[p] {
			t.Errorf("route %q is registered but not classified: add it to gatedRoutes (default) or, if it is ingest or carries its own auth, to publicRoutes AND publicAPIRoute in read_auth.go", p)
		}
	}
	var stale []string
	for p := range classified {
		if !registered[p] {
			stale = append(stale, p)
		}
	}
	sort.Strings(stale)
	for _, p := range stale {
		t.Errorf("classified route %q is no longer registered: remove it from the table", p)
	}
}

// Without credentials, every gated route is refused 401 by the gate and every
// public one is reached.
func TestReadAuth_RouteTable_NoCredentials(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	s.tokenVerifier = auth.NewTokenVerifier(nil)
	patterns := registeredPatterns(t, s)
	h := stubbedGate(s, patterns)

	for _, pr := range probesFor(patterns) {
		w := serve(h, pr.method, pr.target, nil)
		if pr.wantPublic {
			if w.Code != http.StatusOK || w.Body.String() != "reached" {
				t.Errorf("%s %s (pattern %q) is public but got %d %q", pr.method, pr.target, pr.pattern, w.Code, w.Body.String())
			}
			continue
		}
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != dashboardAuthRealm {
			t.Errorf("%s %s (pattern %q) must be refused by the dashboard gate without credentials; got %d %q",
				pr.method, pr.target, pr.pattern, w.Code, w.Body.String())
		}
	}
}

// The same through the server's REAL handler chain (httpHandler): removing
// readAuthMiddleware from httpHandler fails this test. It probes only reads
// that are safe to execute if the gate is missing (temp-dir stores) — the
// full route table is covered against the gate itself above, because some
// handlers have host side effects (/api/select-folder opens a native folder
// picker on the server's desktop).
func TestReadAuth_EndToEnd_GatedRoutesRefuseAnonymous(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}
	// The ones that leaked in prod, by name, plus the WebSocket.
	for _, target := range []string{
		"/api/inbox", "/api/approvals", "/api/approvals/history", "/api/chains", "/api/chains/stats",
		"/api/observatory/traces", "/api/observatory/spans", "/api/observatory/messages", "/api/threads",
		"/api/controlplane/stats", "/api/workspaces", "/api/statistics", "/ws",
	} {
		w := serve(h, http.MethodGet, target, nil)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != dashboardAuthRealm {
			t.Errorf("GET %s: anonymous caller got %d, want the gate's 401", target, w.Code)
		}
	}
	// And the credential gets through the real chain.
	if w := serve(h, http.MethodGet, "/api/observatory/spans", bearer(testAPIKey)); w.Code != http.StatusOK {
		t.Errorf("GET /api/observatory/spans with the API key: got %d, want 200", w.Code)
	}
}

func TestReadAuth_ValidAPIKeyPasses(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	patterns := registeredPatterns(t, s)
	h := stubbedGate(s, patterns)

	for _, pr := range probesFor(patterns) {
		w := serve(h, pr.method, pr.target, bearer(testAPIKey))
		if w.Code != http.StatusOK || w.Body.String() != "reached" {
			t.Errorf("%s %s with the API key: got %d %q, want reached", pr.method, pr.target, w.Code, w.Body.String())
		}
	}
	// The WebSocket's documented external-client contract: ?token=.
	for _, ws := range []string{"/ws", "/ws/observatory"} {
		if w := serve(h, http.MethodGet, ws+"?token="+testAPIKey, nil); w.Body.String() != "reached" {
			t.Errorf("%s?token=<key>: got %d, want reached", ws, w.Code)
		}
	}
}

func TestReadAuth_BadCredentialsRefused(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	h := stubbedGate(s, registeredPatterns(t, s))

	cases := []struct {
		name   string
		target string
		header http.Header
	}{
		{"wrong key", "/api/inbox", bearer("not-the-key")},
		{"key as prefix", "/api/inbox", bearer(testAPIKey + "x")},
		{"non-bearer scheme", "/api/inbox", http.Header{"Authorization": {"Basic " + testAPIKey}}},
		{"REST ignores ?token=", "/api/inbox?token=" + testAPIKey, nil},
		{"wrong ws token", "/ws?token=nope", nil},
		{"traversal into /api", "/benchmarks/../api/inbox", nil},
		{"double slash", "//api/inbox", nil},
	}
	for _, tc := range cases {
		w := serve(h, http.MethodGet, tc.target, tc.header)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: GET %s got %d %q, want 401", tc.name, tc.target, w.Code, w.Body.String())
		}
	}
}

// Fail closed: a non-loopback server with no credential configured refuses
// the gated routes with 503 and still serves the public ones.
func TestReadAuth_FailsClosedWithoutConfig(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0")
	patterns := registeredPatterns(t, s)
	h := stubbedGate(s, patterns)

	for _, pr := range probesFor(patterns) {
		w := serve(h, pr.method, pr.target, bearer("anything"))
		if pr.wantPublic {
			if w.Body.String() != "reached" {
				t.Errorf("%s %s is public but got %d", pr.method, pr.target, w.Code)
			}
		} else if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s with no auth configured: got %d, want 503", pr.method, pr.target, w.Code)
		}
	}
}

// A loopback-bound server (the rig's `ailang serve`) is exempt; any other
// bind is not, configured or not.
func TestReadAuth_LoopbackExemption(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:1957", "127.0.0.2:1957", "[::1]:1957", "localhost:1957"} {
		s := fullRouteServer(t, addr) // no credential configured at all
		h := stubbedGate(s, registeredPatterns(t, s))
		for _, target := range []string{"/api/inbox", "/api/approvals", "/ws", "GET /api/observatory/spans"} {
			target = strings.TrimPrefix(target, "GET ")
			if w := serve(h, http.MethodGet, target, nil); w.Body.String() != "reached" {
				t.Errorf("bind %s: GET %s got %d, want reached (loopback is exempt)", addr, target, w.Code)
			}
		}
	}
	for _, addr := range []string{"0.0.0.0:1957", ":1957", "[::]:1957", "10.1.2.3:1957", "dashboard.example.com:443"} {
		s := fullRouteServer(t, addr, WithAPIKey(testAPIKey))
		h := stubbedGate(s, registeredPatterns(t, s))
		if w := serve(h, http.MethodGet, "/api/inbox", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("bind %s: anonymous GET /api/inbox got %d, want 401", addr, w.Code)
		}
	}
}

// Secret-approval decisions: the signed ntfy ?token= reaches the handler
// (which verifies it); without one — or with token auth disabled — the
// decision needs the dashboard credential. The list and history never pass.
func TestReadAuth_ApprovalRoutes(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	patterns := registeredPatterns(t, s)

	h := stubbedGate(s, patterns) // token auth disabled
	if w := serve(h, http.MethodPost, "/api/approvals/appr-1/approve?token=x", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("token auth off: POST approve?token= got %d, want 401", w.Code)
	}

	signer, err := approvaltoken.NewSigner([]byte("signing-key"))
	if err != nil {
		t.Fatal(err)
	}
	s.SetSecretApprovalAuth(signer)
	h = stubbedGate(s, patterns)

	cases := []struct {
		method, target string
		reached        bool
	}{
		{http.MethodPost, "/api/approvals/appr-1/approve?token=x", true}, // handler verifies x
		{http.MethodPost, "/api/approvals/appr-1/reject?token=x", true},
		{http.MethodPost, "/api/approvals/appr-1/approve", false},
		{http.MethodGet, "/api/approvals/appr-1", true}, // poll: own intake auth
		{http.MethodPost, "/api/approvals", true},       // intake: own auth
		{http.MethodGet, "/api/approvals", false},       // the list
		{http.MethodGet, "/api/approvals/history", false},
		{http.MethodGet, "/api/approvals/history?token=x", false},
		{http.MethodGet, "/api/approvals/appr-1/approve", false},
	}
	for _, tc := range cases {
		w := serve(h, tc.method, tc.target, nil)
		if got := w.Body.String() == "reached"; got != tc.reached {
			t.Errorf("%s %s: reached=%t (status %d), want %t", tc.method, tc.target, got, w.Code, tc.reached)
		}
	}
}

// A signed-in Firebase user is checked by role; this covers the refusal path
// that does not need a live Firebase project.
func TestReadAuth_FirebaseOnlyServerRefusesAnonymous(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0")
	s.tokenVerifier = auth.NewTokenVerifier(nil)
	h := stubbedGate(s, registeredPatterns(t, s))
	w := serve(h, http.MethodGet, "/api/inbox", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Firebase-only server, anonymous GET /api/inbox: got %d, want 401", w.Code)
	}
}

func TestIsLoopbackBind(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:1957": true, "127.9.9.9:80": true, "[::1]:1957": true, "localhost:1957": true, "LOCALHOST:1": true,
		"0.0.0.0:1957": false, ":1957": false, "[::]:1957": false, "192.168.1.5:1957": false,
		"example.com:443": false, "127.0.0.1": false, "": false,
	}
	for addr, want := range cases {
		if got := isLoopbackBind(addr); got != want {
			t.Errorf("isLoopbackBind(%q) = %t, want %t", addr, got, want)
		}
	}
}
