package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
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
		// FIXED (v0.4.2): Add implicit unit parameter for S-CALL0 compatibility
		fn.Params = []*ast.Param{
			{
				Name: "_", // Unnamed parameter (convention for ignored params)
				Type: &ast.SimpleType{Name: "()", Pos: p.curPos()},
				Pos:  p.curPos(),
			},
		}
		// Stay AT the UNIT token (don't advance) — matches non-generic branch convention.
		// The return type check (peekTokenIs(ARROW)) expects curToken to be the last param token.
	} else if hasTypeParams && p.curTokenIs(lexer.LPAREN) {
		// Generic function with parameters: func name[T](x: T)
		// Already at LPAREN after parseTypeParams()
		fn.Params = p.parseParams()
	} else if !hasTypeParams && p.peekTokenIs(lexer.UNIT) {
		// Non-generic function with unit parameter: func name()
		// FIXED (v0.4.2): Add implicit unit parameter for S-CALL0 compatibility
		// Zero-arg syntax func f() is sugar for func f(_: ()) - takes unit parameter
		p.nextToken()
		fn.Params = []*ast.Param{
			{
				Name: "_", // Unnamed parameter (convention for ignored params)
				Type: &ast.SimpleType{Name: "()", Pos: p.curPos()},
				Pos:  p.curPos(),
			},
		}
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
	}

	// Parse effects if present: ! {IO, FS}
	// Effects can appear after return type (-> T ! {IO}) or without one (func f(x) ! {IO})
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		fn.Effects = p.parseEffectAnnotation()
	}

	// Parse contracts if present: requires { ... } ensures { ... }
	// Contracts appear after effects, before tests/properties/body (M-VERIFY)
	// The syntax is:
	//   func name(params) -> type ! {}
	//   requires { pred1, pred2 }
	//   ensures  { pred3, pred4 }
	//   { body }
	requiresContracts, ensuresContracts := p.parseContractBlocks()
	if len(requiresContracts) > 0 || len(ensuresContracts) > 0 {
		// Append contracts to Properties slice (reusing existing infrastructure)
		fn.Properties = append(fn.Properties, requiresContracts...)
		fn.Properties = append(fn.Properties, ensuresContracts...)
	}

	// Parse tests and properties before body (they appear before opening brace)
	// The syntax is:
	//   func name(params) -> type
	//     tests [...]
	//     properties [...]
	//   {
	//     body
	//   }

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
		if body == nil {
			// parseExpression already recorded the underlying error (e.g. an
			// unsupported construct such as index access `x[i]` in the body);
			// don't nil-deref on .Position(). Matches the return nil below.
			return nil
		}
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
		// M-AILANG-ERROR-QUALITY (PAR020): a statement-starting token here means
		// the block is missing a ';' separator (mirror of PAR017's extra ';').
		// This is the #1 unactionable thrash-causer on small models —
		// config_file_parser burned 66 agent turns on a bare "expected }, got if".
		if !p.peekTokenIs(lexer.RBRACE) && p.peekStartsBlockStatement() {
			p.errors = append(p.errors, p.missingBlockSemicolonError())
			return nil
		}
		if !p.expectPeek(lexer.RBRACE) {
			return nil
		}
	}

	endPos := p.curPos()
	fn.Span = ast.Span{Start: startPos, End: endPos}
	return fn
}

// parseExternFunctionDeclaration parses an extern func declaration (Go-implemented function)
// Syntax: extern func name(params) -> ReturnType
// Extern functions have no body - they are implemented in Go
func (p *Parser) parseExternFunctionDeclaration() *ast.FuncDecl {
	startPos := p.curPos()

	if !p.curTokenIs(lexer.FUNC) {
		p.peekError(lexer.FUNC)
		return nil
	}

	fn := &ast.FuncDecl{
		IsExtern: true,
		Pos:      startPos,
		Origin:   "extern_func_decl",
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	fn.Name = p.curToken.Literal

	// Validate: extern functions cannot use underscore-prefixed names
	if strings.HasPrefix(fn.Name, "_") {
		p.errors = append(p.errors, NewParserError(
			"EXT001",
			p.curPos(),
			p.curToken,
			fmt.Sprintf("extern function '%s' cannot have underscore-prefix (reserved for builtins)", fn.Name),
			nil,
			"Use a public name without leading underscore"))
		return nil
	}

	// Extern functions cannot have type parameters (must be monomorphic)
	if p.peekTokenIs(lexer.LBRACKET) {
		p.errors = append(p.errors, NewParserError(
			"EXT002",
			p.curPos(),
			p.curToken,
			"extern functions cannot be polymorphic (no type parameters)",
			nil,
			"Extern functions must use concrete types like int, float, string"))
		return nil
	}

	// Parse parameters
	if p.peekTokenIs(lexer.UNIT) {
		// Zero-arg extern: extern func name()
		p.nextToken()
		fn.Params = []*ast.Param{
			{
				Name: "_",
				Type: &ast.SimpleType{Name: "()", Pos: p.curPos()},
				Pos:  p.curPos(),
			},
		}
	} else {
		if !p.expectPeek(lexer.LPAREN) {
			return nil
		}
		fn.Params = p.parseParams()
	}

	// Extern functions must have explicit return type
	if !p.peekTokenIs(lexer.ARROW) {
		p.errors = append(p.errors, NewParserError(
			"EXT003",
			p.curPos(),
			p.curToken,
			"extern functions must have explicit return type",
			[]lexer.TokenType{lexer.ARROW},
			"Add '-> ReturnType' after parameters"))
		return nil
	}

	p.nextToken() // move to ARROW
	p.nextToken() // move past ARROW to start of type
	fn.ReturnType = p.parseType()

	// Parse effects if present: ! {IO, FS}
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		fn.Effects = p.parseEffectAnnotation()
	}

	// Extern functions have no body
	fn.Body = nil

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

		// Check for trailing semicolon (next token is RBRACE)
		// Don't advance past it so caller can consume the RBRACE
		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		p.nextToken() // move past SEMICOLON

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
