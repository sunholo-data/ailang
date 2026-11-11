package apiserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// M-MCP-OAUTH-CLIENT-MODELS M3a/M3b (D7, D8): serverInfo comes from
// --server-info, and ailang://meta/modules is off /mcp/connect/, projected on
// /mcp/, and turned off by --no-introspection.

// v18Module is design row V18's live repro: a @noexpose helper, the token
// verifier, an @mcp_agent_only tool and a gated tool. On v0.57.1 the listed
// surface's ailang://meta/modules listed all four names, is_no_expose twice,
// and the module's absolute path.
const v18Module = `module test/api/v18

@noexpose
export func secretHelper(x: string) -> string = "secret:${x}"

@mcp_token_verifier
export func verifyTok(token: string) -> bool ! {IO} = token == "good"

-- Sign in with a device code.
@mcp_title("Device login")
@mcp_hints("readOnly")
@mcp_agent_only
export func deviceLogin(label: string) -> string = label

-- Parse a document.
@mcp_title("Parse document")
@mcp_hints("readOnly")
@mcp_auth("oauth2")
export func parseDoc(doc: string) -> string = "parsed:${doc}"
`

func v18Server(t *testing.T, cfg Config) (*httptest.Server, string) {
	t.Helper()
	tmpDir, modPath := writeModule(t, "v18", v18Module)
	cfg.Port, cfg.MCP, cfg.NoFeedbackTool = "0", true, true
	if cfg.OAuthIssuer == "" {
		cfg.OAuthIssuer = gateIssuer
	}
	srv := New(tmpDir, cfg)
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs, tmpDir
}

// rpcRaw posts one JSON-RPC call and returns the decoded message (result or
// error), unlike rpcResult, which fails the test on a JSON-RPC error.
func rpcRaw(t *testing.T, url, method string, params any) map[string]any {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return rpcMessage(t, string(b))
}

func resourceURIs(t *testing.T, url string) []string {
	t.Helper()
	res := rpcResult(t, url, "resources/list", map[string]any{}, nil).(map[string]any)
	var uris []string
	for _, r := range res["resources"].([]any) {
		uris = append(uris, r.(map[string]any)["uri"].(string))
	}
	return uris
}

func hasURI(uris []string, uri string) bool {
	for _, u := range uris {
		if u == uri {
			return true
		}
	}
	return false
}

// TestListedSurfaceNoMetaModules: /mcp/connect/ neither lists nor serves the
// module inventory (D8), with or without --no-introspection.
func TestListedSurfaceNoMetaModules(t *testing.T) {
	for _, noIntro := range []bool{false, true} {
		hs, _ := v18Server(t, Config{NoIntrospection: noIntro})
		url := hs.URL + listedMCPPath
		if uris := resourceURIs(t, url); hasURI(uris, metaModulesURI) {
			t.Fatalf("no-introspection=%v: listed surface lists %s: %v", noIntro, metaModulesURI, uris)
		}
		msg := rpcRaw(t, url, "resources/read", map[string]any{"uri": metaModulesURI})
		if msg["error"] == nil {
			b, _ := json.Marshal(msg["result"])
			t.Fatalf("no-introspection=%v: listed surface serves %s: %s", noIntro, metaModulesURI, b)
		}
	}
}

// TestAgentSurfaceMetaModulesExposedView: /mcp/ still serves the inventory,
// projected like /api/_meta/modules: no @noexpose export, no token verifier,
// no absolute path. The agent surface carries @mcp_agent_only tools, so they
// stay listed there.
func TestAgentSurfaceMetaModulesExposedView(t *testing.T) {
	hs, tmpDir := v18Server(t, Config{})
	url := hs.URL + "/mcp/"
	if uris := resourceURIs(t, url); !hasURI(uris, metaModulesURI) {
		t.Fatalf("/mcp/ should list %s: %v", metaModulesURI, uris)
	}
	res := rpcResult(t, url, "resources/read", map[string]any{"uri": metaModulesURI}, nil).(map[string]any)
	text := res["contents"].([]any)[0].(map[string]any)["text"].(string)
	for _, leak := range []string{"secretHelper", "verifyTok", "is_no_expose", filepath.ToSlash(tmpDir), tmpDir} {
		if strings.Contains(filepath.ToSlash(text), filepath.ToSlash(leak)) {
			t.Errorf("/mcp/ %s leaks %q:\n%s", metaModulesURI, leak, text)
		}
	}
	var doc ModulesListResponse
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("content is not a module list: %v\n%s", err, text)
	}
	var names []string
	for _, m := range doc.Modules {
		if m.Path != "test/api/v18" {
			t.Errorf("module path %q, want the relative test/api/v18", m.Path)
		}
		for _, e := range m.Exports {
			names = append(names, e.Name)
		}
	}
	if strings.Join(names, ",") != "deviceLogin,parseDoc" || doc.Count != 1 {
		t.Fatalf("exports = %v (count %d), want deviceLogin,parseDoc in one module", names, doc.Count)
	}
}

// TestAgentSurfaceMetaModulesHidesRoutedVerifier: a verifier with an @route
// is an HTTP endpoint (so exposedView keeps it), but the MCP inventory still
// omits it: its name says what to guess tokens against.
func TestAgentSurfaceMetaModulesHidesRoutedVerifier(t *testing.T) {
	src := strings.Replace(v18Module, "@mcp_token_verifier\n", "@mcp_token_verifier\n@route(\"POST\", \"/api/verify\")\n", 1)
	tmpDir, modPath := writeModule(t, "v18", src)
	srv := New(tmpDir, Config{Port: "0", MCP: true, NoFeedbackTool: true, OAuthIssuer: gateIssuer})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	var list ModulesListResponse
	getJSON(t, hs.URL+"/api/_meta/modules", &list)
	if b, _ := json.Marshal(list); !strings.Contains(string(b), "verifyTok") {
		t.Fatalf("control: the routed verifier should be on /api/_meta/modules: %s", b)
	}
	res := rpcResult(t, hs.URL+"/mcp/", "resources/read", map[string]any{"uri": metaModulesURI}, nil).(map[string]any)
	if text := res["contents"].([]any)[0].(map[string]any)["text"].(string); strings.Contains(text, "verifyTok") {
		t.Fatalf("/mcp/ inventory names the routed verifier:\n%s", text)
	}
}

// TestNoIntrospectionDropsMetaModules: --no-introspection turns the resource
// off on /mcp/ too; the resources capability stays (ui:// widgets need it).
func TestNoIntrospectionDropsMetaModules(t *testing.T) {
	hs, _ := v18Server(t, Config{NoIntrospection: true})
	url := hs.URL + "/mcp/"
	if uris := resourceURIs(t, url); hasURI(uris, metaModulesURI) {
		t.Fatalf("--no-introspection: /mcp/ lists %s: %v", metaModulesURI, uris)
	}
	if msg := rpcRaw(t, url, "resources/read", map[string]any{"uri": metaModulesURI}); msg["error"] == nil {
		t.Fatalf("--no-introspection: /mcp/ serves %s", metaModulesURI)
	}
	init := rpcResult(t, url, "initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "t", "version": "1"},
	}, nil).(map[string]any)
	if caps := init["capabilities"].(map[string]any); caps["resources"] == nil {
		t.Fatalf("resources capability dropped: %v", caps)
	}
}

func initializeServerInfo(t *testing.T, url string) map[string]any {
	t.Helper()
	init := rpcResult(t, url, "initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "t", "version": "1"},
	}, nil).(map[string]any)
	return init["serverInfo"].(map[string]any)
}

// TestServerInfoFlag: --server-info sets name, title and version on both MCP
// surfaces, the A2A card version and OpenAPI info.version (D7).
func TestServerInfoFlag(t *testing.T) {
	id, err := ParseServerInfo("name=parse,title=AILANG Parse,version=0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	hs, _ := v18Server(t, Config{ServerInfo: id, A2A: true})
	for _, path := range []string{"/mcp/", listedMCPPath} {
		si := initializeServerInfo(t, hs.URL+path)
		if si["name"] != "parse" || si["title"] != "AILANG Parse" || si["version"] != "0.43.0" {
			t.Errorf("%s serverInfo = %v", path, si)
		}
	}
	var card map[string]any
	getJSON(t, hs.URL+"/.well-known/agent.json", &card)
	if card["version"] != "0.43.0" {
		t.Errorf("A2A card version = %v", card["version"])
	}
	var spec map[string]any
	getJSON(t, hs.URL+"/api/_meta/openapi.json", &spec)
	if v := spec["info"].(map[string]any)["version"]; v != "0.43.0" {
		t.Errorf("OpenAPI info.version = %v", v)
	}
}

// TestServerInfoDefault: with no flag the name stays ailang-api, no title is
// sent, and the version is the binary's, never the stale 0.8.1.
func TestServerInfoDefault(t *testing.T) {
	hs, _ := v18Server(t, Config{A2A: true})
	want := DefaultServerInfo().Version
	if want == "" || want == "0.8.1" {
		t.Fatalf("default version %q", want)
	}
	for _, path := range []string{"/mcp/", listedMCPPath} {
		si := initializeServerInfo(t, hs.URL+path)
		if si["name"] != "ailang-api" || si["version"] != want || si["title"] != nil {
			t.Errorf("%s serverInfo = %v, want ailang-api %s with no title", path, si, want)
		}
	}
	var card map[string]any
	getJSON(t, hs.URL+"/.well-known/agent.json", &card)
	if card["version"] != want {
		t.Errorf("A2A card version = %v, want %s", card["version"], want)
	}
	var spec map[string]any
	getJSON(t, hs.URL+"/api/_meta/openapi.json", &spec)
	if v := spec["info"].(map[string]any)["version"]; v != want {
		t.Errorf("OpenAPI info.version = %v, want %s", v, want)
	}
}

func TestServerInfoParse(t *testing.T) {
	def := DefaultServerInfo()
	for _, tc := range []struct {
		in   string
		want ServerIdentity
		err  string
	}{
		{"", def, ""},
		{"name=parse", ServerIdentity{Name: "parse", Version: def.Version}, ""},
		{"title=AILANG Parse, version=1.2.3", ServerIdentity{Name: def.Name, Title: "AILANG Parse", Version: "1.2.3"}, ""},
		{"name=parse,name=other", ServerIdentity{}, "twice"},
		{"name=", ServerIdentity{}, "empty"},
		{"colour=red", ServerIdentity{}, "unknown key"},
		{"parse", ServerIdentity{}, "key=value"},
		{"name=a\nb", ServerIdentity{}, "control"},
		{"name=x,icon=/icon.png", ServerIdentity{Name: "x", Version: def.Version, Icon: "/icon.png"}, ""},
		{"icon=https://cdn.example/i.svg", ServerIdentity{Name: def.Name, Version: def.Version, Icon: "https://cdn.example/i.svg"}, ""},
		{"icon=http://cdn.example/i.svg", ServerIdentity{}, "https"},
		{"icon=icon.png", ServerIdentity{}, "path on this server"},
	} {
		got, err := ParseServerInfo(tc.in)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("ParseServerInfo(%q) err = %v, want %q", tc.in, err, tc.err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseServerInfo(%q) = %+v, %v; want %+v", tc.in, got, err, tc.want)
		}
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
}
