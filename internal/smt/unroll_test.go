package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// --- ReplaceSelfCalls tests ---

func TestReplaceSelfCalls_SimpleVar(t *testing.T) {
	expr := &core.Var{Name: "factorial"}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	v, ok := result.(*core.Var)
	if !ok {
		t.Fatalf("expected *core.Var, got %T", result)
	}
	if v.Name != "factorial_0" {
		t.Errorf("expected factorial_0, got %s", v.Name)
	}
	// Original should be unchanged
	if expr.Name != "factorial" {
		t.Errorf("original mutated: got %s", expr.Name)
	}
}

func TestReplaceSelfCalls_VarGlobal(t *testing.T) {
	expr := &core.VarGlobal{Ref: core.GlobalRef{Module: "mymod", Name: "factorial"}}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_1")
	vg, ok := result.(*core.VarGlobal)
	if !ok {
		t.Fatalf("expected *core.VarGlobal, got %T", result)
	}
	if vg.Ref.Name != "factorial_1" {
		t.Errorf("expected factorial_1, got %s", vg.Ref.Name)
	}
	// Original unchanged
	if expr.Ref.Name != "factorial" {
		t.Errorf("original mutated: got %s", expr.Ref.Name)
	}
}

func TestReplaceSelfCalls_NoMatch(t *testing.T) {
	expr := &core.Var{Name: "other"}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	if result != expr {
		t.Error("expected same pointer when no replacement needed")
	}
}

func TestReplaceSelfCalls_Lit(t *testing.T) {
	expr := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	if result != expr {
		t.Error("Lit should be returned as-is")
	}
}

func TestReplaceSelfCalls_ShadowedByLambda(t *testing.T) {
	// Lambda re-binds "factorial" as a parameter — should NOT replace in body
	expr := &core.Lambda{
		Params: []string{"factorial"},
		Body:   &core.Var{Name: "factorial"},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	// Lambda shadows, so body should NOT be replaced
	if result != expr {
		t.Error("expected same pointer when name is shadowed by lambda param")
	}
}

func TestReplaceSelfCalls_ShadowedByLet(t *testing.T) {
	// Let re-binds "factorial" — value should be replaced, body should NOT
	expr := &core.Let{
		Name:  "factorial",
		Value: &core.Var{Name: "factorial"}, // should be replaced
		Body:  &core.Var{Name: "factorial"}, // should NOT be replaced (shadowed)
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	let, ok := result.(*core.Let)
	if !ok {
		t.Fatalf("expected *core.Let, got %T", result)
	}
	// Value should be replaced
	v, ok := let.Value.(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("let value should be replaced, got %v", let.Value)
	}
	// Body should NOT be replaced (shadowed)
	bv, ok := let.Body.(*core.Var)
	if !ok || bv.Name != "factorial" {
		t.Errorf("let body should NOT be replaced (shadowed), got %v", let.Body)
	}
}

func TestReplaceSelfCalls_NestedInIf(t *testing.T) {
	expr := &core.If{
		Cond: &core.Lit{Kind: core.BoolLit, Value: true},
		Then: &core.Var{Name: "factorial"},
		Else: &core.Lit{Kind: core.IntLit, Value: int64(1)},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	ifExpr, ok := result.(*core.If)
	if !ok {
		t.Fatalf("expected *core.If, got %T", result)
	}
	v, ok := ifExpr.Then.(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("Then branch should be replaced, got %v", ifExpr.Then)
	}
}

func TestReplaceSelfCalls_InApp(t *testing.T) {
	// factorial(n - 1)
	expr := &core.App{
		Func: &core.Var{Name: "factorial"},
		Args: []core.CoreExpr{
			&core.BinOp{
				Op:    "-",
				Left:  &core.Var{Name: "n"},
				Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
		},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	app, ok := result.(*core.App)
	if !ok {
		t.Fatalf("expected *core.App, got %T", result)
	}
	v, ok := app.Func.(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("App.Func should be replaced, got %v", app.Func)
	}
}

func TestReplaceSelfCalls_InMatch(t *testing.T) {
	expr := &core.Match{
		Scrutinee: &core.Var{Name: "n"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: int64(0)},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
			{
				Pattern: &core.VarPattern{Name: "_"},
				Body:    &core.Var{Name: "factorial"},
			},
		},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	match, ok := result.(*core.Match)
	if !ok {
		t.Fatalf("expected *core.Match, got %T", result)
	}
	v, ok := match.Arms[1].Body.(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("match arm body should be replaced, got %v", match.Arms[1].Body)
	}
}

func TestReplaceSelfCalls_InRecord(t *testing.T) {
	expr := &core.Record{
		Fields: map[string]core.CoreExpr{
			"val": &core.Var{Name: "factorial"},
		},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	rec, ok := result.(*core.Record)
	if !ok {
		t.Fatalf("expected *core.Record, got %T", result)
	}
	v, ok := rec.Fields["val"].(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("record field should be replaced, got %v", rec.Fields["val"])
	}
}

func TestReplaceSelfCalls_InIntrinsic(t *testing.T) {
	expr := &core.Intrinsic{
		Op:   core.OpMul,
		Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Var{Name: "factorial"}},
	}
	result := ReplaceSelfCalls(expr, "factorial", "factorial_0")
	intr, ok := result.(*core.Intrinsic)
	if !ok {
		t.Fatalf("expected *core.Intrinsic, got %T", result)
	}
	v, ok := intr.Args[1].(*core.Var)
	if !ok || v.Name != "factorial_0" {
		t.Errorf("intrinsic arg should be replaced, got %v", intr.Args[1])
	}
}

// --- UnrollRecursiveFunction tests ---

func TestUnrollRecursiveFunction_Depth1(t *testing.T) {
	// Simple: f(n) = if n == 0 then 1 else n * f(n-1)
	cfg := UnrollConfig{
		FuncName: "factorial",
		Params: []FunctionParam{
			{Name: "n", Type: &types.TCon{Name: "int"}},
		},
		Body:       makeFactorialBody("factorial"),
		ReturnSort: "Int",
		Depth:      1,
	}

	result, err := UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Depth 1 → 2 declarations (declare-fun + 1 define-fun)
	if len(result.Declarations) != 2 {
		t.Errorf("expected 2 declarations for depth 1, got %d", len(result.Declarations))
	}

	// First should be declare-fun (uninterpreted)
	if !strings.HasPrefix(result.Declarations[0], "(declare-fun factorial_0") {
		t.Errorf("expected declare-fun factorial_0, got %q", result.Declarations[0])
	}

	// Second should be define-fun
	if !strings.HasPrefix(result.Declarations[1], "(define-fun factorial_1") {
		t.Errorf("expected define-fun factorial_1, got %q", result.Declarations[1])
	}

	// define-fun should reference factorial_0, NOT factorial
	if strings.Contains(result.Declarations[1], " factorial ") || strings.Contains(result.Declarations[1], "(factorial ") {
		t.Errorf("define-fun should not reference original name 'factorial': %q", result.Declarations[1])
	}
	if !strings.Contains(result.Declarations[1], "factorial_0") {
		t.Errorf("define-fun at depth 1 should reference factorial_0: %q", result.Declarations[1])
	}

	if result.TopLevelName != "factorial_1" {
		t.Errorf("expected top-level name factorial_1, got %s", result.TopLevelName)
	}
}

func TestUnrollRecursiveFunction_Depth3(t *testing.T) {
	cfg := UnrollConfig{
		FuncName: "factorial",
		Params: []FunctionParam{
			{Name: "n", Type: &types.TCon{Name: "int"}},
		},
		Body:       makeFactorialBody("factorial"),
		ReturnSort: "Int",
		Depth:      3,
	}

	result, err := UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Depth 3 → 4 declarations
	if len(result.Declarations) != 4 {
		t.Errorf("expected 4 declarations for depth 3, got %d", len(result.Declarations))
	}

	// Check naming convention
	if !strings.Contains(result.Declarations[0], "factorial_0") {
		t.Errorf("level 0 should be factorial_0: %q", result.Declarations[0])
	}
	if !strings.Contains(result.Declarations[1], "factorial_1") {
		t.Errorf("level 1 should be factorial_1: %q", result.Declarations[1])
	}
	if !strings.Contains(result.Declarations[2], "factorial_2") {
		t.Errorf("level 2 should be factorial_2: %q", result.Declarations[2])
	}
	if !strings.Contains(result.Declarations[3], "factorial_3") {
		t.Errorf("level 3 should be factorial_3: %q", result.Declarations[3])
	}

	// Each define-fun at level k should reference level k-1
	if !strings.Contains(result.Declarations[1], "factorial_0") {
		t.Errorf("level 1 should call factorial_0: %q", result.Declarations[1])
	}
	if !strings.Contains(result.Declarations[2], "factorial_1") {
		t.Errorf("level 2 should call factorial_1: %q", result.Declarations[2])
	}
	if !strings.Contains(result.Declarations[3], "factorial_2") {
		t.Errorf("level 3 should call factorial_2: %q", result.Declarations[3])
	}

	if result.TopLevelName != "factorial_3" {
		t.Errorf("expected top-level name factorial_3, got %s", result.TopLevelName)
	}
}

func TestUnrollRecursiveFunction_DepthValidation(t *testing.T) {
	cfg := UnrollConfig{
		FuncName:   "f",
		Params:     []FunctionParam{{Name: "n", Type: &types.TCon{Name: "int"}}},
		Body:       &core.Lit{Kind: core.IntLit, Value: int64(0)},
		ReturnSort: "Int",
	}

	// Depth 0 → error
	cfg.Depth = 0
	_, err := UnrollRecursiveFunction(cfg)
	if err == nil {
		t.Error("expected error for depth 0")
	}

	// Depth 11 → error
	cfg.Depth = 11
	_, err = UnrollRecursiveFunction(cfg)
	if err == nil {
		t.Error("expected error for depth 11")
	}

	// Depth 1 → ok
	cfg.Depth = 1
	_, err = UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Errorf("depth 1 should be valid: %v", err)
	}

	// Depth 10 → ok
	cfg.Depth = 10
	result, err := UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Errorf("depth 10 should be valid: %v", err)
	}
	if len(result.Declarations) != 11 {
		t.Errorf("depth 10 should produce 11 declarations, got %d", len(result.Declarations))
	}
}

func TestUnrollRecursiveFunction_NonRecursiveBody(t *testing.T) {
	// Non-recursive body: all levels should produce identical define-fun bodies
	cfg := UnrollConfig{
		FuncName: "id",
		Params: []FunctionParam{
			{Name: "n", Type: &types.TCon{Name: "int"}},
		},
		Body:       &core.Var{Name: "n"},
		ReturnSort: "Int",
		Depth:      3,
	}

	result, err := UnrollRecursiveFunction(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All define-fun bodies should be just "n"
	for i := 1; i < len(result.Declarations); i++ {
		if !strings.Contains(result.Declarations[i], " n)") {
			t.Errorf("non-recursive body at level %d should just be 'n': %q", i, result.Declarations[i])
		}
	}
}

func TestUnrollRecursiveFunction_DeclarationCount(t *testing.T) {
	// Property: depth N → exactly N+1 declarations
	for depth := 1; depth <= 5; depth++ {
		cfg := UnrollConfig{
			FuncName: "f",
			Params: []FunctionParam{
				{Name: "n", Type: &types.TCon{Name: "int"}},
			},
			Body:       &core.Var{Name: "n"},
			ReturnSort: "Int",
			Depth:      depth,
		}
		result, err := UnrollRecursiveFunction(cfg)
		if err != nil {
			t.Fatalf("depth %d: unexpected error: %v", depth, err)
		}
		if len(result.Declarations) != depth+1 {
			t.Errorf("depth %d: expected %d declarations, got %d", depth, depth+1, len(result.Declarations))
		}
	}
}

// --- Integration tests ---

func TestEncodeFunction_WithRecursiveDepth(t *testing.T) {
	// factorial(n) = if n <= 0 then 1 else n * factorial(n-1)
	// requires { n >= 0 }
	// ensures { result >= 1 }
	body := makeFactorialBody("factorial")
	params := []FunctionParam{
		{Name: "n", Type: &types.TCon{Name: "int"}},
	}
	meta := &core.DeclMeta{
		Name:   "factorial",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.RequiresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "n"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
			},
		},
	}

	opts := EncodeFunctionOpts{
		RecursiveDepth: 3,
	}

	result, err := EncodeFunction("factorial", params, body, "Int", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with RecursiveDepth=3: unexpected error: %v", err)
	}

	// Should contain unrolled declarations
	if !strings.Contains(result.SMTLib, "declare-fun factorial_0") {
		t.Error("expected declare-fun factorial_0 in SMT-LIB output")
	}
	if !strings.Contains(result.SMTLib, "define-fun factorial_1") {
		t.Error("expected define-fun factorial_1 in SMT-LIB output")
	}
	if !strings.Contains(result.SMTLib, "define-fun factorial_2") {
		t.Error("expected define-fun factorial_2 in SMT-LIB output")
	}
	if !strings.Contains(result.SMTLib, "define-fun factorial_3") {
		t.Error("expected define-fun factorial_3 in SMT-LIB output")
	}

	// Body should use factorial_3 (top-level)
	if !strings.Contains(result.SMTLib, "factorial_3 n") {
		t.Errorf("expected factorial_3 as body reference in SMT-LIB:\n%s", result.SMTLib)
	}

	// Should contain check-sat
	if !strings.Contains(result.SMTLib, "(check-sat)") {
		t.Error("expected (check-sat) in SMT-LIB output")
	}
}

func TestEncodeFunction_WithoutRecursiveDepth_ReturnsError(t *testing.T) {
	// Without RecursiveDepth, a recursive function should still encode normally
	// (the body contains self-references, but EncodeExpr handles Var nodes)
	body := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "le_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "n"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Else: &core.Var{Name: "n"}, // simplified non-recursive
	}
	params := []FunctionParam{
		{Name: "n", Type: &types.TCon{Name: "int"}},
	}
	meta := &core.DeclMeta{
		Name:   "simple",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.RequiresKind,
				Expr: &core.Lit{Kind: core.BoolLit, Value: true},
			},
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	// No RecursiveDepth — should encode normally
	result, err := EncodeFunction("simple", params, body, "Int", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT contain unrolled declarations
	if strings.Contains(result.SMTLib, "simple_0") {
		t.Error("should not have unrolled declarations without RecursiveDepth")
	}
}

// --- Helpers ---

// makeFactorialBody creates: if n <= 0 then 1 else n * factorial(n - 1)
// using the provided function name for the recursive call.
func makeFactorialBody(funcName string) core.CoreExpr {
	return &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "le_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "n"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Else: &core.Intrinsic{
			Op: core.OpMul,
			Args: []core.CoreExpr{
				&core.Var{Name: "n"},
				&core.App{
					Func: &core.Var{Name: funcName},
					Args: []core.CoreExpr{
						&core.Intrinsic{
							Op: core.OpSub,
							Args: []core.CoreExpr{
								&core.Var{Name: "n"},
								&core.Lit{Kind: core.IntLit, Value: int64(1)},
							},
						},
					},
				},
			},
		},
	}
}
