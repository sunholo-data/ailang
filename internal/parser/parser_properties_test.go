package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestParseProperties_SingleProperty tests parsing a function with one property
func TestParseProperties_SingleProperty(t *testing.T) {
	input := `
	func quicksort(list: [int]) -> [int]
		properties [
			forall(xs: [int]) => length(quicksort(xs)) == length(xs)
		]
	{
		list
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

	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(fn.Properties))
	}

	prop := fn.Properties[0]

	// Check binders: forall(xs: [int])
	if len(prop.Binders) != 1 {
		t.Fatalf("expected 1 binder, got %d", len(prop.Binders))
	}

	binder := prop.Binders[0]
	if binder.Name != "xs" {
		t.Errorf("expected binder name 'xs', got %q", binder.Name)
	}

	// Type should be [int] - DX-17 Phase 2: Now normalized to TypeApp("list", [int])
	typeApp, ok := binder.Type.(*ast.TypeApp)
	if !ok {
		t.Fatalf("expected TypeApp for list type, got %T", binder.Type)
	}
	if typeApp.Constructor != "list" {
		t.Errorf("expected constructor 'list', got %q", typeApp.Constructor)
	}
	// Check element type is int
	if len(typeApp.Args) != 1 {
		t.Fatalf("expected 1 type arg, got %d", len(typeApp.Args))
	}
	simpleType, ok := typeApp.Args[0].(*ast.SimpleType)
	if !ok {
		t.Errorf("expected SimpleType for element, got %T", typeApp.Args[0])
	}
	if simpleType != nil && simpleType.Name != "int" {
		t.Errorf("expected element type 'int', got %q", simpleType.Name)
	}

	// Check that property expression exists (we don't validate the expr tree here)
	if prop.Expr == nil {
		t.Error("expected property expression, got nil")
	}
}

// TestParseProperties_MultipleProperties tests parsing multiple properties
func TestParseProperties_MultipleProperties(t *testing.T) {
	input := `
	func sort(list: [int]) -> [int]
		properties [
			forall(xs: [int]) => length(sort(xs)) == length(xs),
			forall(xs: [int]) => sort(sort(xs)) == sort(xs)
		]
	{
		list
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

	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(fn.Properties))
	}

	// Check first property
	prop1 := fn.Properties[0]
	if len(prop1.Binders) != 1 {
		t.Errorf("expected 1 binder in property 1, got %d", len(prop1.Binders))
	}
	if prop1.Binders[0].Name != "xs" {
		t.Errorf("expected binder name 'xs', got %q", prop1.Binders[0].Name)
	}

	// Check second property
	prop2 := fn.Properties[1]
	if len(prop2.Binders) != 1 {
		t.Errorf("expected 1 binder in property 2, got %d", len(prop2.Binders))
	}
	if prop2.Binders[0].Name != "xs" {
		t.Errorf("expected binder name 'xs', got %q", prop2.Binders[0].Name)
	}
}

// TestParseProperties_MultipleBinders tests forall with multiple binders
func TestParseProperties_MultipleBinders(t *testing.T) {
	input := `
	func max(a: int, b: int) -> int
		properties [
			forall(x: int, y: int) => max(x, y) >= x,
			forall(x: int, y: int) => max(x, y) >= y
		]
	{
		if a > b then a else b
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

	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(fn.Properties))
	}

	// Check first property has 2 binders
	prop1 := fn.Properties[0]
	if len(prop1.Binders) != 2 {
		t.Fatalf("expected 2 binders, got %d", len(prop1.Binders))
	}

	if prop1.Binders[0].Name != "x" {
		t.Errorf("expected binder 1 name 'x', got %q", prop1.Binders[0].Name)
	}
	if prop1.Binders[1].Name != "y" {
		t.Errorf("expected binder 2 name 'y', got %q", prop1.Binders[1].Name)
	}

	// Check second property has 2 binders
	prop2 := fn.Properties[1]
	if len(prop2.Binders) != 2 {
		t.Fatalf("expected 2 binders, got %d", len(prop2.Binders))
	}

	if prop2.Binders[0].Name != "x" {
		t.Errorf("expected binder 1 name 'x', got %q", prop2.Binders[0].Name)
	}
	if prop2.Binders[1].Name != "y" {
		t.Errorf("expected binder 2 name 'y', got %q", prop2.Binders[1].Name)
	}
}

// TestParseProperties_EmptyBlock tests empty properties block
func TestParseProperties_EmptyBlock(t *testing.T) {
	input := `
	func identity(x: int) -> int
		properties []
	{
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

	if len(fn.Properties) != 0 {
		t.Errorf("expected 0 properties in empty block, got %d", len(fn.Properties))
	}
}

// TestParseProperties_WithNewlines tests properties with newlines
func TestParseProperties_WithNewlines(t *testing.T) {
	input := `
	func reverse(list: [int]) -> [int]
		properties [
			forall(xs: [int]) => length(reverse(xs)) == length(xs),

			forall(xs: [int]) => reverse(reverse(xs)) == xs
		]
	{
		list
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

	if len(fn.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(fn.Properties))
	}
}

// TestParseProperties_TrailingComma tests properties with trailing comma
func TestParseProperties_TrailingComma(t *testing.T) {
	input := `
	func abs(x: int) -> int
		properties [
			forall(n: int) => abs(n) >= 0,
		]
	{
		if x < 0 then -x else x
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

	if len(fn.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(fn.Properties))
	}
}

// TestParseProperties_ComplexTypes tests properties with complex types
func TestParseProperties_ComplexTypes(t *testing.T) {
	t.Skip("TODO: Function types in binders (int -> bool) need special parsing")
	// This test is skipped because parsing function types like `int -> bool`
	// in forall binders requires handling the -> arrow carefully to not
	// conflict with the function return type arrow.
	// Example: forall(f: int -> bool, xs: [int]) => ...
	// This will be fixed in Day 3 when we add more sophisticated type parsing.
}

// TestParseProperties_NoProperties tests function without properties
func TestParseProperties_NoProperties(t *testing.T) {
	input := `
	func add(a: int, b: int) -> int {
		a + b
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

	if len(fn.Properties) != 0 {
		t.Errorf("expected no properties, got %d", len(fn.Properties))
	}
}

// TestParseProperties_WithTests tests function with both tests and properties
func TestParseProperties_WithTests(t *testing.T) {
	input := `
	func factorial(n: int) -> int
		tests [
			(0, 1),
			(5, 120)
		]
		properties [
			forall(n: int) => factorial(n) >= 1
		]
	{
		if n <= 1 then 1 else n * factorial(n - 1)
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
		t.Errorf("expected 2 tests, got %d", len(fn.Tests))
	}

	if len(fn.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(fn.Properties))
	}
}
