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

// BearerGate admits or refuses a request for a gated tool. It fails closed:
// only an explicit true from the verifier admits.
type BearerGate struct {
	verify      VerifyFunc
	metadataURL func(*http.Request) string
	timeout     time.Duration
	slots       chan struct{}
}

// NewBearerGate builds a gate. metadataURL returns the protected-resource
// metadata URL named in the challenge. Zero timeout or maxInFlight take the
// defaults.
func NewBearerGate(verify VerifyFunc, metadataURL func(*http.Request) string, timeout time.Duration, maxInFlight int) *BearerGate {
	if timeout <= 0 {
		timeout = DefaultVerifyTimeout
	}
	if maxInFlight <= 0 {
		maxInFlight = DefaultMaxInFlight
	}
	return &BearerGate{verify: verify, metadataURL: metadataURL, timeout: timeout, slots: make(chan struct{}, maxInFlight)}
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
	token := BearerToken(r.Header.Get("Authorization"))
	if token == "" {
		g.challenge(w, r, "")
		return false
	}
	select {
	case g.slots <- struct{}{}:
	default:
		writeUnavailable(w, "token verification is saturated")
		return false
	}
	ctx, cancel := context.WithTimeout(r.Context(), g.timeout)
	defer cancel()

	type outcome struct {
		ok  bool
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
		ok, err := g.verify(ctx, token)
		done <- outcome{ok: ok, err: err}
	}()

	select {
	case res := <-done:
		switch {
		case res.err != nil:
			writeUnavailable(w, "token verification failed")
			return false
		case !res.ok:
			g.challenge(w, r, "invalid_token")
			return false
		}
		return true
	case <-ctx.Done():
		writeUnavailable(w, "token verification timed out")
		return false
	}
}

func (g *BearerGate) challenge(w http.ResponseWriter, r *http.Request, errCode string) {
	params := []string{fmt.Sprintf("resource_metadata=%q", g.metadataURL(r))}
	if errCode != "" {
		params = append(params, fmt.Sprintf("error=%q", errCode))
	}
	w.Header().Set("WWW-Authenticate", "Bearer "+strings.Join(params, ", "))
	writeJSONError(w, http.StatusUnauthorized, "unauthorized", "sign in to use this tool")
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
	type call struct {
		Method string `json:"method"`
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
	var names []string
	for _, m := range msgs {
		if m.Method == "tools/call" {
			names = append(names, m.Params.Name)
		}
	}
	return names
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
