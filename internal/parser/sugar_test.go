package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestSugarCall0_TopLevel tests S-CALL0 sugar at statement level: f() desugars to f(())
func TestSugarCall0_TopLevel(t *testing.T) {
	input := `myFunc()`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	// Check that sugar was used
	if !p.SugarUsed() {
		t.Fatalf("Expected sugarUsed=true for () syntax")
	}

	// Should have one statement
	if len(file.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(file.Statements))
	}

	// Should be a FuncCall
	funcCall, ok := file.Statements[0].(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected FuncCall, got %T", file.Statements[0])
	}

	// Function should be myFunc
	AssertIdentifier(t, funcCall.Func, "myFunc")

	// Should have 1 arg: unit literal
	if len(funcCall.Args) != 1 {
		t.Fatalf("Expected 1 argument (unit), got %d", len(funcCall.Args))
	}

	// Arg should be unit literal
	unitLit, ok := funcCall.Args[0].(*ast.Literal)
	if !ok {
		t.Fatalf("Expected Literal for unit arg, got %T", funcCall.Args[0])
	}
	if unitLit.Kind != ast.UnitLit {
		t.Errorf("Expected UnitLit, got %v", unitLit.Kind)
	}
}

// TestSugarCall0_MultipleTopLevel tests multiple S-CALL0 calls at top level
func TestSugarCall0_MultipleTopLevel(t *testing.T) {
	input := `func1()
func2()`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	// Should have two statements
	if len(file.Statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(file.Statements))
	}

	// First statement: func1()
	funcCall1, ok := file.Statements[0].(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected FuncCall for first statement, got %T", file.Statements[0])
	}
	AssertIdentifier(t, funcCall1.Func, "func1")

	// Second statement: func2()
	funcCall2, ok := file.Statements[1].(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected FuncCall for second statement, got %T", file.Statements[1])
	}
	AssertIdentifier(t, funcCall2.Func, "func2")
}

// TestSugarCall0_ExpressionContext tests S-CALL0 in expression context (already worked)
func TestSugarCall0_ExpressionContext(t *testing.T) {
	input := `let x = if true then myFunc() else 0`
	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	AssertNoErrors(t, p)

	// Check that sugar was used
	if !p.SugarUsed() {
		t.Fatalf("Expected sugarUsed=true for () syntax")
	}

	// Should have one statement
	if len(file.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(file.Statements))
	}

	letExpr, ok := file.Statements[0].(*ast.Let)
	if !ok {
		t.Fatalf("Expected Let expression, got %T", file.Statements[0])
	}

	// Value should be an If expression
	ifExpr, ok := letExpr.Value.(*ast.If)
	if !ok {
		t.Fatalf("Expected If expression, got %T", letExpr.Value)
	}

	// Then branch should be FuncCall
	funcCall, ok := ifExpr.Then.(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected FuncCall in then branch, got %T", ifExpr.Then)
	}

	AssertIdentifier(t, funcCall.Func, "myFunc")

	// Should have unit arg
	if len(funcCall.Args) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(funcCall.Args))
	}

	unitLit, ok := funcCall.Args[0].(*ast.Literal)
	if !ok || unitLit.Kind != ast.UnitLit {
		t.Fatalf("Expected unit literal argument")
	}
}

// TestSugarCall0_StrictMode tests that S-CALL0 is rejected in strict mode
func TestSugarCall0_StrictMode(t *testing.T) {
	input := `myFunc()`
	l := lexer.New(input, "test.ail")
	p := New(l)
	p.SetStrictSyntaxMode(true)
	_ = p.ParseFile()

	// Should have an error
	if len(p.Errors()) == 0 {
		t.Fatalf("Expected error in strict mode, got none")
	}
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
		{"call0", `myFunc()`},
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
