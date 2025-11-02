package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TODO: S-CALL0 sugar requires statement-level parsing changes
// Current limitation: f() without space is parsed as two separate tokens
// at top level. Requires deeper parser refactoring to support.
// For now, canonical syntax f (()) works fine.

// TestSugarCall0_Skip documents the S-CALL0 limitation
func TestSugarCall0_Skip(t *testing.T) {
	t.Skip("S-CALL0 requires statement-level parsing changes - deferred to follow-up")
	// The issue: myFunc() is parsed as [myFunc, ()] at statement level
	// The fix requires: detecting () immediately after identifier at parse level
	// Workaround: Use canonical f (()) syntax (with space and explicit unit)
}

// TestSugarCons_Basic tests S-CONS sugar: x :: xs desugars to ::(x, xs)
func TestSugarCons_Basic(t *testing.T) {
	input := `let list = 1 :: 2 :: []`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	// Check that sugar was used
	if !p.SugarUsed() {
		t.Fatalf("Expected sugarUsed=true for :: syntax")
	}

	// Should have one statement
	if len(file.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(file.Statements))
	}

	letExpr, ok := file.Statements[0].(*ast.Let)
	if !ok {
		t.Fatalf("Expected Let expression, got %T", file.Statements[0])
	}

	// The value should be a FuncCall to "::"
	funcCall, ok := letExpr.Value.(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected FuncCall, got %T", letExpr.Value)
	}

	// Function should be "::"
	AssertIdentifier(t, funcCall.Func, "::")

	// Should have 2 args: 1 and (2 :: [])
	if len(funcCall.Args) != 2 {
		t.Fatalf("Expected 2 arguments, got %d", len(funcCall.Args))
	}

	// First arg should be 1
	AssertLiteralInt(t, funcCall.Args[0], 1)

	// Second arg should be another :: call (right-associative)
	innerCall, ok := funcCall.Args[1].(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected inner FuncCall for right-associativity, got %T", funcCall.Args[1])
	}

	AssertIdentifier(t, innerCall.Func, "::")
}

// TestSugarCons_StrictMode tests that S-CONS is rejected in strict mode
func TestSugarCons_StrictMode(t *testing.T) {
	input := `let list = 1 :: []`
	l := lexer.New(input, "test.ail")
	p := New(l)
	p.SetStrictSyntaxMode(true)
	_ = p.ParseFile()

	// Should have an error
	if len(p.Errors()) == 0 {
		t.Fatalf("Expected error in strict mode, got none")
	}

	// Error should exist (we already checked this above)
}

// TestSugarArrowType_Basic tests S-ARROWTYPE sugar: int -> bool
func TestSugarArrowType_Basic(t *testing.T) {
	input := `let f: int -> bool = \x. x > 0`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	// Check that sugar was used
	if !p.SugarUsed() {
		t.Fatalf("Expected sugarUsed=true for -> syntax")
	}

	// Should have one statement
	if len(file.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(file.Statements))
	}

	letExpr, ok := file.Statements[0].(*ast.Let)
	if !ok {
		t.Fatalf("Expected Let expression, got %T", file.Statements[0])
	}

	// Type should be FuncType
	funcType, ok := letExpr.Type.(*ast.FuncType)
	if !ok {
		t.Fatalf("Expected FuncType, got %T", letExpr.Type)
	}

	// Should have 1 param (int)
	if len(funcType.Params) != 1 {
		t.Fatalf("Expected 1 parameter type, got %d", len(funcType.Params))
	}

	// Param should be "int"
	paramType, ok := funcType.Params[0].(*ast.SimpleType)
	if !ok {
		t.Fatalf("Expected SimpleType for param, got %T", funcType.Params[0])
	}
	if paramType.Name != "int" {
		t.Errorf("Expected param type 'int', got '%s'", paramType.Name)
	}

	// Return should be "bool"
	returnType, ok := funcType.Return.(*ast.SimpleType)
	if !ok {
		t.Fatalf("Expected SimpleType for return, got %T", funcType.Return)
	}
	if returnType.Name != "bool" {
		t.Errorf("Expected return type 'bool', got '%s'", returnType.Name)
	}
}

// TestSugarArrowType_RightAssociative tests that -> is right-associative
func TestSugarArrowType_RightAssociative(t *testing.T) {
	input := `let f: int -> bool -> string = undefined`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	letExpr := file.Statements[0].(*ast.Let)
	funcType, ok := letExpr.Type.(*ast.FuncType)
	if !ok {
		t.Fatalf("Expected FuncType, got %T", letExpr.Type)
	}

	// Should be: int -> (bool -> string)
	// So params = [int], return = FuncType(bool, string)
	if len(funcType.Params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(funcType.Params))
	}

	// Return should be another FuncType
	innerFunc, ok := funcType.Return.(*ast.FuncType)
	if !ok {
		t.Fatalf("Expected inner FuncType for right-associativity, got %T", funcType.Return)
	}

	// Inner function: bool -> string
	if len(innerFunc.Params) != 1 {
		t.Fatalf("Expected 1 parameter in inner func, got %d", len(innerFunc.Params))
	}

	paramType, ok := innerFunc.Params[0].(*ast.SimpleType)
	if !ok || paramType.Name != "bool" {
		t.Errorf("Expected inner param type 'bool', got %v", innerFunc.Params[0])
	}

	returnType, ok := innerFunc.Return.(*ast.SimpleType)
	if !ok || returnType.Name != "string" {
		t.Errorf("Expected inner return type 'string', got %v", innerFunc.Return)
	}
}

// TestSugarArrowType_StrictMode tests that S-ARROWTYPE is rejected in strict mode
func TestSugarArrowType_StrictMode(t *testing.T) {
	input := `let f: int -> bool = undefined`
	l := lexer.New(input, "test.ail")
	p := New(l)
	p.SetStrictSyntaxMode(true)
	_ = p.ParseFile()

	// Should have an error
	if len(p.Errors()) == 0 {
		t.Fatalf("Expected error in strict mode, got none")
	}

	// Error should exist (we already checked this above)
}

// TestSugar_AllDisabled tests that all implemented sugars are disabled in strict mode
func TestSugar_AllDisabled(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// call0 skipped - requires statement-level parsing changes
		{"cons", `let x = 1 :: []`},
		{"arrowtype", `let f: int -> bool = undefined`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			p.SetStrictSyntaxMode(true)
			_ = p.ParseFile()

			if len(p.Errors()) == 0 {
				t.Errorf("Expected error in strict mode for %s, got none", tt.name)
			}
		})
	}
}

// TestSugar_Canonical tests that canonical syntax still works
func TestSugar_Canonical(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"call_with_unit", `let x = myFunc (())`},
		{"call_with_space", `let x = myFunc (1)`},
		{"functype_parens", `let f: (int) -> bool = \x. x > 0`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			p.SetStrictSyntaxMode(true) // Should work even in strict mode
			_ = p.ParseFile()

			AssertNoErrors(t, p)

			// Should NOT have used sugar
			if p.SugarUsed() {
				t.Errorf("Canonical syntax should not set sugarUsed flag")
			}
		})
	}
}
