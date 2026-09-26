package apiserver

import (
	"net/http"

	"github.com/sunholo-data/ailang/internal/platform/originpolicy"
)

// CORS has three modes (M-SERVEAPI-BIND-HOST-CORS M2). The decision itself
// lives in internal/platform/originpolicy, shared with `ailang server`
// (M-SERVER-ORIGIN-POLICY); this file only picks the serve-api mode.
//
//   - off (default): no CORS headers; OPTIONS falls through to the handler.
//   - any (--cors): Access-Control-Allow-Origin: * on every wrapped route.
//   - allowlist (--cors-origin, repeatable): the exact listed origins are
//     echoed back; same-origin passes; any other Origin is refused with 403
//     before dispatch unless its method is GET or HEAD. Requests with no
//     Origin (curl, server-to-server, MCP clients) pass.

const corsAllowMethods = "GET, POST, OPTIONS"

// ValidateCORSConfig rejects conflicting or malformed CORS settings at
// startup (see originpolicy.Validate).
func ValidateCORSConfig(anyOrigin bool, origins []string) error {
	return originpolicy.Validate(anyOrigin, origins)
}

func (s *Server) corsWrap(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Off mode is serve-api's historical default: no enforcement at all
		// (open question Q2 of M-SERVEAPI-BIND-HOST-CORS).
		if s.origins.AnyOrigin() || s.origins.HasAllowlist() {
			if !s.origins.Apply(w, r) {
				return
			}
		}
		handler(w, r)
	}
}
