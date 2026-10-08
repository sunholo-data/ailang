package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// M-MCP-FILE-HANDOFF F1c: MCP Apps serving (@mcp_ui_resource, @mcp_ui,
// @mcp_app_only), end to end through buildRoutes.

var updateUIGolden = flag.Bool("update-ui-golden", false, "rewrite testdata/mcp_ui.golden.json")

const uiModule = `module test/api/widget

-- The upload widget.
@mcp_title("Upload a file")
@mcp_ui_resource("ui://test/upload", "self", "https://cdn.example.com")
export func uploadWidget() -> string = "<!doctype html><p>pick a file</p>"

-- Ask the user for a file.
@mcp_title("Choose a file")
@mcp_hints("readOnly")
@mcp_ui("ui://test/upload")
export func chooseFile() -> string = "widget shown"

-- Bytes from the widget, relayed by the host.
@mcp_title("Upload via host")
@mcp_hints("idempotent")
@mcp_ui("ui://test/upload")
@mcp_app_only
export func uploadViaHost(name: string, data: string) -> string = "got ${name}:${data}"

-- Echo the request headers the tool sees.
@mcp_title("Headers")
@mcp_hints("readOnly")
export func headerKeys(_headers: Json) -> Json = _headers

-- Mounts the listed surface (/mcp/connect/) too.
@mcp_title("Agent only")
@mcp_hints("readOnly")
@mcp_agent_only
export func agentOnly(x: string) -> string = x
`

func uiServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	tmpDir, modPath := writeModule(t, "widget", uiModule)
	srv := New(tmpDir, Config{Port: "0", MCP: true, NoFeedbackTool: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs, srv
}

// rpcResult posts one JSON-RPC call and returns its result member.
func rpcResult(t *testing.T, url, method string, params any, header map[string]string) any {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	m := rpcMessage(t, string(b))
	if m["error"] != nil {
		t.Fatalf("%s: %v", method, m["error"])
	}
	return m["result"]
}

// TestMCPUI_Golden pins resources/list, resources/read and tools/list on
// /mcp/. The test server's origin is written as http://SELF.
func TestMCPUI_Golden(t *testing.T) {
	hs, _ := uiServer(t)
	url := hs.URL + "/mcp/"
	tools := rpcResult(t, url, "tools/list", map[string]any{}, nil).(map[string]any)["tools"].([]any)
	var uiTools []any
	for _, name := range []string{"chooseFile", "uploadViaHost"} {
		uiTools = append(uiTools, toolNamed(tools, name))
	}
	doc := map[string]any{
		"resources/list": rpcResult(t, url, "resources/list", map[string]any{}, nil),
		"resources/read": rpcResult(t, url, "resources/read", map[string]any{"uri": "ui://test/upload"}, nil),
		"tools/list":     uiTools,
	}
	got, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(bytes.ReplaceAll(got, []byte(hs.URL), []byte("http://SELF")), '\n')
	golden := filepath.Join("testdata", "mcp_ui.golden.json")
	if *updateUIGolden {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MCP Apps JSON drifted from %s:\n%s", golden, got)
	}
}

// TestMCPUI_AppOnlyToolCallable: the widget-only tool is listed with
// visibility ["app"] (the host hides it from the model) and still answers a
// tools/call, which is how the widget reaches it.
func TestMCPUI_AppOnlyToolCallable(t *testing.T) {
	hs, _ := uiServer(t)
	for _, surface := range []string{"/mcp/", listedMCPPath} {
		tools := rpcResult(t, hs.URL+surface, "tools/list", map[string]any{}, nil).(map[string]any)["tools"].([]any)
		vis, _ := json.Marshal(toolNamed(tools, "uploadViaHost")["_meta"].(map[string]any)["ui"].(map[string]any)["visibility"])
		if string(vis) != `["app"]` {
			t.Errorf("%s: visibility = %s", surface, vis)
		}
		if _, has := toolNamed(tools, "chooseFile")["_meta"].(map[string]any)["ui"].(map[string]any)["visibility"]; has {
			t.Errorf("%s: chooseFile must keep the default visibility", surface)
		}
		if toolNamed(tools, "uploadWidget") != nil {
			t.Errorf("%s: the HTML function must not be a tool", surface)
		}
		res := rpcResult(t, hs.URL+surface, "tools/call", call("uploadViaHost", map[string]any{"name": "a.docx", "data": "UEsDBA=="}), nil)
		if b, _ := json.Marshal(res); !strings.Contains(string(b), "got a.docx:UEsDBA==") {
			t.Errorf("%s: widget-only tool call: %s", surface, b)
		}
	}
}

// TestMCPUI_SelfIsServerOriginNotClientHeader: "self" is the origin the
// client reached; a client cannot set it through the internal header, and the
// internal header never reaches a _headers binding.
func TestMCPUI_SelfIsServerOriginNotClientHeader(t *testing.T) {
	hs, _ := uiServer(t)
	forged := map[string]string{publicBaseHeader: "https://evil.example"}
	read := rpcResult(t, hs.URL+"/mcp/", "resources/read", map[string]any{"uri": "ui://test/upload"}, forged)
	b, _ := json.Marshal(read)
	if strings.Contains(string(b), "evil.example") || !strings.Contains(string(b), hs.URL) {
		t.Fatalf("self resolved wrongly: %s", b)
	}
	res := rpcResult(t, hs.URL+"/mcp/", "tools/call", call("headerKeys", map[string]any{}), forged)
	if b, _ := json.Marshal(res); strings.Contains(strings.ToLower(string(b)), strings.ToLower(publicBaseHeader)) {
		t.Fatalf("_headers saw the internal header: %s", b)
	}
}

// TestMCPUI_SelfNeedsHTTP: without an HTTP request (stdio, in-memory) "self"
// cannot be resolved, so reading the widget fails loudly.
func TestMCPUI_SelfNeedsHTTP(t *testing.T) {
	_, srv := uiServer(t)
	ms := NewMCPServer(srv)
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := ms.mcpServer.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "ui://test/upload"}); err == nil || !strings.Contains(err.Error(), "needs an HTTP request") {
		t.Fatalf("stdio read of a self widget: %v", err)
	}
}

// TestMCPUI_LoadErrors: every bad MCP Apps declaration stops LoadModules.
func TestMCPUI_LoadErrors(t *testing.T) {
	const res = "@mcp_ui_resource(\"ui://t/w\")\nexport func w() -> string = \"<p>\"\n\n"
	cases := []struct{ name, src, want string }{
		{"unknown ui", res + "@mcp_ui(\"ui://t/nope\")\nexport func f() -> string = \"x\"", `@mcp_ui("ui://t/nope") names no @mcp_ui_resource (declared: ui://t/w)`},
		{"ui not ui://", res + "@mcp_ui(\"https://t/w\")\nexport func f() -> string = \"x\"", `the URI must start with ui://`},
		{"resource not ui://", "@mcp_ui_resource(\"https://t/w\")\nexport func w() -> string = \"<p>\"", `the URI must be ui://<service>/<name>`},
		{"resource takes args", "@mcp_ui_resource(\"ui://t/w\")\nexport func w(x: string) -> string = x", `must take no arguments`},
		{"resource not string", "@mcp_ui_resource(\"ui://t/w\")\nexport func w() -> int = 1", `must return string`},
		{"bad domain", "@mcp_ui_resource(\"ui://t/w\", \"api.example.com\")\nexport func w() -> string = \"<p>\"", `connect domain "api.example.com"`},
		{"domain with path", "@mcp_ui_resource(\"ui://t/w\", \"https://api.example.com/x\")\nexport func w() -> string = \"<p>\"", `an origin has no path`},
		{"duplicate uri", res + "@mcp_ui_resource(\"ui://t/w\")\nexport func w2() -> string = \"<p>\"", `is declared twice`},
		{"ui on resource", "@mcp_ui_resource(\"ui://t/w\")\n@mcp_app_only\nexport func w() -> string = \"<p>\"", `is not a tool`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmpDir, modPath := writeModule(t, "bad", "module test/api/bad\n\n"+c.src+"\n")
			srv := New(tmpDir, Config{Port: "0", MCP: true})
			defer srv.Close()
			err := srv.LoadModules([]string{modPath})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("LoadModules error = %v, want it to contain %q", err, c.want)
			}
		})
	}
}
