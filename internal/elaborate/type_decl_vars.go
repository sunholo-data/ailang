package elaborate

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/types"
)

// checkTypeDeclVars rejects a type declaration whose body uses a lowercase
// name that is neither one of its own `[..]` parameters nor a type in scope
// (TC_UNBOUND_TYPE_003, #2211 follow-up). In a function signature such a name
// is an implicitly quantified variable, supplied at each call. A type
// declaration has no caller, so `type Count = integer` (a typo for int) or
// `type Any = a` would silently make the type stand for any type.
//
// Two shapes reach here: a TypeVar the parser made for an undeclared
// lowercase name (rule 4 of parser_type_names.go; a bound parameter is a
// TypeVar too, and is allowed), and a lowercase SimpleType imported by name
// that is a value, not a type — the elaborator would read it as a variable.
// The second needs the module's imports, so a file-derived scope skips it.
func (e *Elaborator) checkTypeDeclVars(decl *ast.TypeDecl) error {
	params := make(map[string]bool, len(decl.TypeParams))
	for _, p := range decl.TypeParams {
		params[p] = true
	}
	if e.typeScope == nil {
		e.ensureTypeScope(nil)
	}
	var bad *ast.Pos
	var badName string
	visit := func(name string, pos ast.Pos) {
		if bad == nil {
			p := pos
			bad, badName = &p, name
		}
	}
	walkDeclTypes(decl.Definition, func(t ast.Type) {
		switch x := t.(type) {
		case *ast.TypeVar:
			if !params[x.Name] {
				visit(x.Name, x.Pos)
			}
		case *ast.SimpleType:
			if e.typeScopeImports && isLowerName(x.Name) && !ast.IsBuiltinTypeName(x.Name) && !e.typeScope.Declares(x.Name) {
				visit(x.Name, x.Pos)
			}
		}
	})
	if bad == nil {
		return nil
	}
	return fmt.Errorf("at %s: %w", e.sourcePos(*bad), e.typeDeclVarError(decl, badName))
}

// typeDeclVarError builds the TC_UNBOUND_TYPE_003 message: the parameter fix,
// and the type the name most likely meant (an import of it when an imported
// module exports it but it is not imported by name, else a built-in or a type
// in scope within edit distance 2 or differing only by case).
func (e *Elaborator) typeDeclVarError(decl *ast.TypeDecl, name string) error {
	params := append(append([]string{}, decl.TypeParams...), name)
	fix := fmt.Sprintf("write `type %s[%s] = …` to make it a parameter", decl.Name, strings.Join(params, ", "))
	var hint string
	if e.typeScope.Declares(name) {
		if r, _ := e.typeScope.Lookup(name); r.Module != "" && r.Module != e.typeScope.Module {
			hint = fmt.Sprintf(", or import the type: import %s (%s)", r.Module, name)
		}
	} else {
		if best := closestTypeName(name, decl.Name, append(ast.BuiltinTypeNames(), e.typeScope.Names()...)); best != "" {
			hint = fmt.Sprintf(", or did you mean the type `%s`?", best)
		}
	}
	return fmt.Errorf("%s: type variable '%s' is not a parameter of %s: %s%s",
		types.TC_UNBOUND_TYPE_003, name, decl.Name, fix, hint)
}

// closestTypeName picks the type a misspelt name most likely meant: the same
// name up to case, else (for names of four or more letters, where an edit
// distance means something) importhint.Closest. The declaration's own name
// and the unit type are never suggested.
func closestTypeName(name, declName string, cands []string) string {
	keep := make([]string, 0, len(cands))
	for _, c := range cands {
		if c != declName && c != "()" {
			keep = append(keep, c)
		}
	}
	for _, c := range keep {
		if c != name && strings.EqualFold(c, name) {
			return c
		}
	}
	if len(name) < 4 {
		return ""
	}
	return importhint.Closest(name, keep)
}

// walkDeclTypes calls f on every type node in a type declaration's body:
// constructor fields, record fields and an alias target, recursively.
func walkDeclTypes(def ast.TypeDef, f func(ast.Type)) {
	switch d := def.(type) {
	case *ast.AlgebraicType:
		for _, c := range d.Constructors {
			for _, fl := range c.Fields {
				walkType(fl.Type, f)
			}
		}
	case *ast.RecordType:
		walkType(d, f)
	case *ast.TypeAlias:
		walkType(d.Target, f)
	}
}

// walkType visits t and its sub-types (an ast.Type is a tree; no cycles).
func walkType(t ast.Type, f func(ast.Type)) {
	if t == nil {
		return
	}
	f(t)
	switch x := t.(type) {
	case *ast.FuncType:
		for _, p := range x.Params {
			walkType(p, f)
		}
		walkType(x.Return, f)
	case *ast.ListType:
		walkType(x.Element, f)
	case *ast.ArrayType:
		walkType(x.Element, f)
	case *ast.TupleType:
		for _, el := range x.Elements {
			walkType(el, f)
		}
	case *ast.TypeApp:
		for _, a := range x.Args {
			walkType(a, f)
		}
	case *ast.RecordType:
		for _, fl := range x.Fields {
			walkType(fl.Type, f)
		}
	case *ast.LabelledType:
		walkType(x.Base, f)
	}
}
