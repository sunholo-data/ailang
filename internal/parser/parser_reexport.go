package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseReexportDecl parses `export import M (names) [@deprecated("path")]`
// (M-PKG-PARALLEL-AGENTS C1). It is entered at EXPORT with IMPORT next and
// leaves the cursor on the line's last token, like parseImportDecl.
//
// A re-export is always selective: the facade's exports are the names it
// lists, so `export import M` and `export import M as A` are errors. The only
// annotation it takes is a trailing @deprecated naming the new path.
func (p *Parser) parseReexportDecl() *ast.ImportDecl {
	exportPos := p.curPos()
	p.nextToken() // at 'import'
	before := len(p.errors)
	imp := p.parseImportDecl()
	if imp == nil {
		// `export import M` alone fails inside parseImportDecl as a namespace
		// import; say what a re-export needs instead.
		if len(p.errors) == before+1 && isParserErrorCode(p.errors[before], "IMP012_UNSUPPORTED_NAMESPACE") {
			p.errors[before] = NewSuggestionError("PAR_REEXPORT_FORM", exportPos, p.curToken,
				"export import must list the names it re-exports",
				[]string{"export import module/path (Name1, Name2)"},
				"https://ailang.sunholo.com/docs/guides/package-publishing")
		}
		return nil
	}
	imp.Reexport = true
	imp.Pos = exportPos
	imp.Span.Start = exportPos
	if imp.ModuleAlias != "" || len(imp.Symbols) == 0 {
		p.errors = append(p.errors, NewSuggestionError("PAR_REEXPORT_FORM", exportPos, p.curToken,
			"export import must list the names it re-exports",
			[]string{fmt.Sprintf("export import %s (Name1, Name2)", imp.Path)},
			"https://ailang.sunholo.com/docs/guides/package-publishing"))
		p.skipTrailingAnnotations()
		return nil
	}
	if !p.peekTokenIs(lexer.AT) {
		return imp
	}
	p.nextToken() // at '@'
	ann := p.parseAnnotation()
	if ann == nil {
		return nil // parseAnnotation reported it (PAR_DEPRECATED_ARG, ...)
	}
	if ann.Name != ast.AnnotationDeprecated {
		p.report("PAR_REEXPORT_ANNOTATION",
			fmt.Sprintf("@%s cannot annotate export import; only @deprecated(\"new/module/path\") can", ann.Name),
			fmt.Sprintf("export import %s (%s) @deprecated(\"pkg/vendor/name/new/path\")", imp.Path, strings.Join(imp.Symbols, ", ")))
		return nil
	}
	if lit, ok := ann.Args[0].(*ast.Literal); ok {
		imp.MovedTo = strings.TrimSpace(fmt.Sprint(lit.Value))
	}
	imp.Span.End = p.curPos()
	return imp
}

// skipTrailingAnnotations consumes `@name(...)` suffixes after a rejected
// re-export so they do not cascade into further errors.
func (p *Parser) skipTrailingAnnotations() {
	for p.peekTokenIs(lexer.AT) {
		p.nextToken()
		before := len(p.errors)
		p.parseAnnotation()
		p.errors = p.errors[:before]
	}
}

func isParserErrorCode(err error, code string) bool {
	pe, ok := err.(*ParserError)
	return ok && pe.Code == code
}
