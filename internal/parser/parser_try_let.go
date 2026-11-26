package parser

import (
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

var (
	tryBlockDepthMu sync.Mutex
	tryBlockDepths  = map[*Parser]int{}
)

func (p *Parser) enterTryBlock() func() {
	tryBlockDepthMu.Lock()
	tryBlockDepths[p]++
	tryBlockDepthMu.Unlock()
	return func() {
		tryBlockDepthMu.Lock()
		if tryBlockDepths[p] <= 1 {
			delete(tryBlockDepths, p)
		} else {
			tryBlockDepths[p]--
		}
		tryBlockDepthMu.Unlock()
	}
}

func (p *Parser) isModuleTopLevelLet() bool {
	if !p.seenModule || p.curDecl != "" {
		return false
	}
	tryBlockDepthMu.Lock()
	depth := tryBlockDepths[p]
	tryBlockDepthMu.Unlock()
	return depth == 0
}

// M-AI-GO-TO-LANGUAGE M2 (D2=a): `let`-bound `?` error/option propagation.
//
// Postfix `?` is permitted ONLY directly on the RHS of a `let` binding
// (`let x = expr?` or `let (a, b) = expr?` / `let {a, b} = expr?`), either in
// block statement form (`{ let x = expr?; ... }` / newline-separated) or `in`
// expression form (`let x = expr? in body`).
//
// Typed holes `_?` (consumed inside parseIdentifier) and `?name` / bare `?`
// (consumed inside parseHoleExpression) are unaffected because their QUESTION
// token is consumed before parseExpressionCore returns. Requiring the postfix
// `?` to sit on the same line as the end of the RHS expression also ensures a
// newline-separated named hole (`let y = 1\n ?todo`) is never eaten as a try.

const CodeTryLetOnly = "PAR_TRY_LET_ONLY"

// peekSameLineQuestion reports whether peekToken is a `?` on the same source
// line as curToken (a postfix `?` attached to the just-parsed expression).
func (p *Parser) peekSameLineQuestion() bool {
	return p.peekTokenIs(lexer.QUESTION) && p.peekToken.Line == p.curToken.Line
}

// parseLetRHS parses the value expression of a `let` or `let <pattern>` binding
// and consumes an optional trailing same-line `?` (`IsTry = true`).
func (p *Parser) parseLetRHS() (ast.Expr, bool, ast.Pos) {
	val := p.parseExpressionCore(LOWEST)
	if val == nil {
		return &ast.Error{Pos: p.curPos()}, false, ast.Pos{}
	}
	if !p.peekSameLineQuestion() {
		return val, false, ast.Pos{}
	}
	p.nextToken() // move onto QUESTION
	tryPos := p.curPos()
	// Reject chained `expr??` with a concrete two-binding suggestion.
	for p.peekSameLineQuestion() {
		p.reportTryLetOnly(val, "chained '??' is not allowed on a single 'let' binding",
			"Split nested Results/Options into two 'let' bindings: let tmp = expr?; let x = tmp?")
		p.nextToken()
	}
	return val, true, tryPos
}

// parseLetPatternWithTry parses irrefutable tuple/record destructuring `let`
// bindings with optional trailing `?`: `let (a, b) = expr? [in body]`.
func (p *Parser) parseLetPatternWithTry(startPos ast.Pos) ast.Expr {
	p.nextToken() // move onto LPAREN or LBRACE
	patPos := p.curPos()
	pat := p.parsePattern()
	if !isIrrefutableLetPattern(pat) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR_LET_PATTERN",
			patPos,
			p.curToken,
			"only irrefutable tuple/record patterns like 'let (a, b) = ...' or 'let {a, b} = ...' are allowed in 'let' bindings",
			[]string{
				"Use variables, '_', and nested tuples/records: let (a, (b, _)) = expr  or  let {name, ...} = rec",
				"For refutable patterns (constructors, literals, lists), use 'match expr { ... }'",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
		return nil
	}

	lp := &ast.LetPattern{
		Pattern: pat,
		Pos:     startPos,
	}

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		lp.Type = p.parseType()
		if lp.Type == nil {
			lp.Type = &ast.SimpleType{Name: "unknown", Pos: p.curPos()}
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return lp
	}
	p.nextToken()
	lp.Value, lp.IsTry, lp.TryPos = p.parseLetRHS()

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		lp.Body = p.parseExpression(LOWEST)
		if lp.Body == nil {
			lp.Body = &ast.Error{Pos: p.curPos()}
		}
	}

	return lp
}

// checkMidExprTrySuffix detects `expr? <infix>` (such as `g()? + 1` or
// `g()?.field`) inside parseInfixTail, emitting PAR_TRY_LET_ONLY and consuming
// the `?` so the surrounding expression continues parsing cleanly.
func (p *Parser) checkMidExprTrySuffix(leftExp ast.Expr) {
	if leftExp == nil || !p.peekSameLineQuestion() {
		return
	}
	if p.infixParseFns[p.peek2Token.Type] == nil {
		return
	}
	// Mirror M-GAP2: `(` or `()` on a NEXT line begins a new block statement
	// (`{ let x = f()?\n (x, 1) }`), not a call on `f()?`.
	if (p.peek2Token.Type == lexer.LPAREN || p.peek2Token.Type == lexer.UNIT) && p.peek2Token.Line > p.peekToken.Line {
		return
	}
	for p.peekSameLineQuestion() {
		p.reportDisallowedTry(leftExp)
		p.nextToken() // consume '?' to recover into the infix operator
	}
}

// rejectDisallowedTrySuffix rejects a trailing same-line `?` after any
// non-`let`-RHS expression (`f(g()?)`, `1 + g()?`, `{ let x = f()?; g(x)? }`).
// Only `PIPE` (`r |> decode?` inside `parseLetRHS`) leaves a trailing `?` for
// an enclosing `parseLetRHS`; if the pipeline itself sits inside a non-let
// context (`f(r |> decode?)`), the outer `LOWEST` frame catches and reports it.
func (p *Parser) rejectDisallowedTrySuffix(expr ast.Expr, precedence int) ast.Expr {
	if expr == nil || precedence == PIPE {
		return expr
	}
	for p.peekSameLineQuestion() {
		p.reportDisallowedTry(expr)
		p.nextToken() // consume '?' so enclosing delimiters (')', ',', '}', ';') stay aligned
	}
	return expr
}

func (p *Parser) reportDisallowedTry(expr ast.Expr) {
	snippet := tryExprSnippet(expr)
	p.reportTryLetOnly(
		expr,
		fmt.Sprintf("'?' error propagation is only allowed on 'let' bindings, not on sub-expression '%s?'", snippet),
		fmt.Sprintf("Hoist into a 'let' binding first: let x = %s?; ...", snippet),
	)
}

func (p *Parser) reportTryLetOnly(_ ast.Expr, msg string, hoistHint string) {
	qPos := ast.Pos{
		Line:   p.peekToken.Line,
		Column: p.peekToken.Column,
		File:   p.peekToken.File,
	}
	p.errors = append(p.errors, NewSuggestionError(
		CodeTryLetOnly,
		qPos,
		p.peekToken,
		msg,
		[]string{
			hoistHint,
			"Example: { let x = g()?; f(x) }  or  let x = g()? in f(x)",
		},
		"https://ailang.sunholo.com/docs/reference/language-syntax",
	))
}

// tryExprSnippet renders a short human-readable spelling for PAR_TRY_LET_ONLY hints.
func tryExprSnippet(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Name
	case *ast.FuncCall:
		if fn, ok := e.Func.(*ast.Identifier); ok {
			if len(e.Args) == 0 {
				return fn.Name + "()"
			}
			if len(e.Args) == 1 {
				if lit, isUnit := e.Args[0].(*ast.Literal); isUnit && lit.Kind == ast.UnitLit {
					return fn.Name + "()"
				}
			}
			return fn.Name + "(...)"
		}
		return "expr"
	case *ast.RecordAccess:
		return tryExprSnippet(e.Record) + "." + e.Field
	default:
		return "expr"
	}
}
