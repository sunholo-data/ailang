package lsp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.lsp.dev/protocol"
)

const symbolsFixture = `module fixture

type Color = Red | Green | Blue

export pure func describe(c: Color) -> string =
  match c {
    Red => "warm",
    Green => "natural",
    Blue => "cool"
  }
`

// TestDocumentSymbolReturnsFuncAndADT: the fixture has one FuncDecl and one
// ADT with 3 constructors. The outline should report both top-level
// symbols, with the ADT carrying 3 children.
func TestDocumentSymbolReturnsFuncAndADT(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	docURI, _ := fileURI(t, symbolsFixture)
	ctx := context.Background()

	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "ailang", Version: 1, Text: symbolsFixture},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	dsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var raw json.RawMessage
	if _, err := cli.Call(dsCtx, "textDocument/documentSymbol", &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	}, &raw); err != nil {
		t.Fatalf("textDocument/documentSymbol: %v", err)
	}

	var symbols []protocol.DocumentSymbol
	if err := json.Unmarshal(raw, &symbols); err != nil {
		t.Fatalf("unmarshal symbols: %v\nraw: %s", err, string(raw))
	}

	// Expect 2 top-level symbols: the ADT `Color` and the func `describe`.
	// Order doesn't matter for the test — group by name.
	byName := map[string]protocol.DocumentSymbol{}
	for _, s := range symbols {
		byName[s.Name] = s
	}

	color, hasColor := byName["Color"]
	if !hasColor {
		t.Fatalf("missing top-level symbol 'Color'. Got: %+v", symbols)
	}
	if color.Kind != protocol.SymbolKindClass {
		t.Errorf("Color.Kind: got %v, want Class (we map ADT → Class)", color.Kind)
	}
	if len(color.Children) != 3 {
		t.Errorf("Color should have 3 constructor children, got %d:\n%+v", len(color.Children), color.Children)
	}
	ctorNames := map[string]bool{}
	for _, c := range color.Children {
		ctorNames[c.Name] = true
		if c.Kind != protocol.SymbolKindConstructor {
			t.Errorf("constructor %s: Kind got %v, want Constructor", c.Name, c.Kind)
		}
	}
	for _, want := range []string{"Red", "Green", "Blue"} {
		if !ctorNames[want] {
			t.Errorf("missing constructor child: %s", want)
		}
	}

	describe, hasDescribe := byName["describe"]
	if !hasDescribe {
		t.Fatalf("missing top-level symbol 'describe'. Got: %+v", symbols)
	}
	if describe.Kind != protocol.SymbolKindFunction {
		t.Errorf("describe.Kind: got %v, want Function", describe.Kind)
	}

	// Range sanity: starts must precede ends.
	for _, s := range symbols {
		if !rangeWellFormed(s.Range) {
			t.Errorf("symbol %s: malformed Range %+v", s.Name, s.Range)
		}
		if !rangeWellFormed(s.SelectionRange) {
			t.Errorf("symbol %s: malformed SelectionRange %+v", s.Name, s.SelectionRange)
		}
	}
}

// TestDocumentSymbolEmptyFile: an empty buffer returns no symbols (and
// doesn't panic).
func TestDocumentSymbolEmptyFile(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	docURI, _ := fileURI(t, "")
	ctx := context.Background()
	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "ailang", Version: 1, Text: ""},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	var raw json.RawMessage
	if _, err := cli.Call(ctx, "textDocument/documentSymbol", &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	}, &raw); err != nil {
		t.Fatalf("textDocument/documentSymbol: %v", err)
	}

	var symbols []protocol.DocumentSymbol
	_ = json.Unmarshal(raw, &symbols)
	if len(symbols) != 0 {
		t.Errorf("empty file should return no symbols, got %d", len(symbols))
	}
}

func rangeWellFormed(r protocol.Range) bool {
	if r.Start.Line > r.End.Line {
		return false
	}
	if r.Start.Line == r.End.Line && r.Start.Character > r.End.Character {
		return false
	}
	return true
}
