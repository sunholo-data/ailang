package pipeline

import (
	"reflect"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// Deprecated uses in the parts of a module that `check` never elaborates into
// Core: named `test "…" { }` blocks, top-level `property "…" { }` blocks, a
// function's inline `tests [...]` rows and its `properties [...]` (forall)
// clauses. The Core pass cannot see these, so a use there was silent until
// alias removal turned it into a hard IMP010 (ailang-packages: 9 `abs_Float`
// uses in test blocks).
//
// This pass resolves names from the PARSED source alone — the module's import
// list and the binders in scope — so it does not depend on those blocks being
// elaborated or type-checked (they are not: `check` does not type-check named
// test blocks at all, which is a separate, pre-existing issue). It honours
// shadowing by lambda and function-literal parameters, `let`/`letrec`, `let`s
// earlier in a block, match-arm pattern variables and forall binders.
//
// Every other expression-holding construct reaches Core: function bodies,
// requires/ensures/invariant clauses (DeclMeta.Contracts), module-level `let`s
// and top-level expressions. TestDeprecatedUseCoverage pins that split with one
// fixture per construct; TestDeprecatedASTConstructsClassified reads the
// statement and expression kinds from internal/ast's own type definitions and
// fails on any it has not classified; and a statement kind nobody has told this
// pass about is still searched (TestDeprecatedASTPassSearchesEveryStatementKind).

// importedNames maps the names a module can use without qualification to the
// global they denote, and module aliases to their module path. `import M`
// with no list and no alias binds nothing.
func importedNames(f *ast.File) (names map[string]core.GlobalRef, aliases map[string]string) {
	names, aliases = map[string]core.GlobalRef{}, map[string]string{}
	for _, imp := range f.Imports {
		if imp == nil {
			continue
		}
		for _, sym := range imp.Symbols {
			local := sym
			if a, ok := imp.SymbolAliases[sym]; ok && a != "" {
				local = a
			}
			names[local] = core.GlobalRef{Module: imp.Path, Name: sym}
		}
		if imp.ModuleAlias != "" {
			aliases[imp.ModuleAlias] = imp.Path
		}
	}
	return names, aliases
}

// deprecatedUsesOutsideCore returns the deprecated uses in u's AST-only
// constructs.
func deprecatedUsesOutsideCore(u *CompileUnit, idx map[core.GlobalRef]deprecation) []*DeprecatedUseWarning {
	if u == nil || u.Surface == nil {
		return nil
	}
	names, aliases := importedNames(u.Surface)
	relevant := false
	for _, ref := range names {
		if _, ok := idx[ref]; ok {
			relevant = true
			break
		}
	}
	for _, mod := range aliases {
		for ref := range idx {
			if ref.Module == mod {
				relevant = true
			}
		}
	}
	if !relevant {
		return nil
	}
	w := &astUseWalker{names: names, aliases: aliases, idx: idx}
	for _, fd := range u.Surface.Funcs {
		if fd == nil {
			continue
		}
		for _, tc := range fd.Tests {
			if tc == nil {
				continue
			}
			for _, in := range tc.Inputs {
				w.walk(in, nil)
			}
			w.walk(tc.Expected, nil)
		}
		for _, p := range fd.Properties {
			// requires/ensures/invariant clauses are elaborated into Core.
			if p != nil && p.Kind == ast.PropertyKind {
				w.walkProperty(p)
			}
		}
	}
	for _, st := range u.Surface.Statements {
		switch n := st.(type) {
		case *ast.TestDecl:
			// A test body is a sequence: `let y = …;` scopes over what follows.
			w.walk(&ast.Block{Exprs: n.Body}, nil)
		case *ast.PropertyDecl:
			w.walkProperty(n.Property)
		case *ast.FuncDecl, *ast.TypeDecl:
			// Functions: bodies and contracts are in Core, tests and
			// properties are walked above. Type declarations hold no
			// expressions.
		default:
			// A top-level expression (module `let`, bare expression) is
			// elaborated into Core. Anything else — class defaults,
			// instance methods, and any statement kind added later — is
			// searched here, so a new construct is never silently skipped.
			if _, isExpr := st.(ast.Expr); isExpr {
				continue
			}
			w.walkFields(reflect.ValueOf(st), nil)
		}
	}
	return w.found
}

type astUseWalker struct {
	names   map[string]core.GlobalRef
	aliases map[string]string
	idx     map[core.GlobalRef]deprecation
	found   []*DeprecatedUseWarning
}

func (w *astUseWalker) record(pos ast.Pos, ref core.GlobalRef) {
	d, ok := w.idx[ref]
	if !ok {
		return
	}
	w.found = append(w.found, &DeprecatedUseWarning{
		Location: pos, Module: ref.Module, Name: ref.Name,
		Replacement: d.replacement, Note: d.note,
	})
}

func (w *astUseWalker) walkProperty(p *ast.Property) {
	if p == nil {
		return
	}
	scope := map[string]bool{}
	for _, b := range p.Binders {
		if b != nil {
			scope[b.Name] = true
		}
	}
	w.walk(p.Expr, scope)
}

func withNames(scope map[string]bool, names ...string) map[string]bool {
	out := make(map[string]bool, len(scope)+len(names))
	for k := range scope {
		out[k] = true
	}
	for _, n := range names {
		out[n] = true
	}
	return out
}

func paramNames(ps []*ast.Param) []string {
	var out []string
	for _, p := range ps {
		if p != nil {
			out = append(out, p.Name)
		}
	}
	return out
}

// walk visits an expression. Binding forms extend the scope; every other node
// is traversed field by field (walkFields), so a new expression kind is
// searched without being listed here.
func (w *astUseWalker) walk(n interface{}, scope map[string]bool) {
	switch x := n.(type) {
	case nil:
		return
	case *ast.Identifier:
		if x == nil || scope[x.Name] {
			return
		}
		if ref, ok := w.names[x.Name]; ok {
			w.record(x.Pos, ref)
		}
	case *ast.RecordAccess:
		if x == nil {
			return
		}
		if id, ok := x.Record.(*ast.Identifier); ok && id != nil && !scope[id.Name] {
			if mod, ok := w.aliases[id.Name]; ok {
				w.record(x.Pos, core.GlobalRef{Module: mod, Name: x.Field})
				return
			}
		}
		w.walk(x.Record, scope)
	case *ast.Lambda:
		if x != nil {
			w.walk(x.Body, withNames(scope, paramNames(x.Params)...))
		}
	case *ast.FuncLit:
		if x != nil {
			w.walk(x.Body, withNames(scope, paramNames(x.Params)...))
		}
	case *ast.Let:
		if x != nil {
			w.walk(x.Value, scope)
			w.walk(x.Body, withNames(scope, x.Name))
		}
	case *ast.LetRec:
		if x != nil {
			inner := withNames(scope, x.Name)
			w.walk(x.Value, inner)
			w.walk(x.Body, inner)
		}
	case *ast.Block:
		if x == nil {
			return
		}
		s := scope
		for _, e := range x.Exprs {
			w.walk(e, s)
			switch b := e.(type) {
			case *ast.Let:
				if b != nil && b.Body == nil {
					s = withNames(s, b.Name)
				}
			case *ast.LetRec:
				if b != nil && b.Body == nil {
					s = withNames(s, b.Name)
				}
			}
		}
	case *ast.Match:
		if x == nil {
			return
		}
		w.walk(x.Expr, scope)
		for _, c := range x.Cases {
			if c == nil {
				continue
			}
			inner := withNames(scope, patternVars(c.Pattern)...)
			w.walk(c.Guard, inner)
			w.walk(c.Body, inner)
		}
	case *ast.ForallExpr:
		if x != nil {
			w.walk(x.Lo, scope)
			w.walk(x.Hi, scope)
			w.walk(x.Body, withNames(scope, x.Var))
		}
	default:
		w.walkFields(reflect.ValueOf(n), scope)
	}
}

var (
	posType    = reflect.TypeOf(ast.Pos{})
	spanType   = reflect.TypeOf(ast.Span{})
	exprType   = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	typeIfType = reflect.TypeOf((*ast.Type)(nil)).Elem()
)

// walkFields traverses any AST value generically: pointers, structs, slices,
// maps and interfaces, dispatching each ast.Expr it reaches back to walk.
// Positions and type annotations hold no uses and are skipped.
func (w *astUseWalker) walkFields(v reflect.Value, scope map[string]bool) {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		if v.Type().Implements(typeIfType) && !v.Elem().Type().Implements(exprType) {
			return
		}
		w.walk(v.Elem().Interface(), scope)
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Type().Implements(exprType) {
			w.walkStruct(v.Elem(), scope)
			return
		}
		if v.Type().Implements(typeIfType) {
			return
		}
		w.walkFields(v.Elem(), scope)
	case reflect.Struct:
		w.walkStruct(v, scope)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			w.walkFields(v.Index(i), scope)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			w.walkFields(iter.Value(), scope)
		}
	}
}

func (w *astUseWalker) walkStruct(v reflect.Value, scope map[string]bool) {
	if v.Type() == posType || v.Type() == spanType {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !v.Type().Field(i).IsExported() {
			continue
		}
		if f.Kind() == reflect.Interface && !f.IsNil() && f.Elem().Type().Implements(exprType) {
			w.walk(f.Elem().Interface(), scope)
			continue
		}
		if f.Kind() == reflect.Ptr && !f.IsNil() && f.Type().Implements(exprType) {
			w.walk(f.Interface(), scope)
			continue
		}
		w.walkFields(f, scope)
	}
}

// patternVars lists the variables a pattern binds: every Identifier in it
// (constructor names are not Identifiers).
func patternVars(p ast.Pattern) []string {
	var out []string
	var visit func(v reflect.Value)
	visit = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr:
			if v.IsNil() {
				return
			}
			if id, ok := v.Interface().(*ast.Identifier); ok {
				out = append(out, id.Name)
				return
			}
			visit(v.Elem())
		case reflect.Struct:
			if v.Type() == posType || v.Type() == spanType {
				return
			}
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					visit(v.Field(i))
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	if p != nil {
		visit(reflect.ValueOf(p))
	}
	return out
}
