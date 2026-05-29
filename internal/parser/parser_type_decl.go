package parser

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

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

	// Parse optional deriving clause: deriving (Eq)
	// After parseTypeDeclBody, parser may be AT deriving or BEFORE it
	// depending on the type body structure (single vs multiple constructors)
	var deriving []ast.DeriveKind
	if p.curTokenIs(lexer.DERIVING) {
		deriving = p.parseDeriving()
	} else if p.peekTokenIs(lexer.DERIVING) {
		p.nextToken() // move to DERIVING
		deriving = p.parseDeriving()
	}

	return &ast.TypeDecl{
		Name:       name,
		TypeParams: typeParams,
		Definition: definition,
		Deriving:   deriving,
		Exported:   exported,
		Pos:        startPos,
	}
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
			var fields []*ast.ConstructorField
			if !p.curTokenIs(lexer.RPAREN) {
				fields = append(fields, p.parseConstructorField())
				p.nextToken() // advance past the field we just parsed
				for p.curTokenIs(lexer.COMMA) {
					p.nextToken() // consume COMMA
					if p.curTokenIs(lexer.RPAREN) {
						break // trailing comma
					}
					fields = append(fields, p.parseConstructorField())
					p.nextToken() // advance past the field we just parsed
				}
			}
			if !p.curTokenIs(lexer.RPAREN) {
				p.reportExpected(lexer.RPAREN, "Add ')' to close constructor fields")
			}
			// Leave cursor AT RPAREN, matching parseVariant() convention.
			// If more variants follow (PIPE), advance to PIPE for the variant loop below.
			// If single constructor, stay at RPAREN — the main loop (parser_file.go:81)
			// calls nextToken() to advance between declarations.
			firstVariant = &ast.Constructor{
				Name:   name,
				Fields: fields,
				Pos:    p.curPos(),
			}
			if p.peekTokenIs(lexer.PIPE) {
				p.nextToken() // advance from RPAREN to PIPE for multi-variant parsing
			}
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
	var fields []*ast.ConstructorField
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // advance to LPAREN
		p.nextToken() // consume LPAREN
		if !p.curTokenIs(lexer.RPAREN) {
			fields = append(fields, p.parseConstructorField())
			p.nextToken() // advance past the field we just parsed
			for p.curTokenIs(lexer.COMMA) {
				p.nextToken() // consume COMMA
				if p.curTokenIs(lexer.RPAREN) {
					break // trailing comma
				}
				fields = append(fields, p.parseConstructorField())
				p.nextToken() // advance past the field we just parsed
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

// parseConstructorField parses a constructor field.
// Supports both named (x: int) and positional (int) syntax.
func (p *Parser) parseConstructorField() *ast.ConstructorField {
	pos := p.curPos()

	// Check for named field syntax: name: type
	// Look ahead: if current is IDENT and peek is COLON, it's named
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
		name := p.curToken.Literal
		p.nextToken() // consume IDENT
		p.nextToken() // consume COLON
		typ := p.parseType()
		return &ast.ConstructorField{
			Name: name,
			Type: typ,
			Pos:  pos,
		}
	}

	// Positional field (type only)
	typ := p.parseType()
	return &ast.ConstructorField{
		Name: "", // Empty name for positional fields
		Type: typ,
		Pos:  pos,
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
