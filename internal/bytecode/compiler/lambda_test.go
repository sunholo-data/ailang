package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// --- Free-variable analysis (pure unit tests) ------------------------------

func TestFreeVars_NoFreeVars(t *testing.T) {
	lam := stmt.Lambda{
		Params: []stmt.Param{{Name: "x"}},
		Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 1}},
	}
	got := freeVarsLambda(lam)
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

func TestFreeVars_OneFreeVar(t *testing.T) {
	lam := stmt.Lambda{
		Params: []stmt.Param{{Name: "x"}},
		Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.VarRef{Name: "k"}},
	}
	got := freeVarsLambda(lam)
	if len(got) != 1 || got[0] != "k" {
		t.Errorf("got %v, want [k]", got)
	}
}

func TestFreeVars_OrderStable(t *testing.T) {
	lam := stmt.Lambda{
		Params: []stmt.Param{{Name: "x"}},
		Return: stmt.BinOp{
			Op:    stmt.OpAdd,
			Left:  stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "a"}, Right: stmt.VarRef{Name: "b"}},
			Right: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.VarRef{Name: "c"}},
		},
	}
	got := freeVarsLambda(lam)
	want := []string{"a", "b", "c"}
	if len(got) != 3 {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("position %d: got %q, want %q", i, got[i], n)
		}
	}
}

func TestFreeVars_LetShadowsCapture(t *testing.T) {
	// \x. let a = x in a + b
	// `a` is bound by the let → not free. `b` is free.
	lam := stmt.Lambda{
		Params: []stmt.Param{{Name: "x"}},
		Body: []stmt.Stmt{
			stmt.VarDecl{Name: "a", Value: stmt.VarRef{Name: "x"}},
		},
		Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "a"}, Right: stmt.VarRef{Name: "b"}},
	}
	got := freeVarsLambda(lam)
	if len(got) != 1 || got[0] != "b" {
		t.Errorf("got %v, want [b]", got)
	}
}

// --- Closure execution -----------------------------------------------------

func TestCompile_Lambda_NoCapture(t *testing.T) {
	// caller() = (\x. x * 2)(21) — but as a top-level call we need a callable.
	// Build it as: caller() = let f = \x. x*2 in f(21)
	body := []stmt.Stmt{
		stmt.VarDecl{
			Name: "f",
			Value: stmt.Lambda{
				Params: []stmt.Param{{Name: "x"}},
				Return: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}},
			},
		},
	}
	ret := stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.LitInt{Value: 21}}}
	got := runCompiled(t, nil, body, ret, nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}

func TestCompile_Lambda_OneCapture(t *testing.T) {
	// adder(k) = let f = \x. x + k in f(10)  → adder(5) = 15
	params := []stmt.Param{{Name: "k", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	body := []stmt.Stmt{
		stmt.VarDecl{
			Name: "f",
			Value: stmt.Lambda{
				Params: []stmt.Param{{Name: "x"}},
				Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.VarRef{Name: "k"}},
			},
		},
	}
	ret := stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.LitInt{Value: 10}}}
	got := runCompiled(t, params, body, ret, []bytecode.Value{bytecode.NewInt(5)})
	if got.Int != 15 {
		t.Errorf("got %d, want 15", got.Int)
	}
}

func TestCompile_Lambda_TwoCaptures(t *testing.T) {
	// f(a, b) = let g = \x. (x + a) * b in g(3)
	// f(2, 4) = (3 + 2) * 4 = 20
	params := []stmt.Param{
		{Name: "a", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
		{Name: "b", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
	}
	body := []stmt.Stmt{
		stmt.VarDecl{
			Name: "g",
			Value: stmt.Lambda{
				Params: []stmt.Param{{Name: "x"}},
				Return: stmt.BinOp{
					Op: stmt.OpMul,
					Left: stmt.BinOp{
						Op:    stmt.OpAdd,
						Left:  stmt.VarRef{Name: "x"},
						Right: stmt.VarRef{Name: "a"},
					},
					Right: stmt.VarRef{Name: "b"},
				},
			},
		},
	}
	ret := stmt.Call{Func: stmt.VarRef{Name: "g"}, Args: []stmt.Expr{stmt.LitInt{Value: 3}}}
	got := runCompiled(t, params, body, ret, []bytecode.Value{bytecode.NewInt(2), bytecode.NewInt(4)})
	if got.Int != 20 {
		t.Errorf("got %d, want 20", got.Int)
	}
}

// --- Higher-order: lambda passed to apply ---------------------------------

func TestCompile_Lambda_PassedToHigherOrder(t *testing.T) {
	// apply(f, x) = f(x)
	// caller(k) = apply(\y. y + k, 100)  → caller(7) = 107
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name: "apply",
				Params: []stmt.Param{
					{Name: "f", Type: stmt.FuncType{}},
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				},
				Return: stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.VarRef{Name: "x"}}},
			},
			{
				Name:   "caller",
				Params: []stmt.Param{{Name: "k", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "apply"},
					Args: []stmt.Expr{
						stmt.Lambda{
							Params: []stmt.Param{{Name: "y"}},
							Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "y"}, Right: stmt.VarRef{Name: "k"}},
						},
						stmt.LitInt{Value: 100},
					},
				},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", []bytecode.Value{bytecode.NewInt(7)})
	if got.Int != 107 {
		t.Errorf("got %d, want 107", got.Int)
	}
}

// --- Multi-module lambda resolution -----------------------------------------

// TestLambda_MultiModule_SameModuleCall verifies that a lambda body can call
// a function defined in the same module when module prefixes are used.
// This was broken: compileLambda did not propagate currentModule to the inner
// funcCompiler, so canonicalFuncName("", "helper") missed "mymod.helper".
func TestLambda_MultiModule_SameModuleCall(t *testing.T) {
	// mymod.helper(x) = x * 2
	// mymod.caller() = let f = \y. helper(y) in f(21)  → 42
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "mymod",
				Name:   "helper",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}},
			},
			{
				Module:   "mymod",
				Name:     "caller",
				Exported: true,
				Body: []stmt.Stmt{
					stmt.VarDecl{
						Name: "f",
						Value: stmt.Lambda{
							Params: []stmt.Param{{Name: "y"}},
							Return: stmt.Call{
								Func: stmt.VarRef{Name: "helper"},
								Args: []stmt.Expr{stmt.VarRef{Name: "y"}},
							},
						},
					},
				},
				Return: stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.LitInt{Value: 21}}},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Verify the lambda is NOT EvalOnly.
	for _, p := range img.Prototypes {
		if strings.Contains(p.Name, "lambda") && p.EvalOnly {
			t.Fatalf("lambda was bridged to evaluator: %s", p.EvalReason)
		}
	}
	got := runProgram(t, prog, "mymod.caller", nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}

// TestLambda_MultiModule_NestedLambda verifies that nested lambdas (lambda
// inside lambda) both inherit currentModule for name resolution.
func TestLambda_MultiModule_NestedLambda(t *testing.T) {
	// mymod.double(x) = x * 2
	// mymod.caller() = let f = \y. let g = \z. double(z) in g(y) in f(21) → 42
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "mymod",
				Name:   "double",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}},
			},
			{
				Module:   "mymod",
				Name:     "caller",
				Exported: true,
				Body: []stmt.Stmt{
					stmt.VarDecl{
						Name: "f",
						Value: stmt.Lambda{
							Params: []stmt.Param{{Name: "y"}},
							Body: []stmt.Stmt{
								stmt.VarDecl{
									Name: "g",
									Value: stmt.Lambda{
										Params: []stmt.Param{{Name: "z"}},
										Return: stmt.Call{
											Func: stmt.VarRef{Name: "double"},
											Args: []stmt.Expr{stmt.VarRef{Name: "z"}},
										},
									},
								},
							},
							Return: stmt.Call{Func: stmt.VarRef{Name: "g"}, Args: []stmt.Expr{stmt.VarRef{Name: "y"}}},
						},
					},
				},
				Return: stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.LitInt{Value: 21}}},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Verify NO lambda is EvalOnly.
	for _, p := range img.Prototypes {
		if strings.Contains(p.Name, "lambda") && p.EvalOnly {
			t.Fatalf("lambda was bridged: %s — reason: %s", p.Name, p.EvalReason)
		}
	}
	got := runProgram(t, prog, "mymod.caller", nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}

// TestLambda_MultiModule_VarRef verifies that a lambda in a multi-module
// image can reference a same-module function as a value (VarRef, not call).
func TestLambda_MultiModule_VarRef(t *testing.T) {
	// mymod.inc(x) = x + 1
	// mymod.apply(f, x) = f(x)
	// mymod.caller() = let mk = \y. apply(inc, y) in mk(41) → 42
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "mymod",
				Name:   "inc",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 1}},
			},
			{
				Module: "mymod",
				Name:   "apply",
				Params: []stmt.Param{
					{Name: "f", Type: stmt.FuncType{}},
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				},
				Return: stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{stmt.VarRef{Name: "x"}}},
			},
			{
				Module:   "mymod",
				Name:     "caller",
				Exported: true,
				Body: []stmt.Stmt{
					stmt.VarDecl{
						Name: "mk",
						Value: stmt.Lambda{
							Params: []stmt.Param{{Name: "y"}},
							Return: stmt.Call{
								Func: stmt.VarRef{Name: "apply"},
								Args: []stmt.Expr{
									stmt.VarRef{Name: "inc"},
									stmt.VarRef{Name: "y"},
								},
							},
						},
					},
				},
				Return: stmt.Call{Func: stmt.VarRef{Name: "mk"}, Args: []stmt.Expr{stmt.LitInt{Value: 41}}},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	for _, p := range img.Prototypes {
		if strings.Contains(p.Name, "lambda") && p.EvalOnly {
			t.Fatalf("lambda was bridged: %s — reason: %s", p.Name, p.EvalReason)
		}
	}
	got := runProgram(t, prog, "mymod.caller", nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}
