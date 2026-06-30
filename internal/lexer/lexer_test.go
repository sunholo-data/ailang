package lexer

import (
	"strings"
	"testing"
)

func TestHexBinOctalLiterals(t *testing.T) {
	// Radix integer literals (0x/0b/0o) lex as single INT tokens, preserving the prefix in
	// the literal so the parser can base-detect. Leading-zero decimals stay decimal.
	input := `0xE000 0xff 0XaB 0b1010 0B11 0o17 0O7 0 42 0123`
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{INT, "0xE000"},
		{INT, "0xff"},
		{INT, "0XaB"},
		{INT, "0b1010"},
		{INT, "0B11"},
		{INT, "0o17"},
		{INT, "0O7"},
		{INT, "0"},
		{INT, "42"},
		{INT, "0123"},
		{EOF, ""},
	}
	l := New(input, "test.ail")
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType || tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d]: expected %v %q, got %v %q",
				i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken(t *testing.T) {
	input := `let x = 5 + 10
pure func add(a: int, b: int) -> int {
  a + b
}

if x > 10 then "big" else "small"

match value {
  Some(x) => x * 2,
  None => 0
}

[1, 2, 3] ++ [4, 5]
{ name: "Alice", age: 30 }

-- This is a comment
true && false || not true
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
		{PLUS, "+"},
		{INT, "10"},

		{PURE, "pure"},
		{FUNC, "func"},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "a"},
		{COLON, ":"},
		{IDENT, "int"},
		{COMMA, ","},
		{IDENT, "b"},
		{COLON, ":"},
		{IDENT, "int"},
		{RPAREN, ")"},
		{ARROW, "->"},
		{IDENT, "int"},
		{LBRACE, "{"},
		{IDENT, "a"},
		{PLUS, "+"},
		{IDENT, "b"},
		{RBRACE, "}"},

		{IF, "if"},
		{IDENT, "x"},
		{GT, ">"},
		{INT, "10"},
		{THEN, "then"},
		{STRING, "big"},
		{ELSE, "else"},
		{STRING, "small"},

		{MATCH, "match"},
		{IDENT, "value"},
		{LBRACE, "{"},
		{IDENT, "Some"},
		{LPAREN, "("},
		{IDENT, "x"},
		{RPAREN, ")"},
		{FARROW, "=>"},
		{IDENT, "x"},
		{STAR, "*"},
		{INT, "2"},
		{COMMA, ","},
		{IDENT, "None"},
		{FARROW, "=>"},
		{INT, "0"},
		{RBRACE, "}"},

		{LBRACKET, "["},
		{INT, "1"},
		{COMMA, ","},
		{INT, "2"},
		{COMMA, ","},
		{INT, "3"},
		{RBRACKET, "]"},
		{APPEND, "++"},
		{LBRACKET, "["},
		{INT, "4"},
		{COMMA, ","},
		{INT, "5"},
		{RBRACKET, "]"},

		{LBRACE, "{"},
		{IDENT, "name"},
		{COLON, ":"},
		{STRING, "Alice"},
		{COMMA, ","},
		{IDENT, "age"},
		{COLON, ":"},
		{INT, "30"},
		{RBRACE, "}"},

		{TRUE, "true"},
		{AND, "&&"},
		{FALSE, "false"},
		{OR, "||"},
		{NOT, "not"},
		{TRUE, "true"},

		{EOF, ""},
	}

	l := New(input, "test.ail")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestFloatLiterals(t *testing.T) {
	input := `3.14 2.0 1e10 1.5e-3`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{FLOAT, "3.14"},
		{FLOAT, "2.0"},
		{FLOAT, "1e10"},
		{FLOAT, "1.5e-3"},
		{EOF, ""},
	}

	l := New(input, "test.ail")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringEscapes(t *testing.T) {
	input := `"hello\nworld" "tab\there" "quote\"inside\""`

	l := New(input, "test.ail")

	tok1 := l.NextToken()
	if tok1.Type != STRING {
		t.Fatalf("expected STRING, got %q", tok1.Type)
	}
	if tok1.Literal != "hello\nworld" {
		t.Fatalf("expected %q, got %q", "hello\nworld", tok1.Literal)
	}

	tok2 := l.NextToken()
	if tok2.Type != STRING {
		t.Fatalf("expected STRING, got %q", tok2.Type)
	}
	if tok2.Literal != "tab\there" {
		t.Fatalf("expected %q, got %q", "tab\there", tok2.Literal)
	}

	tok3 := l.NextToken()
	if tok3.Type != STRING {
		t.Fatalf("expected STRING, got %q", tok3.Type)
	}
	if tok3.Literal != "quote\"inside\"" {
		t.Fatalf("expected %q, got %q", "quote\"inside\"", tok3.Literal)
	}
}

// TestHexAndUnicodeEscapes covers the C-style escapes added in M-TERMINAL-IO:
// \xHH, \u{...}, \e (ESC), \0 (NUL). These unblock real-time terminal output
// (ANSI sequences) without importing std/bytes.
func TestHexAndUnicodeEscapes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"hex ESC", `"\x1b[0m"`, "\x1b[0m"},
		{"hex uppercase", `"\x1B"`, "\x1b"},
		{"hex bell", `"\x07"`, "\x07"},
		{"esc alias", `"\e[1m"`, "\x1b[1m"},
		{"nul", `"\0"`, "\x00"},
		{"unicode bmp", `"\u{263A}"`, "☺"},
		{"unicode astral snake", `"\u{1F40D}"`, "\U0001F40D"},
		{"unicode 1 digit", `"\u{9}"`, "\t"},
		{"mixed with text and known escapes", `"a\tb\x1bc\u{41}"`, "a\tb\x1bcA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input, "test.ail")
			tok := l.NextToken()
			if tok.Type != STRING {
				t.Fatalf("expected STRING, got %q (literal %q)", tok.Type, tok.Literal)
			}
			if tok.Literal != tt.want {
				t.Fatalf("input %s: expected %q, got %q", tt.input, tt.want, tok.Literal)
			}
		})
	}
}

// TestMalformedEscapesAreIllegal verifies unknown/malformed escapes produce an
// ILLEGAL token carrying a helpful message — NEVER a silent passthrough.
func TestMalformedEscapesAreIllegal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string // substring expected in the ILLEGAL token literal
	}{
		{"hex too short", `"\x1"`, "2 hex digits"},
		{"hex non-hex", `"\xZZ"`, "2 hex digits"},
		{"unicode empty", `"\u{}"`, "at least one hex digit"},
		{"unicode no brace", `"\u41"`, "{HEX}"},
		{"unicode too big", `"\u{110000}"`, "not a valid Unicode"},
		{"unicode too many", `"\u{1234567}"`, "at most 6 hex digits"},
		{"octal", `"\033"`, "octal escapes are not supported"},
		{"unknown letter", `"\q"`, "unknown escape sequence"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input, "test.ail")
			tok := l.NextToken()
			if tok.Type != ILLEGAL {
				t.Fatalf("input %s: expected ILLEGAL, got %q (literal %q)", tt.input, tok.Type, tok.Literal)
			}
			if !strings.Contains(tok.Literal, tt.wantMsg) {
				t.Fatalf("input %s: expected message containing %q, got %q", tt.input, tt.wantMsg, tok.Literal)
			}
		})
	}
}

// TestCharLiteralEscapes verifies the shared escape decoder works in char literals too.
func TestCharLiteralEscapes(t *testing.T) {
	cases := map[string]string{
		`'\x1b'`:   "\x1b",
		`'\e'`:     "\x1b",
		`'\n'`:     "\n",
		`'\u{41}'`: "A",
	}
	for input, want := range cases {
		l := New(input, "test.ail")
		tok := l.NextToken()
		if tok.Type != CHAR {
			t.Fatalf("input %s: expected CHAR, got %q (literal %q)", input, tok.Type, tok.Literal)
		}
		if tok.Literal != want {
			t.Fatalf("input %s: expected %q, got %q", input, want, tok.Literal)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / % == != < > <= >= && || ! -> => <- | ++ :: . ? @ $ #`

	tests := []TokenType{
		PLUS, MINUS, STAR, SLASH, PERCENT,
		EQ, NEQ, LT, GT, LTE, GTE,
		AND, OR, BANG,
		ARROW, FARROW, LARROW,
		PIPE, APPEND, DCOLON, // Note: :: becomes DCOLON
		DOT, QUESTION, AT, DOLLAR, HASH,
		EOF,
	}

	l := New(input, "test.ail")

	for i, expected := range tests {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - wrong token type. expected=%q, got=%q",
				i, expected, tok.Type)
		}
	}
}

func TestUnitLiteral(t *testing.T) {
	input := `() (1, 2) ()`

	l := New(input, "test.ail")

	tok1 := l.NextToken()
	if tok1.Type != UNIT {
		t.Fatalf("expected UNIT, got %q", tok1.Type)
	}

	// Next should be a tuple
	tok2 := l.NextToken()
	if tok2.Type != LPAREN {
		t.Fatalf("expected LPAREN, got %q", tok2.Type)
	}

	// Skip through tuple
	for l.NextToken().Type != RPAREN {
	}

	tok3 := l.NextToken()
	if tok3.Type != UNIT {
		t.Fatalf("expected UNIT, got %q", tok3.Type)
	}
}

func TestKeywords(t *testing.T) {
	keywords := []string{
		"func", "pure", "let", "in", "if", "then", "else",
		"match", "with", "type", "class", "instance",
		"module", "import", "export", "forall", "exists",
		"test", "property", "assert", "spawn", "parallel",
		"select", "channel", "true", "false", "not", "as",
	}

	contextualKeywords := map[string]bool{
		"test": true, "property": true,
	}

	for _, kw := range keywords {
		l := New(kw, "test.ail")
		tok := l.NextToken()

		// Contextual keywords are intentionally returned as IDENT outside their context
		if contextualKeywords[kw] {
			if tok.Type != IDENT {
				t.Errorf("contextual keyword %q: expected IDENT, got %v", kw, tok.Type)
			}
		} else {
			expectedType := LookupIdent(kw)
			if tok.Type != expectedType {
				t.Errorf("keyword %q: expected type %v, got %v", kw, expectedType, tok.Type)
			}

			if tok.Type == IDENT {
				t.Errorf("keyword %q was parsed as IDENT", kw)
			}
		}
	}
}

func TestLineAndColumn(t *testing.T) {
	input := `let x = 5
func add(a, b) {
  a + b
}`

	l := New(input, "test.ail")

	// First line
	tok := l.NextToken() // let
	if tok.Line != 1 || tok.Column != 1 {
		t.Errorf("let: expected 1:1, got %d:%d", tok.Line, tok.Column)
	}

	tok = l.NextToken() // x
	if tok.Line != 1 || tok.Column != 5 {
		t.Errorf("x: expected 1:5, got %d:%d", tok.Line, tok.Column)
	}

	// Skip to second line
	for tok.Type != FUNC {
		tok = l.NextToken()
	}

	if tok.Line != 2 || tok.Column != 1 {
		t.Errorf("func: expected 2:1, got %d:%d", tok.Line, tok.Column)
	}
}

func TestComments(t *testing.T) {
	input := `-- This is a comment
let x = 5 -- inline comment
-- Another comment
func f() { x }`

	l := New(input, "test.ail")

	tok := l.NextToken()
	if tok.Type != LET {
		t.Fatalf("expected LET after comment, got %q", tok.Type)
	}

	// Continue through tokens, comments should be skipped
	expected := []TokenType{
		LET, IDENT, ASSIGN, INT,
		FUNC, IDENT, UNIT, LBRACE, IDENT, RBRACE,
		EOF,
	}

	l = New(input, "test.ail")
	for _, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("expected %v, got %v", exp, tok.Type)
		}
	}
}
