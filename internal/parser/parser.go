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

// Precedence levels
const (
	LOWEST int = iota
	LOGICAL_OR    // ||
	LOGICAL_AND   // &&
	EQUALS        // ==, !=
	LESSGREATER   // >, <, >=, <=
	APPEND        // ++
	CONS          // ::
	SUM           // +, -
	PRODUCT       // *, /, %
	PREFIX        // -x, !x
	CALL          // f(x)
	INDEX         // a[i], r.field
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
	
	// Check if we have a module declaration
	if p.curTokenIs(lexer.MODULE) {
		module := p.parseModule()
		program.Module = module
	} else {
		// Parse as a simple expression or declaration list
		module := &ast.Module{
			Name: "Main",
			Pos:  p.curPos(),
		}
		
		for !p.curTokenIs(lexer.EOF) {
			if decl := p.parseDeclaration(); decl != nil {
				module.Decls = append(module.Decls, decl)
			}
			p.nextToken()
		}
		
		program.Module = module
	}

	return program
}

// parseModule parses a module declaration
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

	p.expectPeek(lexer.STRING)
	imp.Path = p.curToken.Literal

	// Check for specific imports
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		p.nextToken()
		for !p.curTokenIs(lexer.RPAREN) {
			if p.curTokenIs(lexer.IDENT) {
				imp.Symbols = append(imp.Symbols, p.curToken.Literal)
			}
			if !p.peekTokenIs(lexer.RPAREN) {
				p.expectPeek(lexer.COMMA)
				p.nextToken()
			}
		}
		p.expectPeek(lexer.RPAREN)
	}

	return imp
}

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() ast.Node {
	switch p.curToken.Type {
	case lexer.FUNC:
		return p.parseFunctionDeclaration(false)
	case lexer.PURE:
		if p.peekTokenIs(lexer.FUNC) {
			p.nextToken()
			return p.parseFunctionDeclaration(true)
		}
	case lexer.TYPE:
		return p.parseTypeDeclaration()
	case lexer.CLASS:
		return p.parseClassDeclaration()
	case lexer.INSTANCE:
		return p.parseInstanceDeclaration()
	case lexer.TEST:
		return p.parseTestBlock()
	case lexer.PROPERTY:
		return p.parsePropertyBlock()
	default:
		// Try to parse as an expression
		return p.parseExpression(LOWEST)
	}
	return nil
}

// parseFunctionDeclaration parses a function declaration
func (p *Parser) parseFunctionDeclaration(isPure bool) *ast.FuncDecl {
	fn := &ast.FuncDecl{
		IsPure: isPure,
		Pos:    p.curPos(),
	}

	p.expectPeek(lexer.IDENT)
	fn.Name = p.curToken.Literal

	// Parse type parameters if present
	if p.peekTokenIs(lexer.LBRACKET) {
		p.nextToken()
		fn.TypeParams = p.parseTypeParams()
	}

	// Parse parameters
	p.expectPeek(lexer.LPAREN)
	fn.Params = p.parseParams()

	// Parse return type if present
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken()
		p.nextToken()
		fn.ReturnType = p.parseType()
		
		// Parse effects if present
		if p.curTokenIs(lexer.BANG) {
			p.nextToken()
			p.expectPeek(lexer.LBRACE)
			fn.Effects = p.parseEffects()
		}
	}

	// Parse tests if present
	if p.peekTokenIs(lexer.TEST) {
		p.nextToken()
		p.expectPeek(lexer.LBRACKET)
		fn.Tests = p.parseTests()
	}

	// Parse properties if present
	if p.peekTokenIs(lexer.PROPERTY) {
		p.nextToken()
		p.expectPeek(lexer.LBRACKET)
		fn.Properties = p.parseProperties()
	}

	// Parse body
	p.expectPeek(lexer.LBRACE)
	p.nextToken()
	fn.Body = p.parseExpression(LOWEST)
	p.expectPeek(lexer.RBRACE)

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
	
	for !p.curTokenIs(lexer.RBRACKET) {
		list.Elements = append(list.Elements, p.parseExpression(LOWEST))
		
		if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken()
			break
		}
		
		p.expectPeek(lexer.COMMA)
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
	
	for !p.curTokenIs(lexer.RBRACE) {
		field := &ast.Field{
			Pos: p.curPos(),
		}
		
		if !p.curTokenIs(lexer.IDENT) {
			p.errors = append(p.errors, fmt.Errorf("expected field name, got %s", p.curToken.Type))
			return nil
		}
		
		field.Name = p.curToken.Literal
		
		p.expectPeek(lexer.COLON)
		p.nextToken()
		
		field.Value = p.parseExpression(LOWEST)
		record.Fields = append(record.Fields, field)
		
		if !p.peekTokenIs(lexer.RBRACE) {
			p.expectPeek(lexer.COMMA)
			p.nextToken()
		}
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
		
		if !p.curTokenIs(lexer.RBRACE) {
			if p.curTokenIs(lexer.COMMA) || p.curTokenIs(lexer.PIPE) {
				p.nextToken()
			}
		}
	}

	p.expectPeek(lexer.RBRACE)
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
	p.expectPeek(lexer.FUNC)
	lambda := p.parseLambda().(*ast.Lambda)
	// Mark as pure somehow
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
	// TODO: Implement effect parsing
	return []string{}
}

func (p *Parser) parseTests() []*ast.TestCase {
	// TODO: Implement test case parsing
	return []*ast.TestCase{}
}

func (p *Parser) parseProperties() []*ast.Property {
	// TODO: Implement property parsing
	return []*ast.Property{}
}

func (p *Parser) parseConstructorPattern(name string) ast.Pattern {
	// TODO: Implement constructor pattern parsing
	return nil
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