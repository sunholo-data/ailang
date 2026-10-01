package apiserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// A zero-arg export (`func f()`, which the parser desugars into one `()` param
// named "_") and an explicit `unit` param carry no information. They must not
// appear in inputSchema, and tools/call with {} must run the function. Before
// the fix every zero-arg tool advertised a required string "_" and {} was
// rejected — prod AILANG Parse's mcpFormats was uncallable by a normal client.
func TestMCPUnitParams_NotAdvertisedAndCallable(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `module test/api/unitp

export func zeroArg() -> string ! {IO} = "zero"

export func withUnit(u: unit, x: string) -> string = "${x}!"
`
	modPath := filepath.Join(apiDir, "unitp.ail")
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := New(tmpDir, Config{Port: "0"})
	defer srv.Close()
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := NewMCPServer(srv).mcpServer.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	schemas := map[string]string{}
	for _, tl := range res.Tools {
		b, _ := json.Marshal(tl.InputSchema)
		schemas[tl.Name] = string(b)
	}
	if s := schemas["zeroArg"]; strings.Contains(s, `"_"`) {
		t.Errorf("zeroArg advertises the desugared unit param: %s", s)
	}
	if s := schemas["withUnit"]; strings.Contains(s, `"u"`) || !strings.Contains(s, `"x"`) {
		t.Errorf("withUnit schema should list x only: %s", s)
	}

	for _, c := range []struct{ tool, args, want string }{
		{"zeroArg", `{}`, "zero"},
		{"withUnit", `{"x": "hi"}`, "hi!"},
	} {
		var args map[string]any
		_ = json.Unmarshal([]byte(c.args), &args)
		out, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: c.tool, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", c.tool, err)
		}
		text := out.Content[0].(*mcp.TextContent).Text
		if out.IsError || !strings.Contains(text, c.want) {
			t.Errorf("%s(%s) = %q (isError=%v), want %q", c.tool, c.args, text, out.IsError, c.want)
		}
	}
}
