package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// Record literal and record-update parsing. Extracted from parser_expr.go
// (M-RELEASE-GATE follow-up) to keep that file under the 800-line limit.

// parseRecordLiteralContent parses a record literal after the { has been consumed.
// Called when we detect {IDENT/STRING COLON ...} pattern in parseBlockOrExpression.
// Cursor is at the first IDENT or STRING token.
func (p *Parser) parseRecordLiteralContent(startPos ast.Pos) ast.Expr {
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
}

// parseRecordUpdateContent parses a record update after the { has been consumed.
// Called when we detect {IDENT PIPE ...} pattern in parseBlockOrExpression.
// Cursor is at the first IDENT token (the base expression identifier).
func (p *Parser) parseRecordUpdateContent(startPos ast.Pos) ast.Expr {
	// Parse the base expression (the IDENT we're at)
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
}
