package compiler

import (
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// --- IfExpr -----------------------------------------------------------------

func TestCompile_IfExpr_Basic(t *testing.T) {
	// if true then 1 else 2 → 1
	ret := stmt.IfExpr{
		Cond: stmt.LitBool{Value: true},
		Then: stmt.LitInt{Value: 1},
		Else: stmt.LitInt{Value: 2},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Int != 1 {
		t.Errorf("got %d, want 1", got.Int)
	}
	ret2 := stmt.IfExpr{
		Cond: stmt.LitBool{Value: false},
		Then: stmt.LitInt{Value: 1},
		Else: stmt.LitInt{Value: 2},
	}
	got2 := runCompiled(t, nil, nil, ret2, nil)
	if got2.Int != 2 {
		t.Errorf("got %d, want 2", got2.Int)
	}
}

func TestCompile_IfExpr_Abs(t *testing.T) {
	// abs(x) = if x < 0 then 0 - x else x
	params := []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	ret := stmt.IfExpr{
		Cond: stmt.BinOp{Op: stmt.OpLt, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 0}},
		Then: stmt.BinOp{Op: stmt.OpSub, Left: stmt.LitInt{Value: 0}, Right: stmt.VarRef{Name: "x"}},
		Else: stmt.VarRef{Name: "x"},
	}
	for _, tc := range []struct{ in, want int64 }{
		{5, 5}, {-5, 5}, {0, 0}, {-100, 100},
	} {
		got := runCompiled(t, params, nil, ret, []bytecode.Value{bytecode.NewInt(tc.in)})
		if got.Int != tc.want {
			t.Errorf("abs(%d): got %d, want %d", tc.in, got.Int, tc.want)
		}
	}
}

func TestCompile_IfExpr_Nested_Classify(t *testing.T) {
	// classify(n) = if n < 0 then "negative" else if n == 0 then "zero" else "positive"
	params := []stmt.Param{{Name: "n", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	ret := stmt.IfExpr{
		Cond: stmt.BinOp{Op: stmt.OpLt, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 0}},
		Then: stmt.LitString{Value: "negative"},
		Else: stmt.IfExpr{
			Cond: stmt.BinOp{Op: stmt.OpEq, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 0}},
			Then: stmt.LitString{Value: "zero"},
			Else: stmt.LitString{Value: "positive"},
		},
	}
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{-3, "negative"}, {0, "zero"}, {7, "positive"},
	} {
		got := runCompiled(t, params, nil, ret, []bytecode.Value{bytecode.NewInt(tc.in)})
		if got.AsString() != tc.want {
			t.Errorf("classify(%d): got %q, want %q", tc.in, got.AsString(), tc.want)
		}
	}
}

// --- Comparison rewrites ----------------------------------------------------

func TestCompile_Comparisons_AllOps(t *testing.T) {
	cases := []struct {
		op   stmt.BinOpKind
		a, b int64
		want bool
	}{
		{stmt.OpEq, 5, 5, true},
		{stmt.OpEq, 5, 6, false},
		{stmt.OpNeq, 5, 6, true},
		{stmt.OpNeq, 5, 5, false},
		{stmt.OpLt, 5, 6, true},
		{stmt.OpLt, 6, 5, false},
		{stmt.OpLte, 5, 5, true},
		{stmt.OpLte, 6, 5, false},
		{stmt.OpGt, 6, 5, true},
		{stmt.OpGt, 5, 5, false},
		{stmt.OpGte, 5, 5, true},
		{stmt.OpGte, 4, 5, false},
	}
	for _, tc := range cases {
		ret := stmt.BinOp{Op: tc.op, Left: stmt.LitInt{Value: tc.a}, Right: stmt.LitInt{Value: tc.b}}
		got := runCompiled(t, nil, nil, ret, nil)
		if got.Tag != bytecode.TagBool || got.Bool != tc.want {
			t.Errorf("%v(%d,%d): got %v, want %v", tc.op, tc.a, tc.b, got, tc.want)
		}
	}
}

// --- Short-circuit ----------------------------------------------------------

func TestCompile_And_BothBranches(t *testing.T) {
	cases := []struct {
		a, b bool
		want bool
	}{
		{true, true, true},
		{true, false, false},
		{false, true, false},
		{false, false, false},
	}
	for _, tc := range cases {
		ret := stmt.BinOp{
			Op:    stmt.OpAnd,
			Left:  stmt.LitBool{Value: tc.a},
			Right: stmt.LitBool{Value: tc.b},
		}
		got := runCompiled(t, nil, nil, ret, nil)
		if got.Bool != tc.want {
			t.Errorf("%v && %v: got %v, want %v", tc.a, tc.b, got.Bool, tc.want)
		}
	}
}

func TestCompile_Or_BothBranches(t *testing.T) {
	cases := []struct {
		a, b bool
		want bool
	}{
		{true, true, true},
		{true, false, true},
		{false, true, true},
		{false, false, false},
	}
	for _, tc := range cases {
		ret := stmt.BinOp{
			Op:    stmt.OpOr,
			Left:  stmt.LitBool{Value: tc.a},
			Right: stmt.LitBool{Value: tc.b},
		}
		got := runCompiled(t, nil, nil, ret, nil)
		if got.Bool != tc.want {
			t.Errorf("%v || %v: got %v, want %v", tc.a, tc.b, got.Bool, tc.want)
		}
	}
}

func TestCompile_And_ShortCircuit_DoesNotEvalRhs(t *testing.T) {
	// false && (1/0) — must not divide. We use a literal `1/0` BinOp; if RHS
	// were evaluated, the VM would surface a divide-by-zero error.
	ret := stmt.BinOp{
		Op:   stmt.OpAnd,
		Left: stmt.LitBool{Value: false},
		Right: stmt.BinOp{
			Op:    stmt.OpDiv,
			Left:  stmt.LitInt{Value: 1},
			Right: stmt.LitInt{Value: 0},
		},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Bool != false {
		t.Errorf("got %v, want false", got.Bool)
	}
}

func TestCompile_Or_ShortCircuit_DoesNotEvalRhs(t *testing.T) {
	// true || (1/0) — must not divide.
	ret := stmt.BinOp{
		Op:   stmt.OpOr,
		Left: stmt.LitBool{Value: true},
		Right: stmt.BinOp{
			Op:    stmt.OpDiv,
			Left:  stmt.LitInt{Value: 1},
			Right: stmt.LitInt{Value: 0},
		},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Bool != true {
		t.Errorf("got %v, want true", got.Bool)
	}
}

// --- Not --------------------------------------------------------------------

func TestCompile_UnOp_Not(t *testing.T) {
	ret := stmt.UnOp{Op: stmt.OpNot, Operand: stmt.LitBool{Value: true}}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Bool != false {
		t.Errorf("got %v, want false", got.Bool)
	}
}

// --- Concat -----------------------------------------------------------------

func TestCompile_BinOp_Concat(t *testing.T) {
	ret := stmt.BinOp{
		Op:    stmt.OpConcat,
		Left:  stmt.LitString{Value: "Hello, "},
		Right: stmt.LitString{Value: "world"},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.AsString() != "Hello, world" {
		t.Errorf("got %q, want %q", got.AsString(), "Hello, world")
	}
}

// --- Clamp (3-way nested if as IfExpr) --------------------------------------

func TestCompile_Clamp(t *testing.T) {
	// clamp(x, lo, hi) = if x < lo then lo else if x > hi then hi else x
	params := []stmt.Param{
		{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
		{Name: "lo", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
		{Name: "hi", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
	}
	ret := stmt.IfExpr{
		Cond: stmt.BinOp{Op: stmt.OpLt, Left: stmt.VarRef{Name: "x"}, Right: stmt.VarRef{Name: "lo"}},
		Then: stmt.VarRef{Name: "lo"},
		Else: stmt.IfExpr{
			Cond: stmt.BinOp{Op: stmt.OpGt, Left: stmt.VarRef{Name: "x"}, Right: stmt.VarRef{Name: "hi"}},
			Then: stmt.VarRef{Name: "hi"},
			Else: stmt.VarRef{Name: "x"},
		},
	}
	cases := []struct{ x, lo, hi, want int64 }{
		{5, 0, 10, 5},
		{-3, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tc := range cases {
		got := runCompiled(t, params, nil, ret, []bytecode.Value{
			bytecode.NewInt(tc.x), bytecode.NewInt(tc.lo), bytecode.NewInt(tc.hi),
		})
		if got.Int != tc.want {
			t.Errorf("clamp(%d,%d,%d): got %d, want %d", tc.x, tc.lo, tc.hi, got.Int, tc.want)
		}
	}
}

// --- IfStmt (statement form) ------------------------------------------------

func TestCompile_IfStmt_Assignment(t *testing.T) {
	// let result = 0; if x < 0 then result = -x; else result = x; return result
	params := []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	body := []stmt.Stmt{
		stmt.VarDecl{Name: "result", Value: stmt.LitInt{Value: 0}},
		stmt.IfStmt{
			Cond: stmt.BinOp{Op: stmt.OpLt, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 0}},
			Then: []stmt.Stmt{
				stmt.AssignStmt{Name: "result", Value: stmt.BinOp{Op: stmt.OpSub, Left: stmt.LitInt{Value: 0}, Right: stmt.VarRef{Name: "x"}}},
			},
			Else: []stmt.Stmt{
				stmt.AssignStmt{Name: "result", Value: stmt.VarRef{Name: "x"}},
			},
		},
	}
	ret := stmt.VarRef{Name: "result"}
	for _, tc := range []struct{ in, want int64 }{
		{5, 5}, {-7, 7}, {0, 0},
	} {
		got := runCompiled(t, params, body, ret, []bytecode.Value{bytecode.NewInt(tc.in)})
		if got.Int != tc.want {
			t.Errorf("abs(%d): got %d, want %d", tc.in, got.Int, tc.want)
		}
	}
}
