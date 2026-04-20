package lexer

import (
	"testing"
)

// M1_LEXER_INTERP: String interpolation tokenization tests.
//
// Verifies that `"...${expr}..."` is tokenized as:
//   STRING_PART(prefix), INTERP_START, <expr tokens>, INTERP_END, STRING_PART(suffix)
//
// Plain strings (no `${`) remain a single STRING token (zero regression).
// Escaped `\${` produces a literal `${` inside a plain STRING token.

func collectTokens(input string) []Token {
	l := New(input, "test.ail")
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func TestInterp_PlainStringUnchanged(t *testing.T) {
	// No interpolation → single STRING token (regression guard).
	toks := collectTokens(`"hello world"`)
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens (STRING + EOF), got %d: %v", len(toks), toks)
	}
	if toks[0].Type != STRING || toks[0].Literal != "hello world" {
		t.Errorf("expected STRING(%q), got %v(%q)", "hello world", toks[0].Type, toks[0].Literal)
	}
}

func TestInterp_SimpleIdent(t *testing.T) {
	// "hi ${name}" → STRING_PART("hi "), INTERP_START, IDENT(name), INTERP_END, STRING_PART("")
	toks := collectTokens(`"hi ${name}"`)
	expected := []struct {
		typ TokenType
		lit string
	}{
		{STRING_PART, "hi "},
		{INTERP_START, "${"},
		{IDENT, "name"},
		{INTERP_END, "}"},
		{STRING_PART, ""},
		{EOF, ""},
	}
	if len(toks) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(toks), toks)
	}
	for i, e := range expected {
		if toks[i].Type != e.typ || toks[i].Literal != e.lit {
			t.Errorf("token %d: expected %v(%q), got %v(%q)", i, e.typ, e.lit, toks[i].Type, toks[i].Literal)
		}
	}
}

func TestInterp_LeadingInterpolation(t *testing.T) {
	// "${x}!" → STRING_PART(""), INTERP_START, IDENT(x), INTERP_END, STRING_PART("!")
	toks := collectTokens(`"${x}!"`)
	expected := []TokenType{STRING_PART, INTERP_START, IDENT, INTERP_END, STRING_PART, EOF}
	if len(toks) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(toks), toks)
	}
	for i, e := range expected {
		if toks[i].Type != e {
			t.Errorf("token %d: expected %v, got %v", i, e, toks[i].Type)
		}
	}
	if toks[0].Literal != "" {
		t.Errorf("leading STRING_PART should be empty, got %q", toks[0].Literal)
	}
	if toks[4].Literal != "!" {
		t.Errorf("trailing STRING_PART should be %q, got %q", "!", toks[4].Literal)
	}
}

func TestInterp_MultipleInterpolations(t *testing.T) {
	// "a${x}b${y}c"
	toks := collectTokens(`"a${x}b${y}c"`)
	expected := []TokenType{
		STRING_PART, INTERP_START, IDENT, INTERP_END,
		STRING_PART, INTERP_START, IDENT, INTERP_END,
		STRING_PART, EOF,
	}
	if len(toks) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(toks), toks)
	}
	for i, e := range expected {
		if toks[i].Type != e {
			t.Errorf("token %d: expected %v, got %v", i, e, toks[i].Type)
		}
	}
	// Verify literal content
	if toks[0].Literal != "a" {
		t.Errorf("expected %q, got %q", "a", toks[0].Literal)
	}
	if toks[4].Literal != "b" {
		t.Errorf("expected %q, got %q", "b", toks[4].Literal)
	}
	if toks[8].Literal != "c" {
		t.Errorf("expected %q, got %q", "c", toks[8].Literal)
	}
}

func TestInterp_NestedBraces(t *testing.T) {
	// "${compute({a: 1}.a)}" — `{}` inside expression must not prematurely close the interpolation.
	toks := collectTokens(`"${compute({a: 1}.a)}"`)
	// Expected core structure:
	//   STRING_PART("") INTERP_START
	//     IDENT(compute) LPAREN LBRACE IDENT(a) COLON INT(1) RBRACE DOT IDENT(a) RPAREN
	//   INTERP_END STRING_PART("") EOF
	if toks[0].Type != STRING_PART || toks[0].Literal != "" {
		t.Fatalf("expected leading empty STRING_PART, got %v(%q)", toks[0].Type, toks[0].Literal)
	}
	if toks[1].Type != INTERP_START {
		t.Fatalf("expected INTERP_START, got %v", toks[1].Type)
	}
	// Find the INTERP_END (should be exactly one)
	interpEnds := 0
	for _, tok := range toks {
		if tok.Type == INTERP_END {
			interpEnds++
		}
	}
	if interpEnds != 1 {
		t.Errorf("expected exactly 1 INTERP_END, got %d — nested `{}` was handled incorrectly", interpEnds)
	}
	// Verify interior tokens include LBRACE and RBRACE
	sawLBrace := false
	sawRBrace := false
	for _, tok := range toks {
		if tok.Type == LBRACE {
			sawLBrace = true
		}
		if tok.Type == RBRACE {
			sawRBrace = true
		}
	}
	if !sawLBrace {
		t.Errorf("expected LBRACE token inside interpolation expression")
	}
	if !sawRBrace {
		t.Errorf("expected RBRACE token inside interpolation expression")
	}
}

func TestInterp_EscapedDollarBrace(t *testing.T) {
	// `\${literal}` should produce a plain STRING with literal `${literal}` — no interpolation.
	toks := collectTokens(`"\${literal}"`)
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens (STRING + EOF), got %d: %v", len(toks), toks)
	}
	if toks[0].Type != STRING {
		t.Errorf("expected STRING (no interpolation), got %v", toks[0].Type)
	}
	if toks[0].Literal != "${literal}" {
		t.Errorf("expected literal %q, got %q", "${literal}", toks[0].Literal)
	}
}

func TestInterp_Arithmetic(t *testing.T) {
	// "${x + 1}" — expression tokens inside
	toks := collectTokens(`"${x + 1}"`)
	// STRING_PART INTERP_START IDENT PLUS INT INTERP_END STRING_PART EOF
	if len(toks) != 8 {
		t.Fatalf("expected 8 tokens, got %d: %v", len(toks), toks)
	}
	expected := []TokenType{STRING_PART, INTERP_START, IDENT, PLUS, INT, INTERP_END, STRING_PART, EOF}
	for i, e := range expected {
		if toks[i].Type != e {
			t.Errorf("token %d: expected %v, got %v", i, e, toks[i].Type)
		}
	}
}

func TestInterp_FunctionCall(t *testing.T) {
	// "Got: ${show(value)}"
	toks := collectTokens(`"Got: ${show(value)}"`)
	// STRING_PART("Got: ") INTERP_START IDENT(show) LPAREN IDENT(value) RPAREN INTERP_END STRING_PART("") EOF
	if len(toks) != 9 {
		t.Fatalf("expected 9 tokens, got %d: %v", len(toks), toks)
	}
	if toks[0].Literal != "Got: " {
		t.Errorf("expected %q, got %q", "Got: ", toks[0].Literal)
	}
}

func TestInterp_EmptyString(t *testing.T) {
	// "" → single STRING with empty literal
	toks := collectTokens(`""`)
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens (STRING + EOF), got %d: %v", len(toks), toks)
	}
	if toks[0].Type != STRING || toks[0].Literal != "" {
		t.Errorf("expected empty STRING, got %v(%q)", toks[0].Type, toks[0].Literal)
	}
}

func TestInterp_WithEscapes(t *testing.T) {
	// "hi\n${name}\t!" — mix of escapes and interpolation
	toks := collectTokens("\"hi\\n${name}\\t!\"")
	// STRING_PART("hi\n") INTERP_START IDENT INTERP_END STRING_PART("\t!") EOF
	if len(toks) != 6 {
		t.Fatalf("expected 6 tokens, got %d: %v", len(toks), toks)
	}
	if toks[0].Type != STRING_PART || toks[0].Literal != "hi\n" {
		t.Errorf("prefix: expected STRING_PART(%q), got %v(%q)", "hi\n", toks[0].Type, toks[0].Literal)
	}
	if toks[4].Type != STRING_PART || toks[4].Literal != "\t!" {
		t.Errorf("suffix: expected STRING_PART(%q), got %v(%q)", "\t!", toks[4].Type, toks[4].Literal)
	}
}

func TestInterp_LineTracking(t *testing.T) {
	// Line numbers should be preserved for tokens inside interpolation.
	l := New("\n\"hi ${name}\"", "test.ail")
	var toks []Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == EOF {
			break
		}
	}
	// All string-related tokens should be on line 2
	for _, tok := range toks {
		if tok.Type == EOF {
			continue
		}
		if tok.Line != 2 {
			t.Errorf("token %v(%q) should be on line 2, got line %d", tok.Type, tok.Literal, tok.Line)
		}
	}
}
