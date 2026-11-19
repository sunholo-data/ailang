package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// A constructor used as a value lowers by its type: arity 0 is the value
// itself, arity n is its eta-expansion (a lambda building the ADT), so the
// VM holds a function, as the evaluator does (M-V1-SHOW-TOTAL review item 2).
func TestLowerExpr_ConstructorAsValue(t *testing.T) {
	ref := func(id uint64, name string) *core.VarGlobal {
		return &core.VarGlobal{CoreNode: core.CoreNode{NodeID: id}, Ref: core.GlobalRef{Module: "$adt", Name: name}}
	}
	pair := &types.TFunc2{Params: []types.Type{types.TInt, types.TString}, Return: &types.TCon{Name: "Pair"}}
	cti := makeCTI(map[uint64]types.Type{
		1: &types.TCon{Name: "Option"},
		2: pair,
	})

	if got, ok := LowerExpr(ref(1, "make_Option_None"), cti).(stmt.ADTConstructor); !ok || got.Tag != "None" || len(got.Args) != 0 {
		t.Fatalf("nullary constructor: got %#v", LowerExpr(ref(1, "make_Option_None"), cti))
	}

	lam, ok := LowerExpr(ref(2, "make_Pair_MkPair"), cti).(stmt.Lambda)
	if !ok {
		t.Fatalf("binary constructor as a value: got %T, want a lambda", LowerExpr(ref(2, "make_Pair_MkPair"), cti))
	}
	if len(lam.Params) != 2 || len(lam.Body) != 0 {
		t.Fatalf("lambda params %d body %d, want 2 and 0", len(lam.Params), len(lam.Body))
	}
	build, ok := lam.Return.(stmt.ADTConstructor)
	if !ok || build.TypeName != "Pair" || build.Tag != "MkPair" || len(build.Args) != 2 {
		t.Fatalf("lambda body: %#v", lam.Return)
	}
	for i, a := range build.Args {
		if v, ok := a.(stmt.VarRef); !ok || v.Name != lam.Params[i].Name {
			t.Fatalf("arg %d is %#v, want a reference to parameter %q", i, a, lam.Params[i].Name)
		}
	}
	if lam.Params[0].Name == lam.Params[1].Name {
		t.Fatal("parameters must be distinct")
	}
}
