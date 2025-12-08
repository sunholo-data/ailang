package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestDetectJavaScriptNamespaceImport tests detection of "import X from 'Y'" pattern
func TestDetectJavaScriptNamespaceImport(t *testing.T) {
	input := `import http from 'http'`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(p.errors))
	}

	err, ok := p.errors[0].(*ParserError)
	if !ok {
		t.Fatalf("expected *ParserError, got %T", p.errors[0])
	}

	if err.Code != "IMP012_UNSUPPORTED_NAMESPACE" {
		t.Errorf("expected error code IMP012_UNSUPPORTED_NAMESPACE, got %s", err.Code)
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "namespace imports not yet supported") {
		t.Errorf("error message should mention namespace imports: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "import std/net (httpRequest)") {
		t.Errorf("error should suggest std/net import: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "import std/json (encode, decode)") {
		t.Errorf("error should suggest std/json import: %s", errorMsg)
	}

	if len(err.Suggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(err.Suggestions))
	}
}

// TestDetectConstKeyword tests detection of JavaScript "const" keyword
func TestDetectConstKeyword(t *testing.T) {
	input := `const URL = "https://example.com"`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}

	// Find the PAR014 error
	var err *ParserError
	for _, e := range p.errors {
		if pe, ok := e.(*ParserError); ok && pe.Code == "PAR014" {
			err = pe
			break
		}
	}

	if err == nil {
		t.Fatal("expected PAR014 error")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "'const' keyword doesn't exist") {
		t.Errorf("error message should mention const keyword: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "let name = value in") {
		t.Errorf("error should suggest let syntax: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "immutable by default") {
		t.Errorf("error should mention immutability: %s", errorMsg)
	}

	if len(err.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(err.Suggestions))
	}
}

// TestDetectBareAssignment tests detection of Python-style "x = y" without let
func TestDetectBareAssignment(t *testing.T) {
	input := `url = "https://httpbin.org/post"`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	if len(p.errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}

	// Find the PAR015 error
	var err *ParserError
	for _, e := range p.errors {
		if pe, ok := e.(*ParserError); ok && pe.Code == "PAR015" {
			err = pe
			break
		}
	}

	if err == nil {
		t.Fatal("expected PAR015 error")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "bare assignment not supported") {
		t.Errorf("error message should mention bare assignment: %s", errorMsg)
	}

	// Check for variable-specific suggestion (parser_decl.go provides actual variable name)
	if !strings.Contains(errorMsg, "let url = ... in") {
		t.Errorf("error should suggest let binding with variable name: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "requires 'let' keyword") {
		t.Errorf("error should mention let requirement: %s", errorMsg)
	}

	if len(err.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(err.Suggestions))
	}
}

// TestActualEvalFailureExample1 tests Error 1 from the design doc
func TestActualEvalFailureExample1(t *testing.T) {
	// This is the actual code that claude-haiku-4-5 generated
	input := `import http from 'http'

// Make HTTP POST request to httpbin.org
let response = http.post('https://httpbin.org/post', {
  headers: {
    'X-Test-Header': 'value123',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    message: 'Hello from AILANG',
    count: 42
  })
})

// Print the status code
print(response.statusCode)`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	// Should have at least one error for the import statement
	if len(p.errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}

	// Check that first error is about namespace import
	err, ok := p.errors[0].(*ParserError)
	if !ok {
		t.Fatalf("expected *ParserError, got %T", p.errors[0])
	}

	if err.Code != "IMP012_UNSUPPORTED_NAMESPACE" {
		t.Errorf("expected IMP012_UNSUPPORTED_NAMESPACE, got %s", err.Code)
	}

	// Verify suggestions are helpful
	if len(err.Suggestions) == 0 {
		t.Error("expected suggestions, got none")
	}
}

// TestActualEvalFailureExample2 tests Error 2 from the design doc
func TestActualEvalFailureExample2(t *testing.T) {
	// This is the actual code that gemini-2-5-flash generated
	input := `const URL = "https://httpbin.org/post";
const HEADERS = {
    "X-Test-Header": "value123",
    "Content-Type": "application/json"
};`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	// Should have at least one error for const keyword
	if len(p.errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}

	// Check that first error is about const keyword
	err, ok := p.errors[0].(*ParserError)
	if !ok {
		t.Fatalf("expected *ParserError, got %T", p.errors[0])
	}

	if err.Code != "PAR014" {
		t.Errorf("expected PAR014, got %s", err.Code)
	}

	// Verify suggestions mention let
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "let") {
		t.Errorf("error should suggest 'let' keyword: %s", errorMsg)
	}
}

// TestActualEvalFailureExample3 tests Error 3 from the design doc
func TestActualEvalFailureExample3(t *testing.T) {
	// This is the actual code that gpt5-mini generated
	input := `import http
import json

url = "https://httpbin.org/post"
headers = {"X-Test-Header": "value123", "Content-Type": "application/json"}`

	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()

	// Should have errors for both bare imports and bare assignments
	if len(p.errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}

	// Look for bare assignment error (url = ...)
	foundBareAssignment := false
	for _, e := range p.errors {
		if err, ok := e.(*ParserError); ok {
			if err.Code == "PAR015" {
				foundBareAssignment = true
				break
			}
		}
	}

	if !foundBareAssignment {
		t.Error("expected PAR015 error for 'url = ...'")
	}
}

// TestMultipleSuggestionsFormatting tests that multiple suggestions are formatted correctly
func TestMultipleSuggestionsFormatting(t *testing.T) {
	testToken := lexer.Token{
		Type:    lexer.IDENT,
		Literal: "test",
		Line:    1,
		Column:  1,
		File:    "<test>",
	}

	err := NewSuggestionError(
		"TEST_CODE",
		ast.Pos{Line: 1, Column: 1, File: "<test>"},
		testToken,
		"test error message",
		[]string{
			"suggestion 1",
			"suggestion 2",
			"suggestion 3",
		},
		"https://example.com",
	)

	errorMsg := err.Error()

	// Check that all components are present
	if !strings.Contains(errorMsg, "test error message") {
		t.Error("error message should contain main message")
	}

	if !strings.Contains(errorMsg, "Did you mean one of these?") {
		t.Error("error should have suggestions header")
	}

	if !strings.Contains(errorMsg, "suggestion 1") {
		t.Error("error should contain suggestion 1")
	}

	if !strings.Contains(errorMsg, "suggestion 2") {
		t.Error("error should contain suggestion 2")
	}

	if !strings.Contains(errorMsg, "suggestion 3") {
		t.Error("error should contain suggestion 3")
	}

	if !strings.Contains(errorMsg, "https://example.com") {
		t.Error("error should contain help URL")
	}
}

// TestBackwardCompatibilityWithFix tests that old Fix field still works
func TestBackwardCompatibilityWithFix(t *testing.T) {
	testToken := lexer.Token{
		Type:    lexer.IDENT,
		Literal: "test",
		Line:    1,
		Column:  1,
		File:    "<test>",
	}

	err := NewParserError(
		"TEST_CODE",
		ast.Pos{Line: 1, Column: 1, File: "<test>"},
		testToken,
		"test error",
		nil,
		"use this fix instead",
	)

	errorMsg := err.Error()

	// Should show the fix in backward-compatible format
	if !strings.Contains(errorMsg, "Suggestion: use this fix instead") {
		t.Errorf("error should show Fix field: %s", errorMsg)
	}
}
