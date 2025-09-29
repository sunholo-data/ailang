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
		// Error: export must be followed by func or pure
		err := NewParserError(
			"PAR_EXPORT_REQUIRES_FUNC",
			p.curPos(),
			p.curToken,
			fmt.Sprintf("export must be followed by 'func', got '%s'", p.curToken.Literal),
			[]lexer.TokenType{lexer.FUNC, lexer.PURE},
			"Use 'export func name(...) { ... }'",
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
		return p.parseTypeDeclaration()
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
	}

	// Parse parameters
	if p.peekTokenIs(lexer.UNIT) {
		// Empty parameter list
		p.nextToken()
		fn.Params = []*ast.Param{}
	} else {
		p.expectPeek(lexer.LPAREN)
		fn.Params = p.parseParams()
	}

	// Parse return type if present
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken()
		p.nextToken()
		fn.ReturnType = p.parseType()

		// Parse effects if present
		if p.peekTokenIs(lexer.BANG) {
			p.nextToken()
			if p.peekTokenIs(lexer.LBRACE) {
				p.nextToken()
				fn.Effects = p.parseEffects()
			}
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

	// Parse body
	// Check if we're already at LBRACE (after skipping newlines) or need to advance
	if !p.curTokenIs(lexer.LBRACE) {
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
	}
	p.nextToken() // move past LBRACE
	fn.Body = p.parseExpression(LOWEST)
	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	endPos := p.curPos()
	fn.Span = ast.Span{Start: startPos, End: endPos}
	return fn
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

func (p *Parser) parseGroupedExpression() ast.Expr {
	p.nextToken()

	// Check for tuple
	expr := p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.COMMA) {
		// It's a tuple
		tuple := &ast.Tuple{
			Elements: []ast.Expr{expr},
			Pos:      p.curPos(),
		}

		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			tuple.Elements = append(tuple.Elements, p.parseExpression(LOWEST))
		}

		p.expectPeek(lexer.RPAREN)
		return tuple
	}

	p.expectPeek(lexer.RPAREN)
	return expr
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
	record := &ast.Record{
		Pos: p.curPos(),
	}

	p.nextToken()

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
}

func (p *Parser) parsePrefixExpression() ast.Expr {
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
	lambda := &ast.Lambda{
		Pos: p.curPos(),
	}

	p.expectPeek(lexer.LPAREN)
	lambda.Params = p.parseParams()

	// Parse return type and effects if present
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken()
		p.nextToken()
		// Parse return type
		// Parse effects if present
	}

	p.expectPeek(lexer.FARROW)
	p.nextToken()
	lambda.Body = p.parseExpression(LOWEST)

	return lambda
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

func (p *Parser) parseType() ast.Type {
	// Simple type parsing for now
	if p.curTokenIs(lexer.IDENT) {
		return &ast.SimpleType{
			Name: p.curToken.Literal,
			Pos:  p.curPos(),
		}
	}

	if p.curTokenIs(lexer.UNIT) {
		// Unit type ()
		return &ast.SimpleType{
			Name: "()",
			Pos:  p.curPos(),
		}
	}

	if p.curTokenIs(lexer.LPAREN) && p.peekTokenIs(lexer.RPAREN) {
		// Also handle () as unit type
		p.nextToken() // consume RPAREN
		return &ast.SimpleType{
			Name: "()",
			Pos:  p.curPos(),
		}
	}

	if p.curTokenIs(lexer.LBRACKET) {
		p.nextToken()
		elemType := p.parseType()
		p.expectPeek(lexer.RBRACKET)
		return &ast.ListType{
			Element: elemType,
			Pos:     p.curPos(),
		}
	}

	// Add more type parsing as needed
	return nil
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

// Stub implementations for complex parsing

func (p *Parser) parseTypeDeclaration() ast.Node {
	// TODO: Implement type declaration parsing
	return nil
}

func (p *Parser) parseClassDeclaration() ast.Node {
	// TODO: Implement class declaration parsing
	return nil
}

func (p *Parser) parseInstanceDeclaration() ast.Node {
	// TODO: Implement instance declaration parsing
	return nil
}

func (p *Parser) parseTypeParams() []string {
	// TODO: Implement type parameter parsing
	return []string{}
}

func (p *Parser) parseEffects() []string {
	effects := []string{}

	// We're already at the LBRACE token
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		if p.curTokenIs(lexer.IDENT) {
			effects = append(effects, p.curToken.Literal)
		}

		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	p.expectPeek(lexer.RBRACE)
	return effects
}

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
	// TODO: Implement list pattern parsing
	return nil
}

func (p *Parser) parseRecordPattern() ast.Pattern {
	// TODO: Implement record pattern parsing
	return nil
}

func (p *Parser) parseTuplePattern() ast.Pattern {
	// TODO: Implement tuple pattern parsing
	return nil
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
	msg := fmt.Sprintf("expected next token to be %s, got %s instead at %s",
		t, p.peekToken.Type, p.peekToken.Position())
	p.errors = append(p.errors, fmt.Errorf(msg))
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, fmt.Errorf(msg))
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
