package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-VM-COVERAGE G1b: every VarDecl of a local letrec carries the whole
// group, so the bytecode compiler can bind the names before the values.
// Covers both lowering sites: FlattenBlock and flattenValue (a letrec as the
// value of a let).
func TestFlattenBlock_LetRecCarriesGroup(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{})
	letrec := func(id uint64) *core.LetRec {
		return &core.LetRec{
			CoreNode: core.CoreNode{NodeID: id},
			Bindings: []core.RecBinding{
				{Name: "f", Value: litInt(id+1, 1)},
				{Name: "g", Value: litInt(id+2, 2)},
			},
			Body: coreVar(id+3, "f"),
		}
	}
	check := func(t *testing.T, stmts []stmt.Stmt) {
		t.Helper()
		seen := 0
		for _, s := range stmts {
			vd, ok := s.(stmt.VarDecl)
			if !ok || (vd.Name != "f" && vd.Name != "g") {
				continue
			}
			seen++
			if len(vd.RecGroup) != 2 || vd.RecGroup[0] != "f" || vd.RecGroup[1] != "g" {
				t.Errorf("VarDecl %q RecGroup = %v, want [f g]", vd.Name, vd.RecGroup)
			}
		}
		if seen != 2 {
			t.Fatalf("found %d letrec VarDecls, want 2: %#v", seen, stmts)
		}
	}
	t.Run("block", func(t *testing.T) {
		stmts, _ := FlattenBlock(letrec(10), cti)
		check(t, stmts)
	})
	t.Run("let_value", func(t *testing.T) {
		e := &core.Let{CoreNode: core.CoreNode{NodeID: 1}, Name: "x", Value: letrec(20), Body: coreVar(2, "x")}
		stmts, _ := FlattenBlock(e, cti)
		check(t, stmts)
	})
}

// A letrec reached in expression position cannot emit its bindings; it must
// fail the lowering (an EvalOnly stub) rather than drop them and let the
// body's references bind to some other function of the same name.
func TestLowerLetRecExpr_RefusesToDropBindings(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{})
	e := &core.LetRec{
		CoreNode: core.CoreNode{NodeID: 1},
		Bindings: []core.RecBinding{{Name: "go", Value: litInt(2, 1)}},
		Body:     coreVar(3, "go"),
	}
	defer func() {
		if recover() == nil {
			t.Fatal("lowerLetRecExpr dropped the bindings instead of refusing")
		}
	}()
	lowerLetRecExpr(e, cti)
}
