package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// --- IsInlinableHOF tests ---

func TestIsInlinableHOF_MapWithLambda(t *testing.T) {
	// map(\x -> x + 1, xs) → detectable as inlinable HOF
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 1 {
		t.Fatalf("expected 1 inlinable HOF call, got %d", len(calls))
	}
	if calls[0].Kind != HOFMap {
		t.Errorf("expected HOFMap, got %v", calls[0].Kind)
	}
	if calls[0].Lambda == nil {
		t.Error("expected non-nil Lambda")
	}
	if calls[0].ListArg == nil {
		t.Error("expected non-nil ListArg")
	}
}

func TestIsInlinableHOF_MapWithVar(t *testing.T) {
	// map(f, xs) where f is a variable → NOT inlinable
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "f"},
			&core.Var{Name: "xs"},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 0 {
		t.Errorf("expected 0 inlinable HOF calls for variable function arg, got %d", len(calls))
	}
}

func TestIsInlinableHOF_FilterWithLambda(t *testing.T) {
	// filter(\x -> x > 0, xs)
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "filter_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "gt_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 1 {
		t.Fatalf("expected 1 inlinable HOF call, got %d", len(calls))
	}
	if calls[0].Kind != HOFFilter {
		t.Errorf("expected HOFFilter, got %v", calls[0].Kind)
	}
}

func TestIsInlinableHOF_FoldlWithLambda(t *testing.T) {
	// foldl(\acc x -> acc + x, 0, xs)
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "foldl_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"acc", "x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "acc"},
						&core.Var{Name: "x"},
					},
				},
			},
			&core.Lit{Kind: core.IntLit, Value: int64(0)},
			&core.Var{Name: "xs"},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 1 {
		t.Fatalf("expected 1 inlinable HOF call, got %d", len(calls))
	}
	if calls[0].Kind != HOFFoldl {
		t.Errorf("expected HOFFoldl, got %v", calls[0].Kind)
	}
	if calls[0].InitVal == nil {
		t.Error("expected non-nil InitVal for foldl")
	}
}

func TestIsInlinableHOF_StdlistMap(t *testing.T) {
	// std/list.map(\x -> x + 1, xs) — via stdlib module
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "map"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 1 {
		t.Fatalf("expected 1 inlinable HOF call via std/list, got %d", len(calls))
	}
	if calls[0].Kind != HOFMap {
		t.Errorf("expected HOFMap, got %v", calls[0].Kind)
	}
}

func TestIsInlinableHOF_NestedInLet(t *testing.T) {
	// let y = map(\x -> x + 1, xs) in y
	body := &core.Let{
		Name: "y",
		Value: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
			Args: []core.CoreExpr{
				&core.Lambda{
					Params: []string{"x"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
						Args: []core.CoreExpr{
							&core.Var{Name: "x"},
							&core.Lit{Kind: core.IntLit, Value: int64(1)},
						},
					},
				},
				&core.Var{Name: "xs"},
			},
		},
		Body: &core.Var{Name: "y"},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 1 {
		t.Fatalf("expected 1 inlinable HOF call nested in let, got %d", len(calls))
	}
}

func TestIsInlinableHOF_NoHOF(t *testing.T) {
	// Simple arithmetic, no HOF
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Lit{Kind: core.IntLit, Value: int64(1)},
		},
	}

	calls := IsInlinableHOF(body)
	if len(calls) != 0 {
		t.Errorf("expected 0 inlinable HOF calls for arithmetic, got %d", len(calls))
	}
}

// --- SubstituteLambdaVar tests ---

func TestSubstituteLambdaVar_SimpleVar(t *testing.T) {
	// Replace "x" with Var("h") in Var("x") → Var("h")
	body := &core.Var{Name: "x"}
	replacement := &core.Var{Name: "h"}

	result := SubstituteLambdaVar(body, "x", replacement)
	v, ok := result.(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var, got %T", result)
	}
	if v.Name != "h" {
		t.Errorf("expected h, got %s", v.Name)
	}
}

func TestSubstituteLambdaVar_NoMatch(t *testing.T) {
	body := &core.Var{Name: "y"}
	replacement := &core.Var{Name: "h"}

	result := SubstituteLambdaVar(body, "x", replacement)
	v, ok := result.(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var, got %T", result)
	}
	if v.Name != "y" {
		t.Errorf("expected y, got %s", v.Name)
	}
}

func TestSubstituteLambdaVar_InBinOp(t *testing.T) {
	// x + 1 with x → h
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Lit{Kind: core.IntLit, Value: int64(1)},
		},
	}
	replacement := &core.Var{Name: "h"}

	result := SubstituteLambdaVar(body, "x", replacement)
	app, ok := result.(*core.App)
	if !ok {
		t.Fatalf("expected *core.App, got %T", result)
	}
	arg0, ok := app.Args[0].(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var for arg0, got %T", app.Args[0])
	}
	if arg0.Name != "h" {
		t.Errorf("expected h, got %s", arg0.Name)
	}
}

func TestSubstituteLambdaVar_ShadowedByLambda(t *testing.T) {
	// \x -> x + 1: x is shadowed in the lambda
	body := &core.Lambda{
		Params: []string{"x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
		},
	}
	replacement := &core.Var{Name: "h"}

	result := SubstituteLambdaVar(body, "x", replacement)
	lam, ok := result.(*core.Lambda)
	if !ok {
		t.Fatalf("expected *core.Lambda, got %T", result)
	}
	// The inner x should NOT be replaced (shadowed by lambda param)
	innerApp, ok := lam.Body.(*core.App)
	if !ok {
		t.Fatalf("expected *core.App in lambda body, got %T", lam.Body)
	}
	innerVar, ok := innerApp.Args[0].(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var in app arg, got %T", innerApp.Args[0])
	}
	if innerVar.Name != "x" {
		t.Errorf("expected x (shadowed), got %s", innerVar.Name)
	}
}

func TestSubstituteLambdaVar_ShadowedByLet(t *testing.T) {
	// let x = 5 in x: let-bound x shadows the substitution target
	body := &core.Let{
		Name:  "x",
		Value: &core.Lit{Kind: core.IntLit, Value: int64(5)},
		Body:  &core.Var{Name: "x"},
	}
	replacement := &core.Var{Name: "h"}

	result := SubstituteLambdaVar(body, "x", replacement)
	let, ok := result.(*core.Let)
	if !ok {
		t.Fatalf("expected *core.Let, got %T", result)
	}
	// Body should NOT be replaced (shadowed by let binding)
	bv, ok := let.Body.(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var for body, got %T", let.Body)
	}
	if bv.Name != "x" {
		t.Errorf("expected x (shadowed in body), got %s", bv.Name)
	}
}

// --- AllHigherOrderIsInlinable tests ---

func TestAllHigherOrderIsInlinable_True(t *testing.T) {
	// Body with only map(\x -> x + 1, xs) — all HOF is inlinable
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	if !AllHigherOrderIsInlinable(body) {
		t.Error("expected AllHigherOrderIsInlinable to return true for map with literal lambda")
	}
}

func TestAllHigherOrderIsInlinable_False_UnknownHOF(t *testing.T) {
	// Body with unknown HOF: apply(f, x) where f is a lambda
	body := &core.App{
		Func: &core.Var{Name: "apply"},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body:   &core.Var{Name: "x"},
			},
		},
	}

	if AllHigherOrderIsInlinable(body) {
		t.Error("expected AllHigherOrderIsInlinable to return false for unknown HOF")
	}
}

func TestAllHigherOrderIsInlinable_False_MapWithVarArg(t *testing.T) {
	// map(f, xs) where f is variable — higher-order but NOT inlinable
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "f"},
			&core.Var{Name: "xs"},
		},
	}

	// This body IS higher-order? Actually no — Var is not Lambda.
	// walkForHigherOrder checks for Lambda in arg position.
	// A Var in arg position is NOT higher-order. So hasHigherOrder returns false.
	// This test verifies that map(f, xs) passes through normally
	// (it won't be flagged as higher-order by hasHigherOrder).
	if hasHigherOrder(body) {
		t.Error("map(f, xs) with variable f should not be flagged as higher-order")
	}
}

func TestAllHigherOrderIsInlinable_NoHigherOrder(t *testing.T) {
	// No higher-order at all → AllHigherOrderIsInlinable returns true trivially
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}

	if !AllHigherOrderIsInlinable(body) {
		t.Error("expected true when there is no higher-order at all")
	}
}

// --- SpecializeMap tests ---

func TestSpecializeMap_Simple(t *testing.T) {
	// map(\x -> x + 1, xs) → specialized recursive function
	lambda := &core.Lambda{
		Params: []string{"x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
		},
	}

	specBody, specName := SpecializeMap("testFn", lambda, "xs")
	if specName == "" {
		t.Fatal("expected non-empty specialized function name")
	}
	if specBody == nil {
		t.Fatal("expected non-nil specialized body")
	}

	// The specialized body should reference the specialized name (recursive)
	if !containsRef(specBody, specName) {
		t.Error("specialized body should contain a recursive self-reference")
	}
}

func TestSpecializeFilter_Simple(t *testing.T) {
	// filter(\x -> x > 0, xs)
	lambda := &core.Lambda{
		Params: []string{"x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "gt_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
	}

	specBody, specName := SpecializeFilter("testFn", lambda, "xs")
	if specName == "" {
		t.Fatal("expected non-empty specialized function name")
	}
	if specBody == nil {
		t.Fatal("expected non-nil specialized body")
	}

	// The specialized body should be recursive
	if !containsRef(specBody, specName) {
		t.Error("specialized body should contain a recursive self-reference")
	}
}

func TestSpecializeFoldl_Simple(t *testing.T) {
	// foldl(\acc x -> acc + x, 0, xs)
	lambda := &core.Lambda{
		Params: []string{"acc", "x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "acc"},
				&core.Var{Name: "x"},
			},
		},
	}

	specBody, specName := SpecializeFoldl("testFn", lambda, "xs")
	if specName == "" {
		t.Fatal("expected non-empty specialized function name")
	}
	if specBody == nil {
		t.Fatal("expected non-nil specialized body")
	}

	// The specialized body should be recursive
	if !containsRef(specBody, specName) {
		t.Error("specialized body should contain a recursive self-reference")
	}
}

// --- Full pipeline tests ---

func TestSpecializeMapAndUnroll(t *testing.T) {
	// map(\x -> x + 1, xs) → specialize → unroll → valid SMT-LIB
	lambda := &core.Lambda{
		Params: []string{"x"},
		Body: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
		},
	}

	specBody, specName := SpecializeMap("incrementAll", lambda, "xs")

	// Now unroll it
	cfg := UnrollConfig{
		FuncName: specName,
		Params: []FunctionParam{
			{Name: "xs", Type: &types.TList{Element: &types.TCon{Name: "int"}}},
		},
		Body:       specBody,
		ReturnSort: "(Seq Int)",
		Depth:      3,
	}

	result, err := UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Fatalf("unrolling specialized map: %v", err)
	}

	// Should have 4 declarations (level 0 uninterpreted + levels 1-3)
	if len(result.Declarations) != 4 {
		t.Errorf("expected 4 declarations, got %d", len(result.Declarations))
	}

	// Level 0 should be declare-fun
	if !strings.HasPrefix(result.Declarations[0], "(declare-fun") {
		t.Errorf("expected declare-fun at level 0, got: %s", result.Declarations[0])
	}

	// define-fun levels should contain seq. operations (list primitives)
	for i := 1; i < len(result.Declarations); i++ {
		if !strings.Contains(result.Declarations[i], "(define-fun") {
			t.Errorf("expected define-fun at level %d, got: %s", i, result.Declarations[i])
		}
	}

	t.Logf("Top-level name: %s", result.TopLevelName)
	for i, d := range result.Declarations {
		t.Logf("Declaration %d: %s", i, d)
	}
}

func TestInlineHOFCalls_ReplacesMapInBody(t *testing.T) {
	// Test the full InlineHOFCalls function which replaces HOF calls in a body
	// with calls to specialized recursive functions
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	result := InlineHOFCalls("myFunc", body, 3)
	if result == nil {
		t.Fatal("expected non-nil InlineResult")
	}

	// The new body should NOT contain any Lambda in arg position
	if hasHigherOrder(result.NewBody) {
		t.Error("after inlining, body should not have higher-order functions")
	}

	// Should have specialization declarations
	if len(result.Specializations) == 0 {
		t.Error("expected at least one specialization")
	}

	// The specialization should have SMT-LIB declarations
	for _, spec := range result.Specializations {
		if len(spec.Declarations) == 0 {
			t.Error("expected non-empty declarations in specialization")
		}
		if spec.TopLevelName == "" {
			t.Error("expected non-empty TopLevelName")
		}
	}
}

func TestIsSMTEncodable_InlinableMapWithLambda(t *testing.T) {
	// A function with map(\x -> x + 1, xs) should now be encodable
	// because the HOF call is inlinable (literal lambda).
	meta := &core.DeclMeta{
		Name:   "incrementAll",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	ok, reasons := IsSMTEncodable("incrementAll", meta, body)
	if !ok {
		t.Fatalf("expected encodable (inlinable map with lambda), got reasons: %v", reasons)
	}
}

func TestIsSMTEncodable_NonInlinableMapWithVar(t *testing.T) {
	// map(f, xs) where f is a variable — still NOT encodable
	// (hasHigherOrder returns false because Var is not Lambda in arg position,
	// but it's unencodable because map_List is not in the SMT encoding maps)
	meta := &core.DeclMeta{
		Name:   "mapWithVar",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "f"},
			&core.Var{Name: "xs"},
		},
	}

	ok, _ := IsSMTEncodable("mapWithVar", meta, body)
	if ok {
		t.Fatal("expected not encodable — map_List with variable function arg should be rejected")
	}
}

func TestInlineHOFCalls_NoHOF(t *testing.T) {
	// No HOF calls → should return nil (nothing to inline)
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Lit{Kind: core.IntLit, Value: int64(1)},
		},
	}

	result := InlineHOFCalls("myFunc", body, 3)
	if result != nil {
		t.Error("expected nil InlineResult when no HOF calls present")
	}
}

// --- EncodeFunction integration tests ---

func TestEncodeFunction_WithInlinableMap(t *testing.T) {
	// Full pipeline: incrementAll(xs) = map(\x -> x + 1, xs)
	// ensures { _list_length(result) == _list_length(xs) }
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "map_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	params := []FunctionParam{
		{Name: "xs", Type: &types.TList{Element: &types.TCon{Name: "int"}}},
	}

	meta := &core.DeclMeta{
		Name:   "incrementAll",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
							Args: []core.CoreExpr{&core.Var{Name: "result"}},
						},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	opts := EncodeFunctionOpts{
		HOFInlineDepth: 3,
	}

	result, err := EncodeFunction("incrementAll", params, body, "(Seq Int)", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with HOF inlining: unexpected error: %v", err)
	}

	// Should contain specialized map function declarations
	if !strings.Contains(result.SMTLib, "_map_spec_incrementAll") {
		t.Error("expected specialized map function in SMT-LIB output")
	}

	// Should contain declare-fun for level 0
	if !strings.Contains(result.SMTLib, "declare-fun _map_spec_incrementAll_0") {
		t.Error("expected declare-fun for level 0 specialization")
	}

	// Should contain define-fun chains
	if !strings.Contains(result.SMTLib, "define-fun _map_spec_incrementAll_1") {
		t.Error("expected define-fun for level 1 specialization")
	}

	// Should contain check-sat
	if !strings.Contains(result.SMTLib, "(check-sat)") {
		t.Error("expected (check-sat) in SMT-LIB output")
	}

	// Body should NOT contain lambda (it was replaced by specialization)
	if strings.Contains(result.SMTLib, "lambda") {
		t.Error("SMT-LIB output should not contain lambda after HOF inlining")
	}

	t.Logf("Generated SMT-LIB:\n%s", result.SMTLib)
}

func TestEncodeFunction_WithInlinableFilter(t *testing.T) {
	// positives(xs) = filter(\x -> x > 0, xs)
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "filter_List"}},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "gt_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
			&core.Var{Name: "xs"},
		},
	}

	params := []FunctionParam{
		{Name: "xs", Type: &types.TList{Element: &types.TCon{Name: "int"}}},
	}

	meta := &core.DeclMeta{
		Name:   "positives",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
							Args: []core.CoreExpr{&core.Var{Name: "result"}},
						},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	opts := EncodeFunctionOpts{
		HOFInlineDepth: 3,
	}

	result, err := EncodeFunction("positives", params, body, "(Seq Int)", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with filter inlining: unexpected error: %v", err)
	}

	// Should contain specialized filter function declarations
	if !strings.Contains(result.SMTLib, "_filter_spec_positives") {
		t.Error("expected specialized filter function in SMT-LIB output")
	}

	t.Logf("Generated SMT-LIB:\n%s", result.SMTLib)
}
