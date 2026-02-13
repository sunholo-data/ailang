package parser

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseContractBlocks parses requires and ensures blocks after function signature.
// Returns (requires, ensures) as slices of Property with appropriate ContractKind.
//
// Syntax:
//
//	func foo(x: int) -> int ! {}
//	requires { x >= 0, x < 100 }
//	ensures  { result > x }
//	{
//	  x + 1
//	}
//
// Contracts appear after effects annotation, before body/tests/properties.
// This follows the pattern of tests/properties parsing in parser_func.go.
func (p *Parser) parseContractBlocks() (requires, ensures []*ast.Property) {
	// Parse requires block if present
	if p.peekTokenIs(lexer.REQUIRES) {
		p.nextToken() // consume 'requires'
		requires = p.parseContractBlock(ast.RequiresKind)
	}

	// Check for duplicate requires — AIs commonly write two separate blocks
	// instead of comma-separating within one: requires { a } requires { b }
	if p.peekTokenIs(lexer.REQUIRES) {
		p.report("PAR_DUPLICATE_REQUIRES",
			"only one requires block per function; combine with commas: requires { cond1, cond2 }",
			"Merge conditions into a single requires block separated by commas")
		p.nextToken() // consume duplicate 'requires'
		extra := p.parseContractBlock(ast.RequiresKind)
		requires = append(requires, extra...)
	}

	// Parse ensures block if present
	if p.peekTokenIs(lexer.ENSURES) {
		p.nextToken() // consume 'ensures'
		ensures = p.parseContractBlock(ast.EnsuresKind)
	}

	// Check for duplicate ensures — same recovery: merge and warn
	if p.peekTokenIs(lexer.ENSURES) {
		p.report("PAR_DUPLICATE_ENSURES",
			"only one ensures block per function; combine with commas: ensures { cond1, cond2 }",
			"Merge conditions into a single ensures block separated by commas")
		p.nextToken() // consume duplicate 'ensures'
		extra := p.parseContractBlock(ast.EnsuresKind)
		ensures = append(ensures, extra...)
	}

	return requires, ensures
}

// parseContractBlock parses a single contract block: { pred1, pred2, ... }
// Expects parser to be AT the contract keyword (requires/ensures).
// Returns a slice of Property with the given ContractKind.
//
// Grammar:
//
//	contract_block ::= LBRACE pred_list RBRACE
//	pred_list      ::= predicate (',' predicate)*
//	predicate      ::= expression (boolean expression)
func (p *Parser) parseContractBlock(kind ast.ContractKind) []*ast.Property {
	// Expect LBRACE after requires/ensures keyword
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	var contracts []*ast.Property
	startPos := p.curPos()

	p.nextToken() // move past LBRACE to first predicate

	// Handle empty block: requires {}
	if p.curTokenIs(lexer.RBRACE) {
		return contracts
	}

	// Parse first predicate
	contract := p.parseContractPredicate(kind)
	if contract != nil {
		contracts = append(contracts, contract)
	}

	// Parse remaining predicates separated by commas
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume COMMA
		p.nextToken() // move to next predicate

		// Handle trailing comma: requires { x >= 0, }
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		contract = p.parseContractPredicate(kind)
		if contract != nil {
			contracts = append(contracts, contract)
		}
	}

	// Expect closing RBRACE
	if !p.expectPeek(lexer.RBRACE) {
		p.report("PAR_CONTRACT_UNCLOSED",
			"expected '}' to close contract block",
			"Add closing brace after contract predicates")
		return contracts
	}

	// Set position for all contracts to the block start
	for _, c := range contracts {
		if c.Pos.Line == 0 {
			c.Pos = startPos
		}
	}

	return contracts
}

// parseContractPredicate parses a single contract predicate (boolean expression).
// The predicate is stored in a Property struct for reuse of existing infrastructure.
//
// For ensures clauses, the identifier "result" refers to the function return value.
// The evaluator/codegen will substitute this with the actual return value.
func (p *Parser) parseContractPredicate(kind ast.ContractKind) *ast.Property {
	pos := p.curPos()

	// Parse the predicate expression
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		p.report("PAR_CONTRACT_INVALID_PRED",
			"expected boolean expression in contract",
			"Contract predicates must be boolean expressions")
		return nil
	}

	return &ast.Property{
		Name:    "", // Anonymous predicate
		Kind:    kind,
		Binders: nil, // No forall binders for requires/ensures
		Expr:    expr,
		Pos:     pos,
	}
}
