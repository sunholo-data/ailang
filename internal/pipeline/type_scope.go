package pipeline

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/types"
)

// preludeTypeHomes are the types the implicit prelude provides
// (internal/loader/prelude_imports.go), §4.3 rule 4.
var preludeTypeHomes = []types.TypeRef{
	{Module: "std/option", Name: "Option"},
	{Module: "std/result", Name: "Result"},
}

// buildTypeScope builds the one type-name resolver of a module
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3). Bindings are made in precedence order and
// the first binding of a name wins:
//
//  1. type parameters — the elaborator handles these (lowercase TypeVar);
//  2. a type this module declares;
//  3. a type bound by an explicit selective import (`import M (T as U)` binds U);
//  4. the prelude (Option, Result);
//  5. a type exported by a direct dependency (any import form, in source order) —
//     the auto-import channel that lets `-> Json` work without naming Json;
//  6. a type exported by a module of the import closure (sorted module order),
//     which today's bare-name identity let a module name without importing.
//
// Anything else resolves through the builtin home table, else to this module
// (an opaque local name, as before; TC_UNBOUND_TYPE_001 is M3). When rule 5 or
// 6 has two distinct candidates the first is taken (TC_AMBIGUOUS_TYPE_001 is
// M3); either way the name denotes ONE declaration, so unification still
// rejects a value of the other.
func buildTypeScope(file *ast.File, modID string, imports *moduleImports, modLinker *link.ModuleLinker, reach map[string]bool) *types.TypeScope {
	s := types.NewTypeScope(modID)
	for _, name := range elaborate.LocalTypeNames(file) {
		s.Bind(name, types.TypeRef{Module: modID, Name: name})
	}
	explicit := make([]string, 0, len(imports.ExplicitTypes))
	for name := range imports.ExplicitTypes {
		explicit = append(explicit, name)
	}
	sort.Strings(explicit)
	for _, name := range explicit {
		s.Bind(name, imports.ExplicitTypes[name])
	}
	for _, ref := range preludeTypeHomes {
		s.Bind(ref.Name, ref)
	}
	if file != nil {
		for _, imp := range file.Imports {
			bindIfaceTypes(s, modLinker.GetIface(imp.Path))
		}
	}
	loaded := modLinker.GetLoadedModules()
	for _, modPath := range sortedIfaceKeys(loaded) {
		if modPath == "$builtin" || modPath == modID || (reach != nil && !reach[modPath]) {
			continue
		}
		bindIfaceTypes(s, loaded[modPath])
	}
	return s
}

// bindIfaceTypes binds every type an interface declares: its exported types
// and the ADTs of its constructors, in sorted order.
func bindIfaceTypes(s *types.TypeScope, ifc *iface.Iface) {
	if ifc == nil {
		return
	}
	names := make(map[string]bool, len(ifc.Types)+len(ifc.Constructors))
	for name := range ifc.Types {
		names[name] = true
	}
	for _, c := range ifc.Constructors {
		if c != nil {
			names[c.TypeName] = true
		}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, n := range sorted {
		s.Bind(n, types.TypeRef{Module: ifc.Module, Name: n})
	}
}
