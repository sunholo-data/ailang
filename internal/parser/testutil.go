package parser

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// update flag controls whether golden files are updated or compared
// Usage: go test -update ./internal/parser
var update = flag.Bool("update", false, "update golden files")

// goldenCompare compares the given output with a golden file.
// If the -update flag is set, it updates the golden file instead of comparing.
//
// Usage:
//
//	p := parser.New(lexer.New(input, "test://unit"))
//	prog := p.ParseProgram()
//	goldenCompare(t, "expr/int_literal", ast.Print(prog))
func goldenCompare(t *testing.T, name string, got string) {
	t.Helper()

	path := filepath.Join("testdata", "parser", name+".golden")

	if *update {
		// Ensure directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Write golden file
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatalf("Failed to write golden file %s: %v", path, err)
		}

		t.Logf("Updated golden file: %s", path)
		return
	}

	// Read and compare
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v\nRun with -update to create it", path, err)
	}

	if diff := cmp.Diff(string(want), got); diff != "" {
		t.Errorf("Golden mismatch for %s (-want +got):\n%s", name, diff)
		t.Logf("To update: go test -update ./internal/parser")
	}
}

// mustParseError parses input and expects it to fail with parse errors.
// Returns the list of errors.
//
// Usage:
//
//	errs := mustParseError(t, "let x = ")
//	assertHasErrorCode(t, errs, "PAR001")
func mustParseError(t *testing.T, input string) []error {
	t.Helper()

	p := New(lexer.New(input, "test://unit"))
	prog := p.Parse()

	if len(p.Errors()) == 0 {
		t.Fatalf("Expected parse errors but got none. AST:\n%s", ast.PrintProgram(prog))
	}

	return p.Errors()
}

// mustParse parses input and expects it to succeed.
// Returns the parsed program.
//
// Usage:
//
//	prog := mustParse(t, "42")
func mustParse(t *testing.T, input string) *ast.Program {
	t.Helper()

	p := New(lexer.New(input, "test://unit"))
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parse errors:\n%v", p.Errors())
	}

	if prog == nil {
		t.Fatal("Parser returned nil program with no errors")
	}

	return prog
}

// assertHasErrorCode checks that at least one error in the list contains the given code.
// Error codes are extracted from error messages by looking for patterns like "[PAR001]" or "PAR001:".
//
// Usage:
//
//	errs := mustParseError(t, "let x = \nlet y =")
//	assertHasErrorCode(t, errs, "PAR001") // Missing value for x
//	assertHasErrorCode(t, errs, "PAR002") // Missing value for y
func assertHasErrorCode(t *testing.T, errs []error, code string) {
	t.Helper()

	for _, err := range errs {
		msg := err.Error()
		// Check for [CODE] or CODE: patterns
		if contains(msg, "["+code+"]") || contains(msg, code+":") {
			return
		}
	}

	// Error not found
	t.Errorf("Expected error code %s but not found in:\n", code)
	for _, err := range errs {
		t.Errorf("  - %v", err)
	}
}

// assertErrorCount checks that the parser produced exactly n errors
func assertErrorCount(t *testing.T, errs []error, expected int) {
	t.Helper()

	if len(errs) != expected {
		t.Errorf("Expected %d errors, got %d:", expected, len(errs))
		for _, err := range errs {
			t.Errorf("  - %v", err)
		}
	}
}

// assertContains checks that the error message contains the given substring
func assertContains(t *testing.T, err error, substring string) {
	t.Helper()

	if !contains(err.Error(), substring) {
		t.Errorf("Expected error to contain %q, got: %v", substring, err)
	}
}

// contains is a simple substring check helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// parseExpr is a helper that parses a single expression
// Useful for expression-level tests without wrapping in a program
func parseExpr(t *testing.T, input string) ast.Expr {
	t.Helper()

	prog := mustParse(t, input)

	// Extract the expression from the program
	if prog.File != nil && len(prog.File.Statements) > 0 {
		if expr, ok := prog.File.Statements[0].(ast.Expr); ok {
			return expr
		}
	}

	t.Fatalf("Failed to extract expression from parsed program")
	return nil
}

// assertPrecedence parses an expression and checks it matches the expected
// parenthesized form. Useful for testing operator precedence and associativity.
//
// Usage:
//
//	assertPrecedence(t, "1 + 2 * 3", "(1 + (2 * 3))")
//	assertPrecedence(t, "x && y || z", "((x && y) || z)")
func assertPrecedence(t *testing.T, input, expectedForm string) {
	t.Helper()

	expr := parseExpr(t, input)
	got := exprToParenForm(expr)

	if got != expectedForm {
		t.Errorf("Precedence mismatch:\n  input:    %s\n  expected: %s\n  got:      %s",
			input, expectedForm, got)
	}
}

// exprToParenForm converts an expression to its fully parenthesized form
// Example: BinaryOp(+, 1, BinaryOp(*, 2, 3)) -> "(1 + (2 * 3))"
func exprToParenForm(expr ast.Expr) string {
	if expr == nil {
		return "nil"
	}

	switch e := expr.(type) {
	case *ast.Literal:
		return fmt.Sprintf("%v", e.Value)

	case *ast.Identifier:
		return e.Name

	case *ast.BinaryOp:
		left := exprToParenForm(e.Left)
		right := exprToParenForm(e.Right)
		return "(" + left + " " + e.Op + " " + right + ")"

	case *ast.UnaryOp:
		right := exprToParenForm(e.Expr)
		return "(" + e.Op + right + ")"

	default:
		return "<?>"
	}
}

// assertExprType checks that the parsed expression is of the expected type
func assertExprType(t *testing.T, expr ast.Expr, expectedType string) {
	t.Helper()

	obj := ast.Compact(expr)
	if !contains(obj, `"type":"`+expectedType+`"`) {
		t.Errorf("Expected expression type %s, got: %s", expectedType, obj)
	}
}

// parseAndPrint is a convenience helper that parses input and returns the printed AST
// Useful for quick golden file generation
func parseAndPrint(t *testing.T, input string) string {
	t.Helper()

	prog := mustParse(t, input)
	return ast.PrintProgram(prog)
}
