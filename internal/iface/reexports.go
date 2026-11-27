package iface

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/types"
)

// Reexports is what a facade re-exports with `export import`
// (M-PKG-PARALLEL-AGENTS C1).
//
// The entries are kept OUT of the facade's Exports, Constructors and Types:
// those maps mean "declared here", and many passes walk every loaded
// interface's constructors to build ADT identity (match universe, factory
// keys, ambiguity). A re-exported constructor there would be a second, phantom
// ADT. Importers never need them either: the loader rewrites a selective
// import of a re-exported name to its origin. Only the rendered interface
// (signatures, interface_hash_v2), the digest and module-alias access read
// them.
type Reexports struct {
	// Origins maps the facade's name to the definition it stands for, after
	// following facades.
	Origins map[string]ReexportOrigin
	// Items holds the re-exported entries under the facade's names, so the
	// facade renders exactly as if it declared them: a module moved behind a
	// facade keeps its signature lines. A re-exported type brings its
	// constructors, as importing a type does.
	Items *Iface
}

// ReexportOrigin is the definition a re-exported name stands for.
type ReexportOrigin struct {
	Module  string // defining module
	Name    string // the name there
	MovedTo string // the facade line's @deprecated("path"), if any
}

// NewReexports returns an empty table for facade module.
func NewReexports(module string) *Reexports {
	return &Reexports{Origins: map[string]ReexportOrigin{}, Items: NewIface(module)}
}

// Lookup finds name among the interface's own entries and its re-exports:
// the function, type and constructor of that name (any may be nil), plus,
// for a type, its constructors.
func (i *Iface) Lookup(name string) (fn *IfaceItem, typ *TypeExport, ctor *ConstructorScheme, typeCtors map[string]*ConstructorScheme) {
	for _, src := range []*Iface{i, i.reexportItems()} {
		if src == nil {
			continue
		}
		if fn == nil {
			fn = src.Exports[name]
		}
		if typ == nil {
			typ = src.Types[name]
		}
		if ctor == nil {
			ctor = src.Constructors[name]
		}
	}
	if typ != nil {
		typeCtors = map[string]*ConstructorScheme{}
		for _, src := range []*Iface{i, i.reexportItems()} {
			if src == nil {
				continue
			}
			for cn, c := range src.Constructors {
				if c.TypeName == name {
					typeCtors[cn] = c
				}
			}
		}
	}
	return fn, typ, ctor, typeCtors
}

// LookupAlias is the alias body and params of type name, own or re-exported.
func (i *Iface) LookupAlias(name string) (types.Type, []string, bool) {
	for _, src := range []*Iface{i, i.reexportItems()} {
		if src == nil {
			continue
		}
		if t, ok := src.TypeAliases[name]; ok {
			return t, src.AliasParams[name], true
		}
	}
	return nil, nil, false
}

func (i *Iface) reexportItems() *Iface {
	if i == nil || i.Reexported == nil {
		return nil
	}
	return i.Reexported.Items
}

// withReexports is the interface as rendered: its own entries plus its
// re-exports. It never modifies i.
func (i *Iface) withReexports() *Iface {
	items := i.reexportItems()
	if items == nil {
		return i
	}
	out := *i
	out.Exports = mergeMap(i.Exports, items.Exports)
	out.Types = mergeMap(i.Types, items.Types)
	out.Constructors = mergeMap(i.Constructors, items.Constructors)
	out.TypeAliases = mergeMap(i.TypeAliases, items.TypeAliases)
	out.AliasParams = mergeMap(i.AliasParams, items.AliasParams)
	return &out
}

func mergeMap[V any](own, extra map[string]V) map[string]V {
	out := make(map[string]V, len(own)+len(extra))
	for k, v := range extra {
		out[k] = v
	}
	for k, v := range own { // a declaration wins over a re-export of the same name
		out[k] = v
	}
	return out
}

// reexportDigestLines renders the origin table for the digest, sorted. The
// re-exported entries themselves are the origin's, and the origin is in
// every importer's dependency closure, so its digest already covers them.
func (i *Iface) reexportDigestLines() []string {
	if i.Reexported == nil || len(i.Reexported.Origins) == 0 {
		return nil
	}
	lines := make([]string, 0, len(i.Reexported.Origins))
	for name, o := range i.Reexported.Origins {
		lines = append(lines, name+"="+o.Module+"."+o.Name+"@"+o.MovedTo)
	}
	sort.Strings(lines)
	return lines
}
