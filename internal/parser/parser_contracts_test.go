package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

func TestParseContractBlocks_RequiresOnly(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
requires { x >= 0 }
{
  x + 1
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

	if fn.Name != "foo" {
		t.Errorf("expected function name 'foo', got %q", fn.Name)
	}

	// Check contracts
	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(fn.Properties))
	}

	contract := fn.Properties[0]
	if contract.Kind != ast.RequiresKind {
		t.Errorf("expected RequiresKind, got %v", contract.Kind)
	}

	// Check the predicate expression (x >= 0)
	binOp, ok := contract.Expr.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp, got %T", contract.Expr)
	}

	if binOp.Op != ">=" {
		t.Errorf("expected '>=', got %q", binOp.Op)
	}
}

func TestParseContractBlocks_EnsuresOnly(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
ensures { result > x }
{
  x + 1
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

	// Check contracts
	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(fn.Properties))
	}

	contract := fn.Properties[0]
	if contract.Kind != ast.EnsuresKind {
		t.Errorf("expected EnsuresKind, got %v", contract.Kind)
	}

	// Check the predicate uses 'result' identifier
	binOp, ok := contract.Expr.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp, got %T", contract.Expr)
	}

	left, ok := binOp.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier on left, got %T", binOp.Left)
	}

	if left.Name != "result" {
		t.Errorf("expected 'result', got %q", left.Name)
	}
}

func TestParseContractBlocks_BothRequiresAndEnsures(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
requires { x >= 0 }
ensures { result > x }
{
  x + 1
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

	// Check both contracts
	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(fn.Properties))
	}

	// First should be requires
	if fn.Properties[0].Kind != ast.RequiresKind {
		t.Errorf("expected first contract to be RequiresKind, got %v", fn.Properties[0].Kind)
	}

	// Second should be ensures
	if fn.Properties[1].Kind != ast.EnsuresKind {
		t.Errorf("expected second contract to be EnsuresKind, got %v", fn.Properties[1].Kind)
	}
}

func TestParseContractBlocks_MultiplePredicates(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
requires { x >= 0, x < 100 }
{
  x + 1
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

	// Check contracts (both should be RequiresKind)
	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 contracts (multiple requires predicates), got %d", len(fn.Properties))
	}

	for i, contract := range fn.Properties {
		if contract.Kind != ast.RequiresKind {
			t.Errorf("contract %d: expected RequiresKind, got %v", i, contract.Kind)
		}
	}
}

func TestParseContractBlocks_NoContracts(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
{
  x + 1
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

	// Should have no contracts
	if len(fn.Properties) != 0 {
		t.Errorf("expected 0 contracts, got %d", len(fn.Properties))
	}
}

func TestParseContractBlocks_EmptyBlock(t *testing.T) {
	input := `
export func foo(x: int) -> int ! {}
requires {}
{
  x + 1
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

	// Empty requires block should result in no contracts
	if len(fn.Properties) != 0 {
		t.Errorf("expected 0 contracts for empty block, got %d", len(fn.Properties))
	}
}

func TestParseContractBlocks_DuplicateEnsures(t *testing.T) {
	// AIs commonly write two separate ensures blocks instead of comma-separating.
	// The parser should recover by merging them and reporting a warning.
	input := `
export func f(x: int) -> int ! {}
ensures { result >= 0 }
ensures { result <= 100 }
{ x }
`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	// Should have a warning but still parse successfully
	if len(file.Decls) != 1 {
		for _, err := range p.Errors() {
			t.Logf("parser error: %v", err)
		}
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", file.Decls[0])
	}

	// Both ensures predicates should be merged (2 total)
	ensuresCount := 0
	for _, prop := range fn.Properties {
		if prop.Kind == ast.EnsuresKind {
			ensuresCount++
		}
	}
	if ensuresCount != 2 {
		t.Errorf("expected 2 ensures predicates (merged from duplicate blocks), got %d", ensuresCount)
	}

	// Should have reported a PAR_DUPLICATE_ENSURES warning
	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "PAR_DUPLICATE_ENSURES") || strings.Contains(err.Error(), "only one ensures block") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PAR_DUPLICATE_ENSURES warning in parser errors")
	}
}

func TestParseContractBlocks_DuplicateRequires(t *testing.T) {
	input := `
export func g(x: int, y: int) -> int ! {}
requires { x >= 0 }
requires { y >= 0 }
ensures { result >= 0 }
{ x + y }
`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(file.Decls) != 1 {
		for _, err := range p.Errors() {
			t.Logf("parser error: %v", err)
		}
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", file.Decls[0])
	}

	// Should have 2 requires + 1 ensures (merged)
	requiresCount := 0
	ensuresCount := 0
	for _, prop := range fn.Properties {
		if prop.Kind == ast.RequiresKind {
			requiresCount++
		}
		if prop.Kind == ast.EnsuresKind {
			ensuresCount++
		}
	}
	if requiresCount != 2 {
		t.Errorf("expected 2 requires predicates (merged), got %d", requiresCount)
	}
	if ensuresCount != 1 {
		t.Errorf("expected 1 ensures predicate, got %d", ensuresCount)
	}
}
