package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseFunctionDeclaration parses a function declaration
func (p *Parser) parseFunctionDeclaration(isPure bool, isExport bool) *ast.FuncDecl {
	startPos := p.curPos()

	// Handle export prefix if not already set
	if !isExport && p.curTokenIs(lexer.EXPORT) {
		isExport = true
		p.nextToken()
	}

	// Handle pure prefix if not already set
	if !isPure && p.curTokenIs(lexer.PURE) {
		isPure = true
		p.nextToken()
	}

	if !p.curTokenIs(lexer.FUNC) {
		p.peekError(lexer.FUNC)
		return nil
	}

	fn := &ast.FuncDecl{
		IsPure:   isPure,
		IsExport: isExport,
		Pos:      startPos,
		Origin:   "func_decl",
	}

	p.expectPeek(lexer.IDENT)
	fn.Name = p.curToken.Literal

	// Validate: cannot export underscore-prefixed (private) names
	if isExport && strings.HasPrefix(fn.Name, "_") {
		p.errors = append(p.errors, NewParserError(
			"MOD006",
			p.curPos(),
			p.curToken,
			fmt.Sprintf("cannot export private (underscore-prefixed) name '%s'", fn.Name),
			nil,
			"Remove leading underscore or drop 'export' keyword"))
		return nil
	}

	// Parse type parameters if present
	if p.peekTokenIs(lexer.LBRACKET) {
		p.nextToken()
		fn.TypeParams = p.parseTypeParams()
		// After parseTypeParams(), we're now AT the token after ]
		// For generic functions: func name[T](params), we're at (
		// No need to peek - we're already positioned correctly
	}

	// Parse parameters
	hasTypeParams := len(fn.TypeParams) > 0

	if hasTypeParams && p.curTokenIs(lexer.UNIT) {
		// Generic function with unit parameter: func name[T]()
		fn.Params = []*ast.Param{}
		p.nextToken() // consume UNIT
	} else if hasTypeParams && p.curTokenIs(lexer.LPAREN) {
		// Generic function with parameters: func name[T](x: T)
		// Already at LPAREN after parseTypeParams()
		fn.Params = p.parseParams()
	} else if !hasTypeParams && p.peekTokenIs(lexer.UNIT) {
		// Non-generic function with unit parameter: func name()
		p.nextToken()
		fn.Params = []*ast.Param{}
	} else {
		// Non-generic function with parameters: func name(x: int)
		p.expectPeek(lexer.LPAREN)
		fn.Params = p.parseParams()
	}

	// Parse return type if present
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken()
		p.nextToken()
		fn.ReturnType = p.parseType()

		// Parse effects if present: ! {IO, FS}
		if p.peekTokenIs(lexer.BANG) {
			p.nextToken() // move to BANG
			fn.Effects = p.parseEffectAnnotation()
		}
	}

	// Parse tests and properties before body (they appear before opening brace)
	// The syntax is:
	//   func name(params) -> type
	//     tests [...]
	//     properties [...]
	//   {
	//     body
	//   }

	// Skip any newlines/whitespace before tests/properties/body
	for p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	// Parse tests if present (before body)
	// Check for both TESTS token (legacy) and contextual "tests" keyword
	if p.peekTokenIs(lexer.TESTS) || p.peekIsContextualKeyword("tests") {
		p.nextToken() // consume 'tests', now cur=tests, peek=next
		// Now cur should be "tests" identifier
		// Look for [ which should be next (peek)
		if !p.peekTokenIs(lexer.LBRACKET) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected [ after tests keyword", "Check syntax")
		} else {
			p.nextToken() // move to LBRACKET, now cur=[, peek=first_test_token
			fn.Tests = p.parseTestsBlock()
			// parseTestsBlock leaves us at RBRACKET, move past it
			if p.curTokenIs(lexer.RBRACKET) {
				p.nextToken()
			}
		}
	}

	// Parse properties if present (before body)
	// Check for both PROPERTIES token and contextual "properties" keyword
	// Could be in peek (no tests block) or cur (after tests block)
	if p.peekTokenIs(lexer.PROPERTIES) || p.peekIsContextualKeyword("properties") ||
		p.curTokenIs(lexer.PROPERTIES) || (p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "properties") {
		// If in peek, advance to it
		if p.peekTokenIs(lexer.PROPERTIES) || p.peekIsContextualKeyword("properties") {
			p.nextToken() // move to 'properties'
		}
		// Now cur='properties', peek=next (should be [)
		if !p.peekTokenIs(lexer.LBRACKET) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected [ after properties keyword", "Check syntax")
		} else {
			p.nextToken() // move to LBRACKET, now cur=[, peek=first_property_token
			fn.Properties = p.parsePropertiesBlock()
			// parsePropertiesBlock leaves us at RBRACKET, move past it
			if p.curTokenIs(lexer.RBRACKET) {
				p.nextToken()
			}
		}
	}

	// Parse body: either equation-form (= expr) or block ({ ... })
	// Equation-form: export func f(x: int) -> int = x * 2
	// Block-form: export func f(x: int) -> int { x * 2 }

	// Check if we're already at LBRACE (block-form) or ASSIGN (equation-form)
	if p.peekTokenIs(lexer.ASSIGN) {
		// Equation-form: consume = and parse expression
		p.nextToken() // move to ASSIGN
		p.nextToken() // move past ASSIGN to start of expression

		body := p.parseExpression(LOWEST)
		// Wrap single expression in a block for uniform handling
		fn.Body = &ast.Block{
			Exprs: []ast.Expr{body},
			Pos:   body.Position(),
		}
	} else {
		// Block-form: expect LBRACE
		if !p.curTokenIs(lexer.LBRACE) {
			if !p.expectPeek(lexer.LBRACE) {
				return nil
			}
		}
		// Parse body as a block (semicolon-separated expressions)
		fn.Body = p.parseFunctionBody()
		if !p.expectPeek(lexer.RBRACE) {
			return nil
		}
	}

	endPos := p.curPos()
	fn.Span = ast.Span{Start: startPos, End: endPos}
	return fn
}

// parseFunctionBody parses a function body as a block of semicolon-separated expressions
// Assumes we're currently AT the LBRACE token
// Returns either a single expression or a Block containing multiple expressions
func (p *Parser) parseFunctionBody() ast.Expr {
	startPos := p.curPos()
	p.nextToken() // move past LBRACE

	// Empty body: {}
	if p.curTokenIs(lexer.RBRACE) {
		return &ast.Block{
			Exprs: []ast.Expr{},
			Pos:   startPos,
		}
	}

	// Parse first expression
	var exprs []ast.Expr
	expr := p.parseExpression(LOWEST)
	if expr != nil {
		exprs = append(exprs, expr)
	}

	// Continue parsing while we see semicolons
	for p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to SEMICOLON
		p.nextToken() // move past SEMICOLON

		// Skip trailing semicolon before closing brace
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		expr = p.parseExpression(LOWEST)
		if expr != nil {
			exprs = append(exprs, expr)
		}
	}

	// If we only have one expression, return it directly (not wrapped in a Block)
	if len(exprs) == 1 {
		return exprs[0]
	}

	// Multiple expressions: return as a Block
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}

func (p *Parser) parseClassDeclaration() ast.Node {
	// TODO: Implement class declaration parsing
	return nil
}

func (p *Parser) parseInstanceDeclaration() ast.Node {
	// TODO: Implement instance declaration parsing
	return nil
}
