package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// ParserError represents a structured parser error with fix suggestions
type ParserError struct {
	Code        string
	Message     string
	Pos         ast.Pos
	NearToken   lexer.Token
	Expected    []lexer.TokenType
	Fix         string // Deprecated: use Suggestions instead
	Suggestions []string
	HelpURL     string
	Confidence  float64
}

func (e *ParserError) Error() string {
	msg := fmt.Sprintf("%s at %s: %s", e.Code, e.Pos, e.Message)

	// Add suggestions if available
	if len(e.Suggestions) > 0 {
		msg += "\n\nDid you mean one of these?"
		for _, suggestion := range e.Suggestions {
			msg += "\n  " + suggestion
		}
	} else if e.Fix != "" {
		// Backward compatibility with single Fix field
		msg += "\n\nSuggestion: " + e.Fix
	}

	// Add help URL if available
	if e.HelpURL != "" {
		msg += "\n\nSee: " + e.HelpURL
	}

	return msg
}

// NewParserError creates a structured parser error with fix suggestion
func NewParserError(code string, pos ast.Pos, nearToken lexer.Token, message string, expected []lexer.TokenType, fix string) *ParserError {
	return &ParserError{
		Code:       code,
		Message:    message,
		Pos:        pos,
		NearToken:  nearToken,
		Expected:   expected,
		Fix:        fix,
		Confidence: 0.85, // Default confidence for parser fixes
	}
}

// NewSuggestionError creates a parser error with multiple suggestions and optional help URL
func NewSuggestionError(code string, pos ast.Pos, nearToken lexer.Token, message string, suggestions []string, helpURL string) *ParserError {
	return &ParserError{
		Code:        code,
		Message:     message,
		Pos:         pos,
		NearToken:   nearToken,
		Suggestions: suggestions,
		HelpURL:     helpURL,
		Confidence:  0.85,
	}
}

// report is a convenience helper for adding structured errors to the parser
func (p *Parser) report(code string, message string, fix string) {
	err := NewParserError(code, p.curPos(), p.curToken, message, nil, fix)
	p.errors = append(p.errors, err)
}

// reportExpected is a convenience helper for "expected X, got Y" errors
func (p *Parser) reportExpected(expected lexer.TokenType, fix string) {
	message := fmt.Sprintf("expected %s, got %s", expected, p.curToken.Type)
	err := NewParserError(
		"PAR_UNEXPECTED_TOKEN",
		p.curPos(),
		p.curToken,
		message,
		[]lexer.TokenType{expected},
		fix,
	)
	p.errors = append(p.errors, err)
}

func (p *Parser) peekError(t lexer.TokenType) {
	pos := ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File}

	// Check if peekToken is a reserved keyword when we expected IDENT
	if t == lexer.IDENT && p.peekToken.IsKeyword() {
		msg := fmt.Sprintf("expected identifier, got reserved keyword '%s'", p.peekToken.Literal)
		suggestions := []string{
			fmt.Sprintf("Use a different name instead of '%s'", p.peekToken.Literal),
			fmt.Sprintf("'%s' is a reserved keyword in AILANG", p.peekToken.Literal),
		}

		// Add context-specific suggestions
		switch p.peekToken.Type {
		case lexer.EXISTS:
			suggestions = append(suggestions,
				"'exists' is reserved for existential types (future feature)",
				"Try: let found = ... or let doesExist = ...",
			)
		case lexer.FORALL:
			suggestions = append(suggestions,
				"'forall' is reserved for universal type quantification",
			)
		case lexer.IF, lexer.THEN, lexer.ELSE:
			suggestions = append(suggestions,
				"Control flow keywords cannot be used as names",
			)
		case lexer.MATCH, lexer.WITH:
			suggestions = append(suggestions,
				"Pattern matching keywords cannot be used as names",
			)
		}

		err := NewSuggestionError(
			"PAR_RESERVED_KEYWORD",
			pos,
			p.peekToken,
			msg,
			suggestions,
			"https://ailang.sunholo.com/docs/reference/reserved-keywords",
		)
		p.errors = append(p.errors, err)
		return
	}

	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	err := NewParserError(
		"PAR_UNEXPECTED_TOKEN",
		pos,
		p.peekToken,
		msg,
		[]lexer.TokenType{t},
		fmt.Sprintf("Add or correct the %s token", t),
	)
	p.errors = append(p.errors, err)
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := fmt.Sprintf("unexpected token in expression: %s", t)
	fix := "This token cannot start an expression"

	// Detect "const" keyword (JavaScript/TypeScript pattern)
	if t == lexer.IDENT && p.curToken.Literal == "const" {
		err := NewSuggestionError(
			"PAR014",
			p.curPos(),
			p.curToken,
			"'const' keyword doesn't exist in AILANG (JavaScript/TypeScript pattern detected)",
			[]string{
				"Use: let name = value in ...",
				"Note: All bindings in AILANG are immutable by default",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		)
		p.errors = append(p.errors, err)
		return
	}

	// Detect bare assignment (Python pattern): IDENT followed by ASSIGN
	if t == lexer.ASSIGN {
		// Look back at previous token to see if it was IDENT
		// This indicates pattern: identifier = value (without 'let')
		err := NewSuggestionError(
			"PAR015",
			p.curPos(),
			p.curToken,
			"bare assignment not supported (missing 'let' keyword)",
			[]string{
				"Use: let name = value in ...",
				"AILANG requires 'let' keyword for bindings",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		)
		p.errors = append(p.errors, err)
		return
	}

	// Enhanced context-aware hints for common delimiter issues
	if t == lexer.RBRACE || t == lexer.RPAREN || t == lexer.RBRACKET {
		fix = "Check for unmatched delimiters or missing expression"

		// Provide specific hints based on context
		if t == lexer.RBRACE {
			// Check if we're in a deeply nested construct
			delimDepth := len(globalDelimiterTracer.stack)
			if delimDepth > 0 {
				fix += fmt.Sprintf("\n\nContext: Inside nested construct (depth=%d)", delimDepth)
				fix += "\nHint: This may indicate a parser issue with deeply nested match expressions in blocks."
				fix += "\n      Try enabling DEBUG_DELIMITERS=1 to trace delimiter matching."
			} else {
				fix += "\n\nHint: Unexpected closing brace '}'."
				fix += "\n      - Check that all opening braces '{' have matching closing braces"
				fix += "\n      - Verify match expressions and block statements are properly closed"
			}
		} else if t == lexer.RPAREN {
			fix += "\n\nHint: Unexpected closing parenthesis ')'."
			fix += "\n      - Check that all opening parentheses '(' have matching closing ones"
			fix += "\n      - Verify function calls and tuple patterns are properly closed"
		} else if t == lexer.RBRACKET {
			fix += "\n\nHint: Unexpected closing bracket ']'."
			fix += "\n      - Check that all opening brackets '[' have matching closing ones"
			fix += "\n      - Verify list literals are properly closed"
		}

		fix += "\n\nSuggested workaround: Try simplifying nested constructs or using let bindings."
	}

	err := NewParserError(
		"PAR_NO_PREFIX_PARSE",
		p.curPos(),
		p.curToken,
		msg,
		nil,
		fix,
	)
	p.errors = append(p.errors, err)
}

// Errors returns parser errors
func (p *Parser) Errors() []error {
	return p.errors
}
