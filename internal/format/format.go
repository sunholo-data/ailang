// Package format implements ailang fmt's canonical AILANG source printer.
//
// Source(program, options) walks a parsed *ast.Program and emits one canonical
// textual representation. It is an exhaustive visitor: every concrete AST node
// kind has a real printer, or an explicit unsupported-node error. It NEVER falls
// back to a node's debug String() method, which is a non-precedence-safe
// debugging rendering rather than valid source.
//
// The printer consumes only the parsed AST plus fixed options. It does not
// consult the type checker, elaborator, filesystem, environment, or clock, so
// output depends only on the AST. Comments are out of scope for Phase 1: the
// AST carries no trivia, so callers must reject commented input up front with
// HasComments (see comments.go). print.go's JSON contract is never touched.
package format

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Options controls formatter layout. Indent is the per-level indentation unit;
// an empty Indent defaults to two spaces (the canonical AILANG layout).
type Options struct {
	Indent string
}

// printer holds the mutable emission state for one Source call.
type printer struct {
	w *writer
}

// Source renders a parsed program to canonical AILANG source. The program must
// be comment-free (callers enforce this via HasComments) and free of parse
// errors. It returns an error if any node is nil-required, an ast.Error, or an
// unsupported concrete node kind — there is no fallback to original source.
func Source(program *ast.Program, options Options) ([]byte, error) {
	if program == nil {
		return nil, fmt.Errorf("nil program")
	}
	if program.File == nil {
		return nil, fmt.Errorf("program has no File; formatter requires a parsed source file")
	}
	indent := options.Indent
	if indent == "" {
		indent = "  "
	}
	p := &printer{w: newWriter(indent)}
	if err := p.file(program.File); err != nil {
		return nil, err
	}
	out := p.w.string()
	// Guarantee exactly one trailing newline (LF), even for an empty file.
	out = ensureSingleTrailingNewline(out)
	return []byte(out), nil
}

// file emits the module declaration, then imports in AST order, then top-level
// declarations in AST order, with one blank line between top-level items. No
// reordering of imports, symbols, cases, fields, or declarations occurs.
func (p *printer) file(f *ast.File) error {
	// A "unit" is any top-level line group: the module decl, each import, and
	// each declaration. Blank lines separate consecutive units.
	first := true
	sep := func() {
		if !first {
			p.w.blankline()
		}
		first = false
	}

	if f.Module != nil {
		sep()
		p.w.write("module " + f.Module.Path)
	}

	for _, imp := range f.Imports {
		sep()
		p.importDecl(imp)
	}

	for _, d := range f.Decls {
		sep()
		if err := p.decl(d); err != nil {
			return err
		}
	}

	if !first {
		// Close the final unit's line so ensureSingleTrailingNewline sees a break.
		p.w.hardline()
	}
	return nil
}

// importDecl emits a canonical import line, preserving symbol order and aliases.
func (p *printer) importDecl(imp *ast.ImportDecl) {
	p.w.write("import ")
	if imp.ModuleAlias != "" {
		p.w.write(imp.Path + " as " + imp.ModuleAlias)
	} else {
		p.w.write(imp.Path)
	}
	if len(imp.Symbols) > 0 {
		parts := make([]string, len(imp.Symbols))
		for i, sym := range imp.Symbols {
			if alias, ok := imp.SymbolAliases[sym]; ok {
				parts[i] = sym + " as " + alias
			} else {
				parts[i] = sym
			}
		}
		p.w.write(" (")
		for i, part := range parts {
			if i > 0 {
				p.w.write(", ")
			}
			p.w.write(part)
		}
		p.w.write(")")
	}
}

// block prints an ast.Block appearing in expression (value) position — an if/then/
// else branch, a let value, a match-arm body, etc. In these positions the parser
// (parseRecordLiteral's block path) preserves a one-expression block AS a Block,
// unlike a function/func-lit body which unwraps it. So a value-position Block is
// ALWAYS braced, including a single-expression one, to re-parse identically.
func (p *printer) block(b *ast.Block) error {
	return p.blockBraced(b)
}

// bodyBraced prints a func-literal body: a braced block if it is a multi- or
// zero-expression block, otherwise a braced single-statement block. Func-literal
// grammar always requires braces, so even a single expression is wrapped.
func (p *printer) bodyBraced(body ast.Expr) error {
	if blk, ok := body.(*ast.Block); ok {
		return p.blockBraced(blk)
	}
	// Wrap a bare single expression in a one-statement braced block.
	p.w.write("{")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		err = p.blockStatement(body)
		p.w.hardline()
	})
	if err != nil {
		return err
	}
	p.w.write("}")
	return nil
}

// blockBraced prints a braced newline-per-statement block: each non-final
// expression on its own line, the final value expression bare.
//
// The canonical separator is a newline. The parser, however, only treats a
// newline as a statement separator when the FOLLOWING statement begins with a
// statement-starter token (let/letrec/if/match/identifier); a statement that
// begins with `(`, `[`, a literal, `-`, etc. would otherwise be glued onto the
// previous statement as a call/operand (e.g. `f(x)\n(y)` re-parses as
// `f(x)(y)`). To keep the fail-closed round-trip guarantee, a single `;` is
// emitted before such a statement — and ONLY then. The common case stays
// semicolon-free, per the canonical form.
func (p *printer) blockBraced(b *ast.Block) error {
	p.w.write("{")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		for i, e := range b.Exprs {
			if err != nil {
				return
			}
			if err = p.blockStatement(e); err != nil {
				return
			}
			// Decide the separator to the NEXT statement.
			if i < len(b.Exprs)-1 {
				if startsWithStatementStarter(b.Exprs[i+1]) {
					p.w.hardline()
				} else {
					// Unsafe for a bare newline: terminate this statement with `;`.
					p.w.write(";")
					p.w.hardline()
				}
			} else {
				p.w.hardline()
			}
		}
	})
	if err != nil {
		return err
	}
	p.w.write("}")
	return nil
}

// startsWithStatementStarter reports whether the rendered form of a block
// statement begins with a token the parser accepts as a newline-separated
// statement starter (let, letrec, if, match, or an identifier). This mirrors
// parser.peekStartsBlockStatement. When false, the preceding statement must be
// `;`-terminated so the two do not merge on re-parse.
func startsWithStatementStarter(e ast.Expr) bool {
	switch n := e.(type) {
	case *ast.Let, *ast.LetRec, *ast.If, *ast.Match, *ast.Identifier:
		return true
	case *ast.BinaryOp:
		return startsWithStatementStarter(n.Left)
	case *ast.FuncCall:
		// Cons `a :: b` renders infix, starting with its left operand.
		if id, ok := n.Func.(*ast.Identifier); ok && id.Name == "::" && len(n.Args) == 2 {
			return startsWithStatementStarter(n.Args[0])
		}
		return startsWithStatementStarter(n.Func)
	case *ast.RecordAccess:
		return startsWithStatementStarter(n.Record)
	case *ast.Send:
		return startsWithStatementStarter(n.Channel)
	default:
		return false
	}
}

// blockStatement prints one statement inside a sequence. A nil-Body let is a
// statement binding and prints as `let name = value` (no `in`); every other
// expression prints normally.
func (p *printer) blockStatement(e ast.Expr) error {
	if let, ok := e.(*ast.Let); ok && let.Body == nil {
		return p.statementLet(let)
	}
	if lr, ok := e.(*ast.LetRec); ok && lr.Body == nil {
		return p.statementLetRec(lr)
	}
	return p.expr(e, precLowest)
}

func (p *printer) statementLet(let *ast.Let) error {
	p.w.write("let " + let.Name)
	if let.Type != nil {
		ts, err := p.typeString(let.Type)
		if err != nil {
			return err
		}
		p.w.write(": " + ts)
	}
	p.w.write(" = ")
	return p.expr(let.Value, precLowest)
}

func (p *printer) statementLetRec(lr *ast.LetRec) error {
	p.w.write("letrec " + lr.Name)
	if lr.Type != nil {
		ts, err := p.typeString(lr.Type)
		if err != nil {
			return err
		}
		p.w.write(": " + ts)
	}
	p.w.write(" = ")
	return p.expr(lr.Value, precLowest)
}

// ensureSingleTrailingNewline trims trailing newlines and appends exactly one,
// yielding LF output that ends in a single final newline.
func ensureSingleTrailingNewline(s string) string {
	end := len(s)
	for end > 0 && s[end-1] == '\n' {
		end--
	}
	return s[:end] + "\n"
}
