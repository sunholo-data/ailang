package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
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
		Cond: &core.Lit{Kind: core.BoolLit, Value: true},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Else: &core.Lit{Kind: core.IntLit, Value: int64(0)},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(ite true 1 0)" {
		t.Errorf("got %q, want %q", got, "(ite true 1 0)")
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
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
			&core.Var{Name: "y"},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(+ x y)" {
		t.Errorf("got %q, want %q", got, "(+ x y)")
	}
}

func TestEncodeExpr_BuiltinOp_Unary(t *testing.T) {
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "not_Bool"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "x"},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(not x)" {
		t.Errorf("got %q, want %q", got, "(not x)")
	}
}

func TestEncodeExpr_CurriedBuiltin(t *testing.T) {
	// Curried form: App(App($builtin.add_Int, [x]), [y])
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
	expr := &core.Match{
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
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(match season ((LOW_SEASON 5) (HIGH_SEASON 10)))"
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
	expr := &core.Intrinsic{
		Op:   core.OpAdd,
		Args: []core.CoreExpr{&core.Var{Name: "x"}, &core.Var{Name: "y"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(+ x y)" {
		t.Errorf("got %q, want %q", got, "(+ x y)")
	}
}

func TestEncodeExpr_PreLowered_BinOp(t *testing.T) {
	expr := &core.Intrinsic{
		Op:   core.OpMul,
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Lit{Kind: core.IntLit, Value: int64(2)}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(* a 2)" {
		t.Errorf("got %q, want %q", got, "(* a 2)")
	}
}

func TestEncodeExpr_PreLowered_UnOp(t *testing.T) {
	expr := &core.Intrinsic{
		Op:   core.OpNot,
		Args: []core.CoreExpr{&core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(not b)" {
		t.Errorf("got %q, want %q", got, "(not b)")
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
	// Constructor application: make_Season_LOW_SEASON → LOW_SEASON
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$adt", Name: "make_Season_LOW_SEASON"}},
		Args: []core.CoreExpr{},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "LOW_SEASON" {
		t.Errorf("got %q, want %q", got, "LOW_SEASON")
	}
}

func TestEncodeExpr_NestedIfWithBuiltins(t *testing.T) {
	expr := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "lt_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "neg_Int"}},
			Args: []core.CoreExpr{&core.Var{Name: "x"}},
		},
		Else: &core.Var{Name: "x"},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(ite (< x 0) (- x) x)"
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
		{Name: "f", Type: &types.TFunc2{}},
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
