package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// TC_UNBOUND_TYPE_002 — a bare lowercase name read as a type variable that
// might have been meant as a type (#2211). Whether a name in type position is
// a variable is decided by scope (parser/parser_type_names.go): a lowercase
// name is a type only when the file declares it or imports it by name. Two
// readings stay type variables, as they were before #2211, and warn:
//
//   - an implicit variable named like a type that an imported module exports
//     (`import p/a (start)` … `func keep(x: t) -> t` while p/a exports
//     `type t`). An error here would let a library break every importer just
//     by exporting `type t`;
//   - a name imported by name that is a value, not a type
//     (`import std/list (map)` … `(map) -> map`).
//
// The pass reads the parser's record of those uses (File.LowerTypeNames)
// from each module's PARSED source, which the loader re-parses on every run,
// and the imported modules' interfaces, which a cache hit restores — so the
// warning is the same cold and warm.

// TypeNameWarning is the non-blocking TC_UNBOUND_TYPE_002.
type TypeNameWarning struct {
	Location ast.Pos
	Name     string
	From     string // the module that exports the type, or that the value is imported from
	Value    bool   // the name is imported by name as a value
}

var _ elaborate.Warning = (*TypeNameWarning)(nil)

// Position exposes the use's 1-indexed line/col.
func (w *TypeNameWarning) Position() (line, col int) { return w.Location.Line, w.Location.Column }

// Code returns the stable diagnostic code.
func (w *TypeNameWarning) Code() string { return types.TC_UNBOUND_TYPE_002 }

// WarningFile names the file holding the use.
func (w *TypeNameWarning) WarningFile() string { return w.Location.File }

// Message is the warning without its location and fix.
func (w *TypeNameWarning) Message() string {
	if w.Value {
		return fmt.Sprintf("'%s' is treated as a type variable here; it is imported by name from %s, but as a value, not a type", w.Name, w.From)
	}
	return fmt.Sprintf("'%s' is treated as a type variable here; %s exports a type '%s', which is not imported by name", w.Name, w.From, w.Name)
}

// Suggestion is the one-line fix.
func (w *TypeNameWarning) Suggestion() string {
	if w.Value {
		return fmt.Sprintf("write the type you mean, or rename the type variable so it does not share the name of the imported value %s", w.Name)
	}
	return fmt.Sprintf("import it by name, import %s (%s), if you meant the type; otherwise rename the type variable", w.From, w.Name)
}

// String renders the warning for stderr.
func (w *TypeNameWarning) String() string {
	return fmt.Sprintf("%s at %s\n  %s.\n  Fix: %s", types.TC_UNBOUND_TYPE_002, w.Location.String(), w.Message(), w.Suggestion())
}

// exportsType reports whether an interface exports a type named name.
func exportsType(ifc *iface.Iface, name string) bool {
	if ifc == nil {
		return false
	}
	if _, ok := ifc.Types[name]; ok {
		return true
	}
	for _, c := range ifc.Constructors {
		if c != nil && c.TypeName == name {
			return true
		}
	}
	return false
}

// DetectTypeNameWarnings returns TC_UNBOUND_TYPE_002 for every compiled unit,
// once per (declaration, name), in module order then source order. getIface
// looks up a module's interface by path.
func DetectTypeNameWarnings(units map[string]*CompileUnit, getIface func(string) *iface.Iface) []elaborate.Warning {
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	pkgName := ""
	if currentPackageManifest != nil {
		pkgName = currentPackageManifest.Package.Name
	}
	var out []elaborate.Warning
	for _, id := range ids {
		u := units[id]
		if u == nil || u.Surface == nil || len(u.Surface.LowerTypeNames) == 0 {
			continue
		}
		// An installed dependency's names are that package's to fix (as for
		// TC_DEPRECATED_001).
		if strings.HasPrefix(id, "pkg/") && !isOwnPackageModule(id, pkgName) {
			continue
		}
		out = append(out, typeNameWarnings(u.Surface, getIface)...)
	}
	return out
}

func typeNameWarnings(file *ast.File, getIface func(string) *iface.Iface) []elaborate.Warning {
	// byName: what each name imported by name binds (module, symbol).
	type binding struct{ module, sym string }
	byName := map[string]binding{}
	for _, imp := range file.Imports {
		for _, sym := range imp.Symbols {
			bind := sym
			if alias, ok := imp.SymbolAliases[sym]; ok && alias != "" {
				bind = alias
			}
			byName[bind] = binding{imp.Path, sym}
		}
	}
	seen := map[string]bool{}
	var out []elaborate.Warning
	for _, ref := range file.LowerTypeNames {
		key := ref.Decl + "\x00" + ref.Name
		if seen[key] {
			continue
		}
		switch ref.Kind {
		case ast.LowerImported:
			b, ok := byName[ref.Name]
			ifc := getIface(b.module)
			if !ok || ifc == nil || exportsType(ifc, b.sym) {
				continue
			}
			seen[key] = true
			out = append(out, &TypeNameWarning{Location: ref.Pos, Name: ref.Name, From: b.module, Value: true})
		case ast.LowerImplicitVar:
			for _, imp := range file.Imports {
				if exportsType(getIface(imp.Path), ref.Name) {
					seen[key] = true
					out = append(out, &TypeNameWarning{Location: ref.Pos, Name: ref.Name, From: imp.Path})
					break
				}
			}
		}
	}
	return out
}

// withLowerTypeNote adds a note to a module's type error when a lowercase type
// the module declares is used in a signature that also has implicit type
// variables (`type acc = {...}` and `(acc, elem) -> acc`): there the name
// reads like a variable but is the declared type (#2211), and the error itself
// may surface far away, at a use of the function. The error is wrapped, so
// its code and errors.As are unchanged.
func withLowerTypeNote(err error, file *ast.File) error {
	if err == nil || file == nil {
		return err
	}
	hasVar := map[string]bool{}
	for _, ref := range file.LowerTypeNames {
		if ref.Kind == ast.LowerImplicitVar {
			hasVar[ref.Decl] = true
		}
	}
	declPos := map[string]ast.Pos{}
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, d := range list {
			if td, ok := d.(*ast.TypeDecl); ok {
				declPos[td.Name] = td.Pos
			}
		}
	}
	var notes []string
	seen := map[string]bool{}
	for _, ref := range file.LowerTypeNames {
		if ref.Kind != ast.LowerLocal || !hasVar[ref.Decl] || seen[ref.Name] {
			continue
		}
		seen[ref.Name] = true
		notes = append(notes, fmt.Sprintf("'%s' in the signature of %s is the type declared at %s (type %s = ...), not a type variable",
			ref.Name, ref.Decl, declPos[ref.Name].String(), ref.Name))
	}
	if len(notes) == 0 {
		return err
	}
	return fmt.Errorf("%w\n  Note: %s. A lowercase name declared as a type is that type in every signature of the module; rename the type or the variable if a variable was meant", err, strings.Join(notes, "; "))
}
