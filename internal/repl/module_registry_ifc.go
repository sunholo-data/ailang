package repl

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// ifcImports resolves the IFC contracts of every value a module imports from
// previously registered modules (M-IFC-CROSS-MODULE M3, #1134), keyed by the
// same names LoadModule binds for elaboration: selective (`gate`), renamed
// (`g` for `gate as g`) and module-alias qualified (`L.gate`). A module or
// export missing from the registry is skipped here; elaboration rejects it.
func (mr *ModuleRegistry) ifcImports(file *ast.File) types.IFCImports {
	var out types.IFCImports
	if file == nil {
		return out
	}
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	for _, imp := range file.Imports {
		mod, ok := mr.modules[imp.Path]
		if !ok {
			continue
		}
		if imp.ModuleAlias != "" {
			for exportName, export := range mod.Exports {
				out.Add(imp.ModuleAlias+"."+exportName, imp.Path+"."+exportName, export.IFC)
			}
		}
		for _, sym := range imp.Symbols {
			export, ok := mod.Exports[sym]
			if !ok {
				continue
			}
			bindName := sym
			if alias, ok := imp.SymbolAliases[sym]; ok {
				bindName = alias
			}
			out.Add(bindName, imp.Path+"."+sym, export.IFC)
		}
	}
	return out
}

// ifcExportSummaries computes the IFC summary each export of this module
// publishes to later importers, with this module's own imports in scope so a
// wrapper over an imported producer carries its taint. Exports without a
// surface declaration (constants) take the scheme's explicit metadata.
func ifcExportSummaries(file *ast.File, imports types.IFCImports) (map[string]*types.ImportedIFCSig, error) {
	if file == nil {
		return map[string]*types.ImportedIFCSig{}, nil
	}
	summaries, err := types.AnalyzeModuleIFC(file, imports.Sigs)
	if err != nil {
		return nil, fmt.Errorf("IFC export analysis: %w", err)
	}
	return summaries, nil
}

func exportIFC(summaries map[string]*types.ImportedIFCSig, name string, scheme *types.Scheme) *types.ImportedIFCSig {
	if s := summaries[name]; s != nil {
		return s
	}
	return types.IFCSummaryFromScheme(scheme)
}
