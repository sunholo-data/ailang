package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/lexer"
)

// ParserError represents a structured parser error with fix suggestions
type ParserError struct {
	Code       string
	Message    string
	Pos        ast.Pos
	NearToken  lexer.Token
	Expected   []lexer.TokenType
	Fix        string
	Confidence float64
}

func (e *ParserError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Pos, e.Message)
}

// NewParserError creates a structured parser error with fix suggestion
func NewParserError(code string, pos ast.Pos, nearToken lexer.Token, message string, expected []lexer.TokenType, fix string) *ParserError {
	return &ParserError{
		Code:       code,
		Message:    message,
		Pos:        pos,
		NearToken:  nearToken,
		Expected:   expected,
		Fix:        fix,
		Confidence: 0.85, // Default confidence for parser fixes
	}
}

// report is a convenience helper for adding structured errors to the parser
func (p *Parser) report(code string, message string, fix string) {
	err := NewParserError(code, p.curPos(), p.curToken, message, nil, fix)
	p.errors = append(p.errors, err)
}

// reportExpected is a convenience helper for "expected X, got Y" errors
func (p *Parser) reportExpected(expected lexer.TokenType, fix string) {
	message := fmt.Sprintf("expected %s, got %s", expected, p.curToken.Type)
	err := NewParserError(
		"PAR_UNEXPECTED_TOKEN",
		p.curPos(),
		p.curToken,
		message,
		[]lexer.TokenType{expected},
		fix,
	)
	p.errors = append(p.errors, err)
}

// Parser parses AILANG source code into an AST
type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []error

	// Pratt parsing
	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

// Precedence levels - spec compliant ordering
const (
	LOWEST      int = iota
	LAMBDA          // \x. (lowest precedence)
	LogicalOr       // ||
	LogicalAnd      // &&
	EQUALS          // ==, !=
	LESSGREATER     // >, <, >=, <=
	APPEND          // ++
	SUM             // +, -
	PRODUCT         // *, /, %
	PREFIX          // -x, !x (unary)
	CALL            // f(x) (application)
	DotAccess       // r.field (field access - highest)
	HIGHEST
)

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []error{},
	}

	// Register prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.CHAR, p.parseCharLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.UNIT, p.parseUnitLiteral)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACKET, p.parseListLiteral)
	p.registerPrefix(lexer.LBRACE, p.parseRecordLiteral)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.IF, p.parseIfExpression)
	p.registerPrefix(lexer.LET, p.parseLetExpression)
	p.registerPrefix(lexer.MATCH, p.parseMatchExpression)
	p.registerPrefix(lexer.FUNC, p.parseLambda)
	p.registerPrefix(lexer.PURE, p.parsePureLambda)
	p.registerPrefix(lexer.BACKSLASH, p.parseBackslashLambda)

	// Register infix parse functions
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.STAR, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NEQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LTE, p.parseInfixExpression)
	p.registerInfix(lexer.GTE, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
	p.registerInfix(lexer.APPEND, p.parseInfixExpression)
	p.registerInfix(lexer.CONS, p.parseInfixExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.DOT, p.parseRecordAccess)
	p.registerInfix(lexer.LARROW, p.parseSendExpression)

	// Read two tokens to set curToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns parser errors
func (p *Parser) Errors() []error {
	return p.errors
}

// isContextualKeyword checks if the current token is a specific keyword
// This is used for contextual keywords like "tests" that are returned as IDENT

// peekIsContextualKeyword checks if the peek token is a specific keyword
func (p *Parser) peekIsContextualKeyword(keyword string) bool {
	return p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == keyword
}

// Parse parses the input and returns an AST
func (p *Parser) Parse() *ast.Program {
	// Recover from panics and convert to structured errors
	defer func() {
		if r := recover(); r != nil {
			p.errors = append(p.errors, NewParserError(
				"PAR999_INTERNAL_ERROR",
				p.curPos(),
				p.curToken,
				fmt.Sprintf("internal parser panic: %v", r),
				nil,
				"Please report this as a bug at https://github.com/sunholo/ailang/issues"))
		}
	}()

	program := &ast.Program{}

	// Parse as a File structure
	file := p.ParseFile()
	program.File = file

	// Legacy support: also populate Module field
	if file.Module != nil {
		module := &ast.Module{
			Name: file.Module.Path,
			Pos:  file.Module.Pos,
		}
		// Convert ImportDecls to Imports
		for _, imp := range file.Imports {
			module.Imports = append(module.Imports, &ast.Import{
				Path:    imp.Path,
				Symbols: imp.Symbols,
				Pos:     imp.Pos,
			})
		}
		module.Decls = file.Decls
		program.Module = module
	}

	return program
}

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

// parseModule parses a module declaration (legacy)

// parseImport parses an import statement

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

// parseDeclaration parses a top-level declaration (legacy)

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

func (p *Parser) parseIdentifier() ast.Expr {
	return &ast.Identifier{
		Name: p.curToken.Literal,
		Pos:  p.curPos(),
	}
}

func (p *Parser) parseIntegerLiteral() ast.Expr {
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}

	return &ast.Literal{
		Kind:  ast.IntLit,
		Value: value,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseFloatLiteral() ast.Expr {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("could not parse %q as float", p.curToken.Literal))
		return nil
	}

	return &ast.Literal{
		Kind:  ast.FloatLit,
		Value: value,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseStringLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.StringLit,
		Value: p.curToken.Literal,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseCharLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.StringLit, // Treat chars as single-char strings for now
		Value: p.curToken.Literal,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.BoolLit,
		Value: p.curTokenIs(lexer.TRUE),
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseUnitLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.UnitLit,
		Value: nil,
		Pos:   p.curPos(),
	}
}

// parseGroupedExpression parses grouped expressions and tuples
// EBNF:
//
//	tuple_expr := "(" expr "," expr ("," expr)* ","? ")"
//	grouped    := "(" expr ")"
//
// Disambiguation: A comma is required to form a tuple. (e) is grouping, (e,) is a tuple.
func (p *Parser) parseGroupedExpression() ast.Expr {
	startPos := p.curPos()
	p.nextToken() // consume LPAREN

	// Handle empty tuple/unit: ()
	if p.curTokenIs(lexer.RPAREN) {
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Parse first expression
	expr := p.parseExpression(LOWEST)

	// After parsing expression, we're at the last token of that expression
	// Need to advance to see what comes next
	if !p.peekTokenIs(lexer.COMMA) {
		// Just a grouped expression - no comma
		if !p.expectPeek(lexer.RPAREN) {
			p.reportExpected(lexer.RPAREN, "Add ')' to close grouped expression")
		}
		return expr
	}

	// It's a tuple - comma is required
	elements := []ast.Expr{expr}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // move to COMMA
		p.nextToken() // move past COMMA to next element

		// Check for trailing comma
		if p.curTokenIs(lexer.RPAREN) {
			return &ast.Tuple{
				Elements: elements,
				Pos:      startPos,
			}
		}

		elem := p.parseExpression(LOWEST)
		elements = append(elements, elem)
	}

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		p.reportExpected(lexer.RPAREN, "Add ')' to close tuple")
	}

	return &ast.Tuple{
		Elements: elements,
		Pos:      startPos,
	}
}

func (p *Parser) parseListLiteral() ast.Expr {
	list := &ast.List{
		Pos: p.curPos(),
	}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		list.Elements = append(list.Elements, p.parseExpression(LOWEST))

		if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken()
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.RBRACKET) {
		p.expectPeek(lexer.RBRACKET)
	}

	return list
}

func (p *Parser) parseRecordLiteral() ast.Expr {
	startPos := p.curPos()
	p.nextToken() // move past LBRACE

	// Empty block: {}
	if p.curTokenIs(lexer.RBRACE) {
		return &ast.Block{
			Exprs: []ast.Expr{},
			Pos:   startPos,
		}
	}

	// Peek ahead to determine if this is a record literal or a block
	// Record literals have the pattern: IDENT COLON ...
	// Blocks have expressions (which might start with anything)
	isRecord := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)

	if isRecord {
		// Parse as record literal
		record := &ast.Record{
			Pos: startPos,
		}

		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			field := &ast.Field{
				Pos: p.curPos(),
			}

			if !p.curTokenIs(lexer.IDENT) {
				p.errors = append(p.errors, fmt.Errorf("expected field name, got %s", p.curToken.Type))
				return nil
			}

			field.Name = p.curToken.Literal

			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()

			field.Value = p.parseExpression(LOWEST)
			record.Fields = append(record.Fields, field)

			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken()
				break
			}

			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken()
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
			return nil
		}

		return record
	} else {
		// Parse as block (semicolon-separated expressions)
		block := &ast.Block{
			Pos: startPos,
		}

		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			expr := p.parseExpression(LOWEST)
			if expr != nil {
				block.Exprs = append(block.Exprs, expr)
			}

			// If we see a semicolon, consume it and continue
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken() // move to SEMICOLON
				p.nextToken() // move past SEMICOLON
				continue
			}

			// If we see RBRACE next, we're done
			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken() // move to RBRACE
				break
			}

			// Otherwise we expect a semicolon or RBRACE
			if !p.curTokenIs(lexer.RBRACE) {
				p.errors = append(p.errors, fmt.Errorf("expected ; or }, got %s", p.peekToken.Type))
				return nil
			}
			break
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
			return nil
		}

		return block
	}
}

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

// parseType parses a type expression
// Handles: identifiers, type variables, lists, tuples, functions
func (p *Parser) parseType() ast.Type {
	switch p.curToken.Type {
	case lexer.LBRACE:
		// Record type expression: { field: Type, ... }
		return p.parseRecordTypeExpr()

	case lexer.IDENT:
		// Simple type or type variable
		name := p.curToken.Literal
		startPos := p.curPos()

		// Check for type application: List[int], Option[a], etc.
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // consume IDENT
			p.nextToken() // consume LBRACKET

			// For now, parse type args but don't use them
			// TODO: Proper type application parsing with TypeApp AST node
			_ = p.parseType() // first arg
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // move to COMMA
				p.nextToken() // move past COMMA
				_ = p.parseType()
			}

			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}

			// Return a SimpleType for now (proper generics parsing would be more complex)
			return &ast.SimpleType{
				Name: name, // e.g., "Option" or "List"
				Pos:  startPos,
			}
		}

		// Check if it's a built-in type (lowercase but not type vars)
		builtinTypes := map[string]bool{
			"int": true, "float": true, "string": true, "bool": true,
			"unit": true, "char": true,
		}
		if builtinTypes[name] {
			return &ast.SimpleType{
				Name: name,
				Pos:  startPos,
			}
		}

		// Check if it's a type variable (lowercase single letter) or type constructor (uppercase)
		if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
			return &ast.TypeVar{
				Name: name,
				Pos:  startPos,
			}
		}

		return &ast.SimpleType{
			Name: name,
			Pos:  startPos,
		}

	case lexer.UNIT:
		// Unit type ()
		return &ast.SimpleType{
			Name: "()",
			Pos:  p.curPos(),
		}

	case lexer.LBRACKET:
		// List type: [T]
		startPos := p.curPos()
		p.nextToken() // consume LBRACKET
		elemType := p.parseType()
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		return &ast.ListType{
			Element: elemType,
			Pos:     startPos,
		}

	case lexer.LPAREN:
		// Could be:
		// - Unit type: ()
		// - Tuple type: (T1, T2, ...)
		// - Function type: (T1, T2) -> T3
		// - Grouped type: (T)
		startPos := p.curPos()
		p.nextToken() // consume LPAREN

		// Handle unit type
		if p.curTokenIs(lexer.RPAREN) {
			return &ast.SimpleType{
				Name: "()",
				Pos:  startPos,
			}
		}

		// Parse first type
		firstType := p.parseType()

		// Check what comes next
		if p.peekTokenIs(lexer.RPAREN) {
			// Could be (T) or (T) -> ...
			p.nextToken() // move to RPAREN

			// Check for arrow (function type)
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken() // consume RPAREN
				p.nextToken() // consume ARROW
				retType := p.parseType()

				// Parse optional effect annotation: (int) -> string ! {IO}
				var effects []string
				if p.peekTokenIs(lexer.BANG) {
					p.nextToken() // move to BANG
					effects = p.parseEffectAnnotation()
				}

				return &ast.FuncType{
					Params:  []ast.Type{firstType},
					Return:  retType,
					Effects: effects,
					Pos:     startPos,
				}
			}

			// Just a grouped type
			return firstType
		}

		if p.peekTokenIs(lexer.COMMA) {
			// Tuple type: (T1, T2, ...)
			types := []ast.Type{firstType}
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // move to COMMA
				p.nextToken() // move past COMMA
				if p.curTokenIs(lexer.RPAREN) {
					break // trailing comma
				}
				types = append(types, p.parseType())
			}

			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}

			// Check for arrow (function type with multiple params)
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken() // consume RPAREN
				p.nextToken() // consume ARROW
				retType := p.parseType()

				// Parse optional effect annotation: (int, string) -> bool ! {IO, FS}
				var effects []string
				if p.peekTokenIs(lexer.BANG) {
					p.nextToken() // move to BANG
					effects = p.parseEffectAnnotation()
				}

				return &ast.FuncType{
					Params:  types,
					Return:  retType,
					Effects: effects,
					Pos:     startPos,
				}
			}

			// Just a tuple type
			return &ast.TupleType{
				Elements: types,
				Pos:      startPos,
			}
		}

		// Error: unexpected token after type
		p.report("PAR_TYPE_UNEXPECTED", "unexpected token in type expression", "Check type syntax")
		return nil

	default:
		return nil
	}
}

func (p *Parser) parsePattern() ast.Pattern {
	switch p.curToken.Type {
	case lexer.IDENT:
		// Could be a variable pattern or constructor
		name := p.curToken.Literal
		if p.peekTokenIs(lexer.LPAREN) {
			// Constructor with arguments
			p.nextToken()
			return p.parseConstructorPattern(name)
		}
		return &ast.Identifier{
			Name: name,
			Pos:  p.curPos(),
		}
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE:
		return &ast.Literal{
			Kind:  p.literalKind(),
			Value: p.literalValue(),
			Pos:   p.curPos(),
		}
	case lexer.LBRACKET:
		return p.parseListPattern()
	case lexer.LBRACE:
		return p.parseRecordPattern()
	case lexer.LPAREN:
		return p.parseTuplePattern()
	default:
		if p.curToken.Literal == "_" {
			return &ast.WildcardPattern{
				Pos: p.curPos(),
			}
		}
	}
	return nil
}

// Type declaration parsing
// EBNF:
//   type_decl      := export? "type" UIdent type_params? "=" type_body
//   type_params    := "[" type_param ("," type_param)* "]"
//   type_param     := LIdent
//   type_body      := type_alias | sum_type | record_type
//   sum_type       := variant ("|" variant)*
//   variant        := UIdent ("(" type_expr ("," type_expr)* ")")?
//   record_type    := "{" field ("," field)* ","? "}"
//   field          := LIdent ":" type_expr

func (p *Parser) parseTypeDeclaration(exported bool) ast.Node {
	startPos := p.curPos()

	// We're already at TYPE token
	if !p.curTokenIs(lexer.TYPE) {
		p.report("PAR_TYPE_EXPECTED", "expected 'type' keyword", "Add 'type' keyword")
		return nil
	}

	p.nextToken() // consume TYPE

	// Parse type name (must be uppercase identifier)
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_TYPE_NAME_EXPECTED", "expected type name", "Add a type name starting with uppercase letter")
		return nil
	}

	name := p.curToken.Literal
	p.nextToken()

	// Parse optional type parameters [a, b, ...]
	var typeParams []string
	if p.curTokenIs(lexer.LBRACKET) {
		typeParams = p.parseTypeParams()
	}

	// Expect '='
	if !p.curTokenIs(lexer.ASSIGN) {
		p.reportExpected(lexer.ASSIGN, "Add '=' after type name")
		return nil
	}
	p.nextToken() // consume ASSIGN

	// Parse type body (sum, product, or alias)
	definition := p.parseTypeDeclBody()
	if definition == nil {
		return nil
	}

	return &ast.TypeDecl{
		Name:       name,
		TypeParams: typeParams,
		Definition: definition,
		Exported:   exported,
		Pos:        startPos,
	}
}

// hasTopLevelPipe scans ahead to check if there's a pipe (|) at depth 0
// Used to disambiguate type aliases from sum types
// Returns true if finds unbalanced | at depth 0 before newline/EOF
// This is a simple check that peeks at the next few tokens
func (p *Parser) hasTopLevelPipe() bool {
	// Simple heuristic: check if peek token is PIPE
	// or if we're at an identifier and peek is PIPE
	return p.peekTokenIs(lexer.PIPE)
}

func (p *Parser) parseTypeDeclBody() ast.TypeDef {
	// Record type if we see '{'
	if p.curTokenIs(lexer.LBRACE) {
		return p.parseRecordTypeDef()
	}

	// Check if it's a list type alias: type Names = [string]
	if p.curTokenIs(lexer.LBRACKET) {
		typeExpr := p.parseType()
		return &ast.TypeAlias{
			Target: typeExpr,
			Pos:    p.curPos(),
		}
	}

	// For IDENT tokens, we need to disambiguate:
	// - type Color = Red | Green  → sum type
	// - type Names = [string]     → already handled above
	// - type UserId = int         → alias (single identifier)
	// - type Shape = Circle(int)  → could be alias or sum depending on |
	if p.curTokenIs(lexer.IDENT) {
		name := p.curToken.Literal
		var firstVariant *ast.Constructor

		// Check for constructor with fields: Circle(int, int)
		// Always treat as sum type (even single variant is valid)
		// Type aliases to parameterized types like Foo(int) are rare and not currently supported
		if p.peekTokenIs(lexer.LPAREN) {
			// Parse as sum type constructor
			p.nextToken() // advance to LPAREN
			// Parse constructor fields
			p.nextToken() // consume LPAREN
			var fields []ast.Type
			if !p.curTokenIs(lexer.RPAREN) {
				fields = append(fields, p.parseType())
				p.nextToken() // advance past the type we just parsed
				for p.curTokenIs(lexer.COMMA) {
					p.nextToken() // consume COMMA
					if p.curTokenIs(lexer.RPAREN) {
						break // trailing comma
					}
					fields = append(fields, p.parseType())
					p.nextToken() // advance past the type we just parsed
				}
			}
			if !p.curTokenIs(lexer.RPAREN) {
				p.reportExpected(lexer.RPAREN, "Add ')' to close constructor fields")
			} else {
				p.nextToken() // consume RPAREN
			}
			firstVariant = &ast.Constructor{
				Name:   name,
				Fields: fields,
				Pos:    p.curPos(),
			}
		} else {
			// No fields - check if this is a simple type alias or sum type
			// Use hasTopLevelPipe() to decide
			if !p.hasTopLevelPipe() {
				// No pipe → simple type alias like: type UserId = int
				typeExpr := p.parseType()
				return &ast.TypeAlias{
					Target: typeExpr,
					Pos:    p.curPos(),
				}
			}

			// Has pipe → sum type like: type Color = Red | Green | Blue
			// We're still at the identifier token
			firstVariant = &ast.Constructor{
				Name:   name,
				Fields: nil,
				Pos:    p.curPos(),
			}
		}

		// Check if there are more variants (PIPE)
		// If first variant had fields, we're at token after RPAREN (could be PIPE)
		// If first variant had no fields, we're still at variant name, need to peek
		hasMoreVariants := p.curTokenIs(lexer.PIPE) || p.peekTokenIs(lexer.PIPE)
		if hasMoreVariants {
			if !p.curTokenIs(lexer.PIPE) {
				p.nextToken() // advance to PIPE if we were peeking
			}
			variants := []*ast.Constructor{firstVariant}
			for p.curTokenIs(lexer.PIPE) {
				p.nextToken() // consume PIPE
				variant := p.parseVariant()
				if variant != nil {
					variants = append(variants, variant)
				}
				// After parsing variant, we're at the variant name
				// Need to check if there's another PIPE
				if p.peekTokenIs(lexer.PIPE) {
					p.nextToken() // advance to PIPE for next iteration
				}
			}
			return &ast.AlgebraicType{
				Constructors: variants,
				Pos:          p.curPos(),
			}
		}

		// Single constructor, still a sum type
		return &ast.AlgebraicType{
			Constructors: []*ast.Constructor{firstVariant},
			Pos:          p.curPos(),
		}
	}

	p.report("PAR_TYPE_BODY_EXPECTED", "expected type definition", "Add type definition (record, sum type, or alias)")
	return nil
}

func (p *Parser) parseVariant() *ast.Constructor {
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_VARIANT_NAME_EXPECTED", "expected variant name", "Add variant name starting with uppercase letter")
		return nil
	}

	name := p.curToken.Literal

	// Check if name starts with uppercase (convention for constructors)
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		p.report("PAR_VARIANT_NEEDS_UIDENT", "variant must start with uppercase letter", "Change to UpperCamelCase")
	}

	// Parse optional fields (peek ahead to see if there are any)
	var fields []ast.Type
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // advance to LPAREN
		p.nextToken() // consume LPAREN
		if !p.curTokenIs(lexer.RPAREN) {
			fields = append(fields, p.parseType())
			p.nextToken() // advance past the type we just parsed
			for p.curTokenIs(lexer.COMMA) {
				p.nextToken() // consume COMMA
				if p.curTokenIs(lexer.RPAREN) {
					break // trailing comma
				}
				fields = append(fields, p.parseType())
				p.nextToken() // advance past the type we just parsed
			}
		}
		if !p.curTokenIs(lexer.RPAREN) {
			p.reportExpected(lexer.RPAREN, "Add ')' to close variant fields")
		} else {
			p.nextToken() // consume RPAREN
		}
	}

	return &ast.Constructor{
		Name:   name,
		Fields: fields,
		Pos:    p.curPos(),
	}
}

func (p *Parser) parseRecordTypeDef() ast.TypeDef {
	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_TYPE_LBRACE_EXPECTED", "expected '{' for record type", "Add '{' to start record type")
		return nil
	}
	p.nextToken() // consume LBRACE

	var fields []*ast.RecordField
	if !p.curTokenIs(lexer.RBRACE) {
		// Parse first field
		field := p.parseRecordFieldDef()
		if field != nil {
			fields = append(fields, field)
		}
		p.nextToken() // advance past the field we just parsed

		// Parse remaining fields
		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACE) {
				break // trailing comma
			}
			field := p.parseRecordFieldDef()
			if field != nil {
				fields = append(fields, field)
			}
			p.nextToken() // advance past the field
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_TYPE_RBRACE_MISSING", "expected '}' to close record type", "Add '}' to close record type")
	}

	return &ast.RecordType{
		Fields: fields,
		Pos:    p.curPos(),
	}
}

// parseRecordTypeExpr parses a record type expression that can appear in type positions
// Example: { street: string, city: string }
// This is used for nested record types like: type User = { addr: { street: string } }
func (p *Parser) parseRecordTypeExpr() ast.Type {
	startPos := p.curPos()

	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_TYPE_LBRACE_EXPECTED", "expected '{' for record type", "Add '{' to start record type")
		return nil
	}
	p.nextToken() // consume LBRACE

	var fields []*ast.RecordField
	if !p.curTokenIs(lexer.RBRACE) {
		// Parse first field
		field := p.parseRecordFieldDef()
		if field != nil {
			fields = append(fields, field)
		}
		p.nextToken() // advance past the field we just parsed

		// Parse remaining fields
		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACE) {
				break // trailing comma
			}
			field := p.parseRecordFieldDef()
			if field != nil {
				fields = append(fields, field)
			}
			p.nextToken() // advance past the field
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_TYPE_RBRACE_MISSING", "expected '}' to close record type", "Add '}' to close record type")
	}

	return &ast.RecordType{
		Fields: fields,
		Pos:    startPos,
	}
}

func (p *Parser) parseRecordFieldDef() *ast.RecordField {
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_FIELD_NAME_EXPECTED", "expected field name", "Add field name")
		return nil
	}

	name := p.curToken.Literal
	p.nextToken()

	if !p.curTokenIs(lexer.COLON) {
		p.reportExpected(lexer.COLON, "Add ':' after field name")
		return nil
	}
	p.nextToken() // consume COLON

	fieldType := p.parseType()
	if fieldType == nil {
		p.report("PAR_FIELD_TYPE_EXPECTED", "expected field type", "Add field type")
		return nil
	}

	return &ast.RecordField{
		Name: name,
		Type: fieldType,
		Pos:  p.curPos(),
	}
}

func (p *Parser) parseTypeParams() []string {
	if !p.curTokenIs(lexer.LBRACKET) {
		return []string{}
	}
	p.nextToken() // consume LBRACKET

	var params []string
	if !p.curTokenIs(lexer.RBRACKET) {
		if p.curTokenIs(lexer.IDENT) {
			params = append(params, p.curToken.Literal)
			p.nextToken()
		}

		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACKET) {
				break // trailing comma
			}
			if p.curTokenIs(lexer.IDENT) {
				params = append(params, p.curToken.Literal)
				p.nextToken()
			}
		}
	}

	if !p.curTokenIs(lexer.RBRACKET) {
		p.reportExpected(lexer.RBRACKET, "Add ']' to close type parameters")
	} else {
		p.nextToken() // consume RBRACKET
	}

	return params
}

func (p *Parser) parseClassDeclaration() ast.Node {
	// TODO: Implement class declaration parsing
	return nil
}

func (p *Parser) parseInstanceDeclaration() ast.Node {
	// TODO: Implement instance declaration parsing
	return nil
}

// parseEffectAnnotation parses effect annotations: ! {IO, FS, Net}
// Validates effect names and detects duplicates
func (p *Parser) parseEffectAnnotation() []string {
	// Known canonical effect names
	knownEffects := map[string]bool{
		"IO":    true,
		"FS":    true,
		"Net":   true,
		"Clock": true,
		"Rand":  true,
		"DB":    true,
		"Trace": true,
		"Async": true,
	}

	// We're at the BANG token
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	effects := []string{}
	seen := make(map[string]bool)

	// Parse comma-separated effect names
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()

		if !p.curTokenIs(lexer.IDENT) {
			p.report("PAR_EFF004_INVALID",
				"effect name must be an identifier",
				"Use one of: IO, FS, Net, Clock, Rand, DB, Trace, Async")
			continue
		}

		effectName := p.curToken.Literal

		// Check for unknown effects
		if !knownEffects[effectName] {
			// Try to suggest closest match
			suggestion := p.suggestEffect(effectName, knownEffects)
			fix := fmt.Sprintf("Did you mean '%s'?", suggestion)
			p.report("PAR_EFF002_UNKNOWN",
				fmt.Sprintf("unknown effect '%s'", effectName),
				fix)
			// Continue parsing to find more errors
		}

		// Check for duplicates
		if seen[effectName] {
			p.report("PAR_EFF001_DUP",
				fmt.Sprintf("duplicate effect '%s' in annotation", effectName),
				fmt.Sprintf("Remove duplicate '%s'", effectName))
		} else {
			seen[effectName] = true
			effects = append(effects, effectName)
		}

		// Check for comma or closing brace
		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			p.reportExpected(lexer.COMMA, "Add ',' between effect names")
			break
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		p.reportExpected(lexer.RBRACE, "Add '}' to close effect annotation")
	}

	return effects
}

// suggestEffect finds closest matching effect name (simple heuristic)
func (p *Parser) suggestEffect(name string, known map[string]bool) string {
	name = strings.ToLower(name)

	// Check exact match ignoring case
	for k := range known {
		if strings.ToLower(k) == name {
			return k
		}
	}

	// Check prefix match
	for k := range known {
		if strings.HasPrefix(strings.ToLower(k), name) {
			return k
		}
	}

	// Default to IO as most common
	return "IO"
}

// parseEffects is deprecated - use parseEffectAnnotation instead
// Kept for backward compatibility during migration
// parseTestsBlock parses a tests block with the new multi-input format

// parsePropertiesBlock parses a properties block

func (p *Parser) parseConstructorPattern(name string) ast.Pattern {
	constructor := &ast.ConstructorPattern{
		Name:     name,
		Pos:      p.curPos(),
		Patterns: []ast.Pattern{},
	}

	// We're at LPAREN
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume RPAREN
		return constructor
	}

	p.nextToken() // move to first argument
	constructor.Patterns = append(constructor.Patterns, p.parsePattern())

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next argument
		constructor.Patterns = append(constructor.Patterns, p.parsePattern())
	}

	p.expectPeek(lexer.RPAREN)
	return constructor
}

func (p *Parser) parseListPattern() ast.Pattern {
	startPos := p.curPos()
	// We're at LBRACKET
	p.nextToken() // consume LBRACKET

	// Empty list pattern: []
	if p.curTokenIs(lexer.RBRACKET) {
		// Parser convention: leave at last token of pattern (RBRACKET)
		return &ast.ListPattern{
			Elements: []ast.Pattern{},
			Rest:     nil,
			Pos:      startPos,
		}
	}

	// Non-empty list: [x, ...] or [x, y, ...rest]
	elements := []ast.Pattern{}
	var rest ast.Pattern

	for {
		// Check for spread pattern: ...rest
		if p.curTokenIs(lexer.ELLIPSIS) {
			p.nextToken() // consume ELLIPSIS
			if !p.curTokenIs(lexer.IDENT) {
				p.report("PAT_SPREAD_NEEDS_IDENT", "spread in list pattern must bind to a name, e.g. [x, ...xs]", "Add an identifier after ..., like [x, ...rest]")
				return nil
			}
			rest = &ast.Identifier{
				Name: p.curToken.Literal,
				Pos:  p.curPos(),
			}
			p.nextToken() // consume ident
			break         // spread must be last
		}

		// Parse next pattern element
		elem := p.parsePattern()
		if elem == nil {
			return nil
		}
		elements = append(elements, elem)

		// Check what comes next
		p.nextToken() // move past pattern element

		if p.curTokenIs(lexer.RBRACKET) {
			// End of list
			break
		}

		if !p.curTokenIs(lexer.COMMA) {
			p.reportExpected(lexer.COMMA, "Expected ',' or ']' in list pattern")
			return nil
		}

		p.nextToken() // consume comma

		// Check for closing bracket after comma (trailing comma)
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}
	}

	// We should be at RBRACKET now
	if !p.curTokenIs(lexer.RBRACKET) {
		p.reportExpected(lexer.RBRACKET, "Expected ']' to close list pattern")
		return nil
	}
	// Pattern parsing convention: leave current token at the last token of the pattern
	// The caller will advance past it

	return &ast.ListPattern{
		Elements: elements,
		Rest:     rest,
		Pos:      startPos,
	}
}

func (p *Parser) parseRecordPattern() ast.Pattern {
	// TODO: Implement record pattern parsing
	return nil
}

func (p *Parser) parseTuplePattern() ast.Pattern {
	startPos := p.curPos()
	// We're at LPAREN
	p.nextToken() // consume LPAREN

	// Empty tuple: ()
	if p.curTokenIs(lexer.RPAREN) {
		// Empty tuple pattern (same as Unit pattern)
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Parse first element
	first := p.parsePattern()

	// Single element in parens: (x) - not a tuple, just a grouped pattern
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume RPAREN
		return first
	}

	// Must be a comma for tuple
	if !p.peekTokenIs(lexer.COMMA) {
		p.reportExpected(lexer.COMMA, "Expected ',' for tuple pattern or ')' for grouped pattern")
		return nil
	}

	// Parse remaining elements
	elements := []ast.Pattern{first}
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if p.peekTokenIs(lexer.RPAREN) {
			// Trailing comma
			break
		}
		p.nextToken() // move to next element
		elements = append(elements, p.parsePattern())
	}

	p.expectPeek(lexer.RPAREN)

	return &ast.TuplePattern{
		Elements: elements,
		Pos:      startPos,
	}
}

func (p *Parser) literalKind() ast.LiteralKind {
	switch p.curToken.Type {
	case lexer.INT:
		return ast.IntLit
	case lexer.FLOAT:
		return ast.FloatLit
	case lexer.STRING:
		return ast.StringLit
	case lexer.TRUE, lexer.FALSE:
		return ast.BoolLit
	default:
		return ast.StringLit
	}
}

func (p *Parser) literalValue() interface{} {
	switch p.curToken.Type {
	case lexer.INT:
		v, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
		return v
	case lexer.FLOAT:
		v, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		return v
	case lexer.STRING:
		return p.curToken.Literal
	case lexer.TRUE:
		return true
	case lexer.FALSE:
		return false
	default:
		return p.curToken.Literal
	}
}

// Utility functions

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	err := NewParserError(
		"PAR_UNEXPECTED_TOKEN",
		ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File},
		p.peekToken,
		msg,
		[]lexer.TokenType{t},
		fmt.Sprintf("Add or correct the %s token", t),
	)
	p.errors = append(p.errors, err)
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := fmt.Sprintf("unexpected token in expression: %s", t)
	fix := "This token cannot start an expression"
	if t == lexer.RBRACE || t == lexer.RPAREN || t == lexer.RBRACKET {
		fix = "Check for unmatched delimiters or missing expression"
	}
	err := NewParserError(
		"PAR_NO_PREFIX_PARSE",
		p.curPos(),
		p.curToken,
		msg,
		nil,
		fix,
	)
	p.errors = append(p.errors, err)
}

func (p *Parser) curPos() ast.Pos {
	return ast.Pos{
		Line:   p.curToken.Line,
		Column: p.curToken.Column,
		File:   p.curToken.File,
	}
}

func (p *Parser) peekPrecedence() int {
	return p.peekToken.Precedence()
}

func (p *Parser) curPrecedence() int {
	return p.curToken.Precedence()
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
