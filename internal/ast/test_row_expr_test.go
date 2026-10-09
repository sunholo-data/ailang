package ast

import "testing"

func TestRowExprSupported(t *testing.T) {
	lit := &Literal{Kind: IntLit, Value: int64(1)}
	for _, tc := range []struct {
		expr Expr
		code string
	}{
		{lit, ""}, {&Identifier{Name: "Allow"}, ""},
		{&Tuple{Elements: []Expr{lit}}, ""}, {&List{Elements: []Expr{lit}}, ""},
		{&FuncCall{Func: &Identifier{Name: "Deny"}, Args: []Expr{lit}}, ""},
		{&UnaryOp{Op: "-", Expr: lit}, ""},
		{&BinaryOp{Op: "+", Left: lit, Right: lit}, "TST002"},
		{&List{Elements: []Expr{&BinaryOp{Op: "==", Left: lit, Right: lit}}}, "TST002"},
		{&Lambda{}, "TST001"}, {&UnaryOp{Op: "!", Expr: lit}, "TST001"},
		{&Array{}, "TST001"}, {&FuncLit{}, "TST001"}, {&Let{}, "TST001"},
		{&LetRec{}, "TST001"}, {&Block{}, "TST001"}, {&If{}, "TST001"},
		{&Match{}, "TST001"}, {&RecordAccess{}, "TST001"}, {&RecordUpdate{}, "TST001"},
		{&Error{}, "TST001"}, {&QuasiQuote{}, "TST001"}, {&Send{}, "TST001"},
		{&Recv{}, "TST001"}, {&ForallExpr{}, "TST001"},
	} {
		problem := RowExprSupported(tc.expr)
		if tc.code == "" {
			if problem != nil {
				t.Errorf("%T: %v", tc.expr, problem)
			}
		} else if problem == nil || problem.Code != tc.code {
			t.Errorf("%T: got %v want %s", tc.expr, problem, tc.code)
		}
	}
}
