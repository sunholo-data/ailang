package parser

import (
	"fmt"
	"strconv"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

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
func (p *Parser) ParseFile() *ast.File {
	file := &ast.File{
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

// parseModule parses a module declaration (legacy)
func (p *Parser) parseModule() *ast.Module {
	module := &ast.Module{
		Pos: p.curPos(),
	}

	p.expectPeek(lexer.IDENT)
	module.Name = p.curToken.Literal

	p.nextToken()

	// Parse imports
	for p.curTokenIs(lexer.IMPORT) {
		imp := p.parseImport()
		module.Imports = append(module.Imports, imp)
		p.nextToken()
	}

	// Parse exports (if explicit)
	if p.curTokenIs(lexer.EXPORT) {
		p.nextToken()
		// Parse export list
	}

	// Parse declarations
	for !p.curTokenIs(lexer.EOF) {
		if decl := p.parseDeclaration(); decl != nil {
			module.Decls = append(module.Decls, decl)
		}
		p.nextToken()
	}

	return module
}

// parseImport parses an import statement
func (p *Parser) parseImport() *ast.Import {
	imp := &ast.Import{
		Pos: p.curPos(),
	}

	p.nextToken()

	// Parse import path - can be string or identifier path like std/io
	if p.curTokenIs(lexer.STRING) {
		imp.Path = p.curToken.Literal
	} else if p.curTokenIs(lexer.IDENT) {
		// Build path from identifiers and slashes
		path := p.curToken.Literal
		for p.peekTokenIs(lexer.SLASH) {
			p.nextToken() // consume slash
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			path += "/" + p.curToken.Literal
		}
		imp.Path = path
	} else {
		p.errors = append(p.errors, fmt.Errorf("expected import path, got %s", p.curToken.Type))
		return nil
	}

	// Check for specific imports
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		p.nextToken()
		for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.IDENT) {
				imp.Symbols = append(imp.Symbols, p.curToken.Literal)
			}
			if p.peekTokenIs(lexer.RPAREN) {
				p.nextToken()
				break
			}
			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken()
		}

		if !p.curTokenIs(lexer.RPAREN) {
			p.errors = append(p.errors, fmt.Errorf("expected ), got %s", p.curToken.Type))
			return nil
		}
	}

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

// parseImportDecl parses an import declaration
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	startPos := p.curPos()
	imp := &ast.ImportDecl{
		Pos: startPos,
	}

	p.nextToken() // consume 'import'

	// Parse import path - can be string or identifier path like std/io
	if p.curTokenIs(lexer.STRING) {
		imp.Path = p.curToken.Literal
	} else if p.curTokenIs(lexer.IDENT) {
		// Build path from identifiers and slashes
		path := p.curToken.Literal
		for p.peekTokenIs(lexer.SLASH) {
			p.nextToken() // consume slash
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			path += "/" + p.curToken.Literal
		}
		imp.Path = path
	} else {
		p.peekError(lexer.IDENT)
		return nil
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
		return nil
	case lexer.PURE:
		// Check if it's a pure function declaration
		if p.peekTokenIs(lexer.FUNC) {
			p.nextToken() // consume 'pure'
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
func (p *Parser) parseDeclaration() ast.Node {
	return p.parseTopLevelDecl()
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

	// Parse tests if present (before body)
	if p.peekTokenIs(lexer.TESTS) {
		p.nextToken() // consume 'tests'
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // move to LBRACKET
			fn.Tests = p.parseTestsBlock()
		}
	}

	// Parse properties if present (before body)
	if p.peekTokenIs(lexer.PROPERTIES) {
		p.nextToken() // consume 'properties'
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // move to LBRACKET
			fn.Properties = p.parsePropertiesBlock()
		}
	}

	// Parse body
	if !p.expectPeek(lexer.LBRACE) {
		return nil
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

	p.expectPeek(lexer.IDENT)
	let.Name = p.curToken.Literal

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		let.Type = p.parseType()
	}

	p.expectPeek(lexer.ASSIGN)
	p.nextToken()
	let.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		let.Body = p.parseExpression(LOWEST)
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

func (p *Parser) parseTestBlock() ast.Node {
	// TODO: Implement test block parsing
	return nil
}

func (p *Parser) parsePropertyBlock() ast.Node {
	// TODO: Implement property block parsing
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
func (p *Parser) parseTestsBlock() []*ast.TestCase {
	var tests []*ast.TestCase

	// We should be at LBRACKET
	if !p.curTokenIs(lexer.LBRACKET) {
		p.errors = append(p.errors, fmt.Errorf("expected [ for tests block at %s:%d:%d, got %s",
			p.curToken.File, p.curPos().Line, p.curPos().Column, p.curToken.Type))
		return tests
	}

	// Handle empty tests block
	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume RBRACKET
		return tests
	}

	p.nextToken() // Move to first test

	// Parse test cases: (input1, input2, ..., expected) tuples
	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		// Expect LPAREN for test case tuple
		if !p.curTokenIs(lexer.LPAREN) {
			p.errors = append(p.errors, fmt.Errorf("expected ( for test case"))
			// Skip to next test
			for !p.curTokenIs(lexer.COMMA) && !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
				p.nextToken()
			}
			if p.curTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			continue
		}

		p.nextToken() // Move past LPAREN

		// Parse all expressions in the tuple
		var exprs []ast.Expr
		for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
			expr := p.parseExpression(LOWEST)
			if expr == nil {
				return tests
			}
			exprs = append(exprs, expr)

			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next expression
			} else {
				break
			}
		}

		if !p.expectPeek(lexer.RPAREN) {
			return tests
		}

		// Create test case with inputs and expected
		if len(exprs) >= 2 {
			test := &ast.TestCase{
				Inputs:   exprs[:len(exprs)-1],
				Expected: exprs[len(exprs)-1],
				Pos:      p.curPos(),
			}
			tests = append(tests, test)
		}

		// Check for comma before next test
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if !p.peekTokenIs(lexer.RBRACKET) {
			break
		}
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return tests
	}

	return tests
}

// parsePropertiesBlock parses a properties block
func (p *Parser) parsePropertiesBlock() []*ast.Property {
	var props []*ast.Property

	// We should be at LBRACKET
	if !p.curTokenIs(lexer.LBRACKET) {
		p.errors = append(p.errors, fmt.Errorf("expected [ for properties block"))
		return props
	}

	// Handle empty properties block
	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume RBRACKET
		return props
	}

	p.nextToken() // Move to first property

	// Parse properties
	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		prop := &ast.Property{
			Pos: p.curPos(),
		}

		// Parse property name (optional)
		if p.curTokenIs(lexer.STRING) {
			prop.Name = p.curToken.Literal
			p.nextToken()
		}

		// Parse forall bindings
		if p.curTokenIs(lexer.FORALL) {
			p.nextToken() // consume forall
			if p.expectPeek(lexer.LPAREN) {
				p.nextToken() // move past (
				
				// Parse binders
				for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
					if p.curTokenIs(lexer.IDENT) {
						binder := &ast.Binder{
							Name: p.curToken.Literal,
							Pos:  p.curPos(),
						}
						
						// Parse type annotation
						if p.peekTokenIs(lexer.COLON) {
							p.nextToken() // consume :
							p.nextToken() // move to type
							binder.Type = p.parseType()
						}
						
						prop.Binders = append(prop.Binders, binder)
						
						if p.peekTokenIs(lexer.COMMA) {
							p.nextToken() // consume comma
							p.nextToken() // move to next binder
						}
					} else {
						break
					}
				}
				
				if !p.expectPeek(lexer.RPAREN) {
					return props
				}
			}
			
			// Expect =>
			if !p.expectPeek(lexer.FARROW) {
				return props
			}
			p.nextToken() // move past =>
		}

		// Parse property expression
		prop.Expr = p.parseExpression(LOWEST)
		if prop.Expr != nil {
			props = append(props, prop)
		}

		// Check for comma before next property
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if !p.peekTokenIs(lexer.RBRACKET) {
			break
		}
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return props
	}

	return props
}

func (p *Parser) parseTests() []*ast.TestCase {
	var tests []*ast.TestCase

	// We should be at LBRACKET
	if !p.curTokenIs(lexer.LBRACKET) {
		p.errors = append(p.errors, fmt.Errorf("expected [ for tests block at %s:%d:%d, got %s",
			p.curToken.File, p.curPos().Line, p.curPos().Column, p.curToken.Type))
		return tests
	}

	// Handle empty tests block
	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume RBRACKET
		return tests
	}

	p.nextToken() // Move to first test

	// Parse test cases: (input, output) pairs
	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		// Expect LPAREN for test case tuple
		if !p.curTokenIs(lexer.LPAREN) {
			p.errors = append(p.errors, fmt.Errorf("expected ( for test case at %s:%d:%d, got %s",
				p.curToken.File, p.curPos().Line, p.curPos().Column, p.curToken.Type))
			// Try to recover by skipping to next comma or closing bracket
			for !p.curTokenIs(lexer.COMMA) && !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
				p.nextToken()
			}
			if p.curTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			continue
		}

		p.nextToken() // Move past LPAREN

		// Parse input expression
		input := p.parseExpression(LOWEST)
		if input == nil {
			return tests
		}

		// Expect comma
		if !p.expectPeek(lexer.COMMA) {
			return tests
		}
		p.nextToken() // Move past comma

		// Parse expected output
		output := p.parseExpression(LOWEST)
		if output == nil {
			return tests
		}

		// Expect closing paren
		if !p.expectPeek(lexer.RPAREN) {
			return tests
		}

		tests = append(tests, &ast.TestCase{
			Inputs:   []ast.Expr{input},
			Expected: output,
			Pos:      p.curPos(),
		})

		// Check for comma or end of tests
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next test
		} else if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken() // consume closing bracket
			break
		} else {
			p.errors = append(p.errors, fmt.Errorf("expected , or ] after test case at %s:%d:%d, got %s",
				p.peekToken.File, p.peekPos().Line, p.peekPos().Column, p.peekToken.Type))
			return tests
		}
	}

	return tests
}

func (p *Parser) parseProperties() []*ast.Property {
	var properties []*ast.Property

	// We're at LBRACKET
	if !p.curTokenIs(lexer.LBRACKET) {
		return properties
	}

	// Handle empty properties block
	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken()
		return properties
	}

	p.nextToken() // Move to first property

	// Parse properties
	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		var prop ast.Property
		prop.Pos = p.curPos()

		// Check if it's a named property (string literal followed by colon)
		if p.curTokenIs(lexer.STRING) {
			prop.Name = p.curToken.Literal
			if p.peekTokenIs(lexer.COLON) {
				p.nextToken() // consume string
				p.nextToken() // consume colon
			} else {
				// String without colon is an unnamed property expression
				prop.Name = ""
			}
		} else {
			// Unnamed property
			prop.Name = ""
		}

		// Parse property expression (typically starts with forall)
		prop.Expr = p.parseExpression(LOWEST)
		if prop.Expr == nil {
			return properties
		}

		properties = append(properties, &prop)

		// Check for comma or end of properties
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next property
		} else if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken() // consume closing bracket
			break
		} else {
			p.errors = append(p.errors, fmt.Errorf("expected , or ] at %s:%d:%d, got %s",
				p.peekToken.File, p.peekPos().Line, p.peekPos().Column, p.peekToken.Type))
			return properties
		}
	}

	return properties
}

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

func (p *Parser) peekPos() ast.Pos {
	return ast.Pos{
		Line:   p.peekToken.Line,
		Column: p.peekToken.Column,
		File:   p.peekToken.File,
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
