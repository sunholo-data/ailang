package link

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2305: every builtin's scheme quantifies every type variable of its type.
// `_list_length : list[a] -> int` used to leave `a` free (the walk had no case
// for type applications), so all its uses shared one `a`.
func TestBuiltinSchemesQuantifyEveryTypeVariable(t *testing.T) {
	ml := NewModuleLinker(&stubModuleLoader{})
	RegisterBuiltinModule(ml)
	bi := ml.GetIface("$builtin")
	if bi == nil || len(bi.Exports) == 0 {
		t.Fatal("no $builtin interface")
	}
	for name, item := range bi.Exports {
		quantified := map[string]bool{}
		for _, v := range item.Type.TypeVars {
			quantified[v] = true
		}
		for _, v := range types.FreeTypeVarNames(item.Type.Type) {
			if !quantified[v] {
				t.Errorf("%s: type variable %s is not quantified in %s", name, v, item.Type)
			}
		}
	}
	if free := types.NewTypeEnvWithBuiltins().FreeTypeVars(); len(free) != 0 {
		t.Errorf("the builtin environment has free type variables %v", free)
	}
}
