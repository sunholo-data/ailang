package parser

import (
	"fmt"
	"strconv"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
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

	// Peek ahead to determine if this is a record literal, record update, or a block
	// Record literals: IDENT COLON ...
	// Record updates: IDENT PIPE ...
	// Blocks: anything else
	isRecordLiteral := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)
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

			if !p.curTokenIs(lexer.IDENT) {
				p.errors = append(p.errors, fmt.Errorf("expected field name, got %s", p.curToken.Type))
				return nil
			}

			field.Name = p.curToken.Literal

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
			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken()
				break
			}

			if !p.expectPeek(lexer.SEMICOLON) {
				return nil
			}
			p.nextToken() // move past semicolon

			if p.curTokenIs(lexer.RBRACE) {
				// Trailing semicolon before }
				break
			}

			expr := p.parseExpression(LOWEST)
			block.Exprs = append(block.Exprs, expr)
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
