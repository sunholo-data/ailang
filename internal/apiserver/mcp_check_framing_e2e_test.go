package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mcpcheck"
)

// M-MCP-OAUTH-CLIENT-MODELS M3c, end to end: `ailang mcp check` against the
// readyModule listed surface with a CIMD-only authorization server (the
// sunholo/mcp_oauth shape). Before M3c the framing probe sent the non-URL
// client_id ailang-mcp-check-dummy, the AS answered 400, and every CIMD-only
// server got a false authorization-framing WARN (design V20).

// cimdOnlyAS serves RFC 8414 metadata advertising CIMD for public clients
// and an authorization endpoint that accepts only a CIMD client whose
// document (read from docs/static, standing in for the docs site) lists the
// presented redirect_uri, with a 43-character S256 challenge; anything else
// is a 400. An accepted request redirects to a framing-protected login page.
func cimdOnlyAS(t *testing.T) *httptest.Server {
	t.Helper()
	docBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "static", "mcp-check", "client-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ClientID     string   `json:"client_id"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		t.Fatal(err)
	}
	var as *httptest.Server
	as = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                as.URL,
				"authorization_endpoint":                as.URL + "/oauth/authorize",
				"code_challenge_methods_supported":      []string{"S256"},
				"client_id_metadata_document_supported": true,
				"token_endpoint_auth_methods_supported": []string{"none"},
			})
		case "/oauth/authorize":
			q := r.URL.Query()
			redirect, _ := url.Parse(q.Get("redirect_uri"))
			if redirect != nil && redirect.Hostname() == "127.0.0.1" {
				redirect.Host = "127.0.0.1" // loopback: any port (RFC 8252)
			}
			ok := strings.HasPrefix(q.Get("client_id"), "https://") && q.Get("client_id") == doc.ClientID &&
				redirect != nil && len(doc.RedirectURIs) == 1 && redirect.String() == doc.RedirectURIs[0] &&
				q.Get("response_type") == "code" && q.Get("code_challenge_method") == "S256" &&
				len(q.Get("code_challenge")) == 43 && q.Get("state") != ""
			if !ok {
				http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, "/login?handle=h", http.StatusFound)
		case "/login":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
			_, _ = w.Write([]byte("<!doctype html><p>sign in</p>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(as.Close)
	return as
}

func findingOf(fs []mcpcheck.Finding, check string) mcpcheck.Finding {
	for _, f := range fs {
		if f.Check == check {
			return f
		}
	}
	return mcpcheck.Finding{}
}

// TestMCPCheck_CimdOnlyFramingPass is success metric 6: --target both on a
// CIMD-only listed surface reports authorization-framing PASS, the listed
// surface carries no module inventory, and a branded --server-info passes.
func TestMCPCheck_CimdOnlyFramingPass(t *testing.T) {
	as := cimdOnlyAS(t)
	id, err := ParseServerInfo("name=parse,title=AILANG Parse,version=0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	hs := readyServerWith(t, Config{OAuthIssuer: as.URL, ServerInfo: id})
	fs := runCheck(t, hs.URL+listedMCPPath, "both")
	for check, want := range map[string]string{
		"authorization-framing": mcpcheck.Pass,
		"authorization-server":  mcpcheck.Pass,
		"listed-resources":      mcpcheck.Pass,
		"server-info":           mcpcheck.Pass,
	} {
		if f := findingOf(fs, check); f.Status != want {
			t.Errorf("%s = %+v, want %s", check, f, want)
		}
	}
	if !strings.Contains(findingOf(fs, "authorization-framing").Message, mcpcheck.ProbeClientURL) {
		t.Errorf("framing finding should name the probe client: %+v", findingOf(fs, "authorization-framing"))
	}
}

// TestMCPCheck_DefaultServerInfoWarns: without --server-info the listed
// surface is unbranded (ailang-api), a WARN that never fails the run; the
// agent surface /mcp/ still carries the inventory, which the check reports.
func TestMCPCheck_DefaultServerInfoWarns(t *testing.T) {
	as := cimdOnlyAS(t)
	hs := readyServer(t, as.URL)
	fs := runCheck(t, hs.URL+listedMCPPath, "anthropic")
	if f := findingOf(fs, "server-info"); f.Status != mcpcheck.Warn || !strings.Contains(f.Message, "--server-info") {
		t.Errorf("server-info = %+v, want WARN naming --server-info", f)
	}
	if mcpcheck.Failed(fs) {
		t.Fatalf("listed surface must not fail: %+v", fs)
	}
	agent := runCheck(t, hs.URL+"/mcp/", "anthropic")
	if f := findingOf(agent, "listed-resources"); f.Status != mcpcheck.Warn {
		t.Errorf("/mcp/ listed-resources = %+v, want WARN (the agent surface keeps the inventory)", f)
	}
}
