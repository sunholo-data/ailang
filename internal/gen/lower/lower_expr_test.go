package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// --- Expression lowering tests ---

func TestLowerExpr_Literals(t *testing.T) {
	cti := makeCTI(nil)

	tests := []struct {
		name string
		expr core.CoreExpr
		want stmt.Expr
	}{
		{"int", litInt(1, 42), stmt.LitInt{Value: 42}},
		{"bool", litBool(2, true), stmt.LitBool{Value: true}},
		{"string", litStr(3, "hello"), stmt.LitString{Value: "hello"}},
		{"unit", &core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.UnitLit}, stmt.LitUnit{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LowerExpr(tt.expr, cti)
			if got != tt.want {
				t.Errorf("LowerExpr = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLowerExpr_Var(t *testing.T) {
	got := LowerExpr(coreVar(1, "x"), makeCTI(nil))
	if ref, ok := got.(stmt.VarRef); !ok || ref.Name != "x" {
		t.Errorf("expected VarRef{x}, got %#v", got)
	}
}

func TestLowerExpr_BinOp(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 10},
		Op:       "+",
		Left:     litInt(11, 1),
		Right:    litInt(12, 2),
	}
	got := LowerExpr(e, cti)
	bin, ok := got.(stmt.BinOp)
	if !ok {
		t.Fatalf("expected BinOp, got %T", got)
	}
	if bin.Op != stmt.OpAdd {
		t.Errorf("expected OpAdd, got %v", bin.Op)
	}
}

func TestLowerExpr_UnOp(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.UnOp{
		CoreNode: core.CoreNode{NodeID: 10},
		Op:       "-",
		Operand:  litInt(11, 5),
	}
	got := LowerExpr(e, cti)
	un, ok := got.(stmt.UnOp)
	if !ok {
		t.Fatalf("expected UnOp, got %T", got)
	}
	if un.Op != stmt.OpNeg {
		t.Errorf("expected OpNeg, got %v", un.Op)
	}
}

func TestLowerExpr_App(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.App{
		CoreNode: core.CoreNode{NodeID: 10},
		Func:     coreVar(11, "f"),
		Args:     []core.CoreExpr{litInt(12, 1), litInt(13, 2)},
	}
	got := LowerExpr(e, cti)
	call, ok := got.(stmt.Call)
	if !ok {
		t.Fatalf("expected Call, got %T", got)
	}
	if len(call.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(call.Args))
	}
}

func TestLowerExpr_If(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.If{
		CoreNode: core.CoreNode{NodeID: 10},
		Cond:     litBool(11, true),
		Then:     litInt(12, 1),
		Else:     litInt(13, 0),
	}
	got := LowerExpr(e, cti)
	ifExpr, ok := got.(stmt.IfExpr)
	if !ok {
		t.Fatalf("expected IfExpr, got %T", got)
	}
	if _, ok := ifExpr.Cond.(stmt.LitBool); !ok {
		t.Errorf("expected LitBool cond, got %T", ifExpr.Cond)
	}
}

func TestLowerExpr_Intrinsic(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 10},
		Op:       core.OpMul,
		Args:     []core.CoreExpr{litInt(11, 3), litInt(12, 4)},
	}
	got := LowerExpr(e, cti)
	bin, ok := got.(stmt.BinOp)
	if !ok {
		t.Fatalf("expected BinOp, got %T", got)
	}
	if bin.Op != stmt.OpMul {
		t.Errorf("expected OpMul, got %v", bin.Op)
	}
}

func TestLowerExpr_RecordAccess(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.RecordAccess{
		CoreNode: core.CoreNode{NodeID: 10},
		Record:   coreVar(11, "pos"),
		Field:    "x",
	}
	got := LowerExpr(e, cti)
	fa, ok := got.(stmt.FieldAccess)
	if !ok {
		t.Fatalf("expected FieldAccess, got %T", got)
	}
	if fa.Field != "x" {
		t.Errorf("expected field x, got %s", fa.Field)
	}
}

func TestLowerExpr_Tuple(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.Tuple{
		CoreNode: core.CoreNode{NodeID: 10},
		Elements: []core.CoreExpr{litInt(11, 1), litStr(12, "a")},
	}
	got := LowerExpr(e, cti)
	tup, ok := got.(stmt.TupleLit)
	if !ok {
		t.Fatalf("expected TupleLit, got %T", got)
	}
	if len(tup.Elems) != 2 {
		t.Errorf("expected 2 elements, got %d", len(tup.Elems))
	}
}

func TestLowerExpr_DictApp_Num(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.DictApp{
		CoreNode: core.CoreNode{NodeID: 10},
		Dict: &core.DictRef{
			CoreNode:  core.CoreNode{NodeID: 11},
			ClassName: "Num",
			TypeName:  "Int",
		},
		Method: "add",
		Args:   []core.CoreExpr{litInt(12, 1), litInt(13, 2)},
	}
	got := LowerExpr(e, cti)
	bin, ok := got.(stmt.BinOp)
	if !ok {
		t.Fatalf("expected BinOp for Num.add, got %T", got)
	}
	if bin.Op != stmt.OpAdd {
		t.Errorf("expected OpAdd, got %v", bin.Op)
	}
}

func TestLowerExpr_DictApp_Eq(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.DictApp{
		CoreNode: core.CoreNode{NodeID: 10},
		Dict: &core.DictRef{
			CoreNode:  core.CoreNode{NodeID: 11},
			ClassName: "Eq",
			TypeName:  "Int",
		},
		Method: "eq",
		Args:   []core.CoreExpr{litInt(12, 1), litInt(13, 2)},
	}
	got := LowerExpr(e, cti)
	bin, ok := got.(stmt.BinOp)
	if !ok {
		t.Fatalf("expected BinOp for Eq.eq, got %T", got)
	}
	if bin.Op != stmt.OpEq {
		t.Errorf("expected OpEq, got %v", bin.Op)
	}
}

func TestLowerExpr_DictAbs_Erased(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.DictAbs{
		CoreNode: core.CoreNode{NodeID: 10},
		Params:   []core.DictParam{{Name: "d", ClassName: "Num", Type: "Int"}},
		Body:     litInt(11, 42),
	}
	got := LowerExpr(e, cti)
	if lit, ok := got.(stmt.LitInt); !ok || lit.Value != 42 {
		t.Errorf("expected DictAbs to erase to body, got %#v", got)
	}
}

func TestLowerExpr_GlobalRef(t *testing.T) {
	cti := makeCTI(nil)
	e := &core.VarGlobal{
		CoreNode: core.CoreNode{NodeID: 10},
		Ref:      core.GlobalRef{Module: "math", Name: "abs"},
	}
	got := LowerExpr(e, cti)
	gr, ok := got.(stmt.GlobalRef)
	if !ok {
		t.Fatalf("expected GlobalRef, got %T", got)
	}
	if gr.Module != "math" || gr.Name != "abs" {
		t.Errorf("expected math.abs, got %s.%s", gr.Module, gr.Name)
	}
}

func TestLowerExpr_Nil(t *testing.T) {
	got := LowerExpr(nil, makeCTI(nil))
	if _, ok := got.(stmt.LitUnit); !ok {
		t.Errorf("expected LitUnit for nil, got %T", got)
	}
}
