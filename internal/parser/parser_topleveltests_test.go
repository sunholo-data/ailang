package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestParseTestDecl_Simple tests parsing a simple test block
func TestParseTestDecl_Simple(t *testing.T) {
	input := `
	test "simple test" {
		2 + 2 == 4
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if len(file.Decls) != 1 {
		for i, decl := range file.Decls {
			t.Logf("  Decl[%d]: %T", i, decl)
		}
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	testDecl, ok := file.Decls[0].(*ast.TestDecl)
	if !ok {
		t.Fatalf("expected TestDecl, got %T", file.Decls[0])
	}

	if testDecl.Name != "simple test" {
		t.Errorf("expected name 'simple test', got %q", testDecl.Name)
	}

	if len(testDecl.Body) != 1 {
		t.Fatalf("expected 1 expression in body, got %d", len(testDecl.Body))
	}
}

// TestParseTestDecl_MultipleStatements tests parsing test with multiple statements
func TestParseTestDecl_MultipleStatements(t *testing.T) {
	input := `
	test "multiple statements" {
		1 + 1 == 2;
		2 * 2 == 4;
		3 - 1 == 2
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)

	if len(testDecl.Body) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(testDecl.Body))
	}
}

// TestParseTestDecl_WithAssert tests parsing test with assert statements
func TestParseTestDecl_WithAssert(t *testing.T) {
	input := `
	test "with assertions" {
		assert 1 + 1 == 2;
		assert factorial(5) == 120
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)

	if len(testDecl.Body) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(testDecl.Body))
	}

	// Check first statement is assert
	assert1, ok := testDecl.Body[0].(*ast.AssertStmt)
	if !ok {
		t.Fatalf("expected AssertStmt, got %T", testDecl.Body[0])
	}

	if assert1.Condition == nil {
		t.Error("expected assert condition, got nil")
	}

	// Check second statement is assert
	assert2, ok := testDecl.Body[1].(*ast.AssertStmt)
	if !ok {
		t.Fatalf("expected AssertStmt, got %T", testDecl.Body[1])
	}

	if assert2.Condition == nil {
		t.Error("expected assert condition, got nil")
	}
}

// TestParseTestDecl_Empty tests parsing empty test block
func TestParseTestDecl_Empty(t *testing.T) {
	input := `test "empty test" {}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)

	if len(testDecl.Body) != 0 {
		t.Errorf("expected 0 statements in empty test, got %d", len(testDecl.Body))
	}
}

// TestParseTestDecl_WithNewlines tests parsing test with newlines
func TestParseTestDecl_WithNewlines(t *testing.T) {
	input := `
	test "with newlines" {
		1 + 1 == 2

		2 * 2 == 4

		3 - 1 == 2
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)

	if len(testDecl.Body) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(testDecl.Body))
	}
}

// TestParsePropertyDecl_Simple tests parsing a simple property block
func TestParsePropertyDecl_Simple(t *testing.T) {
	input := `
	property "sort preserves length" {
		forall(xs: [int]) => length(sort(xs)) == length(xs)
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if len(file.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	propDecl, ok := file.Decls[0].(*ast.PropertyDecl)
	if !ok {
		t.Fatalf("expected PropertyDecl, got %T", file.Decls[0])
	}

	if propDecl.Name != "sort preserves length" {
		t.Errorf("expected name 'sort preserves length', got %q", propDecl.Name)
	}

	if propDecl.Property == nil {
		t.Fatal("expected property, got nil")
	}

	if len(propDecl.Property.Binders) != 1 {
		t.Errorf("expected 1 binder, got %d", len(propDecl.Property.Binders))
	}
}

// TestParsePropertyDecl_MultipleBinders tests property with multiple binders
func TestParsePropertyDecl_MultipleBinders(t *testing.T) {
	input := `
	property "max returns larger value" {
		forall(x: int, y: int) => max(x, y) >= x
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	propDecl := file.Decls[0].(*ast.PropertyDecl)

	if len(propDecl.Property.Binders) != 2 {
		t.Fatalf("expected 2 binders, got %d", len(propDecl.Property.Binders))
	}

	if propDecl.Property.Binders[0].Name != "x" {
		t.Errorf("expected binder name 'x', got %q", propDecl.Property.Binders[0].Name)
	}

	if propDecl.Property.Binders[1].Name != "y" {
		t.Errorf("expected binder name 'y', got %q", propDecl.Property.Binders[1].Name)
	}
}

// TestParseMixedDeclarations tests file with mix of declarations
func TestParseMixedDeclarations(t *testing.T) {
	input := `
	func factorial(n: int) -> int {
		if n <= 1 then 1 else n * factorial(n - 1)
	}

	test "factorial tests" {
		factorial(0) == 1;
		factorial(5) == 120
	}

	property "factorial always positive" {
		forall(n: int) => factorial(n) >= 1
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if len(file.Decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(file.Decls))
	}

	// Check first is function
	_, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Errorf("expected FuncDecl at index 0, got %T", file.Decls[0])
	}

	// Check second is test
	_, ok = file.Decls[1].(*ast.TestDecl)
	if !ok {
		t.Errorf("expected TestDecl at index 1, got %T", file.Decls[1])
	}

	// Check third is property
	_, ok = file.Decls[2].(*ast.PropertyDecl)
	if !ok {
		t.Errorf("expected PropertyDecl at index 2, got %T", file.Decls[2])
	}
}

// TestParseAssertStmt_Simple tests parsing simple assert statement
func TestParseAssertStmt_Simple(t *testing.T) {
	input := `
	test "assert test" {
		assert true
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)
	assertStmt, ok := testDecl.Body[0].(*ast.AssertStmt)
	if !ok {
		t.Fatalf("expected AssertStmt, got %T", testDecl.Body[0])
	}

	if assertStmt.Condition == nil {
		t.Error("expected condition, got nil")
	}
}

// TestParseAssertStmt_ComplexCondition tests assert with complex expression
func TestParseAssertStmt_ComplexCondition(t *testing.T) {
	input := `
	test "complex assert" {
		assert length([1, 2, 3]) == 3 && head([1, 2]) == 1
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	testDecl := file.Decls[0].(*ast.TestDecl)
	assertStmt, ok := testDecl.Body[0].(*ast.AssertStmt)
	if !ok {
		t.Fatalf("expected AssertStmt, got %T", testDecl.Body[0])
	}

	// Just verify we have a condition (don't check its structure in detail)
	if assertStmt.Condition == nil {
		t.Error("expected condition, got nil")
	}
}
