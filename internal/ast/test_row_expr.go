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

// Inline test rows are run by the test harness (internal/testing) and
// type-checked by the compiler (internal/elaborate, #2232). Both build the
// call of the function under test from a row with these two functions, so a
// row the harness runs is exactly the row the checker types.

// RowInput is the input of an inline test row as the harness stores it: the
// single input itself (`(x, r)`, and also `(((a, b)), r)`, whose one input is
// a tuple), or all the inputs as one tuple (`((a, b), r)`).
func RowInput(row *TestCase) Expr {
	if len(row.Inputs) == 1 {
		return row.Inputs[0]
	}
	return &Tuple{Elements: row.Inputs, Pos: row.Pos}
}

// RowCallArgs is the argument list the function under test is applied to for
// a row input: the elements of a tuple of two or more (so `((a, b), r)` and
// `(((a, b)), r)` both call f(a, b)), else the input as the one argument.
func RowCallArgs(input Expr) []Expr {
	if t, ok := input.(*Tuple); ok && len(t.Elements) > 1 {
		return t.Elements
	}
	return []Expr{input}
}
