package pipeline

import (
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/types"
)

// aliasTables is one module's alias environment, every entry keyed by
// types.NominalKey(defining module, alias name) (M-V1-NOMINAL-TYPE-IDENTITY):
// the module's own aliases and every alias it imports, directly or through its
// import closure. Two modules' same-named aliases are two entries.
type aliasTables struct {
	Bodies map[string]types.Type
	Params map[string][]string
	// Newtypes: keys that are single-constructor ADTs wrapping a record
	// (M-STREAM-DX/M4). They are aliases only for field access; unification
	// and interface closure keep them nominal.
	Newtypes map[string]bool
}

// qualifiedAliasEnv merges a module's own aliases (bare names from the
// elaborator, qualified here with modID) with its imported ones (already
// qualified by resolveModuleImports).
func qualifiedAliasEnv(modID string, imports *moduleImports, elaborator *elaborate.Elaborator) *aliasTables {
	t := &aliasTables{
		Bodies:   make(map[string]types.Type, len(imports.ImportedTypeAliases)),
		Params:   make(map[string][]string, len(imports.ImportedAliasParams)),
		Newtypes: make(map[string]bool, len(imports.ImportedNewtypes)),
	}
	for key, body := range imports.ImportedTypeAliases {
		t.Bodies[key] = body
	}
	for key, params := range imports.ImportedAliasParams {
		t.Params[key] = params
	}
	for key := range imports.ImportedNewtypes {
		t.Newtypes[key] = true
	}
	for name, body := range elaborator.GetTypeAliases() {
		t.Bodies[types.NominalKey(modID, name)] = body
	}
	for name, params := range elaborator.GetTypeAliasParams() {
		t.Params[types.NominalKey(modID, name)] = params
	}
	for name := range elaborator.GetNewtypeADTs() {
		t.Newtypes[types.NominalKey(modID, name)] = true
	}
	return t
}
