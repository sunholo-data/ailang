// Package astedit applies span-anchored, semantic edits to AILANG source.
//
// M-AILANG-NATIVE-HARNESS (semantic-edit experiment): instead of fragile text patches
// or full-file rewrites (the measured agent thrash), an edit names a top-level
// declaration; we locate its EXACT parsed source span and splice new text into just
// that range, preserving the rest of the file byte-for-byte. This gives semantic
// precision (find the decl exactly, via the parser) WITHOUT needing a round-trip
// pretty-printer — the hard part of full AST surgery. A faithful AST↔source formatter
// (backlog) would unlock richer sub-declaration edits; decl-level splice is the cheap
// first rung that is testable now.
package astedit

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// FindFunc parses src and returns the top-level FuncDecl named declName (or nil).
func FindFunc(src, filename, declName string) *ast.FuncDecl {
	prog := parser.New(lexer.New(src, filename)).Parse()
	if prog == nil || prog.File == nil {
		return nil
	}
	for _, fn := range prog.File.Funcs {
		if fn != nil && fn.Name == declName {
			return fn
		}
	}
	return nil
}

// ReplaceDecl replaces the source of the top-level function declaration named declName
// with newText, splicing by its parsed span. The rest of the file is preserved exactly.
// Returns an error if the declaration is not found or its span is unusable. The caller
// should re-run `ailang check` on the result — this function does not validate semantics.
//
// Span handling (experiment-grade): the parser populates Span line/col but not byte
// Offset, and Span.Start begins at `func` (excluding a leading `export`/`pure` modifier).
// We therefore splice from the START OF THE DECL'S LINE (captures the modifier) through
// the End column inclusive. This is correct for top-level decls without preceding-line
// annotations. The PRODUCTION fix is edit-grade parser spans (byte offsets + full-decl
// boundary incl. modifiers/annotations) — see m-ailang-native-harness.md (parser route).
func ReplaceDecl(src, filename, declName, newText string) (string, error) {
	fn := FindFunc(src, filename, declName)
	if fn == nil {
		return "", fmt.Errorf("declaration %q not found", declName)
	}
	start, ok1 := lineColToOffset(src, fn.Span.Start.Line, 1) // line start → includes export/pure
	endAt, ok2 := lineColToOffset(src, fn.Span.End.Line, fn.Span.End.Column)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("declaration %q has unusable span (start=%v end=%v)", declName, fn.Span.Start, fn.Span.End)
	}
	end := endAt + 1 // End col points AT the final char (closing brace) → include it
	if start < 0 || end > len(src) || start >= end {
		return "", fmt.Errorf("declaration %q span out of range [%d,%d) len=%d", declName, start, end, len(src))
	}
	return src[:start] + newText + src[end:], nil
}

// lineColToOffset converts a 1-based line / 1-based column to a byte offset in src.
// (Workaround until the lexer populates Pos.Offset — see the parser route in the design.)
func lineColToOffset(src string, line, col int) (int, bool) {
	if line < 1 || col < 1 {
		return 0, false
	}
	cur, i := 1, 0
	for cur < line && i < len(src) {
		if src[i] == '\n' {
			cur++
		}
		i++
	}
	if cur != line {
		return 0, false
	}
	off := i + (col - 1)
	if off > len(src) {
		return 0, false
	}
	return off, true
}
