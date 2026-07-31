package testing

import "github.com/sunholo-data/ailang/internal/ast"

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

	sourceLines := splitLines(source)
	skipRanges := functionSkipRanges(file, keep)
	skipRanges = append(skipRanges, testAndPropertySkipRanges(sourceLines, file)...)

	lines := make([]string, 0, len(sourceLines))
	for i, line := range sourceLines {
		if !lineInRanges(i+1, skipRanges) {
			lines = append(lines, line)
		}
	}
	return joinLines(lines)
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

// testAndPropertySkipRanges preserves the existing brace-depth treatment of
// declarations already collected as test cases.
func testAndPropertySkipRanges(sourceLines []string, file *ast.File) []sourceLineRange {
	var ranges []sourceLineRange
	for _, decl := range file.Decls {
		var startLine int
		switch d := decl.(type) {
		case *ast.TestDecl:
			startLine = d.Pos.Line
		case *ast.PropertyDecl:
			startLine = d.Pos.Line
		default:
			continue
		}

		depth := 0
		endLine := startLine
		for i := startLine - 1; i < len(sourceLines); i++ {
			for _, ch := range sourceLines[i] {
				switch ch {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						endLine = i + 1
						goto foundEnd
					}
				}
			}
		}
	foundEnd:
		ranges = append(ranges, sourceLineRange{start: startLine, end: endLine})
	}
	return ranges
}

func lineInRanges(line int, ranges []sourceLineRange) bool {
	for _, r := range ranges {
		if line >= r.start && line <= r.end {
			return true
		}
	}
	return false
}
