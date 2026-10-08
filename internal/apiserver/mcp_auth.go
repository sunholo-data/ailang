package apiserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// M-SERVEAPI-DIRECTORY-READY: registration rules for @mcp_auth,
// @mcp_token_verifier and @mcp_secret. Every rule here is an author or
// configuration bug, refused at registration with an ERROR (the posture of an
// invalid @mcp_name): a gated tool that cannot be authorized must not be served
// open, and a "secret" that would still be advertised is not a secret.

const (
	mcpAuthOAuth2 = "oauth2"
	mcpAuthNoAuth = "noauth"
)

// tokenVerifier locates the server's one @mcp_token_verifier function.
type tokenVerifier struct {
	modPath string
	name    string
}

// resolveTokenVerifier finds the single (string) -> bool verifier across the
// loaded modules. It returns (nil, nil) when there is none, and an error when
// there are several or the one found has the wrong signature: either way no
// @mcp_auth("oauth2") tool can be registered.
func resolveTokenVerifier(modules map[string]*ModuleInfo) (*tokenVerifier, error) {
	var found []tokenVerifier
	var badSig []string
	for modPath, info := range modules {
		for _, e := range info.Exports {
			if !e.IsTokenVerifier {
				continue
			}
			found = append(found, tokenVerifier{modPath: modPath, name: e.Name})
			if !e.VerifierSigOK {
				badSig = append(badSig, modPath+"/"+e.Name)
			}
		}
	}
	switch {
	case len(found) == 0:
		return nil, nil
	case len(found) > 1:
		names := make([]string, len(found))
		for i, v := range found {
			names[i] = v.modPath + "/" + v.name
		}
		sort.Strings(names)
		return nil, fmt.Errorf("@mcp_token_verifier: %d verifiers (%s); a server has exactly one", len(found), strings.Join(names, ", "))
	case len(badSig) > 0:
		return nil, fmt.Errorf("@mcp_token_verifier %s: signature must be (token: string) -> bool ! {...}", badSig[0])
	}
	return &found[0], nil
}

// validateMCPAuth checks one export's @mcp_auth and @mcp_secret against the
// server configuration. verifierErr is resolveTokenVerifier's error, if any.
func validateMCPAuth(e ExportInfo, issuer string, verifier *tokenVerifier, verifierErr error) error {
	switch e.MCPAuth {
	case "", mcpAuthNoAuth:
	case mcpAuthOAuth2:
		if issuer == "" {
			return fmt.Errorf("@mcp_auth(%q) needs --oauth-issuer (the authorization server named in the protected-resource metadata)", mcpAuthOAuth2)
		}
		if verifierErr != nil {
			return fmt.Errorf("@mcp_auth(%q): %v", mcpAuthOAuth2, verifierErr)
		}
		if verifier == nil {
			return fmt.Errorf("@mcp_auth(%q) needs one @mcp_token_verifier function (token: string) -> bool", mcpAuthOAuth2)
		}
	default:
		return fmt.Errorf("@mcp_auth(%q): unknown scheme; supported: %q, %q", e.MCPAuth, mcpAuthOAuth2, mcpAuthNoAuth)
	}
	return validateSecretParams(e)
}

// validateSecretParams: each @mcp_secret name must be a declared param that is
// also @optional, because the listed surface drops it and binds its zero value
// through the @optional path. _headers is bound by the server and is never a
// client-supplied secret.
func validateSecretParams(e ExportInfo) error {
	optional := make(map[string]bool, len(e.Optional))
	for _, n := range e.Optional {
		optional[n] = true
	}
	for _, name := range e.MCPSecret {
		isParam := false
		for _, p := range e.ParamNames {
			if p == name {
				isParam = true
				break
			}
		}
		switch {
		case name == headersParam:
			return fmt.Errorf("@mcp_secret(%q): _headers is bound from the request, not supplied by the client", name)
		case !isParam:
			return fmt.Errorf("@mcp_secret(%q): no such parameter (params: %s)", name, strings.Join(e.ParamNames, ", "))
		case !optional[name]:
			return fmt.Errorf("@mcp_secret(%q): must also be @optional(%q) so the listed surface can bind its zero value", name, name)
		}
	}
	return nil
}

// inputSchemaFor is the tool's inputSchema on this surface. The listed
// surface removes @mcp_secret params from properties and required; the agent
// surface (/mcp/) is unchanged.
func (ms *MCPServer) inputSchemaFor(e ExportInfo) map[string]any {
	schema := buildNamedInputSchema(e)
	applyFileSchema(schema, e)
	if !ms.listed || len(e.MCPSecret) == 0 {
		return schema
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for _, name := range e.MCPSecret {
			delete(props, name)
		}
	}
	if req, ok := schema["required"].([]string); ok {
		kept := req[:0:0]
		for _, name := range req {
			if !isSecretParam(e, name) {
				kept = append(kept, name)
			}
		}
		schema["required"] = kept
	}
	return schema
}

func isSecretParam(e ExportInfo, name string) bool {
	for _, s := range e.MCPSecret {
		if s == name {
			return true
		}
	}
	return false
}

// hasListedSurface reports whether any loaded export opts into the directory
// projection. /mcp/connect/ is mounted only then, so servers that use none of
// the listed-surface annotations are unchanged.
func hasListedSurface(modules map[string]*ModuleInfo) bool {
	for _, info := range modules {
		for _, e := range info.Exports {
			if e.MCPAuth != "" || len(e.MCPSecret) > 0 || e.IsAgentOnly || e.IsTokenVerifier {
				return true
			}
		}
	}
	return false
}

// listedMCPPath is where the directory projection is served.
const listedMCPPath = "/mcp/connect/"

// verifyToken calls the service's @mcp_token_verifier function. The call gets
// its own effect context whose GoCtx carries the gate's deadline, so any Net
// request the verifier makes is aborted when the gate stops waiting. Pure
// computation cannot be interrupted (the evaluator takes no context); the
// gate's slot cap bounds that case instead.
func (ms *MCPServer) verifyToken(ctx context.Context, token string) (bool, error) {
	v := ms.verifier
	if v == nil {
		return false, errors.New("no @mcp_token_verifier")
	}
	res, err := ms.server.engine.CallPrepared(func(eff interface{}) {
		if ec, ok := eff.(*effects.EffContext); ok {
			ec.GoCtx = ctx
		}
	}, v.modPath, v.name, token)
	if err != nil {
		return false, err
	}
	val, err := embed.ToGo(res)
	if err != nil {
		return false, err
	}
	ok, isBool := val.(bool)
	if !isBool {
		return false, fmt.Errorf("@mcp_token_verifier %s returned %T, want bool", v.name, val)
	}
	return ok, nil
}

// gatedHandler puts the Bearer gate in front of the listed surface. Only a
// tools/call naming a gated tool is checked; initialize, tools/list and open
// tools pass straight through. The body is read once (bounded by the
// server's upload limit) and restored for the MCP handler.
func (ms *MCPServer) gatedHandler(next http.Handler, maxBody int64) http.Handler {
	gate := protocol.NewBearerGate(ms.verifyToken, listedMetadataURL, 0, 0)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || len(ms.gated) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		for _, c := range protocol.ToolCalls(body) {
			if ms.gated[c.Name] {
				if !gate.AdmitCall(w, r, c.ID) {
					return
				}
				break
			}
		}
		next.ServeHTTP(w, r)
	})
}

// protectedResourcePath is the RFC 9728 metadata path for the listed surface
// (path-inserted form); the bare well-known path serves the same document.
const (
	protectedResourceRoot = "/.well-known/oauth-protected-resource"
	protectedResourcePath = protectedResourceRoot + "/mcp/connect/"
	// The same surface reached without the trailing slash. Directory clients
	// normalise URLs differently: claude.ai's connector check POSTs
	// /mcp/connect and does not follow ServeMux's 307 to /mcp/connect/.
	protectedResourcePathBare = protectedResourceRoot + "/mcp/connect"
)

// bareListedKey marks a request that arrived on the no-slash listed URL, so
// its 401 names the no-slash metadata document. RFC 9728 §3.3: the resource a
// client is told about must be the URL it used. The MCP TypeScript SDK rejects
// /mcp/connect against a resource of /mcp/connect/.
type bareListedKey struct{}

// bareListed serves next for a request on the no-slash form of a slash-
// registered surface: it marks the request and rewrites the path to the
// registered form, instead of the client-visible 307 ServeMux would send.
func bareListed(next http.Handler, registered string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := r.WithContext(context.WithValue(r.Context(), bareListedKey{}, true))
		u := *r.URL
		u.Path = registered
		u.RawPath = ""
		r2.URL = &u
		next.ServeHTTP(w, r2)
	})
}

func isBareListed(r *http.Request) bool {
	v, _ := r.Context().Value(bareListedKey{}).(bool)
	return v
}

func listedMetadataURL(r *http.Request) string {
	if isBareListed(r) {
		return protocol.PublicBaseURL(r) + protectedResourcePathBare
	}
	return protocol.PublicBaseURL(r) + protectedResourcePath
}

// handleProtectedResource serves the listed surface's resource metadata:
// resource = the listed MCP URL as the client reached it, issuer from
// --oauth-issuer.
func (s *Server) handleProtectedResource(w http.ResponseWriter, r *http.Request) {
	resource := protocol.PublicBaseURL(r) + listedMCPPath
	if r.URL.Path == protectedResourcePathBare {
		resource = strings.TrimSuffix(resource, "/")
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(protocol.ProtectedResourceMetadata(resource, s.oauthIssuer))
}
