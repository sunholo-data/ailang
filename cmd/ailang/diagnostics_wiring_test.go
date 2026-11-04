package main

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func TestEffectfulVariantWiring(t *testing.T) {
	if types.EffectfulVariantSuggester == nil {
		t.Fatal("effect variant hook is not wired")
	}
	if got := types.EffectfulVariantSuggester("std/list", "map"); !reflect.DeepEqual(got, []string{"std/list.mapE"}) {
		t.Fatalf("wrong remedy: %v", got)
	}
}
