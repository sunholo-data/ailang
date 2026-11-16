package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// The shared IFC bound walk (#2329) must compute exactly the bound the
// unshared walk computes, for every parameter: it may only skip work whose
// outcome it already has. The shapes are the ones its keys and its recursion
// rule are about: a parameter passed through aliases, one alias reached under
// two labels, mutually recursive declarations (whose walks cut a recursive
// occurrence and so are never remembered), and refinements.
func TestIFCParamBound_SharedWalkEqualsUnsharedWalk(t *testing.T) {
	src := `module app
type Q0[a] = a
type Q1[a] = (Q0[a], Q0[a])
type Q2[a] = (Q1[a], Q1[a])
type Pair[a] = (a, a)
type W[b] = Pair[b]
type Str = string
type L[a] = Cons(a, L[a]) | Nil
type X[a] = MkX(a, Y[a])
type Y[a] = MkY(X[a], a)
type Z[a] = MkZ(Y[a<y>], a<z>)
pure func p1(x: Q2[string<s>]) -> int { 0 }
pure func p2(x: (W[string<s>], W[string{not secret}])) -> int { 0 }
pure func p3(x: { a: Str<s>, b: Str<e> }) -> int { 0 }
pure func p4(x: (L[string<s>], L[string<s>]<e>)) -> int { 0 }
pure func p5(x: (X[string<s>], Y[string<s>]<e>)) -> int { 0 }
pure func p6(x: (Y[string], X[string]<s>, Z[string]<e>)) -> int { 0 }
pure func p7(x: (Pair[W[string<s>]], W[Pair[string{not k}]])) -> int { 0 }
`
	p := parser.New(lexer.New(src, "memo_test.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	ts := newIFCTypes(file)
	checked := 0
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, param := range fn.Params {
			shared := ts.paramBound(param.Type)
			var plain ifcBound // walk == nil: nothing is shared
			ts.addBound(&plain, param.Type, ifcPath{}, nil, map[string]bool{}, 0)
			if plain.has && plain.cover == nil {
				plain.cover = LabelBottom()
			}
			if shared.has != plain.has || (shared.has && !LabelEqual(shared.cover, plain.cover)) ||
				!equalStrings(shared.forbidden(), plain.forbidden()) {
				t.Errorf("%s: shared walk %+v, unshared walk %+v", fn.Name, shared, plain)
			}
			checked++
		}
	}
	if checked != 7 {
		t.Fatalf("checked %d parameters, want 7", checked)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
