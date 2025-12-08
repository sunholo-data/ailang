package parser

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseTestDecl parses a top-level test declaration: test "name" { ... }
func (p *Parser) parseTestDecl() *ast.TestDecl {
	pos := p.curPos()

	// Accept either TEST token or IDENT with literal "test" (contextual keyword)
	if !p.curTokenIs(lexer.TEST) && !(p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "test") {
		p.report("PAR_UNEXPECTED_TOKEN", "expected 'test' keyword", "Check syntax")
		return nil
	}
	p.nextToken() // consume TEST/IDENT

	// Expect string literal for test name
	if !p.curTokenIs(lexer.STRING) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected test name (string)", "Add test name: test \"my test\" {...}")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken() // consume STRING

	// Expect opening brace
	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected { to start test body", "Check syntax")
		return nil
	}
	p.nextToken() // consume LBRACE

	// Parse test body (list of expressions)
	var body []ast.Expr

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// Skip newlines
		if p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
			continue
		}

		// Parse expression or assert statement
		expr := p.parseStatement()
		if expr != nil {
			body = append(body, expr)
		}

		// Skip optional semicolons and newlines
		for p.curTokenIs(lexer.SEMICOLON) || p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected } to close test body", "Check syntax")
		return nil
	}
	// Leave parser AT the RBRACE (parser convention)

	return &ast.TestDecl{
		Name: name,
		Body: body,
		Pos:  pos,
	}
}

// parsePropertyDecl parses a top-level property declaration: property "name" { forall(...) => expr }
func (p *Parser) parsePropertyDecl() *ast.PropertyDecl {
	pos := p.curPos()

	// Accept either PROPERTY token or IDENT with literal "property" (contextual keyword)
	if !p.curTokenIs(lexer.PROPERTY) && !(p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "property") {
		p.report("PAR_UNEXPECTED_TOKEN", "expected 'property' keyword", "Check syntax")
		return nil
	}
	p.nextToken() // consume PROPERTY/IDENT

	// Expect string literal for property name
	if !p.curTokenIs(lexer.STRING) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected property name (string)", "Add property name: property \"my property\" {...}")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken() // consume STRING

	// Expect opening brace
	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected { to start property body", "Check syntax")
		return nil
	}
	p.nextToken() // consume LBRACE

	// Skip newlines
	for p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	// Parse the property (forall(...) => expr)
	property := p.parseProperty()
	if property == nil {
		return nil
	}

	// Skip newlines
	for p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	// Expect closing brace
	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected } to close property body", "Check syntax")
		return nil
	}
	// Leave parser AT the RBRACE (parser convention)

	return &ast.PropertyDecl{
		Name:     name,
		Property: property,
		Pos:      pos,
	}
}

// parseStatement parses a statement (for use in test blocks)
// Returns an Expr (either a regular expression or an AssertStmt which also implements Expr)
func (p *Parser) parseStatement() ast.Expr {
	// Check for assert statement
	if p.curTokenIs(lexer.ASSERT) {
		return p.parseAssertStmt()
	}

	// Otherwise, parse as expression
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}
	p.nextToken() // advance past expression

	return expr
}

// parseAssertStmt parses an assert statement: assert expr
func (p *Parser) parseAssertStmt() *ast.AssertStmt {
	pos := p.curPos()

	if !p.curTokenIs(lexer.ASSERT) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected 'assert' keyword", "Check syntax")
		return nil
	}
	p.nextToken() // consume ASSERT

	// Parse the condition expression
	condition := p.parseExpression(LOWEST)
	if condition == nil {
		p.report("PAR_UNEXPECTED_TOKEN", "expected expression after assert", "Add condition: assert x > 0")
		return nil
	}
	p.nextToken() // advance past expression

	return &ast.AssertStmt{
		Condition: condition,
		Message:   "", // TODO: Support optional message in future
		Pos:       pos,
	}
}

// peekIsContextualKeyword checks if the peek token is a specific keyword
func (p *Parser) peekIsContextualKeyword(keyword string) bool {
	return p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == keyword
}
