package smt

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// --- EncodeExpr tests ---

func TestEncodeExpr_IntLit(t *testing.T) {
	expr := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestEncodeExpr_NegativeIntLit(t *testing.T) {
	expr := &core.Lit{Kind: core.IntLit, Value: int64(-5)}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(- 5)" {
		t.Errorf("got %q, want %q", got, "(- 5)")
	}
}

func TestEncodeExpr_FloatLit(t *testing.T) {
	expr := &core.Lit{Kind: core.FloatLit, Value: float64(3.14)}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.14" {
		t.Errorf("got %q, want %q", got, "3.14")
	}
}

func TestEncodeExpr_BoolLit(t *testing.T) {
	tests := []struct {
		val  bool
		want string
	}{
		{true, "true"},
		{false, "false"},
	}
	for _, tt := range tests {
		expr := &core.Lit{Kind: core.BoolLit, Value: tt.val}
		got, err := EncodeExpr(expr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestEncodeExpr_StringLit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", `"hello"`},
		{"", `""`},
		{`say "hi"`, `"say ""hi"""`}, // SMT-LIB escapes " as ""
		{`back\slash`, `"back\\slash"`},
	}
	for _, tt := range tests {
		expr := &core.Lit{Kind: core.StringLit, Value: tt.input}
		got, err := EncodeExpr(expr)
		if err != nil {
			t.Errorf("unexpected error for string %q: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("encodeLit(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEncodeExpr_Var(t *testing.T) {
	expr := &core.Var{Name: "x"}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "x" {
		t.Errorf("got %q, want %q", got, "x")
	}
}

func TestEncodeExpr_If(t *testing.T) {
	expr := &core.If{
		Cond: &core.Var{Name: "c"},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Else: &core.Lit{Kind: core.IntLit, Value: int64(0)},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(ite c 1 0)" {
		t.Errorf("got %q, want %q", got, "(ite c 1 0)")
	}
}

func TestEncodeExpr_Let(t *testing.T) {
	expr := &core.Let{
		Name:  "x",
		Value: &core.Lit{Kind: core.IntLit, Value: int64(5)},
		Body:  &core.Var{Name: "x"},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(let ((x 5)) x)" {
		t.Errorf("got %q, want %q", got, "(let ((x 5)) x)")
	}
}

func TestEncodeExpr_BuiltinOp_Binary(t *testing.T) {
	// App(VarGlobal($builtin.ge_Int), [x, 0])
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(>= x 0)" {
		t.Errorf("got %q, want %q", got, "(>= x 0)")
	}
}

func TestEncodeExpr_BuiltinOp_Unary(t *testing.T) {
	// App(VarGlobal($builtin.neg_Int), [x])
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "neg_Int"}},
		Args: []core.CoreExpr{&core.Var{Name: "x"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(- x)" {
		t.Errorf("got %q, want %q", got, "(- x)")
	}
}

func TestEncodeExpr_CurriedBuiltin(t *testing.T) {
	// App(App(VarGlobal($builtin.add_Int), [x]), [y]) — curried form
	expr := &core.App{
		Func: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
			Args: []core.CoreExpr{&core.Var{Name: "x"}},
		},
		Args: []core.CoreExpr{&core.Var{Name: "y"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(+ x y)" {
		t.Errorf("got %q, want %q", got, "(+ x y)")
	}
}

func TestEncodeExpr_Match_Enum(t *testing.T) {
	// match season { LOW_SEASON => 15, HIGH_SEASON => 20 }
	expr := &core.Match{
		Scrutinee: &core.Var{Name: "season"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{Name: "LOW_SEASON"},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(15)},
			},
			{
				Pattern: &core.ConstructorPattern{Name: "HIGH_SEASON"},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(20)},
			},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(match season ((LOW_SEASON 15) (HIGH_SEASON 20)))"
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestEncodeExpr_Match_ADTWithFields(t *testing.T) {
	// match x { Some(v) => v, None => 0 }
	expr := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Some",
					Args: []core.CorePattern{&core.VarPattern{Name: "v"}},
				},
				Body: &core.Var{Name: "v"},
			},
			{
				Pattern: &core.ConstructorPattern{Name: "None"},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(match x (((Some v) v) (None 0)))"
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestEncodeExpr_PreLowered_Intrinsic(t *testing.T) {
	// Pre-lowered: Intrinsic(OpGe, [x, 0])
	expr := &core.Intrinsic{
		Op: core.OpGe,
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(>= x 0)" {
		t.Errorf("got %q, want %q", got, "(>= x 0)")
	}
}

func TestEncodeExpr_PreLowered_BinOp(t *testing.T) {
	expr := &core.BinOp{
		Op:    ">=",
		Left:  &core.Var{Name: "x"},
		Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(>= x 0)" {
		t.Errorf("got %q, want %q", got, "(>= x 0)")
	}
}

func TestEncodeExpr_PreLowered_UnOp(t *testing.T) {
	expr := &core.UnOp{
		Op:      "-",
		Operand: &core.Var{Name: "x"},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(- x)" {
		t.Errorf("got %q, want %q", got, "(- x)")
	}
}

func TestEncodeExpr_Lambda_Error(t *testing.T) {
	expr := &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}
	_, err := EncodeExpr(expr)
	if err == nil {
		t.Error("expected error for lambda expression")
	}
}

func TestEncodeExpr_LetRec_Error(t *testing.T) {
	expr := &core.LetRec{
		Bindings: []core.RecBinding{{Name: "f", Value: &core.Var{Name: "f"}}},
		Body:     &core.Var{Name: "f"},
	}
	_, err := EncodeExpr(expr)
	if err == nil {
		t.Error("expected error for letrec expression")
	}
}

func TestEncodeExpr_Nil(t *testing.T) {
	_, err := EncodeExpr(nil)
	if err == nil {
		t.Error("expected error for nil expression")
	}
}

func TestEncodeExpr_ConstructorApp(t *testing.T) {
	// Some(42) encoded from VarGlobal
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "examples/park", Name: "Some"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(42)}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(Some 42)" {
		t.Errorf("got %q, want %q", got, "(Some 42)")
	}
}

func TestEncodeExpr_NestedIfWithBuiltins(t *testing.T) {
	// if age < 5 then 0 else if age >= 65 then 5 else 15
	expr := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "lt_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "age"},
				&core.Lit{Kind: core.IntLit, Value: int64(5)},
			},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.If{
			Cond: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
				Args: []core.CoreExpr{
					&core.Var{Name: "age"},
					&core.Lit{Kind: core.IntLit, Value: int64(65)},
				},
			},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(5)},
			Else: &core.Lit{Kind: core.IntLit, Value: int64(15)},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(ite (< age 5) 0 (ite (>= age 65) 5 15))"
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

// --- EncodeFunction tests ---

func TestEncodeFunction_Absolute(t *testing.T) {
	// func absolute(x: int) -> int
	// requires { x >= 0 }
	// ensures { result >= 0 }
	// { x }
	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
	}
	body := &core.Var{Name: "x"}
	meta := &core.DeclMeta{
		Name:   "absolute",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.RequiresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
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
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	result, err := EncodeFunction("absolute", params, body, "", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the SMT-LIB output contains expected elements
	smtlib := result.SMTLib
	if !strings.Contains(smtlib, "; Verification of absolute") {
		t.Error("missing verification comment")
	}
	if !strings.Contains(smtlib, "(declare-const x Int)") {
		t.Error("missing parameter declaration")
	}
	if !strings.Contains(smtlib, "(assert (>= x 0))") {
		t.Error("missing precondition assertion")
	}
	if !strings.Contains(smtlib, "(define-const result Int x)") {
		t.Error("missing result definition")
	}
	if !strings.Contains(smtlib, "(assert (not (>= result 0)))") {
		t.Error("missing negated postcondition")
	}
	if !strings.Contains(smtlib, "(check-sat)") {
		t.Error("missing check-sat")
	}
	if !strings.Contains(smtlib, "(get-model)") {
		t.Error("missing get-model")
	}
}

func TestEncodeFunction_WithADT(t *testing.T) {
	// func f(season: Season) -> int
	// requires { true }
	// ensures { result >= 0 }
	// { match season { LOW_SEASON => 5, HIGH_SEASON => 10 } }
	params := []FunctionParam{
		{Name: "season", Type: &types.TCon{Name: "Season"}},
	}
	body := &core.Match{
		Scrutinee: &core.Var{Name: "season"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{Name: "LOW_SEASON"},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(5)},
			},
			{
				Pattern: &core.ConstructorPattern{Name: "HIGH_SEASON"},
				Body:    &core.Lit{Kind: core.IntLit, Value: int64(10)},
			},
		},
	}
	meta := &core.DeclMeta{
		Name:   "f",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
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
	adtTypes := map[string][]ADTVariant{
		"Season": {
			{Name: "LOW_SEASON"},
			{Name: "HIGH_SEASON"},
		},
	}

	result, err := EncodeFunction("f", params, body, "", meta, adtTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	smtlib := result.SMTLib
	if !strings.Contains(smtlib, "(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))") {
		t.Error("missing datatype declaration")
	}
	if !strings.Contains(smtlib, "(declare-const season Season)") {
		t.Error("missing season parameter declaration")
	}
	if !strings.Contains(smtlib, "(assert true)") {
		t.Error("missing true precondition")
	}
	if !strings.Contains(smtlib, "(match season ((LOW_SEASON 5) (HIGH_SEASON 10)))") {
		t.Error("missing match expression in result definition")
	}
}

func TestEncodeFunction_NoEnsures(t *testing.T) {
	// Function with only requires (no ensures → nothing to verify)
	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
	}
	body := &core.Var{Name: "x"}
	meta := &core.DeclMeta{
		Name:   "f",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}

	result, err := EncodeFunction("f", params, body, "", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still produce valid SMT-LIB, just no negated postconditions
	if strings.Contains(result.SMTLib, "assert (not") {
		t.Error("should not have negated postcondition when no ensures clause")
	}
}

func TestEncodeFunction_StringParam(t *testing.T) {
	params := []FunctionParam{
		{Name: "s", Type: &types.TCon{Name: "string"}},
	}
	body := &core.Var{Name: "s"}
	meta := &core.DeclMeta{
		Name:   "f",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}

	result, err := EncodeFunction("f", params, body, "String", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error for string parameter: %v", err)
	}
	if !strings.Contains(result.SMTLib, "(declare-const s String)") {
		t.Error("expected string parameter declaration")
	}
	if !strings.Contains(result.SMTLib, "(define-const result String s)") {
		t.Error("expected result as String sort")
	}
}

func TestEncodeFunction_UnencodableParam(t *testing.T) {
	params := []FunctionParam{
		{Name: "f", Type: &types.TFunc{}},
	}
	body := &core.Var{Name: "f"}
	meta := &core.DeclMeta{
		Name:   "g",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}

	_, err := EncodeFunction("g", params, body, "", meta, nil)
	if err == nil {
		t.Error("expected error for function-type parameter")
	}
}

func TestEncodeExpr_AllBuiltinOps(t *testing.T) {
	tests := []struct {
		builtin string
		want    string
	}{
		{"add_Int", "(+ x y)"},
		{"sub_Int", "(- x y)"},
		{"mul_Int", "(* x y)"},
		{"div_Int", "(div x y)"},
		{"mod_Int", "(mod x y)"},
		{"eq_Int", "(= x y)"},
		{"ne_Int", "(distinct x y)"},
		{"lt_Int", "(< x y)"},
		{"le_Int", "(<= x y)"},
		{"gt_Int", "(> x y)"},
		{"ge_Int", "(>= x y)"},
		{"and_Bool", "(and x y)"},
		{"or_Bool", "(or x y)"},
	}
	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			expr := &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: tt.builtin}},
				Args: []core.CoreExpr{&core.Var{Name: "x"}, &core.Var{Name: "y"}},
			}
			got, err := EncodeExpr(expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeExpr_AllUnaryOps(t *testing.T) {
	tests := []struct {
		builtin string
		want    string
	}{
		{"not_Bool", "(not x)"},
		{"neg_Int", "(- x)"},
		{"neg_Float", "(- x)"},
	}
	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			expr := &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: tt.builtin}},
				Args: []core.CoreExpr{&core.Var{Name: "x"}},
			}
			got, err := EncodeExpr(expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferResultSort(t *testing.T) {
	ctx := NewSMTContext()
	ctx.Variables["x"] = "Int"
	ctx.Variables["y"] = "Real"

	tests := []struct {
		name string
		body core.CoreExpr
		want string
	}{
		{"int lit", &core.Lit{Kind: core.IntLit, Value: int64(42)}, "Int"},
		{"float lit", &core.Lit{Kind: core.FloatLit, Value: 3.14}, "Real"},
		{"bool lit", &core.Lit{Kind: core.BoolLit, Value: true}, "Bool"},
		{"var x", &core.Var{Name: "x"}, "Int"},
		{"var y", &core.Var{Name: "y"}, "Real"},
		{"if", &core.If{
			Cond: &core.Lit{Kind: core.BoolLit, Value: true},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
			Else: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		}, "Int"},
		{"unknown var", &core.Var{Name: "z"}, "Int"}, // fallback
		{"nil", nil, "Int"},                          // fallback
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferResultSort(nil, tt.body, ctx, nil)
			if got != tt.want {
				t.Errorf("inferResultSort = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Integration-style test: park.ail admissionFee ---

func TestEncodeFunction_ParkAdmissionFee(t *testing.T) {
	// Simplified version of park.ail:admissionFee
	// match season {
	//   LOW_SEASON => if age < 5 then 0 else if age >= 65 then 5 else 15,
	//   HIGH_SEASON => if age >= 65 then 10 else 20
	// }
	params := []FunctionParam{
		{Name: "age", Type: &types.TCon{Name: "int"}},
		{Name: "season", Type: &types.TCon{Name: "Season"}},
	}

	lowBody := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "lt_Int"}},
			Args: []core.CoreExpr{&core.Var{Name: "age"}, &core.Lit{Kind: core.IntLit, Value: int64(5)}},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.If{
			Cond: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
				Args: []core.CoreExpr{&core.Var{Name: "age"}, &core.Lit{Kind: core.IntLit, Value: int64(65)}},
			},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(5)},
			Else: &core.Lit{Kind: core.IntLit, Value: int64(15)},
		},
	}

	highBody := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
			Args: []core.CoreExpr{&core.Var{Name: "age"}, &core.Lit{Kind: core.IntLit, Value: int64(65)}},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(10)},
		Else: &core.Lit{Kind: core.IntLit, Value: int64(20)},
	}

	body := &core.Match{
		Scrutinee: &core.Var{Name: "season"},
		Arms: []core.MatchArm{
			{Pattern: &core.ConstructorPattern{Name: "LOW_SEASON"}, Body: lowBody},
			{Pattern: &core.ConstructorPattern{Name: "HIGH_SEASON"}, Body: highBody},
		},
	}

	meta := &core.DeclMeta{
		Name:   "admissionFee",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.RequiresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{&core.Var{Name: "age"}, &core.Lit{Kind: core.IntLit, Value: int64(0)}},
				},
			},
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{&core.Var{Name: "result"}, &core.Lit{Kind: core.IntLit, Value: int64(0)}},
				},
			},
		},
	}

	adtTypes := map[string][]ADTVariant{
		"Season": {
			{Name: "LOW_SEASON"},
			{Name: "HIGH_SEASON"},
		},
	}

	result, err := EncodeFunction("admissionFee", params, body, "", meta, adtTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	smtlib := result.SMTLib
	t.Logf("Generated SMT-LIB:\n%s", smtlib)

	// Verify structure
	checks := []struct {
		desc    string
		contain string
	}{
		{"header", "; Verification of admissionFee"},
		{"logic", "(set-logic ALL)"},
		{"season type", "(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))"},
		{"age param", "(declare-const age Int)"},
		{"season param", "(declare-const season Season)"},
		{"precondition", "(assert (>= age 0))"},
		{"result def", "(define-const result Int"},
		{"postcondition negated", "(assert (not (>= result 0)))"},
		{"check-sat", "(check-sat)"},
		{"get-model", "(get-model)"},
	}
	for _, c := range checks {
		if !strings.Contains(smtlib, c.contain) {
			t.Errorf("missing %s: expected %q in output", c.desc, c.contain)
		}
	}
}

func TestStripConstructorPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"make_Season_LOW_SEASON", "LOW_SEASON"},
		{"make_AgeClass_CHILD", "CHILD"},
		{"make_AdmissionDecision_ADMIT", "ADMIT"},
		{"LOW_SEASON", "LOW_SEASON"},     // Already plain
		{"admissionFee", "admissionFee"}, // Not a constructor
		{"make_", "make_"},               // Degenerate
		{"make_X", "make_X"},             // No second underscore
		{"make_X_Y", "Y"},                // Minimal valid
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripConstructorPrefix(tt.input)
			if got != tt.want {
				t.Errorf("stripConstructorPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodeDictApp(t *testing.T) {
	tests := []struct {
		name   string
		method string
		args   []core.CoreExpr
		want   string
	}{
		{
			"ge",
			"ge",
			[]core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
			"(>= x 0)",
		},
		{
			"lt",
			"lt",
			[]core.CoreExpr{
				&core.Var{Name: "age"},
				&core.Lit{Kind: core.IntLit, Value: int64(5)},
			},
			"(< age 5)",
		},
		{
			"add",
			"add",
			[]core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(1)},
			},
			"(+ x 1)",
		},
		{
			"eq",
			"eq",
			[]core.CoreExpr{
				&core.Var{Name: "a"},
				&core.Var{Name: "b"},
			},
			"(= a b)",
		},
		{
			"neq",
			"neq",
			[]core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
			"(distinct x 0)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			da := &core.DictApp{
				Method: tt.method,
				Args:   tt.args,
			}
			got, err := encodeDictApp(da)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("encodeDictApp(%q) = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}

func TestEncodeDictApp_UnsupportedMethod(t *testing.T) {
	da := &core.DictApp{
		Method: "unknown_method",
		Args:   []core.CoreExpr{&core.Var{Name: "x"}},
	}
	_, err := encodeDictApp(da)
	if err == nil {
		t.Error("expected error for unsupported method")
	}
}

func TestEncodeExpr_DictAbs_Transparent(t *testing.T) {
	// DictAbs should transparently encode the body
	expr := &core.DictAbs{
		Body: &core.Var{Name: "x"},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "x" {
		t.Errorf("EncodeExpr(DictAbs) = %q, want %q", got, "x")
	}
}

func TestEncodeFunction_WithReturnSort(t *testing.T) {
	params := []FunctionParam{
		{Name: "age", Type: &types.TCon{Name: "int"}},
	}
	body := &core.If{
		Cond: &core.Lit{Kind: core.BoolLit, Value: true},
		Then: &core.Var{Name: "age"}, // Would infer Int
		Else: &core.Var{Name: "age"},
	}
	meta := &core.DeclMeta{
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}

	// With explicit return sort
	result, err := EncodeFunction("f", params, body, "AgeClass", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.SMTLib, "result AgeClass") {
		t.Error("expected AgeClass sort in result declaration")
	}

	// Without explicit return sort (should infer)
	result2, err := EncodeFunction("f", params, body, "", meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result2.SMTLib, "result Int") {
		t.Errorf("expected Int sort inferred, got: %s", result2.SMTLib)
	}
}

// --- Record expression encoding tests ---

func TestEncodeRecordAccess(t *testing.T) {
	// RecordAccess{Record: Var("p"), Field: "x"} → (x p)
	// RecordAccess doesn't need type info — field name IS the accessor
	setupRecordTestContext()
	defer teardownRecordTestContext()

	ra := &core.RecordAccess{
		Record: &core.Var{Name: "p"},
		Field:  "x",
	}
	got, err := EncodeExpr(ra)
	if err != nil {
		t.Fatalf("encodeRecordAccess unexpected error: %v", err)
	}
	if got != "(x p)" {
		t.Errorf("encodeRecordAccess = %q, want %q", got, "(x p)")
	}
}

func TestEncodeRecordAccess_NestedExpr(t *testing.T) {
	// RecordAccess on a let-bound variable
	setupRecordTestContext()
	defer teardownRecordTestContext()

	ra := &core.RecordAccess{
		Record: &core.Var{Name: "myRecord"},
		Field:  "amount",
	}
	got, err := EncodeExpr(ra)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(amount myRecord)" {
		t.Errorf("got %q, want %q", got, "(amount myRecord)")
	}
}

func TestEncodeRecord_Construction(t *testing.T) {
	// Record{x: 5, y: 10} → (mk_Point 5 10)
	setupRecordTestContext()
	defer teardownRecordTestContext()

	rec := &core.Record{
		Fields: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(5)},
			"y": &core.Lit{Kind: core.IntLit, Value: int64(10)},
		},
	}
	got, err := EncodeExpr(rec)
	if err != nil {
		t.Fatalf("encodeRecord unexpected error: %v", err)
	}
	if got != "(mk_Point 5 10)" {
		t.Errorf("encodeRecord = %q, want %q", got, "(mk_Point 5 10)")
	}
}

func TestEncodeRecordUpdate(t *testing.T) {
	// RecordUpdate{Base: p, Updates: {x: 20}} → (mk_Point 20 (y p))
	setupRecordTestContext()
	defer teardownRecordTestContext()

	ru := &core.RecordUpdate{
		Base: &core.Var{Name: "p"},
		Updates: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(20)},
		},
	}
	got, err := EncodeExpr(ru)
	if err != nil {
		t.Fatalf("encodeRecordUpdate unexpected error: %v", err)
	}
	if got != "(mk_Point 20 (y p))" {
		t.Errorf("encodeRecordUpdate = %q, want %q", got, "(mk_Point 20 (y p))")
	}
}

func TestEncodeRecordUpdate_MultipleFields(t *testing.T) {
	// Update both fields
	setupRecordTestContext()
	defer teardownRecordTestContext()

	ru := &core.RecordUpdate{
		Base: &core.Var{Name: "p"},
		Updates: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(100)},
			"y": &core.Lit{Kind: core.IntLit, Value: int64(200)},
		},
	}
	got, err := EncodeExpr(ru)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(mk_Point 100 200)" {
		t.Errorf("got %q, want %q", got, "(mk_Point 100 200)")
	}
}

func TestEncodeFunction_WithRecordParam(t *testing.T) {
	// Full EncodeFunction with a record parameter
	params := []FunctionParam{
		{Name: "p", Type: &types.TRecord{
			Fields:   map[string]types.Type{"x": &types.TCon{Name: "int"}, "y": &types.TCon{Name: "int"}},
			TypeName: "Point",
		}},
	}
	body := &core.RecordAccess{
		Record: &core.Var{Name: "p"},
		Field:  "x",
	}
	meta := &core.DeclMeta{
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
				Args: []core.CoreExpr{
					&core.Var{Name: "result"},
					&core.Lit{Kind: core.IntLit, Value: int64(0)},
				},
			}, Message: "(result >= 0)"},
		},
	}

	result, err := EncodeFunction("getX", params, body, "Int", meta, nil)
	if err != nil {
		t.Fatalf("EncodeFunction with record param: unexpected error: %v", err)
	}

	// Should contain record type declaration
	if !strings.Contains(result.SMTLib, "declare-datatype Point") {
		t.Errorf("expected Point declaration in SMT-LIB:\n%s", result.SMTLib)
	}
	// Should contain mk_Point constructor
	if !strings.Contains(result.SMTLib, "mk_Point") {
		t.Errorf("expected mk_Point constructor in SMT-LIB:\n%s", result.SMTLib)
	}
	// Should contain field accessors in declaration
	if !strings.Contains(result.SMTLib, "(x Int)") {
		t.Errorf("expected (x Int) accessor in SMT-LIB:\n%s", result.SMTLib)
	}
	// Should declare p as Point
	if !strings.Contains(result.SMTLib, "declare-const p Point") {
		t.Errorf("expected (declare-const p Point) in SMT-LIB:\n%s", result.SMTLib)
	}
	// Body should be (x p)
	if !strings.Contains(result.SMTLib, "(x p)") {
		t.Errorf("expected (x p) in body, SMT-LIB:\n%s", result.SMTLib)
	}
}

// setupRecordTestContext initializes the package-level record context
// with a Point record type {x: Int, y: Int}.
func setupRecordTestContext() {
	activeRecordTypes = map[string]*RecordTypeInfo{
		"Point": {
			SortName:   "Point",
			CtorName:   "mk_Point",
			FieldNames: []string{"x", "y"},
			FieldSorts: map[string]string{"x": "Int", "y": "Int"},
		},
	}
	activeFieldSetToSort = map[string]string{
		"x,y": "Point",
	}
}

func teardownRecordTestContext() {
	activeRecordTypes = nil
	activeFieldSetToSort = nil
}

// --- String builtin encoding tests ---

func TestEncodeStringBuiltin_ConcatString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "concat_String"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "a"},
			&core.Var{Name: "b"},
		},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.++ a b)" {
		t.Errorf("got %q, want %q", got, "(str.++ a b)")
	}
}

func TestEncodeStringBuiltin_EqString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(= a b)" {
		t.Errorf("got %q, want %q", got, "(= a b)")
	}
}

func TestEncodeStringBuiltin_LtString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "lt_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.< a b)" {
		t.Errorf("got %q, want %q", got, "(str.< a b)")
	}
}

func TestEncodeStringBuiltin_GtString_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "gt_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// gt(a,b) → (str.< b a) with flipped args
	if got != "(str.< b a)" {
		t.Errorf("got %q, want %q", got, "(str.< b a)")
	}
}

func TestEncodeStringBuiltin_StrLen(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_len"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.len s)" {
		t.Errorf("got %q, want %q", got, "(str.len s)")
	}
}

func TestEncodeStringBuiltin_StrFind_AppendsZero(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_find"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "t"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_find(s,t) → (str.indexof s t 0)
	if got != "(str.indexof s t 0)" {
		t.Errorf("got %q, want %q", got, "(str.indexof s t 0)")
	}
}

func TestEncodeStringBuiltin_StrSlice_SubstrMode(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_slice"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "s"},
			&core.Var{Name: "start"},
			&core.Var{Name: "end"},
		},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_slice(s, start, end) → (str.substr s start (- end start))
	want := "(str.substr s start (- end start))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeStringBuiltin_StartsWith_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_startsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "prefix"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_startsWith(s, prefix) → (str.prefixof prefix s)
	if got != "(str.prefixof prefix s)" {
		t.Errorf("got %q, want %q", got, "(str.prefixof prefix s)")
	}
}

func TestEncodeStringBuiltin_EndsWith_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_endsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "suffix"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_endsWith(s, suffix) → (str.suffixof suffix s)
	if got != "(str.suffixof suffix s)" {
		t.Errorf("got %q, want %q", got, "(str.suffixof suffix s)")
	}
}

func TestEncodeIntrinsic_OpConcat(t *testing.T) {
	intr := &core.Intrinsic{
		Op:   core.OpConcat,
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(intr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.++ a b)" {
		t.Errorf("got %q, want %q", got, "(str.++ a b)")
	}
}
