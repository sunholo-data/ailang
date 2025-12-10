package block

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// Helper to create a simple Let expression
func mkLet(name string, value, body core.CoreExpr) *core.Let {
	return &core.Let{
		Name:  name,
		Value: value,
		Body:  body,
	}
}

// Helper to create a variable reference
func mkVar(name string) *core.Var {
	return &core.Var{Name: name}
}

// Helper to create an int literal
func mkInt(n int64) *core.Lit {
	return &core.Lit{Value: n}
}

func TestLower_NonLetExpression(t *testing.T) {
	// Non-let expressions produce empty block with just FinalExpr
	expr := mkVar("x")

	block := Lower(expr)

	if !block.IsEmpty() {
		t.Errorf("expected empty block for non-Let, got %d stmts", len(block.Stmts))
	}
	if block.FinalExpr != expr {
		t.Errorf("expected FinalExpr to be original expr")
	}
}

func TestLower_SingleLet(t *testing.T) {
	// let x = 1 in x
	expr := mkLet("x", mkInt(1), mkVar("x"))

	block := Lower(expr)

	if block.Len() != 1 {
		t.Fatalf("expected 1 stmt, got %d", block.Len())
	}
	if block.Stmts[0].Name != "x" {
		t.Errorf("expected stmt name 'x', got %q", block.Stmts[0].Name)
	}
	lit, ok := block.Stmts[0].Value.(*core.Lit)
	if !ok || lit.Value != int64(1) {
		t.Errorf("expected stmt value to be Lit(1), got %T", block.Stmts[0].Value)
	}
	v, ok := block.FinalExpr.(*core.Var)
	if !ok || v.Name != "x" {
		t.Errorf("expected FinalExpr to be Var(x), got %T", block.FinalExpr)
	}
}

func TestLower_NestedLets_FlattenChain(t *testing.T) {
	// let x = 1 in let y = 2 in let z = 3 in x + y + z
	body := &core.BinOp{Op: "+", Left: mkVar("x"), Right: mkVar("y")}
	expr := mkLet("x", mkInt(1),
		mkLet("y", mkInt(2),
			mkLet("z", mkInt(3), body)))

	block := Lower(expr)

	if block.Len() != 3 {
		t.Fatalf("expected 3 stmts, got %d", block.Len())
	}

	// Check evaluation order preserved
	expected := []struct {
		name string
		val  int64
	}{
		{"x", 1},
		{"y", 2},
		{"z", 3},
	}
	for i, exp := range expected {
		if block.Stmts[i].Name != exp.name {
			t.Errorf("stmt[%d] name: expected %q, got %q", i, exp.name, block.Stmts[i].Name)
		}
		lit, ok := block.Stmts[i].Value.(*core.Lit)
		if !ok || lit.Value != exp.val {
			t.Errorf("stmt[%d] value: expected Lit(%d), got %T", i, exp.val, block.Stmts[i].Value)
		}
	}

	// FinalExpr should be the BinOp body
	if _, ok := block.FinalExpr.(*core.BinOp); !ok {
		t.Errorf("expected FinalExpr to be BinOp, got %T", block.FinalExpr)
	}
}

func TestLower_NestedLetInValue_Preserved(t *testing.T) {
	// let x = (let y = 1 in y) in x
	// The inner Let should NOT be flattened - it's in Value position
	innerLet := mkLet("y", mkInt(1), mkVar("y"))
	expr := mkLet("x", innerLet, mkVar("x"))

	block := Lower(expr)

	if block.Len() != 1 {
		t.Fatalf("expected 1 stmt (outer let only), got %d", block.Len())
	}
	if block.Stmts[0].Name != "x" {
		t.Errorf("expected stmt name 'x', got %q", block.Stmts[0].Name)
	}

	// Value should still be the inner Let (not flattened)
	valueLet, ok := block.Stmts[0].Value.(*core.Let)
	if !ok {
		t.Fatalf("expected Value to be Let (preserved), got %T", block.Stmts[0].Value)
	}
	if valueLet.Name != "y" {
		t.Errorf("expected inner let name 'y', got %q", valueLet.Name)
	}
}

func TestLower_DeeplyNestedChain(t *testing.T) {
	// Test with 10 nested lets - should all flatten to 10 stmts
	// Build: let v0 = 0 in let v1 = 1 in ... let v9 = 9 in v0
	var expr core.CoreExpr = mkVar("v0")
	for i := 9; i >= 0; i-- {
		name := "v" + string(rune('0'+i))
		expr = mkLet(name, mkInt(int64(i)), expr)
	}

	block := Lower(expr)

	if block.Len() != 10 {
		t.Fatalf("expected 10 stmts, got %d", block.Len())
	}

	// Verify order (v0, v1, ..., v9)
	for i := 0; i < 10; i++ {
		expected := "v" + string(rune('0'+i))
		if block.Stmts[i].Name != expected {
			t.Errorf("stmt[%d] name: expected %q, got %q", i, expected, block.Stmts[i].Name)
		}
	}
}

func TestLower_AppExpression(t *testing.T) {
	// Function application should return empty block
	expr := &core.App{
		Func: mkVar("f"),
		Args: []core.CoreExpr{mkVar("x")},
	}

	block := Lower(expr)

	if !block.IsEmpty() {
		t.Errorf("expected empty block for App, got %d stmts", block.Len())
	}
	if _, ok := block.FinalExpr.(*core.App); !ok {
		t.Errorf("expected FinalExpr to be App, got %T", block.FinalExpr)
	}
}

func TestLower_IfExpression(t *testing.T) {
	// If expression should return empty block
	expr := &core.If{
		Cond: mkVar("c"),
		Then: mkInt(1),
		Else: mkInt(2),
	}

	block := Lower(expr)

	if !block.IsEmpty() {
		t.Errorf("expected empty block for If, got %d stmts", block.Len())
	}
	if _, ok := block.FinalExpr.(*core.If); !ok {
		t.Errorf("expected FinalExpr to be If, got %T", block.FinalExpr)
	}
}

func TestLower_LetWithIfBody(t *testing.T) {
	// let x = 1 in if c then x else 0
	ifExpr := &core.If{Cond: mkVar("c"), Then: mkVar("x"), Else: mkInt(0)}
	expr := mkLet("x", mkInt(1), ifExpr)

	block := Lower(expr)

	if block.Len() != 1 {
		t.Fatalf("expected 1 stmt, got %d", block.Len())
	}
	if _, ok := block.FinalExpr.(*core.If); !ok {
		t.Errorf("expected FinalExpr to be If, got %T", block.FinalExpr)
	}
}

func TestLowerLetRec_SingleBinding(t *testing.T) {
	// letrec f = \x.f(x) in f(0)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "f", Value: mkVar("lambda_placeholder")},
		},
		Body: &core.App{Func: mkVar("f"), Args: []core.CoreExpr{mkInt(0)}},
	}

	block := LowerLetRec(letrec)

	if block.Len() != 1 {
		t.Fatalf("expected 1 stmt, got %d", block.Len())
	}
	if block.Stmts[0].Name != "f" {
		t.Errorf("expected stmt name 'f', got %q", block.Stmts[0].Name)
	}
	if _, ok := block.FinalExpr.(*core.App); !ok {
		t.Errorf("expected FinalExpr to be App, got %T", block.FinalExpr)
	}
}

func TestLowerLetRec_MutualRecursion(t *testing.T) {
	// letrec even = ..., odd = ... in even(10)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "even", Value: mkVar("even_impl")},
			{Name: "odd", Value: mkVar("odd_impl")},
		},
		Body: &core.App{Func: mkVar("even"), Args: []core.CoreExpr{mkInt(10)}},
	}

	block := LowerLetRec(letrec)

	if block.Len() != 2 {
		t.Fatalf("expected 2 stmts, got %d", block.Len())
	}

	// Order should be preserved
	if block.Stmts[0].Name != "even" {
		t.Errorf("expected first stmt name 'even', got %q", block.Stmts[0].Name)
	}
	if block.Stmts[1].Name != "odd" {
		t.Errorf("expected second stmt name 'odd', got %q", block.Stmts[1].Name)
	}
}

func TestBlock_IsEmpty(t *testing.T) {
	empty := &Block{FinalExpr: mkInt(1)}
	nonEmpty := &Block{Stmts: []Stmt{{Name: "x", Value: mkInt(1)}}, FinalExpr: mkVar("x")}

	if !empty.IsEmpty() {
		t.Error("expected IsEmpty() = true for empty block")
	}
	if nonEmpty.IsEmpty() {
		t.Error("expected IsEmpty() = false for non-empty block")
	}
}

func TestBlock_Len(t *testing.T) {
	block := &Block{
		Stmts: []Stmt{
			{Name: "a", Value: mkInt(1)},
			{Name: "b", Value: mkInt(2)},
			{Name: "c", Value: mkInt(3)},
		},
		FinalExpr: mkVar("a"),
	}

	if block.Len() != 3 {
		t.Errorf("expected Len() = 3, got %d", block.Len())
	}
}
