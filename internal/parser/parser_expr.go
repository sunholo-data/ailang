package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseExpression parses an expression with precedence
func (p *Parser) parseExpression(precedence int) ast.Expr {
	p.debugEnter("parseExpression")
	defer p.debugExit("parseExpression")

	// Loop detection (M-PARSER-LOOP): if we see the same position twice, we're stuck
	startPos := p.curPos()
	if p.lastExprPos == startPos && startPos.Line > 0 {
		p.report("PAR_INFINITE_LOOP",
			fmt.Sprintf("parser stuck at %d:%d - unrecognized syntax", startPos.Line, startPos.Column),
			"Check for unimplemented syntax (tests [...], properties [...], etc.)")
		p.nextToken() // Force advance to break out of loop
		return nil
	}
	p.lastExprPos = startPos

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		// M-GAP2: Break the infix loop when LPAREN appears on a new line
		// This prevents `expr\n(next)` from being parsed as `expr(next)`
		// The LPAREN on a new line should start a new statement, not continue the previous one
		if p.peekToken.Type == lexer.LPAREN && p.peekToken.Line > p.curToken.Line {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

// Prefix parse functions

func (p *Parser) parsePrefixExpression() ast.Expr {
	// Special case: BANG followed by LBRACE is an effect annotation, not a prefix operator
	if p.curTokenIs(lexer.BANG) && p.peekTokenIs(lexer.LBRACE) {
		return nil // Not a prefix expression, let caller handle it
	}

	expr := &ast.UnaryOp{
		Op:  p.curToken.Literal,
		Pos: p.curPos(),
	}

	p.nextToken()
	expr.Expr = p.parseExpression(PREFIX)

	return expr
}

func (p *Parser) parseIfExpression() ast.Expr {
	expr := &ast.If{
		Pos: p.curPos(),
	}

	p.nextToken()
	expr.Condition = p.parseExpression(LOWEST)

	p.expectPeek(lexer.THEN)
	p.nextToken()
	expr.Then = p.parseExpression(LOWEST)

	p.expectPeek(lexer.ELSE)
	p.nextToken()
	expr.Else = p.parseExpression(LOWEST)

	return expr
}

func (p *Parser) parseLetExpression() ast.Expr {
	let := &ast.Let{
		Pos: p.curPos(),
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	let.Name = p.curToken.Literal

	// Detect `let rec name = ...` (ML/Haskell style) — AILANG uses `letrec name = ...`
	if let.Name == "rec" && p.peekTokenIs(lexer.IDENT) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR018",
			p.curPos(),
			p.curToken,
			"'let rec' is not valid AILANG syntax (ML/Haskell pattern detected)",
			[]string{
				fmt.Sprintf("Use: letrec %s = ... in ...", p.peekToken.Literal),
				"AILANG uses 'letrec' (one word) for recursive bindings",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
		return nil
	}

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		let.Type = p.parseType()
		if let.Type == nil {
			// If type parsing failed, continue anyway
			let.Type = &ast.SimpleType{Name: "unknown", Pos: p.curPos()}
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return let // Return partial AST
	}
	p.nextToken()
	let.Value = p.parseExpression(LOWEST)
	if let.Value == nil {
		// If value parsing failed, create error node
		let.Value = &ast.Error{Pos: p.curPos()}
	}

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		let.Body = p.parseExpression(LOWEST)
		if let.Body == nil {
			// If body parsing failed, create error node
			let.Body = &ast.Error{Pos: p.curPos()}
		}
	}

	return let
}

func (p *Parser) parseLetRecExpression() ast.Expr {
	letrec := &ast.LetRec{
		Pos: p.curPos(),
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	letrec.Name = p.curToken.Literal

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		letrec.Type = p.parseType()
		if letrec.Type == nil {
			// If type parsing failed, continue anyway
			letrec.Type = &ast.SimpleType{Name: "unknown", Pos: p.curPos()}
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return letrec // Return partial AST
	}
	p.nextToken()
	letrec.Value = p.parseExpression(LOWEST)
	if letrec.Value == nil {
		// If value parsing failed, create error node
		letrec.Value = &ast.Error{Pos: p.curPos()}
	}

	if p.peekTokenIs(lexer.IN) {
		p.nextToken()
		p.nextToken()
		letrec.Body = p.parseExpression(LOWEST)
		if letrec.Body == nil {
			// If body parsing failed, create error node
			letrec.Body = &ast.Error{Pos: p.curPos()}
		}
	}

	return letrec
}

func (p *Parser) parseMatchExpression() ast.Expr {
	match := &ast.Match{
		Pos: p.curPos(),
	}

	p.nextToken()
	match.Expr = p.parseExpression(LOWEST)

	// Detect `match x with p =>` (ML/Haskell/OCaml syntax) — AILANG uses `match x { p => ... }`
	if p.peekTokenIs(lexer.WITH) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR019",
			ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File},
			p.peekToken,
			"'match ... with' is not valid AILANG syntax (ML/Haskell pattern detected)",
			[]string{
				"Use: match expr { pattern => body, ... }",
				"AILANG uses braces for match arms, not 'with'",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
		return nil
	}

	p.expectPeek(lexer.LBRACE)
	p.traceDelimiterOpen(delimCtxMatch)
	p.traceDelimiterToken(lexer.LBRACE, "consume")
	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		c := p.parseCase()
		if c != nil {
			match.Cases = append(match.Cases, c)
		}

		// Move to next token after parsing case
		p.nextToken()

		// Skip comma if present
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	// We should already be at RBRACE
	if !p.curTokenIs(lexer.RBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
		p.traceDelimiterStack()
	} else {
		p.traceDelimiterToken(lexer.RBRACE, "found")
		p.traceDelimiterClose(delimCtxMatch)
	}

	return match
}

func (p *Parser) parseCase() *ast.Case {
	c := &ast.Case{
		Pos: p.curPos(),
	}

	c.Pattern = p.parsePattern()

	// Optional guard
	if p.peekTokenIs(lexer.IF) {
		p.nextToken()
		p.nextToken()
		c.Guard = p.parseExpression(LOWEST)
	}

	// Match arms separate pattern and body with `=>`, not `->` (which is for type
	// signatures / lambdas). A `->` here is a common dialect slip (M-AGENT-ERGONOMICS):
	// emit a precise fix and recover by treating it as the arrow so the rest of the match
	// still parses (one clear error instead of a cascade).
	if p.peekTokenIs(lexer.ARROW) {
		p.errors = append(p.errors, NewSuggestionError(
			"PAR_MATCH_ARROW",
			ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File},
			p.peekToken,
			"match arms use '=>' between pattern and body, not '->'",
			[]string{
				"Replace '->' with '=>': pattern => body",
				"'->' is for type signatures, e.g. (x: int) -> int",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
		p.nextToken() // recover: consume '->' as if it were the arrow
	} else {
		p.expectPeek(lexer.FARROW)
	}
	p.nextToken()

	// Handle match arm bodies:
	// - If LBRACE, use parseBlockOrExpression which handles:
	//   - Record literals: {field: value, ...}
	//   - Record updates: {base | field: value, ...}
	//   - Block expressions: { expr1; expr2; ... }
	// - Otherwise, use parseExpression for simple expressions
	//
	// parseBlockOrExpression includes delimiter tracing needed for nested matches.
	if p.curTokenIs(lexer.LBRACE) {
		p.traceDelimiterOpen(delimCtxCase)
		c.Body = p.parseBlockOrExpression()
		p.traceDelimiterClose(delimCtxCase)
	} else {
		c.Body = p.parseExpression(LOWEST)
	}

	return c
}

func (p *Parser) parseLambda() ast.Expr {
	pos := p.curPos()

	// Expect opening parenthesis
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Parse parameters
	params := p.parseParams()

	// Check which syntax we're using:
	// - func(x) -> type { body }  (new FuncLit syntax)
	// - func(x) => body           (old Lambda syntax)

	if p.peekTokenIs(lexer.ARROW) {
		// New FuncLit syntax: func(x: int) -> int { body }
		return p.parseFuncLitWithParams(pos, params)
	} else if p.peekTokenIs(lexer.FARROW) {
		// Old Lambda syntax: func(x) => body
		lambda := &ast.Lambda{
			Pos:    pos,
			Params: params,
		}
		p.nextToken() // consume =>
		p.nextToken()
		lambda.Body = p.parseExpression(LOWEST)
		return lambda
	} else {
		p.errors = append(p.errors, fmt.Errorf("expected '->' or '=>' after function parameters at %s", p.peekToken.Position()))
		return nil
	}
}

// parseFuncLitWithParams parses the rest of a function literal after params have been parsed
// Syntax: (already parsed: func(params)) -> returnType ! {effects} { body }
func (p *Parser) parseFuncLitWithParams(pos ast.Pos, params []*ast.Param) ast.Expr {
	funcLit := &ast.FuncLit{
		Pos:    pos,
		Params: params,
	}

	// Consume '->'
	if !p.expectPeek(lexer.ARROW) {
		return nil
	}
	p.nextToken() // move to return type

	// Parse return type
	funcLit.ReturnType = p.parseType()

	// Parse optional effect annotation: func() -> int ! {IO}
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		funcLit.Effects = p.parseEffectAnnotation()
	}

	// Expect body in braces: { expr }
	if !p.expectPeek(lexer.LBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected '{' for function body at %s", p.peekToken.Position()))
		return nil
	}

	// Parse body as a block or expression
	funcLit.Body = p.parseBlockOrExpression()

	return funcLit
}

// parseBlockOrExpression parses either a block { e1; e2; e3 }, record literal, or record update
// This is called when we're at the opening LBRACE
func (p *Parser) parseBlockOrExpression() ast.Expr {
	// We're at LBRACE
	startPos := p.curPos()
	p.traceDelimiterOpen(delimCtxBlock)
	p.traceDelimiterToken(lexer.LBRACE, "consume")
	p.nextToken() // consume LBRACE

	// Check for empty block: {}
	if p.curTokenIs(lexer.RBRACE) {
		// Empty block returns unit
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Check for record literal: {field: value, ...} or {"quoted-key": value, ...}
	// This is detected by IDENT or STRING followed by COLON
	if (p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING)) && p.peekTokenIs(lexer.COLON) {
		p.traceDelimiterClose(delimCtxBlock) // Close the block trace since we're switching to record
		return p.parseRecordLiteralContent(startPos)
	}

	// Check for record update: {base | field: value, ...}
	// This is detected by IDENT followed by PIPE
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.PIPE) {
		p.traceDelimiterClose(delimCtxBlock) // Close the block trace since we're switching to record
		return p.parseRecordUpdateContent(startPos)
	}

	// Parse expressions separated by semicolons (block expression)
	exprs := []ast.Expr{}
	exprs = append(exprs, p.parseExpression(LOWEST))

	// Keep parsing while we see semicolons
	for p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to SEMICOLON

		// Check for trailing semicolon (next token is RBRACE)
		// Keep cursor at semicolon so peek is RBRACE for expectPeek below
		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		p.nextToken() // move past SEMICOLON

		exprs = append(exprs, p.parseExpression(LOWEST))
	}

	// M-AILANG-ERROR-QUALITY (PAR020): if the next token instead begins a new
	// statement (let / if / match / identifier), the block is missing a ';'
	// between statements — the mirror of PAR017's "extra ';' in =-body". This is
	// the #1 unactionable thrash-causer on small models: config_file_parser burned
	// 66 agent turns on a bare "expected }, got if". Give the concrete fix instead.
	if !p.peekTokenIs(lexer.RBRACE) && p.peekStartsBlockStatement() {
		p.errors = append(p.errors, p.missingBlockSemicolonError())
		p.traceDelimiterStack()
		return nil
	}

	// Expect closing brace
	if !p.expectPeek(lexer.RBRACE) {
		p.errors = append(p.errors, fmt.Errorf("expected '}' to close function body at %s", p.peekToken.Position()))
		p.traceDelimiterStack()
		return nil
	}

	p.traceDelimiterToken(lexer.RBRACE, "found")
	p.traceDelimiterClose(delimCtxBlock)

	// If single expression, return it directly (not as block)
	if len(exprs) == 1 {
		return exprs[0]
	}

	// Multiple expressions: return as block
	return &ast.Block{
		Exprs: exprs,
		Pos:   startPos,
	}
}

// peekStartsBlockStatement reports whether the peek token unambiguously begins a
// new block statement (let / letrec / if / match / identifier). Used to detect a
// missing ';' separator between block statements (PAR020). Kept to high-signal
// statement-starters so the hint is precise — literals/operators are excluded
// because they're more likely a mid-expression parse error than a dropped ';'.
func (p *Parser) peekStartsBlockStatement() bool {
	switch p.peekToken.Type {
	case lexer.LET, lexer.LETREC, lexer.IF, lexer.MATCH, lexer.IDENT:
		return true
	default:
		return false
	}
}

// missingBlockSemicolonError builds the PAR020 actionable error for a missing
// ';' between block statements (the mirror of PAR017's "extra ';' in =-body").
// Used by both the block-expression parser and the function-declaration
// block-body parser. Points at the offending (peek) token.
func (p *Parser) missingBlockSemicolonError() *ParserError {
	return NewSuggestionError(
		"PAR020",
		ast.Pos{Line: p.peekToken.Line, Column: p.peekToken.Column, File: p.peekToken.File},
		p.peekToken,
		fmt.Sprintf("missing ';' between block statements (found `%s` where `;` or `}` was expected)", p.peekToken.Literal),
		[]string{
			"Statements inside a `{ }` block are separated by `;`:",
			"    { let x = e1; let y = e2; result }",
			"Add a `;` after the previous statement, before this one.",
			"The block's LAST expression is the return value — no `;` after it.",
		},
		"https://ailang.sunholo.com/docs/reference/language-syntax",
	)
}

// parseRecordLiteralContent / parseRecordUpdateContent moved to parser_record.go
// (M-RELEASE-GATE follow-up: keep parser_expr.go under the 800-line limit).

func (p *Parser) parsePureLambda() ast.Expr {
	// We're already at 'func' token after 'pure'
	lambda := p.parseLambda().(*ast.Lambda)
	// Mark as pure somehow
	return lambda
}

// parseBackslashLambda parses lambda expressions with \x. syntax
func (p *Parser) parseBackslashLambda() ast.Expr {
	lambda := &ast.Lambda{
		Pos: p.curPos(),
	}

	// Parse parameters - support curried sugar \x y z. body
	var params []*ast.Param

	// Keep consuming identifiers until we hit DOT
	for {
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		param := &ast.Param{
			Name: p.curToken.Literal,
			Pos:  p.curPos(),
			// Type will be inferred
		}
		params = append(params, param)

		// Check if next token is DOT (end of params) or another IDENT (more params)
		if p.peekTokenIs(lexer.DOT) {
			break
		} else if p.peekTokenIs(lexer.ARROW) {
			// \x -> body is wrong; AILANG uses \x. body (dot, not arrow)
			p.nextToken() // consume -> to prevent cascading PAR_NO_PREFIX_PARSE
			p.errors = append(p.errors, fmt.Errorf("lambda body separator is '.' not '->' at %s\n\t\twrite: \\%s. <body>", p.curToken.Position(), params[len(params)-1].Name))
			return nil
		} else if !p.peekTokenIs(lexer.IDENT) {
			p.errors = append(p.errors, fmt.Errorf("expected '.' after lambda parameter at %s", p.peekToken.Position()))
			return nil
		}
	}

	// Expect DOT
	if !p.expectPeek(lexer.DOT) {
		return nil
	}

	// Parse body with LOWEST precedence to capture entire expression
	p.nextToken()
	lambda.Body = p.parseExpression(LOWEST)

	// Parse optional effect annotation: \x. body ! {IO}
	if p.peekTokenIs(lexer.BANG) {
		p.nextToken() // move to BANG
		lambda.Effects = p.parseEffectAnnotation()
	}

	// M-GAP2: Keep multi-param lambdas as multi-param (NOT curried)
	// \x y. body should be a single lambda with params=[x,y], NOT nested lambdas
	// This allows \acc x. acc + x to unify with (b, a) -> b for foldl
	if len(params) == 0 {
		p.errors = append(p.errors, fmt.Errorf("lambda requires at least one parameter at %s", lambda.Pos.String()))
		return nil
	}
	lambda.Params = params
	return lambda
}

// Infix parse functions

func (p *Parser) parseInfixExpression(left ast.Expr) ast.Expr {
	expr := &ast.BinaryOp{
		Left: left,
		Op:   p.curToken.Literal,
		Pos:  p.curPos(),
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

// S-CONS: Parse infix cons operator :: (right-associative)
// Desugars x :: xs to ::(x, xs) - a constructor call
func (p *Parser) parseConsExpression(left ast.Expr) ast.Expr {
	consPos := p.curPos()

	// Check if strict syntax mode is enabled
	if p.strictSyntaxMode {
		p.reportSugarError("CONS", "x :: xs", "::(x, xs)")
		// Return a placeholder to avoid cascading errors
		return &ast.FuncCall{
			Func: &ast.Identifier{Name: "::", Pos: consPos},
			Args: []ast.Expr{left},
			Pos:  consPos,
		}
	}

	// Sugar is allowed - mark that it was used
	p.sugarUsed = true

	// Right-associative: parse right side with lower precedence
	// This makes a :: b :: c parse as a :: (b :: c)
	p.nextToken()
	right := p.parseExpression(CONS - 1)

	// Desugar to constructor call: ::(left, right)
	return &ast.FuncCall{
		Func: &ast.Identifier{Name: "::", Pos: consPos},
		Args: []ast.Expr{left, right},
		Pos:  consPos,
	}
}

func (p *Parser) parseCallExpression(fn ast.Expr) ast.Expr {
	call := &ast.FuncCall{
		Func: fn,
		Pos:  p.curPos(),
	}

	call.Args = p.parseCallArguments()
	return call
}

// parseZeroArgCall handles S-CALL0 sugar in expression context: f() → f(())
// This is called when the lexer creates a UNIT token for () without spaces
func (p *Parser) parseZeroArgCall(fn ast.Expr) ast.Expr {
	// Check strict mode
	if p.strictSyntaxMode {
		p.reportSugarError("CALL0", "f()", "f (())")
		// Return the function expression unchanged
		return fn
	}

	// Mark that sugar was used
	p.sugarUsed = true

	// Create call with unit argument
	// Get position from the function expression
	var fnPos ast.Pos
	switch f := fn.(type) {
	case *ast.Identifier:
		fnPos = f.Pos
	case *ast.FuncCall:
		fnPos = f.Pos
	default:
		fnPos = p.curPos()
	}

	call := &ast.FuncCall{
		Func: fn,
		Args: []ast.Expr{
			&ast.Literal{
				Kind:  ast.UnitLit,
				Value: nil,
				Pos:   p.curPos(),
			},
		},
		Pos: fnPos,
	}

	return call
}

func (p *Parser) parseCallArguments() []ast.Expr {
	args := []ast.Expr{}

	// S-CALL0: Check for zero-arg call sugar f()
	// NOTE: Currently only works with space: f ()
	// Without space f() requires statement-level parsing changes (TODO)
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume RPAREN

		// Check if strict syntax mode is enabled
		if p.strictSyntaxMode {
			p.reportSugarError("CALL0", "f()", "f ()")
			return args // Return empty args to avoid cascading errors
		}

		// Sugar is allowed - desugar f() to f(())
		// Mark that sugar was used (for REPL feedback)
		p.sugarUsed = true

		// Return unit literal as single argument
		unitLit := &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   p.curPos(),
		}
		return []ast.Expr{unitLit}
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	p.expectPeek(lexer.RPAREN)
	return args
}

func (p *Parser) parseRecordAccess(record ast.Expr) ast.Expr {
	access := &ast.RecordAccess{
		Record: record,
		Pos:    p.curPos(),
	}

	p.expectPeek(lexer.IDENT)
	access.Field = p.curToken.Literal

	return access
}

func (p *Parser) parseSendExpression(channel ast.Expr) ast.Expr {
	send := &ast.Send{
		Channel: channel,
		Pos:     p.curPos(),
	}

	p.nextToken()
	send.Value = p.parseExpression(LOWEST)

	return send
}

// Helper parsing functions

func (p *Parser) parseParams() []*ast.Param {
	params := []*ast.Param{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	param := &ast.Param{
		Pos: p.curPos(),
	}

	if p.curTokenIs(lexer.IDENT) {
		param.Name = p.curToken.Literal

		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			param.Type = p.parseType()
		}
	}

	params = append(params, param)

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()

		param := &ast.Param{
			Pos: p.curPos(),
		}

		if p.curTokenIs(lexer.IDENT) {
			param.Name = p.curToken.Literal

			if p.peekTokenIs(lexer.COLON) {
				p.nextToken()
				p.nextToken()
				param.Type = p.parseType()
			}
		}

		params = append(params, param)
	}

	p.expectPeek(lexer.RPAREN)
	return params
}
