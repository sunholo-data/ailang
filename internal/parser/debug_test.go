package parser

import (
	"os"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

func TestDebugParserEnabled(t *testing.T) {
	// Save and restore original value
	orig := os.Getenv("DEBUG_PARSER")
	defer func() {
		if orig != "" {
			os.Setenv("DEBUG_PARSER", orig)
		} else {
			os.Unsetenv("DEBUG_PARSER")
		}
	}()

	// Test disabled (default)
	os.Unsetenv("DEBUG_PARSER")
	if debugParserEnabled() {
		t.Error("Expected debugParserEnabled() = false when unset")
	}

	// Test enabled
	os.Setenv("DEBUG_PARSER", "1")
	if !debugParserEnabled() {
		t.Error("Expected debugParserEnabled() = true when set to '1'")
	}

	// Test other values don't enable it
	os.Setenv("DEBUG_PARSER", "true")
	if debugParserEnabled() {
		t.Error("Expected debugParserEnabled() = false when set to 'true' (not '1')")
	}

	os.Setenv("DEBUG_PARSER", "0")
	if debugParserEnabled() {
		t.Error("Expected debugParserEnabled() = false when set to '0'")
	}
}

func TestFormatToken(t *testing.T) {
	tests := []struct {
		name     string
		token    lexer.Token
		expected string
	}{
		{
			name:     "EOF",
			token:    lexer.Token{Type: lexer.EOF},
			expected: "EOF",
		},
		{
			name:     "Integer literal",
			token:    lexer.Token{Type: lexer.INT, Literal: "42"},
			expected: "INT(42)",
		},
		{
			name:     "Float literal",
			token:    lexer.Token{Type: lexer.FLOAT, Literal: "3.14"},
			expected: "FLOAT(3.14)",
		},
		{
			name:     "String literal",
			token:    lexer.Token{Type: lexer.STRING, Literal: "hello"},
			expected: "STRING(hello)",
		},
		{
			name:     "Identifier",
			token:    lexer.Token{Type: lexer.IDENT, Literal: "factorial"},
			expected: "IDENT(factorial)",
		},
		{
			name:     "Plus operator",
			token:    lexer.Token{Type: lexer.PLUS, Literal: "+"},
			expected: "+(+)", // tok.Type.String() returns "+"
		},
		{
			name:     "Keyword let",
			token:    lexer.Token{Type: lexer.LET, Literal: "let"},
			expected: "let(let)", // tok.Type.String() returns "let"
		},
		{
			name:     "LPAREN",
			token:    lexer.Token{Type: lexer.LPAREN, Literal: "("},
			expected: "((()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToken(tt.token)
			if result != tt.expected {
				t.Errorf("formatToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDebugEnterExit(t *testing.T) {
	// This test just verifies that debugEnter/debugExit don't panic
	// We can't easily test the output without capturing stderr

	input := "42"
	l := lexer.New(input, "test.ail")
	p := New(l)

	// Test with DEBUG_PARSER disabled (default)
	p.debugEnter("testFunc")
	p.debugExit("testFunc")
	p.debugLog("testFunc", "test message")

	// Test with DEBUG_PARSER enabled
	orig := os.Getenv("DEBUG_PARSER")
	defer func() {
		if orig != "" {
			os.Setenv("DEBUG_PARSER", orig)
		} else {
			os.Unsetenv("DEBUG_PARSER")
		}
	}()

	os.Setenv("DEBUG_PARSER", "1")
	p.debugEnter("testFunc")
	p.debugExit("testFunc")
	p.debugLog("testFunc", "test message")

	// If we got here without panic, test passes
}

// TestDebugOutputFormat is a manual test that shows debug output format.
// Run with: DEBUG_PARSER=1 go test -run TestDebugOutputFormat -v
func TestDebugOutputFormat(t *testing.T) {
	if os.Getenv("DEBUG_PARSER") != "1" {
		t.Skip("Set DEBUG_PARSER=1 to see debug output format")
	}

	input := "42 + 10"
	l := lexer.New(input, "test.ail")
	p := New(l)

	t.Log("Parser initialized:")
	p.debugEnter("testFunction")

	t.Log("\nMoving to next token:")
	p.nextToken()
	p.debugLog("testFunction", "Advanced parser")

	t.Log("\nExiting function:")
	p.debugExit("testFunction")
}
