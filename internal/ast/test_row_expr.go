package ast

import "fmt"

// RowExprProblem describes syntax that the lightweight inline-test evaluator
// cannot execute. It does not validate names, types or function-call effects.
type RowExprProblem struct {
	Code    string
	Message string
	Pos     Pos
}

func (p *RowExprProblem) Error() string { return p.Code + ": " + p.Message }

// RowExprSupported checks both row arms against the harness grammar. Operators
// requiring dictionary elaboration belong in named test blocks instead.
func RowExprSupported(expr Expr) *RowExprProblem {
	reject := func(code, message string) *RowExprProblem {
		pos := Pos{}
		if expr != nil {
			pos = expr.Position()
		}
		return &RowExprProblem{Code: code, Message: message, Pos: pos}
	}
	walk := func(exprs []Expr) *RowExprProblem {
		for _, child := range exprs {
			if p := RowExprSupported(child); p != nil {
				return p
			}
		}
		return nil
	}
	switch e := expr.(type) {
	case *Literal:
		switch e.Kind {
		case IntLit, FloatLit, BoolLit, StringLit, UnitLit:
			return nil
		}
	case *Identifier:
		return nil
	case *Tuple:
		return walk(e.Elements)
	case *List:
		return walk(e.Elements)
	case *Record:
		for _, field := range e.Fields {
			if p := RowExprSupported(field.Value); p != nil {
				return p
			}
		}
		return nil
	case *FuncCall:
		if p := RowExprSupported(e.Func); p != nil {
			return p
		}
		return walk(e.Args)
	case *UnaryOp:
		if e.Op == "-" {
			return RowExprSupported(e.Expr)
		}
	case *BinaryOp:
		return reject("TST002", fmt.Sprintf("operator %q in a tests row is not evaluated; write the value or move this case to a named test block", e.Op))
	}
	return reject("TST001", fmt.Sprintf("unsupported tests row expression %T; write the value or move this case to a named test block", expr))
}
