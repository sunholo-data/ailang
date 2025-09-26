package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	IDENT  // identifier
	INT    // 123
	FLOAT  // 123.45
	STRING // "abc"
	CHAR   // 'a'

	// Keywords
	FUNC
	PURE
	LET
	IN
	IF
	THEN
	ELSE
	MATCH
	WITH
	TYPE
	CLASS
	INSTANCE
	MODULE
	IMPORT
	EXPORT
	FORALL
	EXISTS
	TEST
	PROPERTY
	ASSERT
	SPAWN
	PARALLEL
	SELECT
	CHANNEL
	SEND
	RECV
	TIMEOUT

	// Operators
	PLUS     // +
	MINUS    // -
	STAR     // *
	SLASH    // /
	PERCENT  // %
	EQ       // ==
	NEQ      // !=
	LT       // <
	GT       // >
	LTE      // <=
	GTE      // >=
	AND      // &&
	OR       // ||
	NOT      // not
	ARROW    // ->
	FARROW   // =>
	LARROW   // <-
	PIPE     // |
	APPEND   // ++
	CONS     // ::
	COMPOSE  // .
	BANG     // !
	QUESTION // ?
	AT       // @
	DOLLAR   // $
	HASH     // #
	ASSIGN   // =
	COLON    // :
	DCOLON   // ::

	// Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	COMMA     // ,
	DOT       // .
	ELLIPSIS  // ...
	SEMICOLON // ;
	NEWLINE   // \n

	// Quasiquote types
	SQL_QUOTE   // sql"""
	HTML_QUOTE  // html"""
	JSON_QUOTE  // json{
	REGEX_QUOTE // regex/
	URL_QUOTE   // url"
	SHELL_QUOTE // shell"""

	// Effect markers
	EFFECT_MARKER // ! {effects}

	// Boolean literals
	TRUE
	FALSE

	// Unit type
	UNIT // ()
)

var tokens = map[TokenType]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	STRING: "STRING",
	CHAR:   "CHAR",

	FUNC:     "func",
	PURE:     "pure",
	LET:      "let",
	IN:       "in",
	IF:       "if",
	THEN:     "then",
	ELSE:     "else",
	MATCH:    "match",
	WITH:     "with",
	TYPE:     "type",
	CLASS:    "class",
	INSTANCE: "instance",
	MODULE:   "module",
	IMPORT:   "import",
	EXPORT:   "export",
	FORALL:   "forall",
	EXISTS:   "exists",
	TEST:     "test",
	PROPERTY: "property",
	ASSERT:   "assert",
	SPAWN:    "spawn",
	PARALLEL: "parallel",
	SELECT:   "select",
	CHANNEL:  "channel",
	SEND:     "send",
	RECV:     "recv",
	TIMEOUT:  "timeout",

	PLUS:     "+",
	MINUS:    "-",
	STAR:     "*",
	SLASH:    "/",
	PERCENT:  "%",
	EQ:       "==",
	NEQ:      "!=",
	LT:       "<",
	GT:       ">",
	LTE:      "<=",
	GTE:      ">=",
	AND:      "&&",
	OR:       "||",
	NOT:      "not",
	ARROW:    "->",
	FARROW:   "=>",
	LARROW:   "<-",
	PIPE:     "|",
	APPEND:   "++",
	CONS:     "::",
	COMPOSE:  ".",
	BANG:     "!",
	QUESTION: "?",
	AT:       "@",
	DOLLAR:   "$",
	HASH:     "#",
	ASSIGN:   "=",
	COLON:    ":",
	DCOLON:   "::",

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACE:    "{",
	RBRACE:    "}",
	LBRACKET:  "[",
	RBRACKET:  "]",
	COMMA:     ",",
	DOT:       ".",
	ELLIPSIS:  "...",
	SEMICOLON: ";",
	NEWLINE:   "\\n",

	SQL_QUOTE:   "sql\"\"\"",
	HTML_QUOTE:  "html\"\"\"",
	JSON_QUOTE:  "json{",
	REGEX_QUOTE: "regex/",
	URL_QUOTE:   "url\"",
	SHELL_QUOTE: "shell\"\"\"",

	EFFECT_MARKER: "!",

	TRUE:  "true",
	FALSE: "false",
	UNIT:  "()",
}

// String returns the string representation of a token type
func (t TokenType) String() string {
	if str, ok := tokens[t]; ok {
		return str
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// Keywords map
var keywords = map[string]TokenType{
	"func":     FUNC,
	"pure":     PURE,
	"let":      LET,
	"in":       IN,
	"if":       IF,
	"then":     THEN,
	"else":     ELSE,
	"match":    MATCH,
	"with":     WITH,
	"type":     TYPE,
	"class":    CLASS,
	"instance": INSTANCE,
	"module":   MODULE,
	"import":   IMPORT,
	"export":   EXPORT,
	"forall":   FORALL,
	"exists":   EXISTS,
	"test":     TEST,
	"property": PROPERTY,
	"assert":   ASSERT,
	"spawn":    SPAWN,
	"parallel": PARALLEL,
	"select":   SELECT,
	"channel":  CHANNEL,
	"send":     SEND,
	"recv":     RECV,
	"timeout":  TIMEOUT,
	"true":     TRUE,
	"false":    FALSE,
	"not":      NOT,
	"and":      AND,
	"or":       OR,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
	File    string
}

// NewToken creates a new token
func NewToken(tokenType TokenType, literal string, line, column int, file string) Token {
	return Token{
		Type:    tokenType,
		Literal: literal,
		Line:    line,
		Column:  column,
		File:    file,
	}
}

// Position returns the position of the token as a string
func (t Token) Position() string {
	return fmt.Sprintf("%s:%d:%d", t.File, t.Line, t.Column)
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token{%s, %q, %s}", t.Type, t.Literal, t.Position())
}

// IsOperator checks if a token is an operator
func (t Token) IsOperator() bool {
	switch t.Type {
	case PLUS, MINUS, STAR, SLASH, PERCENT,
		EQ, NEQ, LT, GT, LTE, GTE,
		AND, OR, NOT,
		APPEND, CONS, COMPOSE,
		PIPE:
		return true
	}
	return false
}

// IsKeyword checks if a token is a keyword
func (t Token) IsKeyword() bool {
	switch t.Type {
	case FUNC, PURE, LET, IN, IF, THEN, ELSE,
		MATCH, WITH, TYPE, CLASS, INSTANCE,
		MODULE, IMPORT, EXPORT,
		FORALL, EXISTS, TEST, PROPERTY, ASSERT,
		SPAWN, PARALLEL, SELECT, CHANNEL,
		SEND, RECV, TIMEOUT,
		TRUE, FALSE:
		return true
	}
	return false
}

// Precedence returns the precedence of an operator
func (t Token) Precedence() int {
	switch t.Type {
	case OR:
		return 1
	case AND:
		return 2
	case EQ, NEQ:
		return 3
	case LT, GT, LTE, GTE:
		return 4
	case APPEND:
		return 5
	case CONS:
		return 5
	case PLUS, MINUS:
		return 6
	case STAR, SLASH, PERCENT:
		return 7
	case COMPOSE:
		return 8
	case DOT:
		return 9
	case LPAREN:
		return 10
	default:
		return 0
	}
}