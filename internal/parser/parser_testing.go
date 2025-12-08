package parser

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseTestsBlock parses inline test cases: tests [(input1, expected1), (input2, expected2)]
// Expects to be called after 'tests' keyword and positioned at LBRACKET
// Leaves parser positioned at RBRACKET
func (p *Parser) parseTestsBlock() []*ast.TestCase {
	if !p.curTokenIs(lexer.LBRACKET) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected [ to start tests block", "Check syntax")
		return nil
	}

	p.nextToken() // consume LBRACKET, move to first test case or RBRACKET

	var tests []*ast.TestCase

	// Handle empty tests block: tests []
	if p.curTokenIs(lexer.RBRACKET) {
		return tests
	}

	for {
		// Skip newlines and commas between test cases
		for p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}

		// Check for end of tests block
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}

		// Parse test case: (input, expected) or ((input1, input2), expected)
		testCase := p.parseTestCase()
		if testCase != nil {
			tests = append(tests, testCase)
		}

		// Skip trailing newlines
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

		// Check if we're done or continuing
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			continue
		}

		// If we reach here without RBRACKET or COMMA, it's an error
		if !p.curTokenIs(lexer.RBRACKET) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected , or ] in tests block", "Check syntax")
			break
		}
	}

	return tests
}

// parseTestCase parses a single test case: (input, expected) or ((input1, input2, ...), expected)
func (p *Parser) parseTestCase() *ast.TestCase {
	pos := p.curPos()

	if !p.curTokenIs(lexer.LPAREN) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected ( to start test case", "Check syntax")
		return nil
	}

	p.nextToken() // consume LPAREN

	// Parse inputs - could be single value or tuple of values
	var inputs []ast.Expr

	// Check if inputs are wrapped in parens: ((input1, input2), expected)
	if p.curTokenIs(lexer.LPAREN) {
		// Multi-arg test: ((arg1, arg2), expected)
		p.nextToken() // consume inner LPAREN

		for {
			if p.curTokenIs(lexer.RPAREN) {
				break
			}

			input := p.parseExpression(LOWEST)
			if input != nil {
				inputs = append(inputs, input)
			}
			p.nextToken() // advance past the input expression

			if p.curTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
			} else if !p.curTokenIs(lexer.RPAREN) {
				p.report("PAR_UNEXPECTED_TOKEN", "expected , or ) in test inputs", "Check syntax")
				return nil
			}
		}

		if !p.curTokenIs(lexer.RPAREN) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected ) to close test inputs", "Check syntax")
			return nil
		}
		p.nextToken() // consume inner RPAREN

		if !p.curTokenIs(lexer.COMMA) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected , between inputs and expected value", "Check syntax")
			return nil
		}
		p.nextToken() // consume comma
	} else {
		// Single-arg test: (input, expected)
		input := p.parseExpression(LOWEST)
		if input == nil {
			p.report("PAR_UNEXPECTED_TOKEN", "expected input value in test case", "Check syntax")
			return nil
		}
		inputs = append(inputs, input)
		p.nextToken() // advance past the input expression

		if !p.curTokenIs(lexer.COMMA) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected , between input and expected value", "Check syntax")
			return nil
		}
		p.nextToken() // consume comma
	}

	// Parse expected value
	expected := p.parseExpression(LOWEST)
	if expected == nil {
		p.report("PAR_UNEXPECTED_TOKEN", "expected output value in test case", "Check syntax")
		return nil
	}
	p.nextToken() // advance past the expected expression

	if !p.curTokenIs(lexer.RPAREN) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected ) to close test case", "Check syntax")
		return nil
	}
	p.nextToken() // consume RPAREN

	return &ast.TestCase{
		Inputs:   inputs,
		Expected: expected,
		Pos:      pos,
	}
}

// parsePropertiesBlock parses a properties block: [ property1, property2, ... ]
// Expects to be AT LBRACKET when called.
// Returns with parser AT RBRACKET.
func (p *Parser) parsePropertiesBlock() []*ast.Property {
	if !p.curTokenIs(lexer.LBRACKET) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected [ to start properties block", "Check syntax")
		return nil
	}

	p.nextToken() // consume LBRACKET, move to first property or RBRACKET

	var properties []*ast.Property

	// Handle empty properties block: properties []
	if p.curTokenIs(lexer.RBRACKET) {
		return properties
	}

	for {
		// Skip newlines and commas between properties
		for p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}

		// Check for end of properties block
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}

		// Parse property: forall(...) => expr
		property := p.parseProperty()
		if property != nil {
			properties = append(properties, property)
		}

		// Skip trailing newlines
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

		// Check if we're done or continuing
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			continue
		}

		// If we reach here without RBRACKET or COMMA, it's an error
		if !p.curTokenIs(lexer.RBRACKET) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected , or ] in properties block", "Check syntax")
			break
		}
	}

	return properties
}

// parseProperty parses a single property: forall(x: Type, y: Type) => expr
// Properties can optionally have names for documentation purposes.
func (p *Parser) parseProperty() *ast.Property {
	pos := p.curPos()

	// Optional property name (for standalone property blocks in future)
	var name string

	// Check for 'forall' keyword
	if !p.curTokenIs(lexer.FORALL) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected forall in property", "Check syntax")
		return nil
	}

	p.nextToken() // consume FORALL

	// Parse binders: (x: Type, y: Type)
	if !p.curTokenIs(lexer.LPAREN) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected ( after forall", "Check syntax")
		return nil
	}

	p.nextToken() // consume LPAREN

	var binders []*ast.Binder

	// Parse binders list
	for {
		if p.curTokenIs(lexer.RPAREN) {
			break
		}

		binder := p.parseBinder()
		if binder != nil {
			binders = append(binders, binder)
		}

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
		} else if !p.curTokenIs(lexer.RPAREN) {
			p.report("PAR_UNEXPECTED_TOKEN", "expected , or ) in forall binders", "Check syntax")
			return nil
		}
	}

	if !p.curTokenIs(lexer.RPAREN) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected ) to close forall binders", "Check syntax")
		return nil
	}
	p.nextToken() // consume RPAREN

	// Expect '=>' (FARROW token)
	if !p.curTokenIs(lexer.FARROW) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected => after forall binders", "Check syntax")
		return nil
	}
	p.nextToken() // consume FARROW (=>)

	// Parse property expression (predicate)
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		p.report("PAR_UNEXPECTED_TOKEN", "expected expression in property", "Check syntax")
		return nil
	}
	p.nextToken() // advance past the expression

	return &ast.Property{
		Name:    name, // Empty for inline properties
		Binders: binders,
		Expr:    expr,
		Pos:     pos,
	}
}

// parseBinder parses a forall binder: name: Type
func (p *Parser) parseBinder() *ast.Binder {
	pos := p.curPos()

	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected identifier in binder", "Check syntax")
		return nil
	}

	name := p.curToken.Literal
	p.nextToken() // consume name

	if !p.curTokenIs(lexer.COLON) {
		p.report("PAR_UNEXPECTED_TOKEN", "expected : after binder name", "Check syntax")
		return nil
	}
	p.nextToken() // consume COLON

	// Parse type
	typ := p.parseType()
	if typ == nil {
		p.report("PAR_UNEXPECTED_TOKEN", "expected type in binder", "Check syntax")
		return nil
	}
	p.nextToken() // advance past the type

	return &ast.Binder{
		Name: name,
		Type: typ,
		Pos:  pos,
	}
}
