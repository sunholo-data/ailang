package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/platform/originpolicy"
)

// Cross-origin policy for `ailang server` (M-SERVER-ORIGIN-POLICY).
//
// The hub used to send Access-Control-Allow-Origin: * on every route and
// accept a WebSocket from any origin, so any page open in the operator's
// browser could read the coordinator/messages/observatory APIs and open the
// event socket. The policy is now the shared originpolicy one: same-origin
// (the embedded UI) and no-Origin callers (CLIs, hooks, OTLP exporters,
// server-to-server) pass; `--cors-origin` adds exact origins; everything
// else gets no ACAO and 403 on state-changing methods. The one public
// exception is /benchmarks/* (read-only data the docs site fetches).

const hubAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"

// benchmarksPublic grants every origin read access to /benchmarks/*: the
// docs site on GitHub Pages fetches it cross-origin at runtime
// (docs/src/lib/benchmarkFetch.js, M-EVAL-DATA-HOSTING-DECOUPLE).
var benchmarksPublic = originpolicy.New(true, nil, "GET, HEAD, OPTIONS")

// WithCORSOrigins adds exact origins (scheme://host[:port]) that may call the
// API and open the WebSocket cross-origin. NewServer validates them.
func WithCORSOrigins(origins []string) ServerOption {
	return func(s *Server) {
		s.corsOrigins = origins
	}
}

func isSafeRead(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// corsMiddleware applies the origin policy to every HTTP route.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logTelemetryRequest(r)

		policy := s.origins
		if strings.HasPrefix(r.URL.Path, "/benchmarks/") && isSafeRead(r.Method) {
			policy = benchmarksPublic
		}
		if !policy.Apply(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// logTelemetryRequest keeps the request logging that used to live in the
// CORS middleware (it helps debug external telemetry exporters).
func logTelemetryRequest(r *http.Request) {
	if r.Method == http.MethodPost {
		log.Printf("POST request: %s (Content-Type: %s, UA: %s)", r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"))
	}
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		log.Printf("OTLP request: %s %s (Content-Type: %s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
	}
	if strings.Contains(r.URL.Path, "trace") || strings.Contains(r.URL.Path, "log") || strings.Contains(r.URL.Path, "metric") {
		log.Printf("Potential telemetry request: %s %s (Content-Type: %s, UA: %s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"))
	}
}
