package repl

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// ifcImports resolves the IFC summary of every value a module imports from a
// previously loaded module, under the name the module binds it — the same
// identity rules as the CLI (pipeline resolveIFCImports): a selective import
// under its (possibly renamed) bound name, a module alias as "Alias.name"
// (M-IFC-CROSS-MODULE M3, #1134). Imports of modules the registry has not
// loaded are left to elaboration, which reports them; an export without
// IFC metadata is an error rather than a clean callee.
func (mr *ModuleRegistry) ifcImports(file *ast.File) (types.IFCImports, error) {
	out := make(types.IFCImports)
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	add := func(bound, path, name string, exp *Export) error {
		if err := exp.IFC.Validate(); err != nil {
			return fmt.Errorf("imported %s.%s: %w", path, name, err)
		}
		out[bound] = types.IFCImport{Module: path, Name: name, Sig: exp.IFC}
		return nil
	}
	for _, imp := range file.Imports {
		mod, ok := mr.modules[imp.Path]
		if !ok {
			continue
		}
		if imp.ModuleAlias != "" {
			for name, exp := range mod.Exports {
				if err := add(imp.ModuleAlias+"."+name, imp.Path, name, exp); err != nil {
					return nil, err
				}
			}
		}
		for _, sym := range imp.Symbols {
			exp, ok := mod.Exports[sym]
			if !ok {
				continue // a type or constructor: no value summary
			}
			bound := sym
			if alias, ok := imp.SymbolAliases[sym]; ok {
				bound = alias
			}
			if err := add(bound, imp.Path, sym, exp); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// exportIFC is the summary a loaded module publishes for one export: the
// analysed function summary, or the scheme's labels for a value with no
// surface function (the iface builder's rule for synthetic exports).
func exportIFC(summaries map[string]*types.ImportedIFCSig, name string, scheme *types.Scheme) *types.ImportedIFCSig {
	if s := summaries[name]; s != nil {
		return s
	}
	return types.IFCSummaryFromScheme(scheme)
}
