package parser

import (
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseType parses a type expression
// Handles: identifiers, type variables, lists, tuples, functions
func (p *Parser) parseType() ast.Type {
	switch p.curToken.Type {
	case lexer.LBRACE:
		// Record type expression: { field: Type, ... }
		return p.parseRecordTypeExpr()

	case lexer.IDENT:
		// Simple type or type variable
		name := p.curToken.Literal
		startPos := p.curPos()

		// Check for type application: List[int], Option[a], etc.
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // consume IDENT
			p.nextToken() // consume LBRACKET

			// For now, parse type args but don't use them
			// TODO: Proper type application parsing with TypeApp AST node
			_ = p.parseType() // first arg
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // move to COMMA
				p.nextToken() // move past COMMA
				_ = p.parseType()
			}

			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}

			// Return a SimpleType for now (proper generics parsing would be more complex)
			return &ast.SimpleType{
				Name: name, // e.g., "Option" or "List"
				Pos:  startPos,
			}
		}

		// Check if it's a built-in type (lowercase but not type vars)
		builtinTypes := map[string]bool{
			"int": true, "float": true, "string": true, "bool": true,
			"unit": true, "char": true,
		}
		if builtinTypes[name] {
			return &ast.SimpleType{
				Name: name,
				Pos:  startPos,
			}
		}

		// Check if it's a type variable (lowercase single letter) or type constructor (uppercase)
		if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
			return &ast.TypeVar{
				Name: name,
				Pos:  startPos,
			}
		}

		return &ast.SimpleType{
			Name: name,
			Pos:  startPos,
		}

	case lexer.UNIT:
		// Unit type ()
		return &ast.SimpleType{
			Name: "()",
			Pos:  p.curPos(),
		}

	case lexer.LBRACKET:
		// List type: [T]
		startPos := p.curPos()
		p.nextToken() // consume LBRACKET
		elemType := p.parseType()
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		return &ast.ListType{
			Element: elemType,
			Pos:     startPos,
		}

	case lexer.LPAREN:
		// Could be:
		// - Unit type: ()
		// - Tuple type: (T1, T2, ...)
		// - Function type: (T1, T2) -> T3
		// - Grouped type: (T)
		startPos := p.curPos()
		p.nextToken() // consume LPAREN

		// Handle unit type
		if p.curTokenIs(lexer.RPAREN) {
			return &ast.SimpleType{
				Name: "()",
				Pos:  startPos,
			}
		}

		// Parse first type
		firstType := p.parseType()

		// Check what comes next
		if p.peekTokenIs(lexer.RPAREN) {
			// Could be (T) or (T) -> ...
			p.nextToken() // move to RPAREN

			// Check for arrow (function type)
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken() // consume RPAREN
				p.nextToken() // consume ARROW
				retType := p.parseType()

				// Parse optional effect annotation: (int) -> string ! {IO}
				var effects []string
				if p.peekTokenIs(lexer.BANG) {
					p.nextToken() // move to BANG
					effects = p.parseEffectAnnotation()
				}

				return &ast.FuncType{
					Params:  []ast.Type{firstType},
					Return:  retType,
					Effects: effects,
					Pos:     startPos,
				}
			}

			// Just a grouped type
			return firstType
		}

		if p.peekTokenIs(lexer.COMMA) {
			// Tuple type: (T1, T2, ...)
			types := []ast.Type{firstType}
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // move to COMMA
				p.nextToken() // move past COMMA
				if p.curTokenIs(lexer.RPAREN) {
					break // trailing comma
				}
				types = append(types, p.parseType())
			}

			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}

			// Check for arrow (function type with multiple params)
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken() // consume RPAREN
				p.nextToken() // consume ARROW
				retType := p.parseType()

				// Parse optional effect annotation: (int, string) -> bool ! {IO, FS}
				var effects []string
				if p.peekTokenIs(lexer.BANG) {
					p.nextToken() // move to BANG
					effects = p.parseEffectAnnotation()
				}

				return &ast.FuncType{
					Params:  types,
					Return:  retType,
					Effects: effects,
					Pos:     startPos,
				}
			}

			// Just a tuple type
			return &ast.TupleType{
				Elements: types,
				Pos:      startPos,
			}
		}

		// Error: unexpected token after type
		p.report("PAR_TYPE_UNEXPECTED", "unexpected token in type expression", "Check type syntax")
		return nil

	default:
		return nil
	}
}

// Type declaration parsing
// EBNF:
//   type_decl      := export? "type" UIdent type_params? "=" type_body
//   type_params    := "[" type_param ("," type_param)* "]"
//   type_param     := LIdent
//   type_body      := type_alias | sum_type | record_type
//   sum_type       := variant ("|" variant)*
//   variant        := UIdent ("(" type_expr ("," type_expr)* ")")?
//   record_type    := "{" field ("," field)* ","? "}"
//   field          := LIdent ":" type_expr

func (p *Parser) parseTypeDeclaration(exported bool) ast.Node {
	startPos := p.curPos()

	// We're already at TYPE token
	if !p.curTokenIs(lexer.TYPE) {
		p.report("PAR_TYPE_EXPECTED", "expected 'type' keyword", "Add 'type' keyword")
		return nil
	}

	p.nextToken() // consume TYPE

	// Parse type name (must be uppercase identifier)
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_TYPE_NAME_EXPECTED", "expected type name", "Add a type name starting with uppercase letter")
		return nil
	}

	name := p.curToken.Literal
	p.nextToken()

	// Parse optional type parameters [a, b, ...]
	var typeParams []string
	if p.curTokenIs(lexer.LBRACKET) {
		typeParams = p.parseTypeParams()
	}

	// Expect '='
	if !p.curTokenIs(lexer.ASSIGN) {
		p.reportExpected(lexer.ASSIGN, "Add '=' after type name")
		return nil
	}
	p.nextToken() // consume ASSIGN

	// Parse type body (sum, product, or alias)
	definition := p.parseTypeDeclBody()
	if definition == nil {
		return nil
	}

	return &ast.TypeDecl{
		Name:       name,
		TypeParams: typeParams,
		Definition: definition,
		Exported:   exported,
		Pos:        startPos,
	}
}

// hasTopLevelPipe scans ahead to check if there's a pipe (|) at depth 0
// Used to disambiguate type aliases from sum types
// Returns true if finds unbalanced | at depth 0 before newline/EOF
// This is a simple check that peeks at the next few tokens
func (p *Parser) hasTopLevelPipe() bool {
	// Simple heuristic: check if peek token is PIPE
	// or if we're at an identifier and peek is PIPE
	return p.peekTokenIs(lexer.PIPE)
}

func (p *Parser) parseTypeDeclBody() ast.TypeDef {
	// Lexer already skips whitespace/newlines, so we don't need to call skipNewlinesAndComments()

	// Allow optional leading '|' for Haskell-style multi-line ADTs:
	// type Tree =
	//   | Leaf(int)
	//   | Node(Tree, int, Tree)
	leadingPipe := false
	if p.curTokenIs(lexer.PIPE) {
		leadingPipe = true
		p.nextToken() // consume PIPE (lexer already skipped whitespace after it)
	}

	// Record type if we see '{'
	if p.curTokenIs(lexer.LBRACE) {
		return p.parseRecordTypeDef()
	}

	// Check if it's a list type alias: type Names = [string]
	if p.curTokenIs(lexer.LBRACKET) {
		typeExpr := p.parseType()
		return &ast.TypeAlias{
			Target: typeExpr,
			Pos:    p.curPos(),
		}
	}

	// For IDENT tokens, we need to disambiguate:
	// - type Color = Red | Green  → sum type
	// - type Names = [string]     → already handled above
	// - type UserId = int         → alias (single identifier)
	// - type Shape = Circle(int)  → could be alias or sum depending on |
	if p.curTokenIs(lexer.IDENT) {
		name := p.curToken.Literal
		var firstVariant *ast.Constructor

		// Check for constructor with fields: Circle(int, int)
		// Always treat as sum type (even single variant is valid)
		// Type aliases to parameterized types like Foo(int) are rare and not currently supported
		if p.peekTokenIs(lexer.LPAREN) {
			// Parse as sum type constructor
			p.nextToken() // advance to LPAREN
			// Parse constructor fields
			p.nextToken() // consume LPAREN
			var fields []ast.Type
			if !p.curTokenIs(lexer.RPAREN) {
				fields = append(fields, p.parseType())
				p.nextToken() // advance past the type we just parsed
				for p.curTokenIs(lexer.COMMA) {
					p.nextToken() // consume COMMA
					if p.curTokenIs(lexer.RPAREN) {
						break // trailing comma
					}
					fields = append(fields, p.parseType())
					p.nextToken() // advance past the type we just parsed
				}
			}
			if !p.curTokenIs(lexer.RPAREN) {
				p.reportExpected(lexer.RPAREN, "Add ')' to close constructor fields")
			} else {
				p.nextToken() // consume RPAREN
			}
			firstVariant = &ast.Constructor{
				Name:   name,
				Fields: fields,
				Pos:    p.curPos(),
			}
			// Lexer skips whitespace, so we're already at next real token (PIPE or other)
		} else {
			// No fields - check if this is a simple type alias or sum type
			// If we saw a leading PIPE, it's definitely a sum type
			if leadingPipe {
				// Definitely a sum type with no fields (e.g., Red, Green, Blue)
				firstVariant = &ast.Constructor{
					Name:   name,
					Fields: nil,
					Pos:    p.curPos(),
				}
				// Advance past the name (lexer already skipped whitespace)
				p.nextToken()
			} else {
				// Check if peek is PIPE to determine if it's a sum type
				if !p.peekTokenIs(lexer.PIPE) {
					// No pipe → simple type alias like: type UserId = int
					// We're at the identifier, parse it as a type
					typeExpr := p.parseType()
					return &ast.TypeAlias{
						Target: typeExpr,
						Pos:    p.curPos(),
					}
				}

				// Has pipe → sum type like: type Color = Red | Green | Blue
				// Advance past the name and save it as first variant
				p.nextToken() // advance to PIPE
				firstVariant = &ast.Constructor{
					Name:   name, // We saved this earlier
					Fields: nil,
					Pos:    p.curPos(),
				}
			}
		}

		// Check if there are more variants (PIPE)
		// At this point, we should be at a PIPE token if there are more variants
		if p.curTokenIs(lexer.PIPE) {
			variants := []*ast.Constructor{firstVariant}
			// Parse remaining variants
			// After first variant, we're at PIPE or end of ADT
			// Lexer skips whitespace/newlines, so no need to check for NEWLINE tokens
			for p.curTokenIs(lexer.PIPE) {
				p.nextToken() // consume PIPE
				variant := p.parseVariant()
				if variant != nil {
					variants = append(variants, variant)
				}
				// parseVariant() leaves us AT the last token (RPAREN or variant name)
				// Check if there's another PIPE by peeking
				if p.peekTokenIs(lexer.PIPE) {
					p.nextToken() // advance to PIPE for next iteration
				} else {
					// No more variants - stay at current position
					break
				}
			}
			return &ast.AlgebraicType{
				Constructors: variants,
				Pos:          p.curPos(),
			}
		}

		// Single constructor, still a sum type
		return &ast.AlgebraicType{
			Constructors: []*ast.Constructor{firstVariant},
			Pos:          p.curPos(),
		}
	}

	p.report("PAR_TYPE_BODY_EXPECTED", "expected type definition", "Add type definition (record, sum type, or alias)")
	return nil
}

func (p *Parser) parseVariant() *ast.Constructor {
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_VARIANT_NAME_EXPECTED", "expected variant name", "Add variant name starting with uppercase letter")
		return nil
	}

	name := p.curToken.Literal

	// Check if name starts with uppercase (convention for constructors)
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		p.report("PAR_VARIANT_NEEDS_UIDENT", "variant must start with uppercase letter", "Change to UpperCamelCase")
	}

	// Parse optional fields (peek ahead to see if there are any)
	var fields []ast.Type
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // advance to LPAREN
		p.nextToken() // consume LPAREN
		if !p.curTokenIs(lexer.RPAREN) {
			fields = append(fields, p.parseType())
			p.nextToken() // advance past the type we just parsed
			for p.curTokenIs(lexer.COMMA) {
				p.nextToken() // consume COMMA
				if p.curTokenIs(lexer.RPAREN) {
					break // trailing comma
				}
				fields = append(fields, p.parseType())
				p.nextToken() // advance past the type we just parsed
			}
		}
		if !p.curTokenIs(lexer.RPAREN) {
			p.reportExpected(lexer.RPAREN, "Add ')' to close variant fields")
			// Return constructor even if there's an error
		}
		// DON'T consume RPAREN - leave it for the caller to handle
		// This matches the pattern where parse functions leave the parser
		// at the last token they recognize, not past it
	}

	return &ast.Constructor{
		Name:   name,
		Fields: fields,
		Pos:    p.curPos(),
	}
}

func (p *Parser) parseRecordTypeDef() ast.TypeDef {
	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_TYPE_LBRACE_EXPECTED", "expected '{' for record type", "Add '{' to start record type")
		return nil
	}
	p.nextToken() // consume LBRACE

	var fields []*ast.RecordField
	if !p.curTokenIs(lexer.RBRACE) {
		// Parse first field
		field := p.parseRecordFieldDef()
		if field != nil {
			fields = append(fields, field)
		}
		p.nextToken() // advance past the field we just parsed

		// Parse remaining fields
		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACE) {
				break // trailing comma
			}
			field := p.parseRecordFieldDef()
			if field != nil {
				fields = append(fields, field)
			}
			p.nextToken() // advance past the field
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_TYPE_RBRACE_MISSING", "expected '}' to close record type", "Add '}' to close record type")
	}

	return &ast.RecordType{
		Fields: fields,
		Pos:    p.curPos(),
	}
}

// parseRecordTypeExpr parses a record type expression that can appear in type positions
// Example: { street: string, city: string }
// This is used for nested record types like: type User = { addr: { street: string } }
func (p *Parser) parseRecordTypeExpr() ast.Type {
	startPos := p.curPos()

	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_TYPE_LBRACE_EXPECTED", "expected '{' for record type", "Add '{' to start record type")
		return nil
	}
	p.nextToken() // consume LBRACE

	var fields []*ast.RecordField
	if !p.curTokenIs(lexer.RBRACE) {
		// Parse first field
		field := p.parseRecordFieldDef()
		if field != nil {
			fields = append(fields, field)
		}
		p.nextToken() // advance past the field we just parsed

		// Parse remaining fields
		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACE) {
				break // trailing comma
			}
			field := p.parseRecordFieldDef()
			if field != nil {
				fields = append(fields, field)
			}
			p.nextToken() // advance past the field
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_TYPE_RBRACE_MISSING", "expected '}' to close record type", "Add '}' to close record type")
	}

	return &ast.RecordType{
		Fields: fields,
		Pos:    startPos,
	}
}

func (p *Parser) parseRecordFieldDef() *ast.RecordField {
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_FIELD_NAME_EXPECTED", "expected field name", "Add field name")
		return nil
	}

	name := p.curToken.Literal
	p.nextToken()

	if !p.curTokenIs(lexer.COLON) {
		p.reportExpected(lexer.COLON, "Add ':' after field name")
		return nil
	}
	p.nextToken() // consume COLON

	fieldType := p.parseType()
	if fieldType == nil {
		p.report("PAR_FIELD_TYPE_EXPECTED", "expected field type", "Add field type")
		return nil
	}

	return &ast.RecordField{
		Name: name,
		Type: fieldType,
		Pos:  p.curPos(),
	}
}

func (p *Parser) parseTypeParams() []string {
	if !p.curTokenIs(lexer.LBRACKET) {
		return []string{}
	}
	p.nextToken() // consume LBRACKET

	var params []string
	if !p.curTokenIs(lexer.RBRACKET) {
		if p.curTokenIs(lexer.IDENT) {
			params = append(params, p.curToken.Literal)
			p.nextToken()
		}

		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACKET) {
				break // trailing comma
			}
			if p.curTokenIs(lexer.IDENT) {
				params = append(params, p.curToken.Literal)
				p.nextToken()
			}
		}
	}

	if !p.curTokenIs(lexer.RBRACKET) {
		p.reportExpected(lexer.RBRACKET, "Add ']' to close type parameters")
	} else {
		p.nextToken() // consume RBRACKET
	}

	return params
}
