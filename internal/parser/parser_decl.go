package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/lexer"
)

// ParseFile parses a complete AILANG source file
func (p *Parser) ParseFile() (file *ast.File) {
	// Add panic recovery to convert panics to parser errors
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to parser error
			var msg string
			if err, ok := r.(error); ok {
				msg = err.Error()
			} else {
				msg = fmt.Sprintf("%v", r)
			}

			p.errors = append(p.errors, NewParserError(
				errors.PAR999, // Generic parser panic code
				p.curPos(),
				p.curToken,
				fmt.Sprintf("parser panic: %s", msg),
				nil,
				"This is an internal parser error. Please report this issue."))

			// Return a minimal valid AST
			if file == nil {
				file = &ast.File{
					Decls:      []ast.Node{},
					Statements: []ast.Node{},
				}
			}
		}
	}()

	file = &ast.File{
		Pos: p.curPos(),
	}

	// Optional module declaration
	if p.curTokenIs(lexer.MODULE) {
		file.Module = p.parseModuleDecl()
		p.nextToken()
	}

	// Import declarations
	for p.curTokenIs(lexer.IMPORT) {
		imp := p.parseImportDecl()
		if imp != nil {
			file.Imports = append(file.Imports, imp)
		}
		p.nextToken()
	}

	// Export declarations (standalone export list)
	if p.curTokenIs(lexer.EXPORT) && p.peekTokenIs(lexer.LBRACE) {
		p.parseExportList()
		p.nextToken()
	}

	// Top-level declarations
	for !p.curTokenIs(lexer.EOF) {
		if decl := p.parseTopLevelDecl(); decl != nil {
			// Separate functions from other statements
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				file.Funcs = append(file.Funcs, funcDecl)
			} else {
				file.Statements = append(file.Statements, decl)
			}
			// Keep in Decls for backward compatibility
			file.Decls = append(file.Decls, decl)
		}
		if !p.curTokenIs(lexer.EOF) {
			p.nextToken()
		}
	}

	return file
}

// parseModuleDecl parses a module declaration
func (p *Parser) parseModuleDecl() *ast.ModuleDecl {
	startPos := p.curPos()
	p.expectPeek(lexer.IDENT)

	// Build module path (e.g., "foo/bar")
	path := p.curToken.Literal
	for p.peekTokenIs(lexer.SLASH) {
		p.nextToken() // consume slash
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		path += "/" + p.curToken.Literal
	}

	endPos := p.curPos()
	return &ast.ModuleDecl{
		Path: path,
		Pos:  startPos,
		Span: ast.Span{Start: startPos, End: endPos},
	}
}

// parseImportDecl parses an import declaration
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	startPos := p.curPos()
	imp := &ast.ImportDecl{
		Pos: startPos,
	}

	p.nextToken() // consume 'import'

	// Parse import path - can be string or path segments: ./relative, ../parent, std/io
	if p.curTokenIs(lexer.STRING) {
		imp.Path = p.curToken.Literal
	} else {
		// Build path from segments: segment ("/" segment)*
		// segment = IDENT | "." | ".."
		path := ""

		// Handle leading dots for relative paths
		if p.curTokenIs(lexer.DOT) {
			path = "."
			// Check for ./ or ../
			if p.peekTokenIs(lexer.DOT) {
				p.nextToken()
				path = ".."
			}
			if p.peekTokenIs(lexer.SLASH) {
				p.nextToken() // consume slash
				path += "/"
				p.nextToken() // move to next segment
			}
		}

		// Parse path segments
		if p.curTokenIs(lexer.IDENT) {
			if path != "" && !strings.HasSuffix(path, "/") {
				path += "/"
			}
			path += p.curToken.Literal

			for p.peekTokenIs(lexer.SLASH) {
				p.nextToken() // consume slash
				p.nextToken() // move to next segment

				if p.curTokenIs(lexer.IDENT) {
					path += "/" + p.curToken.Literal
				} else if p.curTokenIs(lexer.DOT) {
					// Handle .. in middle of path
					if p.peekTokenIs(lexer.DOT) {
						p.nextToken()
						path += "/.."
					} else {
						path += "/."
					}
				} else {
					p.errors = append(p.errors, NewParserError(errors.IMP010, p.curPos(), p.curToken,
						"expected path segment after /",
						[]lexer.TokenType{lexer.IDENT},
						"Add path segment or remove trailing /"))
					return nil
				}
			}
		} else if path == "" {
			// No valid path found
			p.errors = append(p.errors, NewParserError(errors.IMP001, p.curPos(), p.curToken,
				"expected import path",
				[]lexer.TokenType{lexer.STRING, lexer.IDENT, lexer.DOT},
				"Provide a valid import path"))
			return nil
		}

		imp.Path = path
	}

	// Check for selective imports: import module (symbol1, symbol2)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (
		p.nextToken() // move to first symbol

		for !p.curTokenIs(lexer.RPAREN) {
			if p.curTokenIs(lexer.IDENT) {
				imp.Symbols = append(imp.Symbols, p.curToken.Literal)
			}

			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next symbol
			} else {
				break
			}
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	} else {
		// Namespace imports not supported - require selective import
		p.errors = append(p.errors, NewParserError("IMP012_UNSUPPORTED_NAMESPACE", p.curPos(), p.curToken,
			"namespace imports not yet supported",
			[]lexer.TokenType{lexer.LPAREN},
			"Use selective import: import module/path (symbol1, symbol2)"))
		return nil
	}

	endPos := p.curPos()
	imp.Span = ast.Span{Start: startPos, End: endPos}
	return imp
}

// parseExportList parses a standalone export list: export { name1, name2 }
func (p *Parser) parseExportList() []string {
	var exports []string

	if !p.expectPeek(lexer.LBRACE) {
		return exports
	}
	p.nextToken() // move to first export

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			exports = append(exports, p.curToken.Literal)
		}

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next export
		} else {
			break
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		return exports
	}

	// Store exports in File's metadata (we'll need to extend the File struct later)
	return exports
}

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
	default:
		// Try to parse as an expression (for script-style files)
		return p.parseExpression(LOWEST)
	}
}

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
		p.nextToken() // consume 'tests'
		// Skip newlines after 'tests'
		for p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // move to LBRACKET
			// fn.Tests = p._parseTestsBlock() // TODO: Implement tests block
			// parseTestsBlock leaves us at RBRACKET, move past it
			if p.curTokenIs(lexer.RBRACKET) {
				p.nextToken()
			}
		}
		// Skip newlines after tests block
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
	}

	// Parse properties if present (before body)
	// Check for both PROPERTIES token (legacy) and contextual "properties" keyword
	if p.peekTokenIs(lexer.PROPERTIES) || p.peekIsContextualKeyword("properties") {
		p.nextToken() // consume 'properties'
		// Skip newlines after 'properties'
		for p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // move to LBRACKET
			// fn.Properties = p._parsePropertiesBlock() // TODO: Implement properties block
			// parsePropertiesBlock leaves us at RBRACKET, move past it
			if p.curTokenIs(lexer.RBRACKET) {
				p.nextToken()
			}
		}
		// Skip newlines after properties block
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
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

// peekIsContextualKeyword checks if the peek token is a specific keyword
func (p *Parser) peekIsContextualKeyword(keyword string) bool {
	return p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == keyword
}
