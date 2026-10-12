package format

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// sourceMultiline attaches only a complete source string whose decoded payload
// agrees with the AST. Synthetic interpolation fragments never get emitted alone.
func (p *printer) sourceMultiline(expr ast.Expr) (string, bool) {
	if p.att == nil || p.measuring {
		return "", false
	}
	switch expr.(type) {
	case *ast.Literal, *ast.FuncCall:
	default:
		return "", false
	}
	env := p.att.env
	off, err := env.offsetOf(expr.Position().Line, expr.Position().Column)
	if err != nil {
		return "", false
	}
	i := sort.Search(len(p.multilineSpans), func(i int) bool { return p.multilineSpans[i].end > off })
	if i < len(p.multilineSpans) {
		sp := p.multilineSpans[i]
		if off < sp.start {
			return "", false
		}
		raw := env.src[sp.start:sp.end]
		if !strings.HasPrefix(raw, "\"") || strings.HasPrefix(raw, "\"\"\"") || !strings.Contains(raw, "\n") {
			return "", false
		}
		if lit, ok := expr.(*ast.Literal); ok {
			if off != sp.start || lit.Kind != ast.StringLit {
				return "", false
			}
			l := lexer.New(raw, "multiline")
			tok := l.NextToken()
			if tok.Type != lexer.STRING || l.NextToken().Type != lexer.EOF || tok.Literal != lit.Value {
				return "", false
			}
			return raw, true
		}
		// Reparse the full interpolation. AST-only rendering is reversible and
		// checks both decoded text segments and hole expressions, including builtin
		// resolution. It also prevents a partial concat subtree claiming the token.
		rp := parser.New(lexer.New("pure func preserved() = "+raw, "multiline"))
		file := rp.ParseFile()
		if len(rp.Errors()) != 0 || len(file.Funcs) != 1 {
			return "", false
		}
		render := func(e ast.Expr) (string, error) {
			sub := &printer{w: newWriter(p.w.indent), measuring: true}
			err := sub.expr(e, precLowest)
			return sub.w.string(), err
		}
		want, e1 := render(expr)
		body := file.Funcs[0].Body
		if block, ok := body.(*ast.Block); ok && len(block.Exprs) == 1 {
			body = block.Exprs[0]
		}
		got, e2 := render(body)
		if e1 == nil && e2 == nil && want == got {
			return raw, true
		}
		return "", false
	}
	return "", false
}

// multilineStrings builds the per-printer source index once, rather than
// rescanning every source literal on every expression visit.
func (e *Envelope) multilineStrings() []interpHole {
	var spans []interpHole
	for _, sp := range e.stringSpans() {
		raw := e.src[sp.start:sp.end]
		if strings.HasPrefix(raw, "\"") && !strings.HasPrefix(raw, "\"\"\"") && strings.Contains(raw, "\n") {
			spans = append(spans, sp)
		}
	}
	return spans
}

// writeVerbatim indents only the opening line. Interior whitespace is payload,
// while the final column still has to be correct for following width decisions.
func (w *writer) writeVerbatim(s string) {
	w.write(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		w.col = utf8.RuneCountInString(s[i+1:])
		w.atBOL = false
	}
}
