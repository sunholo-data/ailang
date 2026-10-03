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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// M-SERVEAPI-DIRECTORY-READY M2: one process serves the agent surface at
// /mcp/ and the directory projection at /mcp/connect/, built from the same
// exports through the real route table (buildRoutes).

const listedModule = `module test/api/listed

@mcp_secret("apiKey")
@optional("apiKey")
export func whoami(name: string, apiKey: string) -> string ! {IO} = "${name}:${apiKey}"

@mcp_agent_only
export func deviceLogin(label: string) -> string ! {IO} = label

export func openTool(x: string) -> string ! {IO} = x
`

func routedServer(t *testing.T, src string, cfg Config) *httptest.Server {
	t.Helper()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "listed.ail")
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Port, cfg.MCP = "0", true
	srv := New(tmpDir, cfg)
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs
}

func sessionAt(t *testing.T, endpoint string, headers map[string]string) *mcp.ClientSession {
	t.Helper()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(context.Background(),
		&mcp.StreamableClientTransport{
			Endpoint:             endpoint,
			HTTPClient:           &http.Client{Transport: headerRoundTripper{headers}},
			DisableStandaloneSSE: true,
			MaxRetries:           -1,
		}, nil)
	if err != nil {
		t.Fatalf("connect %s: %v", endpoint, err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func toolsByName(t *testing.T, cs *mcp.ClientSession) map[string]*mcp.Tool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*mcp.Tool{}
	for _, tl := range res.Tools {
		out[tl.Name] = tl
	}
	return out
}

func schemaJSON(tl *mcp.Tool) string {
	b, _ := json.Marshal(tl.InputSchema)
	return string(b)
}

func TestListedSurface_Projection(t *testing.T) {
	hs := routedServer(t, listedModule, Config{})
	agent := toolsByName(t, sessionAt(t, hs.URL+"/mcp/", nil))
	listed := toolsByName(t, sessionAt(t, hs.URL+"/mcp/connect/", nil))

	// Agent surface: everything, secrets advertised as before.
	for _, n := range []string{"whoami", "deviceLogin", "openTool", "submit_feedback"} {
		if agent[n] == nil {
			t.Errorf("/mcp/ missing %s", n)
		}
	}
	if s := schemaJSON(agent["whoami"]); !strings.Contains(s, `"apiKey"`) {
		t.Errorf("/mcp/ whoami should still advertise apiKey: %s", s)
	}

	// Listed surface: no agent-only tool, no secret param.
	if listed["deviceLogin"] != nil {
		t.Error("/mcp/connect/ must not list @mcp_agent_only deviceLogin")
	}
	for _, n := range []string{"whoami", "openTool", "submit_feedback"} {
		if listed[n] == nil {
			t.Errorf("/mcp/connect/ missing %s", n)
		}
	}
	if s := schemaJSON(listed["whoami"]); strings.Contains(s, "apiKey") || !strings.Contains(s, `"name"`) {
		t.Errorf("/mcp/connect/ whoami schema should have name and no apiKey: %s", s)
	}
}

// A secret is never taken from the client on the listed surface, even when
// sent anyway (named or positional); on /mcp/ it binds as before.
func TestListedSurface_SecretNotAccepted(t *testing.T) {
	hs := routedServer(t, listedModule, Config{})
	listed := sessionAt(t, hs.URL+"/mcp/connect/", nil)
	agent := sessionAt(t, hs.URL+"/mcp/", nil)

	for _, args := range []map[string]any{
		{"name": "ann", "apiKey": "dp_leak"},
		{"args": []any{"ann", "dp_leak"}},
	} {
		got, isErr := callText(t, listed, "whoami", args)
		if isErr || strings.Contains(got, "dp_leak") || !strings.Contains(got, "ann:") {
			t.Errorf("listed whoami(%v) = %q (isError=%v); secret must bind zero", args, got, isErr)
		}
	}
	got, _ := callText(t, agent, "whoami", map[string]any{"name": "ann", "apiKey": "dp_ok"})
	if !strings.Contains(got, "ann:dp_ok") {
		t.Errorf("/mcp/ whoami = %q, want the argument bound", got)
	}
}

// A server whose modules use none of the listed-surface annotations does not
// get a /mcp/connect/ projection: its route table is as before.
func TestListedSurface_NotMountedWithoutAnnotations(t *testing.T) {
	plain := "module test/api/listed\n\nexport func openTool(x: string) -> string ! {IO} = x\n"
	if hasListedSurface(map[string]*ModuleInfo{"m": {Exports: []ExportInfo{{Name: "openTool"}}}}) {
		t.Fatal("hasListedSurface true for an unannotated export")
	}
	hs := routedServer(t, plain, Config{})
	agent := toolsByName(t, sessionAt(t, hs.URL+"/mcp/", nil))
	if agent["openTool"] == nil {
		t.Fatal("/mcp/ missing openTool")
	}
}
