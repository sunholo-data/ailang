package parser

import (
	"strconv"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

func (p *Parser) parsePattern() ast.Pattern {
	switch p.curToken.Type {
	case lexer.IDENT:
		// Could be a variable pattern or constructor
		name := p.curToken.Literal
		if p.peekTokenIs(lexer.LPAREN) {
			// Constructor with arguments
			p.nextToken()
			return p.parseConstructorPattern(name)
		}
		return &ast.Identifier{
			Name: name,
			Pos:  p.curPos(),
		}
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE:
		return &ast.Literal{
			Kind:  p.literalKind(),
			Value: p.literalValue(),
			Pos:   p.curPos(),
		}
	case lexer.LBRACKET:
		return p.parseListPattern()
	case lexer.LBRACE:
		return p.parseRecordPattern()
	case lexer.LPAREN:
		return p.parseTuplePattern()
	default:
		if p.curToken.Literal == "_" {
			return &ast.WildcardPattern{
				Pos: p.curPos(),
			}
		}
	}
	return nil
}

func (p *Parser) parseConstructorPattern(name string) ast.Pattern {
	constructor := &ast.ConstructorPattern{
		Name:     name,
		Pos:      p.curPos(),
		Patterns: []ast.Pattern{},
	}

	// We're at LPAREN
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume RPAREN
		return constructor
	}

	p.nextToken() // move to first argument
	constructor.Patterns = append(constructor.Patterns, p.parsePattern())

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next argument
		constructor.Patterns = append(constructor.Patterns, p.parsePattern())
	}

	p.expectPeek(lexer.RPAREN)
	return constructor
}

func (p *Parser) parseListPattern() ast.Pattern {
	startPos := p.curPos()
	// We're at LBRACKET
	p.nextToken() // consume LBRACKET

	// Empty list pattern: []
	if p.curTokenIs(lexer.RBRACKET) {
		// Parser convention: leave at last token of pattern (RBRACKET)
		return &ast.ListPattern{
			Elements: []ast.Pattern{},
			Rest:     nil,
			Pos:      startPos,
		}
	}

	// Non-empty list: [x, ...] or [x, y, ...rest]
	elements := []ast.Pattern{}
	var rest ast.Pattern

	for {
		// Check for spread pattern: ...rest
		if p.curTokenIs(lexer.ELLIPSIS) {
			p.nextToken() // consume ELLIPSIS
			if !p.curTokenIs(lexer.IDENT) {
				p.report("PAT_SPREAD_NEEDS_IDENT", "spread in list pattern must bind to a name, e.g. [x, ...xs]", "Add an identifier after ..., like [x, ...rest]")
				return nil
			}
			rest = &ast.Identifier{
				Name: p.curToken.Literal,
				Pos:  p.curPos(),
			}
			p.nextToken() // consume ident
			break         // spread must be last
		}

		// Parse next pattern element
		elem := p.parsePattern()
		if elem == nil {
			return nil
		}
		elements = append(elements, elem)

		// Check what comes next
		p.nextToken() // move past pattern element

		if p.curTokenIs(lexer.RBRACKET) {
			// End of list
			break
		}

		if !p.curTokenIs(lexer.COMMA) {
			p.reportExpected(lexer.COMMA, "Expected ',' or ']' in list pattern")
			return nil
		}

		p.nextToken() // consume comma

		// Check for closing bracket after comma (trailing comma)
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}
	}

	// We should be at RBRACKET now
	if !p.curTokenIs(lexer.RBRACKET) {
		p.reportExpected(lexer.RBRACKET, "Expected ']' to close list pattern")
		return nil
	}
	// Pattern parsing convention: leave current token at the last token of the pattern
	// The caller will advance past it

	return &ast.ListPattern{
		Elements: elements,
		Rest:     rest,
		Pos:      startPos,
	}
}

func (p *Parser) parseRecordPattern() ast.Pattern {
	// TODO: Implement record pattern parsing
	return nil
}

func (p *Parser) parseTuplePattern() ast.Pattern {
	startPos := p.curPos()
	// We're at LPAREN
	p.nextToken() // consume LPAREN

	// Empty tuple: ()
	if p.curTokenIs(lexer.RPAREN) {
		// Empty tuple pattern (same as Unit pattern)
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: nil,
			Pos:   startPos,
		}
	}

	// Parse first element
	first := p.parsePattern()

	// Single element in parens: (x) - not a tuple, just a grouped pattern
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume RPAREN
		return first
	}

	// Must be a comma for tuple
	if !p.peekTokenIs(lexer.COMMA) {
		p.reportExpected(lexer.COMMA, "Expected ',' for tuple pattern or ')' for grouped pattern")
		return nil
	}

	// Parse remaining elements
	elements := []ast.Pattern{first}
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if p.peekTokenIs(lexer.RPAREN) {
			// Trailing comma
			break
		}
		p.nextToken() // move to next element
		elements = append(elements, p.parsePattern())
	}

	p.expectPeek(lexer.RPAREN)

	return &ast.TuplePattern{
		Elements: elements,
		Pos:      startPos,
	}
}

func (p *Parser) literalKind() ast.LiteralKind {
	switch p.curToken.Type {
	case lexer.INT:
		return ast.IntLit
	case lexer.FLOAT:
		return ast.FloatLit
	case lexer.STRING:
		return ast.StringLit
	case lexer.TRUE, lexer.FALSE:
		return ast.BoolLit
	default:
		return ast.StringLit
	}
}

func (p *Parser) literalValue() interface{} {
	switch p.curToken.Type {
	case lexer.INT:
		v, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
		return v
	case lexer.FLOAT:
		v, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		return v
	case lexer.STRING:
		return p.curToken.Literal
	case lexer.TRUE:
		return true
	case lexer.FALSE:
		return false
	default:
		return p.curToken.Literal
	}
}
