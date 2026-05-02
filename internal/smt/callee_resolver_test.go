package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Helper to create a simple Core program with functions for testing.
func makeTestProgram(funcs map[string]core.CoreExpr, metas map[string]*core.DeclMeta) *core.Program {
	prog := &core.Program{
		Meta: metas,
	}
	for name, body := range funcs {
		prog.Decls = append(prog.Decls, &core.LetRec{
			Bindings: []core.RecBinding{{Name: name, Value: body}},
			Body:     &core.Lit{Kind: core.IntLit, Value: int64(0)},
		})
	}
	return prog
}

func pureMeta() *core.DeclMeta {
	return &core.DeclMeta{IsPure: true}
}

func pureMetaWithContracts() *core.DeclMeta {
	return &core.DeclMeta{
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
}

func TestCollectCalleeCalls_NoCallees(t *testing.T) {
	// Function with no user-defined calls: just arithmetic
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"f": body},
		map[string]*core.DeclMeta{"f": pureMeta()},
	)

	calls := collectCalleeCalls(body, "f", prog, nil, 0)
	if len(calls) != 0 {
		t.Errorf("expected 0 callees, got %d: %v", len(calls), calls)
	}
}

func TestCollectCalleeCalls_DirectCall(t *testing.T) {
	// helper() is a function in the program
	helperBody := &core.Lit{Kind: core.IntLit, Value: int64(10)}

	// f calls helper: App(VarGlobal(helper), [arg])
	fBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "helper"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(5)}},
	}

	prog := makeTestProgram(
		map[string]core.CoreExpr{
			"f":      fBody,
			"helper": helperBody,
		},
		map[string]*core.DeclMeta{
			"f":      pureMeta(),
			"helper": pureMeta(),
		},
	)

	calls := collectCalleeCalls(fBody, "f", prog, nil, 0)
	if len(calls) != 1 {
		t.Fatalf("expected 1 callee, got %d: %v", len(calls), calls)
	}
	if calls[0] != "helper" {
		t.Errorf("expected callee 'helper', got %q", calls[0])
	}
}

func TestCollectCalleeCalls_TransitiveCall(t *testing.T) {
	// c() is a leaf function
	cBody := &core.Lit{Kind: core.IntLit, Value: int64(1)}

	// b() calls c()
	bBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "c"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(2)}},
	}

	// a() calls b()
	aBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "b"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(3)}},
	}

	prog := makeTestProgram(
		map[string]core.CoreExpr{"a": aBody, "b": bBody, "c": cBody},
		map[string]*core.DeclMeta{"a": pureMeta(), "b": pureMeta(), "c": pureMeta()},
	)

	calls := collectCalleeCalls(aBody, "a", prog, nil, 0)
	if len(calls) != 2 {
		t.Fatalf("expected 2 callees (b, c), got %d: %v", len(calls), calls)
	}
	// Both b and c should be collected
	found := make(map[string]bool)
	for _, c := range calls {
		found[c] = true
	}
	if !found["b"] || !found["c"] {
		t.Errorf("expected callees b and c, got %v", calls)
	}
}

func TestCollectCalleeCalls_IgnoresBuiltins(t *testing.T) {
	// Call to a builtin should NOT be collected
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: int64(1)},
			&core.Lit{Kind: core.IntLit, Value: int64(2)},
		},
	}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"f": body},
		map[string]*core.DeclMeta{"f": pureMeta()},
	)

	calls := collectCalleeCalls(body, "f", prog, nil, 0)
	if len(calls) != 0 {
		t.Errorf("expected 0 callees for builtin calls, got %d: %v", len(calls), calls)
	}
}

func TestTopoSort_SimpleChain(t *testing.T) {
	// c() is a leaf
	cBody := &core.Lit{Kind: core.IntLit, Value: int64(1)}

	// b() calls c()
	bBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "c"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(2)}},
	}

	prog := makeTestProgram(
		map[string]core.CoreExpr{"b": bBody, "c": cBody},
		map[string]*core.DeclMeta{"b": pureMeta(), "c": pureMeta()},
	)

	order, err := topoSort([]string{"b", "c"}, "a", prog, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// c should come before b (c is a dependency of b)
	cIdx, bIdx := -1, -1
	for i, name := range order {
		if name == "c" {
			cIdx = i
		}
		if name == "b" {
			bIdx = i
		}
	}
	if cIdx == -1 || bIdx == -1 {
		t.Fatalf("expected both b and c in order, got %v", order)
	}
	if cIdx > bIdx {
		t.Errorf("expected c before b, but c at %d, b at %d (order: %v)", cIdx, bIdx, order)
	}
}

func TestTopoSort_CircularCall(t *testing.T) {
	// a() calls b(), b() calls a() — circular
	aBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "b"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	bBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "a"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(2)}},
	}

	prog := makeTestProgram(
		map[string]core.CoreExpr{"a": aBody, "b": bBody},
		map[string]*core.DeclMeta{"a": pureMeta(), "b": pureMeta()},
	)

	_, err := topoSort([]string{"a", "b"}, "root", prog, nil)
	if err == nil {
		t.Fatal("expected circular call error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error message, got: %v", err)
	}
}

func TestResolveCallees_EmptyBody(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"f": body},
		map[string]*core.DeclMeta{"f": pureMeta()},
	)

	defs, err := ResolveCallees("f", body, prog, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
}

func TestResolveCallees_DirectCall(t *testing.T) {
	// helper(x) = x + 1
	helperBody := &core.Lambda{
		Params: []string{"x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
		},
	}

	// f(y) = helper(y)
	fBody := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "helper"}},
		Args: []core.CoreExpr{&core.Var{Name: "y"}},
	}

	prog := makeTestProgram(
		map[string]core.CoreExpr{"f": fBody, "helper": helperBody},
		map[string]*core.DeclMeta{"f": pureMetaWithContracts(), "helper": pureMeta()},
	)

	surfaceParams := map[string][]FunctionParam{
		"helper": {{Name: "x", Type: &types.TCon{Name: "int"}}},
	}
	surfaceReturnSorts := map[string]string{
		"helper": "Int",
	}

	defs, err := ResolveCallees("f", fBody, prog, surfaceParams, surfaceReturnSorts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 callee def, got %d", len(defs))
	}
	if defs[0].Name != "helper" {
		t.Errorf("expected callee 'helper', got %q", defs[0].Name)
	}
	if !strings.Contains(defs[0].SMTLib, "define-fun helper") {
		t.Errorf("expected define-fun helper in SMT-LIB, got: %s", defs[0].SMTLib)
	}
}

func TestResolveCallees_NilProgram(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	defs, err := ResolveCallees("f", body, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defs != nil {
		t.Errorf("expected nil defs for nil program, got %v", defs)
	}
}

func TestBuildDefineFun(t *testing.T) {
	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
		{Name: "y", Type: &types.TCon{Name: "int"}},
	}
	result := buildDefineFun("add", params, "Int", "(+ x y)")
	expected := "(define-fun add ((x Int) (y Int)) Int (+ x y))"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildDefineFun_WithEnum(t *testing.T) {
	params := []FunctionParam{
		{Name: "income", Type: &types.TCon{Name: "int"}},
		{Name: "bracket", Type: &types.TCon{Name: "TaxBracket"}},
	}
	result := buildDefineFun("calculateTax", params, "Int",
		"(match bracket ((EXEMPT 0) (REDUCED (div income 10)) (STANDARD (div income 5))))")
	if !strings.Contains(result, "define-fun calculateTax") {
		t.Errorf("expected define-fun calculateTax, got: %s", result)
	}
	if !strings.Contains(result, "(income Int)") {
		t.Errorf("expected (income Int), got: %s", result)
	}
	if !strings.Contains(result, "(bracket TaxBracket)") {
		t.Errorf("expected (bracket TaxBracket), got: %s", result)
	}
}

func TestIsUserDefinedCall_Builtin(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	prog := makeTestProgram(nil, nil)
	_, ok := IsUserDefinedCall(app, prog)
	if ok {
		t.Error("builtin call should not be user-defined")
	}
}

func TestIsUserDefinedCall_UserDefined(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "helper"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	prog := makeTestProgram(
		map[string]core.CoreExpr{"helper": &core.Lit{Kind: core.IntLit, Value: int64(0)}},
		map[string]*core.DeclMeta{"helper": pureMeta()},
	)
	name, ok := IsUserDefinedCall(app, prog)
	if !ok {
		t.Error("expected user-defined call to be detected")
	}
	if name != "helper" {
		t.Errorf("expected name 'helper', got %q", name)
	}
}

func TestIsSMTEncodableForCallee_Pure(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	meta := pureMeta()
	ok, reasons := IsSMTEncodableForCallee("f", meta, body)
	if !ok {
		t.Errorf("expected pure function to be encodable, got rejections: %v", reasons)
	}
}

func TestIsSMTEncodableForCallee_Impure(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	meta := &core.DeclMeta{IsPure: false}
	ok, _ := IsSMTEncodableForCallee("f", meta, body)
	if ok {
		t.Error("expected impure function to NOT be encodable")
	}
}

func TestIsSMTEncodableForCallee_Recursive(t *testing.T) {
	// f(x) = f(x-1)
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "f"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	meta := pureMeta()
	ok, _ := IsSMTEncodableForCallee("f", meta, body)
	if ok {
		t.Error("expected recursive function to NOT be encodable")
	}
}
