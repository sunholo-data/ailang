package lsp

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.lsp.dev/protocol"
)

const xrefDecl = `module xref_decl

export pure func add(x: int, y: int) -> int = x + y
`

const xrefUse = `module xref_use

export pure func use_add() -> int = add(1, 2)
`

// TestSameFileDefinition: hovering on a use-site of an identifier whose
// declaration is in the SAME file returns a Location pointing at the
// FuncDecl's name (not the keyword the parser assigns Pos to).
func TestSameFileDefinition(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	const src = `module fixture

export pure func add(x: int, y: int) -> int = x + y

export pure func use_add() -> int = add(1, 2)
`
	docURI, _ := fileURI(t, src)
	ctx := context.Background()

	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "ailang", Version: 1, Text: src},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	// `add` callsite at line 5 (LSP 0-indexed: 4), col 37 (LSP: 36).
	defCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var locs []protocol.Location
	if _, err := cli.Call(defCtx, "textDocument/definition", &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 4, Character: 37},
		},
	}, &locs); err != nil {
		t.Fatalf("textDocument/definition: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("definition: got %d locations, want 1", len(locs))
	}
	got := locs[0]
	if got.URI != docURI {
		t.Errorf("definition URI: got %q, want %q", got.URI, docURI)
	}
	// Decl `add` is at line 3 (0-indexed: 2), col 18 (0-indexed: 17).
	// `export pure func ` is 17 chars, so `add` starts at character 17.
	if got.Range.Start.Line != 2 {
		t.Errorf("definition Start.Line: got %d, want 2", got.Range.Start.Line)
	}
	if got.Range.Start.Character != 17 {
		t.Errorf("definition Start.Character: got %d, want 17 (the `add` name, NOT the `pure` keyword)",
			got.Range.Start.Character)
	}
}

// TestSessionScopedReferences: open two documents in the same session,
// reference an identifier defined in one and used in the other; references
// returns locations from BOTH files.
func TestSessionScopedReferences(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	declURI, _ := fileURI(t, xrefDecl)
	useURI, _ := fileURI(t, xrefUse)
	ctx := context.Background()

	for _, doc := range []struct {
		uri  protocol.DocumentURI
		text string
	}{
		{declURI, xrefDecl},
		{useURI, xrefUse},
	} {
		if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{URI: doc.uri, LanguageID: "ailang", Version: 1, Text: doc.text},
		}); err != nil {
			t.Fatalf("didOpen %s: %v", doc.uri, doc.text)
		}
	}

	// Cursor on `add` at the use site in xrefUse (line 3, col 37 → LSP 2,36).
	refCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var locs []protocol.Location
	if _, err := cli.Call(refCtx, "textDocument/references", &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: useURI},
			Position:     protocol.Position{Line: 2, Character: 37},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}, &locs); err != nil {
		t.Fatalf("textDocument/references: %v", err)
	}

	if len(locs) < 2 {
		t.Fatalf("references with IncludeDeclaration=true expected ≥2 locations (use + decl), got %d:\n%+v",
			len(locs), locs)
	}

	// Verify we got at least one hit in EACH file.
	hitDecl, hitUse := false, false
	for _, l := range locs {
		switch l.URI {
		case declURI:
			hitDecl = true
		case useURI:
			hitUse = true
		}
	}
	if !hitDecl {
		t.Errorf("references missed the declaration file %s. Got: %+v", declURI, locs)
	}
	if !hitUse {
		t.Errorf("references missed the use-site file %s. Got: %+v", useURI, locs)
	}
}

// TestReferencesExcludesDeclarationWhenAsked: same setup but
// IncludeDeclaration=false; result must NOT include the declaration line.
func TestReferencesExcludesDeclarationWhenAsked(t *testing.T) {
	t.Parallel()
	_, cli, _, cleanup := startTestServer(t)
	defer cleanup()

	const src = `module fixture

export pure func add(x: int, y: int) -> int = x + y

export pure func use_add() -> int = add(1, 2)
`
	docURI, _ := fileURI(t, src)
	ctx := context.Background()
	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "ailang", Version: 1, Text: src},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	var locs []protocol.Location
	if _, err := cli.Call(ctx, "textDocument/references", &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 4, Character: 37},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	}, &locs); err != nil {
		t.Fatalf("textDocument/references: %v", err)
	}

	// Decl `add` is at LSP-0-indexed line 2, character 17. With
	// IncludeDeclaration=false, no result should land at that exact spot.
	for _, l := range locs {
		if l.Range.Start.Line == 2 && l.Range.Start.Character == 17 {
			t.Errorf("IncludeDeclaration=false but result includes the decl: %+v", l)
		}
	}
}

var _ = strings.Contains // silence unused-import if Contains drops out later
