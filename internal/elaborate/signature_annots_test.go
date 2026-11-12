package elaborate

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2238: every function form records its annotations through
// recordSignatureAnnots; function literals are marked so the checker enforces
// their parameter annotations as equalities.
func TestRecordSignatureAnnots_FunctionLiterals(t *testing.T) {
	for _, c := range []struct {
		name, src      string
		params         []types.Type // nil slot = unannotated; nil slice = none recorded
		ret            types.Type
		literal        bool
		recordedParams bool
	}{
		{"func literal", `let f = func(x: int, y) -> string { "a" } in f`, []types.Type{types.TInt, nil}, types.TString, true, true},
		{"fat-arrow lambda", `let f = func(x: bool) => x in f`, []types.Type{types.TBool}, nil, true, true},
		{"unannotated literal", `let f = func(x) -> int { 1 } in f`, nil, types.TInt, true, false},
		{"backslash lambda", `let f = \x. x in f`, nil, nil, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := parser.New(lexer.New(c.src, "test.ail"))
			prog := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}
			e := NewElaborator()
			coreProg, err := e.Elaborate(prog)
			if err != nil {
				t.Fatalf("elaborate: %v", err)
			}
			lam := firstLambda(coreProg.Decls[0])
			if lam == nil {
				t.Fatal("no core.Lambda produced")
			}
			got, ok := e.GetParamTypeAnnotations()[lam.ID()]
			if ok != c.recordedParams {
				t.Fatalf("param annotations recorded = %v, want %v (%v)", ok, c.recordedParams, got)
			}
			for i, want := range c.params {
				if (want == nil) != (got[i] == nil) || (want != nil && !want.Equals(got[i])) {
					t.Errorf("param %d = %v, want %v", i, got[i], want)
				}
			}
			ret := e.GetReturnTypeAnnotations()[lam.ID()]
			if (c.ret == nil) != (ret == nil) || (c.ret != nil && !c.ret.Equals(ret)) {
				t.Errorf("return annotation = %v, want %v", ret, c.ret)
			}
			// A backslash lambda carries no annotation; it is still a literal.
			if lit := e.GetLiteralLambdas()[lam.ID()]; lit != c.literal {
				t.Errorf("literal = %v, want %v", lit, c.literal)
			}
		})
	}
}

// A top-level func is not a literal: its annotations seed the parameter type.
func TestRecordSignatureAnnots_TopLevelFuncIsNotALiteral(t *testing.T) {
	p := parser.New(lexer.New("module m\nfunc f(x: int) -> int = x\n", "m.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	e := NewElaborator()
	if _, err := e.ElaborateFile(file); err != nil {
		t.Fatalf("elaborate: %v", err)
	}
	if len(e.GetParamTypeAnnotations()) != 1 {
		t.Fatalf("want one annotated lambda, got %v", e.GetParamTypeAnnotations())
	}
	if len(e.GetLiteralLambdas()) != 0 {
		t.Fatalf("a top-level func must not be marked a literal: %v", e.GetLiteralLambdas())
	}
}

// elaborateFuncDecl (the legacy Module-decls path) goes through normalizeLambda,
// which marks a literal; recording the declaration's own signature afterwards
// must unmark it and record the return annotation (#2238 review: mutants M6,
// M12b).
func TestRecordSignatureAnnots_ElaborateFuncDecl(t *testing.T) {
	p := parser.New(lexer.New("module m\nfunc f(x: int) -> string = \"a\"\n", "m.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			fd = f
		}
	}
	if fd == nil {
		t.Fatal("no FuncDecl parsed")
	}
	e := NewElaborator()
	expr, err := e.elaborateFuncDecl(fd)
	if err != nil {
		t.Fatalf("elaborateFuncDecl: %v", err)
	}
	lam := firstLambda(expr)
	if lam == nil {
		t.Fatal("no core.Lambda produced")
	}
	if got := e.GetParamTypeAnnotations()[lam.ID()]; len(got) != 1 || !types.TInt.Equals(got[0]) {
		t.Errorf("param annotations = %v, want [int]", got)
	}
	if ret := e.GetReturnTypeAnnotations()[lam.ID()]; ret == nil || !types.TString.Equals(ret) {
		t.Errorf("return annotation = %v, want string", ret)
	}
	if e.GetLiteralLambdas()[lam.ID()] {
		t.Error("a func declaration must not stay marked as a literal")
	}
}

func firstLambda(expr core.CoreExpr) *core.Lambda {
	switch x := expr.(type) {
	case *core.Lambda:
		return x
	case *core.Let:
		if l := firstLambda(x.Value); l != nil {
			return l
		}
		return firstLambda(x.Body)
	case *core.LetRec:
		for _, b := range x.Bindings {
			if l := firstLambda(b.Value); l != nil {
				return l
			}
		}
		return firstLambda(x.Body)
	}
	return nil
}
