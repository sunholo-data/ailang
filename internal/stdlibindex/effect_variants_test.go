package stdlibindex

import (
	"reflect"
	"testing"
)

func TestEffectVariants(t *testing.T) {
	for _, name := range []string{"map", "filter", "foldl"} {
		if got := EffectVariants("std/list", name); !reflect.DeepEqual(got, []string{"std/list." + name + "E"}) {
			t.Errorf("%s: %v", name, got)
		}
	}
	for _, module := range []string{"std/option", "std/io", ""} {
		if got := EffectVariants(module, "map"); len(got) != 0 {
			t.Errorf("false remedy %s: %v", module, got)
		}
	}
}
