// Package originpolicy is the one cross-origin policy for AILANG's HTTP
// listeners: `ailang serve-api` (internal/apiserver) and `ailang server`
// (internal/server + internal/websocket). One concept, one implementation
// (M-SERVER-ORIGIN-POLICY; extracted from M-SERVEAPI-BIND-HOST-CORS).
//
// A request is admitted when it carries no Origin (curl, CLIs, server-to-
// server, MCP clients), is same-origin (the Origin's host:port equals the
// request Host), or comes from an exact-listed origin. A listed origin gets
// its origin echoed in Access-Control-Allow-Origin. Anything else is
// "foreign": GET/HEAD run but get no ACAO (so the page cannot read the
// answer), and every other method — including the preflight — gets 403
// before dispatch, because CORS alone only stops a page READING a response,
// and a text/plain POST is sent without any preflight.
//
// WebSockets are not governed by CORS at all, so CheckWebSocket is the only
// thing stopping another page in the user's browser from opening a socket
// (cross-site WebSocket hijacking).
package originpolicy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Policy is an immutable origin policy. The zero value is not usable; build
// one with New.
type Policy struct {
	anyOrigin    bool
	allowed      map[string]bool
	allowMethods string
}

// allowHeaders is the request-header set granted to admitted cross-origin
// callers.
const allowHeaders = "Content-Type, Authorization"

// New builds a policy. anyOrigin grants every origin (ACAO: *); origins is an
// exact-match allowlist. Validate the inputs first with Validate — New does
// not, so tests and callers with already-validated config stay simple.
// allowMethods is the Access-Control-Allow-Methods value for admitted
// preflights.
func New(anyOrigin bool, origins []string, allowMethods string) *Policy {
	p := &Policy{anyOrigin: anyOrigin, allowMethods: allowMethods}
	if len(origins) > 0 {
		p.allowed = make(map[string]bool, len(origins))
		for _, o := range origins {
			p.allowed[o] = true
		}
	}
	return p
}

// Validate rejects conflicting or malformed settings at startup. Each origin
// must be exactly scheme://host[:port] with scheme http or https — the form
// a browser puts in the Origin header.
func Validate(anyOrigin bool, origins []string) error {
	if anyOrigin && len(origins) > 0 {
		return fmt.Errorf("--cors (all origins) and --cors-origin (allowlist) are mutually exclusive; pass one")
	}
	for _, o := range origins {
		u, err := url.Parse(o)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
			u.Scheme+"://"+u.Host != o {
			return fmt.Errorf("invalid --cors-origin %q: want scheme://host[:port] with scheme http or https and nothing after the host (e.g. https://app.example.com)", o)
		}
	}
	return nil
}

// AnyOrigin reports whether every origin is granted (--cors).
func (p *Policy) AnyOrigin() bool { return p.anyOrigin }

// HasAllowlist reports whether an exact-origin allowlist is set.
func (p *Policy) HasAllowlist() bool { return len(p.allowed) > 0 }

// Listed reports whether origin is on the exact-match allowlist.
func (p *Policy) Listed(origin string) bool { return p.allowed[origin] }

// SameOrigin reports whether r's Origin names the host:port the request was
// sent to. The scheme is not compared: a TLS-terminating proxy (Cloud Run,
// tailscale serve) forwards https traffic as http.
func SameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return false
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host)
}

// Apply enforces the REST policy on r, writing CORS headers (and, when the
// request is refused or is an answered preflight, the response). It reports
// whether the handler should run.
func (p *Policy) Apply(w http.ResponseWriter, r *http.Request) bool {
	if p.anyOrigin {
		p.setHeaders(w, "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return false
		}
		return true
	}
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	switch {
	case origin == "":
		return true
	case p.allowed[origin]:
		p.setHeaders(w, origin)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return false
		}
		return true
	case SameOrigin(r):
		return true
	case r.Method == http.MethodGet || r.Method == http.MethodHead:
		return true // runs, but with no ACAO the page cannot read the answer
	}
	http.Error(w, fmt.Sprintf("origin %s is not allowed (same-origin, or --cors-origin)", origin), http.StatusForbidden)
	return false
}

// CheckWebSocket enforces the Origin rule for a WebSocket upgrade: same-
// origin and exact-listed origins are admitted; "null" and foreign origins
// are refused. --cors (any origin) is deliberately NOT a WebSocket grant.
// allowMissing admits a request with no Origin header, which only a
// non-browser client can send.
func (p *Policy) CheckWebSocket(r *http.Request, allowMissing bool) error {
	origin := r.Header.Get("Origin")
	switch {
	case origin == "":
		if allowMissing {
			return nil
		}
		return fmt.Errorf("WebSocket refused: missing Origin")
	case p.allowed[origin], SameOrigin(r):
		return nil
	}
	return fmt.Errorf("WebSocket refused: origin %s is not allowed (same-origin, or --cors-origin)", origin)
}

func (p *Policy) setHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", p.allowMethods)
	w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
}
