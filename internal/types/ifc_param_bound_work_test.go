package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// The canonical number and the written size read every written form, and the
// walk stops at its work budget with the information-flow limit.
func TestIFCParamBound_CanonicalFormsAndBudget(t *testing.T) {
	src := `module app
type Pass[a] = (a, a)
type Every[a] = ([a], Array[a], { f: a }, (a) -> a, Option[a<s>], a{not k})
pure func p(x: (Pass[Every[string]], Pass[Every[string]])) -> int { 0 }
`
	p := parser.New(lexer.New(src, "forms.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	ts := newIFCTypes(file)
	var param ast.Type
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			param = fn.Params[0].Type
		}
	}
	shared := ts.paramBound(param)
	var plain ifcBound
	ts.addBound(&plain, param, ifcPath{}, nil, map[string]bool{}, 0)
	if shared.has != plain.has || !equalStrings(shared.forbidden(), plain.forbidden()) {
		t.Fatalf("shared %+v, unshared %+v", shared, plain)
	}
	if size, limit := ts.boundWork(param); size < 20 || limit != workBudget(size) {
		t.Fatalf("written size %d, limit %d", size, limit)
	}
	w := &ifcBoundWalk{canon: map[ifcCanonKey]int{}, interned: map[string]int{}, done: map[ifcBoundKey]bool{}, scopes: map[string]map[string]ifcTypeArg{}}
	first := w.canonical(ifcTypeArg{})
	if again := w.canonical(ifcTypeArg{}); first != again {
		t.Error("the unapplied argument numbered twice")
	}
	if w.canonical(ifcTypeArg{t: &ast.FuncType{}}) != w.canonical(ifcTypeArg{t: &ast.FuncType{Return: &ast.SimpleType{Name: "int"}}}) {
		t.Error("two function types (both leaves to the bound) numbered apart")
	}
	w.limit = 1
	defer func() {
		if limit, ok := recover().(ifcAnalysisLimit); !ok || !strings.Contains(limit.reason, "parameter type work") {
			t.Fatalf("want the work limit, got %v", limit)
		}
	}()
	w.step()
	w.step()
	t.Fatal("the walk ran past its budget")
}
