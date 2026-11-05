package mcpcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// M-MCP-OAUTH-CLIENT-MODELS M3c: the framing probe presents a real client.

// probeDoc is the probe client document as the docs site serves it.
func probeDoc(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "static", "mcp-check", "client-metadata.json"))
	if err != nil {
		t.Fatalf("probe client document: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("probe client document is not JSON: %v", err)
	}
	return doc
}

// TestProbeClientDocument pins the public document: its client_id is its own
// URL (CIMD), it lists the probe's loopback redirect, and it is a public
// client. A docs restructure that moves it fails here.
func TestProbeClientDocument(t *testing.T) {
	if ProbeClientURL != "https://ailang.sunholo.com/mcp-check/client-metadata.json" {
		t.Fatalf("ProbeClientURL moved: %s", ProbeClientURL)
	}
	doc := probeDoc(t)
	if doc["client_id"] != ProbeClientURL {
		t.Errorf("client_id = %v, want %s", doc["client_id"], ProbeClientURL)
	}
	uris, _ := doc["redirect_uris"].([]any)
	if len(uris) != 1 || uris[0] != ProbeRedirectURI {
		t.Errorf("redirect_uris = %v, want [%s]", uris, ProbeRedirectURI)
	}
	if doc["token_endpoint_auth_method"] != "none" {
		t.Errorf("token_endpoint_auth_method = %v", doc["token_endpoint_auth_method"])
	}
	if m, _ := doc["token_endpoint_auth_methods_supported"].([]any); len(m) != 1 || m[0] != "none" {
		t.Errorf("token_endpoint_auth_methods_supported = %v", m)
	}
}

var unreservedRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// cimdAS is a CIMD-only authorization server shaped like sunholo/mcp_oauth:
// it fetches the client document (from docs/static, standing in for the
// docs site), rejects with 400 what beginAuthorize rejects (V29), and
// otherwise redirects to a framing-protected login page. consent false makes
// it bounce straight to the client's redirect_uri instead.
type cimdAS struct {
	t        *testing.T
	srv      *httptest.Server
	consent  bool
	registry map[string]string // pre-registered client_id -> redirect_uri
	mu       sync.Mutex
	queries  []url.Values
}

func newCimdAS(t *testing.T, cimd bool) *cimdAS {
	as := &cimdAS{t: t, consent: true, registry: map[string]string{}}
	as.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			md := map[string]any{
				"issuer":                           as.srv.URL,
				"authorization_endpoint":           as.srv.URL + "/authorize",
				"code_challenge_methods_supported": []string{"S256"},
			}
			if cimd {
				md["client_id_metadata_document_supported"] = true
				md["token_endpoint_auth_methods_supported"] = []string{"none"}
			} else {
				md["token_endpoint_auth_methods_supported"] = []string{"client_secret_basic"}
			}
			_ = json.NewEncoder(w).Encode(md)
		case "/authorize":
			as.authorize(w, r, cimd)
		case "/login":
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("X-Frame-Options", "DENY")
			_, _ = w.Write([]byte("<!doctype html><p>sign in</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(as.srv.Close)
	return as
}

func (as *cimdAS) authorize(w http.ResponseWriter, r *http.Request, cimd bool) {
	q := r.URL.Query()
	as.mu.Lock()
	as.queries = append(as.queries, q)
	as.mu.Unlock()
	bad := func(desc string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"` + desc + `"}`))
	}
	clientID, redirect := q.Get("client_id"), q.Get("redirect_uri")
	var registered []string
	if want, ok := as.registry[clientID]; ok {
		registered = []string{want}
	} else if cimd && clientID == ProbeClientURL {
		doc := probeDoc(as.t)
		if doc["client_id"] != clientID {
			bad("client_id in the document does not equal the URL")
			return
		}
		for _, u := range doc["redirect_uris"].([]any) {
			registered = append(registered, u.(string))
		}
	} else {
		bad("client_id is not an acceptable https client metadata URL")
		return
	}
	switch {
	case q.Get("response_type") != "code":
		bad("response_type must be code")
	case q.Get("code_challenge_method") != "S256":
		bad("code_challenge_method must be S256")
	case len(q.Get("code_challenge")) != 43 || !unreservedRe.MatchString(q.Get("code_challenge")):
		bad("code_challenge must be a 43-character S256 value")
	case !loopbackRedirectOk(redirect, registered):
		bad("redirect_uri is not registered for this client")
	case len(q.Get("state")) > 512 || !unreservedRe.MatchString(q.Get("state")):
		bad("state must be unreserved characters, at most 512")
	case !as.consent:
		http.Redirect(w, r, redirect+"?code=x&state="+q.Get("state"), http.StatusFound)
	default:
		http.Redirect(w, r, "/login?handle=h", http.StatusFound)
	}
}

// loopbackRedirectOk mirrors mcp_oauth core: an exact match, or a loopback
// http redirect matching on everything but the port (RFC 8252 §7.3).
func loopbackRedirectOk(presented string, registered []string) bool {
	u, err := url.Parse(presented)
	if err != nil {
		return false
	}
	if u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost") {
		u.Host = u.Hostname()
	}
	for _, r := range registered {
		if r == presented || r == u.String() {
			return true
		}
	}
	return false
}

func framingFinding(t *testing.T, fs []Finding) Finding {
	t.Helper()
	for _, f := range fs {
		if f.Check == "authorization-framing" {
			return f
		}
	}
	t.Fatalf("no authorization-framing finding: %+v", fs)
	return Finding{}
}

// TestFramingProbeCimdPass: a CIMD-only AS accepts the probe client, so the
// probe reaches the protected login page and PASSes. Before M3c it sent
// ailang-mcp-check-dummy and got 400 -> a false WARN on every CIMD server.
func TestFramingProbeCimdPass(t *testing.T) {
	as := newCimdAS(t, true)
	s := session{opts: Options{URL: "https://svc.example/mcp/connect/", Target: "both", Client: as.srv.Client()}}
	f := framingFinding(t, s.checkAuthorizationServer(context.Background(), as.srv.URL))
	if f.Status != Pass {
		t.Fatalf("CIMD-only AS: %+v, want PASS", f)
	}
	if !strings.Contains(f.Message, ProbeClientURL) {
		t.Errorf("finding should name the probe client: %q", f.Message)
	}
	q := as.queries[0]
	if q.Get("resource") != "https://svc.example/mcp/connect/" {
		t.Errorf("resource = %q, want the MCP URL (RFC 8707)", q.Get("resource"))
	}
	if q.Get("state") != "ailang-mcp-check-dummy-state" || q.Get("response_type") != "code" {
		t.Errorf("state/response_type changed: %v", q)
	}
	// A fresh verifier per run: two runs send different challenges.
	_ = s.checkAuthorizationServer(context.Background(), as.srv.URL)
	if a, b := as.queries[0].Get("code_challenge"), as.queries[1].Get("code_challenge"); a == b {
		t.Errorf("code_challenge reused across runs: %s", a)
	}
}

// TestFramingProbeFlagsClient: an AS without CIMD (pre-registered clients,
// GE-style) is probed with --probe-client-id / --probe-redirect-uri.
func TestFramingProbeFlagsClient(t *testing.T) {
	as := newCimdAS(t, false)
	as.registry["ge-client-1"] = "https://vertexaisearch.cloud.google.com/oauth-redirect"
	s := session{opts: Options{Target: "anthropic", Client: as.srv.Client(),
		ProbeClientID: "ge-client-1", ProbeRedirectURI: "https://vertexaisearch.cloud.google.com/oauth-redirect"}}
	f := framingFinding(t, s.checkAuthorizationServer(context.Background(), as.srv.URL))
	if f.Status != Pass || !strings.Contains(f.Message, "ge-client-1") {
		t.Fatalf("pre-registered probe client: %+v, want PASS naming ge-client-1", f)
	}
	if q := as.queries[0]; q.Get("client_id") != "ge-client-1" || q.Get("redirect_uri") != "https://vertexaisearch.cloud.google.com/oauth-redirect" {
		t.Fatalf("flags not used: %v", q)
	}
}

// TestFramingProbeNoCimdSkips: no CIMD and no flags -> SKIP, never WARN, and
// the AS is never called (in particular, nothing is registered through DCR).
func TestFramingProbeNoCimdSkips(t *testing.T) {
	as := newCimdAS(t, false)
	s := session{opts: Options{Target: "both", Client: as.srv.Client()}}
	f := framingFinding(t, s.checkAuthorizationServer(context.Background(), as.srv.URL))
	if f.Status != Skip || !strings.Contains(f.Message, "--probe-client-id") {
		t.Fatalf("no probe client: %+v, want SKIP naming --probe-client-id", f)
	}
	if len(as.queries) != 0 {
		t.Fatalf("AS was probed %d times without a client", len(as.queries))
	}
}

// TestFramingProbeStopsAtCallback: an AS that skips consent bounces to the
// probe's redirect_uri (127.0.0.1). The probe stops before that hop and WARNs.
func TestFramingProbeStopsAtCallback(t *testing.T) {
	as := newCimdAS(t, true)
	as.consent = false
	var leaked bool
	client := as.srv.Client()
	base := client.Transport
	client.Transport = framingTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "127.0.0.1" && strings.HasPrefix(r.URL.Path, "/ailang-mcp-check/") {
			leaked = true
		}
		return base.RoundTrip(r)
	})
	s := session{opts: Options{Target: "both", Client: client}}
	f := framingFinding(t, s.checkAuthorizationServer(context.Background(), as.srv.URL))
	if leaked {
		t.Fatal("probe followed the redirect to its own callback")
	}
	if f.Status != Warn || !strings.Contains(f.Message, "without a consent page") {
		t.Fatalf("consent-free AS: %+v, want WARN 'without a consent page'", f)
	}
}

// TestFramingProbeNoS256Skips: without S256 the authorization-server check
// already FAILs; the probe does not send a request the AS cannot accept.
func TestFramingProbeNoS256Skips(t *testing.T) {
	s := session{opts: Options{Target: "both", Client: http.DefaultClient}}
	f := s.checkAuthorizationFraming(context.Background(), authServerMetadata{
		AuthorizationEndpoint: "https://as.example/authorize", CIMD: true, AuthMeth: []string{"none"}})
	if f.Status != Skip || !strings.Contains(f.Message, "S256") {
		t.Fatalf("%+v, want SKIP naming S256", f)
	}
}

func TestServerInfoUnbrandedWarn(t *testing.T) {
	for _, tc := range []struct {
		init string
		want string
	}{
		{`{"serverInfo":{"name":"ailang-api","version":"0.57.2"}}`, Warn},
		{`{"serverInfo":{"name":"ailang-api","version":"0.8.1"}}`, Warn},
		{`{"serverInfo":{}}`, Warn},
		{`{}`, Warn},
		{`{"serverInfo":{"name":"parse","title":"AILANG Parse","version":"0.43.0"}}`, Pass},
	} {
		f := checkServerInfo(json.RawMessage(tc.init))
		if f.Check != "server-info" || f.Status != tc.want {
			t.Errorf("%s: %+v, want %s", tc.init, f, tc.want)
		}
		if tc.want == Warn && strings.Contains(tc.init, "ailang-api") && !strings.Contains(f.Message, "--server-info") {
			t.Errorf("%s: the WARN should name the fix: %q", tc.init, f.Message)
		}
	}
}

// mcpStub answers initialize, tools/list and resources/list over JSON.
func mcpStub(t *testing.T, resources string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&msg)
		if msg.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		result := map[string]string{
			"initialize":     `{"protocolVersion":"2025-06-18","capabilities":{"resources":{}},"serverInfo":{"name":"parse","version":"1"}}`,
			"tools/list":     `{"tools":[]}`,
			"resources/list": resources,
		}[msg.Method]
		w.Header().Set("Content-Type", "application/json")
		if result == "" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + result + `}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListedMetaModulesWarn(t *testing.T) {
	for _, tc := range []struct {
		name, resources, want string
	}{
		{"meta listed", `{"resources":[{"uri":"ailang://meta/modules","name":"AILANG Modules"},{"uri":"ui://w"}]}`, Warn},
		{"widgets only", `{"resources":[{"uri":"ui://w"}]}`, Pass},
		{"empty", `{"resources":[]}`, Pass},
		{"no resources method", ``, Skip},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := mcpStub(t, tc.resources)
			fs, err := Run(context.Background(), Options{URL: srv.URL, Target: "anthropic", Client: srv.Client()})
			if err != nil {
				t.Fatal(err)
			}
			var got *Finding
			for i := range fs {
				if fs[i].Check == "listed-resources" {
					got = &fs[i]
				}
			}
			if got == nil || got.Status != tc.want {
				t.Fatalf("listed-resources = %+v, want %s; all: %+v", got, tc.want, fs)
			}
			if Failed(fs) {
				t.Fatalf("a listing-hygiene finding must never FAIL the run: %+v", fs)
			}
		})
	}
}
