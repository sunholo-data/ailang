package smt

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestIsSMTEncodable_NoContracts(t *testing.T) {
	meta := &core.DeclMeta{
		Name:      "foo",
		IsPure:    true,
		Contracts: nil,
	}
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}

	ok, reasons := IsSMTEncodable("foo", meta, body)
	if ok {
		t.Fatal("expected not encodable (no contracts)")
	}
	if len(reasons) == 0 {
		t.Fatal("expected rejection reasons")
	}
	if reasons[0].Code != RejectNoContracts {
		t.Errorf("expected RejectNoContracts, got %s", reasons[0].Code)
	}
}

func TestIsSMTEncodable_NotPure(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "foo",
		IsPure: false,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}

	ok, reasons := IsSMTEncodable("foo", meta, body)
	if ok {
		t.Fatal("expected not encodable (not pure)")
	}
	found := false
	for _, r := range reasons {
		if r.Code == RejectNotPure {
			found = true
		}
	}
	if !found {
		t.Error("expected RejectNotPure in reasons")
	}
}

func TestIsSMTEncodable_Recursive(t *testing.T) {
	// Body references function name "factorial"
	meta := &core.DeclMeta{
		Name:   "factorial",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.App{
		Func: &core.Var{Name: "factorial"},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}

	ok, reasons := IsSMTEncodable("factorial", meta, body)
	if ok {
		t.Fatal("expected not encodable (recursive)")
	}
	found := false
	for _, r := range reasons {
		if r.Code == RejectRecursive {
			found = true
		}
	}
	if !found {
		t.Error("expected RejectRecursive in reasons")
	}
}

func TestIsSMTEncodable_HigherOrder(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "apply",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	// Lambda passed as argument to App
	body := &core.App{
		Func: &core.Var{Name: "map"},
		Args: []core.CoreExpr{
			&core.Lambda{
				Params: []string{"x"},
				Body:   &core.Var{Name: "x"},
			},
		},
	}

	ok, reasons := IsSMTEncodable("apply", meta, body)
	if ok {
		t.Fatal("expected not encodable (higher order)")
	}
	found := false
	for _, r := range reasons {
		if r.Code == RejectHigherOrder {
			found = true
		}
	}
	if !found {
		t.Error("expected RejectHigherOrder in reasons")
	}
}

func TestIsSMTEncodable_DeepPatterns(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "deep",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	// Match with nested constructor pattern: Some(Some(x))
	body := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Some",
					Args: []core.CorePattern{
						&core.ConstructorPattern{
							Name: "Some",
							Args: []core.CorePattern{
								&core.VarPattern{Name: "inner"},
							},
						},
					},
				},
				Body: &core.Var{Name: "inner"},
			},
		},
	}

	ok, reasons := IsSMTEncodable("deep", meta, body)
	if ok {
		t.Fatal("expected not encodable (deep patterns)")
	}
	found := false
	for _, r := range reasons {
		if r.Code == RejectDeepPatterns {
			found = true
		}
	}
	if !found {
		t.Error("expected RejectDeepPatterns in reasons")
	}
}

func TestIsSMTEncodable_StringLiteral_NowEncodable(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "greet",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.Lit{Kind: core.StringLit, Value: "hello"}

	ok, reasons := IsSMTEncodable("greet", meta, body)
	if !ok {
		t.Fatalf("string literals should now be encodable (M-SMT-STRINGS), but got reasons: %v", reasons)
	}
}

func TestIsSMTEncodable_UnencodableTypes_List(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "listFn",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.List{
		Elements: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}

	ok, reasons := IsSMTEncodable("listFn", meta, body)
	if ok {
		t.Fatal("expected not encodable (list)")
	}
	found := false
	for _, r := range reasons {
		if r.Code == RejectUnencodable {
			found = true
		}
	}
	if !found {
		t.Error("expected RejectUnencodable in reasons")
	}
}

func TestIsSMTEncodable_Record_WithIntFields(t *testing.T) {
	// Records with encodable field types (int, bool, etc.) are now allowed
	meta := &core.DeclMeta{
		Name:   "recFn",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.Record{
		Fields: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(1)},
		},
	}

	ok, _ := IsSMTEncodable("recFn", meta, body)
	if !ok {
		t.Fatal("expected encodable — records with int fields should be accepted")
	}
}

func TestIsSMTEncodable_Record_WithStringField_NowSupported(t *testing.T) {
	// Records with string fields are now encodable (M-SMT-STRINGS)
	meta := &core.DeclMeta{
		Name:   "recFn",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.Record{
		Fields: map[string]core.CoreExpr{
			"name": &core.Lit{Kind: core.StringLit, Value: "hello"},
		},
	}

	ok, reasons := IsSMTEncodable("recFn", meta, body)
	if !ok {
		t.Fatalf("record with string field should now be encodable, but got reasons: %v", reasons)
	}
}

func TestIsSMTEncodable_RecordAccess(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "accessFn",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.RecordAccess{
		Record: &core.Var{Name: "p"},
		Field:  "x",
	}

	ok, _ := IsSMTEncodable("accessFn", meta, body)
	if !ok {
		t.Fatal("expected encodable — record access on variable should be accepted")
	}
}

func TestIsSMTEncodable_RecordUpdate(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "updateFn",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	body := &core.RecordUpdate{
		Base: &core.Var{Name: "p"},
		Updates: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(42)},
		},
	}

	ok, _ := IsSMTEncodable("updateFn", meta, body)
	if !ok {
		t.Fatal("expected encodable — record update with int should be accepted")
	}
}

// Test the happy path: a simple pure function with contracts.
func TestIsSMTEncodable_ValidFunction(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "absolute",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	// Simple body: if x >= 0 then x else -x (using post-lowered builtins)
	body := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
			Args: []core.CoreExpr{
				&core.Var{Name: "x"},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: &core.Var{Name: "x"},
		Else: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "neg_Int"}},
			Args: []core.CoreExpr{&core.Var{Name: "x"}},
		},
	}

	ok, reasons := IsSMTEncodable("absolute", meta, body)
	if !ok {
		t.Errorf("expected encodable, got rejection reasons: %v", reasons)
	}
}

// Test with match over enum (shallow pattern — valid).
func TestIsSMTEncodable_EnumMatch(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "admissionFee",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	// match season { LOW_SEASON => 15, HIGH_SEASON => 20 }
	body := &core.Match{
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

	ok, reasons := IsSMTEncodable("admissionFee", meta, body)
	if !ok {
		t.Errorf("expected encodable, got rejection reasons: %v", reasons)
	}
}

// Test with constructor pattern with one field (shallow — valid).
func TestIsSMTEncodable_ShallowADTPattern(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "unwrap",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	// match x { Some(v) => v, None => 0 }
	body := &core.Match{
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

	ok, reasons := IsSMTEncodable("unwrap", meta, body)
	if !ok {
		t.Errorf("expected encodable, got rejection reasons: %v", reasons)
	}
}

func TestIsSMTEncodable_NilMeta(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	ok, reasons := IsSMTEncodable("foo", nil, body)
	if ok {
		t.Fatal("expected not encodable with nil meta")
	}
	// Should reject for no contracts and not pure
	if len(reasons) < 2 {
		t.Errorf("expected at least 2 rejection reasons, got %d", len(reasons))
	}
}

func TestIsSMTEncodable_NilBody(t *testing.T) {
	meta := &core.DeclMeta{
		Name:   "empty",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	ok, _ := IsSMTEncodable("empty", meta, nil)
	if !ok {
		t.Error("nil body should be encodable (no structural issues)")
	}
}

func TestContainsRef_ShadowedByLet(t *testing.T) {
	// let x = 5 in x  — the inner x is NOT a reference to function "x"
	body := &core.Let{
		Name:  "x",
		Value: &core.Lit{Kind: core.IntLit, Value: int64(5)},
		Body:  &core.Var{Name: "x"},
	}
	if containsRef(body, "x") {
		t.Error("let-shadowed name should not count as self-reference")
	}
}

func TestContainsRef_ShadowedByLambda(t *testing.T) {
	body := &core.Lambda{
		Params: []string{"f"},
		Body:   &core.Var{Name: "f"},
	}
	if containsRef(body, "f") {
		t.Error("lambda-shadowed name should not count as self-reference")
	}
}

func TestPatternDepth(t *testing.T) {
	tests := []struct {
		name    string
		pattern core.CorePattern
		want    int
	}{
		{"var", &core.VarPattern{Name: "x"}, 0},
		{"wildcard", &core.WildcardPattern{}, 0},
		{"lit", &core.LitPattern{Value: 42}, 0},
		{"nullary ctor", &core.ConstructorPattern{Name: "None"}, 1},
		{"ctor with var", &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "x"}}}, 1},
		{"nested ctor", &core.ConstructorPattern{
			Name: "Some",
			Args: []core.CorePattern{
				&core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "x"}}},
			},
		}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patternDepth(tt.pattern)
			if got != tt.want {
				t.Errorf("patternDepth(%s) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsStringOrListBuiltin(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"concat_String", false},   // now supported (M-SMT-STRINGS)
		{"eq_String", false},       // now supported
		{"lt_String", false},       // now supported
		{"_str_len", false},        // now supported
		{"_str_find", false},       // now supported
		{"_str_startsWith", false}, // now supported
		{"_str_trim", true},        // unsupported (no Z3 equivalent)
		{"_str_upper", true},       // unsupported
		{"_str_lower", true},       // unsupported
		{"_str_split", true},       // unsupported (returns list)
		{"_str_chars", true},       // unsupported (returns list)
		{"concat_List", true},      // list always unsupported
		{"add_Int", false},
		{"ge_Int", false},
		{"and_Bool", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStringOrListBuiltin(tt.name)
			if got != tt.want {
				t.Errorf("isStringOrListBuiltin(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestSMTRejectionReason_String(t *testing.T) {
	r := SMTRejectionReason{
		Code:    RejectRecursive,
		Message: "Function is recursive",
		Hint:    "Use iteration instead",
	}
	s := r.String()
	if s != "[RECURSIVE] Function is recursive (Use iteration instead)" {
		t.Errorf("unexpected string: %s", s)
	}

	r2 := SMTRejectionReason{
		Code:    RejectNoContracts,
		Message: "No contracts",
	}
	s2 := r2.String()
	if s2 != "[NO_CONTRACTS] No contracts" {
		t.Errorf("unexpected string: %s", s2)
	}
}
