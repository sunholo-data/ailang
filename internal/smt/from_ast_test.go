package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TestAstTypeEncodable covers the M-SMT-CALLEE-SORT-GATE encodability predicate:
// primitives, lists, records, monomorphic ADTs and parametric ADTs at a concrete
// instance are encodable; type variables (also as an ADT argument), a parametric
// ADT used bare or at the wrong arity, and undeclared names are not.
func TestAstTypeEncodable(t *testing.T) {
	declarable := map[string]int{"Region": 0, "Grade": 0, "Option": 1, "Result": 2, "Box": 1}
	app := func(c string, args ...ast.Type) ast.Type { return &ast.TypeApp{Constructor: c, Args: args} }
	simple := func(n string) ast.Type { return &ast.SimpleType{Name: n} }

	tests := []struct {
		name string
		typ  ast.Type
		want bool
	}{
		{"int", &ast.SimpleType{Name: "int"}, true},
		{"float", &ast.SimpleType{Name: "float"}, true},
		{"bool", &ast.SimpleType{Name: "bool"}, true},
		{"string", &ast.SimpleType{Name: "string"}, true},
		{"monomorphic ADT declared", &ast.SimpleType{Name: "Region"}, true},
		{"unknown bare name", &ast.SimpleType{Name: "Widget"}, false},
		{"nil type", nil, true},
		{"list of int", &ast.ListType{Element: &ast.SimpleType{Name: "int"}}, true},
		{"list of ADT", &ast.ListType{Element: &ast.SimpleType{Name: "Region"}}, true},
		{"list of Option[int]", &ast.ListType{Element: app("Option", simple("int"))}, true},
		{"list[] typeapp", &ast.TypeApp{Constructor: "list", Args: []ast.Type{&ast.SimpleType{Name: "int"}}}, true},
		{"Option[float]", app("Option", simple("float")), true},
		{"Result[float, string]", app("Result", simple("float"), simple("string")), true},
		{"Result[int, Region]", app("Result", simple("int"), simple("Region")), true},
		{"Option[list[int]]", app("Option", app("list", simple("int"))), true},
		{"user Box[int]", app("Box", simple("int")), true},
		{"Option[a]", app("Option", &ast.TypeVar{Name: "a"}), false},
		{"Option[Widget] undeclared arg", app("Option", simple("Widget")), false},
		{"Result[(), string] tuple arg", app("Result", &ast.TupleType{}, simple("string")), false},
		{"Box bare (parametric, no args)", simple("Box"), false},
		{"Box[int, int] wrong arity", app("Box", simple("int"), simple("int")), false},
		{"Widget[int] undeclared", app("Widget", simple("int")), false},
		{"type variable", &ast.TypeVar{Name: "a"}, false},
		{"record", &ast.RecordType{}, true},
		{"labelled wraps base", &ast.LabelledType{Base: &ast.SimpleType{Name: "int"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := astTypeEncodable(tt.typ, declarable); got != tt.want {
				t.Fatalf("astTypeEncodable(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestCollectDeclarableTypes verifies every declared type maps to its number of
// type parameters, with Option and Result built in.
func TestCollectDeclarableTypes(t *testing.T) {
	file := &ast.File{
		Decls: []ast.Node{
			&ast.TypeDecl{Name: "Region", TypeParams: nil},
			&ast.TypeDecl{Name: "Grade", TypeParams: []string{}},
			&ast.TypeDecl{Name: "Box", TypeParams: []string{"a"}},
		},
	}
	got := CollectDeclarableTypes([]*ast.File{file, nil})
	want := map[string]int{"Region": 0, "Grade": 0, "Box": 1, "Option": 1, "Result": 2}
	for name, n := range want {
		if m, ok := got[name]; !ok || m != n {
			t.Errorf("%s: got (%d, %v), want %d", name, m, ok, n)
		}
	}
}

// TestDescribeASTType checks the diagnostic renderer names parametric applications
// with their arguments (so the skip reason is actionable).
func TestDescribeASTType(t *testing.T) {
	tests := []struct {
		typ  ast.Type
		want string
	}{
		{&ast.SimpleType{Name: "float"}, "float"},
		{&ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "float"}}}, "Option[float]"},
		{&ast.TypeApp{Constructor: "Result", Args: []ast.Type{&ast.SimpleType{Name: "float"}, &ast.SimpleType{Name: "string"}}}, "Result[float, string]"},
		{&ast.ListType{Element: &ast.SimpleType{Name: "int"}}, "[int]"},
		{&ast.TypeVar{Name: "a"}, "a"},
		// Tuples were rendered with %T ("*ast.TupleType") in user-facing skip reasons.
		{&ast.TupleType{Elements: []ast.Type{&ast.SimpleType{Name: "int"}, &ast.SimpleType{Name: "string"}}}, "(int, string)"},
	}
	for _, tt := range tests {
		if got := describeASTType(tt.typ); got != tt.want {
			t.Fatalf("describeASTType = %q, want %q", got, tt.want)
		}
	}
}
