package lexer

import (
	"testing"
)

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
		"select", "channel", "true", "false", "not",
	}

	for _, kw := range keywords {
		l := New(kw, "test.ail")
		tok := l.NextToken()

		expectedType := LookupIdent(kw)
		if tok.Type != expectedType {
			t.Errorf("keyword %q: expected type %v, got %v", kw, expectedType, tok.Type)
		}

		if tok.Type == IDENT {
			t.Errorf("keyword %q was parsed as IDENT", kw)
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
