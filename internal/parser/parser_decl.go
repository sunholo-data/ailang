package parser

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseTopLevelDecl parses a top-level declaration
func (p *Parser) parseTopLevelDecl() ast.Node {
	switch p.curToken.Type {
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
