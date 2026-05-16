package lsp

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ast"
	"go.lsp.dev/protocol"
)

// DocumentSymbol implements LSP textDocument/documentSymbol. Returns a
// hierarchical list of top-level symbols in the document: functions,
// algebraic-data types (with constructors as children), and type classes.
//
// Imports are intentionally omitted (clutter; LSP outline is meant for
// navigation, not for documenting dependencies — that's what the doc
// header is for).
//
// MVP scope per design doc Decision 1: emits FuncDecl, TypeDecl
// (Algebraic+Record), TypeClass. Top-level let bindings would also be
// emittable but AILANG modules don't conventionally use them; defer if
// needed.
func (s *Server) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	if params == nil {
		return nil, nil
	}
	if !s.initialized.Load() {
		return nil, nil
	}

	docURI := params.TextDocument.URI
	v, ok := s.docs.Load(docURI)
	if !ok {
		return nil, nil
	}
	src, _ := v.(string)
	if src == "" {
		return nil, nil
	}

	path := uriToPath(docURI)
	file := parseFileForLSP(path, src)
	if file == nil {
		return nil, nil
	}

	out := make([]interface{}, 0)

	// Functions.
	for _, fd := range file.Funcs {
		if fd == nil {
			continue
		}
		out = append(out, funcDeclSymbol(src, fd))
	}

	// Type decls (ADTs, records) and type classes live on Statements.
	// f.Decls is deprecated and overlaps; skip it to avoid duplication.
	for _, n := range file.Statements {
		switch d := n.(type) {
		case *ast.TypeDecl:
			out = append(out, typeDeclSymbol(src, d))
		case *ast.TypeClass:
			out = append(out, typeClassSymbol(d))
		}
	}

	return out, nil
}

func funcDeclSymbol(src string, fd *ast.FuncDecl) protocol.DocumentSymbol {
	// Relocate from the keyword position (where the parser anchors
	// FuncDecl.Pos) to the actual function name.
	startLine, startCol := fd.Pos.Line, fd.Pos.Column
	if line, col, ok := findIdentAfter(src, fd.Pos.Line, fd.Pos.Column, fd.Name); ok {
		startLine, startCol = line, col
	}
	rng := nameRange(startLine, startCol, fd.Name)
	return protocol.DocumentSymbol{
		Name:           fd.Name,
		Kind:           protocol.SymbolKindFunction,
		Range:          rng,
		SelectionRange: rng,
	}
}

func typeDeclSymbol(src string, td *ast.TypeDecl) protocol.DocumentSymbol {
	startLine, startCol := td.Pos.Line, td.Pos.Column
	if line, col, ok := findIdentAfter(src, td.Pos.Line, td.Pos.Column, td.Name); ok {
		startLine, startCol = line, col
	}
	rng := nameRange(startLine, startCol, td.Name)

	sym := protocol.DocumentSymbol{
		Name:           td.Name,
		Kind:           protocol.SymbolKindClass,
		Range:          rng,
		SelectionRange: rng,
	}

	// Constructors → children with SymbolKindConstructor.
	if alg, ok := td.Definition.(*ast.AlgebraicType); ok && alg != nil {
		for _, ctor := range alg.Constructors {
			if ctor == nil {
				continue
			}
			cline, ccol := ctor.Pos.Line, ctor.Pos.Column
			if line, col, ok := findIdentAfter(src, ctor.Pos.Line, ctor.Pos.Column, ctor.Name); ok {
				cline, ccol = line, col
			}
			cRng := nameRange(cline, ccol, ctor.Name)
			sym.Children = append(sym.Children, protocol.DocumentSymbol{
				Name:           ctor.Name,
				Kind:           protocol.SymbolKindConstructor,
				Range:          cRng,
				SelectionRange: cRng,
			})
		}
	}
	return sym
}

func typeClassSymbol(tc *ast.TypeClass) protocol.DocumentSymbol {
	rng := nameRange(tc.Pos.Line, tc.Pos.Column, tc.Name)
	return protocol.DocumentSymbol{
		Name:           tc.Name,
		Kind:           protocol.SymbolKindInterface,
		Range:          rng,
		SelectionRange: rng,
	}
}

// nameRange builds a Range that covers a 1-indexed line/col + name length,
// translated to LSP's 0-indexed positions.
func nameRange(line, col int, name string) protocol.Range {
	startLine := uint32(max0(line - 1))
	startCol := uint32(max0(col - 1))
	endCol := startCol + uint32(len(name))
	return protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startCol},
		End:   protocol.Position{Line: startLine, Character: endCol},
	}
}
