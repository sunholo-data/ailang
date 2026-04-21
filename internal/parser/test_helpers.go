package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// AssertTokenPosition helps debug token position issues in tests.
// It verifies that the parser is at the expected current and peek tokens.
//
// Example:
//
//	p.nextToken()
//	expr := p.parseExpression(LOWEST)
//	AssertTokenPosition(t, p, lexer.INT, lexer.COMMA)  // cur=INT, peek=COMMA
func AssertTokenPosition(t *testing.T, p *Parser, expectedCur, expectedPeek lexer.TokenType) {
	t.Helper()
	if !p.curTokenIs(expectedCur) {
		t.Errorf("Expected cur=%s, got %s (literal=%q)", expectedCur, p.curToken.Type, p.curToken.Literal)
	}
	if !p.peekTokenIs(expectedPeek) {
		t.Errorf("Expected peek=%s, got %s (literal=%q)", expectedPeek, p.peekToken.Type, p.peekToken.Literal)
	}
}

// AssertNoErrors verifies that the parser has no errors.
// If there are errors, it prints them before failing the test.
//
// Example:
//
//	p := New(lexer.New(input))
//	file := p.ParseFile()
//	AssertNoErrors(t, p)
func AssertNoErrors(t *testing.T, p *Parser) {
	t.Helper()
	if len(p.Errors()) != 0 {
		// Print errors BEFORE Fatalf
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}
}

// AssertErrorCount verifies that the parser has exactly the expected number of errors.
//
// Example:
//
//	p := New(lexer.New("invalid syntax"))
//	p.ParseFile()
//	AssertErrorCount(t, p, 1)
func AssertErrorCount(t *testing.T, p *Parser, expected int) {
	t.Helper()
	actual := len(p.Errors())
	if actual != expected {
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatalf("Expected %d errors, got %d", expected, actual)
	}
}

// AssertLiteralInt verifies that an expression is an integer literal with the expected value.
//
// Example:
//
//	expr := p.parseExpression(LOWEST)
//	AssertLiteralInt(t, expr, 42)
func AssertLiteralInt(t *testing.T, expr ast.Expr, expected int) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Fatalf("Expected *ast.Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.IntLit {
		t.Fatalf("Expected IntLit, got %v", lit.Kind)
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
		t.Fatalf("Expected int/int64 value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("Expected value %d, got %d", expected, val)
	}
}

// AssertLiteralString verifies that an expression is a string literal with the expected value.
func AssertLiteralString(t *testing.T, expr ast.Expr, expected string) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Fatalf("Expected *ast.Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.StringLit {
		t.Fatalf("Expected StringLit, got %v", lit.Kind)
		return
	}
	val, ok := lit.Value.(string)
	if !ok {
		t.Fatalf("Expected string value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("Expected value %q, got %q", expected, val)
	}
}

// AssertLiteralBool verifies that an expression is a boolean literal with the expected value.
func AssertLiteralBool(t *testing.T, expr ast.Expr, expected bool) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Fatalf("Expected *ast.Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.BoolLit {
		t.Fatalf("Expected BoolLit, got %v", lit.Kind)
		return
	}
	val, ok := lit.Value.(bool)
	if !ok {
		t.Fatalf("Expected bool value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("Expected value %t, got %t", expected, val)
	}
}

// AssertLiteralFloat verifies that an expression is a float literal with the expected value.
func AssertLiteralFloat(t *testing.T, expr ast.Expr, expected float64) {
	t.Helper()
	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Fatalf("Expected *ast.Literal, got %T", expr)
		return
	}
	if lit.Kind != ast.FloatLit {
		t.Fatalf("Expected FloatLit, got %v", lit.Kind)
		return
	}
	val, ok := lit.Value.(float64)
	if !ok {
		t.Fatalf("Expected float64 value, got %T", lit.Value)
		return
	}
	if val != expected {
		t.Errorf("Expected value %f, got %f", expected, val)
	}
}

// AssertIdentifier verifies that an expression is an identifier with the expected name.
//
// Example:
//
//	expr := p.parseExpression(LOWEST)
//	AssertIdentifier(t, expr, "x")
func AssertIdentifier(t *testing.T, expr ast.Expr, expectedName string) {
	t.Helper()
	id, ok := expr.(*ast.Identifier)
	if !ok {
		t.Fatalf("Expected *ast.Identifier, got %T", expr)
		return
	}
	if id.Name != expectedName {
		t.Errorf("Expected identifier name %q, got %q", expectedName, id.Name)
	}
}

// AssertFuncCall verifies that an expression is a function call and returns it for further inspection.
//
// Example:
//
//	expr := p.parseExpression(LOWEST)
//	call := AssertFuncCall(t, expr)
//	if call != nil {
//		AssertIdentifier(t, call.Func, "add")
//		if len(call.Args) != 2 { ... }
//	}
func AssertFuncCall(t *testing.T, expr ast.Expr) *ast.FuncCall {
	t.Helper()
	call, ok := expr.(*ast.FuncCall)
	if !ok {
		t.Fatalf("Expected *ast.FuncCall, got %T", expr)
		return nil
	}
	return call
}

// AssertList verifies that an expression is a list and returns it for further inspection.
//
// Example:
//
//	expr := p.parseExpression(LOWEST)
//	list := AssertList(t, expr)
//	if list != nil {
//		if len(list.Elements) != 3 { ... }
//	}
func AssertList(t *testing.T, expr ast.Expr) *ast.List {
	t.Helper()
	list, ok := expr.(*ast.List)
	if !ok {
		t.Fatalf("Expected *ast.List, got %T", expr)
		return nil
	}
	return list
}

// AssertListLength verifies that an expression is a list with the expected length.
func AssertListLength(t *testing.T, expr ast.Expr, expectedLen int) *ast.List {
	t.Helper()
	list := AssertList(t, expr)
	if list == nil {
		return nil
	}
	if len(list.Elements) != expectedLen {
		t.Errorf("Expected list length %d, got %d", expectedLen, len(list.Elements))
	}
	return list
}

// AssertDeclCount verifies that a file has the expected number of declarations.
//
// Example:
//
//	file := p.ParseFile()
//	AssertDeclCount(t, file, 3)
func AssertDeclCount(t *testing.T, file *ast.File, expected int) {
	t.Helper()
	actual := len(file.Decls)
	if actual != expected {
		t.Fatalf("Expected %d declarations, got %d", expected, actual)
	}
}

// AssertFuncDecl verifies that a node is a function declaration and returns it.
//
// Example:
//
//	file := p.ParseFile()
//	fn := AssertFuncDecl(t, file.Decls[0], "factorial")
//	if fn != nil {
//		// Check function properties
//	}
func AssertFuncDecl(t *testing.T, node ast.Node, expectedName string) *ast.FuncDecl {
	t.Helper()
	fn, ok := node.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Expected *ast.FuncDecl, got %T", node)
		return nil
	}
	if fn.Name != expectedName {
		t.Errorf("Expected function name %q, got %q", expectedName, fn.Name)
	}
	return fn
}

// AssertTypeDecl verifies that a node is a type declaration and returns it.
func AssertTypeDecl(t *testing.T, node ast.Node, expectedName string) *ast.TypeDecl {
	t.Helper()
	td, ok := node.(*ast.TypeDecl)
	if !ok {
		t.Fatalf("Expected *ast.TypeDecl, got %T", node)
		return nil
	}
	if td.Name != expectedName {
		t.Errorf("Expected type name %q, got %q", expectedName, td.Name)
	}
	return td
}

// AssertSimpleType verifies that a type is a simple type with the expected name.
func AssertSimpleType(t *testing.T, typ ast.Type, expectedName string) {
	t.Helper()
	st, ok := typ.(*ast.SimpleType)
	if !ok {
		t.Fatalf("Expected *ast.SimpleType, got %T", typ)
		return
	}
	if st.Name != expectedName {
		t.Errorf("Expected type name %q, got %q", expectedName, st.Name)
	}
}

// AssertListType verifies that a type is a list type and returns the element type.
// DX-17 Phase 2: [T] syntax now parses to TypeApp("list", [T])
func AssertListType(t *testing.T, typ ast.Type) ast.Type {
	t.Helper()
	ta, ok := typ.(*ast.TypeApp)
	if !ok {
		t.Fatalf("Expected *ast.TypeApp for list type, got %T", typ)
		return nil
	}
	if ta.Constructor != "list" {
		t.Fatalf("Expected TypeApp with constructor 'list', got %q", ta.Constructor)
		return nil
	}
	if len(ta.Args) != 1 {
		t.Fatalf("Expected TypeApp with 1 arg, got %d", len(ta.Args))
		return nil
	}
	return ta.Args[0]
}
