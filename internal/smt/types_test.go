package smt

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/types"
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
		{"string", &types.TCon{Name: "string"}, "", true},
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
	fn := &types.TFunc{
		Params: []types.Type{&types.TCon{Name: "int"}},
		Return: &types.TCon{Name: "int"},
	}
	_, err := MapType(fn)
	if err == nil {
		t.Error("MapType(TFunc) expected error")
	}
}

func TestMapType_List(t *testing.T) {
	lst := &types.TList{Element: &types.TCon{Name: "int"}}
	_, err := MapType(lst)
	if err == nil {
		t.Error("MapType(TList) expected error")
	}
}

func TestMapType_Record(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{"x": &types.TCon{Name: "int"}},
	}
	_, err := MapType(rec)
	if err == nil {
		t.Error("MapType(TRecord) expected error")
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
