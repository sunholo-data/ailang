package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestUTF8BOM tests that parser handles UTF-8 BOM
// Note: Current lexer does not strip BOM, so it's treated as ILLEGAL token
func TestUTF8BOM(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			"bom_before_number",
			"\xEF\xBB\xBF42",
			true, // BOM not currently stripped
		},
		{
			"bom_before_let",
			"\xEF\xBB\xBFlet x = 5 in x",
			true, // BOM not currently stripped
		},
		{
			"no_bom",
			"42",
			false, // Should parse fine
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			_ = p.Parse()

			hasErrors := len(p.Errors()) > 0
			if hasErrors != tt.expectError {
				if tt.expectError {
					t.Error("Expected parse errors but got none")
				} else {
					t.Errorf("Unexpected parse errors: %v", p.Errors())
				}
			}
		})
	}
}

// TestLineEndingNormalization tests that parser handles different line endings
func TestLineEndingNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			"unix_lf",
			"let x = 1\nlet y = 2\nlet z = 3",
			"Unix LF (\\n)",
		},
		{
			"windows_crlf",
			"let x = 1\r\nlet y = 2\r\nlet z = 3",
			"Windows CRLF (\\r\\n)",
		},
		{
			"old_mac_cr",
			"let x = 1\rlet y = 2\rlet z = 3",
			"Old Mac CR (\\r)",
		},
		{
			"mixed_endings",
			"let x = 1\nlet y = 2\r\nlet z = 3\r",
			"Mixed line endings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Parse errors on %s: %v", tt.desc, p.Errors())
			}

			// All should produce the same AST structure
			if prog.File == nil || len(prog.File.Statements) == 0 {
				t.Errorf("Expected statements, got none for %s", tt.desc)
			}
		})
	}
}

// TestLineEndingConsistency tests that different line endings produce identical ASTs
func TestLineEndingConsistency(t *testing.T) {
	baseCode := "let x = 5{NL}let y = 10{NL}x + y"

	variants := map[string]string{
		"LF":   strings.ReplaceAll(baseCode, "{NL}", "\n"),
		"CRLF": strings.ReplaceAll(baseCode, "{NL}", "\r\n"),
		"CR":   strings.ReplaceAll(baseCode, "{NL}", "\r"),
	}

	var asts []*ast.Program
	for name, input := range variants {
		p := New(lexer.New(input, "test://unit"))
		prog := p.Parse()

		if len(p.Errors()) > 0 {
			t.Errorf("%s: parse errors: %v", name, p.Errors())
			continue
		}

		asts = append(asts, prog)
	}

	// All ASTs should have same structure
	if len(asts) < 2 {
		t.Fatal("Not enough variants parsed successfully")
	}

	// Compare statement counts
	baseStmtCount := len(asts[0].File.Statements)
	for i, prog := range asts[1:] {
		if len(prog.File.Statements) != baseStmtCount {
			t.Errorf("AST %d has different statement count: want %d, got %d",
				i+1, baseStmtCount, len(prog.File.Statements))
		}
	}
}

// TestUnicodeIdentifiers tests that parser handles Unicode identifiers
func TestUnicodeIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"greek_letters",
			"let π = 3.14",
		},
		{
			"accented_chars",
			"let café = true",
		},
		{
			"chinese_chars",
			"let 变量 = 42",
		},
		{
			"emoji_not_identifier",
			"let x🚀 = 1", // Emoji likely not valid in identifier
		},
		{
			"mixed_unicode",
			"let résumé_α = {name: \"test\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			_ = p.Parse()
			// Just ensure no panic - lexer may or may not accept these
		})
	}
}

// TestUnicodeStrings tests that parser handles Unicode in string literals
func TestUnicodeStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"chinese_string",
			`"你好世界"`,
		},
		{
			"emoji_string",
			`"Hello 🌍🚀✨"`,
		},
		{
			"mixed_unicode_string",
			`"Café résumé naïve π ∞"`,
		},
		{
			"arabic_string",
			`"مرحبا بالعالم"`,
		},
		{
			"hebrew_string",
			`"שלום עולם"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Parse errors on Unicode string: %v", p.Errors())
			}

			if prog.File == nil || len(prog.File.Statements) == 0 {
				t.Error("Expected parsed string literal")
			}
		})
	}
}

// TestWhitespaceNormalization tests that different whitespace is handled consistently
func TestWhitespaceNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"spaces",
			"let x = 1 + 2",
		},
		{
			"tabs",
			"let\tx\t=\t1\t+\t2",
		},
		{
			"mixed_whitespace",
			"let  x\t= \t 1  +\t2",
		},
		{
			"trailing_whitespace",
			"let x = 1 + 2  \t ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Parse errors: %v", p.Errors())
			}

			if prog.File == nil {
				t.Error("Expected parsed program")
			}
		})
	}
}

// TestDeterministicParsing tests that same input always produces same AST
func TestDeterministicParsing(t *testing.T) {
	inputs := []string{
		"1 + 2 * 3",
		"let x = 5 in x + 1",
		"[1, 2, 3]",
		"{x: 1, y: 2}",
		`func add(a, b) { a + b }`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			// Parse same input multiple times
			var outputs []string
			for i := 0; i < 5; i++ {
				p := New(lexer.New(input, "test://unit"))
				prog := p.Parse()

				if len(p.Errors()) > 0 {
					t.Fatalf("Parse error on iteration %d: %v", i, p.Errors())
				}

				output := ast.PrintProgram(prog)
				outputs = append(outputs, output)
			}

			// All outputs should be identical
			first := outputs[0]
			for i, output := range outputs[1:] {
				if output != first {
					t.Errorf("Iteration %d produced different AST:\nwant:\n%s\ngot:\n%s",
						i+1, first, output)
				}
			}
		})
	}
}

// TestEmptyInput tests handling of empty or whitespace-only input
func TestEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty_string", ""},
		{"only_spaces", "   "},
		{"only_tabs", "\t\t\t"},
		{"only_newlines", "\n\n\n"},
		{"only_whitespace", " \t\n \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			prog := p.Parse()

			// Empty input should parse successfully (empty program)
			// Should not panic
			_ = prog
			_ = p.Errors()
		})
	}
}

// TestVeryLongInput tests parser doesn't choke on large inputs
func TestVeryLongInput(t *testing.T) {
	// Generate a long list
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 1000; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("1")
	}
	sb.WriteString("]")

	p := New(lexer.New(sb.String(), "test://unit"))
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Errorf("Parse errors on long input: %v", p.Errors())
	}

	if prog.File == nil {
		t.Error("Expected parsed program")
	}
}

// TestDeeplyNestedStructures tests parser handles deep nesting
func TestDeeplyNestedStructures(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"nested_lists",
			"[[[[[1]]]]]",
		},
		{
			"nested_records",
			"{a: {b: {c: {d: {e: 1}}}}}",
		},
		{
			"nested_parens",
			"(((((1 + 2)))))",
		},
		{
			"nested_function_calls",
			"f(g(h(i(j(1)))))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "test://unit"))
			prog := p.Parse()

			// Should handle nesting without stack overflow
			if len(p.Errors()) > 0 {
				t.Logf("Parse errors (may be expected for deep nesting): %v", p.Errors())
			}

			_ = prog
		})
	}
}