package repl

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/types"
)

// typeScope builds the type-name resolver of a module loaded into the
// registry. The precedence is types.TypeScope's (the CLI pipeline's): the
// module's own types, types named by a selective import, built-ins and the
// prelude, the types of directly imported modules, then the types of every
// other loaded module (the registry has no import closure; every loaded
// module is reachable). Two candidates at one step are TC_AMBIGUOUS_TYPE_001.
func (mr *ModuleRegistry) typeScope(name string, file *ast.File) *types.TypeScope {
	s := types.NewTypeScope(name)
	for _, n := range elaborate.LocalTypeNames(file) {
		s.BindLocal(n)
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
	direct := map[string]bool{}
	if file != nil {
		for _, imp := range file.Imports {
			direct[imp.Path] = true
			for _, sym := range imp.Symbols {
				if !declares(imp.Path, sym) {
					continue
				}
				bind := sym
				if alias, ok := imp.SymbolAliases[sym]; ok {
					bind = alias
				}
				s.BindExplicit(bind, types.TypeRef{Module: imp.Path, Name: sym})
			}
		}
	}
	for modName, mod := range mr.modules {
		if modName == name {
			continue
		}
		for _, t := range mod.TypeNames {
			ref := types.TypeRef{Module: modName, Name: t}
			if direct[modName] {
				s.AddDirect(ref)
			} else {
				s.AddClosure(ref)
			}
		}
	}
	return s
}

// constructorsInScope decides, for each constructor name the registry knows,
// which loaded module's constructor a module being loaded means — with the
// CLI pipeline's rule: imported by name > offered by exactly one directly
// imported module > offered by exactly one loaded module. A name two modules
// offer at the deciding step is returned in ambiguous (patterns resolve it by
// the scrutinee's type; an expression using it is TC_AMBIGUOUS_CTOR_001).
// Caller holds mr.mu.
func (mr *ModuleRegistry) constructorsInScope(name string, file *ast.File) (chosen map[string]*CachedConstructor, ambiguous map[string][]types.TypeRef) {
	byName := map[string][]*CachedConstructor{}
	for _, c := range mr.constructors {
		if c.Module == name {
			continue // reloading: the module's own constructors are declared again
		}
		byName[c.CtorName] = append(byName[c.CtorName], c)
	}
	explicit := map[string]*CachedConstructor{}
	direct := map[string]bool{}
	if file != nil {
		for _, imp := range file.Imports {
			direct[imp.Path] = true
			for _, sym := range imp.Symbols {
				if c, ok := mr.constructors[ctorKey(imp.Path, sym)]; ok {
					bind := sym
					if alias, ok := imp.SymbolAliases[sym]; ok {
						bind = alias
					}
					explicit[bind] = c
				}
			}
		}
	}
	chosen = map[string]*CachedConstructor{}
	ambiguous = map[string][]types.TypeRef{}
	for bind, c := range explicit {
		chosen[bind] = c
	}
	for ctorName, cands := range byName {
		if _, ok := chosen[ctorName]; ok {
			continue
		}
		var near []*CachedConstructor
		for _, c := range cands {
			if direct[c.Module] {
				near = append(near, c)
			}
		}
		pick := near
		if len(pick) == 0 {
			pick = cands
		}
		if len(pick) == 1 {
			chosen[ctorName] = pick[0]
			continue
		}
		sort.Slice(pick, func(i, j int) bool { return pick[i].Module < pick[j].Module })
		refs := make([]types.TypeRef, len(pick))
		for i, c := range pick {
			refs[i] = types.TypeRef{Module: c.Module, Name: c.TypeName}
		}
		chosen[ctorName] = pick[0] // registered so the name is a constructor; marked ambiguous below
		ambiguous[ctorName] = refs
	}
	return chosen, ambiguous
}

// ctorKey keys the registry's constructor table: one entry per (module,
// constructor), so loading a module that declares a same-named constructor no
// longer replaces another module's.
func ctorKey(module, ctor string) string { return module + "::" + ctor }
