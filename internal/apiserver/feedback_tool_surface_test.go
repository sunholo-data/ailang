package apiserver

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func feedbackSurfaceServer(t *testing.T, cfg Config) *Server {
	t.Helper()

	apiDir := filepath.Join(t.TempDir(), "test", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "surface.ail")
	content := `module test/api/surface

@route("GET", "/status")
export pure func status() -> string = "ok"

export pure func helper() -> string = "helper"
`
	if err := os.WriteFile(modPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("AILANG_STDLIB_PATH") == "" {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				t.Setenv("AILANG_STDLIB_PATH", dir)
				break
			}
		}
	}
	srv := New(filepath.Dir(filepath.Dir(apiDir)), cfg)
	if err := srv.LoadModules([]string{modPath}); err != nil {
		srv.Close()
		t.Fatalf("LoadModules: %v", err)
	}
	return srv
}

func listFeedbackSurfaceTools(t *testing.T, srv *Server) (*mcp.ListToolsResult, *mcp.ClientSession) {
	t.Helper()
	ms := NewMCPServer(srv)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := ms.mcpServer.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "surface-test", Version: "1"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	res, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatal("instrument failure - positive control missing: empty tools/list")
	}
	return res, clientSession
}

func feedbackToolSet(tools []*mcp.Tool) map[string]bool {
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	return names
}

func TestFeedbackTool_PresentByDefault(t *testing.T) {
	srv := feedbackSurfaceServer(t, Config{})
	defer srv.Close()
	res, _ := listFeedbackSurfaceTools(t, srv)
	names := feedbackToolSet(res.Tools)
	if !names["status"] {
		t.Fatalf("positive control status missing; tools: %v", toolNames(res.Tools))
	}
	if !names["submit_feedback"] {
		t.Errorf("submit_feedback missing by default; tools: %v", toolNames(res.Tools))
	}
}

func TestFeedbackTool_SuppressedWhenConfigured(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		want []string
	}{
		{"all user exports", Config{NoFeedbackTool: true}, []string{"helper", "status"}},
		{"routes only exact surface", Config{RoutesOnly: true, NoFeedbackTool: true}, []string{"status"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := feedbackSurfaceServer(t, tc.cfg)
			defer srv.Close()
			res, _ := listFeedbackSurfaceTools(t, srv)
			names := feedbackToolSet(res.Tools)
			if !names["status"] {
				t.Fatalf("instrument failure - positive control status missing; tools: %v", toolNames(res.Tools))
			}
			if names["submit_feedback"] {
				t.Errorf("submit_feedback present when suppressed; tools: %v", toolNames(res.Tools))
			}
			got := toolNames(res.Tools)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tools = %v, want exact surface %v", got, tc.want)
			}
		})
	}
}

func TestFeedbackTool_SuppressedIsAlsoUncallable(t *testing.T) {
	srv := feedbackSurfaceServer(t, Config{NoFeedbackTool: true})
	defer srv.Close()
	res, session := listFeedbackSurfaceTools(t, srv)
	if !feedbackToolSet(res.Tools)["status"] {
		t.Fatalf("instrument failure - positive control status missing; tools: %v", toolNames(res.Tools))
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "submit_feedback", Arguments: map[string]any{},
	}); err == nil {
		t.Error("CallTool submit_feedback should error when suppressed")
	}
}

func TestFeedbackTool_RoutesOnlyNotice(t *testing.T) {
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })

	srv := feedbackSurfaceServer(t, Config{RoutesOnly: true})
	defer srv.Close()
	res, _ := listFeedbackSurfaceTools(t, srv)
	names := feedbackToolSet(res.Tools)
	if !names["status"] || !names["submit_feedback"] {
		t.Fatalf("positive controls missing; tools: %v", toolNames(res.Tools))
	}
	if got := buf.String(); !strings.Contains(got, "submit_feedback") || !strings.Contains(got, "--no-feedback-tool") {
		t.Errorf("notice = %q, want tool and suppression flag", got)
	}
}
