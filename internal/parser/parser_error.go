package parser

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
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
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	err := NewParserError(
		"PAR_UNEXPECTED_TOKEN",
		ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File},
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
	if t == lexer.RBRACE || t == lexer.RPAREN || t == lexer.RBRACKET {
		fix = "Check for unmatched delimiters or missing expression"
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
