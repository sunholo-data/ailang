package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	gorillaws "github.com/gorilla/websocket"
)

const (
	hubHost    = "127.0.0.1:1957"
	hubOrigin  = "http://127.0.0.1:1957"
	evilOrigin = "https://evil.example"
	viteOrigin = "http://localhost:3000"
)

// hubRequest sends one request through the server's origin middleware and
// reports the response and whether the wrapped handler ran.
func hubRequest(t *testing.T, s *Server, method, path, origin string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	ran := false
	h := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(method, "http://"+hubHost+path, nil)
	r.Host = hubHost
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, ran
}

func newOriginTestServer(t *testing.T, opts ...ServerOption) *Server {
	t.Helper()
	s, err := NewServer(filepath.Join(t.TempDir(), "collab.db"), hubHost, opts...)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// The hub used to answer every route with Access-Control-Allow-Origin: * —
// any page in the operator's browser could read the coordinator, messages
// and observatory APIs, and drive state-changing POSTs.
func TestHubOrigin_ForeignRefusedByDefault(t *testing.T) {
	s := newOriginTestServer(t)
	for _, path := range []string{"/api/inbox", "/api/coordinator/approve/x", "/health"} {
		w, _ := hubRequest(t, s, http.MethodGet, path, evilOrigin)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("GET %s from foreign origin: ACAO=%q, want none", path, got)
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		w, ran := hubRequest(t, s, m, "/api/coordinator/approve/x", evilOrigin)
		if ran || w.Code != http.StatusForbidden {
			t.Errorf("%s from foreign origin: ran=%v status=%d, want 403 before dispatch", m, ran, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s from foreign origin: ACAO=%q, want none", m, got)
		}
	}
}

func TestHubOrigin_SameOriginAndCLIPass(t *testing.T) {
	s := newOriginTestServer(t)
	for _, origin := range []string{hubOrigin, ""} {
		w, ran := hubRequest(t, s, http.MethodPost, "/api/coordinator/approve/x", origin)
		if !ran || w.Code != http.StatusOK {
			t.Errorf("POST with Origin %q: ran=%v status=%d, want 200", origin, ran, w.Code)
		}
	}
}

func TestHubOrigin_AllowlistedOriginGranted(t *testing.T) {
	s := newOriginTestServer(t, WithCORSOrigins([]string{viteOrigin}))
	w, ran := hubRequest(t, s, http.MethodOptions, "/api/messages", viteOrigin)
	if ran || w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != viteOrigin {
		t.Fatalf("preflight from listed origin: ran=%v status=%d ACAO=%q", ran, w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	w, ran = hubRequest(t, s, http.MethodPost, "/api/messages", viteOrigin)
	if !ran || w.Header().Get("Access-Control-Allow-Origin") != viteOrigin {
		t.Fatalf("POST from listed origin: ran=%v ACAO=%q", ran, w.Header().Get("Access-Control-Allow-Origin"))
	}
	// The allowlist reaches the WebSocket hub too (one policy, both paths).
	go s.wsServer.Run()
	ts := httptest.NewServer(http.HandlerFunc(s.wsServer.HandleWebSocket))
	defer ts.Close()
	for origin, want := range map[string]int{viteOrigin: http.StatusSwitchingProtocols, evilOrigin: http.StatusForbidden} {
		conn, resp, _ := gorillaws.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), http.Header{"Origin": {origin}})
		if conn != nil {
			conn.Close()
		}
		if resp == nil || resp.StatusCode != want {
			t.Errorf("WS upgrade from %s: resp=%v, want status %d", origin, resp, want)
		}
	}
}

// /benchmarks/* is public read-only data the docs site (GitHub Pages)
// fetches cross-origin at runtime (docs/src/lib/benchmarkFetch.js). It keeps
// ACAO: * for reads, and nothing else is opened by that exception.
func TestHubOrigin_BenchmarksStayPublic(t *testing.T) {
	s := newOriginTestServer(t)
	w, ran := hubRequest(t, s, http.MethodGet, "/benchmarks/latest.json", "https://ailang.sunholo.com")
	if !ran || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("GET /benchmarks from docs site: ran=%v ACAO=%q, want *", ran, w.Header().Get("Access-Control-Allow-Origin"))
	}
	w, ran = hubRequest(t, s, http.MethodPost, "/benchmarks/latest.json", evilOrigin)
	if ran || w.Code != http.StatusForbidden {
		t.Fatalf("POST /benchmarks from foreign origin: ran=%v status=%d, want 403", ran, w.Code)
	}
	w, _ = hubRequest(t, s, http.MethodGet, "/benchmarksX", evilOrigin)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("GET /benchmarksX: ACAO=%q, want none (prefix must be /benchmarks/)", got)
	}
}

func TestHubOrigin_InvalidAllowlistRejected(t *testing.T) {
	_, err := NewServer(filepath.Join(t.TempDir(), "collab.db"), hubHost, WithCORSOrigins([]string{"localhost:3000"}))
	if err == nil {
		t.Fatal("malformed --cors-origin accepted")
	}
}
