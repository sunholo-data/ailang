package pipeline

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/types"
)

// resolveIFCImports gathers the IFC summary of every imported VALUE under the
// name this module binds it (M-IFC-CROSS-MODULE M3, #1134). It reads the same
// resolved identity map the elaborator uses (imports.GlobalRefs): selective
// imports under their bound — possibly renamed — name, and "Alias.name" for a
// module alias. Builtins ($builtin) and constructor factories ($adt) carry no
// export summary and keep the checker's existing handling.
//
// An imported export whose interface lacks valid IFC metadata is an error:
// a dependency must never be treated as label-clean by default.
func resolveIFCImports(imports *moduleImports, modLinker *link.ModuleLinker) (types.IFCImports, error) {
	bound := make([]string, 0, len(imports.GlobalRefs))
	for name := range imports.GlobalRefs {
		bound = append(bound, name)
	}
	sort.Strings(bound) // deterministic first error
	out := make(types.IFCImports)
	for _, name := range bound {
		ref := imports.GlobalRefs[name]
		if ref.Module == "$builtin" || ref.Module == "$adt" {
			continue
		}
		dep := modLinker.GetIface(ref.Module)
		if dep == nil {
			continue // unresolved module: resolveModuleImports already skipped it
		}
		item, ok := dep.GetExport(ref.Name)
		if !ok {
			continue
		}
		if err := item.IFC.Validate(); err != nil {
			return nil, fmt.Errorf("imported %s.%s: %w", ref.Module, ref.Name, err)
		}
		out[name] = types.IFCImport{Module: ref.Module, Name: ref.Name, Sig: item.IFC}
	}
	return out, nil
}
