package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

// TestCLIIntegration_JavaScriptImport tests that JavaScript import errors show helpful suggestions
func TestCLIIntegration_JavaScriptImport(t *testing.T) {
	input := `import http from 'http'

let response = http.post('https://httpbin.org/post')
print(response.statusCode)`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) == 0 {
		t.Fatal("expected at least one error")
	}

	// Get the first error and format it as the CLI would
	errorMsg := p.errors[0].Error()

	// Verify the error message contains all the important parts
	checks := []string{
		"namespace imports not yet supported",
		"import std/net (httpRequest)",
		"import std/json (encode, decode)",
		"https://sunholo-data.github.io/ailang/docs/guides/module_execution",
	}

	for _, check := range checks {
		if !strings.Contains(errorMsg, check) {
			t.Errorf("Error message missing expected content: %q\nFull error:\n%s", check, errorMsg)
		}
	}
}

// TestCLIIntegration_ConstKeyword tests that const keyword errors show helpful suggestions
func TestCLIIntegration_ConstKeyword(t *testing.T) {
	input := `const URL = "https://httpbin.org/post"
const HEADERS = {"X-Test-Header": "value123"}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) == 0 {
		t.Fatal("expected at least one error")
	}

	// Find the PAR_CONST_NOT_SUPPORTED error
	var constError *ParserError
	for _, e := range p.errors {
		if pe, ok := e.(*ParserError); ok && pe.Code == "PAR_CONST_NOT_SUPPORTED" {
			constError = pe
			break
		}
	}

	if constError == nil {
		t.Fatal("expected PAR_CONST_NOT_SUPPORTED error")
	}

	errorMsg := constError.Error()

	// Verify suggestions
	checks := []string{
		"'const' keyword doesn't exist",
		"let name = value in",
		"immutable by default",
		"https://sunholo-data.github.io/ailang/docs/reference/language-syntax",
	}

	for _, check := range checks {
		if !strings.Contains(errorMsg, check) {
			t.Errorf("Error message missing expected content: %q\nFull error:\n%s", check, errorMsg)
		}
	}
}

// TestCLIIntegration_BareAssignment tests that bare assignment errors show helpful suggestions
func TestCLIIntegration_BareAssignment(t *testing.T) {
	input := `url = "https://httpbin.org/post"
headers = {"X-Test-Header": "value123"}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) == 0 {
		t.Fatal("expected at least one error")
	}

	// Find the PAR_BARE_ASSIGNMENT error
	var bareAssignError *ParserError
	for _, e := range p.errors {
		if pe, ok := e.(*ParserError); ok && pe.Code == "PAR_BARE_ASSIGNMENT" {
			bareAssignError = pe
			break
		}
	}

	if bareAssignError == nil {
		t.Fatal("expected PAR_BARE_ASSIGNMENT error")
	}

	errorMsg := bareAssignError.Error()

	// Verify suggestions mention the actual variable name
	checks := []string{
		"bare assignment not supported",
		"let url = ... in",
		"requires 'let' keyword",
		"https://sunholo-data.github.io/ailang/docs/reference/language-syntax",
	}

	for _, check := range checks {
		if !strings.Contains(errorMsg, check) {
			t.Errorf("Error message missing expected content: %q\nFull error:\n%s", check, errorMsg)
		}
	}
}

// TestErrorFormattingConsistency ensures all error types format consistently
func TestErrorFormattingConsistency(t *testing.T) {
	tests := []struct {
		name  string
		input string
		code  string
	}{
		{
			name:  "JavaScript import",
			input: `import http from 'http'`,
			code:  "IMP012_UNSUPPORTED_NAMESPACE",
		},
		{
			name:  "const keyword",
			input: `const X = 1`,
			code:  "PAR_CONST_NOT_SUPPORTED",
		},
		{
			name:  "bare assignment",
			input: `x = 1`,
			code:  "PAR_BARE_ASSIGNMENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			_ = p.Parse()

			// Find the expected error
			var found *ParserError
			for _, e := range p.errors {
				if pe, ok := e.(*ParserError); ok && pe.Code == tt.code {
					found = pe
					break
				}
			}

			if found == nil {
				t.Fatalf("expected error with code %s", tt.code)
			}

			errorMsg := found.Error()

			// All our new errors should have:
			// 1. "Did you mean one of these?" header
			// 2. At least one suggestion
			// 3. A help URL
			if !strings.Contains(errorMsg, "Did you mean one of these?") {
				t.Error("error should have 'Did you mean one of these?' header")
			}

			if !strings.Contains(errorMsg, "https://sunholo-data.github.io/ailang") {
				t.Error("error should have help URL")
			}

			if len(found.Suggestions) == 0 {
				t.Error("error should have at least one suggestion")
			}
		})
	}
}
