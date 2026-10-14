package apiserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// --mcp-connect-only (M-AILANG-EXECUTOR-GE P1b): a public service whose tools
// must never run without a token. Every way a tool is reachable on a normal
// server is checked here through the real route table: /mcp/, an @route, the
// /api/ catch-all and OpenAPI. Only /mcp/connect/ with a good token, an
// explicit @mcp_auth("noauth") tool, and @nomcp routes remain.

const connectOnlyModule = `module test/api/conn

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"

-- An executor tool: an @route, so normally also POST /api/exec/run.
@route("POST", "/api/exec/run")
export func runIt(code: string) -> string ! {IO} = "ran:${code}"

-- No annotation at all: normally POST /api/test/api/conn/openTool.
export func openTool(x: string) -> string ! {IO} = "open:${x}"

@mcp_auth("noauth")
export func docsTool(q: string) -> string ! {IO} = "docs:${q}"

-- OAuth plumbing: HTTP only.
@route("GET", "/oauth/meta")
@nomcp
export func oauthMeta() -> string ! {IO} = "meta"
`

func connectOnlyServer(t *testing.T, cfg Config) (*Server, string) {
	t.Helper()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "conn.ail")
	if err := os.WriteFile(modPath, []byte(connectOnlyModule), 0o644); err != nil {
		t.Fatal(err)
	}
	eff := effects.NewEffContext(nil)
	eff.Grant(effects.NewCapability("IO"))
	cfg.Port, cfg.EffCtx = "0", eff
	srv := New(tmpDir, cfg)
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv, modPath
}

func httpDo(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestMCPConnectOnly_OnlyGatedSurface(t *testing.T) {
	srv, _ := connectOnlyServer(t, Config{MCP: true, OAuthIssuer: gateIssuer, MCPConnectOnly: true})
	if err := srv.ValidateMCPConnectOnly(); err != nil {
		t.Fatalf("valid setup refused: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	listed := hs.URL + listedMCPPath

	// Discovery stays open: directory clients list tools before signing in.
	st, _, body := rpc(t, listed, "", "tools/list", map[string]any{})
	if st != 200 || !strings.Contains(body, `"runIt"`) || !strings.Contains(body, `"openTool"`) {
		t.Fatalf("tools/list: %d %s", st, body)
	}
	// Every tool without an explicit @mcp_auth is gated.
	for _, c := range []map[string]any{call("runIt", map[string]any{"code": "x"}), call("openTool", map[string]any{"x": "x"})} {
		if st, _, body := rpc(t, listed, "", "tools/call", c); st != 401 || strings.Contains(body, "ran:") || strings.Contains(body, "open:") {
			t.Errorf("%v without token: %d %s", c["name"], st, body)
		}
		if st, _, _ := rpc(t, listed, "bad", "tools/call", c); st != 401 {
			t.Errorf("%v bad token: %d", c["name"], st)
		}
	}
	if st, _, body := rpc(t, listed, "good", "tools/call", call("runIt", map[string]any{"code": "x"})); st != 200 || !strings.Contains(body, "ran:x") {
		t.Errorf("runIt good token: %d %s", st, body)
	}
	// An explicit "noauth" is the author's decision and is honoured.
	if st, _, body := rpc(t, listed, "", "tools/call", call("docsTool", map[string]any{"q": "q"})); st != 200 || !strings.Contains(body, "docs:q") {
		t.Errorf("noauth tool: %d %s", st, body)
	}

	// No agent surface: it is ungated, so it would bypass the gate.
	for _, p := range []string{"/mcp/", "/mcp"} {
		if st, _, body := rpc(t, hs.URL+p, "", "tools/call", call("runIt", map[string]any{"code": "x"})); st == 200 || strings.Contains(body, "ran:") {
			t.Errorf("%s served a tool: %d %s", p, st, body)
		}
	}
	// No plain-HTTP way to a tool: not its @route, not the /api/ catch-all.
	for _, u := range []string{"/api/exec/run", "/api/test/api/conn/runIt", "/api/test/api/conn/openTool"} {
		if st, body := httpDo(t, http.MethodPost, hs.URL+u, `{"code":"x","x":"x"}`); st == 200 || strings.Contains(body, "ran:") || strings.Contains(body, "open:") {
			t.Errorf("POST %s: %d %s", u, st, body)
		}
	}
	// OpenAPI does not advertise them either.
	if st, body := httpDo(t, http.MethodGet, hs.URL+"/api/_meta/openapi.json", ""); st != 200 || strings.Contains(body, "runIt") || strings.Contains(body, "openTool") {
		t.Errorf("openapi: %d lists a tool: %s", st, body)
	}
	// @nomcp routes (the OAuth plumbing) are still served.
	if st, body := httpDo(t, http.MethodGet, hs.URL+"/oauth/meta", ""); st != 200 || !strings.Contains(body, "meta") {
		t.Errorf("GET /oauth/meta: %d %s", st, body)
	}
}

// Control: the same module without the flag serves the tool on /mcp/ and over
// HTTP, so the assertions above are about the flag and not the fixture.
func TestMCPConnectOnly_ControlWithoutFlag(t *testing.T) {
	srv, _ := connectOnlyServer(t, Config{MCP: true, OAuthIssuer: gateIssuer})
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	if st, _, body := rpc(t, hs.URL+"/mcp/", "", "tools/call", call("runIt", map[string]any{"code": "x"})); st != 200 || !strings.Contains(body, "ran:x") {
		t.Errorf("/mcp/ runIt: %d %s", st, body)
	}
	if st, body := httpDo(t, http.MethodPost, hs.URL+"/api/exec/run", `{"code":"x"}`); st != 200 || !strings.Contains(body, "ran:x") {
		t.Errorf("POST /api/exec/run: %d %s", st, body)
	}
}

func TestMCPConnectOnly_RefusesUngatableSetup(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"no issuer", Config{MCP: true, MCPConnectOnly: true}, "--oauth-issuer"},
		{"no mcp-http", Config{OAuthIssuer: gateIssuer, MCPConnectOnly: true}, "--mcp-http"},
		{"with a2a", Config{MCP: true, A2A: true, OAuthIssuer: gateIssuer, MCPConnectOnly: true}, "--a2a"},
	}
	for _, c := range cases {
		srv, _ := connectOnlyServer(t, c.cfg)
		if err := srv.ValidateMCPConnectOnly(); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want mention of %s", c.name, err, c.want)
		}
	}

	// No verifier: the tools could never be authorized.
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	_ = os.MkdirAll(apiDir, 0o755)
	modPath := filepath.Join(apiDir, "conn.ail")
	_ = os.WriteFile(modPath, []byte("module test/api/conn\n\nexport func runIt(code: string) -> string ! {IO} = code\n"), 0o644)
	srv := New(tmpDir, Config{Port: "0", MCP: true, OAuthIssuer: gateIssuer, MCPConnectOnly: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatal(err)
	}
	if err := srv.ValidateMCPConnectOnly(); err == nil || !strings.Contains(err.Error(), "@mcp_token_verifier") {
		t.Errorf("no verifier: err = %v", err)
	}
}
