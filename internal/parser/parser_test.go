package parser

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

func TestLetExpression(t *testing.T) {
	input := `let x = 5`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("Parse() returned nil")
	}

	if program.Module == nil {
		t.Fatalf("program.Module is nil")
	}

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program.Module.Decls does not contain 1 declaration. got=%d",
			len(program.Module.Decls))
	}

	letExpr, ok := program.Module.Decls[0].(*ast.Let)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not *ast.Let. got=%T",
			program.Module.Decls[0])
	}

	if letExpr.Name != "x" {
		t.Fatalf("letExpr.Name not 'x'. got=%s", letExpr.Name)
	}

	testLiteralExpression(t, letExpr.Value, int64(5))
}

func TestReturnExpressions(t *testing.T) {
	tests := []struct {
		input         string
		expectedValue interface{}
	}{
		{"5", int64(5)},
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input, "test.ail")
		p := New(l)
		program := p.Parse()
		checkParserErrors(t, p)

		if len(program.Module.Decls) != 1 {
			t.Fatalf("program.Module.Decls does not contain 1 declaration. got=%d",
				len(program.Module.Decls))
		}

		expr, ok := program.Module.Decls[0].(ast.Expr)
		if !ok {
			t.Fatalf("program.Module.Decls[0] is not ast.Expr. got=%T",
				program.Module.Decls[0])
		}

		testLiteralExpression(t, expr, tt.expectedValue)
	}
}

func TestIdentifierExpression(t *testing.T) {
	input := "foobar"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program has wrong number of declarations. got=%d",
			len(program.Module.Decls))
	}

	ident, ok := program.Module.Decls[0].(*ast.Identifier)
	if !ok {
		t.Fatalf("program.Module.Decls[0] not *ast.Identifier. got=%T",
			program.Module.Decls[0])
	}

	if ident.Name != "foobar" {
		t.Fatalf("ident.Name not %s. got=%s", "foobar", ident.Name)
	}
}

func TestIntegerLiteralExpression(t *testing.T) {
	input := "5"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program has wrong number of declarations. got=%d",
			len(program.Module.Decls))
	}

	literal, ok := program.Module.Decls[0].(*ast.Literal)
	if !ok {
		t.Fatalf("program.Module.Decls[0] not *ast.Literal. got=%T",
			program.Module.Decls[0])
	}

	if literal.Kind != ast.IntLit {
		t.Fatalf("literal.Kind not IntLit. got=%v", literal.Kind)
	}

	value, ok := literal.Value.(int64)
	if !ok {
		t.Fatalf("literal.Value not int64. got=%T", literal.Value)
	}

	if value != 5 {
		t.Fatalf("literal.Value not %d. got=%d", 5, value)
	}
}

func TestPrefixExpressions(t *testing.T) {
	prefixTests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"!5", "!", int64(5)},
		{"-15", "-", int64(15)},
		{"not true", "not", true},
		{"not false", "not", false},
	}

	for _, tt := range prefixTests {
		l := lexer.New(tt.input, "test.ail")
		p := New(l)
		program := p.Parse()
		checkParserErrors(t, p)

		if len(program.Module.Decls) != 1 {
			t.Fatalf("program.Module.Decls does not contain %d declarations. got=%d\n",
				1, len(program.Module.Decls))
		}

		expr, ok := program.Module.Decls[0].(*ast.UnaryOp)
		if !ok {
			t.Fatalf("stmt is not ast.UnaryOp. got=%T", program.Module.Decls[0])
		}

		if expr.Op != tt.operator {
			t.Fatalf("expr.Op is not '%s'. got=%s",
				tt.operator, expr.Op)
		}

		testLiteralExpression(t, expr.Expr, tt.value)
	}
}

func TestInfixExpressions(t *testing.T) {
	infixTests := []struct {
		input      string
		leftValue  interface{}
		operator   string
		rightValue interface{}
	}{
		{"5 + 5", int64(5), "+", int64(5)},
		{"5 - 5", int64(5), "-", int64(5)},
		{"5 * 5", int64(5), "*", int64(5)},
		{"5 / 5", int64(5), "/", int64(5)},
		{"5 > 5", int64(5), ">", int64(5)},
		{"5 < 5", int64(5), "<", int64(5)},
		{"5 == 5", int64(5), "==", int64(5)},
		{"5 != 5", int64(5), "!=", int64(5)},
		{"true == true", true, "==", true},
		{"true != false", true, "!=", false},
		{"false == false", false, "==", false},
	}

	for _, tt := range infixTests {
		l := lexer.New(tt.input, "test.ail")
		p := New(l)
		program := p.Parse()
		checkParserErrors(t, p)

		if len(program.Module.Decls) != 1 {
			t.Fatalf("program.Module.Decls does not contain %d declarations. got=%d\n",
				1, len(program.Module.Decls))
		}

		expr, ok := program.Module.Decls[0].(*ast.BinaryOp)
		if !ok {
			t.Fatalf("program.Module.Decls[0] is not ast.BinaryOp. got=%T",
				program.Module.Decls[0])
		}

		testLiteralExpression(t, expr.Left, tt.leftValue)

		if expr.Op != tt.operator {
			t.Fatalf("expr.Op is not '%s'. got=%s",
				tt.operator, expr.Op)
		}

		testLiteralExpression(t, expr.Right, tt.rightValue)
	}
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"-a * b",
			"((- a) * b)",
		},
		{
			"!-a",
			"(! (- a))",
		},
		{
			"a + b + c",
			"((a + b) + c)",
		},
		{
			"a + b - c",
			"((a + b) - c)",
		},
		{
			"a * b * c",
			"((a * b) * c)",
		},
		{
			"a * b / c",
			"((a * b) / c)",
		},
		{
			"a + b / c",
			"(a + (b / c))",
		},
		{
			"a + b * c + d / e - f",
			"(((a + (b * c)) + (d / e)) - f)",
		},
		{
			"5 > 4 == 3 < 4",
			"((5 > 4) == (3 < 4))",
		},
		{
			"5 < 4 != 3 > 4",
			"((5 < 4) != (3 > 4))",
		},
		{
			"3 + 4 * 5 == 3 * 1 + 4 * 5",
			"((3 + (4 * 5)) == ((3 * 1) + (4 * 5)))",
		},
		{
			"true && false || true",
			"((true && false) || true)",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input, "test.ail")
		p := New(l)
		program := p.Parse()
		checkParserErrors(t, p)

		actual := program.Module.Decls[0].String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func TestIfExpression(t *testing.T) {
	input := `if x < y then x else y`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program.Module.Decls does not contain %d declarations. got=%d\n",
			1, len(program.Module.Decls))
	}

	expr, ok := program.Module.Decls[0].(*ast.If)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.If. got=%T",
			program.Module.Decls[0])
	}

	testInfixExpression(t, expr.Condition, "x", "<", "y")

	thenExpr, ok := expr.Then.(*ast.Identifier)
	if !ok {
		t.Fatalf("expr.Then is not ast.Identifier. got=%T", expr.Then)
	}
	if thenExpr.Name != "x" {
		t.Fatalf("thenExpr.Name not 'x'. got=%s", thenExpr.Name)
	}

	elseExpr, ok := expr.Else.(*ast.Identifier)
	if !ok {
		t.Fatalf("expr.Else is not ast.Identifier. got=%T", expr.Else)
	}
	if elseExpr.Name != "y" {
		t.Fatalf("elseExpr.Name not 'y'. got=%s", elseExpr.Name)
	}
}

func TestListLiteral(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3]"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	list, ok := program.Module.Decls[0].(*ast.List)
	if !ok {
		t.Fatalf("program.Module.Decls[0] not ast.List. got=%T",
			program.Module.Decls[0])
	}

	if len(list.Elements) != 3 {
		t.Fatalf("len(list.Elements) not 3. got=%d", len(list.Elements))
	}

	testLiteralExpression(t, list.Elements[0], int64(1))
	testInfixExpression(t, list.Elements[1], int64(2), "*", int64(2))
	testInfixExpression(t, list.Elements[2], int64(3), "+", int64(3))
}

func TestParsingRecordLiteral(t *testing.T) {
	input := `{ name: "Alice", age: 30, active: true }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	record, ok := program.Module.Decls[0].(*ast.Record)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Record. got=%T",
			program.Module.Decls[0])
	}

	if len(record.Fields) != 3 {
		t.Fatalf("record.Fields does not contain 3 fields. got=%d",
			len(record.Fields))
	}

	tests := map[string]interface{}{
		"name":   "Alice",
		"age":    int64(30),
		"active": true,
	}

	for _, field := range record.Fields {
		expectedValue, ok := tests[field.Name]
		if !ok {
			t.Errorf("unexpected field %q", field.Name)
			continue
		}

		testLiteralExpression(t, field.Value, expectedValue)
	}
}

func TestFunctionLiteral(t *testing.T) {
	input := `func (x, y) => x + y`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	lambda, ok := program.Module.Decls[0].(*ast.Lambda)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Lambda. got=%T",
			program.Module.Decls[0])
	}

	if len(lambda.Params) != 2 {
		t.Fatalf("lambda literal parameters wrong. want 2, got=%d\n",
			len(lambda.Params))
	}

	testLiteralExpression(t, &ast.Identifier{Name: lambda.Params[0].Name}, "x")
	testLiteralExpression(t, &ast.Identifier{Name: lambda.Params[1].Name}, "y")

	testInfixExpression(t, lambda.Body, "x", "+", "y")
}

func TestCallExpression(t *testing.T) {
	input := "add(1, 2 * 3, 4 + 5)"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	call, ok := program.Module.Decls[0].(*ast.FuncCall)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.FuncCall. got=%T",
			program.Module.Decls[0])
	}

	testIdentifier(t, call.Func, "add")

	if len(call.Args) != 3 {
		t.Fatalf("wrong length of arguments. got=%d", len(call.Args))
	}

	testLiteralExpression(t, call.Args[0], int64(1))
	testInfixExpression(t, call.Args[1], int64(2), "*", int64(3))
	testInfixExpression(t, call.Args[2], int64(4), "+", int64(5))
}

// Helper functions

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %v", msg)
	}
	t.FailNow()
}

func testLiteralExpression(t *testing.T, exp ast.Expr, expected interface{}) bool {
	switch v := expected.(type) {
	case int:
		return testIntegerLiteral(t, exp, int64(v))
	case int64:
		return testIntegerLiteral(t, exp, v)
	case string:
		return testIdentifier(t, exp, v)
	case bool:
		return testBooleanLiteral(t, exp, v)
	}
	t.Errorf("type of exp not handled. got=%T", exp)
	return false
}

func testIntegerLiteral(t *testing.T, exp ast.Expr, value int64) bool {
	lit, ok := exp.(*ast.Literal)
	if !ok {
		t.Errorf("exp not *ast.Literal. got=%T", exp)
		return false
	}

	if lit.Kind != ast.IntLit {
		t.Errorf("lit.Kind not IntLit. got=%v", lit.Kind)
		return false
	}

	intVal, ok := lit.Value.(int64)
	if !ok {
		t.Errorf("lit.Value not int64. got=%T", lit.Value)
		return false
	}

	if intVal != value {
		t.Errorf("lit.Value not %d. got=%d", value, intVal)
		return false
	}

	return true
}

func testBooleanLiteral(t *testing.T, exp ast.Expr, value bool) bool {
	lit, ok := exp.(*ast.Literal)
	if !ok {
		t.Errorf("exp not *ast.Literal. got=%T", exp)
		return false
	}

	if lit.Kind != ast.BoolLit {
		t.Errorf("lit.Kind not BoolLit. got=%v", lit.Kind)
		return false
	}

	boolVal, ok := lit.Value.(bool)
	if !ok {
		t.Errorf("lit.Value not bool. got=%T", lit.Value)
		return false
	}

	if boolVal != value {
		t.Errorf("lit.Value not %t. got=%t", value, boolVal)
		return false
	}

	return true
}

func testIdentifier(t *testing.T, exp ast.Expr, value string) bool {
	ident, ok := exp.(*ast.Identifier)
	if !ok {
		t.Errorf("exp not *ast.Identifier. got=%T", exp)
		return false
	}

	if ident.Name != value {
		t.Errorf("ident.Name not %s. got=%s", value, ident.Name)
		return false
	}

	return true
}

func testInfixExpression(t *testing.T, exp ast.Expr, left interface{},
	operator string, right interface{}) bool {

	opExp, ok := exp.(*ast.BinaryOp)
	if !ok {
		t.Errorf("exp is not ast.BinaryOp. got=%T(%s)", exp, exp)
		return false
	}

	if !testLiteralExpression(t, opExp.Left, left) {
		return false
	}

	if opExp.Op != operator {
		t.Errorf("exp.Op is not '%s'. got=%q", operator, opExp.Op)
		return false
	}

	if !testLiteralExpression(t, opExp.Right, right) {
		return false
	}

	return true
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world"`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	literal, ok := program.Module.Decls[0].(*ast.Literal)
	if !ok {
		t.Fatalf("program.Module.Decls[0] not *ast.Literal. got=%T",
			program.Module.Decls[0])
	}

	if literal.Kind != ast.StringLit {
		t.Fatalf("literal.Kind not StringLit. got=%v", literal.Kind)
	}

	if literal.Value != "hello world" {
		t.Errorf("literal.Value not %q. got=%q", "hello world", literal.Value)
	}
}

func TestBooleanExpression(t *testing.T) {
	tests := []struct {
		input           string
		expectedBoolean bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input, "test.ail")
		p := New(l)
		program := p.Parse()
		checkParserErrors(t, p)

		if len(program.Module.Decls) != 1 {
			t.Fatalf("program has not enough declarations. got=%d",
				len(program.Module.Decls))
		}

		boolean, ok := program.Module.Decls[0].(*ast.Literal)
		if !ok {
			t.Fatalf("program.Module.Decls[0] not *ast.Literal. got=%T",
				program.Module.Decls[0])
		}

		if boolean.Kind != ast.BoolLit {
			t.Fatalf("boolean.Kind not BoolLit. got=%v", boolean.Kind)
		}

		boolVal, ok := boolean.Value.(bool)
		if !ok {
			t.Fatalf("boolean.Value not bool. got=%T", boolean.Value)
		}

		if boolVal != tt.expectedBoolean {
			t.Fatalf("boolean.Value not %t. got=%t", tt.expectedBoolean, boolVal)
		}
	}
}

func TestMatchExpression(t *testing.T) {
	input := `
match value {
  Some(x) => x * 2,
  None => 0
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program.Module.Decls does not contain 1 declaration. got=%d",
			len(program.Module.Decls))
	}

	match, ok := program.Module.Decls[0].(*ast.Match)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Match. got=%T",
			program.Module.Decls[0])
	}

	testIdentifier(t, match.Expr, "value")

	if len(match.Cases) != 2 {
		t.Fatalf("match.Cases does not contain 2 cases. got=%d",
			len(match.Cases))
	}

	// First case: Some(x) => x * 2
	case1 := match.Cases[0]
	if case1.Pattern == nil {
		t.Fatalf("case1.Pattern is nil")
	}
	// TODO: Test constructor pattern

	// Second case: None => 0
	case2 := match.Cases[1]
	if case2.Pattern == nil {
		t.Fatalf("case2.Pattern is nil")
	}
	testLiteralExpression(t, case2.Body, int64(0))
}

func TestParsingErrors(t *testing.T) {
	tests := []string{
		"let x 5",           // Missing =
		"if x < y then x",   // Missing else
		"func (x, y",        // Unclosed parenthesis
		"{ name: }",         // Missing value
		"[1, 2,",           // Unclosed bracket
	}

	for _, input := range tests {
		l := lexer.New(input, "test.ail")
		p := New(l)
		_ = p.Parse()

		if len(p.Errors()) == 0 {
			t.Errorf("expected parser error for: %q", input)
		}
	}
}

func TestModuleParsing(t *testing.T) {
	input := `
module TestModule

import std/io (Console)
import std/collections

func main() -> () ! {IO} {
  print("Hello")
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if program.Module == nil {
		t.Fatalf("program.Module is nil")
	}

	if program.Module.Name != "TestModule" {
		t.Fatalf("module.Name not 'TestModule'. got=%s", program.Module.Name)
	}

	if len(program.Module.Imports) != 2 {
		t.Fatalf("module.Imports does not contain 2 imports. got=%d",
			len(program.Module.Imports))
	}

	// Check first import
	imp1 := program.Module.Imports[0]
	if imp1.Path != "std/io" {
		t.Errorf("import[0].Path not 'std/io'. got=%s", imp1.Path)
	}
	if len(imp1.Symbols) != 1 || imp1.Symbols[0] != "Console" {
		t.Errorf("import[0].Symbols incorrect. got=%v", imp1.Symbols)
	}

	// Check second import
	imp2 := program.Module.Imports[1]
	if imp2.Path != "std/collections" {
		t.Errorf("import[1].Path not 'std/collections'. got=%s", imp2.Path)
	}
}

func TestFunctionDeclaration(t *testing.T) {
	input := `
pure func add(x: int, y: int) -> int {
  x + y
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	if len(program.Module.Decls) != 1 {
		t.Fatalf("program.Module.Decls does not contain 1 declaration. got=%d",
			len(program.Module.Decls))
	}

	fn, ok := program.Module.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.FuncDecl. got=%T",
			program.Module.Decls[0])
	}

	if !fn.IsPure {
		t.Errorf("function not marked as pure")
	}

	if fn.Name != "add" {
		t.Errorf("fn.Name not 'add'. got=%s", fn.Name)
	}

	if len(fn.Params) != 2 {
		t.Fatalf("function literal parameters wrong. want 2, got=%d\n",
			len(fn.Params))
	}

	// Check parameters
	param1 := fn.Params[0]
	if param1.Name != "x" {
		t.Errorf("param[0].Name not 'x'. got=%s", param1.Name)
	}
	
	param2 := fn.Params[1]
	if param2.Name != "y" {
		t.Errorf("param[1].Name not 'y'. got=%s", param2.Name)
	}

	// Check body
	testInfixExpression(t, fn.Body, "x", "+", "y")
}

func TestTupleLiteral(t *testing.T) {
	input := "(1, true, \"hello\")"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	tuple, ok := program.Module.Decls[0].(*ast.Tuple)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Tuple. got=%T",
			program.Module.Decls[0])
	}

	if len(tuple.Elements) != 3 {
		t.Fatalf("tuple.Elements does not contain 3 elements. got=%d",
			len(tuple.Elements))
	}

	testLiteralExpression(t, tuple.Elements[0], int64(1))
	testLiteralExpression(t, tuple.Elements[1], true)
	
	str, ok := tuple.Elements[2].(*ast.Literal)
	if !ok || str.Kind != ast.StringLit || str.Value != "hello" {
		t.Errorf("tuple.Elements[2] not string 'hello'. got=%v", tuple.Elements[2])
	}
}

func TestRecordAccess(t *testing.T) {
	input := "user.name"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	access, ok := program.Module.Decls[0].(*ast.RecordAccess)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.RecordAccess. got=%T",
			program.Module.Decls[0])
	}

	testIdentifier(t, access.Record, "user")

	if access.Field != "name" {
		t.Errorf("access.Field not 'name'. got=%s", access.Field)
	}
}

func TestLetWithTypeAnnotation(t *testing.T) {
	input := "let x: int = 5"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	let, ok := program.Module.Decls[0].(*ast.Let)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Let. got=%T",
			program.Module.Decls[0])
	}

	if let.Name != "x" {
		t.Errorf("let.Name not 'x'. got=%s", let.Name)
	}

	if let.Type == nil {
		t.Fatal("let.Type is nil")
	}

	simpleType, ok := let.Type.(*ast.SimpleType)
	if !ok {
		t.Fatalf("let.Type is not ast.SimpleType. got=%T", let.Type)
	}

	if simpleType.Name != "int" {
		t.Errorf("type not 'int'. got=%s", simpleType.Name)
	}

	testLiteralExpression(t, let.Value, int64(5))
}

func TestLetInExpression(t *testing.T) {
	input := "let x = 5 in x + 1"

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()
	checkParserErrors(t, p)

	let, ok := program.Module.Decls[0].(*ast.Let)
	if !ok {
		t.Fatalf("program.Module.Decls[0] is not ast.Let. got=%T",
			program.Module.Decls[0])
	}

	if let.Name != "x" {
		t.Errorf("let.Name not 'x'. got=%s", let.Name)
	}

	testLiteralExpression(t, let.Value, int64(5))

	if let.Body == nil {
		t.Fatal("let.Body is nil")
	}

	testInfixExpression(t, let.Body, "x", "+", int64(1))
}

// Helper to print AST for debugging
func printAST(node ast.Node) {
	fmt.Printf("AST: %s\n", node.String())
}