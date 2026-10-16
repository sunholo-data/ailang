package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

type sourceLineRange struct {
	start int
	end   int
}

// stripNonPureFunctions removes non-pure function declarations and test/property
// blocks from source code. keepFuncs names declarations that must survive even
// when they are effectful, such as the function currently being extracted.
func (e *Executor) stripNonPureFunctions(source string, file *ast.File, keepFuncs ...string) string {
	keep := make(map[string]bool, len(keepFuncs))
	for _, name := range keepFuncs {
		keep[name] = true
	}

	stripped, _ := e.stripWithLineMap(source, file, keep)
	return stripped
}

// stripWithLineMap is stripNonPureFunctions plus, for each kept line (index
// i = stripped line i+1), its 1-based line in source. Removed blocks shift
// every later line; the named-test batch maps runtime-error positions back
// through it (named_batch.go).
func (e *Executor) stripWithLineMap(source string, file *ast.File, keep map[string]bool) (string, []int) {
	sourceLines := splitLines(source)
	skipRanges := functionSkipRanges(file, keep)
	skipRanges = append(skipRanges, testAndPropertySkipRanges(sourceLines, file)...)

	return stripSourceLines(sourceLines, skipRanges)
}

// stripTestBlocks retains the complete module for named tests and properties.
// Purity is checked on generated entries, rather than inferred from syntax.
func (e *Executor) stripTestBlocks(source string, file *ast.File) (string, []int) {
	lines := splitLines(source)
	return stripSourceLines(lines, testAndPropertySkipRanges(lines, file))
}

func stripSourceLines(sourceLines []string, skipRanges []sourceLineRange) (string, []int) {
	lines := make([]string, 0, len(sourceLines))
	orig := make([]int, 0, len(sourceLines))
	for i, line := range sourceLines {
		if !lineInRanges(i+1, skipRanges) {
			lines = append(lines, line)
			orig = append(orig, i+1)
		}
	}
	return joinLines(lines), orig
}

func functionSkipRanges(file *ast.File, keep map[string]bool) []sourceLineRange {
	var ranges []sourceLineRange
	for _, f := range file.Funcs {
		if keep[f.Name] || isEffectivelyPure(f) {
			continue
		}

		startLine := f.Span.Start.Line
		endLine := f.Span.End.Line
		if endLine == 0 || endLine < startLine {
			startLine = f.Pos.Line
			endLine = f.Pos.Line
		} else {
			for _, annotation := range f.Annotations {
				if annotation.Pos.Line > 0 && annotation.Pos.Line < startLine {
					startLine = annotation.Pos.Line
				}
			}
		}
		ranges = append(ranges, sourceLineRange{start: startLine, end: endLine})
	}
	return ranges
}

// isEffectivelyPure mirrors the fixup at cmd/ailang/verify.go:160-168 and
// cmd/ailang/ai_check.go:231-232: an explicit empty effect annotation (`! {}`)
// is semantically pure, although the parser sets IsPure only for `pure`.
// This stays local because internal/format/decl.go uses IsPure to emit source.
func isEffectivelyPure(f *ast.FuncDecl) bool {
	return f.IsPure || (f.Effects != nil && len(f.Effects) == 0)
}

// testAndPropertySkipRanges trusts parser-recorded closing positions. The lexer
// scan is retained only for manually built ASTs with an unset End, preserving
// compatibility with existing callers without re-lexing parser-created nodes.
func testAndPropertySkipRanges(sourceLines []string, file *ast.File) []sourceLineRange {
	var braces []lexer.Token
	collected := false
	var ranges []sourceLineRange
	for _, decl := range file.Decls {
		var startLine, endLine int
		switch d := decl.(type) {
		case *ast.TestDecl:
			startLine, endLine = d.Pos.Line, d.End.Line
		case *ast.PropertyDecl:
			startLine, endLine = d.Pos.Line, d.End.Line
		default:
			continue
		}
		if endLine != 0 {
			ranges = append(ranges, sourceLineRange{start: startLine, end: endLine})
			continue
		}
		if !collected {
			braces = braceTokens(joinLines(sourceLines))
			collected = true
		}
		ranges = append(ranges, sourceLineRange{start: startLine, end: blockEndLine(braces, startLine)})
	}
	return ranges
}

// braceTokens lexes source and keeps only its LBRACE/RBRACE tokens, in order.
func braceTokens(source string) []lexer.Token {
	var out []lexer.Token
	l := lexer.New(source, "")
	for {
		tok := l.NextToken()
		if tok.Type == lexer.EOF {
			return out
		}
		if tok.Type == lexer.LBRACE || tok.Type == lexer.RBRACE {
			out = append(out, tok)
		}
	}
}

// blockEndLine returns the line of the brace closing the first block opened at
// or after startLine. An unterminated block yields startLine, as before.
func blockEndLine(braces []lexer.Token, startLine int) int {
	depth := 0
	for _, tok := range braces {
		if tok.Line < startLine {
			continue
		}
		if tok.Type == lexer.LBRACE {
			depth++
			continue
		}
		if depth == 0 {
			continue
		}
		depth--
		if depth == 0 {
			return tok.Line
		}
	}
	return startLine
}

func lineInRanges(line int, ranges []sourceLineRange) bool {
	for _, r := range ranges {
		if line >= r.start && line <= r.end {
			return true
		}
	}
	return false
}
