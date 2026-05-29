package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// --- Block flattening tests ---

func TestFlattenBlock_SimpleLet(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{
		2: types.TInt,
	})

	// let a = 42 in a
	e := &core.Let{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "a",
		Value:    litInt(2, 42),
		Body:     coreVar(3, "a"),
	}

	stmts, ret := FlattenBlock(e, cti)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}

	vd, ok := stmts[0].(stmt.VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", stmts[0])
	}
	if vd.Name != "a" {
		t.Errorf("expected var name a, got %s", vd.Name)
	}

	ref, ok := ret.(stmt.VarRef)
	if !ok {
		t.Fatalf("expected VarRef return, got %T", ret)
	}
	if ref.Name != "a" {
		t.Errorf("expected return ref a, got %s", ref.Name)
	}
}

func TestFlattenBlock_NestedLet(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{
		2: types.TInt,
		4: types.TInt,
	})

	// let a = 1 in let b = 2 in a + b
	e := &core.Let{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "a",
		Value:    litInt(2, 1),
		Body: &core.Let{
			CoreNode: core.CoreNode{NodeID: 3},
			Name:     "b",
			Value:    litInt(4, 2),
			Body: &core.BinOp{
				CoreNode: core.CoreNode{NodeID: 5},
				Op:       "+",
				Left:     coreVar(6, "a"),
				Right:    coreVar(7, "b"),
			},
		},
	}

	stmts, ret := FlattenBlock(e, cti)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}

	// Both should be VarDecl.
	for i, s := range stmts {
		if _, ok := s.(stmt.VarDecl); !ok {
			t.Errorf("stmt[%d]: expected VarDecl, got %T", i, s)
		}
	}

	// Return should be BinOp.
	if _, ok := ret.(stmt.BinOp); !ok {
		t.Errorf("expected BinOp return, got %T", ret)
	}
}

func TestFlattenBlock_AtomicExpr(t *testing.T) {
	// No let-chain — just a literal.
	stmts, ret := FlattenBlock(litInt(1, 99), makeCTI(nil))
	if len(stmts) != 0 {
		t.Errorf("expected 0 statements, got %d", len(stmts))
	}
	if lit, ok := ret.(stmt.LitInt); !ok || lit.Value != 99 {
		t.Errorf("expected LitInt{99}, got %#v", ret)
	}
}

func TestFlattenBlock_IfSimple(t *testing.T) {
	// if true then 1 else 0
	e := &core.If{
		CoreNode: core.CoreNode{NodeID: 1},
		Cond:     litBool(2, true),
		Then:     litInt(3, 1),
		Else:     litInt(4, 0),
	}
	stmts, ret := FlattenBlock(e, makeCTI(nil))
	if len(stmts) != 0 {
		t.Errorf("expected 0 statements for simple if, got %d", len(stmts))
	}
	if _, ok := ret.(stmt.IfExpr); !ok {
		t.Errorf("expected IfExpr, got %T", ret)
	}
}
