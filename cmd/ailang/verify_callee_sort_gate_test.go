package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TestAstTypeEncodable covers the M-SMT-CALLEE-SORT-GATE encodability predicate:
// primitives, lists, records, and monomorphic ADTs are encodable; parametric ADT
// applications (Option[float], Result[e,a]) and bare type variables are not.
func TestAstTypeEncodable(t *testing.T) {
	declarable := map[string]bool{"Region": true, "Grade": true}

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
		{"list of parametric", &ast.ListType{Element: &ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "int"}}}}, false},
		{"list[] typeapp", &ast.TypeApp{Constructor: "list", Args: []ast.Type{&ast.SimpleType{Name: "int"}}}, true},
		{"Option[float] parametric", &ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "float"}}}, false},
		{"Result[float,string] parametric", &ast.TypeApp{Constructor: "Result", Args: []ast.Type{&ast.SimpleType{Name: "float"}, &ast.SimpleType{Name: "string"}}}, false},
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

// TestCollectMonomorphicTypeNames verifies only type declarations WITHOUT type
// parameters are treated as declarable SMT sorts. A parametric type like Box[a]
// must be excluded — the encoder cannot monomorphize it.
func TestCollectMonomorphicTypeNames(t *testing.T) {
	file := &ast.File{
		Decls: []ast.Node{
			&ast.TypeDecl{Name: "Region", TypeParams: nil},
			&ast.TypeDecl{Name: "Grade", TypeParams: []string{}},
			&ast.TypeDecl{Name: "Box", TypeParams: []string{"a"}},
		},
	}
	got := collectMonomorphicTypeNames([]*ast.File{file, nil})
	if !got["Region"] || !got["Grade"] {
		t.Fatalf("expected monomorphic types Region, Grade in %v", got)
	}
	if got["Box"] {
		t.Fatalf("parametric type Box[a] must NOT be declarable, got %v", got)
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
	}
	for _, tt := range tests {
		if got := describeASTType(tt.typ); got != tt.want {
			t.Fatalf("describeASTType = %q, want %q", got, tt.want)
		}
	}
}
