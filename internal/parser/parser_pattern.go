package parser

import (
	"strconv"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func (p *Parser) parsePattern() ast.Pattern {
	// Parse base pattern first
	pat := p.parseBasePattern()
	if pat == nil {
		return nil
	}

	// Right-associative :: sugar (S-CONS pattern extension v0.4.3)
	// After parsing base pattern, check if next token is :: for cons sugar
	// Example: x :: xs desugars to ::(x, xs)
	for p.peekTokenIs(lexer.DCOLON) {
		consPos := p.curPos()

		// Check if strict syntax mode is enabled
		if p.strictSyntaxMode {
			p.reportSugarError("CONS_PATTERN", "x :: xs", "::(x, xs)")
			// Return current pattern to avoid cascading errors
			return pat
		}

		// Sugar is allowed - mark that it was used
		p.sugarUsed = true

		// Move to DCOLON, then past it to RHS
		p.nextToken() // now at DCOLON
		p.nextToken() // now at start of RHS pattern

		// Recursively parse right side (allows chaining: a :: b :: c)
		rhs := p.parsePattern()
		if rhs == nil {
			return nil
		}

		// Desugar to canonical constructor pattern: ::(pat, rhs)
		// See internal/elaborate/patterns.go for elaboration to ListPattern
		pat = &ast.ConstructorPattern{
			Name:     "::",
			Patterns: []ast.Pattern{pat, rhs},
			Pos:      consPos,
		}
	}

	return pat
}

// parseBasePattern parses atomic patterns (no infix operators)
func (p *Parser) parseBasePattern() ast.Pattern {
	switch p.curToken.Type {
	case lexer.IDENT:
		// Could be a variable pattern or constructor
		name := p.curToken.Literal
		if p.peekTokenIs(lexer.LPAREN) {
			// Constructor with arguments, located at its name (not the `(`).
			namePos := p.curPos()
			p.nextToken()
			return p.parseConstructorPattern(name, namePos)
		}
		return &ast.Identifier{
			Name: name,
			Pos:  p.curPos(),
		}
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE:
		value, err := p.literalValue()
		if err != nil {
			// #1481: an out-of-range literal used to become a silent
			// MaxInt64 / +Inf pattern. It is a positioned PAR021 now.
			p.errors = append(p.errors, err)
			return nil
		}
		return &ast.Literal{
			Kind:  p.literalKind(),
			Value: value,
			Pos:   p.curPos(),
		}
	case lexer.DCOLON:
		// Handle :: (cons) constructor pattern - canonical form only
		// :: is the list constructor, equivalent to Cons in other languages
		// See internal/elaborate/patterns.go for elaboration to ListPattern (not ConstructorPattern!)
		name := "::"
		if p.peekTokenIs(lexer.LPAREN) {
			namePos := p.curPos()
			p.nextToken() // move to LPAREN
			return p.parseConstructorPattern(name, namePos)
		}
		// :: without arguments is invalid (must be ::(head, tail) or use infix x :: xs)
		p.report("PAT_INVALID_CONS", ":: constructor requires arguments in pattern", "Use ::(head, tail) or x :: xs pattern")
		return nil
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

// parseConstructorPattern parses `name(args)` with the cursor at `(`; pos is
// the constructor name's position, which diagnostics about the pattern use.
func (p *Parser) parseConstructorPattern(name string, pos ast.Pos) ast.Pattern {
	constructor := &ast.ConstructorPattern{
		Name:     name,
		Pos:      pos,
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
	startPos := p.curPos()
	// We're at LBRACE
	p.nextToken() // consume LBRACE

	// Empty record pattern: {}
	if p.curTokenIs(lexer.RBRACE) {
		return &ast.RecordPattern{
			Fields: []*ast.FieldPattern{},
			Rest:   false,
			Pos:    startPos,
		}
	}

	var fields []*ast.FieldPattern
	hasRest := false

	for {
		// Check for rest pattern: ...
		if p.curTokenIs(lexer.ELLIPSIS) {
			hasRest = true
			p.nextToken() // consume ELLIPSIS
			break         // rest must be last
		}

		// Expect field name (identifier, or contextual contract keyword when renamed with ':')
		if !p.curTokenIs(lexer.IDENT) && !(isContextualFieldKeyword(p.curToken.Type) && p.peekTokenIs(lexer.COLON)) {
			p.reportExpected(lexer.IDENT, "Expected field name in record pattern")
			return nil
		}

		fieldName := p.curToken.Literal
		fieldPos := p.curPos()

		// Check what comes next: COLON (renaming) or COMMA/RBRACE (shorthand)
		if p.peekTokenIs(lexer.COLON) {
			// Renaming syntax: {name: n} or {user: {nested}}
			p.nextToken() // consume field name, now at COLON
			p.nextToken() // consume COLON, now at pattern

			// Parse the pattern (can be nested record, identifier, etc.)
			pat := p.parsePattern()
			if pat == nil {
				return nil
			}

			fields = append(fields, &ast.FieldPattern{
				Name:    fieldName,
				Pattern: pat,
				Pos:     fieldPos,
			})

			p.nextToken() // move past pattern
		} else {
			// Shorthand syntax: {name} binds to variable "name"
			fields = append(fields, &ast.FieldPattern{
				Name: fieldName,
				Pattern: &ast.Identifier{
					Name: fieldName,
					Pos:  fieldPos,
				},
				Pos: fieldPos,
			})

			p.nextToken() // move past identifier
		}

		// Check what comes next
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		if !p.curTokenIs(lexer.COMMA) {
			p.reportExpected(lexer.COMMA, "Expected ',' or '}' in record pattern")
			return nil
		}

		p.nextToken() // consume comma

		// Check for closing brace after comma (trailing comma)
		if p.curTokenIs(lexer.RBRACE) {
			break
		}
	}

	// We should be at RBRACE now
	if !p.curTokenIs(lexer.RBRACE) {
		p.reportExpected(lexer.RBRACE, "Expected '}' to close record pattern")
		return nil
	}

	return &ast.RecordPattern{
		Fields: fields,
		Rest:   hasRest,
		Pos:    startPos,
	}
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

// literalValue converts the current literal token for a pattern. Range errors
// are returned, never discarded: Go's parsers return a clamped value
// (MaxInt64, MaxUint64, +Inf) alongside the error (#1481).
func (p *Parser) literalValue() (interface{}, error) {
	switch p.curToken.Type {
	case lexer.INT:
		v, err := parseIntLiteralValue(p.curToken.Literal)
		if err != nil {
			return nil, intLiteralError(p.curToken, p.curPos())
		}
		return v, nil
	case lexer.FLOAT:
		v, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			return nil, floatLiteralError(p.curToken, p.curPos())
		}
		return v, nil
	case lexer.STRING:
		return p.curToken.Literal, nil
	case lexer.TRUE:
		return true, nil
	case lexer.FALSE:
		return false, nil
	default:
		return p.curToken.Literal, nil
	}
}

// parseLetPattern parses irrefutable destructuring `let` bindings:
//   - tuple destructuring: `let (p1, p2, ...) = value [in body]` (M-SYNTAX-ERGONOMICS-V1 E1)
//   - record destructuring: `let {a, b: x, ...} = value [in body]` (M-SYNTAX-ERGONOMICS-V2 E5)
//
// Only irrefutable patterns (identifiers, wildcards, and nested tuples/records
// of those) are accepted; refutable patterns direct the user to `match`.
func (p *Parser) parseLetPattern(startPos ast.Pos) ast.Expr {
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
	lp.Value = p.parseExpression(LOWEST)
	if lp.Value == nil {
		lp.Value = &ast.Error{Pos: p.curPos()}
	}

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

// isIrrefutableLetPattern reports whether pat is an irrefutable tuple or record
// pattern allowed in a `let` destructuring binding.
func isIrrefutableLetPattern(pat ast.Pattern) bool {
	switch p := pat.(type) {
	case *ast.TuplePattern:
		return isIrrefutableTuplePattern(p)
	case *ast.RecordPattern:
		return isIrrefutableRecordPattern(p)
	default:
		return false
	}
}

// isIrrefutablePatternElem reports whether a sub-pattern inside a tuple/record
// destructuring pattern is irrefutable.
func isIrrefutablePatternElem(pat ast.Pattern) bool {
	switch p := pat.(type) {
	case *ast.Identifier, *ast.WildcardPattern:
		return true
	case *ast.TuplePattern:
		return isIrrefutableTuplePattern(p)
	case *ast.RecordPattern:
		return isIrrefutableRecordPattern(p)
	default:
		return false
	}
}

// isIrrefutableTuplePattern reports whether pat is a tuple pattern of ≥2 elements
// whose elements are all irrefutable.
func isIrrefutableTuplePattern(pat ast.Pattern) bool {
	tp, ok := pat.(*ast.TuplePattern)
	if !ok || len(tp.Elements) < 2 {
		return false
	}
	for _, el := range tp.Elements {
		if !isIrrefutablePatternElem(el) {
			return false
		}
	}
	return true
}

// isIrrefutableRecordPattern reports whether pat is a record pattern whose field
// sub-patterns are all irrefutable (both closed `{a, b}` and open `{a, ...}`).
func isIrrefutableRecordPattern(pat ast.Pattern) bool {
	rp, ok := pat.(*ast.RecordPattern)
	if !ok {
		return false
	}
	if len(rp.Fields) == 0 && !rp.Rest {
		return false
	}
	for _, f := range rp.Fields {
		if !isIrrefutablePatternElem(f.Pattern) {
			return false
		}
	}
	return true
}
