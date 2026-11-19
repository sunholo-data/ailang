package pipeline

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/types"
)

// buildTypeScope builds the one type-name resolver of a module
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3). The precedence order itself lives in one
// place, types.TypeScope; this only fills its steps: the module's own types,
// the explicit selective imports, the types of every direct dependency (any
// import form) and the types declared anywhere in the import closure.
func buildTypeScope(file *ast.File, modID string, imports *moduleImports, modLinker *link.ModuleLinker, reach map[string]bool) *types.TypeScope {
	s := types.NewTypeScope(modID)
	for _, name := range elaborate.LocalTypeNames(file) {
		s.BindLocal(name)
	}
	for name, ref := range imports.ExplicitTypes {
		s.BindExplicit(name, ref)
	}
	direct := make(map[string]bool)
	if file != nil {
		for _, imp := range file.Imports {
			direct[imp.Path] = true
			forIfaceTypes(modLinker.GetIface(imp.Path), s.AddDirect)
		}
	}
	for modPath, ifc := range modLinker.GetLoadedModules() {
		if modPath == "$builtin" || modPath == modID || direct[modPath] || (reach != nil && !reach[modPath]) {
			continue
		}
		forIfaceTypes(ifc, s.AddClosure)
	}
	return s
}

// forIfaceTypes calls add for every type an interface declares: its exported
// types and the ADTs of its constructors.
func forIfaceTypes(ifc *iface.Iface, add func(types.TypeRef)) {
	if ifc == nil {
		return
	}
	for name := range ifc.Types {
		add(types.TypeRef{Module: ifc.Module, Name: name})
	}
	for _, c := range ifc.Constructors {
		if c != nil {
			add(types.TypeRef{Module: ifc.Module, Name: c.TypeName})
		}
	}
}
