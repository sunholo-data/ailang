package apiserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	listedOrigin   = "https://daneel.ts.net"
	unlistedOrigin = "https://evil.example"
	fnPath         = "/api/test/api/greet/add"
)

// countingCORS wraps a handler that counts its invocations in s.corsWrap.
func countingCORS(s *Server) (http.HandlerFunc, *int) {
	calls := 0
	return s.corsWrap(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}), &calls
}

func allowlistServer(t *testing.T) *Server {
	t.Helper()
	return newListenServer(t, Config{Port: "0", CORSOrigins: []string{listedOrigin}})
}

func do(h http.Handler, method, path, origin, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", "POST")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// With no CORS flag, no route grants cross-origin access.
func TestCORS_OffByDefault(t *testing.T) {
	srv := testServerWith(t, Config{Port: "0"})
	defer srv.Close()
	mux := srv.buildRoutes()

	for _, path := range []string{fnPath, "/api/_health"} {
		w := do(mux, http.MethodOptions, path, unlistedOrigin, "", "")
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("OPTIONS %s with CORS off: ACAO = %q, want none", path, got)
		}
		if w.Code == http.StatusNoContent {
			t.Errorf("OPTIONS %s with CORS off answered 204 as if preflight were granted", path)
		}
	}
	w := do(mux, http.MethodGet, "/api/_health", unlistedOrigin, "", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("GET /api/_health with CORS off: ACAO = %q, want none", got)
	}
}

func TestCORS_AllowlistPreflightDenied(t *testing.T) {
	h, calls := countingCORS(allowlistServer(t))
	w := do(h, http.MethodOptions, fnPath, unlistedOrigin, "", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("preflight from unlisted origin: status %d, want 403", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("preflight from unlisted origin: ACAO = %q, want none", got)
	}
	if *calls != 0 {
		t.Fatalf("handler ran %d times for a denied preflight", *calls)
	}
}

func TestCORS_AllowlistPreflightAllowed(t *testing.T) {
	h, calls := countingCORS(allowlistServer(t))
	w := do(h, http.MethodOptions, fnPath, listedOrigin, "", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight from listed origin: status %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != listedOrigin {
		t.Fatalf("ACAO = %q, want the exact origin %q", got, listedOrigin)
	}
	if !strings.Contains(w.Header().Get("Vary"), "Origin") {
		t.Fatalf("Vary = %q, want it to include Origin", w.Header().Get("Vary"))
	}
	if *calls != 0 {
		t.Fatalf("handler ran for a preflight")
	}

	// A real request from the listed origin runs and gets the echo.
	w = do(h, http.MethodPost, fnPath, listedOrigin, "application/json", `{"args":[1,2]}`)
	if w.Code != http.StatusOK || *calls != 1 {
		t.Fatalf("POST from listed origin: status %d, calls %d; want 200 and 1", w.Code, *calls)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != listedOrigin {
		t.Fatalf("POST from listed origin: ACAO = %q, want %q", got, listedOrigin)
	}
}

// Regression for the execution finding: a text/plain POST is a CORS "simple
// request" a browser sends with no preflight. With an allowlist set it must
// be refused before the function runs.
func TestCORS_AllowlistBlocksSimplePost(t *testing.T) {
	h, calls := countingCORS(allowlistServer(t))
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		w := do(h, method, fnPath, unlistedOrigin, "text/plain", `{"args":[1,2]}`)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s text/plain from unlisted origin: status %d, want 403", method, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s from unlisted origin: ACAO = %q, want none", method, got)
		}
	}
	if *calls != 0 {
		t.Fatalf("handler ran %d times for cross-origin requests from an unlisted origin", *calls)
	}

	// Safe methods from an unlisted origin still run, with no ACAO, so the
	// page cannot read the answer.
	w := do(h, http.MethodGet, fnPath, unlistedOrigin, "", "")
	if w.Code != http.StatusOK || *calls != 1 {
		t.Fatalf("GET from unlisted origin: status %d, calls %d; want 200 and 1", w.Code, *calls)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("GET from unlisted origin: ACAO = %q, want none", got)
	}
}

// Requests without an Origin header (curl, server-to-server, MCP clients)
// are unaffected by the allowlist, end to end through the real routes.
func TestCORS_NoOriginPasses(t *testing.T) {
	srv := testServerWith(t, Config{Port: "0", CORSOrigins: []string{listedOrigin}})
	defer srv.Close()
	mux := srv.buildRoutes()

	w := do(mux, http.MethodPost, fnPath, "", "text/plain", `{"args":[3,4]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST with no Origin: status %d, want 200: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"result":7`) {
		t.Fatalf("POST with no Origin: body %s, want result 7", w.Body.String())
	}

	// And the same request from an unlisted origin is refused through the
	// real routes, not only through the bare wrapper.
	w = do(mux, http.MethodPost, fnPath, unlistedOrigin, "text/plain", `{"args":[3,4]}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST from unlisted origin via routes: status %d, want 403", w.Code)
	}
}

// --cors keeps the previous any-origin behaviour (replaces TestCORSHeaders).
func TestCORS_AnyMode(t *testing.T) {
	srv := testServer(t) // CORS: true
	defer srv.Close()
	mux := srv.buildRoutes()

	w := do(mux, http.MethodOptions, "/api/_health", unlistedOrigin, "", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight in any mode: status %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("preflight in any mode: ACAO = %q, want *", got)
	}
	w = do(mux, http.MethodGet, "/api/_health", "", "", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("GET in any mode: ACAO = %q, want *", got)
	}
	w = do(mux, http.MethodPost, fnPath, unlistedOrigin, "application/json", `{"args":[1,2]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST in any mode: status %d, want 200", w.Code)
	}
}

func TestCORS_FlagConflict(t *testing.T) {
	if err := ValidateCORSConfig(true, []string{listedOrigin}); err == nil {
		t.Error("--cors with --cors-origin: want a startup error")
	}
	for _, ok := range []string{"https://daneel.ts.net", "http://localhost:5173", "http://127.0.0.1:8080"} {
		if err := ValidateCORSConfig(false, []string{ok}); err != nil {
			t.Errorf("origin %q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"daneel.ts.net", "https://daneel.ts.net/", "https://daneel.ts.net/app",
		"*", "null", "ftp://x.example", "https://", "https://x.example?q=1", "https://u@x.example", ""} {
		if err := ValidateCORSConfig(false, []string{bad}); err == nil {
			t.Errorf("malformed origin %q accepted", bad)
		}
	}
	if err := ValidateCORSConfig(true, nil); err != nil {
		t.Errorf("--cors alone: %v", err)
	}
	if err := ValidateCORSConfig(false, nil); err != nil {
		t.Errorf("no CORS flags: %v", err)
	}
}
