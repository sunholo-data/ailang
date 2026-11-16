package pipeline

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// The universe holds every constructor of a type, keyed by its defining
// module, sorted and unique; a module's own declaration replaces a loaded
// copy of the same type; imported elaborator entries are never a universe.
func TestMatchUniverse(t *testing.T) {
	u := make(types.MatchUniverse)
	addIfaceToMatchUniverse(u, &iface.Iface{Module: "q/a", Constructors: map[string]*iface.ConstructorScheme{
		"C":    {TypeName: "T", CtorName: "C", Arity: 2},
		"A":    {TypeName: "T", CtorName: "A"},
		"B":    {TypeName: "T"}, // name from the map key
		"Nil":  nil,
		"Same": {TypeName: "U", CtorName: "Same"},
	}})
	addMatchCtor(u, types.NominalKey("q/a", "T"), "A", 0) // duplicate: ignored
	want := []exhaust.CtorSig{{Name: "A"}, {Name: "B"}, {Name: "C", Arity: 2}}
	if got := u[types.NominalKey("q/a", "T")]; !reflect.DeepEqual(got, want) {
		t.Fatalf("q/a T = %v, want %v", got, want)
	}

	loaded := types.MatchUniverse{types.NominalKey("m", "Old"): {{Name: "Stale"}}, types.NominalKey("q/a", "T"): want}
	mod := moduleMatchUniverse(loaded, "m", map[string]*ConstructorInfo{
		"Fresh": {TypeName: "Old", CtorName: "Fresh"},
		"Alias": {TypeName: "Old"},
		"skip":  nil,
	})
	if got := mod[types.NominalKey("m", "Old")]; !reflect.DeepEqual(got, []exhaust.CtorSig{{Name: "Alias"}, {Name: "Fresh"}}) {
		t.Errorf("local declaration must replace the loaded copy: %v", got)
	}
	if !reflect.DeepEqual(loaded[types.NominalKey("m", "Old")], []exhaust.CtorSig{{Name: "Stale"}}) {
		t.Error("the loaded universe was modified")
	}
	if !reflect.DeepEqual(mod[types.NominalKey("q/a", "T")], want) {
		t.Error("loaded types must be kept")
	}

	single := elaboratorMatchUniverse(map[string]*elaborate.ConstructorInfo{
		"Red":  {TypeName: "Color", Module: "f", CtorName: "Red"},
		"Blue": {TypeName: "Color", Module: "f"},
		"Some": {TypeName: "Option", Module: "std/option", CtorName: "Some", IsImported: true, Arity: 1},
		"nil":  nil,
	})
	if len(single) != 1 || !reflect.DeepEqual(single[types.NominalKey("f", "Color")], []exhaust.CtorSig{{Name: "Blue"}, {Name: "Red"}}) {
		t.Errorf("single-file universe = %v", single)
	}
}
