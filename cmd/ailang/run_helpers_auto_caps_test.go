package main

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestResolveAutoCaps(t *testing.T) {
	moduleIface := iface.NewIface("test")
	moduleIface.AddExport("main", &types.Scheme{
		Type: &types.TFunc2{
			EffectRow: &types.Row{
				Kind: types.EffectRow,
				Labels: map[string]types.Type{
					"IO": types.Unit(),
					"FS": types.Unit(),
				},
			},
			Return: types.Unit(),
		},
	}, false)

	want := []string{"FS", "IO"}
	if got := resolveAutoCaps(moduleIface, "main"); !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveAutoCaps() = %v, want %v", got, want)
	}
}
