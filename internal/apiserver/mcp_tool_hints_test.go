package apiserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// @mcp_title / @mcp_hints end to end: AILANG source → serve-api → tools/list
// over a real go-sdk client session. MCP directories (Anthropic, OpenAI) refuse
// tools without a title and a readOnly/destructive hint.
func TestMCPToolHints_ToolsList(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `module test/api/hints

@mcp_title("Read a thing")
@mcp_hints("readOnly", "openWorld")
export func readThing(x: string) -> string ! {IO} { x }

@mcp_title("Delete a thing")
@mcp_hints("destructive", "idempotent")
export func deleteThing(x: string) -> string ! {IO} { x }

export pure func pureThing(x: int) -> int = x

export func unhintedThing(x: string) -> string ! {IO} { x }

@mcp_hints()
export func appendOnly(x: string) -> string ! {IO} { x }

export pure func pureKeywordWithEffects(x: string) -> string ! {IO} { x }

@mcp_hints("readonly")
export func typoThing(x: string) -> string ! {IO} { x }
`
	modPath := filepath.Join(apiDir, "hints.ail")
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := New(tmpDir, Config{Port: "0"})
	defer srv.Close()
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

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
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tools := map[string]*mcp.Tool{}
	for _, tl := range res.Tools {
		tools[tl.Name] = tl
	}

	if tl := tools["readThing"]; tl == nil || tl.Title != "Read a thing" || tl.Annotations == nil ||
		!tl.Annotations.ReadOnlyHint || tl.Annotations.DestructiveHint != nil ||
		tl.Annotations.OpenWorldHint == nil || !*tl.Annotations.OpenWorldHint {
		t.Errorf("readThing: %+v", annOf(tl))
	}
	if tl := tools["deleteThing"]; tl == nil || tl.Title != "Delete a thing" || tl.Annotations == nil ||
		tl.Annotations.ReadOnlyHint || tl.Annotations.DestructiveHint == nil || !*tl.Annotations.DestructiveHint ||
		!tl.Annotations.IdempotentHint || *tl.Annotations.OpenWorldHint {
		t.Errorf("deleteThing: %+v", annOf(tl))
	}
	// Pure with no @mcp_hints: the type system proves read-only, closed-world.
	if tl := tools["pureThing"]; tl == nil || tl.Annotations == nil || !tl.Annotations.ReadOnlyHint ||
		*tl.Annotations.OpenWorldHint {
		t.Errorf("pureThing: %+v", annOf(tl))
	}
	// Effectful with no @mcp_hints: advertised, but nothing guessed.
	if tl := tools["unhintedThing"]; tl == nil || tl.Annotations != nil || tl.Title != "" {
		t.Errorf("unhintedThing must be listed without annotations: %+v", annOf(tl))
	}
	// The `pure` keyword is not a proof (the checker accepts it beside a
	// declared row); only an empty effect row is. Nor is ExportInfo.Pure,
	// which the iface builder sets true for everything.
	if tl := tools["pureKeywordWithEffects"]; tl == nil || tl.Annotations != nil {
		t.Errorf("pure keyword + ! {IO} must not be marked read-only: %+v", annOf(tl))
	}
	// The Go-side built-in is listed on every server that keeps it on, so it
	// must be directory-ready too.
	if tl := tools["submit_feedback"]; tl == nil || tl.Title == "" || tl.Annotations == nil ||
		tl.Annotations.ReadOnlyHint || tl.Annotations.DestructiveHint == nil || *tl.Annotations.DestructiveHint {
		t.Errorf("submit_feedback: %+v", annOf(tl))
	}
	// @mcp_hints() is complete and empty: writes, additive, closed-world.
	if tl := tools["appendOnly"]; tl == nil || tl.Annotations == nil || tl.Annotations.ReadOnlyHint ||
		tl.Annotations.DestructiveHint == nil || *tl.Annotations.DestructiveHint ||
		tl.Annotations.OpenWorldHint == nil || *tl.Annotations.OpenWorldHint || tl.Annotations.IdempotentHint {
		t.Errorf("appendOnly (@mcp_hints()): %+v", annOf(tl))
	}
	// Unknown hint word: an author bug, so the tool is not registered.
	if _, ok := tools["typoThing"]; ok {
		t.Error("typoThing (@mcp_hints(\"readonly\")) must not be registered")
	}
}

func annOf(tl *mcp.Tool) any {
	if tl == nil {
		return "<missing>"
	}
	return struct {
		Title string
		Ann   *mcp.ToolAnnotations
	}{tl.Title, tl.Annotations}
}
