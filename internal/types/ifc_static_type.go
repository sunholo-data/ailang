package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_static_type.go — the TYPE half of the IFC label of an expression.
//
// M-IFC-DECLARED-RECORD-LABELS (#1523): a label written inside a type
// annotation — a declared record's field, an alias, an ADT constructor field, a
// list or tuple element, a let or lambda annotation — is a source label for
// every value of that type. The IFC pass (ifc_check.go) is a surface-AST
// analysis that never sees HM types, so this file names the static type of an
// expression where the surface AST alone determines it, and computes the
// labels that type declares:
//
//	labelOf(e) = flowOf(e) ⊔ deepLabel(staticType(e))
//
// Only module-local type declarations are resolved. An imported type name has
// no declaration here, so its deep label is ⊥ — the cross-module hole tracked by
// design_docs/planned/m-ifc-cross-module-labels.md.

// ifcTypes indexes the module-local type declarations the IFC pass can resolve.
type ifcTypes struct {
	decls map[string]*ast.TypeDecl // type name -> declaration
	ctors map[string]*ast.TypeDecl // ADT constructor name -> owning declaration
	deep  map[string]Label         // memoised deepLabel of a declared type name
}

func newIFCTypes(file *ast.File) *ifcTypes {
	ts := &ifcTypes{
		decls: make(map[string]*ast.TypeDecl),
		ctors: make(map[string]*ast.TypeDecl),
		deep:  make(map[string]Label),
	}
	for _, d := range file.Decls {
		td, ok := d.(*ast.TypeDecl)
		if !ok || td == nil {
			continue
		}
		ts.decls[td.Name] = td
		if alg, ok := td.Definition.(*ast.AlgebraicType); ok {
			for _, ctor := range alg.Constructors {
				if ctor != nil {
					ts.ctors[ctor.Name] = td
				}
			}
		}
	}
	return ts
}

// deepLabel is the join of every <label> appearing anywhere in t, resolving
// module-local declared type names transitively. A function type contributes
// only its return type: a function value carries what it returns (the same rule
// the Lambda case of the flow walk applies).
func (ts *ifcTypes) deepLabel(t ast.Type) Label {
	result := LabelBottom()
	ts.walkType(t, func(l Label) { result = LabelJoin(result, l) }, func(name string) {
		result = LabelJoin(result, ts.deepDecl(name))
	})
	return result
}

// deepDecl is the deep label of a declared type name: the join of the labels
// written directly in every declaration reachable from it. Computing the
// reachable set first (instead of recursing through deepLabel) keeps mutually
// recursive declarations exact and makes the memo safe.
func (ts *ifcTypes) deepDecl(name string) Label {
	if l, ok := ts.deep[name]; ok {
		return l
	}
	if _, ok := ts.decls[name]; !ok {
		return LabelBottom() // builtin or imported: no labels visible here
	}
	seen := map[string]bool{}
	stack := []string{name}
	result := LabelBottom()
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		td, ok := ts.decls[n]
		if !ok || seen[n] {
			continue
		}
		seen[n] = true
		for _, t := range declBodyTypes(td) {
			ts.walkType(t, func(l Label) { result = LabelJoin(result, l) }, func(ref string) {
				stack = append(stack, ref)
			})
		}
	}
	ts.deep[name] = result
	return result
}

// walkType reports every label written in t and every type name t references.
func (ts *ifcTypes) walkType(t ast.Type, onLabel func(Label), onName func(string)) {
	switch tt := t.(type) {
	case nil:
	case *ast.LabelledType:
		if tt.Label != nil {
			onLabel(LabelConst(tt.Label.Name))
		}
		ts.walkType(tt.Base, onLabel, onName)
	case *ast.SimpleType:
		onName(tt.Name)
	case *ast.RecordType:
		for _, f := range tt.Fields {
			if f != nil {
				ts.walkType(f.Type, onLabel, onName)
			}
		}
	case *ast.ListType:
		ts.walkType(tt.Element, onLabel, onName)
	case *ast.ArrayType:
		ts.walkType(tt.Element, onLabel, onName)
	case *ast.TupleType:
		for _, el := range tt.Elements {
			ts.walkType(el, onLabel, onName)
		}
	case *ast.TypeApp:
		onName(tt.Constructor)
		for _, arg := range tt.Args {
			ts.walkType(arg, onLabel, onName)
		}
	case *ast.FuncType:
		ts.walkType(tt.Return, onLabel, onName)
	case *ast.TypeVar:
		// A type parameter carries no label of its own.
	default:
		panic(fmt.Sprintf("ifc walkType: unhandled ast.Type %T", t))
	}
}

// declBodyTypes lists the type expressions a declaration's body is made of.
func declBodyTypes(td *ast.TypeDecl) []ast.Type {
	switch def := td.Definition.(type) {
	case *ast.RecordType:
		return []ast.Type{def}
	case *ast.TypeAlias:
		return []ast.Type{def.Target}
	case *ast.AlgebraicType:
		var out []ast.Type
		for _, ctor := range def.Constructors {
			if ctor == nil {
				continue
			}
			for _, f := range ctor.Fields {
				if f != nil {
					out = append(out, f.Type)
				}
			}
		}
		return out
	}
	return nil
}

// recordTypeOf resolves t to a record type through LabelledType wrappers and
// non-generic module-local declarations/aliases, or returns nil. A generic
// declaration is not resolved: its field types may mention type parameters
// whose instantiation the surface AST does not carry, so callers fall back to
// the whole value's label (sound, imprecise).
func (ts *ifcTypes) recordTypeOf(t ast.Type) *ast.RecordType {
	for depth := 0; depth < 16 && t != nil; depth++ {
		switch tt := t.(type) {
		case *ast.LabelledType:
			t = tt.Base
		case *ast.RecordType:
			return tt
		case *ast.SimpleType:
			td, ok := ts.decls[tt.Name]
			if !ok || len(td.TypeParams) > 0 {
				return nil
			}
			switch def := td.Definition.(type) {
			case *ast.RecordType:
				return def
			case *ast.TypeAlias:
				t = def.Target
			default:
				return nil
			}
		case *ast.TypeVar, *ast.FuncType, *ast.ListType, *ast.ArrayType, *ast.TupleType, *ast.TypeApp:
			return nil // not a record
		default:
			panic(fmt.Sprintf("ifc recordTypeOf: unhandled ast.Type %T", t))
		}
	}
	return nil
}

// fieldType is the declared type of field `name` in record type t, or nil.
func (ts *ifcTypes) fieldType(t ast.Type, name string) ast.Type {
	rt := ts.recordTypeOf(t)
	if rt == nil {
		return nil
	}
	for _, f := range rt.Fields {
		if f != nil && f.Name == name {
			return f.Type
		}
	}
	return nil
}

// staticType names the type of expr where the surface AST alone determines it,
// or returns nil. It never walks sub-expressions for labels, so it has no
// diagnostic side effects and may be called freely.
func (c *ifcChecker) staticType(expr ast.Expr, env ifcEnv) ast.Type {
	switch e := expr.(type) {
	case *ast.Identifier:
		if v, ok := env[e.Name]; ok {
			return v.typ
		}
	case *ast.FuncCall:
		name := calleeName(e.Func)
		if _, shadowed := env[name]; shadowed {
			return nil
		}
		if sig, ok := c.sigs[name]; ok {
			return sig.decl.ReturnType
		}
		if td, ok := c.types.ctors[name]; ok {
			return &ast.SimpleType{Name: td.Name, Pos: e.Pos}
		}
	case *ast.RecordAccess:
		return c.types.fieldType(c.staticType(e.Record, env), e.Field)
	}
	return nil
}

// sameType reports whether two annotations are written identically. String()
// renders labels in place (`string<authcode>`, field by field), so two types
// that differ only in where a label sits are NOT the same; a declared name is
// not the same as its expansion (the conservative direction).
func sameType(a, b ast.Type) bool {
	return a != nil && b != nil && a.String() == b.String()
}
