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
	"strings"

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

// InjectContract splices contract clauses (e.g. "requires { x >= 0 }\nensures { result > 0 }")
// into the top-level function declName, between the signature and the body — exactly where a
// hand-written contract sits (after `-> T ! {E}`, before the `=` of an equation-form body or
// the `{` of a block-form body). This is the R1 moat primitive: a best-of-N selector can inject
// a benchmark-PROVIDED spec the model omitted, then `ailang run --verify-contracts` becomes a
// reference-free oracle that rejects runs-but-WRONG candidates — something a general harness on
// an untyped language can't do. Returns an error if the decl or its body can't be located. The
// caller MUST re-run `ailang check` on the result (this does not validate semantics).
func InjectContract(src, filename, declName, contractText string) (string, error) {
	fn := FindFunc(src, filename, declName)
	if fn == nil {
		return "", fmt.Errorf("InjectContract: function %q not found in %s", declName, filename)
	}
	if fn.Body == nil {
		return "", fmt.Errorf("InjectContract: function %q has no body (extern?)", declName)
	}
	declOff, ok := lineColToOffset(src, fn.Pos.Line, fn.Pos.Column)
	if !ok {
		return "", fmt.Errorf("InjectContract: could not resolve decl offset for %q", declName)
	}
	bp := fn.Body.Position()
	bodyOff, ok := lineColToOffset(src, bp.Line, bp.Column)
	if !ok || bodyOff <= declOff {
		return "", fmt.Errorf("InjectContract: could not resolve body offset for %q", declName)
	}
	// Locate the body delimiter — `=` (equation form) or the body-opening `{` (block form) — by
	// scanning the signature at bracket depth 0. We skip params `(...)`, type brackets `[...]`,
	// and the effects brace `! {...}` (a `{` immediately preceded by `!`). Body.Position() is
	// unreliable here (it points at the body expression / a binop operator / the block's first
	// inner expr, not at the delimiter), so we cannot derive the delimiter from it.
	insertOff := -1
	depth := 0
	var prevNonWS byte
	for off := declOff; off < bodyOff; off++ {
		c := src[off]
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case '{':
			if depth == 0 && prevNonWS != '!' {
				insertOff = off
			}
			depth++
		case '}':
			depth--
		case '=':
			if depth == 0 && off+1 < len(src) && src[off+1] != '=' {
				insertOff = off
			}
		}
		if insertOff >= 0 {
			break
		}
		if c > ' ' {
			prevNonWS = c
		}
	}
	if insertOff < 0 {
		return "", fmt.Errorf("InjectContract: could not locate body delimiter for %q (record-type returns unsupported)", declName)
	}
	// Don't duplicate: if the decl already carries a contract (requires/ensures sits between its
	// signature and body, i.e. before insertOff), leave the source unchanged so the candidate's
	// OWN contract is verified directly — injecting a second clause makes the decl fail to parse.
	if strings.Contains(src[declOff:insertOff], "ensures") || strings.Contains(src[declOff:insertOff], "requires") {
		return src, nil
	}
	return src[:insertOff] + contractText + "\n" + src[insertOff:], nil
}
