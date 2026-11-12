package parser

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// What a bare name in type position means is decided by scope, not by the
// case of its first letter (#2211, #2240). In order:
//
//  1. a name bound by an enclosing type-parameter list — `func id[T](x: T)`,
//     `type Box[T] = Box(T)` — is that type variable, whatever its case, and
//     shadows any type of the same name. It cannot be applied to arguments
//     (`m[int]`: PAR_TYPE_PARAM_APPLIED — there are no higher-kinded types);
//  2. a built-in primitive (int, string, ...: ast.IsBuiltinTypeName) is that
//     type; a type parameter may not take one of these names
//     (PAR_TYPE_PARAM_BUILTIN);
//  3. a lowercase name this file declares as a type (`type point = ...`) or
//     imports by name (`import std/sem (sem_frame)`) is a type. An import by
//     name that turns out to be a value, not a type, is read as a type
//     variable by the elaborator (with warning TC_UNBOUND_TYPE_002), as before
//     — such a variable is an ast.SimpleType in the AST, so a pass that
//     collects type variables must also ask types.TypeScope.ReadsAsTypeVar
//     (see its comment for the contract);
//  4. any other lowercase name is an implicitly quantified type variable
//     (`func f(x: a) -> a`), as before;
//  5. any other name is a type constructor, resolved by the module's type
//     scope (types.TypeScope).
//
// Rules 1 and 3 used to be missing: an uppercase type parameter was a type
// constructor named T, and a lowercase declared type was a fresh type variable
// at every use, so its declaration was never enforced. Each use resolved by
// rule 3 or 4 is recorded in File.LowerTypeNames for the diagnostics that
// need the module's imports (pipeline/warn_type_names.go).

// pushTypeParams opens the scope of a declaration's type-parameter list. The
// caller pops it (popTypeParams) when the declaration ends.
func (p *Parser) pushTypeParams(params []string) {
	frame := make(map[string]bool, len(params))
	for _, name := range params {
		if ast.IsBuiltinTypeName(name) {
			p.report("PAR_TYPE_PARAM_BUILTIN",
				fmt.Sprintf("type parameter '%s' has the name of the built-in type %s", name, name),
				"Rename the type parameter (e.g. [a] or [T]) and use that name in the declaration")
			continue
		}
		frame[name] = true
	}
	p.typeParamScopes = append(p.typeParamScopes, frame)
}

// enterDecl records the top-level declaration being parsed (for
// File.LowerTypeNames) and returns the function that restores the previous one.
func (p *Parser) enterDecl(name string) func() {
	prev := p.curDecl
	p.curDecl = name
	return func() { p.curDecl = prev }
}

// popTypeParams closes the innermost type-parameter scope.
func (p *Parser) popTypeParams() {
	if n := len(p.typeParamScopes); n > 0 {
		p.typeParamScopes = p.typeParamScopes[:n-1]
	}
}

// boundTypeParam reports whether an enclosing type-parameter list binds name.
func (p *Parser) boundTypeParam(name string) bool {
	for i := len(p.typeParamScopes) - 1; i >= 0; i-- {
		if p.typeParamScopes[i][name] {
			return true
		}
	}
	return false
}

// typeNameNode builds the node for a bare (unapplied) name in type position,
// following the rules above.
func (p *Parser) typeNameNode(name string, pos ast.Pos) ast.Type {
	switch {
	case p.boundTypeParam(name):
		return &ast.TypeVar{Name: name, Pos: pos}
	case ast.IsBuiltinTypeName(name) || !isLowerTypeName(name):
		return &ast.SimpleType{Name: name, Pos: pos}
	}
	kind, isType := p.lowerTypeNames[name]
	if !isType {
		kind = ast.LowerImplicitVar
	}
	p.lowerRefs = append(p.lowerRefs, ast.LowerTypeName{Name: name, Pos: pos, Kind: kind, Decl: p.curDecl})
	if isType {
		return &ast.SimpleType{Name: name, Pos: pos}
	}
	return &ast.TypeVar{Name: name, Pos: pos}
}

// checkTypeParamNotApplied reports PAR_TYPE_PARAM_APPLIED for a type
// parameter written with type arguments (`func f[m](x: m[int])`). Before
// #2240 the parameter list was ignored, so `m[int]` silently named an opaque
// type `m` that no value could have.
func (p *Parser) checkTypeParamNotApplied(name string, pos ast.Pos) {
	if !p.boundTypeParam(name) {
		return
	}
	p.errors = append(p.errors, NewParserError("PAR_TYPE_PARAM_APPLIED", pos, p.curToken,
		fmt.Sprintf("type parameter '%s' cannot be applied to type arguments: higher-kinded type parameters are not supported", name),
		nil, fmt.Sprintf("Use a concrete type constructor (e.g. Option[int] or [int]) instead of %s[...]", name)))
}

func isLowerTypeName(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z'
}

// declareLowerTypeNames records the lowercase names a file declares as types,
// found by a token pre-scan so a use before the declaration resolves too
// (declarations may come in any order). Only `type <name>` sequences count;
// a field or label spelled `type` is followed by `:` or another token.
func (p *Parser) declareLowerTypeNames() {
	if p.l == nil {
		return
	}
	scan := p.l.Rewind()
	prev := lexer.Token{Type: lexer.EOF}
	for {
		tok := scan.NextToken()
		if tok.Type == lexer.EOF {
			return
		}
		if prev.Type == lexer.TYPE && tok.Type == lexer.IDENT && isLowerTypeName(tok.Literal) && !ast.IsBuiltinTypeName(tok.Literal) {
			p.addLowerTypeName(tok.Literal, ast.LowerLocal)
		}
		prev = tok
	}
}

// importLowerTypeNames records the lowercase names a selective import binds
// (the alias, for `import M (x as y)`). The parser cannot tell an imported
// type from an imported value, so in type position such a name is read as
// the import; the elaborator reads it as a type variable (warning
// TC_UNBOUND_TYPE_002) if the import is not a type.
func (p *Parser) importLowerTypeNames(imp *ast.ImportDecl) {
	if imp == nil {
		return
	}
	for _, sym := range imp.Symbols {
		bind := sym
		if alias, ok := imp.SymbolAliases[sym]; ok && alias != "" {
			bind = alias
		}
		if isLowerTypeName(bind) && !ast.IsBuiltinTypeName(bind) {
			p.addLowerTypeName(bind, ast.LowerImported)
		}
	}
}

func (p *Parser) addLowerTypeName(name string, kind ast.LowerTypeNameKind) {
	if p.lowerTypeNames == nil {
		p.lowerTypeNames = make(map[string]ast.LowerTypeNameKind)
	}
	// A declaration in the file wins over an import (types.TypeScope step 1).
	if prev, ok := p.lowerTypeNames[name]; ok && prev == ast.LowerLocal {
		return
	}
	p.lowerTypeNames[name] = kind
}
