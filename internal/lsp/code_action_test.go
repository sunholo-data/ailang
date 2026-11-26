package lsp

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/quickfix"
	"go.lsp.dev/protocol"
)

// TestCodeAction_EndToEnd_QuickFixes verifies that `textDocument/codeAction`
// returns valid quick-fix WorkspaceEdits over the live JSON-RPC transport for
// parser errors, import errors, missing stdlib imports, and TC_MATCH_002.
func TestCodeAction_EndToEnd_QuickFixes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		src             string
		wantCode        string
		wantNewTextSub  string
		wantTitlePrefix string
	}{
		{
			name:            "PAR_HASH_COMMENT",
			src:             "module fixture\n# Python comment\nexport pure func f() -> int = 1\n",
			wantCode:        "PAR_HASH_COMMENT",
			wantNewTextSub:  "--",
			wantTitlePrefix: "Replace",
		},
		{
			name:            "PAR018_let_rec",
			src:             "module fixture\nexport pure func f(n: int) -> int {\n  let rec go = \\x. x;\n  go(n)\n}\n",
			wantCode:        "PAR018",
			wantNewTextSub:  "letrec",
			wantTitlePrefix: "Replace",
		},
		{
			name: "TC_MATCH_002_missing_arm",
			src: `module fixture
type Color = Red | Green | Blue
export pure func code(c: Color) -> int =
  match c {
    Red => 1,
    Green => 2
  }
`,
			wantCode:        "TC_MATCH_002",
			wantNewTextSub:  "Blue => _?",
			wantTitlePrefix: "Add",
		},
		{
			name:            "missing_stdlib_import",
			src:             "module fixture\nexport pure func f(xs: [int]) -> int = foldl(\\a b. a + b, 0, xs)\n",
			wantCode:        "ERROR",
			wantNewTextSub:  "import std/list (foldl)",
			wantTitlePrefix: "Add",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cli, sink, cleanup := startTestServer(t)
			defer cleanup()

			docURI, _ := fileURI(t, tc.src)
			if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        docURI,
					LanguageID: "ailang",
					Version:    1,
					Text:       tc.src,
				},
			}); err != nil {
				t.Fatalf("didOpen: %v", err)
			}

			pub, ok := sink.wait(docURI, diagWaitTimeout())
			if !ok {
				t.Fatalf("no publishDiagnostics arrived for %s", tc.name)
			}
			if len(pub.Diagnostics) == 0 {
				t.Fatalf("expected >=1 diagnostic for %s, got 0", tc.name)
			}

			var targetDiag protocol.Diagnostic
			found := false
			for _, d := range pub.Diagnostics {
				if diagnosticCodeString(d.Code) == tc.wantCode {
					targetDiag = d
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("did not find diagnostic with code %q in %+v", tc.wantCode, pub.Diagnostics)
			}

			var actions []protocol.CodeAction
			if _, err := cli.Call(ctx, "textDocument/codeAction", &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Range:        targetDiag.Range,
				Context: protocol.CodeActionContext{
					Diagnostics: []protocol.Diagnostic{targetDiag},
					Only:        []protocol.CodeActionKind{protocol.QuickFix},
				},
			}, &actions); err != nil {
				t.Fatalf("textDocument/codeAction: %v", err)
			}

			if len(actions) != 1 {
				t.Fatalf("got %d code actions, want 1: %+v", len(actions), actions)
			}
			act := actions[0]
			if act.Kind != protocol.QuickFix {
				t.Errorf("Kind = %q, want %q", act.Kind, protocol.QuickFix)
			}
			if !strings.HasPrefix(act.Title, tc.wantTitlePrefix) {
				t.Errorf("Title = %q, want prefix %q", act.Title, tc.wantTitlePrefix)
			}
			if act.Edit == nil || len(act.Edit.Changes[docURI]) == 0 {
				t.Fatalf("expected WorkspaceEdit.Changes for %s, got %+v", docURI, act.Edit)
			}
			edit := act.Edit.Changes[docURI][0]
			if !strings.Contains(edit.NewText, tc.wantNewTextSub) {
				t.Errorf("edit.NewText = %q, want substring %q", edit.NewText, tc.wantNewTextSub)
			}
		})
	}
}

// TestCodeAction_ParityWithQuickFix verifies that LSP `textDocument/codeAction`
// produces the exact 0-based counterpart of quickfix.ComputeEdits (used by
// `ailang check --json`) for all 6 supported diagnostic families.
func TestCodeAction_ParityWithQuickFix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		diag quickfix.Diagnostic
	}{
		{
			name: "PAR017",
			src:  "module m\nexport pure func f(x: int) -> int = {\n  let y = x + 1;\n  y\n}\n",
			diag: quickfix.Diagnostic{Code: "PAR017", Line: 3, Column: 16},
		},
		{
			name: "PAR018",
			src:  "module m\nexport pure func f(n: int) -> int {\n  let rec go = \\x. x;\n  go(n)\n}\n",
			diag: quickfix.Diagnostic{Code: "PAR018", Line: 3, Column: 7},
		},
		{
			name: "PAR_HASH_COMMENT",
			src:  "module m\n# comment\nexport pure func f() -> int = 1\n",
			diag: quickfix.Diagnostic{Code: "PAR_HASH_COMMENT", Line: 2, Column: 1},
		},
		{
			name: "IMP010",
			src:  "module m\nimport std/string (nthChar)\nexport pure func f(s: string) -> string = s\n",
			diag: quickfix.Diagnostic{
				Code:    "IMP010",
				Message: "symbol 'nthChar' not exported by 'std/string' — did you mean 'charAt'? (exported by 'std/string')",
				Line:    2,
				Column:  1,
			},
		},
		{
			name: "missing_stdlib_import",
			src:  "module m\nexport pure func f(xs: [int]) -> int = foldl(\\a b. a + b, 0, xs)\n",
			diag: quickfix.Diagnostic{
				Code:    "ERROR",
				Message: "undefined variable: foldl at m.ail:2:40",
				Line:    2,
				Column:  40,
			},
		},
		{
			name: "TC_MATCH_002",
			src:  "module m\ntype Color = Red | Blue\nexport pure func f(c: Color) -> int =\n  match c {\n    Red => 1\n  }\n",
			diag: quickfix.Diagnostic{
				Code:    "TC_MATCH_002",
				Message: "non-exhaustive match on Color — no arm matches: Blue",
				Line:    4,
				Column:  3,
			},
		},
	}

	srv := NewServer(nil)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			docURI, path := fileURI(t, tc.src)
			srv.docs.Store(docURI, tc.src)

			cliEdits := quickfix.ComputeEdits(tc.diag, path, tc.src)
			if len(cliEdits) != 1 {
				t.Fatalf("quickfix.ComputeEdits returned %d edits, want 1", len(cliEdits))
			}

			lspDiag := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(max0(tc.diag.Line - 1)),
						Character: uint32(max0(tc.diag.Column - 1)),
					},
					End: protocol.Position{
						Line:      uint32(max0(tc.diag.Line - 1)),
						Character: uint32(max0(tc.diag.Column)),
					},
				},
				Code:    tc.diag.Code,
				Message: tc.diag.Message,
			}

			actions, err := srv.CodeAction(context.Background(), &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Range:        lspDiag.Range,
				Context: protocol.CodeActionContext{
					Diagnostics: []protocol.Diagnostic{lspDiag},
				},
			})
			if err != nil {
				t.Fatalf("CodeAction error: %v", err)
			}
			if len(actions) != 1 {
				t.Fatalf("got %d actions, want 1", len(actions))
			}
			lspEdits := actions[0].Edit.Changes[docURI]
			if len(lspEdits) != len(cliEdits) {
				t.Fatalf("edit count mismatch: lsp=%d cli=%d", len(lspEdits), len(cliEdits))
			}
			want := quickfixEditToLSP(cliEdits[0])
			if lspEdits[0] != want {
				t.Errorf("LSP edit %+v != CLI edit %+v", lspEdits[0], want)
			}
		})
	}
}

func TestCodeAction_OnlyFilter(t *testing.T) {
	t.Parallel()
	srv := NewServer(nil)
	src := "module m\n# comment\nexport pure func f() -> int = 1\n"
	docURI, _ := fileURI(t, src)
	srv.docs.Store(docURI, src)

	actions, err := srv.CodeAction(context.Background(), &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{protocol.SourceOrganizeImports},
			Diagnostics: []protocol.Diagnostic{{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 1},
				},
				Code: "PAR_HASH_COMMENT",
			}},
		},
	})
	if err != nil {
		t.Fatalf("CodeAction error: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions when Only=[source.organizeImports], got %d", len(actions))
	}
}
