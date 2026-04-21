package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func TestMapType_Primitives(t *testing.T) {
	tests := []struct {
		name    string
		typ     types.Type
		want    string
		wantErr bool
	}{
		{"int", &types.TCon{Name: "int"}, "Int", false},
		{"float", &types.TCon{Name: "float"}, "Real", false},
		{"bool", &types.TCon{Name: "bool"}, "Bool", false},
		{"string", &types.TCon{Name: "string"}, "String", false},
		{"unit", &types.TCon{Name: "unit"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapType(tt.typ)
			if tt.wantErr {
				if err == nil {
					t.Errorf("MapType(%s) expected error, got %q", tt.name, got)
				}
				return
			}
			if err != nil {
				t.Errorf("MapType(%s) unexpected error: %v", tt.name, err)
				return
			}
			if got != tt.want {
				t.Errorf("MapType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestMapType_ADT(t *testing.T) {
	// ADT types show up as TCon with the type name
	season := &types.TCon{Name: "Season"}
	got, err := MapType(season)
	if err != nil {
		t.Errorf("MapType(Season) unexpected error: %v", err)
	}
	if got != "Season" {
		t.Errorf("MapType(Season) = %q, want %q", got, "Season")
	}
}

func TestMapType_TypeVar(t *testing.T) {
	tv := &types.TVar{Name: "a"}
	_, err := MapType(tv)
	if err == nil {
		t.Error("MapType(TVar) expected error")
	}
}

func TestMapType_Function(t *testing.T) {
	fn := &types.TFunc2{
		Params: []types.Type{&types.TCon{Name: "int"}},
		Return: &types.TCon{Name: "int"},
	}
	_, err := MapType(fn)
	if err == nil {
		t.Error("MapType(TFunc2) expected error")
	}
}

func TestMapType_List(t *testing.T) {
	// TList{int} → (Seq Int)
	lst := &types.TList{Element: &types.TCon{Name: "int"}}
	got, err := MapType(lst)
	if err != nil {
		t.Fatalf("MapType(TList{int}) unexpected error: %v", err)
	}
	if got != "(Seq Int)" {
		t.Errorf("MapType(TList{int}) = %q, want %q", got, "(Seq Int)")
	}

	// TApp(list, string) → (Seq String)
	app := &types.TApp{
		Constructor: &types.TCon{Name: "list"},
		Args:        []types.Type{&types.TCon{Name: "string"}},
	}
	got, err = MapType(app)
	if err != nil {
		t.Fatalf("MapType(TApp(list, string)) unexpected error: %v", err)
	}
	if got != "(Seq String)" {
		t.Errorf("MapType(TApp(list, string)) = %q, want %q", got, "(Seq String)")
	}

	// Nested: TList{TList{int}} → (Seq (Seq Int))
	nested := &types.TList{Element: &types.TList{Element: &types.TCon{Name: "int"}}}
	got, err = MapType(nested)
	if err != nil {
		t.Fatalf("MapType(nested list) unexpected error: %v", err)
	}
	if got != "(Seq (Seq Int))" {
		t.Errorf("MapType(nested list) = %q, want %q", got, "(Seq (Seq Int))")
	}
}

func TestMapType_Record(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{"x": &types.TCon{Name: "int"}},
	}
	got, err := MapType(rec)
	if err != nil {
		t.Fatalf("MapType(TRecord) unexpected error: %v", err)
	}
	if got != "Record_x" {
		t.Errorf("MapType(TRecord{x:int}) = %q, want %q", got, "Record_x")
	}
}

func TestMapType_Nil(t *testing.T) {
	_, err := MapType(nil)
	if err == nil {
		t.Error("MapType(nil) expected error")
	}
}

func TestDeclareEnumDatatype(t *testing.T) {
	got := DeclareEnumDatatype("Season", []string{"LOW_SEASON", "HIGH_SEASON"})
	want := "(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))"
	if got != want {
		t.Errorf("DeclareEnumDatatype =\n  %s\nwant:\n  %s", got, want)
	}
}

func TestDeclareDatatype_WithFields(t *testing.T) {
	got := DeclareDatatype("Shape", []ADTVariant{
		{Name: "Circle", Fields: []ADTField{{Name: "radius", Sort: "Int"}}},
		{Name: "Rect", Fields: []ADTField{
			{Name: "width", Sort: "Int"},
			{Name: "height", Sort: "Int"},
		}},
	})
	want := "(declare-datatype Shape ((Circle (radius Int)) (Rect (width Int) (height Int))))"
	if got != want {
		t.Errorf("DeclareDatatype =\n  %s\nwant:\n  %s", got, want)
	}
}

func TestDeclareDatatype_Mixed(t *testing.T) {
	// Option-like: Some(value) | None
	got := DeclareDatatype("MyOption", []ADTVariant{
		{Name: "MySome", Fields: []ADTField{{Name: "val", Sort: "Int"}}},
		{Name: "MyNone"},
	})
	want := "(declare-datatype MyOption ((MySome (val Int)) (MyNone)))"
	if got != want {
		t.Errorf("DeclareDatatype =\n  %s\nwant:\n  %s", got, want)
	}
}

func TestDeclareConst(t *testing.T) {
	got := DeclareConst("x", "Int")
	if got != "(declare-const x Int)" {
		t.Errorf("DeclareConst = %q", got)
	}
}

func TestAssert(t *testing.T) {
	got := Assert("(>= x 0)")
	if got != "(assert (>= x 0))" {
		t.Errorf("Assert = %q", got)
	}
}

func TestAssertNot(t *testing.T) {
	got := AssertNot("(>= result 0)")
	if got != "(assert (not (>= result 0)))" {
		t.Errorf("AssertNot = %q", got)
	}
}

func TestCheckSat(t *testing.T) {
	if CheckSat() != "(check-sat)" {
		t.Error("CheckSat mismatch")
	}
}

func TestGetModel(t *testing.T) {
	if GetModel() != "(get-model)" {
		t.Error("GetModel mismatch")
	}
}

func TestBuiltinToSMTOp_Coverage(t *testing.T) {
	// Verify all expected builtins are mapped
	expected := []string{
		"add_Int", "sub_Int", "mul_Int", "div_Int", "mod_Int",
		"add_Float", "sub_Float", "mul_Float", "div_Float", "mod_Float",
		"eq_Int", "ne_Int", "lt_Int", "le_Int", "gt_Int", "ge_Int",
		"eq_Float", "ne_Float", "lt_Float", "le_Float", "gt_Float", "ge_Float",
		"eq_Bool", "ne_Bool",
		"and_Bool", "or_Bool", "not_Bool",
		"neg_Int", "neg_Float",
	}
	for _, name := range expected {
		if _, ok := BuiltinToSMTOp[name]; !ok {
			t.Errorf("missing builtin mapping for %q", name)
		}
	}
}

func TestBuiltinToSMTOp_Values(t *testing.T) {
	tests := []struct {
		builtin string
		want    string
	}{
		{"add_Int", "+"},
		{"sub_Int", "-"},
		{"mul_Int", "*"},
		{"div_Int", "div"},
		{"mod_Int", "mod"},
		{"eq_Int", "="},
		{"ne_Int", "distinct"},
		{"lt_Int", "<"},
		{"le_Int", "<="},
		{"gt_Int", ">"},
		{"ge_Int", ">="},
		{"and_Bool", "and"},
		{"or_Bool", "or"},
		{"not_Bool", "not"},
		{"neg_Int", "-"},
	}
	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			got := BuiltinToSMTOp[tt.builtin]
			if got != tt.want {
				t.Errorf("BuiltinToSMTOp[%q] = %q, want %q", tt.builtin, got, tt.want)
			}
		})
	}
}

func TestNewSMTContext(t *testing.T) {
	ctx := NewSMTContext()
	if ctx == nil {
		t.Fatal("NewSMTContext returned nil")
	}
	if ctx.Variables == nil {
		t.Error("Variables map is nil")
	}
	if ctx.DeclaredTypes == nil {
		t.Error("DeclaredTypes map is nil")
	}
	if len(ctx.Declarations) != 0 {
		t.Error("Declarations should be empty")
	}
}

func TestMapType_TApp_WithTCon(t *testing.T) {
	// TApp with a TCon constructor represents parameterized ADT
	app := &types.TApp{
		Constructor: &types.TCon{Name: "Option"},
		Args:        []types.Type{&types.TCon{Name: "int"}},
	}
	got, err := MapType(app)
	if err != nil {
		t.Errorf("MapType(TApp(Option, int)) unexpected error: %v", err)
	}
	if got != "Option" {
		t.Errorf("MapType(TApp(Option, int)) = %q, want %q", got, "Option")
	}
}

func TestMapType_TApp_WithTVar(t *testing.T) {
	// TApp with a TVar constructor should fail
	app := &types.TApp{
		Constructor: &types.TVar{Name: "a"},
		Args:        []types.Type{&types.TCon{Name: "int"}},
	}
	_, err := MapType(app)
	if err == nil {
		t.Error("MapType(TApp(TVar, int)) expected error")
	}
	if !strings.Contains(err.Error(), "parameterized type") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Record type tests ---

func TestMapType_Record_Named(t *testing.T) {
	rec := &types.TRecord{
		Fields:   map[string]types.Type{"x": &types.TCon{Name: "int"}, "y": &types.TCon{Name: "int"}},
		TypeName: "Point",
	}
	got, err := MapType(rec)
	if err != nil {
		t.Fatalf("MapType(named record) unexpected error: %v", err)
	}
	if got != "Point" {
		t.Errorf("MapType(named record) = %q, want %q", got, "Point")
	}
}

func TestMapType_Record_Anonymous(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{"x": &types.TCon{Name: "int"}, "y": &types.TCon{Name: "float"}},
	}
	got, err := MapType(rec)
	if err != nil {
		t.Fatalf("MapType(anonymous record) unexpected error: %v", err)
	}
	// Anonymous records get deterministic name from sorted fields
	if got != "Record_x_y" {
		t.Errorf("MapType(anonymous record) = %q, want %q", got, "Record_x_y")
	}
}

func TestMapRecordSortName_Named(t *testing.T) {
	rec := &types.TRecord{
		Fields:   map[string]types.Type{"name": &types.TCon{Name: "string"}},
		TypeName: "Person",
	}
	if got := MapRecordSortName(rec); got != "Person" {
		t.Errorf("MapRecordSortName(Person) = %q, want %q", got, "Person")
	}
}

func TestMapRecordSortName_Anonymous_DeterministicOrder(t *testing.T) {
	// Fields added in different order should produce the same name
	rec1 := &types.TRecord{
		Fields: map[string]types.Type{"b": &types.TCon{Name: "int"}, "a": &types.TCon{Name: "int"}},
	}
	rec2 := &types.TRecord{
		Fields: map[string]types.Type{"a": &types.TCon{Name: "int"}, "b": &types.TCon{Name: "int"}},
	}
	if MapRecordSortName(rec1) != MapRecordSortName(rec2) {
		t.Errorf("expected same sort name regardless of field insertion order: %q vs %q",
			MapRecordSortName(rec1), MapRecordSortName(rec2))
	}
}

func TestMapRecordFields(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{
			"x":      &types.TCon{Name: "int"},
			"y":      &types.TCon{Name: "float"},
			"active": &types.TCon{Name: "bool"},
		},
	}
	fields, err := MapRecordFields(rec)
	if err != nil {
		t.Fatalf("MapRecordFields unexpected error: %v", err)
	}
	expected := map[string]string{"x": "Int", "y": "Real", "active": "Bool"}
	for k, v := range expected {
		if fields[k] != v {
			t.Errorf("field %q: got %q, want %q", k, fields[k], v)
		}
	}
}

func TestMapRecordFields_StringFieldSupported(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{
			"x":    &types.TCon{Name: "int"},
			"name": &types.TCon{Name: "string"},
		},
	}
	result, err := MapRecordFields(rec)
	if err != nil {
		t.Fatalf("unexpected error for record with string field: %v", err)
	}
	if result["name"] != "String" {
		t.Errorf("expected String sort for 'name' field, got %q", result["name"])
	}
	if result["x"] != "Int" {
		t.Errorf("expected Int sort for 'x' field, got %q", result["x"])
	}
}

func TestMapRecordFields_UnsupportedFieldType(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{
			"x":  &types.TCon{Name: "int"},
			"fn": &types.TFunc2{}, // function type not supported
		},
	}
	_, err := MapRecordFields(rec)
	if err == nil {
		t.Fatal("expected error for record with function field")
	}
	if !strings.Contains(err.Error(), "fn") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestDeclareRecordDatatype_Simple(t *testing.T) {
	got := DeclareRecordDatatype("Point", map[string]string{"x": "Int", "y": "Int"})
	want := "(declare-datatype Point ((mk_Point (x Int) (y Int))))"
	if got != want {
		t.Errorf("DeclareRecordDatatype:\n  got:  %s\n  want: %s", got, want)
	}
}

func TestDeclareRecordDatatype_AlphabeticalOrder(t *testing.T) {
	got := DeclareRecordDatatype("Config", map[string]string{
		"z_flag":  "Bool",
		"a_count": "Int",
		"m_value": "Real",
	})
	// Fields should be sorted: a_count, m_value, z_flag
	want := "(declare-datatype Config ((mk_Config (a_count Int) (m_value Real) (z_flag Bool))))"
	if got != want {
		t.Errorf("DeclareRecordDatatype alphabetical:\n  got:  %s\n  want: %s", got, want)
	}
}

func TestDeclareRecordDatatype_WithADTField(t *testing.T) {
	got := DeclareRecordDatatype("Order", map[string]string{
		"amount": "Int",
		"status": "OrderStatus",
	})
	want := "(declare-datatype Order ((mk_Order (amount Int) (status OrderStatus))))"
	if got != want {
		t.Errorf("DeclareRecordDatatype with ADT field:\n  got:  %s\n  want: %s", got, want)
	}
}

func TestRecordConstructorName(t *testing.T) {
	tests := []struct {
		sort string
		want string
	}{
		{"Point", "mk_Point"},
		{"Record_x_y", "mk_Record_x_y"},
		{"Config", "mk_Config"},
	}
	for _, tt := range tests {
		if got := RecordConstructorName(tt.sort); got != tt.want {
			t.Errorf("RecordConstructorName(%q) = %q, want %q", tt.sort, got, tt.want)
		}
	}
}
