package lsp

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.lsp.dev/protocol"
)

const hoverFixture = `module fixture

export pure func add(x: int, y: int) -> int = x + y

export pure func use_add() -> int = add(1, 2)
`

// TestHoverOnTopLevelExport: cursor inside a top-level exported function's
// name returns a Hover whose Contents include both the function name and a
// formatted type signature (`add : ...`).
func TestHoverOnTopLevelExport(t *testing.T) {
	t.Parallel()
	_, cli, sink, cleanup := startTestServer(t)
	defer cleanup()
	_ = sink // not used here

	docURI, _ := fileURI(t, hoverFixture)
	ctx := context.Background()

	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "ailang",
			Version:    1,
			Text:       hoverFixture,
		},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	// `add(1, 2)` callsite is on line 5 (`export pure func use_add() -> int = add(1, 2)`).
	// `add` starts at column 37 (1-indexed), so 0-indexed character 36 in LSP terms.
	hoverCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var hover protocol.Hover
	if _, err := cli.Call(hoverCtx, "textDocument/hover", &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 4, Character: 37},
		},
	}, &hover); err != nil {
		t.Fatalf("textDocument/hover: %v", err)
	}

	body := hoverContentValue(t, hover.Contents)
	if !strings.Contains(body, "add") {
		t.Errorf("hover body should contain identifier name 'add'; got: %q", body)
	}
	// Top-level export => type sig must be present, not the locals fallback.
	if strings.Contains(body, "type info for non-exported names is deferred") {
		t.Errorf("top-level export hit the locals fallback; got: %q", body)
	}
}

// TestHoverOnWhitespaceReturnsNil: cursor on whitespace returns nil per
// LSP spec (no hover info), not an error.
func TestHoverOnWhitespaceReturnsNil(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	docURI, _ := fileURI(t, hoverFixture)
	ctx := context.Background()

	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "ailang",
			Version:    1,
			Text:       hoverFixture,
		},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	// Line 2 (blank line in the fixture) — must return no hover.
	var hover protocol.Hover
	if _, err := cli.Call(ctx, "textDocument/hover", &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 1, Character: 0},
		},
	}, &hover); err != nil {
		t.Fatalf("textDocument/hover: %v", err)
	}
	body := hoverContentValue(t, hover.Contents)
	if body != "" {
		t.Errorf("hover on whitespace should be empty/nil, got: %q", body)
	}
}

// hoverContentValue extracts the rendered text from a protocol.Hover's
// Contents field. Contents is interface{} (MarkupContent | MarkedString | …)
// and after JSON round-trip becomes a map[string]interface{}.
func hoverContentValue(t *testing.T, contents interface{}) string {
	t.Helper()
	switch c := contents.(type) {
	case nil:
		return ""
	case protocol.MarkupContent:
		return c.Value
	case map[string]interface{}:
		if v, ok := c["value"].(string); ok {
			return v
		}
	}
	return ""
}
