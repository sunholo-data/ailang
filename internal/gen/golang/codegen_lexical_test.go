package golang

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestLexicalReferencesStayWithinScope(t *testing.T) {
	local := &core.Var{Name: "decode"}
	unbound := &core.Var{Name: "decode"}
	initializer := &core.Var{Name: "decode"}
	prog := &core.Program{Decls: []core.CoreExpr{&core.If{
		Cond: &core.Lit{Kind: core.BoolLit, Value: true},
		Then: &core.Let{Name: "decode", Value: initializer, Body: local}, Else: unbound,
	}}}
	refs := localReferences(prog)
	if !refs[local] || refs[unbound] || refs[initializer] {
		t.Fatal("let binding escaped its body or shadowed its own initializer")
	}
	g := New("lexical")
	g.lexicalVarRefs = refs
	if err := g.generateVarGlobal(&core.VarGlobal{Ref: core.GlobalRef{Module: "std/json", Name: "decode"}}); err != nil {
		t.Fatal(err)
	}
	if got := g.buf.String(); got != "Decode" {
		t.Fatalf("explicit std/json.decode resolved to %q", got)
	}
}

func TestTopLevelDeclarationsAreNotLexicalLocals(t *testing.T) {
	global := &core.Var{Name: "helper"}
	caller := &core.Lambda{Params: []string{"_"}, Body: global}
	prog := &core.Program{Decls: []core.CoreExpr{
		&core.Let{Name: "helper", Value: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
			Body: &core.Let{Name: "caller", Value: caller, Body: &core.Var{Name: "caller"}}},
		&core.Let{Name: "caller", Value: caller, Body: &core.Var{Name: "caller"}},
	}}
	if localReferences(prog)[global] {
		t.Fatal("global helper reference was incorrectly marked local through residual top-level Body")
	}
}
