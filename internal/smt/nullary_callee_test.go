package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// M-SMT-NULLARY-CALLEE: a zero-arg function `f()` desugars to `f(_: ())` and is
// called with one unit literal. A nullary pure callee is a named constant and must
// encode as a nullary define-fun, not make its caller unencodable.

func unitLit() core.CoreExpr { return &core.Lit{Kind: core.UnitLit, Value: nil} }

func intLit(v int64) core.CoreExpr { return &core.Lit{Kind: core.IntLit, Value: v} }

func nullaryFuncDecl(name, ret string) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name:       name,
		Params:     []*ast.Param{{Name: "_", Type: &ast.SimpleType{Name: "()"}}},
		ReturnType: &ast.SimpleType{Name: ret},
	}
}

// callGlobal builds App(VarGlobal(test.name), args).
func callGlobal(name string, args ...core.CoreExpr) *core.App {
	return &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: name}}, Args: args}
}

func TestFirstUnencodableCalleeType_NullaryCalleeIsEncodable(t *testing.T) {
	// bound(x) = x + cap(())
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{&core.Var{Name: "x"}, callGlobal("cap", unitLit())},
	}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"bound": body, "cap": &core.Lambda{Params: []string{"_"}, Body: intLit(1000)}},
		map[string]*core.DeclMeta{"bound": pureMetaWithContracts(), "cap": pureMeta()},
	)
	astFuncs := map[string]*ast.FuncDecl{"cap": nullaryFuncDecl("cap", "int")}

	if callee, bad := FirstUnencodableCalleeType("bound", body, prog, nil, astFuncs, nil); callee != "" {
		t.Fatalf("nullary int callee rejected: %q uses %q", callee, bad)
	}

	// Control: a unit RETURN type is still unencodable — the gate must keep firing.
	astFuncs["cap"] = nullaryFuncDecl("cap", "()")
	if callee, _ := FirstUnencodableCalleeType("bound", body, prog, nil, astFuncs, nil); callee != "cap" {
		t.Fatalf("unit-returning callee not gated, got %q", callee)
	}
}

func TestSurfaceFunctionParams_DropsUnitParam(t *testing.T) {
	if got := SurfaceFunctionParams(nullaryFuncDecl("cap", "int")); len(got) != 0 {
		t.Fatalf("nullary function params = %+v, want none", got)
	}
	fd := &ast.FuncDecl{Name: "f", Params: []*ast.Param{{Name: "x", Type: &ast.SimpleType{Name: "int"}}}}
	got := SurfaceFunctionParams(fd)
	if len(got) != 1 || got[0].Name != "x" {
		t.Fatalf("unary function params = %+v, want [x]", got)
	}
}

func TestEncodeUserFunctionCall_NullaryIsBareConstant(t *testing.T) {
	got, err := encodeUserFunctionCall("cap", []core.CoreExpr{unitLit()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cap" {
		t.Fatalf("nullary call encoded as %q, want bare %q", got, "cap")
	}
	got, err = encodeUserFunctionCall("add", []core.CoreExpr{intLit(1), intLit(2)})
	if err != nil || got != "(add 1 2)" {
		t.Fatalf("binary call = %q, %v; want (add 1 2)", got, err)
	}
}

// TestResolveCalleesFromRoots_ChainedNullaryAndContractRoot: doubled() = cap() * 2
// is only called from an ensures predicate. Both callees must be defined, cap first,
// and doubled's body must apply cap as a constant (not a constructor over a unit arg).
func TestResolveCalleesFromRoots_ChainedNullaryAndContractRoot(t *testing.T) {
	capBody := &core.Lambda{Params: []string{"_"}, Body: intLit(1000)}
	doubledBody := &core.Lambda{Params: []string{"_"}, Body: &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "mul_Int"}},
		Args: []core.CoreExpr{callGlobal("cap", unitLit()), intLit(2)},
	}}
	fBody := &core.Var{Name: "x"}
	ensures := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "le_Int"}},
		Args: []core.CoreExpr{&core.Var{Name: "result"}, callGlobal("doubled", unitLit())},
	}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"f": fBody, "cap": capBody, "doubled": doubledBody},
		map[string]*core.DeclMeta{"f": pureMetaWithContracts(), "cap": pureMeta(), "doubled": pureMeta()},
	)
	returnSorts := map[string]string{"cap": "Int", "doubled": "Int"}

	// Body alone does not reach the callees.
	if defs, _ := ResolveCallees("f", fBody, prog, nil, returnSorts, nil); len(defs) != 0 {
		t.Fatalf("body-only resolution found %d callees, want 0", len(defs))
	}

	defs, err := ResolveCalleesFromRoots("f", []core.CoreExpr{fBody, ensures}, prog, nil, returnSorts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 2 || defs[0].Name != "cap" || defs[1].Name != "doubled" {
		t.Fatalf("defs = %+v, want [cap doubled]", defs)
	}
	if defs[0].SMTLib != "(define-fun cap () Int 1000)" {
		t.Fatalf("cap def = %q", defs[0].SMTLib)
	}
	if !strings.Contains(defs[1].SMTLib, "(* cap 2)") {
		t.Fatalf("doubled must apply cap as a constant, got %q", defs[1].SMTLib)
	}
	if activeResolvedCallees != nil {
		t.Fatal("ResolveCalleesFromRoots leaked activeResolvedCallees")
	}
}

// TestFirstUnencodableCalleeInFunc_GatesContractPredicates: contract predicates are
// callee roots now, so the sort gate must see a callee reached only from an ensures.
func TestFirstUnencodableCalleeInFunc_GatesContractPredicates(t *testing.T) {
	body := &core.Var{Name: "x"}
	prog := makeTestProgram(
		map[string]core.CoreExpr{
			"f":    body,
			"conv": &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
			"cap":  &core.Lambda{Params: []string{"_"}, Body: intLit(1000)},
		},
		map[string]*core.DeclMeta{"f": pureMetaWithContracts(), "conv": pureMeta(), "cap": pureMeta()},
	)
	astFuncs := map[string]*ast.FuncDecl{
		"conv": {
			Name:       "conv",
			Params:     []*ast.Param{{Name: "x", Type: &ast.SimpleType{Name: "float"}}},
			ReturnType: &ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "float"}}},
		},
		"cap": nullaryFuncDecl("cap", "int"),
	}
	ensures := func(callee string, arg core.CoreExpr) []*core.Contract {
		return []*core.Contract{
			nil, // tolerated
			{Kind: core.EnsuresKind, Expr: callGlobal(callee, arg)},
		}
	}

	if callee, _ := firstUnencodableCalleeInFunc("f", body, nil, prog, nil, astFuncs, nil); callee != "" {
		t.Fatalf("no contracts: got %q, want clean", callee)
	}
	if callee, bad := firstUnencodableCalleeInFunc("f", body, ensures("conv", &core.Var{Name: "result"}), prog, nil, astFuncs, nil); callee != "conv" || bad != "Option[float]" {
		t.Fatalf("Option callee in ensures: got (%q, %q), want (conv, Option[float])", callee, bad)
	}
	if callee, _ := firstUnencodableCalleeInFunc("f", body, ensures("cap", unitLit()), prog, nil, astFuncs, nil); callee != "" {
		t.Fatalf("nullary callee in ensures rejected: %q", callee)
	}
	// Body is still checked first.
	if callee, _ := firstUnencodableCalleeInFunc("f", callGlobal("conv", body), nil, prog, nil, astFuncs, nil); callee != "conv" {
		t.Fatalf("Option callee in body: got %q, want conv", callee)
	}
}
