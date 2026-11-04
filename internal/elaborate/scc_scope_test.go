package elaborate

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"reflect"
	"testing"
)

func TestFindReferencesScopes(t *testing.T) {
	id := func(s string) *ast.Identifier { return &ast.Identifier{Name: s} }
	param := []*ast.Param{{Name: "t"}}
	cases := []struct {
		name string
		expr ast.Expr
		want []string
	}{
		{"lambda", &ast.Lambda{Params: param, Body: id("t")}, nil},
		{"func_literal", &ast.FuncLit{Params: param, Body: id("t")}, nil},
		{"let_value_outer", &ast.Let{Name: "t", Value: id("t"), Body: id("t")}, []string{"t"}},
		{"local_letrec", &ast.LetRec{Name: "t", Value: id("t"), Body: id("t")}, nil},
		{"module_letrec", &ast.LetRec{Name: "t", Value: id("t")}, []string{"t"}},
		{"match_guard_and_body", &ast.Match{Expr: id("xs"), Cases: []*ast.Case{{Pattern: id("t"), Guard: id("t"), Body: id("t")}, {Pattern: &ast.WildcardPattern{}, Body: id("t")}}}, []string{"xs", "t"}},
		{"block_forward", &ast.Block{Exprs: []ast.Expr{id("t"), &ast.Let{Name: "t", Value: id("v")}, id("t")}}, []string{"t", "v"}},
		{"block_nested_no_leak", &ast.Block{Exprs: []ast.Expr{&ast.Block{Exprs: []ast.Expr{&ast.Let{Name: "t", Value: id("v")}, id("t")}}, id("t")}}, []string{"v", "t"}},
		{"let_body_no_leak", &ast.Block{Exprs: []ast.Expr{&ast.Let{Name: "t", Value: id("v"), Body: id("t")}, id("t")}}, []string{"v", "t"}},
		{"forall", &ast.ForallExpr{Var: "t", Lo: id("t"), Hi: id("hi"), Body: id("t")}, []string{"t", "hi"}},
		{"send", &ast.Send{Channel: id("ch"), Value: id("v")}, []string{"ch", "v"}},
		{"recv", &ast.Recv{Channel: id("ch")}, []string{"ch"}},
		{"assert", &ast.AssertStmt{Condition: id("ok")}, []string{"ok"}},
		{"quasiquote", &ast.QuasiQuote{Interpolations: []*ast.Interpolation{{Expr: id("v")}}}, []string{"v"}},
		{"sibling_scope", &ast.Tuple{Elements: []ast.Expr{&ast.Lambda{Params: param, Body: id("t")}, id("t")}}, []string{"t"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := findReferences(tt.expr); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("refs=%v want=%v", got, tt.want)
			}
		})
	}
}

// Embedding the interface lets this test model a future AST variant.
type unhandledSCCExpr struct{ ast.Expr }

func TestFindReferencesStrict(t *testing.T) {
	t.Setenv("DEBUG_STRICT", "1")
	defer func() {
		if recover() == nil {
			t.Fatal("unhandled variant did not panic")
		}
	}()
	findReferences(&unhandledSCCExpr{})
}
