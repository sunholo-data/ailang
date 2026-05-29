package parser

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseType parses a type expression
// Handles: identifiers, type variables, lists, tuples, functions
// S-ARROWTYPE: Supports bare function type arrows (int -> bool)
func (p *Parser) parseType() ast.Type {
	p.debugEnter("parseType")
	defer p.debugExit("parseType")

	var typ ast.Type // Store primary type for arrow sugar checking

	switch p.curToken.Type {
	case lexer.LBRACE:
		// Record type expression: { field: Type, ... }
		return p.parseRecordTypeExpr()

	case lexer.IDENT:
		// Simple type or type variable
		name := p.curToken.Literal
		startPos := p.curPos()

		// Check for type application: List[int], Array[T], Option[a], etc.
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // consume IDENT
			p.nextToken() // consume LBRACKET

			// Parse the first type argument (element type for Array/List)
			elemType := p.parseType()

			// Parse additional type arguments (for multi-arg generics like Result[T, E])
			// M-TAPP-FIX: Collect ALL type arguments, not just the first
			typeArgs := []ast.Type{elemType}
			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // move to COMMA
				p.nextToken() // move past COMMA
				typeArgs = append(typeArgs, p.parseType())
			}

			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}

			// M-TYPE1: Special-case Array to preserve element types
			// This enables proper unification: Array[T] in ADT params works with #[...] literals
			// DX-17 Phase 2: List[T] now uses TypeApp for uniform representation
			switch name {
			case "Array":
				typ = &ast.ArrayType{Element: elemType, Pos: startPos}
			case "List":
				// DX-17 Phase 2: Normalize List[T] to lowercase "list" for consistency with [T] syntax
				typ = &ast.TypeApp{Constructor: "list", Args: typeArgs, Pos: startPos}
			default:
				// M-TAPP-FIX: Use TypeApp to preserve type arguments for generic types
				// This enables proper type checking of Option[T], Result[T, E], etc.
				typ = &ast.TypeApp{Constructor: name, Args: typeArgs, Pos: startPos}
			}
			goto checkArrow
		}

		// Check if it's a built-in type (lowercase but not type vars)
		builtinTypes := map[string]bool{
			"int": true, "float": true, "string": true, "bool": true,
			"unit": true, "char": true,
		}
		if builtinTypes[name] {
			typ = &ast.SimpleType{
				Name: name,
				Pos:  startPos,
			}
			goto checkArrow
		}

		// Check if it's a type variable (lowercase single letter) or type constructor (uppercase)
		if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
			typ = &ast.TypeVar{
				Name: name,
				Pos:  startPos,
			}
			goto checkArrow
		}

		typ = &ast.SimpleType{
			Name: name,
			Pos:  startPos,
		}
		goto checkArrow

	case lexer.UNIT:
		// Unit type ()
		typ = &ast.SimpleType{
			Name: "()",
			Pos:  p.curPos(),
		}
		goto checkArrow

	case lexer.LBRACKET:
		// List type: [T] - normalize to TypeApp("list", [T]) for uniform representation
		// DX-17 Phase 2: Syntactic sugar [T] normalizes to TypeApp at parse time
		// Note: uses lowercase "list" to match internal types.Builder convention
		startPos := p.curPos()
		p.nextToken() // consume LBRACKET
		elemType := p.parseType()
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		typ = &ast.TypeApp{
			Constructor: "list",
			Args:        []ast.Type{elemType},
			Pos:         startPos,
		}
		goto checkArrow

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
				var effects []ast.EffectAnnotation
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
				var effects []ast.EffectAnnotation
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

checkArrow:
	// M-TAINT-TYPES: Check for label suffix T<label> or refinement T{not IDENT}
	typ = p.parseLabelOrRefinementSuffix(typ)

	// S-ARROWTYPE: Check for function type arrow (int -> bool)
	if p.peekTokenIs(lexer.ARROW) {
		startPos := typ.Position()
		p.nextToken() // consume current token (move to ARROW)

		// Check if strict syntax mode is enabled
		if p.strictSyntaxMode {
			p.reportSugarError("ARROWTYPE", "T -> U", "funcType T U")
			// Return incomplete type to avoid cascading errors
			return typ
		}

		// Sugar is allowed - mark that it was used
		p.sugarUsed = true

		p.nextToken()               // move past ARROW
		returnType := p.parseType() // Right-associative: recursively parse return type

		// Parse optional effect annotation: int -> string ! {IO}
		var effects []ast.EffectAnnotation
		if p.peekTokenIs(lexer.BANG) {
			p.nextToken() // move to BANG
			effects = p.parseEffectAnnotation()
		}

		// Desugar to FuncType
		return &ast.FuncType{
			Params:  []ast.Type{typ},
			Return:  returnType,
			Effects: effects,
			Pos:     startPos,
		}
	}

	// No arrow, return primary type as-is
	return typ
}

// parseRecordTypeExpr parses a record type expression that can appear in type positions
// Example: { street: string, city: string }
// Open record: { name: string | r } - accepts records with at least 'name' field
// This is used for nested record types like: type User = { addr: { street: string } }
func (p *Parser) parseRecordTypeExpr() ast.Type {
	startPos := p.curPos()

	if !p.curTokenIs(lexer.LBRACE) {
		p.report("PAR_TYPE_LBRACE_EXPECTED", "expected '{' for record type", "Add '{' to start record type")
		return nil
	}
	p.nextToken() // consume LBRACE

	var fields []*ast.RecordField
	var rowVar *ast.TypeVar

	if !p.curTokenIs(lexer.RBRACE) {
		// Parse first field
		field := p.parseRecordFieldDef()
		if field != nil {
			fields = append(fields, field)
		}
		p.nextToken() // advance past the field we just parsed

		// Parse remaining fields or row variable
		for p.curTokenIs(lexer.COMMA) {
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RBRACE) {
				break // trailing comma
			}
			// M-GAP4: Check for ellipsis sugar after comma: { a: T, ... }
			if p.curTokenIs(lexer.ELLIPSIS) {
				p.sugarUsed = true
				rowVar = &ast.TypeVar{
					Name: p.freshRowVarName(),
					Pos:  p.curPos(),
				}
				p.nextToken() // consume ELLIPSIS
				break         // stop field parsing
			}
			field := p.parseRecordFieldDef()
			if field != nil {
				fields = append(fields, field)
			}
			p.nextToken() // advance past the field
		}

		// M-GAP4: Check for row variable syntax: { field: T | r }
		// Also handle sugar syntax: { field: T, .. } (desugars to fresh row variable)
		if p.curTokenIs(lexer.PIPE) {
			p.nextToken() // consume PIPE
			if !p.curTokenIs(lexer.IDENT) {
				p.report("PAR_ROW_VAR_EXPECTED", "expected row variable name after '|'", "Add row variable name (e.g., '| r')")
			} else {
				rowVar = &ast.TypeVar{
					Name: p.curToken.Literal,
					Pos:  p.curPos(),
				}
				p.nextToken() // consume row variable name
			}
		} else if p.curTokenIs(lexer.ELLIPSIS) {
			// M-GAP4: Sugar syntax {a: T, ..} desugars to {a: T | _rN} with fresh row variable
			p.sugarUsed = true
			rowVar = &ast.TypeVar{
				Name: p.freshRowVarName(),
				Pos:  p.curPos(),
			}
			p.nextToken() // consume ELLIPSIS
		}
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.report("PAR_TYPE_RBRACE_MISSING", "expected '}' to close record type", "Add '}' to close record type")
	}

	return &ast.RecordType{
		Fields: fields,
		Row:    rowVar,
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

// parseDeriving parses a deriving clause: deriving (Eq) or deriving (Eq, Ord)
// Returns a list of DeriveKind values for which type classes to derive
func (p *Parser) parseDeriving() []ast.DeriveKind {
	// We're already at DERIVING token
	if !p.curTokenIs(lexer.DERIVING) {
		return nil
	}
	p.nextToken() // consume DERIVING

	// Expect LPAREN
	if !p.curTokenIs(lexer.LPAREN) {
		p.reportExpected(lexer.LPAREN, "Add '(' after 'deriving'")
		return nil
	}
	p.nextToken() // consume LPAREN

	var derives []ast.DeriveKind

	// Parse derive list: Eq, Ord, Show, etc.
	if !p.curTokenIs(lexer.RPAREN) {
		kind := p.parseDeriveName()
		if kind != ast.DeriveNone {
			derives = append(derives, kind)
		}

		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // move to COMMA
			p.nextToken() // consume COMMA
			if p.curTokenIs(lexer.RPAREN) {
				break // trailing comma
			}
			kind := p.parseDeriveName()
			if kind != ast.DeriveNone {
				derives = append(derives, kind)
			}
		}
	}

	// Expect RPAREN
	if !p.expectPeek(lexer.RPAREN) {
		p.report("PAR_DERIVING_RPAREN", "expected ')' to close deriving clause", "Add ')' after type class list")
	}

	return derives
}

// parseDeriveName parses a single derive name and returns the corresponding DeriveKind
func (p *Parser) parseDeriveName() ast.DeriveKind {
	if !p.curTokenIs(lexer.IDENT) {
		p.report("PAR_DERIVING_NAME", "expected type class name in deriving clause", "Add 'Eq', 'Ord', or other derivable class")
		return ast.DeriveNone
	}

	name := p.curToken.Literal
	switch name {
	case "Eq":
		return ast.DeriveEq
	default:
		p.report("PAR_DERIVING_UNSUPPORTED", "unsupported deriving: "+name+", only 'Eq' is currently supported", "Use 'Eq'")
		return ast.DeriveNone
	}
}

// parseLabelOrRefinementSuffix attempts to parse a label or refinement suffix
// after the primary type has been parsed:
//
//	T<label>     → LabelledType{Base: T, Label: &LabelExpr{Name: label}}
//	T{not IDENT} → LabelledType{Base: T, Refinement: &RefinementExpr{NotLabel: IDENT}}
//
// Unsupported forms ({!label}, {label=...}, {not a && not b}) produce structured
// parse errors with a hint pointing to the MVP grammar.
// The cursor is assumed to be AT the last token of the base type on entry.
func (p *Parser) parseLabelOrRefinementSuffix(base ast.Type) ast.Type {
	if base == nil {
		return base
	}
	pos := base.Position()

	// T<label> — label annotation
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // advance: cur = LT
		if !p.peekTokenIs(lexer.IDENT) {
			p.report("PAR_LABEL_IDENT_EXPECTED",
				"expected label name after '<' in label annotation",
				"Use T<label> syntax, e.g. string<email>")
			return base
		}
		p.nextToken() // cur = IDENT (label name)
		labelName := p.curToken.Literal
		if !p.expectPeek(lexer.GT) {
			p.report("PAR_LABEL_GT_EXPECTED",
				"expected '>' to close label annotation",
				"Close label with '>': string<email>")
			return base
		}
		// cur = GT
		return &ast.LabelledType{
			Base:  base,
			Label: &ast.LabelExpr{Name: labelName, Pos: pos},
			Pos:   pos,
		}
	}

	// T{not IDENT} — refinement annotation.
	// Use 4-token lookahead to disambiguate the refinement form from a function
	// body that opens with a boolean negation. The two are otherwise identical
	// at peek=LBRACE peek2=NOT:
	//
	//   T{not LABEL}                        →  peek3=IDENT, peek4=RBRACE       (refinement)
	//   func name(...) -> T { not f(x) }    →  peek3=IDENT, peek4=LPAREN/etc.  (body)
	//
	// The previous 2-token lookahead misclaimed the function-body case and
	// emitted PAR_REFINE_MVP — broke motoko_agent's idiomatic
	// `func is_extension_tool_call(...) -> bool { not is_native(c.tool) }`.
	// See M-PARSER-REFINEMENT-LOOKAHEAD (v0.15.2) for the regression history.
	if p.peekTokenIs(lexer.LBRACE) {
		// Safe to enter the refinement path only if we see a complete
		//   `{ not IDENT }` form
		// or the malformed `{ ! ... }` BANG form (still ours to diagnose).
		// Everything else (record literals, function bodies, conjunctions
		// like `{not a && not b}`) defers to the appropriate downstream parser.
		isWellFormedNot := p.peek2TokenIs(lexer.NOT) &&
			p.peek3TokenIs(lexer.IDENT) &&
			p.peek4TokenIs(lexer.RBRACE)
		isMalformedBang := p.peek2TokenIs(lexer.BANG)
		if !isWellFormedNot && !isMalformedBang {
			return base
		}

		p.nextToken() // cur = LBRACE

		if p.peekTokenIs(lexer.NOT) {
			p.nextToken() // cur = NOT
			if !p.peekTokenIs(lexer.IDENT) {
				p.report("PAR_REFINE_IDENT_EXPECTED",
					"expected label name after 'not' in refinement",
					"Use T{not LABEL} syntax, e.g. string{not email} — MVP grammar: only 'not IDENT' is supported")
				p.skipToRBrace()
				return base
			}
			p.nextToken() // cur = IDENT (label name)
			labelName := p.curToken.Literal
			if p.peekTokenIs(lexer.RBRACE) {
				p.nextToken() // cur = RBRACE
				return &ast.LabelledType{
					Base:       base,
					Refinement: &ast.RefinementExpr{NotLabel: labelName, Pos: pos},
					Pos:        pos,
				}
			}
			// Something follows the label before } — conjunction or other unsupported form
			p.report("PAR_REFINE_MVP",
				"unsupported refinement form — MVP grammar only supports 'not IDENT'",
				"Use T{not LABEL} with a single label, e.g. string{not email}. Conjunctions like {not a && not b} are not supported in MVP.")
			p.skipToRBrace()
			return base
		}

		// peek2=BANG branch: {!email} — common mistake
		p.report("PAR_REFINE_BANG",
			"'!' is not valid in refinements — use the 'not' keyword",
			"Use T{not LABEL} syntax, e.g. string{not email} (MVP grammar: only 'not IDENT')")
		p.skipToRBrace()
		return base
	}

	return base
}

// skipToRBrace advances the parser until it finds a RBRACE or EOF,
// used to recover after a malformed refinement expression.
func (p *Parser) skipToRBrace() {
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		p.nextToken()
	}
}
