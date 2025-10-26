package parser

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/lexer"
)

// debugParserEnabled returns true if DEBUG_PARSER environment variable is set to "1"
func debugParserEnabled() bool {
	return os.Getenv("DEBUG_PARSER") == "1"
}

// debugEnter logs entry to a parser function with current token state.
// Only logs if DEBUG_PARSER=1 is set.
//
// Example output:
//
//	[ENTER parseExpression] cur=INT(42) peek=PLUS(+)
func (p *Parser) debugEnter(funcName string) {
	if !debugParserEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "[ENTER %s] cur=%s peek=%s\n",
		funcName,
		formatToken(p.curToken),
		formatToken(p.peekToken),
	)
}

// debugExit logs exit from a parser function with current token state.
// Only logs if DEBUG_PARSER=1 is set.
//
// Example output:
//
//	[EXIT parseExpression] cur=INT(42) peek=PLUS(+)
func (p *Parser) debugExit(funcName string) {
	if !debugParserEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "[EXIT %s] cur=%s peek=%s\n",
		funcName,
		formatToken(p.curToken),
		formatToken(p.peekToken),
	)
}

// debugLog logs a custom message during parsing.
// Only logs if DEBUG_PARSER=1 is set.
//
// Example output:
//
//	[DEBUG parseExpression] Found binary operator: +
func (p *Parser) debugLog(funcName, message string) {
	if !debugParserEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", funcName, message)
}

// formatToken formats a token for debug output.
// Shows token type and literal value.
//
// Examples:
//   - INT(42)
//   - PLUS(+)
//   - IDENT(factorial)
//   - EOF
func formatToken(tok lexer.Token) string {
	if tok.Type == lexer.EOF {
		return "EOF"
	}

	// Empty literal means the token type is enough context
	if tok.Literal == "" {
		return tok.Type.String()
	}

	// Show type(literal) for all tokens with literals
	return fmt.Sprintf("%s(%s)", tok.Type, tok.Literal)
}

// Example usage in parser functions:
//
// func (p *Parser) parseExpression(precedence int) ast.Expr {
//     p.debugEnter("parseExpression")
//     defer p.debugExit("parseExpression")
//
//     // ... parsing logic ...
//
//     if someCondition {
//         p.debugLog("parseExpression", "Found special case")
//     }
//
//     return expr
// }
