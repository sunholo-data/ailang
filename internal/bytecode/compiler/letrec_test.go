package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// M-VM-COVERAGE G1: local letrec groups compile natively, and a group name
// never resolves to a top-level function of the same name.

func intParam(name string) stmt.Param {
	return stmt.Param{Name: name, Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}
}

func v(name string) stmt.VarRef { return stmt.VarRef{Name: name} }
func lit(n int64) stmt.LitInt   { return stmt.LitInt{Value: n} }
func bin(op stmt.BinOpKind, l, r stmt.Expr) stmt.BinOp {
	return stmt.BinOp{Op: op, Left: l, Right: r}
}
func call(f string, args ...stmt.Expr) stmt.Call {
	return stmt.Call{Func: v(f), Args: args}
}

// constFunc is a top-level `func name(params) -> int = k`: the decoy a
// miscompiled letrec name would bind to.
func constFunc(name string, k int64, params ...string) stmt.FuncDecl {
	ps := make([]stmt.Param, len(params))
	for i, p := range params {
		ps[i] = intParam(p)
	}
	return stmt.FuncDecl{Name: name, Params: ps, Return: lit(k)}
}

func entryProto(t *testing.T, img *bytecode.BytecodeImage, name string) *bytecode.FuncPrototype {
	t.Helper()
	for _, p := range img.Prototypes {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("prototype %q not found", name)
	return nil
}

// sumTo(n) { letrec go = \i acc. if i > n then acc else go(i+1, acc+i); go(1, 0) }
// with a top-level `go(a, b) = 1000` in the same image. The evaluator says
// sumTo(4) = 10; before the fix the VM called the top-level go and said 1000.
func TestLetRec_SelfReferenceShadowsTopLevel(t *testing.T) {
	group := []string{"go"}
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		constFunc("go", 1000, "a", "b"),
		{
			Name:   "sumTo",
			Params: []stmt.Param{intParam("n")},
			Body: []stmt.Stmt{stmt.VarDecl{
				Name:     "go",
				RecGroup: group,
				Value: stmt.Lambda{
					Params: []stmt.Param{intParam("i"), intParam("acc")},
					Return: stmt.IfExpr{
						Cond: bin(stmt.OpGt, v("i"), v("n")),
						Then: v("acc"),
						Else: call("go", bin(stmt.OpAdd, v("i"), lit(1)), bin(stmt.OpAdd, v("acc"), v("i"))),
					},
				},
			}},
			Return:   call("go", lit(1), lit(0)),
			Exported: true,
		},
	}}
	got := runProgram(t, prog, "sumTo", []bytecode.Value{bytecode.NewInt(4)})
	if got.Int != 10 {
		t.Fatalf("sumTo(4) = %d, want 10 (1000 means the letrec name bound to the top-level go)", got.Int)
	}
}

// A two-member group (mutual recursion). Source syntax cannot express one
// today, but the lowering carries a whole core.LetRec group, so the
// compiler must handle it. Both names also exist at top level as decoys.
func TestLetRec_MutualGroup(t *testing.T) {
	group := []string{"isEven", "isOdd"}
	member := func(name, other string, base bool) stmt.VarDecl {
		return stmt.VarDecl{
			Name:     name,
			RecGroup: group,
			Value: stmt.Lambda{
				Params: []stmt.Param{intParam("k")},
				Return: stmt.IfExpr{
					Cond: bin(stmt.OpEq, v("k"), lit(0)),
					Then: stmt.LitBool{Value: base},
					Else: call(other, bin(stmt.OpSub, v("k"), lit(1))),
				},
			},
		}
	}
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		{Name: "isEven", Params: []stmt.Param{intParam("n")}, Return: stmt.LitBool{Value: false}},
		{Name: "isOdd", Params: []stmt.Param{intParam("n")}, Return: stmt.LitBool{Value: true}},
		{
			Name:     "parity",
			Params:   []stmt.Param{intParam("n")},
			Body:     []stmt.Stmt{member("isEven", "isOdd", true), member("isOdd", "isEven", false)},
			Return:   call("isEven", v("n")),
			Exported: true,
		},
	}}
	for n, want := range map[int64]bool{0: true, 7: false, 10: true} {
		if got := runProgram(t, prog, "parity", []bytecode.Value{bytecode.NewInt(n)}); got.Bool != want {
			t.Errorf("parity(%d) = %v, want %v", n, got.Bool, want)
		}
	}
}

// A body-local `let` that shadows a captured name must not leak into the
// closure rebuilt for the self call: captures are read by register.
//
//	f(n) { let k = 10;
//	       letrec go = \i acc. if i <= 0 then acc + k else { let k = 999; go(i-1, acc+1) };
//	       go(n, 0) }
func TestLetRec_RebuiltClosureIgnoresShadowedCapture(t *testing.T) {
	group := []string{"go"}
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{{
		Name:   "f",
		Params: []stmt.Param{intParam("n")},
		Body: []stmt.Stmt{
			stmt.VarDecl{Name: "k", Value: lit(10)},
			stmt.VarDecl{
				Name:     "go",
				RecGroup: group,
				Value: stmt.Lambda{
					Params: []stmt.Param{intParam("i"), intParam("acc")},
					Body: []stmt.Stmt{stmt.IfStmt{
						Cond: bin(stmt.OpLte, v("i"), lit(0)),
						Then: []stmt.Stmt{stmt.ReturnStmt{Value: bin(stmt.OpAdd, v("acc"), v("k"))}},
						Else: []stmt.Stmt{
							stmt.VarDecl{Name: "k", Value: lit(999)},
							stmt.ReturnStmt{Value: call("go", bin(stmt.OpSub, v("i"), lit(1)), bin(stmt.OpAdd, v("acc"), lit(1)))},
						},
					}},
				},
			},
		},
		Return:   call("go", v("n"), lit(0)),
		Exported: true,
	}}}
	if got := runProgram(t, prog, "f", []bytecode.Value{bytecode.NewInt(3)}); got.Int != 13 {
		t.Fatalf("f(3) = %d, want 13", got.Int)
	}
}

// A nested lambda inside a member body that names the member must capture
// the member, not the top-level decoy.
//
//	f(n) { letrec go = \i. if i <= 0 then 0 else (\j. go(j))(i - 1) + 1; go(n) }
func TestLetRec_NestedLambdaCapturesMember(t *testing.T) {
	group := []string{"go"}
	inner := stmt.Lambda{Params: []stmt.Param{intParam("j")}, Return: call("go", v("j"))}
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		constFunc("go", 1000, "a"),
		{
			Name:   "f",
			Params: []stmt.Param{intParam("n")},
			Body: []stmt.Stmt{stmt.VarDecl{
				Name:     "go",
				RecGroup: group,
				Value: stmt.Lambda{
					Params: []stmt.Param{intParam("i")},
					Return: stmt.IfExpr{
						Cond: bin(stmt.OpLte, v("i"), lit(0)),
						Then: lit(0),
						Else: bin(stmt.OpAdd, stmt.Call{Func: inner, Args: []stmt.Expr{bin(stmt.OpSub, v("i"), lit(1))}}, lit(1)),
					},
				},
			}},
			Return:   call("go", v("n")),
			Exported: true,
		},
	}}
	if got := runProgram(t, prog, "f", []bytecode.Value{bytecode.NewInt(5)}); got.Int != 5 {
		t.Fatalf("f(5) = %d, want 5 (1000+ means the nested lambda bound the top-level go)", got.Int)
	}
}

// A self tail call goes through OpTailCall on the rebuilt closure, so a long
// loop runs in constant VM stack.
func TestLetRec_TailLoopConstantStack(t *testing.T) {
	group := []string{"loop"}
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{{
		Name:   "f",
		Params: []stmt.Param{intParam("n")},
		Body: []stmt.Stmt{
			stmt.VarDecl{Name: "step", Value: lit(3)},
			stmt.VarDecl{
				Name:     "loop",
				RecGroup: group,
				Value: stmt.Lambda{
					Params: []stmt.Param{intParam("i"), intParam("acc")},
					Return: stmt.IfExpr{
						Cond: bin(stmt.OpGte, v("i"), v("n")),
						Then: v("acc"),
						Else: call("loop", bin(stmt.OpAdd, v("i"), lit(1)), bin(stmt.OpAdd, v("acc"), v("step"))),
					},
				},
			},
		},
		Return:   call("loop", lit(0), lit(0)),
		Exported: true,
	}}}
	if got := runProgram(t, prog, "f", []bytecode.Value{bytecode.NewInt(1_000_000)}); got.Int != 3_000_000 {
		t.Fatalf("f(1e6) = %d, want 3000000", got.Int)
	}
}

// A letrec binding that is not a lambda cannot be rebuilt; the enclosing
// function must fall back (EvalOnly) rather than guess.
func TestLetRec_NonLambdaRefuses(t *testing.T) {
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{{
		Name:     "f",
		Body:     []stmt.Stmt{stmt.VarDecl{Name: "x", RecGroup: []string{"x"}, Value: lit(1)}},
		Return:   v("x"),
		Exported: true,
	}}}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	p := entryProto(t, img, "f")
	if !p.EvalOnly || !strings.Contains(p.EvalReason, "not a lambda") {
		t.Fatalf("EvalOnly=%v reason=%q, want EvalOnly naming the non-lambda binding", p.EvalOnly, p.EvalReason)
	}
}
