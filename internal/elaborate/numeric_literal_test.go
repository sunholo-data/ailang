package elaborate

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-SOUND-NUMERIC-LITERALS: dictionary elaboration is where an integer
// literal's resolved Num constraint meets the runtime. At Float it must become
// a float literal; at Int (or unresolved) it stays an integer literal. Tuples,
// arrays and record updates must be traversed — they used to fall through to
// `default` untransformed.
func TestElaborateIntLiteralAtFloat(t *testing.T) {
	lit := func(id uint64, v int64) *core.Lit {
		return &core.Lit{CoreNode: core.CoreNode{NodeID: id}, Kind: core.IntLit, Value: v}
	}
	resolved := map[uint64]*types.ResolvedConstraint{
		1: {NodeID: 1, ClassName: "Fractional", Type: &types.TCon{Name: "Float"}},
		2: {NodeID: 2, ClassName: "Num", Type: &types.TCon{Name: "Int"}},
		4: {NodeID: 4, ClassName: "Fractional", Type: &types.TCon{Name: "Float"}},
		5: {NodeID: 5, ClassName: "Fractional", Type: &types.TCon{Name: "Float"}},
		6: {NodeID: 6, ClassName: "Fractional", Type: &types.TCon{Name: "Float"}},
	}
	prog := &core.Program{Decls: []core.CoreExpr{
		lit(1, 0),
		lit(2, 7),
		lit(3, 9), // no resolved constraint
		&core.Tuple{CoreNode: core.CoreNode{NodeID: 10}, Elements: []core.CoreExpr{lit(4, 1), lit(2, 7)}},
		&core.Array{CoreNode: core.CoreNode{NodeID: 11}, Elements: []core.CoreExpr{lit(5, 2)}},
		&core.RecordUpdate{CoreNode: core.CoreNode{NodeID: 12}, Base: &core.Var{Name: "r"},
			Updates: map[string]core.CoreExpr{"x": lit(6, 3)}},
	}}
	out, err := ElaborateWithDictionaries(prog, resolved)
	if err != nil {
		t.Fatal(err)
	}
	wantFloat := func(e core.CoreExpr, v float64, where string) {
		t.Helper()
		l, ok := e.(*core.Lit)
		if !ok || l.Kind != core.FloatLit || l.Value != v {
			t.Errorf("%s: got %#v, want FloatLit %v", where, e, v)
			return
		}
		if l.ID() == 0 {
			t.Errorf("%s: rewritten literal lost its node ID (CoreTypeInfo would miss it)", where)
		}
	}
	wantInt := func(e core.CoreExpr, where string) {
		t.Helper()
		if l, ok := e.(*core.Lit); !ok || l.Kind != core.IntLit {
			t.Errorf("%s: got %#v, want IntLit", where, e)
		}
	}
	wantFloat(out.Decls[0], 0, "literal resolved at Float")
	wantInt(out.Decls[1], "literal resolved at Int")
	wantInt(out.Decls[2], "literal without a resolved constraint")
	tup := out.Decls[3].(*core.Tuple)
	wantFloat(tup.Elements[0], 1, "tuple element at Float")
	wantInt(tup.Elements[1], "tuple element at Int")
	wantFloat(out.Decls[4].(*core.Array).Elements[0], 2, "array element at Float")
	wantFloat(out.Decls[5].(*core.RecordUpdate).Updates["x"], 3, "record update at Float")
}
