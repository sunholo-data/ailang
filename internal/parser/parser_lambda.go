package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

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

func (p *Parser) parsePureLambda() ast.Expr {
	// We're already at 'func' token after 'pure'.
	// parseLambda returns a nil ast.Expr on a malformed lambda (having already
	// recorded the real parse error). Guard the assertion so a bad `pure func ...`
	// surfaces that clean error instead of a PAR999 panic the agent can't act on.
	// M-AGENT-STUCK-FIXES M1: this assertion looped a benchmark agent for 87 steps.
	lambda, ok := p.parseLambda().(*ast.Lambda)
	if !ok {
		return nil
	}
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
		} else if p.peekTokenIs(lexer.ARROW) {
			// \x -> body is wrong; AILANG uses \x. body (dot, not arrow)
			p.nextToken() // consume -> to prevent cascading PAR_NO_PREFIX_PARSE
			p.errors = append(p.errors, fmt.Errorf("lambda body separator is '.' not '->' at %s\n\t\twrite: \\%s. <body>", p.curToken.Position(), params[len(params)-1].Name))
			return nil
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

	// M-GAP2: Keep multi-param lambdas as multi-param (NOT curried)
	// \x y. body should be a single lambda with params=[x,y], NOT nested lambdas
	// This allows \acc x. acc + x to unify with (b, a) -> b for foldl
	if len(params) == 0 {
		p.errors = append(p.errors, fmt.Errorf("lambda requires at least one parameter at %s", lambda.Pos.String()))
		return nil
	}
	lambda.Params = params
	return lambda
}

// Infix parse functions

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
