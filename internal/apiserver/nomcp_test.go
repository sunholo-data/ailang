package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/ast"
)

// M-SERVEAPI-NOMCP M1
//
// @nomcp is a parameterless annotation that hides an export from the MCP tool
// surface ONLY (tools/list and tools/call). HTTP, OpenAPI, and the A2A card are
// unaffected. Unlike @noexpose, @nomcp is NOT reset by @route: a @route @nomcp
// handler is served over HTTP (200) and appears in OpenAPI while being absent
// from MCP. IsNoExpose and IsNoMCP are independent flags.

// --- Extractor unit test (mirrors TestExtractNoExposeAnnotations) ---

func TestExtractNoMCPAnnotations(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/billing",
		Exports: []ExportInfo{
			// @route @nomcp — the flag must survive @route (no RoutePath gate).
			{Name: "getKeyUsage", Type: "string -> Json", Arity: 1, RoutePath: "/api/v1/keys", RouteMethod: "GET"},
			// bare @nomcp — flag set unconditionally.
			{Name: "requestHistory", Type: "string -> Json", Arity: 1},
			// @noexpose only — must NOT get IsNoMCP (flags independent).
			{Name: "internalHelper", Type: "int -> int", Arity: 1},
			// plain export — neither flag.
			{Name: "publicHelper", Type: "int -> int", Arity: 1},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{Name: "getKeyUsage", IsExport: true, Annotations: []*ast.Annotation{
				{Name: "route"}, {Name: "nomcp"},
			}},
			{Name: "requestHistory", IsExport: true, Annotations: []*ast.Annotation{{Name: "nomcp"}}},
			{Name: "internalHelper", IsExport: true, Annotations: []*ast.Annotation{{Name: "noexpose"}}},
			{Name: "publicHelper", IsExport: true},
		},
	}

	// Emulate the load order: @route override runs first (resets IsNoExpose),
	// then noexpose, then nomcp.
	extractNoExposeAnnotations(modInfo, file)
	extractNoMCPAnnotations(modInfo, file)

	// getKeyUsage: @route @nomcp — IsNoMCP set UNCONDITIONALLY (survives @route).
	if !modInfo.Exports[0].IsNoMCP {
		t.Error("getKeyUsage (@route @nomcp) should keep IsNoMCP=true — @route must NOT reset it")
	}

	// requestHistory: bare @nomcp — flag set.
	if !modInfo.Exports[1].IsNoMCP {
		t.Error("requestHistory (@nomcp) should be marked IsNoMCP=true")
	}

	// internalHelper: @noexpose only — must NOT get IsNoMCP (flags independent).
	if modInfo.Exports[2].IsNoMCP {
		t.Error("internalHelper (@noexpose only) must NOT be marked IsNoMCP")
	}
	if !modInfo.Exports[2].IsNoExpose {
		t.Error("internalHelper (@noexpose, no @route) should be IsNoExpose=true")
	}

	// publicHelper: neither flag.
	if modInfo.Exports[3].IsNoMCP {
		t.Error("publicHelper should NOT be marked IsNoMCP")
	}
}

// TestNoMCPAndNoExposeIndependent asserts the two flags do not interact:
// a @route @noexpose export gets IsNoExpose=false (existing @route override)
// while a @route @nomcp export keeps IsNoMCP=true.
func TestNoMCPAndNoExposeIndependent(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/x",
		Exports: []ExportInfo{
			{Name: "routeNoExpose", Type: "int -> int", Arity: 1, RoutePath: "/a", RouteMethod: "GET"},
			{Name: "routeNoMCP", Type: "int -> int", Arity: 1, RoutePath: "/b", RouteMethod: "GET"},
		},
	}
	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{Name: "routeNoExpose", IsExport: true, Annotations: []*ast.Annotation{{Name: "route"}, {Name: "noexpose"}}},
			{Name: "routeNoMCP", IsExport: true, Annotations: []*ast.Annotation{{Name: "route"}, {Name: "nomcp"}}},
		},
	}

	extractNoExposeAnnotations(modInfo, file)
	extractNoMCPAnnotations(modInfo, file)

	// @route @noexpose: @route wins → IsNoExpose stays false (visible over HTTP).
	if modInfo.Exports[0].IsNoExpose {
		t.Error("@route @noexpose: @route override should keep IsNoExpose=false")
	}
	if modInfo.Exports[0].IsNoMCP {
		t.Error("@route @noexpose should not set IsNoMCP")
	}

	// @route @nomcp: IsNoMCP survives @route (MCP-only exclusion).
	if !modInfo.Exports[1].IsNoMCP {
		t.Error("@route @nomcp should keep IsNoMCP=true")
	}
	if modInfo.Exports[1].IsNoExpose {
		t.Error("@route @nomcp should not set IsNoExpose")
	}
}

// nomcpTestServer boots a real Server loaded with a module exercising three
// annotation states: a plain @route (visible everywhere), a @route @nomcp
// (HTTP/OpenAPI yes, MCP no), and a @noexpose (hidden everywhere).
func nomcpTestServer(t *testing.T) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	modContent := `module test/api/keys

@route("GET", "/api/v1/status")
export pure func status() -> string = "ok"

@route("GET", "/api/v1/keys")
@nomcp
export pure func getKeyUsage() -> string = "usage"

@noexpose
export pure func internalSecret() -> string = "secret"
`
	modPath := filepath.Join(apiDir, "keys.ail")
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		t.Fatal(err)
	}

	stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
	if stdlibPath == "" {
		cwd, _ := os.Getwd()
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				stdlibPath = dir
				break
			}
		}
	}
	if stdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", stdlibPath)
	}

	srv := New(tmpDir, Config{Port: "0", CORS: true})
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv
}

// findExport locates an export by name across all loaded modules.
func findExport(t *testing.T, srv *Server, name string) ExportInfo {
	t.Helper()
	for _, mod := range srv.GetModules() {
		for _, exp := range mod.Exports {
			if exp.Name == name {
				return exp
			}
		}
	}
	t.Fatalf("export %q not found", name)
	return ExportInfo{}
}

// TestNoMCP_FlagSetOnLoad verifies the annotation survives the real load path
// (parser → extractNoMCPAnnotations) with the @route override in play.
func TestNoMCP_FlagSetOnLoad(t *testing.T) {
	srv := nomcpTestServer(t)
	defer srv.Close()

	getKeyUsage := findExport(t, srv, "getKeyUsage")
	if !getKeyUsage.IsNoMCP {
		t.Error("getKeyUsage (@route @nomcp) should have IsNoMCP=true after load")
	}
	if getKeyUsage.IsNoExpose {
		t.Error("getKeyUsage should NOT be IsNoExpose (it has @route)")
	}
	if getKeyUsage.RoutePath != "/api/v1/keys" {
		t.Errorf("getKeyUsage RoutePath = %q, want /api/v1/keys", getKeyUsage.RoutePath)
	}

	status := findExport(t, srv, "status")
	if status.IsNoMCP {
		t.Error("status (plain @route) should NOT have IsNoMCP")
	}
}

// TestNoMCP_HiddenFromMCPToolSurface is the E2E MCP enumeration test. It wires
// a go-sdk in-memory client session and asserts the @nomcp tool is absent from
// tools/list and errors on tools/call, while the plain @route tool is present.
func TestNoMCP_HiddenFromMCPToolSurface(t *testing.T) {
	srv := nomcpTestServer(t)
	defer srv.Close()

	ms := NewMCPServer(srv)

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Server must connect before the client (client initializes the session).
	serverSession, err := ms.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}

	// Plain @route export IS present in the MCP tool surface.
	if !names["status"] {
		t.Errorf("plain @route tool 'status' should be in tools/list; got tools: %v", toolNames(res.Tools))
	}
	// @route @nomcp export is ABSENT from the MCP tool surface.
	if names["getKeyUsage"] {
		t.Errorf("@nomcp tool 'getKeyUsage' must NOT be in tools/list; got tools: %v", toolNames(res.Tools))
	}
	// @noexpose export is hidden from MCP too (regression).
	if names["internalSecret"] {
		t.Errorf("@noexpose tool 'internalSecret' must NOT be in tools/list; got tools: %v", toolNames(res.Tools))
	}

	// tools/call on the @nomcp tool name errors (unregistered).
	_, callErr := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "getKeyUsage",
		Arguments: map[string]any{},
	})
	if callErr == nil {
		t.Error("CallTool on @nomcp tool 'getKeyUsage' should error (unregistered)")
	}
}

func toolNames(tools []*mcp.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

// TestNoMCP_StillServedOverHTTPAndOpenAPI asserts the @route @nomcp handler is
// unaffected on the HTTP surface: it answers 200 and appears in the OpenAPI
// spec. @noexpose remains hidden from HTTP (regression).
func TestNoMCP_StillServedOverHTTPAndOpenAPI(t *testing.T) {
	srv := nomcpTestServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	// @route @nomcp handler still answers HTTP 200.
	req := httptest.NewRequest("GET", "/api/v1/keys", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("@nomcp route /api/v1/keys should answer 200 over HTTP, got %d: %s", w.Code, w.Body.String())
	}

	// @route @nomcp handler present in the OpenAPI spec.
	oreq := httptest.NewRequest("GET", "/api/_meta/openapi.json", nil)
	ow := httptest.NewRecorder()
	mux.ServeHTTP(ow, oreq)
	if ow.Code != http.StatusOK {
		t.Fatalf("openapi.json should answer 200, got %d", ow.Code)
	}
	spec := ow.Body.String()
	if !strings.Contains(spec, "/api/v1/keys") {
		t.Errorf("@nomcp route /api/v1/keys should be present in OpenAPI spec")
	}

	// @noexpose export stays hidden from HTTP (regression): no route registered.
	nreq := httptest.NewRequest("POST", "/test/api/keys/internalSecret", nil)
	nw := httptest.NewRecorder()
	mux.ServeHTTP(nw, nreq)
	if nw.Code == http.StatusOK {
		t.Error("@noexpose export 'internalSecret' must NOT be reachable over HTTP")
	}
}
