package apiserver

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listToolsFromSource loads src as one module into a serve-api Server built
// with cfg, lists its MCP tools over an in-memory go-sdk session, and returns
// them by name together with everything registration logged.
func listToolsFromSource(t *testing.T, src string, cfg Config) (map[string]*mcp.Tool, string, *Server) {
	t.Helper()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "authmod.ail")
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg.Port == "" {
		cfg.Port = "0"
	}
	srv := New(tmpDir, cfg)
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	ms := NewMCPServer(srv)
	log.SetOutput(prev)

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := ms.mcpServer.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tools := map[string]*mcp.Tool{}
	for _, tl := range res.Tools {
		tools[tl.Name] = tl
	}
	return tools, logs.String(), srv
}

const authModuleHeader = "module test/api/authmod\n\n"

const goodVerifier = `
@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"
`

const gatedTool = `
@mcp_auth("oauth2")
@mcp_secret("apiKey")
@optional("apiKey")
export func gated(doc: string, apiKey: string) -> string ! {IO} = doc
`

func TestMCPAuth_ValidSetupRegisters(t *testing.T) {
	tools, logs, srv := listToolsFromSource(t, authModuleHeader+goodVerifier+gatedTool, Config{OAuthIssuer: "https://auth.example.com"})
	if _, ok := tools["gated"]; !ok {
		t.Fatalf("gated tool should register with issuer + verifier; logs:\n%s", logs)
	}
	// The verifier is never a tool, and never an HTTP endpoint without @route:
	// exposed, it would be a token-guessing oracle.
	if _, ok := tools["verifyToken"]; ok {
		t.Error("verifyToken must not be an MCP tool")
	}
	v := findExport(t, srv, "verifyToken")
	if !v.IsTokenVerifier || !v.VerifierSigOK || !v.IsNoMCP || !v.IsNoExpose {
		t.Errorf("verifier flags: %+v", v)
	}
	g := findExport(t, srv, "gated")
	if g.MCPAuth != "oauth2" || len(g.MCPSecret) != 1 || g.MCPSecret[0] != "apiKey" {
		t.Errorf("gated extraction: auth=%q secret=%v", g.MCPAuth, g.MCPSecret)
	}
}

// Each registration rule refuses the gated tool and logs why. An open tool in
// the same module is unaffected by every one of them.
func TestMCPAuth_RegistrationRules(t *testing.T) {
	const open = "\nexport func openTool(x: string) -> string ! {IO} = x\n"
	cases := []struct {
		name, src, issuer, wantLog string
	}{
		{"no issuer", goodVerifier + gatedTool, "", "needs --oauth-issuer"},
		{"no verifier", gatedTool, "https://a", "needs one @mcp_token_verifier"},
		{"two verifiers", goodVerifier + `
@mcp_token_verifier
export func verifyToken2(token: string) -> bool ! {IO} = false
` + gatedTool, "https://a", "2 verifiers"},
		{"bad verifier signature", `
@mcp_token_verifier
export func verifyToken(token: string, extra: int) -> bool ! {IO} = true
` + gatedTool, "https://a", "signature must be (token: string) -> bool"},
		{"unknown scheme", goodVerifier + `
@mcp_auth("basic")
export func gated(doc: string) -> string ! {IO} = doc
`, "https://a", `unknown scheme`},
		{"secret not a param", goodVerifier + `
@mcp_auth("oauth2")
@mcp_secret("nope")
export func gated(doc: string) -> string ! {IO} = doc
`, "https://a", `@mcp_secret("nope"): no such parameter`},
		{"secret not optional", goodVerifier + `
@mcp_secret("apiKey")
export func gated(doc: string, apiKey: string) -> string ! {IO} = doc
`, "", `must also be @optional("apiKey")`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tools, logs, _ := listToolsFromSource(t, authModuleHeader+c.src+open, Config{OAuthIssuer: c.issuer})
			if _, ok := tools["gated"]; ok {
				t.Errorf("gated must not register; logs:\n%s", logs)
			}
			if !strings.Contains(logs, "ERROR") || !strings.Contains(logs, c.wantLog) {
				t.Errorf("want ERROR containing %q; logs:\n%s", c.wantLog, logs)
			}
			if _, ok := tools["openTool"]; !ok {
				t.Errorf("openTool must still register; logs:\n%s", logs)
			}
		})
	}
}
