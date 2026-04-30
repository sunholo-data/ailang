package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// Parser parses AILANG source code into an AST
type Parser struct {
	l          *lexer.Lexer
	curToken   lexer.Token
	peekToken  lexer.Token
	peek2Token lexer.Token // 2nd lookahead — used by parseLabelOrRefinementSuffix
	errors     []error

	// Pratt parsing
	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn

	// Surface sugar control (S-CALL0, S-CONS, S-ARROWTYPE)
	strictSyntaxMode bool // When true, syntactic sugar is not allowed
	sugarUsed        bool // Tracks if sugar was used in this parse (for REPL feedback)

	// Loop detection (M-PARSER-LOOP): track last position to detect infinite loops
	lastExprPos ast.Pos

	// M-GAP4: Fresh row variable counter for sugar syntax {a: T, ..}
	rowVarCounter int
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

// Precedence levels — C-standard dedicated bands.
// These MUST match the values returned by token.Precedence().
const (
	LOWEST      = 0
	LAMBDA      = 1  // \x. (lowest precedence)
	LogicalOr   = 2  // ||
	LogicalAnd  = 3  // &&
	BitwiseXor  = 5  // ^   (4 reserved for future bitwise OR |)
	BitwiseAnd  = 6  // &
	EQUALS      = 7  // ==, !=
	LESSGREATER = 8  // >, <, >=, <=
	SHIFT       = 9  // <<, >>
	CONS        = 10 // :: (list cons - right associative)
	APPEND      = 11 // ++
	SUM         = 12 // +, -
	PRODUCT     = 13 // *, /, %
	PREFIX      = 14 // -x, !x, ~x (unary)
	CALL        = 15 // f(x) (application)
	DotAccess   = 16 // r.field (field access - highest)
	HIGHEST     = 17
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
	p.registerPrefix(lexer.STRING_PART, p.parseInterpolatedString)
	p.registerPrefix(lexer.CHAR, p.parseCharLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.UNIT, p.parseUnitLiteral)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACKET, p.parseListLiteral)
	p.registerPrefix(lexer.HASH, p.parseArrayLiteral)
	p.registerPrefix(lexer.LBRACE, p.parseRecordLiteral)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.TILDE, p.parsePrefixExpression)
	p.registerPrefix(lexer.IF, p.parseIfExpression)
	p.registerPrefix(lexer.LET, p.parseLetExpression)
	p.registerPrefix(lexer.LETREC, p.parseLetRecExpression)
	p.registerPrefix(lexer.MATCH, p.parseMatchExpression)
	p.registerPrefix(lexer.FUNC, p.parseLambda)
	p.registerPrefix(lexer.PURE, p.parsePureLambda)
	p.registerPrefix(lexer.BACKSLASH, p.parseBackslashLambda)
	p.registerPrefix(lexer.FORALL, p.parseForallExpression)

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
	p.registerInfix(lexer.AMPERSAND, p.parseInfixExpression) // bitwise AND
	p.registerInfix(lexer.CARET, p.parseInfixExpression)     // bitwise XOR
	p.registerInfix(lexer.SHL, p.parseInfixExpression)       // left shift
	p.registerInfix(lexer.SHR, p.parseInfixExpression)       // right shift
	p.registerInfix(lexer.DCOLON, p.parseConsExpression)     // S-CONS: :: sugar
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.UNIT, p.parseZeroArgCall) // S-CALL0: f() sugar (expression context)
	p.registerInfix(lexer.DOT, p.parseRecordAccess)
	p.registerInfix(lexer.LARROW, p.parseSendExpression)

	// Read three tokens to prime curToken, peekToken, and peek2Token
	p.nextToken() // peek2Token ← first token from lexer
	p.nextToken() // peekToken  ← peek2Token (first), peek2Token ← second
	p.nextToken() // curToken   ← peekToken (first), peekToken ← second, peek2Token ← third

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
				"Please report this as a bug at https://github.com/sunholo-data/ailang/issues"))
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

// Surface sugar control methods

// SetStrictSyntaxMode enables or disables strict syntax mode.
// When strict mode is enabled, all syntactic sugar is rejected with helpful errors.
func (p *Parser) SetStrictSyntaxMode(strict bool) {
	p.strictSyntaxMode = strict
}

// SugarUsed returns true if any syntactic sugar was used during parsing.
// This is used by the REPL to show "(desugared)" feedback to users.
func (p *Parser) SugarUsed() bool {
	return p.sugarUsed
}

// reportSugarError reports an error when sugar is used in strict syntax mode.
// Provides helpful error messages with canonical equivalents.
func (p *Parser) reportSugarError(sugarName, example, canonical string) {
	p.report(
		fmt.Sprintf("SUGAR_%s", sugarName),
		fmt.Sprintf("%s sugar not allowed in strict mode", sugarName),
		fmt.Sprintf("Use `%s` (canonical syntax) instead of `%s`", canonical, example),
	)
}

// Utility functions

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.peek2Token
	p.peek2Token = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) peek2TokenIs(t lexer.TokenType) bool {
	return p.peek2Token.Type == t
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

// freshRowVarName generates a fresh row variable name for sugar syntax {a: T, ..}
// Names are like _r0, _r1, etc. - underscore prefix indicates compiler-generated
func (p *Parser) freshRowVarName() string {
	name := fmt.Sprintf("_r%d", p.rowVarCounter)
	p.rowVarCounter++
	return name
}
