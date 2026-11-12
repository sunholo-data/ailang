package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// M-SERVEAPI-DIRECTORY-READY: lazy OAuth for an MCP surface. Discovery and open
// tools answer without a token; a tools/call naming a gated tool must carry a
// Bearer token the host's verifier accepts. Directory clients (Claude) start
// sign-in only when the HTTP request itself fails with 401 and a
// WWW-Authenticate header. A tool result cannot do that, so the check runs
// before dispatch. Both MCP implementations (serve-api's go-sdk path and
// mcphttp) use this one gate, so they answer identically.

// Tool auth schemes for ToolDescriptor.Auth.
const (
	ToolAuthNone   = ""
	ToolAuthNoAuth = "noauth"
	ToolAuthOAuth2 = "oauth2"
)

// Gate defaults (design doc D7). The timeout sits below Claude's 10 s budget
// for auth endpoints.
const (
	DefaultVerifyTimeout = 5 * time.Second
	DefaultMaxInFlight   = 32
)

// VerifyFunc reports whether a Bearer token is valid. An error means the
// token could not be checked, which is distinct from a rejected token.
type VerifyFunc func(ctx context.Context, token string) (bool, error)

// Identity is who a verified Bearer token belongs to
// (M-MCP-OAUTH-CLIENT-MODELS D5): the verifier's subject, the OAuth client the
// token was issued to, and its scopes.
type Identity struct {
	Subject  string   `json:"subject"`
	ClientID string   `json:"client_id"`
	Scopes   []string `json:"scopes"`
}

// IdentityVerifyFunc returns the identity a Bearer token carries, or nil for a
// rejected token. An error means the token could not be checked.
type IdentityVerifyFunc func(ctx context.Context, token string) (*Identity, error)

// BearerGate admits or refuses a request for a gated tool. It fails closed:
// only an explicit true from a bool verifier, or an identity with a non-empty
// subject from an identity verifier, admits.
type BearerGate struct {
	verify      IdentityVerifyFunc
	identity    bool // built by NewIdentityGate: AdmitCallIdentity returns the identity
	metadataURL func(*http.Request) string
	timeout     time.Duration
	slots       chan struct{}
}

// NewBearerGate builds a gate around a bool verifier. metadataURL returns the
// protected-resource metadata URL named in the challenge. Zero timeout or
// maxInFlight take the defaults. Its AdmitCallIdentity admits with a nil
// identity: a bool verifier names no subject.
func NewBearerGate(verify VerifyFunc, metadataURL func(*http.Request) string, timeout time.Duration, maxInFlight int) *BearerGate {
	return newGate(func(ctx context.Context, token string) (*Identity, error) {
		ok, err := verify(ctx, token)
		if err != nil || !ok {
			return nil, err
		}
		return &Identity{}, nil
	}, false, metadataURL, timeout, maxInFlight)
}

// NewIdentityGate builds a gate around an identity verifier
// (M-MCP-OAUTH-CLIENT-MODELS D5). A nil identity, or one whose Subject is
// empty, is a rejected token (401), never an anonymous admit.
func NewIdentityGate(verify IdentityVerifyFunc, metadataURL func(*http.Request) string, timeout time.Duration, maxInFlight int) *BearerGate {
	return newGate(verify, true, metadataURL, timeout, maxInFlight)
}

func newGate(verify IdentityVerifyFunc, identity bool, metadataURL func(*http.Request) string, timeout time.Duration, maxInFlight int) *BearerGate {
	if timeout <= 0 {
		timeout = DefaultVerifyTimeout
	}
	if maxInFlight <= 0 {
		maxInFlight = DefaultMaxInFlight
	}
	return &BearerGate{verify: verify, identity: identity, metadataURL: metadataURL, timeout: timeout, slots: make(chan struct{}, maxInFlight)}
}

// Admit returns true when the request may proceed. Otherwise it has written
// the refusal and the caller must not run the tool:
//   - no Bearer token, or the verifier says false → 401 + WWW-Authenticate
//   - verifier error, panic, timeout, or all slots busy → 503 + Retry-After
//
// 503 rather than 401 for an unavailable verifier, so a backend outage does
// not send the client through a fresh OAuth flow. A verifier that outlives
// the timeout keeps its slot until it returns, so in-flight verifications
// never exceed the cap.
func (g *BearerGate) Admit(w http.ResponseWriter, r *http.Request) bool {
	return g.AdmitCall(w, r, nil)
}

// AdmitCall is Admit for one JSON-RPC tools/call whose id is callID. A 401
// keeps its status and WWW-Authenticate header, and its body becomes the
// JSON-RPC tool error ChatGPT reads (see challengeResult). A nil callID
// writes the plain JSON error body Admit always wrote.
func (g *BearerGate) AdmitCall(w http.ResponseWriter, r *http.Request, callID json.RawMessage) bool {
	_, ok := g.AdmitCallIdentity(w, r, callID)
	return ok
}

// AdmitCallIdentity is AdmitCall that also returns who was admitted. On a gate
// built by NewIdentityGate an admit always carries a non-empty Subject; on a
// NewBearerGate gate the identity is nil. The refusals are AdmitCall's.
func (g *BearerGate) AdmitCallIdentity(w http.ResponseWriter, r *http.Request, callID json.RawMessage) (*Identity, bool) {
	token := BearerToken(r.Header.Get("Authorization"))
	if token == "" {
		g.challenge(w, r, "", callID)
		return nil, false
	}
	select {
	case g.slots <- struct{}{}:
	default:
		writeUnavailable(w, "token verification is saturated")
		return nil, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), g.timeout)
	defer cancel()

	type outcome struct {
		id  *Identity
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		defer func() { <-g.slots }()
		defer func() {
			if p := recover(); p != nil {
				done <- outcome{err: fmt.Errorf("token verifier panicked: %v", p)}
			}
		}()
		id, err := g.verify(ctx, token)
		done <- outcome{id: id, err: err}
	}()

	select {
	case res := <-done:
		switch {
		case res.err != nil:
			writeUnavailable(w, "token verification failed")
			return nil, false
		case res.id == nil, g.identity && res.id.Subject == "":
			g.challenge(w, r, "invalid_token", callID)
			return nil, false
		}
		if !g.identity {
			return nil, true
		}
		return res.id, true
	case <-ctx.Done():
		writeUnavailable(w, "token verification timed out")
		return nil, false
	}
}

// WriteRateLimited writes a 429 refusal with Retry-After (whole seconds, at
// least 1). With a callID the body is a JSON-RPC isError tool result, as for
// a challenge; without one it is the plain JSON error body.
func WriteRateLimited(w http.ResponseWriter, callID json.RawMessage, retryAfter time.Duration) {
	secs := int((retryAfter + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", fmt.Sprint(secs))
	msg := fmt.Sprintf("Rate limit exceeded for this account: retry after %d s.", secs)
	if callID == nil {
		writeJSONError(w, http.StatusTooManyRequests, "rate_limited", msg)
		return
	}
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      callID,
		"result": map[string]any{
			"content": []map[string]any{{"type": "text", "text": msg}},
			"isError": true,
		},
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write(b)
}

func (g *BearerGate) challenge(w http.ResponseWriter, r *http.Request, errCode string, callID json.RawMessage) {
	params := []string{fmt.Sprintf("resource_metadata=%q", g.metadataURL(r))}
	if errCode != "" {
		params = append(params, fmt.Sprintf("error=%q", errCode))
	}
	value := "Bearer " + strings.Join(params, ", ")
	w.Header().Set("WWW-Authenticate", value)
	if callID == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "sign in to use this tool")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write(ChallengeResult(callID, MetaChallenge(value, errCode)))
}

// MetaChallenge is the _meta["mcp/www_authenticate"] value for a refused
// call: the HTTP challenge plus the error and error_description parameters
// OpenAI requires. Apps SDK, Authentication
// (https://developers.openai.com/apps-sdk/build/auth, fetched 2026-11-12):
// "make sure the value contains both an `error` and `error_description`
// parameter"; this payload "is what actually triggers the tool-level OAuth
// UI". The HTTP header keeps RFC 6750 §3.1 (no error code when the request
// carried no token), which is what Claude starts sign-in from.
//
// With no token the error is insufficient_scope, as in OpenAI's own example
// for "no access token provided"; a rejected token keeps its code
// (invalid_token).
func MetaChallenge(httpChallenge, errCode string) string {
	desc := "Sign in to use this tool."
	if errCode == "" {
		errCode = "insufficient_scope"
	} else {
		desc = "Your sign-in has expired or was revoked. Sign in again to use this tool."
	}
	if !strings.Contains(httpChallenge, "error=") {
		httpChallenge += fmt.Sprintf(", error=%q", errCode)
	}
	return httpChallenge + fmt.Sprintf(", error_description=%q", desc)
}

// WWWAuthenticateMetaKey is the tool-result _meta key that carries the
// challenge to ChatGPT.
const WWWAuthenticateMetaKey = "mcp/www_authenticate"

// ChallengeResult is the JSON-RPC response for a refused tools/call: an
// isError tool result whose _meta["mcp/www_authenticate"] holds the
// challenge (MetaChallenge: the HTTP WWW-Authenticate header plus error and
// error_description).
//
// OpenAI Apps SDK, Authentication (https://developers.openai.com/apps-sdk/build/auth,
// fetched 2026-10-07): mixed auth "requires both metadata (securitySchemes and
// the resource metadata document) and runtime errors that carry
// _meta["mcp/www_authenticate"]". Its example result:
//
//	{"jsonrpc": "2.0", "id": 4, "result": {"content": [{"type": "text",
//	 "text": "Authentication required: no access token provided."}],
//	 "_meta": {"mcp/www_authenticate": ["Bearer resource_metadata=\"…\",
//	 error=\"insufficient_scope\", error_description=\"…\""]}, "isError": true}}
//
// The reference types the key "string or string[]"; this uses the array form
// of the example. The HTTP status stays 401: Claude starts sign-in from the
// status and header, not from the body.
func ChallengeResult(callID json.RawMessage, wwwAuthenticate string) []byte {
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      callID,
		"result": map[string]any{
			"content": []map[string]any{{"type": "text", "text": "Authentication required: sign in to use this tool."}},
			"isError": true,
			"_meta":   map[string]any{WWWAuthenticateMetaKey: []string{wwwAuthenticate}},
		},
	})
	return b
}

func writeUnavailable(w http.ResponseWriter, msg string) {
	w.Header().Set("Retry-After", "5")
	writeJSONError(w, http.StatusServiceUnavailable, "token_verification_unavailable", msg)
}

func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": msg})
}

// BearerToken extracts the token from an Authorization header value ("" when
// the scheme is not Bearer or the token is empty).
func BearerToken(authorization string) string {
	fields := strings.Fields(authorization)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
		return ""
	}
	return fields[1]
}

// ToolCallNames returns the tool names of every tools/call in a JSON-RPC body,
// a single message or a batch. A body that is not JSON-RPC yields none and is
// left for the MCP handler to reject.
func ToolCallNames(body []byte) []string {
	var names []string
	for _, c := range ToolCalls(body) {
		names = append(names, c.Name)
	}
	return names
}

// ToolCall is one tools/call in a JSON-RPC body: the tool and the request id.
type ToolCall struct {
	Name string
	ID   json.RawMessage
}

// ToolCalls returns every tools/call in a JSON-RPC body, a single message or
// a batch, in order.
func ToolCalls(body []byte) []ToolCall {
	type call struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	var msgs []call
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if json.Unmarshal(trimmed, &msgs) != nil {
			return nil
		}
	} else {
		var one call
		if json.Unmarshal(trimmed, &one) != nil {
			return nil
		}
		msgs = []call{one}
	}
	var calls []ToolCall
	for _, m := range msgs {
		if m.Method == "tools/call" {
			calls = append(calls, ToolCall{Name: m.Params.Name, ID: m.ID})
		}
	}
	return calls
}

// ProtectedResourceMetadata is the RFC 9728 document for an MCP resource.
func ProtectedResourceMetadata(resource, issuer string) []byte {
	b, _ := json.Marshal(map[string]any{
		"resource":                 resource,
		"authorization_servers":    []string{issuer},
		"bearer_methods_supported": []string{"header"},
	})
	return b
}

// PublicBaseURL is the scheme://host a client used to reach r, honouring
// X-Forwarded-Proto from a TLS-terminating proxy (Cloud Run, load balancers).
func PublicBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
