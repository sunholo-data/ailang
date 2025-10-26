// Package testing provides test collection and execution for AILANG programs.
// This implements the M-TESTING feature for property-based testing and inline tests.
package testing

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
)

// TestCase represents a single test case extracted from the AST.
type TestCase struct {
	Name        string     // Test name (from test "name" { ... })
	Body        []ast.Expr // Test body expressions
	Location    ast.Pos    // Source location
	IsInline    bool       // true for tests[...], false for test "name" {}
	FunctionCtx string     // Function name if this is an inline test
}

// PropertyCase represents a property-based test extracted from the AST.
type PropertyCase struct {
	Name     string        // Property name
	Property *ast.Property // The property specification (forall(...) => expr)
	Location ast.Pos       // Source location
	IsInline bool          // true for properties[...], false for property "name" {}
}

// TestSuite represents all tests and properties extracted from a module.
type TestSuite struct {
	ModulePath string         // Module path (e.g., "std/list")
	Tests      []TestCase     // All test cases
	Properties []PropertyCase // All property cases
}

// Collector extracts tests and properties from an AST.
type Collector struct {
	suite TestSuite
}

// NewCollector creates a new test collector.
func NewCollector(modulePath string) *Collector {
	return &Collector{
		suite: TestSuite{
			ModulePath: modulePath,
			Tests:      []TestCase{},
			Properties: []PropertyCase{},
		},
	}
}

// Collect extracts all tests and properties from an AST file.
func (c *Collector) Collect(file *ast.File) *TestSuite {
	// Collect top-level test blocks: test "name" { ... }
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.TestDecl:
			c.collectTestDecl(d)
		case *ast.PropertyDecl:
			c.collectPropertyDecl(d)
		case *ast.FuncDecl:
			c.collectInlineTests(d)
		}
	}

	return &c.suite
}

// collectTestDecl extracts a top-level test block.
func (c *Collector) collectTestDecl(decl *ast.TestDecl) {
	testCase := TestCase{
		Name:        decl.Name,
		Body:        decl.Body,
		Location:    decl.Pos,
		IsInline:    false,
		FunctionCtx: "", // No function context for top-level tests
	}
	c.suite.Tests = append(c.suite.Tests, testCase)
}

// collectPropertyDecl extracts a top-level property block.
func (c *Collector) collectPropertyDecl(decl *ast.PropertyDecl) {
	propCase := PropertyCase{
		Name:     decl.Name,
		Property: decl.Property,
		Location: decl.Pos,
		IsInline: false,
	}
	c.suite.Properties = append(c.suite.Properties, propCase)
}

// collectInlineTests extracts inline tests from function declarations.
// Handles: func foo(x: int) tests [...] properties [...] { body }
func (c *Collector) collectInlineTests(decl *ast.FuncDecl) {
	funcName := decl.Name

	// Collect inline tests: tests [test1, test2, ...]
	if decl.Tests != nil {
		for i, testCase := range decl.Tests {
			// Each TestCase has Inputs and Expected fields
			// For now, we create a test expression that checks: funcName(inputs...) == expected
			// In full implementation, we'd evaluate this properly
			tc := TestCase{
				Name:        fmt.Sprintf("%s_test_%d", funcName, i+1), // e.g., "factorial_test_1"
				Body:        []ast.Expr{testCase.Expected},            // Store expected as body for now
				Location:    testCase.Pos,
				IsInline:    true,
				FunctionCtx: funcName,
			}
			c.suite.Tests = append(c.suite.Tests, tc)
		}
	}

	// Collect inline properties: properties [prop1, prop2, ...]
	if decl.Properties != nil {
		for i, prop := range decl.Properties {
			propCase := PropertyCase{
				Name:     fmt.Sprintf("%s_property_%d", funcName, i+1), // e.g., "factorial_property_1"
				Property: prop,
				Location: prop.Pos,
				IsInline: true,
			}
			c.suite.Properties = append(c.suite.Properties, propCase)
		}
	}
}

// GetSuite returns the collected test suite.
func (c *Collector) GetSuite() *TestSuite {
	return &c.suite
}
