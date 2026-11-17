package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// A top-level recursive group lowers to one FuncDecl per binding, and a
// recursive reference inside a body stays the bare name: the bytecode
// compiler looks functions up by FuncDecl.Name (nothing qualifies them since
// the Go emitter's QualifyFuncRefs pass was removed).
func TestLowerTopLevelLetRec_OneFuncPerBindingBareRefs(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{})
	f := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 2},
		Params:   []string{"n"},
		Body: &core.App{
			CoreNode: core.CoreNode{NodeID: 3},
			Func:     coreVar(4, "f"),
			Args:     []core.CoreExpr{coreVar(5, "n")},
		},
	}
	g := &core.Lambda{CoreNode: core.CoreNode{NodeID: 6}, Params: []string{"x"}, Body: coreVar(7, "x")}
	e := &core.LetRec{
		CoreNode: core.CoreNode{NodeID: 1},
		Bindings: []core.RecBinding{{Name: "f", Value: f}, {Name: "g", Value: g}},
		Body:     coreVar(8, "f"),
	}
	meta := map[string]*core.DeclMeta{"f": {IsExport: true}}

	decls, err := lowerTopLevelDecl(e, meta, cti)
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 2 || decls[0].Name != "f" || decls[1].Name != "g" {
		t.Fatalf("got %d decls %+v, want f and g", len(decls), decls)
	}
	if !decls[0].Exported || decls[1].Exported {
		t.Errorf("exported = %v/%v, want f exported and g not", decls[0].Exported, decls[1].Exported)
	}
	var refs []string
	walkExprsInFunc(&decls[0], func(x stmt.Expr) {
		if v, ok := x.(stmt.VarRef); ok {
			refs = append(refs, v.Name)
		}
	})
	found := false
	for _, r := range refs {
		if r == "f" {
			found = true
		}
		if r != "f" && r != "n" {
			t.Errorf("unexpected reference %q in f's body (a qualified or rewritten name?)", r)
		}
	}
	if !found {
		t.Errorf("f's recursive reference is not the bare name f: refs %v", refs)
	}
}
