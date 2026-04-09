package lower

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// Helper to create a Core node with a specific ID and type in CTI.
func litInt(id uint64, val int64) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.IntLit,
		Value:    val,
	}
}

func litBool(id uint64, val bool) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.BoolLit,
		Value:    val,
	}
}

func litStr(id uint64, val string) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.StringLit,
		Value:    val,
	}
}

func coreVar(id uint64, name string) *core.Var {
	return &core.Var{
		CoreNode: core.CoreNode{NodeID: id},
		Name:     name,
	}
}

func makeCTI(entries map[uint64]types.Type) types.CoreTypeInfo {
	cti := make(types.CoreTypeInfo)
	for k, v := range entries {
		cti[k] = v
	}
	return cti
}

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

// --- Match lowering tests ---

func TestLowerMatchStmt_Constructor(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:   core.CoreNode{NodeID: 1},
		Scrutinee:  coreVar(2, "c"),
		Exhaustive: true,
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{Name: "Red"},
				Body:    litInt(3, 0),
			},
			{
				Pattern: &core.ConstructorPattern{Name: "Green"},
				Body:    litInt(4, 1),
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    litInt(5, 2),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	sw, ok := result.(stmt.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", result)
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	if sw.Cases[0].Tag != "Red" {
		t.Errorf("expected first case Red, got %s", sw.Cases[0].Tag)
	}
	if len(sw.Default) == 0 {
		t.Error("expected default branch")
	}
}

func TestLowerMatchStmt_ConstructorWithBindings(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "opt"),
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Some",
					Args: []core.CorePattern{&core.VarPattern{Name: "x"}},
				},
				Body: coreVar(3, "x"),
			},
			{
				Pattern: &core.ConstructorPattern{Name: "None"},
				Body:    litInt(4, 0),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	sw, ok := result.(stmt.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", result)
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	// First case should have a binding.
	if len(sw.Cases[0].Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(sw.Cases[0].Bindings))
	}
	if sw.Cases[0].Bindings[0].Name != "x" {
		t.Errorf("expected binding name x, got %s", sw.Cases[0].Bindings[0].Name)
	}
}

// TestLowerMatchStmt_ConstructorWithLiteralArg is the regression test for
// Bug A.2 (M-LOWER-FIX follow-up): a constructor pattern with a literal
// sub-argument (e.g., `Num(0) => true, _ => false`) must compile so that
// the literal value is actually checked, not silently ignored. The lowered
// SwitchStmt should have:
//
//   - A binding for the literal field (named `_lit_0`).
//   - A wrapping IfStmt whose condition compares the bound value to the
//     literal, with the default body in the else branch.
func TestLowerMatchStmt_ConstructorWithLiteralArg(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "e"),
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{
					Name: "Num",
					Args: []core.CorePattern{&core.LitPattern{Value: int64(0)}},
				},
				Body: litBool(3, true),
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    litBool(4, false),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	sw, ok := result.(stmt.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", result)
	}
	if len(sw.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(sw.Cases))
	}
	c := sw.Cases[0]
	if c.Tag != "Num" {
		t.Errorf("expected tag Num, got %s", c.Tag)
	}
	// Bug A.2 fix: literal sub-pattern must be bound to a temp.
	if len(c.Bindings) != 1 || c.Bindings[0].Name != "_lit_0" {
		t.Fatalf("expected binding _lit_0, got %+v", c.Bindings)
	}
	// Body must be wrapped in an IfStmt that compares _lit_0 to 0.
	if len(c.Body) != 1 {
		t.Fatalf("expected case body length 1 (the wrapping if), got %d", len(c.Body))
	}
	ifStmt, ok := c.Body[0].(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected case body to be IfStmt (literal-guard wrapper), got %T", c.Body[0])
	}
	binOp, ok := ifStmt.Cond.(stmt.BinOp)
	if !ok || binOp.Op != stmt.OpEq {
		t.Fatalf("expected guard cond to be Eq BinOp, got %+v", ifStmt.Cond)
	}
	if v, ok := binOp.Left.(stmt.VarRef); !ok || v.Name != "_lit_0" {
		t.Errorf("expected guard left to reference _lit_0, got %+v", binOp.Left)
	}
	if v, ok := binOp.Right.(stmt.LitInt); !ok || v.Value != 0 {
		t.Errorf("expected guard right to be LitInt 0, got %+v", binOp.Right)
	}
	// Else branch must be the default body (so the case falls through
	// to "false" when the guard fails, instead of silently exiting).
	if len(ifStmt.Else) == 0 {
		t.Error("expected guard else branch to contain default body, got empty")
	}
}

// TestLowerMatchStmt_ConsConstructorHead — M5 regression test.
//
// Reproduces the "unbound variable t" bug observed in docparse/main.ail:
// when a ListPattern element is a ConstructorPattern (e.g., TextBlock(t) :: rest),
// lowerPatternBindings previously only handled VarPattern elements,
// silently skipping constructor sub-patterns. This left inner bindings like
// `t` undeclared. Additionally, lowerPatternCond only checked list length
// but never verified the constructor tag on the head element.
//
// Shape of the failing pattern:
//
//	match blocks {
//	  [] => "none",
//	  TextBlock(t) :: rest => t.style,
//	  _ :: rest => "other"
//	}
func TestLowerMatchStmt_ConsConstructorHead(t *testing.T) {
	cti := makeCTI(nil)

	tail := core.CorePattern(&core.VarPattern{Name: "rest"})
	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "blocks"),
		Arms: []core.MatchArm{
			{
				// [] => "none"
				Pattern: &core.ListPattern{},
				Body:    litStr(3, "none"),
			},
			{
				// TextBlock(t) :: rest => t
				Pattern: &core.ListPattern{
					Elements: []core.CorePattern{
						&core.ConstructorPattern{
							Name: "TextBlock",
							Args: []core.CorePattern{&core.VarPattern{Name: "t"}},
						},
					},
					Tail: &tail,
				},
				Body: coreVar(4, "t"),
			},
			{
				// _ :: rest => "other"
				Pattern: &core.WildcardPattern{},
				Body:    litStr(5, "other"),
			},
		},
	}

	result := LowerMatchStmt(m, cti)
	// This is an if-chain (mixed pattern types, not pure constructors).
	ifStmt, ok := result.(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt (if-chain), got %T", result)
	}

	// The first arm is [] => "none" (empty list check).
	// The second arm is TextBlock(t) :: rest => t.
	// Walk to the second arm (the else branch of the first if).
	if len(ifStmt.Else) == 0 {
		t.Fatalf("expected else branch for second arm, got empty")
	}
	innerIf, ok := ifStmt.Else[0].(stmt.IfStmt)
	if !ok {
		t.Fatalf("expected inner IfStmt for second arm, got %T", ifStmt.Else[0])
	}

	// Bug 1 fix: condition must include a tag check for "TextBlock" (not just length).
	condStr := stmtCondString(innerIf.Cond)
	if condStr == "" {
		t.Fatalf("could not stringify condition: %#v", innerIf.Cond)
	}
	if !containsTagCheck(innerIf.Cond, "TextBlock") {
		t.Errorf("expected tag check for TextBlock in condition, got: %s", condStr)
	}

	// Bug 2 fix: the Then branch must declare `t` before referencing it.
	declared := map[string]bool{}
	for _, s := range innerIf.Then {
		if vd, ok := s.(stmt.VarDecl); ok {
			declared[vd.Name] = true
		}
	}
	if !declared["t"] {
		t.Errorf("expected VarDecl for 't' in Then branch, got stmts: %#v", innerIf.Then)
	}
	// Should also declare `rest`.
	if !declared["rest"] {
		t.Errorf("expected VarDecl for 'rest' in Then branch, got stmts: %#v", innerIf.Then)
	}
}

// containsTagCheck recursively checks whether an expression contains
// a BinOp{OpEq, FieldAccess{Field:"Tag"}, LitString{Value:tag}}.
func containsTagCheck(e stmt.Expr, tag string) bool {
	switch e := e.(type) {
	case stmt.BinOp:
		if e.Op == stmt.OpEq {
			fa, faOk := e.Left.(stmt.FieldAccess)
			ls, lsOk := e.Right.(stmt.LitString)
			if faOk && lsOk && fa.Field == "Tag" && ls.Value == tag {
				return true
			}
		}
		return containsTagCheck(e.Left, tag) || containsTagCheck(e.Right, tag)
	default:
		return false
	}
}

// stmtCondString returns a debug-friendly string for a condition expression.
func stmtCondString(e stmt.Expr) string {
	switch e := e.(type) {
	case stmt.BinOp:
		return "(" + stmtCondString(e.Left) + " op" + fmt.Sprintf("%d", e.Op) + " " + stmtCondString(e.Right) + ")"
	case stmt.LitInt:
		return fmt.Sprintf("%d", e.Value)
	case stmt.LitString:
		return fmt.Sprintf("%q", e.Value)
	case stmt.LitBool:
		return fmt.Sprintf("%v", e.Value)
	case stmt.VarRef:
		return e.Name
	case stmt.FieldAccess:
		return stmtCondString(e.Record) + "." + e.Field
	case stmt.BuiltinCall:
		return e.Name + "(...)"
	default:
		return fmt.Sprintf("%T", e)
	}
}

func TestLowerMatchStmt_LitPattern(t *testing.T) {
	cti := makeCTI(nil)

	m := &core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "n"),
		Arms: []core.MatchArm{
			{Pattern: &core.LitPattern{Value: int64(0)}, Body: litStr(3, "zero")},
			{Pattern: &core.LitPattern{Value: int64(1)}, Body: litStr(4, "one")},
			{Pattern: &core.WildcardPattern{}, Body: litStr(5, "other")},
		},
	}

	result := LowerMatchStmt(m, cti)
	// Should be an if-chain (lit patterns aren't constructor patterns).
	if _, ok := result.(stmt.IfStmt); !ok {
		t.Errorf("expected IfStmt for lit patterns, got %T", result)
	}
}

// --- Program lowering tests ---

func TestLowerProgram_Simple(t *testing.T) {
	// A simple program with one function: func add(x, y) = x + y
	lamID := uint64(100)
	cti := makeCTI(map[uint64]types.Type{
		lamID: &types.TFunc2{
			Params: []types.Type{types.TInt, types.TInt},
			Return: types.TInt,
		},
	})

	coreProg := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				CoreNode: core.CoreNode{NodeID: 1},
				Name:     "add",
				Value: &core.Lambda{
					CoreNode: core.CoreNode{NodeID: lamID},
					Params:   []string{"x", "y"},
					Body: &core.BinOp{
						CoreNode: core.CoreNode{NodeID: 101},
						Op:       "+",
						Left:     coreVar(102, "x"),
						Right:    coreVar(103, "y"),
					},
				},
				Body: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.UnitLit},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"add": {Name: "add", IsExport: true},
		},
	}

	prog, err := LowerProgram(coreProg, cti, nil, "main")
	if err != nil {
		t.Fatalf("LowerProgram failed: %v", err)
	}

	if prog.Package != "main" {
		t.Errorf("expected package main, got %s", prog.Package)
	}
	if len(prog.FuncDecls) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.FuncDecls))
	}

	fd := prog.FuncDecls[0]
	if fd.Name != "add" {
		t.Errorf("expected func name add, got %s", fd.Name)
	}
	if !fd.Exported {
		t.Error("expected exported")
	}
	if len(fd.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fd.Params))
	}
	if fd.Params[0].Type.GoString() != "int64" {
		t.Errorf("expected param type int64, got %s", fd.Params[0].Type.GoString())
	}
	if fd.ReturnType.GoString() != "int64" {
		t.Errorf("expected return type int64, got %s", fd.ReturnType.GoString())
	}
}

// TestLower_PreservesTmpBindings — M4_LOWER_TMP_SCOPE regression test.
//
// Reproduces the "$tmpN / user-var unbound" bug observed in docparse/main.ail:
// when a Let's VALUE is an If (or Match) whose branches contain their own
// Let-chains, the inner bindings were silently dropped by lowerLetExpr. The
// fix teaches flattenValue to hoist If/Match branches into IfStmt/SwitchStmt
// with a temp variable, preserving the inner bindings as proper statements.
//
// Shape of the failing input:
//
//	let combinedBlocks =
//	  if cond then
//	    let inner = 42 in inner
//	  else
//	    0
//	in combinedBlocks
//
// Before the fix, the VarDecl for `inner` was dropped and `combinedBlocks`
// would reference an undeclared variable via an IfExpr whose Then collapsed
// to just `inner`. After the fix, the resulting statement sequence must
// bind `inner` before it is used.
func TestLower_PreservesTmpBindings(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{
		1:  types.TInt, // outer Let type
		2:  types.TBool,
		3:  types.TInt, // inner Let type
		4:  types.TInt,
		5:  types.TInt,
		6:  types.TInt,
		10: types.TInt,
	})

	// Core:
	//   Let combinedBlocks = If(cond, Let(inner = 42 in inner), 0)
	//       in combinedBlocks
	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "inner",
		Value:    litInt(4, 42),
		Body:     coreVar(5, "inner"),
	}
	ifExpr := &core.If{
		CoreNode: core.CoreNode{NodeID: 10},
		Cond:     litBool(2, true),
		Then:     innerLet,
		Else:     litInt(6, 0),
	}
	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "combinedBlocks",
		Value:    ifExpr,
		Body:     coreVar(11, "combinedBlocks"),
	}

	stmts, ret := FlattenBlock(outerLet, cti)

	// Collect every variable name that is declared or assigned anywhere
	// in the emitted statement sequence (including inside IfStmt branches).
	declared := map[string]bool{}
	var collectDecls func(ss []stmt.Stmt)
	collectDecls = func(ss []stmt.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case stmt.VarDecl:
				declared[s.Name] = true
			case stmt.AssignStmt:
				declared[s.Name] = true
			case stmt.IfStmt:
				collectDecls(s.Then)
				collectDecls(s.Else)
			case stmt.SwitchStmt:
				for _, c := range s.Cases {
					collectDecls(c.Body)
				}
				collectDecls(s.Default)
			}
		}
	}
	collectDecls(stmts)

	// Collect every variable name REFERENCED in the emitted statements/return.
	referenced := map[string]bool{}
	var collectRefsExpr func(e stmt.Expr)
	var collectRefsStmts func(ss []stmt.Stmt)
	collectRefsExpr = func(e stmt.Expr) {
		switch e := e.(type) {
		case stmt.VarRef:
			referenced[e.Name] = true
		case stmt.BinOp:
			collectRefsExpr(e.Left)
			collectRefsExpr(e.Right)
		case stmt.UnOp:
			collectRefsExpr(e.Operand)
		case stmt.IfExpr:
			collectRefsExpr(e.Cond)
			collectRefsExpr(e.Then)
			collectRefsExpr(e.Else)
		case stmt.Call:
			collectRefsExpr(e.Func)
			for _, a := range e.Args {
				collectRefsExpr(a)
			}
		}
	}
	collectRefsStmts = func(ss []stmt.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case stmt.VarDecl:
				collectRefsExpr(s.Value)
			case stmt.AssignStmt:
				collectRefsExpr(s.Value)
			case stmt.ExprStmt:
				collectRefsExpr(s.Value)
			case stmt.ReturnStmt:
				collectRefsExpr(s.Value)
			case stmt.IfStmt:
				collectRefsExpr(s.Cond)
				collectRefsStmts(s.Then)
				collectRefsStmts(s.Else)
			case stmt.SwitchStmt:
				collectRefsExpr(s.Scrutinee)
				for _, c := range s.Cases {
					collectRefsStmts(c.Body)
				}
				collectRefsStmts(s.Default)
			}
		}
	}
	collectRefsStmts(stmts)
	if ret != nil {
		collectRefsExpr(ret)
	}

	// The critical assertion: every referenced variable must have a
	// corresponding declaration somewhere. Before the fix, `inner` is
	// referenced but never declared because lowerLetExpr dropped it.
	for name := range referenced {
		if !declared[name] {
			t.Errorf("variable %q referenced but never declared; "+
				"stmts=%#v ret=%#v", name, stmts, ret)
		}
	}

	// Targeted check: `inner` must be declared. Without this guardrail
	// the loop above could be bypassed if the lowering accidentally
	// rewrote the reference to something else.
	if !declared["inner"] {
		t.Errorf("expected `inner` to be declared in lowered statements; "+
			"got stmts=%#v ret=%#v", stmts, ret)
	}
}
