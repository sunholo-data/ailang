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
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// Options controls formatter layout. Indent is the per-level indentation unit;
// an empty Indent defaults to two spaces (the canonical AILANG layout).
type Options struct {
	Indent string
}

// printer holds the mutable emission state for one Source call.
type printer struct {
	w *writer
	// att indexes comment attachments by owner+place+boundary for interleaving.
	// nil when formatting comment-free input (Phase-1 path), so the comment-free
	// output is byte-identical to Phase 1.
	att *attachIndex
}

// attachIndex is a fast lookup over the attachment set, keyed by owner pointer.
type attachIndex struct {
	// leading/floating keyed by (owner, boundaryIndex); trailing keyed by
	// (owner, childIndex).
	leading  map[attKey][]lexer.Comment
	floating map[attKey][]lexer.Comment
	trailing map[attKey][]lexer.Comment
	env      *Envelope
}

type attKey struct {
	owner any
	index int
}

func newAttachIndex(env *Envelope, atts []Attachment) *attachIndex {
	ix := &attachIndex{
		leading:  map[attKey][]lexer.Comment{},
		floating: map[attKey][]lexer.Comment{},
		trailing: map[attKey][]lexer.Comment{},
		env:      env,
	}
	for _, at := range atts {
		k := attKey{owner: at.Owner, index: at.Index}
		switch at.Place {
		case PlaceLeading:
			ix.leading[k] = append(ix.leading[k], at.Comment)
		case PlaceFloating:
			ix.floating[k] = append(ix.floating[k], at.Comment)
		case PlaceTrailing:
			ix.trailing[k] = append(ix.trailing[k], at.Comment)
		}
	}
	return ix
}

// commentText normalizes a scanned comment to its canonical single-line spelling
// (introducer + trimmed body); text is emitted verbatim from normalized source.
func commentText(c lexer.Comment) string { return strings.TrimRight(c.Text, " \t") }

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

// SourceWithComments renders a parsed program to canonical AILANG source with
// comments re-attached losslessly. source is the ORIGINAL file bytes (used to
// build the token-anchored envelope + collect comments); program is that source
// parsed. It fails closed (returning an error, no partial output) on any
// envelope/attachment inconsistency or a comment inside an interpolation hole.
//
// For comment-free input this produces byte-identical output to Source (the
// attachment set is empty and every interleaving lookup misses).
func SourceWithComments(program *ast.Program, source []byte, options Options) ([]byte, error) {
	if program == nil {
		return nil, fmt.Errorf("nil program")
	}
	if program.File == nil {
		return nil, fmt.Errorf("program has no File; formatter requires a parsed source file")
	}
	env, err := NewEnvelope(source)
	if err != nil {
		return nil, err
	}
	atts, err := AttachComments(env, program.File)
	if err != nil {
		return nil, err
	}
	indent := options.Indent
	if indent == "" {
		indent = "  "
	}
	p := &printer{w: newWriter(indent), att: newAttachIndex(env, atts)}
	if err := p.file(program.File); err != nil {
		return nil, err
	}
	out := ensureSingleTrailingNewline(p.w.string())
	return []byte(out), nil
}

// emitLeading emits any leading comments attached at (owner, boundary) each on
// its own line, before the child at that boundary.
func (p *printer) emitLeading(owner any, boundary int) {
	if p.att == nil {
		return
	}
	for _, c := range p.att.leading[attKey{owner: owner, index: boundary}] {
		p.w.write(commentText(c))
		p.w.hardline()
	}
}

// emitFloating emits any floating comments attached at (owner, boundary) each on
// its own line. followedByChild controls whether a hardline is emitted after the
// group (true when another child follows).
func (p *printer) emitFloating(owner any, boundary int) {
	if p.att == nil {
		return
	}
	group := p.att.floating[attKey{owner: owner, index: boundary}]
	for i, c := range group {
		p.w.write(commentText(c))
		if i < len(group)-1 {
			p.w.hardline()
		}
	}
}

// hasFloating reports whether any floating comment sits at (owner, boundary).
func (p *printer) hasFloating(owner any, boundary int) bool {
	if p.att == nil {
		return false
	}
	return len(p.att.floating[attKey{owner: owner, index: boundary}]) > 0
}

// emitTrailing emits a same-line trailing comment for (owner, child) after the
// child's text, on the same line (two-space gutter).
func (p *printer) emitTrailing(owner any, child int) {
	if p.att == nil {
		return
	}
	for _, c := range p.att.trailing[attKey{owner: owner, index: child}] {
		p.w.write("  " + commentText(c))
	}
}

// file emits the module declaration, then imports in AST order, then top-level
// declarations in AST order, with one blank line between top-level items. No
// reordering of imports, symbols, cases, fields, or declarations occurs.
func (p *printer) file(f *ast.File) error {
	// A "unit" is any top-level line group: the module decl, each import, and
	// each declaration. Blank lines separate consecutive units. The file top-level
	// is a multi-line ordered child list (M0 site #1): comments interleave at its
	// boundaries. Owner is the *ast.File; boundary k is before top-level child k.
	first := true
	sep := func() {
		if !first {
			p.w.blankline()
		}
		first = false
	}
	// idx tracks the current top-level boundary index across module/imports/decls.
	idx := 0
	// boundaryComments emits (after the separator) leading comments directly above
	// the upcoming unit, and floating comments as their own blank-line-separated
	// group above it. Order within a boundary is source order (floating then
	// leading is disallowed; the attacher assigns each comment one place, and a
	// floating group always precedes the leading run in source since a blank line
	// separates them).
	boundaryComments := func() {
		if p.hasFloating(f, idx) {
			sep()
			p.emitFloating(f, idx)
		}
		if p.att != nil && len(p.att.leading[attKey{owner: f, index: idx}]) > 0 {
			sep()
			p.emitLeading(f, idx)
			// Leading comments are directly above their unit: no blank between them
			// and the unit, so mark the run as "already open" — the unit follows on
			// the next line, not after another blankline.
			first = true
		}
	}

	emitUnit := func(emit func() error) error {
		boundaryComments()
		sep()
		if err := emit(); err != nil {
			return err
		}
		p.emitTrailing(f, idx)
		idx++
		return nil
	}

	if f.Module != nil {
		if err := emitUnit(func() error { p.w.write("module " + f.Module.Path); return nil }); err != nil {
			return err
		}
	}
	for _, imp := range f.Imports {
		imp := imp
		if err := emitUnit(func() error { p.importDecl(imp); return nil }); err != nil {
			return err
		}
	}
	for _, d := range f.Decls {
		d := d
		if err := emitUnit(func() error { return p.decl(d) }); err != nil {
			return err
		}
	}

	// Trailing comments after the last top-level node (boundary len) — rule 4 tail.
	if p.hasFloating(f, idx) || (p.att != nil && len(p.att.leading[attKey{owner: f, index: idx}]) > 0) {
		sep()
		p.emitFloating(f, idx)
		p.emitLeading(f, idx)
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
			// Boundary i: leading + floating comments before statement i.
			p.emitLeading(b, i)
			if p.hasFloating(b, i) {
				p.emitFloating(b, i)
				p.w.hardline()
			}
			if err = p.blockStatement(e); err != nil {
				return
			}
			// Emit the `;` separator (if the next statement needs it for round-trip
			// safety) FIRST — before the trailing comment — since a `--` comment runs
			// to end of line and would otherwise swallow the separator.
			if i < len(b.Exprs)-1 && !startsWithStatementStarter(b.Exprs[i+1]) {
				p.w.write(";")
			}
			// Same-line trailing comment after the statement (and its `;`).
			p.emitTrailing(b, i)
			p.w.hardline()
		}
		// Floating comments after the last statement (boundary len) — before close.
		if p.hasFloating(b, len(b.Exprs)) {
			p.emitFloating(b, len(b.Exprs))
			p.w.hardline()
		}
		p.emitLeading(b, len(b.Exprs))
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
//
// A block statement is rendered at precLowest, so the recursion tracks the
// parent precedence: if a subexpression renders PARENTHESISED (its own
// precedence is below the parent's required minimum), its first token is `(`,
// which is NOT a statement starter. Ignoring parenthesisation was a latent
// round-trip bug — e.g. `(a + b) * c` starts with `(` at the statement level,
// but its BinaryOp.Left recursion reaches the identifier `a` and wrongly
// reported "starter", so no `;` was emitted and re-parse glued it as a call.
func startsWithStatementStarter(e ast.Expr) bool {
	return startsWithStatementStarterAt(e, precLowest)
}

// startsWithStatementStarterAt mirrors the precedence-driven parenthesisation in
// expr.go (wrap/binaryOp/funcCall/recordAccess) to determine the first RENDERED
// token of e when emitted at parentPrec.
func startsWithStatementStarterAt(e ast.Expr, parentPrec int) bool {
	switch n := e.(type) {
	case *ast.Let, *ast.LetRec, *ast.If, *ast.Match, *ast.Identifier:
		return true
	case *ast.BinaryOp:
		prec := binaryPrecedence(n.Op)
		if prec < parentPrec {
			return false // renders parenthesised → starts with '('
		}
		leftMin := prec
		if rightAssociative(n.Op) {
			leftMin = prec + 1
		}
		return startsWithStatementStarterAt(n.Left, leftMin)
	case *ast.FuncCall:
		// Cons `a :: b` renders infix at precCons, starting with its left operand.
		if id, ok := n.Func.(*ast.Identifier); ok && id.Name == "::" && len(n.Args) == 2 {
			if precCons < parentPrec {
				return false
			}
			return startsWithStatementStarterAt(n.Args[0], precCons+1)
		}
		if precCall < parentPrec {
			return false
		}
		return startsWithStatementStarterAt(n.Func, precCall)
	case *ast.RecordAccess:
		if precDotAccess < parentPrec {
			return false
		}
		return startsWithStatementStarterAt(n.Record, precDotAccess)
	case *ast.Send:
		return startsWithStatementStarterAt(n.Channel, precLowest+1)
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
