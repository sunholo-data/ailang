package lsp

import (
	"context"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Definition implements LSP textDocument/definition. Returns a single
// Location pointing at the declaration site of the identifier under the
// cursor: same-file FuncDecl, cross-module export, or stdlib export.
//
// MVP scope: definition resolves identifiers that name a top-level
// FuncDecl. Local bindings (let-bound, lambda params) are deferred to
// the same M-AILANG-LSP-LOCAL-TYPES follow-up that holds local hover.
func (s *Server) Definition(_ context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	if params == nil {
		return nil, nil
	}
	if !s.initialized.Load() {
		return nil, nil
	}

	docURI := params.TextDocument.URI
	path := uriToPath(docURI)

	v, ok := s.docs.Load(docURI)
	if !ok {
		return nil, nil
	}
	src, _ := v.(string)
	if src == "" {
		return nil, nil
	}

	idx := s.indexes.get(path)
	if idx == nil {
		idx = BuildPositionIndex(path, src)
		if idx == nil {
			return nil, nil
		}
		s.indexes.put(path, idx)
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1
	id := idx.Lookup(line, col)
	if id == nil {
		return nil, nil
	}

	// Re-run pipeline so we have the loaded modules table for cross-file
	// resolution. Same in-memory buffer as the M2/M3 path; same LSP-tuned
	// Config (MOD010 relaxed — see diagnostics.go).
	srcBundle := pipeline.Source{Code: src, Filename: path}
	result, _ := pipeline.Run(lspPipelineConfig(), srcBundle)

	// 1. Same-file: walk root file's FuncDecls.
	if loc, ok := s.findFuncInFile(path, src, id.Name); ok {
		return []protocol.Location{loc}, nil
	}

	// 2. Loaded modules (transitive imports walked by pipeline.Run).
	for _, mod := range result.Modules {
		if mod == nil || mod.Path == "" || mod.Exports == nil {
			continue
		}
		if _, ok := mod.Exports[id.Name]; !ok {
			continue
		}
		modSrcBytes, err := os.ReadFile(mod.Path)
		if err != nil {
			continue
		}
		if loc, ok := s.findFuncInFile(mod.Path, string(modSrcBytes), id.Name); ok {
			return []protocol.Location{loc}, nil
		}
	}

	// 3. Stdlib fallback: if the file imports a std/X module that exports
	// the name, resolve it through the stdlib resolver.
	if loc, ok := s.findInStdlibImports(path, src, id.Name); ok {
		return []protocol.Location{loc}, nil
	}

	return nil, nil
}

// findFuncInFile parses src and looks for a FuncDecl named name.
// Returns a Location whose Range covers the function name (NOT the keyword
// the parser assigns Pos to).
func (s *Server) findFuncInFile(path string, src string, name string) (protocol.Location, bool) {
	file := s.parseFile(path, src)
	if file == nil {
		return protocol.Location{}, false
	}
	for _, fd := range file.Funcs {
		if fd == nil || fd.Name != name {
			continue
		}
		return funcDeclLocation(path, src, fd), true
	}
	return protocol.Location{}, false
}

// funcDeclLocation builds an LSP Location for a FuncDecl, correcting
// FuncDecl.Pos (which points at the introducing keyword like `pure` or
// `func`, depending on the modifier set) to point at the name itself.
// We do this by scanning the source for the next occurrence of the name
// after the FuncDecl's start offset.
func funcDeclLocation(path string, src string, fd *ast.FuncDecl) protocol.Location {
	startLine, startCol := fd.Pos.Line, fd.Pos.Column
	// Try to relocate to the actual name occurrence.
	if fd.Name != "" {
		if line, col, ok := findIdentAfter(src, fd.Pos.Line, fd.Pos.Column, fd.Name); ok {
			startLine, startCol = line, col
		}
	}
	endCol := startCol + len(fd.Name)
	return protocol.Location{
		URI: protocol.DocumentURI(uri.File(path)),
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(max0(startLine - 1)), Character: uint32(max0(startCol - 1))},
			End:   protocol.Position{Line: uint32(max0(startLine - 1)), Character: uint32(max0(endCol - 1))},
		},
	}
}

// findIdentAfter scans src starting at (line, col) (1-indexed) for the
// next bare-ident occurrence of name. Returns its 1-indexed line+col.
// Bare = not part of a longer identifier (boundary-checked).
func findIdentAfter(src string, fromLine, fromCol int, name string) (int, int, bool) {
	lines := strings.SplitAfter(src, "\n")
	curLine := 1
	for _, line := range lines {
		if curLine < fromLine {
			curLine++
			continue
		}
		startInLine := 0
		if curLine == fromLine {
			startInLine = fromCol - 1
			if startInLine < 0 {
				startInLine = 0
			}
			if startInLine > len(line) {
				startInLine = len(line)
			}
		}
		searchSlice := line[startInLine:]
		offset := 0
		for {
			i := strings.Index(searchSlice[offset:], name)
			if i < 0 {
				break
			}
			abs := startInLine + offset + i
			if isIdentBoundary(line, abs, len(name)) {
				return curLine, abs + 1, true
			}
			offset += i + 1
			if offset >= len(searchSlice) {
				break
			}
		}
		curLine++
	}
	return 0, 0, false
}

func isIdentBoundary(line string, start, length int) bool {
	if start > 0 {
		c := line[start-1]
		if isIdentChar(c) {
			return false
		}
	}
	end := start + length
	if end < len(line) {
		c := line[end]
		if isIdentChar(c) {
			return false
		}
	}
	return true
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// findInStdlibImports walks the file's imports for std/X paths, resolves
// each via the stdlib resolver, and looks for a FuncDecl named `name` in
// any of the resolved files. Best-effort fallback for the case where
// pipeline.Run didn't load stdlib modules into result.Modules.
func (s *Server) findInStdlibImports(path string, src string, name string) (protocol.Location, bool) {
	file := s.parseFile(path, src)
	if file == nil || len(file.Imports) == 0 {
		return protocol.Location{}, false
	}
	resolver := loader.NewStdlibResolver("", false, false)
	for _, imp := range file.Imports {
		if imp == nil || !strings.HasPrefix(imp.Path, "std/") {
			continue
		}
		stdPath, err := resolver.ResolveStdlib(imp.Path)
		if err != nil {
			continue
		}
		stdSrcBytes, err := os.ReadFile(stdPath)
		if err != nil {
			continue
		}
		if loc, ok := s.findFuncInFile(stdPath, string(stdSrcBytes), name); ok {
			return loc, true
		}
	}
	return protocol.Location{}, false
}

// parseFileForLSP wraps lexer + parser for use by definition / references /
// document-symbols handlers. Tolerates parse errors (returns the partial AST
// when the parser produced one).
func parseFileForLSP(path string, src string) *ast.File {
	l := lexer.New(src, path)
	p := parser.New(l)
	return p.ParseFile()
}

func (s *Server) parseFile(path string, src string) *ast.File {
	return parseFileForLSP(path, src)
}
