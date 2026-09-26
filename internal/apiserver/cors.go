package apiserver

import (
	"fmt"
	"net/http"
	"net/url"
)

// CORS has three modes (M-SERVEAPI-BIND-HOST-CORS M2):
//
//   - off (default): no CORS headers; OPTIONS falls through to the handler.
//   - any (--cors): Access-Control-Allow-Origin: * on every wrapped route.
//   - allowlist (--cors-origin, repeatable): the exact listed origins are
//     echoed back. A request whose Origin is present and NOT listed is
//     refused with 403 before dispatch unless its method is GET or HEAD —
//     CORS alone only stops a page reading the answer, and a text/plain POST
//     is sent by a browser without any preflight, so the call itself must be
//     refused. Requests with no Origin (curl, server-to-server, MCP clients)
//     pass.

const (
	corsAllowMethods = "GET, POST, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization"
)

// ValidateCORSConfig rejects conflicting or malformed CORS settings at
// startup. Each origin must be exactly scheme://host[:port] with scheme http
// or https — the form a browser puts in the Origin header.
func ValidateCORSConfig(anyOrigin bool, origins []string) error {
	if anyOrigin && len(origins) > 0 {
		return fmt.Errorf("--cors (all origins) and --cors-origin (allowlist) are mutually exclusive; pass one")
	}
	for _, o := range origins {
		if err := validateOrigin(o); err != nil {
			return err
		}
	}
	return nil
}

func validateOrigin(o string) error {
	u, err := url.Parse(o)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.Scheme+"://"+u.Host != o {
		return fmt.Errorf("invalid --cors-origin %q: want scheme://host[:port] with scheme http or https and nothing after the host (e.g. https://app.example.com)", o)
	}
	return nil
}

func (s *Server) corsWrap(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case s.cors:
			setCORSHeaders(w, "*")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		case len(s.corsOrigins) > 0:
			if !s.allowlistCORS(w, r) {
				return
			}
		}
		handler(w, r)
	}
}

// allowlistCORS applies the allowlist mode and reports whether the handler
// should run.
func (s *Server) allowlistCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if s.corsOrigins[origin] {
		setCORSHeaders(w, origin)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return false
		}
		return true
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true // runs, but with no ACAO the page cannot read the answer
	}
	http.Error(w, fmt.Sprintf("origin %s is not allowed (serve-api --cors-origin)", origin), http.StatusForbidden)
	return false
}

func setCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
	w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
}

func originSet(origins []string) map[string]bool {
	if len(origins) == 0 {
		return nil
	}
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		set[o] = true
	}
	return set
}
