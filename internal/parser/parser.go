package parser

import (
	"fmt"

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
	p.registerPrefix(lexer.LETREC, p.parseLetRecExpression)
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

func (p *Parser) curPos() ast.Pos {
	return ast.Pos{
		Line:   p.curToken.Line,
		Column: p.curToken.Column,
		File:   p.curToken.File,
	}
}

// peekNonNewline returns the next non-newline, non-comment token type
// without advancing the parser
func (p *Parser) peekNonNewline() lexer.TokenType {
	// If peek is not a newline/comment, return it directly
	if p.peekToken.Type != lexer.NEWLINE && p.peekToken.Type != lexer.COMMENT {
		return p.peekToken.Type
	}

	// Otherwise, we need to look ahead more (this is expensive, use sparingly)
	// For now, just return peek - we'll handle this in hasTopLevelPipe
	return p.peekToken.Type
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
