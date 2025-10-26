package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestParseInlineTests_SingleArg tests parsing inline tests with single arguments
func TestParseInlineTests_SingleArg(t *testing.T) {
	input := `
	func factorial(n: int) -> int
		tests [
			(0, 1),
			(1, 1),
			(5, 120)
		]
	{
		if n == 0 then 1 else n * factorial(n - 1)
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if len(file.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(file.Decls))
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", file.Decls[0])
	}

	if fn.Name != "factorial" {
		t.Fatalf("expected function name 'factorial', got '%s'", fn.Name)
	}

	if len(fn.Tests) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(fn.Tests))
	}

	// Check first test case: (0, 1)
	test0 := fn.Tests[0]
	if len(test0.Inputs) != 1 {
		t.Errorf("test 0: expected 1 input, got %d", len(test0.Inputs))
	}
	checkIntLiteral(t, test0.Inputs[0], 0)
	checkIntLiteral(t, test0.Expected, 1)

	// Check second test case: (1, 1)
	test1 := fn.Tests[1]
	if len(test1.Inputs) != 1 {
		t.Errorf("test 1: expected 1 input, got %d", len(test1.Inputs))
	}
	checkIntLiteral(t, test1.Inputs[0], 1)
	checkIntLiteral(t, test1.Expected, 1)

	// Check third test case: (5, 120)
	test2 := fn.Tests[2]
	if len(test2.Inputs) != 1 {
		t.Errorf("test 2: expected 1 input, got %d", len(test2.Inputs))
	}
	checkIntLiteral(t, test2.Inputs[0], 5)
	checkIntLiteral(t, test2.Expected, 120)
}

// TestParseInlineTests_MultiArg tests parsing inline tests with multiple arguments
func TestParseInlineTests_MultiArg(t *testing.T) {
	input := `
	func add(x: int, y: int) -> int
		tests [
			((1, 2), 3),
			((5, 5), 10),
			((0, 0), 0)
		]
	{
		x + y
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(fn.Tests))
	}

	// Check first test: ((1, 2), 3)
	test0 := fn.Tests[0]
	if len(test0.Inputs) != 2 {
		t.Fatalf("test 0: expected 2 inputs, got %d", len(test0.Inputs))
	}
	checkIntLiteral(t, test0.Inputs[0], 1)
	checkIntLiteral(t, test0.Inputs[1], 2)
	checkIntLiteral(t, test0.Expected, 3)

	// Check second test: ((5, 5), 10)
	test1 := fn.Tests[1]
	if len(test1.Inputs) != 2 {
		t.Fatalf("test 1: expected 2 inputs, got %d", len(test1.Inputs))
	}
	checkIntLiteral(t, test1.Inputs[0], 5)
	checkIntLiteral(t, test1.Inputs[1], 5)
	checkIntLiteral(t, test1.Expected, 10)

	// Check third test: ((0, 0), 0)
	test2 := fn.Tests[2]
	if len(test2.Inputs) != 2 {
		t.Fatalf("test 2: expected 2 inputs, got %d", len(test2.Inputs))
	}
	checkIntLiteral(t, test2.Inputs[0], 0)
	checkIntLiteral(t, test2.Inputs[1], 0)
	checkIntLiteral(t, test2.Expected, 0)
}

// TestParseInlineTests_EmptyBlock tests parsing empty tests block
func TestParseInlineTests_EmptyBlock(t *testing.T) {
	input := `
	func noop() -> int
		tests []
	{
		42
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 0 {
		t.Fatalf("expected 0 test cases, got %d", len(fn.Tests))
	}
}

// TestParseInlineTests_StringInputs tests parsing tests with string literals
func TestParseInlineTests_StringInputs(t *testing.T) {
	input := `
	func greet(name: string) -> string
		tests [
			("Alice", "Hello, Alice"),
			("Bob", "Hello, Bob")
		]
	{
		concat("Hello, ", name)
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(fn.Tests))
	}

	// Check first test: ("Alice", "Hello, Alice")
	test0 := fn.Tests[0]
	checkStringLiteral(t, test0.Inputs[0], "Alice")
	checkStringLiteral(t, test0.Expected, "Hello, Alice")

	// Check second test: ("Bob", "Hello, Bob")
	test1 := fn.Tests[1]
	checkStringLiteral(t, test1.Inputs[0], "Bob")
	checkStringLiteral(t, test1.Expected, "Hello, Bob")
}

// TestParseInlineTests_BooleanInputs tests parsing tests with boolean literals
func TestParseInlineTests_BooleanInputs(t *testing.T) {
	input := `
	func negate(b: bool) -> bool
		tests [
			(true, false),
			(false, true)
		]
	{
		not b
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(fn.Tests))
	}

	// Check first test: (true, false)
	test0 := fn.Tests[0]
	checkBoolLiteral(t, test0.Inputs[0], true)
	checkBoolLiteral(t, test0.Expected, false)

	// Check second test: (false, true)
	test1 := fn.Tests[1]
	checkBoolLiteral(t, test1.Inputs[0], false)
	checkBoolLiteral(t, test1.Expected, true)
}

// TestParseInlineTests_ListInputs tests parsing tests with list literals
func TestParseInlineTests_ListInputs(t *testing.T) {
	input := `
	func length(list: [int]) -> int
		tests [
			([], 0),
			([1], 1),
			([1, 2, 3], 3)
		]
	{
		list_len(list)
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(fn.Tests))
	}

	// Check first test: ([], 0)
	test0 := fn.Tests[0]
	list0 := test0.Inputs[0].(*ast.List)
	if len(list0.Elements) != 0 {
		t.Errorf("test 0: expected empty list, got %d elements", len(list0.Elements))
	}
	checkIntLiteral(t, test0.Expected, 0)

	// Check second test: ([1], 1)
	test1 := fn.Tests[1]
	list1 := test1.Inputs[0].(*ast.List)
	if len(list1.Elements) != 1 {
		t.Errorf("test 1: expected 1 element, got %d", len(list1.Elements))
	}
	checkIntLiteral(t, test1.Expected, 1)

	// Check third test: ([1, 2, 3], 3)
	test2 := fn.Tests[2]
	list2 := test2.Inputs[0].(*ast.List)
	if len(list2.Elements) != 3 {
		t.Errorf("test 2: expected 3 elements, got %d", len(list2.Elements))
	}
	checkIntLiteral(t, test2.Expected, 3)
}

// TestParseInlineTests_WithNewlines tests parsing with newlines between test cases
func TestParseInlineTests_WithNewlines(t *testing.T) {
	input := `
	func double(x: int) -> int
		tests [
			(1, 2),

			(2, 4),


			(3, 6)
		]
	{
		x * 2
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(fn.Tests))
	}
}

// TestParseInlineTests_TrailingComma tests parsing with trailing comma
func TestParseInlineTests_TrailingComma(t *testing.T) {
	input := `
	func square(x: int) -> int
		tests [
			(1, 1),
			(2, 4),
			(3, 9),
		]
	{
		x * x
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(fn.Tests))
	}
}

// TestParseInlineTests_ComplexExpressions tests with complex expressions
func TestParseInlineTests_ComplexExpressions(t *testing.T) {
	input := `
	func calculate(x: int, y: int) -> int
		tests [
			((1 + 1, 2 * 2), 2 + 4),
			((3, 4), 3 + 4)
		]
	{
		x + y
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if len(fn.Tests) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(fn.Tests))
	}

	// Just verify structure - not checking expression details
	if len(fn.Tests[0].Inputs) != 2 {
		t.Errorf("test 0: expected 2 inputs, got %d", len(fn.Tests[0].Inputs))
	}
	if len(fn.Tests[1].Inputs) != 2 {
		t.Errorf("test 1: expected 2 inputs, got %d", len(fn.Tests[1].Inputs))
	}
}

// TestParseInlineTests_NoTests tests function without tests block
func TestParseInlineTests_NoTests(t *testing.T) {
	input := `
	func identity(x: int) -> int {
		x
	}
	`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	fn := file.Decls[0].(*ast.FuncDecl)
	if fn.Tests != nil && len(fn.Tests) != 0 {
		t.Fatalf("expected no tests, got %d", len(fn.Tests))
	}
}

// Helper functions for assertions

func checkIntLiteral(t *testing.T, expr ast.Expr, expected int) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Errorf("expected Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.IntLit {
		t.Errorf("expected IntLit, got %v", lit.Kind)
		return
	}
	// Handle both int and int64 (lexer returns int64)
	var val int
	switch v := lit.Value.(type) {
	case int:
		val = v
	case int64:
		val = int(v)
	default:
		t.Errorf("expected int/int64 value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("expected value %d, got %d", expected, val)
	}
}

func checkStringLiteral(t *testing.T, expr ast.Expr, expected string) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Errorf("expected Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.StringLit {
		t.Errorf("expected StringLit, got %v", lit.Kind)
		return
	}
	val, ok := lit.Value.(string)
	if !ok {
		t.Errorf("expected string value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("expected value %q, got %q", expected, val)
	}
}

func checkBoolLiteral(t *testing.T, expr ast.Expr, expected bool) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Errorf("expected Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.BoolLit {
		t.Errorf("expected BoolLit, got %v", lit.Kind)
		return
	}
	val, ok := lit.Value.(bool)
	if !ok {
		t.Errorf("expected bool value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("expected value %t, got %t", expected, val)
	}
}
