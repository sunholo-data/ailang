package parser

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseExpression parses an expression with precedence
func (p *Parser) parseExpression(precedence int) ast.Expr {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

// Prefix parse functions

func (p *Parser) parsePrefixExpression() ast.Expr {
	// Special case: BANG followed by LBRACE is an effect annotation, not a prefix operator
	if p.curTokenIs(lexer.BANG) && p.peekTokenIs(lexer.LBRACE) {
		return nil // Not a prefix expression, let caller handle it
	}

	expr := &ast.UnaryOp{
		Op:  p.curToken.Literal,
		Pos: p.curPos(),
	}

	p.nextToken()
	expr.Expr = p.parseExpression(PREFIX)

	return expr
}

func (p *Parser) parseIfExpression() ast.Expr {
	expr := &ast.If{
		Pos: p.curPos(),
	}

	p.nextToken()
	expr.Condition = p.parseExpression(LOWEST)

	p.expectPeek(lexer.THEN)
	p.nextToken()
	expr.Then = p.parseExpression(LOWEST)

	p.expectPeek(lexer.ELSE)
	p.nextToken()
	expr.Else = p.parseExpression(LOWEST)

	return expr
}

func (p *Parser) parseLetExpression() ast.Expr {
	let := &ast.Let{
		Pos: p.curPos(),
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	let.Name = p.curToken.Literal

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		let.Type = p.parseType()
		if let.Type == nil {
			// If type parsing failed, continue anyway
			let.Type = &ast.SimpleType{Name: "unknown", Pos: p.curPos()}
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return let // Return partial AST
	}
	p.nextToken()
	let.Value = p.parseExpression(LOWEST)
	if let.Value == nil {
		// If value parsing failed, create error node
		let.Value = &ast.Error{Pos: p.curPos()}
	}

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		let.Body = p.parseExpression(LOWEST)
		if let.Body == nil {
			// If body parsing failed, create error node
			let.Body = &ast.Error{Pos: p.curPos()}
		}
	}

	return let
}

func (p *Parser) parseLetRecExpression() ast.Expr {
	letrec := &ast.LetRec{
		Pos: p.curPos(),
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	letrec.Name = p.curToken.Literal

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		letrec.Type = p.parseType()
		if letrec.Type == nil {
			// If type parsing failed, continue anyway
			letrec.Type = &ast.SimpleType{Name: "unknown", Pos: p.curPos()}
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return letrec // Return partial AST
	}
	p.nextToken()
	letrec.Value = p.parseExpression(LOWEST)
	if letrec.Value == nil {
		// If value parsing failed, create error node
		letrec.Value = &ast.Error{Pos: p.curPos()}
	}

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		letrec.Body = p.parseExpression(LOWEST)
		if letrec.Body == nil {
			// If body parsing failed, create error node
			letrec.Body = &ast.Error{Pos: p.curPos()}
		}
	}

	return letrec
}

func (p *Parser) parseMatchExpression() ast.Expr {
	match := &ast.Match{
		Pos: p.curPos(),
	}

	p.nextToken()
	match.Expr = p.parseExpression(LOWEST)

	p.expectPeek(lexer.LBRACE)
	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		c := p.parseCase()
		if c != nil {
			match.Cases = append(match.Cases, c)
		}

		// Move to next token after parsing case
		p.nextToken()

		// Skip comma if present
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	// We should already be at RBRACE
	if !p.curTokenIs(lexer.RBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
	}

	return match
}

func (p *Parser) parseCase() *ast.Case {
	c := &ast.Case{
		Pos: p.curPos(),
	}

	c.Pattern = p.parsePattern()

	// Optional guard
	if p.peekTokenIs(lexer.IF) {
		p.nextToken()
		p.nextToken()
		c.Guard = p.parseExpression(LOWEST)
	}

	p.expectPeek(lexer.FARROW)
	p.nextToken()
	c.Body = p.parseExpression(LOWEST)

	return c
}

func (p *Parser) parseLambda() ast.Expr {
	pos := p.curPos()

	// Expect opening parenthesis
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Parse parameters
	params := p.parseParams()

	// Check which syntax we're using:
	// - func(x) -> type { body }  (new FuncLit syntax)
	// - func(x) => body           (old Lambda syntax)

	if p.peekTokenIs(lexer.ARROW) {
		// New FuncLit syntax: func(x: int) -> int { body }
		return p.parseFuncLitWithParams(pos, params)
	} else if p.peekTokenIs(lexer.FARROW) {
		// Old Lambda syntax: func(x) => body
		lambda := &ast.Lambda{
			Pos:    pos,
			Params: params,
		}
		p.nextToken() // consume =>
		p.nextToken()
		lambda.Body = p.parseExpression(LOWEST)
		return lambda
	} else {
		p.errors = append(p.errors, fmt.Errorf("expected '->' or '=>' after function parameters at %s", p.peekToken.Position()))
		return nil
	}
}

// parseFuncLitWithParams parses the rest of a function literal after params have been parsed
// Syntax: (already parsed: func(params)) -> returnType ! {effects} { body }
func (p *Parser) parseFuncLitWithParams(pos ast.Pos, params []*ast.Param) ast.Expr {
	funcLit := &ast.FuncLit{
		Pos:    pos,
		Params: params,
	}

	// Consume '->'
	if !p.expectPeek(lexer.ARROW) {
		return nil
	}
	p.nextToken() // move to return type

	// Parse return type
	funcLit.ReturnType = p.parseType()

	// Parse optional effect annotation: func() -> int ! {IO}
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		funcLit.Effects = p.parseEffectAnnotation()
	}

	// Expect body in braces: { expr }
	if !p.expectPeek(lexer.LBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected '{' for function body at %s", p.peekToken.Position()))
		return nil
	}

	// Parse body as a block or expression
	funcLit.Body = p.parseBlockOrExpression()

	return funcLit
}

// parseBlockOrExpression parses either a block { e1; e2; e3 } or a single expression
// This is called when we're at the opening LBRACE
func (p *Parser) parseBlockOrExpression() ast.Expr {
	// We're at LBRACE
	startPos := p.curPos()
	p.nextToken() // consume LBRACE

	// Check for empty block: {}
	if p.curTokenIs(lexer.RBRACE) {
		// Empty block returns unit
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Parse expressions separated by semicolons
	exprs := []ast.Expr{}
	exprs = append(exprs, p.parseExpression(LOWEST))

	// Keep parsing while we see semicolons
	for p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to SEMICOLON
		p.nextToken() // move past SEMICOLON

		// Check for trailing semicolon before RBRACE
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		exprs = append(exprs, p.parseExpression(LOWEST))
	}

	// Expect closing brace
	if !p.expectPeek(lexer.RBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected '}' to close function body at %s", p.peekToken.Position()))
		return nil
	}

	// If single expression, return it directly (not as block)
	if len(exprs) == 1 {
		return exprs[0]
	}

	// Multiple expressions: return as block
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}

func (p *Parser) parsePureLambda() ast.Expr {
	// We're already at 'func' token after 'pure'
	lambda := p.parseLambda().(*ast.Lambda)
	// Mark as pure somehow
	return lambda
}

// parseBackslashLambda parses lambda expressions with \x. syntax
func (p *Parser) parseBackslashLambda() ast.Expr {
	lambda := &ast.Lambda{
		Pos: p.curPos(),
	}

	// Parse parameters - support curried sugar \x y z. body
	var params []*ast.Param

	// Keep consuming identifiers until we hit DOT
	for {
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		param := &ast.Param{
			Name: p.curToken.Literal,
			Pos:  p.curPos(),
			// Type will be inferred
		}
		params = append(params, param)

		// Check if next token is DOT (end of params) or another IDENT (more params)
		if p.peekTokenIs(lexer.DOT) {
			break
		} else if !p.peekTokenIs(lexer.IDENT) {
			p.errors = append(p.errors, fmt.Errorf("expected '.' after lambda parameter at %s", p.peekToken.Position()))
			return nil
		}
	}

	// Expect DOT
	if !p.expectPeek(lexer.DOT) {
		return nil
	}

	// Parse body with LOWEST precedence to capture entire expression
	p.nextToken()
	lambda.Body = p.parseExpression(LOWEST)

	// Parse optional effect annotation: \x. body ! {IO}
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		lambda.Effects = p.parseEffectAnnotation()
	}

	// Convert curried parameters to nested lambdas: \x y. body -> \x. \y. body
	if len(params) == 0 {
		p.errors = append(p.errors, fmt.Errorf("lambda requires at least one parameter at %s", lambda.Pos.String()))
		return nil
	} else if len(params) == 1 {
		lambda.Params = params
	} else {
		// Create nested lambdas for curried syntax
		lambda.Params = []*ast.Param{params[0]}

		// Create nested lambda for remaining parameters
		innerLambda := &ast.Lambda{
			Pos:  lambda.Pos,
			Body: lambda.Body,
		}

		// Recursively create nested structure
		current := innerLambda
		for i := 1; i < len(params)-1; i++ {
			current.Params = []*ast.Param{params[i]}
			next := &ast.Lambda{
				Pos: lambda.Pos,
			}
			current.Body = next
			current = next
		}

		// Set the last parameter and body
		current.Params = []*ast.Param{params[len(params)-1]}
		current.Body = lambda.Body

		lambda.Body = innerLambda
	}

	return lambda
}

// Infix parse functions

func (p *Parser) parseInfixExpression(left ast.Expr) ast.Expr {
	expr := &ast.BinaryOp{
		Left: left,
		Op:   p.curToken.Literal,
		Pos:  p.curPos(),
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parseCallExpression(fn ast.Expr) ast.Expr {
	call := &ast.FuncCall{
		Func: fn,
		Pos:  p.curPos(),
	}

	call.Args = p.parseCallArguments()
	return call
}

func (p *Parser) parseCallArguments() []ast.Expr {
	args := []ast.Expr{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	p.expectPeek(lexer.RPAREN)
	return args
}

func (p *Parser) parseRecordAccess(record ast.Expr) ast.Expr {
	access := &ast.RecordAccess{
		Record: record,
		Pos:    p.curPos(),
	}

	p.expectPeek(lexer.IDENT)
	access.Field = p.curToken.Literal

	return access
}

func (p *Parser) parseSendExpression(channel ast.Expr) ast.Expr {
	send := &ast.Send{
		Channel: channel,
		Pos:     p.curPos(),
	}

	p.nextToken()
	send.Value = p.parseExpression(LOWEST)

	return send
}

// Helper parsing functions

func (p *Parser) parseParams() []*ast.Param {
	params := []*ast.Param{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	param := &ast.Param{
		Pos: p.curPos(),
	}

	if p.curTokenIs(lexer.IDENT) {
		param.Name = p.curToken.Literal

		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			param.Type = p.parseType()
		}
	}

	params = append(params, param)

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()

		param := &ast.Param{
			Pos: p.curPos(),
		}

		if p.curTokenIs(lexer.IDENT) {
			param.Name = p.curToken.Literal

			if p.peekTokenIs(lexer.COLON) {
				p.nextToken()
				p.nextToken()
				param.Type = p.parseType()
			}
		}

		params = append(params, param)
	}

	p.expectPeek(lexer.RPAREN)
	return params
}
