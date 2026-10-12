package apiserver

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestDeclaredHeaderAliasesAndResult(t *testing.T) {
	str := &ast.SimpleType{Name: "string"}
	record := func(name string, typ ast.Type) *ast.RecordType {
		return &ast.RecordType{Fields: []*ast.RecordField{{Name: name, Type: typ}}}
	}
	file := &ast.File{Decls: []ast.Node{
		&ast.TypeDecl{Name: "Headers", Definition: record("x_frame_options", str)},
		&ast.TypeDecl{Name: "JSONHeaders", Definition: &ast.TypeAlias{Target: &ast.SimpleType{Name: "Json"}}},
		&ast.TypeDecl{Name: "Response", TypeParams: []string{"h"}, Definition: &ast.TypeAlias{Target: record("_headers", &ast.TypeVar{Name: "h"})}},
		&ast.TypeDecl{Name: "BadHeaders", Definition: record("x_bad", &ast.SimpleType{Name: "int"})},
	}}
	for _, tc := range []struct {
		name   string
		result ast.Type
		bad    bool
	}{
		{"record alias", record("_headers", &ast.SimpleType{Name: "Headers"}), false},
		{"json alias", record("_headers", &ast.SimpleType{Name: "JSONHeaders"}), false},
		{"generic response", &ast.TypeApp{Constructor: "Response", Args: []ast.Type{&ast.SimpleType{Name: "Headers"}}}, false},
		{"result", &ast.TypeApp{Constructor: "Result", Args: []ast.Type{record("_headers", &ast.SimpleType{Name: "Headers"}), str}}, false},
		{"result bad", &ast.TypeApp{Constructor: "Result", Args: []ast.Type{record("_headers", &ast.SimpleType{Name: "BadHeaders"}), str}}, true},
		{"imported alias", record("_headers", &ast.SimpleType{Name: "ImportedJsonAlias"}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issue := declaredResponseHeadersIssue(&ast.FuncDecl{ReturnType: tc.result}, file)
			if (issue != "") != tc.bad {
				t.Fatalf("issue=%q bad=%v", issue, tc.bad)
			}
			if tc.bad && !strings.Contains(issue, "x_bad") {
				t.Fatalf("missing field: %s", issue)
			}
		})
	}
}
