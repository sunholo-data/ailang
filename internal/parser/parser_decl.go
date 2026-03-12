package parser

import (
	"fmt"
	"strconv"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseVerifyAttribute parses an @verify(depth: N) attribute.
// Expects the parser to be AT the '@' token.
// Returns the parsed depth value, or nil if no @verify attribute.
// On error, reports a parser error and returns nil.
func (p *Parser) parseVerifyAttribute() *int {
	// We're at '@', consume it
	p.nextToken() // move past '@' to identifier

	if !p.curTokenIs(lexer.IDENT) || p.curToken.Literal != "verify" {
		p.report("PAR_INVALID_ATTRIBUTE",
			fmt.Sprintf("unknown attribute '@%s'; only @verify is supported", p.curToken.Literal),
			"Use @verify(depth: N) before a function declaration")
		return nil
	}

	// Consume 'verify', expect '('
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken() // move past '(' to key

	// Expect 'depth' identifier
	if !p.curTokenIs(lexer.IDENT) || p.curToken.Literal != "depth" {
		p.report("PAR_VERIFY_ATTR_KEY",
			fmt.Sprintf("expected 'depth' key in @verify attribute, got '%s'", p.curToken.Literal),
			"Use @verify(depth: N) where N is a positive integer")
		return nil
	}

	// Expect ':'
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Expect integer literal
	if !p.expectPeek(lexer.INT) {
		return nil
	}

	// Parse the integer value
	depth, err := strconv.Atoi(p.curToken.Literal)
	if err != nil {
		p.report("PAR_VERIFY_ATTR_VALUE",
			fmt.Sprintf("invalid depth value '%s': must be a positive integer", p.curToken.Literal),
			"Use @verify(depth: N) where N is 1-10")
		return nil
	}

	if depth < 0 || depth > 10 {
		p.report("PAR_VERIFY_ATTR_RANGE",
			fmt.Sprintf("depth %d out of range; must be 0-10", depth),
			"Use @verify(depth: N) where N is 0-10")
		return nil
	}

	// Expect ')'
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return &depth
}

// parseTopLevelDecl parses a top-level declaration
func (p *Parser) parseTopLevelDecl() ast.Node {
	switch p.curToken.Type {
	case lexer.AT:
		// Parse @verify(depth: N) attribute before function declaration
		verifyDepth := p.parseVerifyAttribute()
		p.nextToken() // move past ')' to next token

		// The next token must begin a function declaration (export, pure, func)
		var fn *ast.FuncDecl
		switch p.curToken.Type {
		case lexer.EXPORT:
			p.nextToken()
			if p.curTokenIs(lexer.FUNC) || p.curTokenIs(lexer.PURE) {
				fn = p.parseFunctionDeclaration(false, true)
			} else {
				p.report("PAR_ATTR_REQUIRES_FUNC",
					"@verify attribute must be followed by a function declaration",
					"Use @verify(depth: N) before 'func', 'export func', or 'pure func'")
				return nil
			}
		case lexer.PURE:
			if p.peekTokenIs(lexer.FUNC) {
				p.nextToken()
				fn = p.parseFunctionDeclaration(true, false)
			} else {
				p.report("PAR_ATTR_REQUIRES_FUNC",
					"@verify attribute must be followed by a function declaration",
					"Use @verify(depth: N) before 'func', 'export func', or 'pure func'")
				return nil
			}
		case lexer.FUNC:
			fn = p.parseFunctionDeclaration(false, false)
		default:
			p.report("PAR_ATTR_REQUIRES_FUNC",
				"@verify attribute must be followed by a function declaration",
				"Use @verify(depth: N) before 'func', 'export func', or 'pure func'")
			return nil
		}

		if fn != nil && verifyDepth != nil {
			fn.VerifyDepth = verifyDepth
		}
		return fn

	case lexer.EXPORT:
		// Handle export prefix
		p.nextToken()
		if p.curTokenIs(lexer.FUNC) || p.curTokenIs(lexer.PURE) {
			return p.parseFunctionDeclaration(false, true) // not pure yet, is export
		}
		if p.curTokenIs(lexer.TYPE) {
			return p.parseTypeDeclaration(true) // exported=true
		}
		if p.curTokenIs(lexer.LET) {
			// Error: export let not supported
			err := NewParserError(
				"PAR_UNSUPPORTED_EXPORT_LET",
				p.curPos(),
				p.curToken,
				"export let is not supported; use export func instead",
				[]lexer.TokenType{lexer.FUNC},
				"Change 'export let' to 'export func' with explicit parameters",
			)
			p.errors = append(p.errors, err)
			return nil
		}
		// Error: export must be followed by func, type, or pure
		err := NewParserError(
			"PAR_EXPORT_REQUIRES_FUNC",
			p.curPos(),
			p.curToken,
			fmt.Sprintf("export must be followed by 'func' or 'type', got '%s'", p.curToken.Literal),
			[]lexer.TokenType{lexer.FUNC, lexer.PURE, lexer.TYPE},
			"Use 'export func name(...) { ... }' or 'export type Name = ...'",
		)
		p.errors = append(p.errors, err)
		return nil
	case lexer.EXTERN:
		// Handle extern function declaration (Go-implemented functions)
		if p.peekTokenIs(lexer.FUNC) {
			p.nextToken() // consume 'extern'
			return p.parseExternFunctionDeclaration()
		}
		// Error: extern must be followed by func
		err := NewParserError(
			"PAR_EXTERN_REQUIRES_FUNC",
			p.curPos(),
			p.curToken,
			"extern must be followed by 'func'",
			[]lexer.TokenType{lexer.FUNC},
			"Use 'extern func name(params) -> ReturnType'",
		)
		p.errors = append(p.errors, err)
		return nil
	case lexer.PURE:
		// Check if it's a pure function declaration
		if p.peekTokenIs(lexer.FUNC) {
			p.nextToken()                                  // consume 'pure'
			return p.parseFunctionDeclaration(true, false) // is pure, not export yet
		}
		// Otherwise treat as expression
		return p.parseExpression(LOWEST)
	case lexer.FUNC:
		return p.parseFunctionDeclaration(false, false) // not pure, not export
	case lexer.TYPE:
		return p.parseTypeDeclaration(false) // exported=false
	case lexer.CLASS:
		return p.parseClassDeclaration()
	case lexer.INSTANCE:
		return p.parseInstanceDeclaration()
	case lexer.TEST:
		return p.parseTestDecl()
	case lexer.PROPERTY:
		return p.parsePropertyDecl()
	case lexer.IDENT:
		// Check for contextual keywords that look like identifiers
		if p.curToken.Literal == "test" {
			return p.parseTestDecl()
		}
		if p.curToken.Literal == "property" {
			return p.parsePropertyDecl()
		}
		// Otherwise continue with normal IDENT handling
		// Detect JavaScript/Python patterns that AIs commonly generate
		if p.curToken.Literal == "const" {
			// JavaScript const keyword
			err := NewSuggestionError(
				"PAR014",
				p.curPos(),
				p.curToken,
				"'const' keyword doesn't exist in AILANG (JavaScript/TypeScript pattern detected)",
				[]string{
					"Use: let name = value in ...",
					"Note: All bindings in AILANG are immutable by default",
				},
				"https://ailang.sunholo.com/docs/reference/language-syntax",
			)
			p.errors = append(p.errors, err)
			return nil
		}
		// Check for bare assignment (Python-style: x = y without let)
		if p.peekTokenIs(lexer.ASSIGN) {
			err := NewSuggestionError(
				"PAR015",
				p.curPos(),
				p.curToken,
				"bare assignment not supported (missing 'let' keyword)",
				[]string{
					fmt.Sprintf("Use: let %s = ... in", p.curToken.Literal),
					"AILANG requires 'let' keyword for bindings",
				},
				"https://ailang.sunholo.com/docs/reference/language-syntax",
			)
			p.errors = append(p.errors, err)
			return nil
		}
		// Otherwise treat as expression
		expr := p.parseExpression(LOWEST)

		// S-CALL0: Check for statement-level zero-arg call pattern: f()
		// The lexer tokenizes () as a single UNIT token when there's no space.
		// If we just parsed an identifier and peekToken is UNIT, this is f()
		if ident, ok := expr.(*ast.Identifier); ok {
			if p.peekTokenIs(lexer.UNIT) {
				// Detected f() pattern: identifier followed by () unit literal
				// This should be a zero-arg function call, not an identifier followed by unit
				p.nextToken() // advance to UNIT token

				// Check strict mode
				if p.strictSyntaxMode {
					p.reportSugarError("CALL0", "f()", "f (())")
					return expr // Return identifier unchanged
				}

				// Create call expression with unit argument
				p.sugarUsed = true
				call := &ast.FuncCall{
					Func: ident,
					Args: []ast.Expr{
						&ast.Literal{
							Kind:  ast.UnitLit,
							Value: nil,
							Pos:   p.curPos(),
						},
					},
					Pos: ident.Pos,
				}
				return call
			}
		}

		return expr
	default:
		// Try to parse as an expression (for script-style files)
		expr := p.parseExpression(LOWEST)

		// S-CALL0: Check for statement-level zero-arg call pattern: f()
		if ident, ok := expr.(*ast.Identifier); ok {
			if p.peekTokenIs(lexer.UNIT) {
				p.nextToken() // advance to UNIT token

				if p.strictSyntaxMode {
					p.reportSugarError("CALL0", "f()", "f (())")
					return expr
				}

				p.sugarUsed = true
				call := &ast.FuncCall{
					Func: ident,
					Args: []ast.Expr{
						&ast.Literal{
							Kind:  ast.UnitLit,
							Value: nil,
							Pos:   p.curPos(),
						},
					},
					Pos: ident.Pos,
				}
				return call
			}
		}

		return expr
	}
}
