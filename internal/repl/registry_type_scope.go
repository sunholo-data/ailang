package repl

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/types"
)

// typeScope builds the type-name resolver of a module loaded into the
// registry, with the precedence of the CLI pipeline's buildTypeScope
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3): the module's own types, then types named
// by a selective import, then the prelude (Option, Result), then types of
// directly imported modules, then any other loaded module's types (the
// registry has no import closure; every loaded module is reachable). Anything
// else resolves through the builtin home table, else to this module.
func (mr *ModuleRegistry) typeScope(name string, file *ast.File) *types.TypeScope {
	s := types.NewTypeScope(name)
	for _, n := range elaborate.LocalTypeNames(file) {
		s.Bind(n, types.TypeRef{Module: name, Name: n})
	}
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	declares := func(modName, typeName string) bool {
		mod, ok := mr.modules[modName]
		if !ok {
			return false
		}
		for _, t := range mod.TypeNames {
			if t == typeName {
				return true
			}
		}
		return false
	}
	if file != nil {
		for _, imp := range file.Imports {
			for _, sym := range imp.Symbols {
				if !declares(imp.Path, sym) {
					continue
				}
				bind := sym
				if alias, ok := imp.SymbolAliases[sym]; ok {
					bind = alias
				}
				s.Bind(bind, types.TypeRef{Module: imp.Path, Name: sym})
			}
		}
	}
	s.Bind("Option", types.TypeRef{Module: "std/option", Name: "Option"})
	s.Bind("Result", types.TypeRef{Module: "std/result", Name: "Result"})
	bindModule := func(modName string) {
		if mod, ok := mr.modules[modName]; ok {
			for _, t := range mod.TypeNames {
				s.Bind(t, types.TypeRef{Module: modName, Name: t})
			}
		}
	}
	if file != nil {
		for _, imp := range file.Imports {
			bindModule(imp.Path)
		}
	}
	names := make([]string, 0, len(mr.modules))
	for n := range mr.modules {
		if n != name {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		bindModule(n)
	}
	return s
}
