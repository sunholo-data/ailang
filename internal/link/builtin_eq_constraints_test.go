package link

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
)

// #2291: the list builtins that compare elements carry Eq on the element
// type, and EqVars naming a variable the type does not have is a
// registration bug that fails loudly.
func TestBuiltinEqConstraints(t *testing.T) {
	ml := NewModuleLinker(&stubModuleLoader{})
	RegisterBuiltinModule(ml)
	bi := ml.GetIface("$builtin")
	for _, name := range []string{"_list_contains", "_list_member", "_list_dedup", "_list_union", "_list_intersect", "_list_difference"} {
		item := bi.Exports[name]
		if item == nil || len(item.Type.Constraints) != 1 || item.Type.Constraints[0].Class != "Eq" {
			t.Errorf("%s must carry Eq on its element type, got %v", name, item)
		}
	}
	if cs := bi.Exports["_list_length"].Type.Constraints; len(cs) != 0 {
		t.Errorf("_list_length compares nothing and must carry no constraint, got %v", cs)
	}

	defer func() {
		if recover() == nil {
			t.Error("EqVars naming a variable the type does not have must panic")
		}
	}()
	builtinEqConstraints(&builtins.BuiltinSpec{Name: "_bad", EqVars: []string{"z"}}, []string{"a"})
}
