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
	LETREC
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
	EXTERN // extern func declarations for Go interop
	FORALL
	EXISTS
	TEST
	TESTS // tests block
	PROPERTY
	PROPERTIES // properties block
	ASSERT
	SPAWN
	PARALLEL
	SELECT
	CHANNEL
	SEND
	RECV
	TIMEOUT
	AS       // as (import aliasing)
	DERIVING // deriving (type class derivation)

	// Contract keywords (M-VERIFY)
	REQUIRES  // requires
	ENSURES   // ensures
	INVARIANT // invariant

	// Operators
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	PERCENT   // %
	EQ        // ==
	NEQ       // !=
	LT        // <
	GT        // >
	LTE       // <=
	GTE       // >=
	AND       // &&
	OR        // ||
	NOT       // not
	ARROW     // ->
	FARROW    // =>
	LARROW    // <-
	PIPE      // |
	APPEND    // ++
	CONS      // ::
	COMPOSE   // .
	BANG      // !
	QUESTION  // ?
	AT        // @
	DOLLAR    // $
	HASH      // #
	ASSIGN    // =
	COLON     // :
	DCOLON    // ::
	BACKSLASH // \

	// Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	COMMA     // ,
	DOT       // .
	DOTDOT    // ..
	ELLIPSIS  // ...
	SEMICOLON // ;
	NEWLINE   // \n

	// Quasiquote types
	SQLQuote   // sql"""
	HTMLQuote  // html"""
	JSONQuote  // json{
	RegexQuote // regex/
	URLQuote   // url"
	ShellQuote // shell"""

	// Effect markers
	EffectMarker // ! {effects}

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

	FUNC:       "func",
	PURE:       "pure",
	LET:        "let",
	LETREC:     "letrec",
	IN:         "in",
	IF:         "if",
	THEN:       "then",
	ELSE:       "else",
	MATCH:      "match",
	WITH:       "with",
	TYPE:       "type",
	CLASS:      "class",
	INSTANCE:   "instance",
	MODULE:     "module",
	IMPORT:     "import",
	EXPORT:     "export",
	EXTERN:     "extern",
	FORALL:     "forall",
	EXISTS:     "exists",
	TEST:       "test",
	TESTS:      "tests",
	PROPERTY:   "property",
	PROPERTIES: "properties",
	ASSERT:     "assert",
	SPAWN:      "spawn",
	PARALLEL:   "parallel",
	SELECT:     "select",
	CHANNEL:    "channel",
	SEND:       "send",
	RECV:       "recv",
	TIMEOUT:    "timeout",
	AS:         "as",
	DERIVING:   "deriving",
	REQUIRES:   "requires",
	ENSURES:    "ensures",
	INVARIANT:  "invariant",

	PLUS:      "+",
	MINUS:     "-",
	STAR:      "*",
	SLASH:     "/",
	PERCENT:   "%",
	EQ:        "==",
	NEQ:       "!=",
	LT:        "<",
	GT:        ">",
	LTE:       "<=",
	GTE:       ">=",
	AND:       "&&",
	OR:        "||",
	NOT:       "not",
	ARROW:     "->",
	FARROW:    "=>",
	LARROW:    "<-",
	PIPE:      "|",
	APPEND:    "++",
	CONS:      "::",
	COMPOSE:   ".",
	BANG:      "!",
	QUESTION:  "?",
	AT:        "@",
	DOLLAR:    "$",
	HASH:      "#",
	ASSIGN:    "=",
	COLON:     ":",
	DCOLON:    "::",
	BACKSLASH: "\\",

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACE:    "{",
	RBRACE:    "}",
	LBRACKET:  "[",
	RBRACKET:  "]",
	COMMA:     ",",
	DOT:       ".",
	DOTDOT:    "..",
	ELLIPSIS:  "...",
	SEMICOLON: ";",
	NEWLINE:   "\\n",

	SQLQuote:   "sql\"\"\"",
	HTMLQuote:  "html\"\"\"",
	JSONQuote:  "json{",
	RegexQuote: "regex/",
	URLQuote:   "url\"",
	ShellQuote: "shell\"\"\"",

	EffectMarker: "!",

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
	"func":       FUNC,
	"pure":       PURE,
	"let":        LET,
	"letrec":     LETREC,
	"in":         IN,
	"if":         IF,
	"then":       THEN,
	"else":       ELSE,
	"match":      MATCH,
	"with":       WITH,
	"type":       TYPE,
	"class":      CLASS,
	"instance":   INSTANCE,
	"module":     MODULE,
	"import":     IMPORT,
	"export":     EXPORT,
	"extern":     EXTERN,
	"forall":     FORALL,
	"exists":     EXISTS,
	"test":       TEST,
	"tests":      TESTS,
	"property":   PROPERTY,
	"properties": PROPERTIES,
	"assert":     ASSERT,
	"spawn":      SPAWN,
	"parallel":   PARALLEL,
	"select":     SELECT,
	"channel":    CHANNEL,
	"send":       SEND,
	"recv":       RECV,
	"timeout":    TIMEOUT,
	"as":         AS,
	"deriving":   DERIVING,
	"requires":   REQUIRES,
	"ensures":    ENSURES,
	"invariant":  INVARIANT,
	"true":       TRUE,
	"false":      FALSE,
	"not":        NOT,
	"and":        AND,
	"or":         OR,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// LookupIdentContextual checks if an identifier is a keyword, but treats
// test/tests/properties as contextual (can be used as identifiers in some contexts)
func LookupIdentContextual(ident string) TokenType {
	// Contextual keywords that can be used as identifiers in some contexts
	switch ident {
	case "test", "tests", "properties", "property":
		// These are only keywords in specific contexts (after func declarations)
		// Return IDENT and let the parser decide based on context
		return IDENT
	default:
		// For all other keywords, use strict lookup
		return LookupIdent(ident)
	}
}

// IsReservedKeyword checks if a string is a reserved keyword
// This is used to prevent keywords from being used as identifiers
func IsReservedKeyword(ident string) bool {
	_, ok := keywords[ident]
	return ok
}

// IsContextualKeyword checks if a token type is only reserved in specific contexts
// For now, all keywords are strictly reserved, but this allows future flexibility
func IsContextualKeyword(t TokenType) bool {
	// In the future, we might allow 'tests' or 'properties' as field names
	// For now, keep all keywords strictly reserved
	return false
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
		MODULE, IMPORT, EXPORT, EXTERN,
		FORALL, EXISTS, TEST, TESTS, PROPERTY, PROPERTIES, ASSERT,
		SPAWN, PARALLEL, SELECT, CHANNEL,
		SEND, RECV, TIMEOUT, AS, DERIVING,
		REQUIRES, ENSURES, INVARIANT, // M-VERIFY contract keywords
		TRUE, FALSE:
		return true
	}
	return false
}

// Precedence returns the precedence of an operator - spec compliant ordering
func (t Token) Precedence() int {
	switch t.Type {
	case BACKSLASH:
		return 1 // LAMBDA - lowest precedence
	case OR:
		return 2 // LOGICAL_OR
	case AND:
		return 3 // LOGICAL_AND
	case EQ, NEQ:
		return 4 // EQUALS
	case LT, GT, LTE, GTE:
		return 5 // LESSGREATER
	case DCOLON:
		return 6 // CONS (:: list cons - right associative, S-CONS sugar)
	case APPEND:
		return 7 // APPEND (++ string concatenation)
	case PLUS, MINUS:
		return 8 // SUM
	case STAR, SLASH, PERCENT:
		return 9 // PRODUCT
	case NOT:
		return 10 // PREFIX (unary operators)
	case LPAREN, UNIT:
		return 11 // CALL (function application) - UNIT handles f() sugar
	case DOT:
		return 12 // DOT_ACCESS (field access - highest)
	default:
		return 0
	}
}
