package pipeline

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// Contracts survive the cache's iface.json codec (cache key v14). A v13 entry
// decoded with none, so a warm publish would have read a contract change as
// unchanged (M-PKG-PARALLEL-AGENTS M2).
func TestCacheIfaceRoundTripKeepsContracts(t *testing.T) {
	i := iface.NewIface("q/lib")
	i.AddExport("pos", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt}}, true)
	want := []string{"requires (n > 0)", "ensures (result > 0)"}
	i.Exports["pos"].Contracts = want
	i.AddExport("plain", &types.Scheme{Type: types.TInt}, true)

	raw, err := marshalIfaceFull(i)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := unmarshalIfaceFull(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Exports["pos"].Contracts; !reflect.DeepEqual(got, want) {
		t.Errorf("pos contracts = %#v, want %#v", got, want)
	}
	if got := decoded.Exports["plain"].Contracts; got != nil {
		t.Errorf("plain contracts = %#v, want nil", got)
	}
}
