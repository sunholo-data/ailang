package parser

import (
	"fmt"
	"strconv"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// Prefix parse functions for literals and identifiers

func (p *Parser) parseIdentifier() ast.Expr {
	return &ast.Identifier{
		Name: p.curToken.Literal,
		Pos:  p.curPos(),
	}
}

func (p *Parser) parseIntegerLiteral() ast.Expr {
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}

	return &ast.Literal{
		Kind:  ast.IntLit,
		Value: value,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseFloatLiteral() ast.Expr {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("could not parse %q as float", p.curToken.Literal))
		return nil
	}

	return &ast.Literal{
		Kind:  ast.FloatLit,
		Value: value,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseStringLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.StringLit,
		Value: p.curToken.Literal,
		Pos:   p.curPos(),
	}
}

// parseInterpolatedString desugars an interpolated string literal
// (a token sequence STRING_PART, INTERP_START, expr, INTERP_END, STRING_PART, ...)
// into a chain of `concat_String` calls with each interpolated expression wrapped
// in `show(...)`. Empty STRING_PART segments at the head or tail are elided so
// `"${x}"` desugars to `show(x)` instead of `concat_String(concat_String("", show(x)), "")`.
//
// M2_PARSER_TYPECHECK_INTERP: Phase 1 of M-CONCAT-DISAMBIG. Evaluator and
// codegen see only existing nodes (FuncCall, Literal), so no runtime changes are required.
func (p *Parser) parseInterpolatedString() ast.Expr {
	startPos := p.curPos()

	// Collect the ordered segments: alternating string literal parts and
	// interpolated expressions (wrapped in show()).
	parts := []ast.Expr{}

	// Current token is the first STRING_PART.
	if lit := p.curToken.Literal; lit != "" {
		parts = append(parts, &ast.Literal{
			Kind:  ast.StringLit,
			Value: lit,
			Pos:   p.curPos(),
		})
	}

	for p.peekTokenIs(lexer.INTERP_START) {
		p.nextToken() // consume STRING_PART → INTERP_START
		p.nextToken() // move past INTERP_START to start of expression

		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}

		// Wrap in show(expr). The Show class dispatches to the right instance
		// at elaboration time (show_Int, show_String≡id, etc.), so string-typed
		// expressions incur no runtime cost beyond the dictionary lookup.
		showCall := &ast.FuncCall{
			Func: &ast.Identifier{Name: "show", Pos: expr.Position()},
			Args: []ast.Expr{expr},
			Pos:  expr.Position(),
		}
		parts = append(parts, showCall)

		if !p.expectPeek(lexer.INTERP_END) {
			return nil
		}
		if !p.expectPeek(lexer.STRING_PART) {
			return nil
		}
		if lit := p.curToken.Literal; lit != "" {
			parts = append(parts, &ast.Literal{
				Kind:  ast.StringLit,
				Value: lit,
				Pos:   p.curPos(),
			})
		}
	}

	// Fold into a left-associative chain of concat_String(a, b).
	// Invariant: parts has at least one element (a `${x}` alone still yields show(x)).
	if len(parts) == 0 {
		// Pathological case: both surrounding STRING_PARTs were empty and no
		// interpolation occurred — emit an empty string literal.
		return &ast.Literal{Kind: ast.StringLit, Value: "", Pos: startPos}
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result = &ast.FuncCall{
			Func: &ast.Identifier{Name: "concat_String", Pos: result.Position()},
			Args: []ast.Expr{result, parts[i]},
			Pos:  result.Position(),
		}
	}
	return result
}

func (p *Parser) parseCharLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.StringLit, // Treat chars as single-char strings for now
		Value: p.curToken.Literal,
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.BoolLit,
		Value: p.curTokenIs(lexer.TRUE),
		Pos:   p.curPos(),
	}
}

func (p *Parser) parseUnitLiteral() ast.Expr {
	return &ast.Literal{
		Kind:  ast.UnitLit,
		Value: nil,
		Pos:   p.curPos(),
	}
}

// parseGroupedExpression parses grouped expressions and tuples
// EBNF:
//
//	tuple_expr := "(" expr "," expr ("," expr)* ","? ")"
//	grouped    := "(" expr ")"
//
// Disambiguation: A comma is required to form a tuple. (e) is grouping, (e,) is a tuple.
func (p *Parser) parseGroupedExpression() ast.Expr {
	startPos := p.curPos()
	p.nextToken() // consume LPAREN

	// Handle empty tuple/unit: ()
	if p.curTokenIs(lexer.RPAREN) {
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Parse first expression
	expr := p.parseExpression(LOWEST)

	// After parsing expression, we're at the last token of that expression
	// Need to advance to see what comes next
	if !p.peekTokenIs(lexer.COMMA) {
		// Just a grouped expression - no comma
		if !p.expectPeek(lexer.RPAREN) {
			p.reportExpected(lexer.RPAREN, "Add ')' to close grouped expression")
		}
		return expr
	}

	// It's a tuple - comma is required
	elements := []ast.Expr{expr}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // move to COMMA
		p.nextToken() // move past COMMA to next element

		// Check for trailing comma
		if p.curTokenIs(lexer.RPAREN) {
			return &ast.Tuple{
				Elements: elements,
				Pos:      startPos,
			}
		}

		elem := p.parseExpression(LOWEST)
		elements = append(elements, elem)
	}

	// Expect closing paren
	if !p.expectPeek(lexer.RPAREN) {
		p.reportExpected(lexer.RPAREN, "Add ')' to close tuple")
	}

	return &ast.Tuple{
		Elements: elements,
		Pos:      startPos,
	}
}

func (p *Parser) parseListLiteral() ast.Expr {
	list := &ast.List{
		Pos: p.curPos(),
	}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		list.Elements = append(list.Elements, p.parseExpression(LOWEST))

		if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken()
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.RBRACKET) {
		p.expectPeek(lexer.RBRACKET)
	}

	return list
}

// parseArrayLiteral parses array literals: #[1, 2, 3]
func (p *Parser) parseArrayLiteral() ast.Expr {
	arr := &ast.Array{
		Pos: p.curPos(),
	}

	// Current token is HASH, expect LBRACKET next
	if !p.expectPeek(lexer.LBRACKET) {
		return nil
	}

	p.nextToken() // move past LBRACKET

	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))

		if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken()
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.RBRACKET) {
		p.expectPeek(lexer.RBRACKET)
	}

	return arr
}

func (p *Parser) parseRecordLiteral() ast.Expr {
	startPos := p.curPos()
	p.nextToken() // move past LBRACE

	// Empty block: {}
	if p.curTokenIs(lexer.RBRACE) {
		return &ast.Block{
			Exprs: []ast.Expr{},
			Pos:   startPos,
		}
	}

	// Detect {field = value} pattern (ML/Haskell/Python dict syntax)
	// and suggest using ':' instead of '='
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.ASSIGN) {
		fieldName := p.curToken.Literal
		p.errors = append(p.errors, NewSuggestionError(
			"PAR016",
			p.curPos(),
			p.curToken,
			fmt.Sprintf("record field '%s' uses '=' but AILANG records use ':'", fieldName),
			[]string{
				fmt.Sprintf("Use: {%s: value} instead of {%s = value}", fieldName, fieldName),
				"AILANG record syntax: {field1: value1, field2: value2}",
			},
			"https://ailang.sunholo.com/docs/reference/language-syntax",
		))
		return nil
	}

	// Peek ahead to determine if this is a record literal, record update, or a block
	// Record literals: IDENT COLON ...
	// Record updates: IDENT PIPE ...
	// Blocks: anything else
	isRecordLiteral := (p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING)) && p.peekTokenIs(lexer.COLON)
	isRecordUpdate := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.PIPE)

	if isRecordUpdate {
		// Record update: {base | field: value, ...}
		base := p.parseExpression(LOWEST)

		if !p.expectPeek(lexer.PIPE) {
			return nil
		}
		p.nextToken() // move past PIPE

		update := &ast.RecordUpdate{
			Base: base,
			Pos:  startPos,
		}

		// Parse updated fields
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			field := &ast.Field{
				Pos: p.curPos(),
			}

			if !p.curTokenIs(lexer.IDENT) {
				p.errors = append(p.errors, fmt.Errorf("expected field name in record update, got %s", p.curToken.Type))
				return nil
			}

			field.Name = p.curToken.Literal

			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()

			field.Value = p.parseExpression(LOWEST)
			update.Fields = append(update.Fields, field)

			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken()
				break
			}

			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken()
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected } in record update, got %s", p.curToken.Type))
			return nil
		}

		return update
	} else if isRecordLiteral {
		// Regular record literal: {field: value, ...}
		record := &ast.Record{
			Pos: startPos,
		}

		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			field := &ast.Field{
				Pos: p.curPos(),
			}

			if p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING) {
				field.Name = p.curToken.Literal
			} else {
				p.errors = append(p.errors, fmt.Errorf("expected field name (identifier or quoted string), got %s", p.curToken.Type))
				return nil
			}

			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()

			field.Value = p.parseExpression(LOWEST)
			record.Fields = append(record.Fields, field)

			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken()
				break
			}

			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken()
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
			return nil
		}

		return record
	} else if p.curTokenIs(lexer.IDENT) {
		// Could still be a record update with a more complex base expression
		// Like {foo.bar | x: 1} or {f() | x: 1}
		// Try to parse as expression and check for PIPE
		startExpr := p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.PIPE) {
			// This is a record update with complex base
			p.nextToken() // move to PIPE
			p.nextToken() // move past PIPE

			update := &ast.RecordUpdate{
				Base: startExpr,
				Pos:  startPos,
			}

			// Parse updated fields
			for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
				field := &ast.Field{
					Pos: p.curPos(),
				}

				if !p.curTokenIs(lexer.IDENT) {
					p.errors = append(p.errors, fmt.Errorf("expected field name in record update, got %s", p.curToken.Type))
					return nil
				}

				field.Name = p.curToken.Literal

				if !p.expectPeek(lexer.COLON) {
					return nil
				}
				p.nextToken()

				field.Value = p.parseExpression(LOWEST)
				update.Fields = append(update.Fields, field)

				if p.peekTokenIs(lexer.RBRACE) {
					p.nextToken()
					break
				}

				if !p.expectPeek(lexer.COMMA) {
					return nil
				}
				p.nextToken()
			}

			if !p.curTokenIs(lexer.RBRACE) {
				p.errors = append(p.errors, fmt.Errorf("expected } in record update, got %s", p.curToken.Type))
				return nil
			}

			return update
		}

		// Not a record update, must be a block starting with an expression
		// Create a block with the parsed expression as the first element
		block := &ast.Block{
			Pos:   startPos,
			Exprs: []ast.Expr{startExpr},
		}

		// Parse remaining expressions in the block
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			// Check if next token is RBRACE (end of block)
			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken()
				break
			}

			// Check for semicolon (more expressions follow)
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken() // move to SEMICOLON
				p.nextToken() // move past SEMICOLON

				// Check for trailing semicolon
				if p.curTokenIs(lexer.RBRACE) {
					break
				}

				expr := p.parseExpression(LOWEST)
				block.Exprs = append(block.Exprs, expr)
				continue
			}

			// No semicolon and no RBRACE in peek.
			// This could be valid if we're at the end of the block but the block
			// is used in a context where something else follows (e.g., comma in match arm).
			// Check if we're at a reasonable stopping point.
			// For now, we'll break and let the caller handle it.
			break
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
			return nil
		}

		return block
	} else {
		// Parse as block (semicolon-separated expressions)
		block := &ast.Block{
			Pos: startPos,
		}

		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			expr := p.parseExpression(LOWEST)
			if expr != nil {
				block.Exprs = append(block.Exprs, expr)
			}

			// If we see a semicolon, consume it and continue
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken() // move to SEMICOLON
				p.nextToken() // move past SEMICOLON
				continue
			}

			// If we see RBRACE next, we're done
			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken() // move to RBRACE
				break
			}

			// Otherwise we expect a semicolon or RBRACE
			if !p.curTokenIs(lexer.RBRACE) {
				p.errors = append(p.errors, fmt.Errorf("expected ; or }, got %s", p.peekToken.Type))
				return nil
			}
			break
		}

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, fmt.Errorf("expected }, got %s", p.curToken.Type))
			return nil
		}

		return block
	}
}
