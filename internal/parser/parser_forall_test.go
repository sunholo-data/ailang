package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestParseForall_Basic(t *testing.T) {
	// forall i: 0..10 => i >= 0 should parse as ForallExpr
	input := `
export func foo(n: int) -> int ! {}
ensures { forall i: 0..10 => i >= 0 }
{
  n + 1
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", file.Decls[0])
	}

	// Check ensures contract
	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(fn.Properties))
	}

	contract := fn.Properties[0]
	if contract.Kind != ast.EnsuresKind {
		t.Errorf("expected EnsuresKind, got %v", contract.Kind)
	}

	// The predicate should be a ForallExpr
	forallExpr, ok := contract.Expr.(*ast.ForallExpr)
	if !ok {
		t.Fatalf("expected ForallExpr, got %T (%s)", contract.Expr, contract.Expr)
	}

	if forallExpr.Var != "i" {
		t.Errorf("expected variable name 'i', got %q", forallExpr.Var)
	}

	// Check lower bound is 0
	loLit, ok := forallExpr.Lo.(*ast.Literal)
	if !ok {
		t.Fatalf("expected Literal for lower bound, got %T", forallExpr.Lo)
	}
	if loLit.Value.(int64) != 0 {
		t.Errorf("expected lower bound 0, got %v", loLit.Value)
	}

	// Check upper bound is 10
	hiLit, ok := forallExpr.Hi.(*ast.Literal)
	if !ok {
		t.Fatalf("expected Literal for upper bound, got %T", forallExpr.Hi)
	}
	if hiLit.Value.(int64) != 10 {
		t.Errorf("expected upper bound 10, got %v", hiLit.Value)
	}

	// Check body is i >= 0
	body, ok := forallExpr.Body.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp for body, got %T", forallExpr.Body)
	}
	if body.Op != ">=" {
		t.Errorf("expected '>=', got %q", body.Op)
	}
}

func TestParseForall_WithExprBounds(t *testing.T) {
	// forall with expression bounds: forall i: 0..n => P(i)
	input := `
export func foo(n: int) -> int ! {}
ensures { forall i: 0..n => i >= 0 }
{
  n + 1
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	contract := fn.Properties[0]
	forallExpr, ok := contract.Expr.(*ast.ForallExpr)
	if !ok {
		t.Fatalf("expected ForallExpr, got %T", contract.Expr)
	}

	// Upper bound should be identifier 'n'
	hiIdent, ok := forallExpr.Hi.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier for upper bound, got %T", forallExpr.Hi)
	}
	if hiIdent.Name != "n" {
		t.Errorf("expected upper bound 'n', got %q", hiIdent.Name)
	}
}

func TestParseForall_InEnsuresWithResult(t *testing.T) {
	// forall in ensures with result reference
	input := `
export func foo(n: int) -> int ! {}
ensures { forall i: 0..n => result >= i }
{
  n * 2
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	contract := fn.Properties[0]
	forallExpr, ok := contract.Expr.(*ast.ForallExpr)
	if !ok {
		t.Fatalf("expected ForallExpr, got %T", contract.Expr)
	}

	// Body should reference 'result'
	body := forallExpr.Body.(*ast.BinaryOp)
	left := body.Left.(*ast.Identifier)
	if left.Name != "result" {
		t.Errorf("expected 'result' on left side, got %q", left.Name)
	}
}

func TestParseForall_String(t *testing.T) {
	forallExpr := &ast.ForallExpr{
		Var: "i",
		Lo:  &ast.Literal{Kind: ast.IntLit, Value: int64(0)},
		Hi:  &ast.Identifier{Name: "n"},
		Body: &ast.BinaryOp{
			Left:  &ast.Identifier{Name: "i"},
			Op:    ">=",
			Right: &ast.Literal{Kind: ast.IntLit, Value: int64(0)},
		},
	}

	expected := "forall i: 0..n => (i >= 0)"
	if forallExpr.String() != expected {
		t.Errorf("expected %q, got %q", expected, forallExpr.String())
	}
}

func TestParseForall_WithFuncCallBounds(t *testing.T) {
	// forall with function call bounds to test parsing stops correctly
	input := `
export func foo(xs: [int]) -> [int] ! {}
ensures { forall i: 0..10 => i >= 0 }
{
  xs
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(fn.Properties))
	}

	forallExpr, ok := fn.Properties[0].Expr.(*ast.ForallExpr)
	if !ok {
		t.Fatalf("expected ForallExpr, got %T", fn.Properties[0].Expr)
	}

	if forallExpr.Var != "i" {
		t.Errorf("expected variable name 'i', got %q", forallExpr.Var)
	}
}

func TestParseForall_MultipleContractsWithForall(t *testing.T) {
	// Multiple contracts including forall
	input := `
export func foo(n: int) -> int ! {}
requires { n >= 0 }
ensures { forall i: 0..n => i >= 0 }
{
  n + 1
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(fn.Properties))
	}

	// First is requires
	if fn.Properties[0].Kind != ast.RequiresKind {
		t.Errorf("expected first contract RequiresKind, got %v", fn.Properties[0].Kind)
	}

	// Second is ensures with forall
	if fn.Properties[1].Kind != ast.EnsuresKind {
		t.Errorf("expected second contract EnsuresKind, got %v", fn.Properties[1].Kind)
	}

	_, ok := fn.Properties[1].Expr.(*ast.ForallExpr)
	if !ok {
		t.Errorf("expected ForallExpr in ensures, got %T", fn.Properties[1].Expr)
	}
}
