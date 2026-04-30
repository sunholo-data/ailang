package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// --- EncodeExpr extended builtin/unary/inferResultSort tests ---

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
		{"age param", "(declare-const $p_age Int)"},
		{"season param", "(declare-const $p_season Season)"},
		{"precondition", "(assert (>= $p_age 0))"},
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

	if !strings.Contains(result.SMTLib, "declare-datatype Point") {
		t.Errorf("expected Point declaration in SMT-LIB:\n%s", result.SMTLib)
	}
	if !strings.Contains(result.SMTLib, "mk_Point") {
		t.Errorf("expected mk_Point constructor in SMT-LIB:\n%s", result.SMTLib)
	}
	if !strings.Contains(result.SMTLib, "(x Int)") {
		t.Errorf("expected (x Int) accessor in SMT-LIB:\n%s", result.SMTLib)
	}
	if !strings.Contains(result.SMTLib, "declare-const $p_p Point") {
		t.Errorf("expected (declare-const $p_p Point) in SMT-LIB:\n%s", result.SMTLib)
	}
	if !strings.Contains(result.SMTLib, "(x $p_p)") {
		t.Errorf("expected (x $p_p) in body, SMT-LIB:\n%s", result.SMTLib)
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
