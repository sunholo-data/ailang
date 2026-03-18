package apiserver

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
)

// authMiddleware wraps a handler with API key authentication.
// If no API key is configured (apiKeyHeader or apiKeyEnv is empty), returns the handler unchanged.
// Requests to meta endpoints (/api/_*), MCP (/mcp/), and A2A (/a2a/) bypass auth.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	if s.apiKeyHeader == "" || s.apiKeyEnv == "" {
		return next // no auth configured
	}

	expected := os.Getenv(s.apiKeyEnv)
	if expected == "" {
		log.Fatalf("serve-api: --api-key-env %q is set but the environment variable is empty or not defined", s.apiKeyEnv)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Exempt meta, health, MCP, A2A, and well-known endpoints from auth
		if strings.HasPrefix(path, "/api/_") ||
			strings.HasPrefix(path, "/mcp/") ||
			strings.HasPrefix(path, "/a2a/") ||
			strings.HasPrefix(path, "/.well-known/") {
			next(w, r)
			return
		}

		// Check the configured header
		key := r.Header.Get(s.apiKeyHeader)

		// Also accept standard Authorization: Bearer <token>
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if key == "" || subtle.ConstantTimeCompare([]byte(key), []byte(expected)) != 1 {
			writeJSON(w, http.StatusUnauthorized, FunctionCallResponse{
				Error: "unauthorized: invalid or missing API key",
			})
			return
		}

		next(w, r)
	}
}
