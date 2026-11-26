package parser

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// isBareLet reports whether expr is a `let`, `let (pat)`/`let {pat}`, or
// `letrec` binding without an `in` body — a statement-style binding that must
// be followed by another statement/expression to produce a value.
func isBareLet(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Let:
		return e.Body == nil
	case *ast.LetPattern:
		return e.Body == nil
	case *ast.LetRec:
		return e.Body == nil
	default:
		return false
	}
}

// parseIfThenBranch parses the `then` branch of an `if` expression.
//
// Because `then` is always terminated by the reserved keyword `else`, an
// unbraced multi-statement sequence (semicolon-separated or newline-separated)
// is completely unambiguous:
//
//	if cond then
//	  let x = e1
//	  x + 1
//	else
//	  0
//
// Single-expression branches return the expression directly (identical to
// pre-M-AI-PREFERRED-V2 AST); multi-statement branches return an *ast.Block
// which `ailang fmt` normalizes to `{ ... }`.
func (p *Parser) parseIfThenBranch() ast.Expr {
	startPos := p.curPos()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	exprs := []ast.Expr{first}

	for p.peekTokenIs(lexer.SEMICOLON) || p.peekStartsNewlineBlockStatement() {
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken() // move to SEMICOLON
			if p.peekTokenIs(lexer.ELSE) || p.peekTokenIs(lexer.EOF) {
				break // trailing ';' before else
			}
			p.nextToken() // move past SEMICOLON to next statement
		} else {
			p.nextToken() // newline separator — advance to next statement
		}
		next := p.parseExpression(LOWEST)
		if next == nil {
			break
		}
		exprs = append(exprs, next)
	}

	if last := exprs[len(exprs)-1]; isBareLet(last) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR_IF_BRANCH_BLOCK",
			last.Position(),
			p.curToken,
			"bare 'let' binding in 'then' branch must be followed by a body expression",
			[]string{
				"Add a body expression after 'let': if cond then { let x = ...; result } else ...",
				"Or chain with 'in': if cond then let x = ... in result else ...",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
	}

	if len(exprs) == 1 {
		return exprs[0]
	}
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}

// parseIfElseBranch parses the `else` branch of an `if` expression.
//
// Unlike `then`, `else` has no closing keyword, so statements after a non-let
// expression belong to any enclosing block (`let a = if c then 1 else 2\n a + 1`
// or `let a = if c then 1 else 2; a + 1`). However, when `else` begins with a
// bare `let`/`letrec` (without `in`), the binding unambiguously requires a
// following expression: we continue consuming `;`- or newline-separated
// statements ONLY while the previous statement is a bare `let`, stopping as
// soon as the final value expression is parsed.
func (p *Parser) parseIfElseBranch() ast.Expr {
	startPos := p.curPos()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	if !isBareLet(first) {
		return first
	}

	exprs := []ast.Expr{first}
	for isBareLet(exprs[len(exprs)-1]) {
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken() // move to SEMICOLON
			if p.peekIsElseChainStop() {
				break
			}
			p.nextToken() // move past SEMICOLON to next statement
		} else if p.peekStartsNewlineBlockStatement() {
			p.nextToken() // newline separator — advance to next statement
		} else {
			break
		}
		next := p.parseExpression(LOWEST)
		if next == nil {
			break
		}
		exprs = append(exprs, next)
	}

	if last := exprs[len(exprs)-1]; isBareLet(last) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR_IF_BRANCH_BLOCK",
			last.Position(),
			p.curToken,
			"bare 'let' binding in 'else' branch must be followed by a body expression",
			[]string{
				"Add a body expression after 'let': ... else { let y = ...; result }",
				"Or chain with 'in': ... else let y = ... in result",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
	}

	if len(exprs) == 1 {
		return exprs[0]
	}
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}

// peekIsElseChainStop reports whether the peek token is an enclosing delimiter
// that cannot begin a statement after a `;` in an `else` bare-let chain.
func (p *Parser) peekIsElseChainStop() bool {
	switch p.peekToken.Type {
	case lexer.RBRACE, lexer.COMMA, lexer.ELSE, lexer.RPAREN, lexer.RBRACKET, lexer.EOF:
		return true
	default:
		return false
	}
}

// parseMatchArmBody parses an unbraced match arm body (`pat => ...`).
//
// Two unambiguous multi-statement forms are accepted and returned as an
// *ast.Block (which `ailang fmt` normalizes to `{ ... }`):
//  1. Semicolon-separated sequences (`pat => e1; e2`): `;` never separates
//     match arms (arms use `,` or newline), so `;` is always an intra-arm
//     statement separator.
//  2. Newline-separated bare `let` chains (`pat => let x = e1\n let y = e2\n y`):
//     continuing across a newline ONLY after a bare `let` preserves comma-less
//     match arms (`Some(v) => v\n None => 0`) with zero ambiguity.
func (p *Parser) parseMatchArmBody() ast.Expr {
	startPos := p.curPos()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	exprs := []ast.Expr{first}

	for {
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken() // move to SEMICOLON
			// Trailing ';' before ',' or '}': stop, leaving ',' or '}' in peek.
			if p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.RBRACE) || p.peekTokenIs(lexer.EOF) {
				break
			}
			p.nextToken() // move past SEMICOLON to next statement
		} else if isBareLet(exprs[len(exprs)-1]) && p.peekStartsNewlineBlockStatement() {
			p.nextToken() // newline separator after bare let — advance to next statement
		} else {
			break
		}
		next := p.parseExpression(LOWEST)
		if next == nil {
			break
		}
		exprs = append(exprs, next)
	}

	if last := exprs[len(exprs)-1]; isBareLet(last) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR_MATCH_ARM_BLOCK",
			last.Position(),
			p.curToken,
			"bare 'let' binding in match arm must be followed by a body expression",
			[]string{
				"Add a body expression after 'let': pattern => { let x = ...; result }",
				"Or use an expression body with 'in': pattern => let x = ... in result",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
	}

	if len(exprs) == 1 {
		return exprs[0]
	}
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}
