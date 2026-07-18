package format

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// expr.go prints expression nodes. Each printer receives the parent binding
// power (parentPrec) and reconstructs parentheses whenever omitting them could
// re-associate the parsed AST. Because the AST has no ParenExpr node, redundant
// source parentheses are simply absent from the tree and therefore dropped.

// expr renders an expression into the writer. parentPrec is the minimum binding
// power the enclosing context imposes: if the expression's own precedence is
// lower, it must be wrapped in parentheses.
func (p *printer) expr(e ast.Expr, parentPrec int) error {
	if e == nil {
		return fmt.Errorf("nil expression node")
	}
	switch n := e.(type) {
	case *ast.Identifier:
		p.w.write(n.Name)
		return nil
	case *ast.Literal:
		s, err := literalString(n)
		if err != nil {
			return err
		}
		p.w.write(s)
		return nil
	case *ast.BinaryOp:
		return p.binaryOp(n, parentPrec)
	case *ast.UnaryOp:
		return p.unaryOp(n, parentPrec)
	case *ast.FuncCall:
		return p.funcCall(n, parentPrec)
	case *ast.RecordAccess:
		return p.recordAccess(n, parentPrec)
	case *ast.Lambda:
		return p.wrap(parentPrec, precLambda, func() error { return p.lambda(n) })
	case *ast.FuncLit:
		return p.wrap(parentPrec, precAtom, func() error { return p.funcLit(n) })
	case *ast.Let:
		// A nil-Body let is a statement binding; it should only appear inside a
		// block statement list, printed by the block emitter. Reaching it here
		// (as a value expression) means it is an explicit let..in with a Body, or
		// a malformed tree. Explicit let..in binds loosely, like a lambda.
		return p.wrap(parentPrec, precLambda, func() error { return p.letIn(n) })
	case *ast.LetRec:
		return p.wrap(parentPrec, precLambda, func() error { return p.letRecIn(n) })
	case *ast.Block:
		return p.block(n)
	case *ast.If:
		return p.wrap(parentPrec, precLambda, func() error { return p.ifExpr(n) })
	case *ast.Match:
		return p.wrap(parentPrec, precLambda, func() error { return p.match(n) })
	case *ast.List:
		return p.seq("[", "]", n.Elements)
	case *ast.Array:
		return p.seq("#[", "]", n.Elements)
	case *ast.Tuple:
		return p.tuple(n)
	case *ast.Record:
		return p.record(n)
	case *ast.RecordUpdate:
		return p.recordUpdate(n)
	case *ast.QuasiQuote:
		return p.quasiQuote(n)
	case *ast.Send:
		return p.wrap(parentPrec, precLowest+1, func() error { return p.send(n) })
	case *ast.Recv:
		return p.wrap(parentPrec, precPrefix, func() error { return p.recv(n) })
	case *ast.ForallExpr:
		return p.wrap(parentPrec, precLambda, func() error { return p.forall(n) })
	case *ast.AssertStmt:
		// AssertStmt implements exprNode() for use inside test bodies.
		return p.assertExpr(n)
	case *ast.Error:
		return fmt.Errorf("cannot format ast.Error node: %s", n.Msg)
	default:
		return fmt.Errorf("unsupported expression node: %T", e)
	}
}

// wrap runs emit, surrounding it with parentheses when the node's own precedence
// (selfPrec) is lower than the parent context requires.
func (p *printer) wrap(parentPrec, selfPrec int, emit func() error) error {
	paren := selfPrec < parentPrec
	if paren {
		p.w.write("(")
	}
	if err := emit(); err != nil {
		return err
	}
	if paren {
		p.w.write(")")
	}
	return nil
}

func (p *printer) binaryOp(n *ast.BinaryOp, parentPrec int) error {
	prec := binaryPrecedence(n.Op)
	return p.wrap(parentPrec, prec, func() error {
		// For left-associative ops, the left operand may share the same
		// precedence without parens; the right operand needs prec+1. Cons (::)
		// is right-associative, so the sides swap.
		leftMin, rightMin := prec+1, prec+1
		if rightAssociative(n.Op) {
			leftMin = prec + 1
			rightMin = prec
		} else {
			leftMin = prec
			rightMin = prec + 1
		}
		if err := p.expr(n.Left, leftMin); err != nil {
			return err
		}
		p.w.write(" " + n.Op + " ")
		return p.expr(n.Right, rightMin)
	})
}

func (p *printer) unaryOp(n *ast.UnaryOp, parentPrec int) error {
	return p.wrap(parentPrec, precPrefix, func() error {
		// `not` is an alphabetic operator and needs a trailing space; symbolic
		// operators (-, !, ~) bind directly to their operand.
		if isWordOperator(n.Op) {
			p.w.write(n.Op + " ")
		} else {
			p.w.write(n.Op)
		}
		return p.expr(n.Expr, precPrefix)
	})
}

// isWordOperator reports whether a unary operator is spelled with letters and
// therefore needs a separating space before its operand.
func isWordOperator(op string) bool {
	for _, r := range op {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func (p *printer) funcCall(n *ast.FuncCall, parentPrec int) error {
	// The cons operator `a :: b` desugars to FuncCall{Func: `::`, Args: [a, b]}.
	// `::` is an infix-only operator: the parser has no prefix/call form for it,
	// so `::(a, b)` does NOT re-parse. It must be re-emitted in infix form to
	// round-trip. Cons is right-associative at precCons.
	if id, ok := n.Func.(*ast.Identifier); ok && id.Name == "::" && len(n.Args) == 2 {
		return p.wrap(parentPrec, precCons, func() error {
			if err := p.expr(n.Args[0], precCons+1); err != nil {
				return err
			}
			p.w.write(" :: ")
			return p.expr(n.Args[1], precCons)
		})
	}
	return p.wrap(parentPrec, precCall, func() error {
		// The callee binds at call precedence; a lower-precedence callee (e.g. a
		// lambda) is parenthesised by its own printer via precCall.
		if err := p.expr(n.Func, precCall); err != nil {
			return err
		}
		p.w.write("(")
		for i, a := range n.Args {
			if i > 0 {
				p.w.write(", ")
			}
			if err := p.expr(a, precLowest); err != nil {
				return err
			}
		}
		p.w.write(")")
		return nil
	})
}

func (p *printer) recordAccess(n *ast.RecordAccess, parentPrec int) error {
	return p.wrap(parentPrec, precDotAccess, func() error {
		if err := p.expr(n.Record, precDotAccess); err != nil {
			return err
		}
		p.w.write("." + n.Field)
		return nil
	})
}

func (p *printer) lambda(n *ast.Lambda) error {
	p.w.write("\\")
	for i, param := range n.Params {
		if i > 0 {
			p.w.write(" ")
		}
		p.w.write(param.Name)
	}
	p.w.write(". ")
	return p.expr(n.Body, precLambda)
}

func (p *printer) funcLit(n *ast.FuncLit) error {
	p.w.write("func(")
	if err := p.params(n.Params); err != nil {
		return err
	}
	p.w.write(")")
	if n.ReturnType != nil {
		rt, err := p.typeString(n.ReturnType)
		if err != nil {
			return err
		}
		p.w.write(" -> " + rt)
	}
	if eff := ast.FormatEffects(n.Effects); eff != "" {
		p.w.write(" " + eff)
	}
	p.w.write(" ")
	// A func literal body is a single expression; wrap in a braced block only if
	// it is itself a multi-expression block.
	return p.bodyBraced(n.Body)
}

// letIn prints an explicit `let name = value in body`.
func (p *printer) letIn(n *ast.Let) error {
	p.w.write("let " + n.Name)
	if n.Type != nil {
		ts, err := p.typeString(n.Type)
		if err != nil {
			return err
		}
		p.w.write(": " + ts)
	}
	p.w.write(" = ")
	if err := p.expr(n.Value, precLowest); err != nil {
		return err
	}
	p.w.write(" in ")
	return p.expr(n.Body, precLambda)
}

func (p *printer) letRecIn(n *ast.LetRec) error {
	p.w.write("letrec " + n.Name)
	if n.Type != nil {
		ts, err := p.typeString(n.Type)
		if err != nil {
			return err
		}
		p.w.write(": " + ts)
	}
	p.w.write(" = ")
	if err := p.expr(n.Value, precLowest); err != nil {
		return err
	}
	p.w.write(" in ")
	return p.expr(n.Body, precLambda)
}

func (p *printer) ifExpr(n *ast.If) error {
	p.w.write("if ")
	if err := p.expr(n.Condition, precLowest); err != nil {
		return err
	}
	p.w.write(" then ")
	if err := p.expr(n.Then, precLambda); err != nil {
		return err
	}
	p.w.write(" else ")
	return p.expr(n.Else, precLambda)
}

func (p *printer) match(n *ast.Match) error {
	p.w.write("match ")
	if err := p.expr(n.Expr, precLowest); err != nil {
		return err
	}
	p.w.write(" {")
	p.w.hardline()
	// Emit cases at one indentation level, comma-separated per source grammar.
	var caseErr error
	p.w.indented(func() {
		for i, c := range n.Cases {
			if caseErr != nil {
				return
			}
			pat, err := p.patternString(c.Pattern)
			if err != nil {
				caseErr = err
				return
			}
			p.w.write(pat)
			if c.Guard != nil {
				p.w.write(" if ")
				if err := p.expr(c.Guard, precLowest); err != nil {
					caseErr = err
					return
				}
			}
			p.w.write(" => ")
			if err := p.expr(c.Body, precLambda); err != nil {
				caseErr = err
				return
			}
			if i < len(n.Cases)-1 {
				p.w.write(",")
			}
			p.w.hardline()
		}
	})
	if caseErr != nil {
		return caseErr
	}
	p.w.write("}")
	return nil
}

// seq prints a bracketed, comma-separated element sequence on one line.
func (p *printer) seq(open, close string, elems []ast.Expr) error {
	p.w.write(open)
	for i, e := range elems {
		if i > 0 {
			p.w.write(", ")
		}
		if err := p.expr(e, precLowest); err != nil {
			return err
		}
	}
	p.w.write(close)
	return nil
}

func (p *printer) tuple(n *ast.Tuple) error {
	// A tuple always prints with parentheses; a 1-tuple keeps a trailing comma so
	// it does not re-parse as a grouped expression.
	p.w.write("(")
	for i, e := range n.Elements {
		if i > 0 {
			p.w.write(", ")
		}
		if err := p.expr(e, precLowest); err != nil {
			return err
		}
	}
	if len(n.Elements) == 1 {
		p.w.write(",")
	}
	p.w.write(")")
	return nil
}

func (p *printer) record(n *ast.Record) error {
	if len(n.Fields) == 0 {
		p.w.write("{}")
		return nil
	}
	p.w.write("{ ")
	for i, f := range n.Fields {
		if i > 0 {
			p.w.write(", ")
		}
		p.w.write(f.Name + ": ")
		if err := p.expr(f.Value, precLowest); err != nil {
			return err
		}
	}
	p.w.write(" }")
	return nil
}

func (p *printer) recordUpdate(n *ast.RecordUpdate) error {
	p.w.write("{ ")
	if err := p.expr(n.Base, precLowest); err != nil {
		return err
	}
	p.w.write(" | ")
	for i, f := range n.Fields {
		if i > 0 {
			p.w.write(", ")
		}
		p.w.write(f.Name + ": ")
		if err := p.expr(f.Value, precLowest); err != nil {
			return err
		}
	}
	p.w.write(" }")
	return nil
}

func (p *printer) quasiQuote(n *ast.QuasiQuote) error {
	// Quasiquote template contents are never reinterpreted: emit verbatim between
	// the kind keyword and the triple-quote delimiters.
	if strings.Contains(n.Template, `"""`) {
		return fmt.Errorf("quasiquote template contains triple-quote delimiter; cannot format losslessly")
	}
	p.w.write(n.Kind + `"""` + n.Template + `"""`)
	return nil
}

func (p *printer) send(n *ast.Send) error {
	if err := p.expr(n.Channel, precLowest+1); err != nil {
		return err
	}
	p.w.write(" <- ")
	return p.expr(n.Value, precLowest+1)
}

func (p *printer) recv(n *ast.Recv) error {
	p.w.write("<- ")
	return p.expr(n.Channel, precPrefix)
}

func (p *printer) forall(n *ast.ForallExpr) error {
	p.w.write("forall " + n.Var + ": ")
	if err := p.expr(n.Lo, precLowest); err != nil {
		return err
	}
	p.w.write("..")
	if err := p.expr(n.Hi, precLowest); err != nil {
		return err
	}
	p.w.write(" => ")
	return p.expr(n.Body, precLambda)
}

func (p *printer) assertExpr(n *ast.AssertStmt) error {
	p.w.write("assert ")
	if err := p.expr(n.Condition, precLowest); err != nil {
		return err
	}
	if n.Message != "" {
		p.w.write(", " + escapeString(n.Message))
	}
	return nil
}
