package pipeline

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-TYPE-NAME-SHADOW (#1322, #1706) made a module's own type declarations win
// over same-named imported types by deleting the local names from the
// name-keyed imported tables, and withheld (TC_TYPE_SHADOW_001) any imported
// alias whose body named a type the module also declared, because expanding
// it would have resolved that name to the LOCAL type.
//
// M-V1-NOMINAL-TYPE-IDENTITY replaced both mechanisms: every imported table is
// keyed by types.NominalKey(defining module, name) and every TCon in an alias
// body carries its defining module, so a local `Row` and an imported `Row` are
// different keys and an imported body can no longer be captured. Local-wins is
// now rule 2 of the type scope (buildTypeScope). What remains here are the
// sorting helpers the alias closure and cache code share.

func sortedTypeKeys(m map[string]types.Type) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
