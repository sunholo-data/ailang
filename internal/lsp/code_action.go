package lsp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/quickfix"
	"go.lsp.dev/protocol"
)

// CodeAction implements LSP `textDocument/codeAction`, returning deterministic
// quick-fix CodeActions computed by the shared internal/quickfix engine so
// CLI (`ailang check --json` / `--fix`) and LSP produce identical edits.
func (s *Server) CodeAction(_ context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	if params == nil {
		return []protocol.CodeAction{}, nil
	}
	if !wantsQuickFix(params.Context.Only) {
		return []protocol.CodeAction{}, nil
	}

	docURI := params.TextDocument.URI
	path := uriToPath(docURI)
	src, ok := s.loadDocText(docURI, path)
	if !ok {
		return []protocol.CodeAction{}, nil
	}

	diags := params.Context.Diagnostics
	if len(diags) == 0 {
		for _, d := range runPipelineForDiagnostics(path, src) {
			if rangesOverlap(d.Range, params.Range) {
				diags = append(diags, d)
			}
		}
	}

	actions := make([]protocol.CodeAction, 0, len(diags))
	seen := make(map[string]bool)
	for _, d := range diags {
		qfDiag := protocolDiagnosticToQuickFix(d, path)
		edits := quickfix.ComputeEdits(qfDiag, path, src)
		if len(edits) == 0 {
			continue
		}

		lspEdits := make([]protocol.TextEdit, len(edits))
		for i, e := range edits {
			lspEdits[i] = quickfixEditToLSP(e)
		}

		title := edits[0].Title
		if title == "" {
			title = "Apply AILANG quick fix"
		}
		key := fmt.Sprintf("%s|%d:%d-%d:%d|%s",
			title,
			lspEdits[0].Range.Start.Line, lspEdits[0].Range.Start.Character,
			lspEdits[0].Range.End.Line, lspEdits[0].Range.End.Character,
			lspEdits[0].NewText,
		)
		if seen[key] {
			continue
		}
		seen[key] = true

		actions = append(actions, protocol.CodeAction{
			Title:       title,
			Kind:        protocol.QuickFix,
			Diagnostics: []protocol.Diagnostic{d},
			IsPreferred: true,
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					docURI: lspEdits,
				},
			},
		})
	}

	return actions, nil
}

func (s *Server) loadDocText(docURI protocol.DocumentURI, path string) (string, bool) {
	if v, ok := s.docs.Load(docURI); ok {
		if str, ok := v.(string); ok {
			return str, true
		}
	}
	if path == "" {
		return "", false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func wantsQuickFix(only []protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}
	for _, k := range only {
		ks := string(k)
		if ks == "" || ks == string(protocol.QuickFix) || strings.HasPrefix(string(protocol.QuickFix), ks+".") {
			return true
		}
	}
	return false
}

func protocolDiagnosticToQuickFix(d protocol.Diagnostic, path string) quickfix.Diagnostic {
	return quickfix.Diagnostic{
		Code:    diagnosticCodeString(d.Code),
		Message: d.Message,
		File:    path,
		Line:    int(d.Range.Start.Line) + 1,
		Column:  int(d.Range.Start.Character) + 1,
	}
}

func diagnosticCodeString(code interface{}) string {
	switch c := code.(type) {
	case nil:
		return ""
	case string:
		return c
	default:
		return fmt.Sprint(c)
	}
}

func quickfixEditToLSP(e quickfix.TextEdit) protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(max0(e.StartLine - 1)),
				Character: uint32(max0(e.StartCol - 1)),
			},
			End: protocol.Position{
				Line:      uint32(max0(e.EndLine - 1)),
				Character: uint32(max0(e.EndCol - 1)),
			},
		},
		NewText: e.NewText,
	}
}

func rangesOverlap(a, b protocol.Range) bool {
	// If b is completely before a, or a is completely before b, they don't overlap.
	if posLess(b.End, a.Start) || posLess(a.End, b.Start) {
		return false
	}
	return true
}

func posLess(a, b protocol.Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Character < b.Character
}
