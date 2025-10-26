package testing

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

func TestCollector_EmptyFile(t *testing.T) {
	input := `
	func add(x: int, y: int) -> int {
		x + y
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(suite.Tests))
	}

	if len(suite.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(suite.Properties))
	}
}

func TestCollector_TopLevelTest(t *testing.T) {
	input := `
	test "simple test" {
		2 + 2 == 4
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(suite.Tests))
	}

	test := suite.Tests[0]
	if test.Name != "simple test" {
		t.Errorf("expected name 'simple test', got %q", test.Name)
	}

	if test.IsInline {
		t.Errorf("expected top-level test (IsInline=false), got IsInline=true")
	}

	if test.FunctionCtx != "" {
		t.Errorf("expected no function context, got %q", test.FunctionCtx)
	}

	if len(test.Body) != 1 {
		t.Errorf("expected 1 expression in body, got %d", len(test.Body))
	}
}

func TestCollector_TopLevelProperty(t *testing.T) {
	input := `
	property "always positive" {
		forall(n: int) => n + 1 > n
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(suite.Properties))
	}

	prop := suite.Properties[0]
	if prop.Name != "always positive" {
		t.Errorf("expected name 'always positive', got %q", prop.Name)
	}

	if prop.IsInline {
		t.Errorf("expected top-level property (IsInline=false), got IsInline=true")
	}

	if prop.Property == nil {
		t.Fatal("expected property specification, got nil")
	}
}

func TestCollector_InlineTests(t *testing.T) {
	t.Skip("Inline tests (tests [...]) parser not yet implemented - Days 1-2")

	input := `
	func factorial(n: int) -> int
	tests [
		factorial(0) == 1,
		factorial(5) == 120
	]
	{
		if n <= 1 then 1 else n * factorial(n - 1)
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 2 {
		t.Fatalf("expected 2 inline tests, got %d", len(suite.Tests))
	}

	// Check first test
	test1 := suite.Tests[0]
	if test1.Name != "factorial_test_1" {
		t.Errorf("expected name 'factorial_test_1', got %q", test1.Name)
	}

	if !test1.IsInline {
		t.Errorf("expected inline test (IsInline=true), got IsInline=false")
	}

	if test1.FunctionCtx != "factorial" {
		t.Errorf("expected function context 'factorial', got %q", test1.FunctionCtx)
	}

	// Check second test
	test2 := suite.Tests[1]
	if test2.Name != "factorial_test_2" {
		t.Errorf("expected name 'factorial_test_2', got %q", test2.Name)
	}
}

func TestCollector_InlineProperties(t *testing.T) {
	input := `
	func abs(x: int) -> int
	properties [
		forall(x: int) => abs(x) >= 0,
		forall(x: int) => abs(-x) == abs(x)
	]
	{
		if x < 0 then -x else x
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Properties) != 2 {
		t.Fatalf("expected 2 inline properties, got %d", len(suite.Properties))
	}

	// Check first property
	prop1 := suite.Properties[0]
	if prop1.Name != "abs_property_1" {
		t.Errorf("expected name 'abs_property_1', got %q", prop1.Name)
	}

	if !prop1.IsInline {
		t.Errorf("expected inline property (IsInline=true), got IsInline=false")
	}

	// Check second property
	prop2 := suite.Properties[1]
	if prop2.Name != "abs_property_2" {
		t.Errorf("expected name 'abs_property_2', got %q", prop2.Name)
	}
}

func TestCollector_MixedTestsAndProperties(t *testing.T) {
	input := `
	func fibonacci(n: int) -> int {
		if n <= 1 then n else fibonacci(n - 1) + fibonacci(n - 2)
	}

	test "fibonacci basics" {
		fibonacci(0) == 0;
		fibonacci(1) == 1
	}

	property "fibonacci always non-negative" {
		forall(n: int) => fibonacci(n) >= 0
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 1 {
		t.Errorf("expected 1 test, got %d", len(suite.Tests))
	}

	if len(suite.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(suite.Properties))
	}

	if suite.ModulePath != "test.ail" {
		t.Errorf("expected module path 'test.ail', got %q", suite.ModulePath)
	}
}

func TestCollector_MultipleTopLevelTests(t *testing.T) {
	input := `
	test "test one" {
		1 + 1 == 2
	}

	test "test two" {
		2 + 2 == 4
	}

	test "test three" {
		3 + 3 == 6
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(suite.Tests))
	}

	expectedNames := []string{"test one", "test two", "test three"}
	for i, test := range suite.Tests {
		if test.Name != expectedNames[i] {
			t.Errorf("test %d: expected name %q, got %q", i, expectedNames[i], test.Name)
		}
	}
}

func TestCollector_TestWithAssertions(t *testing.T) {
	input := `
	test "test with assertions" {
		assert 2 + 2 == 4;
		assert 3 > 2
	}
	`

	file := parseInput(t, input)
	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(suite.Tests))
	}

	test := suite.Tests[0]
	if len(test.Body) != 2 {
		t.Errorf("expected 2 expressions in body (2 assertions), got %d", len(test.Body))
	}

	// Check that assertions are in the body
	for i, expr := range test.Body {
		if _, ok := expr.(*ast.AssertStmt); !ok {
			t.Errorf("expression %d: expected AssertStmt, got %T", i, expr)
		}
	}
}

// Helper function to parse AILANG code
func parseInput(t *testing.T, input string) *ast.File {
	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors:\n")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
	}

	return file
}
