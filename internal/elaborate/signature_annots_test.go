package elaborate

import (
	"testing"

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
