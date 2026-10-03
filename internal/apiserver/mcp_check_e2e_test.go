package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mcpcheck"
)

// M-SERVEAPI-DIRECTORY-READY M4: `ailang mcp check` against real serve-api
// surfaces. A directory-ready module on /mcp/connect/ passes every check; the
// same module's /mcp/ fails exactly as today's prod Parse /mcp/ does (a
// credential parameter, and no OAuth gate).

const readyModule = `module test/api/ready

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"

-- Parse a document (needs a signed-in account).
@mcp_title("Parse document")
@mcp_hints("readOnly", "openWorld")
@mcp_auth("oauth2")
@mcp_secret("apiKey")
@optional("apiKey")
export func parseDoc(doc: string, apiKey: string) -> string ! {IO} = "parsed:${doc}"

-- List supported formats.
@mcp_title("List formats")
export func formats() -> string = "docx,pdf"

-- Device sign-in for agents (not for the directory).
@mcp_title("Sign in with a device code")
@mcp_hints("openWorld")
@mcp_agent_only
export func deviceLogin(label: string) -> string ! {IO} = label
`

// authServer serves RFC 8414 metadata; s256 false omits S256 support.
func authServer(t *testing.T, s256 bool) *httptest.Server {
	t.Helper()
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server" {
			http.NotFound(w, r)
			return
		}
		methods := []string{"plain"}
		if s256 {
			methods = []string{"S256"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                "http://" + r.Host,
			"code_challenge_methods_supported":      methods,
			"client_id_metadata_document_supported": true,
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	}))
	t.Cleanup(as.Close)
	return as
}

func readyServer(t *testing.T, issuer string) *httptest.Server {
	t.Helper()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "ready.ail")
	if err := os.WriteFile(modPath, []byte(readyModule), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := New(tmpDir, Config{Port: "0", MCP: true, OAuthIssuer: issuer})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs
}

func runCheck(t *testing.T, url, target string) []mcpcheck.Finding {
	t.Helper()
	fs, err := mcpcheck.Run(context.Background(), mcpcheck.Options{URL: url, Target: target})
	if err != nil {
		t.Fatalf("mcpcheck.Run(%s): %v", url, err)
	}
	return fs
}

func statusOf(fs []mcpcheck.Finding, check string) string {
	worst := ""
	for _, f := range fs {
		if f.Check != check {
			continue
		}
		if f.Status == mcpcheck.Fail || worst == "" {
			worst = f.Status
		}
	}
	return worst
}

func TestMCPCheck_ListedSurfaceReady(t *testing.T) {
	as := authServer(t, true)
	hs := readyServer(t, as.URL)
	fs := runCheck(t, hs.URL+listedMCPPath, "anthropic")
	for _, c := range []string{"annotations", "credentials", "zero-arg", "lazy-auth", "authorization-server"} {
		if got := statusOf(fs, c); got != mcpcheck.Pass {
			t.Errorf("%s = %s, want PASS; findings: %+v", c, got, fs)
		}
	}
	if mcpcheck.Failed(fs) {
		t.Fatalf("directory-ready surface failed: %+v", fs)
	}
}

func TestMCPCheck_AgentSurfaceFailsLikeProdParse(t *testing.T) {
	as := authServer(t, true)
	hs := readyServer(t, as.URL)
	fs := runCheck(t, hs.URL+"/mcp/", "anthropic")
	if statusOf(fs, "credentials") != mcpcheck.Fail || statusOf(fs, "lazy-auth") != mcpcheck.Fail {
		t.Fatalf("/mcp/ should fail credentials and lazy-auth: %+v", fs)
	}
	var msg strings.Builder
	for _, f := range fs {
		msg.WriteString(f.Message)
	}
	if !strings.Contains(msg.String(), `"apiKey"`) {
		t.Errorf("credentials finding should name apiKey: %+v", fs)
	}
}

func TestMCPCheck_AuthServerWithoutS256Fails(t *testing.T) {
	as := authServer(t, false)
	hs := readyServer(t, as.URL)
	fs := runCheck(t, hs.URL+listedMCPPath, "anthropic")
	if statusOf(fs, "authorization-server") != mcpcheck.Fail {
		t.Fatalf("missing S256 must fail: %+v", fs)
	}
}
